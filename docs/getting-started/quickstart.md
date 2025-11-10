# Quick Start

Get up and running with APX's canonical repo pattern using canonical import paths in under 10 minutes! This guide covers the essential workflow for organization-wide API schema management.

## Overview: Canonical Import Paths

APX uses a **canonical import path approach** with two types of repos:

1. **Canonical Repo** (`github.com/<org>/apis`) - Single source of truth for all organization APIs
2. **App Repos** - Where teams author schemas, generate stubs with canonical import paths, and publish to canonical repo

**Key Benefits:**
- **One import path everywhere** - no rewrites when switching from local to published
- **go.work overlays** - seamless local development with canonical paths
- **No replace gymnastics** - clean dependency management

## 1. Bootstrap the Canonical API Repo

First, create your organization's canonical API repository:

```bash
# Create github.com/<org>/apis (public or private)
# This is what consumers depend on
```

### Initialize the Structure

```bash
git clone https://github.com/<org>/apis.git
cd apis

# Create the canonical structure
apx init canonical --org=<org>
```

This creates:

```
apis/
├── buf.yaml                 # org-wide lint/breaking policy  
├── buf.work.yaml            # workspace aggregating version dirs
├── CODEOWNERS               # per-path ownership
├── catalog/
│  └── catalog.yaml          # generated index of APIs/owners/tags
└── proto/                   # (add openapi/avro/jsonschema/parquet as needed)
   └── payments/
      └── ledger/
         ├── go.mod          # v1 module: module github.com/<org>/apis/proto/payments/ledger
         ├── v1/
         │  └── ledger.proto # package <org>.payments.ledger.v1
         └── v2/             # (empty until you add v2)
```

### Protection Settings

- **Protect `main` branch** - require PR reviews
- **Protect tag patterns**: `proto/**/v*`, `openapi/**/v*` - only CI can create tags

## 2. Bootstrap an App Repo (Author & Publish)

Now set up an app repository where teams author schemas:

```bash
cd /path/to/your-app-repo

# Initialize for authoring (Buf-focused by default)
apx init app internal/apis/proto/payments/ledger
```

This creates the **app repo structure**:

```
<app-repo>/
├── internal/
│  └── apis/
│     └── proto/
│        └── payments/
│           └── ledger/
│              ├── v1/
│              │  └── ledger.proto     # package <org>.payments.ledger.v1
│              └── v2/                 # add when introducing breaking major
├── buf.work.yaml       # Buf workspace config
├── apx.yaml          # APX configuration  
└── apx.lock           # Pinned toolchain versions
```

### Key Configuration Files

**apx.yaml** (app repo):
```yaml
apis:
  - kind: proto
    path: internal/apis/proto/payments/ledger/v1
    canonical: proto/payments/ledger/v1
codegen:
  out: internal/gen
  languages: [go, python, java]
```

**buf.work.yaml** (app repo):
```yaml
version: v1
directories:
  - internal/apis/proto/**/v1
  - internal/apis/proto/**/v2
```

## 3. Author Your Schema

Edit your schema files:

```bash
# Create your protobuf schema
vim internal/apis/proto/payments/ledger/v1/ledger.proto
```

### Example Protocol Buffer

```protobuf
syntax = "proto3";

package <org>.payments.ledger.v1;

option go_package = "github.com/<org>/apis/proto/payments/ledger/v1";

service LedgerService {
  rpc CreateEntry(CreateEntryRequest) returns (CreateEntryResponse);
  rpc GetEntry(GetEntryRequest) returns (GetEntryResponse);
}

message LedgerEntry {
  string id = 1;
  string account_id = 2;
  int64 amount = 3;
  string currency = 4;
  int64 created_at = 5;
}

message CreateEntryRequest {
  string account_id = 1;
  int64 amount = 2;
  string currency = 3;
}

message CreateEntryResponse {
  LedgerEntry entry = 1;
}

message GetEntryRequest {
  string id = 1;
}

message GetEntryResponse {
  LedgerEntry entry = 1;
}
```

:::{note}
**No local `go.mod` required** for authoring. Buf ignores it. `apx publish` synthesizes the correct `go.mod` in the PR to canonical repo.
:::

## 4. Local Development with Canonical Import Paths

Validate and test your schemas locally using canonical import paths:

```bash
# Download pinned toolchain
apx fetch

# Validate schema
apx lint         # buf lint + other format linters

# Check for breaking changes (if you have a baseline)
apx breaking     # buf breaking / oasdiff / avro compat

# Generate code with canonical import paths (never committed)
apx gen go       # writes to internal/gen/go/<api>@<ver>/ with canonical imports
apx sync         # updates go.work to overlay canonical paths to local stubs
apx gen python   # writes to internal/gen/python/<api>@<ver>/

# Test your code - imports use canonical paths, resolved via go.work
go test ./...    # your code imports github.com/<org>/apis/proto/..., resolves to local stubs
```

**App repo layout with canonical imports:**

