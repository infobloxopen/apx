# API Dependencies Management

APX provides powerful tools for discovering, adding, and managing API dependencies across your organization.

:::{note}
Full per-topic guides are in progress. See sub-pages once available.
:::

## Overview

Dependency management workflow:

::::{grid} 1 1 2 2
:gutter: 3

:::{grid-item-card} **Discover**
^^^
- Search organization APIs
- Browse by domain or format
- View available versions
- Understand compatibility
:::

:::{grid-item-card} **Add & Pin**
^^^
- Add specific versions
- Pin in `apx.lock`
- Record codegen preferences  
- Manage transitive deps
:::

:::{grid-item-card} **Generate**
^^^
- Multi-language code generation
- Never commit generated code
- Reproducible builds
- Format-specific generators
:::

:::{grid-item-card} **Update**
^^^
- Compatible updates (patch/minor)
- Major version upgrades
- Breaking change analysis
- Migration assistance
:::

::::

## Quick Reference

### Discovery Commands
```bash
# Search APIs by keyword
apx search payments

# Filter by format, lifecycle, or domain
apx search --format=proto --lifecycle=stable
apx search --domain=payments

# Show full identity and catalog data for an API
apx show proto/payments/ledger/v1
```

```{admonition} Planned — not yet available
:class: note
`apx list apis` is planned for a future release.
```

### Dependency Management
```bash
# Add specific version
apx add proto/payments/ledger/v1@v1.2.3
```

```{admonition} Planned — not yet available
:class: note
`apx update` and `apx upgrade` are planned for a future release.
To pin a newer version, re-add the dependency: `apx add proto/payments/ledger/v1@v1.3.0`
```

### Code Generation
```bash
# Generate for specific language
apx gen go
apx gen python  
apx gen java

# Generate all configured languages
apx gen
```

## Lock File Management

APX uses `apx.lock` to ensure reproducible builds:

```yaml
# apx.lock
version: 1
toolchains:
  buf:
    version: v1.28.1
    checksum: "sha256:abc123..."
dependencies:
  proto/payments/ledger/v1:
    repo: github.com/myorg/apis
    ref: proto/payments/ledger/v1.2.3
    modules:
      - proto/payments/ledger
  proto/users/v1:
    repo: github.com/myorg/apis
    ref: proto/users/v1.0.5
    modules:
      - proto/users
```

## Generation Output Structure

Generated code follows a consistent pattern:

```
internal/gen/
├── go/
│  ├── proto_payments_ledger_v1@v1.2.3/
│  │  ├── ledger.pb.go
│  │  └── ledger_grpc.pb.go
│  └── proto_users_v1@v1.0.5/
│     ├── users.pb.go
│     └── users_grpc.pb.go
├── python/
│  ├── proto_payments_ledger_v1@v1.2.3/
│  │  └── ledger_pb2.py
│  └── proto_users_v1@v1.0.5/
│     └── users_pb2.py
└── java/
   └── proto_payments_ledger_v1@v1.2.3/
      └── com/myorg/payments/ledger/v1/
```

## Best Practices

### Version Selection
- **Pin specific versions** in production
- **Use latest patch** for development
- **Test major upgrades** in feature branches
- **Coordinate breaking changes** across teams

### Code Generation
- **Never commit** generated code (use `.gitignore`)
- **Regenerate in CI** for consistency  
- **Pin generator versions** via `apx.lock`
- **Validate imports** after generation

### Dependency Hygiene
- **Review dependencies** regularly
- **Remove unused** dependencies
- **Document breaking changes** when upgrading
- **Test compatibility** before upgrading dependents

## Next Steps

1. [Learn API discovery techniques](discovery.md)
2. [Add your first dependencies](adding-dependencies.md)
3. [Set up code generation](code-generation.md)
4. [Understand update strategies](updates-and-upgrades.md)
5. [Review versioning best practices](versioning-strategy.md)