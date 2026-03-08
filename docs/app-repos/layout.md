# App Repo Layout

This page describes the directory structure of an APX app repository — where files live, what each configuration file does, and how generated code and overlays fit into the tree.

## Complete Directory Structure

```
<app-repo>/
├── go.mod                          # your application module
├── go.sum                          # Go checksum database
├── go.work                         # managed by apx sync — overlays canonical → local
├── apx.yaml                       # APX configuration (identity, publishing, policy)
├── apx.lock                       # pinned toolchain and dependency versions
├── buf.yaml                       # Buf lint and breaking-change policy
├── buf.gen.yaml                   # Buf code generation plugin config
├── buf.work.yaml                  # Buf workspace — aggregates version dirs
├── .gitignore                     # must exclude internal/gen/ and .apx-tools/
├── .github/
│   └── workflows/
│       └── apx-publish.yml        # CI workflow for tag-triggered publishing
├── internal/
│   ├── apis/                      # schema source files (committed)
│   │   └── proto/
│   │       └── payments/
│   │           └── ledger/
│   │               ├── v1/
│   │               │   └── ledger.proto
│   │               └── v2/        # future breaking version line
│   │                   └── ledger.proto
│   ├── gen/                       # generated code (git-ignored, never committed)
│   │   ├── go/
│   │   │   └── proto/payments/ledger@v1.2.3/
│   │   │       ├── go.mod         # module github.com/<org>/apis/proto/payments/ledger
│   │   │       └── v1/
│   │   │           ├── ledger.pb.go
│   │   │           └── ledger_grpc.pb.go
│   │   ├── python/
│   │   │   └── proto/payments/ledger/
│   │   │       └── ledger_pb2.py
│   │   └── java/
│   │       └── proto/payments/ledger/
│   │           └── LedgerServiceGrpc.java
│   └── service/                   # your application code
│       └── payment_service.go     # imports canonical paths
├── cmd/
│   └── server/
│       └── main.go                # imports github.com/<org>/apis/proto/payments/ledger/v1
├── .apx-tools/                    # cached toolchain binaries (git-ignored)
│   ├── buf
│   └── protoc-gen-go
└── Makefile                       # optional — can wrap apx commands
```

---

## Top-Level Files

### apx.yaml

The primary APX configuration file. Created by `apx init app` and committed to the repo.

```yaml
version: 1
org: <org>
repo: <app-repo>
module_roots:
  - internal/apis/proto

api:
  id: proto/payments/ledger/v1
  format: proto
  domain: payments
  name: ledger
  line: v1
  lifecycle: beta              # experimental → beta → stable → deprecated → sunset

source:
  repo: github.com/<org>/apis
  path: proto/payments/ledger/v1

releases:
  current: v1.0.0-beta.1

languages:
  go:
    module: github.com/<org>/apis/proto/payments/ledger
    import: github.com/<org>/apis/proto/payments/ledger/v1

language_targets:
  go:
    enabled: true
    plugins:
      - name: protoc-gen-go
        version: v1.64.0
      - name: protoc-gen-go-grpc
        version: v1.5.0

publishing:
  tag_format: "{subdir}/v{version}"
  ci_only: true
```

Most fields after `api.id`, `source.repo`, and `api.lifecycle` are **derived automatically** by `apx init app` or `apx identity`. They are shown here for reference.

### apx.lock

Pins exact versions and checksums for every tool and dependency. Created on first `apx fetch` or `apx add`, and should be committed.

```yaml
version: 1
toolchains:
  buf:
    version: v1.50.0
    checksum: sha256:abc123...
  protoc-gen-go:
    version: v1.64.0
    checksum: sha256:def456...
dependencies:
  proto/users/profile/v1:
    repo: github.com/<org>/apis
    ref: proto/users/profile/v1/v1.0.1
    modules:
      - proto/users/profile
```

### buf.yaml

Buf linting and breaking-change policy for protobuf schemas:

```yaml
version: v1
name: buf.build/<org>/<app-repo>
lint:
  use:
    - DEFAULT
  except:
    - UNARY_RPC
breaking:
  use:
    - FILE
```

### buf.gen.yaml

Buf code generation plugin configuration:

```yaml
version: v1
plugins:
  - plugin: buf.build/protocolbuffers/go
    out: gen/go
    opt: paths=source_relative
  - plugin: buf.build/grpc/go
    out: gen/go
    opt: paths=source_relative
```

:::{note}
During `apx gen go`, APX uses the overlay manager rather than raw `buf generate` to ensure the generated code has canonical module paths and `go.mod` files. The `buf.gen.yaml` configuration is used as input but the output location is managed by APX.
:::

### buf.work.yaml

Buf workspace that aggregates all version directories:

```yaml
version: v1
directories:
  - internal/apis/proto/**/v1
  - internal/apis/proto/**/v2
```

### go.work

Managed by `apx sync`. Maps canonical module paths to local overlay directories:

