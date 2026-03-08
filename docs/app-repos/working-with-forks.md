# Working with Forks

When contributing to an API repository you don't own, you'll typically work on a **fork** — a personal copy of the canonical repo under your own GitHub organization or user account. APX is fork-aware and handles most of this automatically, but there are important limitations around publishing.

## How Fork Detection Works

When you run `apx init` (or any command that needs org/repo defaults), APX inspects your git remotes:

1. **`origin`** — your fork (e.g. `git@github.com:dgarcia/apis.git`)
2. **`upstream`** — the canonical repo (e.g. `git@github.com:Infoblox-CTO/apis.git`)

If both remotes exist and point to **different organizations**, APX assumes you're working on a fork and automatically uses the **upstream org** for all consumption paths. This ensures generated import paths, `go_package` options, and dependency references all point to the canonical repository — not your fork.

```
# Your remotes
origin    → git@github.com:dgarcia/apis.git        (your fork)
upstream  → git@github.com:Infoblox-CTO/apis.git   (canonical)

# APX detects:
#   Org = Infoblox-CTO  (from upstream, not dgarcia)
#   Repo = apis
#   UpstreamOrg = Infoblox-CTO
```

### Setting Up Your Fork

```bash
# Fork the canonical repo on GitHub, then:
git clone git@github.com:<your-user>/apis.git
cd apis
git remote add upstream git@github.com:<canonical-org>/apis.git

# APX will now auto-detect the canonical org
apx init canonical --non-interactive
# → org: <canonical-org>  (detected from upstream remote)
```

## What Works on a Fork

All **consumption and authoring** workflows work correctly on forks:

| Workflow | Status | Details |
|----------|--------|---------|
| `apx init` | Works | Auto-detects canonical org from upstream remote |
| `apx lint` | Works | Schema validation is local |
| `apx breaking` | Works | Compatibility checks against local refs |
| `apx gen` | Works | Code generation uses canonical import paths |
| `apx show` | Works | Displays canonical identity correctly |
| `apx inspect identity` | Works | Shows canonical `go_package` and module paths |
| `apx explain go-path` | Works | Derives paths from canonical org/repo |
| `apx search` | Works | Queries the catalog |
| Local development | Works | `go.work` overlays resolve to local code |

## What Does NOT Work on a Fork

**Publishing from a fork is not supported.** Several operations require write access to the canonical repository, which fork contributors typically don't have:

### 1. `apx publish` — Subtree Push

`apx publish` pushes a git subtree to the canonical repo (`github.com/<canonical-org>/apis`). This requires **push access** to that repo. From a fork, this will fail with a permission error.

### 2. `apx release tag` — Tag Creation

Release tags (e.g. `proto/payments/ledger/v1/v1.0.0`) are created on the canonical repo. Fork contributors cannot create tags on a repo they don't own.

### 3. `apx publish --pr` — Pull Request Creation

The PR created by `apx publish` targets the canonical repo. While GitHub supports cross-fork PRs, the current implementation assumes same-repo PRs and the fork's CI token doesn't have the right permissions.

### 4. CI-Triggered Publishing

CI workflows on your fork run with your fork's credentials, not the canonical repo's. Any CI step that calls `apx publish` will fail because the fork's `GITHUB_TOKEN` doesn't have write access to the upstream repo.

## Recommended Fork Workflow

The correct pattern is to **author on the fork, publish from the canonical repo's CI**:

```
┌─────────────────────────┐     ┌──────────────────────────┐
│    Your Fork              │     │  Canonical Repo           │
│                           │     │                            │
│  1. Author schemas        │     │                            │
│  2. apx lint              │     │                            │
│  3. apx breaking          │     │                            │
│  4. apx gen go            │     │                            │
│  5. git push origin       │     │                            │
│                           │     │                            │
│  6. Open PR ─────────────────── │  7. CI validates PR        │
│                           │     │  8. Reviewers approve      │
│                           │     │  9. Merge to main          │
│                           │     │ 10. Post-merge CI:         │
│                           │     │     - apx publish          │
│                           │     │     - apx release tag      │
│                           │     │     - catalog regeneration  │
└─────────────────────────┘     └──────────────────────────┘
```

**Steps 1–6** happen on your fork with your credentials. All consumption-side commands work because APX resolves the canonical org from your `upstream` remote.

**Steps 7–10** happen on the canonical repo's CI, which has the credentials to push subtrees, create tags, and update the catalog.

### Example CI Configuration (Canonical Repo)

```yaml
# .github/workflows/publish.yml — runs on the canonical repo, not forks
name: Publish APIs
on:
  push:
    branches: [main]

jobs:
  publish:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0  # full history for subtree

      - name: Install APX
        run: curl -sSL https://raw.githubusercontent.com/infobloxopen/apx/main/install.sh | bash

      - name: Publish changed APIs
        run: apx publish --ci
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}

      - name: Tag releases
        run: apx release tag --ci
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

## Troubleshooting

### APX uses my fork org instead of the canonical org

Ensure you have an `upstream` remote pointing to the canonical repo:

```bash
git remote -v
# Should show:
# origin    git@github.com:<your-user>/apis.git (fetch)
# upstream  git@github.com:<canonical-org>/apis.git (fetch)

# If upstream is missing:
git remote add upstream git@github.com:<canonical-org>/apis.git
```

### `apx publish` fails with "permission denied"

You're likely running publish from a fork. Publishing must happen from the canonical repo's CI after your PR is merged. See [Recommended Fork Workflow](#recommended-fork-workflow) above.

### Import paths show my fork org in generated code

This means APX didn't detect the fork. Check that:
1. The `upstream` remote exists and points to the canonical repo
2. The `upstream` org is different from `origin` — APX only overrides when orgs differ
3. Run `apx inspect identity` to verify the resolved org
