# apx – API Publishing eXperience CLI

[![Go Version](https://img.shields.io/github/go-mod/go-version/infobloxopen/apx)](https://golang.org/)
[![Release](https://img.shields.io/github/release/infobloxopen/apx.svg)](https://github.com/infobloxopen/apx/releases/latest)
[![License](https://img.shields.io/github/license/infobloxopen/apx)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/infobloxopen/apx)](https://goreportcard.com/report/github.com/infobloxopen/apx)
[![Documentation](https://img.shields.io/badge/docs-github.io-blue)](https://infobloxopen.github.io/apx/)

**apx** is a CLI tool that implements the **canonical repository pattern** for API schema management. It enables organizations to centralize API schemas in a single source of truth while allowing teams to author schemas in their application repositories with canonical import paths.

## Key Features

- 🎯 **Canonical Import Paths**: Single import path that works in development and production
- 🔄 **go.work Overlays**: Seamless transition between local development and published modules
- 🏢 **Organization-Wide Catalog**: Centralized API discovery across all teams
- 🚀 **Multi-Format Support**: Protocol Buffers, OpenAPI, Avro, JSON Schema, Parquet
- 🔍 **Schema Validation**: Automated linting and breaking change detection
- 📦 **Code Generation**: Generate client code for Go, Python, and Java
- 🔐 **Policy Enforcement**: Org-wide lint and breaking change policies

## Architecture Overview

APX implements a two-repository pattern:

1. **Canonical Repository** (`github.com/<org>/apis`): Single source of truth for all published APIs
2. **App Repositories**: Where teams author schemas and generate code with canonical import paths

**Benefits:**
- Import paths never change when switching from dev to production
- No `replace` directives or import rewrites
- Clean dependency management via `go.work` overlays

## Quick Start

See the [Quick Start Guide](https://infobloxopen.github.io/apx/getting-started/quickstart.html) for a comprehensive walkthrough.

### 1. Bootstrap Canonical Repository

```bash
# Create your organization's canonical API repository
git clone https://github.com/<org>/apis.git
cd apis

# Initialize the canonical structure
apx init canonical --org=<org> --repo=apis
```

Creates:
```
apis/
├── buf.yaml              # Org-wide lint/breaking policy
├── buf.work.yaml         # Workspace config
├── CODEOWNERS            # Per-path ownership
├── catalog/
│   └── catalog.yaml      # API discovery catalog
└── proto/                # Schema directories
    └── openapi/
    └── avro/
    └── jsonschema/
    └── parquet/
```

### 2. Author API in App Repository

```bash
# In your application repository
cd /path/to/your-app

# Initialize app structure
apx init app internal/apis/proto/payments/ledger

# Lint your schema
apx lint internal/apis/proto/payments/ledger

# Check for breaking changes
apx breaking internal/apis/proto/payments/ledger

# Publish to canonical repo
apx publish --module-path=internal/apis/proto/payments/ledger
```

### 3. Consume API in Another Service

```bash
# Search for APIs
apx search payment

# Add dependency
apx add proto/payments/ledger/v1@v1.2.3

# Generate code with canonical imports
apx gen go

# Your code now uses: github.com/<org>/apis/proto/payments/ledger/v1
# Works seamlessly via go.work overlay!

# When ready, switch to published module
apx unlink proto/payments/ledger/v1
```

## Installation

### Homebrew (macOS/Linux)

```bash
brew install infobloxopen/tap/apx
```

### Download Binary

Download the latest release from [GitHub Releases](https://github.com/infobloxopen/apx/releases) for your platform.

### Build from Source

```bash
go install github.com/infobloxopen/apx/cmd/apx@latest
```

### Install Tool Dependencies

APX integrates with format-specific tooling:

```bash
# Install all required tools
curl -sSL https://raw.githubusercontent.com/infobloxopen/apx/main/scripts/install-tools.sh | bash
```

Or install individually:
- [buf](https://github.com/bufbuild/buf) - Protocol Buffer linting and breaking change detection
- [spectral](https://github.com/stoplightio/spectral) - OpenAPI linting (optional)
- [oasdiff](https://github.com/Tufin/oasdiff) - OpenAPI breaking changes (optional)

## Command Reference

### Repository Initialization

#### `apx init canonical`

Bootstrap a canonical API repository.

```bash
apx init canonical --org=myorg --repo=apis
```

**Flags:**
- `--org`: Organization name (required)
- `--repo`: Repository name (required)
- `--skip-git`: Skip git initialization
- `--non-interactive`: Skip interactive prompts

#### `apx init app <module-path>`

Bootstrap an application repository for schema authoring.

```bash
apx init app internal/apis/proto/payments/ledger
```

**Flags:**
- `--org`: Organization name (required)
- `--non-interactive`: Skip interactive prompts

**Auto-detects format** from path:
- `/proto/` → Protocol Buffers
- `/openapi/` → OpenAPI
- `/avro/` → Avro
- `/jsonschema/` → JSON Schema
- `/parquet/` → Parquet

### Schema Validation

#### `apx lint [path]`

Validate schema files for syntax and style issues.

```bash
apx lint                                  # Lint current directory
apx lint internal/apis/proto/payments    # Lint specific path
apx lint --format=proto                   # Explicit format
```

#### `apx breaking [path]`

Check for breaking changes.

```bash
apx breaking internal/apis/proto/payments
apx breaking --format=openapi
```

### Publishing

#### `apx publish`

Publish schema module to canonical repository.

```bash
apx publish --module-path=internal/apis/proto/payments/ledger
apx publish --module-path=... --canonical-repo=git@github.com:org/apis.git
apx publish --module-path=... --base-branch=develop
```

**Flags:**
- `--module-path`: Path to module in app repo (required)
- `--canonical-repo`: Canonical repository URL
- `--base-branch`: Target branch (default: main)

### Consumer Workflow

#### `apx search [query]`

Search for APIs in the canonical catalog.

```bash
apx search                          # List all APIs
apx search payment                  # Search by keyword
apx search --format=proto           # Filter by format
apx search --catalog=path/to/catalog.yaml
```

#### `apx add <module-path>[@version]`

Add a schema dependency.

```bash
apx add proto/payments/ledger/v1@v1.2.3
apx add proto/users/profile/v1              # Uses latest
```

Updates both `apx.yaml` and `apx.lock` files.

#### `apx gen <language> [path]`

Generate client code from dependencies.

```bash
apx gen go                    # Generate Go code
apx gen python                # Generate Python code  
apx gen java                  # Generate Java code
```

**Generated structure:**
```
/internal/gen/
├── go/
│   └── proto/payments/ledger@v1.2.3/
├── python/
│   └── proto/payments/ledger/
└── java/
    └── proto/payments/ledger/
```

**Note:** `/internal/gen/` is git-ignored. Never commit generated code.

#### `apx sync`

Synchronize `go.work` with active Go overlays.

```bash
apx sync
```

Regenerates `go.work` to include all overlays in `/internal/gen/go/`.

#### `apx unlink <module-path>`

Remove overlay and switch to published module.

```bash
apx unlink proto/payments/ledger/v1
```

Removes overlay from `/internal/gen/` and updates `go.work`.

## Configuration Files

### `apx.yaml` (App Repository)

Generated by `apx init app`:

```yaml
kind: proto
module: payments.ledger.v1
org: myorg
version: v1
```

### `apx.lock` (App Repository)

Pinned dependency versions (generated by `apx add`):

```yaml
dependencies:
  proto/payments/ledger/v1:
    repo: github.com/myorg/apis
    ref: v1.2.3
    modules:
      - proto/payments/ledger/v1
```

### `catalog/catalog.yaml` (Canonical Repository)

API discovery catalog (auto-generated):

```yaml
version: 1
org: myorg
repo: apis
modules:
  - name: proto/payments/ledger/v1
    format: proto
    description: Payment ledger API
    version: v1.2.3
    path: proto/payments/ledger/v1
```

## How It Works: Canonical Import Paths

### Development Flow

1. **Generate overlay**: `apx gen go` creates `/internal/gen/go/proto/payments/ledger@v1.2.3/`
2. **go.work magic**: `apx sync` updates `go.work` to map canonical path to local overlay
3. **Your code imports**: `import "github.com/myorg/apis/proto/payments/ledger/v1"`
4. **Go resolves**: Via `go.work`, imports resolve to your local overlay

### Production Flow

1. **Remove overlay**: `apx unlink proto/payments/ledger/v1`
2. **Add published module**: `go get github.com/myorg/apis/proto/payments/ledger/v1@v1.2.3`
3. **Imports unchanged**: Same `import "github.com/myorg/apis/proto/payments/ledger/v1"`
4. **Go resolves**: From published module in `go.mod`

**No import rewrites. No replace directives. It just works.**

## Multi-Language Support

### Overlay Structure

```
/internal/gen/
├── go/                           # Go overlays
│   ├── proto/payments/ledger@v1.2.3/
│   └── proto/users/profile@v1.0.1/
├── python/                       # Python packages
│   ├── proto/payments/ledger/
│   └── proto/users/profile/
└── java/                         # Java packages
    ├── proto/payments/ledger/
    └── proto/users/profile/
```

**Why language subdirectories?**
- Prevents conflicts when generating for multiple languages
- Each language has its own namespace and structure
- Overlay manager handles language-specific path resolution

## Global Flags

- `--config <file>`: Specify config file (default: `apx.yaml`)
- `--verbose`: Enable verbose output
- `--quiet`: Suppress output
- `--json`: Output in JSON format (planned)
- `--no-color`: Disable colored output

## CI/CD Integration

### GitHub Actions Example

```yaml
name: API Schema Workflow

on:
  pull_request:
    paths:
      - 'internal/apis/**'

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.24'

      - name: Install APX
        run: go install github.com/infobloxopen/apx/cmd/apx@latest

      - name: Lint Schemas
        run: apx lint internal/apis

      - name: Check Breaking Changes
        run: apx breaking internal/apis

  publish:
    if: github.ref == 'refs/heads/main'
    needs: validate
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
          token: ${{ secrets.CANONICAL_REPO_TOKEN }}

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.24'

      - name: Install APX
        run: go install github.com/infobloxopen/apx/cmd/apx@latest

      - name: Publish to Canonical Repo
        run: |
          apx publish --module-path=internal/apis/proto/payments/ledger \
            --canonical-repo=git@github.com:myorg/apis.git
        env:
          GIT_SSH_COMMAND: "ssh -i ${{ secrets.DEPLOY_KEY }}"
```

## Documentation

**📚 [Full Documentation](https://infobloxopen.github.io/apx/)** - Complete documentation hosted on GitHub Pages

### Quick Links

- 📖 [Quick Start Guide](https://infobloxopen.github.io/apx/getting-started/quickstart/) - Complete walkthrough
- 🏗️ [Canonical Repository Structure](https://infobloxopen.github.io/apx/canonical-repo/structure/)
- 📦 [Dependency Management](https://infobloxopen.github.io/apx/dependencies/)
- 🔧 [Interactive Init Guide](https://infobloxopen.github.io/apx/getting-started/interactive-init/)
- 🚀 [Publishing Guide](https://infobloxopen.github.io/apx/publishing/)
- 🔧 [CLI Reference](https://infobloxopen.github.io/apx/cli-reference/)
- ❓ [FAQ & Troubleshooting](https://infobloxopen.github.io/apx/troubleshooting/faq/)

## Development Status

### Implemented Features ✅

- ✅ Canonical repository initialization
- ✅ App repository scaffolding
- ✅ Schema validation (lint, breaking)
- ✅ Code generation (Go, Python, Java)
- ✅ Overlay management with go.work
- ✅ API discovery and search
- ✅ Dependency management (apx.lock)
- ✅ Publishing workflow (git subtree + PR)
- ✅ Multi-language overlay structure

### Planned Features 🚧

- 🚧 JSON output for CI automation (`--json` flag)
- 🚧 Offline/air-gapped mode via `apx fetch`
- 🚧 GitHub Enterprise Server support
- 🚧 Performance instrumentation
- 🚧 Enhanced error messages with actionable guidance

See [CHANGELOG.md](CHANGELOG.md) for detailed release notes.

## Exit Codes

- `0`: Success
- `1`: General error
- `2`: Validation/lint errors
- `3`: Breaking changes detected
- `4`: Dependency not found
- `5`: Configuration error

## Development

### Building

```bash
make build
```

### Testing

```bash
make test                    # Unit tests
go test -run TestScript      # Integration testscripts
go test ./tests/integration  # Full integration tests
```

### Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development guidelines.

## License

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE) for details.

## Support

- 📚 [Documentation](https://infobloxopen.github.io/apx/) - Full documentation on GitHub Pages
- 🐛 [Issues](https://github.com/infobloxopen/apx/issues) - Bug reports and feature requests
- 💬 [Discussions](https://github.com/infobloxopen/apx/discussions) - Questions and community support