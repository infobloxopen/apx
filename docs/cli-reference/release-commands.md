# Release Commands

The `apx release` command family manages releases through an explicit state machine.
While [`apx publish`](publishing-commands.md) is a single fire-and-forget command,
`apx release` breaks the workflow into discrete, auditable phases — ideal for CI
pipelines and production-grade release governance.

## Overview

| Subcommand | Purpose |
|------------|---------|
| `prepare`  | Validate and create a release manifest |
| `submit`   | Push the prepared release to the canonical repo |
| `finalize` | Tag, catalog-update, and emit a release record (canonical CI) |
| `inspect`  | Display the current release state or list tags |
| `history`  | List all published versions for an API |
| `promote`  | Create a manifest for a lifecycle promotion |

## State Machine

Every release progresses through a well-defined set of states:

```
draft → validated → version-selected → prepared → submitted →
  canonical-pr-open → canonical-validated → canonical-released → package-published
```

A release can also transition to **failed** from any state.  The current state
is persisted in `.apx-release.yaml` so the pipeline can resume or diagnose
failures at any point.

## `apx release prepare`

Validate schemas, lifecycle policy, version-line compatibility, and `go_package`
/ `go.mod` consistency, then write a release manifest.

```bash
apx release prepare <api-id> --version <semver> [flags]
```

### Examples

```bash
# Prepare an alpha release
apx release prepare proto/payments/ledger/v1 --version v1.0.0-alpha.1

# Prepare with explicit lifecycle
apx release prepare proto/payments/ledger/v1 --version v1.0.0 --lifecycle stable

# Strict mode — fail on go_package mismatch
apx release prepare proto/payments/ledger/v1 --version v1.0.0 --strict

# Override lifecycle checks
apx release prepare proto/payments/ledger/v1 --version v2.0.0 --force
```

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--version` | string | *(required)* | SemVer version to release |
| `--lifecycle` | string | auto | Lifecycle state (`experimental`, `preview`, `stable`, `deprecated`, `sunset`) |
| `--canonical-repo` | string | from config | Canonical repository URL |
| `--strict` | bool | false | Fail on `go_package` validation warnings |
| `--skip-gomod` | bool | false | Skip `go.mod` validation |
| `--force` | bool | false | Override lifecycle and policy checks |

### What Happens

1. API ID is parsed into format, domain, name, and line
2. Lifecycle policy is enforced (v0 line restrictions, transition rules)
3. Version-line compatibility is validated (major must match line)
4. `go_package` options in `.proto` files are checked against derived import paths
5. `go.mod` module path is validated if present
6. An idempotency check is run against existing tags (SHA-256 content hash)
7. Source commit is captured
8. The manifest (`.apx-release.yaml`) is written in `prepared` state

If the same version with identical content has already been published, the command
reports success and skips to `package-published`.

---

## `apx release submit`

Read the manifest and open a pull request on the canonical repository with the
prepared release content.

```bash
apx release submit [flags]
```

### Examples

```bash
# Submit the prepared release as a PR
apx release submit

# Preview without actually submitting
apx release submit --dry-run
```

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--dry-run` | bool | false | Show what would be submitted without doing it |

### What Happens

1. `.apx-release.yaml` is read — must be in `prepared` state
2. `gh` CLI is verified (installed and authenticated)
3. The canonical repo is cloned to a temp directory
4. A release branch `apx/release/<api-id>/<version>` is created
5. Snapshot files are copied and `go.mod` is generated if needed
6. Changes are committed and force-pushed to the release branch
7. A pull request is opened (or an existing one is detected)
8. PR metadata (number, URL, branch) is recorded in the manifest
9. Manifest transitions to `canonical-pr-open`

This operation is idempotent: re-running after a partial failure will detect
existing branches and PRs, recovering gracefully without creating duplicates.
Re-running after a full success reports the existing PR and exits.

### CI Provenance

When running in CI (GitHub Actions, GitLab CI, or Jenkins), the PR body
automatically includes the CI provider name and run URL for audit trail.
CI metadata is also recorded in the manifest (`ci_provider`, `ci_run_url`).

---

## `apx release finalize`

Run by **canonical CI** after a release has been submitted.  Re-validates the
schema, creates the official tag, updates the catalog, and emits an immutable
release record.

```bash
apx release finalize [flags]
```

### Examples

