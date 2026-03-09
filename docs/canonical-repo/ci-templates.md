# CI Templates

APX generates GitHub Actions workflow files for both canonical and app repositories. These templates are built into the APX binary and can be regenerated at any time with `apx workflows sync`.

## Overview

| Workflow | Repository type | File | Trigger |
|----------|----------------|------|---------|
| **Schema CI** | Canonical | `.github/workflows/ci.yml` | Pull requests to `main` |
| **On Merge** | Canonical | `.github/workflows/on-merge.yml` | Push to `main` |
| **Release** | App | `.github/workflows/apx-release.yml` | Tag push matching APX patterns |

## Canonical Repository Workflows

Canonical repos receive two workflow files during `apx init canonical --setup-github` or `apx workflows sync`.

### `ci.yml` — Schema Validation on PRs

Runs on every pull request targeting `main`. This is the required status check enforced by branch protection.

```yaml
name: APX Schema CI

on:
  pull_request:
    branches: [main]

permissions:
  contents: read

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Install APX
        uses: infobloxopen/apx@v1

      - name: Lint schemas
        run: apx lint

      - name: Check for breaking changes
        run: apx breaking --against origin/main
```

**What it validates:**

- **`apx lint`** — runs Buf lint (for proto) or format-specific validators on all schemas in the repo
- **`apx breaking --against origin/main`** — detects backward-incompatible changes against the main branch

:::{note}
`fetch-depth: 0` is required so that `apx breaking` can compare against `origin/main`.
:::

The `validate` job name matches the required status check configured by branch protection (see [Protection](protection.md)).

---

### `on-merge.yml` — Catalog Build & Publish on Merge

Runs when a PR merges to `main`. Validates schemas, generates catalog data, builds a Docker image with OCI labels, pushes to GHCR, and attests the build.

```yaml
name: APX On Merge

on:
  push:
    branches: [main]

permissions:
  contents: read
  packages: write
  id-token: write
  attestations: write

jobs:
  catalog:
    runs-on: ubuntu-latest
    env:
      IMAGE: ghcr.io/<org>/${{ github.event.repository.name }}-catalog
    steps:
      - name: Generate App Token
        id: app-token
        uses: actions/create-github-app-token@v1
        with:
          app-id: ${{ secrets.APX_APP_ID }}
          private-key: ${{ secrets.APX_APP_PRIVATE_KEY }}

      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
          token: ${{ steps.app-token.outputs.token }}

      - name: Install APX
        uses: infobloxopen/apx@v1

      - name: Validate schemas
        run: apx lint

      - name: Generate catalog data
        run: apx catalog generate --output catalog/catalog.yaml

      - name: Log in to GHCR
        uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ steps.app-token.outputs.token }}

      - name: Build catalog image
        run: |
          docker build \
            --build-arg CREATED="$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
            --build-arg REVISION="${{ github.sha }}" \
            --build-arg SOURCE="https://github.com/${{ github.repository }}" \
            --build-arg VERSION="${{ github.sha }}" \
            -t "$IMAGE:latest" \
            -t "$IMAGE:sha-${GITHUB_SHA::7}" \
            catalog/

      - name: Push catalog image
        run: |
          docker push "$IMAGE:latest"
          docker push "$IMAGE:sha-${GITHUB_SHA::7}"

      - name: Attest build provenance
        uses: actions/attest-build-provenance@v2
        with:
          subject-name: ${{ env.IMAGE }}
          push-to-registry: true
```

**What it does:**

1. **Generates a GitHub App token** — uses the APX GitHub App credentials (`APX_APP_ID` and `APX_APP_PRIVATE_KEY` org secrets) to get a token with `packages:write` permission
2. **Validates schemas** — re-runs lint as a post-merge safety check
3. **Generates catalog data** — `apx catalog generate` scans all modules and git tags to build `catalog/catalog.yaml` (not committed — gitignored as a CI artifact)
4. **Builds a Docker image** — uses the `catalog/Dockerfile` (scaffolded by `apx init canonical`) with OCI best-practice labels injected via build args
5. **Pushes to GHCR** — tags both `:latest` (always overridden) and `:sha-<short>` (audit trail)
6. **Attests the build** — uses GitHub's build provenance attestation for supply-chain security

:::{important}
The catalog data (`catalog/catalog.yaml`) is gitignored and not committed. It is a CI-only artifact that is baked into the Docker image and pushed to GHCR. Consumers discover APIs by pulling the catalog image from the registry.
:::

---

## App Repository Workflow

App repos receive one workflow file during `apx init app` or `apx workflows sync`.

### `apx-release.yml` — Release on Tag

Triggered when you push a tag matching the APX naming pattern. Validates the schema and releases to the canonical repository via PR.

