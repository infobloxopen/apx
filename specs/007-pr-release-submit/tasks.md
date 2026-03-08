# Tasks: PR-First Canonical Release Submission

**Input**: Design documents from `specs/007-pr-release-submit/`  
**Prerequisites**: plan.md, spec.md  
**Tests**: Included — constitution requires test-first development (Principle III)

**Organization**: Tasks are grouped by user story to enable independent implementation and testing. US4 (Direct Submit Without PR) is excluded per plan.md — the subtree path is removed entirely.

## Format: `[ID] [P?] [Story?] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: Which user story this task belongs to (US1, US2, US3, US5)
- Setup and Foundational phases have no story label
- All file paths are relative to the repository root

---

## Phase 1: Setup

**Purpose**: Extend the manifest schema and add publisher primitives before any command wiring

- [X] T001 Add PR metadata fields (PRNumber, PRURL, PRBranch) to ReleaseManifest in internal/publisher/manifest.go
- [X] T002 Add unit tests for PR metadata fields round-trip in internal/publisher/manifest_test.go
- [X] T003 [P] Add FindExistingPR function to internal/publisher/pr.go that queries `gh pr list --head <branch> --repo <nwo> --json number,url,state`
- [X] T004 [P] Add unit tests for FindExistingPR with stubbed GHRun in internal/publisher/pr_test.go

**Checkpoint**: Manifest can store PR metadata; FindExistingPR can detect existing PRs. All existing tests still pass.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Build the core `SubmitReleaseWithPR` publisher function that all user stories depend on

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T005 Implement SubmitReleaseWithPR function in internal/publisher/pr.go — accepts ReleaseManifest + source dir, clones canonical, creates release branch `apx/release/<api-id-normalized>/<version>`, copies snapshot, generates go.mod if needed, commits, pushes, creates PR, returns PRResponse
- [X] T006 Add unit tests for SubmitReleaseWithPR with stubbed GHRun in internal/publisher/pr_test.go — verify branch naming, commit message, PR title/body, go.mod generation
- [X] T007 Add unit test for SubmitReleaseWithPR retry path — verify force-push on existing branch and FindExistingPR detection in internal/publisher/pr_test.go

**Checkpoint**: `SubmitReleaseWithPR` is fully tested in isolation. Foundation ready — user story implementation can begin.

---

## Phase 3: User Story 1 — Submit a Prepared Release as a PR (Priority: P1) 🎯 MVP

**Goal**: A developer runs `apx release submit` and gets a reviewable PR on the canonical repository with metadata recorded in the manifest

**Independent Test**: Run `apx release submit` against a prepared `.apx-release.yaml` and verify a PR is opened, state transitions to `canonical-pr-open`, and PR metadata is saved

### Tests for User Story 1

- [X] T008 [P] [US1] Add testscript for successful release submit in testdata/script/release_submit.txt — prepare manifest, exec apx release submit, verify stdout contains PR URL, verify .apx-release.yaml has state canonical-pr-open and pr_url field
- [X] T009 [P] [US1] Add testscript for submit with missing manifest in testdata/script/release_submit.txt — exec apx release submit without .apx-release.yaml, verify stderr contains "prepare" hint
- [X] T010 [P] [US1] Add testscript for submit with failed manifest state in testdata/script/release_submit.txt — create manifest in failed state, verify stderr contains failure reason

### Implementation for User Story 1

- [X] T011 [US1] Rewrite releaseSubmitAction in cmd/apx/commands/release.go — remove subtree publish path, remove --create-pr flag, wire SubmitReleaseWithPR as the only submit mechanism
- [X] T012 [US1] Update newReleaseSubmitCmd in cmd/apx/commands/release.go — remove --create-pr flag definition, keep --dry-run flag
- [X] T013 [US1] Add manifest state guards in releaseSubmitAction in cmd/apx/commands/release.go — prepared → proceed, package-published → report success, failed → error with reason, other → error
- [X] T014 [US1] Add gh CLI preflight check in releaseSubmitAction in cmd/apx/commands/release.go — call CheckGHCLI before SubmitReleaseWithPR, exit with clear error and `gh auth login` hint on failure
- [X] T015 [US1] Record PR metadata back into manifest after SubmitReleaseWithPR succeeds in cmd/apx/commands/release.go — set PRNumber, PRURL, PRBranch from PRResponse, transition to StateCanonicalPROpen, write manifest
- [X] T016 [US1] Add error handling for push and PR creation failures in cmd/apx/commands/release.go — set manifest to StateFailed with ErrCodePushFailed or ErrCodePRCreationFailed, write manifest before returning error

