# Feature Specification: Catalog resource-type resolution (`type → module`)

**Feature Branch**: `009-catalog-type-resolution`
**Created**: 2026-07-01
**Status**: Clarified 2026-07-01 (endpoint/host + unserved-type resolved; ready for Plan)
**Input**: WS-021 **P1 / WP-A** (hub `specs/cross-service-federated-querying-proposal.md`). Cross-service
querying needs to resolve a resource **type** (from a `google.api.resource_reference`) to the module
that serves it. Clarify D-1 chose to **reuse the standard `google.api.resource_reference`** (no new
schema), so this feature is purely a **catalog index + resolution** — the "canonical via apx" piece of
ratified D3. Counterpart to devedge-sdk **F041** (`specs/041-cross-service-references`), which emits the
reference metadata and guarantees `BatchGet`; this feature answers *"where does type T live?"*.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Index the resource types each module serves (Priority: P1)

When apx generates or updates a repo's catalog, it records, per module, the **AIP-122 resource types**
that module defines (read from the `google.api.resource` annotations already present in the module's
protos). The catalog then knows not only *which modules exist* but *which resource types each serves*.

**Why this priority**: Nothing can resolve a cross-service reference until the catalog knows which
module owns a type. This is the foundational index; everything else reads it. It is derived from
annotations that already exist, so it requires **no schema release** (D3+D-1).

**Independent Test**: Generate a catalog for a fixture repo whose protos declare
`google.api.resource` types across two modules; assert the catalog records each type against its
owning module (type → module ID), with no manual entry.

**Acceptance Scenarios**:

1. **Given** a module whose proto declares `option (google.api.resource) = { type: "iam.example.com/User", … }`, **When** apx builds the catalog, **Then** the catalog records `iam.example.com/User → <that module's ID>`.
2. **Given** a repo with several modules, **When** the catalog is built, **Then** every declared resource type across all modules is indexed exactly once, keyed by its globally-unique AIP-122 type string.
3. **Given** a proto with no `google.api.resource` annotation, **When** the catalog is built, **Then** no type entry is created for it (only annotated resources are indexed).

---

### User Story 2 - Resolve a type to its serving module (Priority: P1)

A consumer holding a resource type (from a `google.api.resource_reference.type`) asks apx to resolve
it to the serving module — via both a **library API** and a **CLI command** — getting the module's ID,
domain, API line, version, and lifecycle. Because AIP-122 types are globally unique, `type → module`
is deterministic.

**Why this priority**: This is the query the P2 REST link-expansion (and a future gateway) calls to
learn where to fetch a referenced resource. Without it the reference metadata from F041 is inert.

**Independent Test**: Against a built catalog, resolve a known type and assert the returned module
coordinates; resolve an unknown type and assert a clear "unresolved reference" error; resolve a type
claimed by two modules and assert an "ambiguous" error (never a silent pick).

**Acceptance Scenarios**:

1. **Given** a catalog indexing `iam.example.com/User → proto/iam/v1`, **When** a consumer resolves `iam.example.com/User`, **Then** apx returns that module's coordinates (ID, domain, api_line, version, lifecycle).
2. **Given** a type absent from the catalog, **When** a consumer resolves it, **Then** apx fails with a clear **unresolved-reference** error naming the type — never an empty/nil success (matches F041 D-4 fail-loud).
3. **Given** two modules both declaring the same resource type, **When** a consumer resolves it, **Then** apx fails with an **ambiguous-type** error listing the claimants — never silently picking one.
4. **Given** the serving module's lifecycle is `deprecated`/`sunset`, **When** a consumer resolves the type, **Then** the result surfaces that lifecycle so the consumer can warn.

---

### User Story 3 - Carry the index in the published catalog artifact (Priority: P2)

