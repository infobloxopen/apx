# Format Maturity Matrix

APX supports five schema formats. Implementation depth varies by format —
this page documents exactly which capabilities are available today.

## Support Matrix

| Capability | Proto | OpenAPI | Avro | JSON Schema | Parquet |
|------------|:-----:|:-------:|:----:|:-----------:|:-------:|
| **Linting** | ✅ | ✅ | ✅ | ❌ | ❌ |
| **Breaking-change detection** | ✅ | ✅ | ❌ | ✅ | ❌ |
| **Release** | ✅ | ✅ | ✅ | ✅ | ✅ |
| **Code generation** | ✅ | ✅ | ✅ | ✅ | ✅ |
| **Catalog / discovery** | ✅ | ✅ | ✅ | ✅ | ✅ |
| **Policy enforcement** | ✅ | Partial | Partial | Partial | ❌ |

**Legend:** ✅ = implemented and tested, Partial = config validation only,
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

### Avro — Tier 3 (partially supported)

Lint works via `avro-tools`; breaking-change detection is not yet implemented.

| Feature | Implementation |
|---------|----------------|
| Lint | Delegates to `java -jar avro-tools compile schema` |
| Breaking | **Not implemented** — returns `ErrNotImplemented` |
| Release | Format-agnostic pipeline |
| Codegen | Overlay system (format-agnostic) |
| Catalog | Tag-based discovery |
| Policy | Validates compatibility mode string (`BACKWARD`, `FORWARD`, etc.) |

### JSON Schema — Tier 4 (partially supported)

Breaking-change detection works via `jsonschema-diff`; linting is not yet
implemented.

| Feature | Implementation |
|---------|----------------|
| Lint | **Not implemented** — returns `ErrNotImplemented` |
| Breaking | Delegates to `jsonschema-diff` |
| Release | Format-agnostic pipeline |
| Codegen | Overlay system (format-agnostic) |
| Catalog | Tag-based discovery |
| Policy | Validates `breaking_mode` string (`strict` or `lenient`) |

### Parquet — Tier 5 (scaffold only)

Release, codegen, and catalog work because they are format-agnostic. Format-specific
features (lint, breaking, policy) are stubs.

| Feature | Implementation |
|---------|----------------|
| Lint | **Not implemented** — returns `ErrNotImplemented` |
| Breaking | **Not implemented** — returns `ErrNotImplemented` |
| Release | Format-agnostic pipeline |
| Codegen | Overlay system (format-agnostic) |
| Catalog | Tag-based discovery |
| Policy | Stub (no-op) |

## Format-Agnostic vs Format-Specific

Three capabilities — **release**, **code generation**, and **catalog/discovery** —
are format-agnostic by design. They work for any format string because they operate
on directory structure and git tags, not on schema file contents. These show as ✅
for all formats.

The truly format-specific capabilities are **linting**, **breaking-change detection**,
and **policy enforcement**. These delegate to external tools and require format-specific
integration:

| Format | Lint tool | Breaking tool |
|--------|-----------|---------------|
| Proto | `buf lint` | `buf breaking` |
| OpenAPI | Spectral | oasdiff |
| Avro | `avro-tools` | *(not yet wired)* |
| JSON Schema | *(not yet wired)* | `jsonschema-diff` |
| Parquet | *(not yet wired)* | *(not yet wired)* |

## Roadmap

Formats with missing capabilities are tracked for future implementation:

- **Avro breaking detection**: Integrate Avro schema compatibility checker
- **JSON Schema linting**: Integrate a JSON Schema linter (e.g. `ajv`)
- **Parquet lint and breaking**: Requires a Parquet schema evolution tool

## See Also

- [Schema Validation](../cli-reference/core-commands.md) — `apx lint` and `apx breaking`
- [Policy Enforcement](../cli-reference/core-commands.md) — `apx policy check`
- [Troubleshooting Validation](../troubleshooting/common-errors.md)
