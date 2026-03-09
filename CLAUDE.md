# APX — Claude Code Instructions

## Critical Rules

### specs/ directory is READ-ONLY
**NEVER edit any file under `specs/` under any circumstances.**

The `specs/` directory contains feature specifications that are instructions/source material only. They are not code targets. Do not modify, update, reformat, or touch them in any way unless the user explicitly references running a spec kit agent for a specific spec.

If you find specs/ files modified, run: `git checkout -- specs/`

### Commit Messages
**NEVER add any Claude/Anthropic attribution to commit messages.** No `Co-Authored-By: Claude`, no `Generated with Claude Code`, nothing referencing Claude or Anthropic. Commit messages should only describe the change.

### Build
```bash
GOTOOLCHAIN=go1.26.1 go build ./...
GOTOOLCHAIN=go1.26.1 go test ./...
GOTOOLCHAIN=go1.26.1 make dev   # rebuild binary to bin/apx
```

## Project Maturity (Pre-Stable)
APX has not yet made a stable release. Until v1.0.0:
- **Backward compatibility is NOT a concern.** Commands, flags, config formats, and internal APIs may change without deprecation.
- **Consistency over stability.** Inconsistencies are corrected immediately, even if they break prior behavior.
- **Docs and implementation must converge.** When they disagree, whichever is wrong gets fixed. There is no legacy behavior to preserve.

## Documentation-Driven Development (NON-NEGOTIABLE)
The `/docs/` directory defines the **target state**. All implementation MUST align with documented workflows.

- Documentation is written FIRST before implementation
- Changes to user-facing behavior require documentation updates BEFORE code changes
- CLI commands, flags, and outputs MUST match documented examples exactly
- Implementation validates against docs, not the other way around

## Cross-Platform Path Operations (NON-NEGOTIABLE)
All path operations MUST work identically on Unix and Windows.

- **Use `filepath` package** for all filesystem operations (`filepath.Join`, `filepath.Rel`, `filepath.Abs`)
- **Normalize for git/config**: Use `filepath.ToSlash()` before string-based path manipulation
- **Git operations**: Always use forward slashes for git paths, branch names, and config files
- **Never hardcode separators**: No string literals with `/` or `\` for path construction

```go
// ❌ WRONG
strings.Split(path, "/")
branchName := "release/" + moduleDir

// ✅ CORRECT
path = filepath.ToSlash(path)
strings.Split(path, "/")
normalizedPath := filepath.ToSlash(moduleDir)
branchName := "release/" + normalizedPath
```

## Test-First Development (NON-NEGOTIABLE)
- **Unit tests** MUST exist for every `internal/` package
- **Integration tests** MUST use testscript for CLI validation (`testdata/script/*.txt`)
- **GitHub integration tests** MUST use Gitea (`tests/e2e/`)
- Tests are written BEFORE implementation; code without tests is not merged
- Minimum coverage: 80% for business logic packages

## Code Organization
- CLI logic: `cmd/apx/commands/` (each command file <200 lines)
- Business logic: `internal/` packages (independently testable)
- User output: `internal/ui` package only
- Error handling: always wrap with `fmt.Errorf("context: %w", err)`

## Vocabulary
- `apx release` is the release pipeline (NOT `apx publish` — that was removed)
- Lifecycle values: `experimental`, `beta` (canonical), `stable`, `deprecated`, `sunset`
- `preview` is accepted as a backward-compatible alias for `beta` only
- API ID formats: 4-part `format/domain/name/line` (e.g. `proto/payments/ledger/v1`) or 3-part `format/name/line` (e.g. `proto/orders/v1`)
- `release prepare` = local-only (no network); `release submit` = creates PR on canonical repo

## Canonical Import Paths (Architecture Constraint)
Generated Go code MUST use canonical import paths:
- `github.com/<org>/apis/proto/<domain>/<api>/v1` (no `/v1` suffix for v0/v1, `/v2+` for v2+)
- Never use `replace` directives or relative paths in generated code
- `go.work` overlays (managed by `apx sync`) handle local development
