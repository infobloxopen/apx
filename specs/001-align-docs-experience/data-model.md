# Data Model: Docs-Aligned APX Experience

## Overview
The feature coordinates documentation-driven CLI workflows for managing canonical API repositories, application repositories, schema artifacts, and toolchains. Entities below capture the state required to align implementation with the `/docs` getting started guide.

## Entities

### CanonicalRepository
- **Description**: Organization-wide source of truth for published APIs.
- **Fields**:
  - `org` (string, required): Organization slug, e.g., `myorg`.
  - `name` (string, required): Repository name, default `apis`.
  - `defaultBranch` (string, default `main`): Protected branch.
  - `protectionRules` (map<string, ProtectionRule>): Branch/tag protections.
  - `catalogPath` (string, default `catalog/catalog.yaml`): Location of generated catalog.
  - `schemaRoots` (array<string>): Paths like `proto/`, `openapi/`, etc.
  - `toolchainLock` (ToolchainProfileRef): Pinset for validators.
  - `subtreeConfig` (SubtreePolicy): Settings for git subtree publishing.
- **Relationships**:
  - One CanonicalRepository aggregates many `SchemaModule`s (one per API format & domain).
  - Owns one `Catalog` entry per API.

### ApplicationRepository
- **Description**: Producer-owned workspace for authoring schemas prior to publication.
- **Fields**:
  - `modulePath` (string, required): Root path for schema sources, e.g., `internal/apis/proto/payments/ledger`.
  - `apxConfig` (ConfigSnapshot): Parsed `apx.yaml` representing API entries and codegen preferences.
  - `bufWorkspace` (string): Path to `buf.work.yaml` for proto workflows.
  - `toolchainLock` (ToolchainProfileRef): Tools used during authoring.
  - `overlays` (array<OverlayEntry>): go.work overlay entries managed by `apx sync`.
  - `publishedRefs` (array<GitTagRef>): Tags following documented naming.
- **Relationships**:
  - Many ApplicationRepositories may target the same CanonicalRepository.
  - Each publishes one or more `SchemaVersion`s.

### SchemaModule
- **Description**: Canonical API module representing a specific domain and format.
- **Fields**:
  - `format` (enum: proto, openapi, avro, jsonschema, parquet).
  - `domain` (string, required).
  - `api` (string, required).
  - `majorVersion` (string, required): `v1`, `v2`, etc.
  - `moduleImportPath` (string, required): Canonical Go module path, e.g., `github.com/<org>/apis-go/proto/<domain>/<api>`.
  - `owners` (array<string>): CODEOWNERS mapping.
  - `catalogEntry` (CatalogEntryRef).
- **Relationships**:
  - Has many `SchemaVersion`s.
  - Linked to generated language artifacts tracked via `ToolchainProfile`.

### SchemaVersion
- **Description**: Published artifact for a specific API version.
- **Fields**:
  - `version` (semver string, required): e.g., `v1.2.3`.
  - `sourceTag` (GitTagRef, required): Tag in app repo, `proto/<domain>/<api>/v1/v1.2.3`.
  - `canonicalTag` (GitTagRef, required): Tag in canonical repo, `proto/<domain>/<api>/v1.2.3`.
  - `artifactDigest` (string): Hash of schema contents post-validation.
  - `validationReport` (ValidationSummary): Results from lint/breaking checks.
  - `generatedOutputs` (array<GeneratedArtifactRef>): Codegen outputs (Go, Python, etc.).
- **Relationships**:
  - Belongs to one `SchemaModule`.
  - Referenced by `DependencyOverlay` entries in consuming apps.

### ToolchainProfile
- **Description**: Versioned set of external tools required for schema validation and code generation.
- **Fields**:
  - `profileId` (string, required): Derived from `apx.lock`.
  - `tools` (array<ToolRef>): Each includes name, version, checksum, download source.
  - `updatedAt` (timestamp).
  - `offlineBundle` (bool): Indicates whether mirrored artifacts are available for air-gapped use.
- **Relationships**:
  - Linked from both CanonicalRepository and ApplicationRepository.
  - Used when generating `ValidationSummary` and `GeneratedArtifactRef` records.

### Catalog
- **Description**: Aggregated index describing published APIs for discovery.
- **Fields**:
  - `entries` (array<CatalogEntry>): Domain, API name, versions, owners, tags, description.
  - `generatedAt` (timestamp).
  - `format` (enum: yaml, json): Output target.
- **Relationships**:
  - One Catalog belongs to a CanonicalRepository.
  - Entries reference `SchemaModule` and latest `SchemaVersion`.

### DependencyOverlay
- **Description**: Representation of consumer-side overlays managed by `apx gen` and `apx sync`.
- **Fields**:
  - `consumerRepo` (string, required).
  - `schemaRef` (SchemaModuleRef, required).
  - `version` (string, required): Specific SchemaVersion used.
  - `languages` (array<string>): Generated languages present.
  - `goWorkEntries` (array<string>): `use ./internal/gen/...` entries inserted.
  - `linked` (bool): Indicates whether overlay is active (`apx sync`) or replaced with published module (`apx unlink`).
- **Relationships**:
  - Many DependencyOverlays can exist per ApplicationRepository or consumer service repo.

### ValidationSummary
- **Description**: Aggregated output from format validators and breaking change tools.
- **Fields**:
  - `format` (enum).
  - `lintStatus` (enum: pass, warn, fail).
  - `breakingStatus` (enum: pass, warn, fail).
  - `reports` (array<ReportRef>): Paths to lint/breaking reports or inline details.
  - `executedTools` (array<ToolInvocation>): Tool name, version, exit code, duration.
- **Relationships**:
  - Attached to SchemaVersion and stored in CI artifacts for auditing.

## Relationships Diagram (Conceptual)

```
CanonicalRepository
  ├─ Catalog
  ├─ ToolchainProfile
  └─ SchemaModule ──┬─ SchemaVersion ──┬─ ValidationSummary
                    │                  └─ GeneratedArtifactRef
                    └─ Dependencies (via ApplicationRepository / DependencyOverlay)

ApplicationRepository
  ├─ ToolchainProfile
  ├─ SchemaVersion (pre-publish state)
  └─ DependencyOverlay
```

## Validation Rules
- Canonical repository protection rules must include branch protection for `main` and tag protection matching `proto/**/v*` (and analogous patterns for other formats).
- Schema module import paths must always use canonical organization prefix with semantic import versioning rules (no `/v1` suffix for v1 modules).
- Published schema versions require successful lint + breaking checks across all configured formats before tagging.
- Dependency overlays must ensure generated code directories stay under gitignore (`internal/gen/**`).
- Toolchain profiles must validate checksums prior to execution to support offline/air-gapped integrity requirements.
