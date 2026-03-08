# Utility Commands

Commands for configuration management, toolchain setup, identity inspection, and workflow management.

## `apx config`

Manage the APX configuration file.

### `apx config init`

Create a default `apx.yaml` configuration file.

```bash
apx config init
```

Writes a default configuration using the canonical schema. Fails if `apx.yaml` already exists.

### `apx config validate`

Validate the configuration file against the APX schema.

```bash
apx config validate
apx config validate --config custom-apx.yaml
apx --json config validate
```

Returns a `ValidationResult` with errors and warnings. Exit code `6` indicates an invalid configuration.

**Example output:**

```
✔ Configuration is valid
  Warnings:
    - field "languages.python": language target not yet supported
```

### `apx config migrate`

Migrate a configuration file to the current schema version.

```bash
apx config migrate
apx --json config migrate
```

Creates a backup of the original file before migrating. Reports the from/to version and changes applied.

---

## `apx fetch`

Download and cache toolchain dependencies for offline use.

```bash
apx fetch
```

### Flags

| Flag | Shorthand | Type | Default | Description |
|------|-----------|------|---------|-------------|
| `--config` | `-c` | string | `apx.yaml` | Path to configuration file |
| `--output` | | string | `.apx-tools` | Output directory for cached tools |
| `--verify` | | bool | `true` | Verify checksums after download |

### What It Downloads

Tools are resolved from `apx.lock` and cached in `.apx-tools/`:

```
.apx-tools/
├── buf                    # Buf CLI
├── protoc-gen-go          # Go protobuf plugin
└── protoc-gen-go-grpc     # Go gRPC plugin
```

### Example

```bash
# Download all pinned tools
apx fetch

# Verify checksums match apx.lock
apx fetch --verify

# Custom output directory
apx fetch --output /tmp/apx-tools
```

---

## `apx sync`

Synchronize `go.work` overlays with generated code directories.

```bash
apx sync
```

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--clean` | bool | `false` | Remove all overlays before syncing |
| `--dry-run` | bool | `false` | Show what would change without modifying files |

### What It Does

Scans `internal/gen/go/` for overlay directories and updates `go.work` with a `use` directive for each one:

```
go 1.22
use .
use ./internal/gen/go/proto/payments/ledger@v1.2.3
use ./internal/gen/go/proto/users/profile@v1.0.1
```

### Examples

```bash
# Sync after generating code
apx gen go && apx sync

# Preview changes
apx sync --dry-run

# Clean stale overlays and resync
apx sync --clean
```

---

## `apx inspect`

Inspect API identity, releases, and derived coordinates.

### `apx inspect identity`

Show the full canonical identity for an API.

```bash
apx inspect identity <api-id>
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--source-repo` | string | from apx.yaml | Source repository |
| `--lifecycle` | string | `""` | Lifecycle state |

**Example:**

```bash
$ apx inspect identity proto/payments/ledger/v1
API:        proto/payments/ledger/v1
Format:     proto
Domain:     payments
Name:       ledger
Line:       v1
Source:     github.com/acme-corp/apis/proto/payments/ledger/v1
Go module:  github.com/acme-corp/apis/proto/payments/ledger
Go import:  github.com/acme-corp/apis/proto/payments/ledger/v1
```

### `apx inspect release`

Show identity for a specific API release.

```bash
apx inspect release <api-id>@<version>
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--source-repo` | string | from apx.yaml | Source repository |

Lifecycle is inferred from the version's prerelease tag (alpha → experimental, beta/rc → beta, none → stable).

---

## `apx explain`

Explain how APX derives language-specific paths and coordinates.

### `apx explain go-path`

Explain Go module and import path derivation rules.

```bash
apx explain go-path <api-id>
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--source-repo` | string | from apx.yaml | Source repository |

Shows the derivation rules including Go's v2+ major version suffix convention.

**Example:**

```bash
$ apx explain go-path proto/payments/ledger/v2
API ID:     proto/payments/ledger/v2
Go module:  github.com/acme-corp/apis/proto/payments/ledger/v2
Go import:  github.com/acme-corp/apis/proto/payments/ledger/v2

Derivation:
  v1 modules omit the version suffix: .../proto/payments/ledger
  v2+ modules include the suffix:     .../proto/payments/ledger/v2

Usage:
  go.mod:   require github.com/acme-corp/apis/proto/payments/ledger/v2 v2.0.0
  import:   import ledgerv2 "github.com/acme-corp/apis/proto/payments/ledger/v2"
```

---

## `apx workflows`

Manage GitHub Actions workflow files.

### `apx workflows sync`

Regenerate workflow files from the latest APX templates.

```bash
apx workflows sync
apx workflows sync --dry-run
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--dry-run` | bool | `false` | Show what would be written without modifying files |

Detects the repository type automatically:

| Repository type | Detection method | Generated files |
|----------------|-----------------|----------------|
| Canonical | `ci.yml` or `on-merge.yml` exists, or top-level `proto/`/`catalog/` dirs | `ci.yml`, `on-merge.yml` |
| App | `apx-publish.yml` exists, or `module_roots` in config | `apx-publish.yml` |

Reads `org` and `repo` from `apx.yaml` (or falls back to git remote detection).

---

## `apx catalog`

Manage the API catalog in the canonical repository.

### `apx catalog generate`

Generate `catalog.yaml` from git tags in the canonical repo.

```bash
apx catalog generate
```

| Flag | Shorthand | Type | Default | Description |
|------|-----------|------|---------|-------------|
| `--output` | `-o` | string | `catalog/catalog.yaml` | Output path |
| `--org` | | string | from apx.yaml | Organization name |
| `--repo` | | string | from apx.yaml | Repository name |
| `--dir` | | string | `.` | Git repository directory to scan |

Scans git tags matching `<format>/<domain>/<name>/<line>/v<semver>` and generates a structured catalog. Typically run by `on-merge.yml` in the canonical repo.

### `apx catalog build`

Build the module catalog (stub — reserved for future use).

```bash
apx catalog build
```

## See Also

- [Global Options](global-options.md) — flags available on every command
- [Core Commands](core-commands.md) — init, lint, breaking, inspect
- [CI Templates](../canonical-repo/ci-templates.md) — how workflows are used
