# Feature Specification: End-to-End Integration Test Suite

**Feature Branch**: `003-e2e-integration-suite`  
**Created**: November 22, 2025  
**Status**: Draft  
**Input**: User description: "Create an end to end integration suite that uses kind or k3d to deploy gitea. This will bootstrap an API repo, bootstrap one new app repo that creates a local API and publishes it, then bootstrap another app repo (app2) that consumes that API and publishes its own API as well."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Complete Publishing Workflow Validation (Priority: P1)

As a developer contributing to the APX project, I need automated end-to-end tests that validate the complete schema publishing workflow from app repository creation through canonical repository publication, so that I can confidently merge changes knowing they work in realistic git hosting scenarios.

**Why this priority**: This is the core value proposition of APX - the ability to publish schemas from app repos to canonical repos via git operations. Without validated E2E tests, we risk shipping breaking changes that only manifest in production GitHub workflows.

**Independent Test**: Can be fully tested by running a single test suite that spins up Gitea, creates repositories, executes `apx init`, `apx publish`, and validates the resulting PR and git history. Delivers confidence that the publishing workflow works end-to-end.

**Acceptance Scenarios**:

1. **Given** a fresh Gitea instance and no existing repositories, **When** I run `apx init canonical` to bootstrap an API repository, **Then** the canonical repository is created with correct structure (buf.yaml, buf.work.yaml, CODEOWNERS, catalog/catalog.yaml) and initial commit

2. **Given** a bootstrapped canonical repository, **When** I create a new app repository and run `apx init app internal/apis/proto/payments/ledger`, **Then** the app repository contains the correct schema structure with apx.yaml configuration

3. **Given** an app repository with a Protocol Buffer schema, **When** I run `apx publish --module-path=internal/apis/proto/payments/ledger`, **Then** a Pull Request is created in the canonical repository with git subtree history preserved, showing all commits from the app repository

4. **Given** a published schema in the canonical repository, **When** I create a second app repository and run `apx add proto/payments/ledger/v1@v1.0.0`, **Then** the dependency is added to apx.lock and go.work overlay is created in internal/gen/go

5. **Given** an app repository with an overlay dependency, **When** I run `apx gen go`, **Then** generated Go code uses canonical import paths (github.com/org/apis/proto/payments/ledger/v1) that resolve to the local overlay

---

### User Story 2 - Cross-Repository Dependency Chain (Priority: P2)

As a developer, I need to validate that schema dependencies work across multiple application repositories, so that I can trust the consumer workflow when teams depend on each other's APIs.

**Why this priority**: Multi-repository dependencies are a common real-world scenario. This validates that the overlay mechanism and dependency resolution work when app2 consumes app1's published schema while also publishing its own schema.

**Independent Test**: Can be tested by creating app1 (publishes payment API), app2 (consumes payment API + publishes user API), and validating that app2 can generate code with correct import paths and that app2's publication succeeds.

**Acceptance Scenarios**:

1. **Given** app1 has published `proto/payments/ledger/v1` to the canonical repo, **When** app2 runs `apx add proto/payments/ledger/v1@v1.0.0` and `apx gen go`, **Then** app2's internal/gen/go contains the overlay with canonical import paths

2. **Given** app2 has added app1's schema as a dependency, **When** app2 defines its own schema `proto/users/profile/v1` that imports `proto/payments/ledger/v1`, **Then** the schema compilation succeeds and resolves imports correctly

3. **Given** app2 has both consumed app1's schema and defined its own schema, **When** app2 runs `apx publish --module-path=internal/apis/proto/users/profile`, **Then** a PR is created in the canonical repo for app2's schema, independent of app1's publication

4. **Given** both app1 and app2 have published to the canonical repo, **When** a third app (app3) runs `apx search`, **Then** the catalog shows both payment and user APIs as available dependencies

---

### User Story 3 - Breaking Change Detection Across Repositories (Priority: P3)

As a developer, I need to validate that breaking change detection works when schemas evolve across app and canonical repositories, so that I can prevent incompatible API changes from being published.

**Why this priority**: Breaking change detection is critical for API governance but less urgent than the basic publishing workflow. This can be validated after P1 and P2 are working.

**Independent Test**: Can be tested by publishing v1.0.0 of a schema, modifying it in a breaking way, attempting to publish v1.1.0, and validating that `apx breaking` detects the violation and prevents publication (or requires --force flag).

**Acceptance Scenarios**:

1. **Given** a schema published as v1.0.0 in the canonical repo, **When** the app repo modifies the schema by removing a field and runs `apx breaking`, **Then** the command fails with a clear error message indicating the breaking change

2. **Given** a schema with a detected breaking change, **When** the developer runs `apx publish` without --force, **Then** the publish command fails and suggests reviewing breaking changes first

3. **Given** a schema with a non-breaking change (added optional field), **When** the developer runs `apx breaking`, **Then** the command succeeds with no errors

4. **Given** a schema published as v1.x, **When** the developer creates a v2 directory with breaking changes and runs `apx publish`, **Then** the publication succeeds because it's a new major version

---

### User Story 4 - Git History and Authorship Preservation (Priority: P2)

As a developer, I need to verify that git subtree publishing preserves commit history and authorship attribution, so that contributors receive proper credit and we maintain audit trails for API evolution.

**Why this priority**: History preservation is a documented requirement and differentiates APX from simple file-copy approaches. Critical for compliance and contributor recognition.

**Independent Test**: Can be tested by creating multiple commits in an app repository with different authors, publishing via `apx publish`, and verifying that the canonical repo PR shows all commits with original timestamps and authors.

**Acceptance Scenarios**:

