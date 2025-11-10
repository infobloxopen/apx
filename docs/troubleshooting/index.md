# Troubleshooting

Common issues and solutions when working with APX.

```{toctree}
:maxdepth: 2

common-errors
buf-issues
versioning-problems
publishing-failures
code-generation
faq
```

## Common Error Categories

::::{grid} 1 1 2 3
:gutter: 3

:::{grid-item-card} **Schema Validation**
^^^
- Buf compilation errors
- Breaking change detection
- Policy violations
:::

:::{grid-item-card} **Versioning**
^^^
- SemVer mismatches
- Go module path errors
- Tag format issues
:::

:::{grid-item-card} **Publishing**
^^^
- Subtree merge conflicts
- Permission errors
- CI failures
:::

:::{grid-item-card} **Code Generation**
^^^
- Missing generators
- Import path issues
- Output directory problems
:::

:::{grid-item-card} **Dependencies**
^^^
- Version resolution
- Circular dependencies
- Cache corruption
:::

:::{grid-item-card} **Configuration**
^^^
- Invalid YAML syntax
- Path resolution
- Environment issues
:::

::::

## Quick Fixes

### Schema Won't Compile

**Buf complaints about versioning**:
```bash
# Ensure proto package ends with vN
package myorg.payments.ledger.v1  # ✓ correct

# Ensure files are under .../vN/ directory
internal/apis/proto/payments/ledger/v1/ledger.proto  # ✓ correct
```

**Missing imports**:
```bash
# Check buf.work.yaml includes your directories
apx lint --verbose  # shows detailed buf output
```

### Version Errors

**Go mod path errors**:
- **v1**: module path has **no `/v1`** suffix
- **v2+**: module path **must** end with `/v2`, `/v3`, etc.

```go
// ✓ Correct v1 module
module github.com/myorg/apis/proto/payments/ledger

// ✓ Correct v2 module  
module github.com/myorg/apis/proto/payments/ledger/v2

// ✗ Wrong - v1 with suffix
module github.com/myorg/apis/proto/payments/ledger/v1
```

**Publish blocked for SemVer**:
```bash
# Run version suggestion and update tag
apx version suggest
# Update your tag to match (MAJOR/MINOR/PATCH)
```

### Generated Code Issues

**Generated code committed**:
```bash
# Remove from VCS and add to .gitignore
git rm -r internal/gen/
echo "internal/gen/" >> .gitignore
git commit -m "Remove generated code from VCS"

# Re-run generation in CI only
apx gen go
```

**Import path errors**:
```bash
# Check go_package option in proto files
option go_package = "github.com/myorg/apis/proto/payments/ledger/v1";

# Verify module path matches
grep "module " go.mod
```

## Diagnostic Commands

### Check APX Status
```bash
# Verify installation
apx --version

# Check configuration
apx config validate

# Show current dependencies
apx list dependencies

# Verify toolchain
apx fetch --dry-run
```

### Debug Schema Issues
```bash
# Verbose linting
apx lint --verbose

# Check specific files
buf lint internal/apis/proto/domain/service/v1

# Validate workspace
buf ls-files
```

### Debug Publishing
```bash
# Dry run publish
apx publish --dry-run \
  --module-path=internal/apis/proto/domain/service/v1 \
  --canonical-repo=github.com/myorg/apis

# Check tag format
apx version verify
```

## Recovery Procedures

### Corrupted Cache
```bash
# Clear APX cache
rm -rf ~/.cache/apx/  # or $APX_CACHE_DIR

# Re-download tools
apx fetch
```

### Failed Publish
```bash
# Retry publishing after resolving conflicts
apx publish \
  --module-path=internal/apis/proto/domain/service/v1 \
  --canonical-repo=github.com/myorg/apis

# Check canonical repo is up to date and resolve conflicts manually
```

### Breaking Dependencies
```bash
# Rollback to last working version
apx add proto/payments/ledger/v1@v1.1.0  # known good version

# Regenerate code
apx gen go

# Test and update gradually
```

## Log Analysis

### Enable Verbose Output
```bash
# For single command
apx lint --verbose

# For all commands in session
export APX_VERBOSE=true
apx lint  # now verbose by default
```

### Common Log Patterns

**Schema validation failure**:
```
ERROR buf lint failed:
  internal/apis/proto/payments/ledger/v1/ledger.proto:15:1:
    Package name "payments.ledger.v1" should be "myorg.payments.ledger.v1"
```

**Breaking change detected**:
```
ERROR breaking changes detected:
  FILE_SAME_PACKAGE: proto/payments/ledger/v1/ledger.proto:
    Package changed from "myorg.payments.ledger.v1" to "myorg.payments.ledger.v2"
```

**Version mismatch**:
```
ERROR version verification failed:
  Suggested: v1.3.0 (MINOR - new backwards compatible features)
  Tagged: v1.2.1 (PATCH - backwards compatible bug fixes)
  Reason: New RPC method added requires minor version bump
```

## Environment-Specific Issues

### CI/CD Environments

**Missing tools**:
```yaml
# Ensure apx fetch runs in CI
steps:
  - run: apx fetch --ci  # downloads to well-known location
  - run: apx lint
```

**Permission errors**:
```yaml
# Ensure proper GitHub token permissions
permissions:
  contents: read
  pull-requests: write  # for apx publish
  packages: write       # for package publishing
```

### Container Environments

**Use container execution**:
```bash
# Force container mode
apx --use-container gen go

# Or set environment variable
export APX_USE_CONTAINER=true
apx gen go
```

### Network Restrictions

**Proxy configuration**:
```bash
# APX respects standard proxy environment variables
export HTTP_PROXY=http://proxy.company.com:8080
export HTTPS_PROXY=http://proxy.company.com:8080
apx fetch
```

## Getting Help

### Debug Information
```bash
# Collect debug info for bug reports
apx debug info > debug.txt

# Include in GitHub issues:
# - APX version
# - Operating system  
# - Go version (if relevant)
# - Full error output with --verbose
```

### Community Resources

- **GitHub Issues**: [Report bugs](https://github.com/infobloxopen/apx/issues)
- **Discussions**: [Ask questions](https://github.com/infobloxopen/apx/discussions)
- **Documentation**: [Browse guides](https://apx.infoblox.com)

### Internal Support

If using APX in an organization:
- Check with your **API governance team**
- Review **organization-specific runbooks**
- Consult **#api-platform** Slack channel (if available)

## Next Steps

- [Review common errors](common-errors.md) for your specific issue
- [Debug Buf integration](buf-issues.md) for Protocol Buffer problems
- [Resolve versioning problems](versioning-problems.md) for Go module issues
- [Fix publishing failures](publishing-failures.md) for CI problems
- [Check the FAQ](faq.md) for frequently asked questions