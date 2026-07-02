# Tasks: Catalog resource-type resolution (`type → module`)

**Input**: `specs/009-catalog-type-resolution/{spec.md,plan.md}` (WS-021 P1 / WP-A)
**Prerequisites**: plan.md (done), spec.md (done)

**Tests**: REQUESTED by the spec (US1/US2/US3 Independent Tests + SC-005). Test-First is a
constitution NON-NEGOTIABLE, so test tasks are included and written before/with implementation.

## Format: `[ID] [P?] [Story] [S|C] Description`

- **[P]**: can run in parallel (different files, no dependency)
- **[Story]**: US1 (index), US2 (resolve), US3 (artifact parity), SC5 (F041 seam), DOC/POL cross-cut
- **[S]/[C]**: `[S]` simple/mechanical (Sonnet-eligible) · `[C]` complex (Opus)

---

## Phase 1: Docs-first (Documentation-Driven Development, NON-NEGOTIABLE)

- [ ] T001 [DOC] [S] Document the `resource_types` module field + "derived from `google.api.resource`
  at generate time" in `docs/dependencies/catalog-schema.md` (Identity/Metadata table + generation
  section). Traces FR-001, FR-007.
- [ ] T002 [DOC] [C] Document `apx catalog resolve <type>` (synopsis, flags `--catalog`/`--json`,
  fail-loud unknown/ambiguous, unserved-type warning, path-coordinates-only note) in
  `docs/cli-reference/core-commands.md`. Traces FR-002, FR-003, FR-005, FR-009.

---

## Phase 2: Foundational — catalog data model (BLOCKS all stories)

- [ ] T003 [US1] [S] Add `ResourceTypes []string \`yaml:"resource_types,omitempty"\`` to
  `catalog.Module` in `internal/catalog/generator.go`. Traces FR-001, Key Entities.

**Checkpoint**: the catalog struct can carry the index; it serializes to YAML and (unchanged) into the
OCI layer → FR-004 satisfied structurally.

---

## Phase 3: User Story 1 — index resource types per module (P1) 🎯 MVP

**Goal**: `apx catalog generate` records each module's declared AIP-122 types, no manual entry.
**Independent Test**: generate a catalog for a fixture repo whose protos declare types across two
modules; assert each type is indexed against its owning module.

- [ ] T004 [P] [US1] [C] `internal/catalog/prototypes_test.go`: scanner unit tests — single resource,
  multiple resources in one dir, no-annotation proto (no entry), commented-out/`//`-ignored lines,
  multi-line `option (google.api.resource) = { ... type: "svc/Kind" ... }`, dedupe. FAIL first.
- [ ] T005 [US1] [C] `internal/catalog/prototypes.go`: `ScanResourceTypes(dir string) ([]string, error)`
  — walk `.proto` files under dir, extract `google.api.resource` `type:` values, strip comments,
  dedupe + sort, `filepath`-safe, missing dir → empty (not error). Traces FR-001, FR-008 (scan only,
  no compile/codegen).
- [ ] T006 [US1] [S] Wire scanning into generation: in `catalogGenerateAction`
  (`cmd/apx/commands/catalog.go`), for each proto module resolve its on-disk dir from `--dir` + module
  `Path` and set `m.ResourceTypes`. Only when the module dir exists locally (CI/canonical repo case);
  remote/sourced modules keep whatever the source provided. Traces FR-001, FR-007.
- [ ] T007 [US1] [S] Extend `testdata/script/catalog_generate.txt` (or add fixture protos under the
  test canonical repo) to assert `resource_types:` appears for an annotated module and is absent for
  a module with no annotation. Traces US1 scenarios 1–3.

**Checkpoint**: catalog generation populates the type index from existing annotations, zero manual entry.

---

## Phase 4: User Story 2 — resolve a type to its serving module (P1)

**Goal**: library API + CLI map `type → module coordinates`; fail loud on unknown/ambiguous.
**Independent Test**: resolve a known type → coordinates; unknown → clear error; two claimants → ambiguous.

- [ ] T008 [P] [US2] [C] `internal/catalog/typeindex_test.go`: resolver unit tests — happy path (one
  module → full coordinates incl. lifecycle), unknown type → `ErrUnresolved` naming the type, two
  modules claiming one type → `ErrAmbiguous` listing claimants, deprecated/sunset lifecycle surfaced
  (US2 s4), external/forked module resolves to `ManagedRepo` (FR-006), declared-but-unserved type →
  success + `NoServingSurface` warning (Edge case). FAIL first.
