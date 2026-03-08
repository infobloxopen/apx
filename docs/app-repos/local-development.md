# Local Development

This page covers the day-to-day development workflow in an app repository — authoring schemas, validating them, generating code with canonical import paths, and testing locally before publishing.

## Prerequisites

- **APX CLI** installed ([Installation](../getting-started/installation.md))
- **App repo initialized** with `apx init app` ([Quickstart](../getting-started/quickstart.md))
- **Buf CLI** available (APX installs it automatically via `apx fetch`)
- **Go** 1.22+ (for Go code generation and `go.work` overlays)

## The Development Loop

```bash
# 1. Author or edit schemas
vim internal/apis/proto/payments/ledger/v1/ledger.proto

# 2. Validate
apx lint
apx breaking --against HEAD^

# 3. Generate code with canonical imports
apx gen go

# 4. Sync go.work overlays
apx sync

# 5. Build and test — imports use canonical paths
go test ./...

# 6. Repeat
```

## Authoring Schemas

Schema files live under `internal/apis/` in the app repo, organized by format, domain, name, and version line:

```
internal/apis/proto/payments/ledger/v1/ledger.proto
internal/apis/proto/payments/ledger/v1/types.proto
internal/apis/openapi/billing/invoices/v1/invoices.yaml
```

### Setting `go_package`

For protobuf schemas, set `go_package` to the **canonical** import path — not your app repo path:

```protobuf
syntax = "proto3";

package acme.payments.ledger.v1;

// Points to the canonical repo, not the app repo
option go_package = "github.com/acme-corp/apis/proto/payments/ledger/v1";

service LedgerService {
  rpc CreateEntry(CreateEntryRequest) returns (CreateEntryResponse);
}
```

APX validates this during `apx lint`, `apx publish`, and `apx release prepare`, warning if the `go_package` doesn't match the canonical path derived from the API ID.

:::{note}
**No local `go.mod` is needed** for the schema directory. Buf ignores `go.mod`. APX synthesizes the correct `go.mod` when publishing to the canonical repo.
:::

---

## Validation

### Lint

Run format-specific linting on all schemas:

```bash
apx lint
```

For protobuf, this runs `buf lint` against your `buf.yaml` configuration. For OpenAPI, it runs OpenAPI-specific validators.

### Breaking Change Detection

Compare your working tree against a baseline to detect backward-incompatible changes:

```bash
# Against the previous commit
apx breaking --against HEAD^

# Against the main branch
apx breaking --against origin/main

# Against a specific tag
apx breaking --against proto/payments/ledger/v1/v1.0.0
```

For protobuf, this runs `buf breaking`. It checks for field removals, type changes, renumbering, and other wire-incompatible changes.

### SemVer Suggestion

APX can suggest the appropriate version bump based on detected changes:

```bash
apx semver suggest --against HEAD^
# → minor (new fields added, no breaking changes)
```

---

## Code Generation

Generate language-specific code from your schemas:

```bash
# Generate Go code
apx gen go

# Generate Python code
apx gen python

# Generate Java code
apx gen java

# Clean output before regenerating
apx gen go --clean
```

### Output Structure

Generated code is written to `internal/gen/<lang>/` with canonical module structure:

```
internal/gen/
├── go/
│   └── proto/payments/ledger@v1.2.3/
│       ├── go.mod           # module github.com/acme-corp/apis/proto/payments/ledger
│       └── v1/
│           ├── ledger.pb.go       # package ledgerv1
│           └── ledger_grpc.pb.go  # gRPC stubs
└── python/
    └── proto/payments/ledger@v1.2.3/
        └── ledger_pb2.py
```

The generated `go.mod` uses the **canonical module path** (`github.com/<org>/apis/proto/...`), which is the key to the overlay system.

:::{important}
**Never commit generated code.** The `internal/gen/` directory should be in `.gitignore`. Commit `apx.lock` instead — it ensures reproducible generation.
:::

---

## go.work Overlays

The overlay system is what makes canonical import paths work during local development.

### How It Works

1. `apx gen go` generates code into `internal/gen/go/<api>@<version>/` with a `go.mod` that declares the canonical module path
2. `apx sync` adds a `use` directive in `go.work` pointing to each generated directory
3. When you run `go build` or `go test`, Go resolves canonical import paths to the local generated code via the `go.work` overlay

```
# go.work (managed by apx sync)
go 1.22
use .
use ./internal/gen/go/proto/payments/ledger@v1.2.3
use ./internal/gen/go/proto/users/profile@v1.0.1
```

