# Code Generation Troubleshooting

Troubleshooting guide for `apx gen`, `apx sync`, and related code generation issues.

## Tool Resolution

### `buf: executable file not found`

```
exec: "buf": executable file not found in $PATH
```

**Cause:** The Buf CLI is not installed or not in PATH.

**Fix:**
```bash
# Preferred — uses pinned version from apx.lock
apx fetch

# Or install globally
brew install bufbuild/buf/buf
```

### `protoc-gen-go: plugin not found`

```
Error: protoc-gen-go: plugin not found
```

**Cause:** Go protobuf plugins are not installed.

**Fix:**
```bash
apx fetch    # downloads all pinned tools

# Or install manually
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

---

## Generation Errors

### `no schema files found`

```
Error: no schema files found for format "proto" in path ...
```

**Cause:** The specified path doesn't contain schema files for the given format.

**Fix:**
- Check that `module_roots` in `apx.yaml` points to the correct directory
- Verify schema files exist (e.g. `.proto` files for protobuf)
- If auto-detection fails, specify the format explicitly: `apx gen go --format proto`

### `buf generate` fails with import errors

```
proto/payments/ledger/v1/ledger.proto:5:1: import "google/protobuf/timestamp.proto": not found
```

**Cause:** Proto imports can't be resolved. Buf needs to know where to find dependencies.

**Fix:**
- Ensure `buf.yaml` has the correct `deps` section:
  ```yaml
  deps:
    - buf.build/googleapis/googleapis
    - buf.build/grpc/grpc
  ```
- Run `buf dep update` to fetch dependencies
- Verify `buf.lock` is committed

### `output directory is not clean`

```
Warning: output directory internal/gen/go/ contains files not managed by apx
```

**Fix:** Use `--clean` to remove stale files before generating:
```bash
apx gen go --clean
```

---

## Overlay & Sync Issues

### `go.work` not updated after generation

**Cause:** `apx sync` wasn't run after code generation.

**Fix:**
```bash
apx gen go && apx sync
```

Always run `apx sync` after generating Go code to update `go.work` with the new overlay directories.

### Stale generated files after schema changes

**Cause:** Old generated files remain from a previous schema structure.

**Fix:**
```bash
apx gen go --clean    # removes old output, regenerates fresh
apx sync --clean      # removes stale go.work entries
```

### `go.work` changes not reflected

If `go build` doesn't see the new modules:

```bash
# Verify go.work has the right entries
cat go.work

# Re-sync
apx sync

# Force Go to re-read go.work
go mod tidy
```

---

## Language-Specific Issues

### Go: `module path mismatch`

```
go: inconsistent module path: go.mod says "github.com/acme-corp/apis/proto/payments/ledger"
   but expected "github.com/acme-corp/other-path"
```

**Cause:** The `go.mod` in the generated overlay has a different module path than expected.

**Fix:**
- Check `go_package` in your `.proto` files matches the canonical import path
- Regenerate: `apx gen go --clean`
- Verify with: `apx inspect identity proto/payments/ledger/v1`

### Go: `cannot find module providing package`

```
cannot find module providing package github.com/acme-corp/apis/proto/payments/ledger/v1
```

**Fix:**
- Ensure `go.work` includes the generated overlay directory
- Run `apx sync` to update `go.work`
- Verify the overlay directory exists: `ls internal/gen/go/proto/payments/ledger/`

### Python: import failures

```python
ModuleNotFoundError: No module named 'proto.payments.ledger.v1'
```

**Fix:**
- Add the generated output directory to `PYTHONPATH`:
  ```bash
  export PYTHONPATH="$PWD/internal/gen/python:$PYTHONPATH"
  ```
- Verify `__init__.py` files exist in each package directory
- Regenerate if needed: `apx gen python --clean`

---

## Manifest Issues

### `--manifest` file not found

```
Error: manifest file "gen-manifest.yaml" not found
```

**Cause:** The manifest file specified with `--manifest` doesn't exist.

**Fix:**
- Check the file path and name
- Create a manifest file if needed (see `apx gen --help` for format)

---

## Debugging Tips

```bash
# Verbose output shows each tool invocation
apx gen go --verbose

# Dry run to see what would be generated
apx sync --dry-run

# Check which tools APX resolved
ls -la .apx-tools/

# Verify schema detection
apx lint --verbose
```

## See Also

- [Code Generation](../dependencies/code-generation.md) — how code generation works
- [Buf Issues](buf-issues.md) — Buf-specific troubleshooting
- [Common Errors](common-errors.md) — general error reference
