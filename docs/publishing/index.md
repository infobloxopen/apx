# Publishing Workflow

APX implements a **PR-first release model**: every API change reaches the
canonical repository through a pull request that is validated before merge
and tagged after merge.

Two paths lead to that PR — choose the one that fits your workflow:

| Path | Best for | Steps |
|------|----------|-------|
| **Release pipeline** (`apx release`) | CI pipelines, production releases, audit trails | `prepare` → `submit` → `finalize` |
| **Quick publish** (`apx publish`) | Local development, quick iterations | Single command |

Both paths perform the same core operations: validate the API, push a
snapshot to the canonical repo, and open a pull request via the `gh` CLI.
The release pipeline adds a manifest, idempotency checks, catalog updates,
and an immutable release record.

## The Release Pipeline (Recommended)

::::{grid} 1 1 3 3
:gutter: 2

:::{grid-item-card} **1. Prepare**
^^^
```bash
apx release prepare \
  proto/payments/ledger/v1 \
  --version v1.0.0-beta.1 \
  --lifecycle beta
```
Validates schemas, lifecycle policy, version-line compatibility.
Writes `.apx-release.yaml`.
:::

:::{grid-item-card} **2. Submit**
^^^
```bash
apx release submit
```
Clones the canonical repo, copies the snapshot to a release branch,
opens a PR via `gh`.  Idempotent — safe to retry.
:::

:::{grid-item-card} **3. Finalize**
^^^
```bash
apx release finalize
```
Run by canonical CI after merge: re-validates schemas, creates the
official tag, updates the catalog, emits a release record.
:::

::::

See [Release Commands](../cli-reference/release-commands.md) for full
flag reference and state machine details.

### Manifest and Release Record

Every release pipeline run produces two artifacts:

| Artifact | Created by | Purpose |
|----------|-----------|---------|
| `.apx-release.yaml` | `prepare` (updated by each phase) | Tracks state, identity, validation results, PR metadata |
| `.apx-release-record.yaml` | `finalize` | Immutable audit record with CI provenance, artifacts, catalog status |

### Lifecycle Promotions

The release pipeline also supports lifecycle transitions:

```bash
# Promote from beta to stable
apx release promote proto/payments/ledger/v1 --to stable --version v1.0.0
apx release submit

# Mark as deprecated
apx release promote proto/payments/ledger/v1 --to deprecated
apx release submit
```

See [Lifecycle Reference](lifecycle.md) for the full lifecycle model.

## Quick Publish (Convenience Path)

For fast, one-shot publishing — ideal during local development:

```bash
apx publish proto/payments/ledger/v1 \
  --version v1.0.0-beta.1 \
  --lifecycle beta
```

This single command validates, pushes a snapshot branch, and opens a PR.
It does **not** write a manifest, update the catalog, or emit a release
record.

See [Publish Command](publish-command.md) for full usage.

## Validation Pipeline

Every publish or release goes through validation at multiple stages:

::::{grid} 1 1 2 2
:gutter: 3

:::{grid-item-card} **Pre-Publish (Local / App CI)**
^^^
- `apx lint` — schema linting
- `apx breaking` — backward compatibility
- `apx semver suggest` — version recommendation
- `apx policy check` — organizational policy
:::

:::{grid-item-card} **Prepare / Publish**
^^^
- API ID parsing and identity derivation
- Lifecycle-version compatibility
- `go_package` consistency (proto)
- `go.mod` module path validation
- Idempotency check (release pipeline)
:::

:::{grid-item-card} **Canonical CI (PR)**
^^^
- Re-validates schemas in canonical context
- Re-checks breaking changes against previous tag
- Runs policy validation
:::

:::{grid-item-card} **Finalize (Release Pipeline)**
^^^
- Re-runs lint and breaking checks
- Creates annotated git tag
- Updates catalog
- Emits release record with CI provenance
:::

::::

See [Publishing Validation](validation.md) for the full validation matrix.

## Subdirectory Tagging

APX uses **subdirectory-scoped git tags** so every API line is versioned
independently within a single canonical repository.

```
<api-id>/v<semver>
```

Examples:

```
proto/payments/ledger/v1/v1.0.0-beta.1
proto/payments/ledger/v1/v1.0.0
proto/users/profile/v1/v1.1.0
```

Tags are created by `apx release finalize` (or by canonical CI after
an `apx publish` PR is merged).

See [Tagging Strategy](tagging-strategy.md) for full details.

## Example: End-to-End Release

### 1. Validate Locally

```bash
apx fetch                          # ensure latest toolchain
apx lint                           # check schema quality
apx breaking --against=HEAD^       # verify backward compatibility
apx semver suggest --against=HEAD^ # get recommended version bump
```

### 2. Prepare the Release

```bash
apx release prepare proto/payments/ledger/v1 \
  --version v1.2.0 \
  --lifecycle stable
```

### 3. Submit to Canonical Repo

```bash
apx release submit
# ✓ Pull request created
# PR: https://github.com/acme/apis/pull/42
```

### 4. Canonical CI Processes

After the PR is reviewed and merged, canonical CI runs:

```bash
apx release finalize
```

This creates the tag `proto/payments/ledger/v1/v1.2.0`, updates the
catalog, and writes the release record.

## Error Handling

Common publishing errors and solutions:

| Error | Cause | Solution |
|-------|-------|----------|
| Version mismatch | Tagged version doesn't match suggested | Update version to match breaking changes |
| Breaking change on stable | Breaking change with minor/patch bump | Use a new major API line (`v2`) |
| Policy violation | Banned annotation detected | Remove service-specific annotations |
| Lifecycle mismatch | Version tag incompatible with lifecycle | Use the correct prerelease tag for the lifecycle |
| `gh` not found | GitHub CLI not installed | `brew install gh` and `gh auth login` |

## Best Practices

- **Use the release pipeline for CI** — it provides audit trails and idempotency
- **Use quick publish for local iteration** — faster feedback loop
- **Run `apx lint` and `apx breaking` before publish** — catch issues early
- **Follow lifecycle conventions** — `experimental` for alpha, `beta` for beta/rc, `stable` for GA
- **Coordinate major versions** across teams — new API lines (`v2`) affect all consumers

## Next Steps

- [Release Commands](../cli-reference/release-commands.md) — full release state machine reference
- [Publish Command](publish-command.md) — quick publish CLI reference
- [Publishing Overview](overview.md) — identity model and two-path architecture
- [Lifecycle Reference](lifecycle.md) — lifecycle states, transitions, and compatibility promises
- [Tagging Strategy](tagging-strategy.md) — subdirectory tag format
- [Release Guardrails](release-guardrails.md) — policy enforcement during releases
- [Publishing Validation](validation.md) — validation pipeline details
