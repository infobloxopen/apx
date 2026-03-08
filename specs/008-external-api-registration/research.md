# Research: External API Registration

**Feature**: 008-external-api-registration  
**Date**: 2026-03-08

## Decision 1: API Identity Model for External APIs

**Question**: How should external APIs fit into APX's 4-segment identity model (`format/domain/name/line`) when their filesystem paths do not follow the canonical `<format>/<domain>/<name>/<line>/` layout?

**Decision**: External APIs use the standard 4-segment API ID as their **logical identity** but carry an explicit **managed path** that maps to the actual filesystem layout. The managed path is the source of truth for where schemas live; the API ID is the source of truth for catalog lookup and dependency resolution.

**Rationale**: The existing `Module` struct already has separate `ID` and `Path` fields (both YAML-serialized), but the codebase always sets them identically. For external APIs, `Path` diverges from `ID`:

| Field | Example (Google Pub/Sub) |
|-------|--------------------------|
| `ID` (identity) | `proto/google/pubsub/v1` |
| `Path` (filesystem) | `google/pubsub/v1` |
| Upstream path | `google/pubsub/v1` |

The existing `DeriveSourcePath()` remains the default for first-party APIs (identity function: ID = Path). External APIs override this via their registration metadata.

**Alternatives considered**:
- Require external APIs to use the `<format>/` prefix in their filesystem layout → Rejected: would break upstream import paths (`import "google/api/annotations.proto"` would need to become `import "proto/google/api/annotations.proto"`), violating the core feature requirement.
- Create a new identity format for external APIs (e.g., 5-segment) → Rejected: unnecessary complexity; the 4-segment model works for all identification purposes; only the path mapping needs to be explicit.

## Decision 2: Catalog Module Extension

**Question**: How should the `Module` struct represent external provenance metadata?

**Decision**: Add five new optional fields to the `Module` struct:

| Field | Type | Default | Purpose |
|-------|------|---------|---------|
| `Origin` | `string` | `""` (first-party) | `"external"` or `"forked"` |
| `ManagedRepo` | `string` | `""` | Internal curating repo (e.g., `github.com/Infoblox-CTO/apis-contrib-google`) |
| `UpstreamRepo` | `string` | `""` | Original external repo (e.g., `github.com/googleapis/googleapis`) |
| `UpstreamPath` | `string` | `""` | Path in upstream repo (e.g., `google/pubsub/v1`) |
| `ImportMode` | `string` | `""` | `"preserve"` or `"rewrite"` |

**Rationale**: These fields are additive and optional. Existing catalog entries (first-party) are unaffected — they have empty origin/upstream fields, which the system treats as first-party. The `Origin` field replaces the spec's "classification" term for consistency with the `--origin` search filter. The term "classification" is reserved for possible future taxonomy use.

**Alternatives considered**:
- Nested struct (e.g., `Provenance` sub-object in YAML) → Rejected: flat fields are simpler for YAML serialization, search filtering, and JSON output. The flat representation avoids nil-pointer complexity and keeps the Module struct query-friendly.
- Separate catalog file for external APIs → Rejected: splits discovery and defeats single-catalog search.

## Decision 3: Import Mode Semantics

**Question**: What exactly does "preserve" vs. "rewrite" import mode mean at the filesystem and compilation level?

**Decision**:

