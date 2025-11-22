# Data Model: End-to-End Integration Test Suite

**Feature**: 003-e2e-integration-suite  
**Date**: November 22, 2025  
**Purpose**: Define entities, relationships, and state management for E2E test infrastructure

---

## Entity Definitions

### 1. Test Environment

Represents the complete E2E test execution context including k3d cluster and Gitea instance.

**Attributes**:
- `ClusterID` (string): Unique k3d cluster name (e.g., `apx-e2e-20251122-143052`)
- `GiteaURL` (string): Base URL for Gitea instance (e.g., `http://localhost:3000`)
- `AdminToken` (string): Gitea admin API token for repository management
- `GiteaPort` (int): Host port mapped to Gitea service (dynamically allocated)
- `Repositories` (map[string]Repository): Created repositories indexed by name
- `State` (EnvironmentState): Current lifecycle state
- `CreatedAt` (time.Time): Environment creation timestamp
- `CleanupHandlers` ([]func()): Cleanup functions to run on teardown

**Enumeration: EnvironmentState**:
- `Setup`: Environment is being initialized
- `Running`: Cluster and Gitea ready for tests
- `Teardown`: Cleanup in progress
- `Failed`: Setup or test execution failed

**Lifecycle**:
```
Setup → Running → Teardown
  ↓
Failed (from any state)
```

**Validation Rules**:
- ClusterID must be unique per test run (includes timestamp)
- GiteaURL must be reachable before transitioning to Running
- AdminToken must have repo and admin scopes
- Cleanup handlers execute in LIFO order (reverse of registration)

**Relationships**:
- Environment HAS MANY Repositories
- Environment HAS ONE Gitea Instance
- Environment HAS ONE k3d Cluster

---

### 2. Test Repository

Represents a git repository in the test environment (canonical or app repository).

**Attributes**:
- `Name` (string): Repository name (e.g., `api-schemas`, `payment-service`)
- `Type` (RepositoryType): Repository classification
- `Owner` (string): Gitea organization or username (e.g., `testorg`)
- `CloneURL` (string): Git clone URL (e.g., `http://localhost:3000/testorg/api-schemas.git`)
- `LocalPath` (string): Absolute path to cloned repository on test runner
- `Commits` ([]Commit): Git commits for history validation
- `PublishedModules` ([]PublishedModule): Schemas published from this repo
- `Dependencies` ([]Dependency): Schema dependencies (app repos only)
- `CreatedAt` (time.Time): Repository creation timestamp
- `GiteaID` (int64): Gitea internal repository ID

**Enumeration: RepositoryType**:
- `Canonical`: Central API repository (one per environment)
- `App1`: First application repository (publishes payment API)
- `App2`: Second application repository (consumes app1, publishes user API)
- `App3`: Additional application repository (for edge cases)

**Validation Rules**:
- Name must match regex `^[a-z0-9-]+$` (lowercase, alphanumeric, hyphens)
- LocalPath must be absolute and within test temp directory
- Canonical repos have zero dependencies
- App repos may have zero or more dependencies

**Relationships**:
- Repository BELONGS TO Environment
- Repository HAS MANY Commits
- Repository HAS MANY PublishedModules
- Repository HAS MANY Dependencies (app repos)

---

### 3. Commit

Represents a git commit for history validation.

**Attributes**:
- `SHA` (string): Git commit SHA (40 characters)
- `Message` (string): Commit message
- `Author` (string): Author name
- `AuthorEmail` (string): Author email
- `Timestamp` (time.Time): Commit timestamp
- `ParentSHAs` ([]string): Parent commit SHAs

**Validation Rules**:
- SHA must be 40 hexadecimal characters
- Author and AuthorEmail must be non-empty
- Timestamp must be chronologically ordered (no future commits)

**Use Cases**:
- Validate git subtree preserves commit history
- Verify authorship attribution in canonical repo PRs
- Check commit message formatting

---

### 4. Published Module

Represents a schema module published from an app repository to the canonical repository.

**Attributes**:
- `ModulePath` (string): Module path in repository (e.g., `proto/payments/ledger/v1`)
- `Version` (string): Semantic version (e.g., `v1.0.0`)
- `AppTag` (string): Git tag in app repository (e.g., `proto/payments/ledger/v1/v1.0.0`)
- `CanonicalTag` (string): Git tag in canonical repository (e.g., `proto/payments/ledger/v1.0.0`)
- `PRID` (int64): Pull request ID in canonical repository (0 if not yet created)
- `PRState` (string): Pull request state (`open`, `merged`, `closed`)

