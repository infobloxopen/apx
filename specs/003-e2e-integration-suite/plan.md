# Implementation Plan: End-to-End Integration Test Suite

**Branch**: `003-e2e-integration-suite` | **Date**: November 22, 2025 | **Spec**: [spec.md](./spec.md)  
**Input**: Feature specification from `/specs/003-e2e-integration-suite/spec.md`

## Summary

Create a comprehensive end-to-end integration test suite that validates the complete APX workflow using k3d for Gitea hosting simulation and testscript for test orchestration. The suite will test the full lifecycle: canonical repository bootstrap → app1 schema publication → app2 dependency consumption → cross-repository validation, ensuring git history preservation, overlay management, and breaking change detection work correctly in realistic multi-repository scenarios.

## Technical Context

**Language/Version**: Go 1.24 (matches APX project)  
**Primary Dependencies**: 
- `rogpeppe/go-internal/testscript` (existing - test orchestration)
- `k3d` CLI tool (new - k3s cluster management for Gitea hosting)
- `gitea/gitea` Docker image (new - git hosting simulation)
- `stretchr/testify` (existing - assertions)

**Storage**: 
- Ephemeral k3d volumes for Gitea data (SQLite backend)
- Temporary directories for git repositories
- Container-managed storage for k3d cluster state

**Testing**: 
- Testscript-based E2E scenarios in `testdata/script/e2e_*.txt`
- Go unit tests for test infrastructure (`tests/e2e/`)
- Integration with existing testscript runner (`testscript_test.go`)

**Target Platform**: Linux/macOS (primary CI), Windows (best-effort via WSL2)  
**Project Type**: Single project (Go CLI tool)  
**Performance Goals**: 
- Complete E2E suite execution in <5 minutes (SC-001)
- Gitea startup/teardown in <30 seconds per test
- Zero flakiness (100% pass rate on valid code - SC-007)

**Constraints**: 
- Must run in GitHub Actions without privileged Docker access
- Must clean up all containers/volumes on failure (SC-006)
- Must preserve existing testscript patterns for consistency
- Cannot require manual setup (one-command execution - SC-008)

**Scale/Scope**: 
- 2-3 app repositories per test scenario
- 1 canonical repository shared across tests
- ~10-15 testscript scenarios covering P1-P3 user stories
- ~5 edge case scenarios
- Validates 18 functional requirements from spec

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Test-First Development (Section III) ✅
- **Compliance**: This feature IS the testing infrastructure
- **Actions**: 
  - [ ] Write testscript scenarios BEFORE implementing test helpers
  - [ ] Create contract tests for Gitea API interactions
  - [ ] Unit test k3d lifecycle management functions

### Cross-Platform Path Operations (Section II) ⚠️ **CRITICAL**
- **Compliance**: All git repository paths, module paths, and overlay paths must use `filepath.ToSlash()`
- **Actions**:
  - [ ] Use `filepath.Join()` for all path construction in test helpers
  - [ ] Normalize to forward slashes before git operations
  - [ ] Test on Windows CI runner (best-effort via Docker Desktop/WSL2)

### Documentation-Driven Development (Section I) ✅
- **Compliance**: Tests validate documented workflows in `/docs/getting-started/quickstart.html`
- **Actions**:
  - [ ] Each testscript scenario maps to documented user journey
  - [ ] Test output validation matches documented command outputs
  - [ ] Edge cases derived from documentation assumptions

### GitHub Integration Testing (Section III - Testing Requirements) ✅
- **Compliance**: Uses Gitea as GitHub test double (constitution requirement)
- **Actions**:
  - [ ] Gitea container per test suite for isolation
  - [ ] API token authentication for PR creation
  - [ ] CODEOWNERS enforcement validation
  - [ ] Tag/branch protection pattern validation

### Code Quality & Maintainability (Section IV) ✅
- **Compliance**: Test infrastructure follows same standards as production code
- **Actions**:
  - [ ] Separate concerns: `tests/e2e/gitea/` for Gitea mgmt, `tests/e2e/k3d/` for cluster
  - [ ] Each helper package <300 lines
  - [ ] Comprehensive error messages for test failures (SC-004)

