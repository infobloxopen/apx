# Branch and Tag Protection

The canonical API repository relies on two layers of protection to ensure that only reviewed, validated changes reach `main` and only CI creates release tags. APX can configure both automatically via `apx init canonical --setup-github`, or you can set them up manually through the GitHub UI.

## Why Protection Matters

Without protection rules, any contributor with write access could:

- Push directly to `main`, bypassing lint and breaking-change checks
- Create release tags manually, skipping the validated CI pipeline
- Delete or overwrite existing tags, corrupting consumer dependency resolution

APX's protection model prevents all three:

| Threat | Protection Layer |
|--------|-----------------|
| Unreviewed code on `main` | Branch protection with required PR reviews |
| Invalid schemas merged | Required status checks (`validate` job) |
| Bypassing CODEOWNERS | Required Code Owner reviews |
| Manual tag creation | Tag protection ruleset (`apx-tag-protection`) |
| Tag deletion/overwrite | Tag protection ruleset (restrict deletions) |

## Branch Protection on `main`

### What APX Configures

When you run `--setup-github`, APX calls the GitHub Branch Protection API (`PUT repos/{owner}/{repo}/branches/main/protection`) with these settings:

```json
{
  "required_status_checks": {
    "strict": true,
    "contexts": ["validate"]
  },
  "enforce_admins": false,
  "required_pull_request_reviews": {
    "required_approving_review_count": 1,
    "require_code_owner_reviews": true,
    "dismiss_stale_reviews": true
  },
  "restrictions": null
}
```

### Settings Breakdown

| Setting | Value | Purpose |
|---------|-------|---------|
| **Required status checks** | `validate` (strict) | The `ci.yml` workflow's `validate` job must pass. "Strict" means the branch must be up to date with `main` before merging. |
| **Required approving reviews** | 1 | At least one team member must approve the PR. |
| **Require Code Owner reviews** | `true` | If the changed path has a CODEOWNERS entry, a member of that team must approve. |
| **Dismiss stale reviews** | `true` | Pushing new commits to the PR dismisses previous approvals, requiring re-review. |
| **Enforce admins** | `false` | Org admins can bypass in emergencies (e.g., hotfixes). Set to `true` for stricter governance. |
| **Restrictions** | `null` | No push restrictions beyond the PR requirement — anyone with write access can open a PR. |

### The `validate` Status Check

The `validate` job comes from the generated `.github/workflows/ci.yml`:

```yaml
name: APX Schema CI
on:
  pull_request:
    branches: [main]

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: infobloxopen/apx@v1
      - name: Lint schemas
        run: apx lint
      - name: Check for breaking changes
        run: apx breaking --against origin/main
```

This ensures every PR is linted and checked for breaking changes before it can be merged.

### Idempotent Behavior

`EnsureBranchProtection` is idempotent — it checks whether protection already exists before creating it:

- **Already exists**: Reported as `✓ Already configured: branch protection on main`
- **Created**: Reported as `✓ Created: branch protection on main`
- **Permission denied (403)**: Reported as `⚠ Requires admin: branch protection on main`

## Tag Protection Ruleset

### Why Rulesets (Not Legacy Tag Protection)

GitHub offers two tag protection mechanisms:

1. **Legacy tag protection rules** — Simple pattern matching, limited control
2. **Repository rulesets** — Fine-grained control with bypass actors, enforcement levels, and audit logging

APX uses **repository rulesets** because they support bypass actors, allowing the GitHub App (via CI) and organization admins to create tags while blocking everyone else.

### What APX Configures

APX creates a ruleset named `apx-tag-protection` via the GitHub Rulesets API (`POST repos/{owner}/{repo}/rulesets`):

```json
{
  "name": "apx-tag-protection",
  "target": "tag",
  "enforcement": "active",
  "conditions": {
    "ref_name": {
      "include": ["~ALL"],
      "exclude": []
    }
  },
  "rules": [
    { "type": "creation" },
    { "type": "deletion" }
  ],
  "bypass_actors": [
    {
      "actor_type": "OrganizationAdmin",
      "actor_id": 1,
      "bypass_mode": "always"
    }
  ]
}
```

### Settings Breakdown

| Setting | Value | Purpose |
|---------|-------|---------|
| **Target** | `tag` | Applies to git tags, not branches |
| **Enforcement** | `active` | Rules are enforced (not in "evaluate" audit-only mode) |
| **Ref pattern** | `~ALL` | Protects all tags — not just APX-formatted ones |
| **Restrict creation** | Enabled | Prevents direct `git tag` + `git push --tags` |
| **Restrict deletion** | Enabled | Prevents `git push --delete origin <tag>` |
| **Bypass: OrgAdmin** | Always | Organization admins can bypass for emergencies |