**Validation Rules**:
- ModulePath must match format: `<format>/<domain>/<api>/<version>`
- Version must follow semver (v[major].[minor].[patch])
- AppTag includes full module path; CanonicalTag omits repo prefix
- PRID must exist in canonical repository if PR was created

**Relationships**:
- PublishedModule BELONGS TO Repository (app repo)
- PublishedModule CREATES PullRequest (in canonical repo)

---

### 5. Dependency

Represents a schema dependency in an app repository's apx.lock file.

**Attributes**:
- `ModulePath` (string): Module path (e.g., `proto/payments/ledger/v1`)
- `Version` (string): Pinned version (e.g., `v1.0.0`)
- `SourceRepo` (string): Canonical repository (e.g., `github.com/testorg/apis`)
- `SourceRef` (string): Git reference in canonical repo (tag or commit SHA)
- `OverlayPath` (string): Local overlay path (e.g., `internal/gen/go/proto/payments/ledger@v1.0.0`)

**Validation Rules**:
- ModulePath must exist in canonical repository
- Version must match a published tag in canonical repository
- OverlayPath must exist after `apx gen go` execution
- SourceRef must be resolvable in canonical repository

**Relationships**:
- Dependency BELONGS TO Repository (app repo)
- Dependency REFERENCES PublishedModule (in canonical repo)

---

### 6. Test Assertion

Represents a validation check performed during test execution.

**Attributes**:
- `Type` (AssertionType): Kind of assertion
- `ExpectedValue` (string): Expected outcome (exact match or regex pattern)
- `ActualValue` (string): Captured value during execution
- `Passed` (bool): Assertion result
- `ErrorContext` (string): Detailed error message for failures (SC-004 requirement)
- `ScenarioFile` (string): Testscript file that originated assertion
- `LineNumber` (int): Line number in testscript file

**Enumeration: AssertionType**:
- `FileExists`: Check file exists at path
- `FileContains`: Check file contains pattern (regex)
- `DirExists`: Check directory exists
- `GitLogContains`: Check git log contains commit/message
- `GitTagExists`: Check git tag exists with name
- `PRCreated`: Check pull request was created
- `PRState`: Check pull request state (open/merged/closed)
- `OverlayExists`: Check go.work overlay path exists
- `CommandSuccess`: Check apx command succeeded
- `CommandFailed`: Check apx command failed with expected error

**Validation Rules**:
- ExpectedValue must be non-empty
- ErrorContext must include actionable fix suggestion for failures
- ScenarioFile must be valid testscript path

**Use Cases**:
- Validate repository structure after `apx init`
- Verify git history preservation in PRs
- Check overlay creation after `apx gen go`
- Validate error messages for edge cases

---

### 7. Gitea API Response Entities

These entities model Gitea API responses used for repository operations and validation.

#### 7.1 Gitea Repository

**Attributes**:
- `ID` (int64): Gitea internal repository ID
- `Name` (string): Repository name
- `Owner` (GiteaUser): Repository owner
- `CloneURL` (string): HTTP clone URL
- `SSHURL` (string): SSH clone URL
- `DefaultBranch` (string): Default branch name (usually `main`)
- `Private` (bool): Repository visibility
- `CreatedAt` (time.Time): Repository creation timestamp

#### 7.2 Gitea Pull Request

**Attributes**:
- `ID` (int64): Pull request ID
- `Number` (int): Pull request number (index)
- `Title` (string): PR title
- `Body` (string): PR description
- `State` (string): `open`, `closed`, or `merged`
- `HeadBranch` (string): Source branch
- `BaseBranch` (string): Target branch
- `Commits` (int): Number of commits in PR
- `Additions` (int): Lines added
- `Deletions` (int): Lines deleted
- `CreatedAt` (time.Time): PR creation timestamp
- `UpdatedAt` (time.Time): Last update timestamp
- `MergedAt` (*time.Time): Merge timestamp (nil if not merged)

**Use Cases**:
- Validate `apx publish` creates PR with correct title/body
- Verify PR contains expected number of commits (git history preservation)
- Check PR file changes match published module

#### 7.3 Gitea Tag

**Attributes**:
- `Name` (string): Tag name
- `CommitSHA` (string): Commit the tag points to
- `Message` (string): Tag annotation message
- `CreatedAt` (time.Time): Tag creation timestamp

