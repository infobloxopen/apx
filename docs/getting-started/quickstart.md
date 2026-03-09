# Quick Start

Get up and running with APX's canonical repo pattern using canonical import paths in under 10 minutes! This guide covers the essential workflow for organization-wide API schema management.

## Overview: Canonical Import Paths

APX uses a **canonical import path approach** with two types of repos:

1. **Canonical Repo** (`github.com/<org>/apis`) - Single source of truth for all organization APIs
2. **App Repos** - Where teams author schemas, generate stubs with canonical import paths, and release to canonical repo

**Key Benefits:**
- **One import path everywhere** - no rewrites when switching from local to released
- **go.work overlays** - seamless local development with canonical paths
- **No replace gymnastics** - clean dependency management

## Path Mapping Reference

Every API in the canonical repo maps to a deterministic set of coordinates:

| Coordinate | Example |
|------------|---------|
| **API ID** | `proto/payments/ledger/v1` |
| **Source path** | `proto/payments/ledger/v1` |
| **Proto package** | `myorg.payments.ledger.v1` |
| **Go module (v1)** | `github.com/myorg/apis/proto/payments/ledger` |
| **Go import (v1)** | `github.com/myorg/apis/proto/payments/ledger/v1` |
| **Go module (v2)** | `github.com/myorg/apis/proto/payments/ledger/v2` |
| **Go import (v2)** | `github.com/myorg/apis/proto/payments/ledger/v2` |
| **Git tag** | `proto/payments/ledger/v1/v1.2.3` |

**One canonical repo. One default import root. One path model.**

