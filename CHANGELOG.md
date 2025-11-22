# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

#### Schema Publishing Commands
- **`apx publish`**: Publish schema modules to canonical repository
  - Uses git subtree to extract module-specific history
  - Creates GitHub/Gitea pull requests automatically
  - Supports `--module-path`, `--canonical-repo`, and `--base-branch` flags
  - Handles tag creation for published versions

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

- **`apx unlink`**: Remove overlay and switch to published module
  - Validates dependency exists before removal
  - Removes overlay from `/internal/gen/`
  - Updates `go.work` to exclude removed overlay
  - Provides guidance for adding published module to `go.mod`

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
