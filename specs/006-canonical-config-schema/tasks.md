# Tasks: Canonical APX Configuration Model

**Input**: Design documents from `/specs/006-canonical-config-schema/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/cli-contract.md, quickstart.md

## Format: `[ID] [P?] [Story?] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3, US4)
- Include exact file paths in descriptions

---

## Phase 1: Setup

**Purpose**: Define the canonical schema data structures that all subsequent phases depend on

- [ ] T001 Define FieldType enum and FieldDef struct in internal/config/schema.go
- [ ] T002 Define SchemaVersion, SchemaRegistry, and CurrentVersion constant in internal/config/schema.go
- [ ] T003 Build complete v1 field-definition tree (all fields from data-model.md field map) in internal/config/schema.go
- [ ] T004 Implement DefaultConfig() factory function that returns a valid Config with v1 defaults in internal/config/schema.go

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Extend ValidationError and ValidationResult types used by all user stories

**CRITICAL**: No user story work can begin until this phase is complete

- [ ] T005 Extend ValidationError struct with Kind, Line, and Hint fields in internal/config/config.go
- [ ] T006 Define ErrorKind enum (Missing, InvalidType, InvalidValue, UnknownKey, Deprecated) in internal/config/config.go
- [ ] T007 Define ValidationResult struct (Errors, Warnings, Valid) in internal/config/config.go
- [ ] T008 Add MarshalJSON methods to ValidationError and ValidationResult for --json output in internal/config/config.go

**Checkpoint**: Foundation ready — core types defined; user story implementation can begin

---

## Phase 3: User Story 1 — Validate Existing apx.yaml (Priority: P1) MVP

**Goal**: `apx config validate` reports all schema violations with field paths, line numbers, types, and remediation hints

**Independent Test**: Run `apx config validate` against valid and intentionally malformed apx.yaml files; every violation produces a structured error

### Implementation for User Story 1

- [ ] T009 [US1] Implement parseYAMLNode() that reads a file into *yaml.Node with syntax error handling in internal/config/validate.go
- [ ] T010 [US1] Implement extractVersion() that reads the version field from a yaml.Node tree in internal/config/validate.go
- [ ] T011 [US1] Implement walkNode() recursive validator that checks required fields, types, and unknown keys against FieldDef tree in internal/config/validate.go
- [ ] T012 [US1] Implement enum value validation within walkNode for fields with EnumValues in internal/config/validate.go
- [ ] T013 [US1] Implement pattern validation within walkNode for fields with Pattern (e.g., publishing.tag_format must contain {version}) in internal/config/validate.go
- [ ] T014 [US1] Implement deprecated-field detection within walkNode that emits warnings (not errors) in internal/config/validate.go
- [ ] T015 [US1] Implement ValidateFile() public function that orchestrates parse → version lookup → walk → return ValidationResult in internal/config/validate.go
- [ ] T016 [US1] Add unit tests for ValidateFile: valid minimal file, valid full file, missing required fields in internal/config/validate_test.go
- [ ] T017 [US1] Add unit tests for ValidateFile: unknown top-level key, unknown nested key, wrong type, invalid enum value in internal/config/validate_test.go
- [ ] T018 [US1] Add unit tests for ValidateFile: empty file, whitespace-only file, unsupported version, version higher than current in internal/config/validate_test.go
- [ ] T019 [US1] Add unit tests for ValidateFile: deprecated field emits warning not error, pattern validation on tag_format in internal/config/validate_test.go
- [ ] T020 [US1] Update newConfigValidateCmd() to call ValidateFile(), render errors/warnings with ui package, support --json output in cmd/apx/commands/config.go
- [ ] T021 [US1] Add testscript integration test for apx config validate (valid file, missing field, unknown key, --json output) in testdata/script/config-validate.txt
- [ ] T022 [US1] Validate apx.example.yaml passes ValidateFile in internal/config/validate_test.go

**Checkpoint**: `apx config validate` is fully functional with strict schema checking, field-path errors, and JSON output

---

## Phase 4: User Story 2 — Init Generates Conforming apx.yaml (Priority: P1)

