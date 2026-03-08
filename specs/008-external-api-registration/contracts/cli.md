# CLI Contracts: External API Registration

**Feature**: 008-external-api-registration  
**Date**: 2026-03-08

## New Commands

### `apx external register`

Register an external API in the organization's APX catalog.

```
Usage:
  apx external register <api-id> [flags]

Args:
  api-id    Canonical API identity (format/domain/name/line)

Flags:
      --managed-repo string     Internal repository hosting curated snapshots (required)
      --managed-path string     Filesystem path in managed repository (required)
      --upstream-repo string    Original external repository URL (required)
      --upstream-path string    Path in upstream repository (required)
      --import-mode string      Import path handling: preserve, rewrite (default "preserve")
      --description string      Human-readable description
      --lifecycle string        Lifecycle state (experimental, beta, stable, deprecated, sunset)
      --version string          Current version of the managed snapshot (e.g., v1.0.0)
      --owners strings          Comma-separated list of owners
      --tags strings            Comma-separated list of tags
      --config string           Config file path (default "apx.yaml")

Global Flags:
  -q, --quiet       Suppress output
      --verbose      Verbose output
      --json         Output in JSON format
      --no-color     Disable colored output
```

**Outputs:**

Success (human):
```
✓ Registered external API: proto/google/pubsub/v1
  Managed:  github.com/Infoblox-CTO/apis-contrib-google :: google/pubsub/v1
  Upstream: github.com/googleapis/googleapis :: google/pubsub/v1
  Import:   preserve
  Origin:   external
```

Success (JSON):
```json
{
  "id": "proto/google/pubsub/v1",
  "managed_repo": "github.com/Infoblox-CTO/apis-contrib-google",
  "managed_path": "google/pubsub/v1",
  "upstream_repo": "github.com/googleapis/googleapis",
  "upstream_path": "google/pubsub/v1",
  "import_mode": "preserve",
  "origin": "external",
  "description": "Google Cloud Pub/Sub API",
  "lifecycle": "stable"
}
```

Error (duplicate ID):
```
✗ Registration failed: API ID "proto/google/pubsub/v1" already exists in catalog
```

Error (path conflict):
```
✗ Registration failed: managed path "google/pubsub/v1" conflicts with existing module "proto/google/pubsub/v1"
```

---

### `apx external list`

List all registered external APIs.

```
Usage:
  apx external list [flags]

Flags:
      --origin string     Filter by origin: external, forked (default: all)
      --config string     Config file path (default "apx.yaml")
```

**Outputs:**

Success (human):
```
External APIs (2 registered):

  proto/google/api/v1          [external] preserve
    Managed:  github.com/Infoblox-CTO/apis-contrib-google :: google/api
    Upstream: github.com/googleapis/googleapis :: google/api

  proto/google/pubsub/v1       [external] preserve
    Managed:  github.com/Infoblox-CTO/apis-contrib-google :: google/pubsub/v1
    Upstream: github.com/googleapis/googleapis :: google/pubsub/v1
```

Success (JSON):
```json
[
  {
    "id": "proto/google/api/v1",
    "origin": "external",
    "import_mode": "preserve",
    "managed_repo": "github.com/Infoblox-CTO/apis-contrib-google",
    "managed_path": "google/api",
    "upstream_repo": "github.com/googleapis/googleapis",
    "upstream_path": "google/api"
  }
]
```

Empty:
```
No external APIs registered.
```

---

### `apx external transition`

Transition an external API between registered and forked classification.

```
Usage:
  apx external transition <api-id> --to <external|forked> [flags]

Args:
  api-id    API identity to transition

Flags:
      --to string       Target classification: external or forked (required)
      --config string   Config file path (default "apx.yaml")
```

**Outputs:**

Success (human):
```
✓ Transitioned proto/google/pubsub/v1: external → forked
  Import mode changed: preserve → rewrite
  Upstream origin retained for provenance.
```

Reverse transition:
```
✓ Transitioned proto/google/pubsub/v1: forked → external
  Import mode changed: rewrite → preserve
```

