# Feature Specification: Package Installer Support

**Feature Branch**: `004-package-installers`  
**Created**: 2026-03-07  
**Status**: Draft  
**Input**: User description: "add support for installing apx over popular package installers — popular installers like brew, uv and some on windows let you install cli package"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Install APX via Homebrew on macOS/Linux (Priority: P1)

A developer on macOS or Linux wants to install APX using Homebrew, the most widely-used package manager for macOS and common on Linux. They run a single `brew install` command and get a working APX binary with shell completions.

**Why this priority**: Homebrew is the dominant package manager for macOS (APX's primary audience). The GoReleaser config already declares a `brews` section targeting `infobloxopen/homebrew-tap`, but the tap repository does not exist yet. This is the highest-impact, lowest-effort installer to ship.

**Independent Test**: Can be fully tested by running `brew install infobloxopen/tap/apx && apx --version` on a fresh macOS or Linux machine and verifying the binary, shell completions (bash/zsh/fish), and `brew upgrade` work.

**Acceptance Scenarios**:

1. **Given** a macOS or Linux machine with Homebrew installed, **When** the user runs `brew install infobloxopen/tap/apx`, **Then** APX is installed, `apx --version` prints the current version, and shell completions are available for bash, zsh, and fish.
2. **Given** APX was previously installed via Homebrew, **When** a new version is released and the user runs `brew upgrade apx`, **Then** APX is upgraded to the latest version.
3. **Given** APX is installed via Homebrew, **When** the user runs `brew uninstall apx`, **Then** APX and its completions are fully removed.

---

### User Story 2 - Install APX via Scoop on Windows (Priority: P2)

A developer on Windows wants to install APX using Scoop, a popular command-line installer for Windows. They add the APX bucket and run `scoop install apx` to get a working binary with PATH configured automatically.

**Why this priority**: Windows developers need a native package management story. Scoop is the easiest Windows package manager to support — it only requires a JSON manifest in a GitHub "bucket" repository with no signing or review process, and GoReleaser can auto-generate it.

**Independent Test**: Can be fully tested by running `scoop bucket add infobloxopen https://github.com/infobloxopen/scoop-bucket && scoop install apx && apx --version` on a Windows machine with Scoop installed.

**Acceptance Scenarios**:

1. **Given** a Windows machine with Scoop installed, **When** the user runs `scoop bucket add infobloxopen https://github.com/infobloxopen/scoop-bucket && scoop install apx`, **Then** APX is installed and `apx --version` prints the current version.
2. **Given** APX was previously installed via Scoop, **When** a new version is released and the user runs `scoop update apx`, **Then** APX is upgraded to the latest version.
3. **Given** APX is installed via Scoop, **When** the user runs `scoop uninstall apx`, **Then** APX is fully removed.

---

### User Story 3 - Install APX via shell one-liner (Priority: P3)

A developer on any Unix-like system (macOS/Linux) wants to install APX with a single curl command without needing a package manager. This is common in CI environments, Docker containers, and quick-start tutorials.

**Why this priority**: A curl-based installer provides a universal fallback that works everywhere without any prerequisites. It's also the standard approach for CI pipelines that don't want to install Homebrew first.

**Independent Test**: Can be fully tested by running the install script on a clean Ubuntu container or macOS shell and verifying `apx --version` works.

**Acceptance Scenarios**:

1. **Given** a macOS or Linux machine with curl installed, **When** the user runs `curl -sSL https://raw.githubusercontent.com/infobloxopen/apx/main/scripts/install.sh | bash`, **Then** APX is installed to a discoverable location and `apx --version` works.
2. **Given** the install script is run, **When** the user specifies a version via environment variable (e.g., `VERSION=1.2.3`), **Then** that specific version is installed instead of the latest.
3. **Given** the install script is run in a CI environment, **When** there is no interactive terminal, **Then** the script completes without prompts and exits with code 0 on success.

---

### User Story 4 - Automated release publishing to all package managers (Priority: P1)

A maintainer tags a new release. The CI pipeline builds binaries, creates a GitHub Release, and automatically publishes updated manifests/formulas to all configured package manager repositories (Homebrew tap, Scoop bucket, winget manifest PR).

**Why this priority**: Without automation, each release would require manual updates to 3+ package manager repositories. This is error-prone and unsustainable. GoReleaser already handles most of this — the work is creating the target repositories and wiring up the CI.

**Independent Test**: Can be tested by creating a pre-release tag, verifying the release workflow creates a GitHub Release with binaries, updates the Homebrew tap formula, and updates the Scoop bucket manifest.

**Acceptance Scenarios**:

1. **Given** a maintainer pushes a semantic version tag (e.g., `v1.2.3`), **When** the release CI workflow runs, **Then** GitHub Releases are created with binaries for all platforms (linux/darwin amd64+arm64, windows amd64), checksums, and changelogs.
2. **Given** the release workflow completes, **When** the Homebrew formula is generated, **Then** the `infobloxopen/homebrew-tap` repository is automatically updated with the new version's formula.
3. **Given** the release workflow completes, **When** the Scoop manifest is generated, **Then** the `infobloxopen/scoop-bucket` repository is automatically updated with the new version's manifest.

---

### Edge Cases

- What happens when a user has APX installed via multiple methods (e.g., both Homebrew and `go install`)? — The package manager's binary should take precedence based on PATH ordering. Documentation should warn against installing via multiple methods.
- What happens when the Homebrew tap repository doesn't exist yet during a release? — The release workflow should fail clearly with instructions to create the repository first.
- What happens when a Windows user doesn't have Scoop? — Documentation should guide them to the GitHub Releases download page as a fallback.
- What happens when the install script is run behind a corporate proxy? — The script should respect `https_proxy` / `HTTPS_PROXY` environment variables and provide clear error messages on network failures.
- What happens when a release tag is pushed but the release workflow fails partway? — Each package manager update should be independent so partial failures don't block all updates. Failed steps should be retryable.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The project MUST have a Homebrew tap repository (`infobloxopen/homebrew-tap`) that contains auto-generated formulas for APX.
- **FR-002**: The project MUST have a Scoop bucket repository (`infobloxopen/scoop-bucket`) that contains auto-generated manifests for APX.
- **FR-003**: GoReleaser configuration MUST auto-publish Homebrew formulas and Scoop manifests on tagged releases.
- **FR-004**: The Homebrew formula MUST install shell completions for bash, zsh, and fish.
- **FR-005**: The project MUST provide a standalone shell installer script (`scripts/install.sh`) that downloads and installs the correct binary for the user's platform from GitHub Releases.
- **FR-006**: The install script MUST detect the user's OS (Linux, macOS) and architecture (amd64, arm64) and download the appropriate binary.
- **FR-007**: The install script MUST support installing a specific version via a `VERSION` environment variable, defaulting to the latest release.
- **FR-008**: The install script MUST work non-interactively in CI environments without prompts.
- **FR-009**: The install script MUST verify downloaded binary integrity using SHA256 checksums from the release.
- **FR-010**: Documentation (README, installation guide) MUST list all supported installation methods with copy-pasteable commands.
- **FR-011**: A GitHub Actions release workflow MUST trigger on semantic version tags and orchestrate the full release pipeline (build, publish, package manager updates).
- **FR-012**: The release workflow MUST use a dedicated GitHub token or deploy key with write access to the tap/bucket repositories.

### Key Entities

- **Homebrew Tap**: A separate GitHub repository (`infobloxopen/homebrew-tap`) containing Ruby formula files that Homebrew uses to install APX. Auto-updated by GoReleaser.
- **Scoop Bucket**: A separate GitHub repository (`infobloxopen/scoop-bucket`) containing JSON manifest files that Scoop uses to install APX. Auto-updated by GoReleaser.
- **Install Script**: A standalone bash script hosted in the APX repository that downloads pre-built binaries from GitHub Releases. Works without any package manager.
- **GoReleaser Config**: The `.goreleaser.yml` file that defines build targets, archive formats, and package manager publishing rules. Already partially configured.
- **Release Workflow**: A GitHub Actions workflow that triggers on version tags and runs GoReleaser to build, publish, and distribute to all package managers.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can install APX via `brew install infobloxopen/tap/apx` on macOS and Linux and have a working binary within 60 seconds.
- **SC-002**: Users can install APX via `scoop install apx` on Windows (after adding the bucket) and have a working binary within 60 seconds.
- **SC-003**: Users can install APX via a single curl one-liner on any Unix-like system and have a working binary within 30 seconds.
- **SC-004**: When a new version tag is pushed, all package manager repositories are automatically updated within 10 minutes with no manual intervention.
- **SC-005**: The install script correctly detects and installs the right binary for at least 4 platform combinations: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64.
- **SC-006**: Installation documentation covers all supported methods and is reachable from both the README and the docs site.
- **SC-007**: `brew upgrade apx`, `scoop update apx`, and re-running the install script all successfully upgrade to the latest version.

## Assumptions

- The `infobloxopen` GitHub organization has permission to create new repositories (`homebrew-tap`, `scoop-bucket`).
- GoReleaser is already integrated and configured (confirmed: `.goreleaser.yml` exists with `brews` and `nfpms` sections).
- GitHub Actions has access to a token with write permission to the tap/bucket repos (typically `HOMEBREW_TAP_TOKEN` or similar secret).
- The existing `specs/002-brew-install` directory was an earlier placeholder for this work and is now superseded by this broader specification.
- APX targets amd64 and arm64 on Linux/macOS, and amd64 on Windows (arm64 Windows excluded per `.goreleaser.yml` config).
- winget is explicitly excluded — it requires submitting to `microsoft/winget-pkgs`, a third-party central registry. Windows users are served by Scoop and direct GitHub Releases download.
- The user mentioned "uv" as a package installer — since APX is a Go binary (not a Python package), `uv` is not applicable. The equivalent cross-platform CLI installation needs are covered by the shell installer script (Story 3) and the platform-specific package managers.
