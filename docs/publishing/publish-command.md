# The publish Command

The `apx publish` command publishes an API module to the canonical repository.

## Usage

```bash
# Identity-based publish (recommended)
apx publish <api-id> --version <semver> [--lifecycle <state>]

# Legacy publish
apx publish --module-path <path> --canonical-repo <url> --version <semver>
```

## Examples

### New API — alpha release

```bash
apx publish proto/payments/ledger/v1 --version v1.0.0-alpha.1 --lifecycle experimental
```

### Beta release

```bash
apx publish proto/payments/ledger/v1 --version v1.0.0-beta.1 --lifecycle beta
```

### GA release

```bash
apx publish proto/payments/ledger/v1 --version v1.0.0 --lifecycle stable
```

### Additive change (minor bump)

```bash
apx publish proto/payments/ledger/v1 --version v1.1.0 --lifecycle stable
```

### Breaking change (new API line)

```bash
apx publish proto/payments/ledger/v2 --version v2.0.0-alpha.1 --lifecycle experimental
```

### Dry run

```bash
apx publish proto/payments/ledger/v1 --version v1.0.0 --dry-run
```

## Flags

| Flag | Description |
|------|-------------|
| `--version` | SemVer version to publish (required) |
| `--lifecycle` | Lifecycle state (experimental, beta, stable, deprecated, sunset; `preview` accepted as alias for `beta`) |
| `--canonical-repo` | Canonical repository URL (auto-derived from apx.yaml) |
| `--module-path` | Module path (legacy; prefer positional api-id) |
| `--dry-run` | Show what would be published without publishing |
| `--strict` | Make `go_package` mismatches an error instead of a warning |
| `--skip-gomod` | Skip `go.mod` generation and validation |
| `--create-pr` | Create a pull request on the canonical repo (default behavior) |

:::{tip}
For CI pipelines and production releases, consider using the
[release pipeline](../cli-reference/release-commands.md) instead.
It provides a manifest, idempotency checks, catalog updates, and
an immutable release record.
:::

## What Publish Does

1. **Parses** the API ID into format, domain, name, and line
2. **Derives** the canonical source path from the API ID
3. **Derives** Go module and import paths
4. **Validates** consistency of paths and `go_package` options
5. **Generates** or validates `go.mod` for the module (unless `--skip-gomod`)
6. **Publishes** the module via PR:
   - Shallow-clones the canonical repo, copies files to a feature branch, pushes, and opens a PR via `gh`

### PR-based Publish

When publishing, APX:

1. Verifies the `gh` CLI is installed and authenticated
2. Shallow-clones the canonical repo to a temp directory
3. Creates a feature branch named `apx/publish/<api-id>/<version>`
4. Copies module files from your local repo into the canonical path
5. Generates `go.mod` if missing
6. Commits, pushes the branch, and opens a PR

```bash
# Requires: gh auth login  (one-time setup)
apx publish proto/payments/ledger/v1 --version v1.0.0-beta.1 --lifecycle beta
```

The PR title follows the convention `publish: <api-id>@<version>`.

## Identity Integration

When using the API ID form, publish automatically computes:

- **Source path**: `proto/payments/ledger/v1`
- **Git tag**: `proto/payments/ledger/v1/v1.0.0-beta.1`
- **Go module**: `github.com/<org>/apis/proto/payments/ledger`
- **Go import**: `github.com/<org>/apis/proto/payments/ledger/v1`
