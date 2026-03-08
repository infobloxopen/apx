# Publishing Commands

## `apx publish`

Publish an API module to the canonical repository.

### Identity-Based Publish (Recommended)

```bash
apx publish <api-id> --version <semver> [--lifecycle <state>]
```

Examples:

```bash
# Alpha release
apx publish proto/payments/ledger/v1 --version v1.0.0-alpha.1 --lifecycle experimental

# Beta release
apx publish proto/payments/ledger/v1 --version v1.0.0-beta.1 --lifecycle beta

# GA release
apx publish proto/payments/ledger/v1 --version v1.0.0 --lifecycle stable

# Minor update
apx publish proto/payments/ledger/v1 --version v1.1.0 --lifecycle stable

# Breaking change (new API line)
apx publish proto/payments/ledger/v2 --version v2.0.0-alpha.1 --lifecycle experimental
```

### Legacy Publish

```bash
apx publish --module-path <path> --canonical-repo <url> --version <semver>
```

### Flags

| Flag | Type | Description |
|------|------|-------------|
| `--version` | string | SemVer version to publish (required) |
| `--lifecycle` | string | Lifecycle state: experimental, beta, stable, deprecated, sunset |
| `--canonical-repo` | string | Canonical repo URL (auto-derived from config) |
| `--module-path` | string | Module path (legacy mode) |
| `--dry-run` | bool | Preview without publishing |
| `--create-pr` | bool | Create PR instead of pushing |

### What Happens During Publish

1. API ID is parsed into format, domain, name, and line
2. Source path is derived from the API ID
3. Go module and import paths are computed
4. A subdirectory-scoped git tag is created
5. The module is pushed to the canonical repository

### Tag Format

Tags follow the pattern `<api-id>/v<semver>`:

```
proto/payments/ledger/v1/v1.0.0-beta.1
proto/payments/ledger/v1/v1.0.0
proto/payments/ledger/v2/v2.0.0
```