**Preserve mode** (default for `origin: external`):
- Proto `import` statements are left unchanged (e.g., `import "google/api/annotations.proto"` remains as-is).
- Schemas are stored in the managed repo at their upstream-matching path (e.g., `google/pubsub/v1/`), NOT under a `proto/` prefix.
- `protoc`/`buf` must be invoked with an include path (`-I`) pointing at the root of the managed repo.
- `option go_package` is left unchanged (preserves upstream's Go package conventions).
- APX does NOT derive Go module/import paths for preserve-mode external APIs — those are determined by the upstream's `go_package` declarations.

**Rewrite mode** (default for `origin: forked`):
- Proto `import` statements are rewritten to use internal canonical paths.
- Schemas may be stored under the standard `<format>/<domain>/<name>/<line>/` layout.
- Standard APX path derivation applies (`DeriveSourcePath`, `DeriveGoModule`, etc.).
- `option go_package` values are updated to match internal conventions.

**Rationale**: Preserve mode is the entire point of this feature — allowing organizations to use upstream APIs without breaking their import contracts. Rewrite mode exists for the fork/internalize transition where the organization intentionally diverges.

**Alternatives considered**:
- Only support preserve mode → Rejected: organizations need the ability to fork and internalize when they diverge from upstream.
- Automatic rewriting based on heuristics → Rejected: too error-prone. Import mode must be an explicit operator choice.

## Decision 4: apx.yaml Configuration Extension

**Question**: Where in `apx.yaml` should external API registrations be stored?

**Decision**: Add a top-level `external_apis` list to the `Config` struct. Each entry represents one registration:

```yaml
external_apis:
  - id: proto/google/pubsub/v1
    managed_repo: github.com/Infoblox-CTO/apis-contrib-google
    managed_path: google/pubsub/v1
    upstream_repo: github.com/googleapis/googleapis
    upstream_path: google/pubsub/v1
    import_mode: preserve      # default
    origin: external           # default
    description: "Google Cloud Pub/Sub API"
    lifecycle: stable
    owners:
      - platform-team
    tags:
      - google
      - messaging
```

This lives in the **catalog repository's** `apx.yaml`, not in consuming app repos. App repos reference external APIs through the dependency system.

**Rationale**: The catalog repo's `apx.yaml` is the natural home for organizational API registry. This parallels how `module_roots` and policy settings live in the catalog repo config. The `external_apis` list is explicitly separate from the `api:` identity block (which describes a single first-party API being authored in the current repo) and from the `modules` list in `catalog.yaml` (which is generated, not hand-authored).

**Alternatives considered**:
- Store registrations only in `catalog.yaml` → Rejected: `catalog.yaml` is generated from git tags and filesystem scans. Hand-authored registrations would be overwritten on next catalog generation.
- Separate `external.yaml` file → Rejected: adds complexity without benefit. The `apx.yaml` already holds all other configuration.

## Decision 5: Catalog Generation with External APIs

**Question**: How does `apx catalog build` / `apx catalog generate` incorporate external API registrations?

**Decision**: The catalog generator merges two sources:
1. **Auto-discovered modules** from filesystem scan (`ScanDirectory`) and git tag scan (`GenerateFromTags`) — these produce first-party entries.
2. **External API registrations** from `apx.yaml`'s `external_apis` list — these are read from config and merged into the catalog.

Merge rules:
- External registrations are converted to `Module` entries with the appropriate provenance fields set.
- If an auto-discovered module has the same `ID` as an external registration, the catalog build fails with an error (conflict detection per FR-004).
- If two external registrations have the same `ID`, config validation rejects the duplicate (FR-004).
- If an external registration's `managed_path` overlaps with an existing module's `Path`, the catalog build fails (FR-005).

**Rationale**: This keeps the auto-generation pipeline intact for first-party APIs while treating external registrations as additive configuration. The merge approach ensures a single unified catalog for discovery.

**Alternatives considered**:
- Only allow external APIs via manual catalog editing → Rejected: error-prone, not validated, and lost on regeneration.
- Generate external entries from git tags too → Rejected: external APIs may not have tags that follow APX conventions.

## Decision 6: Dependency Lock File Extension

**Question**: How should `DependencyLock` represent external API provenance?

**Decision**: Add optional fields to `DependencyLock`:

```go
type DependencyLock struct {
    Repo       string   `yaml:"repo"`
    Ref        string   `yaml:"ref"`
    Modules    []string `yaml:"modules"`
    // New fields for external provenance
    Origin       string `yaml:"origin,omitempty"`       // "external" or "forked"
    UpstreamRepo string `yaml:"upstream_repo,omitempty"` // e.g., "github.com/googleapis/googleapis"
    UpstreamPath string `yaml:"upstream_path,omitempty"` // e.g., "google/pubsub/v1"
    ImportMode   string `yaml:"import_mode,omitempty"`   // "preserve" or "rewrite"
}
```

For external APIs with `import_mode: preserve`:
- `Repo` is set to the managed repo (fetch source).
- `Origin`, `UpstreamRepo`, `UpstreamPath`, `ImportMode` record provenance.

**Rationale**: The lock file must record enough information for `apx dep` to resolve schemas correctly and for developers to understand where dependencies come from (`apx show` reads lock metadata). Adding optional fields preserves backward compatibility.

**Alternatives considered**:
- Separate lock file for external deps → Rejected: complicates dependency resolution; all deps should be in one lock file.
- Record only the managed repo (omit upstream metadata) → Rejected: provenance is critical for `apx show`/`apx inspect` and for the fork/internalize transition.

## Decision 7: CLI Command Design

**Question**: What CLI commands should be added/modified?

**Decision**:

**New commands:**
- `apx external register <api-id>` — Register an external API. Flags: `--managed-repo`, `--managed-path`, `--upstream-repo`, `--upstream-path`, `--import-mode` (default: preserve), `--description`, `--lifecycle`, `--owners`, `--tags`.
- `apx external list` — List all registered external APIs.
- `apx external transition <api-id> --to <registered|forked>` — Change classification.

**Modified commands:**
- `apx search` — New `--origin <first-party|external|forked>` filter flag.
- `apx show <api-id>` — Display provenance section for external APIs.
- `apx inspect identity <api-id>` — Display provenance metadata.
- `apx catalog build` / `apx catalog generate` — Merge external registrations into catalog.
- `apx dep add <api-id>` — Resolve from managed repo for external APIs.

**Rationale**: The `apx external` parent command groups registration-specific operations without polluting the main command namespace. Existing commands gain external-awareness through additional flags and output sections.

**Alternatives considered**:
- `apx register` as a top-level command → Rejected: "register" is too generic and could conflict with future features.
- Inline registration in `apx catalog build` → Rejected: registration is a deliberate operator action, not an automatic side-effect.

## Decision 8: Google APIs Dependency Chain

**Question**: How should APX handle the transitive dependency chain for googleapis protos?

**Decision**: Each independently-usable googleapis directory is registered as a separate external API:

| Registration | API ID | Managed Path | Dependencies |
|-------------|--------|-------------|--------------|
| Google API framework protos | `proto/google/api/v1` | `google/api` | none (within googleapis) |
| Google common types | `proto/google/type/v1` | `google/type` | none |
| Google RPC status | `proto/google/rpc/v1` | `google/rpc` | `google/protobuf/*` (runtime) |
| Google Pub/Sub | `proto/google/pubsub/v1` | `google/pubsub/v1` | `proto/google/api/v1` |

Notes:
- `google/protobuf/*` files are NOT registered — they come from the protobuf compiler's built-in include path.
- Dependency relationships between external APIs are expressed through the standard APX dependency system.
- The managed repo (`apis-contrib-google`) contains all registered directories in a flat layout matching upstream.

**Rationale**: Fine-grained registration matches how the protos are actually used. `google/api/` is a shared dependency used by virtually every googleapis API. Registering it separately enables proper dependency tracking.

**Alternatives considered**:
- Register entire googleapis as one entry → Rejected: too coarse; consumers would pull in thousands of unused protos.
- Auto-detect dependencies from import statements → Rejected: out of scope for initial implementation; manual dependency declaration is sufficient.

## Decision 9: Validation Rules for External APIs

**Question**: What validation should APX perform on external API registrations and their schemas?

**Decision**:

**Registration validation** (`apx external register`):
- API ID must be valid 4-segment format.
- Managed repo URL must be well-formed.
- Upstream repo URL must be well-formed.
- Managed path must not conflict with existing modules.
- API ID must not already exist in catalog or external registrations.

**Schema validation** (preserve mode):
- `apx lint` skips `go_package` canonical validation for preserve-mode external APIs (upstream's `go_package` is authoritative).
- `apx lint` still validates proto syntax/style (buf lint runs normally).
- Import path integrity: verify that all `import` statements in the managed path resolve to files that exist either in the managed repo or in the protobuf include path. This catches broken upstream snapshots.

**Schema validation** (rewrite mode):
- Standard APX validation applies fully (including `go_package` canonical path checks).

**Rationale**: Preserve mode explicitly relaxes canonical path assumptions because the API is not authored internally. Rewrite mode means the organization has taken ownership and full APX governance applies.

**Alternatives considered**:
- Apply full validation to all external APIs → Rejected: preserve-mode APIs intentionally use non-canonical paths.
- Skip all validation for external APIs → Rejected: basic integrity checks (file existence, valid proto syntax) are still valuable.

## Decision 10: Managed Path vs API ID Divergence

**Question**: When API ID is `proto/google/pubsub/v1` but files must live at `google/pubsub/v1/`, how does this affect the broader system?

**Decision**: Introduce the concept of an "effective source path" that is:
- For first-party APIs: `DeriveSourcePath(apiID)` = `apiID` (unchanged).
- For external APIs: `managed_path` from registration metadata.

Functions that need the effective source path:
- `DeriveSourcePath()` → gains an optional `Config` or `Module` parameter to check for external registration.
- `ScanDirectory()` → external APIs are NOT discovered by scanning; they are injected from config.
- `GenerateFromTags()` → external API tags use the managed path, not the API ID, as the tag prefix. Alternatively, external API versions are registered via `apx external register --version` rather than tag-based discovery.

**Rationale**: The API ID is a stable catalog identifier. The managed path is the operational truth for filesystem operations. These two concerns must be decoupled for external APIs.

**Alternatives considered**:
- Force the managed path to equal the API ID → Would require `proto/google/pubsub/v1/` directory layout, breaking upstream import paths.
- Change the API ID format for external APIs → Would break the unified 4-segment identity model.
