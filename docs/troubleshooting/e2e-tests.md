# E2E Test Troubleshooting

Common issues and solutions when running the APX E2E test suite.

## Tests Skipped

```
SKIP: E2E tests only run with E2E_ENABLED=1
```

**Solution**: The E2E tests require the `E2E_ENABLED=1` environment variable. Use `make test-e2e` which sets this automatically, or run:

```bash
E2E_ENABLED=1 go test -v -timeout 15m ./tests/e2e/...
```

## Docker Not Running

```
Cannot connect to the Docker daemon
```

**Solution**: Start Docker Desktop (macOS/Windows) or the Docker daemon (Linux):

```bash
# macOS
open -a Docker

# Linux
sudo systemctl start docker
```

## k3d Cluster Creation Fails

```
failed to create k3d cluster: permission denied
```

**Solutions**:
- Ensure your user is in the `docker` group: `sudo usermod -aG docker $USER`
- Log out and back in after adding to docker group
- On macOS, ensure Docker Desktop is running

## Port 3000 Already in Use

```
failed to create cluster: port 3000 already allocated
```

**Solution**: Stop any service using port 3000:

```bash
# Find what's using port 3000
lsof -i :3000

# Kill the process
kill -9 <PID>

# Or clean up stale k3d clusters
make clean-e2e
```

## Gitea Deployment Timeout

```
timeout waiting for Gitea to be ready
```

**Solutions**:
1. Check Docker has enough resources (~2GB RAM minimum)
2. Check Gitea pod logs:
   ```bash
   kubectl --kubeconfig=/tmp/k3d-kubeconfig-* logs -n gitea-e2e -l app=gitea
   ```
3. Check pod status:
   ```bash
   kubectl --kubeconfig=/tmp/k3d-kubeconfig-* get pods -n gitea-e2e
   ```
4. On slow systems, the default timeout may be too short — the suite allows up to 5 minutes

## Tests Hang Indefinitely

**Solution**: Always set a timeout:

```bash
go test -timeout 15m ./tests/e2e/...
```

If a test hangs during Gitea health checks, it's usually a Docker resource issue. Run `make clean-e2e` and try again.

## Stale Clusters from Previous Runs

```
WARN: cluster already exists
```

**Solution**: Clean up leftover resources:

```bash
make clean-e2e

# Or manually:
k3d cluster list
k3d cluster delete <cluster-name>
docker ps -a --filter "name=k3d-apx-e2e" --format "{{.ID}}" | xargs docker rm -f
docker volume ls --filter "name=k3d-apx-e2e" --format "{{.Name}}" | xargs docker volume rm
```

## `buf` Not Found

```
exec: "buf": executable file not found in $PATH
```

**Solution**: Install buf:

```bash
# macOS
brew install bufbuild/buf/buf

# Linux
curl -sSL "https://github.com/bufbuild/buf/releases/download/v1.66.1/buf-Linux-x86_64" -o /tmp/buf
chmod +x /tmp/buf
sudo mv /tmp/buf /usr/local/bin/buf
```

## `apx` Binary Not Found in Tests

```
exec: "apx": executable file not found in $PATH
```

The E2E runner builds the `apx` binary automatically. If this fails:

```bash
# Build manually
go build -o ./bin/apx ./cmd/apx

# Then run tests
make test-e2e
```

## Debug Mode

To keep the k3d cluster and Gitea running after test completion for manual investigation:

```bash
E2E_DEBUG=1 make test-e2e
```

The test output will show:
- Gitea URL (typically `http://localhost:3000`)
- Admin credentials and API token
- Cluster name

Access Gitea in your browser and inspect repositories, PRs, and tags manually. Clean up when done:

```bash
make clean-e2e
```

## macOS-Specific Issues

### Docker Desktop Resource Limits

Docker Desktop on macOS defaults to limited resources. Go to **Docker Desktop → Settings → Resources** and ensure:
- **Memory**: ≥4GB (2GB minimum for k3d)
- **CPUs**: ≥2

### Apple Silicon (M1/M2/M3)

The E2E suite works on Apple Silicon. The Gitea Docker image supports `linux/arm64`. k3d handles architecture translation automatically.

## Linux-Specific Issues

### cgroup v2

If using cgroup v2 (default on modern Linux), ensure Docker is configured for it. k3d supports cgroup v2 natively since v5.x.

### rootless Docker

k3d has limited support for rootless Docker. If tests fail with permission errors, consider using standard Docker with user group access.
