# Updates and Upgrades

```{admonition} Planned
:class: note
`apx update` and `apx upgrade` commands are planned for a future release. This page documents the intended design and the current manual workaround.
```

## Current Workaround

To pin a newer version of a dependency today, re-add it at the desired version:

```bash
# Update to a newer version
apx add proto/payments/ledger/v1@v1.3.0

# Regenerate code
apx gen go
apx sync
```

This updates `apx.lock`, regenerates code into `internal/gen/`, and refreshes `go.work` overlays.

## Planned: `apx update`

`apx update` will check for compatible (patch and minor) updates for all pinned dependencies and apply them:

```bash
# Check for compatible updates across all dependencies
apx update

# Update a specific dependency
apx update proto/payments/ledger/v1

# Preview what would be updated
apx update --dry-run
```

**Intended behavior:**

- Reads `apx.lock` for currently pinned versions
- Queries the canonical repo catalog for newer versions within the same API line
- Applies only **compatible** updates (patch and minor bumps)
- Respects lifecycle transitions (won't downgrade from `stable` to `beta`)
- Updates `apx.lock` with new pinned versions
- Regenerates code automatically

## Planned: `apx upgrade`

`apx upgrade` will handle major version upgrades that may involve breaking changes or API line transitions:

```bash
# Upgrade to a new API line (breaking change)
apx upgrade proto/payments/ledger/v1 --to v2

# Preview breaking changes before upgrading
apx upgrade proto/payments/ledger/v1 --to v2 --dry-run
```

**Intended behavior:**

- Analyzes breaking changes between the current version and the target
- Reports import path changes (e.g. `ledger/v1` → `ledger/v2`)
- Updates `apx.lock` with the new API line and version
- Regenerates code with new import paths
- Updates `go.work` overlays accordingly

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
