# Tasks: Package Installer Support

**Input**: Design documents from `/specs/004-package-installers/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Tests**: No tests explicitly requested. GoReleaser config validation (`goreleaser check`) and snapshot builds serve as verification.

**Organization**: Tasks grouped by user story. US1 (Homebrew) and US4 (Release Pipeline) are co-dependent P1 — the release workflow is needed for Homebrew to work. US2 (Scoop) and US3 (Shell Installer) can proceed independently after.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3, US4)
- Exact file paths included

## Current State

| Item | Status |
|------|--------|
| `install.sh` (repo root) | ✅ Already written — full script with OS/arch detection, checksum verification, PATH helper |
| `infobloxopen/homebrew-tap` repo | ✅ Created (empty) |
| `infobloxopen/scoop-bucket` repo | ✅ Created (empty) |
| `infobloxopen-release-bot` GitHub App | ✅ Created (App ID: 3033530) |
| `RELEASE_APP_ID` org secret | ✅ Configured |
| `RELEASE_APP_PRIVATE_KEY` org secret | ✅ Configured |
| `.goreleaser.yml` `brews` section | ✅ Exists — needs `token` field |
| `.goreleaser.yml` `scoops` section | ❌ Missing |
| `.github/workflows/release.yml` | ❌ Missing |
| README.md Scoop/shell installer sections | ❌ Missing |
| `docs/getting-started/installation.md` Scoop/shell sections | ❌ Missing |
| `Makefile` `GORELEASER_VERSION` | ⚠️ Points to v1.21.2 — needs v2.x update |

---

## Phase 1: Setup

**Purpose**: Update tooling versions and validate existing config

- [x] T001 Update GORELEASER_VERSION from v1.21.2 to v2.6.1 in Makefile
- [x] T002 Validate existing `.goreleaser.yml` passes `goreleaser check`

---

## Phase 2: Foundational (Release Workflow)

**Purpose**: The release workflow is required before any package manager can be tested. This is the blocking prerequisite.

**⚠️ CRITICAL**: Without the release workflow, Homebrew tap and Scoop bucket cannot receive updates.

- [x] T003 Create `.github/workflows/release.yml` with GitHub App token minting via `actions/create-github-app-token@v1`, GoReleaser v2, Docker Buildx, GHCR login — triggered on `v*` tags
- [x] T004 Update `.goreleaser.yml` — add `token: "{{ .Env.HOMEBREW_TAP_TOKEN }}"` to `brews[0].repository`
- [x] T005 [P] Update `.goreleaser.yml` — add `scoops` section targeting `infobloxopen/scoop-bucket` with `token: "{{ .Env.SCOOP_BUCKET_TOKEN }}"` after `brews` section
- [x] T006 Validate updated `.goreleaser.yml` with `goreleaser check` and `goreleaser release --snapshot --clean`

**Checkpoint**: Release pipeline is configured. Push a test tag to validate end-to-end (T006 snapshot is local-only; full validation requires a real tag push after merge).

---

## Phase 3: User Story 1 — Install APX via Homebrew (Priority: P1) 🎯 MVP

**Goal**: Users run `brew install infobloxopen/tap/apx` and get a working binary with shell completions.

**Independent Test**: After the first tagged release, run `brew install infobloxopen/tap/apx && apx --version` on macOS.

### Implementation for User Story 1

- [x] T007 [US1] Verify `brews` section in `.goreleaser.yml` has correct `install` block with shell completions (bash, zsh, fish) and `test` block — already present, confirm no changes needed
- [x] T008 [US1] Add initial README to `infobloxopen/homebrew-tap` repo (brief description, usage instructions, auto-generated notice) at `/Users/dgarcia/go/src/github.com/infobloxopen/homebrew-tap/README.md`

**Checkpoint**: Homebrew tap is ready. GoReleaser will populate `Formula/apx.rb` on first release.

---

## Phase 4: User Story 2 — Install APX via Scoop on Windows (Priority: P2)

**Goal**: Windows users run `scoop bucket add infobloxopen ... && scoop install infobloxopen/apx` and get a working binary.

**Independent Test**: On a Windows machine with Scoop: `scoop bucket add infobloxopen https://github.com/infobloxopen/scoop-bucket && scoop install infobloxopen/apx && apx --version`.

### Implementation for User Story 2

- [x] T009 [US2] Verify `scoops` section added in T005 matches GoReleaser v2 schema — confirm `name`, `repository.owner`, `repository.name`, `repository.token`, `homepage`, `description`, `license` fields
- [x] T010 [US2] Add initial README to `infobloxopen/scoop-bucket` repo (brief description, usage instructions, auto-generated notice) at `/Users/dgarcia/go/src/github.com/infobloxopen/scoop-bucket/README.md`

**Checkpoint**: Scoop bucket is ready. GoReleaser will populate `apx.json` on first release.

---

## Phase 5: User Story 3 — Install APX via shell one-liner (Priority: P3)

**Goal**: Users run `curl -fsSL .../install.sh | bash` and get a working binary with checksum verification.

**Independent Test**: `curl -fsSL https://raw.githubusercontent.com/infobloxopen/apx/main/install.sh | bash && ~/.local/bin/apx --version` on a clean macOS/Linux shell.

### Implementation for User Story 3

