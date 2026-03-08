# Tagging Strategy

APX uses **subdirectory-scoped git tags** to version individual API lines independently within a single canonical repository.

## Tag Format

Tags follow the pattern:

```
<api-id>/v<semver>
```

Examples:

| API ID | Version | Git Tag |
|--------|---------|---------|
| `proto/payments/ledger/v1` | `v1.0.0-alpha.1` | `proto/payments/ledger/v1/v1.0.0-alpha.1` |
| `proto/payments/ledger/v1` | `v1.0.0-beta.1` | `proto/payments/ledger/v1/v1.0.0-beta.1` |
| `proto/payments/ledger/v1` | `v1.0.0` | `proto/payments/ledger/v1/v1.0.0` |
| `proto/payments/ledger/v2` | `v2.0.0-alpha.1` | `proto/payments/ledger/v2/v2.0.0-alpha.1` |

## API Line vs Release Version

These are different concepts:

- **API line** (`v1`, `v2`) = compatibility namespace. Only changes on breaking changes.
- **Release version** (`v1.0.0-beta.1`, `v1.1.0`) = SemVer release of that line.

### What stays stable

The import path does **not** change between pre-release and GA:

```go
// This import is the same whether you're on alpha, beta, or GA
import ledgerv1 "github.com/acme/apis/proto/payments/ledger/v1"
```

Only the resolved module version changes:

```bash
# During beta
go get github.com/acme/apis/proto/payments/ledger@v1.0.0-beta.1

# At GA
go get github.com/acme/apis/proto/payments/ledger@v1.0.0
```

### What triggers a new API line

Only **breaking changes** create a new API line (`v1` → `v2`). This means a new directory, new import path, and consumers must explicitly opt in:

```go
import ledgerv2 "github.com/acme/apis/proto/payments/ledger/v2"
```

## Lifecycle Metadata

Lifecycle state is tracked separately from the version tag:

| Lifecycle | Description |
|-----------|-------------|
| `experimental` | Early exploration, no compatibility guarantees |
| `preview` | API surface is stabilizing; minor breaking changes still possible |
| `stable` | Production-ready, backward-compatible within the API line |
| `deprecated` | Superseded by a newer line, still supported |
| `sunset` | End-of-life, will be removed |

:::{note}
`beta` is accepted as a backward-compatible alias for `preview`. New projects should use `preview`.
:::

## Inspect Tags

Use `apx inspect` to see what tags would be created:

```bash
apx inspect release proto/payments/ledger/v1@v1.0.0-beta.1
```
