# End-to-End Integration Tests

This directory contains the E2E integration test suite for APX, which validates the complete schema publishing workflow using a real k3d cluster and Gitea instance.

## Overview

The E2E test suite provides:
- **k3d cluster management** - Ephemeral Kubernetes clusters for testing
- **Gitea deployment** - Git hosting simulation in k3d
- **Complete workflow testing** - From `apx init` through `apx publish` with PR validation
- **Realistic fixtures** - Canonical and app repositories with Proto schemas

## Architecture

```
tests/e2e/
├── main_test.go           # Test runner with k3d + Gitea orchestration
├── k3d/                   # k3d cluster management
│   ├── cluster.go         # Cluster lifecycle (create, delete, wait)
│   ├── config.go          # Configuration and manifest generation
│   └── cleanup.go         # Resource cleanup utilities
├── gitea/                 # Gitea lifecycle and API
│   ├── lifecycle.go       # Deploy, readiness, health checks
│   └── client.go          # REST API client (repos, PRs, tags, users)
├── testhelpers/           # Test utilities
│   ├── git.go             # Git operations wrapper
│   ├── apx.go             # APX command wrappers
│   ├── assertions.go      # Custom test assertions
│   └── environment.go     # Test workspace manager
└── fixtures/              # Test data
    ├── canonical-repo/    # Shared schemas (Money, Address)
    ├── app1-payment/      # Payment service schemas
    └── app2-user/         # User service schemas
```

## Prerequisites

### Required Tools
```bash
# Install k3d (lightweight Kubernetes)
make install-e2e-deps

# Or manually:
brew install k3d kubectl   # macOS
# OR
curl -s https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh | bash  # Linux
```

### System Requirements
- Docker running
- ~2GB free RAM for k3d cluster
- Ports 3000 available for Gitea

## Running Tests

### Quick Start
```bash
# Run all E2E tests (creates k3d cluster + Gitea)
make test-e2e

# Or with go test directly
E2E_ENABLED=1 go test -v ./tests/e2e -run TestE2E
```

### Individual Test Scenarios
```bash
# Run specific testscript
E2E_ENABLED=1 go test -v ./tests/e2e -run TestE2E -testscript.filter=e2e_basic_setup

# Run with verbose output
E2E_ENABLED=1 go test -v ./tests/e2e -run TestE2E -testscript.verbose
```

### Debug Mode
```bash
# Keep environment running after test (for debugging)
E2E_DEBUG=1 E2E_ENABLED=1 go test -v ./tests/e2e -run TestE2E

# The test will print:
# - Gitea URL: http://localhost:3000
# - Admin token: <token>
# - Cluster: apx-e2e-<timestamp>

# Cleanup manually when done:
make clean-e2e
```

### Keep Failed Test Artifacts
```bash
# Keep temporary workspace when test fails
E2E_KEEP_FAILED=1 E2E_ENABLED=1 go test -v ./tests/e2e -run TestE2E

# Location will be printed on failure:
# Test failed - keeping workspace: /tmp/apx-e2e-<prefix>-<id>
```

## Test Scenarios

### Phase 3: Basic Workflow (User Story 1) ✅
- `e2e_basic_setup.txt` - Validates k3d + Gitea are working
- `e2e_complete_workflow.txt` - Complete publishing workflow:
  - Canonical repository initialization
  - App repository creation with dependencies
  - Schema publication (PRs to canonical)
  - Dependency consumption (`apx add`, `apx gen`)
  - Overlay validation (import path resolution)

### Phase 4: Cross-Repository Dependencies (User Story 2) ✅
- `e2e_cross_repo_deps.txt` - App2 consumes App1's published schema and publishes its own
- `e2e_catalog_validation.txt` - `apx search` shows both payment and user APIs

### Phase 5: Breaking Change Detection (User Story 3) ✅
- `e2e_breaking_detection.txt` - Detects breaking changes (removed field) via `apx breaking`
- `e2e_non_breaking_changes.txt` - Allows non-breaking changes (added optional field)
- `e2e_major_version_bump.txt` - Allows breaking changes in new major version (v2)

### Phase 6: Git History Preservation (User Story 4) ✅
- `e2e_git_history.txt` - Verifies commit history and authorship through subtree publish

### Phase 7: Edge Cases ✅
- `e2e_gitea_unreachable.txt` - Gitea unavailability error handling
- `e2e_existing_pr.txt` - PR update when PR already exists
- `e2e_corrupted_git_history.txt` - Git subtree split failure
- `e2e_circular_deps.txt` - Circular dependency detection
- `e2e_duplicate_tag.txt` - Tag conflict error handling
- `e2e_overlay_deletion.txt` - `apx sync` after manual overlay deletion
- `e2e_concurrent_publish.txt` - Multiple apps publishing to same module path
- `e2e_codeowners.txt` - CODEOWNERS enforcement validation

