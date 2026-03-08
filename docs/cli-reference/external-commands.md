# External Commands

Commands for managing external API registrations.

## `apx external register`

Register an external API in the organization's APX catalog.

### Usage

```
apx external register <api-id> [flags]
```

### Arguments

| Argument | Description |
|----------|-------------|
| `api-id` | Canonical API identity (`format/domain/name/line`) |

### Flags

| Flag | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `--managed-repo` | string | Yes | | Internal repository hosting curated snapshots |
| `--managed-path` | string | Yes | | Filesystem path in managed repository |
| `--upstream-repo` | string | Yes | | Original external repository URL |
| `--upstream-path` | string | Yes | | Path in upstream repository |
| `--import-mode` | string | No | `preserve` | Import path handling: `preserve` or `rewrite` |
| `--description` | string | No | | Human-readable description |
| `--lifecycle` | string | No | | Lifecycle state (`experimental`, `beta`, `stable`, `deprecated`, `sunset`) |
| `--version` | string | No | | Current version of the managed snapshot |
| `--owners` | string | No | | Comma-separated list of owners |
| `--tags` | string | No | | Comma-separated list of tags |

### Examples

```bash
# Register a Google API
apx external register proto/google/pubsub/v1 \
  --managed-repo github.com/acme/apis-contrib-google \
  --managed-path google/pubsub/v1 \
  --upstream-repo github.com/googleapis/googleapis \
  --upstream-path google/pubsub/v1 \
  --description "Google Cloud Pub/Sub API" \
  --lifecycle stable \
  --version v1.0.0

# Register with minimal flags
apx external register proto/vendor/auth/v1 \
  --managed-repo github.com/acme/apis-contrib \
  --managed-path vendor/auth/v1 \
  --upstream-repo github.com/vendor/api-protos \
  --upstream-path auth/v1
```

### Output

```
✓ Registered external API: proto/google/pubsub/v1
  Managed:  github.com/acme/apis-contrib-google :: google/pubsub/v1
  Upstream: github.com/googleapis/googleapis :: google/pubsub/v1
  Import:   preserve
  Origin:   external
```

---

## `apx external list`

List all registered external APIs.

### Usage

```
apx external list [flags]
```

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--origin` | string | (all) | Filter by origin: `external` or `forked` |

### Examples

```bash
# List all external APIs
apx external list

# List only forked APIs
apx external list --origin forked
```

### Output

```
External APIs (2 registered):

  proto/google/pubsub/v1              [external] preserve
    Managed:  github.com/acme/apis-contrib-google :: google/pubsub/v1
    Upstream: github.com/googleapis/googleapis :: google/pubsub/v1

  proto/google/api/v1                 [forked] rewrite
    Managed:  github.com/acme/apis-contrib-google :: google/api
    Upstream: github.com/googleapis/googleapis :: google/api
```

---

## `apx external transition`

Transition an external API between `external` and `forked` classification.

### Usage

```
apx external transition <api-id> --to <external|forked> [flags]
```

### Arguments

| Argument | Description |
|----------|-------------|
| `api-id` | API identity to transition |

### Flags

| Flag | Type | Required | Description |
|------|------|----------|-------------|
| `--to` | string | Yes | Target classification: `external` or `forked` |

### Behavior

| Transition | Origin Change | Import Mode Change |
|-----------|---------------|-------------------|
| `external` → `forked` | `external` → `forked` | `preserve` → `rewrite` |
| `forked` → `external` | `forked` → `external` | `rewrite` → `preserve` |

Upstream provenance is always retained.

### Examples

```bash
# Transition to forked (rewrite imports)
apx external transition proto/google/pubsub/v1 --to forked

# Transition back to external (preserve imports)
apx external transition proto/google/pubsub/v1 --to external
```

### Output

```
✓ Transitioned proto/google/pubsub/v1: external → forked
  Import mode changed: preserve → rewrite
  Upstream origin retained for provenance.
```

### Errors

| Error | Cause |
|-------|-------|
| API not found | API ID not in external registrations |
| Not external | Attempted to transition a first-party API |
| Already at target | API is already at the requested classification |

---

## Modified Commands

The following existing commands are enhanced when external APIs are registered:

### `apx search --origin`

Filter search results by API origin.

```bash
apx search --origin external    # External APIs only
apx search --origin forked      # Forked APIs only
apx search --origin first-party # First-party APIs only
```

External APIs display an `[external]` or `[forked]` tag in output, plus `Managed:` and `Import:` lines.

### `apx show` (Provenance section)

When showing an external API, a **Provenance** section appears:

```
Provenance
  Origin:         external
  Import mode:    preserve
  Managed repo:   github.com/acme/apis-contrib-google
  Managed path:   google/pubsub/v1
  Upstream repo:  github.com/googleapis/googleapis
  Upstream path:  google/pubsub/v1
```

### `apx inspect identity` (provenance lines)

External APIs include additional provenance lines:

```
Origin:     external
Import:     preserve
Managed:    github.com/acme/apis-contrib-google/google/pubsub/v1
Upstream:   github.com/googleapis/googleapis/google/pubsub/v1
```

### `apx add` (provenance in lock file)

Adding an external API as a dependency records provenance in `apx.lock`:

```
✓ Added dependency: proto/google/pubsub/v1@v1.0.0
  Source: github.com/acme/apis-contrib-google (external, preserve)
```

### `apx catalog generate` (merged externals)

Catalog generation merges external APIs from `apx.yaml`:

```
✓ Catalog generated: 15 modules (12 first-party, 3 external)
```
