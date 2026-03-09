# Format Maturity Matrix

APX supports five schema formats. Implementation depth varies by format —
this page documents exactly which capabilities are available today.

## Support Matrix

| Capability | Proto | OpenAPI | Avro | JSON Schema | Parquet |
|------------|:-----:|:-------:|:----:|:-----------:|:-------:|
| **Linting** | ✅ | ✅ | ✅ | ✅ | ✅ |
| **Breaking-change detection** | ✅ | ✅ | ✅ | Partial | ✅ |
| **Release** | ✅ | ✅ | ✅ | ✅ | ✅ |
| **Code generation** | ✅ | ✅ | ✅ | ✅ | ✅ |
| **Catalog / discovery** | ✅ | ✅ | ✅ | ✅ | ✅ |
| **Policy enforcement** | ✅ | Partial | Partial | Partial | Partial |

**Legend:** ✅ = implemented and tested, Partial = limited scope (see format details),
❌ = returns `ErrNotImplemented` or is a stub

## Format Details

### Protocol Buffers — Tier 1 (fully supported)

All six capabilities are implemented and tested.

| Feature | Implementation |
|---------|----------------|
| Lint | Delegates to `buf lint` via toolchain resolver |
| Breaking | Delegates to `buf breaking --against` |
| Release | Full release pipeline with `go_package` validation, `go.mod` generation |
| Codegen | `apx gen go` with overlay system and `buf generate` |
| Catalog | Tag-based discovery (`proto/<domain>/<name>/<line>/v<semver>`) |
| Policy | Scans `.proto` files for forbidden options, validates `buf.gen.yaml` plugin allowlist |

### OpenAPI — Tier 2 (mostly supported)

Five of six capabilities are fully implemented; policy only checks for Spectral
ruleset file existence.

| Feature | Implementation |
|---------|----------------|
| Lint | Delegates to Spectral (`spectral lint`) |
| Breaking | Delegates to oasdiff (`oasdiff breaking`) |
| Release | Format-agnostic identity and release pipeline |
| Codegen | Overlay system (format-agnostic) |
| Catalog | Tag-based discovery |
| Policy | Checks that the configured Spectral ruleset file exists (does not run Spectral) |

### Avro — Tier 2 (fully supported)

Both lint and breaking-change detection are implemented natively in Go — no
external tools required.

| Feature | Implementation |
|---------|----------------|
| Lint | Native Go: parses JSON, validates `type`/`name`/`fields` structure |
| Breaking | Native Go: BACKWARD/FORWARD/FULL/NONE compatibility rules |
| Release | Format-agnostic pipeline |
| Codegen | Overlay system (format-agnostic) |
| Catalog | Tag-based discovery |
| Policy | Validates compatibility mode string (`BACKWARD`, `FORWARD`, `FULL`, `NONE`) |

**Avro breaking-change rules (BACKWARD mode):**

- New field without a default: **breaking** (old data lacks the field)
- New field with a default or nullable union (`["null", ...]`): safe
- Field type change: **breaking**
- Removed field: safe (reader ignores unknown writer fields)

### JSON Schema — Tier 3 (mostly supported)

Linting is implemented natively in Go. Breaking-change detection delegates to
`jsonschema-diff` (must be installed separately).

| Feature | Implementation |
|---------|----------------|
| Lint | Native Go: validates JSON syntax, `$schema` URI, `type`, `properties`, `required` |
| Breaking | Delegates to `jsonschema-diff` (requires external tool) |
| Release | Format-agnostic pipeline |
| Codegen | Overlay system (format-agnostic) |
| Catalog | Tag-based discovery |
| Policy | Validates `breaking_mode` string (`strict` or `lenient`) |

### Parquet — Tier 2 (fully supported)

Both lint and breaking-change detection are implemented natively in Go using the
Parquet message-notation schema format.

APX represents Parquet schemas as `.parquet` text files using the message notation
(`message name { required binary id (STRING); ... }`). This is the same schema
format that `parquet-tools schema` outputs.

| Feature | Implementation |
|---------|----------------|
| Lint | Native Go: parses message notation, validates physical types and repetition levels |
| Breaking | Native Go: additive-nullable policy enforcement |
| Release | Format-agnostic pipeline |
| Codegen | Overlay system (format-agnostic) |
| Catalog | Tag-based discovery |
| Policy | Validates additive-nullable-only policy |

**Parquet breaking-change rules:**

- New optional column: safe (additive nullable)
- New required column: **breaking** (old data has no values for it)
- Removed column: **breaking**
- Physical type change: **breaking**
- `optional` → `required`: **breaking** (old data may contain nulls)
- `required` → `optional`: safe (relaxing the constraint)
- Logical type annotation change: **breaking** (affects deserialization)

## Format-Agnostic vs Format-Specific

Three capabilities — **release**, **code generation**, and **catalog/discovery** —
are format-agnostic by design. They work for any format string because they operate
on directory structure and git tags, not on schema file contents. These show as ✅
for all formats.

The format-specific capabilities are **linting**, **breaking-change detection**,
and **policy enforcement**:

| Format | Lint | Breaking | External tool required? |
|--------|------|----------|------------------------|
| Proto | `buf lint` | `buf breaking` | Yes (`buf`) |
| OpenAPI | Spectral | oasdiff | Yes (`spectral`, `oasdiff`) |
| Avro | Native Go | Native Go | No |
| JSON Schema | Native Go | `jsonschema-diff` | Breaking only (`jsonschema-diff`) |
| Parquet | Native Go | Native Go | No |

## See Also

- [Schema Validation](../cli-reference/core-commands.md) — `apx lint` and `apx breaking`
- [Policy Enforcement](../cli-reference/core-commands.md) — `apx policy check`
- [Troubleshooting Validation](../troubleshooting/common-errors.md)
