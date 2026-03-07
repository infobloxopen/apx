# Quick Start: Running E2E Integration Tests

**Feature**: 003-e2e-integration-suite  
**Date**: November 22, 2025  
**Purpose**: Developer guide for running end-to-end integration tests locally and in CI

---

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Installation](#installation)
3. [Running Tests](#running-tests)
4. [Debugging](#debugging)
5. [CI Integration](#ci-integration)
6. [Troubleshooting](#troubleshooting)

---

## Prerequisites

### Required Software

- **Docker**: Docker Desktop (macOS/Windows) or Docker Engine (Linux)
  - Version: 20.10+ required
  - macOS: Docker Desktop 4.0+ (includes k3s support)
  - Windows: Docker Desktop 4.0+ with WSL2 backend
  - Linux: Docker Engine 20.10+

- **Go**: Version 1.24 or later
  - Check: `go version`
  - Install: https://golang.org/dl/

- **Make**: GNU Make for running targets
  - macOS: Pre-installed or via Xcode Command Line Tools
  - Linux: `sudo apt-get install build-essential`
  - Windows: Use WSL2 Ubuntu

### Platform-Specific Notes

**macOS**:
- Docker Desktop must be running before tests
- Rosetta 2 recommended for M1/M2 Macs
- Tests run natively, no special configuration

**Linux**:
- Fastest performance (native Docker containers)
- Primary CI platform (GitHub Actions ubuntu-latest)
- No special configuration needed

**Windows**:
- Must use WSL2 with Ubuntu distribution
- Install Docker Desktop with WSL2 backend enabled
- Run all commands inside WSL2 shell (not PowerShell)
- Example:
  ```powershell
  # In PowerShell
  wsl
  
  # Now in WSL2 Ubuntu
  cd /mnt/c/Users/YourName/apx
  make test-e2e
  ```

---

## Installation

### 1. Install E2E Dependencies

From the repository root, run:

```bash
make install-e2e-deps
```

This installs:
- **k3d**: Lightweight Kubernetes (k3s) cluster manager
- **kubectl**: Kubernetes CLI (for debugging)

**Verification**:
```bash
k3d version
# Expected output: k3d version v5.x.x

kubectl version --client
# Expected output: Client Version: v1.x.x
```

### 2. Pre-pull Gitea Image (Optional but Recommended)

Speeds up first test run:

```bash
docker pull gitea/gitea:1.22
```

---

## Running Tests

### Full E2E Test Suite

Run all end-to-end test scenarios:

```bash
make test-e2e
```

**What this does**:
1. Builds the `apx` binary
2. Creates a k3d cluster (`apx-e2e-<timestamp>`)
3. Deploys Gitea to the cluster
4. Runs all testscript scenarios in `testdata/script/e2e/`
5. Cleans up cluster and containers

**Expected Duration**: ~3-4 minutes for all scenarios

**Expected Output**:
```
==> Building apx binary...
==> Creating k3d cluster...
==> Deploying Gitea...
==> Running E2E tests...
PASS: e2e_complete_workflow.txt (12.34s)
PASS: e2e_cross_repo_deps.txt (15.67s)
PASS: e2e_git_history.txt (10.23s)
...
PASS: All E2E tests passed
==> Cleaning up...
```

### Run Specific Test Scenario

Run a single testscript scenario:

```bash
go test ./tests/e2e -run TestE2E/e2e_complete_workflow
```

**Available scenarios** (in `testdata/script/e2e/`):
- `e2e_complete_workflow` - Full publish/consume cycle (P1)
- `e2e_cross_repo_deps` - App2 consumes App1, publishes own API (P2)
- `e2e_git_history` - Verify git subtree history preservation (P2)
- `e2e_breaking_detection` - Breaking change validation (P3)
- `e2e_concurrent_publish` - Concurrent publication edge case
- `e2e_existing_pr` - PR already exists edge case
- `e2e_circular_deps` - Circular dependency detection
- `e2e_codeowners` - CODEOWNERS enforcement

### Run Tests with Verbose Output

See detailed test execution logs:

```bash
go test ./tests/e2e -v -run TestE2E
```

### Run Unit Tests for E2E Infrastructure

Test the test infrastructure itself:

```bash
go test ./tests/e2e/gitea -v
go test ./tests/e2e/k3d -v
go test ./tests/e2e/testhelpers -v
```

---

## Debugging

### Debug Mode: Keep Gitea Running

Prevent automatic cleanup to inspect state:

```bash
E2E_DEBUG=1 make test-e2e
```

**What this does**:
- Test runs normally but doesn't tear down cluster/Gitea
- Prints Gitea URL and credentials
- Allows manual inspection

**Example output**:
```
==> E2E_DEBUG=1 detected - cluster will remain running
==> Gitea URL: http://localhost:3000
==> Admin username: gitea_admin
==> Admin token: <token>
==> Cluster name: apx-e2e-20251122-143052

PASS: All tests passed

==> DEBUG MODE: Cluster NOT cleaned up
==> To access Gitea web UI: http://localhost:3000
==> To cleanup manually: make clean-e2e
```

### Access Gitea Web UI

While tests are running (or in debug mode):

1. Open browser to http://localhost:3000
2. Login with:
   - Username: `gitea_admin`
   - Password: `admin123` (default test password)
3. Browse repositories, pull requests, tags

### Inspect Running Cluster

```bash
# List k3d clusters
k3d cluster list

# Get cluster info
k3d cluster get apx-e2e-<timestamp>

# View Gitea logs
kubectl logs -n default deployment/gitea

# Access Gitea container
kubectl exec -it deployment/gitea -- /bin/sh
```

### View Test Artifacts

After a test run:

```bash
# Testscript creates temporary directories
ls -la /tmp/go-build*/testscript*/

# View test logs
cat /tmp/go-build*/testscript*/e2e_complete_workflow/work/stdout.txt
```

---

## CI Integration

### GitHub Actions Workflow

Add E2E tests to `.github/workflows/test.yml`:

```yaml
name: Tests

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  e2e-tests:
    name: End-to-End Tests
    runs-on: ubuntu-latest
    timeout-minutes: 10
    
    steps:
      - name: Checkout code
        uses: actions/checkout@v4
      
      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.24'
      
      - name: Install E2E dependencies
        run: make install-e2e-deps
      
      - name: Pull Gitea image (cache)
        run: docker pull gitea/gitea:1.22
      
      - name: Run E2E tests
        run: make test-e2e
      
      - name: Cleanup on failure
        if: failure()
        run: make clean-e2e
```

### macOS CI (Optional)

E2E tests on macOS runners:

```yaml
  e2e-tests-macos:
    name: End-to-End Tests (macOS)
    runs-on: macos-latest
    timeout-minutes: 15  # Slower due to Docker Desktop startup
    
    steps:
      - uses: actions/checkout@v4
      
      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'
      
      # Docker Desktop not pre-installed on macOS runners
      - name: Install Docker Desktop
        run: |
          brew install --cask docker
          open -a Docker
          # Wait for Docker to start
          while ! docker info >/dev/null 2>&1; do sleep 1; done
      
      - name: Install E2E dependencies
        run: make install-e2e-deps
      
      - name: Run E2E tests
        run: make test-e2e
```

---

## Troubleshooting

### Problem: k3d cluster creation fails

**Symptoms**:
```
ERRO[0000] Failed to create cluster 'apx-e2e-...'
ERRO[0000] Port '3000' is already allocated
```

**Solution**:
```bash
# Find conflicting cluster
k3d cluster list

# Delete old cluster
k3d cluster delete apx-e2e-<old-timestamp>

# Or delete all e2e clusters
make clean-e2e

# Retry test
make test-e2e
```

### Problem: Gitea container fails to start

**Symptoms**:
```
ERRO[0030] Gitea readiness check timed out
```

**Solution**:
```bash
# Check container logs
kubectl logs -n default deployment/gitea

# Common causes:
# 1. Port conflict - delete old cluster
k3d cluster delete apx-e2e-*

# 2. Image pull failure - manually pull
docker pull gitea/gitea:1.22

# 3. Resource constraints - close other apps
```

### Problem: Gitea slow to start

**Symptoms**:
- Tests take >5 minutes
- Gitea readiness check takes >30 seconds

**Solutions**:
```bash
# Pre-pull Gitea image to cache layers
docker pull gitea/gitea:1.22

# On macOS: Increase Docker Desktop resource limits
# Docker Desktop → Settings → Resources
# - CPUs: 4+ recommended
# - Memory: 8GB+ recommended
```

### Problem: Tests fail with "permission denied"

**Symptoms**:
```
Error: permission denied while trying to connect to Docker daemon socket
```

**Solution (Linux)**:
```bash
# Add user to docker group
sudo usermod -aG docker $USER

# Re-login or run
newgrp docker

# Verify
docker ps
```

**Solution (Windows)**:
```powershell
# Ensure WSL2 integration is enabled
# Docker Desktop → Settings → Resources → WSL Integration
# Enable for your Ubuntu distribution
```

### Problem: Orphaned resources after failed test

**Symptoms**:
```
# Old clusters remain
k3d cluster list
# apx-e2e-20251122-120000 (from failed test)
# apx-e2e-20251122-143052 (current)
```

**Solution**:
```bash
# Cleanup all E2E resources
make clean-e2e

# Verify cleanup
k3d cluster list  # Should show no apx-e2e-* clusters
docker ps | grep gitea  # Should show no gitea containers
```

### Problem: Test scenario fails intermittently (flaky test)

**Investigation**:
```bash
# Run scenario 10 times to reproduce
for i in {1..10}; do
  echo "Run $i"
  go test ./tests/e2e -run TestE2E/e2e_complete_workflow || break
done
```

**Common causes**:
1. **Race condition**: Gitea not fully ready when test starts
   - **Fix**: Increase readiness check timeout in `tests/e2e/gitea/lifecycle.go`
   
2. **Timing-dependent assertion**: Test assumes operation completes immediately
   - **Fix**: Add polling/retry logic to assertion
   
3. **Shared state pollution**: Previous test scenario didn't clean up
   - **Fix**: Ensure repository cleanup between scenarios

**Report flaky tests**:
- Include scenario name and failure frequency
- Attach full test output (`-v` flag)
- Note platform (Linux/macOS/Windows)

### Problem: Windows (WSL2) path issues

**Symptoms**:
```
Error: /mnt/c/Users/.../apx: no such file or directory
```

**Solution**:
```bash
# Clone repo inside WSL2 filesystem (not /mnt/c)
cd ~
git clone https://github.com/infobloxopen/apx.git
cd apx

# Run tests from WSL2 path
make test-e2e
```

### Problem: Network connectivity to Gitea

**Symptoms**:
```
Error: dial tcp 127.0.0.1:3000: connect: connection refused
```

**Solution**:
```bash
# Check if Gitea pod is running
kubectl get pods

# Check if port forwarding is active
netstat -tuln | grep 3000

# Restart cluster
k3d cluster delete apx-e2e-*
make test-e2e
```

---

## Manual Cleanup Commands

If tests leave orphaned resources:

```bash
# Delete all E2E clusters
k3d cluster delete $(k3d cluster list -o json | jq -r '.[].name | select(startswith("apx-e2e-"))')

# Delete Gitea containers
docker rm -f $(docker ps -aq --filter "name=k3d-apx-e2e")

# Delete Gitea volumes
docker volume rm $(docker volume ls -q --filter "name=k3d-apx-e2e")

# All-in-one cleanup (use with caution)
make clean-e2e
```

---

## Performance Benchmarks

Expected timings on different platforms:

| Platform | Cluster Setup | Gitea Startup | Per Scenario | Full Suite (10 scenarios) |
|----------|---------------|---------------|--------------|---------------------------|
| Linux (Ubuntu 22.04) | ~20s | ~12s | ~10-15s | ~3min |
| macOS (Intel) | ~25s | ~15s | ~12-18s | ~4min |
| macOS (M1/M2) | ~25s | ~15s | ~12-18s | ~4min |
| Windows (WSL2) | ~30s | ~20s | ~15-20s | ~5min |

**Note**: First run is slower due to Docker image pulls. Subsequent runs benefit from caching.

---

## Next Steps

- ✅ E2E tests installed and running
- ⏳ Proceed to implementation (`/speckit.tasks`)
- ⏳ Write first testscript scenario
- ⏳ Implement test infrastructure (k3d, Gitea helpers)

---

## References

- [k3d Documentation](https://k3d.io/)
- [Gitea API Reference](https://docs.gitea.com/api/)
- [Testscript Package](https://pkg.go.dev/github.com/rogpeppe/go-internal/testscript)
- [APX Documentation](https://infobloxopen.github.io/apx/)
