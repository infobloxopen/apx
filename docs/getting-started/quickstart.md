# Quick Start

Get up and running with APX in under 5 minutes. This guide sets up the two repos you need and validates that everything works.

!!! tip "Prerequisites"
    [Install APX](installation.md) and have Git and a GitHub organization ready.

## What You're Building

APX uses two types of repos:

1. **Canonical Repo** (`github.com/<org>/apis`) — single source of truth for all organization APIs
2. **App Repos** — where teams author schemas, then release to the canonical repo via PRs

## 1. Create the Canonical Repo

Create and clone `github.com/<org>/apis`, then initialize:

```bash
git clone https://github.com/<org>/apis.git
cd apis

apx init canonical --org=<org> --repo=apis
```

This scaffolds the canonical structure:

```
apis/
├── buf.yaml            # org-wide lint/breaking policy
├── buf.work.yaml       # workspace config
├── CODEOWNERS          # per-path ownership
├── catalog/
│  ├── .gitignore
│  └── Dockerfile
└── proto/              # (+ openapi, avro, etc. as needed)
```

Commit and push, then protect the `main` branch (require PR reviews) and tag patterns (`proto/**/v*` — only CI creates tags).

## 2. Initialize an App Repo

In your service repository, initialize an API module:

```bash
cd /path/to/your-service
apx init app --org=<org> --repo=<service> internal/apis/proto/payments/ledger
```

This creates:

```
<service>/
├── apx.yaml            # API identity and coordinates
├── apx.lock            # pinned toolchain versions
├── buf.work.yaml       # Buf workspace config
└── internal/
   └── apis/
      └── proto/
         └── payments/
            └── ledger/
               └── v1/
                  └── ledger.proto
```

## 3. Validate Your Setup

Fetch the pinned toolchain and run lint to confirm everything is wired correctly:

```bash
apx fetch
apx lint
```

If `apx lint` passes with no errors, you're ready to start authoring schemas.

## What's Next?

<div class="grid cards" markdown>

-   :material-book-open-variant: **Tutorial**

    ---

    Walk through the complete APX workflow: authoring, local development with canonical imports, releasing, and consuming APIs.

    [:octicons-arrow-right-24: Full tutorial](tutorial.md)

-   :material-cog: **Initialization Guide**

    ---

    Smart defaults, interactive prompts, CLI flags, and team onboarding scripts for `apx init`.

    [:octicons-arrow-right-24: Initialization details](initialization.md)

-   :material-console: **CLI Reference**

    ---

    Complete reference for all APX commands, flags, and configuration.

    [:octicons-arrow-right-24: CLI docs](../cli-reference/index.md)

</div>

---

**Questions?** Check the [Troubleshooting FAQ](../troubleshooting/faq.md) or open a [discussion](https://github.com/infobloxopen/apx/discussions).
