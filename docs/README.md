# APX Documentation

This directory contains the APX documentation site, built with [MkDocs Material](https://squidfunk.github.io/mkdocs-material/).

## Local Development

### Setup

```bash
cd docs
make setup            # creates venv and installs dependencies
source venv/bin/activate
```

### Build and Serve

```bash
make html             # build static site to ../site/
make serve            # live-reload server at http://localhost:8000
make clean            # remove build output
```

### Generated Doc Includes

Some documentation fragments are generated from language plugin metadata by `cmd/docgen`. These live in `_generated/` and are included via `--8<--` snippets. To regenerate:

```bash
make gen              # runs go generate ./internal/language/...
```

If Go is not installed locally, the build uses existing `_generated/` files.

## Writing Documentation

### Adding Pages

1. Create a `.md` file in the appropriate section directory
2. Add it to the `nav:` section in `mkdocs.yml`
3. Use kebab-case filenames

### Syntax Reference

MkDocs Material uses standard Markdown with pymdownx extensions:

- **Admonitions**: `!!! note "Title"` with 4-space indented content
- **Tabbed blocks**: `=== "Tab"` for language-specific examples
- **Code blocks**: triple backticks with language identifier, copy button included
- **Grid cards**: `<div class="grid cards" markdown>` with `-   **Title**` items
- **Snippet includes**: `--8<-- "_generated/file.md"` to include generated fragments

### Deployment

Documentation deploys automatically via GitHub Actions when changes to `docs/` are pushed to `main`. PRs trigger a build check without deploying.

## Resources

- [MkDocs Material](https://squidfunk.github.io/mkdocs-material/)
- [PyMdown Extensions](https://facelessuser.github.io/pymdown-extensions/)