**Checkpoint**: `apx release submit` creates a PR, records metadata, and transitions state. MVP is complete and independently testable.

---

## Phase 4: User Story 2 — Dry-Run Preview (Priority: P1)

**Goal**: A developer runs `apx release submit --dry-run` and sees what would be submitted without any side effects

**Independent Test**: Run `apx release submit --dry-run` and verify output shows branch name, file list, and go.mod preview — no PR created, no state change

### Tests for User Story 2

- [X] T017 [P] [US2] Add testscript for dry-run submit in testdata/script/release_submit.txt — exec apx release submit --dry-run, verify stdout contains branch name and file list, verify .apx-release.yaml state unchanged

### Implementation for User Story 2

- [X] T018 [US2] Implement dry-run path in releaseSubmitAction in cmd/apx/commands/release.go — compute branch name, list snapshot files, show go.mod preview if applicable, display manifest report, exit without cloning/pushing/creating PR
- [X] T019 [US2] Add ComputeReleaseBranchName helper to internal/publisher/pr.go — takes api ID + version, returns deterministic `apx/release/<normalized-id>/<version>` branch name (extracted from SubmitReleaseWithPR for dry-run reuse)

**Checkpoint**: `--dry-run` shows full preview without side effects

---

## Phase 5: User Story 3 — Retry Failed Submission (Priority: P1)

**Goal**: A developer re-runs `apx release submit` after a network failure and it recovers gracefully — no duplicate branches or PRs

**Independent Test**: Simulate a partial failure (branch pushed but PR not created), re-run submit, verify it detects existing branch and creates PR without duplication

### Tests for User Story 3

- [X] T020 [P] [US3] Add unit test for retry when manifest is already in canonical-pr-open state in internal/publisher/pr_test.go — verify early return with existing PR details
- [X] T021 [P] [US3] Add testscript for retry after interrupted submit in testdata/script/release_submit.txt — create manifest in canonical-pr-open state with pr_url, exec apx release submit, verify stdout reports existing PR

### Implementation for User Story 3

- [X] T022 [US3] Add canonical-pr-open state guard in releaseSubmitAction in cmd/apx/commands/release.go — if manifest is already canonical-pr-open with PRUrl set, report existing PR and exit successfully
- [X] T023 [US3] Add retry logic to SubmitReleaseWithPR in internal/publisher/pr.go — on push, use force-push to handle existing branch; after push, call FindExistingPR before CreatePR; if PR exists, return existing PRResponse
- [X] T024 [US3] Handle prepared-state retry when PR already exists in cmd/apx/commands/release.go — if SubmitReleaseWithPR returns a PR that already existed, still update manifest with PR metadata and transition to canonical-pr-open

**Checkpoint**: Re-running submit after any failure is safe and idempotent

---

## Phase 6: User Story 5 — CI Pipeline Submit with Provenance (Priority: P2)

**Goal**: When `apx release submit` runs in CI, the PR body includes CI provenance (run URL, system name) for audit trail

**Independent Test**: Set `GITHUB_ACTIONS=true` and related env vars, run submit, verify PR body includes CI run URL

### Tests for User Story 5

- [X] T025 [P] [US5] Add unit test for CI provenance in PR body in internal/publisher/pr_test.go — stub environment variables for GitHub Actions, verify PR body contains run URL

### Implementation for User Story 5

- [X] T026 [US5] Integrate DetectCI into SubmitReleaseWithPR PR body generation in internal/publisher/pr.go — call DetectCI(), if CI detected, append CI provider name and run URL to the PR body
- [X] T027 [US5] Record CI metadata in manifest after submit in cmd/apx/commands/release.go — if CI detected, store provider and run URL in manifest (reuse existing DetectCI from record.go)

**Checkpoint**: CI pipelines get provenance-rich PRs; non-CI usage is unaffected

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Documentation, cleanup, and validation across all stories

