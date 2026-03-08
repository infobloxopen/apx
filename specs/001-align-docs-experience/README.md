# Feature: Docs-Aligned APX Experience

**Branch**: `001-align-docs-experience`  
**Status**: Implementation In Progress (84% Complete - 76/90 tasks)

## Overview

Align the APX CLI experience with the `/docs` getting-started workflows, ensuring commands, prompts, and outputs match documented behavior for canonical repository initialization, schema publishing, and consumer overlay management.

## Documentation

- [spec.md](./spec.md) - Feature specification with user stories and requirements
- [plan.md](./plan.md) - Implementation plan with technical context and architecture
- [research.md](./research.md) - Research findings and technical decisions
- [data-model.md](./data-model.md) - Entity definitions and relationships
- [quickstart.md](./quickstart.md) - Developer quickstart guide
- **[overlays.md](./overlays.md)** - **Go workspace overlays design documentation**
- [contracts/](./contracts/) - API contracts and test specifications
- [tasks.md](./tasks.md) - Detailed task breakdown and execution plan

## Key Concepts

### Go Workspace Overlays

The overlay mechanism enables applications to use **canonical import paths** (e.g., `github.com/org/apis/proto/payments/ledger/v1`) during local development while transparently resolving them to locally generated code in `internal/gen/`. When ready, developers remove the overlay and fetch the published module - the same import paths now resolve to the published package **without any code changes**.

See [overlays.md](./overlays.md) for complete design documentation including:
- Problem statement and solution approach
- Directory structure and go.work management
- Overlay lifecycle (create, sync, remove)
- Developer workflows and best practices
- Troubleshooting guide

## Progress

- ✅ Phase 1: Setup (5/5 tasks)
- ✅ Phase 2: Foundation (15/15 tasks)
- ✅ Phase 3: User Story 1 - Canonical Repository Bootstrap (12/12 tasks)
- ✅ Phase 4: User Story 2 - Schema Publishing Workflow (19/20 tasks, T052 optional)
- ✅ Phase 5: User Story 3 - Consumer Overlay Management (24/24 tasks)
- 🔄 Phase 6: Polish (0/14 tasks)

**Total**: 76/90 tasks complete (84%)

## Getting Started

See [quickstart.md](./quickstart.md) for developer setup and validation workflows.

## Constitution Compliance

This feature adheres to the APX project constitution:
- ✅ Documentation-Driven Development (docs define UX)
- ✅ Test-First Development (TDD methodology)
- ✅ Code Quality & Maintainability
- ✅ Canonical Import Paths (via overlays)
- ✅ Git Subtree Publishing
- ✅ Multi-Format Schema Support
