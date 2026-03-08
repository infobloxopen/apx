# External API Registration

APX supports registering third-party APIs alongside your first-party APIs. External APIs participate in catalog generation, dependency resolution, search, and the full identity system — they are treated as first-class citizens.

## Overview

External API registration enables organizations to:

- **Curate** third-party APIs in a managed repository with consistent structure
- **Track provenance** — where the API came from and how imports are handled
- **Discover** external APIs alongside first-party APIs via `apx search`
- **Depend** on external APIs with full provenance in lock files
- **Transition** between "external" (preserved imports) and "forked" (rewritten imports)

## Key Concepts

### Origin Classification

Every registered external API has an **origin**:

| Origin | Import Mode | Description |
|--------|------------|-------------|
| `external` | `preserve` | Third-party API with original import paths preserved |
| `forked` | `rewrite` | Third-party API with import paths rewritten to your organization |

### Managed vs Upstream

External APIs reference two repositories:

- **Managed repo**: Your organization's curated copy (e.g., `github.com/acme/apis-contrib-google`)
- **Upstream repo**: The original source (e.g., `github.com/googleapis/googleapis`)

The managed path in your repo may differ from the upstream path. APX tracks both for provenance.

## Workflows

### Register an External API

```bash
apx external register proto/google/pubsub/v1 \
  --managed-repo github.com/acme/apis-contrib-google \
  --managed-path google/pubsub/v1 \
  --upstream-repo github.com/googleapis/googleapis \
  --upstream-path google/pubsub/v1 \
  --description "Google Cloud Pub/Sub API" \
  --lifecycle stable \
  --version v1.0.0
```

This adds the registration to `apx.yaml` under `external_apis`. Required flags: `--managed-repo`, `--managed-path`, `--upstream-repo`, `--upstream-path`.

### List Registered APIs

```bash
# List all external APIs
apx external list

# Filter by origin
apx external list --origin external
apx external list --origin forked
```

### Discover External APIs

External APIs appear in search results with an origin tag:

```bash
# Search all APIs
apx search

# Filter to external APIs only
apx search --origin external

# Filter to first-party APIs only
apx search --origin first-party
```

### Inspect Provenance

Use `apx show` or `apx inspect identity` to view full provenance:

```bash
apx show proto/google/pubsub/v1
# Displays Provenance section with origin, import mode,
# managed repo/path, and upstream repo/path

apx inspect identity proto/google/pubsub/v1
# Displays Origin, Import, Managed, and Upstream lines
```

### Add as a Dependency

External APIs can be added as dependencies just like first-party APIs:

```bash
apx add proto/google/pubsub/v1@v1.0.0
```

The lock file records provenance fields (`origin`, `upstream_repo`, `upstream_path`, `import_mode`) so downstream consumers know the dependency's source.

### Transition Between External and Forked

When you need to modify import paths (e.g., forking a third-party API):

```bash
# Transition from external to forked
apx external transition proto/google/pubsub/v1 --to forked
# Import mode changes: preserve → rewrite

# Transition back to external
apx external transition proto/google/pubsub/v1 --to external
# Import mode changes: rewrite → preserve
```

Upstream provenance is always retained for traceability.

## Catalog Integration

When you run `apx catalog generate`, external APIs registered in `apx.yaml` are automatically merged into the catalog alongside first-party modules. The output reports the breakdown:

```
✓ Catalog generated: 15 modules (12 first-party, 3 external)
```

External modules in the catalog include all provenance fields (`origin`, `managed_repo`, `upstream_repo`, `upstream_path`, `import_mode`), making them fully inspectable.

### Version Tracking

External APIs support the same versioning model as first-party APIs:

- Stable versions (e.g., `v1.0.0`) populate `latest_stable`
- Prerelease versions (e.g., `v1.1.0-beta.1`) populate `latest_prerelease`
- Lifecycle states (`experimental`, `beta`, `stable`, `deprecated`, `sunset`) work identically

## Configuration

External APIs are stored in `apx.yaml` under the `external_apis` key:

```yaml
external_apis:
  - id: proto/google/pubsub/v1
    managed_repo: github.com/acme/apis-contrib-google
    managed_path: google/pubsub/v1
    upstream_repo: github.com/googleapis/googleapis
    upstream_path: google/pubsub/v1
    import_mode: preserve
    origin: external
    description: Google Cloud Pub/Sub API
    lifecycle: stable
    version: v1.0.0
```

### Supported Fields

| Field | Required | Description |
|-------|----------|-------------|
| `id` | Yes | Canonical API identity (`format/domain/name/line`) |
| `managed_repo` | Yes | Internal repository hosting the curated copy |
| `managed_path` | Yes | Path within the managed repository |
| `upstream_repo` | Yes | Original external repository URL |
| `upstream_path` | Yes | Path in the upstream repository |
| `import_mode` | No | `preserve` (default) or `rewrite` |
| `origin` | No | `external` (default) or `forked` |
| `description` | No | Human-readable description |
| `lifecycle` | No | Lifecycle state |
| `version` | No | Current version of the managed snapshot |
| `owners` | No | List of owners |
| `tags` | No | List of tags |

## Error Handling

| Error | Cause | Resolution |
|-------|-------|------------|
| Duplicate ID | API ID already registered | Remove existing registration first |
| Path conflict | Managed path conflicts with another module | Use a different managed path |
| Invalid API ID | ID doesn't match `format/domain/name/line` | Fix the API ID format |
| Not found | API ID not in external registrations | Check the ID with `apx external list` |
| Already at target | Transition to current classification | No action needed |
