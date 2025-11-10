# Configuration file for the Sphinx documentation builder.
#
# For the full list of built-in configuration values, see the documentation:
# https://www.sphinx-doc.org/en/master/usage/configuration.html

# -- Project information -----------------------------------------------------
# https://www.sphinx-doc.org/en/master/usage/configuration.html#project-information

project = 'APX'
copyright = '2025, Infoblox'
author = 'Infoblox'
release = '1.0.0'

# -- General configuration ---------------------------------------------------
# https://www.sphinx-doc.org/en/master/usage/configuration.html#general-configuration

extensions = [
    'myst_parser',
    'sphinx.ext.autodoc',
    'sphinx.ext.viewcode',
    'sphinx.ext.napoleon',
    'sphinx.ext.intersphinx',
    'sphinx_copybutton',
    'sphinx_design',
    'sphinx_togglebutton',
]

templates_path = ['_templates']
exclude_patterns = ['_build', 'Thumbs.db', '.DS_Store', 'venv', 'README.md', 'target.md']

# -- MyST Configuration ------------------------------------------------------
myst_enable_extensions = [
    "amsmath",
    "colon_fence",
    "deflist",
    "dollarmath",
    "fieldlist",
    "html_admonition",
    "html_image",
    "linkify",
    "replacements",
    "smartquotes",
    "strikethrough",
    "substitution",
    "tasklist",
]

myst_heading_anchors = 3

# -- Options for HTML output -------------------------------------------------
# https://www.sphinx-doc.org/en/master/usage/configuration.html#options-for-html-output

html_theme = 'sphinx_book_theme'
html_title = 'APX Documentation'

html_theme_options = {
    "repository_url": "https://github.com/infobloxopen/apx",
    "use_repository_button": True,
    "use_edit_page_button": True,
    "use_issues_button": True,
    "use_download_button": True,
    "path_to_docs": "docs/",
    "show_navbar_depth": 2,
    "show_toc_level": 2,
    "announcement": "🚀 APX - API Publishing eXperience CLI is now available!",
}

html_static_path = ['_static']
html_css_files = ['custom.css']

# -- Intersphinx mapping ----------------------------------------------------
intersphinx_mapping = {
    'python': ('https://docs.python.org/3', None),
}

# -- Copy button configuration -----------------------------------------------
copybutton_prompt_text = r">>> |\.\.\. |\$ |In \[\d*\]: | {2,5}\.\.\.: | {5,8}: "
copybutton_prompt_is_regexp = True
copybutton_remove_prompts = True

# -- Favicon -----------------------------------------------------------------
html_favicon = '_static/favicon.ico'

# -- Logo --------------------------------------------------------------------
html_logo = '_static/logo.png'

# -- Social cards ------------------------------------------------------------
html_meta = {
    "description": "APX - API Publishing eXperience CLI for managing schema evolution",
    "property=og:description": "APX - API Publishing eXperience CLI for managing schema evolution",
    "property=og:title": "APX Documentation",
    "property=og:type": "website",
    "property=og:url": "https://infobloxopen.github.io/apx/",
    "property=og:image": "https://infobloxopen.github.io/apx/_static/social-card.png",
    "name=twitter:card": "summary_large_image",
    "name=twitter:title": "APX Documentation",
    "name=twitter:description": "APX - API Publishing eXperience CLI for managing schema evolution",
    "name=twitter:image": "https://infobloxopen.github.io/apx/_static/social-card.png",
}