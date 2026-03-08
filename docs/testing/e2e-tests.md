# End-to-End Testing Guide

The APX E2E test suite validates the complete schema publishing workflow against a real git server, using **k3d** (lightweight Kubernetes) to host a **Gitea** instance.

## What It Tests

| Area | Scenarios | Description |
|------|-----------|-------------|
| Complete Workflow | 2 | `apx init` → `apx publish` → PR creation |
| Cross-Repo Deps | 2 | App2 consumes App1's schema, both publish independently |
| Breaking Changes | 3 | Detection, non-breaking allowance, major version bumps |
| Git History | 1 | Commit preservation through PR-based publishing |
| Edge Cases | 8 | Error handling, concurrency, cleanup |

**Total: 16 testscript scenarios** covering all 18 functional requirements.

## Quick Start

```bash
# 1. Install prerequisites
make install-e2e-deps

# 2. Run E2E tests
make test-e2e

# 3. Clean up (if needed)
make clean-e2e
```

## Prerequisites

- **Docker** — running and accessible
- **k3d** — installed via `make install-e2e-deps`
- **kubectl** — installed via `make install-e2e-deps`
- **buf** — installed via `make tools` or `brew install bufbuild/buf/buf`
- **~2GB free RAM** for the k3d cluster
- **Port 3000** available (Gitea NodePort)

## How It Works

1. **k3d cluster creation** — A lightweight Kubernetes cluster named `apx-e2e-<timestamp>` is created in Docker
2. **Gitea deployment** — Gitea 1.22 is deployed as a Kubernetes Deployment with SQLite storage
3. **Admin setup** — An admin user and API token are created automatically
4. **Testscript execution** — Each `.txt` file in `testdata/script/e2e/` runs as an independent test
5. **Cleanup** — The k3d cluster and all resources are destroyed after tests complete

## Running Individual Tests

```bash
# Run a specific scenario
go test -v -timeout 15m ./tests/e2e/... -run TestE2E/e2e_breaking_detection

# Run with debug mode (keeps cluster running after test)
E2E_DEBUG=1 go test -v -timeout 15m ./tests/e2e/... -run TestE2E
```

## Writing New Tests

Create a `.txt` file in `testdata/script/e2e/` using [testscript syntax](https://pkg.go.dev/github.com/rogpeppe/go-internal/testscript):

```
# testdata/script/e2e/e2e_my_test.txt

# Setup git identity
exec git config --global user.name 'Test User'
exec git config --global user.email 'test@example.com'

# Initialize a canonical repo
mkdir my-canonical
cd my-canonical
exec apx init canonical --org testorg --repo my-canonical --skip-git --non-interactive
exists buf.yaml

# Create a proto file
cp ../common.proto schemas/common.proto

# Initialize git and push
exec git init
exec git add .
exec git commit -m 'Initial commit'
exec git remote add origin ${GITEA_URL}/testorg/my-canonical.git
exec git push -u origin main

-- common.proto --
syntax = "proto3";
package common.v1;
message Timestamp {
  int64 seconds = 1;
}
```

### Available Environment Variables

| Variable | Description |
|----------|-------------|
| `GITEA_URL` | Gitea base URL (e.g., `http://localhost:3000`) |
| `GITEA_TOKEN` | Admin API token for Gitea |
| `APX_DISABLE_TTY` | Set to `1` (disables interactive prompts) |
| `NO_COLOR` | Set to `1` (disables color output) |
| `CI` | Set to `1` |

### Rules

- All commands must use `exec` prefix (`RequireExplicitExec: true`)
- `stdout` assertions must immediately follow their `exec` command
- Archive sections (`-- filename --`) are placed at the end of the file
- Files in archive sections are in `$WORK`, not in subdirectories

## Architecture

```
tests/e2e/
├── main_test.go          # Orchestrates k3d + Gitea + testscript runner
├── k3d/
│   ├── cluster.go        # Cluster lifecycle
│   ├── config.go         # Kubernetes manifests
│   └── cleanup.go        # Resource cleanup
├── gitea/
│   ├── lifecycle.go      # Gitea deployment & health checks
│   └── client.go         # REST API client
├── testhelpers/
│   ├── git.go            # Git operation wrappers
│   ├── apx.go            # APX command wrappers
│   ├── assertions.go     # Custom assertions
│   └── environment.go    # Test workspace setup
└── fixtures/             # Seed data for test repos
```

## CI Integration

E2E tests run automatically in CI:

- **Ubuntu** — runs on every push/PR (required)
- **macOS** — runs on push to `main` only (optional, uses colima for Docker)

See `.github/workflows/ci.yml` for the full configuration.

## Performance

| Phase | Duration |
|-------|----------|
| k3d cluster creation | ~15-30s |
| Gitea deployment + readiness | ~15-30s |
| 16 testscript scenarios | ~10-20s |
| Cleanup | ~5s |
| **Total** | **~56s** |
