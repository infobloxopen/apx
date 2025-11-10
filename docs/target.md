# apx — Getting Started & User Manual

> A tiny CLI and repo pattern for publishing, discovering, and consuming organization-wide API schemas using canonical import paths with go.work overlays. Primary: **Protobuf**. Also: **OpenAPI**, **Avro**, **JSON Schema**, **Parquet**. No long-running service. Canonical distribution via a single GitHub repo and Go modules, with CI-only releases.

---

## Table of Contents
- [apx — Getting Started \& User Manual](#apx--getting-started--user-manual)
  - [Table of Contents](#table-of-contents)
  - [What is apx?](#what-is-apx)
  - [Install](#install)
  - [Bootstrap the Canonical API Repo](#bootstrap-the-canonical-api-repo)
    - [1) Create the repo structure](#1-create-the-repo-structure)
    - [2) Protect branches \& tags](#2-protect-branches--tags)
    - [3) Add CI (validate + release)](#3-add-ci-validate--release)
  - [Bootstrap an App Repo (to author \& publish)](#bootstrap-an-app-repo-to-author--publish)
    - [1) Layout (Buf-only by default)](#1-layout-buf-only-by-default)
    - [2) Local dev](#2-local-dev)
    - [3) Enable publish-from-app workflow](#3-enable-publish-from-app-workflow)
  - [Discover, Add, Generate, and Update Dependencies](#discover-add-generate-and-update-dependencies)
    - [Discover APIs](#discover-apis)
    - [Add a dependency](#add-a-dependency)
    - [Generate stubs (never committed)](#generate-stubs-never-committed)
    - [Update to latest compatible](#update-to-latest-compatible)
    - [Upgrade to a new major](#upgrade-to-a-new-major)
  - [Publish Flow (tag-in-app → PR-to-canonical)](#publish-flow-tag-in-app--pr-to-canonical)
  - [Release Guardrails (CI/Policy)](#release-guardrails-cipolicy)
  - [Versioning \& Layout (v1, v2)](#versioning--layout-v1-v2)
  - [CI Templates](#ci-templates)
    - [App repo — publish on tag](#app-repo--publish-on-tag)
    - [Canonical repo — validate \& release](#canonical-repo--validate--release)
  - [FAQ](#faq)
  - [Troubleshooting](#troubleshooting)
    - [Appendix: Command Reference (selected)](#appendix-command-reference-selected)

---

## What is apx?
**apx** is a small CLI that standardizes how teams:
- author schemas locally (inside their app repos),
- publish those schemas to a **single canonical `apis` repo**, and
- consume versioned APIs with deterministic codegen.

**Key ideas**
- Canonical source of truth: `github.com/<org>/apis` (one repo, many submodules).
- App teams tag releases **in their app repo**; `apx publish` opens a PR to the canonical repo using **git subtree** (history-preserving).
- Only CI in the canonical repo creates tags and optional language packages (Maven, wheels, OCI bundles).
- **Canonical import paths everywhere**: Generated code uses the canonical import path (e.g. `github.com/<org>/apis/proto/<domain>/<api>`) even during local development.
- **go.work overlays**: Local development uses workspace overlays to resolve canonical paths to local generated stubs.
- Protobuf is primary; OpenAPI/Avro/JSONSchema/Parquet supported with format-specific breaking checks.

---

## The Canonical Import Path Approach

APX uses a **single canonical import path** approach that eliminates import rewrites and `replace` gymnastics:

* **Generate Go stubs locally** into your app's `internal/gen/**`.
* Those stubs **use the canonical import path** (e.g. `github.com/<org>/apis/proto/<domain>/<api>`), **even if it isn't published yet**.
* A workspace file **`go.work` overlays** that canonical path to the local generated stubs during dev.
* When a canonical module is published, **drop the overlay** and `go get` the real thing.
  → **One import path everywhere**, no `replace` gymnastics, no code rewrites.

### How it works

**App repo layout**

```
your-app/
  go.mod                     # module github.com/mycompany/payment-service
  go.work                    # managed by apx
  main.go                    # imports github.com/<org>/apis-go/proto/<domain>/<api>/v1
  internal/
    gen/
      go/proto/<domain>/<api>@v1.2.3/
        go.mod               # module github.com/<org>/apis-go/proto/<domain>/<api>
        v1/
          *.pb.go            # package <api>v1, canonical imports
          *.pb_grpc.go       # imports canonical paths
    apis/
      proto/<domain>/<api>/v1/...  # (your proto sources, not committed elsewhere)
```

**Concrete example:**

```
your-app/
  go.mod                     # module github.com/mycompany/payment-service
  go.work                    # managed by apx
  main.go                    # imports github.com/myorg/apis-go/proto/payments/ledger/v1
  internal/
    gen/
      go/proto/payments/ledger@v1.2.3/
        go.mod               # module github.com/myorg/apis-go/proto/payments/ledger
        v1/
          *.pb.go            # package ledgerv1, canonical imports
          *.pb_grpc.go       # imports canonical paths
    apis/
      proto/payments/ledger/v1/
        ledger.proto         # package myorg.payments.ledger.v1
```

**Workspace overlay (`go.work`)**

```txt
go 1.22
use .
use ./internal/gen/go/proto/<domain>/<api>@v1.2.3
# apx adds one "use" per pinned API while you're iterating locally
```

**Concrete example:**

```txt
go 1.22
use .
use ./internal/gen/go/proto/payments/ledger@v1.2.3
use ./internal/gen/go/proto/users/profile@v1.0.1
use ./internal/gen/go/proto/inventory/products@v2.1.0
```

**Commands**

```bash
# Pattern: apx add proto/<domain>/<api>/v1@v1.2.3
apx add proto/payments/ledger/v1@v1.2.3
apx add proto/users/profile/v1@v1.0.1
apx add proto/inventory/products/v2@v2.1.0

apx gen go          # generates canonical-import stubs into internal/gen/...
apx sync            # (re)writes go.work overlays to those local stubs

# Your code imports: github.com/<org>/apis-go/proto/<domain>/<api>/v1
# Go resolves to local stubs via go.work overlay
go test ./...       

# Later, when canonical module exists:
# Pattern: apx unlink proto/<domain>/<api>/v1
apx unlink proto/payments/ledger/v1    # remove overlay
# Pattern: go get github.com/<org>/apis-go/proto/<domain>/<api>@v1.2.3
go get github.com/myorg/apis-go/proto/payments/ledger@v1.2.3
```

**Application code example:**

```go
// main.go - your application
package main

import (
    "context"
    
    // Pattern: github.com/<org>/apis-go/proto/<domain>/<api>/v1
    ledgerv1 "github.com/myorg/apis-go/proto/payments/ledger/v1"
    usersv1 "github.com/myorg/apis-go/proto/users/profile/v1"
    productsv2 "github.com/myorg/apis-go/proto/inventory/products/v2"
)

func main() {  
    // Use generated types and clients - same code whether local or published
    ledgerReq := &ledgerv1.CreateEntryRequest{
        AccountId: "acc-123",
        Amount:    1000,
        Currency:  "USD",
    }
    
    userReq := &usersv1.GetProfileRequest{
        UserId: "user-456",
    }
    
    productReq := &productsv2.GetProductRequest{
        ProductId: "prod-789",
    }
    // ... rest of application logic
}
```

**Domain/API mapping examples:**
- `payments/ledger` → `github.com/<org>/apis-go/proto/payments/ledger/v1`
- `users/profile` → `github.com/<org>/apis-go/proto/users/profile/v1`
- `inventory/products` → `github.com/<org>/apis-go/proto/inventory/products/v2`
- `billing/invoices` → `github.com/<org>/apis-go/proto/billing/invoices/v1`

**Versioning rules**

* **Buf**: proto packages & dirs end with `vN` (e.g., `...ledger.v1` in `.../v1/`).
* **Go SIV**: v1 module path **no `/v1`**; v2+ paths **end with `/vN`**.

### Producer flow (author & publish)

1. Develop schemas under `internal/apis/**/vN/*.proto`

   ```bash
   apx lint && apx breaking
   apx gen go && apx sync      # local overlay on canonical path
   go test ./...
   ```

2. Tag & publish schemas (no generated code)

   * Tag from app or directly in the monorepo (your process), open PR to `/<org>/apis`.
   * Canonical CI: re-checks, creates **subdirectory tag**, optionally triggers language package builds (e.g., Go module in `apis-go`, Maven, wheels).

3. Switch app off overlay once published

   ```bash
   apx unlink <api>
   go get github.com/<org>/apis/proto/<domain>/<api>@v1.2.3
   ```

### Consumer flow (use existing APIs)

* **Fast path: just `go get` the published module(s).**

  ```bash
  go get github.com/myorg/apis-go/proto/payments/ledger@v1.2.3
  ```

  ```go
  // your-service/main.go
  import ledgerv1 "github.com/myorg/apis-go/proto/payments/ledger/v1"
  ```

* **If you need a producer's unreleased change:**

  ```bash
  apx add proto/payments/ledger/v1@v1.3.0-dev  # unreleased version
  apx gen go && apx sync                        # generate local overlay
  # build against local overlay until the tag lands, then unlink + go get
  ```

  ```go
  // your-service/main.go - same import, resolved to local overlay
  import ledgerv1 "github.com/myorg/apis-go/proto/payments/ledger/v1"
  ```

  **When the official release is available:**

  ```bash
  apx unlink proto/payments/ledger/v1          # remove overlay  
  go get github.com/myorg/apis-go/proto/payments/ledger@v1.3.0  # get published
  # No code changes needed - same import path!
  ```

---

## Install
```bash
# Homebrew (example)
brew install <org>/tap/apx

# Or download from GitHub Releases and place on PATH
chmod +x apx && mv apx /usr/local/bin/apx

# Verify
apx --version
```

> apx bundles pinned generators via `apx.lock` or can run in a container (`--use-container`).

---

## Bootstrap the Canonical API Repo
Target: `github.com/<org>/apis` (public or private). This is what consumers depend on.

### 1) Create the repo structure
```
apis/
├─ buf.yaml                 # org-wide lint/breaking policy
├─ buf.work.yaml            # workspace aggregating version dirs
├─ CODEOWNERS               # per-path ownership
├─ catalog/
│  └─ catalog.yaml          # generated index of APIs/owners/tags
└─ proto/                   # (add openapi/avro/jsonschema/parquet peers as needed)
   └─ payments/
      └─ ledger/
         ├─ go.mod          # v1 module: module github.com/<org>/apis/proto/payments/ledger
         ├─ v1/
         │  └─ ledger.proto # package <org>.payments.ledger.v1
         └─ v2/             # (empty until you add v2)
```

**buf.yaml (example)**
```yaml
version: v2
lint:
  use: [STANDARD]
breaking:
  use: [FILE, WIRE]
```

**buf.work.yaml**
```yaml
version: v1
directories:
  - proto/**/v1
  - proto/**/v2
```

### 2) Protect branches & tags
- Protect `main`.
- Protect tag patterns: `proto/**/v*`, `openapi/**/v*`, etc. Only CI can create them.

### 3) Add CI (validate + release)
- **Validate PRs** touching `proto/**`, `openapi/**`, `avro/**`, `jsonschema/**`, `parquet/**` with `apx lint`, `apx breaking`, `apx policy check`.
- **On merge** of PR created by `apx publish`, run `apx version verify`, create **subdirectory tag**, and publish optional artifacts (Maven/wheel/OCI). See [CI Templates](#ci-templates).

---

## Bootstrap an App Repo (to author & publish)
App repos own day-to-day authoring. They publish via tag + PR to canonical.

### 1) Layout (Buf-only by default)
```
<app-repo>/
├─ internal/
│  └─ apis/
│     └─ proto/
│        └─ payments/
│           └─ ledger/
│              ├─ v1/
│              │  └─ ledger.proto     # package <org>.payments.ledger.v1
│              └─ v2/                 # add when you introduce a breaking major
├─ buf.work.yaml
├─ apx.yaml
└─ apx.lock
```
> **No local `go.mod` required** for authoring; Buf ignores it. `apx publish` can synthesize a correct `go.mod` inside the PR to the canonical repo. If you prefer to keep a `go.mod` locally, ensure it’s canonical (no `replace`, correct module path) and it will be imported verbatim.

**buf.work.yaml (app repo)**
```yaml
version: v1
directories:
  - internal/apis/proto/**/v1
  - internal/apis/proto/**/v2
```

**apx.yaml (app repo)**
```yaml
apis:
  - kind: proto
    path: internal/apis/proto/payments/ledger/v1
    canonical: proto/payments/ledger/v1
codegen:
  out: internal/gen
  languages: [go, python, java]
```

### 2) Local dev
```bash
apx fetch        # pull pinned toolchain
apx lint         # buf lint + other format linters
apx breaking     # buf breaking / oasdiff / avro compat / jsonschema diff
apx gen go       # writes to internal/gen/go/<api>@<ver>/
```

### 3) Enable publish-from-app workflow
Add a GitHub Action that triggers on tags like `proto/<domain>/<api>/v1/v1.2.3` and runs `apx publish` to open a PR to `github.com/<org>/apis`.

---

## Discover, Add, Generate, and Update Dependencies

### Discover APIs
```bash
apx search payments ledger   # fuzzy search the catalog (from canonical/cached)
```

### Add a dependency
```bash
apx add proto/payments/ledger/v1@v1.2.3
# - pins in apx.lock
# - records codegen convention (languages, options)
```

### Generate stubs (never committed)
```bash
apx gen go       # → internal/gen/go/<api>@<ver>/ with canonical import paths
apx sync         # updates go.work to overlay canonical paths to local stubs
apx gen python   # → internal/gen/python/<api>@<ver>/...
apx gen java     # → internal/gen/java/<api>@<ver>/...
```

> **Policy**: `/internal/gen/**` is git-ignored. Commit `apx.lock` instead. Generated Go code uses canonical import paths and is resolved via `go.work` overlays.

### Update to latest compatible
```bash
# Patch/minor within current major
apx update proto/payments/ledger/v1   # chooses latest v1.x.y and updates apx.lock
apx gen go && apx sync                # regenerate stubs and update go.work overlays
```

### Upgrade to a new major
```bash
apx upgrade proto/payments/ledger/v2@v2.0.0
apx gen go && apx sync
# Update imports from .../ledger → .../ledger/v2 where applicable
# go.work automatically resolves new canonical paths to local stubs
```

### Switch from overlay to published module
```bash
# Once the canonical module is published
apx unlink proto/payments/ledger/v1     # removes go.work overlay
go get github.com/<org>/apis/proto/payments/ledger@v1.2.3   # get real module
# No import path changes needed - same canonical path everywhere
```

---

## Publish Flow (tag-in-app → PR-to-canonical)

1) **Validate locally**
```bash
apx lint && apx breaking && apx version suggest
```

2) **Tag in the app repo** (subdir-style tag)
```bash
# Example for v1
git tag proto/payments/ledger/v1/v1.2.3
git push --follow-tags
```

3) **App CI runs `apx publish`**
```bash
apx publish \
  --module-path=internal/apis/proto/payments/ledger/v1 \
  --canonical-repo=github.com/<org>/apis
```

4) **Canonical PR**
- Contains the versioned directory (e.g., `proto/payments/ledger/v1/...`),
- Adds `go.mod` (if not present) with correct module path,
- Includes CHANGELOG + lint/breaking reports.

5) **Canonical CI on PR merge**
- Re-runs checks.
- Verifies SemVer (`apx version verify`).
- Creates the **subdirectory tag** (`proto/payments/ledger/v1.2.3`).
- Publishes optional language packages.

---

## Release Guardrails (CI/Policy)

**Automated checks** (run in app CI before PR; re-run in canonical CI):
- **Protobuf**: `buf lint`, `buf breaking`.
- **OpenAPI**: `oasdiff breaking`, `spectral lint`.
- **Avro**: compatibility (default BACKWARD; fields need defaults; aliases for renames).
- **JSON Schema**: schema diff; forbid tightenings without major.
- **Parquet**: custom checker—additive **nullable** columns only.
- **Policy**: ban service/ORM annotations (e.g., any `(gorm.*)`) and unapproved generators.
- **SemVer**: `apx version suggest` (PR must match or CI fails).
- **Only CI can tag**: protected tag patterns.

**Human gates**
- `CODEOWNERS` per API path.
- Waivers (time-boxed) for exceptional cases.

---

## Versioning & Layout (v1, v2)

**Go Modules (Semantic Import Versioning)**
- **v1** module path has **no `/v1` suffix**: `module github.com/<org>/apis/proto/payments/ledger`.
- **v2+** module path ends with `/vN`: `module github.com/<org>/apis/proto/payments/ledger/v2`.

**Buf package & directory**
- Proto **package** ends with version (e.g., `.v1`, `.v2`).
- Files live under matching version directories: `.../v1/*.proto`, `.../v2/*.proto`.

**Canonical layout example**
```
proto/payments/ledger/
├─ go.mod           # v1 module
├─ v1/
│  └─ ledger.proto  # package <org>.payments.ledger.v1
└─ v2/
   ├─ go.mod        # v2 module
   └─ ledger.proto  # package <org>.payments.ledger.v2
```

**Tags**
- v1 releases: `proto/payments/ledger/v1.2.3`
- v2 releases: `proto/payments/ledger/v2.0.0`

---

## CI Templates

### App repo — publish on tag
```yaml
name: Publish API from App Repo
on:
  push:
    tags:
      - 'proto/*/*/v*/v*'   # e.g., proto/payments/ledger/v1/v1.2.3
jobs:
  publish:
    runs-on: ubuntu-latest
    permissions: { contents: read, pull-requests: write }
    steps:
      - uses: actions/checkout@v4
        with: { fetch-depth: 0 }
      - run: apx fetch --ci
      - run: apx lint && apx breaking && apx policy check
      - run: apx publish \
               --module-path=internal/apis/${GITHUB_REF_NAME%/v*} \
               --canonical-repo=github.com/<org>/apis
```

### Canonical repo — validate & release
```yaml
name: Validate + Release API Modules
on:
  pull_request:
    paths: ['proto/**', 'openapi/**', 'avro/**', 'jsonschema/**', 'parquet/**']
  push:
    branches: [ main ]

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: apx fetch --ci
      - run: apx lint && apx breaking && apx policy check

  tag_and_publish:
    if: github.ref == 'refs/heads/main'
    needs: [validate]
    runs-on: ubuntu-latest
    permissions: { contents: write, packages: write }
    steps:
      - uses: actions/checkout@v4
      - run: apx version verify
      - run: apx tag subdir proto/payments/ledger v1.2.3  # inferred in real workflow
      - run: apx packages publish
```

---

## FAQ
**Q: Do I need a `go.mod` in my app repo for authoring?**  
A: No. Buf ignores `go.mod`. `apx publish` will add a canonical `go.mod` in the PR. If you do keep one locally, ensure it’s canonical (no `replace`), and it will be imported verbatim.

**Q: Will v2 code leak into v1 consumers?**  
A: No. v2 lives in a separate module (`.../v2` with its own `go.mod`). v1 imports never see it unless explicitly referenced.

**Q: Can apx discover APIs?**  
A: Yes. `apx search <keywords>` queries the catalog (generated in the canonical repo). You can also browse tags.

**Q: Can apx update dependencies?**  
A: Yes. `apx update <api>` gets the latest patch/minor within the current major; `apx upgrade` targets a specific major.

**Q: How does APX publish APIs?**  
A: APX uses git subtree to preserve commit history when publishing APIs to the canonical repository.

**Q: How do we prevent service-specific options (e.g., gorm) in shared schemas?**  
A: `apx policy check` fails on `(gorm.*)` and unapproved generators, both in app CI (pre-PR) and canonical CI.

---

## Troubleshooting
- **Buf complaints about versioning**: ensure proto package ends with `vN` and files are under `.../vN/`.
- **Go mod path errors**: in v1, module path has **no `/v1`**; in v2+, module path **must** end with `/v2`.
- **Publish blocked for SemVer**: run `apx version suggest` and update your tag to match (`MAJOR/MINOR/PATCH`).
- **Generated code committed**: remove from VCS and add `/internal/gen/` to `.gitignore`; re-run `apx gen` in CI.

---

### Appendix: Command Reference (selected)
- `apx fetch` — download pinned toolchain; respects `apx.lock`.
- `apx lint` — run Buf/Spectral/etc.
- `apx breaking` — format-specific breaking checks.
- `apx policy check` — enforce banned options/plugins (e.g., `gorm`).
- `apx search <q>` — discover available APIs.
- `apx add <api>@<ver>` — add dependency; pin in `apx.lock`.
- `apx update <api>` / `apx upgrade <api>@<ver>` — bump dependencies.
- `apx gen <lang>` — generate stubs with canonical import paths into `internal/gen/<lang>/<api>@<ver>/`.
- `apx sync` — update `go.work` overlays to map canonical paths to local generated stubs.
- `apx unlink <api>` — remove `go.work` overlay for an API (switch to published module).
- `apx publish --module-path=... --canonical-repo=...` — open PR to canonical repo using git subtree.
- `apx version suggest|set|verify` — compute/validate required SemVer.
- `apx tag subdir <path> <version>` — create a subdirectory tag (CI only).

