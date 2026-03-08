# Data Model: External API Registration

**Feature**: 008-external-api-registration  
**Date**: 2026-03-08

## Entities

### 1. ExternalRegistration

Represents a registered external API in the organization's APX configuration. Stored in `apx.yaml` under the `external_apis` list.

| Field | Type | Required | Default | Validation | Description |
|-------|------|----------|---------|------------|-------------|
| `id` | string | yes | — | Valid 4-segment API ID (`format/domain/name/line`) | Canonical APX identity |
| `managed_repo` | string | yes | — | Well-formed repository URL | Internal repo hosting curated snapshots |
| `managed_path` | string | yes | — | Non-empty, no leading/trailing slashes, no `..` | Filesystem path in managed repo |
| `upstream_repo` | string | yes | — | Well-formed repository URL | Original external repository |
| `upstream_path` | string | yes | — | Non-empty, no leading/trailing slashes | Path in upstream repository |
| `import_mode` | string | no | `"preserve"` | One of: `preserve`, `rewrite` | Controls proto import path handling |
| `origin` | string | no | `"external"` | One of: `external`, `forked` | Classification/lifecycle mode |
| `description` | string | no | `""` | — | Human-readable description |
| `lifecycle` | string | no | `""` | Valid lifecycle value | Lifecycle state of the managed snapshot |
| `version` | string | no | `""` | Valid semver | Current version of the managed snapshot |
| `owners` | []string | no | `[]` | — | Team or individual owners |
| `tags` | []string | no | `[]` | — | Searchable tags |

**Uniqueness constraints**:
- `id` must be unique across all external registrations AND all auto-discovered first-party modules.
- `managed_path` must not overlap with any existing module's path.

**Relationships**:
- An ExternalRegistration maps 1:1 to a `Module` entry in the generated catalog.
- An ExternalRegistration may be referenced by zero or more `DependencyLock` entries in consuming projects.

### 2. Module (extended)

The existing `Module` struct in `internal/catalog/generator.go`, extended with five new fields.

| New Field | Type | YAML Key | Default | Description |
|-----------|------|----------|---------|-------------|
| `Origin` | string | `origin` | `""` (first-party) | API origin: `""` (first-party), `"external"`, `"forked"` |
| `ManagedRepo` | string | `managed_repo` | `""` | Internal curating repository (empty for first-party) |
| `UpstreamRepo` | string | `upstream_repo` | `""` | Original external repository |
| `UpstreamPath` | string | `upstream_path` | `""` | Path in upstream repository |
| `ImportMode` | string | `import_mode` | `""` | `"preserve"` or `"rewrite"` (empty for first-party) |

**Existing fields preserved** (no changes):
- `ID`, `Name`, `Format`, `Domain`, `APILine`, `Description`, `Version`, `LatestStable`, `LatestPrerelease`, `Lifecycle`, `Compatibility`, `ProductionUse`, `Path`, `Tags`, `Owners`

**Behavior changes**:
- For first-party modules: all new fields are empty. Behavior unchanged.
- For external modules: `Path` may differ from `ID` (e.g., `Path: google/pubsub/v1`, `ID: proto/google/pubsub/v1`).

### 3. DependencyLock (extended)

The existing `DependencyLock` struct in `internal/config/config.go`, extended with provenance fields.

| New Field | Type | YAML Key | Default | Description |
|-----------|------|----------|---------|-------------|
| `Origin` | string | `origin` | `""` | `"external"` or `"forked"` (empty for first-party) |
| `UpstreamRepo` | string | `upstream_repo` | `""` | Original external repository |
| `UpstreamPath` | string | `upstream_path` | `""` | Path in upstream repository |
| `ImportMode` | string | `import_mode` | `""` | `"preserve"` or `"rewrite"` |

**Existing fields preserved**:
- `Repo` (set to managed repo for external deps), `Ref`, `Modules`

### 4. SearchOptions (extended)

The existing `SearchOptions` struct in `internal/catalog/search.go`, extended with origin filtering.

| New Field | Type | Description |
|-----------|------|-------------|
| `Origin` | string | Filter by origin: `"first-party"`, `"external"`, `"forked"`, or `""` (all) |

**Filter logic**:
- `"first-party"` → only modules with `Origin == ""`
- `"external"` → only modules with `Origin == "external"`
- `"forked"` → only modules with `Origin == "forked"`
- `""` (empty) → all modules (default)

### 5. Config (extended)

The existing `Config` struct in `internal/config/config.go`, extended with external APIs.

| New Field | Type | YAML Key | Default | Description |
|-----------|------|----------|---------|-------------|
| `ExternalAPIs` | `[]ExternalRegistration` | `external_apis` | `[]` | List of registered external APIs |

## State Transitions

### Origin Lifecycle

```
                 ┌──────────────┐
     Register    │              │     Transition
    ──────────►  │   external   │  ──────────────►  ┌─────────┐
                 │  (preserve)  │                    │  forked │
                 │              │  ◄──────────────  │(rewrite)│
                 └──────────────┘     Transition     └─────────┘
                                       (reverse)
```

