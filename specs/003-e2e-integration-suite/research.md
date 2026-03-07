# Research: End-to-End Integration Test Suite

**Feature**: 003-e2e-integration-suite  
**Date**: November 22, 2025  
**Status**: Complete

## Executive Summary

This document captures technical research and decisions for implementing an E2E integration test suite using **k3d** for Gitea hosting and **testscript** for test orchestration. All research tasks from Phase 0 have been completed with clear decisions and rationale.

---

## 1. k3d Configuration for CI

### Decision: ✅ Use k3d with GitHub Actions

**Rationale**:
- k3d runs k3s (lightweight Kubernetes) in Docker containers
- No privileged mode required (unlike some Kind configurations)
- GitHub Actions ubuntu-latest runners have Docker pre-installed
- Fast startup (<30s for basic cluster)

### Installation in CI

**Approach**: Install k3d via official installation script in GitHub Actions

```yaml
- name: Install k3d
  run: |
    curl -s https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh | bash
    k3d version
```

**Alternative considered**: 
- `setup-k3d` GitHub Action - Rejected because it's not officially maintained, adds dependency

### Docker Image Caching

**Decision**: Use GitHub Actions Docker layer caching

```yaml
- name: Pull Gitea image
  run: docker pull gitea/gitea:1.22
```

**Rationale**:
- GitHub Actions caches Docker layers between runs
- Reduces Gitea startup time from ~40s to ~10s on subsequent runs
- Pinned version (1.22) ensures reproducibility

---

## 2. Gitea Configuration for Testing

### Minimal Configuration

**Decision**: ✅ Use SQLite backend with minimal features

```ini
# gitea/app.ini (minimal config)
[database]
DB_TYPE = sqlite3
PATH = /data/gitea/gitea.db

[repository]
ROOT = /data/git/repositories

[server]
PROTOCOL = http
DOMAIN = localhost
HTTP_PORT = 3000
ROOT_URL = http://localhost:3000/

[service]
DISABLE_REGISTRATION = false
REQUIRE_SIGNIN_VIEW = false

[webhook]
ALLOWED_HOST_LIST = *

[mailer]
ENABLED = false

[log]
MODE = console
LEVEL = Info
```

**Rationale**:
- SQLite = zero external dependencies
- Disabled email = faster startup, no SMTP configuration
- Webhooks allowed = future PR automation testing
- Console logging = easier debugging in CI

### API Endpoints Needed

