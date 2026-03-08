# Dependency Discovery

APX provides two commands for discovering APIs published to the canonical repository: `apx search` for finding APIs by keyword or filter, and `apx show` for viewing full identity and release details of a specific API.

## `apx search`

Search the canonical repository catalog for available APIs.

### Usage

```bash
apx search [query] [flags]
```

### Examples

```bash
# List all APIs in the catalog
apx search

# Search by keyword
apx search payments
apx search ledger

# Filter by schema format
apx search --format=proto
apx search --format=openapi

# Filter by lifecycle state
apx search --lifecycle=stable
apx search --lifecycle=beta

# Filter by domain
apx search --domain=payments

# Combine keyword with filters
apx search payment --format=proto --lifecycle=stable

# Filter by API line
apx search --api-line=v2

# JSON output (pipe to jq, scripts, etc.)
apx --json search payments
```

### Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--format` | `-f` | Filter by schema format (`proto`, `openapi`, `avro`, `jsonschema`, `parquet`) |
| `--lifecycle` | `-l` | Filter by lifecycle (`experimental`, `beta`, `stable`, `deprecated`, `sunset`) |
| `--domain` | `-d` | Filter by domain (e.g. `payments`, `billing`) |
| `--api-line` | | Filter by API line (e.g. `v1`, `v2`) |
| `--catalog` | `-c` | Path to catalog file (default: `catalog/catalog.yaml`) |

### Output

For each matching API, search displays:

- **Display name** — the full API ID
- **Description** — from the catalog entry
- **Format** — schema format (proto, openapi, etc.)
- **Domain** — organizational domain
- **Line** — API version line
- **Lifecycle** — current lifecycle state
- **Version** — latest version
- **Latest stable** — latest stable release tag
- **Latest prerelease** — latest prerelease tag
- **Owners** — team or individual owners

Use `apx --json search` for machine-readable output.

### How It Works

`apx search` reads the `catalog/catalog.yaml` file in the canonical repository. This file is automatically maintained by `apx catalog generate` (run by canonical CI on merge). To search the catalog, you need either:

- A local clone of the canonical repo, or
- A fetched copy via `apx fetch`

---

## `apx show`

Display the full identity, derived coordinates, and catalog release data for a specific API.

### Usage

```bash
apx show <api-id> [flags]
```

### Examples

```bash
# Show full details for an API
apx show proto/payments/ledger/v1

# Show with explicit source repo
apx show --source-repo github.com/acme/apis proto/payments/ledger/v1

# JSON output
apx --json show proto/payments/ledger/v1
```

### Flags

| Flag | Description |
|------|-------------|
| `--source-repo` | Source repository (defaults to `github.com/<org>/<repo>` from `apx.yaml`) |
| `--catalog` | Path to `catalog.yaml` (default: `catalog/catalog.yaml`) |

### Output

`apx show` merges two data sources:

1. **Derived fields** computed from the API ID — Go module path, import path, tag pattern, source path
2. **Catalog fields** read from `catalog.yaml` — latest stable/prerelease versions, lifecycle, owners

Example output:

```
API:        proto/payments/ledger/v1
Format:     proto
Domain:     payments
Name:       ledger
Line:       v1
Lifecycle:  stable
Source:     github.com/acme/apis/proto/payments/ledger/v1

Compatibility
  Level:    full
  Promise:  Backward compatible within the v1 line
  Breaking: Not allowed (use v2 line for breaking changes)
  Use:      Recommended for production

Latest stable:      v1.2.3
Latest prerelease:  v1.3.0-beta.1

Go module:  github.com/acme/apis/proto/payments/ledger
Go import:  github.com/acme/apis/proto/payments/ledger/v1
```

If no catalog data is found, only derived fields are shown and a warning suggests running `apx catalog generate`.

---

## Planned Commands

```{admonition} Planned — not yet available
:class: note
`apx list apis` is planned for a future release. In the meantime, use `apx search` with no arguments to list all APIs in the catalog.
```

---

## Workflow

A typical discovery workflow:

```bash
# 1. Search for APIs related to your domain
apx search payments

# 2. Inspect a specific API
apx show proto/payments/ledger/v1

# 3. Add as a dependency
apx add proto/payments/ledger/v1@v1.2.3

# 4. Generate client code
apx gen go
```

## Next Steps

- [Adding Dependencies](adding-dependencies.md) — pin an API and generate code
- [Code Generation](code-generation.md) — multi-language code generation
- [Versioning Strategy](versioning-strategy.md) — understand version lines and SemVer
