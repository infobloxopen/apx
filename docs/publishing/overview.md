# Publishing Overview

APX's publishing model is built on a **canonical identity** system that separates three concerns:

1. **API identity** — what contract you are talking about (`proto/payments/ledger/v1`)
2. **Artifact version** — which published build you want (`v1.0.0-beta.1`, `v1.2.3`)
3. **Lifecycle state** — how much confidence/support it has (`experimental`, `beta`, `stable`, `deprecated`, `sunset`)

## The Identity Model

Every API in APX has a **canonical API ID** with the format:

```
<format>/<domain>/<name>/<line>
```

For example: `proto/payments/ledger/v1` or `proto/payments/ledger/v0`

This identity drives everything:
- The **source path** in the canonical repository
- The **Go module and import paths**
- The **git tags** for releases
- The **catalog entry** for discovery

## Key Principles

### Path = Compatibility

The API line (`v0`, `v1`, `v2`) appears in the path and determines compatibility scope. Only breaking changes create a new API line and therefore a new import path.  `v0` is a special line where breaking changes are expected (see [Versioning Strategy](../dependencies/versioning-strategy.md#v0-initial-development)).

### Tag = Release Version

SemVer release versions (including pre-releases like `-alpha.1`, `-beta.1`, `-rc.1`) are expressed in **git tags**, not in import paths. This means consumers never rewrite imports between alpha → beta → GA.

### Lifecycle = Support Signal

The `lifecycle` field signals maturity and support level independently from the version number:

| Lifecycle | Signal |
|-----------|--------|
| `experimental` | Early exploration — no compatibility guarantee |
| `beta` | Stabilizing — breaking changes still possible |
| `stable` | Production-ready — full backward compatibility |
| `deprecated` | Superseded — maintained for existing users |
| `sunset` | End of life — no further releases |

:::{note}
`preview` is accepted as a backward-compatible alias for `beta`.
:::

See [Lifecycle Reference](lifecycle.md) for detailed lifecycle rules, the compatibility promise model, and enforcement policies.

## Publishing Flow

When you run `apx publish`, APX:

1. Reads or parses the API ID
2. Derives the canonical source path
3. Derives language-specific coordinates (Go module/import paths)
4. Validates `go_package` and module path consistency
5. Enforces lifecycle policy rules (v0 line restrictions, lifecycle-version compatibility)
6. Publishes the module via pull request: clones the canonical repo, copies the snapshot to a release branch, pushes, and opens a PR via the `gh` CLI
7. Records lifecycle, compatibility, and version information

```bash
# Publish a beta release on an upcoming stable line
apx publish proto/payments/ledger/v1 --version v1.0.0-beta.1 --lifecycle beta

# Publish GA
apx publish proto/payments/ledger/v1 --version v1.0.0 --lifecycle stable

# Publish a rolling preview on v0
apx publish proto/payments/ledger/v0 --version 0.3.0 --lifecycle experimental
```

## One Canonical Repository

APX uses a single canonical repository (`github.com/<org>/apis`) as both the source of truth and the default Go distribution root for all API schemas. There is no separate `apis-go` or language-specific distribution repo. This repository:

- Contains all API definitions organized by format and domain
- Hosts generated code alongside schemas
- Uses subdirectory-scoped tags for independent versioning
- Serves as the Go module root for consumers
- Is the sole target of `apx publish`

Release artifacts and tags belong to this one repo. Local overlays (`go.work`) are a development convenience — they do not represent a distinct public distribution identity.

## Two Publishing Paths

APX provides two ways to get an API into the canonical repository:

### Release Pipeline (`apx release`) — Recommended

A multi-step workflow with explicit phases, manifest persistence, idempotency
checks, and immutable audit records.  Best for CI pipelines and production
releases.

```bash
# 1. Validate and create a release manifest
apx release prepare proto/payments/ledger/v1 --version v1.0.0

# 2. Submit as a pull request on the canonical repository
apx release submit

# 3. Tag, update catalog, emit release record (canonical CI)
apx release finalize
```

The release pipeline also supports lifecycle promotions:

```bash
apx release promote proto/payments/ledger/v1 --to stable --version v1.0.0
apx release submit
```

See [Release Commands](../cli-reference/release-commands.md) for full usage.

### Quick Publish (`apx publish`)

A single fire-and-forget command that validates, pushes a snapshot branch, and
opens a PR.  Best for local development and quick iterations.

```bash
apx publish proto/payments/ledger/v1 --version v1.0.0 --lifecycle stable
```

Quick publish does **not** write a manifest, update the catalog, or emit a
release record.  Tagging happens after merge in canonical CI.

See [Publish Command](publish-command.md) for full usage.

See [Tagging Strategy](tagging-strategy.md) for details on how tags are constructed.

## Publishing by Lifecycle Stage

Lifecycle and version work together: the lifecycle declares the maturity signal while
the SemVer prerelease tag encodes the release phase.  APX enforces their consistency
automatically.

| Lifecycle | Required version tag | Suggested by `apx semver suggest` |
|-----------|---------------------|------------------------------------|
| `experimental` | `-alpha.*` | `-alpha.N` |
| `beta` | `-alpha.*`, `-beta.*`, or `-rc.*` | `-beta.N` |
| `stable` | *(no prerelease)* | clean semver (e.g. `1.0.0`) |
| `deprecated` | any | *(caller warned)* |
| `sunset` | **blocked** | *(releases not allowed)* |

:::{note}
`preview` is accepted as a backward-compatible alias for `beta`.
:::

### Experimental — early exploration

Publish under `experimental` when the API is still forming.  No compatibility
guarantee; anything may change.

```bash
apx publish proto/payments/ledger/v1 --version v1.0.0-alpha.1 --lifecycle experimental

# Or with the release pipeline
apx release prepare proto/payments/ledger/v1 --version v1.0.0-alpha.1 --lifecycle experimental
apx release submit
```

For APIs that will break frequently, use a `v0` line instead:

```bash
apx publish proto/payments/ledger/v0 --version v0.1.0 --lifecycle experimental
```

### Beta — stabilizing toward GA

Publish under `beta` when the API design is mostly settled but still
converging.  Consumers can start integrating, but minor breaking changes remain
possible.

```bash
# Beta release
apx publish proto/payments/ledger/v1 --version v1.0.0-beta.1 --lifecycle beta

# Release candidate
apx publish proto/payments/ledger/v1 --version v1.0.0-rc.1 --lifecycle beta
```

### Stable — production-ready (GA)

Publish under `stable` for general availability.  Full backward compatibility
within the API line.  Version must not have a prerelease tag.

```bash
apx publish proto/payments/ledger/v1 --version v1.0.0 --lifecycle stable

# Or promote from beta to stable
apx release promote proto/payments/ledger/v1 --to stable --version v1.0.0
apx release submit
```

### Deprecated — superseded

Mark an API as `deprecated` when a successor exists.  Maintenance continues, but
no new features.  APX prints a warning on every publish.

```bash
apx publish proto/payments/ledger/v1 --version v1.2.1 --lifecycle deprecated
```

### Sunset — end of life

An API in `sunset` blocks all new releases by default.  This signals that
consumers must migrate.

```bash
# This will fail:
apx publish proto/payments/ledger/v1 --version v1.2.2 --lifecycle sunset
# Error: lifecycle "sunset" blocks new releases; use --force to override
```

### The full progression

A typical API moves through these stages over its lifetime:

```bash
# 1. Experimental — early exploration
apx publish proto/payments/ledger/v1 --version v1.0.0-alpha.1 --lifecycle experimental

# 2. Beta — stabilizing, beta out to early adopters
apx publish proto/payments/ledger/v1 --version v1.0.0-beta.1  --lifecycle beta

# 3. Stable — GA
apx publish proto/payments/ledger/v1 --version v1.0.0          --lifecycle stable

# 4. Stable updates
apx publish proto/payments/ledger/v1 --version v1.1.0          --lifecycle stable

# 5. Deprecated — new line exists
apx release promote proto/payments/ledger/v1 --to deprecated

# 6. Sunset — end of life
apx release promote proto/payments/ledger/v1 --to sunset
```

Throughout this entire progression, consumers use the same import path:

```go
import ledgerv1 "github.com/acme/apis/proto/payments/ledger/v1"
```

Only the resolved module version changes.

See [Lifecycle Reference](lifecycle.md) for the full lifecycle model, transition
rules, v0 line policy, and compatibility promise tables.
