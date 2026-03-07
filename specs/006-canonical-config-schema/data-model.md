# Data Model: Canonical APX Configuration Model

**Feature**: 006-canonical-config-schema  
**Date**: 2026-03-07

## Entity Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                      SchemaRegistry                             │
│  Versions: map[int]SchemaVersion                                │
│  CurrentVersion: int                                            │
│  Migrations: map[int]MigrationFunc                              │
└──────────┬──────────────────────────────────────────────────────┘
           │ defines
           ▼
┌──────────────────────────────────────────────────────────────────┐
│                      SchemaVersion                               │
│  Version: int                                                    │
│  Fields: map[string]FieldDef (recursive)                         │
│  RequiredFields: []string                                        │
│  Deprecated: []DeprecatedField                                   │
└──────────┬───────────────────────────────────────────────────────┘
           │ contains
           ▼
┌──────────────────────────────────────────────────────────────────┐
│                        FieldDef                                   │
│  Name: string                                                     │
│  Type: FieldType (int, string, bool, list, map, struct)          │
│  Required: bool                                                   │
│  Default: any                                                     │
│  Description: string                                              │
│  EnumValues: []string (optional)                                  │
│  Pattern: string (optional, regex for value validation)           │
│  Children: map[string]FieldDef (for struct type)                 │
│  ItemDef: *FieldDef (for list type, defines list element shape)  │
│  DeprecatedSince: int (0 = not deprecated)                       │
│  Replacement: string (field path of replacement)                 │
└──────────────────────────────────────────────────────────────────┘
```

## Entities

### 1. SchemaRegistry

The singleton that holds all known schema versions and the migration chain. Lives as package-level state in `internal/config/schema.go`.

| Attribute | Type | Description |
|-----------|------|-------------|
| Versions | `map[int]SchemaVersion` | All defined schema versions, keyed by version integer |
| CurrentVersion | `int` | The latest schema version supported by this binary |
| Migrations | `map[int]MigrationFunc` | Migration from version N to N+1, keyed by source version |

**Relationships**: Contains SchemaVersion instances. Referenced by Validate() and Migrate().

### 2. SchemaVersion

Defines the complete allowed structure for `apx.yaml` at a given version.

| Attribute | Type | Description |
|-----------|------|-------------|
| Version | `int` | Schema version number (monotonically increasing) |
| Fields | `map[string]FieldDef` | Top-level fields and their recursive definitions |
| RequiredFields | `[]string` | Top-level field names that must be present |
| Deprecated | `[]DeprecatedField` | Fields that are accepted but produce warnings |

**Relationships**: Contains FieldDef instances. Referenced by SchemaRegistry.

### 3. FieldDef

Metadata for a single configuration field. Used recursively for nested structures.

| Attribute | Type | Description |
|-----------|------|-------------|
| Name | `string` | Field name as it appears in YAML |
| Type | `FieldType` | One of: `TypeInt`, `TypeString`, `TypeBool`, `TypeList`, `TypeMap`, `TypeStruct` |
| Required | `bool` | Whether the field must be present in its parent |
| Default | `any` | Default value when omitted (nil = no default) |
| Description | `string` | Human-readable description for docs and error messages |
| EnumValues | `[]string` | If non-nil, the field value must be one of these |
| Pattern | `string` | If non-empty, the field value must match this regex |
| Children | `map[string]FieldDef` | For `TypeStruct`: allowed child fields |
| ItemDef | `*FieldDef` | For `TypeList`: schema for each list element |
| DeprecatedSince | `int` | Schema version in which this field was deprecated (0 = active) |
| Replacement | `string` | Dotted field path of the replacement (empty = no replacement) |

**Validation rules derived from FieldDef**:
- `Required && missing` → error: "field '{path}' is required"
- `Type mismatch` → error: "field '{path}' expects {Type}, got {actual}"
- `EnumValues && value not in set` → error: "field '{path}' must be one of [{values}], got '{actual}'"
- `Pattern && !match` → error: "field '{path}' must match pattern '{pattern}'"
- `Unknown key (not in Children)` → error: "unknown field '{path}'"
- `DeprecatedSince > 0` → warning: "field '{path}' is deprecated since version {v}; use '{Replacement}' instead"

### 4. ValidationError

The structured output of a single validation violation.

| Attribute | Type | Description |
|-----------|------|-------------|
| Field | `string` | Dotted path to the offending field (e.g., `policy.openapi.spectral_ruleset`) |
| Kind | `ErrorKind` | Enum: `Missing`, `InvalidType`, `InvalidValue`, `UnknownKey`, `Deprecated` |
| Message | `string` | Human-readable description of the violation |
| Line | `int` | YAML source line number (0 if unavailable) |
| Hint | `string` | Suggested fix (e.g., "add 'org: your-org-name' to apx.yaml") |

**Relationships**: Produced by Validate(). Consumed by CLI rendering and `--json` output.

### 5. ValidationResult

Aggregated outcome of validating an entire `apx.yaml` file.

| Attribute | Type | Description |
|-----------|------|-------------|
| Errors | `[]ValidationError` | Hard errors that make the file invalid |
| Warnings | `[]ValidationError` | Non-fatal warnings (deprecated fields) |
| Valid | `bool` | True if Errors is empty |

### 6. Change (migration)

A single transformation applied during migration.

| Attribute | Type | Description |
|-----------|------|-------------|
| Action | `string` | One of: `added`, `removed`, `renamed`, `changed_default` |
| Field | `string` | Dotted field path affected |
| Detail | `string` | Human-readable explanation (e.g., "added 'execution.container_image' with default ''") |

**Relationships**: Produced by MigrationFunc. Rendered by `apx config migrate` output.

---

## Current Config Struct → Schema Version 1 Field Map

This table maps the existing `Config` struct in `internal/config/config.go` to the FieldDef tree for schema version 1:

| YAML Path | Type | Required | Default | Enum Values | Description |
|-----------|------|----------|---------|-------------|-------------|
| `version` | int | yes | — | — | Schema version number |
| `org` | string | yes | — | — | GitHub organization name |
| `repo` | string | yes | — | — | Canonical API repository name |
| `module_roots` | list(string) | no | `["proto"]` | — | Directories containing schema modules |
| `language_targets` | map(string→struct) | no | `{}` | — | Code generation targets keyed by language |
| `language_targets.<lang>.enabled` | bool | no | `false` | — | Whether this language target is active |
| `language_targets.<lang>.tool` | string | no | `""` | — | Tool name (e.g., grpcio-tools) |
| `language_targets.<lang>.version` | string | no | `""` | — | Tool version |
| `language_targets.<lang>.plugins` | list(map) | no | `[]` | — | List of plugin name/version maps |
| `policy` | struct | no | `{}` | — | Validation policy settings |
| `policy.forbidden_proto_options` | list(string) | no | `[]` | — | Regex patterns for forbidden proto options |
| `policy.allowed_proto_plugins` | list(string) | no | `[]` | — | Allowed protoc plugin names |
| `policy.openapi` | struct | no | `{}` | — | OpenAPI-specific policy |
| `policy.openapi.spectral_ruleset` | string | no | `""` | — | Path to Spectral ruleset file |
| `policy.avro` | struct | no | `{}` | — | Avro-specific policy |
| `policy.avro.compatibility` | string | no | `"BACKWARD"` | `BACKWARD`, `FORWARD`, `FULL`, `NONE` | Avro compatibility mode |
| `policy.jsonschema` | struct | no | `{}` | — | JSON Schema policy |
| `policy.jsonschema.breaking_mode` | string | no | `"strict"` | `strict`, `lenient` | Breaking change detection mode |
| `policy.parquet` | struct | no | `{}` | — | Parquet policy |
| `policy.parquet.allow_additive_nullable_only` | bool | no | `true` | — | Whether to restrict to additive nullable columns |
| `publishing` | struct | no | `{}` | — | Publishing configuration |
| `publishing.tag_format` | string | no | `"{subdir}/v{version}"` | — | Tag pattern; must contain `{version}` |
| `publishing.ci_only` | bool | no | `true` | — | Restrict publishing to CI environments |
| `tools` | struct | no | `{}` | — | Pinned tool versions |
| `tools.buf.version` | string | no | `""` | — | Buf CLI version |
| `tools.oasdiff.version` | string | no | `""` | — | oasdiff version |
| `tools.spectral.version` | string | no | `""` | — | Spectral version |
| `tools.avrotool.version` | string | no | `""` | — | Avro tools version |
| `tools.jsonschemadiff.version` | string | no | `""` | — | JSON Schema diff version |
| `execution` | struct | no | `{}` | — | Execution environment settings |
| `execution.mode` | string | no | `"local"` | `local`, `container` | Where tools run |
| `execution.container_image` | string | no | `""` | — | Container image when mode=container |

---

## State Transitions

### Config File Lifecycle

```
                 ┌──────────┐
                 │ No File  │
                 └────┬─────┘
                      │ apx init / apx config init
                      ▼
                ┌───────────┐
                │  Valid v1  │◄──── apx config validate (PASS)
                └────┬──────┘
                     │ schema evolves
                     ▼
              ┌──────────────┐
              │  Outdated vN │──── apx config validate (FAIL: old version)
              └────┬─────────┘
                   │ apx config migrate
                   ▼
           ┌────────────────┐
           │ Valid vCurrent  │◄── apx config validate (PASS)
           └────────────────┘
```

### Validation State Machine

```
  Input YAML ──► Parse (yaml.Node)
                    │
                    ├── Parse error → ValidationResult{Errors: [syntax error]}
                    │
                    ▼
               Walk tree against SchemaVersion.Fields
                    │
                    ├── Unknown key → collect UnknownKey error
                    ├── Missing required → collect Missing error
                    ├── Wrong type → collect InvalidType error
                    ├── Invalid value → collect InvalidValue error
                    ├── Deprecated → collect Deprecated warning
                    │
                    ▼
               ValidationResult{Errors: [...], Warnings: [...]}
```
