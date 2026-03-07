# Contract: Release Workflow

**File**: `.github/workflows/release.yml`  
**Type**: New file

## Trigger

```yaml
on:
  push:
    tags:
      - "v*"
```

Fires on any tag matching `v*` (e.g., `v1.0.0`, `v2.1.0-rc.1`). GoReleaser's `prerelease: auto` in `.goreleaser.yml` handles pre-release detection.

## Permissions

```yaml
permissions:
  contents: write    # Create GitHub Release + upload assets
  packages: write    # Push Docker images to ghcr.io
```

## Authentication

Uses the `infobloxopen-release-bot` GitHub App (App ID: 3033530) to mint short-lived tokens at runtime via `actions/create-github-app-token@v1`. The minted token has write access to `homebrew-tap` and `scoop-bucket` repos.

| Secret | Scope | Description |
|--------|-------|-------------|
| `GITHUB_TOKEN` | Built-in | Create release, upload assets |
| `RELEASE_APP_ID` | Org secret | GitHub App ID for `infobloxopen-release-bot` |
| `RELEASE_APP_PRIVATE_KEY` | Org secret | Private key for `infobloxopen-release-bot` |

The minted app token is passed to GoReleaser as `HOMEBREW_TAP_TOKEN` and `SCOOP_BUCKET_TOKEN` environment variables.

## Workflow Steps

```yaml
name: Release
on:
  push:
    tags:
      - "v*"

permissions:
  contents: write
  packages: write

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
          cache: true

      - name: Generate GitHub App token
        id: app-token
        uses: actions/create-github-app-token@v1
        with:
          app-id: ${{ secrets.RELEASE_APP_ID }}
          private-key: ${{ secrets.RELEASE_APP_PRIVATE_KEY }}
          owner: infobloxopen
          repositories: homebrew-tap,scoop-bucket

      - name: Login to GHCR
        uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - name: Set up QEMU
        uses: docker/setup-qemu-action@v3

      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3

      - name: Run GoReleaser
        uses: goreleaser/goreleaser-action@v6
        with:
          version: "~> v2"
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          HOMEBREW_TAP_TOKEN: ${{ steps.app-token.outputs.token }}
          SCOOP_BUCKET_TOKEN: ${{ steps.app-token.outputs.token }}
```

## Outputs

On success, GoReleaser produces:
1. **GitHub Release** with binaries, checksums, changelog
2. **Homebrew formula** pushed to `infobloxopen/homebrew-tap` → `Formula/apx.rb`
3. **Scoop manifest** pushed to `infobloxopen/scoop-bucket` → `apx.json`
4. **Docker images** pushed to `ghcr.io/infobloxopen/apx`
5. **deb/rpm packages** attached to the GitHub Release

## Failure Modes

| Failure | Impact | Recovery |
|---------|--------|----------|
| GitHub App key rotated | Tap/bucket not updated; release still succeeds | Update `RELEASE_APP_PRIVATE_KEY` org secret |
| Build failure | No release created | Fix code, delete tag, re-tag |
| Docker push failure | No images; release + pkg managers still succeed | Re-run workflow |
