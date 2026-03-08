# API Lifecycle Reference

APX models **lifecycle** as a first-class concept, independent of version numbers and API lines.  The lifecycle tells consumers — and APX's own tooling — the maturity, compatibility promise, and production-readiness of every API.

## Why Lifecycle Matters

Version strings alone do not answer important questions:

| Question | Version string answers? | Lifecycle answers? |
|----------|:-----------------------:|:-----------------:|
| What build am I running? | ✓ | |
| Is this safe to depend on? | | ✓ |
| Can it break at any time? | | ✓ |
| Is it intended for production? | | ✓ |
| Is it deprecated or headed for removal? | | ✓ |

APX separates three signals so each one does exactly one job:

| Signal | Layer | Examples |
|--------|-------|---------|
| **Compatibility scope** | API line | `v0`, `v1`, `v2` |
| **Release phase** | SemVer version | `1.0.0-alpha.1`, `1.0.0-beta.1`, `1.0.0` |
| **Support/stability posture** | Lifecycle | `experimental`, `beta`, `stable` |

## Lifecycle States

### `experimental`

Early exploration.  The API is still forming — its shape, semantics, and scope may change without notice.

- **Compatibility:** no guarantee
- **Breaking changes:** allowed at any time
- **Production use:** not recommended
- **Required versions:** must carry an `-alpha.*` prerelease tag

### `beta`

The API surface is stabilizing.  The design is mostly defined, but minor breaking changes are still possible as the contract converges.

- **Compatibility:** stabilizing — minor breaking changes possible
- **Breaking changes:** may occur between prereleases, with warnings
- **Production use:** use with caution
- **Required versions:** must carry `-alpha.*`, `-beta.*`, or `-rc.*` prerelease tag

:::{note}
`preview` is accepted as a backward-compatible alias for `beta`.  The canonical name is `beta`; APX normalizes `preview` → `beta` internally.
:::

### `stable`

Production-ready.  Full backward compatibility is maintained within the major version line.

- **Compatibility:** full backward compatibility
- **Breaking changes:** blocked on this line
- **Production use:** recommended
- **Required versions:** must **not** have a prerelease tag (e.g. `v1.0.0`, `v1.1.0`)

### `deprecated`

Superseded by a newer API or version line.  The API is still maintained for existing consumers, but no new features will be added.

- **Compatibility:** maintained (bug/security fixes only)
- **Breaking changes:** none — maintenance only
- **Production use:** migrate away; maintenance only
- **Required versions:** any

### `sunset`

End of life.  No further releases will be made.

- **Compatibility:** none — end of life
- **Breaking changes:** N/A — no new releases
- **Production use:** do not use
- **Required versions:** releases are blocked by default

## Lifecycle Transitions

Lifecycle states progress forward and cannot be reversed:

```
experimental → beta → stable → deprecated → sunset
```

APX enforces this ordering.  You cannot move a `stable` API back to `beta`, for example.

| From | Allowed targets |
|------|----------------|
| `experimental` | `beta`, `stable`, `deprecated`, `sunset` |
| `beta` | `stable`, `deprecated`, `sunset` |
| `stable` | `deprecated`, `sunset` |
| `deprecated` | `sunset` |
| `sunset` | *(none — terminal state)* |

## v0 Line Policy

API lines starting with `v0` have special rules rooted in SemVer's definition of major version zero: initial development where anything may change at any time.

| Rule | Detail |
|------|--------|
| **Allowed lifecycles** | `experimental` or `beta` only |
| **Stable promotion** | Not allowed — graduate the API to a `v1` line instead |
| **Breaking changes** | Allowed; APX bumps the minor version instead of rejecting |
| **Production use** | Not recommended |

### Why v0 cannot be stable

`v0` communicates "initial development" by SemVer convention.  Declaring `v0` as `stable` would be contradictory: stable implies full backward compatibility, but v0 explicitly allows breaking changes.  When the API is ready for production, create the `v1` line.

## Compatibility Promise

APX derives a **compatibility promise** from the combination of API line and lifecycle.  This promise is shown by `apx show` and recorded in the catalog.

