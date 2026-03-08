# Publishing Failures

Troubleshooting guide for errors during `apx publish`, `apx release`, and canonical CI.

## Authentication & Permissions

### `gh: not authenticated`

```
Error: gh: not authenticated. Run "gh auth login" to authenticate.
```

**Cause:** The GitHub CLI is required for PR-based publishing but is not authenticated.

**Fix:**
```bash
gh auth login
# Choose "GitHub.com" → "HTTPS" → authenticate via browser
```

### CI: `could not create installation token`

```
Error: could not create installation token for app
```

**Cause:** The GitHub App is not installed on the target repository, or the secrets are misconfigured.

**Fix:**
1. Ensure the GitHub App is installed on both the source and canonical repos
2. Verify org-level secrets:
   - `APX_APP_ID` — the GitHub App's numeric ID
   - `APX_APP_PRIVATE_KEY` — the App's PEM private key
3. Confirm the App has write access to Contents, Pull Requests, and Metadata

### `remote: Permission denied`

```
remote: Permission to org/apis.git denied to github-actions[bot]
```

**Cause:** The generated token doesn't have write access to the canonical repo.

**Fix:**
- Confirm the GitHub App is installed on the **canonical** repo (not just the source)
- Check the App's repository access settings in **Organization → Settings → GitHub Apps**

---

## PR Creation Failures

### `gh pr create` fails with 422

```
GraphQL: Validation Failed (422)
```

**Cause:** A PR with the same branch name already exists, or the branch is empty.

**Fix:**
- Check for existing open PRs: `gh pr list --repo org/apis --head apx/proto/payments/ledger/v1/v1.0.0`
- If a stale PR exists, close it and retry
- Verify the snapshot actually contains file changes

### PR opened but CI fails on canonical repo

The canonical repo's `ci.yml` re-validates schemas in the canonical context.

**Common causes:**
- **Conflicting proto packages** — another API in the canonical repo uses the same proto package name
- **Inconsistent `buf.yaml`** — the canonical repo's `buf.yaml` has different lint rules
- **Breaking changes detected** — `apx breaking --against origin/main` fails because the baseline in the canonical repo differs from the app repo

**Fix:** Fix the issue locally, bump the version or correct the schema, and re-publish.

---

## Version & Release Errors

### `version already published`

```
Release proto/payments/ledger/v1@v1.0.0 already exists with identical content (SHA-256 match)
```

This is an **informational message**, not an error. APX's idempotency check detected that the exact same content was already published at this version. No action is needed.

### `version v2.0.0 is incompatible with API line v1`

The SemVer major version must match the declared API line. To release v2.0.0, create a new `v2` API directory:

```
proto/payments/ledger/v2/
```

Then publish with `proto/payments/ledger/v2` as the API ID.

### `lifecycle "stable" requires a stable version`

Prerelease versions (e.g. `-beta.1`) cannot be published under the `stable` lifecycle. Either:
- Change the lifecycle to `preview` (allows prerelease versions)
- Remove the prerelease suffix to publish a stable version

---

## Release Pipeline Errors

### `release state is "draft", expected "prepared"`

```
Error: cannot submit: release state is "draft", expected "prepared"
```

**Cause:** Attempted to run `apx release submit` before `apx release prepare`.

**Fix:** Follow the pipeline in order:
```bash
apx release prepare    # 1. validate and stage
apx release submit     # 2. create PR on canonical
apx release finalize   # 3. tag after merge
```

### `release submit` on a non-existent canonical repo

```
Error: repository "org/apis" not found
```

**Fix:**
- Verify `canonical_repo` in `apx.yaml` matches the actual repo name
- Ensure the GitHub App (CI) or your credentials (local) have access to the repo

### Merge conflicts on canonical PR

If another API was merged to the canonical repo between `submit` and merge, the PR may have conflicts.

**Fix:**
1. Close the existing PR
2. Re-run `apx release submit` — it will create a fresh PR based on latest `main`

---

## Tag Errors

### `tag already exists`

```
Error: tag proto/payments/ledger/v1/v1.0.0 already exists
```

**Cause:** The version was previously published and tagged.

**Fix:** Bump the version (patch at minimum) for any new release.

### Tags not visible after finalize

```bash
# Fetch tags from canonical
git fetch --tags upstream
git tag -l 'proto/payments/ledger/v1/*'
```

**Cause:** Tags are pushed to the canonical remote, not the app repo. If your git remotes don't include the canonical repo, the tags won't appear.

---

## Dry Run

Use `--dry-run` to preview the full publishing flow without creating branches, PRs, or tags:

```bash
apx publish proto/payments/ledger/v1 --version v1.0.0 --dry-run
apx release submit --dry-run
```

## See Also

- [Common Errors](common-errors.md) — general APX error reference
- [Release Guardrails](../publishing/release-guardrails.md) — lifecycle and version enforcement
- [Publish Command](../publishing/publish-command.md) — full flag reference
- [CI Integration](../app-repos/ci-integration.md) — CI workflow setup
