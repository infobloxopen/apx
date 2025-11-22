# Feature Specification: Docs-Aligned APX Experience

**Feature Branch**: `001-align-docs-experience`  
**Created**: 2025-11-21  
**Status**: Draft  
**Input**: User description: "Use the /docs directory as the target state application experience. We want to build an API management tool that is simple to use, highly portable and able to be self hosted with GitHub."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Bootstrap Canonical API Workspace (Priority: P1)

A platform administrator sets up the canonical API repository and guardrails exactly as described in `/docs/getting-started/quickstart.md`, ensuring the CLI experience matches documented prompts, generated files, and governance defaults.

**Why this priority**: The canonical repo is the backbone of the API management tool; without it the remaining flows cannot operate.

**Independent Test**: Run `apx init canonical --org=<org>` (interactive and flag-driven) in a clean GitHub Enterprise Server (GHES) project and verify the resulting structure, policies, and protection guidance match the documented quickstart artifacts.

**Acceptance Scenarios**:

1. **Given** a new empty repository on GitHub Enterprise Server 3.x, **When** the administrator runs `apx init canonical --org=<org>` using defaults, **Then** the tool scaffolds the directory tree, policy files, and catalog templates exactly as depicted in the quickstart documentation.
2. **Given** an organization that operates in an offline or air-gapped environment, **When** the administrator provides cached tool bundles and runs the same command, **Then** the scaffolding completes without reaching external networks and still matches the documented structure.

---

### User Story 2 - Author & Publish an API Schema (Priority: P1)

An API producer team initializes an application repository, authors a schema, validates it, and publishes it to the canonical repo using the documented lint, breaking, version, and publish workflow.

**Why this priority**: Publishing schemas is the core value proposition; producers must be able to progress from local authoring to canonical publication without deviation from docs.

**Independent Test**: Follow the quickstart flow end-to-end with the sample payments ledger API, invoking `apx init app`, `apx lint`, `apx breaking`, tagging, and `apx publish`, verifying each CLI output and GitHub interaction mirrors `/docs/getting-started/quickstart.md` and `/docs/publishing/index.md`.

**Acceptance Scenarios**:

1. **Given** an initialized app repository created with `apx init app internal/apis/proto/payments/ledger`, **When** the producer validates and publishes a schema, **Then** linting, breaking checks, tagging, and git subtree publishing succeed and produce the canonical PR artifacts described in the publishing guide.
2. **Given** an app repository that targets OpenAPI instead of Protocol Buffers, **When** the producer runs the same workflow, **Then** the correct format-specific validators and breaking checkers execute while keeping the CLI messaging consistent with documentation examples.

---

### User Story 3 - Consume a Published API with Canonical Imports (Priority: P2)

An application developer discovers an API via the catalog, generates client code, overlays canonical imports locally, and later transitions to the published module without editing application imports.

**Why this priority**: Simple consumption validates the portability promise; consumers must experience frictionless onboarding that matches the documented discovery and synchronization steps.

**Independent Test**: Using the published ledger API, execute `apx search`, `apx add`, `apx gen <lang>`, `apx sync`, and `apx unlink` while confirming generated overlays, go.work entries, and CLI messaging match `/docs/getting-started/quickstart.md`.

**Overlay Mechanism**: Go workspace overlays enable applications to use canonical import paths (e.g., `github.com/org/apis-go/proto/payments/ledger/v1`) during local development, transparently resolving them to locally generated code in `internal/gen/`. When ready, developers remove the overlay and fetch the published module - the same import paths now resolve to the published package without any code changes. See [overlays.md](./overlays.md) for complete design documentation.

**Acceptance Scenarios**:

1. **Given** a service repository with no prior APX configuration, **When** the developer uses `apx search` and `apx add` for `proto/payments/ledger/v1`, **Then** the catalog lookup results, prompts, and lockfile updates match the documentation.
2. **Given** local overlays generated for Go and Python, **When** the developer runs `apx unlink` followed by `go get` or equivalent package restores, **Then** the application compiles without modifying import statements, demonstrating portability as described in the quickstart.

---

### Edge Cases

