# Feature Specification: First-Class External API Registration

**Feature Branch**: `008-external-api-registration`  
**Created**: 2026-03-10  
**Status**: Draft  
**Input**: User description: "Add first-class external API registration so teams can catalog, govern, version, and reference third-party APIs (e.g., Google APIs from googleapis) without rewriting upstream schema import paths or pretending those APIs are first-party."

## User Scenarios & Testing

### User Story 1 — Register an External API (Priority: P1)

A platform team wants to make a third-party API — such as Google Cloud Pub/Sub from the upstream `googleapis` repository — discoverable and referenceable inside the organization's APX catalog. They register the API by declaring its managed location (the internal canonical repository and path where curated snapshots live), its upstream origin (the external repository and path where the original schemas are maintained), and an import mode that controls whether downstream consumers use the upstream schema import paths as-is or rewrite them to internal paths. The default import mode is "preserve," meaning consumers import schemas using the original upstream paths.

**Why this priority**: Without registration, external APIs are invisible to `apx search`, cannot participate in APX dependency resolution, and must be managed through ad-hoc scripts. This is the foundational capability that every other story depends on.

**Independent Test**: Can be fully tested by registering a single external API (e.g., `proto/google/pubsub/v1`) with managed and upstream coordinates, then verifying it appears in the catalog with correct provenance metadata and classification.

**Acceptance Scenarios**:

1. **Given** a canonical repository with an `apx.yaml` configuration, **When** an operator registers an external API by specifying its API ID, managed repository/path, upstream repository/path, and import mode "preserve," **Then** the catalog includes a new entry with the API ID, both managed and upstream coordinates, import mode set to "preserve," and classification set to "registered."
2. **Given** a registration command is issued without specifying an import mode, **When** the registration completes, **Then** the import mode defaults to "preserve."
3. **Given** an API ID that already exists in the catalog as a first-party API, **When** an operator attempts to register it as an external API, **Then** the system rejects the registration with an error indicating the ID conflict.
4. **Given** an API ID that already exists in the catalog as a registered external API, **When** an operator attempts to register it again, **Then** the system rejects the duplicate registration with an error indicating the external API is already registered.

---

### User Story 2 — Discover and Inspect External APIs (Priority: P1)

A developer wants to find available external APIs in the catalog. They use `apx search` to discover APIs and `apx inspect identity` to view detailed provenance — including whether an API is first-party or external, where the managed copy is curated, and where the upstream origin lives.

**Why this priority**: Discovery is equally critical to registration. If external APIs cannot be found and distinguished from first-party APIs, teams will not adopt the registration model. This story can be shipped alongside or independently of Story 1.

**Independent Test**: Can be tested by registering one or more external APIs, then running search and inspect commands and verifying the output includes external provenance indicators, managed location, upstream origin, and import mode.

**Acceptance Scenarios**:

1. **Given** a catalog containing both first-party and registered external APIs, **When** a developer runs `apx search` with a free-text query matching an external API, **Then** the search results include the external API and its listing clearly indicates it is external (e.g., a provenance label or flag).
2. **Given** a catalog containing registered external APIs, **When** a developer runs `apx search --origin external`, **Then** only external APIs appear in the results.
3. **Given** a registered external API with ID `proto/google/pubsub/v1`, **When** a developer runs `apx inspect identity proto/google/pubsub/v1`, **Then** the output shows the API ID, managed repository and path, upstream repository and path, import mode, classification ("registered"), and lifecycle.
4. **Given** a catalog with no external APIs, **When** a developer runs `apx search --origin external`, **Then** the results are empty with a message indicating no external APIs are registered.

---

### User Story 3 — Depend on an External API (Priority: P2)

A developer building an internal API wants to declare a dependency on a registered external API (e.g., `proto/google/pubsub/v1`) through the standard APX dependency workflow. When the import mode is "preserve," the dependency resolution fetches schemas from the managed repository but leaves the upstream import paths intact so that `.proto` files continue to use `import "google/pubsub/v1/pubsub.proto"` rather than rewritten internal paths.

**Why this priority**: Dependency resolution is the primary consumption path. However, it depends on the API being registered and discoverable first (Stories 1 & 2), and the existing dependency manager already handles internal APIs — extending it to external APIs is an incremental enhancement.

**Independent Test**: Can be tested by registering an external API, adding it as a dependency to a consuming project's `apx.yaml`, running `apx dep add`, and verifying that the lock file records the external API with its managed source coordinates and that resolved schema files retain their upstream import paths.

**Acceptance Scenarios**:

