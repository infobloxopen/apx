---
description: "Task breakdown for E2E Integration Test Suite implementation"
---

# Tasks: End-to-End Integration Test Suite

**Feature**: 003-e2e-integration-suite  
**Input**: Design documents from `/specs/003-e2e-integration-suite/`  
**Prerequisites**: plan.md ✅, spec.md ✅, research.md ✅, data-model.md ✅, contracts/ ✅

**Tests**: Testscript scenarios ARE the tests - this feature implements comprehensive test infrastructure

**Organization**: Tasks grouped by user story to enable independent implementation and testing

## Format: `- [ ] [ID] [P?] [Story?] Description with file path`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3, US4)
- All file paths are absolute from repository root

---

## Phase 1: Setup (Project Initialization)

**Purpose**: Initialize E2E test infrastructure directories and tooling

- [x] T001 Create directory structure: tests/e2e/{gitea,k3d,testhelpers,fixtures}/ and testdata/script/e2e/
- [x] T002 Create tests/e2e/main_test.go with testscript runner setup
- [x] T003 Add Makefile targets: install-e2e-deps, test-e2e, clean-e2e in Makefile
- [x] T004 Create scripts/install-e2e-tools.sh for k3d and kubectl installation
- [x] T005 Update go.mod with testscript dependency (rogpeppe/go-internal/testscript)

---

## Phase 2: Foundational (Blocking Infrastructure)

**Purpose**: Core k3d and Gitea infrastructure that ALL user stories depend on

**⚠️ CRITICAL**: No user story implementation can begin until this phase is complete

### k3d Cluster Management

- [ ] T006 [P] Implement k3d cluster creation in tests/e2e/k3d/cluster.go
- [ ] T007 [P] Implement k3d cluster configuration templates in tests/e2e/k3d/config.go
- [ ] T008 [P] Implement k3d cleanup utilities in tests/e2e/k3d/cleanup.go

### Gitea Lifecycle Management

- [ ] T009 [P] Implement Gitea container deployment in tests/e2e/gitea/lifecycle.go
- [ ] T010 [P] Implement Gitea readiness checks and health validation in tests/e2e/gitea/lifecycle.go
- [ ] T011 [P] Implement admin token creation in tests/e2e/gitea/lifecycle.go

### Gitea API Client

- [ ] T012 [P] Implement repository operations (create, delete, get) in tests/e2e/gitea/client.go
- [ ] T013 [P] Implement pull request operations (create, get, list, commits) in tests/e2e/gitea/client.go
- [ ] T014 [P] Implement tag operations (list, get) in tests/e2e/gitea/client.go
- [ ] T015 [P] Implement user operations (create user, create token) in tests/e2e/gitea/client.go

### Test Helpers

- [ ] T016 [P] Implement git operations wrapper (clone, commit, tag, push) in tests/e2e/testhelpers/git.go
- [ ] T017 [P] Implement APX command wrappers (init, publish, add, gen, sync) in tests/e2e/testhelpers/apx.go
- [ ] T018 [P] Implement custom assertions (git history, PR state, file checks) in tests/e2e/testhelpers/assertions.go
- [ ] T019 [P] Implement environment setup helper in tests/e2e/testhelpers/environment.go

### Test Fixtures

- [ ] T020 [P] Create canonical repository seed data in tests/e2e/fixtures/canonical-repo/
- [ ] T021 [P] Create app1 payment service test schemas in tests/e2e/fixtures/app1-payment/
- [ ] T022 [P] Create app2 user service test schemas in tests/e2e/fixtures/app2-user/

### Integration with Existing Test Infrastructure

- [ ] T023 Update testscript_test.go to conditionally setup Gitea for E2E scenarios

**Checkpoint**: Foundation ready - all user story testscript scenarios can now be implemented and run

---

## Phase 3: User Story 1 - Complete Publishing Workflow Validation (Priority: P1) 🎯 MVP

**Goal**: Validate the complete schema publishing workflow from app repository creation through canonical repository publication

**Independent Test**: Run `make test-e2e` with only `e2e_complete_workflow.txt` - should bootstrap Gitea, create repositories, execute `apx init`, `apx publish`, and validate PR creation with git history preservation