```
your-app/
├── go.mod                    # your app's module
├── go.work                   # managed by apx - overlays canonical → local
├── internal/
│   ├── gen/
│   │   └── go/proto/<domain>/<api>@v1.2.3/
│   │       ├── go.mod        # module github.com/<org>/apis-go/proto/<domain>/<api>
│   │       └── v1/*.pb.go    # imports canonical path above
│   └── apis/...              # your proto sources
└── main.go                   # imports github.com/<org>/apis-go/proto/<domain>/<api>/v1
```

**Concrete example:**

```
payment-service/
├── go.mod                    # module github.com/mycompany/payment-service
├── go.work                   # managed by apx
├── internal/
│   ├── gen/
│   │   ├── go/proto/payments/ledger@v1.2.3/
│   │   │   ├── go.mod        # module github.com/myorg/apis-go/proto/payments/ledger
│   │   │   └── v1/*.pb.go    # package ledgerv1
│   │   └── go/proto/users/profile@v1.0.1/
│   │       ├── go.mod        # module github.com/myorg/apis-go/proto/users/profile
│   │       └── v1/*.pb.go    # package profilev1
│   └── apis/...              # your proto sources
└── main.go                   # imports github.com/myorg/apis-go/proto/payments/ledger/v1
```

**go.work overlay:**

```txt
go 1.22
use .
# Pattern: use ./internal/gen/go/proto/<domain>/<api>@v1.2.3
use ./internal/gen/go/proto/payments/ledger@v1.2.3
use ./internal/gen/go/proto/users/profile@v1.0.1
# apx sync adds one "use" per pinned API during local development
```

**Application code using canonical imports:**

```go
// main.go - your application imports canonical paths
package main

import (
    "context"
    
    // Pattern: github.com/<org>/apis-go/proto/<domain>/<api>/v1
    ledgerv1 "github.com/myorg/apis-go/proto/payments/ledger/v1"
    usersv1 "github.com/myorg/apis-go/proto/users/profile/v1"
    "google.golang.org/grpc"
)

func main() {
    conn, _ := grpc.Dial("localhost:9090", grpc.WithInsecure())
    defer conn.Close()
    
    // Use generated clients from canonical imports
    ledgerClient := ledgerv1.NewLedgerServiceClient(conn)
    usersClient := usersv1.NewProfileServiceClient(conn)
    
    // All imports resolve to local overlays during development
    ledgerResp, err := ledgerClient.CreateEntry(context.Background(), &ledgerv1.CreateEntryRequest{
        AccountId: "account-123",
        Amount:    1000,
        Currency:  "USD",
    })
    
    userResp, err := usersClient.GetProfile(context.Background(), &usersv1.GetProfileRequest{
        UserId: "user-456",
    })
    // ... handle responses
}
```

**Generated code structure (local overlay):**

```
internal/gen/go/proto/payments/ledger@v1.2.3/
├── go.mod                          # module github.com/myorg/apis-go/proto/payments/ledger
├── v1/
│   ├── ledger.pb.go               # package ledgerv1
│   └── ledger_grpc.pb.go          # imports canonical path
```

:::{important}
**Policy**: `/internal/gen/**` is git-ignored. Never commit generated code. Commit `apx.lock` instead. Generated Go code uses canonical import paths resolved via go.work overlays.
:::

## 5. Publishing Workflow

When ready to publish your schema:

### 1. Validate Locally
```bash
apx lint && apx breaking && apx version suggest
```

### 2. Tag in App Repo
```bash
# Example for v1 (subdir-style tag)
git tag proto/payments/ledger/v1/v1.2.3
git push --follow-tags
```

### 3. App CI Publishes Automatically
Your app's CI will run `apx publish` and open a PR to the canonical repo:

```bash
apx publish \
  --module-path=internal/apis/proto/payments/ledger/v1 \
  --canonical-repo=github.com/<org>/apis
```

### 4. Canonical Repo CI
On PR merge, canonical CI:
- Re-validates schema
- Verifies SemVer
- Creates subdirectory tag (`proto/payments/ledger/v1.2.3`)
- Publishes language packages (Maven, wheels, OCI)

## 6. Consuming APIs with Canonical Import Paths

Other teams can now discover and use your published API with seamless local-to-published transitions:

### Discover APIs
```bash
apx search payments ledger   # fuzzy search the catalog
```

### Add Dependencies
```bash
# Add a specific version
apx add proto/payments/ledger/v1@v1.2.3

# This pins in apx.lock and records codegen preferences
```

### Generate Client Code with Canonical Imports
```bash
apx gen go       # → internal/gen/go/<api>@<ver>/ with canonical import paths
apx sync         # updates go.work to overlay canonical → local stubs
apx gen python   # → internal/gen/python/<api>@<ver>/...
```

**Your application code imports canonical paths:**