**Transitions**:
- `external` → `forked`: operator explicitly transitions; import mode changes from `preserve` to `rewrite`; upstream origin metadata retained.
- `forked` → `external`: operator reverses transition; import mode reverts to `preserve`.
- First-party → `external` or `forked`: only via explicit `apx external register` with the appropriate flags. Cannot convert an auto-discovered first-party module.
- `external`/`forked` → first-party: not supported in this feature scope. Would require a full migration workflow.

### Import Mode Rules

| Origin | Import Mode | Path Derivation | go_package Validation | Import Rewriting |
|--------|------------|-----------------|----------------------|------------------|
| first-party | N/A | `DeriveSourcePath(ID)` = ID | Full canonical check | N/A |
| external | preserve | `managed_path` (from config) | Skipped | None |
| external | rewrite | `DeriveSourcePath(ID)` = ID | Full canonical check | Active |
| forked | rewrite | `DeriveSourcePath(ID)` = ID | Full canonical check | Active |
| forked | preserve | `managed_path` (from config) | Skipped | None |

## Validation Rules

### Registration Validation

| Rule | Error Condition | Error Message |
|------|----------------|---------------|
| VR-001 | API ID fails `ParseAPIID()` | `"invalid API ID: expected format/<domain>/<name>/<line>"` |
| VR-002 | `managed_repo` is empty | `"managed_repo is required"` |
| VR-003 | `managed_path` is empty | `"managed_path is required"` |
| VR-004 | `upstream_repo` is empty | `"upstream_repo is required"` |
| VR-005 | `upstream_path` is empty | `"upstream_path is required"` |
| VR-006 | `import_mode` not in {`preserve`, `rewrite`, `""`} | `"invalid import_mode: must be 'preserve' or 'rewrite'"` |
| VR-007 | `origin` not in {`external`, `forked`, `""`} | `"invalid origin: must be 'external' or 'forked'"` |
| VR-008 | Duplicate `id` in external_apis list | `"duplicate external API ID: %s"` |
| VR-009 | `id` conflicts with first-party module | `"API ID %s conflicts with existing first-party module"` |
| VR-010 | `managed_path` conflicts with existing module path | `"managed path %s conflicts with existing module at %s"` |
| VR-011 | `lifecycle` is invalid | `"invalid lifecycle: must be one of experimental, beta, stable, deprecated, sunset"` |
| VR-012 | `managed_repo` is malformed | `"invalid managed_repo URL"` |
| VR-013 | `upstream_repo` is malformed | `"invalid upstream_repo URL"` |

### Import Integrity Validation (preserve mode)

| Rule | Check | Severity |
|------|-------|----------|
| VI-001 | All `import` statements in managed_path resolve to existing files in managed repo or protobuf include path | Error |
| VI-002 | No `import` statements have been rewritten from upstream originals | Warning |
| VI-003 | `option go_package` matches upstream value (if known) | Warning |

## YAML Examples

### apx.yaml with external APIs (catalog repo)

```yaml
version: 1
org: infoblox
repo: apis
module_roots:
  - proto
  - openapi

external_apis:
  - id: proto/google/api/v1
    managed_repo: github.com/Infoblox-CTO/apis-contrib-google
    managed_path: google/api
    upstream_repo: github.com/googleapis/googleapis
    upstream_path: google/api
    import_mode: preserve
    origin: external
    description: "Google API framework protos (annotations, resources, etc.)"
    lifecycle: stable
    owners:
      - platform-team
    tags:
      - google
      - framework

  - id: proto/google/pubsub/v1
    managed_repo: github.com/Infoblox-CTO/apis-contrib-google
    managed_path: google/pubsub/v1
    upstream_repo: github.com/googleapis/googleapis
    upstream_path: google/pubsub/v1
    import_mode: preserve
    origin: external
    description: "Google Cloud Pub/Sub API"
    lifecycle: stable
    version: v1.0.0
    owners:
      - platform-team
    tags:
      - google
      - messaging
      - pubsub
```

### Generated catalog.yaml (with external APIs merged)

```yaml
version: 1
org: infoblox
repo: apis
modules:
  # First-party API (auto-discovered)
  - id: proto/payments/ledger/v1
    format: proto
    domain: payments
    api_line: v1
    path: proto/payments/ledger/v1
    lifecycle: stable
    latest_stable: v1.2.0
    owners:
      - payments-team

  # External API (from registration)
  - id: proto/google/pubsub/v1
    format: proto
    domain: google
    api_line: v1
    path: google/pubsub/v1
    origin: external
    managed_repo: github.com/Infoblox-CTO/apis-contrib-google
    upstream_repo: github.com/googleapis/googleapis
    upstream_path: google/pubsub/v1
    import_mode: preserve
    description: "Google Cloud Pub/Sub API"
    lifecycle: stable
    version: v1.0.0
    owners:
      - platform-team
    tags:
      - google
      - messaging
      - pubsub
```

### apx.lock with external dependency (consumer app repo)

```yaml
version: 1
dependencies:
  proto/google/pubsub/v1:
    repo: github.com/Infoblox-CTO/apis-contrib-google
    ref: v1.0.0
    modules:
      - google/pubsub/v1
    origin: external
    upstream_repo: github.com/googleapis/googleapis
    upstream_path: google/pubsub/v1
    import_mode: preserve
```
