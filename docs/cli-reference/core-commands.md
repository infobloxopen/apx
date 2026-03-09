# Core Commands

APX provides commands for managing API schemas across their full lifecycle.

## `apx init`

Initialize a new APX project.

```bash
apx init [kind] [modulePath]   # Interactive or direct module init
apx init canonical             # Initialize canonical API repository
apx init app <modulePath>      # Initialize application repository
```

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

## `apx explain`

Explain how APX derives language-specific paths.

```bash
apx explain go-path <api-id>    # Explain Go module/import path derivation
```

## `apx publish`

Publish an API module. See [Publishing Commands](publishing-commands.md).

## `apx search`

Search the API catalog. Supports local file paths and remote URLs.

```bash
apx search <query>
apx search --tag=public --lifecycle=stable
apx search --catalog=https://raw.githubusercontent.com/org/apis/main/catalog/catalog.yaml
```

## `apx show`

Display full identity and catalog data for a given API.

```bash
apx show <api-id>                                           # Show API with config-derived source repo
apx show --source-repo github.com/acme/apis <id>            # Override source repo
apx show --catalog path/to/catalog.yaml <id>                # Custom catalog path
apx show --catalog https://example.com/catalog.yaml <id>    # Remote catalog URL
apx --json show <api-id>                                    # JSON output
```

This command merges two data sources:

1. **Derived fields** computed from the API ID — Go module/import paths, tag pattern, source path.
   When `import_root` is set, Go paths use the custom root.
2. **Catalog fields** from `catalog.yaml` — latest stable/prerelease versions, lifecycle, owners, tags

The catalog source is resolved in order: `--catalog` flag → `catalog_url` from `apx.yaml` → `catalog/catalog.yaml`.
If no catalog is available, only derived fields are shown, with a note to run `apx catalog generate`.

Example:

```bash
$ apx show proto/payments/ledger/v1
API:        proto/payments/ledger/v1
Format:     proto
Domain:     payments
Name:       ledger
Line:       v1
Lifecycle:  stable
Source:     github.com/acme/apis/proto/payments/ledger/v1
Latest stable:      v1.2.3
Latest prerelease:  v1.3.0-beta.1
Go module:  github.com/acme/apis/proto/payments/ledger
Go import:  github.com/acme/apis/proto/payments/ledger/v1
Owners:     @platform/payments
Compatibility:
  Level:    full
  Promise:  full backward compatibility within the major version line
  Breaking: backward-incompatible changes are blocked on this line
  Use:      recommended for production
```

For a v0 API the lifecycle section shows the reduced compatibility guarantee:

```bash
$ apx show proto/payments/ledger/v0
API:        proto/payments/ledger/v0
Format:     proto
Domain:     payments
Name:       ledger
Line:       v0
Lifecycle:  experimental
Source:     github.com/acme/apis/proto/payments/ledger/v0
Latest stable:      (none)
Latest prerelease:  0.3.0
Go module:  github.com/acme/apis/proto/payments/ledger
Go import:  github.com/acme/apis/proto/payments/ledger/v0
Compatibility:
  Level:    none
  Promise:  no backward-compatibility guarantee; anything may change
  Breaking: breaking changes are allowed (minor version bump)
  Use:      not recommended for production
```

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
`apx publish`, and `apx release` all use the custom root:

```
Go module:  go.acme.dev/apis/proto/payments/ledger
Go import:  go.acme.dev/apis/proto/payments/ledger/v1
```

When `import_root` is omitted or empty, the source repository path is used
as before, preserving full backward compatibility.
