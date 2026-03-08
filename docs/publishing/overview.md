# Publishing Overview

APX's publishing model is built on a **canonical identity** system that separates three concerns:

1. **API identity** — what contract you are talking about
2. **Artifact version** — which published build you want
3. **Lifecycle state** — how much confidence/support it has

## The Identity Model

Every API in APX has a **canonical API ID** with the format:

```
<format>/<domain>/<name>/<line>
```

For example: `proto/payments/ledger/v1`

This identity drives everything:
- The **source path** in the canonical repository
- The **Go module and import paths**
- The **git tags** for releases
- The **catalog entry** for discovery

## Key Principles

### Path = Compatibility

The API line (`v1`, `v2`) appears in the path and determines compatibility scope. Only breaking changes create a new API line and therefore a new import path.

### Tag = Release Version

SemVer release versions (including pre-releases like `-alpha.1`, `-beta.1`) are expressed in **git tags**, not in import paths. This means consumers never rewrite imports between alpha → beta → GA.

### Lifecycle = Support Signal

The `lifecycle` field (`experimental`, `beta`, `stable`, `deprecated`, `sunset`) is metadata that signals maturity and support level independently from the version number.

## Publishing Flow

When you run `apx publish`, APX:

1. Reads or parses the API ID
2. Derives the canonical source path
3. Derives language-specific coordinates (Go module/import paths)
4. Validates `go_package` and module path consistency
5. Creates the subdirectory-scoped git tag
6. Records lifecycle and version information

```bash
# Publish a beta release
apx publish proto/payments/ledger/v1 --version v1.0.0-beta.1 --lifecycle beta

# Publish GA
apx publish proto/payments/ledger/v1 --version v1.0.0 --lifecycle stable
```

## One Canonical Repository

APX uses a single canonical repository (`github.com/<org>/apis`) as the source of truth for all API schemas. This repository:

- Contains all API definitions organized by format and domain
- Hosts generated code alongside schemas
- Uses subdirectory-scoped tags for independent versioning
- Serves as the Go module root for consumers

See [Tagging Strategy](tagging-strategy.md) for details on how tags are constructed.

See [Publish Command](publish-command.md) for CLI usage details.