```go
// service.go - consuming the payments ledger API
package service

import (
    "context"
    
    // Canonical import - resolved to local overlay during development
    ledgerv1 "github.com/myorg/apis-go/proto/payments/ledger/v1"
    usersv1 "github.com/myorg/apis-go/proto/users/profile/v1"
)

type PaymentService struct {
    ledgerClient ledgerv1.LedgerServiceClient
    usersClient  usersv1.ProfileServiceClient
}

func (s *PaymentService) ProcessPayment(ctx context.Context, userID, amount string) error {
    // Use generated types and clients from canonical imports
    user, err := s.usersClient.GetProfile(ctx, &usersv1.GetProfileRequest{
        UserId: userID,
    })
    if err != nil {
        return err
    }
    
    _, err = s.ledgerClient.CreateEntry(ctx, &ledgerv1.CreateEntryRequest{
        AccountId: user.Profile.AccountId,
        Amount:    parseInt64(amount),
        Currency:  "USD",
    })
    return err
}
```

**During development, Go resolves these imports via go.work overlay to local generated stubs.**

### Update Dependencies
```bash
# Update to latest compatible (patch/minor within current major)
apx update proto/payments/ledger/v1
apx gen go && apx sync  # regenerate with new version and update overlays

# Upgrade to new major version
apx upgrade proto/payments/ledger/v2@v2.0.0
apx gen go && apx sync  # canonical import path changes to .../v2
```

### Switch to Published Module (No Import Changes!)
```bash
# Once the canonical module is published, seamlessly switch:
apx unlink proto/payments/ledger/v1     # remove go.work overlay
go get github.com/myorg/apis-go/proto/payments/ledger@v1.2.3

# Your application code remains completely unchanged:
# import ledgerv1 "github.com/myorg/apis-go/proto/payments/ledger/v1"
# 
# Before: resolved to ./internal/gen/go/proto/payments/ledger@v1.2.3 via go.work
# After:  resolved to published module github.com/myorg/apis-go/proto/payments/ledger@v1.2.3
#
# No find/replace, no import rewrites, no replace directives needed!
```

**Example transition:**

```diff
# go.work (before unlink)
go 1.22
use .
-use ./internal/gen/go/proto/payments/ledger@v1.2.3
-use ./internal/gen/go/proto/users/profile@v1.0.1

# go.mod (after go get)
module github.com/mycompany/payment-service

go 1.22

require (
+    github.com/myorg/apis-go/proto/payments/ledger v1.2.3
+    github.com/myorg/apis-go/proto/users/profile v1.0.1
)

// service.go - application code UNCHANGED
import (
    ledgerv1 "github.com/myorg/apis-go/proto/payments/ledger/v1"  // same import!
    usersv1 "github.com/myorg/apis-go/proto/users/profile/v1"    // same import!
)
```

## Common CI/CD Patterns

### App Repo CI (Publish on Tag)
```yaml
name: Publish API from App Repo
on:
  push:
    tags: ['proto/*/*/v*/v*']   # e.g., proto/payments/ledger/v1/v1.2.3

jobs:
  publish:
    runs-on: ubuntu-latest
    permissions: { contents: read, pull-requests: write }
    steps:
      - uses: actions/checkout@v4
        with: { fetch-depth: 0 }
      - run: apx fetch --ci
      - run: apx lint && apx breaking
      - run: apx publish --module-path=internal/apis/${GITHUB_REF_NAME%/v*} \
               --canonical-repo=github.com/<org>/apis
```

### Canonical Repo CI (Validate & Release)
```yaml
name: Validate + Release API Modules
on:
  pull_request:
    paths: ['proto/**', 'openapi/**']
  push:
    branches: [main]

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: apx fetch --ci
      - run: apx lint && apx breaking

  tag_and_publish:
    if: github.ref == 'refs/heads/main'
    needs: [validate]
    runs-on: ubuntu-latest
    permissions: { contents: write, packages: write }
    steps:
      - uses: actions/checkout@v4
      - run: apx version verify
      - run: apx tag subdir proto/payments/ledger v1.2.3
      - run: apx packages publish
```

## Troubleshooting

### Schema validation fails

1. Check the specific error messages from `apx lint`
2. For proto files, ensure buf.yaml configuration is correct
3. Verify schema syntax matches the expected format

### Code generation fails

1. Ensure target language tools are installed (protoc, etc.)
2. Check `buf.gen.yaml` configuration for proto files
3. Verify output directories have write permissions

### Interactive mode doesn't work

APX automatically detects non-interactive environments (CI, etc.). Use explicit flags:

```bash
apx init --non-interactive proto com.example.service.v1
```

## What's Next?

- Learn about [Interactive Initialization](interactive-init.md) features
- Explore the [User Guide](../user-guide/index.md) for advanced workflows
- Check out [Examples](../examples/index.md) for specific use cases
- Review the [CLI Reference](../cli-reference/index.md) for all commands

---

**Questions?** Check the [FAQ](../user-guide/faq.md) or open a [discussion](https://github.com/infobloxopen/apx/discussions).