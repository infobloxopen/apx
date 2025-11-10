# Makefile for Sphinx documentation

# You can set these variables from the command line, and also
# from the environment for the first two.
SPHINXOPTS    ?= --keep-going
SPHINXBUILD  ?= sphinx-build
SOURCEDIR    = .
BUILDDIR     = _build
VENV         = venv
PYTHON       = python3

# Put it first so that "make" without argument is like "make help".
help:
	@echo "APX Documentation Build Commands:"
	@echo ""
	@echo "Setup (local development):"
	@echo "  setup       Create virtual environment and install dependencies"
	@echo "  install     Install dependencies (CI or existing venv)"
	@echo ""
	@echo "Building:"
	@echo "  html        Build HTML documentation"
	@echo "  livehtml    Build and serve with live reload (kills port 8000 first)"
	@echo "  clean       Clean build directory"
	@echo ""
	@echo "Quality:"
	@echo "  linkcheck   Check for broken links"
	@echo "  spelling    Spell check documentation"
	@echo ""
	@echo "Note: Use 'make setup' for first-time local development"

.PHONY: help Makefile setup venv-check

# Create virtual environment and install dependencies (local development)
setup: $(VENV)/bin/activate
	@echo "✅ Virtual environment ready. Activate with: source $(VENV)/bin/activate"

$(VENV)/bin/activate:
	@echo "🔧 Creating virtual environment..."
	$(PYTHON) -m venv $(VENV)
	$(VENV)/bin/pip install --upgrade pip
	$(VENV)/bin/pip install -r requirements.txt

# Check if we're in a virtual environment or CI
venv-check:
	@if [ -z "$$VIRTUAL_ENV" ] && [ -z "$$CI" ] && [ ! -f "$(VENV)/bin/activate" ]; then \
		echo "❌ No virtual environment detected."; \
		echo ""; \
		echo "For local development, run:"; \
		echo "  make setup"; \
		echo "  source $(VENV)/bin/activate"; \
		echo ""; \
		echo "Or activate an existing virtual environment."; \
		exit 1; \
	fi

# Install Python dependencies (CI or if venv already active)
install: venv-check
	pip install -r requirements.txt

# Build documentation
html: venv-check
	@$(SPHINXBUILD) -M html "$(SOURCEDIR)" "$(BUILDDIR)" $(SPHINXOPTS) $(O)

# Clean build directory
clean:
	@$(SPHINXBUILD) -M clean "$(SOURCEDIR)" "$(BUILDDIR)" $(SPHINXOPTS) $(O)

# Live rebuild with auto-refresh
livehtml: venv-check
	@echo "🔍 Checking for existing processes on port 8000..."
	@-lsof -ti:8000 | xargs kill -9 2>/dev/null || true
	@echo "🚀 Starting live documentation server on http://localhost:8000"
	$(VENV)/bin/sphinx-autobuild "$(SOURCEDIR)" "$(BUILDDIR)/html" --host 0.0.0.0 --port 8000 $(SPHINXOPTS) $(O)

# Check for broken links
linkcheck: venv-check
	@$(SPHINXBUILD) -M linkcheck "$(SOURCEDIR)" "$(BUILDDIR)" $(SPHINXOPTS) $(O)

# Spell check (requires sphinxcontrib-spelling)
spelling: venv-check
	@$(SPHINXBUILD) -M spelling "$(SOURCEDIR)" "$(BUILDDIR)" $(SPHINXOPTS) $(O)

# Build all formats
all: clean html

# Clean everything including virtual environment
clean-all: clean
	rm -rf $(VENV)

# Catch-all target: route all unknown targets to Sphinx using the new
# "make mode" option.  $(O) is meant as a shortcut for $(SPHINXOPTS).
%: Makefile
	@$(SPHINXBUILD) -M $@ "$(SOURCEDIR)" "$(BUILDDIR)" $(SPHINXOPTS) $(O)