- [x] T011 [US3] Review existing `install.sh` (repo root) against contracts/install-script.md — verify OS/arch detection, version resolution, SHA256 verification, `main()` wrapper, non-interactive behavior, INSTALL_DIR/VERSION/GITHUB_TOKEN env vars
- [x] T012 [US3] Run shellcheck on `install.sh` and fix any warnings

**Checkpoint**: Install script is ready. Works immediately once a GitHub Release exists with binaries and checksums.

---

## Phase 6: User Story 4 — Automated release publishing (Priority: P1)

**Goal**: Pushing a `v*` tag triggers the full release pipeline — GitHub Release, Homebrew tap, Scoop bucket, Docker images — with zero manual steps.

**Independent Test**: Push a pre-release tag (e.g., `v0.0.1-rc.1`), verify GitHub Release is created, `homebrew-tap` gets a formula commit, `scoop-bucket` gets a manifest commit.

### Implementation for User Story 4

- [ ] T013 [US4] End-to-end validation: push a pre-release tag to the `004-package-installers` branch, verify release workflow triggers and completes (builds, GitHub Release, tap update, bucket update)
- [ ] T014 [US4] Verify `infobloxopen/homebrew-tap` received `Formula/apx.rb` after release
- [ ] T015 [US4] Verify `infobloxopen/scoop-bucket` received `apx.json` after release
- [ ] T016 [US4] Verify install script works against the new release: `curl -fsSL https://raw.githubusercontent.com/infobloxopen/apx/main/install.sh | bash`

**Checkpoint**: Full release pipeline validated end-to-end.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Documentation updates and cleanup

- [x] T017 [P] Update README.md Installation section — add Scoop, Shell Installer subsections between Homebrew and Download Binary
- [x] T018 [P] Update `docs/getting-started/installation.md` — add Scoop section, Shell Installer section, update overview paragraph, fix Homebrew example to use `infobloxopen/tap/apx`
- [x] T019 [P] Update `specs/004-package-installers/quickstart.md` — replace PAT references with GitHub App token approach (RELEASE_APP_ID + RELEASE_APP_PRIVATE_KEY)
- [x] T020 [P] Update `specs/004-package-installers/research.md` — update R-006 token strategy to reflect GitHub App decision instead of PATs
- [x] T021 [P] Update `specs/004-package-installers/data-model.md` — replace PAT references with GitHub App secrets
- [x] T022 [P] Update `specs/004-package-installers/contracts/release-workflow.md` — replace PAT env vars with `actions/create-github-app-token@v1` step
- [x] T023 [P] Update `specs/004-package-installers/contracts/install-script.md` — update file path from `scripts/install.sh` to `install.sh` (repo root)
- [x] T024 [P] Update `specs/004-package-installers/contracts/documentation.md` — update curl URLs from `scripts/install.sh` to `install.sh`
- [ ] T025 Run `specs/004-package-installers/quickstart.md` validation — walk through release process end-to-end

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately
- **Foundational (Phase 2)**: Depends on Setup — BLOCKS all user stories
- **US1 Homebrew (Phase 3)**: Depends on Foundational (T004 specifically)
- **US2 Scoop (Phase 4)**: Depends on Foundational (T005 specifically)
- **US3 Shell Installer (Phase 5)**: Can start after Setup (install.sh already exists, no release workflow dependency for review)
- **US4 Release Pipeline (Phase 6)**: Depends on ALL of Phase 2, 3, 4 — end-to-end validation
- **Polish (Phase 7)**: Can start in parallel with US phases for documentation tasks

### User Story Dependencies

- **US1 (Homebrew)**: Needs release workflow (Phase 2) + `brews.token` (T004)
- **US2 (Scoop)**: Needs release workflow (Phase 2) + `scoops` section (T005)
- **US3 (Shell Installer)**: Independent — install.sh already exists, just needs review
- **US4 (Release Pipeline)**: Integrates all — must run after US1 + US2 + US3

### Parallel Opportunities

```text
After Phase 2 (Foundational) completes:

  ┌─ US1 (T007–T008) Homebrew tap setup
  ├─ US2 (T009–T010) Scoop bucket setup
  ├─ US3 (T011–T012) Install script review
  └─ Phase 7 (T017–T024) Documentation updates [P]

All four can run in parallel since they touch different files.
```

---

## Implementation Strategy

### MVP First (Phase 1 + 2 + 3 + 6)

1. Complete Phase 1: Setup (Makefile update)
2. Complete Phase 2: Foundational (release.yml + goreleaser changes)
3. Complete Phase 3: US1 Homebrew (tap README)
4. Push a test tag → validate release pipeline
5. **STOP and VALIDATE**: `brew install infobloxopen/tap/apx` works

### Incremental Delivery

1. Setup + Foundational → Release pipeline ready
2. US1 Homebrew → `brew install` works (MVP!)
3. US2 Scoop → `scoop install` works
4. US3 Shell Installer → `curl | bash` works
5. US4 Release Pipeline → Full end-to-end verified
6. Polish → Docs updated, specs reflect actual decisions

### Task Count Summary

| Phase | Tasks | Parallelizable |
|-------|-------|---------------|
| Phase 1: Setup | 2 | 0 |
| Phase 2: Foundational | 4 | 1 |
| Phase 3: US1 Homebrew | 2 | 0 |
| Phase 4: US2 Scoop | 2 | 0 |
| Phase 5: US3 Shell Installer | 2 | 0 |
| Phase 6: US4 Release Pipeline | 4 | 0 |
| Phase 7: Polish | 9 | 8 |
| **Total** | **25** | **9** |
