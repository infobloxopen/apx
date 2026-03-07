# Implementation Plan: Docs-CLI Consistency

**Branch**: `005-docs-cli-consistency` | **Date**: 2026-03-07 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/005-docs-cli-consistency/spec.md`

## Summary

Make all APX documentation match the current Cobra-based CLI implementation. The binary is the source of truth. Documentation that references non-existent commands, wrong flags, or incompatible config schemas is rewritten to match what the CLI actually does. Commands that are documented but not implemented are marked with a visible "Planned" admonition. A doc-parity test suite ensures docs stay accurate across releases.

## Technical Context

**Language/Version**: Go 1.26.1
**Primary Dependencies**: cobra v1.10.2, charmbracelet/huh v0.8.0, gopkg.in/yaml.v3
**Storage**: Files (`apx.yaml`, `apx-lock.yaml`, `catalog/catalog.yaml`)
**Testing**: go test + rogpeppe/go-internal/testscript, existing `doc_parity_test.go`
**Target Platform**: macOS (primary), Linux, Windows
**Project Type**: CLI tool
**Performance Goals**: N/A (documentation changes + test additions)
**Constraints**: All docs must work with Sphinx/MyST + sphinx_design (for admonitions)
**Scale/Scope**: ~16 markdown files, ~14 command files, 1 config struct, 1 lockfile struct

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Documentation-Driven Development | **TENSION** | Constitution says "docs first, implementation validates against docs." This feature inverts that: implementation is source of truth because docs are stale. Justified: the docs were written speculatively before commands existed. The correct action is to fix docs to match what was actually built, then re-establish the docs-first workflow going forward. |
| II. Cross-Platform Path Operations | PASS | No path code changes. |
| III. Test-First Development | PASS | Doc-parity tests will be written/fixed before docs are updated. |
| III. Code Quality & Maintainability | PASS | Minimal code changes (only `init`-generated YAML and `apx.example.yaml`). |
| IV. Developer Experience First | PASS | Core goal is making the developer experience trustworthy. |
| V. Canonical Import Paths | PASS | No changes to import path strategy. |
| VI. Git Subtree Publishing | PASS | No changes to publishing strategy. |
| VII. Multi-Format Schema Support | PASS | No changes to format support. |

**Tension Resolution for Principle I**: The constitution assumes docs are authoritative and implementation tracks them. This feature is a one-time correction where speculative docs diverged from reality (9 commands documented that were never implemented, wrong command names, incompatible config schemas). After this feature, the docs-first workflow resumes: docs are accurate, and any future implementation change must update docs first. This is explicitly called out in the spec's Assumptions section.

## Project Structure

### Documentation (this feature)

```text
specs/005-docs-cli-consistency/
├── plan.md              # This file
├── research.md          # Phase 0: research findings
├── data-model.md        # Phase 1: entity definitions
├── quickstart.md        # Phase 1: implementation quickstart
├── contracts/           # Phase 1: test contracts
│   └── doc-parity-test-contract.md
└── checklists/
    └── requirements.md  # Spec quality checklist
```

### Source Code (changes scoped to)

```text
cmd/apx/commands/
├── doc_parity_test.go   # MODIFY — expand to cover all commands/flags
└── (no other command changes)

internal/
├── schema/
│   └── app.go           # MODIFY — fix generated apx.yaml to match Config struct
└── config/
    └── config.go        # READ ONLY — source of truth for config schema

testdata/
├── golden/
│   └── help/
│       └── main.golden  # Already up-to-date (cobra format)
└── script/
    └── doc_parity.txt   # NEW — testscript for doc examples

docs/                    # MODIFY — all 14+ markdown files
├── getting-started/
│   ├── quickstart.md
│   ├── installation.md
│   └── interactive-init.md
├── cli-reference/
│   └── index.md
├── publishing/
│   └── index.md
├── app-repos/
│   └── index.md
├── canonical-repo/
│   ├── index.md
│   └── structure.md
├── dependencies/
│   └── index.md
├── troubleshooting/
│   └── faq.md
├── index.md
└── target.md

apx.example.yaml         # MODIFY — add missing fields
README.md                 # MODIFY — fix command names, flags, examples
INTERACTIVE_INIT.md       # MODIFY — remove urfave/cli references
```

## Phase 0: Research

All research findings are consolidated in [research.md](research.md).

### Key Decisions

1. **Source of truth**: The compiled binary (cobra command tree) is authoritative. All docs align to it.
2. **Non-existent commands**: Marked as "Planned" using sphinx_design `{admonition}` blocks, not removed — they represent roadmap intent.
3. **Config schema fix**: `apx init app` will be fixed to generate YAML that matches the `Config` struct (code fix, not just docs).
4. **`apx.example.yaml`**: Already matches `Config` struct (has `version: 1`, `org`, `repo`). No change needed.
5. **Parity tests**: Expand existing `doc_parity_test.go` to cover every command + flag, and add a testscript that validates command examples from docs.
6. **Toctree broken links**: Create minimal stub pages for the 27 broken references rather than removing them (they represent planned doc sections).
7. **Exit codes**: Define one canonical table in `cli-reference/index.md` based on `main.go` actual codes (0 success, 6 validation, 1 other).
8. **`apx init` with 1 arg**: The code requires 0 or 2 args. Docs will be updated to match (0 = interactive, 2 = `<kind> <modulePath>`).

## Phase 1: Design

See [data-model.md](data-model.md), [contracts/](contracts/), and [quickstart.md](quickstart.md).

## Phase 2: Tasks

Tasks will be generated by `/speckit.tasks`. High-level work breakdown:

### Task Group 1: Code Fixes (must happen first)
- T001: Fix `apx init app` to generate valid `Config`-compatible YAML
- T002: Fix `doc_parity_test.go` — correct search args, add all-commands coverage, add all-flags coverage

### Task Group 2: Core Documentation Rewrites
- T003: Rewrite `docs/cli-reference/index.md` — rebuild from cobra command tree
- T004: Rewrite `docs/getting-started/quickstart.md` — fix all command examples
- T005: Rewrite `docs/getting-started/interactive-init.md` — fix init examples, remove urfave refs
- T006: Fix `README.md` — command names, flags, examples, exit codes

### Task Group 3: Workflow Documentation
- T007: Rewrite `docs/publishing/index.md` — fix CI templates, command names
- T008: Rewrite `docs/dependencies/index.md` — fix lockfile format, mark planned commands
- T009: Fix `docs/app-repos/index.md` — fix workflow examples
- T010: Fix `docs/canonical-repo/index.md` + `structure.md` — fix catalog format
- T011: Fix `docs/troubleshooting/faq.md` — fix all command references

### Task Group 4: Broken Links & Stubs
- T012: Create stub pages for all 27 broken toctree references
- T013: Fix `INTERACTIVE_INIT.md` — remove urfave/cli references

### Task Group 5: Validation
- T014: Add `testdata/script/doc_parity.txt` testscript — validate key doc examples
- T015: Run Sphinx build, confirm zero broken-link warnings
- T016: Run full test suite, confirm all parity tests pass
