# Releasing


APX implements a **PR-first release model**: every API change reaches the
canonical repository through a pull request that is validated before merge
and tagged after merge.

The `apx release` pipeline is the single path to get an API into the
canonical repository.  It validates the API, pushes a snapshot to the
canonical repo, opens a pull request via the `gh` CLI, and provides
manifest persistence, idempotency checks, catalog updates, and an
immutable release record.

## The Release Pipeline

<div class="grid cards" markdown>
-   **1. Prepare**

    ---

    ```bash
    apx release prepare \
    proto/payments/ledger/v1 \
    --version v1.0.0-beta.1 \
    --lifecycle beta
    ```
    Validates schemas, lifecycle policy, version-line compatibility.
    Writes `.apx-release.yaml`.

-   **2. Submit**

    ---

    ```bash
    apx release submit
    ```
    Clones the canonical repo, copies the snapshot to a release branch,
    opens a PR via `gh`.  Idempotent — safe to retry.

-   **3. Finalize**

    ---

    ```bash
    apx release finalize
    ```
    Run by canonical CI after merge: re-validates schemas, creates the
    official tag, updates the catalog, emits a release record.

</div>

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

## Validation Pipeline

Every release goes through validation at multiple stages:

<div class="grid cards" markdown>
-   **Pre-Release (Local / App CI)**

    ---

    - `apx lint` — schema linting
    - `apx breaking` — backward compatibility
    - `apx semver suggest` — version recommendation
    - `apx policy check` — organizational policy

-   **Prepare**

    ---

    - API ID parsing and identity derivation
    - Lifecycle-version compatibility
    - `go_package` consistency (proto)
    - `go.mod` module path validation
    - Idempotency check (SHA-256 content hash)

-   **Canonical CI (PR)**

    ---

    - Re-validates schemas in canonical context
    - Re-checks breaking changes against previous tag
    - Runs policy validation

-   **Finalize (Release Pipeline)**

    ---

    - Re-runs lint and breaking checks
    - Creates annotated git tag
    - Updates catalog
    - Emits release record with CI provenance

</div>

See [Release Validation](validation.md) for the full validation matrix.

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

Tags are created by `apx release finalize` after the PR is merged.

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

Common release errors and solutions:

| Error | Cause | Solution |
|-------|-------|----------|
| Version mismatch | Tagged version doesn't match suggested | Update version to match breaking changes |
| Breaking change on stable | Breaking change with minor/patch bump | Use a new major API line (`v2`) |
| Policy violation | Banned annotation detected | Remove service-specific annotations |
| Lifecycle mismatch | Version tag incompatible with lifecycle | Use the correct prerelease tag for the lifecycle |
| `gh` not found | GitHub CLI not installed | `brew install gh` and `gh auth login` |

## Best Practices

- **Use `apx release` for all releases** — it provides audit trails, idempotency, and catalog updates
- **Run `apx lint` and `apx breaking` before releasing** — catch issues early
- **Follow lifecycle conventions** — `experimental` for alpha, `beta` for beta/rc, `stable` for GA
- **Coordinate major versions** across teams — new API lines (`v2`) affect all consumers

## Next Steps

- [Release Commands](../cli-reference/release-commands.md) — full release state machine reference
- [Releasing Overview](overview.md) — identity model and release pipeline architecture
- [Lifecycle Reference](lifecycle.md) — lifecycle states, transitions, and compatibility promises
- [Tagging Strategy](tagging-strategy.md) — subdirectory tag format
- [Release Guardrails](release-guardrails.md) — policy enforcement during releases
- [Release Validation](validation.md) — validation pipeline details
