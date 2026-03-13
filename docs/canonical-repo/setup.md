# Canonical Repository Setup

The canonical repository (`github.com/<org>/apis`) is the single source of truth for all organization API schemas.
This guide walks through creating the repository, scaffolding the directory structure, and configuring GitHub protections — either automatically via `--setup-github` or manually.

## Prerequisites

| Tool | Purpose | Install |
|------|---------|---------|
| **apx** | Scaffold & manage schemas | `brew install infobloxopen/tap/apx` or [installation guide](../getting-started/installation.md) |
| **gh** | GitHub CLI (optional, required for `--setup-github`) | `brew install gh` or [cli.github.com](https://cli.github.com) |
| **git** | Version control | Pre-installed on most systems |
| **buf** | Protocol Buffer tooling (optional) | `brew install bufbuild/buf/buf` |

!!! tip
    If you plan to use `--setup-github` for automated protection rules and org secrets, authenticate `gh` first:

    ```bash
    gh auth login
    gh auth refresh -h github.com -s admin:org   # needed for org secrets
    ```

## Step 1: Create the GitHub Repository

Create the canonical repo on GitHub before scaffolding:

```bash
# Option A: via gh CLI
gh repo create <org>/apis --public --clone
cd apis

# Option B: Create on github.com, then clone
git clone https://github.com/<org>/apis.git
cd apis
```

!!! note
    The repo name `apis` is conventional but not required. Use whatever name fits your organization — just pass `--repo=<name>` to `apx init canonical`.

## Step 2: Scaffold the Structure

### Interactive Mode (default)

```bash
apx init canonical
```

APX auto-detects your `org` and `repo` from the git remote. If it can't detect them, it prompts:

```
🚀 Initializing canonical API repository!

Organization name: [auto-detected or enter manually]
Repository name:   [auto-detected or enter manually]
```

### Non-Interactive Mode

```bash
apx init canonical --org=<org> --repo=apis --non-interactive
```

All flags:

| Flag | Description | Default |
|------|-------------|---------|
| `--org` | GitHub organization name | Auto-detected from git remote |
| `--repo` | Repository name | Auto-detected from current directory |
| `--skip-git` | Skip the "next steps" git instructions | `false` |
| `--non-interactive` | Disable prompts; require all flags | `false` |
| `--setup-github` | Configure GitHub protections via `gh` CLI | `false` |
| `--app-id` | GitHub App ID (with `--setup-github`) | Cached or created via manifest flow |
| `--app-pem-file` | Path to GitHub App private key PEM (with `--setup-github`) | Cached or created via manifest flow |

### What Gets Generated

```
apis/
├── apx.yaml                     # APX project configuration
├── buf.yaml                     # Buf v2 lint/breaking policy
├── buf.work.yaml                # Buf workspace (all format directories)
├── CODEOWNERS                   # Per-path team ownership
├── README.md                    # Repo overview
├── catalog/
│  ├── .gitignore                # ignores generated catalog.yaml
│  └── Dockerfile                # scratch-based image with OCI labels
├── .github/
│  └── workflows/
│     ├── ci.yml                 # PR validation (lint + breaking)
│     └── on-merge.yml           # Post-merge tagging + catalog update
├── proto/
│  └── .gitkeep
├── openapi/
│  └── .gitkeep
├── avro/
│  └── .gitkeep
├── jsonschema/
│  └── .gitkeep
└── parquet/
   └── .gitkeep
```

The command reports each generated file:

```
Initializing canonical API repository...
Organization: acme
Repository: apis

✓ Created directory structure
✓ Generated buf.yaml
✓ Generated CODEOWNERS
✓ Generated catalog/Dockerfile
✓ Generated README.md
✓ Generated apx.yaml
✓ Generated .github/workflows/ci.yml
✓ Generated .github/workflows/on-merge.yml

✓ Canonical API repository initialized successfully!
```

!!! tip
    Running `apx init canonical` is **idempotent** — it skips files that already exist, so you can safely re-run it.

## Step 3: Configure GitHub Protections

You have two options: **automated** (recommended) or **manual**.

### Option A: Automated Setup (`--setup-github`)

```bash
apx init canonical --org=acme --repo=apis --setup-github
```

This performs four operations via the GitHub API:

1. **GitHub App creation** — If no App is cached, APX opens your browser to create one via the [GitHub App manifest flow](https://docs.github.com/en/apps/sharing-github-apps/registering-a-github-app-from-a-manifest). The App is named `apx-<repo>-<org>` and granted `contents:write`, `pull_requests:write`, and `metadata:read` permissions.

2. **Org secrets** — Sets two organization-level Actions secrets:
   - `APX_APP_ID` — The App's numeric ID
   - `APX_APP_PRIVATE_KEY` — The App's private key (PEM)

3. **Branch protection on `main`** — Requires:
   - Pull request reviews (1 approval, CODEOWNERS review, dismiss stale reviews)
   - Status checks (`validate` must pass)
   - Strict status checks (branch must be up to date)

4. **Tag protection ruleset** — Creates an `apx-tag-protection` ruleset that prevents direct tag creation/deletion, allowing only organization admins as bypass actors. This ensures CI is the sole creator of release tags.

**What you see:**

```
Opening browser to create GitHub App for org "acme"...
App created! App ID: 123456
Opening browser to install the App...
Waiting for installation to complete...
App installed on "acme"!

Configuring GitHub repository...
  ✓ Created: org secret APX_APP_ID
  ✓ Created: org secret APX_APP_PRIVATE_KEY
  ✓ Created: branch protection on main
  ✓ Created: tag protection ruleset

✓ Canonical API repository initialized successfully!
```

On subsequent runs, already-configured items are reported as skipped:

```
  ✓ Already configured: org secret APX_APP_ID
  ✓ Already configured: branch protection on main
```

!!! important
    The `--setup-github` flow requires the `admin:org` OAuth scope. If your token is missing it, APX will tell you:

    ```
    gh token is missing the 'admin:org' scope needed for org secrets.
    Run: gh auth refresh -h github.com -s admin:org
    ```

    Some operations (org secrets, tag rulesets) require **organization admin** privileges. If you lack them, APX logs a warning with the manual command to run as an admin.

#### Credential Caching

APX caches GitHub App credentials locally at `~/.config/apx/`:

| File | Contents |
|------|----------|
| `<org>-app-id` | GitHub App numeric ID |
| `<org>-app-slug` | GitHub App slug (e.g. `apx-apis-acme`) |
| `<org>-app.pem` | Private key (mode `0600`) |

On subsequent runs, APX uses cached credentials instead of re-creating the App. To supply credentials manually on a new machine:

```bash
apx init canonical --setup-github \
  --app-id=123456 \
  --app-pem-file=/path/to/private-key.pem
```

### Option B: Manual Setup

If you prefer to configure protections by hand (or lack org admin access), skip `--setup-github` and follow the printed instructions:

```
Next steps:
1. Initialize git: git init
2. Add files: git add .
3. Commit: git commit -m 'Initial canonical repository scaffold'
4. Create GitHub repository and set up branch protection:
   - Require pull request reviews
   - Require status checks (lint, breaking)
   - Require CODEOWNERS review
   - Restrict direct pushes to main
5. Push: git remote add origin <url> && git push -u origin main

Or re-run with --setup-github to configure automatically via gh CLI.
```

#### Manual Branch Protection

Go to **Settings → Branches → Add rule** for `main`:

- [x] Require a pull request before merging (1 approval)
- [x] Require review from Code Owners
- [x] Dismiss stale pull request approvals
- [x] Require status checks to pass → add `validate`
- [x] Require branches to be up to date before merging

#### Manual Tag Protection

Go to **Settings → Rules → Rulesets → New tag ruleset**:

- **Name**: `apx-tag-protection`
- **Target**: All tags
- **Rules**: Restrict creations, Restrict deletions
- **Bypass**: Organization admins only

This ensures only CI (via the GitHub App token) can create release tags.

## Generated File Details

### `buf.yaml`

Organization-wide lint and breaking change policy for Protocol Buffers:

```yaml
version: v2
modules:
  - path: proto
breaking:
  use:
    - FILE
lint:
  use:
    - STANDARD
```

### `buf.work.yaml`

Buf workspace configuration aggregating all schema format directories:

```yaml
version: v2
directories:
  - proto
  - openapi
  - avro
  - jsonschema
  - parquet
```

### `CODEOWNERS`

Per-path ownership rules. Customize team names after scaffolding:

```
* @<org>/api-owners

/proto/    @<org>/proto-owners
/openapi/  @<org>/openapi-owners
/avro/     @<org>/avro-owners
/jsonschema/ @<org>/jsonschema-owners
/parquet/  @<org>/parquet-owners
```

### `catalog/Dockerfile`

Scratch-based Dockerfile for building the catalog OCI image. CI injects OCI labels via build args:

```dockerfile
ARG CREATED
ARG REVISION
ARG SOURCE
ARG VERSION

FROM scratch

LABEL org.opencontainers.image.title="API Catalog" \
      org.opencontainers.image.description="APX API catalog data for discovery and search" \
      org.opencontainers.image.source="${SOURCE}" \
      org.opencontainers.image.created="${CREATED}" \
      org.opencontainers.image.revision="${REVISION}" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.vendor="<org>" \
      dev.apx.type="catalog"

COPY catalog.yaml /catalog.yaml
```

### `catalog/.gitignore`

Ensures the generated `catalog.yaml` is not committed — it is a CI artifact baked into the Docker image:

```
catalog.yaml
```

### `.github/workflows/ci.yml`

Runs on every pull request to `main`. Installs APX, lints schemas, and checks for breaking changes:

```yaml
name: APX Schema CI
on:
  pull_request:
    branches: [main]
jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: infobloxopen/apx@main
      - run: apx lint
      - run: apx breaking --against origin/main
```

### `.github/workflows/on-merge.yml`

Runs on push to `main`. Generates catalog data, builds and pushes a Docker image to GHCR, and attests the build:

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
      - uses: actions/create-github-app-token@v1
        id: app-token
        with:
          app-id: ${{ secrets.APX_APP_ID }}
          private-key: ${{ secrets.APX_APP_PRIVATE_KEY }}
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
          token: ${{ steps.app-token.outputs.token }}
      - uses: infobloxopen/apx@main
      - run: apx lint
      - run: apx catalog generate --output catalog/catalog.yaml
      - uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ steps.app-token.outputs.token }}
      - run: |
          docker build \
            --build-arg CREATED="$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
            --build-arg REVISION="${{ github.sha }}" \
            --build-arg SOURCE="https://github.com/${{ github.repository }}" \
            --build-arg VERSION="${{ github.sha }}" \
            -t "$IMAGE:latest" \
            -t "$IMAGE:sha-${GITHUB_SHA::7}" \
            catalog/
      - run: |
          docker push "$IMAGE:latest"
          docker push "$IMAGE:sha-${GITHUB_SHA::7}"
      - uses: actions/attest-build-provenance@v2
        with:
          subject-name: ${{ env.IMAGE }}
          push-to-registry: true
      - uses: anchore/sbom-action@v0
        with:
          image: ${{ env.IMAGE }}:latest
          output-file: sbom.spdx.json
      - uses: actions/attest-sbom@v2
        with:
          subject-name: ${{ env.IMAGE }}
          sbom-path: sbom.spdx.json
          push-to-registry: true
```

## Verifying the Setup

After pushing your scaffold and configuring protections:

```bash
# Confirm the scaffold is committed
git log --oneline -1
# → abc1234 Initial canonical repository scaffold

# Verify branch protection (requires gh)
gh api repos/<org>/apis/branches/main/protection --jq '.required_pull_request_reviews'

# Verify tag ruleset
gh api repos/<org>/apis/rulesets --jq '.[].name'
# → apx-tag-protection

# Verify org secrets exist
gh secret list --org <org> | grep APX_
# → APX_APP_ID          Updated ...
# → APX_APP_PRIVATE_KEY Updated ...
```

## Next Steps

Once the canonical repo is set up:

1. **[Understand the directory structure](structure.md)** — Learn how APIs are organized by format, domain, and version
2. **[Set up CI templates](ci-templates.md)** — Customize validation and release workflows
3. **[Configure branch & tag protection](protection.md)** — Deeper dive into protection rules
4. **[Set up an app repo](../app-repos/index.md)** — Start authoring schemas with `apx init app`
5. **[Release your first API](../releasing/overview.md)** — Walk through the release submission flow
