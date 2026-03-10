# APX — API Schema Management


<div class="grid cards" markdown>

-   :material-rocket-launch: **Getting Started**

    ---

    Install APX and learn the core concepts for API schema management.

    [:octicons-arrow-right-24: Get started](getting-started/index.md)

-   :material-bank: **Canonical Repo**

    ---

    Set up the organization-wide source of truth for all API schemas.

    [:octicons-arrow-right-24: Learn more](canonical-repo/index.md)

-   :material-application-brackets: **App Repos**

    ---

    Author and release schemas from your application repositories.

    [:octicons-arrow-right-24: Learn more](app-repos/index.md)

-   :material-package-variant: **Dependencies**

    ---

    Discover, add, and update API dependencies with versioning.

    [:octicons-arrow-right-24: Learn more](dependencies/index.md)

-   :material-tag-arrow-up: **Releasing**

    ---

    Tag-based release workflow from app repos to canonical repo.

    [:octicons-arrow-right-24: Learn more](publishing/index.md)

-   :material-github: **GitHub**

    ---

    View source code, report issues, and contribute to APX.

    [:octicons-arrow-right-24: View source](https://github.com/infobloxopen/apx)

</div>

## What is APX?

**APX** is a tiny CLI and repo pattern for releasing, discovering, and consuming organization-wide API schemas. **Primary: Protobuf**. Also: **OpenAPI**, **Avro**, **JSON Schema**, **Parquet**. No long-running service. Canonical distribution via a single GitHub repo and Go modules, with CI-only releases.

### Key Ideas

- **Canonical source of truth**: `github.com/<org>/apis` (one repo, many submodules)
- **Custom import roots**: Optional `import_root` decouples public Go import paths from Git hosting — use vanity domains like `go.acme.dev/apis` while hosting at `github.com/acme/apis`
- **App teams tag releases** in their app repo; `apx release prepare` + `apx release submit` opens a PR to the canonical repo (files are copied to a feature branch and submitted for review)
- **Only CI** in the canonical repo creates tags; Go modules are available automatically via the tag, while other language packages (Maven, wheels, OCI) require CI plugins teams configure separately
- **Protobuf is primary**; OpenAPI/Avro/JSONSchema/Parquet supported at varying maturity levels (see [Format Maturity Matrix](testing/format-maturity.md))

### Architecture Overview

<div class="grid cards" markdown>
-   **App Repos**

    ---

    - Teams author schemas locally
    - Tag releases in app repo
    - `apx release prepare` + `submit` opens PRs
    - CI validates before release

-   **Canonical Repo**

    ---

    - Single source of truth
    - Versioned API modules
    - Protected branches & tags
    - CI creates releases

</div>

### Quick Start

```bash
# Install APX
brew install --cask <org>/tap/apx
# or download from GitHub Releases

# Verify installation
apx --version

# Bootstrap canonical repo
apx init canonical github.com/<org>/apis

# Bootstrap app repo for authoring
apx init app internal/apis/proto/payments/ledger

# Add dependencies
apx add proto/payments/ledger/v1@v1.2.3

# Generate code stubs
apx gen go
```

### Supported Schema Formats

<div class="grid cards" markdown>
-   **Protocol Buffers (Primary)**

    ---

    - Buf integration and workspace
    - gRPC service definitions
    - Semantic Import Versioning
    - Breaking change detection

-   **OpenAPI**

    ---

    - OpenAPI 3.0+ specifications
    - oasdiff breaking checks
    - Spectral linting
    - REST API definitions

-   **Apache Avro**

    ---

    - Schema compatibility checks
    - BACKWARD compatibility default
    - Field defaults and aliases
    - Data serialization schemas

-   **JSON Schema & Parquet**

    ---

    - JSON Schema validation
    - Parquet additive nullable columns
    - Custom compatibility rules
    - Schema evolution support

</div>

### Release Guardrails

**Automated Checks** (CI-enforced):
- Format-specific linting and breaking change detection
- Policy enforcement (ban service/ORM annotations)
- SemVer guidance with `apx semver suggest`
- Protected tag patterns (only CI can create tags)

**Human Gates**:
- `CODEOWNERS` per API path
- Time-boxed waivers for exceptions

---

## Platform Requirements (v1.0 Scope)

APX 1.0 will support **GitHub** and **Gitea** as the Git hosting platform for the canonical repository. All core features — tag-based releases, PR-based submissions, branch/tag protection, and the GitHub App bot identity — are designed and tested against these two platforms.

**Supported hosting platforms:**

| Platform | Support level |
|----------|---------------|
| GitHub (github.com) | ✅ Primary — fully supported |
| GitHub Enterprise Server | ✅ Supported |
| Gitea (self-hosted) | ✅ Supported (used for E2E test suite) |
| GitLab | ❌ Non-goal for v1.0 |
| Bitbucket | ❌ Non-goal for v1.0 |
| Azure DevOps | ❌ Non-goal for v1.0 |
| Other Git hosts | ❌ Non-goal for v1.0 |

**CI runtime** (where `apx release` commands run) is separate from hosting: GitHub Actions is primary, and `apx release submit` also detects GitLab CI and Jenkins environments for setting CI metadata. The canonical repository itself must live on GitHub or Gitea.

Supporting additional hosting platforms is out of scope for v1.0 and will be evaluated for future releases based on demand.

---

*APX standardizes how teams author, release, and consume versioned APIs across your organization.*