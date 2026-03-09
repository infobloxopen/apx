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
- Go 1.26.1 + cobra (CLI), yaml.v3 (config), golang.org/x/mod (semver), testify (testing), go-internal (testscript), charmbracelet/huh (interactive TUI) (008-external-api-registration)
- YAML files (apx.yaml, catalog.yaml, apx.lock) (008-external-api-registration)

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
- 008-external-api-registration: Added Go 1.26.1 + cobra (CLI), yaml.v3 (config), golang.org/x/mod (semver), testify (testing), go-internal (testscript), charmbracelet/huh (interactive TUI)
- 006-canonical-config-schema: Added Go 1.26.1 + cobra (CLI), gopkg.in/yaml.v3 (YAML parsing), testify (assertions), go-internal/testscript (integration tests)
- 005-docs-cli-consistency: Added Go 1.26.1 + cobra v1.10.2, charmbracelet/huh v0.8.0, gopkg.in/yaml.v3


<!-- MANUAL ADDITIONS START -->

## Language Plugin Architecture
- Multi-language support uses a plugin system in `internal/language/`
- Each language (Go, Python, Java, TypeScript) is a registered plugin implementing `LanguagePlugin` interface
- Plugins self-register via `init()` — no central wiring needed
- Optional interfaces: `Scaffolder`, `PostGenHook`, `Linker`, `DocContributor`
- Plugins co-locate documentation fragments in `<lang>_doc/` directories
- To add a new language, follow the step-by-step guide: `internal/language/CONTRIBUTING.md`
- Generated doc includes live in `docs/_generated/` (never edit manually)
- Build: `GOTOOLCHAIN=go1.26.1 go generate ./internal/language/...` regenerates doc includes
- Manifest/record schema uses `Languages map[string]config.LanguageCoords` instead of flat per-language fields
- Core derivation functions live in `internal/config/identity.go`; plugins wrap them
- `language.DeriveAllCoords()` replaces the old `config.DeriveLanguageCoordsWithRoot()`
- `language.FormatIdentityReport()` replaces the old `config.FormatIdentityReport()`

<!-- MANUAL ADDITIONS END -->