- How does the workflow proceed when the canonical repository already contains partially initialized structures (e.g., existing buf configuration or CODEOWNERS entries)?
- What happens when an organization relies on GitHub Enterprise Server without Actions enabled and must run publishing jobs in self-hosted runners or alternative CI systems?
- How does the system handle schema formats that mix Protocol Buffers and OpenAPI within the same app repository when running lint and breaking commands?
- What occurs if the tool detects divergence between CLI outputs and the examples captured in `/docs/` (e.g., additional flags or reordered text)?

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The tool MUST scaffold the canonical API repository structure, policy files, and protection guidance exactly as defined in `/docs/getting-started/quickstart.md` and `/docs/canonical-repo/structure.md`.
- **FR-002**: The tool MUST scaffold application repositories with the documented internal schema layout, configuration files (`apx.yaml`, `buf.work.yaml`), and ignore rules to ensure portability.
- **FR-003**: The CLI MUST support both interactive survey-driven flows and fully scripted flag-driven flows for every documented command (`init`, `lint`, `breaking`, `gen`, `publish`, `sync`, `search`, `add`, `update`, `upgrade`, `unlink`).
- **FR-004**: Validation commands (`apx lint`, `apx breaking`) MUST execute format-specific analyzers for Protocol Buffers, OpenAPI, Avro, JSON Schema, and Parquet while presenting user-facing messaging consistent with `/docs/dependencies/index.md` and `/docs/troubleshooting/index.md`.
- **FR-005**: Code generation and synchronization (`apx gen`, `apx sync`) MUST emit canonical import paths, manage go.work overlays, and keep generated artifacts out of version control as described in `/docs/getting-started/quickstart.md`.
- **FR-006**: Publishing (`apx publish`) MUST create git subtree-based pull requests, preserve commit history, and produce tag naming that matches `/docs/publishing/index.md`, including support for GitHub Enterprise Server and mirrored GitHub deployments.
- **FR-007**: Dependency management commands (`apx add`, `apx update`, `apx upgrade`, `apx unlink`) MUST update lockfiles, regenerate overlays, and preserve import stability exactly as depicted in the quickstart examples.
- **FR-008**: The tool MUST operate in self-hosted GitHub environments, including GitHub Enterprise Server and air-gapped deployments, by allowing offline tool bundles (`apx fetch --ci`) and configurable endpoints.
- **FR-009**: The system MUST continuously verify that CLI outputs, prompts, and generated artifacts remain synchronized with `/docs/` content, failing validation if drift is detected so documentation stays the authoritative source.

### Key Entities *(include if feature involves data)*

- **Canonical API Repository**: The GitHub repository that houses organization-wide APIs, policy files, catalogs, and version directories; primary attributes include org identifier, protected branches, tag policies, and catalog metadata.
- **Application Repository**: The producer-owned workspace containing schemas, generated overlays, configuration files, and publishing automation; tracks associated canonical targets and toolchain locks.
- **API Catalog Entry**: Metadata record generated during publishing and catalog synchronization capturing API domain, version, owners, available formats, and discoverability tags for search.
- **Toolchain Profile**: Resolved set of external validators, generators, and versions derived from `apx.lock` that ensures consistent behavior across portable environments.

## Assumptions & Dependencies

- GitHub Enterprise Server 3.x or later is available for self-hosted deployments, with access tokens and repository permissions aligned to the documented workflows.
- Air-gapped environments maintain mirrored copies of required validator and generator binaries referenced in `/docs/dependencies/index.md` so `apx fetch` can hydrate local tool caches.
- Teams agree to treat `/docs/` as the single source of truth for user experience, updating documentation before altering CLI behavior or workflows.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Platform administrators complete canonical repository bootstrapping (including protection rules and catalog generation) in under 15 minutes using only the documented CLI steps, with zero deviations in generated files.
- **SC-002**: API producer teams publish a new schema version—covering lint, breaking checks, tagging, and publishing—in under 30 minutes with a first-attempt success rate of 95% across supported schema formats.
- **SC-003**: At least 90% of surveyed consumers report that generating clients and transitioning from overlays to published modules requires no code changes beyond running documented commands.
- **SC-004**: The CLI passes parity checks that ensure 100% of documented command examples (arguments, prompts, outputs) remain accurate across quarterly releases, preventing documentation drift.
