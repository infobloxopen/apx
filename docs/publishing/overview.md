# Releasing Overview

APX's release model is built on a **canonical identity** system that separates three concerns:

1. **API identity** — what contract you are talking about (`proto/payments/ledger/v1`)
2. **Artifact version** — which released build you want (`v1.0.0-beta.1`, `v1.2.3`)
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

## Release Flow

When you run `apx release prepare`, APX:

1. Reads or parses the API ID
2. Derives the canonical source path
3. Derives language-specific coordinates (Go module/import paths)
4. Validates `go_package` and module path consistency
5. Enforces lifecycle policy rules (v0 line restrictions, lifecycle-version compatibility)
6. Writes a release manifest (`.apx-release.yaml`)

Then `apx release submit` releases the module via pull request: clones the canonical repo, copies the snapshot to a release branch, pushes, and opens a PR via the `gh` CLI. Finally, `apx release finalize` records lifecycle, compatibility, and version information.

```bash
# Release a beta on an upcoming stable line
apx release prepare proto/payments/ledger/v1 --version v1.0.0-beta.1 --lifecycle beta
apx release submit

# Release GA
apx release prepare proto/payments/ledger/v1 --version v1.0.0 --lifecycle stable
apx release submit

# Release a rolling preview on v0
apx release prepare proto/payments/ledger/v0 --version 0.3.0 --lifecycle experimental
apx release submit
```

## One Canonical Repository

APX uses a single canonical repository (`github.com/<org>/apis`) as both the source of truth and the default Go distribution root for all API schemas. There is no separate `apis-go` or language-specific distribution repo. This repository:

- Contains all API definitions organized by format and domain
- Hosts generated code alongside schemas
- Uses subdirectory-scoped tags for independent versioning
- Serves as the Go module root for consumers
- Is the sole target of `apx release submit`

Release artifacts and tags belong to this one repo. Local overlays (`go.work`) are a development convenience — they do not represent a distinct public distribution identity.

## Responsibility Boundary

APX handles schema lifecycle management end-to-end. Language-specific package
builds are the responsibility of CI plugins or workflow steps that teams
configure outside APX.

| Step | Who | What |
|------|-----|------|
| Schema validation | APX | `lint`, `breaking`, `policy check` |
| Identity & coordinates | APX | API ID → paths, Go module, tag pattern |
| PR to canonical | APX | `release submit` |
| Tag creation | APX (finalize) | Annotated subdirectory git tag |
| Catalog update | APX (finalize) | `catalog.yaml` entry |
| Go module availability | Git + Go proxy | Automatic once the tag exists |
| Maven JARs | External CI | Team-configured workflow step |
| Python wheels | External CI | Team-configured workflow step |
| npm packages | External CI | Team-configured workflow step |
| OCI bundles | External CI | Team-configured workflow step |

The `--skip-packages` flag on `release finalize` controls whether Go module
artifact metadata is _recorded_ in the release record — it does not build or
publish packages to any registry.

## The Release Pipeline

APX provides one path to get an API into the canonical repository: the
`apx release` pipeline.

A multi-step workflow with explicit phases, manifest persistence, idempotency
checks, and immutable audit records.

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

See [Tagging Strategy](tagging-strategy.md) for details on how tags are constructed.

## Releasing by Lifecycle Stage

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

Release under `experimental` when the API is still forming.  No compatibility
guarantee; anything may change.

```bash
apx release prepare proto/payments/ledger/v1 --version v1.0.0-alpha.1 --lifecycle experimental
apx release submit
```

For APIs that will break frequently, use a `v0` line instead:

```bash
apx release prepare proto/payments/ledger/v0 --version v0.1.0 --lifecycle experimental
apx release submit
```

### Beta — stabilizing toward GA

Release under `beta` when the API design is mostly settled but still
converging.  Consumers can start integrating, but minor breaking changes remain
possible.

```bash
# Beta release
apx release prepare proto/payments/ledger/v1 --version v1.0.0-beta.1 --lifecycle beta
apx release submit

# Release candidate
apx release prepare proto/payments/ledger/v1 --version v1.0.0-rc.1 --lifecycle beta
apx release submit
```

### Stable — production-ready (GA)

Release under `stable` for general availability.  Full backward compatibility
within the API line.  Version must not have a prerelease tag.

```bash
apx release prepare proto/payments/ledger/v1 --version v1.0.0 --lifecycle stable
apx release submit

# Or promote from beta to stable
apx release promote proto/payments/ledger/v1 --to stable --version v1.0.0
apx release submit
```

### Deprecated — superseded

Mark an API as `deprecated` when a successor exists.  Maintenance continues, but
no new features.  APX prints a warning on every release.

```bash
apx release prepare proto/payments/ledger/v1 --version v1.2.1 --lifecycle deprecated
apx release submit
```

### Sunset — end of life

An API in `sunset` blocks all new releases by default.  This signals that
consumers must migrate.

```bash
# This will fail:
apx release prepare proto/payments/ledger/v1 --version v1.2.2 --lifecycle sunset
# Error: lifecycle "sunset" blocks new releases; use --force to override
```

### The full progression

A typical API moves through these stages over its lifetime:

```bash
# 1. Experimental — early exploration
apx release prepare proto/payments/ledger/v1 --version v1.0.0-alpha.1 --lifecycle experimental
apx release submit

# 2. Beta — stabilizing, beta out to early adopters
apx release prepare proto/payments/ledger/v1 --version v1.0.0-beta.1  --lifecycle beta
apx release submit

# 3. Stable — GA
apx release prepare proto/payments/ledger/v1 --version v1.0.0          --lifecycle stable
apx release submit

# 4. Stable updates
apx release prepare proto/payments/ledger/v1 --version v1.1.0          --lifecycle stable
apx release submit

# 5. Deprecated — new line exists
apx release promote proto/payments/ledger/v1 --to deprecated
apx release submit

# 6. Sunset — end of life
apx release promote proto/payments/ledger/v1 --to sunset
apx release submit
```

Throughout this entire progression, consumers use the same import path:

```go
import ledgerv1 "github.com/acme/apis/proto/payments/ledger/v1"
```

Only the resolved module version changes.

:::{tip}
If your organization sets `import_root` in `apx.yaml` (e.g. `import_root: go.acme.dev/apis`), the import path above would instead be `go.acme.dev/apis/proto/payments/ledger/v1`. See [Configuration Reference](../cli-reference/configuration.md#import_root).
:::

See [Lifecycle Reference](lifecycle.md) for the full lifecycle model, transition
rules, v0 line policy, and compatibility promise tables.

See [Release Commands](../cli-reference/release-commands.md) for full usage.
