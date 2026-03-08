# Implementation Plan: First-Class External API Registration

**Branch**: `008-external-api-registration` | **Date**: 2026-03-08 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/008-external-api-registration/spec.md`

## Summary

Add first-class external API registration to APX so teams can catalog, govern, version, and reference third-party APIs (e.g., Google APIs from `googleapis`) without rewriting upstream schema import paths. Extends the existing identity, catalog, dependency, and validation subsystems with external provenance metadata (managed repo/path, upstream repo/path, import mode, classification). Preserves APX's canonical identity model (`format/domain/name/line`) and adds new CLI workflows (`apx external register`, search filtering, inspect provenance).

## Technical Context

**Language/Version**: Go 1.26.1  
**Primary Dependencies**: cobra (CLI), yaml.v3 (config), golang.org/x/mod (semver), testify (testing), go-internal (testscript), charmbracelet/huh (interactive TUI)  
**Storage**: YAML files (apx.yaml, catalog.yaml, apx.lock)  
**Testing**: `go test` + testscript integration tests + Gitea-based GitHub simulation  
**Target Platform**: Linux, macOS, Windows (cross-platform paths via filepath)  
**Project Type**: CLI tool  
**Performance Goals**: Registration/search completes in <1s for catalogs with <1000 entries  
**Constraints**: Must coexist with existing first-party API model; no breaking changes to existing YAML schemas (additive only)  
**Scale/Scope**: Typical catalog: 10–200 modules. External APIs: 10–50 registrations per organization.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Documentation-Driven Development | PASS | Spec written first. Docs updates planned for search, inspect, catalog, dependency pages. |
| II. Cross-Platform Path Operations | PASS | All new path logic will use `filepath.Join`/`filepath.ToSlash`. External paths (upstream URLs) are always forward-slash. |
| III. Test-First Development | PASS | Unit tests for new types, testscript integration tests for CLI commands, fixture-based tests with Google API directory structures. |
| III. Code Quality & Maintainability | PASS | New code in `internal/config/` (types), `internal/catalog/` (search/display), `cmd/apx/commands/` (CLI). Each file < 300 lines. |
| IV. Developer Experience First | PASS | Smart defaults (import mode → preserve), clear provenance in search/inspect output, `--json` support. |
| V. Canonical Import Paths | PASS | External APIs in "preserve" mode explicitly bypass canonical path derivation — this is intentional and documented. "Rewrite" mode uses standard canonical derivation. |
| VI. Git Subtree Publishing Strategy | N/A | External API registration does not use subtree publishing. External snapshots are curated in managed repos via separate workflows. |
| VII. Multi-Format Schema Support | PASS | External APIs use the same format taxonomy (proto, openapi, avro, jsonschema, parquet). Import mode "preserve" applies to proto `import` statements initially; other formats addressed per format. |

**Result**: All gates PASS. No violations require justification.

## Project Structure

### Documentation (this feature)

```text
specs/008-external-api-registration/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output (CLI contract)
└── tasks.md             # Phase 2 output (not created by /speckit.plan)
```

### Source Code (repository root)

```text
internal/
├── config/
│   ├── config.go              # MODIFY: add ExternalAPIs field to Config
│   ├── external.go            # NEW: ExternalRegistration type, validation, persistence
│   ├── identity.go            # MODIFY: path derivation aware of external APIs
│   └── dependencies.go        # MODIFY: DependencyLock gains external provenance fields
├── catalog/
│   ├── generator.go           # MODIFY: Module gains external provenance fields
│   ├── search.go              # MODIFY: SearchOptions gains Origin filter
│   └── external.go            # NEW: external registration catalog operations
└── validator/
    └── external.go            # NEW: validate external registrations, import mode checks

cmd/apx/commands/
├── external.go                # NEW: `apx external` parent + `register`, `transition`
├── search.go                  # MODIFY: add --origin flag
├── show.go                    # MODIFY: display external provenance
└── inspect.go                 # MODIFY: display external provenance in identity view

testdata/
├── script/
│   └── external_*.txt         # NEW: testscript integration tests
└── golden/
    └── external/              # NEW: golden file fixtures (Google API structures)

tests/
└── integration/
    └── external_test.go       # NEW: end-to-end integration tests

docs/
├── dependencies/
│   └── external-apis.md       # NEW: documentation for external API workflows
└── cli-reference/
    └── external-commands.md   # NEW: CLI reference for external commands
```

**Structure Decision**: Follows existing APX architecture — business logic in `internal/` packages, CLI in `cmd/apx/commands/`, tests alongside code and in `testdata/script/`. No new packages needed; external API support is an extension of existing config, catalog, and validator packages.

## Complexity Tracking

No constitution violations. No complexity justifications needed.