1. **Given** a registered external API `proto/google/pubsub/v1` with import mode "preserve," **When** a developer runs `apx dep add proto/google/pubsub/v1`, **Then** the lock file records the dependency with the managed repository as the fetch source, the resolved schema files use original upstream import paths (e.g., `google/pubsub/v1/pubsub.proto`), and no path rewriting occurs.
2. **Given** a registered external API with import mode "rewrite," **When** a developer runs `apx dep add proto/google/pubsub/v1`, **Then** the lock file records the dependency and the resolved schema files have their import paths rewritten to the internal canonical paths.
3. **Given** an API ID that is not registered in the catalog, **When** a developer runs `apx dep add proto/unknown/service/v1`, **Then** the system returns an error indicating the API is not found and suggests running `apx search` to discover available APIs.
4. **Given** a project already depending on an external API, **When** a developer runs `apx dep update proto/google/pubsub/v1`, **Then** the system fetches the latest version from the managed repository and updates the lock file while preserving the configured import mode.

---

### User Story 4 — Transition from Registered to Forked/Internalized (Priority: P3)

An organization has been consuming a registered external API with upstream import paths preserved. They now need to diverge from upstream — perhaps to add internal extensions, fix a bug, or customize the schema. They transition the API's classification from "registered" (upstream-preserving) to "forked" (internalized), which signals that the managed copy is now the authoritative source and import paths may be rewritten to internal coordinates.

**Why this priority**: Transition is an important lifecycle capability but represents an exceptional workflow. Most teams will consume external APIs without forking. This story can be deferred without blocking the core registration and consumption loop.

**Independent Test**: Can be tested by registering an external API as "registered," running a transition command, and verifying the classification changes to "forked," the upstream origin is preserved for provenance, and the import mode switches to "rewrite."

**Acceptance Scenarios**:

1. **Given** a registered external API `proto/google/pubsub/v1` with classification "registered" and import mode "preserve," **When** an operator transitions it to "forked," **Then** the catalog entry's classification changes to "forked," the import mode changes to "rewrite," and the upstream origin coordinates are retained for historical provenance.
2. **Given** a forked external API, **When** a developer runs `apx inspect identity proto/google/pubsub/v1`, **Then** the output shows classification "forked," import mode "rewrite," and the original upstream origin.
3. **Given** a forked external API, **When** an operator attempts to transition it back to "registered," **Then** the system allows the reverse transition and restores import mode to "preserve."
4. **Given** a first-party API (not registered as external), **When** an operator attempts to transition it to "forked," **Then** the system rejects the operation with an error indicating that only external APIs can be transitioned.

---

### User Story 5 — Version and Lifecycle External APIs (Priority: P2)

A platform team wants to tag and version a curated snapshot of an external API in the managed repository, following the same versioning and lifecycle model used for first-party APIs, so that consumers can pin to stable releases.

**Why this priority**: Version governance of external APIs prevents teams from silently pulling breaking upstream changes. It builds on Story 1 (registration) and enables reproducible builds, but the core registration and discovery loop can function without it initially.

**Independent Test**: Can be tested by registering an external API, publishing a versioned snapshot to the managed repository, and verifying that the catalog and version tags reflect the new version and lifecycle.

**Acceptance Scenarios**:

1. **Given** a registered external API `proto/google/pubsub/v1` with a curated schema snapshot in the managed repository, **When** an operator publishes version `v1.0.0` with lifecycle "stable," **Then** the catalog entry's version and lifecycle fields are updated, and a version tag is created in the managed repository.
2. **Given** a registered external API at version `v1.0.0` (stable), **When** upstream publishes breaking changes, **Then** the managed copy remains unchanged at `v1.0.0` until an operator explicitly curates and publishes a new version.
3. **Given** a registered external API, **When** an operator publishes a new version `v1.1.0-beta.1` with lifecycle "beta," **Then** both the latest stable and latest prerelease fields are maintained independently in the catalog.

---

### Edge Cases

- What happens when the upstream repository becomes unavailable (deleted, made private, or moved)? The managed copy remains unaffected because APX serves schemas from the managed repository. The registration entry's upstream coordinates become stale metadata; the system warns but does not break.
- What happens when someone registers an external API with an import mode of "rewrite" but the consuming project's build toolchain does not support path rewriting? The build fails at code generation time with clear errors referencing the unresolvable imports. APX surfaces this as a validation warning during `apx dep add`.
- What happens when two teams register the same upstream API under different API IDs? The system allows it — each registration is a distinct managed entry. Discovery results show both entries with their respective managed paths.
- What happens when a registered external API's managed path overlaps with a first-party API's path in the canonical repository? Registration is rejected with an error indicating the path conflict.
- What happens when import mode is "preserve" but the upstream import paths conflict with internal namespace conventions? The system proceeds (preserve means no rewriting), but `apx validate` may surface naming convention warnings if the organization has configured namespace policies.
- What happens when a "forked" API wants to track selective upstream changes (cherry-pick)? This is outside the scope of this feature. Forked APIs are fully managed internally. Tracking upstream diffs is a separate future capability.