### Re-evaluation After Design ⏳
*To be checked after Phase 1 completes*

## Project Structure

### Documentation (this feature)

```text
specs/003-e2e-integration-suite/
├── plan.md              # This file
├── spec.md              # Feature specification (completed)
├── research.md          # Phase 0: k3d config, Gitea setup, testscript patterns
├── data-model.md        # Phase 1: test entities, Gitea API contracts
├── quickstart.md        # Phase 1: running E2E tests locally
├── contracts/           # Phase 1: Gitea API specs, test fixtures
│   ├── gitea-api.yaml   # OpenAPI spec for Gitea endpoints we use
│   └── test-repos.yaml  # Repository setup contracts
├── checklists/
│   └── requirements.md  # Specification quality checklist (completed)
└── tasks.md             # Phase 2: NOT created by this command
```

### Source Code (repository root)

```text
# E2E Test Infrastructure (NEW)
tests/e2e/
├── main_test.go                 # Test suite entry point
├── gitea/
│   ├── client.go                # Gitea API client (repos, PRs, tags)
│   ├── lifecycle.go             # Container lifecycle (start/stop/cleanup)
│   └── fixtures.go              # Test data (users, repos, tokens)
├── k3d/
│   ├── cluster.go               # k3d cluster management
│   ├── config.go                # Cluster configuration templates
│   └── cleanup.go               # Resource cleanup utilities
├── testhelpers/
│   ├── git.go                   # Git operations (clone, commit, tag)
│   ├── apx.go                   # APX command execution wrappers
│   └── assertions.go            # Custom assertions (git history, PR state)
└── fixtures/
    ├── canonical-repo/          # Canonical repo seed data
    ├── app1-payment/            # App1 test schemas
    └── app2-user/               # App2 test schemas

# Testscript Scenarios (NEW)
testdata/script/e2e/
├── e2e_complete_workflow.txt         # P1: Full publish/consume cycle
├── e2e_cross_repo_deps.txt           # P2: App2 consumes App1, publishes own
├── e2e_git_history.txt               # P2: Verify subtree history preservation
├── e2e_breaking_detection.txt        # P3: Breaking change validation
├── e2e_concurrent_publish.txt        # Edge: Multiple apps publish to same module
├── e2e_existing_pr.txt               # Edge: PR already exists
├── e2e_circular_deps.txt             # Edge: Circular dependency detection
└── e2e_codeowners.txt                # Edge: CODEOWNERS enforcement

# Makefile Updates (MODIFY)
Makefile                          # Add: test-e2e, install-e2e-deps, clean-e2e

# CI Workflows (MODIFY)
.github/workflows/
└── test.yml                      # Add E2E test job

# Existing Test Infrastructure (MODIFY)
testscript_test.go                # Add E2E-specific setup for Gitea
```

