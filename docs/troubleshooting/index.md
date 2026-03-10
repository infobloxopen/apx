# Troubleshooting

Common issues and solutions when working with APX.


## Common Error Categories

<div class="grid cards" markdown>
-   **Schema Validation**

    ---

    - Buf compilation errors
    - Breaking change detection
    - Policy violations

-   **Versioning**

    ---

    - SemVer mismatches
    - Go module path errors
    - Tag format issues

-   **Releasing**

    ---

    - PR creation errors
    - Permission errors
    - CI failures

-   **Code Generation**

    ---

    - Missing generators
    - Import path issues
    - Output directory problems

-   **Dependencies**

    ---

    - Version resolution
    - Circular dependencies
    - Cache corruption

-   **Configuration**

    ---

    - Invalid YAML syntax
    - Path resolution
    - Environment issues

</div>

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

**Release blocked for SemVer**:
```bash
# Run version suggestion to see recommended bump
apx semver suggest --against=HEAD^
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

### Debug Releasing
```bash
# Dry run release
apx release prepare --dry-run \
  --module-path=internal/apis/proto/domain/service/v1 \
  --canonical-repo=github.com/myorg/apis
```

## Recovery Procedures

### Corrupted Cache
```bash
# Clear APX cache
rm -rf ~/.cache/apx/

# Re-download tools
apx fetch
```

### Failed Release
```bash
# Retry releasing after resolving conflicts
apx release prepare \
  --module-path=internal/apis/proto/domain/service/v1 \
  --canonical-repo=github.com/myorg/apis
apx release submit

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
  - run: apx fetch  # downloads tools
  - run: apx lint
```

**Permission errors**:
```yaml
# Ensure proper GitHub token permissions
permissions:
  contents: read
  pull-requests: write  # for apx release submit
  packages: write       # for package releasing
```

### Container Environments

!!! note "Planned"
    Container-based execution (`--use-container` / `APX_USE_CONTAINER`) is planned for a future release.
    Currently, APX manages reproducible builds via pinned toolchain versions in `apx.lock`.

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
- [Fix release failures](release-failures.md) for CI problems
- [Check the FAQ](faq.md) for frequently asked questions