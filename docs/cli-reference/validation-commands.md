# Validation Commands

Commands for validating schemas, detecting breaking changes, classifying operations, and suggesting version bumps.

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
| `--operation-ids` | | bool | `true` | OpenAPI only: fail when an operation has no `operationId` or an `operationId` repeats. Opt out with `--operation-ids=false`. |

### Format-Specific Validation

| Format | Tool | What it checks |
|--------|------|----------------|
| Protocol Buffers | `buf lint` | Naming conventions, package structure, field numbering, service definitions |
| OpenAPI | Native Go + Spectral | Built-in `operationId` check (see below), then schema structure, endpoint definitions, response formats |
| Avro | Native Go | Record structure, field type validity, required `name`/`fields` presence |
| JSON Schema | Native Go | JSON syntax, `$schema` URI, `type`, `properties`, `required` |
| Parquet | Native Go | Message-notation syntax, physical type validity, repetition levels |

For protobuf, APX also runs `go_package` validation — warning if the `go_package` option doesn't match the canonical import path.

### OpenAPI `operationId` check

For OpenAPI, `apx lint` has a built-in check that does not depend on Spectral or on a Spectral ruleset. It runs before Spectral, and it also runs when Spectral is not installed. It fails when:

- an operation has no `operationId`;
- an `operationId` repeats inside one document;
- an operation has an `x-operation-class` that is not `read`, `list`, `write` or `custom`.

The `operationId` is the stable name of an operation. Generated clients, dashboards and per-operation metrics use it as a key, so each operation must have one and each one must be unique.

```text
operationId check failed for ledger.yaml (2 problems):
  GET /entries/{id}: operationId is missing
  operationId "entries" is not unique: GET /entries, POST /entries
```

The check is on by default. A repository that cannot fix its specs immediately can turn it off with `apx lint --operation-ids=false`. `apx release finalize` runs the same check during re-validation and has the same flag.

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
| `--advisory` | | bool | `false` | Report breaking changes and exit 0 |
| `--operation-ids` | | bool | `true` | OpenAPI only: report a renamed or removed `operationId` as a breaking change. Opt out with `--operation-ids=false`. |

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
| OpenAPI | `oasdiff breaking` + native Go | Endpoint removal, required field additions, response type changes, renamed or removed `operationId` |
| Avro | Native Go | New fields without defaults, type changes (BACKWARD/FORWARD/FULL/NONE modes) |
| JSON Schema | Native Go | Property removal, type change, required field additions |
| Parquet | Native Go | New required columns, removed columns, type/annotation changes, optional→required promotion |

### Examples

```bash
apx breaking --against HEAD^
apx breaking internal/apis/proto/ --against origin/main
apx breaking --format openapi --against v1.0.0
```

### OpenAPI `operationId` changes

`oasdiff breaking` rates a renamed or removed `operationId` as information (`api-operation-id-removed`), not as an error, so it does not fail on it. APX therefore compares the `operationId` of each operation of the baseline with the revision. Operations are matched by HTTP method and path; the names of path variables do not matter. A breaking finding is:

- an operation that stays on the same method and path with another `operationId`, or with none;
- an operation that is gone, when no operation of the revision has its `operationId`.

```text
operationId changes (1); a renamed or removed operationId breaks generated clients and per-operation history:
  GET /entries: operationId "listEntries" was renamed to "listAllEntries"
```

A finding fails the command like any other breaking change. `--advisory` reports it and exits 0. `--operation-ids=false` turns the comparison off. `apx release finalize` runs the same comparison against the previous release and has the same flag.

---

## `apx opclass`

Classify each operation of an OpenAPI 3.x or Swagger 2 document as `read`, `list`, `write` or `custom`.

```bash
apx opclass <spec>
```

