# apx Development Guidelines

Auto-generated from all feature plans. Last updated: 2025-11-21

## Active Technologies
- Go 1.24 (matches APX project) (003-e2e-integration-suite)
- Go 1.26.1 (CLI), Bash (install script), YAML (CI/GoReleaser config) + GoReleaser v2 (release automation), GitHub Actions (CI/CD), goreleaser-action@v6 (004-package-installers)
- N/A — no runtime data; config files only (004-package-installers)
- Go 1.26.1 + cobra v1.10.2, charmbracelet/huh v0.8.0, gopkg.in/yaml.v3 (005-docs-cli-consistency)
- Files (`apx.yaml`, `apx-lock.yaml`, `catalog/catalog.yaml`) (005-docs-cli-consistency)
- Go 1.26.1 + cobra (CLI), gopkg.in/yaml.v3 (YAML parsing), testify (assertions), go-internal/testscript (integration tests) (006-canonical-config-schema)
- Filesystem (`apx.yaml`, `apx.lock`) (006-canonical-config-schema)

- Go 1.24 + `github.com/urfave/cli/v2` (command wiring), `github.com/AlecAivazis/survey/v2` (interactive prompts), `github.com/infobloxopen/apx/internal/*` packages for business logic, external CLI toolchains (buf, spectral, oasdiff, etc.) (001-align-docs-experience)

## Project Structure

```text
src/
tests/
```

## Commands

# Add commands for Go 1.24

## Code Style

Go 1.24: Follow standard conventions

## Recent Changes
- 006-canonical-config-schema: Added Go 1.26.1 + cobra (CLI), gopkg.in/yaml.v3 (YAML parsing), testify (assertions), go-internal/testscript (integration tests)
- 005-docs-cli-consistency: Added Go 1.26.1 + cobra v1.10.2, charmbracelet/huh v0.8.0, gopkg.in/yaml.v3
- 004-package-installers: Added Go 1.26.1 (CLI), Bash (install script), YAML (CI/GoReleaser config) + GoReleaser v2 (release automation), GitHub Actions (CI/CD), goreleaser-action@v6


<!-- MANUAL ADDITIONS START -->
<!-- MANUAL ADDITIONS END -->
