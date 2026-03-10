# Configuration Reference

The APX configuration file (`apx.yaml`) defines your organization's schema management settings. It controls validation policies, release behavior, tool versions, and code generation targets.

## Overview

Every APX workspace requires an `apx.yaml` at the repository root. The file follows a strict schema — use `apx config validate` to verify correctness and `apx config migrate` to upgrade to newer schema versions.

### Quick Start

```bash
# Initialize a new configuration
apx config init

# Validate an existing configuration
apx config validate

# Validate with JSON output
apx config validate --json

# Migrate to the latest schema version
apx config migrate
```

## Schema Version

The current schema version is **1**. The `version` field is required and must be set to a supported version number.

```yaml
version: 1
```

## Required Fields

| Field     | Type   | Description                    |
|-----------|--------|--------------------------------|
| `version` | integer | Schema version number         |
| `org`     | string  | GitHub organization name (hyphens and mixed case are normalized per-language; see [Org Name Normalization](../concepts/multi-language-strategy.md#organization-name-normalization)) |
| `repo`    | string  | Canonical API repository name |

## Complete Field Reference

<!-- This table is generated from the canonical schema definition in internal/config/schema.go -->

| YAML Path | Type | Required | Default | Allowed Values | Description |
|-----------|------|----------|---------|----------------|-------------|
| `version` | integer | yes |  |  | Schema version number |
| `org` | string | yes |  |  | GitHub organization name |
| `repo` | string | yes |  |  | Canonical API repository name |
| `import_root` | string | no |  |  | Custom public Go import prefix (e.g. `go.acme.dev/apis`). Overrides `source.repo` for Go module/import paths. |
| `site_url` | string | no |  |  | Custom domain for the catalog site (e.g. `apis.internal.infoblox.dev`). Defaults to `{org}.github.io/{repo}`. |
| `catalog_url` | string | no |  |  | Remote catalog URL for dependency discovery. Used by `apx search`, `apx show`, `apx add`, `apx update`, and `apx upgrade` when `--catalog` is not specified. |
| `catalog_registries` | list | no |  |  | OCI catalog registries for API discovery. Each entry maps to ghcr.io/<org>/<repo>-catalog:latest. |
| `catalog_registries[].org` | string | yes |  |  | GitHub organization |
| `catalog_registries[].repo` | string | yes |  |  | Canonical API repository name |
| `module_roots` | list | no | `[proto]` |  | Directories containing schema modules |
| `language_targets` | map | no |  |  | Code generation targets keyed by language |
| `language_targets.<key>` | struct |  |  |  | Code generation target for a language |
| `language_targets.<key>.enabled` | boolean | no | `false` |  | Whether this language target is active |
| `language_targets.<key>.tool` | string | no |  |  | Tool name (e.g., grpcio-tools) |
| `language_targets.<key>.version` | string | no |  |  | Tool version |
| `language_targets.<key>.plugins` | list | no |  |  | List of plugin name/version maps |
| `policy` | struct | no |  |  | Validation policy settings |
| `policy.forbidden_proto_options` | list | no |  |  | Regex patterns for forbidden proto options |
| `policy.allowed_proto_plugins` | list | no |  |  | Allowed protoc plugin names |
| `policy.openapi` | struct | no |  |  | OpenAPI-specific policy |
| `policy.openapi.spectral_ruleset` | string | no |  |  | Path to Spectral ruleset file |
| `policy.avro` | struct | no |  |  | Avro-specific policy |
| `policy.avro.compatibility` | string | no | `BACKWARD` | BACKWARD, FORWARD, FULL, NONE | Avro compatibility mode |
| `policy.jsonschema` | struct | no |  |  | JSON Schema policy |
| `policy.jsonschema.breaking_mode` | string | no | `strict` | strict, lenient | Breaking change detection mode |
| `policy.parquet` | struct | no |  |  | Parquet policy |
| `policy.parquet.allow_additive_nullable_only` | boolean | no | `true` |  | Whether to restrict to additive nullable columns |
| `release` | struct | no |  |  | Release configuration |
| `release.tag_format` | string | no | `{subdir}/v{version}` |  | Tag pattern; must contain {version} |
| `release.ci_only` | boolean | no | `true` |  | Restrict releasing to CI environments |
| `tools` | struct | no |  |  | Pinned tool versions |
| `tools.buf` | struct | no |  |  | Buf CLI settings |
| `tools.buf.version` | string | no |  |  | Buf CLI version |
| `tools.oasdiff` | struct | no |  |  | oasdiff settings |
| `tools.oasdiff.version` | string | no |  |  | oasdiff version |
| `tools.spectral` | struct | no |  |  | Spectral settings |
| `tools.spectral.version` | string | no |  |  | Spectral version |
| `tools.avrotool` | struct | no |  |  | Avro tools settings |
| `tools.avrotool.version` | string | no |  |  | Avro tools version |
| `tools.jsonschemadiff` | struct | no |  |  | JSON Schema diff settings |
| `tools.jsonschemadiff.version` | string | no |  |  | JSON Schema diff version |
| `execution` | struct | no |  |  | Execution environment settings |
| `execution.mode` | string | no | `local` | local, container | Where tools run |
| `execution.container_image` | string | no |  |  | Container image when mode=container |
| `external_apis` | list | no |  |  | External API registrations |
| `external_apis[].id` | string | yes |  |  | Canonical API identity (format/domain/name/line) |
| `external_apis[].managed_repo` | string | yes |  |  | Internal repository hosting curated snapshots |
| `external_apis[].managed_path` | string | yes |  |  | Filesystem path in managed repository |
| `external_apis[].upstream_repo` | string | yes |  |  | Original external repository URL |
| `external_apis[].upstream_path` | string | yes |  |  | Path in upstream repository |
| `external_apis[].import_mode` | string | no | `preserve` | preserve, rewrite | Import path handling strategy |
| `external_apis[].origin` | string | no | `external` | external, forked | Classification of the external API |
| `external_apis[].description` | string | no |  |  | Human-readable description |
| `external_apis[].lifecycle` | string | no |  | experimental, beta, stable, deprecated, sunset | Lifecycle state |
| `external_apis[].version` | string | no |  |  | Current version of the managed snapshot |
| `external_apis[].owners` | list | no |  |  | List of owners |
| `external_apis[].tags` | list | no |  |  | List of tags |
| `api` | struct | no |  |  | Canonical API identity |
| `api.id` | string | no |  |  | Full API identifier (format/domain/name/line) |
| `api.format` | string | no |  | proto, openapi, avro, jsonschema, parquet | Schema format |
| `api.domain` | string | no |  |  | Business domain for the API |
| `api.name` | string | no |  |  | API name within the domain |
| `api.line` | string | no |  |  | API compatibility line (e.g. v1, v2) |
| `api.lifecycle` | string | no |  | experimental, beta, stable, deprecated, sunset | Maturity/support state of this API line (`preview` accepted as alias for `beta`) |
| `source` | struct | no |  |  | Canonical source repository identity |
| `source.repo` | string | no |  |  | Canonical source repository (e.g. github.com/acme/apis) |
| `source.path` | string | no |  |  | Path within the canonical repo (derived from api.id) |
| `releases` | struct | no |  |  | Release version tracking |
| `releases.current` | string | no |  |  | Current release version (SemVer) |
| `languages` | map | no |  |  | Derived language-specific coordinates keyed by language |
| `languages.<key>` | struct |  |  |  | Language-specific module and import paths |
| `languages.<key>.module` | string | no |  |  | Module/package path for the language |
| `languages.<key>.import` | string | no |  |  | Import path for the language |

## Section Details

### `import_root`

Decouples the public Go import path from your Git hosting URL. When set, all derived Go module and import paths use this root instead of `source.repo`.

```yaml
import_root: go.acme.dev/apis
```

**Without `import_root`** (default):
```
Go module: github.com/acme/apis/proto/payments/ledger
Go import: github.com/acme/apis/proto/payments/ledger/v1
```

**With `import_root: go.acme.dev/apis`**:
```
Go module: go.acme.dev/apis/proto/payments/ledger
Go import: go.acme.dev/apis/proto/payments/ledger/v1
```

The `source.repo` field still reflects the actual Git repository. Only the public import identity changes.

**Use cases:**
- Vanity domain imports (`go.company.dev/apis`)
- Migration-proof paths — change Git hosts without breaking consumer imports
- Custom module registries (`buf.build/gen/go/acme/apis`)

**Affected commands:** `inspect identity`, `inspect release`, `explain go-path`, `show`, `release prepare`, `release promote`.

!!! tip "Auto-detection"
    When running `apx init app`, APX automatically detects `import_root` from the canonical repo's `apx.yaml` (or from the cached catalog) and pre-fills it as the default. You can accept or override the detected value.

### `catalog_url`

Points `apx search`, `apx show`, `apx add`, `apx update`, and `apx upgrade` at your organization's canonical catalog without passing `--catalog` on every command. Accepts a local file path or an `https://` URL.

```yaml
catalog_url: https://raw.githubusercontent.com/acme/apis/main/catalog/catalog.yaml
```

**Resolution order** (same as the unified catalog resolution below):

1. `--catalog` flag (if provided)
2. `catalog_registries` from `apx.yaml` (if configured)
3. Auto-discover from `org` (query GHCR for `*-catalog` packages)
4. `catalog_url` from `apx.yaml`
5. Local `catalog/catalog.yaml`

**Use cases:**
- App repos that don't clone the canonical repo still get `apx search`/`apx show` working
- CI pipelines that pin a specific catalog revision via URL
- Teams on a fork pointing at the upstream catalog

See [Dependency Discovery](../dependencies/discovery.md) for full examples.

### `catalog_registries`

Lists OCI-based catalog registries for API discovery. Each entry identifies a GHCR-hosted catalog container derived from a canonical repository. APX pulls catalog data from these containers, caches it locally, and aggregates results from multiple registries.

```yaml
catalog_registries:
  - org: acme
    repo: apis            # → ghcr.io/acme/apis-catalog:latest
  - org: acme
    repo: shared-schemas  # → ghcr.io/acme/shared-schemas-catalog:latest
  - org: partner-co
    repo: public-apis     # cross-org discovery
```

**Resolution order** for catalog commands:

1. `--catalog` flag (if provided)
2. `catalog_registries` from `apx.yaml` (this field)
3. Auto-discover from `org` (query GHCR for `*-catalog` packages)
4. `catalog_url` from `apx.yaml`
5. Local `catalog/catalog.yaml`

**Use cases:**
- Aggregate API catalogs from multiple canonical repos in a single `apx search`
- Cross-org discovery when depending on partner APIs
- Offline resilience via local cache with automatic refresh

### `module_roots`

Lists the directories that contain schema modules. Each entry is a relative path from the repository root.

```yaml
module_roots:
  - proto
  - openapi
  - avro
  - jsonschema
  - parquet
```

### `language_targets`

Configures code generation for each target language. Each key is a language name with settings for plugins and tools.

```yaml
language_targets:
  go:
    enabled: true
    plugins:
      - name: protoc-gen-go
        version: v1.64.0
      - name: protoc-gen-go-grpc
        version: v1.5.0
  python:
    enabled: true
    tool: grpcio-tools
    version: "1.64.0"
```

### `policy`

Controls validation rules for schema files across all supported formats.

```yaml
policy:
  forbidden_proto_options:
    - "^gorm\\."
  allowed_proto_plugins:
    - protoc-gen-go
    - protoc-gen-go-grpc
  openapi:
    spectral_ruleset: ".spectral.yaml"
  avro:
    compatibility: "BACKWARD"
  jsonschema:
    breaking_mode: "strict"
  parquet:
    allow_additive_nullable_only: true
```

### `release`

Controls how schema versions are tagged and released.

```yaml
release:
  tag_format: "{subdir}/v{version}"
  ci_only: true
```

The `tag_format` must contain `{version}`. The `ci_only` flag restricts releasing to CI environments (recommended for production use).

### `tools`

Pins specific versions of external tools used by APX.

```yaml
tools:
  buf:
    version: v1.45.0
  oasdiff:
    version: v1.9.6
  spectral:
    version: v6.11.0
  avrotool:
    version: "1.11.3"
  jsonschemadiff:
    version: "0.3.0"
```

### `execution`

Controls where tools run.

```yaml
execution:
  mode: "local"
  container_image: ""
```

| Mode | Description |
|------|-------------|
| `local` | Run tools directly on the host machine |
| `container` | Run tools inside a container (requires `container_image`) |

### `external_apis`

Registers third-party APIs for inclusion in the catalog and dependency system. See [External API Registration](../dependencies/external-apis.md) for full workflow documentation.

```yaml
external_apis:
  - id: proto/google/pubsub/v1
    managed_repo: github.com/acme/apis-contrib-google
    managed_path: google/pubsub/v1
    upstream_repo: github.com/googleapis/googleapis
    upstream_path: google/pubsub/v1
    import_mode: preserve
    origin: external
    description: Google Cloud Pub/Sub API
    lifecycle: stable
    version: v1.0.0
```

Each entry requires `id`, `managed_repo`, `managed_path`, `upstream_repo`, and `upstream_path`. Optional fields include `import_mode` (default: `preserve`), `origin` (default: `external`), `description`, `lifecycle`, `version`, `owners`, and `tags`.

### `api`

Defines the canonical identity of an API. The API ID uses a four-part format: `<format>/<domain>/<name>/<line>`.

```yaml
api:
  id: proto/payments/ledger/v1
  format: proto
  domain: payments
  name: ledger
  line: v1
  lifecycle: beta
```

The `line` field represents the API compatibility line (`v1`, `v2`, etc.). Only breaking changes create a new line. The `lifecycle` field tracks the maturity state independently from the release version.

| Lifecycle | Meaning |
|-----------|--------|
| `experimental` | Early exploration, no compatibility guarantees |
| `beta` | API surface is stabilizing; minor breaking changes still possible |
| `stable` | Production-ready, backward-compatible within the API line |
| `deprecated` | Superseded by a newer line, still supported |
| `sunset` | End-of-life, will be removed |

!!! note
    `preview` is accepted as a backward-compatible alias for `beta`.

### `source`

Identifies where the canonical source lives.

```yaml
source:
  repo: github.com/acme/apis
  path: proto/payments/ledger/v1
```

The `path` is typically identical to the `api.id` and represents the directory within the canonical repository.

### `releases`

Tracks the current release version for this API line.

```yaml
releases:
  current: v1.0.0-beta.1
```

Release versions follow SemVer. Alpha/beta status is expressed in the version string (e.g. `v1.0.0-alpha.1`, `v1.0.0-beta.1`), not in the import path. This ensures consumers never need to rewrite imports between pre-release and GA.

### `languages`

Derived language-specific coordinates. APX computes these automatically from the API identity and source repository.

```yaml
languages:
  go:
    module: github.com/acme/apis/proto/payments/ledger
    import: github.com/acme/apis/proto/payments/ledger/v1
```

**Go module path rules:**

| API Line | Module Path | Import Path |
|----------|-------------|-------------|
| `v1` | `<repo>/<format>/<domain>/<name>` (no suffix) | `<repo>/<format>/<domain>/<name>/v1` |
| `v2+` | `<repo>/<format>/<domain>/<name>/v<N>` | `<repo>/<format>/<domain>/<name>/v<N>` |

This follows Go's major version suffix convention: v1 modules have no suffix, v2+ modules include `/vN` in the module path.

### Identity Inspection

Use `apx inspect` and `apx explain` to query the identity model:

```bash
# Show full identity for an API
apx inspect identity proto/payments/ledger/v1

# Show identity for a specific release
apx inspect release proto/payments/ledger/v1@v1.0.0-beta.1

# Explain Go path derivation rules
apx explain go-path proto/payments/ledger/v1
```

## Example Configuration

A complete minimal configuration:

```yaml
version: 1
org: your-org-name
repo: apis
# import_root: go.your-org-name.dev/apis   # optional: custom Go import prefix
# site_url: apis.internal.your-org-name.dev  # optional: custom domain for catalog site

module_roots:
  - proto
  - openapi
  - avro
  - jsonschema
  - parquet

language_targets:
  go:
    enabled: true
    plugins:
      - name: protoc-gen-go
        version: v1.64.0
      - name: protoc-gen-go-grpc
        version: v1.5.0

policy:
  forbidden_proto_options:
    - "^gorm\\."
  allowed_proto_plugins:
    - protoc-gen-go
    - protoc-gen-go-grpc
  openapi:
    spectral_ruleset: ".spectral.yaml"
  avro:
    compatibility: "BACKWARD"
  jsonschema:
    breaking_mode: "strict"
  parquet:
    allow_additive_nullable_only: true

release:
  tag_format: "{subdir}/v{version}"
  ci_only: true

tools:
  buf:
    version: v1.45.0
  oasdiff:
    version: v1.9.6
  spectral:
    version: v6.11.0
  avrotool:
    version: "1.11.3"
  jsonschemadiff:
    version: "0.3.0"

execution:
  mode: "local"
  container_image: ""
```

## Validation

Run `apx config validate` to check your configuration against the canonical schema:

```bash
$ apx config validate
Configuration is valid

$ apx config validate --json
{
  "valid": true,
  "errors": [],
  "warnings": []
}
```

Validation checks include:
- **Required fields**: `version`, `org`, `repo` must be present
- **Type checking**: Each field must match the expected type (string, boolean, integer, list, map)
- **Enum validation**: Fields with allowed values are checked (e.g., `execution.mode` must be `local` or `container`)
- **Pattern validation**: `release.tag_format` must contain `{version}`
- **Unknown keys**: Any field not in the schema is reported as an error
- **Deprecated fields**: Fields marked as deprecated emit warnings (not errors)

## Migration

When the schema version changes, use `apx config migrate` to upgrade:

```bash
$ apx config migrate
apx.yaml is already at version 1 (current). No migration needed.
```

Migration automatically:
- Backs up the original file to `apx.yaml.bak`
- Applies all version-to-version migration steps
- Reports changes made during migration
