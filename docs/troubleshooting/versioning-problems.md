# Versioning Problems

Troubleshooting guide for version, lifecycle, and release-related issues.

## Lifecycle-Version Mismatches

### `lifecycle "stable" requires a stable version`

```
Error: lifecycle "stable" requires a stable version (no prerelease), got v1.0.0-beta.1
```

The compatibility matrix:

| Lifecycle | Allowed versions |
|-----------|------------------|
| `experimental` | `-alpha.*` prerelease only |
| `preview` | `-alpha.*`, `-beta.*`, or `-rc.*` |
| `stable` | Stable only (no prerelease suffix) |
| `deprecated` | Any version |
| `sunset` | No new releases |

**Fix:** Either change the lifecycle or adjust the version to match.

### `lifecycle "experimental" cannot use -beta prerelease`

```
Error: lifecycle "experimental" only allows -alpha.* prerelease versions
```

**Fix:** Use `-alpha.N` for experimental, or promote the lifecycle to `preview`.

---

## Version-Line Compatibility

### `version v2.0.0 is incompatible with API line v1`

The SemVer major version must match the declared version line:

| API line | Valid versions | Invalid |
|----------|---------------|---------|
| `v0` | `v0.1.0`, `v0.2.0-alpha.1` | `v1.0.0` |
| `v1` | `v1.0.0`, `v1.5.2-beta.1` | `v2.0.0`, `v0.1.0` |
| `v2` | `v2.0.0`, `v2.1.0-rc.1` | `v1.0.0`, `v3.0.0` |

**Fix:** Use a version whose major number matches the line, or create a new API line for breaking changes.

---

## v0 Line Issues

### `v0 APIs cannot use lifecycle "stable"`

```
Error: v0 APIs are limited to "experimental" or "preview" lifecycle
```

The `v0` line is reserved for APIs under active development. Stable graduation requires moving to `v1`:

1. Create `proto/payments/ledger/v1/` (copy and evolve from v0)
2. Publish with `proto/payments/ledger/v1` as the API ID
3. Set lifecycle to `stable`
4. Deprecate the v0 line

### Breaking changes in v0

Breaking changes are **allowed** in v0 with a minor version bump (e.g. `v0.1.0` → `v0.2.0`). This is intentional — v0 signals instability to consumers.

---

## Lifecycle Transitions

### `backward lifecycle transition`

```
Error: cannot transition from "stable" to "preview"
```

Lifecycle must progress forward:

```
experimental → preview → stable → deprecated → sunset
```

Backward transitions are rejected. If you need to revert a lifecycle:

1. If the version was never consumed, contact the canonical repo maintainer to remove the tag
2. For published APIs, create a new version at the desired lifecycle instead

---

## Semver Suggest Disagreements

### `apx semver suggest` recommends unexpected bump

**Scenario:** You made a minor change but `apx semver suggest` recommends a major bump.

**Cause:** The schema diff detected a breaking change you may not have intended:

```bash
# See what breaking changes were detected
apx breaking --against HEAD^ --verbose
```

**Common accidental breaking changes:**
- Renaming a field (removal + addition = breaking)
- Changing a field number in proto
- Making an optional field required in OpenAPI
- Changing a field type (even if semantically equivalent)

**Fix:** Undo the breaking change or, if intentional, create a new version line.

### `apx semver suggest` returns `patch` when you expected `minor`

**Cause:** The schema files themselves didn't change — only non-schema files (docs, config, etc.) were modified.

Patch is correct here. Minor bumps require additive schema changes (new fields, methods, or endpoints).

---

## Tag Format Errors

### `invalid tag format`

```
Error: tag "v1.0.0" does not match expected pattern: proto/payments/ledger/v1/v1.0.0
```

APX tags follow the pattern `<api-id>/v<major>.<minor>.<patch>[-prerelease]`:

```
proto/payments/ledger/v1/v1.0.0          ✔ stable
proto/payments/ledger/v1/v1.1.0-beta.1   ✔ preview
v1.0.0                                    ✘ missing API ID prefix
proto/payments/ledger/1.0.0               ✘ missing "v" prefix
```

### Tags not appearing in catalog

```bash
# Regenerate the catalog from tags
apx catalog generate

# Verify tags exist
git tag -l 'proto/payments/*'
```

**Cause:** Tags were created but `catalog.yaml` wasn't regenerated. This happens automatically via `on-merge.yml` in the canonical repo, but may need a manual trigger.

---

## Version Ordering

SemVer ordering follows the specification:

```
v1.0.0-alpha.1 < v1.0.0-alpha.2 < v1.0.0-beta.1 < v1.0.0-rc.1 < v1.0.0
```

If `apx semver suggest` or `apx release history` shows unexpected ordering, ensure all tags follow strict SemVer format.

## See Also

- [Release Guardrails](../publishing/release-guardrails.md) — full lifecycle and version enforcement rules
- [Versioning Strategy](../dependencies/versioning-strategy.md) — the three-layer versioning model
- [Publishing Failures](publishing-failures.md) — publish and release errors
- [Common Errors](common-errors.md) — general error reference
