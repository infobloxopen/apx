# Quickstart: Canonical APX Configuration Model

**Feature**: 006-canonical-config-schema  
**Date**: 2026-03-07

## What This Feature Does

This feature establishes a single, authoritative schema for `apx.yaml` — the configuration file that controls APX behavior. It introduces:

1. **Strict validation** (`apx config validate`) that reports all errors with field paths and fix suggestions
2. **Schema versioning** so the config format can evolve without silent breakage
3. **Migration support** (`apx config migrate`) to upgrade older files to the current version
4. **Unified init output** so every `apx init` variant produces a valid, current-schema file

## Developer Workflow

### Validate your config

```bash
# Validate the default apx.yaml
apx config validate

# Validate a specific file
apx config validate --config path/to/apx.yaml

# Get JSON output for CI
apx config validate --json
```

### Create a new config

```bash
# Option 1: Standalone config init
apx config init

# Option 2: Full project init (also creates config)
apx init canonical --org myorg --repo my-apis
```

Both produce an `apx.yaml` that passes validation immediately.

### Migrate an older config

```bash
# Migrate to current schema version
apx config migrate

# The original file is backed up to apx.yaml.bak
```

### Read validation errors

Each error tells you exactly what to fix:

```
✗ Validation failed (2 errors)
  line 1: field 'version' is required
  line 8: unknown field 'foobar' — remove it or check spelling
```

## Minimal Valid apx.yaml

```yaml
version: 1
org: my-org
repo: my-apis
```

All other fields are optional and have documented defaults.

## Key Rules

- `version` is a required integer. It must match a version this APX binary supports.
- `org` and `repo` are required strings. They identify the GitHub organization and canonical API repo.
- Unknown top-level and nested keys are rejected (not silently ignored).
- Deprecated fields produce warnings, not errors — with a pointer to the replacement field.
- The config file path can be overridden via `--config` flag or `APX_CONFIG` environment variable.

## Implementation Notes for Contributors

### Where the code lives

| Concern | File |
|---------|------|
| Schema definitions (field tree) | `internal/config/schema.go` |
| Validation engine | `internal/config/validate.go` |
| Migration engine | `internal/config/migrate.go` |
| Config struct & Load() | `internal/config/config.go` |
| CLI wiring | `cmd/apx/commands/config.go` |

### Architecture principle

The `Config` struct and `SchemaVersion` field tree are the **single source of truth**. All YAML emitters (init, scaffold) marshal from the struct. All validators walk the field tree. Documentation is verified against the field tree.
