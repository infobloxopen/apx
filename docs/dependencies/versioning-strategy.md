# Versioning Strategy

APX uses a three-layer versioning model that keeps API identity, release versioning, and lifecycle metadata cleanly separated.

## The Three Layers

### 1. API Line (Compatibility Namespace)

The API line (`v1`, `v2`) appears in the API ID and determines the compatibility scope:

```
proto/payments/ledger/v1    ← all backward-compatible changes
proto/payments/ledger/v2    ← breaking change, new namespace
```

**Rule:** Only breaking changes create a new API line.

### 2. Release Version (SemVer)

Each API line can have multiple SemVer releases:

```
v1.0.0-alpha.1   ← first exploratory build
v1.0.0-beta.1    ← feature-complete preview
v1.0.0           ← general availability
v1.1.0           ← additive change
v1.1.1           ← patch fix
```

Pre-release identifiers (`-alpha`, `-beta`, `-rc`) go in the version tag, **not** in the import path.

### 3. Lifecycle (Support Signal)

The lifecycle field communicates maturity independently:

| State | Meaning |
|-------|---------|
| `experimental` | Early exploration, no guarantees |
| `beta` | Feature-complete, may change |
| `stable` | Production-ready |
| `deprecated` | Superseded, still supported |
| `sunset` | End-of-life |

## Why This Matters

### Stable Import Paths

Consumers use the same import path from alpha through GA:

```go
import ledgerv1 "github.com/acme/apis/proto/payments/ledger/v1"
```

Only the `go get` version changes:

```bash
go get github.com/acme/apis/proto/payments/ledger@v1.0.0-beta.1  # during beta
go get github.com/acme/apis/proto/payments/ledger@v1.0.0          # at GA
```

### No Import Rewrites

Because alpha/beta is in the **version**, not the **path**, moving from pre-release to GA requires zero code changes — just a version bump in `go.mod`.

## Go Module Versioning

APX follows Go's major version suffix convention:

| API Line | Go Module Path | Go Import Path |
|----------|---------------|----------------|
| `v1` | `github.com/<org>/apis/<format>/<domain>/<name>` | `github.com/<org>/apis/<format>/<domain>/<name>/v1` |
| `v2` | `github.com/<org>/apis/<format>/<domain>/<name>/v2` | `github.com/<org>/apis/<format>/<domain>/<name>/v2` |

For v1, the module path has **no** version suffix (Go convention). For v2+, both module and import paths include the `/vN` suffix.

## Inspecting Versions

```bash
# See the full identity and version info
apx inspect release proto/payments/ledger/v1@v1.0.0-beta.1

# Understand Go path derivation
apx explain go-path proto/payments/ledger/v1
```