- [X] T028 [P] Update submit section in docs/cli-reference/release-commands.md — remove --create-pr flag docs, document PR-only submit flow, update examples
- [X] T029 [P] Update docs/publishing/overview.md — reflect that release submit always creates a PR, remove subtree references from release path
- [X] T030 Remove SubtreePublisher usage from releaseSubmitAction in cmd/apx/commands/release.go — delete dead subtree code paths and imports (subtree.go itself is kept for apx publish)
- [X] T031 Run full test suite to verify no regressions — `make test` (unit + integration), verify testscript scenarios pass
- [X] T032 Verify release inspect displays PR metadata — run `apx release inspect` after submit and confirm pr_url, pr_number, pr_branch appear in output

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: No dependencies — can start immediately
- **Phase 2 (Foundational)**: Depends on Phase 1 (manifest fields + FindExistingPR)
- **Phase 3 (US1)**: Depends on Phase 2 (SubmitReleaseWithPR function) — **BLOCKS other stories**
- **Phase 4 (US2)**: Depends on Phase 2 (ComputeReleaseBranchName) — can start in parallel with US1
- **Phase 5 (US3)**: Depends on Phase 3 (releaseSubmitAction rewrite must exist for retry guards)
- **Phase 6 (US5)**: Depends on Phase 3 (releaseSubmitAction + SubmitReleaseWithPR operational)
- **Phase 7 (Polish)**: Depends on all user stories being complete

### User Story Dependencies

- **US1 (P1)**: Depends on Foundational — no story dependencies. This is the MVP.
- **US2 (P1)**: Depends on Foundational — can start in parallel with US1 (dry-run path is independent of the actual submit)
- **US3 (P1)**: Depends on US1 — retry logic augments the submit flow written in US1
- **US5 (P2)**: Depends on US1 — CI provenance adds to the PR body built in US1

### Within Each User Story

- Tests written first (testscript scenarios + unit tests)
- Tests must fail before implementation
- Implementation in dependency order (publisher functions → command wiring → error handling)
- Story complete before moving to next priority (except US2 which can parallel US1)

### Parallel Opportunities

- T001/T002 can parallel with T003/T004 (manifest fields vs FindExistingPR — different files)
- T008/T009/T010 can all run in parallel (testscript scenarios in same file but independent)
- T017 can parallel with T020/T021 (US2 tests vs US3 tests — different stories)
- T025 can start as soon as Phase 2 is done (independent test)
- T028/T029 can parallel with each other (different doc files)

---

## Parallel Example: Phase 1

```bash
# Launch in parallel (different files):
Task T001: "Add PR metadata fields to ReleaseManifest in internal/publisher/manifest.go"
Task T003: "Add FindExistingPR function in internal/publisher/pr.go"

# Then in parallel (test files for above):
Task T002: "Unit tests for PR metadata fields in internal/publisher/manifest_test.go"
Task T004: "Unit tests for FindExistingPR in internal/publisher/pr_test.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (T001–T004)
2. Complete Phase 2: Foundational (T005–T007)
3. Complete Phase 3: User Story 1 (T008–T016)
4. **STOP and VALIDATE**: Run `make test`, verify `apx release submit` produces a PR
5. Deploy/demo if ready — this alone replaces the stubbed `--create-pr`

### Incremental Delivery

1. Setup + Foundational → Publisher primitives ready
2. US1 → PR submission works → **MVP!**
3. US2 → Dry-run preview works → Developer confidence
4. US3 → Retry is safe → Production readiness
5. US5 → CI provenance → Enterprise audit trail
6. Polish → Docs + cleanup → Release quality

### Sequential Execution (Single Developer)

1. T001 → T002 → T003 → T004 (Phase 1: ~1 hour)
2. T005 → T006 → T007 (Phase 2: ~2 hours)
3. T008–T010 → T011–T016 (Phase 3/US1: ~3 hours)
4. T017 → T018–T019 (Phase 4/US2: ~1 hour)
5. T020–T021 → T022–T024 (Phase 5/US3: ~2 hours)
6. T025 → T026–T027 (Phase 6/US5: ~1 hour)
7. T028–T032 (Phase 7: ~1 hour)

---

## Notes

- US4 (Direct Submit Without PR) is **excluded** — per plan.md, subtree path is removed entirely from `release submit`
- `--create-pr` flag is removed (PR is the only submit path); `--dry-run` is retained
- `internal/publisher/subtree.go` is NOT deleted — it's still used by `apx publish`
- `StateCanonicalPROpen` (ordinal 5) already exists in state.go — no state machine changes needed
- `ErrCodePRCreationFailed` already exists in errors.go — no error code additions needed
- `GHRun` function variable in pr.go enables all gh CLI calls to be stubbed in tests
- `DetectCI()` already exists in record.go — reuse for CI provenance, no new detection code
