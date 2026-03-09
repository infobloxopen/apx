# Code Generation

APX generates language-specific code from schema files, placing the output in overlay directories with canonical import paths. This page covers the generation process, supported languages, and output structure.

## Overview

```bash
apx gen <lang> [path]
```

The `apx gen` command reads your schemas from `internal/apis/`, generates code using format-specific toolchains (e.g. Buf for protobuf), and writes the output to `internal/gen/<lang>/` with canonical module paths.

**Supported languages:** `go`, `python`, `java`

---

## How It Works

1. **Load dependencies** — reads `apx.lock` for pinned tool and dependency versions
2. **Create overlays** — for each dependency, creates a directory in `internal/gen/<lang>/` with the canonical module structure
3. **Generate code** — runs format-specific code generators (e.g. `buf generate` for protobuf)
4. **Synthesize go.mod** — for Go, creates a `go.mod` with the canonical module path so `go.work` can resolve imports
5. **Sync go.work** — for Go, automatically adds `use` directives for each overlay

---

## Go Generation

```bash
apx gen go
```

### Output Structure

```
internal/gen/go/
└── proto/payments/ledger@v1.2.3/
    ├── go.mod                       # module github.com/<org>/apis/proto/payments/ledger
    └── v1/
        ├── ledger.pb.go             # generated protobuf code
        └── ledger_grpc.pb.go        # generated gRPC stubs
```

The `@v1.2.3` suffix is the pinned version from `apx.lock`. The `go.mod` declares the **canonical module path**, enabling `go.work` overlay resolution.

### go.work Integration

After generating Go code, `apx gen go` automatically calls `apx sync` to update `go.work`:

```
go 1.22
use .
use ./internal/gen/go/proto/payments/ledger@v1.2.3
use ./internal/gen/go/proto/users/profile@v1.0.1
```

Your application code then imports canonical paths:

```go
import ledgerv1 "github.com/acme-corp/apis/proto/payments/ledger/v1"
```

Go resolves this to the local overlay via `go.work` during development.

---

## Python Generation

```bash
apx gen python
```

When `org` is configured in `apx.yaml`, `apx gen python` scaffolds each overlay as an installable Python package with:

- **`pyproject.toml`** — PEP 621 metadata with deterministic dist name (`<org>-<domain>-<api>-<line>`)
- **`__init__.py` hierarchy** — namespace packages using `pkgutil.extend_path`

### Output Structure

```
internal/gen/python/
└── proto/payments/ledger/v1/
    ├── pyproject.toml                    # name = "acme-payments-ledger-v1"
    └── acme_apis/
        ├── __init__.py                   # pkgutil.extend_path (namespace root)
        └── payments/
            ├── __init__.py
            └── ledger/
                ├── __init__.py
                └── v1/
                    └── __init__.py       # leaf — generated code lands here
```

### Editable Install

Link Python overlays into your virtualenv for import resolution:

```bash
# Activate your virtualenv
source .venv/bin/activate

# Link all Python overlays (runs pip install -e for each)
apx link python

# Then import in your code:
# from acme_apis.payments.ledger.v1 import ledger_pb2
```

This mirrors Go's `go.work` overlay approach — code is generated locally, not pulled from a registry. You control the grpc/protobuf versions in your own virtualenv.

### Switching to Released Package

```bash
apx unlink proto/payments/ledger/v1
pip install acme-payments-ledger-v1==1.2.3
```

Your import paths stay the same — `from acme_apis.payments.ledger.v1 import ...`

---

## Java Generation

```bash
apx gen java
```

### Output Structure

```
internal/gen/java/
└── proto/payments/ledger/
    └── LedgerServiceGrpc.java
```

Java overlays use their own classpath configuration.

---

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--out` | string | `""` | Override the output directory |
| `--clean` | bool | `false` | Remove existing output before generating |
| `--manifest` | bool | `false` | Emit a generation manifest listing all produced files |

### Clean Generation

Use `--clean` when switching versions or after changing dependencies to avoid stale files:

```bash
apx gen go --clean
```

This removes all existing overlays in `internal/gen/go/` before regenerating.

---

## Format-Specific Toolchains

Code generation uses format-specific tools, resolved from `apx.lock`:

| Format | Tool | Plugins |
|--------|------|---------|
| Protocol Buffers | `buf` | `protoc-gen-go`, `protoc-gen-go-grpc` |
| OpenAPI | — | Language-specific client generators |
| Avro | `avro-tools` | Language-specific serializers |
| JSON Schema | — | Schema-to-code generators |
| Parquet | — | Schema readers |

Toolchain versions are pinned in `apx.lock` and downloaded with `apx fetch`.

---

## .gitignore Policy

Generated code must **never be committed**. Add to `.gitignore`:

```gitignore
internal/gen/
```

CI regenerates code from `apx.lock` during each pipeline run, ensuring consistency.

---

## Workflow Integration

### Local Development — Go

```bash
apx gen go && apx sync
go test ./...
```

### Local Development — Python

```bash
apx gen python
source .venv/bin/activate
apx link python
pytest
```

### CI Pipeline

```bash
apx fetch          # download pinned tools
apx gen go         # regenerate from lock file
apx sync           # update go.work
go test ./...      # canonical imports resolve via overlay
```

## See Also

- [Local Development](../app-repos/local-development.md) — full development workflow
- [Adding Dependencies](adding-dependencies.md) — pin schemas before generating
- [App Repo Layout](../app-repos/layout.md) — where generated code lives