:::{tip}
**Custom import roots**: Set `import_root` in `apx.yaml` to use a vanity domain instead of the Git hosting path. For example, with `import_root: go.myorg.dev/apis`, the Go module becomes `go.myorg.dev/apis/proto/payments/ledger` and the import becomes `go.myorg.dev/apis/proto/payments/ledger/v1`. See [Configuration Reference](../cli-reference/configuration.md#import_root) for details.
:::

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
apx init canonical --org=<org> --repo=apis
```

This creates:

```
apis/
├── buf.yaml                 # org-wide lint/breaking policy  
├── buf.work.yaml            # workspace aggregating version dirs
├── CODEOWNERS               # per-path ownership
├── catalog/
│  ├── .gitignore            # ignores generated catalog.yaml
│  └── Dockerfile            # scratch-based image with OCI labels
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

## 2. Bootstrap an App Repo (Author & Release)

Now set up an app repository where teams author schemas:

```bash
cd /path/to/your-app-repo

# Initialize for authoring (Buf-focused by default)
apx init app --org=<org> --repo=<app-repo> internal/apis/proto/payments/ledger
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
version: 1
org: <org>
repo: <app-repo>
# import_root: go.<org>.dev/apis   # optional: custom Go import prefix
module_roots:
  - internal/apis/proto

# ── Identity block ───────────────────────────────────────────
# Describes *what* API this module represents, where the
# canonical source lives, the current release, and the
# language-specific coordinates consumers will import.

api:                                  # canonical API identity
  id: proto/payments/ledger/v1        # <format>/<domain>/<name>/<line>
  format: proto
  domain: payments
  name: ledger
  line: v1
  lifecycle: beta                  # experimental → beta → stable → deprecated → sunset

source:                               # where the canonical copy lives
  repo: github.com/<org>/apis
  path: proto/payments/ledger/v1

releases:
  current: v1.0.0-beta.1             # latest SemVer tag for this line

languages:                            # per-language derived coordinates
  go:
    module: github.com/<org>/apis/proto/payments/ledger      # go.mod module path
    import: github.com/<org>/apis/proto/payments/ledger/v1   # Go import path
```

:::{tip}
You only need to supply the **API ID** (`api.id`), **source repo**, and **lifecycle**.
APX derives the remaining fields — `format`, `domain`, `name`, `line`,
`source.path`, and all `languages` coordinates — automatically via
`apx init app` or `apx inspect identity`.  They are shown here so you can see
the full coordinate model that the rest of this guide builds on.
:::

**buf.work.yaml** (app repo):
```yaml
version: v1
directories:
  - internal/apis/proto/**/v1
  - internal/apis/proto/**/v2
```

### How Identity Flows Through the Workflow

The identity fields you see in `apx.yaml` drive every command in this guide:

| Field | Where it shows up |
|-------|-------------------|
| `api.id` | Git tag prefix (`proto/payments/ledger/v1/v1.2.3`), overlay directory name, `apx search` results |
| `api.lifecycle` | SemVer guardrails — `beta` APIs may only release `0.x` or pre-release versions |
| `source.repo` + `source.path` | Target for `apx release` PRs; base path in the canonical repo |
| `languages.go.module` | `go.mod` synthesised during `apx gen go` and overlay setup |
| `languages.go.import` | The import path your application code uses — unchanged from local dev through production |

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
**No local `go.mod` required** for authoring. Buf ignores it. `apx release` synthesizes the correct `go.mod` in the PR to canonical repo.
:::

## 4. Local Development with Canonical Import Paths

Validate and test your schemas locally using canonical import paths:

```bash
# Download pinned toolchain
apx fetch

# Validate schema
apx lint         # buf lint + other format linters

# Check for breaking changes (if you have a baseline)
apx breaking --against=HEAD^  # buf breaking / oasdiff / avro compat

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
│   │       ├── go.mod        # module github.com/<org>/apis/proto/<domain>/<api>
│   │       └── v1/*.pb.go    # imports canonical path above
│   └── apis/...              # your proto sources
└── main.go                   # imports github.com/<org>/apis/proto/<domain>/<api>/v1
```

**Concrete example:**

```
payment-service/
├── go.mod                    # module github.com/mycompany/payment-service
├── go.work                   # managed by apx
├── internal/
│   ├── gen/
│   │   ├── go/proto/payments/ledger@v1.2.3/
│   │   │   ├── go.mod        # module github.com/myorg/apis/proto/payments/ledger
│   │   │   └── v1/*.pb.go    # package ledgerv1
│   │   └── go/proto/users/profile@v1.0.1/
│   │       ├── go.mod        # module github.com/myorg/apis/proto/users/profile
│   │       └── v1/*.pb.go    # package profilev1
│   └── apis/...              # your proto sources
└── main.go                   # imports github.com/myorg/apis/proto/payments/ledger/v1
```

**go.work overlay:**

```text
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
    
    // Pattern: github.com/<org>/apis/proto/<domain>/<api>/v1
    ledgerv1 "github.com/myorg/apis/proto/payments/ledger/v1"
    usersv1 "github.com/myorg/apis/proto/users/profile/v1"
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
├── go.mod                          # module github.com/myorg/apis/proto/payments/ledger
├── v1/
│   ├── ledger.pb.go               # package ledgerv1
│   └── ledger_grpc.pb.go          # imports canonical path
```

:::{important}
**Policy**: `/internal/gen/**` is git-ignored. Never commit generated code. Commit `apx.lock` instead. Generated Go code uses canonical import paths resolved via go.work overlays.
:::

## 5. Release Workflow

When ready to release your schema:

### 1. Validate Locally
```bash
apx lint && apx breaking --against=HEAD^ && apx semver suggest --against=HEAD^
```

### 2. Release via PR

The simplest path for teams: `apx release prepare` copies your module into
the canonical repo on a feature branch, and `apx release submit` opens a pull request via the `gh` CLI.

```bash
# One-time: gh auth login
apx release prepare proto/payments/ledger/v1 \
  --version v1.0.0-beta.1 \
  --lifecycle beta \
&& apx release submit
```

APX will:
1. Shallow-clone the canonical repo
2. Copy your module files into `proto/payments/ledger/v1/`
3. Generate `go.mod` if missing
4. Push a feature branch `apx/release/proto-payments-ledger-v1/v1.0.0-beta.1`
5. Open a PR on the canonical repo

### 3. Canonical Repo CI
On PR merge, canonical CI:
- Re-validates schema
- Verifies SemVer
- Creates subdirectory tag (`proto/payments/ledger/v1/v1.0.0-beta.1`)
- Go modules work automatically (Go proxy picks up the tag)
- Other language packages (Maven, wheels, OCI) require optional CI plugins

## 6. Consuming APIs with Canonical Import Paths

Other teams can now discover and use your released API with seamless local-to-released transitions:

### Discover APIs
```bash
apx search payments   # search the catalog
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
    ledgerv1 "github.com/myorg/apis/proto/payments/ledger/v1"
    usersv1 "github.com/myorg/apis/proto/users/profile/v1"
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
# Update to latest compatible version (minor/patch)
apx update proto/payments/ledger/v1

# Upgrade to a new major API line
apx upgrade proto/payments/ledger/v1 --to v2
```

### Switch to Released Module (No Import Changes!)
```bash
# Once the canonical module is released, seamlessly switch:
apx unlink proto/payments/ledger/v1     # remove go.work overlay
go get github.com/myorg/apis/proto/payments/ledger@v1.2.3

# Your application code remains completely unchanged:
# import ledgerv1 "github.com/myorg/apis/proto/payments/ledger/v1"
# 
# Before: resolved to ./internal/gen/go/proto/payments/ledger@v1.2.3 via go.work
# After:  resolved to released module github.com/myorg/apis/proto/payments/ledger@v1.2.3
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
+    github.com/myorg/apis/proto/payments/ledger v1.2.3
+    github.com/myorg/apis/proto/users/profile v1.0.1
)

// service.go - application code UNCHANGED
import (
    ledgerv1 "github.com/myorg/apis/proto/payments/ledger/v1"  // same import!
    usersv1 "github.com/myorg/apis/proto/users/profile/v1"    // same import!
)
```

## Common CI/CD Patterns

### App Repo CI (Release on Tag)
```yaml
name: Release API from App Repo
on:
  push:
    tags: ['proto/*/*/v*/v*']   # e.g., proto/payments/ledger/v1/v1.2.3

jobs:
  release:
    runs-on: ubuntu-latest
    permissions: { contents: read, pull-requests: write }
    steps:
      - uses: actions/checkout@v4
        with: { fetch-depth: 0 }
      - run: apx fetch
      - run: apx lint && apx breaking --against=HEAD^
      # Extract API ID and version from the tag
      # Tag format: proto/payments/ledger/v1/v1.2.3
      - run: |
          TAG="${GITHUB_REF_NAME}"
          API_ID="${TAG%/v*}"                    # proto/payments/ledger/v1
          VERSION="${TAG##*/}"                   # v1.2.3
          apx release prepare "$API_ID" --version "$VERSION" && apx release submit
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
      - run: apx fetch
      - run: apx lint && apx breaking --against=origin/main

  # NOTE: For automated tag creation and package releasing,
  # use `apx release prepare` + `apx release submit` + `apx release finalize`
  # See the release docs for the full release pipeline.
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

### Interactive mode in CI / non-TTY environments

APX automatically detects non-interactive environments (CI, piped input, etc.) and disables prompts. In those environments, use explicit flags:

```bash
apx init app --org=myorg --repo=myapp --non-interactive internal/apis/proto/payments/ledger
```

Interactive mode works normally in a terminal with TTY support.

## What's Next?

- Learn about [Interactive Initialization](interactive-init.md) features
- Explore [Local Development](../app-repos/local-development.md) for advanced workflows
- Review the [CLI Reference](../cli-reference/index.md) for all commands

---

**Questions?** Check the [Troubleshooting FAQ](../troubleshooting/faq.md) or open a [discussion](https://github.com/infobloxopen/apx/discussions).