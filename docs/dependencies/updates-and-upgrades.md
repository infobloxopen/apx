# Updates and Upgrades

APX provides two commands for managing dependency versions:

- **`apx update`** — compatible updates within the same API line (minor/patch bumps)
- **`apx upgrade`** — major version transitions across API lines (e.g. v1 → v2)

Both commands query the canonical catalog for available versions and update `apx.lock`.

## `apx update`

Check for compatible (same API line, higher minor/patch) updates and apply them:

```bash
# Check and apply all compatible updates
apx update

# Update a specific dependency
apx update proto/payments/ledger/v1

# Preview what would be updated
apx update --dry-run
```

**Behavior:**

- Reads `apx.lock` for currently pinned versions
- Queries the canonical repo catalog for newer versions within the same API line
- Applies only **compatible** updates (same major version, higher minor/patch)
- Respects lifecycle: prefers latest stable version, falls back to prerelease
- Updates `apx.lock` with new pinned versions
- Supports `--dry-run` for previewing changes and `--json` for CI integration

## `apx upgrade`

Handle major version upgrades that involve API line transitions:

```bash
# Upgrade from v1 to v2
apx upgrade proto/payments/ledger/v1 --to v2

# Preview breaking changes before upgrading
apx upgrade proto/payments/ledger/v1 --to v2 --dry-run
```

**Behavior:**

- Verifies the target API line exists in the catalog
- Reports the import path change (e.g. `proto/payments/ledger/v1` → `proto/payments/ledger/v2`)
- Removes the old API line from `apx.lock` and adds the new one
- Pins the latest available version on the target line
- Supports `--dry-run` for previewing the upgrade plan

**After upgrading:**

1. Regenerate code: `apx gen go && apx sync`
2. Update import paths in your code (the command prints the mapping)
3. Run `apx breaking` to inspect breaking changes between API lines

## Manual Workaround

You can also update a dependency manually by re-adding it at the desired version:

```bash
# Update to a newer version
apx add proto/payments/ledger/v1@v1.3.0

# Regenerate code
apx gen go
apx sync
```

This updates `apx.lock`, regenerates code into `internal/gen/`, and refreshes `go.work` overlays.

## Version Selection Strategy

When choosing versions (manually today, or via `update`/`upgrade` in the future):

| Goal | Command | Version constraint |
|------|---------|--------------------|
| Latest patch fix | Re-add at new version | Same minor, higher patch |
| Latest compatible | Re-add at new version | Same major, higher minor/patch |
| Breaking upgrade | New API line | New major/line |
| Pin exact version | `apx add ...@vX.Y.Z` | Exact match |

## Best Practices

- **Pin specific versions** in production — don't rely on "latest"
- **Test in feature branches** before upgrading dependencies
- **Check breaking changes** with `apx breaking` before and after version bumps
- **Coordinate across teams** when upgrading shared APIs
- **Review the catalog** with `apx search` to see available versions before upgrading

## Next Steps

- [Adding Dependencies](adding-dependencies.md) — how to add and pin APIs today
- [Versioning Strategy](versioning-strategy.md) — understand API lines and SemVer
- [Discovery](discovery.md) — find available APIs and versions
