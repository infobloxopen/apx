# Brownfield Onboarding

This guide covers onboarding an **existing service** whose proto or OpenAPI files already live in a non-standard directory layout, alongside generated code, and whose team does not want to restructure the repository to match the APX greenfield convention.

For new services starting from scratch, see the [Quick Start](quickstart.md) and [Initialization Guide](initialization.md) instead.

## Automated onboarding with `apx onboard`

The quickest way to onboard is to run:

```bash
apx onboard <api-id>
```

For example:

```bash
apx onboard proto/infoblox.dev/appkit-canary/v1
```

APX prompts for any missing information (canonical repo, schema directory, lifecycle, audience) and generates all required files: `apx.yaml`, `.apx-publish.yaml`, `Makefile` `fds` target, both GitHub workflow callers, and `.gitignore` entries.

```text
apx onboard proto/infoblox.dev/appkit-canary/v1
  Canonical repo: github.com/Infoblox-CTO/apis
  Schema directory (for buf build): canary-scaffold/apis
  Lifecycle [beta]:
  Audience [public]:

✓ Generated apx.yaml
✓ Generated .apx-publish.yaml
✓ Patched Makefile (fds target added)
✓ Generated .github/workflows/api-catalog-pr.yml
✓ Generated .github/workflows/api-catalog-publish.yml
✓ Updated .gitignore
```

Use `--non-interactive` to skip prompts in CI or scripts:

```bash
apx onboard proto/infoblox.dev/appkit-canary/v1 \
  --canonical-repo github.com/Infoblox-CTO/apis \
  --schema-dir canary-scaffold/apis \
  --lifecycle beta \
  --audience public
```

The rest of this page describes each generated file in detail and covers manual setup for cases where the generated output needs customization.

## Overview

The brownfield path uses the **publish-on-change workflow** (`apx-publish.sh`) rather than `apx init app`. You declare what to publish in a single `.apx-publish.yaml` file; the workflow handles FDS generation, drift detection, breaking-change gating, and catalog PR submission automatically on every merge.

Three files, two workflow callers:

| File | Purpose |
|------|---------|
| `apx.yaml` | APX project config; must set `module_roots: [build/apx-modules]` |
| `.apx-publish.yaml` | Declares modules, FDS target, canonical repo, branch routing |
| `Makefile` | `make fds` target that compiles your protos into a `.binpb` |
| `.github/workflows/api-catalog-pr.yml` | PR check: drift + breaking report |
| `.github/workflows/api-catalog-publish.yml` | Publish on merge: opens catalog PR |

## Step 1 — `apx.yaml`

Place at the repository root. The `module_roots` entry must point to where the workflow stages converted specs:

```yaml
version: 1
org: <your-org>
repo: <your-repo>
module_roots:
  - build/apx-modules
release:
  tag_format: '{subdir}/v{version}'
  ci_only: true
tools:
  spectral:
    version: v6.11.0
  oasdiff:
    version: v1.9.6
execution:
  mode: local
```

## Step 2 — `.apx-publish.yaml`

```yaml
canonical_repo: github.com/<org>/apis
forge: github
fds_target: fds
version_bump: minor

branch_targets:
  main: main      # stable (GA) channel → apis main
  develop: develop  # pre-release (beta) channel → apis develop

modules:
  - id: proto/<domain>/<service>/v1   # format inferred from id prefix
    fds: api-v1.binpb
    lifecycle: stable
    audience: public       # public = breaking changes block; internal = advisory only
    verify_clients: off    # proto FDS is not an OpenAPI spec; oapi-codegen does not apply
```

!!! note "Format is inferred from the id prefix"
    A module with id `proto/...` uses the proto format; `openapi/...` uses the OpenAPI format.
    No explicit `format:` field is needed.

