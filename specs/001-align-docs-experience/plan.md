# Implementation Plan: Docs-Aligned APX Experience

**Branch**: `001-align-docs-experience` | **Date**: 2025-11-21 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/001-align-docs-experience/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Deliver APX CLI behavior that mirrors the `/docs` getting started workflows end to end—from canonical repo initialization through schema authoring/publishing and consumer overlays—using `urfave/cli/v2` commands enhanced with survey-driven interactivity, doc-parity validation, and self-hosted GitHub compatibility.

## Technical Context

<!--
  ACTION REQUIRED: Replace the content in this section with the technical details
  for the project. The structure here is presented in advisory capacity to guide
  the iteration process.
-->

**Language/Version**: Go 1.24  
**Primary Dependencies**: `github.com/urfave/cli/v2` (command wiring), `github.com/AlecAivazis/survey/v2` (interactive prompts), `github.com/infobloxopen/apx/internal/*` packages for business logic, external CLI toolchains (buf, spectral, oasdiff, etc.)  
**Storage**: N/A (file system + git only)  
**Testing**: `go test` for unit packages, `testscript` suites in `testdata/script` for CLI flows, Gitea-backed integration harness for GitHub workflows  
**Target Platform**: Developer workstations (macOS/Linux) plus CI runners (GitHub Actions & self-hosted)  
**Project Type**: Go monorepo delivering a CLI binary (`cmd/apx`)  
**Performance Goals**: CLI validations finish in <5s for typical schemas; doc parity checks run as part of command execution without noticeable lag  
**Constraints**: Must operate offline/air-gapped with pre-fetched tool bundles, align CLI outputs verbatim with `/docs`, enforce canonical import paths, adhere to documentation-first + TDD constitution  
**Scale/Scope**: Supports organization-level API portfolios (dozens of domains, multi-format schemas) with simultaneous producer/consumer teams

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Documentation-Driven Development**: `/docs` defines the canonical UX; plan must trace every CLI flow back to documented steps before implementation begins. ✅ Planned to implement getting started guide first and add doc parity checks.
- **Test-First Development**: Commit to writing unit, testscript, and Gitea scenarios prior to coding (`internal/*`, CLI commands, GH workflows). ✅ Will design tests per user stories before implementation tasks.
- **Code Quality & Maintainability**: Keep CLI orchestration in `cmd/apx/commands`, delegate logic to focused `internal/` packages, enforce error wrapping and UI output patterns. ✅ Plan respects existing package boundaries.
- **Canonical Import Paths**: Any code generation or sync work must preserve canonical imports with `go.work` overlays; generated artifacts remain untracked. ✅ Scope includes validation of overlays and sync behaviors.
- **Git Subtree Publishing**: Publishing workflows must continue to use `git subtree` preserving history. ✅ Publishing enhancements will interact via `internal/publisher` without altering strategy.
- **Multi-Format Schema Support**: Breaking/lint commands must route to format-specific validators. ✅ Research phase will map required tool integrations per format.

**Gate Status (Initial)**: PASS (no constitution violations identified; all principles acknowledged in plan scope).

**Gate Status (Post-Design Review)**: PASS (research, data model, contracts, and quickstart reinforce documentation-first, TDD, architecture, and multi-format commitments without requiring exceptions).

## Project Structure

### Documentation (this feature)

```text
specs/[###-feature]/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── overlays.md          # Overlay design documentation (created during implementation)
├── contracts/           # Phase 1 output (/speckit.plan command)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)
```text
cmd/apx/
├── main.go               # CLI entrypoint (urfave/cli wiring)
└── commands/
    ├── init.go
    ├── lint.go
    ├── breaking.go
    ├── publish.go
    ├── gen.go
    ├── config.go
    ├── semver.go
    ├── catalog.go
    ├── policy.go
    └── common.go

internal/
├── config/
├── detector/
├── interactive/
├── schema/
├── publisher/
├── ui/
└── validator/           # (planned) format-specific lint/break checks

testdata/
└── script/              # testscript integration suites for CLI flows

tests/
└── integration/

docs/
└── getting-started/     # target-state experience referenced by spec
```

**Structure Decision**: Maintain existing CLI command layout under `cmd/apx/commands`, extend `internal/` packages (e.g., `validator`, `publisher`) to encapsulate business logic, and expand `testdata/script` plus Gitea-backed integration suites to mirror documented workflows.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

N/A (no constitution violations proposed).
