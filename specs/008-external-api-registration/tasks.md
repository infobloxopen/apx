# Tasks: First-Class External API Registration

**Input**: Design documents from `/specs/008-external-api-registration/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/cli.md, quickstart.md

**Tests**: Included per constitution principle III (Test-First Development). Unit tests for new types, testscript integration tests for CLI commands, fixture-based tests with Google API directory structures.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Create test fixtures and directory structure needed by all stories

- [X] T001 Create golden test fixtures for Google API directory structures in testdata/golden/external/

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Define all new types and struct extensions that multiple user stories depend on

**⚠️ CRITICAL**: No user story work can begin until this phase is complete. T002 must complete before T003–T007 (they reference the ExternalRegistration type).

- [X] T002 Define ExternalRegistration struct, origin/import-mode constants, and error sentinels in internal/config/external.go
- [X] T003 [P] Add ExternalAPIs field to Config struct and add 4 provenance fields (Origin, UpstreamRepo, UpstreamPath, ImportMode) to DependencyLock struct in internal/config/config.go
- [X] T004 [P] Add 5 external provenance fields (Origin, ManagedRepo, UpstreamRepo, UpstreamPath, ImportMode) to Module struct in internal/catalog/generator.go
- [X] T005 [P] Add Origin field to SearchOptions struct and SearchModulesOpts filtering in internal/catalog/search.go
- [X] T006 [P] Create external registration validation helpers (ValidateRegistration, ValidateImportMode, ValidateOrigin, ValidateRepoURL) in internal/validator/external.go
- [X] T007 [P] Update DeriveSourcePath to return managed_path for external APIs (accept optional Module or config context) in internal/config/identity.go

**Checkpoint**: All new types defined — user story implementation can now begin

---

## Phase 3: User Story 1 — Register an External API (Priority: P1) 🎯 MVP

**Goal**: Platform teams can register third-party APIs with managed/upstream coordinates and have them appear in the catalog

**Independent Test**: Register `proto/google/pubsub/v1` with managed and upstream coordinates, verify it appears in `apx external list` and in regenerated `catalog.yaml` with correct provenance metadata

### Tests for User Story 1

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [X] T008 [P] [US1] Write unit tests for ExternalRegistration validation (valid registration, missing fields, invalid API ID, invalid import_mode, invalid origin, malformed URLs) in internal/config/external_test.go
- [X] T009 [P] [US1] Write unit tests for external catalog merge (merge external into modules, duplicate ID detection, path conflict detection, first-party conflict) in internal/catalog/external_test.go

### Implementation for User Story 1

- [X] T010 [US1] Implement ExternalRegistration validation methods (Validate, defaults for import_mode and origin) in internal/config/external.go
- [X] T011 [US1] Implement ExternalRegistration CRUD operations (AddExternal, RemoveExternal, ListExternals, FindExternalByID) with apx.yaml persistence in internal/config/external.go
- [X] T012 [US1] Implement external catalog merge logic (MergeExternalAPIs: convert ExternalRegistration list to Module entries, detect ID/path conflicts, set provenance fields) in internal/catalog/external.go
- [X] T013 [US1] Integrate external merge into catalog build pipeline (call MergeExternalAPIs after ScanDirectory/GenerateFromTags) in internal/catalog/generator.go
- [X] T014 [US1] Implement `apx external` parent command with `register` and `list` subcommands per contracts/cli.md in cmd/apx/commands/external.go
- [X] T015 [US1] Register external command group in cmd/apx/commands/root.go
- [X] T016 [US1] Write testscript integration test for external register and list (register google/pubsub/v1, verify apx.yaml updated, verify list output) in testdata/script/external_register.txt

**Checkpoint**: External APIs can be registered and listed. Catalog build merges them. User Story 1 is fully functional and testable independently.

---

## Phase 4: User Story 2 — Discover and Inspect External APIs (Priority: P1)

**Goal**: Developers can find external APIs via search and view full provenance via show/inspect

**Independent Test**: With registered external APIs, run `apx search --origin external`, `apx show proto/google/pubsub/v1`, and `apx inspect identity proto/google/pubsub/v1` — verify output includes origin indicator, managed location, upstream origin, and import mode

### Tests for User Story 2

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [X] T017 [P] [US2] Write testscript integration test for search with --origin flag and show/inspect with external provenance in testdata/script/external_search.txt

### Implementation for User Story 2

- [X] T018 [P] [US2] Add --origin flag (first-party, external, forked) and [external]/[forked] tags in search output per contracts/cli.md in cmd/apx/commands/search.go
- [X] T019 [P] [US2] Add Provenance section (Origin, Import mode, Managed repo/path, Upstream repo/path) to show output for external APIs per contracts/cli.md in cmd/apx/commands/show.go
- [X] T020 [P] [US2] Add Origin, Import, Managed, Upstream lines to inspect identity output for external APIs per contracts/cli.md in cmd/apx/commands/inspect.go
- [X] T021 [US2] Write unit test for SearchModulesOpts origin filtering (first-party only, external only, forked only, all) in internal/catalog/search_test.go

**Checkpoint**: External APIs are discoverable via search and inspectable with full provenance. User Stories 1 AND 2 are both independently functional.

---

## Phase 5: User Story 3 — Depend on an External API (Priority: P2)

**Goal**: Developers can add external APIs as dependencies; lock file records provenance; import paths are preserved or rewritten per import_mode

**Independent Test**: Register an external API, run `apx dep add proto/google/pubsub/v1`, verify lock file contains origin, upstream_repo, upstream_path, import_mode fields and that repo points to managed repository

### Tests for User Story 3

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [X] T022 [US3] Write testscript integration test for dep add with external API (add dependency, verify lock file provenance, verify dep add output shows external source) in testdata/script/external_dep.txt

### Implementation for User Story 3

- [X] T023 [US3] Extend DependencyManager.Add to look up external registration metadata from catalog and populate provenance fields (Origin, UpstreamRepo, UpstreamPath, ImportMode) in lock file in internal/config/dependencies.go
- [X] T024 [US3] Update dep add command output to show external provenance ("Source: managed-repo (external, preserve)") in cmd/apx/commands/add.go

**Checkpoint**: External APIs participate in dependency resolution with full provenance tracking. User Stories 1, 2, AND 3 are all independently functional.

---

## Phase 6: User Story 5 — Version and Lifecycle External APIs (Priority: P2)

**Goal**: External APIs follow the same versioning model as first-party APIs — version tags, lifecycle states, and version queries all work identically

**Independent Test**: Register an external API with `--version v1.0.0 --lifecycle stable`, rebuild catalog, verify catalog entry shows version and lifecycle; register a second version `v1.1.0-beta.1` with lifecycle `beta`, verify latest_stable and latest_prerelease are maintained independently

### Tests for User Story 5

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [X] T025 [US5] Write testscript integration test for versioned external API (register with version/lifecycle, rebuild catalog, verify version fields in catalog output) in testdata/script/external_version.txt

### Implementation for User Story 5

- [X] T026 [US5] Ensure catalog merge maps version and lifecycle from ExternalRegistration to Module fields (Version, Lifecycle, LatestStable, LatestPrerelease) in internal/catalog/external.go

**Checkpoint**: External APIs are versioned and lifecycle-managed identically to first-party APIs.

---

## Phase 7: User Story 4 — Transition from Registered to Forked (Priority: P3)

**Goal**: Operators can transition an external API between "external" (preserve imports) and "forked" (rewrite imports), preserving upstream provenance

**Independent Test**: Register an external API as `external`/`preserve`, transition to `forked`, verify origin changes to `forked` and import_mode changes to `rewrite`; reverse transition back to `external`, verify import_mode reverts to `preserve`

### Tests for User Story 4

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [X] T027 [US4] Write testscript integration test for transition (forward transition external→forked, verify state; reverse transition forked→external, verify state; error on first-party transition; error on already-at-target) in testdata/script/external_transition.txt

### Implementation for User Story 4

- [X] T028 [US4] Implement TransitionExternal method (validate target state, update origin and import_mode, persist to apx.yaml) in internal/config/external.go
- [X] T029 [US4] Implement `apx external transition` subcommand with --to flag per contracts/cli.md in cmd/apx/commands/external.go

**Checkpoint**: Full external API lifecycle is supported — register, discover, depend, version, and transition.

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Documentation, cleanup, and end-to-end validation across all user stories

- [X] T030 [P] Write documentation for external API workflows (registration, discovery, dependencies, transition) in docs/dependencies/external-apis.md
- [X] T031 [P] Write CLI reference for external commands (register, list, transition, modified search/show/inspect flags) in docs/cli-reference/external-commands.md
- [X] T032 Run all 4 quickstart.md scenarios (A: common protos, B: Pub/Sub, C: consumer usage, D: fork transition) as end-to-end validation
- [X] T033 Code cleanup: verify all new files are <300 lines, run go vet and linting, ensure cross-platform path usage (filepath.Join/filepath.ToSlash)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion — BLOCKS all user stories
  - T002 must complete before T003–T007 (they reference ExternalRegistration type)
  - T003–T007 can proceed in parallel (different files)
- **User Stories (Phases 3–7)**: All depend on Foundational phase completion
  - US1 and US2 are both P1 and can proceed in parallel (different files)
  - US3 and US5 are both P2 — can start after US1/US2 or in parallel with them
  - US4 (P3) can start after Foundational but benefits from US1 being complete
- **Polish (Phase 8)**: Depends on all user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) — No dependencies on other stories
- **User Story 2 (P1)**: Can start after Foundational (Phase 2) — Benefits from US1 for test fixtures but independently testable
- **User Story 3 (P2)**: Can start after Foundational (Phase 2) — Integrates with US1 catalog data but independently testable
- **User Story 5 (P2)**: Can start after Foundational (Phase 2) — Extends US1 catalog merge but independently testable
- **User Story 4 (P3)**: Can start after Foundational (Phase 2) — Extends US1 registration CRUD but independently testable

### Within Each User Story

- Tests MUST be written and FAIL before implementation
- Type/struct definitions before business logic
- Business logic before CLI commands
- Core implementation before integration
- Story complete before moving to next priority

### Parallel Opportunities

- T003–T007 (Foundational struct extensions) — all different files
- T008 and T009 (US1 tests) — different packages
- T017 (US2 testscript), T018, T019, T020 (US2 command modifications) — different files
- T030 and T031 (documentation) — different files
- US1 and US2 can be worked on simultaneously by different developers
- US3 and US5 can be worked on simultaneously

---

## Parallel Example: User Story 1

```bash
# Launch US1 tests together (must FAIL initially):
Task T008: "Unit tests for ExternalRegistration validation in internal/config/external_test.go"
Task T009: "Unit tests for catalog merge in internal/catalog/external_test.go"