1. **Given** an app repository with 5 commits from 3 different authors to a schema module, **When** `apx publish` creates a PR to the canonical repo, **Then** all 5 commits appear in the PR with original commit messages, authors, and timestamps

2. **Given** a published schema with preserved history, **When** reviewing the canonical repository's git log for that module path, **Then** the complete evolution of the schema is visible with proper attribution

3. **Given** an app repository with commits that modify both schema files and non-schema files, **When** `apx publish` extracts the schema subtree, **Then** only commits touching the schema directory are included in the canonical repo PR

---

### Edge Cases

- What happens when a Gitea instance is unreachable during test execution? (Should fail fast with clear error, retry logic for transient failures)
- How does the system handle PR creation when a PR already exists for the same module? (Should update existing PR or create new one based on policy)
- What happens when git subtree split fails due to corrupted git history? (Should report error with actionable troubleshooting steps)
- How does the system handle schema dependencies that form circular references (app1 depends on app2, app2 depends on app1)? (Should detect cycle and prevent with clear error)
- What happens when a schema is published with a tag that already exists in the canonical repo? (Should fail with conflict error unless --force flag is used)
- How does overlay management handle when internal/gen/go is manually deleted? (apx sync should detect and regenerate)
- What happens when multiple app repositories publish to the same canonical module path simultaneously? (PR creation should handle conflicts gracefully)
- How does the system handle when canonical repo has CODEOWNERS restrictions that block automated PR creation? (Should validate permissions before attempting publish)

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Test suite MUST provision an isolated Gitea instance per test run to simulate GitHub hosting
- **FR-002**: Test suite MUST support both kind (Kubernetes in Docker) and k3d (lightweight k3s) as container orchestration backends
- **FR-003**: Test suite MUST bootstrap a canonical repository using `apx init canonical` and validate the resulting structure
- **FR-004**: Test suite MUST create at least two application repositories to simulate multi-team scenarios
- **FR-005**: Test suite MUST execute the complete publishing workflow (`apx init app`, schema creation, `apx publish`) for app1
- **FR-006**: Test suite MUST validate that published schemas create PRs in the canonical repository with preserved git history
- **FR-007**: Test suite MUST execute the consumer workflow for app2 (`apx add`, `apx gen go`) consuming app1's published schema
- **FR-008**: Test suite MUST validate that app2 can publish its own schema after consuming app1's schema
- **FR-009**: Test suite MUST verify that generated code in app2 uses canonical import paths that resolve via go.work overlays
- **FR-010**: Test suite MUST clean up all containers, volumes, and test repositories after execution (both success and failure)
- **FR-011**: Test suite MUST validate git commit authorship and timestamps are preserved in canonical repo PRs
- **FR-012**: Test suite MUST test Protocol Buffer schemas as the primary format (other formats optional for this scope)
- **FR-013**: Test suite MUST validate that `apx breaking` detects breaking changes when comparing against published versions
- **FR-014**: Test suite MUST verify that apx.lock files correctly pin dependency versions
- **FR-015**: Test suite MUST validate that `apx sync` correctly updates go.work with active overlays
- **FR-016**: Test suite MUST support running tests in CI environments (GitHub Actions, GitLab CI) without requiring privileged Docker access
- **FR-017**: Test suite MUST provide clear test output showing which phase failed (Gitea startup, repo creation, apx command execution, validation)
- **FR-018**: Test suite MUST validate that published tags follow the correct format (proto/payments/ledger/v1/v1.0.0 in app repo, proto/payments/ledger/v1.0.0 in canonical repo)

### Key Entities

- **Test Environment**: Represents the isolated testing context including Gitea instance, container cluster (kind/k3d), and ephemeral git repositories
- **Canonical Repository**: The central API repository created via `apx init canonical`, contains published schemas from multiple app repositories
- **App Repository 1 (Payment Service)**: First application repository that defines and publishes a payment ledger API schema
- **App Repository 2 (User Service)**: Second application repository that consumes the payment API and publishes its own user profile API schema
- **Schema Module**: A versioned API schema located at a specific path (e.g., proto/payments/ledger/v1), tracked in both app and canonical repositories
- **Pull Request**: Git hosting mechanism for proposing schema publications to the canonical repository, contains subtree-split commits with preserved history
- **Overlay**: Local development artifact in app repository's internal/gen/go directory that maps canonical import paths to locally generated code
- **Test Assertions**: Validation checks that verify expected outcomes at each workflow stage (repository structure, file contents, git history, PR state)

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Test suite executes the complete E2E workflow (Gitea provisioning, 2 app repos, publish, consume, cleanup) in under 5 minutes
- **SC-002**: Test suite detects at least 95% of regressions that would break documented workflows in /docs/getting-started/quickstart.md
- **SC-003**: Test suite runs successfully in CI environments (GitHub Actions) with 100% pass rate on main branch
- **SC-004**: Test failures provide actionable error messages identifying which specific workflow step failed and why (e.g., "PR creation failed: Gitea returned 403, check CODEOWNERS permissions")
- **SC-005**: Test suite validates all mandatory success criteria from documented workflows (repository structure, git history preservation, canonical import paths, overlay resolution)
- **SC-006**: Test cleanup leaves zero orphaned containers or volumes after execution, even when tests fail
- **SC-007**: Test suite can run 10 times in sequence without flakiness (100% pass rate when code is correct)
- **SC-008**: Developers can run the full test suite locally with a single command (e.g., `make test-e2e`) without manual setup
- **SC-009**: Test suite validates cross-platform compatibility by running on both Linux and macOS runners in CI
- **SC-010**: Test coverage includes validation of at least 3 edge cases (concurrent publishes, existing PRs, breaking changes)
