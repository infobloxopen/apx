# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

#### `crd` schema format — catalog Kubernetes CRDs as versioned capabilities (WS-036)
- A first-class **`crd`** format, alongside `proto`/`openapi`/`avro`/`jsonschema`/`parquet`.
  A Kubernetes `CustomResourceDefinition` becomes a versioned, lint-checked,
  breaking-analyzed, releasable, catalog-resolvable module — so a GVK is a
  version-constrainable capability token.
- **Detection** by content (`apiextensions.k8s.io/*` + `kind: CustomResourceDefinition`).
- **Identity**: GVK maps to `crd/<group>/<kind>/<version>` (group = domain, kind = name,
  CRD version = API line). One apx module per CRD version. Alpha/beta/GA maturity maps to
  the experimental/beta/stable lifecycle; the K8s version major is the module's semver major.
- **Lint**: Kubernetes structural-schema rules and CRD conventions on the embedded
  `openAPIV3Schema` (root object, typed nodes, `x-kubernetes-preserve-unknown-fields` /
  `x-kubernetes-int-or-string` escape hatches, single storage version, unique served versions).
- **Breaking analysis**: a CRD-aware served-version compatibility checker (can't remove/narrow
  a served field, add a required field, tighten a constraint, remove enum values, or drop
  preserve-unknown-fields), with the Kubernetes rule that **alpha versions carry no
  compatibility guarantee**. Feeds `apx semver`. This is not raw oasdiff.
- **Release + catalog**: CRD modules flow through `release prepare/submit/finalize` and appear
  in `catalog generate/show/search/inspect`; catalog entries carry `crd_group`, `crd_kind`,
  `crd_scope`, `served_versions`, and `storage_version`. `apx init crd` scaffolds a starter CRD.
- Codegen stays out of apx: `controller-gen`/`kubebuilder` author CRDs; apx lifecycles them.

#### Release UX for `ci_only` canonical repos (#11)
- `apx release finalize` now **detects `release.ci_only: true`** and, when run
  locally (outside CI, without `--local`), fails fast with actionable guidance:
  the exact CI prerequisites (the finalize GitHub App install, `APX_APP_ID` /
  `APX_APP_PRIVATE_KEY` org secrets, and a tag-ruleset bypass for the app) and a
  copy-pasteable CI-mode finalize command — instead of an opaque CI error.
- New `apx release finalize --local` flag runs the CI-mode finalize from a
  contributor's machine when they control the credentials. It never silently
  pushes a protected tag: if the protected-tag push fails, finalize fails loudly
  with guidance rather than leaving a local-only tag.
- `apx release prepare` and `apx release submit` print a **preflight notice** on
  `ci_only` repos so the CI finalize handoff and its prerequisites are visible up
  front. (These org-level prerequisites cannot be probed with a contributor
  token, so they are surfaced, not verified.)
- `apx release submit` handles the **empty-PR / no-diff** case: when the prepared
  snapshot matches canonical, it exits cleanly with `Nothing to release` and a
  recommended next step instead of GitHub's opaque `HTTP 422`.
- `apx release finalize` **surfaces catalog drift** — modules that have release
  tags but no `catalog.yaml` entry — while idempotently reconciling the released
  module's entry.
- New docs: [CI-only Finalize](releasing/ci-only-finalize.md) documents the
  end-to-end contributor flow, the CI handoff and prerequisites, the `--local`
  fallback, and the downstream tag-before-consume sequencing (plus the `replace`
  bridge for local development).

