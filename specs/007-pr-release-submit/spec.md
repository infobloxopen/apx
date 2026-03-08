# Feature Specification: PR-First Canonical Release Submission

**Feature Branch**: `007-pr-release-submit`  
**Created**: 2026-03-08  
**Status**: Draft  
**Input**: User description: "Implement PR-first canonical release submission for APX so release promotion into the canonical APIs repository is safe, reviewable, and consistent."

## User Scenarios & Testing

### User Story 1 — Submit a Prepared Release as a Pull Request (Priority: P1)

A developer has run `apx release prepare` and has a valid `.apx-release.yaml` manifest in the `prepared` state. They want to submit the release to the canonical repository for human review rather than pushing directly to the main branch.

**Why this priority**: This is the core value proposition. Without it, the only submit path is a direct subtree push to `main`, which bypasses review and makes mistakes hard to catch.

**Independent Test**: Can be fully tested by running `apx release submit --create-pr` against a prepared manifest and verifying that a PR is opened on the canonical repository with the correct files, branch name, and metadata recorded in the manifest.

**Acceptance Scenarios**:

1. **Given** a valid `.apx-release.yaml` in `prepared` state and `gh` is authenticated, **When** the developer runs `apx release submit --create-pr`, **Then** a release branch is created on the canonical repository, a PR is opened against the default branch, the manifest transitions to `canonical-pr-open`, and PR metadata (URL, number, branch name) is recorded in the manifest.
2. **Given** no `.apx-release.yaml` exists, **When** the developer runs `apx release submit --create-pr`, **Then** the command exits with an error instructing the user to run `apx release prepare` first.
3. **Given** a manifest in `failed` state, **When** the developer runs `apx release submit --create-pr`, **Then** the command exits with an error showing the failure reason and a hint to re-run `apx release prepare`.
4. **Given** a manifest in `package-published` state, **When** the developer runs `apx release submit --create-pr`, **Then** the command reports success ("already published") and exits without creating a duplicate PR.

---

### User Story 2 — Dry-Run Preview of PR Submission (Priority: P1)

A developer wants to inspect what a PR submission would look like before actually creating the branch and PR.

**Why this priority**: Equally critical to the core submit flow — gives developers confidence in what will be pushed and prevents accidental submissions.

**Independent Test**: Can be tested by running `apx release submit --create-pr --dry-run` and verifying it displays the release snapshot, target branch name, and diff without creating any branches or PRs.

**Acceptance Scenarios**:

1. **Given** a valid manifest in `prepared` state, **When** the developer runs `apx release submit --create-pr --dry-run`, **Then** the command displays the release identity, version, canonical path, computed branch name, the files that would be included, and a summary of changes — without creating any branches, pushing, or opening a PR.
2. **Given** a manifest with a `go_module` set and no existing `go.mod`, **When** a dry-run is performed, **Then** the output indicates that a `go.mod` would be generated and shows its expected content.

---

### User Story 3 — Retry a Failed or Interrupted PR Submission (Priority: P1)

A developer ran `apx release submit --create-pr` but the push or PR creation failed partway through (network error, authentication timeout, etc.). They want to re-run the command without creating duplicate branches or corrupting state.

**Why this priority**: Network failures during push or PR creation are common. Without safe retry, developers must manually clean up branches and reset manifest state.

**Independent Test**: Can be tested by simulating a failure after branch push but before PR creation, then re-running the submit command and verifying it detects the existing branch and creates the PR without duplicating work.

**Acceptance Scenarios**:

1. **Given** the release branch already exists on the canonical repository from a prior interrupted run, **When** the developer re-runs `apx release submit --create-pr`, **Then** the command detects the existing branch, force-pushes the current snapshot to it, and creates the PR (or detects that the PR already exists and reports its URL).
2. **Given** a manifest in `canonical-pr-open` state with a recorded PR URL, **When** the developer re-runs `apx release submit --create-pr`, **Then** the command reports the existing PR details and exits successfully without creating a duplicate PR.
3. **Given** a release branch and PR already exist but the local manifest is still in `prepared` state (e.g., the manifest write was interrupted), **When** the developer re-runs the command, **Then** the command detects the existing PR, updates the manifest to `canonical-pr-open` with the correct PR metadata, and reports success.

---

### User Story 4 — Direct Submit Without PR (Priority: P2)

A developer in a small team or automated pipeline wants to submit directly (subtree push to main) without a PR, preserving the existing behavior.

**Why this priority**: The existing direct-push path must continue working for users who do not need PR review. It is lower priority because this flow already works today.

**Independent Test**: Can be tested by running `apx release submit` (without `--create-pr`) and verifying the subtree push to main works as it does today.

**Acceptance Scenarios**:

1. **Given** a valid manifest in `prepared` state, **When** the developer runs `apx release submit` (no `--create-pr`), **Then** the module is published via subtree push to the canonical repository's main branch, exactly as the current behavior.

---

### User Story 5 — Submit with PR in CI Pipelines (Priority: P2)

A CI pipeline (GitHub Actions, GitLab CI) automates the release and wants the submit step to open a PR for gated merge, with CI provenance recorded.

**Why this priority**: CI integration is the primary production use case for the release pipeline, but it depends on the core PR submission working first.

**Independent Test**: Can be tested by running the submit command in a CI-like environment and verifying that the PR body includes CI provenance links and the manifest records the CI context.

**Acceptance Scenarios**:

1. **Given** a prepared manifest and the command is run inside GitHub Actions, **When** `apx release submit --create-pr` runs, **Then** the PR body includes the CI run URL and the manifest records the CI system name and run ID.
2. **Given** a prepared manifest and the command is run outside CI, **When** `apx release submit --create-pr` runs, **Then** the PR body omits CI provenance and the submit still succeeds.

---

### Edge Cases

- What happens when the `gh` CLI is not installed or not authenticated? The command exits with a clear error message and a hint to run `gh auth login`.
- What happens when the canonical repository URL cannot be parsed into an `owner/repo` pair? The command exits with an error identifying the malformed URL.
- What happens when the developer lacks push access to the canonical repository? The git push fails and the command records the failure in the manifest with a hint about repository permissions.
- What happens when the PR target branch (e.g., `main`) does not exist on the canonical repo? The command exits with an error indicating the missing base branch.
- What happens when a PR already exists for the same branch but with different content? The command force-pushes the updated content to the branch; the existing PR automatically reflects the new changes.
- What happens when the canonical repo is not on GitHub (e.g., GitLab)? The command exits with a clear error stating that PR-based submit currently requires GitHub and the `gh` CLI.

## Requirements

### Functional Requirements

- **FR-001**: System MUST support `--create-pr` flag on `apx release submit` that creates a pull request on the canonical repository instead of pushing directly to the main branch.
- **FR-002**: System MUST read the release manifest (`.apx-release.yaml`) as the sole source of truth for release identity, version, lifecycle, source repository, source commit, canonical repository, destination path, validation results, and derived language coordinates.
- **FR-003**: System MUST validate that the manifest is in `prepared` state before beginning PR submission. Manifests in `package-published` state MUST be treated as already complete. Manifests in `failed` state MUST produce an error with the failure reason.
- **FR-004**: System MUST create a deterministic release branch on the canonical repository using the naming convention `apx/release/<api-id-normalized>/<version>` (where slashes in the API ID are replaced with dashes).
- **FR-005**: System MUST materialize the canonical release snapshot from the manifest, including all schema files from the source path, generated `go.mod` if the module path is set and no `go.mod` exists, and any files required for the canonical repository layout.
- **FR-006**: System MUST commit the release snapshot with a structured commit message that includes the API ID, version, and lifecycle.
- **FR-007**: System MUST push the release branch to the canonical repository.
- **FR-008**: System MUST open a pull request against the canonical repository's default branch using the `gh` CLI, with a title following the convention `release: <api-id>@<version>` and a body summarizing the release identity, lifecycle, validation status, and source provenance.
- **FR-009**: System MUST record PR metadata (PR number, PR URL, branch name) back into the release manifest after successful PR creation.
- **FR-010**: System MUST transition the manifest state to `canonical-pr-open` after the PR is successfully created.
- **FR-011**: System MUST support `--dry-run` combined with `--create-pr`, displaying the release snapshot, computed branch name, file list, and diff without creating any branches or PRs.
- **FR-012**: System MUST handle retries gracefully: if the release branch already exists, force-push the current snapshot; if the PR already exists for that branch, report the existing PR details without creating a duplicate.
- **FR-013**: System MUST verify that `gh` is installed and authenticated before attempting PR creation, and exit with a clear error and remediation hint if either check fails.
- **FR-014**: System MUST preserve the existing direct-push behavior when `--create-pr` is not specified.
- **FR-015**: System MUST write the updated manifest to `.apx-release.yaml` after each significant state change so that interrupted runs can be resumed.

### Key Entities

- **Release Manifest**: The `.apx-release.yaml` file that tracks the release through the state machine. Extended with PR metadata fields: PR number, PR URL, and PR branch name.
- **Release Branch**: A deterministic, namespaced branch on the canonical repository (e.g., `apx/release/proto-payments-ledger-v1/v1.0.0`) that carries the release snapshot.
- **Release Snapshot**: The complete set of files that constitute the canonical release — schema files from the source path, generated `go.mod`, and any derived metadata.
- **Pull Request**: A GitHub PR opened against the canonical repository's default branch, representing the release for human review.

## Success Criteria

### Measurable Outcomes

- **SC-001**: A developer can run `apx release submit --create-pr` and receive a reviewable pull request on the canonical repository within the same time it takes the current `apx publish --create-pr` flow (established baseline).
- **SC-002**: The PR branch name, commit message, and PR title are deterministic — running the same release twice produces the same branch name and PR title.
- **SC-003**: A failed or interrupted submission can be retried by re-running the same command without creating duplicate branches or PRs.
- **SC-004**: The dry-run mode shows the complete release snapshot and computed metadata without making any changes to the canonical repository.
- **SC-005**: The release manifest records PR metadata (URL, number, branch) that can be inspected with `apx release inspect` after submission.
- **SC-006**: The existing direct-push submit path (`apx release submit` without `--create-pr`) continues to work identically to its current behavior.
- **SC-007**: Every PR-based release submission enters the `canonical-pr-open` state, providing a clear audit trail from preparation through PR creation.

## Assumptions

- The canonical repository is hosted on GitHub. PR-based submission depends on the `gh` CLI for PR creation. Non-GitHub hosts are out of scope for this feature.
- The `gh` CLI is pre-installed and authenticated on the developer's machine or CI environment. APX does not manage `gh` installation.
- The canonical repository's default branch is the merge target for release PRs. APX does not need to support configurable base branches for release PRs.
- The `apx release finalize` command (run after PR merge in canonical CI) does not need changes — it already handles the `canonical-pr-open` state as a valid starting point.
- The PR branch naming convention (`apx/release/...`) is distinct from the publish branch convention (`apx/publish/...`) to avoid collisions between the two flows.
