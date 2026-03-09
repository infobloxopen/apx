# App Repository Setup

App repositories are where teams author schemas day-to-day and generate code with canonical import paths. They release via tag + PR to the canonical repository, enabling distributed authoring with centralized governance and seamless local-to-released transitions.

```{toctree}
:maxdepth: 2

layout
local-development
working-with-forks
publishing-workflow
ci-integration
```

## Overview

App repos handle the development lifecycle:

::::{grid} 1 1 2 2
:gutter: 3

:::{grid-item-card} **Daily Authoring**
^^^
- Teams work in familiar app repositories
- Schema files alongside application code
- Local validation and testing
- Standard development workflows
:::

:::{grid-item-card} **Buf-First Approach**
^^^
- No local `go.mod` required for authoring
- Buf workspace configuration
- APX synthesizes canonical `go.mod` on release
- Clean separation of concerns
:::

:::{grid-item-card} **Automated Releasing**
^^^
- Tag releases in app repo
- CI automatically opens canonical repo PR
- PR-based release to canonical repo
- Validation before and after release
:::

:::{grid-item-card} **Canonical Import Paths**
^^^
- Generated code uses canonical import paths
- `go.work` overlays for local development
- Seamless switch to released modules
- No import rewrites or replace directives
:::

::::

## Key Benefits

- **Familiar Workflow**: Teams work in their existing repositories
- **Canonical Import Paths**: Single import path from local dev to production
- **Local Testing**: Full validation and code generation with canonical imports
- **go.work Overlays**: Seamless local development without import gymnastics  
- **Automated Releasing**: No manual canonical repo management
- **Clean History**: PR-based workflow with full audit trail in canonical repo
- **Policy Enforcement**: APX validates schemas before releasing

## Directory Structure

Standard app repo layout for schema authoring:

```
<app-repo>/
├── go.mod                          # your app's module
├── go.work                         # managed by apx - overlays canonical → local
├── internal/
│  ├── gen/
│  │  └── go/proto/<domain>/<api>@v1.2.3/
│  │     ├── go.mod                 # module github.com/<org>/apis/proto/...
│  │     └── *.pb.go                # imports canonical path above
│  └── apis/
│     └── proto/                    # or openapi/, avro/, etc.
│        └── payments/
│           └── ledger/
│              ├── v1/
│              │  └── ledger.proto  # your schema files
│              └── v2/              # future breaking versions
├── buf.work.yaml                   # Buf workspace config
├── apx.yaml                       # APX configuration
├── apx.lock                        # Pinned toolchain versions
└── .gitignore                      # excludes internal/gen/
```

## Configuration Files

### apx.yaml
APX configuration for the app repository:

```yaml
version: 1
org: <org>
repo: <app-repo>
module_roots:
  - internal/apis/proto
language_targets:
  go:
    enabled: true
release:
  tag_format: "{subdir}/v{version}"
  ci_only: true
```

### buf.work.yaml
Buf workspace covering all version directories:

```yaml
version: v1
directories:
  - internal/apis/proto/**/v1
  - internal/apis/proto/**/v2
  # Automatically includes future versions
```

## Next Steps

1. [Set up the directory layout](layout.md)
2. [Configure local development](local-development.md)
3. [Implement release workflow](publishing-workflow.md)
4. [Add CI integration](ci-integration.md)