### How CI Creates Tags

The `on-merge.yml` workflow runs with a GitHub App token that has `contents:write` permission. Since the App is installed at the organization level, it can bypass the tag ruleset:

```yaml
- name: Generate App Token
  id: app-token
  uses: actions/create-github-app-token@v1
  with:
    app-id: ${{ secrets.APX_APP_ID }}
    private-key: ${{ secrets.APX_APP_PRIVATE_KEY }}

- uses: actions/checkout@v4
  with:
    token: ${{ steps.app-token.outputs.token }}
```

The App token carries the `contents:write` permission required to push tags.

### Tag Patterns

APX's tagging convention uses subdirectory-scoped tags:

```
<format>/<domain>/<api>/<line>/v<semver>
```

Examples:

```bash
proto/payments/ledger/v1/v1.0.0
proto/payments/ledger/v1/v1.2.3
openapi/users/v1/v1.0.0
avro/events/v1/v1.5.0
```

The `~ALL` pattern protects every tag, which is the safest default. If you need to allow non-APX tags (e.g., application release tags), you can edit the ruleset conditions to target specific patterns:

```json
"conditions": {
  "ref_name": {
    "include": [
      "refs/tags/proto/**",
      "refs/tags/openapi/**",
      "refs/tags/avro/**",
      "refs/tags/jsonschema/**",
      "refs/tags/parquet/**"
    ],
    "exclude": []
  }
}
```

### Idempotent Behavior

`EnsureTagProtection` checks for an existing ruleset named `apx-tag-protection` before creating:

- **Already exists**: Reported as `✓ Already configured: tag protection ruleset`
- **Name collision**: Detected via `Name must be unique` error, reported as skipped
- **Created**: Reported as `✓ Created: tag protection ruleset`
- **Permission denied**: Reported as `⚠ Requires admin: tag protection ruleset: <detail>`

## GitHub App & Org Secrets

The protection model depends on a GitHub App providing CI with elevated permissions without sharing personal access tokens.

### Required Org Secrets

| Secret | Purpose | Visibility |
|--------|---------|------------|
| `APX_APP_ID` | GitHub App numeric ID | All repositories |
| `APX_APP_PRIVATE_KEY` | GitHub App private key (PEM) | All repositories |

These secrets are consumed by `actions/create-github-app-token@v1` in the CI workflows to generate short-lived installation tokens.

### App Permissions

The GitHub App created by `--setup-github` requests these permissions:

| Permission | Level | Purpose |
|------------|-------|---------|
| `contents` | `write` | Push tags, commit catalog updates |
| `pull_requests` | `write` | Create release submission PRs |
| `metadata` | `read` | Basic repo metadata access |

### Why a GitHub App (Not a PAT)

| Aspect | Personal Access Token | GitHub App |
|--------|----------------------|------------|
| **Scope** | Tied to a user account | Org-level installation |
| **Audit** | Actions appear as the user | Actions appear as `apx-<repo>-<org>[bot]` |
| **Rotation** | Manual | Automatic (short-lived tokens) |
| **Revocation** | Affects all repos using that PAT | Per-installation control |
| **Offboarding** | Token invalid when user leaves | Survives personnel changes |

## Automated Setup

Run the full automated setup:

```bash
apx init canonical --org=<org> --repo=apis --setup-github
```

This performs all four steps in sequence:

1. Creates (or reuses) the GitHub App
2. Sets `APX_APP_ID` and `APX_APP_PRIVATE_KEY` as org secrets
3. Configures branch protection on `main`
4. Creates the `apx-tag-protection` tag ruleset

See [Canonical Repository Setup](setup.md) for the complete walkthrough.

## Manual Setup

If you lack org admin access or prefer manual configuration:

### Branch Protection (GitHub UI)

1. Go to **Settings → Branches → Add branch protection rule**
2. Branch name pattern: `main`
3. Enable:
   - [x] Require a pull request before merging
     - Required approvals: **1**
     - [x] Require review from Code Owners
     - [x] Dismiss stale pull request approvals when new commits are pushed
   - [x] Require status checks to pass before merging
     - [x] Require branches to be up to date before merging
     - Status checks: add **`validate`**
   - [ ] Require signed commits (optional, recommended)
4. Click **Create**

### Tag Ruleset (GitHub UI)

