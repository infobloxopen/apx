# Core Commands

APX provides commands for managing API schemas across their full lifecycle.

## `apx init`

Initialize a new APX project.

```bash
apx init [kind] [modulePath]   # Interactive or direct module init
apx init canonical             # Initialize canonical API repository
apx init app <modulePath>      # Initialize application repository
```

### Common Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--non-interactive` | bool | false | Disable interactive prompts and use defaults |
| `--org` | string | auto-detected | Organization name |
| `--repo` | string | auto-detected | Repository name |
| `--languages` | []string | `[go]` | Target languages (auto-detected from project files) |

### `apx init canonical` Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--org` | string | | Organization name |
| `--repo` | string | | Repository name |
| `--import-root` | string | | Custom public Go import prefix (e.g. `go.acme.dev/apis`) |
| `--skip-git` | bool | false | Skip git initialization |
| `--setup-github` | bool | false | Configure GitHub repo settings (branch/tag protection, org secrets) via `gh` CLI |
| `--app-id` | string | | GitHub App ID for org secrets (used with `--setup-github`) |
| `--app-pem-file` | string | | Path to GitHub App private key PEM file (used with `--setup-github`) |

### `apx init app` Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--org` | string | | Organization name |
| `--repo` | string | | Repository name |
| `--import-root` | string | | Custom public Go import prefix (e.g. `go.acme.dev/apis`) |
| `--setup-github` | bool | false | Configure GitHub repo settings (branch protection) via `gh` CLI |

## `apx gen`

Generate code for the specified language.

```bash
apx gen <lang> [path]
```

Supported languages are listed dynamically; run `apx gen --help` to see them.

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--out` | string | | Output directory |
| `--clean` | bool | false | Clean output directory before generation |
| `--manifest` | bool | false | Emit generation manifest |

## `apx lint`

Run linting checks on schema files.

```bash
apx lint <module-path>
```

## `apx breaking`

Check for breaking changes between schema versions.

```bash
apx breaking <module-path>
```

## `apx inspect`

Inspect API identity, releases, and derived coordinates.
When a catalog is available (local or remote via `catalog_url`), `inspect identity`
also shows latest versions, owners, and tags.

If `import_root` is set in `apx.yaml`, Go module and import paths use the
custom root instead of the source repository path.

```bash
apx inspect identity <api-id>              # Show full API identity + catalog data
apx inspect release <api-id>@<version>     # Show identity for a release
```

Example:

```bash
$ apx inspect identity proto/payments/ledger/v1
API:        proto/payments/ledger/v1
Format:     proto
Domain:     payments
Name:       ledger
Line:       v1
Source:     github.com/acme/apis/proto/payments/ledger/v1
Go module:  github.com/acme/apis/proto/payments/ledger
Go import:  github.com/acme/apis/proto/payments/ledger/v1
Latest stable:      v1.2.3
Latest prerelease:  v1.3.0-beta.1
Owners:     @platform/payments
Tags:       public, core
```

## `apx search`, `apx show`

See [Dependency Commands](dependency-commands.md) for full flag tables and examples.

## `apx explain`

See [Utility Commands](utility-commands.md#apx-explain) for full flag tables and examples.

## `apx config`

Manage the APX configuration file.

```bash
apx config validate    # Validate apx.yaml against schema
apx config migrate     # Migrate to latest schema version
```

## Configuration: `import_root`

By default, Go module and import paths are derived from the source repository
(e.g. `github.com/<org>/<repo>`). Set `import_root` in `apx.yaml` to decouple
the public Go import path from the hosting location:

```yaml
# apx.yaml
org: acme
repo: apis
import_root: go.acme.dev/apis   # custom Go module prefix
```

With this configuration, `apx inspect`, `apx show`, `apx explain go-path`,
and `apx release` all use the custom root:

```
Go module:  go.acme.dev/apis/proto/payments/ledger
Go import:  go.acme.dev/apis/proto/payments/ledger/v1
```

When `import_root` is omitted or empty, the source repository path is used
as before, preserving full backward compatibility.