# After tests written, implement sequentially:
Task T010: "ExternalRegistration validation methods"
Task T011: "ExternalRegistration CRUD operations"
Task T012: "External catalog merge logic"
Task T013: "Catalog build pipeline integration"
Task T014: "apx external register + list commands"
Task T015: "Register command group in root.go"

# Verify with integration test:
Task T016: "Testscript integration test for register/list"
```

## Parallel Example: User Story 2

```bash
# Write integration test first:
Task T017: "Testscript for search/show/inspect with external APIs"

# Launch all command modifications in parallel (different files):
Task T018: "Search command --origin flag in search.go"
Task T019: "Show command provenance section in show.go"
Task T020: "Inspect command provenance lines in inspect.go"

# Unit test for search filtering:
Task T021: "SearchModulesOpts origin filtering in search_test.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (golden fixtures)
2. Complete Phase 2: Foundational (all type definitions)
3. Complete Phase 3: User Story 1 (register + list + catalog merge)
4. **STOP and VALIDATE**: Register `proto/google/pubsub/v1`, list it, rebuild catalog — all work independently
5. This is already useful — external APIs are cataloged

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add User Story 1 → Test independently → MVP (external APIs registered and cataloged)
3. Add User Story 2 → Test independently → APIs discoverable via search/show/inspect
4. Add User Story 3 → Test independently → External APIs as dependencies with provenance
5. Add User Story 5 → Test independently → Versioned external APIs
6. Add User Story 4 → Test independently → Full lifecycle with fork/unfork transition
7. Each story adds value without breaking previous stories

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together
2. Once Foundational is done:
   - Developer A: User Story 1 (register/list/catalog merge)
   - Developer B: User Story 2 (search/show/inspect modifications)