1. Go to **Settings → Rules → Rulesets → New ruleset → New tag ruleset**
2. Configure:
   - **Ruleset name**: `apx-tag-protection`
   - **Enforcement status**: Active
   - **Bypass list**: Add "Organization admin" with "Always" bypass mode
   - **Target tags**: All tags
   - **Rules**: Enable "Restrict creations" and "Restrict deletions"
3. Click **Create**

### Org Secrets (GitHub UI or CLI)

If someone else created the GitHub App, get the App ID and PEM from them and set:

```bash
# Via gh CLI (requires admin:org scope)
gh secret set APX_APP_ID --org <org> --visibility all --body "<app-id>"
gh secret set APX_APP_PRIVATE_KEY --org <org> --visibility all < private-key.pem
```

Or via **Organization Settings → Secrets and variables → Actions → New organization secret**.

## Verifying Protection

### Branch Protection

```bash
# Check branch protection exists
gh api repos/<org>/apis/branches/main/protection \
  --jq '{
    reviews: .required_pull_request_reviews.required_approving_review_count,
    codeowners: .required_pull_request_reviews.require_code_owner_reviews,
    status_checks: .required_status_checks.contexts
  }'
```

Expected output:

```json
{
  "reviews": 1,
  "codeowners": true,
  "status_checks": ["validate"]
}
```

### Tag Ruleset

```bash
# List rulesets
gh api repos/<org>/apis/rulesets --jq '.[].name'
# → apx-tag-protection

# Verify tag push is blocked (should fail for non-admin users)
git tag test-protection && git push origin test-protection
# → remote: error: GH013: Repository rule violations found ...
```

### Org Secrets

```bash
gh secret list --org <org> | grep APX_
# → APX_APP_ID          Updated 2026-03-08
# → APX_APP_PRIVATE_KEY Updated 2026-03-08
```

## Customization

### Stricter Branch Protection

To also enforce rules on admins:

```bash
gh api repos/<org>/apis/branches/main/protection \
  --method PUT --input - <<'EOF'
{
  "required_status_checks": { "strict": true, "contexts": ["validate"] },
  "enforce_admins": true,
  "required_pull_request_reviews": {
    "required_approving_review_count": 2,
    "require_code_owner_reviews": true,
    "dismiss_stale_reviews": true
  },
  "restrictions": null
}
EOF
```

### Adding More Status Checks

If you add custom CI jobs (e.g., integration tests), add them to the required contexts:

```bash
gh api repos/<org>/apis/branches/main/protection \
  --method PUT --input - <<'EOF'
{
  "required_status_checks": {
    "strict": true,
    "contexts": ["validate", "integration-tests", "security-scan"]
  },
  "enforce_admins": false,
  "required_pull_request_reviews": {
    "required_approving_review_count": 1,
    "require_code_owner_reviews": true,
    "dismiss_stale_reviews": true
  },
  "restrictions": null
}
EOF
```

### Scoped Tag Protection

To protect only APX tag patterns while allowing other tags:

1. Go to **Settings → Rules → Rulesets → apx-tag-protection → Edit**
2. Change target from "All tags" to "Include by pattern"
3. Add patterns: `proto/**`, `openapi/**`, `avro/**`, `jsonschema/**`, `parquet/**`

## Troubleshooting

### "403 Forbidden" on Branch Protection

You need admin access to the repository. Ask an org admin to run:

```bash
apx init canonical --org=<org> --repo=apis --setup-github
```

Or configure manually in the GitHub UI under **Settings → Branches**.

### "admin:org scope" Error

Your `gh` token needs the `admin:org` scope for org-level secrets:

```bash
gh auth refresh -h github.com -s admin:org
```

### Tag Push Rejected

If `git push --tags` fails with a ruleset violation, the tag protection is working correctly. Tags should only be created by CI via the `on-merge.yml` workflow. If you need to create a tag manually in an emergency, use an org admin account.

### Stale Review Dismissal Confusion

If approvals keep getting dismissed, it's because `dismiss_stale_reviews` is enabled. Any new push to the PR requires re-approval. This is intentional — it prevents approving a PR and then pushing additional unreviewed changes.

## Related Pages

- [Canonical Repository Setup](setup.md) — Full setup walkthrough including `--setup-github`
- [CI Templates](ci-templates.md) — The workflows that depend on these protections
- [Repository Structure](structure.md) — Directory layout and CODEOWNERS patterns
- [Tagging Strategy](../publishing/tagging-strategy.md) — How tags are formatted and versioned
