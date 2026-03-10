# Makefile for APX documentation (MkDocs Material)

VENV    = venv
PYTHON  = python3
BUILDDIR = ../site

# Default: show help
help:
	@echo "APX Documentation Build Commands:"
	@echo ""
	@echo "Setup (local development):"
	@echo "  setup       Create virtual environment and install dependencies"
	@echo "  install     Install dependencies (CI or existing venv)"
	@echo ""
	@echo "Building:"
	@echo "  html        Build HTML documentation"
	@echo "  serve       Serve with live reload (port 8000)"
	@echo "  clean       Clean build directory"
	@echo ""

.PHONY: help setup install gen html serve clean clean-all

# Create virtual environment and install dependencies (local development)
setup: $(VENV)/bin/activate
	@echo "Virtual environment ready. Activate with: source $(VENV)/bin/activate"

$(VENV)/bin/activate:
	@echo "Creating virtual environment..."
	$(PYTHON) -m venv $(VENV)
	$(VENV)/bin/pip install --upgrade pip
	$(VENV)/bin/pip install -r requirements.txt

# Check if we're in a virtual environment or CI
venv-check:
	@if [ -z "$$VIRTUAL_ENV" ] && [ -z "$$CI" ] && [ ! -f "$(VENV)/bin/activate" ]; then \
		echo "No virtual environment detected."; \
		echo ""; \
		echo "For local development, run:"; \
		echo "  make setup"; \
		echo "  source $(VENV)/bin/activate"; \
		exit 1; \
	fi

# Install Python dependencies (CI or if venv already active)
install: venv-check
	pip install -r requirements.txt

# Generate language plugin doc includes (requires Go toolchain).
# Skips gracefully if Go is not available (e.g. editing prose docs locally).
gen:
	@if command -v go >/dev/null 2>&1; then \
		echo "Generating language plugin doc includes..."; \
		cd .. && GOTOOLCHAIN=go1.26.1 go generate ./internal/language/...; \
	elif [ ! -d "_generated" ]; then \
		echo "Go toolchain not found and _generated/ is missing."; \
		echo "Install Go or run 'go generate ./internal/language/...' from the repo root."; \
		exit 1; \
	else \
		echo "Skipping doc generation (Go not available, using existing _generated/)."; \
	fi

# Build documentation (regenerates plugin includes first)
html: venv-check gen
	mkdocs build -f mkdocs.yml

# Serve with live reload
serve: venv-check gen
	@echo "Starting live documentation server on http://localhost:8000"
	mkdocs serve -f mkdocs.yml -a localhost:8000

# Clean build directory
clean:
	rm -rf $(BUILDDIR)

# Clean everything including virtual environment
clean-all: clean
	rm -rf $(VENV)