```yaml
name: APX Release

on:
  push:
    tags:
      - "proto/**/v[0-9]*"
      - "openapi/**/v[0-9]*"
      - "avro/**/v[0-9]*"
      - "jsonschema/**/v[0-9]*"
      - "parquet/**/v[0-9]*"

permissions:
  contents: read

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - name: Generate App Token
        id: app-token
        uses: actions/create-github-app-token@v1
        with:
          app-id: ${{ secrets.APX_APP_ID }}
          private-key: ${{ secrets.APX_APP_PRIVATE_KEY }}
          owner: <org>
          repositories: <canonical-repo>

      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Install APX
        uses: infobloxopen/apx@v1

      - name: Parse tag
        id: tag
        run: |
          TAG="${GITHUB_REF#refs/tags/}"
          echo "tag=${TAG}" >> "$GITHUB_OUTPUT"

      - name: Validate
        run: |
          apx lint
          apx breaking --against HEAD^ || true

      - name: Extract API ID and version from tag
        id: parse
        run: |
          TAG="${{ steps.tag.outputs.tag }}"
          # Tag format: <api-id>/<version>  e.g. proto/payments/ledger/v1/v1.0.0
          VERSION="${TAG##*/}"            # last component
          API_ID="${TAG%/*}"              # everything before last /
          echo "api_id=${API_ID}" >> "$GITHUB_OUTPUT"
          echo "version=${VERSION}" >> "$GITHUB_OUTPUT"

      - name: Release to canonical repo
        env:
          GITHUB_TOKEN: ${{ steps.app-token.outputs.token }}
        run: |
          apx release prepare "${{ steps.parse.outputs.api_id }}" \
            --version "${{ steps.parse.outputs.version }}" \
            --canonical-repo=github.com/<org>/<canonical-repo>
          apx release submit
```

**What it does:**

1. **Matches tag patterns** — triggers on tags like `proto/payments/ledger/v1/v1.0.0` across all supported schema formats
2. **Generates a cross-repo token** — the GitHub App token is scoped to the canonical repository (`owner` + `repositories` fields) so the workflow can push branches and open PRs there
3. **Parses the tag** — extracts the full tag string for `apx release prepare`
4. **Validates locally** — runs lint and breaking-change checks before releasing
5. **Releases via PR** — `apx release prepare` validates and stages the release, then `apx release submit` clones the canonical repo, copies module files to a release branch, and opens a pull request

:::{note}
The `owner` and `repositories` fields in the token step are filled in by `apx init app` or `apx workflows sync` based on your `apx.yaml` configuration.
:::

---

## Managing Workflows

### `apx workflows sync`

Regenerate workflow files from the latest APX templates:

```bash
# Regenerate workflows
apx workflows sync

# Preview without writing
apx workflows sync --dry-run
```

**How detection works:**

1. If `.github/workflows/ci.yml` or `on-merge.yml` exists → canonical repo
2. If `.github/workflows/apx-release.yml` exists → app repo
3. Fallback: if `proto/`, `openapi/`, `avro/`, or `catalog/` directories exist → canonical repo
4. Fallback: if `module_roots` is set in `apx.yaml` → app repo

The org and repo values are read from `apx.yaml`. If no config file exists, APX falls back to detecting from the `origin` git remote.

### When to Sync

Run `apx workflows sync` after:

- **Upgrading APX** — templates may have been updated
- **Changing org or repo** — the on-merge and release workflows embed org/repo names
- **Adding a canonical repo** — if your app repo also hosts canonical schemas

### After Syncing

```bash
# Review what changed
git diff .github/workflows/

# Commit
git add .github/workflows/ && git commit -m 'chore: sync APX workflows'
```

---

## Prerequisites

All three workflows require:

| Requirement | Purpose |
|-------------|---------|
| **APX GitHub App** | Provides tokens for push/PR operations |
| **`APX_APP_ID` org secret** | GitHub App ID |
| **`APX_APP_PRIVATE_KEY` org secret** | GitHub App private key (PEM) |

The app release workflow additionally requires the GitHub App to be installed on the canonical repository with `contents:write` and `pull_requests:write` permissions.

See [Protection](protection.md) for how to set up the GitHub App and org secrets.

## Customization

The generated workflows are standard GitHub Actions YAML — you can customize them after generation. Common additions:

- **Additional validation steps** (e.g., custom policy checks with `apx policy check`)
- **Notification steps** (Slack, email on release)
- **Language package releasing** in the on-merge workflow
- **Matrix builds** for multi-format repos

:::{warning}
Running `apx workflows sync` will **overwrite** your customizations. If you've modified the generated workflows, either skip syncing or re-apply your changes after syncing.
:::

## Next Steps

- [Protection](protection.md) — set up branch protection and the GitHub App
- [Setup](setup.md) — scaffold a new canonical repository
- [App Repo CI Integration](../app-repos/ci-integration.md) — CI from the app repo perspective
