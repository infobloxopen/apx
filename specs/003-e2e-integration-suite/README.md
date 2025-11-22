# Feature: End-to-End Integration Test Suite

**Branch**: `003-e2e-integration-suite`  
**Created**: November 22, 2025  
**Status**: Planning Complete - Ready for Implementation

## Overview

Comprehensive end-to-end integration test suite that validates the complete APX workflow using k3d for Gitea hosting simulation and testscript for test orchestration. Tests the full lifecycle: canonical repository bootstrap → app1 schema publication → app2 dependency consumption → cross-repository validation.

## Documentation

- **[spec.md](./spec.md)** - Feature specification with user stories and requirements
- **[plan.md](./plan.md)** - Implementation plan with technical context and architecture
- **[research.md](./research.md)** - Research findings and technical decisions (Phase 0 ✅)
- **[data-model.md](./data-model.md)** - Entity definitions and relationships (Phase 1 ✅)
- **[quickstart.md](./quickstart.md)** - Developer quickstart guide (Phase 1 ✅)
- **[contracts/](./contracts/)** - API contracts and test specifications (Phase 1 ✅)
  - `gitea-api.yaml` - OpenAPI spec for Gitea endpoints
  - `test-repos.md` - Repository setup contracts and validations
- **[checklists/](./checklists/)** - Quality checklists
  - `requirements.md` - Specification quality checklist ✅
- **tasks.md** - NOT YET CREATED (use `/speckit.tasks` to generate)

## Key Features

### Testing Infrastructure

**k3d Cluster Management**:
- Lightweight k3s clusters in Docker
- Fast startup (<30s) and teardown (<5s)
- No privileged mode required for GitHub Actions
- Cross-platform support (Linux, macOS, Windows/WSL2)

**Gitea Git Hosting**:
- Realistic GitHub simulation without API limits
- SQLite backend for zero external dependencies
- Complete git protocol support (clone, push, PR, tags)
- API token authentication for test automation

**Testscript Orchestration**:
- Leverages existing APX test infrastructure
- Readable, declarative test scenarios
- Easy to add new scenarios without Go code
- Excellent error messages and debugging support

### Test Coverage

**Priority 1: Complete Publishing Workflow** (5 scenarios)
- Canonical repository bootstrap
- App repository initialization
- Schema publication via `apx publish`
- Pull request creation and validation
- Overlay generation for consumers

**Priority 2: Cross-Repository Dependencies** (4 scenarios)
- App2 consumes App1's published API
- Dependency resolution via `apx add`
- Overlay creation via `apx gen go`
- App2 publishes its own API while consuming App1

**Priority 2: Git History Preservation** (3 scenarios)
- Validate git subtree split preserves commits
- Verify authorship attribution in canonical repo PRs
- Check commit messages and timestamps intact

**Priority 3: Breaking Change Detection** (4 scenarios)
- Detect breaking schema changes
- Prevent incompatible API evolution
- Allow non-breaking additions
- Support major version bumps

**Edge Cases** (8 scenarios)
- Concurrent publication to same module
- PR already exists for module
- Circular dependency detection
- CODEOWNERS enforcement
- Gitea connectivity failures
- Tag conflicts
- Resource cleanup validation

## Technical Architecture

### Stack

- **Language**: Go 1.24
- **Cluster**: k3d (k3s in Docker)
- **Git Hosting**: Gitea 1.22 (SQLite backend)
- **Test Framework**: go-internal/testscript
- **Assertions**: stretchr/testify
- **CI**: GitHub Actions (ubuntu-latest primary)

### Directory Structure

```
tests/e2e/                          # E2E test infrastructure
├── main_test.go                    # Test suite entry point
├── gitea/                          # Gitea management
│   ├── client.go                   # API client
│   ├── lifecycle.go                # Container lifecycle
│   └── fixtures.go                 # Test data
├── k3d/                            # k3d cluster management
│   ├── cluster.go
│   ├── config.go
│   └── cleanup.go
├── testhelpers/                    # Reusable helpers
│   ├── git.go
│   ├── apx.go
│   └── assertions.go
└── fixtures/                       # Test fixtures
    ├── canonical-repo/
    ├── app1-payment/
    └── app2-user/

testdata/script/e2e/                # Testscript scenarios
├── e2e_complete_workflow.txt
├── e2e_cross_repo_deps.txt
├── e2e_git_history.txt
├── e2e_breaking_detection.txt
└── ... (edge cases)
```

