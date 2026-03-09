# Release Validation

APX runs a series of validation checks before any schema is released to the canonical repository. These checks ensure consistency, compatibility, and governance compliance.

## Validation Pipeline

Validation runs at multiple points in the release flow:

| Stage | Commands | What's validated |
|-------|----------|------------------|
| **Local (pre-release)** | `apx lint`, `apx breaking`, `apx policy check` | Schema syntax, backward compatibility, policy compliance |
| **Prepare** | `apx release prepare` | Identity consistency, lifecycle-version compatibility, go_package correctness, go.mod validity |
| **Canonical CI (PR)** | `ci.yml` → `apx lint` + `apx breaking` | Re-validates schemas in the canonical repo context |
| **Finalize** | `apx release finalize` | Re-runs lint and breaking checks against the previous tag |

---

## Schema Validation

### Lint

Runs format-specific linting on all schema files:

| Format | Tool | Checks |
|--------|------|--------|
| Protocol Buffers | `buf lint` | Naming conventions, package structure, field numbering |
| OpenAPI | Spectral | Endpoint definitions, response formats, schema structure |
| Avro | avro-tools | Record structure, field defaults |
| JSON Schema | json-schema-diff | Schema validity, reference resolution |
| Parquet | built-in | Column definitions, type constraints |

### Breaking Changes

Detects backward-incompatible changes:

| Format | Tool | Detects |
|--------|------|---------|
| Protocol Buffers | `buf breaking` | Field removal/renumbering, type changes, service removal |
| OpenAPI | `oasdiff breaking` | Endpoint removal, required field additions |
| Avro | avro-tools | Field removal, type narrowing |
| JSON Schema | json-schema-diff | Property removal, type restrictions |
| Parquet | built-in | Column removal, type changes |

---

## Identity Validation

During `apx release prepare`, APX validates the full identity block:

### API ID Parsing

The API ID must follow the format `<format>/<domain>/<name>/<line>`:

```
proto/payments/ledger/v1       ✔
proto/payments/ledger           ✘  (missing version line)
payments/ledger/v1              ✘  (missing format prefix)
```

### Lifecycle-Version Compatibility

| Lifecycle | Allowed versions | Rejected versions |
|-----------|-----------------|-------------------|
| `experimental` | `-alpha.*` prerelease only | Stable, `-beta.*`, `-rc.*` |
| `beta` | `-alpha.*`, `-beta.*`, or `-rc.*` | Stable versions |
| `stable` | Stable only (no prerelease) | Any prerelease |
| `deprecated` | Any | *(none)* |
| `sunset` | Blocked (no new releases) | All (unless `--force`) |

### Version-Line Compatibility

The SemVer major version must match the declared API line:

```bash
# OK — v1 line, v1.x.x version
apx release prepare proto/payments/ledger/v1 --version v1.2.3

# ERROR — v1 line, v2.0.0 version
apx release prepare proto/payments/ledger/v1 --version v2.0.0
# → "version v2.0.0 is incompatible with API line v1"
```

### v0 Line Restrictions

| Rule | Detail |
|------|--------|
| Allowed lifecycles | `experimental` or `beta` only |
| Stable promotion | Rejected — graduate to `v1` |
| Breaking changes | Allowed with minor version bump |

### Lifecycle Transitions

Lifecycle must progress forward:

```
experimental → beta → stable → deprecated → sunset
```

Backward transitions (e.g. `stable` → `beta`) are rejected.

---

## Go-Specific Validation

### `go_package` Check

For protobuf schemas, APX validates that the `go_package` option matches the canonical import path:

```protobuf
// Expected for proto/payments/ledger/v1 in github.com/acme-corp/apis:
option go_package = "github.com/acme-corp/apis/proto/payments/ledger/v1";
```

By default, mismatches produce a **warning**. Use `--strict` to make them errors:

```bash
apx release prepare proto/payments/ledger/v1 --version v1.0.0 --strict
```

### `go.mod` Validation

APX checks the `go.mod` module directive if present, or generates one if missing:

```go
// Expected:
module github.com/acme-corp/apis/proto/payments/ledger
```

Skip with `--skip-gomod` if you manage `go.mod` externally.

---

## Idempotency Check

`apx release prepare` checks whether the exact same content was already released at the declared version by comparing SHA-256 content hashes against existing tags. If an identical release already exists, APX reports success without creating a duplicate.

---

## Dry Run

Preview what would be validated and released without making changes:

```bash
apx release prepare proto/payments/ledger/v1 --version v1.0.0 --dry-run
apx release submit --dry-run
```

Dry run shows:
- Identity report (API ID, source path, Go coordinates)
- Snapshot files that would be committed
- PR title and branch name
- Any validation warnings or errors

---

## CI Re-Validation

When a PR is opened on the canonical repo, `ci.yml` re-validates the schemas:

```yaml
- name: Lint schemas
  run: apx lint

- name: Check for breaking changes
  run: apx breaking --against origin/main
```

This catches issues that may not be visible in the app repo context (e.g. conflicts with other APIs in the canonical repo).

## See Also

- [Release Guardrails](release-guardrails.md) — lifecycle and version enforcement rules
- [Release Commands](../cli-reference/release-commands.md) — full flag reference
- [Versioning Strategy](../dependencies/versioning-strategy.md) — the three-layer versioning model
