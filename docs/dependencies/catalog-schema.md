# Catalog Schema

The `catalog/catalog.yaml` file is the organization-wide index of released API schemas. It is generated automatically by `apx catalog generate` (run by canonical CI on every merge) and consumed by `apx search`, `apx show`, `apx add`, `apx update`, and `apx upgrade`.

## Catalog File Structure

```yaml
version: 1
org: acme
repo: apis
modules:
  - id: proto/payments/ledger/v1
    format: proto
    domain: payments
    api_line: v1
    description: Payments ledger service API
    lifecycle: stable
    version: v1.2.3
    latest_stable: v1.2.3
    latest_prerelease: v1.3.0-beta.1
    path: proto/payments/ledger/v1
    tags: [payments, internal]
    owners: [payments-team]
```

## Top-Level Fields

| Field | Type | Description |
|-------|------|-------------|
| `version` | integer | Catalog schema version (always `1`) |
| `org` | string | GitHub organization name |
| `repo` | string | Canonical API repository name |
| `modules` | list | List of API module entries |

## Module Entry Fields

Each entry in `modules` describes a single released API line.

### Identity

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | yes | Canonical API ID: `<format>/<domain>/<name>/<line>` |
| `format` | string | yes | Schema format: `proto`, `openapi`, `avro`, `jsonschema`, `parquet` |
| `domain` | string | no | Business domain (e.g. `payments`, `billing`) |
| `api_line` | string | no | API compatibility line (e.g. `v1`, `v2`) |
| `path` | string | yes | Filesystem path in the canonical repository |
| `description` | string | no | Human-readable description |

### Versions

| Field | Type | Description |
|-------|------|-------------|
| `version` | string | Current/latest version of this API line |
| `latest_stable` | string | Latest stable release tag (no prerelease suffix) |
| `latest_prerelease` | string | Latest prerelease tag (`-alpha.*`, `-beta.*`, `-rc.*`) |

`apx update` uses `latest_stable` first, falling back to `latest_prerelease`, then `version`.

### Lifecycle and Compatibility

| Field | Type | Allowed Values | Description |
|-------|------|----------------|-------------|
| `lifecycle` | string | `experimental`, `beta`, `stable`, `deprecated`, `sunset` | Maturity/support state |
| `compatibility` | string | `none`, `stabilizing`, `full`, `maintenance`, `eol` | Derived compatibility signal |
| `production_use` | string | | Human-readable production recommendation |

### Metadata

| Field | Type | Description |
|-------|------|-------------|
| `tags` | list of strings | Searchable tags (e.g. `["payments", "public"]`) |
| `owners` | list of strings | Team or individual owners (e.g. `["payments-team"]`) |

### External API Provenance

These fields are populated only for external and forked APIs (registered via `external_apis` in `apx.yaml`). First-party APIs leave them empty.

| Field | Type | Allowed Values | Description |
|-------|------|----------------|-------------|
| `origin` | string | `external`, `forked` | Classification of the API source |
| `managed_repo` | string | | Internal repository hosting curated snapshots |
| `upstream_repo` | string | | Original external repository URL |
| `upstream_path` | string | | Path within the upstream repository |
| `import_mode` | string | `preserve`, `rewrite` | Import path handling strategy |

## How the Catalog Is Generated

`apx catalog generate` (or `apx catalog generate --from-tags`) scans the canonical repository and builds `catalog.yaml` from:

1. **Git tags** — tags matching `<format>/<domain>/<name>/<line>/v<semver>` are parsed to populate `version`, `latest_stable`, and `latest_prerelease`
2. **External API registrations** — `external_apis` entries in `apx.yaml` are merged in to add provenance fields

The canonical CI workflow runs `apx catalog generate` on every merge to keep the catalog current.

```bash
# Regenerate from git tags (canonical CI)
apx catalog generate --from-tags

# Regenerate from directory scan
apx catalog generate
```

## Configuring Remote Catalog Access

App repos and CI pipelines that don't clone the canonical repository can point at a hosted catalog via `catalog_url` in `apx.yaml`:

```yaml
catalog_url: https://raw.githubusercontent.com/acme/apis/main/catalog/catalog.yaml
```

All five consumer commands (`search`, `show`, `add`, `update`, `upgrade`) check `catalog_url` automatically when `--catalog` is not provided. See [Configuration Reference](../cli-reference/configuration.md#catalog_url).

## See Also

- [Dependency Discovery](discovery.md) — search and show
- [Adding Dependencies](adding-dependencies.md) — `apx add`
- [Updates and Upgrades](updates-and-upgrades.md) — `apx update` / `apx upgrade`
- [External APIs](external-apis.md) — registering third-party APIs