### Syncing

After generating code, sync the `go.work` file:

```bash
apx sync
```

This scans `internal/gen/go/` for all overlay directories and updates `go.work` accordingly.

```bash
# Preview what would change
apx sync --dry-run

# Clean stale overlays and resync
apx sync --clean
```

### Using Canonical Imports

Your application code imports canonical paths as if using the published module:

```go
package main

import (
    "context"

    ledgerv1 "github.com/acme-corp/apis/proto/payments/ledger/v1"
    usersv1 "github.com/acme-corp/apis/proto/users/profile/v1"
)

func main() {
    // These imports resolve to ./internal/gen/go/... via go.work
    client := ledgerv1.NewLedgerServiceClient(conn)
    resp, err := client.CreateEntry(context.Background(), &ledgerv1.CreateEntryRequest{
        AccountId: "acct-123",
        Amount:    1000,
        Currency:  "USD",
    })
}
```

During local development, Go resolves these imports via `go.work` to the local overlay. After publishing, you can switch to the real published module with no import changes.

---

## Adding Dependencies

To use schemas published by other teams:

```bash
# Add a dependency at a specific version
apx add proto/payments/ledger/v1@v1.2.3

# Add without a version (uses latest)
apx add proto/users/profile/v1

# Generate code for the dependency
apx gen go
apx sync
```

Dependencies are recorded in `apx.yaml` and pinned in `apx.lock`.

See [Adding Dependencies](../dependencies/adding-dependencies.md) for details.

---

## Switching to Published Modules

When the schema is published to the canonical repo and you're ready to consume the real module instead of the local overlay:

```bash
# Remove the overlay
apx unlink proto/payments/ledger/v1

# Add the published module to go.mod
go get github.com/acme-corp/apis/proto/payments/ledger@v1.2.3
```

Your application code stays **exactly the same** — the import path `github.com/acme-corp/apis/proto/payments/ledger/v1` now resolves to the published module instead of the local overlay.

```diff
# go.work (after unlink)
go 1.22
use .
-use ./internal/gen/go/proto/payments/ledger@v1.2.3

# go.mod (after go get)
require (
+    github.com/acme-corp/apis/proto/payments/ledger v1.2.3
)

# main.go — UNCHANGED
import ledgerv1 "github.com/acme-corp/apis/proto/payments/ledger/v1"
```

---

## Fetching Toolchains

APX manages toolchain versions (Buf, protoc plugins, etc.) via `apx.lock`:

```bash
# Download and cache all pinned tools
apx fetch

# Verify checksums
apx fetch --verify
```

Tools are cached in `.apx-tools/` (also gitignored). This ensures everyone on the team uses identical tool versions.

---

## Common Workflows

### New Schema from Scratch

```bash
# Initialize app repo
apx init app --org acme-corp --repo payment-service internal/apis/proto/payments/ledger

# Author your schema
vim internal/apis/proto/payments/ledger/v1/ledger.proto

# Validate
apx lint

# Generate and test
apx gen go && apx sync
go test ./...
```

### Iterating on an Existing Schema

```bash
# Edit schema
vim internal/apis/proto/payments/ledger/v1/ledger.proto

# Check for breaking changes
apx breaking --against HEAD^

# Suggest version bump
apx semver suggest --against HEAD^

# Regenerate and test
apx gen go --clean && apx sync
go test ./...
```

### Consuming a Team's Published API

```bash
# Discover available APIs
apx search payments

# Inspect details
apx show proto/payments/ledger/v1

# Add as dependency
apx add proto/payments/ledger/v1@v1.2.3

# Generate client code
apx gen go && apx sync

# Use in your code with canonical imports
# import ledgerv1 "github.com/acme-corp/apis/proto/payments/ledger/v1"
```

---

## Tips

- **Run `apx sync` after every `apx gen go`** — the overlay won't work without the `go.work` entry
- **Use `apx gen go --clean`** when switching versions to avoid stale generated files
- **Check `apx inspect identity`** to verify the canonical coordinates APX will use
- **Keep `internal/gen/` in `.gitignore`** — commit `apx.lock` for reproducibility
- **Use `apx --json show`** to get machine-readable API metadata for scripts

## Next Steps

- [Publishing Workflow](publishing-workflow.md) — publish to the canonical repo
- [CI Integration](ci-integration.md) — automate validation and publishing
- [Code Generation](../dependencies/code-generation.md) — multi-language generation details
- [Versioning Strategy](../dependencies/versioning-strategy.md) — SemVer and API line conventions
