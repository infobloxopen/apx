# Publishing Commands

APX offers two publishing paths:

- **`apx publish`** — a single fire-and-forget command for quick iterations and local development.
- **`apx release`** — a [multi-step pipeline](release-commands.md) with validation gates, manifest persistence, and immutable audit records. Use this for CI and production releases.

---

## `apx publish`

Publish an API module to the canonical repository in a single step.

### Identity-Based Publish (Recommended)

```bash
apx publish <api-id> --version <semver> [--lifecycle <state>]
```

Examples:

```bash
# Alpha release
apx publish proto/payments/ledger/v1 --version v1.0.0-alpha.1 --lifecycle experimental

# Preview release (beta prerelease tag, preview lifecycle)
apx publish proto/payments/ledger/v1 --version v1.0.0-beta.1 --lifecycle preview

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
| `--lifecycle` | string | Lifecycle state: experimental, preview, stable, deprecated, sunset (`beta` accepted as alias for `preview`) |
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

## Next Steps

For production CI pipelines, see [Release Commands](release-commands.md) — the
multi-step alternative that adds manifest tracking, idempotency, catalog updates,
and immutable release records.