```bash
# Standard finalization
apx release finalize

# Custom catalog path
apx release finalize --catalog catalog.yaml

# Skip language package publication
apx release finalize --skip-packages

# Skip catalog update
apx release finalize --skip-catalog
```

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--catalog` | string | `catalog.yaml` | Path to the catalog file |
| `--skip-packages` | bool | false | Skip language package publication |
| `--skip-catalog` | bool | false | Skip catalog update |
| `--record-path` | string | `.apx-release-record.yaml` | Path to write the release record |

### What Happens

1. Manifest is read — must be in `submitted` or `canonical-pr-open` state
2. Schema is re-validated (lint + breaking-change check against the previous version)
3. Policy validation is run
4. An annotated git tag is created and pushed
5. The catalog entry is created or updated (version, lifecycle, latest-stable/prerelease)
6. Language package artifacts are recorded
7. An immutable **release record** (`.apx-release-record.yaml`) is written with CI provenance (auto-detects GitHub Actions, GitLab CI, Jenkins)

---

## `apx release inspect`

Display the current release state from the manifest, or list published tags for
an API.

```bash
apx release inspect [api-id] [flags]
```

### Examples

```bash
# Show current manifest state
apx release inspect

# Show tags for a specific API (when no manifest exists)
apx release inspect proto/payments/ledger/v1

# JSON output
apx release inspect --json
```

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--json` | bool | false | Output in JSON format |

---

## `apx release history`

List all published versions for an API, extracted from git tags.  Versions are
sorted newest-first.

```bash
apx release history <api-id> [flags]
```

### Examples

```bash
# Table output (default)
apx release history proto/payments/ledger/v1

# JSON output
apx release history proto/payments/ledger/v1 --format json
```

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--format` | string | `table` | Output format: `table`, `json` |

### Sample Output

```
Release history for proto/payments/ledger/v1:

  VERSION              LIFECYCLE      TAG
  -------              ---------      ---
  v1.2.0               stable         proto/payments/ledger/v1/v1.2.0
  v1.1.0               stable         proto/payments/ledger/v1/v1.1.0
  v1.0.0               stable         proto/payments/ledger/v1/v1.0.0
  v1.0.0-beta.1        preview        proto/payments/ledger/v1/v1.0.0-beta.1
  v1.0.0-alpha.1       experimental   proto/payments/ledger/v1/v1.0.0-alpha.1

Total: 5 release(s)
```

---

## `apx release promote`

Create a release manifest that moves an API forward in its lifecycle
(e.g. `preview` → `stable`).  The promotion produces a prepared manifest that
is then submitted with `apx release submit`.

```bash
apx release promote <api-id> --to <lifecycle> [flags]
```

### Examples

```bash
# Promote to stable with an explicit version
apx release promote proto/payments/ledger/v1 --to stable --version v1.0.0

# Promote to deprecated (version is auto-derived)
apx release promote proto/payments/ledger/v1 --to deprecated

# Override lifecycle transition checks
apx release promote proto/payments/ledger/v1 --to stable --force
```

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--to` | string | *(required)* | Target lifecycle (`preview`, `stable`, `deprecated`, `sunset`) |
| `--version` | string | auto | Version for the promoted release |
| `--canonical-repo` | string | from config | Canonical repository URL |
| `--force` | bool | false | Override lifecycle checks |

### What Happens

1. Current lifecycle is resolved from the manifest, config, or latest git tag
2. Lifecycle transition is validated (must move forward: experimental → preview → stable → deprecated → sunset)
3. If `--version` is omitted, a version is auto-derived (e.g. strip prerelease for stable promotion)
4. A prepared manifest is written — the next step is `apx release submit`

---

## Release Manifest

The release manifest (`.apx-release.yaml`) is the central artifact that tracks
a release through the state machine.  It records:

- **Identity** — API ID, format, domain, name, line
- **Source provenance** — repo, path, commit SHA
- **Version & tag** — requested version, derived tag
- **Go coordinates** — module path, import path
- **Validation results** — lint, breaking, policy, go\_package, go.mod
- **Timestamps** — created, last-updated
- **Error info** — error code, message, hint, phase (if failed)

The manifest is written by `prepare` and read/updated by every subsequent phase.

## Release Record

The immutable release record (`.apx-release-record.yaml`) is emitted by
`finalize`.  It captures everything from the manifest plus:

- **Canonical commit** — the commit SHA in the canonical repo
- **Published artifacts** — type, name, version, status for each language package
- **Catalog update** — whether the catalog was updated and which file
- **CI provenance** — auto-detected CI system name, job ID, run URL

## `publish` vs `release`

| Aspect | `apx publish` | `apx release` |
|--------|---------------|----------------|
| Steps | Single command | `prepare` → `submit` → `finalize` |
| Manifest | None | `.apx-release.yaml` persisted between steps |
| Audit trail | Minimal | Immutable release record with CI provenance |
| Idempotency | Best-effort | SHA-256 content hashing with explicit result codes |
| Catalog | Not updated | Updated during `finalize` |
| Best for | Quick iterations, local development | CI pipelines, production releases |

Use `apx publish` when you want a fast, one-shot publish.  Use `apx release`
when you need a traceable, multi-step pipeline with validation gates and audit
records.