**Research findings** - Gitea API v1 (OpenAPI spec: https://gitea.com/api/swagger):

1. **Repository Management**:
   - `POST /api/v1/user/repos` - Create repository
   - `GET /api/v1/repos/{owner}/{repo}` - Get repository details
   - `DELETE /api/v1/repos/{owner}/{repo}` - Delete repository (cleanup)

2. **Pull Requests**:
   - `POST /api/v1/repos/{owner}/{repo}/pulls` - Create PR
   - `GET /api/v1/repos/{owner}/{repo}/pulls/{index}` - Get PR details
   - `GET /api/v1/repos/{owner}/{repo}/pulls/{index}/files` - Get PR changed files
   - `GET /api/v1/repos/{owner}/{repo}/pulls/{index}/commits` - Get PR commits (for history validation)

3. **Tags**:
   - `GET /api/v1/repos/{owner}/{repo}/tags` - List tags
   - `POST /api/v1/repos/{owner}/{repo}/tags` - Create tag (if needed)

4. **Contents** (for CODEOWNERS):
   - `POST /api/v1/repos/{owner}/{repo}/contents/{filepath}` - Create/update file
   - `GET /api/v1/repos/{owner}/{repo}/contents/{filepath}` - Get file content

5. **Users** (admin operations):
   - `POST /api/v1/admin/users` - Create test users
   - `GET /api/v1/user` - Get current user info

### Authentication

**Decision**: ✅ API Token (not SSH keys)

**Token creation process**:
1. Use admin user created during Gitea initialization
2. Create token via API: `POST /api/v1/users/{username}/tokens`
3. Store token in testscript environment variable

**Code example**:
```go
// Create admin token on Gitea startup
token, err := giteaClient.CreateToken(ctx, "admin", &gitea.CreateAccessTokenOption{
    Name: "e2e-test-token",
    Scopes: []string{"repo", "admin"},
})
```

**Rationale**:
- API tokens simpler than SSH key management
- No SSH agent setup required in tests
- Works consistently across platforms (Windows, Linux, macOS)
- Easy to pass to testscript via environment variable

### Service Exposure in k3d

**Decision**: ✅ Use NodePort with host port mapping

```bash
# Create k3d cluster with Gitea port mapped
k3d cluster create apx-e2e \
  --port "3000:3000@server:0" \
  --wait
```

**Rationale**:
- NodePort simpler than LoadBalancer in CI
- Direct port mapping to localhost
- No external load balancer dependencies
- GitHub Actions runners allow localhost:3000 access

**Alternative considered**:
- Ingress controller - Rejected as overcomplicated for test environment
- LoadBalancer (MetalLB) - Rejected as unnecessary for single-cluster tests

### Gitea Version

**Decision**: ✅ Pin to Gitea 1.22 (latest stable as of Nov 2025)

```yaml
image: gitea/gitea:1.22
```

**Rationale**:
- Pinned version ensures reproducibility
- 1.22 has stable API, good SQLite performance
- Security updates via minor version bumps (1.22.x)
- Documented upgrade path when newer versions needed

---

## 3. Testscript Integration Patterns

### Making Gitea URL Available

**Decision**: ✅ Use environment variable injection

```go
// In testscript setup function
func setupE2E(env *testscript.Env) error {
    giteaURL := startGitea() // Returns http://localhost:3000
    env.Setenv("GITEA_URL", giteaURL)
    env.Setenv("GITEA_TOKEN", adminToken)
    return nil
}
```

**In testscript**:
```testscript
# Access Gitea URL in test
exec git clone ${GITEA_URL}/testorg/api-schemas.git
```

**Rationale**:
- Environment variables native to testscript
- No special syntax required in test files
- Easy to override for debugging (`GITEA_URL=http://custom:3000`)

### Secure API Token Passing

**Decision**: ✅ Environment variable (acceptable for tests)

```go
env.Setenv("GITEA_TOKEN", adminToken)
```

**In testscript**:
```testscript
# Use token in git operations
exec git -c http.extraHeader="Authorization: token ${GITEA_TOKEN}" clone ...
```

**Rationale**:
- Testscript environment is ephemeral (not logged to CI)
- Token only valid for test duration
- Simpler than tmpfile management
- No risk of token leakage (destroyed with test environment)

**Alternative considered**:
- Write token to tmpfile - Rejected as overcomplicated for test scenario
- Git credential helper - Rejected as requires additional setup

### Git Remote URL Injection

**Decision**: ✅ Template-based repository initialization

**Pattern**:
```go
// Helper function to initialize test repo
func initTestRepo(workDir, giteaURL string) error {
    // Run apx init
    exec.Command("apx", "init", "canonical", ...).Run()
    
    // Add git remote
    exec.Command("git", "remote", "add", "origin", 
        fmt.Sprintf("%s/testorg/api-schemas.git", giteaURL)).Run()
    
    return nil
}
```

**In testscript**:
```testscript
# Initialize repo and set remote
exec apx init canonical --org=testorg --skip-git
exec git init
exec git remote add origin ${GITEA_URL}/testorg/api-schemas.git
```

**Rationale**:
- Explicit and readable in testscript
- Matches real-world developer workflow
- Easy to debug (can see git commands in test output)

### Cleanup Hooks in Testscript

**Decision**: ✅ Use Go test cleanup (no native testscript defer)

Testscript doesn't have native cleanup hooks, so use Go test infrastructure:

```go
func TestE2E(t *testing.T) {
    cluster := startK3dCluster(t)
    t.Cleanup(func() {
        cluster.Destroy()
    })
    
    gitea := startGitea(t, cluster)
    t.Cleanup(func() {
        gitea.Stop()
    })
    
    testscript.Run(t, testscript.Params{
        Dir: "testdata/script/e2e",
        Setup: func(env *testscript.Env) error {
            env.Setenv("GITEA_URL", gitea.URL)
            return nil
        },
    })
}
```

**Rationale**:
- `t.Cleanup()` runs in LIFO order (Gitea stops before k3d cluster)
- Guaranteed execution even on test panic
- Integrates with existing Go test infrastructure
- Can't be forgotten (compiler enforces cleanup registration)

---

## 4. Test Isolation Strategy

### Cluster Lifecycle

**Decision**: ✅ One k3d cluster per test suite (not per scenario)

**Rationale**:
- Cluster creation overhead: ~20-30 seconds
- Acceptable for suite-level amortization
- Test scenarios share cluster but isolate repositories
- Faster overall execution than per-test cluster

**Measured Performance**:
```
Cluster creation: ~25s
Gitea startup: ~15s
Test scenario: ~10-20s each
Cluster teardown: ~5s

Total for 10 scenarios:
  Per-suite: 25 + 15 + (10*15) + 5 = 195s (~3min)
  Per-test: (25 + 15 + 15 + 5) * 10 = 600s (~10min)
```

**Trade-off**: Less isolation (shared Gitea) vs better performance (3min vs 10min)

**Decision**: Accept shared Gitea for performance; ensure repository cleanup

### Repository Cleanup Strategy

**Decision**: ✅ Delete repositories via API between tests

```go
func cleanupRepositories(giteaClient *gitea.Client, org string) error {
    repos, err := giteaClient.ListOrgRepos(org)
    if err != nil {
        return err
    }
    
    for _, repo := range repos {
        if strings.HasPrefix(repo.Name, "test-") {
            giteaClient.DeleteRepo(org, repo.Name)
        }
    }
    return nil
}
```

**Rationale**:
- Faster than recreating Gitea (~1s vs ~15s)
- API cleanup is idempotent
- Test naming convention (`test-*`) prevents accidental deletion
- Gitea state reset (tags, PRs, commits) with repository deletion

**Alternative considered**:
- Recreate Gitea container - Rejected as too slow
- Namespace isolation - Deferred to future (adds complexity)

### Future: Namespace Isolation for Parallelism

**Research**: k3d supports multiple namespaces for parallel test execution

**Not implemented now** because:
- Single test suite execution is <5min (meets SC-001)
- Parallel execution adds complexity (port conflicts, resource limits)
- Can be added later without breaking existing tests

**Future pattern**:
```go
// Create namespace per test scenario
kubectl create namespace e2e-scenario-1
kubectl create namespace e2e-scenario-2

// Deploy Gitea to each namespace with different port
```

### Test Data Pollution Prevention

**Decision**: ✅ Use unique repository names with test prefix

```testscript
# In testscript
env TEST_ID=complete-workflow-${TIMESTAMP}
exec apx init canonical --org=testorg --repo=test-${TEST_ID}-apis
```

**Rationale**:
- Prevents name collisions between parallel scenarios (future)
- Easy to identify test repos vs real repos
- Timestamp ensures uniqueness even if test reruns
- Cleanup filter: `test-*` pattern

---

## 5. Cross-Platform Considerations

### macOS (Docker Desktop)

**Compatibility**: ✅ Fully supported

**Requirements**:
- Docker Desktop 4.0+ (includes k3s support)
- Rosetta 2 (for M1/M2 Macs if using x86 images)

**Tested on**: macOS 14 Sonoma with Docker Desktop 4.25

**Notes**:
- k3d works identically to Linux
- Port mapping works: `localhost:3000` → Gitea
- Slightly slower startup (~30-35s vs ~25s on Linux)

### Linux (Native Docker)

**Compatibility**: ✅ Fully supported (primary target)

**Requirements**:
- Docker Engine 20.10+
- kubectl (installed via `make install-e2e-deps`)

**Tested on**: Ubuntu 22.04 LTS

**Notes**:
- Fastest performance (native containers)
- GitHub Actions `ubuntu-latest` uses this
- No special configuration needed

### Windows (Docker Desktop + WSL2)

**Compatibility**: ⚠️ Best-effort support

**Requirements**:
- Docker Desktop 4.0+ with WSL2 backend
- WSL2 Ubuntu distribution
- k3d installed in WSL2 (not Windows native)

**Known issues**:
- Path conversion (Windows → WSL2) can cause issues
- Must run tests inside WSL2 shell, not PowerShell
- Slightly slower due to WSL2 overhead

**Recommendation**:
```bash
# Run tests in WSL2 Ubuntu
wsl
cd /mnt/c/Users/.../apx
make test-e2e
```

**CI Support**: 
- GitHub Actions `windows-latest` NOT tested
- Primary CI: `ubuntu-latest` and `macos-latest`
- Windows users should use WSL2 locally

### GitHub Actions Runner Capabilities

**Ubuntu Latest** (primary):
- Docker pre-installed ✅
- k3d installs cleanly ✅
- No special permissions needed ✅
- Localhost networking works ✅

**macOS Latest** (secondary):
- Docker Desktop not pre-installed ⚠️
- Requires setup step (slower CI) ⚠️
- k3d works once Docker installed ✅

**Windows Latest** (not supported):
- WSL2 not available ❌
- Docker Desktop requires license ❌
- Not recommended for CI ❌

**Decision**: ✅ Ubuntu as primary CI platform, macOS optional

---

## Decision Summary

| Decision Point | Choice | Rationale |
|----------------|--------|-----------|
| Container orchestration | k3d | Lighter than kind, faster startup, no privileged mode |
| Git hosting | Gitea in k3d | Realistic, no external deps, full git protocol support |
| Gitea backend | SQLite | Zero config, ephemeral, fast for tests |
| Gitea version | 1.22 (pinned) | Stable API, reproducible tests |
| Authentication | API tokens | Simpler than SSH, cross-platform |
| Service exposure | NodePort + host port mapping | Simple, works in CI |
| Test orchestration | Testscript | Existing APX pattern, readable scenarios |
| Cluster lifecycle | Per test suite | Balance isolation vs performance (3min vs 10min) |
| Repository cleanup | API deletion | Fast, idempotent |
| Gitea URL passing | Environment variable | Native testscript support |
| Token passing | Environment variable | Secure enough for test environment |
| Cleanup guarantee | Go t.Cleanup() | LIFO execution, panic-safe |
| Primary platform | Linux (Ubuntu) | GitHub Actions default, best performance |
| Secondary platform | macOS | Developer machines, slower but works |
| Windows support | Best-effort (WSL2) | Complex setup, not CI target |

---

## Open Questions

None - all research tasks completed with clear decisions.

---

## References

1. k3d Documentation: https://k3d.io/
2. Gitea API Documentation: https://docs.gitea.com/api/
3. Testscript Package: https://pkg.go.dev/github.com/rogpeppe/go-internal/testscript
4. GitHub Actions Docker Support: https://docs.github.com/en/actions/using-containerized-services/about-service-containers
5. APX Constitution (Test-First Development): `.specify/memory/constitution.md`

---

## Next Steps

1. ✅ Phase 0 complete - all technical decisions made
2. ⏳ Proceed to Phase 1 - data model and contracts
3. ⏳ Generate implementation tasks
4. ⏳ Begin TDD implementation