The operation class is the value of the extension `x-operation-class`. It is separate from `x-aip-method`: `x-aip-method` marks only operations with the strict AIP request and response shapes, and client generators depend on that meaning. `apx opclass` never writes or changes `x-aip-method`.

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--format` | string | `text` | Output format: `text` or `json`. The global `--json` flag also selects `json`. |
| `--write` | bool | `false` | Add `x-operation-class` to each operation of the file that lacks one |

### Rules

An `x-operation-class` that is already on an operation always wins. Its value must be `read`, `list`, `write` or `custom`. Without it, the first rule that matches decides. The `RULE` column of the output names the rule.

| Order | Rule | Match | Class |
|-------|------|-------|-------|
| 0 | `explicit` | `x-operation-class` is on the operation | its value |
| 1 | `aip` | `x-aip-method` is `Get` or `BatchGet` | `read` |
| | | `x-aip-method` is `List` or `Search` | `list` |
| | | `x-aip-method` is `Create`, `Update`, `Delete`, `Undelete`, `Purge`, `BatchCreate`, `BatchUpdate` or `BatchDelete` | `write` |
| 2 | `name` | The `operationId` starts or ends with the word `List` | `list` |
| | | The `operationId` starts or ends with the word `Read` or `Get` | `read` |
| | | The `operationId` starts or ends with the word `Create`, `Update`, `Patch` or `Delete` | `write` |
| 3 | `shape` | `GET` or `HEAD` on a path whose last segment is one variable (`/users/{id}`) | `read` |
| | | `GET` or `HEAD` on another path (`/users`) | `list` |
| | | `POST`, `PUT`, `PATCH` or `DELETE` | `write` |
| 4 | `default` | Nothing matched | `custom` |

`x-aip-method: Custom` and unknown `x-aip-method` values do not match rule 1. `GET` on the root path and `GET` on a path with a custom verb (`/users/{id}:export`) do not match rule 3.

**How rule 2 reads an `operationId`.** The id is split into words at each character that is not a letter or a digit, at each change from lower case to upper case (`getUser`), and at the end of an acronym (`HTTPServer`). The comparison ignores case. In the published protobuf form `Service_Method`, only the method part is used, and leading method words that repeat the service name are dropped: `Accounts_AccountListUsers` is read as `List`, `Users`. The first word decides when it is a verb; otherwise the last word decides. An id that starts with `Get` or `Read` and ends with `List` (`GetUserList`) is `list`.

Only whole words match: `Blacklist` and `Checklist` do not match `List`. The name rule cannot see a plural: `getDashboards` is `read`. When a rule gives the wrong class, set `x-operation-class` on the operation.

### Streaming

The `STREAMING` column is `true` when a response of the operation has the content type `text/event-stream` or `application/x-ndjson`, when a Swagger 2 `produces` list has one of them, or when the operation has `x-streaming: true`.

### Output

```text
METHOD  PATH          OPERATION ID                    CLASS  STREAMING  RULE
GET     /groups       AccountOrganizationGroups_List  list   false      name
POST    /groups       -                               write  false      shape
GET     /groups/{id}  AccountOrganizationGroups_Read  read   false      name

3 operations: read=1 list=1 write=1 custom=0
```

Operations are sorted by path, then by method. `PATH` is the path as the document writes it; the JSON output also has `basePath`, the path of `servers[0].url` (or the Swagger 2 `basePath`). A `-` in `OPERATION ID` means that the operation has no `operationId`.

```json
{
  "spec": "groups.yaml",
  "basePath": "/api/v1",
  "operations": [
    {"method": "GET", "path": "/groups", "operationId": "AccountOrganizationGroups_List", "class": "list", "streaming": false, "rule": "name"}
  ],
  "counts": {"custom": 0, "list": 1, "read": 0, "write": 0}
}
```

### Writing the class into the spec

`apx opclass <spec> --write` adds `x-operation-class` to each operation that lacks one. An operation that already has the extension keeps its value. With `--write`, the JSON output also has `added`, the number of operations that changed.

APX does not re-encode the file. It inserts text and leaves every other byte as it is: comments, key order, quoting, indentation and line endings.

- Block-style YAML: one new line after the `operationId` line, or before the first key of the operation when the `operationId` is absent or spans lines.
- Flow-style YAML and JSON: a quoted member, as the first member of the operation object.

Limits:

- An operation that the document reaches through a YAML alias (`get: *shared`) is not changed.
- An `x-operation-class` that arrives through a merge key (`<<`) is not seen; the operation gets its own.
- Only the first YAML document of a stream is read, and a `$ref` on a path item is not followed.

### Examples

```bash
# Print the table
apx opclass openapi/payments/ledger/v1/ledger.yaml

# JSON output
apx opclass ledger.yaml --format json

# Add x-operation-class where it is missing
apx opclass ledger.yaml --write
```

---

## `apx semver`

Suggest a semantic version bump based on schema changes.

```bash
apx semver [path] --against <ref>
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
| `beta` | `-beta.<n>` |
| `stable` | *(none)* |

### Examples

```bash
# Suggest based on changes since last commit
apx semver --against HEAD^
# → minor (new fields added, no breaking changes)

# With explicit API ID and lifecycle
apx semver --api-id proto/payments/ledger/v1 --lifecycle beta --against HEAD^
# → v1.1.0-beta.1

# JSON output
apx --json semver --against origin/main
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

- [Release Guardrails](../releasing/release-guardrails.md) — version and lifecycle enforcement
- [Versioning Strategy](../dependencies/versioning-strategy.md) — the three-layer versioning model
- [Buf Issues](../troubleshooting/buf-issues.md) — troubleshooting Buf-related problems
