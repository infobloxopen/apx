# Validation Commands

Commands for validating schemas, detecting breaking changes, and suggesting version bumps.

## `apx lint`

Validate schema files for syntax and style issues.

```bash
apx lint [path]
```

If `path` is omitted, APX detects schema directories from `module_roots` in `apx.yaml`. If `path` is an API ID (e.g. `proto/payments/ledger/v1`), APX resolves it to the local filesystem path.

### Flags

| Flag | Shorthand | Type | Default | Description |
|------|-----------|------|---------|-------------|
| `--format` | `-f` | string | auto-detected | Schema format: proto, openapi, avro, jsonschema, parquet |

### Format-Specific Validation

| Format | Tool | What it checks |
|--------|------|----------------|
| Protocol Buffers | `buf lint` | Naming conventions, package structure, field numbering, service definitions |
| OpenAPI | Spectral | Schema structure, endpoint definitions, response formats |
| Avro | avro-tools | Record structure, field defaults, type compatibility |
| JSON Schema | json-schema-diff | Schema validity, reference resolution |
| Parquet | built-in | Column definitions, type constraints |

For protobuf, APX also runs `go_package` validation — warning if the `go_package` option doesn't match the canonical import path.

### Examples

```bash
# Lint all schemas in the project
apx lint

# Lint a specific directory
apx lint internal/apis/proto/payments/ledger/v1

# Force format detection
apx lint --format proto internal/apis/

# Verbose output with details
apx lint --verbose
```

---

## `apx breaking`

Check for breaking changes between the current schemas and a baseline.

```bash
apx breaking [path] --against <ref>
```

The `--against` flag is required and specifies the baseline to compare against.

### Flags

| Flag | Shorthand | Type | Default | Description |
|------|-----------|------|---------|-------------|
| `--against` | | string | *(required)* | Git reference or path to compare against |
| `--format` | `-f` | string | auto-detected | Schema format |

### Supported Baselines

```bash
# Compare against previous commit
apx breaking --against HEAD^

# Compare against a branch
apx breaking --against origin/main

# Compare against a specific tag
apx breaking --against proto/payments/ledger/v1/v1.0.0

# Compare against a specific commit
apx breaking --against abc1234
```

### Format-Specific Checks

| Format | Tool | Breaking changes detected |
|--------|------|---------------------------|
| Protocol Buffers | `buf breaking` | Field removal/renumbering, type changes, service/method removal |
| OpenAPI | `oasdiff breaking` | Endpoint removal, required field additions, response type changes |
| Avro | avro-tools | Field removal, type narrowing, default changes |
| JSON Schema | json-schema-diff | Property removal, type restriction, required additions |
| Parquet | built-in | Column removal, type changes |

### Examples

```bash
apx breaking --against HEAD^
apx breaking internal/apis/proto/ --against origin/main
apx breaking --format openapi --against v1.0.0
```

---

## `apx semver suggest`

Suggest a semantic version bump based on schema changes.

```bash
apx semver suggest [path] --against <ref>
```

### Flags

| Flag | Shorthand | Type | Default | Description |
|------|-----------|------|---------|-------------|
| `--against` | | string | *(required)* | Git reference to compare against |
| `--api-id` | | string | `""` | API ID (e.g. proto/payments/ledger/v1) |
| `--lifecycle` | | string | `""` | Lifecycle state |
| `--format` | `-f` | string | auto-detected | Schema format |

### How It Works

1. Runs `apx breaking` to detect breaking changes
2. Lists existing git tags to find the current latest version
3. Applies versioning rules:

| Change type | Bump | Rationale |
|-------------|------|-----------|
| Breaking changes | **Rejected** | Requires a new major API line (e.g. v1 → v2) |
| Non-breaking additive changes | **Minor** | New fields, RPCs, or endpoints |
| No schema changes | **Patch** | Documentation, metadata, or tooling-only changes |

4. Applies lifecycle mapping for prerelease tags:

| Lifecycle | Prerelease format |
|-----------|------------------|
| `experimental` | `-alpha.<n>` |
| `preview` | `-beta.<n>` |
| `stable` | *(none)* |

### Examples

```bash
# Suggest based on changes since last commit
apx semver suggest --against HEAD^
# → minor (new fields added, no breaking changes)

# With explicit API ID and lifecycle
apx semver suggest --api-id proto/payments/ledger/v1 --lifecycle beta --against HEAD^
# → v1.1.0-beta.1

# JSON output
apx --json semver suggest --against origin/main
```

---

## `apx policy check`

Check schema files against organization policies.

```bash
apx policy check [path]
```

Validates schemas against policy rules configured in `apx.yaml`:

- **Forbidden proto options** — rejects schemas using banned annotations (e.g. `gorm.*`)
- **Allowed proto plugins** — ensures only approved code generation plugins are used
- **OpenAPI ruleset** — applies custom Spectral rulesets
- **Avro compatibility** — enforces compatibility mode (BACKWARD, FORWARD, FULL)
- **Parquet rules** — enforces additive nullable-only column additions

### Example

```bash
apx policy check
apx policy check internal/apis/proto/
```

---

## Validation in CI

A typical CI pipeline runs all three validation commands:

```yaml
- name: Validate schemas
  run: |
    apx lint
    apx breaking --against origin/main
    apx policy check
```

## See Also

- [Release Guardrails](../publishing/release-guardrails.md) — version and lifecycle enforcement
- [Versioning Strategy](../dependencies/versioning-strategy.md) — the three-layer versioning model
- [Buf Issues](../troubleshooting/buf-issues.md) — troubleshooting Buf-related problems