3. After US1 + US2:
   - Developer A: User Story 3 (dependency provenance)
   - Developer B: User Story 5 (versioning) + User Story 4 (transition)
4. Stories complete and integrate independently

---

## Entity → Task Mapping

| Entity (data-model.md) | Foundational Task | Story Tasks |
|-------------------------|-------------------|-------------|
| ExternalRegistration | T002 (struct) | T008, T010, T011 (US1), T028 (US4) |
| Module (extended) | T004 (fields) | T012, T013 (US1), T018–T020 (US2) |
| DependencyLock (extended) | T003 (fields) | T023 (US3) |
| SearchOptions (extended) | T005 (field) | T018, T021 (US2) |
| Config (extended) | T003 (field) | T011, T014 (US1) |

## Contract → Task Mapping

| Contract (cli.md) | Task |
|--------------------|------|
| `apx external register` | T014 (command), T010 (validation), T011 (persistence) |
| `apx external list` | T014 (command), T011 (ListExternals) |
| `apx external transition` | T029 (command), T028 (logic) |
| `apx search --origin` | T018 (flag), T005 (filter field), T021 (filter test) |
| `apx show` provenance | T019 |
| `apx inspect identity` provenance | T020 |
| `apx catalog build` merge | T012 (merge), T013 (pipeline) |
| `apx dep add` provenance | T023 (manager), T024 (output) |

## Research Decision → Task Mapping

| Decision (research.md) | Task |
|-------------------------|------|
| D1: API ID ≠ filesystem path | T007 (DeriveSourcePath), T002 (managed_path field) |
| D2: Flat Module fields | T004 |
| D3: Import mode semantics | T006, T010 |
| D4: external_apis in apx.yaml | T003, T011 |
| D5: Catalog merge | T012, T013 |
| D6: DependencyLock extension | T003, T023 |
| D7: CLI command design | T014, T018–T020, T029 |
| D8: Google API dependency chain | T001 (fixtures) |
| D9: Validation rules | T006, T008, T010 |
| D10: Effective source path | T007 |

---

## Notes

- [P] tasks = different files, no unresolved dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Verify tests fail before implementing
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- All new files must stay under 300 lines per constitution
- Use filepath.Join/filepath.ToSlash for all path operations per constitution principle II
