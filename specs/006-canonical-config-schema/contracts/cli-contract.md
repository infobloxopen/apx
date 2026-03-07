# CLI Contract: apx config

**Feature**: 006-canonical-config-schema  
**Date**: 2026-03-07

## Commands

### `apx config validate`

Validate an `apx.yaml` file against the canonical schema for its declared version.

**Usage**:
```
apx config validate [flags]
```

**Flags**:

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--config` | string | `apx.yaml` | Path to the configuration file (inherited from root) |

**Exit Codes**:

| Code | Meaning |
|------|---------|
| 0 | File is valid (may have warnings) |
| 1 | File has validation errors |

**Output (default)**:
```
# Success
✓ Configuration is valid

# Success with warnings
⚠ field 'old_field' is deprecated since version 2; use 'new_field' instead
✓ Configuration is valid (1 warning)

# Failure
✗ Validation failed (3 errors)
  line 5: field 'org' is required
  line 8: unknown field 'foobar'
  line 12: field 'execution.mode' must be one of [local, container], got 'docker'
```

**Output (`--json`)**:
```json
{
  "valid": false,
  "errors": [
    {
      "field": "org",
      "kind": "missing",
      "message": "field 'org' is required",
      "line": 5,
      "hint": "add 'org: your-org-name' to apx.yaml"
    },
    {
      "field": "foobar",
      "kind": "unknown_key",
      "message": "unknown field 'foobar'",
      "line": 8,
      "hint": "remove 'foobar' from apx.yaml; see 'apx config validate --help' for valid fields"
    }
  ],
  "warnings": []
}
```

**Environment Variables**:

| Variable | Effect |
|----------|--------|
| `APX_CONFIG` | Overrides default config file path (same as `--config`) |

---

### `apx config migrate`

Upgrade an `apx.yaml` file from an older schema version to the current version.

**Usage**:
```
apx config migrate [flags]
```

**Flags**:

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--config` | string | `apx.yaml` | Path to the configuration file (inherited from root) |

**Exit Codes**:

| Code | Meaning |
|------|---------|
| 0 | Migration applied successfully, or no migration needed |
| 1 | Migration failed (parse error, unsupported version) |

**Output (default)**:
```
# Migration applied
Migrating apx.yaml from version 1 to version 2...
  Backed up original to apx.yaml.bak
  added: execution.container_image (default: "")
  renamed: policy.proto_forbidden → policy.forbidden_proto_options
✓ Migration complete. apx.yaml is now version 2.

# No migration needed
✓ apx.yaml is already at version 2 (current). No migration needed.

# Unsupported version
✗ apx.yaml declares version 99, but this APX binary only supports up to version 2.
  Upgrade APX to handle this configuration version.
```

**Output (`--json`)**:
```json
{
  "migrated": true,
  "from_version": 1,
  "to_version": 2,
  "backup": "apx.yaml.bak",
  "changes": [
    {
      "action": "added",
      "field": "execution.container_image",
      "detail": "added with default ''"
    },
    {
      "action": "renamed",
      "field": "policy.forbidden_proto_options",
      "detail": "renamed from 'policy.proto_forbidden'"
    }
  ]
}
```

---

### `apx config init`

Initialize a default `apx.yaml` configuration file. (Existing command — contract unchanged.)

**Usage**:
```
apx config init
```

**Exit Codes**:

| Code | Meaning |
|------|---------|
| 0 | File created successfully |
| 1 | File already exists or write error |

**Behavioral Contract Change**: The generated file MUST pass `apx config validate` without modification. This means `config.Init()` must emit the current schema version with all required fields.

---

## Validation Error Kinds

| Kind | Trigger | Example Message |
|------|---------|-----------------|
| `missing` | Required field absent | `field 'org' is required` |
| `invalid_type` | Wrong YAML type | `field 'version' expects integer, got string` |
| `invalid_value` | Value outside allowed set or pattern | `field 'execution.mode' must be one of [local, container], got 'docker'` |
| `unknown_key` | Key not in schema definition | `unknown field 'foobar'` |
| `deprecated` | Deprecated field present (warning only) | `field 'old' is deprecated since version 2; use 'new' instead` |

---

## Backward Compatibility

Per the constitution ("APX is in active pre-stable development. Backward compatibility is NOT a concern."), introducing strict validation that rejects previously-tolerated unknown keys is acceptable. Files that were silently accepted before may now fail validation. The `apx config migrate` command provides the transition path.