### Testscript Scenario for User Story 1

- [ ] T024 [US1] Create e2e_complete_workflow.txt testscript in testdata/script/e2e/ covering all 5 acceptance scenarios:
  - Canonical repo initialization (apx init canonical)
  - App repo creation (apx init app)
  - Schema publication (apx publish)
  - Dependency consumption (apx add, apx gen go)
  - Overlay validation (import path resolution)

**Checkpoint**: User Story 1 complete and independently testable - this is the MVP for E2E testing infrastructure

---

## Phase 4: User Story 2 - Cross-Repository Dependency Chain (Priority: P2)

**Goal**: Validate that schema dependencies work across multiple application repositories (app2 consumes app1's published schema while publishing its own)

**Independent Test**: Run testscript scenario that creates app1 (publishes payment API), app2 (consumes payment API + publishes user API), and validates both overlay resolution and app2's independent publication succeeds

### Testscript Scenarios for User Story 2

- [ ] T025 [P] [US2] Create e2e_cross_repo_deps.txt testscript in testdata/script/e2e/ covering acceptance scenarios 1-3:
  - App2 adds app1 dependency (apx add, apx gen go)
  - App2 schema imports app1 schema and compiles
  - App2 publishes its own schema independently

- [ ] T026 [P] [US2] Create e2e_catalog_validation.txt testscript in testdata/script/e2e/ covering acceptance scenario 4:
  - Validate apx search shows both payment and user APIs after publication

**Checkpoint**: User Stories 1 AND 2 both work independently - can validate complex dependency scenarios

---

## Phase 5: User Story 3 - Breaking Change Detection (Priority: P3)

**Goal**: Validate that breaking change detection works when schemas evolve across app and canonical repositories

**Independent Test**: Run testscript scenario that publishes v1.0.0 of a schema, modifies it with breaking change, attempts to publish v1.1.0, and validates that `apx breaking` detects the violation

### Testscript Scenarios for User Story 3

- [ ] T027 [P] [US3] Create e2e_breaking_detection.txt testscript in testdata/script/e2e/ covering acceptance scenarios 1-2:
  - Detect breaking changes (removed field) via apx breaking
  - Block publication of breaking changes without --force flag

- [ ] T028 [P] [US3] Create e2e_non_breaking_changes.txt testscript in testdata/script/e2e/ covering acceptance scenario 3:
  - Allow non-breaking changes (added optional field)

- [ ] T029 [P] [US3] Create e2e_major_version_bump.txt testscript in testdata/script/e2e/ covering acceptance scenario 4:
  - Allow breaking changes in new major version (v2 directory)

**Checkpoint**: All priority user stories (P1-P3) are independently functional with comprehensive test coverage

---

## Phase 6: User Story 4 - Git History and Authorship Preservation (Priority: P2)

**Goal**: Verify that git subtree publishing preserves commit history and authorship attribution

**Independent Test**: Run testscript scenario that creates multiple commits in an app repository with different authors, publishes via `apx publish`, and verifies that the canonical repo PR shows all commits with original metadata

### Testscript Scenario for User Story 4

- [ ] T030 [US4] Create e2e_git_history.txt testscript in testdata/script/e2e/ covering all 3 acceptance scenarios:
  - Validate 5 commits from 3 authors preserved in canonical PR
  - Verify commit messages, authors, and timestamps intact
  - Confirm only schema directory commits included (not unrelated commits)

**Checkpoint**: All user stories (P1-P3 + history preservation) are independently testable

---

## Phase 7: Edge Cases (Cross-Cutting Concerns)

**Purpose**: Validate error handling and edge cases that span multiple user stories

### Edge Case Testscript Scenarios

- [ ] T031 [P] Create e2e_gitea_unreachable.txt testscript in testdata/script/e2e/ testing Gitea unavailability error handling
- [ ] T032 [P] Create e2e_existing_pr.txt testscript in testdata/script/e2e/ testing PR update when PR already exists
- [ ] T033 [P] Create e2e_corrupted_git_history.txt testscript in testdata/script/e2e/ testing git subtree split failure
- [ ] T034 [P] Create e2e_circular_deps.txt testscript in testdata/script/e2e/ testing circular dependency detection
- [ ] T035 [P] Create e2e_duplicate_tag.txt testscript in testdata/script/e2e/ testing tag conflict error handling
- [ ] T036 [P] Create e2e_overlay_deletion.txt testscript in testdata/script/e2e/ testing apx sync after manual overlay deletion
- [ ] T037 [P] Create e2e_concurrent_publish.txt testscript in testdata/script/e2e/ testing multiple apps publishing to same module path
- [ ] T038 [P] Create e2e_codeowners.txt testscript in testdata/script/e2e/ testing CODEOWNERS enforcement validation

**Checkpoint**: All edge cases validated - comprehensive error handling in place

---

## Phase 8: CI Integration & Documentation

**Purpose**: Enable CI execution and developer onboarding

### CI Workflow

- [ ] T039 Add E2E test job to .github/workflows/test.yml for ubuntu-latest
- [ ] T040 [P] Add optional macOS E2E test job to .github/workflows/test.yml (separate job)

### Documentation Updates

- [ ] T041 [P] Update README.md with E2E test suite section and quickstart link
- [ ] T042 [P] Create docs/testing/e2e-tests.md comprehensive developer guide
- [ ] T043 [P] Add troubleshooting section to docs/troubleshooting/e2e-tests.md

**Checkpoint**: CI running E2E tests on every PR, developers can run locally with `make test-e2e`

---

## Phase 9: Polish & Validation

**Purpose**: Performance optimization, code quality, and final validation

### Performance & Quality

- [ ] T044 Add timeout validation to ensure full suite completes in <5 minutes (SC-001)
- [ ] T045 Run E2E suite 10 times to validate zero flakiness (SC-007)
- [ ] T046 [P] Optimize Gitea startup time with health check tuning
- [ ] T047 [P] Add comprehensive error context messages per SC-004 requirement

### Final Validation

- [ ] T048 Validate all 18 functional requirements from spec.md are covered by testscript scenarios
- [ ] T049 Run cross-platform validation on Linux and macOS (SC-009)
- [ ] T050 Verify cleanup leaves zero orphaned resources (docker ps, k3d cluster list)
- [ ] T051 Execute quickstart.md validation to ensure developer documentation is accurate

**Checkpoint**: E2E test suite production-ready with <5min execution, zero flakiness, comprehensive coverage

---

## Dependencies & Execution Order

### Phase Dependencies

```
Phase 1: Setup
   ↓
Phase 2: Foundational ← CRITICAL BLOCKER (all user stories depend on this)
   ↓
Phase 3-6: User Stories (can run in parallel once Phase 2 complete)
   ↓
Phase 7: Edge Cases (depends on user story infrastructure)
   ↓
Phase 8: CI Integration (depends on working test scenarios)
   ↓
Phase 9: Polish (depends on all functionality complete)
```

### User Story Dependencies

- **User Story 1 (P1)**: Depends ONLY on Phase 2 completion - no dependencies on other stories
- **User Story 2 (P2)**: Depends ONLY on Phase 2 completion - tests independently of US1
- **User Story 3 (P3)**: Depends ONLY on Phase 2 completion - tests independently of US1/US2
- **User Story 4 (P2)**: Depends ONLY on Phase 2 completion - tests independently of other stories

**Key Insight**: Once Phase 2 (Foundational) is complete, ALL user stories can be implemented in parallel by different developers!

### Within Each Phase

**Phase 2 (Foundational)**:
- k3d tasks (T006-T008) can run in parallel
- Gitea lifecycle tasks (T009-T011) can run in parallel
- Gitea API client tasks (T012-T015) can run in parallel
- Test helper tasks (T016-T019) can run in parallel
- Test fixture tasks (T020-T022) can run in parallel
- Integration task (T023) must wait for completion of all parallel groups

**Phase 7 (Edge Cases)**:
- All edge case testscript scenarios (T031-T038) can run in parallel - different files

---

## Parallel Execution Examples

### Phase 2: Foundational Infrastructure

```bash
# Launch all k3d tasks together:
Task T006: "Implement k3d cluster creation in tests/e2e/k3d/cluster.go"
Task T007: "Implement k3d configuration templates in tests/e2e/k3d/config.go"
Task T008: "Implement k3d cleanup utilities in tests/e2e/k3d/cleanup.go"

# Launch all Gitea API client tasks together:
Task T012: "Implement repository operations in tests/e2e/gitea/client.go"
Task T013: "Implement pull request operations in tests/e2e/gitea/client.go"
Task T014: "Implement tag operations in tests/e2e/gitea/client.go"
Task T015: "Implement user operations in tests/e2e/gitea/client.go"

# Launch all test helper tasks together:
Task T016: "Implement git operations wrapper in tests/e2e/testhelpers/git.go"
Task T017: "Implement APX command wrappers in tests/e2e/testhelpers/apx.go"
Task T018: "Implement custom assertions in tests/e2e/testhelpers/assertions.go"
Task T019: "Implement environment setup helper in tests/e2e/testhelpers/environment.go"
```

### Phase 3-6: User Stories (Team Parallelism)

```bash
# With a team of 4 developers (after Phase 2 complete):
Developer A: T024 (User Story 1 - Complete workflow)
Developer B: T025-T026 (User Story 2 - Cross-repo deps)
Developer C: T027-T029 (User Story 3 - Breaking changes)
Developer D: T030 (User Story 4 - Git history)

# All stories complete independently and can be tested in isolation
```

### Phase 7: Edge Cases

```bash
# Launch all edge case scenarios together:
Task T031: "e2e_gitea_unreachable.txt"
Task T032: "e2e_existing_pr.txt"
Task T033: "e2e_corrupted_git_history.txt"
Task T034: "e2e_circular_deps.txt"
Task T035: "e2e_duplicate_tag.txt"
Task T036: "e2e_overlay_deletion.txt"
Task T037: "e2e_concurrent_publish.txt"
Task T038: "e2e_codeowners.txt"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only - Fastest Path to Value)

1. **Phase 1**: Setup (T001-T005) - ~1 hour
2. **Phase 2**: Foundational infrastructure (T006-T023) - ~1-2 days
   - CRITICAL: This blocks everything else
   - Parallelize within phase for speed
3. **Phase 3**: User Story 1 only (T024) - ~4-6 hours
4. **STOP and VALIDATE**: Run `make test-e2e` - should have 1 passing scenario
5. **Demo/Deploy**: MVP E2E test infrastructure is now functional!

**Estimated MVP Effort**: 2-3 days for complete Phase 1-3

### Incremental Delivery (Add Stories Sequentially)

After MVP:
1. Add User Story 2 (T025-T026) → Test independently → ~4 hours
2. Add User Story 3 (T027-T029) → Test independently → ~4 hours
3. Add User Story 4 (T030) → Test independently → ~2 hours
4. Add Edge Cases (T031-T038) → Test independently → ~1 day
5. CI Integration (T039-T043) → ~4 hours
6. Polish (T044-T051) → ~1 day

**Total Incremental**: ~5-6 days from MVP to production-ready

### Parallel Team Strategy (Fastest Overall)

With 4 developers:

**Week 1**:
- **All**: Phase 1 Setup (T001-T005) - Day 1 morning
- **All**: Phase 2 Foundational (T006-T023) - Day 1 afternoon + Day 2
  - Parallelize tasks within Phase 2 as shown above
- **CHECKPOINT**: Foundation complete by end of Day 2

**Week 2** (all in parallel):
- **Dev A**: User Story 1 (T024) - Day 3
- **Dev B**: User Story 2 (T025-T026) - Day 3-4
- **Dev C**: User Story 3 (T027-T029) - Day 3-4
- **Dev D**: User Story 4 (T030) - Day 3
- **CHECKPOINT**: All user stories complete by end of Day 4

**Week 2 continued**:
- **All**: Edge Cases (T031-T038) parallelized - Day 5
- **Dev A**: CI Integration (T039-T040) - Day 5
- **Dev B/C**: Documentation (T041-T043) - Day 5
- **Dev D**: Polish & Validation (T044-T051) - Day 5

**Total Parallel Effort**: ~5 days with 4 developers

---

## Success Criteria Mapping

| Task(s) | Success Criterion | Validation Method |
|---------|-------------------|-------------------|
| T044 | SC-001: <5min execution | CI workflow duration logs, timeout enforcement |
| T024-T051 | SC-002: 95% regression detection | Map testscript scenarios to documented workflows |
| T039-T040 | SC-003: 100% CI pass rate | GitHub Actions status badge |
| T047 | SC-004: Actionable errors | Manual review of error messages in assertions |
| T024-T038 | SC-005: Workflow validation | All 18 FRs covered by testscript scenarios |
| T050 | SC-006: Zero orphaned resources | Post-test `docker ps`, `k3d cluster list` |
| T045 | SC-007: Zero flakiness | 10x sequential CI runs, 100% pass rate |
| T003 | SC-008: One-command execution | `make test-e2e` works without manual steps |
| T049 | SC-009: Cross-platform | CI matrix (Linux/macOS) both pass |
| T031-T038 | SC-010: Edge case coverage | 8 edge case scenarios validated |

---

## Task Statistics

**Total Tasks**: 51

**By Phase**:
- Phase 1 (Setup): 5 tasks
- Phase 2 (Foundational): 18 tasks (CRITICAL BLOCKER)
- Phase 3 (US1 - MVP): 1 task
- Phase 4 (US2): 2 tasks
- Phase 5 (US3): 3 tasks
- Phase 6 (US4): 1 task
- Phase 7 (Edge Cases): 8 tasks
- Phase 8 (CI/Docs): 5 tasks
- Phase 9 (Polish): 8 tasks

**By User Story**:
- User Story 1 (Complete workflow): 1 testscript scenario
- User Story 2 (Cross-repo deps): 2 testscript scenarios
- User Story 3 (Breaking changes): 3 testscript scenarios
- User Story 4 (Git history): 1 testscript scenario
- Edge Cases: 8 testscript scenarios

**Parallelizable Tasks**: 39 tasks marked [P] (76% can run in parallel within constraints)

**Independent Test Scenarios**: 15 testscript files

---

## Notes

- **[P] tasks**: Different files, no dependencies - can run in parallel
- **[Story] labels**: Map tasks to user stories for traceability and independent delivery
- **Testscript-first approach**: Each user story delivers one or more `.txt` test scenarios
- **Phase 2 is critical**: All 18 foundational tasks must complete before ANY user story work begins
- **TDD compliance**: Testscript scenarios ARE the tests - write them before helper implementation
- **Path handling**: All git/file operations must use `filepath.Join()` and `filepath.ToSlash()` for Windows compatibility
- **Cleanup guarantee**: Use `t.Cleanup()` for guaranteed resource cleanup on test failure
- **Error messages**: Include actionable "How to fix" context per SC-004
- **Platform targets**: Primary Linux/macOS, Windows best-effort via WSL2

---

## Implementation Workflow

### For Each Task

1. **Read context**: Review plan.md, data-model.md, contracts/ for task context
2. **Write test first** (if applicable): Testscript scenarios define expected behavior
3. **Implement**: Write minimal code to make test pass
4. **Validate**: Run specific test or `make test-e2e`
5. **Commit**: Clear commit message referencing task ID
6. **Update status**: Mark task complete in this file

### For Each User Story Phase

1. **Complete all tasks** in the user story phase
2. **Run independent test**: Verify story works in isolation
3. **Integration check**: Ensure no regressions in previous stories
4. **Checkpoint**: Demo/review before moving to next story

### For Pull Requests

- **Setup + Foundational**: Single PR (large but foundational)
- **Each User Story**: Separate PR for independent review
- **Edge Cases**: Can be combined or split based on complexity
- **CI/Docs/Polish**: Separate PRs for easier review

---

## Next Steps

**Immediate**:
1. ✅ Review this task breakdown with team
2. ✅ Confirm implementation strategy (MVP vs Incremental vs Parallel)
3. ⏳ Begin Phase 1: Setup (T001-T005)
4. ⏳ Begin Phase 2: Foundational infrastructure (CRITICAL PATH)

**After Foundational Complete**:
5. ⏳ Implement User Story 1 (MVP)
6. ⏳ Validate MVP with stakeholders
7. ⏳ Incrementally add remaining user stories
8. ⏳ CI integration and polish

**Success Milestone**: When `make test-e2e` runs all scenarios in <5min with zero flakiness and 100% pass rate