**Goal**: Every `apx init` variant produces an apx.yaml that passes `apx config validate` without modification

**Independent Test**: Run `apx init canonical` and `apx init app` in clean dirs; load each generated apx.yaml through ValidateFile; both pass

### Implementation for User Story 2

- [ ] T023 [US2] Implement MarshalConfig() function that serializes a Config struct to YAML bytes in internal/config/config.go
- [ ] T024 [US2] Refactor config.Init() to use DefaultConfig() + MarshalConfig() instead of inline string template in internal/config/config.go
- [ ] T025 [P] [US2] Refactor schema.Initializer.createConfigWithDefaults() to use config.DefaultConfig() + customize + config.MarshalConfig() in internal/schema/init.go
- [ ] T026 [P] [US2] Refactor schema.AppScaffolder.generateApxYaml() to use config.DefaultConfig() + customize + config.MarshalConfig() in internal/schema/app.go
- [ ] T027 [US2] Add unit test: config.Init() output passes ValidateFile in internal/config/validate_test.go
- [ ] T028 [P] [US2] Add unit test: DefaultConfig() produces a Config that round-trips through MarshalConfig+Load without error in internal/config/schema_test.go
- [ ] T029 [US2] Add testscript integration test for apx init canonical + apx config validate in testdata/script/config-validate.txt

**Checkpoint**: All init code paths emit schema-compliant YAML from the single Config struct

---

## Phase 5: User Story 3 — Migrate Older apx.yaml (Priority: P2)

**Goal**: `apx config migrate` upgrades an apx.yaml from any prior version to current, preserving comments, backing up the original, and reporting changes

**Independent Test**: Given a v1 apx.yaml with pre-change fields, run `apx config migrate`; output file passes ValidateFile; changes are listed on terminal

### Implementation for User Story 3

- [ ] T030 [US3] Define Change struct and MigrationFunc type in internal/config/migrate.go
- [ ] T031 [US3] Implement backupFile() that copies apx.yaml to apx.yaml.bak (with timestamp fallback) in internal/config/migrate.go
- [ ] T032 [US3] Implement MigrateFile() that parses yaml.Node, reads version, chains migrations, writes result in internal/config/migrate.go
- [ ] T033 [US3] Implement no-op detection: if file is already at CurrentVersion, return early with "no migration needed" in internal/config/migrate.go
- [ ] T034 [US3] Implement unsupported-version detection: if file version > CurrentVersion, return error with upgrade hint in internal/config/migrate.go
- [ ] T035 [US3] Add unit tests for MigrateFile: already-current file returns no changes in internal/config/migrate_test.go
- [ ] T036 [US3] Add unit tests for MigrateFile: unsupported version returns error in internal/config/migrate_test.go
- [ ] T037 [US3] Add unit tests for backupFile: creates .bak, handles existing .bak with timestamp in internal/config/migrate_test.go
- [ ] T038 [US3] Wire newConfigMigrateCmd() subcommand into config command with --config flag, human and --json output in cmd/apx/commands/config.go
- [ ] T039 [US3] Add testscript integration test for apx config migrate (no-op case, unsupported version case) in testdata/script/config-migrate.txt

**Checkpoint**: `apx config migrate` is functional with backup, change reporting, and edge case handling

---

## Phase 6: User Story 4 — Schema Reference Documentation (Priority: P2)

**Goal**: A configuration reference doc derived from the schema definition covers every field with type, required/optional, default, and description

**Independent Test**: Hand-craft an apx.yaml from only the reference doc; run `apx config validate`; it passes

### Implementation for User Story 4

- [ ] T040 [P] [US4] Implement GenerateSchemaDoc() that walks the FieldDef tree and emits a Markdown reference table in internal/config/schema.go
- [ ] T041 [US4] Add unit test: GenerateSchemaDoc output contains every field from v1 field map in internal/config/schema_test.go
- [ ] T042 [US4] Write docs/cli-reference/configuration.md using GenerateSchemaDoc output as the field reference section
- [ ] T043 [US4] Add unit test: every field name in docs/cli-reference/configuration.md exists in the v1 FieldDef tree (parity check) in internal/config/schema_test.go