The resource-type index travels inside the **published OCI catalog artifact**, so a consumer resolving
from the RegistrySource/HTTPSource/AggregateSource gets the **same** answer as from a LocalSource —
enabling cross-repo / cross-team resolution (team B resolves a type served by team A's published API).

**Why this priority**: Local resolution proves the mechanism (P1); cross-team resolution is what makes
federation across separately-published services real, but it can follow the local path.

**Independent Test**: Publish a catalog to a local OCI registry, pull it via RegistrySource, resolve a
type, and assert byte-parity of the resolution result with the LocalSource answer.

**Acceptance Scenarios**:

1. **Given** a catalog published as an OCI artifact, **When** it is pulled via RegistrySource and a type is resolved, **Then** the result equals the LocalSource resolution for the same type.
2. **Given** multiple published catalogs merged via AggregateSource, **When** a type is resolved, **Then** de-duplication by (org/repo/moduleID) is honored and a cross-catalog type collision surfaces as ambiguous (US2 scenario 3).

---

### Edge Cases

- **External / forked-imported modules** (`origin: external`/`forked`): a type served by an imported
  module resolves to the **managing** module (`managed_repo`), not the upstream, so consumers call the
  curated surface.
- **Type declared but no service serves it** (schema-only module, e.g. a shared message library):
  **RESOLVED (Clarify 2026-07-01)** — resolution **succeeds with a "no serving surface" warning**, not
  an error; the consumer decides whether that is fatal for its use.
- **Version skew**: the same type across two `api_line`s/versions of one module — resolution returns
  the version coordinates so the consumer can pin (US2 result carries version).
- **Endpoint vs module**: **RESOLVED (Clarify 2026-07-01)** — apx resolves `type → module` (+
  domain/api_line, from which the **API path** is derivable via `apilayout`) and returns **path
  coordinates only**. The concrete **network host** stays **consumer/environment-supplied** (devedge
  routing / WS-018 shell) — the catalog does **not** carry a base-URL hint. Keeps apx a schema-catalog
  concern, not a service registry.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: apx MUST index, per catalog module, the AIP-122 resource **types** the module declares
  (read from `google.api.resource` annotations) during catalog generation — no manual entry.
- **FR-002**: apx MUST provide a **resolution API (library) and a CLI command** that maps a resource
  type → its serving module coordinates (ID, domain, api_line, version, lifecycle).
- **FR-003**: Resolution MUST **fail loud** on an **unknown** type (unresolved) and on an **ambiguous**
  type (>1 module claims it) — never a silent empty success or arbitrary pick.
- **FR-004**: The resource-type index MUST be carried in the **published catalog artifact** (OCI) so
  Registry/HTTP/Aggregate sources resolve identically to Local.
- **FR-005**: Resolution results MUST surface the serving module's **lifecycle** (`deprecated`/`sunset`)
  so consumers can warn/pin.
- **FR-006**: The index MUST resolve **external/forked-imported** modules to the **managing** module
  (`managed_repo`), not the upstream.
- **FR-007**: apx MUST NOT require a **schema release** to populate or update the index — it is derived
  at catalog-generation time from existing annotations (consistent with D-1: no new schema).
- **FR-008**: The feature MUST stay within apx's **lifecycle-not-codegen** charter — it indexes and
  resolves over the catalog; it does not generate code or transform schemas.
- **FR-009** (RESOLVED, Clarify 2026-07-01): resolution returns **path coordinates only** (module +
  domain/api_line → API path via `apilayout`); the network **host is consumer/environment-supplied**.
  The catalog carries **no** base-URL hint (apx stays a schema catalog, not a service registry).

### Key Entities *(include if feature involves data)*

- **Resource type**: an AIP-122 `{service}/{Kind}` string (e.g. `iam.example.com/User`), globally
  unique; the resolution key. Sourced from `google.api.resource.type`.
- **Catalog module** (existing `Module{ID, Format, Domain, APILine, Version, Lifecycle, Origin,
  ManagedRepo, …}`): the resolution target.
- **Type→module index entry**: `resource type → module ID`, plus the module coordinates returned on
  resolution. Built at catalog generation, carried in the catalog artifact.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Given a resource type present in a built catalog, apx resolves it to **exactly one**
  serving module with full coordinates.
- **SC-002**: An **unknown** or **ambiguous** type fails resolution with a clear, actionable error
  **100%** of the time — no silent pick, no nil success.
- **SC-003**: Resolution from the **published OCI catalog** returns the **same** module for a type as
  resolution from the local catalog (Local/Registry parity).
- **SC-004**: For a repo whose protos already carry `google.api.resource`, the index is populated with
  **zero** schema releases and **zero** manual entries (fully derived at generation).
- **SC-005**: The devedge-sdk **F041** two-service fixture resolves service A's reference to service
  B's type → B's module via this API, then batch-fetches via B's `BatchGet` — the end-to-end WS-021 P1
  acceptance (one BatchGet per collection) passes using this resolver as the catalog-backed
  implementation of F041's `ReferenceResolver` seam.

---

**Dependencies / relationships**: devedge-sdk **F041** (WP-B) emits the reference metadata + guarantees
`BatchGet` and defines a `ReferenceResolver` seam with a static impl; **this feature is the
catalog-backed implementation of that seam** and is consumed by WS-021 **P2** (REST `?expand=`). No
`apis` schema release (D-1). **Next gate**: `/speckit.plan` → `/speckit.tasks` (all clarifications
resolved 2026-07-01: path coords only, host env-supplied; unserved-type = warning).