- [ ] T009 [US2] [C] `internal/catalog/typeindex.go`:
  - `Resolution` struct: `Type, ModuleID, Domain, APILine, Version, Lifecycle, Origin, ManagedRepo string;
    Warning string` (Warning="no serving surface" when the owning module has no serving surface).
  - `BuildTypeIndex(cat *Catalog) map[string][]Module` (each type → claiming modules; dedupe by module ID).
  - `ResolveType(cat *Catalog, resourceType string) (*Resolution, error)`: 0 → `ErrUnresolved`,
    >1 distinct claimant → `ErrAmbiguous`, exactly 1 → `Resolution`. For external/forked, populate
    `ManagedRepo` so the consumer calls the curated surface. Sentinel errors `ErrUnresolved`,
    `ErrAmbiguous` (wrapped with the type name / claimant list). Traces FR-002, FR-003, FR-005, FR-006.
- [ ] T010 [US2] [C] `cmd/apx/commands/catalog_resolve.go`: `apx catalog resolve <type>` — resolve via
  `resolveCatalogSource` (same as search/show), print coordinates (text + `--json`), print the
  serving-surface warning to stderr via `ui.Warning`, return a non-zero error on unknown/ambiguous.
  Register in `catalog.go`. Matches documented behavior from T002. Traces FR-002, FR-003, FR-009.
- [ ] T011 [P] [US2] [S] `cmd/apx/commands/catalog_resolve_test.go`: command test asserting text/JSON
  output + non-zero exit on unknown/ambiguous, aligned with the T002 docs (doc-parity style).
- [ ] T012 [US2] [S] `testdata/script/catalog_resolve.txt`: end-to-end testscript — generate a catalog
  with annotated fixtures, `apx catalog resolve <known>` → coordinates, `<unknown>` → error, ambiguous
  → error. Traces US2 Independent Test.

**Checkpoint**: US1+US2 fully functional locally.

---

## Phase 5: User Story 3 — carry the index in the published artifact (P2)

**Goal**: Registry/HTTP/Aggregate resolve identically to Local.
**Independent Test**: publish → pull via RegistrySource → resolve → byte-parity with LocalSource.

- [ ] T013 [US3] [C] `internal/catalog/typeindex_test.go` (add cases): resolve over a catalog loaded
  from a RegistrySource-style round-trip (reuse `createTarGz` + `extractCatalog` fixture from
  `registry_test.go`) and assert the `Resolution` equals the LocalSource resolution for the same type;
  AggregateSource merge preserves per-module types and a cross-catalog collision surfaces as ambiguous.
  Traces FR-004, SC-003, US3 scenarios 1–2. (No production code change expected — the index rides
  inside the serialized `Catalog`; this test PROVES parity.)

**Checkpoint**: cross-team resolution proven; if this test needs a code change, that change is [C].

---

## Phase 6: SC-005 — expose resolver to external consumer (F041 seam)

**Goal**: devedge-sdk's catalog-backed `ReferenceResolver` can import and call apx's resolver.

- [ ] T014 [SC5] [C] `pkg/typeresolver/typeresolver.go`: exported thin wrapper —
  `Resolve(src catalog.CatalogSource, resourceType string) (*catalog.Resolution, error)` (loads the
  catalog from the source, delegates to `catalog.ResolveType`), re-export `Resolution` +
  `ErrUnresolved`/`ErrAmbiguous`. Minimal surface (SC-005; keep clean). Traces SC-005.
- [ ] T015 [P] [SC5] [S] `pkg/typeresolver/typeresolver_test.go`: prove an external-style caller can
  resolve a known type from a LocalSource and receives the same `Resolution`, and gets the sentinel
  errors on unknown/ambiguous (matches F041's fail-loud contract).

---

## Phase 7: Polish & gates

- [ ] T016 [POL] [S] `gofmt`, `go vet ./...`, `go build ./...`, `go test ./...` all green
  (`GOTOOLCHAIN=go1.26.1`). Run repo lint if present.
- [ ] T017 [POL] [S] Scope diff: every change traces to an FR/SC/task; no base-URL hint, no service
  registry, no new schema, no proto-compiler dependency (FR-007/008/009).

---

## Dependencies & Execution Order

- **Phase 1 (docs)** first per Documentation-Driven Development.
- **Phase 2 (T003)** blocks everything (adds the field).
- **US1 (P3)** before **US2 (P4)** in practice (resolution needs an index), though US2 tests can use a
  hand-built catalog fixture and run independently.
- **US3 (P5)** and **SC5 (P6)** depend on the resolver (US2) existing.
- **Polish (P7)** last.

### Parallel opportunities
- T004 (scanner test) ∥ T008 (resolver test) — different files.
- T011, T015 marked [P].

## Notes
- `[C]` = the annotation scanner and resolver semantics (comment-stripping, multi-line options,
  ambiguity/dedup, external→managing, unserved-warning) carry correctness risk → Opus.
- `[S]` = struct field, CLI wiring following existing idioms, testscript, docs prose, gate runs.