#### Go client generator (`apx client generate --generator go`)
- A `go` client generator orchestrates [`oapi-codegen`](https://github.com/oapi-codegen/oapi-codegen)
  to emit a typed, buildable Go module (client + models) from an OpenAPI v3 spec — the Go client the
  devedge CLI and Terraform provider (WS-024) sit on. For this generator `--package` is the Go
  **module path** and `--scope` is ignored.
- New optional `Builder`/`Publisher` interfaces on `internal/client` (type-asserted, mirroring
  `internal/language`): the go generator's `Builder` runs `go build`, its `Publisher` records a
  `go-module` artifact honoring `--dry-run`. The `typescript-angular` npm build/publish path is
  unchanged.
- Generated Go types honor the enriched devedge-sdk contract (`enum`, `required`,
  `readOnly`/`writeOnly`); unknown `x-aip-*` vendor extensions are ignored.

### Changed

#### Single Canonical Import Root
- **Breaking**: APX no longer uses or documents `apis-go` as a default distribution repo.
  The canonical repo `github.com/<org>/apis` is now the single default source **and** Go
  distribution root.
  - All docs, specs, code comments, and templates updated from `apis-go` → `apis`
  - `apx lint` now warns if a proto file's `go_package` contains the deprecated `/apis-go/` path
  - Overlay `go.mod` module paths and `apx unlink` hints now reference `apis` directly
  - **Migration**: update `go_package` options, generated overlay config, and imports from
    `github.com/<org>/apis-go/...` to `github.com/<org>/apis/...`
  - Added "Path Mapping" reference table to Quick Start and Canonical Repo Structure docs

### Added

#### Repository Initialization Commands
- **`apx init canonical`**: Bootstrap canonical API repository structure
  - Creates organizational schema directories (proto, openapi, avro, jsonschema, parquet)
  - Generates `buf.yaml` for org-wide lint/breaking policies
  - Generates `buf.work.yaml` workspace configuration
  - Creates `CODEOWNERS` file with per-path ownership templates
  - Creates `catalog/catalog.yaml` for API discovery
  - Supports `--org`, `--repo`, `--skip-git`, and `--non-interactive` flags

- **`apx init app`**: Bootstrap application repository for schema authoring
  - Scaffolds module directory structure matching canonical import paths
  - Generates `apx.yaml` configuration file
  - Generates `buf.work.yaml` for workspace management
  - Creates `.gitignore` with `/internal/gen/` pattern
  - Auto-detects schema format from path (proto, openapi, avro, jsonschema, parquet)
  - Generates example schema files based on detected format
  - Supports `--org` and `--non-interactive` flags

#### Schema Validation Commands
- **`apx lint`**: Validate schema files for syntax and style issues
  - Auto-detects format from path or accepts `--format` flag
  - Integrates with format-specific tooling (buf for proto)
  - Provides clear error messages with file/line context

- **`apx breaking`**: Check for breaking changes in schema updates
  - Compares current schema against base reference
  - Auto-detects format or accepts `--format` flag
  - Reports breaking changes with detailed context

#### Schema Release Commands
- **`apx release`**: Release schema modules to canonical repository
  - Uses git subtree to extract module-specific history
  - Creates GitHub/Gitea pull requests automatically
  - Supports `--module-path`, `--canonical-repo`, and `--base-branch` flags
  - Handles tag creation for released versions

#### Consumer Workflow Commands
- **`apx search`**: Discover APIs in the canonical catalog
  - Searches `catalog/catalog.yaml` by name or description
  - Supports `--format` filter (proto, openapi, avro, jsonschema, parquet)
  - Accepts `--catalog` flag for custom catalog location

- **`apx add`**: Add dependencies to `apx.lock`
  - Pins schema module versions for reproducible builds
  - Updates both `apx.yaml` and `apx.lock` files
  - Validates dependency existence in canonical repository

- **`apx gen`**: Generate client code from schema dependencies
  - Supports Go, Python, and Java code generation
  - Creates overlays in `/internal/gen/<language>/` structure
  - Preserves canonical import paths for seamless development
  - Auto-syncs `go.work` for Go language overlays

- **`apx sync`**: Synchronize `go.work` with active overlays
  - Scans `/internal/gen/go/` for overlay directories
  - Regenerates `go.work` with all Go overlays
  - Idempotent operation safe to run multiple times

- **`apx unlink`**: Remove overlay and switch to released module
  - Validates dependency exists before removal
  - Removes overlay from `/internal/gen/`
  - Updates `go.work` to exclude removed overlay
  - Provides guidance for adding released module to `go.mod`

#### Configuration and Tooling
- **`apx config`**: Configuration management operations
- **`apx fetch`**: Hydrate toolchain dependencies for offline use
- **Overlay Management**: Multi-language support with `/internal/gen/<language>/` structure
  - Prevents conflicts when generating for multiple languages
  - Go overlays use `@version` suffix for unique paths
  - Python and Java overlays follow language-specific conventions

### Changed
- Aligned CLI commands with documentation in `/docs/getting-started/quickstart.md`
- Standardized overlay directory structure to support multi-language generation
- Improved error messages with actionable guidance

### Fixed
- Canonical init now creates `catalog/catalog.yaml` in subdirectory (not root)
- App init generates `buf.work.yaml` for workspace configuration
- `.gitignore` uses `/internal/gen/` pattern with leading slash
- Unlink command validates dependency existence before removal
- Flag inheritance works correctly from parent to subcommands

### Internal
- Created comprehensive doc parity test suite to ensure CLI matches documentation
- Implemented testscript-based integration tests for all user workflows
- Added dependency manager for `apx.yaml` and `apx.lock` synchronization
- Created overlay manager for `go.work` lifecycle management
- Implemented format-specific validators with toolchain resolution

## [0.1.0] - Initial Release

### Added
- Initial project structure
- Basic CLI framework
- Module scaffolding

---

[Unreleased]: https://github.com/infobloxopen/apx/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/infobloxopen/apx/releases/tag/v0.1.0
