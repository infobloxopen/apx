# Implementation Plan: Catalog resource-type resolution (`type → module`)

**Branch**: `009-catalog-type-resolution` | **Date**: 2026-07-01 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `specs/009-catalog-type-resolution/spec.md` (WS-021 P1 / WP-A)

## Summary

apx must resolve an AIP-122 resource **type** (e.g. `iam.example.com/User`, sourced from a standard
`google.api.resource_reference.type`) to the catalog **module that serves it**, returning path
coordinates only (module ID + domain + api_line + version + lifecycle). This is a pure **catalog
index + resolution** feature: apx indexes the `google.api.resource` annotations already present in
each module's protos at catalog-generation time, carries that index in the catalog (and therefore in
the published OCI artifact), and exposes both a **library API** and a **CLI command** that fail loud
on unknown/ambiguous types. No new schema, no `apis` release, no codegen — apx stays
lifecycle-not-codegen. External/forked-imported types resolve to the **managing** module; a
declared-but-unserved type resolves with a "no serving surface" warning.

**Technical approach**: add a `ResourceTypes []string` field to the existing `catalog.Module`, populate
it during `apx catalog generate` by scanning each module's `path` directory for `.proto`
`option (google.api.resource) = { type: "..." }` annotations (a focused source-level scanner — the
repo has no protobuf toolchain dependency and adding one would breach the lifecycle-not-codegen
charter and bloat the dependency graph). Because the whole `Catalog` struct is serialized into
`catalog.yaml` and packaged verbatim into the OCI layer, the index travels to Registry/HTTP/Aggregate
sources for free. A new `internal/catalog` resolver walks all modules, builds a `type → []module`
map, and returns the single match (or a typed unknown/ambiguous error). A `apx catalog resolve <type>`
CLI command surfaces it (text + `--json`).

## Technical Context

**Language/Version**: Go 1.26.1 (`GOTOOLCHAIN=go1.26.1`)
**Primary Dependencies**: stdlib + existing repo deps (`gopkg.in/yaml.v3`, `spf13/cobra`,
`golang.org/x/mod/semver`). **No new dependency** — proto scanning is a stdlib regex/line scanner.
**Storage**: `catalog/catalog.yaml` (local) and the OCI catalog artifact on GHCR (published).
**Testing**: Go unit tests per `internal/` package; testscript CLI integration
(`testdata/script/*.txt`); existing Gitea e2e suite (`tests/e2e/`) unchanged.
**Target Platform**: cross-platform CLI (Linux/macOS/Windows) — all path ops via `filepath`.
**Project Type**: single Go CLI + internal libraries.
**Performance Goals**: catalog generation scans protos once per module dir; resolution is an in-memory
map lookup — no perf-sensitive path.
**Constraints**: lifecycle-not-codegen charter (FR-008); no new schema / no `apis` release (FR-007);
path coordinates only, no base-URL / service-registry (FR-009); fail-loud (FR-003).
**Scale/Scope**: catalogs of tens–hundreds of modules; each module dir has a handful of protos.

## Constitution Check

*GATE: Must pass before implementation. Re-checked after design.*

- **I. Documentation-Driven Development (NON-NEGOTIABLE)**: the new `apx catalog resolve` command and
  the `resource_types` catalog field are documented FIRST — `docs/dependencies/catalog-schema.md`
  (field) and `docs/cli-reference/core-commands.md` (command) are updated before/with implementation,
  and a testscript asserts the documented CLI behavior. PASS (planned).
- **II. Cross-Platform Path Operations (NON-NEGOTIABLE)**: proto scanning uses `filepath.Walk` /
  `filepath.Join`; module `path` normalized with `filepath.ToSlash` where compared as a string. PASS.
- **III. Test-First Development (NON-NEGOTIABLE)**: unit tests for the proto scanner and the resolver
  (incl. unknown, ambiguous, external/forked→managing, unserved-type warning) plus a testscript for
  the CLI. PASS (planned).
- **V. Canonical Import Paths**: N/A — no generated Go code is produced by this feature.
- **VII. Multi-format**: the resource-type index is proto-specific (only proto carries
  `google.api.resource`); non-proto modules simply contribute no types. Consistent, not a regression.

No violations → Complexity Tracking empty.

## Project Structure

### Documentation (this feature)

```text
specs/009-catalog-type-resolution/
├── spec.md              # Input (read-only)
├── plan.md              # This file
└── tasks.md             # Task breakdown ([S]/[C] tagged)
```

### Source Code (repository root)

```text
internal/catalog/
├── generator.go             # Module: add ResourceTypes []string field
├── prototypes.go            # NEW: scan a module dir for google.api.resource type annotations
├── prototypes_test.go       # NEW: scanner unit tests
├── typeindex.go             # NEW: ResolveType(cat, type) → Resolution; typed errors; BuildTypeIndex
├── typeindex_test.go        # NEW: resolver unit tests (unknown/ambiguous/external/unserved)
└── (registry.go, source.go, aggregate.go unchanged — index travels inside the Catalog struct)

cmd/apx/commands/
├── catalog.go               # register `resolve` subcommand
├── catalog_resolve.go       # NEW: `apx catalog resolve <type>` (text + --json)
└── catalog_resolve_test.go  # NEW: command/doc-parity test

testdata/script/
└── catalog_resolve.txt      # NEW: testscript — generate w/ fixture protos, resolve, unknown, ambiguous

docs/
├── dependencies/catalog-schema.md   # document `resource_types` field + how it's derived
└── cli-reference/core-commands.md   # document `apx catalog resolve`
```

**Structure Decision**: single Go CLI. Indexing lives in `internal/catalog` next to the generator and
sources it depends on; resolution is a sibling library function in the same package (so the resolver
can be called by any command AND by an external consumer that imports `internal/catalog`… see below);
the CLI is a thin `cmd/apx/commands` wrapper matching `catalog show`/`catalog search` idioms.

**Public surface for F041 (SC-005)**: the F041 `ReferenceResolver` seam is defined and consumed in
devedge-sdk, not here. apx's obligation is to expose a stable, importable resolution entry point that
a consumer can wrap. Because `internal/` is not importable across module boundaries, we expose the
resolver through a thin **exported package** `pkg/typeresolver` that re-exports the result type and a
`Resolve(source catalog.CatalogSource, resourceType string)` function over any `CatalogSource`
(Local/Registry/HTTP/Aggregate). The devedge-sdk catalog-backed `ReferenceResolver` calls
`typeresolver.Resolve(...)`. Keep this surface minimal: one function, one result struct, two sentinel
errors (`ErrUnresolved`, `ErrAmbiguous`).

## Complexity Tracking

> No constitution violations — table intentionally empty.

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| — | — | — |
