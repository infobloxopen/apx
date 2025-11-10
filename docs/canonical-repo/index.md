# Canonical Repository Setup

The canonical repository (`github.com/<org>/apis`) is the single source of truth for all organization APIs. This is what consumers depend on and where CI creates official releases.

```{toctree}
:maxdepth: 2

setup
structure
ci-templates
protection
```

## Overview

The canonical repo pattern centralizes API governance while allowing distributed authoring:

::::{grid} 1 1 2 2
:gutter: 3

:::{grid-item-card} **Single Source of Truth**
^^^
- All organization APIs in one repo
- Consistent versioning and tagging
- Centralized governance policies
- Protected release process
:::

:::{grid-item-card} **Consumer-Friendly**
^^^
- Go modules with semantic versioning
- Discoverable via `apx search`
- Generated catalog and documentation
- Stable import paths
:::

:::{grid-item-card} **CI-Only Releases**
^^^
- Only automated CI creates tags
- Protected branch and tag patterns
- Automated breaking change detection
- Consistent release artifacts
:::

:::{grid-item-card} **Multi-Format Support**
^^^
- Protocol Buffers (primary)
- OpenAPI, Avro, JSON Schema
- Parquet schema support
- Format-specific validation
:::

::::

## Key Benefits

- **Governance**: Centralized policies and CODEOWNERS per API path
- **Discovery**: Teams can find and consume APIs via search/catalog
- **Versioning**: Semantic versioning with automated compatibility checks
- **Automation**: CI handles validation, tagging, and package publishing
- **Protection**: Only CI can create tags; human review via PRs

## Next Steps

1. [Set up the canonical repo structure](setup.md)
2. [Configure CI templates](ci-templates.md)  
3. [Implement branch and tag protection](protection.md)
4. [Understand the directory structure](structure.md)