!!! warning "`verify_clients` for proto-format modules"
    Set `verify_clients: off` for all `proto`-format modules. The client verification gate
    currently passes the `.binpb` FileDescriptorSet to `oapi-codegen`, which fails because
    it expects an OpenAPI spec. See [issue #43](https://github.com/infobloxopen/apx/issues/43).

### OpenAPI modules (with swagger)

If your service produces a swagger file, use the `openapi` format instead:

```yaml
modules:
  - id: openapi/<domain>/<service>/v1
    swagger: path/to/service.swagger.json
    fds: api-v1.binpb
    lifecycle: stable
    audience: public
```

The workflow converts swagger + FDS through `openapiv2to3` to produce an enriched OpenAPI v3 spec.

### `fds_key_files` for buf-based repos

The FDS generation result is cached by hashing proto files and the gentool pin. The default
key covers `**/*.proto` and `**/Makefile*`. If your gentool pin lives in `buf.yaml` instead
of a Makefile, override this:

```yaml
fds_key_files:
  - '**/*.proto'
  - 'path/to/buf.yaml'
```

## Step 3 — `make fds`

The workflow calls `make <fds_target>` (default: `fds`) to emit the FileDescriptorSet.

### Using buf (buf.yaml-based repos)

```makefile
.PHONY: fds

# buf is provided by the apx-toolchain CI image (ghcr.io/infobloxopen/apx-toolchain:v1).
fds:
	buf build path/to/proto/dir --as-file-descriptor-set -o api-v1.binpb
```

!!! tip "buf.yaml v2 compatibility"
    In buf.yaml v2, you cannot specify `name:` at the top level when `modules:` is also
    present. If your buf.yaml has both, move `name:` onto the individual module entry or
    remove it:

    ```yaml
    # WRONG (buf v2 rejects this)
    version: v2
    modules:
      - path: proto
    name: buf.build/myorg/myrepo

    # CORRECT
    version: v2
    modules:
      - path: proto
    ```

### Using atlas-gentool (protoc-based repos)

```makefile
.PHONY: fds

DOCKER_GENERATOR := ghcr.io/myorg/atlas-gentool:v21.8.17@sha256:<digest>
GENERATOR        := docker run --rm -v $(PWD):/defs $(DOCKER_GENERATOR)

fds:
	$(GENERATOR) --include_imports \
		--descriptor_set_out=/defs/api-v1.binpb \
		/defs/path/to/service.proto
```

!!! important "Pin the gentool image by digest"
    Always pin atlas-gentool by `tag@sha256:…` (not a bare tag). The FDS cache key hashes
    the pin string — a re-pushed tag can silently serve a stale FDS on a cache hit.

## Step 4 — GitHub workflow callers

Two thin callers that reference the reusable `apx-publish.yml` workflow:

**`.github/workflows/api-catalog-pr.yml`**

```yaml
name: API catalog — PR check
on:
  pull_request:
    paths:
      - 'path/to/protos/**/*.proto'
      - '.apx-publish.yaml'
      - 'apx.yaml'
      - 'Makefile'
permissions:
  contents: read
  pull-requests: write   # required to post PR comments
jobs:
  check:
    uses: infobloxopen/apx-action/.github/workflows/apx-publish.yml@v1
    with:
      mode: check
      apx-action-ref: v1
      breaking-gate: auto
    secrets:
      apx-app-id:          ${{ secrets.APX_SUBMIT_APP_ID }}
      apx-app-private-key: ${{ secrets.APX_SUBMIT_APP_PRIVATE_KEY }}
```

**`.github/workflows/api-catalog-publish.yml`**

```yaml
name: API catalog — publish on change
on:
  push:
    branches: [main, develop]
    paths:
      - 'path/to/protos/**/*.proto'
      - '.apx-publish.yaml'
      - 'apx.yaml'
permissions:
  contents: read
  issues: write   # required to open drift issues
jobs:
  publish:
    uses: infobloxopen/apx-action/.github/workflows/apx-publish.yml@v1
    with:
      mode: publish
      apx-action-ref: v1
    secrets:
      apx-app-id:          ${{ secrets.APX_SUBMIT_APP_ID }}
      apx-app-private-key: ${{ secrets.APX_SUBMIT_APP_PRIVATE_KEY }}
```

!!! note "Org secrets"
    `APX_SUBMIT_APP_ID` and `APX_SUBMIT_APP_PRIVATE_KEY` must be configured as org-level
    secrets by an org admin. They are shared across all adopting repos — no per-repo secret
    management required.

## Step 5 — Update `.gitignore`

```text
# apx publish-on-change artifacts (generated, not committed)
api-v1.binpb
build/apx-modules/

# apx release manifest (ephemeral, written by apx release prepare)
.apx-release.yaml
```

## Testing locally

### Prerequisites

```bash
brew install bufbuild/buf/buf   # or your platform equivalent
brew install bash               # macOS only — apx-publish.sh requires bash 4+
```

`apx`, `yq` (mikefarah v4), and `jq` must also be on `PATH`.

### Run the check pipeline

```bash
# Clone the canonical catalog
git clone https://github.com/<org>/apis /tmp/apis-local

# Download both scripts (apx-publish.sh sources supersede.sh from the same dir)
curl -fsSL https://raw.githubusercontent.com/infobloxopen/apx-action/v1/scripts/apx-publish.sh \
  -o /tmp/apx-publish.sh
curl -fsSL https://raw.githubusercontent.com/infobloxopen/apx-action/v1/scripts/supersede.sh \
  -o /tmp/supersede.sh
chmod +x /tmp/apx-publish.sh

# Run check mode (no network writes)
bash /tmp/apx-publish.sh \
  --mode check \
  --canonical-dir /tmp/apis-local
```

!!! warning "macOS bash version"
    `apx-publish.sh` uses `mapfile`, which requires bash 4+. macOS ships bash 3.2.
    Always invoke the script with `bash /tmp/apx-publish.sh`, not as `./apx-publish.sh`.

### Run the publish pipeline locally

Publish mode opens a real PR on the canonical repo. Your token must have `contents:write`
and `pull_requests:write` on the canonical repo:

```bash
GITHUB_TOKEN=$(gh auth token) bash /tmp/apx-publish.sh \
  --mode publish \
  --canonical-dir /tmp/apis-local
```

Write access to the canonical repo is normally gated on the `APX_SUBMIT_APP` GitHub App
(configured by an org admin). If your personal token lacks write access, trigger publish
via CI instead: push to your service repo and let the workflow run with the App credentials.
