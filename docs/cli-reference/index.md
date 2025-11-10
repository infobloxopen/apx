# CLI Reference

Complete reference for all APX commands and options.

```{toctree}
:maxdepth: 2

core-commands
dependency-commands  
publishing-commands
validation-commands
utility-commands
global-options
```

## Command Categories

APX commands are organized into logical categories:

::::{grid} 1 1 2 3
:gutter: 3

:::{grid-item-card} **Core Commands**
^^^
- `apx init` - Initialize projects
- `apx fetch` - Download toolchain  
- `apx gen` - Generate code with canonical imports
- `apx sync` - Update go.work overlays
- `apx unlink` - Remove overlays for published APIs
:::

:::{grid-item-card} **Dependencies**
^^^
- `apx search` - Discover APIs
- `apx add` - Add dependencies
- `apx update` - Update versions
:::

:::{grid-item-card} **Publishing**
^^^
- `apx publish` - Publish to canonical
- `apx tag` - Create tags
- `apx version` - Version management
:::

:::{grid-item-card} **Validation**
^^^  
- `apx lint` - Schema validation
- `apx breaking` - Breaking changes
- `apx policy` - Policy enforcement
:::

:::{grid-item-card} **Utilities**
^^^
- `apx list` - List APIs/versions
- `apx show` - Show API details
- `apx config` - Configuration
:::

:::{grid-item-card} **Global Options**
^^^
- `--use-container` - Containerized execution
- `--verbose` - Detailed output
- `--config` - Custom config file
:::

::::

## Quick Reference

### Most Common Commands

```bash
# Initialize new project (interactive)
apx init

# Add dependency
apx add proto/payments/ledger/v1@v1.2.3

# Validate schemas
apx lint && apx breaking

# Generate code with canonical import paths
apx gen go       # generates stubs with canonical imports
apx sync         # updates go.work overlays

# Switch from overlay to published module
apx unlink proto/payments/ledger/v1
go get github.com/myorg/apis-go/proto/payments/ledger@v1.2.3

# Publish from app repo
apx publish --module-path=internal/apis/proto/domain/api/v1 \
           --canonical-repo=github.com/org/apis
```

**Application code using canonical imports:**

```go
// service.go - your application
package service

import (
    // Pattern: github.com/<org>/apis-go/proto/<domain>/<api>/v1
    ledgerv1 "github.com/myorg/apis-go/proto/payments/ledger/v1"  
    usersv1 "github.com/myorg/apis-go/proto/users/profile/v1"
    productsv2 "github.com/myorg/apis-go/proto/inventory/products/v2"
)

type Service struct {
    ledgerClient   ledgerv1.LedgerServiceClient
    usersClient    usersv1.ProfileServiceClient
    productsClient productsv2.ProductServiceClient
}
```

## Path Mapping Reference

**APX API Path → Go Import Path**

| APX API Path | Go Module Path | Go Import Path |
|--------------|----------------|----------------|
| `proto/payments/ledger/v1` | `github.com/<org>/apis-go/proto/payments/ledger` | `github.com/<org>/apis-go/proto/payments/ledger/v1` |
| `proto/users/profile/v1` | `github.com/<org>/apis-go/proto/users/profile` | `github.com/<org>/apis-go/proto/users/profile/v1` |
| `proto/inventory/products/v2` | `github.com/<org>/apis-go/proto/inventory/products/v2` | `github.com/<org>/apis-go/proto/inventory/products/v2` |
| `proto/billing/invoices/v1` | `github.com/<org>/apis-go/proto/billing/invoices` | `github.com/<org>/apis-go/proto/billing/invoices/v1` |

**Local Overlay Paths**

| APX API Path | Local Generated Path | Overlay in go.work |
|--------------|---------------------|-------------------|
| `proto/payments/ledger/v1@v1.2.3` | `internal/gen/go/proto/payments/ledger@v1.2.3/` | `use ./internal/gen/go/proto/payments/ledger@v1.2.3` |
| `proto/users/profile/v1@v1.0.1` | `internal/gen/go/proto/users/profile@v1.0.1/` | `use ./internal/gen/go/proto/users/profile@v1.0.1` |

### Discovery & Search

```bash
# Search for APIs
apx search payments ledger

# List all APIs
apx list apis

# Show API details
apx show proto/payments/ledger/v1

# List available versions
apx list versions proto/payments/ledger
```

### Version Management

```bash
# Get version suggestion
apx version suggest

# Set version manually
apx version set v1.2.3

# Verify version matches changes
apx version verify
```

## Exit Codes

APX uses consistent exit codes:

- **0**: Success
- **1**: General error (validation failed, command error)
- **2**: Configuration error (invalid config, missing files)
- **3**: Network error (cannot fetch dependencies, API unreachable)
- **4**: Permission error (cannot write files, access denied)
- **5**: Breaking change detected (when used with `--fail-on-breaking`)

## Environment Variables

### APX_CONFIG
Override default configuration file location:
```bash
export APX_CONFIG=/path/to/custom/apx.yaml
apx lint  # uses custom config
```

### APX_CACHE_DIR
Set custom cache directory for downloaded tools:
```bash
export APX_CACHE_DIR=/tmp/apx-cache
apx fetch  # downloads to custom location
```

### APX_USE_CONTAINER
Force container execution:
```bash
export APX_USE_CONTAINER=true
apx gen go  # runs generators in container
```

### APX_VERBOSE
Enable verbose output by default:
```bash
export APX_VERBOSE=true
apx lint  # shows detailed output
```

## Configuration File

APX uses `apx.yaml` for configuration:

```yaml
# Project configuration
project:
  org: myorg
  name: myproject

# API definitions
apis:
  - kind: proto
    path: internal/apis/proto/domain/service/v1
    canonical: proto/domain/service/v1

# Code generation settings
codegen:
  out: internal/gen
  languages: [go, python, java]
  options:
    go:
      canonical_imports: true  # generates with canonical import paths
      workspace_overlay: true  # manages go.work overlays
    python:
      package_name: myproject_apis

# Validation settings  
validation:
  breaking:
    ignore_patterns:
      - "*.internal.*"  # ignore internal packages
  policy:
    banned_annotations:
      - "gorm.*"
      - "database.*"

# Publishing settings
publishing:
  canonical_repo: github.com/myorg/apis
  strategy: subtree
  
# Tool versions (usually in apx.lock)
toolchain:
  buf: v1.28.1
  protoc: v24.4
  protoc-gen-go: v1.31.0
```

## Shell Completion

Enable shell completion for better CLI experience:

### Bash
```bash
# Add to ~/.bashrc
source <(apx completion bash)

# Or install system-wide  
apx completion bash | sudo tee /etc/bash_completion.d/apx
```

### Zsh
```bash
# Add to ~/.zshrc
source <(apx completion zsh)

# Or for oh-my-zsh
apx completion zsh > "${fpath[1]}/_apx"
```

### Fish
```bash
apx completion fish | source

# Or install permanently
apx completion fish > ~/.config/fish/completions/apx.fish
```

## Next Steps

- [Learn core commands](core-commands.md) for project setup
- [Master dependency commands](dependency-commands.md) for API management  
- [Understand publishing commands](publishing-commands.md) for releases
- [Use validation commands](validation-commands.md) for quality assurance
- [Explore utility commands](utility-commands.md) for daily workflows