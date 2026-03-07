# Data Model: Package Installer Support

**Feature**: 004-package-installers  
**Date**: 2026-03-07

## Entities

This feature is primarily infrastructure/CI — it has no runtime data model in the APX Go codebase. Instead, the "entities" are configuration files, external repositories, and release artifacts.

### GoReleaser Configuration (`.goreleaser.yml`)

Extends the existing configuration file with Scoop support and token references.

| Field | Type | Description |
|-------|------|-------------|
| `brews[].repository.token` | string (env ref) | GitHub App token for pushing to Homebrew tap repo |
| `scoops[]` | array | New section for Scoop bucket publishing |
| `scoops[].name` | string | Package name (`apx`) |
| `scoops[].repository.owner` | string | GitHub org (`infobloxopen`) |
| `scoops[].repository.name` | string | Bucket repo name (`scoop-bucket`) |
| `scoops[].repository.token` | string (env ref) | GitHub App token for pushing to Scoop bucket repo |
| `scoops[].homepage` | string | Project homepage URL |
| `scoops[].description` | string | Package description |
| `scoops[].license` | string | License identifier |

### Homebrew Formula (auto-generated in `infobloxopen/homebrew-tap`)

GoReleaser generates `Formula/apx.rb` — a Ruby file. Not hand-authored.

| Field | Description |
|-------|-------------|
| `desc` | Package description |
| `homepage` | Project URL |
| `url` | Download URL for the release archive |
| `sha256` | SHA256 hash of the archive |
| `version` | Release version |
| `install` | Installation instructions (binary + completions) |
| `test` | Verification command (`apx --version`) |

### Scoop Manifest (auto-generated in `infobloxopen/scoop-bucket`)

GoReleaser generates `apx.json` — a JSON file in the repo root.

| Field | Type | Description |
|-------|------|-------------|
| `version` | string | Release version |
| `architecture.64bit.url` | string | Download URL for windows/amd64 archive |
| `architecture.64bit.hash` | string | SHA256 hash |
| `architecture.64bit.bin` | string | Binary name (`apx.exe`) |
| `homepage` | string | Project URL |
| `license` | string | License identifier |
| `description` | string | Package description |

### Release Workflow (`.github/workflows/release.yml`)

GitHub Actions workflow definition — YAML, not runtime data.

| Field | Description |
|-------|-------------|
| `on.push.tags` | Trigger pattern (`v*`) |
| `permissions.contents` | `write` for GitHub Release creation |
| `env.GITHUB_TOKEN` | Built-in token for release assets |
| `env.HOMEBREW_TAP_TOKEN` | GitHub App token (minted via `actions/create-github-app-token@v1`) for Homebrew tap push |
| `env.SCOOP_BUCKET_TOKEN` | GitHub App token (minted via `actions/create-github-app-token@v1`) for Scoop bucket push |

### Install Script (`install.sh`)

Standalone bash script at the repo root — no data model, but key parameters:

| Parameter | Source | Default | Description |
|-----------|--------|---------|-------------|
| `VERSION` | env var | latest release | Specific version to install |
| `INSTALL_DIR` | env var | `~/.local/bin` | Target directory for binary |
| `GITHUB_TOKEN` | env var | none | Optional token for private repos / rate limits |

## Relationships

```
Tag Push (v*.*.*)
    └── Release Workflow (.github/workflows/release.yml)
            ├── GoReleaser (.goreleaser.yml)
            │       ├── Build binaries (linux/darwin/windows × amd64/arm64)
            │       ├── Create GitHub Release + checksums
            │       ├── Push Formula/apx.rb → infobloxopen/homebrew-tap
            │       └── Push apx.json → infobloxopen/scoop-bucket
            └── (Independent) Install script downloads from GitHub Release
```

## State Transitions

No runtime state transitions. The release is a one-shot pipeline:

```
Tag Pushed → Workflow Triggered → GoReleaser Running → Artifacts Published → Package Managers Updated
```

If any package manager update fails (e.g., token expired), the GitHub Release still exists and other package managers still update. Each is independent.

## External Repositories Required

| Repository | Purpose | Initial State | Created By |
|-----------|---------|---------------|------------|
| `infobloxopen/homebrew-tap` | Homebrew formula hosting | Empty repo | Org admin (manual) |
| `infobloxopen/scoop-bucket` | Scoop manifest hosting | Empty repo | Org admin (manual) |

## Secrets Required

| Secret Name | Scope | Purpose |
|------------|-------|---------|
| `RELEASE_APP_ID` | Org secret | GitHub App ID for `infobloxopen-release-bot` |
| `RELEASE_APP_PRIVATE_KEY` | Org secret | Private key for `infobloxopen-release-bot` |
