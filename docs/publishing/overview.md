# Publishing Overview

APX's publishing model is built on a **canonical identity** system that separates three concerns:

1. **API identity** — what contract you are talking about (`proto/payments/ledger/v1`)
2. **Artifact version** — which published build you want (`v1.0.0-beta.1`, `v1.2.3`)
3. **Lifecycle state** — how much confidence/support it has (`experimental`, `preview`, `stable`, `deprecated`, `sunset`)

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
| `preview` | Stabilizing — breaking changes still possible |
| `stable` | Production-ready — full backward compatibility |
| `deprecated` | Superseded — maintained for existing users |
| `sunset` | End of life — no further releases |

:::{note}
`beta` is accepted as a backward-compatible alias for `preview`.  New projects should use `preview`.
:::

See [Lifecycle Reference](lifecycle.md) for detailed lifecycle rules, the compatibility promise model, and enforcement policies.

## Publishing Flow

When you run `apx publish`, APX:

1. Reads or parses the API ID
2. Derives the canonical source path
3. Derives language-specific coordinates (Go module/import paths)
4. Validates `go_package` and module path consistency
5. Enforces lifecycle policy rules (v0 line restrictions, lifecycle-version compatibility)
6. Publishes the module — either directly or via pull request:
   - **Direct push** (default): subtree split + push to `main` on the canonical repo
   - **PR-based** (`--create-pr`): shallow-clone canonical, copy files to a feature branch, push, and open a PR via the `gh` CLI
7. Records lifecycle, compatibility, and version information

```bash
# Publish a preview release on an upcoming stable line
apx publish proto/payments/ledger/v1 --version v1.0.0-beta.1 --lifecycle preview

# Publish GA
apx publish proto/payments/ledger/v1 --version v1.0.0 --lifecycle stable

# Publish via PR (recommended for teams with branch protection)
apx publish proto/payments/ledger/v1 --version v1.0.0 --lifecycle stable --create-pr

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

See [Tagging Strategy](tagging-strategy.md) for details on how tags are constructed.

See [Publish Command](publish-command.md) for CLI usage details.

See [Lifecycle Reference](lifecycle.md) for lifecycle state definitions and policy enforcement.
