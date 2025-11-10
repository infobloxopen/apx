# APX — API Schema Management

```{toctree}
:maxdepth: 2
:caption: Contents

getting-started/index
canonical-repo/index
app-repos/index
dependencies/index
publishing/index
cli-reference/index
troubleshooting/index
```

::::{grid} 1 1 2 3
:gutter: 3

:::{grid-item-card}
:link: getting-started/index
:link-type: doc
:class-header: bg-light

🚀 **Getting Started**
^^^

Install APX and learn the core concepts for API schema management.

:::

:::{grid-item-card}
:link: canonical-repo/index
:link-type: doc
:class-header: bg-light

🏛️ **Canonical Repo**
^^^

Set up the organization-wide source of truth for all API schemas.

:::

:::{grid-item-card}
:link: app-repos/index
:link-type: doc
:class-header: bg-light

� **App Repos**
^^^

Author and publish schemas from your application repositories.

:::

:::{grid-item-card}
:link: dependencies/index
:link-type: doc
:class-header: bg-light

� **Dependencies**
^^^

Discover, add, and update API dependencies with versioning.

:::

:::{grid-item-card}
:link: publishing/index
:link-type: doc
:class-header: bg-light

� **Publishing**
^^^

Tag-based publishing workflow from app repos to canonical repo.

:::

:::{grid-item-card}
:link: https://github.com/infobloxopen/apx
:class-header: bg-light

🐙 **GitHub**
^^^

View source code, report issues, and contribute to APX.

:::

::::

## What is APX?

**APX** is a tiny CLI and repo pattern for publishing, discovering, and consuming organization-wide API schemas. **Primary: Protobuf**. Also: **OpenAPI**, **Avro**, **JSON Schema**, **Parquet**. No long-running service. Canonical distribution via a single GitHub repo and Go modules, with CI-only releases.

### Key Ideas

- **Canonical source of truth**: `github.com/<org>/apis` (one repo, many submodules)
- **App teams tag releases** in their app repo; `apx publish` opens a PR to the canonical repo using **git subtree** (history-preserving) or **copy** (simple snapshot)
- **Only CI** in the canonical repo creates tags and optional language packages (Maven, wheels, OCI bundles)
- **Protobuf is primary**; OpenAPI/Avro/JSONSchema/Parquet supported with format-specific breaking checks

### Architecture Overview

::::{grid} 1 1 1 2
:gutter: 3

:::{grid-item-card} **App Repos**
^^^
- Teams author schemas locally
- Tag releases in app repo
- `apx publish` opens PRs
- CI validates before publish
:::

:::{grid-item-card} **Canonical Repo**
^^^
- Single source of truth
- Versioned API modules
- Protected branches & tags
- CI creates releases
:::

::::

### Quick Start

```bash
# Install APX
brew install <org>/tap/apx
# or download from GitHub Releases

# Verify installation
apx --version

# Bootstrap canonical repo
apx init canonical github.com/<org>/apis

# Bootstrap app repo for authoring
apx init app internal/apis/proto/payments/ledger

# Add dependencies
apx add proto/payments/ledger/v1@v1.2.3

# Generate code stubs
apx gen go
```

### Supported Schema Formats

::::{grid} 1 1 2 2
:gutter: 2

:::{grid-item-card}
**Protocol Buffers** (Primary)
^^^
- Buf integration and workspace
- gRPC service definitions  
- Semantic Import Versioning
- Breaking change detection
:::

:::{grid-item-card}
**OpenAPI**
^^^
- OpenAPI 3.0+ specifications
- oasdiff breaking checks
- Spectral linting
- REST API definitions
:::

:::{grid-item-card}
**Apache Avro**
^^^
- Schema compatibility checks
- BACKWARD compatibility default
- Field defaults and aliases
- Data serialization schemas
:::

:::{grid-item-card}
**JSON Schema & Parquet**
^^^
- JSON Schema validation
- Parquet additive nullable columns
- Custom compatibility rules
- Schema evolution support
:::

::::

### Release Guardrails

**Automated Checks** (CI-enforced):
- Format-specific linting and breaking change detection
- Policy enforcement (ban service/ORM annotations)
- SemVer validation with `apx version suggest`
- Protected tag patterns (only CI can create tags)

**Human Gates**:
- `CODEOWNERS` per API path
- Time-boxed waivers for exceptions

---

*APX standardizes how teams author, publish, and consume versioned APIs across your organization.*