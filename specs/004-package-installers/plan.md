# Implementation Plan: Package Installer Support

**Branch**: `004-package-installers` | **Date**: 2026-03-07 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/004-package-installers/spec.md`

## Summary

Add support for installing APX via Homebrew (macOS/Linux), Scoop (Windows), and a shell one-liner (curl|bash), all driven from the APX repo with no third-party registry registration. Implementation extends the existing GoReleaser v2 configuration, adds a GitHub Actions release workflow triggered on version tags, and provides a standalone install script. All package manager repositories (`homebrew-tap`, `scoop-bucket`) are self-hosted under the `infobloxopen` org.

## Technical Context

**Language/Version**: Go 1.26.1 (CLI), Bash (install script), YAML (CI/GoReleaser config)  
**Primary Dependencies**: GoReleaser v2 (release automation), GitHub Actions (CI/CD), goreleaser-action@v6  
**Storage**: N/A — no runtime data; config files only  
**Testing**: `goreleaser check` (config validation), `goreleaser release --snapshot --clean` (dry run), manual verification post-release  
**Target Platform**: linux/darwin × amd64/arm64, windows/amd64  
**Project Type**: CLI — this feature is build/release infrastructure, not application code  
**Performance Goals**: Release pipeline completes in <10 minutes; install script completes in <30 seconds  
**Constraints**: No third-party registry registration; everything self-hosted under `infobloxopen` GitHub org  
**Scale/Scope**: 3 package managers (Homebrew, Scoop, shell installer) + GitHub Releases (already working)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### I. Documentation-Driven Development — PASS
- Documentation updates are part of the spec (FR-010): README.md and `docs/getting-started/installation.md` are updated with all new installation methods BEFORE announcing the feature.
- Contract: [contracts/documentation.md](contracts/documentation.md) defines exact content changes.

### II. Cross-Platform Path Operations — PASS (N/A)
- This feature does not add Go code with path operations. The install script uses `uname` for OS detection, not Go `filepath`. No path manipulation in Go.

### III. Test-First Development — PARTIAL PASS
- GoReleaser config is validated with `goreleaser check` and `--snapshot` builds.
- Install script can be tested in CI with a mock release.
- No new Go code is introduced, so no new unit/integration tests are required for `internal/` packages.
- Note: A testscript for `goreleaser check` could be added but is not strictly required since this is config, not business logic.

### IV. Code Quality & Maintainability — PASS
- No new Go packages or code files. Changes are config (YAML) and scripts (bash).
- Install script follows bash best practices: `set -euo pipefail`, `main()` wrapper, shellcheck-clean.

### V. Developer Experience First — PASS
- Copy-pasteable install commands for all platforms.
- One-line install for quick onboarding.
- Automated release — maintainers just push a tag.

### VI. Canonical Import Paths — PASS (N/A)
- No import path changes.

### VII. Multi-Format Schema Support — PASS (N/A)
- No schema format changes.

**Gate Result**: PASS — no violations. Proceed to implementation.

### Post-Design Re-Check (after Phase 1)

All gates remain PASS. No new Go code was introduced in the design. The install script, GoReleaser config, and release workflow are all infrastructure artifacts that don't interact with APX's internal packages or path operations.

## Project Structure

### Documentation (this feature)

```text
specs/004-package-installers/
├── plan.md              # This file
├── research.md          # Phase 0: 7 research decisions (R-001 through R-007)
├── data-model.md        # Phase 1: config entities and relationships
├── quickstart.md        # Phase 1: end-to-end release guide
├── contracts/           # Phase 1: interface contracts
│   ├── goreleaser.md    #   GoReleaser config changes
│   ├── release-workflow.md  # GitHub Actions release workflow
│   ├── install-script.md    # Shell installer interface
│   └── documentation.md     # README/docs updates
└── tasks.md             # Phase 2 (created by /speckit.tasks)
```

### Source Code (repository root)

```text
.goreleaser.yml                          # MODIFY: add scoops section, add token to brews
.github/workflows/release.yml           # CREATE: release workflow triggered on v* tags
scripts/install.sh                       # CREATE: curl|bash installer
Makefile                                 # MODIFY: update GORELEASER_VERSION to v2.x
README.md                               # MODIFY: add Scoop + shell installer to Installation
docs/getting-started/installation.md    # MODIFY: add Scoop + shell installer sections
```

**Structure Decision**: This feature is purely infrastructure — no new Go packages, no new `internal/` code, no new `cmd/` commands. All changes are config files, scripts, and documentation.

### External Prerequisites (not in this repo)

| Item | Owner | Action |
|------|-------|--------|
| `infobloxopen/homebrew-tap` repo | Org admin | Create empty repo |
| `infobloxopen/scoop-bucket` repo | Org admin | Create empty repo |
| `HOMEBREW_TAP_TOKEN` secret | Org admin | Fine-grained PAT → repo secret |
| `SCOOP_BUCKET_TOKEN` secret | Org admin | Fine-grained PAT → repo secret |

## Complexity Tracking

No constitution violations to justify. This feature is straightforward infrastructure work with no new Go code, no new packages, and no architectural changes.