**Total: 16 testscript scenarios covering 18 functional requirements**

## Writing New Tests

### Testscript Format
```bash
# tests/testdata/script/e2e/my_test.txt

# Only run when E2E_ENABLED=1
[!exec:e2e] skip 'E2E tests only run with E2E_ENABLED=1'

# Environment variables available:
# - GITEA_URL: http://localhost:3000
# - GITEA_TOKEN: Admin API token
# - E2E_CLUSTER: Cluster name (apx-e2e-<timestamp>)

# Test commands
exec apx init canonical my-canonical --dir=canonical
exists canonical/apx.yaml

# Git operations
cd canonical
exec git config user.name 'Test Bot'
exec git config user.email 'test@example.com'
exec git add .
exec git commit -m 'Initial commit'

# Assertions
stdout 'Initial commit'
grep 'type: canonical' apx.yaml
```

### Using Test Helpers (Go tests)
```go
func TestMyE2E(t *testing.T) {
    // Create test environment
    env := testhelpers.NewTestEnvironment(t, "my-test")
    
    // Initialize git repo
    repo, err := testhelpers.InitRepo(env.Path("canonical"))
    testhelpers.AssertNoError(t, err, "init repo")
    
    // Configure git
    err = repo.ConfigureUser("Test Bot", "test@example.com")
    testhelpers.AssertNoError(t, err, "configure git")
    
    // Write files
    err = repo.WriteFile("apx.yaml", "type: canonical\nname: test")
    testhelpers.AssertNoError(t, err, "write file")
    
    // Commit
    err = repo.Add(".")
    err = repo.Commit("Initial commit")
    testhelpers.AssertNoError(t, err, "commit")
    
    // Assert git history
    testhelpers.AssertGitHistory(t, repo, []string{"Initial commit"})
}
```

## Cleanup

```bash
# Remove all E2E test resources
make clean-e2e

# Or manually:
k3d cluster delete --all
docker ps -a --filter "name=k3d-apx-e2e-" -q | xargs docker rm -f
docker volume ls --filter "name=k3d-apx-e2e-" -q | xargs docker volume rm -f
```

## Troubleshooting

### Tests are skipped
```
SKIP: E2E tests only run with E2E_ENABLED=1
```
**Solution**: Set `E2E_ENABLED=1` environment variable

### k3d cluster creation fails
```
failed to create k3d cluster: permission denied
```
**Solution**: Ensure Docker is running and your user has Docker permissions

### Gitea deployment timeout
```
timeout waiting for Gitea to be ready
```
**Solutions**:
- Increase timeout (system may be slow)
- Check Docker resources (need ~2GB RAM)
- Check logs: `kubectl logs -n default -l app=gitea`

### Port 3000 already in use
```
failed to create cluster: port 3000 already allocated
```
**Solution**: Stop service using port 3000 or modify `giteaPort` in test code

### Tests hang indefinitely
**Solution**: Set timeout: `go test -timeout=10m ./tests/e2e`

## CI Integration

### GitHub Actions Example
```yaml
name: E2E Tests

on:
  push:
    branches: [main]
  pull_request:

jobs:
  e2e:
    runs-on: ubuntu-latest
    timeout-minutes: 20
    
    steps:
      - uses: actions/checkout@v4
      
      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'
      
      - name: Install k3d
        run: make install-e2e-deps
      
      - name: Run E2E tests
        run: make test-e2e
        env:
          E2E_ENABLED: 1
      
      - name: Cleanup on failure
        if: failure()
        run: make clean-e2e
```

## Performance

- **Cluster creation**: ~30-60s (one-time per test run)
- **Gitea deployment**: ~30-45s (one-time per test run)
- **Per-test overhead**: ~1-2s (testscript setup)
- **Total test run**: ~2-5 minutes (setup + all scenarios)

## Design Decisions

### Why k3d?
- Lightweight (runs in Docker)
- Fast cluster creation (~30s)
- No external dependencies
- Works in CI environments

### Why Gitea?
- Lightweight Git hosting
- Compatible with Git/GitHub APIs
- Runs in k3d cluster
- No external services needed

### Why testscript?
- Declarative test format
- Easy to read and maintain
- Built-in file operations
- Shell command execution

## Contributing

When adding new E2E tests:

1. **Use testscript format** for integration scenarios
2. **Add to appropriate phase** in tasks.md
3. **Include [!exec:e2e] condition** at the top
4. **Document in this README** under "Test Scenarios"
5. **Verify cleanup** - tests should not leave resources
6. **Test in debug mode** - verify E2E_DEBUG works

## References

- [Testscript Documentation](https://pkg.go.dev/github.com/rogpeppe/go-internal/testscript)
- [k3d Documentation](https://k3d.io/)
- [Gitea API Documentation](https://docs.gitea.io/en-us/api-usage/)
- [APX Specification](../../specs/003-e2e-integration-suite/spec.md)