## Success Criteria

| Criterion | Target | Status |
|-----------|--------|--------|
| Execution time | <5 minutes | Plan defined |
| Regression detection | 95%+ | Plan defined |
| CI pass rate | 100% on main | Plan defined |
| Actionable errors | All failures | Plan defined |
| Workflow coverage | All FR validated | Plan defined |
| Resource cleanup | Zero orphaned | Plan defined |
| Flakiness | 0% (100 pass rate) | Plan defined |
| One-command execution | `make test-e2e` | Plan defined |
| Cross-platform | Linux + macOS | Plan defined |
| Edge case coverage | 3+ validated | Plan defined |

## Progress

### Phase 0: Research ✅ COMPLETE
- [x] k3d configuration for CI
- [x] Gitea minimal setup (SQLite, API endpoints)
- [x] Testscript integration patterns
- [x] Test isolation strategy
- [x] Cross-platform considerations

### Phase 1: Design ✅ COMPLETE
- [x] Data model (test entities, relationships)
- [x] API contracts (Gitea OpenAPI spec)
- [x] Repository contracts (canonical, app1, app2)
- [x] Quickstart guide (running tests locally)
- [x] Agent context updated

### Phase 2: Tasks ⏳ PENDING
- [ ] Execute `/speckit.tasks` to generate detailed task breakdown
- [ ] Estimate: ~29 tasks across infrastructure, helpers, scenarios, CI

### Phase 3: Implementation ⏳ PENDING
- [ ] TDD approach: write testscript scenarios first
- [ ] Implement infrastructure (k3d, Gitea)
- [ ] Implement test helpers
- [ ] Integrate with CI

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Cluster orchestration | k3d | Lighter than kind, faster startup, no privileged mode |
| Git hosting | Gitea in k3d | Realistic, no external deps, full git protocol |
| Gitea backend | SQLite | Zero config, ephemeral, fast |
| Gitea version | 1.22 (pinned) | Stable API, reproducible tests |
| Authentication | API tokens | Simpler than SSH, cross-platform |
| Service exposure | NodePort + port mapping | Simple, works in CI |
| Test orchestration | Testscript | Existing APX pattern, readable |
| Cluster lifecycle | Per test suite | Balance isolation vs performance (3min vs 10min) |
| Repository cleanup | API deletion | Fast, idempotent |
| Gitea URL passing | Environment variable | Native testscript support |
| Cleanup guarantee | Go t.Cleanup() | LIFO execution, panic-safe |
| Primary platform | Linux (Ubuntu) | GitHub Actions default, best performance |
| Secondary platform | macOS | Developer machines |
| Windows support | Best-effort (WSL2) | Complex setup, not CI target |

## Next Steps

1. **Execute `/speckit.tasks`**: Generate detailed task breakdown (`tasks.md`)
2. **Review plan**: Team approval of architecture and approach
3. **Begin implementation**: Start with infrastructure (k3d, Gitea)
4. **Write scenarios**: Create testscript files for P1 user stories
5. **Implement helpers**: Build test utilities and assertions
6. **CI integration**: Add GitHub Actions workflow
7. **Iterate**: Refine based on real execution feedback

## Constitution Compliance

✅ **Test-First Development** - This feature IS the testing infrastructure  
✅ **Cross-Platform Paths** - All path operations use `filepath.ToSlash()`  
✅ **Documentation-Driven** - Tests validate documented quickstart workflows  
✅ **GitHub Integration** - Uses Gitea as test double (constitution requirement)  
✅ **Code Quality** - Test infrastructure follows production code standards

No constitution violations - all requirements align with existing principles.

## References

- [k3d Documentation](https://k3d.io/)
- [Gitea API Reference](https://docs.gitea.com/api/)
- [Testscript Package](https://pkg.go.dev/github.com/rogpeppe/go-internal/testscript)
- [APX Constitution](../../.specify/memory/constitution.md)
- [APX Documentation](https://infobloxopen.github.io/apx/)

---

**Planning Status**: ✅ Complete - All Phase 0 and Phase 1 deliverables generated  
**Ready for**: `/speckit.tasks` to generate implementation tasks  
**Branch**: `003-e2e-integration-suite` (active)
