# Releasing on a `ci_only` Repo (CI Handoff)

Some canonical repositories set `release.ci_only: true` in `apx.yaml` (for
example `infobloxopen/apis`). On these repos, `apx release finalize` — the step
that creates the version **tag** and updates `catalog.yaml` — runs in
**canonical CI**, not on a contributor's machine. This page documents the
end-to-end contributor flow, the CI handoff, the prerequisites, and how
downstream consumers sequence a `go get` around the new tag.

## Why finalize is CI-gated

Creating the release tag on a `ci_only` repo writes a **protected** git tag.
Only the canonical CI identity — a GitHub App with a **tag-ruleset bypass** —
is allowed to push it. A schema author typically cannot (and should not) hold
those credentials, so finalize is delegated to CI:

- `prepare` and `submit` are **local** steps a contributor runs.
- `finalize` is a **CI** step that runs after the release PR merges.

## The contributor flow

```bash
# 1. Prepare — local. Validates and writes .apx-release.yaml.
apx release prepare proto/infoblox/field/v1 \
  --version v1.0.0-alpha.2 --lifecycle experimental

# 2. Submit — local. Opens a PR against the canonical repo.
apx release submit
```

On a `ci_only` repo, both `prepare` and `submit` print a **preflight notice**
that finalize will run in CI and what CI needs. This makes the handoff visible
up front instead of surfacing as an opaque wall later.

```text
This canonical repo is ci_only — 'apx release finalize' runs in canonical CI, not locally.
After your release PR merges, canonical CI finalizes it. That requires:
  - the apx-release (canonical CI GitHub App) installed on the canonical org
  - org secrets APX_APP_ID and APX_APP_PRIVATE_KEY set for that app
  - a tag-ruleset bypass entry for the app so it can push the protected version tag
```

```text
# 3. Merge the release PR (review as usual).

# 4. Finalize — CI. Canonical CI runs, on the merge commit:
apx release finalize --api proto/infoblox/field/v1 --version v1.0.0-alpha.2
```

Because CI runs on the merge commit (where the producer's local
`.apx-release.yaml` is not present), it uses **CI mode**: `--api` and
`--version` reconstruct the manifest from the canonical repo's config.

## CI prerequisites (one-time setup)

An org admin configures these once on the canonical repo/org:

| Prerequisite | What it is | How to satisfy |
|--------------|-----------|----------------|
| GitHub App install | The `apx-release` app that finalizes releases | Install the app on the canonical org with `contents:write` |
| `APX_APP_ID` | Org secret: the app's numeric ID | Set as an **organization** (or repo) Actions secret |
| `APX_APP_PRIVATE_KEY` | Org secret: the app's private key (PEM) | Set as an **organization** (or repo) Actions secret |
| Tag-ruleset bypass | Lets the app push the protected version tag | Add the app to the tag ruleset's **bypass list** |

If any piece is missing, finalize cannot produce the tag. `apx` fails fast with
this exact list rather than emitting a generic CI error.

!!! note
    These are org-level settings a contributor usually cannot inspect with their
    own token, so `apx` **surfaces** them (in the preflight notice and the
    finalize guidance) rather than probing for them.

## Local fallback: `apx release finalize --local`

If you *do* control the credentials (a personal access token with
`contents:write` on the canonical repo **and** a tag-ruleset bypass for your
identity), you can run the CI-mode finalize from your machine:

```bash
apx release finalize --local \
  --api proto/infoblox/field/v1 --version v1.0.0-alpha.2
```

`--local` bypasses the CI-only guard. It does **not** relax tag protection: if
the push of the protected tag fails (missing bypass or token scope), finalize
**fails loudly** with guidance — it never leaves a local-only tag that looks
released but isn't.

Without `--local` (and outside CI), finalize on a `ci_only` repo fails fast:

```text
finalize runs in canonical CI for this repo (release.ci_only: true) and is not
completable locally by default.
...
Recommended path (no local credentials needed):
  1. Get the release PR reviewed and merged.
  2. Canonical CI runs, on the merge commit:
       apx release finalize --api proto/infoblox/field/v1 --version v1.0.0-alpha.2
Local fallback (only if you control the credentials): re-run with --local ...
```

## Nothing to release (empty diff)

If the prepared snapshot is byte-identical to what's already in the canonical
repo, `apx release submit` exits cleanly with a clear message instead of
producing an opaque GitHub `HTTP 422` from an empty pull request:

```text
Nothing to release: the prepared snapshot for proto/infoblox/field/v1 @ v1.0.0-alpha.2
is identical to the canonical repo.

This usually means the content was already submitted or merged.
Next step: if the release tag proto/infoblox/field/v1.0.0-alpha.2 does not exist yet, finalize it:
  (ci_only repo) merge the release PR — canonical CI runs 'apx release finalize'.
```

## Catalog reconcile and drift

`finalize` idempotently reconciles `catalog.yaml`: it creates or updates the
entry for the module being released (re-running is safe). It also **surfaces
drift** — modules that have release **tags** but no `catalog.yaml` entry (for
example, a tag created by an earlier partial run):

```text
Catalog drift: 1 tagged module(s) missing from catalog.yaml:
  - proto/infoblox/field (has release tag(s), no catalog entry)
Reconcile with 'apx catalog generate' or finalize each missing module.
```

## Downstream: tag-before-consume sequencing

A Go consumer cannot `go get` a new API version until its **tag exists** in the
canonical repo. On a `ci_only` repo, the tag only lands **after** CI finalize
runs — so the ordering is:

1. Merge the release PR.
2. Canonical CI runs finalize and pushes the tag.
3. Consumers `go get <module>@<version>`.

### The `replace` bridge for local development

While the tag is still in flight (PR open, or CI not yet finalized), a consumer
can develop against the not-yet-tagged code using a temporary Go module
`replace` directive:

```go
// go.mod (TEMPORARY — remove once the tag is published)
replace github.com/infobloxopen/apis/proto/infoblox/field/v1 => \
    ../apis/proto/infoblox/field/v1
```

Or against a fork/branch checkout. Once CI publishes the tag, remove the
`replace` and pin the real version:

```bash
go get github.com/infobloxopen/apis/proto/infoblox/field/v1@v1.0.0-alpha.2
go mod tidy
```

!!! warning
    A `replace` directive is a **local bridge only**. Never release or publish a
    module that still carries a `replace` to an unreleased dependency — `apx`'s
    release drift gate blocks this by design.

## See also

- [Releasing Overview](overview.md) — identity model and responsibility boundary
- [Canonical Pull Request](canonical-pr.md) — the submit → PR flow
- [Release Commands](../cli-reference/release-commands.md) — full flag reference
