# API Dependencies Management

APX provides powerful tools for discovering, adding, and managing API dependencies across your organization. All dependency management is based on the canonical repository as the source of truth.

```{toctree}
:maxdepth: 2

discovery
adding-dependencies
code-generation
updates-and-upgrades
versioning-strategy
```

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
# Search APIs by keywords
apx search payments ledger

# List all available APIs
apx list apis

# Show API details and versions
apx show proto/payments/ledger/v1
```

### Dependency Management
```bash
# Add specific version
apx add proto/payments/ledger/v1@v1.2.3

# Update to latest compatible
apx update proto/payments/ledger/v1

# Upgrade to new major version
apx upgrade proto/payments/ledger/v2@v2.0.0
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
dependencies:
  - api: proto/payments/ledger/v1
    version: v1.2.3
    resolved_commit: abc1234
    generators:
      - go
      - python
  - api: proto/users/v1  
    version: v1.0.5
    resolved_commit: def5678
    generators:
      - go

toolchain:
  buf: v1.28.1
  protoc: v24.4
  protoc-gen-go: v1.31.0
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