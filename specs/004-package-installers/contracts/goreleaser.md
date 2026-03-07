# Contract: GoReleaser Configuration Changes

**File**: `.goreleaser.yml`  
**Type**: Configuration delta (modify existing)

## Changes Required

### 1. Add `token` to `brews[0].repository`

```yaml
brews:
  - name: apx
    repository:
      owner: infobloxopen
      name: homebrew-tap
      token: "{{ .Env.HOMEBREW_TAP_TOKEN }}"   # <-- ADD THIS
```

GoReleaser uses `GITHUB_TOKEN` by default for cross-repo pushes, but the built-in Actions `GITHUB_TOKEN` is scoped to the current repo. Specifying `token` tells GoReleaser to use a separate PAT stored as a secret.

### 2. Add `scoops` section (after `brews`)

```yaml
scoops:
  - name: apx
    repository:
      owner: infobloxopen
      name: scoop-bucket
      token: "{{ .Env.SCOOP_BUCKET_TOKEN }}"
    homepage: https://github.com/infobloxopen/apx
    description: "API Publishing eXperience CLI"
    license: Apache-2.0
```

This generates an `apx.json` manifest in the root of the `scoop-bucket` repo on each release. GoReleaser automatically sets `version`, `architecture`, download URLs, and SHA256 hashes.

### 3. No other changes needed

The `builds`, `archives`, `checksum`, `changelog`, `nfpms`, `dockers`, `docker_manifests`, and `release` sections are correct as-is.

## Validation

After changes, run:
```bash
goreleaser check
goreleaser release --snapshot --clean
```

The `--snapshot` build should succeed (it skips cross-repo pushes). The `check` command validates YAML syntax and GoReleaser v2 schema compliance.
