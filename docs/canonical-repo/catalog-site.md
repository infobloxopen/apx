# Catalog Site

The catalog site is a static API explorer generated from `catalog.yaml`. It lets teams browse all APIs, filter by format and lifecycle, and view language-specific import coordinates — all in a self-contained website deployable to GitHub Pages.

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
| `--no-open` | `false` | Skip opening the browser automatically |

## Features

The generated site includes:

- **Search** — free-text search across API IDs, descriptions, domains, and tags
- **Filters** — filter by schema format, lifecycle state, domain, and origin (first-party/external/forked)
- **API detail** — click any API to see version history, lifecycle compatibility, and language coordinates
- **Language coordinates** — tabbed view of Go, Python, Java, TypeScript, Rust, and C++ import paths
- **Deep linking** — hash-based URLs (e.g., `#proto/payments/ledger/v1`) for sharing
- **Dark mode** — automatic light/dark theme based on system preference

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
        run: apx catalog site generate --output=_site

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
4. Writes a `data/index.json` file containing all API metadata
5. Copies the embedded HTML/CSS/JS shell to the output directory

The frontend is a vanilla JavaScript single-page application that loads `index.json` and provides client-side search and filtering — no server required.

## See Also

- [Catalog Schema](../dependencies/catalog-schema.md) — structure of `catalog.yaml`
- [CI Templates](ci-templates.md) — other CI workflows for canonical repos
