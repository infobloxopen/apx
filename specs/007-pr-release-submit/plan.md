# Implementation Plan: PR-First Canonical Release Submission

**Branch**: `007-pr-release-submit` | **Date**: 2026-03-08 | **Spec**: [spec.md](spec.md)  
**Input**: Feature specification from `specs/007-pr-release-submit/spec.md`

## Summary

Replace the stubbed `--create-pr` flag on `apx release submit` with a working PR-based canonical submission flow. The implementation replaces the subtree push mechanism entirely: `release submit` will clone the canonical repository, create a deterministic release branch, export the release snapshot, push the branch, and open a pull request via the `gh` CLI. PR metadata (number, URL, branch) is recorded back into the release manifest and the state transitions to `canonical-pr-open`. The existing `PublishModuleWithPR` in `internal/publisher/pr.go` provides the proven pattern; this feature adapts that pattern into a release-specific function with retry safety, dry-run support, and manifest-driven provenance.

**Key design change from spec**: Per user direction, the subtree-based direct push path is **removed**, not preserved. `release submit` always creates a PR. The `--create-pr` flag is removed (PR is the only path). A `--dry-run` flag remains for preview.

## Technical Context

**Language/Version**: Go 1.26.1  
**Primary Dependencies**: cobra (CLI), gh CLI (PR creation), git (branch/push), yaml.v3 (manifest I/O)  
**Storage**: File-based (`.apx-release.yaml` manifest, `.apx-release-record.yaml` record)  
**Testing**: testify (unit), testscript (CLI integration), Gitea+k3d (e2e); `GHRun` function variable for gh CLI stubbing  
**Target Platform**: Linux, macOS, Windows (cross-platform paths via `filepath`)  
**Project Type**: CLI tool  
**Performance Goals**: Submit completes within the time of a shallow git clone + single push + single `gh pr create` call (typically <30s)  
**Constraints**: Requires `gh` CLI installed and authenticated; canonical repo must be on GitHub  
**Scale/Scope**: Single command change (`release submit`), manifest schema extension, ~300 lines of new publisher code, ~200 lines of command rewiring

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Documentation-Driven Development | **PASS** | Spec written first. Docs updates (release-commands.md, publishing-commands.md) are part of the task list. |
| II. Cross-Platform Path Operations | **PASS** | All path construction uses `filepath.Join`/`filepath.ToSlash`. Branch names use forward slashes (git convention). `copyDir` helper already handles cross-platform. |
| III. Test-First Development | **PASS** | Unit tests for new publisher functions (`SubmitReleaseWithPR`, `FindExistingPR`, manifest PR fields). Testscript for CLI flow. E2e via Gitea. |
| III. Code Quality & Maintainability | **PASS** | Business logic stays in `internal/publisher/`. Command logic in `cmd/apx/commands/release.go`. No god objects — new code extends existing focused packages. |
| IV. Developer Experience First | **PASS** | Clear PR URL output after submit. Dry-run shows full diff preview. Helpful errors for missing `gh`, auth failures, push failures. |
| V. Canonical Import Paths | **N/A** | This feature does not generate Go code or import paths. |
| VI. Git Subtree Publishing Strategy | **JUSTIFIED VIOLATION** | The user explicitly requested removing the subtree flow from `release submit`. The PR-based snapshot approach replaces it. `apx publish` retains subtree as a separate code path. See Complexity Tracking. |
| VII. Multi-Format Schema Support | **PASS** | No format-specific changes. Snapshot copies whatever schema files exist in the source path. |

## Project Structure

### Documentation (this feature)

```text
specs/007-pr-release-submit/
├── plan.md              # This file
├── spec.md              # Feature specification
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output
│   └── cli.md           # CLI contract for release submit
└── checklists/
    └── requirements.md  # Quality checklist
```

### Source Code (repository root)

```text
cmd/apx/commands/
├── release.go           # MODIFY: rewrite releaseSubmitAction, remove subtree path, remove --create-pr flag
└── release_test.go      # MODIFY: add tests for new submit flow

internal/publisher/
├── pr.go                # MODIFY: add SubmitReleaseWithPR, FindExistingPR, release branch naming
├── pr_test.go           # MODIFY: add tests for SubmitReleaseWithPR, FindExistingPR
├── manifest.go          # MODIFY: add PR metadata fields (PRNumber, PRURL, PRBranch)
├── manifest_test.go     # MODIFY: update manifest tests for new fields
├── subtree.go           # MODIFY: remove SubtreePublisher usage from release path (kept for apx publish)
├── state.go             # NO CHANGE: StateCanonicalPROpen already exists
└── errors.go            # NO CHANGE: ErrCodePRCreationFailed already exists

testdata/script/
└── release_submit.txt   # CREATE: testscript for release submit PR flow

docs/cli-reference/
└── release-commands.md  # MODIFY: update submit docs to reflect PR-only flow
```

**Structure Decision**: Existing Go project structure is preserved. Changes are scoped to the command layer (`release.go`) and the publisher package (`pr.go`, `manifest.go`). No new packages needed.

## Complexity Tracking

> Constitution Check violation justification

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| VI. Git Subtree removed from release submit | User explicitly requested removing subtree flows. PR-based snapshot provides better safety (review before merge), retry safety, and audit trail. | Keeping subtree as fallback adds complexity, confuses users with two paths, and subtree push bypasses review — the exact problem this feature solves. `apx publish` retains subtree for backward compatibility in non-release workflows. |
