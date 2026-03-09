# Common Errors

Quick reference for frequently encountered APX errors and their solutions.

## Configuration Errors

### `apx.yaml not found`

```
Error: apx.yaml not found in current directory or parent directories
```

**Cause:** APX commands that need configuration can't find `apx.yaml`.

**Fix:** Either:
- Run `apx init app` or `apx init canonical` to create one
- Use `--config /path/to/apx.yaml` to specify a custom location
- Ensure you're in the repository root directory

### `apx.yaml already exists`

```
Error: apx.yaml already exists
```

**Cause:** `apx init` or `apx config init` won't overwrite an existing config.

**Fix:** Delete or rename the existing file, or edit it directly.

### Configuration validation failed (exit code 6)

```
Error: configuration is invalid
  - field "org": missing required field
  - field "language_targets.go.plugins": invalid type
```

**Cause:** `apx.yaml` has structural or value errors.

**Fix:** Run `apx config validate` to see all errors, then fix them. Run `apx config migrate` if the issue is a schema version mismatch.

---

## Schema Format Detection

### `could not detect schema format`

```
Error: could not detect schema format for path: internal/apis/
```

**Cause:** APX couldn't find recognizable schema files (`.proto`, `.yaml`, `.avsc`, etc.) in the given directory.

**Fix:**
- Verify the path points to a directory containing schema files
- Use `--format proto` (or openapi, avro, etc.) to specify the format explicitly
- Check that `module_roots` in `apx.yaml` points to the correct directory

---

## Identity Errors

### `invalid API ID format`

```
Error: invalid API ID: "payments/ledger"
  Expected format: <format>/<domain>/<name>/<line>
  Example: proto/payments/ledger/v1
```

**Cause:** The API ID is missing required components.

**Fix:** Include all four parts: format, domain, name, and version line.

### `version incompatible with API line`

```
Error: version v2.0.0 is incompatible with API line v1
```

**Cause:** The SemVer major version doesn't match the declared API line.

**Fix:** Use a version that matches the line (e.g. `v1.x.x` for the `v1` line), or create a new API line for breaking changes.

### `lifecycle-version mismatch`

```
Error: lifecycle "stable" requires a stable version (no prerelease), got v1.0.0-beta.1
```

**Cause:** The version's prerelease tag doesn't match the declared lifecycle.

**Fix:** See [Release Guardrails](../publishing/release-guardrails.md) for the compatibility matrix.

---

## Tool Resolution

### `failed to resolve buf`

```
Error: failed to resolve buf: tool not found in .apx-tools/
```

**Cause:** The Buf CLI is not cached locally.

**Fix:** Run `apx fetch` to download and cache all pinned tools.

### `buf: executable file not found in $PATH`

```
exec: "buf": executable file not found in $PATH
```

**Cause:** Buf is not installed and APX couldn't resolve it from the cache.

**Fix:**
```bash
apx fetch              # preferred — uses pinned version from apx.lock
# or
brew install bufbuild/buf/buf   # install system-wide
```

---

## Git & GitHub Errors

### `gh: not authenticated`

```
Error: gh: not authenticated. Run "gh auth login" to authenticate.
```

**Cause:** The GitHub CLI is not authenticated, required for `apx release submit`.

**Fix:**
```bash
gh auth login
```

### `permission denied` when pushing

```
Error: remote: Permission to org/apis.git denied
```

**Cause:** Your credentials don't have write access to the canonical repo.

**Fix:**
- For local releases: ensure you have push access to the canonical repo
- For CI: verify the GitHub App is installed and org secrets (`APX_APP_ID`, `APX_APP_PRIVATE_KEY`) are set

### `could not determine org and repo`

```
Error: could not determine org and repo.
Either create an apx.yaml or ensure a git remote named 'origin' is configured.
```

**Cause:** No `apx.yaml` found and git remote detection failed.

**Fix:** Run `apx init app --org=<org> --repo=<repo> ...` or ensure a git remote named `origin` exists.

---

## Dependency Errors

### Module not found in catalog

```
No results found for "nonexistent-api"
```

**Cause:** The API doesn't exist in the catalog, or the catalog is out of date.

**Fix:**
- Verify the API ID with `apx search`
- Run `apx catalog generate` in the canonical repo to refresh the catalog

---

## Exit Codes

| Code | Meaning | Common triggers |
|------|---------|-----------------|
| `0` | Success | — |
| `1` | General error | Schema validation failure, release error, tool not found |
| `6` | Config validation error | Invalid `apx.yaml` |

## See Also

- [Buf Issues](buf-issues.md) — Buf-specific troubleshooting
- [Release Failures](publishing-failures.md) — release pipeline errors
- [Versioning Problems](versioning-problems.md) — version and lifecycle issues
- [Code Generation](code-generation.md) — generation troubleshooting
- [FAQ](faq.md) — frequently asked questions