Error (not external):
```
✗ Transition failed: "proto/payments/ledger/v1" is a first-party API. Only external APIs can be transitioned.
```

Error (already at target):
```
✗ Transition failed: "proto/google/pubsub/v1" is already classified as "forked".
```

---

## Modified Commands

### `apx search` (modified)

New flag added:

```
Flags:
      --origin string    Filter by origin: first-party, external, forked
```

**Output changes** — external APIs include origin indicator:

```
Found 3 API(s):

  proto/payments/ledger/v1
    Format: proto
    Domain: payments
    Line: v1
    Lifecycle: stable
    Latest stable: v1.2.0

  proto/google/pubsub/v1                    [external]
    Format: proto
    Domain: google
    Line: v1
    Lifecycle: stable
    Version: v1.0.0
    Managed: github.com/Infoblox-CTO/apis-contrib-google
    Import: preserve

  proto/google/api/v1                       [external]
    Format: proto
    Domain: google
    Line: v1
    Lifecycle: stable
    Managed: github.com/Infoblox-CTO/apis-contrib-google
    Import: preserve
```

With `--origin external`:
```
Found 2 API(s):
  [only external APIs shown]
```

JSON: Module objects include new fields when non-empty.

---

### `apx show` (modified)

**Output changes** — external APIs display Provenance section:

```
API
  ID:       proto/google/pubsub/v1
  Format:   proto
  Domain:   google
  Name:     pubsub
  Line:     v1

Provenance
  Origin:         external
  Import mode:    preserve
  Managed repo:   github.com/Infoblox-CTO/apis-contrib-google
  Managed path:   google/pubsub/v1
  Upstream repo:  github.com/googleapis/googleapis
  Upstream path:  google/pubsub/v1

Source
  Repo:   github.com/Infoblox-CTO/apis-contrib-google
  Path:   google/pubsub/v1

Lifecycle
  State:  stable
  Version: v1.0.0

Owners
  platform-team
```

For first-party APIs, the Provenance section is omitted (no output change).

---

### `apx inspect identity` (modified)

**Output changes** for external APIs:

```
API:        proto/google/pubsub/v1
Format:     proto
Domain:     google
Name:       pubsub
Line:       v1
Lifecycle:  stable
Origin:     external
Import:     preserve
Managed:    github.com/Infoblox-CTO/apis-contrib-google/google/pubsub/v1
Upstream:   github.com/googleapis/googleapis/google/pubsub/v1
```

For first-party APIs, Origin/Import/Managed/Upstream lines are omitted.

---

### `apx catalog build` / `apx catalog generate` (modified)

**Behavior change**: After auto-discovering first-party modules, the catalog generator reads `external_apis` from `apx.yaml` and merges them into the catalog. Conflict detection runs before writing.

No flag changes. Output reports merged external APIs:

```
✓ Catalog generated: 15 modules (12 first-party, 3 external)
```

---

### `apx dep add` (modified)

**Behavior change**: When adding a dependency on an external API, the dependency manager reads the external registration metadata from the catalog to populate provenance fields in the lock file.

No flag changes. Output indicates external provenance:

```
✓ Added dependency: proto/google/pubsub/v1@v1.0.0
  Source: github.com/Infoblox-CTO/apis-contrib-google (external, preserve)
```

## Error Codes

| Code | Command | Condition |
|------|---------|-----------|
| `ErrExternalDuplicateID` | `external register` | API ID already exists |
| `ErrExternalPathConflict` | `external register` | Managed path conflicts with existing module |
| `ErrExternalNotFound` | `external transition` | API ID not found in external registrations |
| `ErrExternalNotExternal` | `external transition` | API ID is first-party, cannot transition |
| `ErrExternalAlreadyTarget` | `external transition` | API already at target classification |
| `ErrExternalInvalidMode` | `external register` | Invalid import_mode value |
| `ErrExternalInvalidOrigin` | `external register` | Invalid origin value |
