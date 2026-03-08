# Quickstart: External API Registration

**Feature**: 008-external-api-registration  
**Date**: 2026-03-08

This guide walks through registering, discovering, and consuming external APIs with APX, using Google APIs as real-world examples.

## Prerequisites

- APX CLI installed (`apx version` returns 0.x.x+)
- A canonical API repository with `apx.yaml` (e.g., `github.com/your-org/apis`)
- A managed contributing repository for external API snapshots (e.g., `github.com/your-org/apis-contrib-google`)

## Scenario A: Register Google Common Protos

Google's API framework protos (`google/api/*`) are a shared dependency for virtually all Google Cloud APIs. Register them first.

### Step 1 — Register google/api

```bash
apx external register proto/google/api/v1 \
  --managed-repo github.com/Infoblox-CTO/apis-contrib-google \
  --managed-path google/api \
  --upstream-repo github.com/googleapis/googleapis \
  --upstream-path google/api \
  --description "Google API framework protos (annotations, resources, HTTP mapping)" \
  --lifecycle stable \
  --owners platform-team \
  --tags google,framework
```

Output:
```
✓ Registered external API: proto/google/api/v1
  Managed:  github.com/Infoblox-CTO/apis-contrib-google :: google/api
  Upstream: github.com/googleapis/googleapis :: google/api
  Import:   preserve
  Origin:   external
```

### Step 2 — Register google/type

```bash
apx external register proto/google/type/v1 \
  --managed-repo github.com/Infoblox-CTO/apis-contrib-google \
  --managed-path google/type \
  --upstream-repo github.com/googleapis/googleapis \
  --upstream-path google/type \
  --description "Google common types (Date, DateTime, Money, LatLng, etc.)" \
  --lifecycle stable \
  --owners platform-team \
  --tags google,types
```

### Step 3 — Verify registrations

```bash
apx external list
```

Output:
```
External APIs (2 registered):

  proto/google/api/v1          [external] preserve
    Managed:  github.com/Infoblox-CTO/apis-contrib-google :: google/api
    Upstream: github.com/googleapis/googleapis :: google/api

  proto/google/type/v1         [external] preserve
    Managed:  github.com/Infoblox-CTO/apis-contrib-google :: google/type
    Upstream: github.com/googleapis/googleapis :: google/type
```

### Step 4 — Rebuild catalog

```bash
apx catalog build .
```

Output:
```
✓ Catalog generated: 14 modules (12 first-party, 2 external)
```

The external APIs now appear in `catalog/catalog.yaml` alongside first-party APIs.

## Scenario B: Register Google Pub/Sub

### Step 1 — Register the API

```bash
apx external register proto/google/pubsub/v1 \
  --managed-repo github.com/Infoblox-CTO/apis-contrib-google \
  --managed-path google/pubsub/v1 \
  --upstream-repo github.com/googleapis/googleapis \
  --upstream-path google/pubsub/v1 \
  --description "Google Cloud Pub/Sub API" \
  --lifecycle stable \
  --version v1.0.0 \
  --owners platform-team \
  --tags google,messaging,pubsub
```

### Step 2 — Verify with search and inspect

```bash
apx search pubsub
```

Output:
```
Found 1 API(s):

  proto/google/pubsub/v1                    [external]
    Description: Google Cloud Pub/Sub API
    Format: proto
    Domain: google
    Line: v1
    Lifecycle: stable
    Version: v1.0.0
    Managed: github.com/Infoblox-CTO/apis-contrib-google
    Import: preserve
```

```bash
apx inspect identity proto/google/pubsub/v1
```

Output:
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

### Step 3 — Verify import preservation

The managed repository at `github.com/Infoblox-CTO/apis-contrib-google` contains:

```
google/
├── api/
│   ├── annotations.proto
│   ├── http.proto
│   ├── client.proto
│   ├── field_behavior.proto
│   ├── resource.proto
│   └── launch_stage.proto
├── type/
│   ├── date.proto
│   ├── datetime.proto
│   └── ...
└── pubsub/
    └── v1/
        ├── pubsub.proto
        └── schema.proto
```

