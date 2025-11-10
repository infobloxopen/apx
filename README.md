# apx – API Publishing eXperience CLI

[![Go Version](https://img.shields.io/github/go-mod/go-version/infobloxopen/apx)](https://golang.org/)
[![Release](https://img.shields.io/github/release/infobloxopen/apx.svg)](https://github.com/infobloxopen/apx/releases/latest)
[![License](https://img.shields.io/github/license/infobloxopen/apx)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/infobloxopen/apx)](https://goreportcard.com/report/github.com/infobloxopen/apx)

**apx** is a cross-platform CLI tool that orchestrates API schema management, validation, and publishing. It provides a unified interface for working with Protocol Buffers, OpenAPI, Avro, JSON Schema, and Parquet schemas while enforcing organizational policies and automating semantic versioning.

## Features

- 🚀 **Multi-format support**: Protocol Buffers, OpenAPI, Avro, JSON Schema, Parquet
- 🔍 **Schema validation**: Lint and validate schemas using industry-standard tools
- 💥 **Breaking change detection**: Automatically detect breaking changes and suggest semantic version bumps
- 🏗️ **Code generation**: Generate Go, Python, and Java code from schemas
- 📋 **Policy enforcement**: Enforce organizational policies and allowed plugins
- 🏷️ **Git integration**: Create subdirectory tags for modular releases
- 🐳 **Container support**: Run tools locally or in containers
- 📊 **Rich output**: Human-friendly CLI output with optional JSON for automation

## Quick Start

### Installation

#### Homebrew (macOS/Linux)

```bash
brew install infobloxopen/tap/apx
```

#### Download Binary

Download the latest release from [GitHub Releases](https://github.com/infobloxopen/apx/releases) for your platform.

#### Build from Source

```bash
go install github.com/infobloxopen/apx/cmd/apx@latest
```

### Install Dependencies

Install required external tools:

```bash
curl -sSL https://raw.githubusercontent.com/infobloxopen/apx/main/scripts/install-tools.sh | bash
```

Or manually install:
- [buf](https://github.com/bufbuild/buf) - Protocol Buffer tooling
- [spectral](https://github.com/stoplightio/spectral) - OpenAPI linting
- [oasdiff](https://github.com/Tufin/oasdiff) - OpenAPI diff tool
- [protoc](https://github.com/protocolbuffers/protobuf) - Protocol Buffer compiler

### Initialize Configuration

```bash
apx config init
```

This creates an `apx.yaml` configuration file with sensible defaults.

## Usage

### Basic Commands

```bash
# Initialize a new schema module
apx init proto payment/v1

# Lint schemas
apx lint

# Check for breaking changes
apx breaking --against main

# Generate code
apx gen go
apx gen python
apx gen java

# Check policy compliance
apx policy check

# Suggest semantic version bump
apx semver suggest --against v1.0.0

# Publish a release (CI only by default)
apx publish --version v1.1.0
```

### Global Flags

- `--config <file>`: Specify config file (default: `apx.yaml`)
- `--verbose`: Enable verbose output
- `--quiet`: Suppress output
- `--json`: Output in JSON format
- `--no-color`: Disable colored output

## Configuration

The `apx.yaml` file controls all behavior:

```yaml
version: 1
org: your-org-name
repo: your-apis-repo

module_roots:
  - proto
  - openapi
  - avro
  - jsonschema
  - parquet

language_targets:
  go:
    enabled: true
    plugins:
      - name: protoc-gen-go
        version: v1.64.0
      - name: protoc-gen-go-grpc
        version: v1.5.0

policy:
  forbidden_proto_options:
    - "^gorm\\."
  allowed_proto_plugins:
    - protoc-gen-go
    - protoc-gen-go-grpc

publishing:
  tag_format: "{subdir}/v{version}"
  ci_only: true

tools:
  buf:
    version: v1.45.0
  spectral:
    version: v6.11.0

execution:
  mode: "local"  # or "container"
```

See [apx.example.yaml](apx.example.yaml) for a complete configuration reference.

## Command Reference

### `apx init <kind> <modulePath>`

Initialize a new schema module.

**Kinds**: `proto`, `openapi`, `avro`, `jsonschema`, `parquet`

```bash
# Create a new protobuf module
apx init proto payment/ledger/v1

# Create an OpenAPI module
apx init openapi user/v2
```

### `apx lint [path]`

Lint and validate schema files.

```bash
# Lint current directory
apx lint

# Lint specific path
apx lint proto/payment/v1

# JSON output for CI
apx lint --json
```

### `apx breaking --against <ref> [path]`

Check for breaking changes against a Git reference.

```bash
# Check against main branch
apx breaking --against main

# Check against specific tag
apx breaking --against v1.0.0

# Check specific module
apx breaking --against main proto/payment/v1
```

### `apx semver suggest --against <ref> [path]`

Suggest semantic version bump based on changes.

```bash
# Get version suggestion
apx semver suggest --against v1.0.0

# Output: MAJOR (due to breaking changes)
```

### `apx gen <lang> [path]`

Generate code for the specified language.

```bash
# Generate Go code
apx gen go --out ./gen/go

# Clean output before generation
apx gen python --clean --out ./gen/python

# Generate with manifest
apx gen java --manifest
```

**Languages**: `go`, `python`, `java`

### `apx policy check [path]`

Check policy compliance.

```bash
# Check all modules
apx policy check

# Check specific path
apx policy check proto/payment/v1
```

### `apx catalog build`

Build a catalog of all discovered modules.

```bash
apx catalog build
# Creates catalog.json
```

### `apx publish --version <version> [path]`

Publish a module release.

```bash
# Publish with version (CI only by default)
apx publish --version v1.2.3

# Create tag only
apx publish --version v1.2.3 --tag-only

# Force local publish
apx publish --version v1.2.3 --force
```

### `apx config`

Configuration management.

```bash
# Initialize config
apx config init

# Validate config
apx config validate
```

## Module Discovery

APX automatically discovers schema modules by scanning configured `module_roots` for:

- **Protocol Buffers**: `*.proto` files
- **OpenAPI**: `openapi.{yaml,yml,json}` files
- **Avro**: `*.avsc` files
- **JSON Schema**: Files with `$schema` property
- **Parquet**: `schema.{json,ddl}` files

## Breaking Change Detection

APX uses industry-standard tools to detect breaking changes:

- **Protocol Buffers**: `buf breaking`
- **OpenAPI**: `oasdiff breaking`
- **Avro**: Compatibility mode validation
- **JSON Schema**: `jsonschema-diff`
- **Parquet**: Conservative policy checks

## Semantic Versioning

APX automatically suggests version bumps based on detected changes:

- **MAJOR**: Any breaking changes in any format
- **MINOR**: New features, additive changes
- **PATCH**: Bug fixes, documentation updates

## Policy Enforcement

Enforce organizational standards:

```yaml
policy:
  forbidden_proto_options:
    - "^gorm\\."        # Ban GORM options
    - "validate\\."     # Ban validation options
  
  allowed_proto_plugins:
    - protoc-gen-go
    - protoc-gen-go-grpc
  
  openapi:
    spectral_ruleset: ".spectral.yaml"
  
  avro:
    compatibility: "BACKWARD"
```

## Git Integration

APX creates subdirectory tags for modular repositories:

```bash
# Creates tag: proto/payment/v1/v1.2.3
apx publish --version v1.2.3 proto/payment/v1
```

Tag format is configurable via `publishing.tag_format`.

## CI/CD Integration

### GitHub Actions

```yaml
name: API Schema CI

on:
  pull_request:
    paths:
      - 'proto/**'
      - 'openapi/**'

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Setup Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.25'

      - name: Install APX
        run: go install github.com/infobloxopen/apx/cmd/apx@latest

      - name: Install Tools
        run: |
          curl -sSL https://raw.githubusercontent.com/infobloxopen/apx/main/scripts/install-tools.sh | bash

      - name: Lint Schemas
        run: apx lint --json

      - name: Check Breaking Changes
        run: apx breaking --against origin/main --json

      - name: Check Policy
        run: apx policy check --json

  publish:
    if: github.ref == 'refs/heads/main'
    needs: validate
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Setup Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.25'

      - name: Install APX
        run: go install github.com/infobloxopen/apx/cmd/apx@latest

      - name: Get Version Suggestion
        id: version
        run: |
          VERSION=$(apx semver suggest --against $(git describe --tags --abbrev=0))
          echo "version=$VERSION" >> $GITHUB_OUTPUT

      - name: Publish Release
        if: steps.version.outputs.version != 'NONE'
        run: |
          apx publish --version v$(steps.version.outputs.version)
        env:
          CI: true
```

## Exit Codes

- `0`: Success
- `2`: Lint errors
- `3`: Breaking changes detected
- `4`: Policy violations
- `5`: Tool execution failure
- `6`: Configuration error

## Development

### Building

```bash
make build
```

### Testing

```bash
make test
make test-integration
```

### Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development guidelines.

## License

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE) for details.

## Support

- 📚 [Documentation](https://github.com/infobloxopen/apx/wiki)
- 🐛 [Issues](https://github.com/infobloxopen/apx/issues)
- 💬 [Discussions](https://github.com/infobloxopen/apx/discussions)