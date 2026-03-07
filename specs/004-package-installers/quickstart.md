# Quickstart: Package Installer Release Pipeline

**Feature**: 004-package-installers

## Prerequisites (One-Time Setup)

Before the first release, three things need to exist:

### 1. External Repositories

Create two empty repositories in the `infobloxopen` GitHub organization:

| Repository | Purpose |
|-----------|---------|
| `infobloxopen/homebrew-tap` | Homebrew formula hosting |
| `infobloxopen/scoop-bucket` | Scoop manifest hosting |

Both can be completely empty (no README, no license). GoReleaser creates the files on the first release.

### 2. GitHub App & Secrets

The release workflow uses the `infobloxopen-release-bot` GitHub App (App ID: 3033530) to mint short-lived tokens at runtime. No personal access tokens are needed.

Add these **org-level** secrets (already configured):

| Secret | Scope | Description |
|--------|-------|-------------|
| `RELEASE_APP_ID` | Org secret | GitHub App ID for `infobloxopen-release-bot` |
| `RELEASE_APP_PRIVATE_KEY` | Org secret | Private key for `infobloxopen-release-bot` |

The release workflow uses `actions/create-github-app-token@v1` to mint a short-lived token scoped to `homebrew-tap` and `scoop-bucket` repos.

### 3. Configuration Files

These are created by the implementation:

- `.goreleaser.yml` — already exists, needs `scoops` section and `token` in `brews`
- `.github/workflows/release.yml` — new release workflow
- `install.sh` — existing shell installer (repo root)

## How to Release

### Step 1: Tag

```bash
git tag v1.0.0
git push origin v1.0.0
```

### Step 2: Wait

The release workflow runs automatically (~5 minutes):

1. GoReleaser builds binaries for all platforms
2. Creates a GitHub Release with binaries + checksums
3. Pushes `Formula/apx.rb` to `infobloxopen/homebrew-tap`
4. Pushes `apx.json` to `infobloxopen/scoop-bucket`
5. Pushes Docker images to `ghcr.io/infobloxopen/apx`

### Step 3: Verify

```bash
# Check GitHub Release exists
open https://github.com/infobloxopen/apx/releases/latest

# Verify Homebrew
brew update && brew install infobloxopen/tap/apx
apx --version

# Verify Scoop (on Windows)
scoop bucket add infobloxopen https://github.com/infobloxopen/scoop-bucket
scoop install infobloxopen/apx
apx --version

# Verify shell installer
curl -fsSL https://raw.githubusercontent.com/infobloxopen/apx/main/install.sh | bash
~/.local/bin/apx --version
```

## Rollback

If a release has issues:

1. **Delete the GitHub Release** (UI or `gh release delete v1.0.0`)
2. **Delete the tag**: `git push origin :v1.0.0 && git tag -d v1.0.0`
3. The Homebrew tap and Scoop bucket still have the old version's formula/manifest until the next release overwrites it

## Testing Locally

```bash
# Validate GoReleaser config
goreleaser check

# Build without publishing (snapshot)
goreleaser release --snapshot --clean

# Check generated artifacts
ls dist/
```

## Troubleshooting

| Problem | Cause | Fix |
|---------|-------|-----|
| "token" error in release log | GitHub App key rotated or App uninstalled | Update `RELEASE_APP_PRIVATE_KEY` org secret or reinstall App |
| Homebrew formula not updated | App token minting failed | Check `RELEASE_APP_ID` and `RELEASE_APP_PRIVATE_KEY` org secrets |
| Scoop manifest not updated | App token minting failed | Check `RELEASE_APP_ID` and `RELEASE_APP_PRIVATE_KEY` org secrets |
| Docker push fails | Not logged into GHCR | Check `docker/login-action` step |
| `brew install` gets 404 | Tap repo doesn't exist | Create `infobloxopen/homebrew-tap` |
