# Implementation Plan: Canonical APX Configuration Model

**Branch**: `006-canonical-config-schema` | **Date**: 2026-03-07 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/006-canonical-config-schema/spec.md`

## Summary

Establish a single authoritative schema definition for `apx.yaml` that drives validation, init-template generation, migration, and documentation. The current codebase has three independent YAML-emitting code paths (config.Init, schema.Initializer.createConfigWithDefaults, schema.AppScaffolder.generateApxYaml) and a minimal three-field validator (version, org, repo). This feature introduces a versioned schema registry in `internal/config/`, a strict `apx config validate` that rejects unknown keys and reports field-path errors, an `apx config migrate` command for forward-compatible upgrades, and a generated schema reference doc that stays in sync with the implementation.

## Technical Context

**Language/Version**: Go 1.26.1  
**Primary Dependencies**: cobra (CLI), gopkg.in/yaml.v3 (YAML parsing), testify (assertions), go-internal/testscript (integration tests)  
**Storage**: Filesystem (`apx.yaml`, `apx.lock`)  
**Testing**: `go test` with table-driven unit tests + testscript integration tests  
**Target Platform**: macOS, Linux, Windows (cross-platform per constitution)  
**Project Type**: CLI tool  
**Performance Goals**: Validation completes in <200ms for typical `apx.yaml` files  
**Constraints**: No new external dependencies for schema validation; use stdlib + yaml.v3  
**Scale/Scope**: Single config file, typically <100 lines; ~5-10 schema versions over the product lifetime

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Documentation-Driven Development | PASS | Spec written first; schema reference doc will be generated from code to maintain parity |
| II. Cross-Platform Path Operations | PASS | Config file paths use `filepath` package; `GetConfigPath()` already uses `filepath.Join` |
| III. Test-First Development | PASS | Unit tests for schema validation, testscript integration tests for CLI commands |
| III. Code Quality & Maintainability | PASS | New code goes in `internal/config/` (schema, validation, migration); CLI wiring in `cmd/apx/commands/config.go` |
| IV. Developer Experience First | PASS | Clear validation errors with field paths, remediation hints; `migrate` provides automated upgrade |
| V. Canonical Import Paths | N/A | This feature is about config files, not Go import paths |
| VI. Git Subtree Publishing | N/A | Not relevant to config schema |
| VII. Multi-Format Schema Support | PASS | Schema validates format-specific policy sections (proto, openapi, avro, jsonschema, parquet) |
| Package Structure | PASS | Business logic in `internal/config/`, CLI in `cmd/apx/commands/` |
| Testing Coverage | PASS | Target 90%+ for `internal/config/` per constitution |

**Pre-design gate: PASS** — no violations.

## Project Structure

### Documentation (this feature)

```text
specs/006-canonical-config-schema/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output (CLI contract)
│   └── cli-contract.md
└── tasks.md             # Phase 2 output (NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
internal/config/
├── config.go            # Config struct, Load(), GetConfigPath() — EXISTING, extended
├── schema.go            # NEW: SchemaRegistry, field definitions, version map
├── schema_test.go       # NEW: Unit tests for schema definitions
├── validate.go          # NEW: Strict validation engine (unknown keys, types, enums, required)
├── validate_test.go     # NEW: Comprehensive validation test suite
├── migrate.go           # NEW: Version-to-version migration transformations
├── migrate_test.go      # NEW: Migration test suite
├── dependencies.go      # EXISTING, no changes expected
└── dependencies_test.go # EXISTING, no changes expected

cmd/apx/commands/
├── config.go            # EXISTING, extended with migrate subcommand
└── config_test.go       # NEW: testscript-based CLI tests

testdata/script/
├── config-validate.txt  # NEW: testscript for apx config validate
└── config-migrate.txt   # NEW: testscript for apx config migrate

docs/
└── cli-reference/
    └── configuration.md # NEW or UPDATED: generated schema reference
```

**Structure Decision**: Extends the existing `internal/config/` package with new files for schema definition, validation, and migration. No new packages are created. CLI wiring extends the existing `cmd/apx/commands/config.go`. This follows the constitution's package structure and keeps all config-related business logic in one package.

## Complexity Tracking

> No constitution violations. Table not needed.
