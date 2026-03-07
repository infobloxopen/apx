# Configuration Reference

The APX configuration file (`apx.yaml`) defines your organization's schema management settings. It controls validation policies, publishing behavior, tool versions, and code generation targets.

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
| `org`     | string  | GitHub organization name      |
| `repo`    | string  | Canonical API repository name |

## Complete Field Reference

<!-- This table is generated from the canonical schema definition in internal/config/schema.go -->

| YAML Path | Type | Required | Default | Allowed Values | Description |
|-----------|------|----------|---------|----------------|-------------|
| `version` | integer | yes |  |  | Schema version number |
| `org` | string | yes |  |  | GitHub organization name |
| `repo` | string | yes |  |  | Canonical API repository name |
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
| `publishing` | struct | no |  |  | Publishing configuration |
| `publishing.tag_format` | string | no | `{subdir}/v{version}` |  | Tag pattern; must contain {version} |
| `publishing.ci_only` | boolean | no | `true` |  | Restrict publishing to CI environments |
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

## Section Details

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

### `publishing`

Controls how schema versions are tagged and published.

```yaml
publishing:
  tag_format: "{subdir}/v{version}"
  ci_only: true
```

The `tag_format` must contain `{version}`. The `ci_only` flag restricts publishing to CI environments (recommended for production use).

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

## Example Configuration

A complete minimal configuration:

```yaml
version: 1
org: your-org-name
repo: your-apis-repo

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

publishing:
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
- **Pattern validation**: `publishing.tag_format` must contain `{version}`
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