```
go 1.22
use .
use ./internal/gen/go/proto/payments/ledger@v1.2.3
use ./internal/gen/go/proto/users/profile@v1.0.1
```

:::{warning}
Do not edit `go.work` manually. Use `apx sync` to regenerate it. CI environments regenerate it from `apx.lock` via `apx gen go && apx sync`.
:::

### .gitignore

Must exclude generated code and cached tools:

```gitignore
# APX generated code — regenerated from apx.lock
internal/gen/

# APX toolchain cache
.apx-tools/

# Go workspace — regenerated by apx sync
go.work
go.work.sum
```

---

## Source Schemas — `internal/apis/`

Schema source files are the only files you author directly. They live under `internal/apis/` organized by format, domain, name, and version line:

```
internal/apis/
└── proto/
    └── payments/
        └── ledger/
            ├── v1/
            │   ├── ledger.proto         # service + messages
            │   └── types.proto          # shared types
            └── v2/                      # separate version line
                └── ledger.proto
```

**Convention:**
- `internal/` prevents Go from vendoring schema directories
- Format directories (`proto/`, `openapi/`, `avro/`, etc.) match the canonical repo layout
- Version directories (`v1/`, `v2/`) separate breaking API lines
- Schema files set `go_package` to the canonical import path

### Supported Formats

| Format | Directory | File Extensions |
|--------|-----------|-----------------|
| Protocol Buffers | `proto/` | `.proto` |
| OpenAPI | `openapi/` | `.yaml`, `.json` |
| Avro | `avro/` | `.avsc` |
| JSON Schema | `jsonschema/` | `.json` |
| Parquet | `parquet/` | `.parquet` |

---

## Generated Code — `internal/gen/`

Generated code lives under `internal/gen/`, organized by language. This entire directory is **git-ignored** — it is regenerated from `apx.lock` by running `apx gen <lang>`.

### Go Overlays

Go overlays include a synthesized `go.mod` with the canonical module path, enabling `go.work` resolution:

```
internal/gen/go/
└── proto/payments/ledger@v1.2.3/
    ├── go.mod                       # module github.com/<org>/apis/proto/payments/ledger
    └── v1/
        ├── ledger.pb.go             # package ledgerv1
        └── ledger_grpc.pb.go        # gRPC stubs
```

The `@v1.2.3` suffix in the directory name is the pinned version from `apx.lock`. It ensures that different versions of the same API don't collide.

### Other Languages

Non-Go languages use their own resolution mechanisms and don't need `go.mod` or `go.work` entries:

```
internal/gen/python/proto/payments/ledger/
└── ledger_pb2.py

internal/gen/java/proto/payments/ledger/
└── LedgerServiceGrpc.java
```

---

## Toolchain Cache — `.apx-tools/`

`apx fetch` downloads pinned tool versions into `.apx-tools/`. This directory is git-ignored:

```
.apx-tools/
├── buf                    # Buf CLI
├── protoc-gen-go          # Go protobuf plugin
└── protoc-gen-go-grpc     # Go gRPC plugin
```

Versions and checksums are recorded in `apx.lock`, ensuring all team members and CI use identical toolchains.

---

## CI Workflows — `.github/workflows/`

### apx-publish.yml

Generated by `apx workflows sync`. Triggered when you push a tag matching the configured `tag_format`:

```
.github/workflows/
└── apx-publish.yml        # tag-triggered: validates → opens PR to canonical repo
```

See [CI Integration](ci-integration.md) for details on the workflow contents and triggers.

---

## Comparison with Canonical Repo

| Aspect | App Repo | Canonical Repo |
|--------|----------|----------------|
| **Schemas** | `internal/apis/proto/...` | `proto/...` (top-level) |
| **go.mod** | App module (`go.mod`) + overlays (`internal/gen/`) | Per-API `go.mod` at each module root |
| **go.work** | Yes — maps overlays to canonical paths | Not used |
| **Generated code** | `internal/gen/` (git-ignored) | Not present — consumers generate from published schemas |
| **Config** | `apx.yaml` + `apx.lock` | `apx.yaml` + `catalog/catalog.yaml` |
| **CI** | `apx-publish.yml` | `ci.yml` + `on-merge.yml` |
| **Buf config** | `buf.yaml` + `buf.gen.yaml` + `buf.work.yaml` | `buf.yaml` + `buf.work.yaml` |

---

## Bootstrapping

To create this layout from scratch:

```bash
cd /path/to/your-app-repo

# Initialize — creates apx.yaml, buf configs, schema dirs, .gitignore entries
apx init app --org=<org> --repo=<app-repo> internal/apis/proto/payments/ledger

# Download pinned toolchain
apx fetch

# Generate code and sync overlays
apx gen go && apx sync
```

See [Quickstart](../getting-started/quickstart.md) for a complete walkthrough.

## Next Steps

- [Local Development](local-development.md) — day-to-day workflow with these files
- [Publishing Workflow](publishing-workflow.md) — how schemas move from app repo to canonical repo
- [CI Integration](ci-integration.md) — automated validation and publishing