**Use Cases**:
- Validate tags exist after `apx publish`
- Verify app repo tag format (`proto/payments/ledger/v1/v1.0.0`)
- Verify canonical repo tag format (`proto/payments/ledger/v1.0.0`)

#### 7.4 Gitea User

**Attributes**:
- `ID` (int64): User ID
- `Username` (string): Username
- `Email` (string): Email address
- `IsAdmin` (bool): Admin privileges

**Use Cases**:
- Admin user for repository creation
- Test users for multi-author commit scenarios

---

## Entity Relationships

```
Environment (1) ──< (∞) Repository
                  │
                  ├──< (∞) Commit
                  ├──< (∞) PublishedModule
                  └──< (∞) Dependency

Repository ──< (∞) TestAssertion

PublishedModule ──> (1) PullRequest (in Gitea)

Dependency ──> (1) PublishedModule (references)
```

---

## State Transitions

### Environment Lifecycle

```
[Initial] 
   ↓
Setup: Create k3d cluster
   ↓
Setup: Deploy Gitea to cluster
   ↓
Setup: Wait for Gitea readiness
   ↓
Setup: Create admin token
   ↓
Running: Ready for tests
   ↓
[Tests execute]
   ↓
Teardown: Delete test repositories
   ↓
Teardown: Stop Gitea
   ↓
Teardown: Delete k3d cluster
   ↓
[Terminal]

Failure can occur at any step, triggering immediate cleanup.
```

### Repository Lifecycle (Canonical)

```
[Not Exists]
   ↓
Create via Gitea API
   ↓
Clone to local path
   ↓
apx init canonical
   ↓
Commit + push to Gitea
   ↓
[Ready for PR acceptance]
```

### Repository Lifecycle (App)

```
[Not Exists]
   ↓
Create via Gitea API
   ↓
Clone to local path
   ↓
apx init app <module-path>
   ↓
Commit schema
   ↓
apx publish (creates PR in canonical)
   ↓
[Schema available for consumption]
```

### Dependency Resolution Lifecycle

```
[App repo initialized]
   ↓
apx add <module>@<version>
   ↓
Updates apx.lock
   ↓
apx gen go
   ↓
Creates overlay in internal/gen/go
   ↓
apx sync
   ↓
Updates go.work
   ↓
[Imports resolve via overlay]
```

---

## Validation Matrix

| Entity | Validation Point | Check |
|--------|------------------|-------|
| Environment | Before Running state | Gitea `/api/v1/version` returns 200 |
| Environment | Before Running state | Admin token can create repository |
| Repository | After creation | Git clone succeeds |
| Repository | After apx init | Expected directory structure exists |
| Commit | After apx publish | All commits present in canonical PR |
| PublishedModule | After apx publish | PR created with correct title |
| PublishedModule | After apx publish | Tags exist in both repos |
| Dependency | After apx add | apx.lock contains correct version |
| Dependency | After apx gen go | Overlay directory exists |
| TestAssertion | After each operation | All assertions pass |

---

## Error Handling

**Principle**: All failures include actionable context (SC-004)

**Error Context Format**:
```
What failed: <specific operation>
Why it failed: <root cause analysis>
How to fix: <specific remediation steps>
Context: <relevant state information>
```

**Example**:
```
What failed: Gitea repository creation for 'payment-service'
Why it failed: HTTP 409 Conflict - repository already exists
How to fix: Delete existing repository or use unique name
Context: Repository 'testorg/payment-service' was created in previous test run
         Run 'make clean-e2e' to remove orphaned test repositories
```

---

## Performance Considerations

**Environment Setup**: ~40 seconds
- k3d cluster creation: 25s
- Gitea deployment + readiness: 15s

**Repository Operations**: ~2-5 seconds each
- Create via API: 1s
- Clone: 1-2s
- apx init: 1-2s

**Test Scenario**: ~10-20 seconds
- Depends on number of operations
- Most time in git operations

**Suite Execution**: ~3-4 minutes (10 scenarios)
- Amortized environment setup
- Sequential scenario execution

---

## Next Steps

1. ✅ Phase 1: Data model complete
2. ⏳ Generate `contracts/` (Gitea API spec, test repo contracts)
3. ⏳ Generate `quickstart.md` (developer guide)
4. ⏳ Update agent context with E2E knowledge
