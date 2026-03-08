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

```bash
apx inspect identity <api-id>              # Show full API identity
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
```

## `apx explain`

Explain how APX derives language-specific paths.

```bash
apx explain go-path <api-id>    # Explain Go module/import path derivation
```

## `apx publish`

Publish an API module. See [Publishing Commands](publishing-commands.md).

## `apx search`

Search the API catalog.

```bash
apx search <query>
```

## `apx show`

Display full identity and catalog data for a given API.

```bash
apx show <api-id>                                # Show API with config-derived source repo
apx show --source-repo github.com/acme/apis <id> # Override source repo
apx show --catalog path/to/catalog.yaml <id>     # Custom catalog path
apx --json show <api-id>                         # JSON output
```

This command merges two data sources:

1. **Derived fields** computed from the API ID — Go module/import paths, tag pattern, source path
2. **Catalog fields** from `catalog/catalog.yaml` — latest stable/prerelease versions, lifecycle, owners

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
```

## `apx config`

Manage the APX configuration file.

```bash
apx config validate    # Validate apx.yaml against schema
apx config migrate     # Migrate to latest schema version
```
