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
| `--lifecycle` | Lifecycle state (experimental, beta, stable, deprecated, sunset) |
| `--canonical-repo` | Canonical repository URL (auto-derived from apx.yaml) |
| `--module-path` | Module path (legacy; prefer positional api-id) |
| `--dry-run` | Show what would be published without publishing |
| `--create-pr` | Create a pull request instead of pushing directly |

## What Publish Does

1. **Parses** the API ID into format, domain, name, and line
2. **Derives** the canonical source path from the API ID
3. **Derives** Go module and import paths
4. **Validates** consistency of paths and `go_package` options
5. **Creates** a subdirectory-scoped git tag
6. **Pushes** the module to the canonical repository

## Identity Integration

When using the API ID form, publish automatically computes:

- **Source path**: `proto/payments/ledger/v1`
- **Git tag**: `proto/payments/ledger/v1/v1.0.0-beta.1`
- **Go module**: `github.com/<org>/apis/proto/payments/ledger`
- **Go import**: `github.com/<org>/apis/proto/payments/ledger/v1`
