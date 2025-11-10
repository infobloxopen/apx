# APX Documentation Setup

This guide explains how to set up and maintain the APX documentation site using MyST/Sphinx with GitHub Pages.

## Setup GitHub Pages

To enable automatic documentation deployment:

1. **Enable GitHub Pages**:
   - Go to your repository Settings → Pages
   - Under "Source", select "GitHub Actions"
   - This allows the workflow to deploy directly

2. **Required Permissions**:
   The workflow is configured with the necessary permissions:
   - `contents: read` - to checkout code
   - `pages: write` - to deploy to GitHub Pages  
   - `id-token: write` - for authentication

## Documentation Structure

```
docs/
├── conf.py              # Sphinx configuration
├── index.md             # Main landing page
├── getting-started/     # User guides
│   ├── installation.md
│   ├── quickstart.md
│   └── interactive-init.md
├── requirements.txt     # Python dependencies
├── Makefile            # Build automation
└── _static/
    └── custom.css      # Custom styling
```

## Local Development

### Prerequisites

- Python 3.11+
- pip

### Setup

```bash
cd docs
pip install -r requirements.txt
```

### Build Documentation

```bash
# Build HTML documentation
make html

# Serve with live reload (recommended for development)
make livehtml

# Check for broken links
make linkcheck

# Clean build artifacts
make clean
```

The built documentation will be in `_build/html/`.

## Writing Documentation

### MyST Markdown

We use MyST (Markedly Structured Text) which extends Markdown with powerful features:

```markdown
# Standard Markdown
**Bold text** and *italic text*

# MyST Extensions
:::{note}
This is a note admonition.
:::

:::{grid} 2
:::{grid-item-card} Feature 1
Description here
:::
:::{grid-item-card} Feature 2  
Description here
:::
:::
```

### Adding New Pages

1. Create a new `.md` file in the appropriate directory
2. Add it to the `toctree` in the parent page or `index.md`
3. Use descriptive filenames (kebab-case)

Example `toctree`:
```markdown
```{toctree}
:maxdepth: 2
:hidden:

getting-started/index
api/index  
examples/index
```

### Code Examples

Include APX command examples:

````markdown
```bash
# Initialize a new APX project
apx init myorg/myservice

# Run interactive setup
apx init
```
````

## Deployment

### Automatic Deployment

The documentation is automatically built and deployed when:

- Changes are pushed to the `main` branch in the `docs/` directory
- A PR modifies documentation (build-only, no deployment)
- Manually triggered via GitHub Actions UI

### Manual Deployment

You can trigger a manual deployment:

1. Go to Actions → "Deploy Documentation"
2. Click "Run workflow" → "Run workflow"

## Customization

### Theme and Styling

- Theme: `sphinx-book-theme` (configured in `conf.py`)
- Custom CSS: `_static/custom.css`
- Logo: Add to `_static/` and reference in `conf.py`

### Sphinx Extensions

Current extensions in `conf.py`:
- `myst_parser` - MyST markdown support
- `sphinx.ext.autodoc` - API documentation
- `sphinx.ext.viewcode` - Source code links
- `sphinx_design` - UI components (cards, grids)
- `sphinx_copybutton` - Copy code buttons

### Configuration

Key settings in `conf.py`:
- `html_title` - Site title
- `html_theme_options` - Theme customization
- `myst_enable_extensions` - MyST features
- `html_static_path` - Static files location

## Troubleshooting

### Build Errors

1. **Import errors**: Check `requirements.txt` dependencies
2. **MyST syntax errors**: Validate MyST markdown syntax
3. **Broken links**: Run `make linkcheck` to identify issues

### GitHub Pages Issues

1. **Permission denied**: Ensure repository has Pages enabled with "GitHub Actions" source
2. **Build failures**: Check GitHub Actions logs for detailed error messages
3. **404 errors**: Verify file paths and internal links

### Performance

- Keep images optimized and reasonably sized
- Use `sphinx.ext.viewcode` sparingly for large codebases
- Consider excluding large directories in `conf.py` if needed

## Resources

- [MyST Parser Documentation](https://myst-parser.readthedocs.io/)
- [Sphinx Book Theme](https://sphinx-book-theme.readthedocs.io/) 
- [Sphinx Design Components](https://sphinx-design.readthedocs.io/)
- [GitHub Pages with Actions](https://docs.github.com/en/pages/getting-started-with-github-pages/configuring-a-publishing-source-for-your-github-pages-site#publishing-with-a-custom-github-actions-workflow)