## Requirements

### Functional Requirements

- **FR-001**: System MUST support registering an external API by specifying: API ID (format/domain/name/line), managed repository, managed path, upstream repository, upstream path, and import mode.
- **FR-002**: System MUST default the import mode to "preserve" when not explicitly specified during registration.
- **FR-003**: System MUST classify registered external APIs as "registered" (upstream-preserving) or "forked" (internalized), with "registered" as the default classification.
- **FR-004**: System MUST reject registration of an API ID that already exists in the catalog as either a first-party or external API.
- **FR-005**: System MUST reject registration when the managed path conflicts with an existing entry's path in the canonical repository.
- **FR-006**: System MUST persist external API registrations in the catalog with all provenance metadata (managed coordinates, upstream coordinates, import mode, classification).
- **FR-007**: System MUST expose external APIs through existing search and discovery commands, with filtering by origin type (first-party vs. external).
- **FR-008**: System MUST display complete provenance information (managed location, upstream origin, import mode, classification) when inspecting an external API's identity.
- **FR-009**: System MUST support adding external APIs as dependencies through the existing dependency workflow, fetching schemas from the managed repository.
- **FR-010**: System MUST preserve upstream import paths in resolved schemas when import mode is "preserve."
- **FR-011**: System MUST rewrite import paths to internal canonical paths when import mode is "rewrite."
- **FR-012**: System MUST support transitioning an external API's classification between "registered" and "forked," updating the import mode accordingly.
- **FR-013**: System MUST retain the original upstream origin coordinates when an API transitions from "registered" to "forked," preserving provenance history.
- **FR-014**: System MUST support versioning and lifecycle management for external APIs using the same model as first-party APIs (version tags, lifecycle states: experimental, beta, stable, deprecated, sunset).
- **FR-015**: System MUST allow curated external API snapshots to be versioned independently from upstream releases.
- **FR-016**: System MUST validate that external API registrations use a valid API ID format (format/domain/name/line) and valid format types.
- **FR-017**: System MUST validate that the upstream repository URL is well-formed.

### Key Entities

- **External API Registration**: A catalog entry representing a third-party API. Attributes: API ID, managed repository, managed path, upstream repository, upstream path, import mode ("preserve" or "rewrite"), classification ("registered" or "forked"), lifecycle, version, and standard catalog metadata (description, tags, owners).
- **Import Mode**: A setting controlling how downstream consumers reference schemas from an external API. "Preserve" keeps original upstream import paths intact. "Rewrite" transforms import paths to match internal canonical conventions.
- **Classification**: A label distinguishing two operational modes for external APIs. "Registered" means the managed copy faithfully mirrors upstream and imports are preserved. "Forked" means the organization has diverged from upstream and the managed copy is authoritative.
- **Managed Location**: The internal canonical repository and path where curated snapshots of an external API's schemas are stored and served to consumers.
- **Upstream Origin**: The original external repository and path where the third-party API's schemas are maintained by the external owner. Retained as provenance metadata even after forking.

## Success Criteria

### Measurable Outcomes

- **SC-001**: A platform team can register an external API and have it appear in catalog search results within a single command invocation.
- **SC-002**: Developers can discover all external APIs in the catalog using a single search command with an origin filter, with results clearly distinguishing external APIs from first-party APIs.
- **SC-003**: A developer can add a registered external API as a dependency and have schemas resolved with correct import paths (preserved or rewritten per configuration) without manual path manipulation.
- **SC-004**: 100% of registered external APIs retain their upstream origin metadata after transitioning to "forked" classification, preserving full provenance history.
- **SC-005**: External APIs follow the same versioning model as first-party APIs — version tags, lifecycle states, and version queries all work identically.
- **SC-006**: Registration of a conflicting API ID (duplicate or path collision) is rejected before any catalog state is modified.
- **SC-007**: A registered external API's managed copy is unaffected by upstream repository outages or changes, ensuring zero disruption to downstream consumers.

## Assumptions

- The managed repository (where curated external API snapshots are stored) is the organization's existing canonical API repository. External APIs are colocated alongside first-party APIs in the same canonical repo.
- Upstream repository snapshots are curated manually or through a separate sync tool. The initial scope of this feature does not include automated upstream tracking or sync — it focuses on registration, discovery, dependency resolution, and governance.
- Import path rewriting (import mode "rewrite") applies to proto `import` statements and analogous references in other schema formats. The rewriting logic is format-specific and will be implemented per format as needed.
- The existing APX identity model (format/domain/name/line) is sufficient for identifying external APIs. No new ID format is required.
- External API snapshots in the managed repository are stored under paths that follow the same conventions as first-party APIs (e.g., `proto/google/pubsub/v1/`).
- Organizations that want to preserve upstream import paths (the default) accept that their internal namespace may contain paths that do not follow internal naming conventions.
