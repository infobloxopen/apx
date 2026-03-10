# Catalog Site

!!! warning "In Development"
    The catalog site is under active development. Commands, flags, and output may change without notice.

The catalog site is a static API explorer generated from `catalog.yaml`. It lets teams browse all APIs, filter by format and lifecycle, view language-specific import coordinates, and inspect the actual schema structure (services, endpoints, messages, fields) — all in a self-contained website deployable to GitHub Pages.

## Generating the Site

Run in your canonical API repository after generating the catalog:

```bash
apx catalog generate                         # build catalog.yaml from git tags
apx catalog site generate                    # generate the static site
```

The site is written to `_site/` by default:

```
_site/
  index.html           # single-page app shell
  assets/
    app.js             # frontend logic
    style.css          # styles
  data/
    index.json         # all API metadata + language coordinates
```

### Options

| Flag | Default | Description |
|------|---------|-------------|
| `--output, -o` | `_site` | Output directory |
| `--catalog, -c` | (auto) | Path or URL to `catalog.yaml` (same resolution as `apx search`) |
| `--base-path` | `` | URL base path (e.g., `/catalog` if deployed at `example.com/catalog/`) |
| `--dir` | `` | Path to repo root for schema extraction (see [Schema Content](#schema-content)) |

## Local Preview

To generate the site and preview it locally in one step:

```bash
apx catalog site serve
```

This generates the site to a temporary directory, starts a local HTTP server on port **10451**, and opens your browser. Press `Ctrl+C` to stop.

| Flag | Default | Description |
|------|---------|-------------|
| `--port, -p` | `10451` | Port to serve on |
| `--catalog` | (auto) | Path or URL to `catalog.yaml` |
| `--dir` | `` | Path to repo root for schema extraction |
| `--no-open` | `false` | Skip opening the browser automatically |

## Features

The generated site includes:

- **Search** — free-text search across API IDs, descriptions, domains, and tags
- **Filters** — filter by schema format, lifecycle state, domain, and origin (first-party/external/forked)
- **API detail** — click any API to see version history, lifecycle compatibility, and language coordinates
- **Schema content** — when `--dir` is set, shows the actual structure: proto services/RPCs/messages, OpenAPI endpoints, Avro records, JSON Schema properties, and Parquet columns
- **Language coordinates** — tabbed view of Go, Python, Java, TypeScript, Rust, and C++ import paths
- **Deep linking** — hash-based URLs (e.g., `#proto/payments/ledger/v1`) for sharing
- **Dark mode** — automatic light/dark theme based on system preference

## Schema Content

By default the site shows only metadata (name, version, lifecycle, coordinates). To include the actual schema structure — services, endpoints, messages, fields — pass `--dir` pointing to your repository root:

```bash
apx catalog site generate --dir=.              # from the repo root
apx catalog site serve --dir=.                 # local preview with schemas
```

The `--dir` flag tells APX where to find schema files on disk. For each API in the catalog, it reads the files at `Module.Path` relative to `--dir` and extracts structural information using built-in parsers:

| Format | Files scanned | What's extracted |
|--------|--------------|-----------------|
| proto | `*.proto` | Services, RPCs, messages, fields, enums, comments |
| openapi | `*.yaml`, `*.yml`, `*.json` | Paths, HTTP operations, parameters, component schemas |
| avro | `*.avsc`, `*.json` | Records, fields with types, enums, documentation |
| jsonschema | `*.json` | Properties, types, required markers, nested objects |
| parquet | `*.parquet` | Message name, columns with physical/logical types |

When `--dir` is not set (the default), schema extraction is skipped entirely and the site works exactly as before — metadata only.

!!! note "Pure-Go parsers"
    Schema extraction uses built-in parsers with no external dependencies. It does not invoke `buf`, `protoc`, `spectral`, or any other tool. The parsers extract structural information from the raw source files.

## Custom Domain

By default the catalog site is available at `{org}.github.io/{repo}`. To host it on a custom domain, set `site_url` in `apx.yaml`:

```yaml
version: 1
org: Infoblox-CTO
repo: apis
site_url: apis.internal.infoblox.dev
```

When `site_url` is set and `--setup-github` is used during `apx init canonical`, APX will:

1. **Enable GitHub Pages** with Actions-based deployment
2. **Set visibility** to private if the repository is private
3. **Configure the custom domain** on GitHub Pages
4. **Probe DNS** to verify a CNAME record points to `{org}.github.io`

If the CNAME is missing or incorrect, APX prints a warning but continues — you can configure DNS later.

### DNS Setup

Create a CNAME record pointing your custom domain to GitHub Pages:

| Type | Name | Value |
|------|------|-------|
| CNAME | `apis.internal.infoblox.dev` | `infoblox-cto.github.io` |

!!! note "No apex domains"
    GitHub Pages custom domains require a CNAME record. Apex domains (e.g. `infoblox.dev`) cannot use CNAME records — use a subdomain instead.

When `site_url` is empty or omitted, APX defaults to `{org}.github.io/{repo}` and skips custom domain configuration.

## GitHub Pages Deployment

Add a workflow to your canonical repository:

```yaml
# .github/workflows/catalog-site.yml
name: Deploy API Catalog

on:
  push:
    branches: [main]
    paths:
      - 'catalog/**'
      - 'proto/**'
      - 'openapi/**'
      - 'avro/**'
      - 'jsonschema/**'
      - 'parquet/**'

permissions:
  contents: read
  pages: write
  id-token: write

concurrency:
  group: pages
  cancel-in-progress: false

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0  # needed for git tags

      - name: Install APX
        run: |
          curl -sSL https://github.com/infobloxopen/apx/releases/latest/download/apx_linux_amd64.tar.gz | tar xz
          sudo mv apx /usr/local/bin/

      - name: Generate catalog
        run: apx catalog generate

      - name: Generate site
        run: apx catalog site generate --output=_site --dir=.

      - uses: actions/configure-pages@v4
      - uses: actions/upload-pages-artifact@v3
        with:
          path: _site

  deploy:
    needs: build
    runs-on: ubuntu-latest
    environment:
      name: github-pages
      url: ${{ steps.deployment.outputs.page_url }}
    steps:
      - name: Deploy to GitHub Pages
        id: deployment
        uses: actions/deploy-pages@v4
```

!!! tip "No extra dependencies"
    The APX binary carries the entire site template embedded — no Node.js, Python, or additional tools needed in CI.

## How It Works

The `apx catalog site generate` command:

1. Loads `catalog.yaml` using the same resolution as `apx search` (explicit flag, config registries, `catalog_url`, local file)
2. For each API module, derives language-specific coordinates using the same plugin system as `apx show` and `apx inspect identity`
3. Enriches lifecycle information with compatibility promises and production recommendations
4. When `--dir` is set, reads schema files at each module's path and extracts structural content
5. Writes a `data/index.json` file containing all API metadata (and schema content when available)
6. Copies the embedded HTML/CSS/JS shell to the output directory

The frontend is a vanilla JavaScript single-page application that loads `index.json` and provides client-side search and filtering — no server required.

## See Also

- [Catalog Schema](../dependencies/catalog-schema.md) — structure of `catalog.yaml`
- [CI Templates](ci-templates.md) — other CI workflows for canonical repos
