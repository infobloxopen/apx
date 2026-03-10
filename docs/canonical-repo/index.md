# Canonical Repository

The canonical repository (`github.com/<org>/apis`) is the single source of truth for all organization APIs. This is what consumers depend on and where CI creates official releases.

!!! important
    **APX 1.0 requires GitHub or Gitea** as the hosting platform for the canonical repository. GitHub Actions is the primary CI platform. Supporting GitLab, Bitbucket, Azure DevOps, or other hosting platforms is a non-goal for v1.0.


## Overview

The canonical repo pattern centralizes API governance while allowing distributed authoring:

<div class="grid cards" markdown>
-   **Single Source of Truth**

    ---

    - All organization APIs in one repo
    - Consistent versioning and tagging
    - Centralized governance policies
    - Protected release process

-   **Consumer-Friendly**

    ---

    - Go modules with semantic versioning
    - Discoverable via `apx search`
    - Generated catalog and documentation
    - Stable import paths

-   **CI-Only Releases**

    ---

    - Only automated CI creates tags
    - Protected branch and tag patterns
    - Automated breaking change detection
    - Consistent release artifacts

-   **Multi-Format Support**

    ---

    - Protocol Buffers (primary)
    - OpenAPI, Avro, JSON Schema
    - Parquet schema support
    - Format-specific validation

</div>

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