The proto files retain their upstream import statements unchanged:
```protobuf
// google/pubsub/v1/pubsub.proto — imports are UNCHANGED from upstream
import "google/api/annotations.proto";
import "google/api/client.proto";
import "google/api/field_behavior.proto";
import "google/api/resource.proto";
import "google/protobuf/duration.proto";
import "google/protobuf/empty.proto";
import "google/protobuf/timestamp.proto";
```

No APX-specific path prefixes are injected.

## Scenario C: Consumer Usage

A developer building an internal service wants to depend on the registered Google Pub/Sub API.

### Step 1 — Add the dependency

In the consuming project's directory:

```bash
apx dep add proto/google/pubsub/v1
```

Output:
```
✓ Added dependency: proto/google/pubsub/v1@v1.0.0
  Source: github.com/Infoblox-CTO/apis-contrib-google (external, preserve)
```

### Step 2 — Check the lock file

```bash
cat apx.lock
```

```yaml
version: 1
dependencies:
  proto/google/pubsub/v1:
    repo: github.com/Infoblox-CTO/apis-contrib-google
    ref: v1.0.0
    modules:
      - google/pubsub/v1
    origin: external
    upstream_repo: github.com/googleapis/googleapis
    upstream_path: google/pubsub/v1
    import_mode: preserve
```

### Step 3 — Inspect the dependency

```bash
apx show proto/google/pubsub/v1
```

Output:
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

### Step 4 — Use in your proto files

```protobuf
// internal/apis/proto/notifications/alerts/v1/alerts.proto
syntax = "proto3";
package notifications.alerts.v1;

// Import the registered external API — using upstream paths, no rewriting
import "google/pubsub/v1/pubsub.proto";
import "google/api/annotations.proto";
```

These imports resolve directly because APX fetches the schemas from the managed repository at their original paths.

## Scenario D: Transition to Forked/Internalized

After consuming Google Pub/Sub for months, the organization decides to add custom extensions and diverge from upstream.

### Step 1 — Transition the classification

```bash
apx external transition proto/google/pubsub/v1 --to forked
```

Output:
```
✓ Transitioned proto/google/pubsub/v1: external → forked
  Import mode changed: preserve → rewrite
  Upstream origin retained for provenance.
```

### Step 2 — Verify the change

```bash
apx inspect identity proto/google/pubsub/v1
```

Output:
```
API:        proto/google/pubsub/v1
Format:     proto
Domain:     google
Name:       pubsub
Line:       v1
Lifecycle:  stable
Origin:     forked
Import:     rewrite
Managed:    github.com/Infoblox-CTO/apis-contrib-google/google/pubsub/v1
Upstream:   github.com/googleapis/googleapis/google/pubsub/v1
```

### Step 3 — Understanding the impact

After forking:
- The managed copy is now the **authoritative source** — the organization can modify schemas freely.
- Import mode changes to `rewrite`, meaning APX may rewrite import paths to internal conventions when consumers update their dependencies.
- The upstream origin is **retained for provenance** — `apx inspect` still shows where the API originally came from.
- Consumers receive a notification or warning when they next run `apx dep update` that the API has transitioned.

### Step 4 — Reverse transition (if needed)

If the organization decides to re-align with upstream:

```bash
apx external transition proto/google/pubsub/v1 --to external
```

```
✓ Transitioned proto/google/pubsub/v1: forked → external
  Import mode changed: rewrite → preserve
```

## Search Workflows

### Find all external APIs

```bash
apx search --origin external
```

### Find all Google APIs

```bash
apx search google --origin external
```

### Find all forked APIs

```bash
apx search --origin forked
```

### List all APIs (first-party + external)

```bash
apx search
```

The output distinguishes external APIs with an `[external]` or `[forked]` tag.
