# Versioning Strategy

APX uses a three-layer versioning model that keeps API identity, release versioning, and lifecycle metadata cleanly separated.  Understanding these three layers is essential for publishing, consuming, and governing APIs across your organization.

## The Three Layers

### 1. API Line (Compatibility Namespace)

The API line (`v0`, `v1`, `v2`, …) appears in the API ID and determines the compatibility scope:

```
proto/payments/ledger/v0    ← initial development, breaking changes expected
proto/payments/ledger/v1    ← all backward-compatible changes within v1
proto/payments/ledger/v2    ← breaking change from v1, new namespace
```

**Rules:**
- Only backward-incompatible changes create a new API line for stable APIs.
- `v0` is a special line for initial development — breaking changes are allowed at any time (see [v0 Policy](#v0-initial-development) below).

### 2. Release Version (SemVer)

Each API line can have multiple SemVer releases:

```
v1.0.0-alpha.1   ← early exploratory build
v1.0.0-beta.1    ← feature-complete preview
v1.0.0-rc.1      ← release candidate
v1.0.0           ← general availability (GA)
v1.1.0           ← additive change
v1.1.1           ← patch fix
```

Pre-release identifiers (`-alpha`, `-beta`, `-rc`) go in the version tag, **not** in the import path.

### 3. Lifecycle (Support Signal)

The lifecycle field communicates maturity and support level **independently** from the version number:

| State | Meaning | Compatibility | Production use |
|-------|---------|---------------|----------------|
| `experimental` | Early exploration, API still forming | No guarantee | Not recommended |
| `preview` | API surface is stabilizing | Stabilizing — minor breaking changes possible | Use with caution |
| `stable` | Production-ready, fully supported | Full backward compatibility within the line | Recommended |
| `deprecated` | Superseded, maintained for existing users | Maintained — no new features | Migrate away |
| `sunset` | End of life, no further releases | End of life | Do not use |

:::{note}
`beta` is accepted as a backward-compatible alias for `preview`.  New projects should prefer `preview`.
:::

### Why separate layers?

Each layer answers a different question:

| Question | Layer | Example |
|----------|-------|---------|
| What compatibility scope am I in? | **API line** | `v0`, `v1`, `v2` |
| Which build am I running? | **Release version** | `1.0.0-beta.1`, `1.2.3` |
| Is this safe to depend on in production? | **Lifecycle** | `experimental`, `preview`, `stable` |

Without separating these, teams overload version strings to convey lifecycle meaning — which is fragile and confusing.

## v0: Initial Development

APX supports `v0` API lines for APIs that are still taking shape.  SemVer defines major version zero as initial development where anything may change at any time.

### v0 Policy

| Rule | Detail |
|------|--------|
| Allowed lifecycles | `experimental` or `preview` only |
| Breaking changes | Allowed — bumps the minor version |
| Production use | Not recommended |
| Promotion to stable | Not permitted on v0; graduate to `v1` instead |

### v0 Version Examples

```
0.1.0   ← first preview
0.2.0   ← breaking change (minor bump)
0.2.1   ← patch fix
0.3.0   ← another breaking change
```

When the API is ready for production, create the `v1` line and publish `v1.0.0` with lifecycle `stable`.

## Two Preview Workflows

APX supports two official workflows for pre-production APIs:

### Workflow 1 — Rolling Preview (v0 + experimental)

For APIs that are still being shaped and may break frequently:

```bash
apx release prepare proto/payments/ledger/v0 \
  --version 0.4.0 --lifecycle experimental
```

Use this when:
- The API is under active design iteration
- Others may observe or consume it, but everyone understands it can break
- You are not yet committed to a stable contract

### Workflow 2 — Prerelease on Upcoming Stable Line (v1 + preview)

For APIs approaching GA that need integration testing:

```bash
apx release prepare proto/payments/ledger/v1 \
  --version 1.0.0-alpha.1 --lifecycle preview

apx release prepare proto/payments/ledger/v1 \
  --version 1.0.0-beta.1  --lifecycle preview

apx release prepare proto/payments/ledger/v1 \
  --version 1.0.0-rc.1    --lifecycle preview

apx release prepare proto/payments/ledger/v1 \
  --version 1.0.0          --lifecycle stable
```

Use this when:
- You want preview users to test the actual `v1` contract before GA
- The API is mostly defined and you are converging toward a release
- Pre-release labels (`alpha`, `beta`, `rc`) communicate release phase

## Compatibility Promise

APX derives a **compatibility promise** from the API line and lifecycle:

| API Line | Lifecycle | Compatibility Level | Promise |
|----------|-----------|-------------------|---------|
| `v0` | `experimental` | None | No backward-compatibility guarantee; anything may change |
| `v0` | `preview` | None | No backward-compatibility guarantee; breaking changes expected |
| `v1+` | `experimental` | None | No backward-compatibility guarantee |
| `v1+` | `preview` | Stabilizing | API surface is stabilizing; minor breaking changes possible |
| `v1+` | `stable` | Full | Full backward compatibility within the major version line |
| any | `deprecated` | Maintenance | Bug fixes only; no new features; migrate to successor |
| any | `sunset` | End of life | No further releases |

This promise is shown when you run `apx show`:

```bash
$ apx show proto/payments/ledger/v1
# ...
Compatibility:
  Level:    full
  Promise:  full backward compatibility within the major version line
  Breaking: backward-incompatible changes are blocked on this line
  Use:      recommended for production
```

## Why This Matters

### Stable Import Paths

Consumers use the same import path from alpha through GA:

```go
import ledgerv1 "github.com/acme/apis/proto/payments/ledger/v1"
```

Only the `go get` version changes:

```bash
go get github.com/acme/apis/proto/payments/ledger@v1.0.0-beta.1  # during preview
go get github.com/acme/apis/proto/payments/ledger@v1.0.0          # at GA
```

### No Import Rewrites

Because alpha/beta is in the **version**, not the **path**, moving from pre-release to GA requires zero code changes — just a version bump in `go.mod`.

## Go Module Versioning

APX follows Go's major version suffix convention:

| API Line | Go Module Path | Go Import Path |
|----------|---------------|----------------|
| `v0` | `github.com/<org>/apis/<format>/<domain>/<name>` | `github.com/<org>/apis/<format>/<domain>/<name>/v0` |
| `v1` | `github.com/<org>/apis/<format>/<domain>/<name>` | `github.com/<org>/apis/<format>/<domain>/<name>/v1` |
| `v2` | `github.com/<org>/apis/<format>/<domain>/<name>/v2` | `github.com/<org>/apis/<format>/<domain>/<name>/v2` |

For v0 and v1, the module path has **no** version suffix (Go convention). For v2+, both module and import paths include the `/vN` suffix.

## Inspecting Versions

```bash
# See the full identity and version info
apx inspect release proto/payments/ledger/v1@v1.0.0-beta.1

# Understand Go path derivation
apx explain go-path proto/payments/ledger/v1

# See lifecycle and compatibility details
apx show proto/payments/ledger/v1
```
