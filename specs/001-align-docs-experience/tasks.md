# Tasks: Docs-Aligned APX Experience

**Input**: Design documents from `/specs/001-align-docs-experience/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Tests**: Tests are included per TDD constitution requirement.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `- [ ] [ID] [P?] [Story?] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3)
- Include exact file paths in descriptions

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure

- [X] T001 Verify existing project structure matches plan.md layout (cmd/apx/, internal/, testdata/, tests/)
- [X] T002 [P] Create internal/validator/ package directory for format-specific validators
- [X] T003 [P] Add apx.lock schema definition to internal/config/config.go
- [X] T004 [P] Create testdata/golden/ directory for doc parity fixtures
- [X] T005 [P] Update .gitignore to ensure internal/gen/** remains untracked

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T006 Implement toolchain profile resolver in internal/validator/toolchain.go
- [X] T007 [P] Create internal/validator/proto.go for buf integration (lint, breaking)
- [X] T008 [P] Create internal/validator/openapi.go for spectral/oasdiff integration
- [X] T009 [P] Create internal/validator/avro.go for avro compatibility checks
- [X] T010 [P] Create internal/validator/jsonschema.go for jsonschema-diff integration
- [X] T011 [P] Create internal/validator/parquet.go for parquet schema validation
- [X] T012 Implement validator facade in internal/validator/validator.go routing by format
- [X] T013 Create internal/publisher/ package directory and subtree.go stub
- [X] T014 [P] Create internal/publisher/pr.go for GitHub PR creation
- [X] T015 [P] Create internal/publisher/tags.go for tag naming and creation
- [X] T016 Implement catalog generation helpers in internal/catalog/generator.go
- [X] T017 [P] Create internal/overlay/manager.go for go.work overlay operations
- [X] T018 [P] Add FetchCommand to cmd/apx/commands/fetch.go for toolchain hydration
- [X] T019 [P] Add SyncCommand to cmd/apx/commands/sync.go for overlay management
- [X] T020 Create Makefile targets: install-tools, up-gitea, reset-gitea, test-integration

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - Bootstrap Canonical API Workspace (Priority: P1) 🎯 MVP

**Goal**: Administrators scaffold canonical repository structure matching `/docs/getting-started/quickstart.md`

**Independent Test**: Run `apx init canonical --org=<org>` and verify generated files match documentation fixtures

### Tests for User Story 1

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [X] T021 [P] [US1] Unit test for canonical scaffold logic in internal/schema/canonical_test.go
- [X] T022 [P] [US1] Testscript scenario in testdata/script/init_canonical.txt verifying file structure
- [X] T023 [P] [US1] Golden fixture comparison test in cmd/apx/commands/init_test.go for CLI output
- [X] T024 [US1] Integration test in tests/integration/canonical_bootstrap_test.go with Gitea repo

### Implementation for User Story 1

- [X] T025 [P] [US1] Extend InitCommand in cmd/apx/commands/init.go to accept `canonical` subcommand
- [X] T026 [US1] Implement canonical scaffolding in internal/schema/canonical.go (depends on T025)
- [X] T027 [P] [US1] Create buf.yaml template generator in internal/schema/templates/buf.go
- [X] T028 [P] [US1] Create CODEOWNERS template generator in internal/schema/templates/codeowners.go
- [X] T029 [P] [US1] Create catalog.yaml template generator in internal/schema/templates/catalog.go
- [X] T030 [US1] Wire canonical mode into interactive setup in internal/interactive/setup.go
- [X] T031 [US1] Add protection rule guidance output to canonical scaffolding flow
- [X] T032 [US1] Update help text and examples in cmd/apx/commands/init.go for canonical mode

**Checkpoint**: At this point, User Story 1 should be fully functional and testable independently

---

## Phase 4: User Story 2 - Author & Publish an API Schema (Priority: P1)

**Goal**: Producers author, validate, and publish schemas to canonical repo via documented workflow

**Independent Test**: Follow quickstart.md publish flow with payments/ledger and verify GitHub PR matches docs

### Tests for User Story 2

- [X] T033 [P] [US2] Unit test for app scaffolding in internal/schema/app_test.go
- [X] T034 [P] [US2] Unit test for lint routing in internal/validator/validator_test.go
- [X] T035 [P] [US2] Unit test for breaking change detection in internal/validator/proto_test.go
- [X] T036 [P] [US2] Testscript for `apx init app` in testdata/script/init_app.txt
- [X] T037 [P] [US2] Testscript for `apx lint` workflow in testdata/script/lint_proto.txt
- [X] T038 [P] [US2] Testscript for `apx breaking` workflow in testdata/script/breaking_proto.txt
- [X] T039 [P] [US2] Testscript for `apx publish` in testdata/script/publish_ledger.txt
- [X] T040 [US2] Integration test for git subtree publishing in tests/integration/publish_workflow_test.go with Gitea

### Implementation for User Story 2

- [X] T041 [P] [US2] Extend InitCommand in cmd/apx/commands/init.go to accept `app` subcommand
- [X] T042 [US2] Implement app scaffolding in internal/schema/app.go (depends on T041)
- [X] T043 [P] [US2] Wire format detection into lint command in cmd/apx/commands/lint.go
- [X] T044 [US2] Implement lintAction to call internal/validator facade (depends on T012, T043)
- [X] T045 [P] [US2] Wire format detection into breaking command in cmd/apx/commands/breaking.go
- [X] T046 [US2] Implement breakingAction to call internal/validator facade (depends on T012, T045)
- [X] T047 [P] [US2] Implement git subtree split in internal/publisher/subtree.go
- [X] T048 [P] [US2] Implement PR creation logic in internal/publisher/pr.go for GitHub/Gitea
- [X] T049 [US2] Wire publish command in cmd/apx/commands/publish.go to call internal/publisher
- [X] T050 [US2] Add tag creation and validation to internal/publisher/tags.go
- [X] T051 [US2] Handle OpenAPI format workflow in lint/breaking/publish commands
- [ ] T052 [US2] Add offline/air-gapped mode support via apx fetch to toolchain resolver

**Checkpoint**: At this point, User Stories 1 AND 2 should both work independently

---

## Phase 5: User Story 3 - Consume a Published API with Canonical Imports (Priority: P2)

**Goal**: Consumers discover APIs, generate overlays, and transition to published modules without import changes

**Independent Test**: Execute `apx search`, `apx add`, `apx gen`, `apx sync`, `apx unlink` and verify no import edits needed

### Tests for User Story 3

- [X] T053 [P] [US3] Unit test for catalog search in internal/catalog/search_test.go
- [X] T054 [P] [US3] Unit test for dependency addition in internal/config/dependencies_test.go
- [X] T055 [P] [US3] Unit test for overlay generation in internal/overlay/manager_test.go
- [X] T056 [P] [US3] Testscript for `apx search` in testdata/script/search_catalog.txt
- [X] T057 [P] [US3] Testscript for `apx add` in testdata/script/add_dependency.txt
- [X] T058 [P] [US3] Testscript for `apx gen go` in testdata/script/gen_go_overlay.txt
- [X] T059 [P] [US3] Testscript for `apx sync` in testdata/script/sync_overlays.txt
- [X] T060 [P] [US3] Testscript for `apx unlink` in testdata/script/unlink_overlay.txt
- [X] T061 [US3] Integration test validating import stability in tests/integration/consumer_workflow_test.go

### Implementation for User Story 3

- [X] T062 [P] [US3] Create SearchCommand in cmd/apx/commands/search.go
- [X] T063 [P] [US3] Implement catalog search in internal/catalog/search.go
- [X] T064 [US3] Wire SearchCommand to catalog search (depends on T062, T063)
- [X] T065 [P] [US3] Create AddCommand in cmd/apx/commands/add.go
- [X] T066 [P] [US3] Implement dependency addition in internal/config/dependencies.go
- [X] T067 [US3] Wire AddCommand to update apx.lock (depends on T065, T066)
- [X] T068 [P] [US3] Implement code generation dispatcher in internal/gen/generator.go
- [X] T069 [P] [US3] Wire GenCommand in cmd/apx/commands/gen.go to generation dispatcher
- [X] T070 [US3] Implement go.work overlay creation in internal/overlay/manager.go
- [X] T071 [US3] Wire SyncCommand to overlay manager (depends on T019, T070)
- [X] T072 [P] [US3] Create UnlinkCommand in cmd/apx/commands/unlink.go
- [X] T073 [US3] Implement overlay removal in internal/overlay/manager.go unlink method
- [X] T074 [US3] Wire UnlinkCommand to overlay removal (depends on T072, T073)
- [X] T075 [US3] Add catalog update on publish to internal/publisher/catalog.go
- [X] T076 [US3] Support Python and Java codegen in internal/gen/generator.go

**Checkpoint**: All user stories should now be independently functional

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [X] T077 [P] Create doc parity test harness in cmd/apx/commands/doc_parity_test.go
- [X] T078 [P] Generate golden fixtures from `/docs/getting-started/quickstart.md` into testdata/golden/
- [X] T079 Implement doc drift detection comparing CLI outputs to golden fixtures
- [ ] T080 [P] Add --json flag support to all commands for CI automation
- [ ] T081 [P] Add comprehensive error messages with context wrapping per constitution
- [ ] T082 [P] Add GitHub Enterprise Server endpoint configuration to apx.yaml
- [ ] T083 [P] Implement air-gapped bundle validation with checksums in internal/validator/toolchain.go
- [ ] T084 [P] Add performance instrumentation to validation commands (<5s goal)
- [ ] T085 [P] Update `/docs/getting-started/quickstart.md` with any discovered refinements
- [ ] T086 [P] Create user-facing troubleshooting guide based on quickstart.md edge cases
- [X] T087 Run full test suite per quickstart.md validation matrix
- [ ] T088 Verify constitution compliance across all implemented commands
- [X] T089 [P] Add CHANGELOG.md entries for new commands and workflows
- [X] T090 Update README.md with feature completion status

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3-5)**: All depend on Foundational phase completion
  - User Story 1 (P1): Can start after Foundational - No dependencies on other stories
  - User Story 2 (P1): Can start after Foundational - No dependencies on other stories
  - User Story 3 (P2): Can start after Foundational - Benefits from US2 published APIs but independently testable
- **Polish (Phase 6)**: Depends on all user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Independently implements `apx init canonical`
- **User Story 2 (P1)**: Independently implements `apx init app`, `apx lint`, `apx breaking`, `apx publish`
- **User Story 3 (P2)**: Independently implements `apx search`, `apx add`, `apx gen`, `apx sync`, `apx unlink`

### Within Each User Story

- Tests MUST be written and FAIL before implementation (TDD requirement)
- Unit tests can run in parallel
- Testscript scenarios can run in parallel
- Integration tests run after unit/testscript validation
- CLI command wiring after business logic implementation
- Interactive mode wiring after non-interactive validation

### Parallel Opportunities

**Phase 2 (Foundational)**:
- T007-T011 (all format validators in parallel)
- T014-T015 (publisher components in parallel)
- T018-T019 (new commands in parallel)

**User Story 1 Tests**:
- T021-T023 (unit, testscript, golden tests in parallel)

**User Story 1 Implementation**:
- T027-T029 (template generators in parallel)

**User Story 2 Tests**:
- T033-T039 (all unit and testscript tests in parallel)

**User Story 2 Implementation**:
- T043, T045 (format detection in parallel)
- T047-T048 (subtree and PR logic in parallel)

**User Story 3 Tests**:
- T053-T060 (all unit and testscript tests in parallel)

**User Story 3 Implementation**:
- T062-T063 (search command and logic in parallel)
- T065-T066 (add command and logic in parallel)
- T068-T069 (gen command components in parallel)
- T072-T073 (unlink command and logic in parallel)

**Polish Phase**:
- T077-T090 (most polish tasks can run in parallel)

---

## Parallel Example: User Story 2

```bash
# Launch all tests for User Story 2 together:
Task: "Unit test for app scaffolding in internal/schema/app_test.go"
Task: "Unit test for lint routing in internal/validator/validator_test.go"
Task: "Unit test for breaking change detection in internal/validator/proto_test.go"
Task: "Testscript for `apx init app` in testdata/script/init_app.txt"
Task: "Testscript for `apx lint` workflow in testdata/script/lint_proto.txt"

# Launch parallel implementation tasks:
Task: "Wire format detection into lint command in cmd/apx/commands/lint.go"
Task: "Wire format detection into breaking command in cmd/apx/commands/breaking.go"
Task: "Implement git subtree split in internal/publisher/subtree.go"
Task: "Implement PR creation logic in internal/publisher/pr.go"
```

---

## Implementation Strategy

### MVP First (User Stories 1 & 2 Only)

1. Complete Phase 1: Setup (T001-T005)
2. Complete Phase 2: Foundational (T006-T020) - CRITICAL blocking phase
3. Complete Phase 3: User Story 1 (T021-T032)
   - **STOP and VALIDATE**: Test canonical bootstrap independently
4. Complete Phase 4: User Story 2 (T033-T052)
   - **STOP and VALIDATE**: Test publish workflow independently
5. Deploy/demo if ready (covers 95% of producer workflows per success criteria)

### Incremental Delivery

1. Setup + Foundational → Foundation ready
2. Add User Story 1 → Test independently → Deploy/Demo (canonical repos)
3. Add User Story 2 → Test independently → Deploy/Demo (publish workflows) ✅ MVP
4. Add User Story 3 → Test independently → Deploy/Demo (consumer experience)
5. Polish phase → Final hardening and doc parity validation

### Parallel Team Strategy

With multiple developers (after Foundational phase completes):

1. Team completes Setup + Foundational together (T001-T020)
2. Once Foundational is done:
   - **Developer A**: User Story 1 (T021-T032) - Canonical bootstrap
   - **Developer B**: User Story 2 (T033-T052) - Publish workflow
   - **Developer C**: User Story 3 (T053-T076) - Consumer experience
3. Converge on Polish phase (T077-T090)

Stories integrate cleanly due to focused internal package boundaries.

---

## Notes

- All tasks follow TDD: tests written first, fail, then implement
- [P] tasks target different files and have no cross-dependencies
- [Story] labels enable independent story completion and validation
- Canonical import path enforcement runs through all phases
- Doc parity validation (T077-T079) ensures CLI matches `/docs/`
- Constitution adherence verified in T088 before completion
- Gitea integration tests validate GitHub workflows without external dependencies
- Offline/air-gapped support built into toolchain resolution (T006, T083)
