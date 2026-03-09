# Dependency Commands

Commands for discovering, adding, inspecting, and removing API dependencies.

## `apx search`

Search the canonical repository catalog for APIs.

```bash
apx search [query]
```

Without a query, lists all APIs in the catalog.

### Flags

| Flag | Shorthand | Type | Default | Description |
|------|-----------|------|---------|-------------|
| `--format` | `-f` | string | `""` | Filter by schema format (proto, openapi, avro, etc.) |
| `--lifecycle` | `-l` | string | `""` | Filter by lifecycle state |
| `--domain` | `-d` | string | `""` | Filter by domain |
| `--api-line` | | string | `""` | Filter by API line (v1, v2, etc.) |
| `--origin` | | string | `""` | Filter by origin (first-party, external, forked) |
| `--tag` | | string | `""` | Filter by tag |
| `--catalog` | `-c` | string | (see below) | Path or URL to catalog file (default: `catalog_url` from `apx.yaml`, then `catalog/catalog.yaml`) |

### Examples

```bash
# Search by keyword
apx search payments

# Filter by format and lifecycle
apx search --format proto --lifecycle stable

# Filter by domain
apx search --domain billing

# JSON output
apx --json search payments

# List all APIs
apx search
```

---

## `apx show`

Display full identity and catalog data for a specific API.

```bash
apx show <api-id>
```

Merges two data sources:
1. **Derived fields** — Go module/import paths, tag pattern, source path (computed from the API ID)
2. **Catalog fields** — latest stable/prerelease versions, lifecycle, owners (from `catalog/catalog.yaml`)

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--source-repo` | string | `""` | Source repository (defaults from apx.yaml) |
| `--catalog` | string | (see search) | Path or URL to catalog file (default: `catalog_url` from `apx.yaml`, then `catalog/catalog.yaml`) |

### Example

```bash
$ apx show proto/payments/ledger/v1
API:          proto/payments/ledger/v1
Format:       proto
Domain:       payments
Name:         ledger
Line:         v1
Source:       github.com/acme-corp/apis/proto/payments/ledger/v1
Go module:    github.com/acme-corp/apis/proto/payments/ledger
Go import:    github.com/acme-corp/apis/proto/payments/ledger/v1
Tag pattern:  proto/payments/ledger/v1/v*
Lifecycle:    stable
Latest:       v1.2.3

$ apx --json show proto/payments/ledger/v1
```

---

## `apx add`

Add a schema dependency to `apx.yaml` and `apx.lock`.

```bash
apx add <module-path>[@version]
```

The `@version` suffix is optional. If omitted, the latest version is used.

### Flags

| Flag | Shorthand | Type | Default | Description |
|------|-----------|------|---------|-------------|
| `--catalog` | `-c` | string | (see search) | Path or URL to catalog file (default: `catalog_url` from `apx.yaml`, then `catalog/catalog.yaml`) |

The catalog is consulted to detect external API provenance (origin, managed repo, import mode). If the catalog cannot be loaded, the dependency is still added without provenance metadata.

### Examples

```bash
# Add at a specific version
apx add proto/payments/ledger/v1@v1.2.3

# Add latest version
apx add proto/users/profile/v1

# Add using a remote catalog
apx add proto/payments/ledger/v1 --catalog https://raw.githubusercontent.com/acme/apis/main/catalog/catalog.yaml
```

After adding, generate code:

```bash
apx gen go && apx sync
```

---

## `apx unlink`

Remove a local overlay and switch to the published module.

```bash
apx unlink <module-path>
```

### What It Does

1. Removes the dependency from `apx.lock`
2. Deletes the overlay directory from `internal/gen/`
3. Prints a hint to run `go get` to add the published module to `go.mod`

### Example

```bash
apx unlink proto/payments/ledger/v1
# → Removed overlay for proto/payments/ledger/v1
# → Run: go get github.com/acme-corp/apis/proto/payments/ledger@v1.2.3
```

Your import paths remain unchanged — they now resolve to the published module instead of the local overlay.

---

## `apx update`

Check for compatible (same API line) updates and apply them.

```bash
apx update [module-path]
```

Without arguments, checks all dependencies. With a module path, updates only
that dependency.

### Flags

| Flag | Shorthand | Type | Default | Description |
|------|-----------|------|---------|-------------|
| `--dry-run` | | bool | `false` | Preview updates without applying them |
| `--catalog` | `-c` | string | (see search) | Path or URL to catalog file (default: `catalog_url` from `apx.yaml`, then `catalog/catalog.yaml`) |

### Examples

```bash
# Check and apply all compatible updates
apx update

# Update a specific dependency
apx update proto/payments/ledger/v1

# Preview what would be updated
apx update --dry-run

# JSON output
apx --json update
```

After updating, regenerate code:

```bash
apx gen go && apx sync
```

---

## `apx upgrade`

Upgrade a dependency to a new API line (major version transition).

```bash
apx upgrade <module-path> --to <line>
```

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--to` | string | (required) | Target API line (e.g. `v2`) |
| `--dry-run` | bool | `false` | Preview upgrade without applying |
| `--catalog` | string | (see search) | Path or URL to catalog file (default: `catalog_url` from `apx.yaml`, then `catalog/catalog.yaml`) |

### Examples

```bash
# Upgrade from v1 to v2
apx upgrade proto/payments/ledger/v1 --to v2

# Preview the upgrade plan
apx upgrade proto/payments/ledger/v1 --to v2 --dry-run

# JSON output for CI
apx --json upgrade proto/payments/ledger/v1 --to v2 --dry-run
```

After upgrading:

1. Regenerate code: `apx gen go && apx sync`
2. Update import paths in your code (the command prints the mapping)
3. Run `apx breaking` to inspect breaking changes

---

## Workflow

```bash
# 1. Discover available APIs
apx search payments

# 2. Inspect details
apx show proto/payments/ledger/v1

# 3. Add as dependency
apx add proto/payments/ledger/v1@v1.2.3

# 4. Generate client code
apx gen go && apx sync

# 5. Use canonical imports in your code
# import ledgerv1 "github.com/acme-corp/apis/proto/payments/ledger/v1"

# 6. When ready to consume published module
apx unlink proto/payments/ledger/v1
go get github.com/acme-corp/apis/proto/payments/ledger@v1.2.3
```

## See Also

- [Adding Dependencies](../dependencies/adding-dependencies.md) — detailed guide
- [Discovery](../dependencies/discovery.md) — search and show details
- [Code Generation](../dependencies/code-generation.md) — generating client code
