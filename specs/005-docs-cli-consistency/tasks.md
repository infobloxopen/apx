# Tasks: Docs-CLI Consistency

**Input**: Design documents from `/specs/005-docs-cli-consistency/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

---

## Phase 1: Setup

**Purpose**: No new project setup needed — this feature modifies an existing codebase. This phase handles prerequisite verification.

- [ ] T001 Verify branch 005-docs-cli-consistency is checked out and all tests pass via `go test ./...`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Code fixes that MUST be complete before documentation can be written accurately. These fix the source of truth that docs will reference.

**⚠️ CRITICAL**: Docs cannot be written to match CLI behavior until these code fixes land.

- [ ] T002 Fix `apx init app` config generation to produce valid Config-compatible YAML in internal/schema/app.go
  - Change `generateApxYaml()` to output `version: 1` (int, not `v1` string)
  - Add `repo` field (required by `validateConfig`)
  - Replace `kind`/`module` fields with `module_roots` (matches Config struct)
  - Ensure `AppScaffolder` has access to `repo` value (may need constructor change)
- [ ] T003 Fix search example bug in existing doc parity test in cmd/apx/commands/doc_parity_test.go
  - Change `{"search", []string{"apx", "search", "payments", "ledger"}}` to `{"search", []string{"apx", "search", "payments"}}` (MaximumNArgs(1))
- [ ] T004 Add comprehensive command existence parity test in cmd/apx/commands/doc_parity_test.go
  - `TestDocParity_AllCommandsExist`: loop over all 14 commands + 4 subcommands + completion, verify `root.Find()` succeeds
  - Per Contract 1 in contracts/doc-parity-test-contract.md
- [ ] T005 [P] Add comprehensive flag existence parity test in cmd/apx/commands/doc_parity_test.go
  - `TestDocParity_AllFlagsExist`: loop over every command→flag mapping from data-model.md per-command flags table
  - Per Contract 2 in contracts/doc-parity-test-contract.md
- [ ] T006 [P] Add required flag enforcement parity test in cmd/apx/commands/doc_parity_test.go
  - `TestDocParity_RequiredFlags`: verify `breaking` without `--against` fails, `semver suggest` without `--against` fails
  - Per Contract 3 in contracts/doc-parity-test-contract.md
- [ ] T007 Add config roundtrip parity test in cmd/apx/commands/doc_parity_test.go
  - `TestDocParity_ConfigRoundtrip`: run `init canonical` → load generated apx.yaml with `config.Load()` → assert no error
  - Run `init app` → load generated apx.yaml → assert no error (depends on T002 fix)
  - Validate `apx.example.yaml` loads without error
  - Per Contract 5 in contracts/doc-parity-test-contract.md
- [ ] T008 Run `go test ./cmd/apx/commands/ -run TestDocParity -v` and confirm all parity tests pass

**Checkpoint**: All code fixes landed, all parity tests green. Documentation can now be written to match the fixed CLI behavior.

---

## Phase 3: User Story 1 — Quickstart End-to-End (Priority: P1) 🎯 MVP

**Goal**: A developer can follow the quickstart guide end-to-end without hitting unknown commands, wrong flags, or incorrect examples.

**Independent Test**: Run every `apx` command from `docs/getting-started/quickstart.md` against the compiled binary. Zero "unknown command/flag" errors.

### Implementation for User Story 1

- [ ] T009 [US1] Rewrite docs/getting-started/quickstart.md to match actual CLI
  - Replace `apx version suggest` → `apx semver suggest`
  - Replace `apx search payments ledger` → `apx search payments` (MaximumNArgs(1))
  - Add `--against` flag to all `apx breaking` examples
  - Add `--against` flag to all `apx semver suggest` examples
  - Fix `apx.yaml` examples to match Config struct (version: 1, org, repo, module_roots)
  - Fix `apx init` usage to show `apx init <kind> <modulePath>` (MaximumNArgs(2)) or interactive (0 args)
  - Remove any references to non-existent flags or commands
- [ ] T010 [P] [US1] Fix docs/getting-started/installation.md to match current installation methods
  - Verify brew install command matches current tap/cask setup
  - Verify binary download instructions are current
- [ ] T011 [P] [US1] Fix docs/getting-started/interactive-init.md to match current interactive flow
  - Remove any urfave/cli or survey references
  - Update to reflect charmbracelet/huh-based interactive prompts
  - Fix command examples to match actual init flags (--non-interactive, --org, --repo, --languages)

**Checkpoint**: Quickstart guide is accurate. A new developer can follow it end-to-end.

---

## Phase 4: User Story 2 — CLI Reference Accuracy (Priority: P1)

**Goal**: The CLI reference documents every command, flag, arg count, and env var that exists in the binary. Nothing is missing, nothing is fabricated.

**Independent Test**: For each command in docs/cli-reference/index.md, run `apx <command> --help` and confirm flags/args/description match.

### Implementation for User Story 2

- [ ] T012 [US2] Rewrite docs/cli-reference/index.md from scratch using data-model.md command tree
  - Document all 14 commands + subcommands with Use, Short, flags table, examples
  - Include global persistent flags section (--quiet, --verbose, --json, --no-color, --config)
  - Mark required flags (--against) clearly
  - Add canonical exit-code table (0=success, 1=general error, 6=validation failure)
  - Document environment variables: APX_CONFIG (implemented), APX_VERBOSE/APX_USE_CONTAINER/APX_CACHE_DIR (mark as Planned)
  - Remove the toctree referencing non-existent sub-pages (core-commands, dependency-commands, etc.) — all content goes in index.md
- [ ] T013 [P] [US2] Update README.md to match actual CLI commands and flags
  - Fix all command name references (version suggest → semver suggest)
  - Fix all flag references (add --against where required)
  - Fix all examples to use correct argument counts
  - Remove stale urfave/cli references if any remain

**Checkpoint**: CLI reference is a complete, accurate mirror of `apx --help` and per-command `--help` output.

---

## Phase 5: User Story 3 — Config File Format Consistency (Priority: P1)

**Goal**: `apx init` generates config that `apx config validate` accepts. Documented config schema matches Config struct.

**Independent Test**: Run `apx init canonical --org=test --repo=apis --skip-git --non-interactive` then `apx config validate` → exit 0. Same for `apx init app`.

### Implementation for User Story 3

- [ ] T014 [US3] Update all apx.yaml examples in docs/ to match Config struct schema
  - Every `apx.yaml` code block must use `version: 1` (int), include `org`, `repo`
  - Remove `kind`/`module` fields from examples (not in Config struct)
  - Add `module_roots`, `language_targets`, `policy`, `publishing`, `tools`, `execution` as appropriate
  - Files to check: docs/getting-started/quickstart.md, docs/cli-reference/index.md, docs/app-repos/index.md, docs/canonical-repo/index.md
- [ ] T015 [P] [US3] Verify apx.example.yaml passes config validate — no changes expected but confirm with test in T007

**Checkpoint**: Every config example in docs, when pasted into apx.yaml, passes `apx config validate`.

---

## Phase 6: User Story 4 — Publishing and CI Workflow (Priority: P2)

**Goal**: CI template commands in publishing docs work with the current release.

**Independent Test**: Extract every `apx` command from CI template code blocks in docs/publishing/index.md, verify each exists with correct flags.

### Implementation for User Story 4

- [ ] T016 [US4] Rewrite docs/publishing/index.md to use only valid commands and flags
  - Replace `apx fetch --ci` → `apx fetch --verify` (or `--output`)
  - Replace `apx tag subdir` → mark as Planned or remove
  - Replace `apx packages publish` → mark as Planned or remove
  - Replace `apx version verify` → mark as Planned or remove
  - Fix all CI template YAML blocks to use valid commands
  - Use `{admonition} Planned — not yet available` blocks for roadmap commands
- [ ] T017 [P] [US4] Fix docs/app-repos/index.md workflow examples
  - Fix command names and flags to match actual CLI
  - Fix CI integration examples
  - Mark planned commands with admonition blocks
- [ ] T018 [P] [US4] Fix docs/canonical-repo/index.md and docs/canonical-repo/structure.md
  - Fix catalog format examples to match catalog/catalog.yaml schema from data-model.md
  - Fix any command references

**Checkpoint**: A developer can paste CI templates from docs into their pipeline without hitting unknown commands.

---

## Phase 7: User Story 5 — Dependency and Lockfile Workflows (Priority: P2)

**Goal**: Dependency docs show the correct lockfile format and mark unimplemented commands as Planned.

**Independent Test**: Compare documented lockfile format in docs/dependencies/index.md with actual LockFile struct. All fields match.

### Implementation for User Story 5

- [ ] T019 [US5] Rewrite docs/dependencies/index.md to match actual lockfile schema and commands
  - Fix lockfile format: show map-based Dependencies (not flat list), with Repo/Ref/Modules fields
  - Mark `apx update`, `apx upgrade`, `apx list`, `apx show` as Planned using `{admonition}` blocks
  - Fix `apx search` examples to use MaximumNArgs(1) — single query arg
  - Fix `apx add` examples to show ExactArgs(1) — `apx add <module-path[@version]>`
  - Document actual dependency commands: search, add, sync, unlink, gen

**Checkpoint**: Lockfile format in docs matches LockFile struct exactly. Planned commands clearly marked.

---

## Phase 8: User Story 6 — Broken Internal Links Resolved (Priority: P3)

**Goal**: Sphinx build produces zero broken-link warnings.

**Independent Test**: Run `sphinx-build -W -b html docs docs/_build` → exit 0.

### Implementation for User Story 6

The following toctree entries reference non-existent files (29 total). Create stub pages for each.

- [ ] T020 [P] [US6] Create stub pages for docs/canonical-repo/ broken toctree refs
  - docs/canonical-repo/setup.md — stub: "Setting up a canonical repository (coming soon)"
  - docs/canonical-repo/ci-templates.md — stub: "CI templates for canonical repos (coming soon)"
  - docs/canonical-repo/protection.md — stub: "Branch and tag protection (coming soon)"
- [ ] T021 [P] [US6] Create stub pages for docs/app-repos/ broken toctree refs
  - docs/app-repos/layout.md — stub
  - docs/app-repos/local-development.md — stub
  - docs/app-repos/publishing-workflow.md — stub
  - docs/app-repos/ci-integration.md — stub
- [ ] T022 [P] [US6] Create stub pages for docs/dependencies/ broken toctree refs
  - docs/dependencies/discovery.md — stub
  - docs/dependencies/adding-dependencies.md — stub
  - docs/dependencies/code-generation.md — stub
  - docs/dependencies/updates-and-upgrades.md — stub
  - docs/dependencies/versioning-strategy.md — stub
- [ ] T023 [P] [US6] Create stub pages for docs/publishing/ broken toctree refs
  - docs/publishing/overview.md — stub
  - docs/publishing/validation.md — stub
  - docs/publishing/tagging-strategy.md — stub
  - docs/publishing/publish-command.md — stub
  - docs/publishing/canonical-pr.md — stub
  - docs/publishing/release-guardrails.md — stub
- [ ] T024 [P] [US6] Create stub pages for docs/troubleshooting/ broken toctree refs
  - docs/troubleshooting/common-errors.md — stub
  - docs/troubleshooting/buf-issues.md — stub
  - docs/troubleshooting/versioning-problems.md — stub
  - docs/troubleshooting/publishing-failures.md — stub
  - docs/troubleshooting/code-generation.md — stub
- [ ] T025 [P] [US6] Create stub pages for docs/cli-reference/ broken toctree refs
  - docs/cli-reference/core-commands.md — stub
  - docs/cli-reference/dependency-commands.md — stub
  - docs/cli-reference/publishing-commands.md — stub
  - docs/cli-reference/validation-commands.md — stub
  - docs/cli-reference/utility-commands.md — stub
  - docs/cli-reference/global-options.md — stub

**Checkpoint**: `sphinx-build -W -b html docs docs/_build` exits 0 with no toctree warnings.

---

## Phase 9: Polish & Cross-Cutting Concerns

**Purpose**: Clean up stale references, final validation across all stories.

- [ ] T026 [P] Remove all urfave/cli and survey references from user-facing docs
  - Search: `grep -rn "urfave\|survey\|cli/v2" docs/ README.md INTERACTIVE_INIT.md`
  - Remove or replace every match
- [ ] T027 [P] Fix INTERACTIVE_INIT.md — remove stale framework references, update to reflect huh-based prompts
- [ ] T028 [P] Fix docs/troubleshooting/faq.md — update all command references to match actual CLI
- [ ] T029 Run full test suite: `go test ./...` — confirm all tests pass including parity tests
- [ ] T030 Run Sphinx build: `sphinx-build -W -b html docs docs/_build` — confirm zero warnings
- [ ] T031 Manual walkthrough: follow quickstart.md end-to-end against compiled binary, confirm SC-001

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately
- **Foundational (Phase 2)**: Depends on Phase 1 — BLOCKS all user stories
  - T002 (code fix) must land before T007 (config roundtrip test) and T014 (doc config examples)
  - T003-T006 (test additions) are independent of T002 except T007
- **User Stories (Phase 3-8)**: All depend on Phase 2 completion
  - US1 (quickstart), US2 (CLI ref), US3 (config) are all P1 — do in order or parallel
  - US4 (publishing), US5 (dependencies) are P2 — after P1 stories
  - US6 (broken links) is P3 — can actually run in parallel with anything (different files)
- **Polish (Phase 9)**: After all user stories complete

### User Story Dependencies

- **US1 (Quickstart)**: Depends on T002 (init fix). Independent of other stories.
- **US2 (CLI Reference)**: Independent. Can run in parallel with US1.
- **US3 (Config Format)**: Depends on T002 (init fix). Can run in parallel with US1/US2.
- **US4 (Publishing/CI)**: Independent of US1-US3. Can start after Phase 2.
- **US5 (Dependencies/Lockfile)**: Independent. Can start after Phase 2.
- **US6 (Broken Links)**: Fully independent — different files. Can start any time.

### Within Each User Story

- Fix command names/flags before writing new examples
- Update code blocks before prose
- Validate each page against CLI --help before moving to next

### Parallel Opportunities

- T004 + T005 + T006 can all run in parallel (different test functions, same file but independent additions)
- T010 + T011 can run in parallel (different doc files)
- T013 can run in parallel with T012 (README.md vs cli-reference/index.md)
- T017 + T018 can run in parallel (different doc files)
- T020 + T021 + T022 + T023 + T024 + T025 can ALL run in parallel (all different directories)
- T026 + T027 + T028 can run in parallel (different files)

---

## Parallel Example: User Story 6 (Broken Links)

```bash
# All stub page creation tasks can run simultaneously:
Task T020: Create 3 stub pages in docs/canonical-repo/
Task T021: Create 4 stub pages in docs/app-repos/
Task T022: Create 5 stub pages in docs/dependencies/
Task T023: Create 6 stub pages in docs/publishing/
Task T024: Create 5 stub pages in docs/troubleshooting/
Task T025: Create 6 stub pages in docs/cli-reference/
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (T001)
2. Complete Phase 2: Foundational code fixes + parity tests (T002–T008)
3. Complete Phase 3: User Story 1 — Quickstart (T009–T011)
4. **STOP and VALIDATE**: Walk through quickstart end-to-end against binary
5. Every subsequent story adds incremental trust

### Incremental Delivery

1. Setup + Foundational → Code is correct, tests enforce parity
2. Add US1 (Quickstart) → New users can trust the getting-started path
3. Add US2 (CLI Reference) → Power users can trust the reference
4. Add US3 (Config Format) → Config examples are accurate
5. Add US4 (Publishing/CI) → CI templates work
6. Add US5 (Dependencies) → Lockfile format is documented correctly
7. Add US6 (Broken Links) → Clean Sphinx build, no 404s
8. Polish → Zero stale references, full validation pass

### Suggested MVP Scope

Complete through User Story 1 (Phase 3, T011). This ensures the most critical user-facing path (first-time quickstart) is trustworthy. Total: 11 tasks.

---

## Notes

- [P] tasks = different files, no dependencies on in-progress tasks
- [US#] label maps task to specific user story for traceability
- Each user story is independently completable and testable
- Commit after each phase or logical task group
- 29 stub pages (T020-T025) are intentionally minimal — just a title and "coming soon" note
- Total: 31 tasks across 9 phases
- Tests are included because the spec explicitly calls for parity tests (FR-001 through FR-018, SC-001 through SC-010)
