# Publishing Workflow

APX implements a **tag-in-app → PR-to-canonical** publishing workflow that preserves team autonomy while ensuring governance and consistency.

:::{note}
Full per-step guides are in progress. See sub-pages once available.
:::

## Overview

The publishing workflow connects app repo development to canonical repo releases:

::::{grid} 2 2 2 2
:gutter: 2

:::{grid-item-card} **1. Local Validation**
^^^
```bash
apx lint
apx breaking --against=HEAD^
apx semver suggest --against=HEAD^
```
:::

:::{grid-item-card} **2. Tag in App Repo**
^^^
```bash
git tag proto/domain/api/v1/v1.2.3
git push --follow-tags
```
:::

:::{grid-item-card} **3. App CI Publishes**
^^^
```bash
apx publish \
  --canonical-repo=...
```
:::

:::{grid-item-card} **4. Canonical CI Releases**
^^^
- Validates changes
- Creates subdirectory tag
- Publishes packages
:::

::::

## Publishing Strategies

APX uses git subtree for publishing:

### Git Subtree Publishing
- **Preserves commit history** in canonical repo
- **Maintains authorship and timestamps** for full traceability
- **Better for debugging** and understanding API evolution
- **Transparent process** with complete audit trail
- **Industry standard** git tooling

## Subdirectory Tagging

APX uses **subdirectory tags** to version individual APIs:

```
# App repo tags (triggers CI)
proto/payments/ledger/v1/v1.2.3
proto/users/profile/v1/v1.0.1

# Canonical repo tags (created by CI)  
proto/payments/ledger/v1.2.3
proto/users/profile/v1.0.1
```

### Tag Format Rules

- **Schema format prefix**: `proto/`, `openapi/`, `avro/`, etc.
- **Domain organization**: `domain/service/` grouping
- **Version directory**: `/v1/`, `/v2/` for major versions
- **Semantic version**: `/v1.2.3` following semver

## Validation Pipeline

Every publish goes through comprehensive validation:

::::{grid} 1 1 2 2  
:gutter: 3

:::{grid-item-card} **Pre-Publish (App CI)**
^^^
- Schema linting
- Breaking change detection
- Policy compliance
- Version suggestion validation
:::

:::{grid-item-card} **Post-Publish (Canonical CI)**
^^^
- Re-validate all changes
- Verify SemVer compliance
- Check CODEOWNERS approval
- Create official tags
:::

:::{grid-item-card} **Format-Specific Checks**
^^^
- **Proto**: buf lint, buf breaking
- **OpenAPI**: spectral, oasdiff  
- **Avro**: compatibility rules
- **JSON Schema**: diff analysis
:::

:::{grid-item-card} **Policy Enforcement**
^^^
- Ban service annotations (gorm, etc.)
- Approved generators only
- Breaking change justification
- Security vulnerability scans
:::

::::

## Example Publishing Flow

### 1. Prepare for Release

```bash
# Validate locally
apx fetch                    # ensure latest toolchain
apx lint                     # check schema quality
apx breaking --against=HEAD^ # verify compatibility
apx semver suggest --against=HEAD^ # get recommended version bump

# Expected output:
# Suggested version: v1.2.3 (PATCH - backwards compatible bug fixes)
```

### 2. Tag in App Repository

```bash
# Create and push tag (matches suggested version)
git tag proto/payments/ledger/v1/v1.2.3
git push --follow-tags

# This triggers app CI to run apx publish
```

### 3. App CI Automation

App CI runs the publish command:

```bash
apx publish proto/payments/ledger/v1 \
  --version v1.2.3 \
  --lifecycle stable \
  --create-pr
```

### 4. Canonical Repository PR

The PR contains:
- **Schema files** in correct canonical structure
- **go.mod** with proper module path (if not present locally)
- **CHANGELOG.md** with breaking changes summary
- **Validation reports** from lint/breaking checks

### 5. Canonical CI Processing

On PR merge, canonical CI:

1. **Re-validates** all changes
2. **Verifies SemVer** matches content
3. **Creates subdirectory tag**: `proto/payments/ledger/v1.2.3`
4. **Publishes packages** (Maven, wheels, OCI, etc.)
5. **Updates catalog** for discovery

## Error Handling

Common publishing errors and solutions:

### Version Mismatch
```
Error: Tagged version v1.2.3 doesn't match suggested v1.3.0
```
**Solution**: Update tag to match breaking changes, or justify version choice

### Breaking Changes Without Major Bump
```
Error: Breaking changes detected but only minor version bump requested
```
**Solution**: Use major version bump or provide breaking change waiver

### Policy Violations
```
Error: Detected banned annotation: (gorm.column)
```
**Solution**: Remove service-specific annotations from shared schemas

### Merge Conflicts
```
Error: Cannot merge subtree - conflicts in canonical repo
```
**Solution**: Resolve conflicts manually by updating your branch or coordinating with the canonical repo maintainers

## Best Practices

### Pre-Publish Checklist
- [ ] Run full validation locally
- [ ] Coordinate with downstream consumers
- [ ] Document breaking changes
- [ ] Test generated code
- [ ] Verify CODEOWNERS approval

### Tagging Best Practices
- **Tag after merge** to main branch
- **Use annotated tags** with release notes
- **Follow semver strictly** 
- **Coordinate major versions** across teams

### Emergency Releases
For urgent fixes:
1. **Create hotfix branch** from last release
2. **Apply minimal changes**
3. **Tag patch version**
4. **Expedite canonical CI** if needed

## Next Steps

1. Review the [full CLI reference](../cli-reference/index.md) for all publishing flags
2. Read the [Lifecycle Reference](lifecycle.md) for lifecycle states, v0 policy, and compatibility promises
3. See [Release Guardrails](release-guardrails.md) for policy enforcement during releases
4. Understand the [Versioning Strategy](../dependencies/versioning-strategy.md) three-layer model