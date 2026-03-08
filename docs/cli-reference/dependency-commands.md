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
| `--catalog` | `-c` | string | `catalog/catalog.yaml` | Path to catalog file |

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
| `--catalog` | string | `catalog/catalog.yaml` | Path to catalog.yaml |

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

### Examples

```bash
# Add at a specific version
apx add proto/payments/ledger/v1@v1.2.3

# Add latest version
apx add proto/users/profile/v1
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