**Checkpoint**: Schema reference docs are derived from code and verified to stay in sync

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Final quality, cleanup, and validation

- [ ] T044 [P] Integrate ValidateFile into Load() so all commands reject invalid config early in internal/config/config.go
- [ ] T045 [P] Remove legacy validateConfig() function (replaced by ValidateFile) in internal/config/config.go
- [ ] T046 Run quickstart.md validation: execute every command from specs/006-canonical-config-schema/quickstart.md and verify output
- [ ] T047 [P] Verify apx.example.yaml and cmd/apx/apx.yaml both pass apx config validate
- [ ] T048 Run full test suite (go test ./... -count=1) and confirm no regressions

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: Depends on Phase 1 (uses FieldDef, SchemaVersion types)
- **US1 Validate (Phase 3)**: Depends on Phase 2 — BLOCKS US2, US3, US4
- **US2 Init (Phase 4)**: Depends on Phase 3 (needs ValidateFile to verify output)
- **US3 Migrate (Phase 5)**: Depends on Phase 2 (uses SchemaRegistry, FieldDef); can start in parallel with US1 if staffed
- **US4 Docs (Phase 6)**: Depends on Phase 1 (needs FieldDef tree); can start in parallel with US1 if staffed
- **Polish (Phase 7)**: Depends on Phases 3, 4, 5, 6

### User Story Dependencies

- **US1 (P1)**: Foundation only — no cross-story deps. This is the MVP.
- **US2 (P1)**: Requires US1 ValidateFile() to verify generated files pass validation
- **US3 (P2)**: Requires Foundation only (SchemaRegistry). Can run in parallel with US1.
- **US4 (P2)**: Requires Foundation only (FieldDef tree). Can run in parallel with US1.

### Within Each User Story

- Core logic before CLI wiring
- Unit tests alongside implementation (TDD per constitution)
- Integration tests after CLI wiring

### Parallel Opportunities

Phase 4 (US2) parallelism:
- T025 (init.go) and T026 (app.go) can run in parallel after T023/T024 complete
- T027 (validate_test.go) and T028 (schema_test.go) can run in parallel

Phase 5 + Phase 6 parallelism:
- US3 (migrate) and US4 (docs) can run in parallel with each other once Foundation is complete

Phase 7 parallelism:
- T044, T045, T047 touch different concerns and can run in parallel

---

## Parallel Example: User Story 1

```
# Core validation engine (sequential, each builds on previous):
T009 → T010 → T011 → T012/T013/T014 → T015

# Once T015 is done, tests and CLI wiring can parallelize:
T016 + T017 + T018 + T019  (all in validate_test.go, different test functions)
T020                        (config.go CLI wiring)
T021                        (testscript)
T022                        (example file validation)
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (T001–T004) — schema definitions
2. Complete Phase 2: Foundation (T005–T008) — error types
3. Complete Phase 3: US1 Validate (T009–T022) — strict validation
4. **STOP AND VALIDATE**: `apx config validate` works end-to-end
5. This alone delivers SC-001 (actionable errors) and the core of the feature

### Incremental Delivery

1. Setup + Foundation → Schema types ready
2. Add US1 (Validate) → Test independently → MVP!
3. Add US2 (Init unification) → All init paths produce valid files → SC-002
4. Add US3 (Migrate) → Forward migration works → SC-003, SC-005
5. Add US4 (Docs) → Schema reference verified → SC-004, SC-006
6. Polish → Full integration, legacy cleanup

---

## Notes

- Total tasks: 48
- US1 (Validate): 14 tasks (MVP)
- US2 (Init): 7 tasks
- US3 (Migrate): 10 tasks
- US4 (Docs): 4 tasks
- Setup: 4 tasks
- Foundation: 4 tasks
- Polish: 5 tasks
- All tasks follow strict checklist format: checkbox, ID, optional [P], optional [Story], description with file path
- No external dependencies required; all implementation uses stdlib + gopkg.in/yaml.v3