**Structure Decision**: 
- **tests/e2e/** contains Go test infrastructure (Gitea/k3d management, helpers)
- **testdata/script/e2e/** contains testscript scenarios (user-facing E2E flows)
- Separation allows unit testing of infrastructure independently of scenarios
- Integrates with existing `testscript_test.go` via conditional Gitea setup
- Follows APX constitution: business logic in `tests/e2e/`, orchestration in testscript

## Complexity Tracking

> **No Constitution violations** - all requirements align with existing principles.
> This feature enhances Test-First Development (Section III) by providing comprehensive E2E validation infrastructure.

---

## Phase 0: Research & Technology Decisions

**Goal**: Resolve all technical uncertainties and establish implementation approach

### Research Tasks

1. **k3d Configuration for CI** ✅ (User specified k3d)
   - Already decided: k3d for lighter weight and faster startup
   - Research: k3d works in GitHub Actions without privileged mode
   - Research: k3d installation via GitHub Actions (setup-k3d action vs curl)
   - Research: Docker image caching strategy for Gitea

2. **Gitea Configuration for Testing**
   - Research: Minimal Gitea config (SQLite, no email, no webhooks needed)
   - Research: Gitea API endpoints for repos, PRs, tags, CODEOWNERS
   - Research: Authentication methods (API token creation, lifetime)
   - Research: Service exposure in k3d (NodePort vs LoadBalancer)
   - Decision: Gitea version (latest stable vs pinned for reproducibility)

3. **Testscript Integration Patterns**
   - Research: How to make Gitea URL available to testscript (env var injection)
   - Research: Secure API token passing (tmpfile vs env var vs testscript variable)
   - Research: Git remote URL injection into test repos
   - Research: Cleanup hooks in testscript (no native defer, use exec on failure)
   - Research: Existing testscript examples in APX codebase

4. **Test Isolation Strategy**
   - Research: Cost/benefit of one k3d cluster per test suite vs per scenario
   - Decision: Repository cleanup approach (delete via API vs recreate Gitea)
   - Research: Future: namespace isolation for parallel execution
   - Research: Test data pollution prevention (unique repo names with timestamps)

5. **Cross-Platform Considerations**
   - Research: k3d on macOS (Docker Desktop requirement, version constraints)
   - Research: k3d on Linux (native Docker, any special config)
   - Research: k3d on Windows (Docker Desktop + WSL2, known issues)
   - Decision: Primary support Linux/macOS, Windows best-effort
   - Research: GitHub Actions runner capabilities (ubuntu-latest Docker access)

**Deliverable**: `research.md` with decisions, alternatives considered, and rationale

---

## Phase 1: Design & Contracts

**Goal**: Define data models, APIs, and test contracts

### 1. Data Model (`data-model.md`)

**Test Environment Entity**:
- Cluster ID (k3d cluster name: `apx-e2e-<timestamp>`)
- Gitea instance URL (`http://localhost:<port>`)
- Admin API token (generated on Gitea startup)
- Created repositories (map: name → clone URL)
- Lifecycle state (setup, running, teardown, failed)
- Cleanup handlers (functions to run on teardown)

**Test Repository Entity**:
- Repository name (`api-schemas`, `payment-service`, `user-service`)
- Repository type (canonical, app1, app2)
- Owner (Gitea organization or user)
- Clone URL (`http://localhost:<port>/testorg/api-schemas.git`)
- Local path (where cloned on test runner)
- Git commits (list of commit SHAs for history validation)
- Published modules (module path, version, git tag)
- Dependencies (for app repos: module path → version)

**Test Assertion Entity**:
- Assertion type (file_exists, git_log_contains, pr_created, tag_exists)
- Expected value (exact match or regex pattern)
- Actual value (captured during execution)
- Pass/fail result (boolean)
- Error context (for SC-004 actionable failure messages)
- Test scenario (which testscript file originated assertion)

**Gitea API Response Entities**:
- Repository (id, name, owner, clone_url, default_branch)
- Pull Request (id, title, state, head_branch, base_branch, commits)
- Tag (name, commit_sha, message)
- User (id, username, email, is_admin)

### 2. API Contracts (`contracts/`)

**Gitea API Contract** (`contracts/gitea-api.yaml`):
```yaml
openapi: 3.0.0
info:
  title: Gitea API (E2E Test Subset)
  version: 1.22.0
paths:
  /api/v1/user/repos:
    post:
      summary: Create repository
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                name: string
                description: string
                private: boolean
                auto_init: boolean
      responses:
        201:
          description: Repository created
          
  /api/v1/repos/{owner}/{repo}:
    get:
      summary: Get repository details
      responses:
        200:
          description: Repository details
          
  /api/v1/repos/{owner}/{repo}/pulls:
    post:
      summary: Create pull request
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                title: string
                head: string
                base: string
                body: string
                
  /api/v1/repos/{owner}/{repo}/tags:
    get:
      summary: List tags
      responses:
        200:
          description: Tag list
```

**Test Repository Setup Contract** (`contracts/test-repos.yaml`):
```yaml
# Canonical Repository Setup
canonical_repo:
  name: api-schemas
  owner: testorg
  type: canonical
  initialize_via: apx init canonical --org=testorg --repo=api-schemas --skip-git
  expected_structure:
    - proto/
    - openapi/
    - avro/
    - jsonschema/
    - parquet/
    - buf.yaml
    - buf.work.yaml
    - catalog/catalog.yaml
    - CODEOWNERS
  validation:
    - file_exists: buf.yaml
    - file_contains: 
        file: buf.yaml
        pattern: "version: v2"
    - file_exists: catalog/catalog.yaml
    - file_contains:
        file: catalog/catalog.yaml
        pattern: "org: testorg"

# App Repository 1: Payment Service
app1_payment:
  name: payment-service
  owner: testorg
  type: app
  initialize_via: apx init app --org=testorg --non-interactive internal/apis/proto/payments/ledger/v1
  expected_structure:
    - internal/apis/proto/payments/ledger/v1/
    - internal/apis/proto/payments/ledger/v1/ledger.proto
    - apx.yaml
  publishes:
    - module: proto/payments/ledger/v1
      version: v1.0.0
      tag: proto/payments/ledger/v1/v1.0.0
      canonical_tag: proto/payments/ledger/v1.0.0
  validation:
    - file_exists: internal/apis/proto/payments/ledger/v1/ledger.proto
    - git_tag_exists: proto/payments/ledger/v1/v1.0.0
      
# App Repository 2: User Service
app2_user:
  name: user-service
  owner: testorg
  type: app
  initialize_via: apx init app --org=testorg --non-interactive internal/apis/proto/users/profile/v1
  depends_on:
    - module: proto/payments/ledger/v1
      version: v1.0.0
      via: apx add proto/payments/ledger/v1@v1.0.0
  publishes:
    - module: proto/users/profile/v1
      version: v1.0.0
      tag: proto/users/profile/v1/v1.0.0
  validation:
    - file_exists: apx.lock
    - file_contains:
        file: apx.lock
        pattern: "proto/payments/ledger/v1"
    - dir_exists: internal/gen/go/proto/payments/ledger@v1.0.0
```

### 3. Quickstart Guide (`quickstart.md`)

Content will include:

**Prerequisites**:
- Docker Desktop (macOS/Windows) or Docker Engine (Linux)
- Go 1.24+
- Make

**Installation**:
```bash
# Install k3d
make install-e2e-deps

# Verify installation
k3d version
kubectl version --client
```

**Running Tests**:
```bash
# Run full E2E suite
make test-e2e

# Run specific scenario
go test ./tests/e2e -run TestE2E/e2e_complete_workflow

# Debug mode (keeps Gitea running after test)
E2E_DEBUG=1 make test-e2e

# View Gitea web UI during debug
# Navigate to http://localhost:<port> (port printed in logs)
```

**Troubleshooting**:
- k3d cluster stuck: `k3d cluster delete apx-e2e-*`
- Orphaned containers: `make clean-e2e`
- Gitea slow startup: Pre-pull image with `docker pull gitea/gitea:1.22`

**CI Integration**:
```yaml
# .github/workflows/test.yml
e2e-tests:
  runs-on: ubuntu-latest
  timeout-minutes: 10
  steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-go@v5
      with:
        go-version: '1.24'
    - name: Install k3d
      run: make install-e2e-deps
    - name: Run E2E Tests
      run: make test-e2e
```

---

## Phase 2: Task Breakdown

**This phase will be executed by `/speckit.tasks` to generate `tasks.md`**

Estimated task structure:

### Infrastructure (8 tasks)
- T001: Create k3d cluster manager
- T002: Create Gitea lifecycle manager
- T003: Implement Gitea API client
- T004: Create cleanup utilities
- T005: Add Makefile targets
- T006: Add GitHub Actions workflow
- T007: Create test fixtures
- T008: Integration with testscript_test.go

### Test Helpers (6 tasks)
- T009: Git operations wrapper
- T010: APX command wrappers
- T011: Custom assertions
- T012: Environment setup helper
- T013: Repository validation helper
- T014: PR validation helper

### Testscript Scenarios (12 tasks)
- T015: e2e_complete_workflow.txt (P1)
- T016: e2e_cross_repo_deps.txt (P2)
- T017: e2e_git_history.txt (P2)
- T018: e2e_breaking_detection.txt (P3)
- T019-T026: Edge case scenarios

### Documentation & Polish (3 tasks)
- T027: Update README with E2E test instructions
- T028: Create troubleshooting guide
- T029: Performance benchmarking

**Total**: ~29 tasks

---

## Implementation Notes

### Key Technical Decisions

1. **k3d over kind**: 
   - Lighter weight (k3s vs full Kubernetes)
   - Faster startup (<30s vs ~1-2min)
   - Better suited for ephemeral test clusters
   - Simpler networking (direct port mapping)

2. **Testscript orchestration**: 
   - Leverages existing APX test infrastructure
   - Readable, declarative test scenarios
   - Easy to add new scenarios without Go code
   - Good error messages out of the box

3. **Gitea in k3d**: 
   - Realistic git hosting without GitHub API limits
   - No external dependencies or auth setup
   - Complete control over repository state
   - Supports all git operations (clone, push, PR, tags)

4. **Per-suite cluster** (not per-test):
   - Balance between isolation and performance
   - Cluster creation overhead (~20-30s) amortized
   - Repository cleanup via API between tests
   - Future: parallel test execution via namespaces

5. **Cleanup via defer**:
   - Go test cleanup handlers (t.Cleanup)
   - Guaranteed execution even on panic
   - Idempotent cleanup (safe to run multiple times)
   - `make clean-e2e` safety net for manual cleanup

### Risk Mitigation

| Risk | Impact | Mitigation | Residual Risk |
|------|--------|------------|---------------|
| k3d not in CI | High | Install via Makefile, cache binaries | Low - documented install process |
| Gitea slow startup | Medium | Pre-pull image, minimal config, health checks | Low - startup time acceptable |
| Flaky tests | High | Explicit wait/poll, deterministic data, retry logic | Low - testscript provides stability |
| Resource leaks | Medium | Defer cleanup, post-test validation, `make clean-e2e` | Low - multiple safety nets |
| Cross-platform fail | Low | Primary Linux/macOS, Windows best-effort | Acceptable - most users on Unix |
| Port conflicts | Low | Dynamic port allocation, check before start | Very Low - ephemeral ports |

### Success Metrics Mapping

| Success Criterion | Validation Method | Acceptance Criteria |
|-------------------|-------------------|---------------------|
| SC-001: <5min execution | CI workflow duration logs | All scenarios complete in <5min |
| SC-002: 95% regression detection | Map tests to documented workflows | All documented scenarios covered |
| SC-003: 100% CI pass rate | GitHub Actions status | No failures on main branch |
| SC-004: Actionable errors | Manual test review | Error messages include fix suggestions |
| SC-005: Workflow validation | Testscript scenario completion | All FR requirements validated |
| SC-006: Zero orphaned resources | Post-test `docker ps`, `k3d cluster list` | No containers or clusters remain |
| SC-007: Zero flakiness | 10x sequential CI runs | 100% pass rate (no intermittent failures) |
| SC-008: One-command execution | Developer testing | `make test-e2e` works without manual steps |
| SC-009: Cross-platform | CI matrix (Linux/macOS) | Tests pass on both platforms |
| SC-010: Edge case coverage | Test scenario count | At least 3 edge cases validated |

---

## Next Steps

**Immediate** (this command execution):
1. ✅ Constitution Check passed
2. ⏳ Generate `research.md` (Phase 0)
3. ⏳ Generate `data-model.md` (Phase 1)
4. ⏳ Generate `contracts/` files (Phase 1)
5. ⏳ Generate `quickstart.md` (Phase 1)
6. ⏳ Update agent context file

**Future** (separate commands):
7. Execute `/speckit.tasks` to generate detailed `tasks.md`
8. Begin implementation following TDD approach
9. Iterate on test scenarios based on real execution
10. Integrate with CI pipeline