| API Line | Lifecycle | Level | Summary |
|----------|-----------|-------|---------|
| `v0` | `experimental` | none | No backward-compatibility guarantee; anything may change |
| `v0` | `beta` | none | No backward-compatibility guarantee; breaking changes expected |
| `v1+` | `experimental` | none | No backward-compatibility guarantee |
| `v1+` | `beta` | stabilizing | API surface is stabilizing; minor breaking changes possible |
| `v1+` | `stable` | full | Full backward compatibility within the major version line |
| any | `deprecated` | maintenance | Bug fixes only; no new features; migrate to successor |
| any | `sunset` | eol | End of life; no further releases |

### Breaking Change Policy

| API Line | Lifecycle | Policy |
|----------|-----------|--------|
| `v0` | any | Breaking changes allowed (minor version bump) |
| `v1+` | `experimental` | Breaking changes allowed (prerelease scope) |
| `v1+` | `beta` | Breaking changes may occur between prereleases |
| `v1+` | `stable` | Backward-incompatible changes are blocked |
| any | `deprecated` | No changes expected (maintenance only) |
| any | `sunset` | No releases permitted |

## Two Beta Workflows

APX supports two official workflows for pre-production APIs.  Choose the one that matches your situation.

### Workflow 1: Rolling Preview (v0 + experimental)

For APIs that are still being shaped and may break frequently:

```bash
apx release prepare proto/payments/ledger/v0 \
  --version 0.1.0 --lifecycle experimental

apx release prepare proto/payments/ledger/v0 \
  --version 0.2.0 --lifecycle experimental   # breaking change → minor bump

apx release prepare proto/payments/ledger/v0 \
  --version 0.3.0 --lifecycle experimental   # another breaking change
```

**When to use:**
- The API is under active design iteration
- Breaking changes are frequent and expected
- Others may observe or consume it, but everyone understands it can break

**When the API is ready for production:** graduate to `v1`:

```bash
apx release prepare proto/payments/ledger/v1 \
  --version 1.0.0 --lifecycle stable
```

### Workflow 2: Prerelease on Upcoming Stable Line (v1 + beta)

For APIs approaching GA that need integration testing:

```bash
apx release prepare proto/payments/ledger/v1 \
  --version 1.0.0-alpha.1 --lifecycle beta

apx release prepare proto/payments/ledger/v1 \
  --version 1.0.0-beta.1  --lifecycle beta

apx release prepare proto/payments/ledger/v1 \
  --version 1.0.0-rc.1    --lifecycle beta

apx release prepare proto/payments/ledger/v1 \
  --version 1.0.0          --lifecycle stable
```

**When to use:**
- The API is mostly defined and converging toward a release
- You want beta users to test the actual `v1` contract before GA
- Consumers benefit from alpha → beta → rc progression signals

**Key benefit:** consumers never rewrite imports — the import path (`proto/payments/ledger/v1`) stays the same from alpha through GA.

## CLI Examples

### Viewing lifecycle details

```bash
$ apx show proto/payments/ledger/v1
API:        proto/payments/ledger/v1
Format:     proto
Domain:     payments
Name:       ledger
Line:       v1
Lifecycle:  stable
Source:     github.com/acme/apis/proto/payments/ledger/v1
Latest stable:      v1.2.3
Latest prerelease:  v1.3.0-beta.1
Compatibility:
  Level:    full
  Promise:  full backward compatibility within the major version line
  Breaking: backward-incompatible changes are blocked on this line
  Use:      recommended for production
```

### Preparing a release with lifecycle enforcement

```bash
# APX enforces v0 lifecycle policy
$ apx release prepare proto/payments/ledger/v0 \
    --version 0.4.0 --lifecycle stable
Error: v0 line must use lifecycle "experimental" or "beta", got "stable"

# Valid v0 release
$ apx release prepare proto/payments/ledger/v0 \
    --version 0.4.0 --lifecycle experimental
```

### Promoting a release

```bash
# Promote from beta to stable
$ apx release promote proto/payments/ledger/v1 \
    --target-lifecycle stable --version 1.0.0

# v0 cannot be promoted to stable
$ apx release promote proto/payments/ledger/v0 \
    --target-lifecycle stable
Error: v0 line must use lifecycle "experimental" or "beta", got "stable"
```

## See Also

- [Versioning Strategy](../dependencies/versioning-strategy.md) — the three-layer model in detail
- [Publishing Overview](overview.md) — how publishing uses lifecycle
- [Release Guardrails](release-guardrails.md) — policy enforcement during releases
