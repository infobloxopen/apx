# Canonical Pull Request

`apx release submit` opens a pull request against the canonical repository
(`github.com/<org>/apis`).  This page describes the end-to-end flow.

## Prerequisites

| Requirement | How to satisfy |
|-------------|----------------|
| `gh` CLI installed | `brew install gh` or [cli.github.com](https://cli.github.com) |
| `gh` authenticated | `gh auth login` |
| Push access to canonical repo | Org admin grants write or maintainer role |

## Flow

```
App repo                          Canonical repo
────────                          ──────────────
 1. apx release prepare
    (validate, write manifest)

 2. apx release submit
    ├─ shallow-clone canonical
    ├─ checkout -b apx/release/<api>/<ver>
    ├─ copy module files
    ├─ generate go.mod (if missing)
    ├─ git commit + push
    └─ gh pr create ───────────►  PR opened
                                   │
                                   ▼
                                  3. Canonical CI validates
                                     (lint, breaking, policy)
                                   │
                                   ▼
                                  4. Reviewer merges PR
                                   │
                                   ▼
                                  5. apx release finalize
                                     (tag, catalog, release record)
```

## Branch Naming

Feature branches follow the pattern:

```
apx/release/<api-id-dashes>/<version>
```

For example, `apx/release/proto-payments-ledger-v1/v1.0.0-beta.1`.

## PR Metadata

| Field | Value |
|-------|-------|
| **Title** | `release: <api-id>@<version>` |
| **Body** | `Automated release of API \`<api-id>\` at version \`<version>\`.` |
| **Base** | `main` |
| **Head** | the feature branch above |

## What Gets Committed

The PR contains the module's source files copied from the app repo into the
canonical path.  For a proto API `proto/payments/ledger/v1`, the PR diff shows
changes under `proto/payments/ledger/`:

```
proto/payments/ledger/
├── go.mod          ← generated if missing
└── v1/
    └── ledger.proto
```

## Example

```bash
# From your app repo
apx release prepare proto/payments/ledger/v1 \
  --version v1.0.0-beta.1 \
  --lifecycle beta

apx release submit

# Output:
#   Submitting release proto/payments/ledger/v1 @ v1.0.0-beta.1
#   ✓ Release submitted successfully
#   PR: https://github.com/acme/apis/pull/42
```

## Troubleshooting

### "gh CLI not found"

Install the GitHub CLI: `brew install gh` or see [cli.github.com](https://cli.github.com).

### "gh is not authenticated"

Run `gh auth login` and follow the prompts.

### "permission denied" on push

You need write access to the canonical repo.  Ask an org admin to grant
you the **Write** or **Maintainer** role on the repo.

### "no changes to release"

The canonical repo already contains identical content for this module.
Verify you have new changes to release.
