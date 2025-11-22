# APX Constitution

## Core Principles

### I. Documentation-Driven Development (NON-NEGOTIABLE)
The `/docs/` directory defines the **target state** for user experience and product direction. All implementation MUST align with documented workflows, commands, and behavior.

**Rules:**
- Documentation is written FIRST before implementation
- Changes to user-facing behavior require documentation updates BEFORE code changes
- CLI commands, flags, and outputs MUST match documented examples exactly
- The canonical repo pattern and import path strategy defined in docs are immutable design constraints
- Implementation validates against docs, not the other way around

**Rationale:** Users depend on documented behavior. Documentation-first ensures intentional design, prevents feature drift, and maintains consistency across the product lifecycle.

### II. Cross-Platform Path Operations (NON-NEGOTIABLE)
All path operations MUST work identically on Unix (Linux/macOS) and Windows systems.

**Rules:**
- **Use `filepath` package**: All file system operations MUST use `filepath.Join()`, `filepath.Rel()`, `filepath.Abs()`, etc.
- **Normalize for git/config**: Use `filepath.ToSlash()` before any string-based path manipulation (Split/Join/Contains/HasPrefix/TrimPrefix)
- **Git operations**: Always use forward slashes for git paths, branch names, and configuration files (git expects forward slashes even on Windows)
- **Module paths**: Normalize paths to forward slashes for module names, import paths, and YAML/JSON configuration
- **Binary names**: Add `.exe` extension on Windows for executable files (`runtime.GOOS == "windows"`)
- **Never hardcode separators**: No string literals with `/` or `\` for path construction or manipulation
- **Test on Windows CI**: All path-related code MUST pass tests on Windows GitHub Actions runners

**Forbidden patterns:**
```go
// ❌ WRONG - hardcoded separator
strings.Split(path, "/")
strings.TrimPrefix(absPath, repoPath + "/")
branchName := "publish/" + moduleDir  // fails on Windows

// ✅ CORRECT - normalize then manipulate
path = filepath.ToSlash(path)
strings.Split(path, "/")

relPath, _ := filepath.Rel(repoPath, absPath)
relPath = filepath.ToSlash(relPath)  // for git operations

normalizedPath := filepath.ToSlash(moduleDir)
branchName := "publish/" + normalizedPath
```

**Rationale:** APX must work on developer machines across all platforms. Windows uses backslashes for file paths but git operations require forward slashes. Inconsistent path handling causes failures in schema detection, git operations, and configuration generation.

### III. Test-First Development (NON-NEGOTIABLE)
Every feature follows strict TDD: Tests written → User/reviewer approval → Tests fail (Red) → Implement (Green) → Refactor.

**Rules:**
- **Unit tests** MUST exist for every internal package (`internal/*`)
- **Integration tests** MUST use testscript for CLI command validation
- **GitHub integration tests** MUST use Gitea to simulate real GitHub workflows (PRs, tags, releases)
- Tests are written and reviewed BEFORE implementation begins
- Code without tests is not merged
- Minimum coverage: 80% for business logic packages

**Rationale:** APX orchestrates critical workflows (schema publishing, versioning, breaking change detection). Testing first ensures correctness, prevents regressions, and documents expected behavior.

### III. Code Quality & Maintainability
Code must be clean, well-organized, and maintainable following Go best practices.

**Rules:**
- **Separation of concerns**: CLI logic in `cmd/apx/commands/`, business logic in `internal/` packages
- **Single Responsibility**: Each file/package has one clear purpose (max ~300 lines per command file)
- **No God objects**: Distribute functionality across focused packages
- **Error handling**: All errors must be properly wrapped with context using `fmt.Errorf("context: %w", err)`
- **Logging**: Use `internal/ui` for all user-facing output with consistent formatting
- **Code reviews**: Mandatory for all changes, focusing on testability and clarity

**Rationale:** APX's refactoring from 1,117-line main.go to modular structure demonstrates our commitment to maintainability. Clean code reduces bugs and accelerates feature development.

### IV. Developer Experience First
Every interaction with APX must feel intuitive, fast, and helpful.

**Rules:**
- **Interactive prompts**: Use `survey` for guided setup when no flags provided
- **Smart defaults**: Auto-detect project context (git org, languages, repo structure)
- **Clear output**: Human-readable by default, `--json` flag for automation
- **Helpful errors**: Error messages MUST include context and suggest fixes
- **Fast feedback**: Local validation completes in <5 seconds for typical schemas
- **Progressive disclosure**: Simple tasks stay simple, complex tasks stay possible

**Example:**
```bash
# Smart defaults - no flags needed for common case
apx init proto payment/ledger/v1

# Interactive mode when ambiguous
apx init  # Prompts for schema type, module path, org, languages

# JSON output for CI/automation
apx lint --json
```

**Rationale:** CLI tools succeed when they reduce friction. APX competes with manual git operations and ad-hoc scripts—superior UX is our competitive advantage.

### V. Canonical Import Paths (Architecture Constraint)
All generated Go code MUST use canonical import paths with `go.work` overlays for local development. This enables seamless transition from local stubs to published modules.

**Rules:**
- Generated code imports ONLY canonical paths: `github.com/<org>/apis-go/proto/<domain>/<api>/v1`
- Never use `replace` directives or relative paths in generated code
- `apx sync` manages `go.work` overlays for local development
- `/internal/gen/**` is git-ignored (never commit generated code)
- Module paths follow semantic import versioning (no `/v1` suffix for v1, `/v2+` for v2+)

**Rationale:** This pattern eliminates import path rewrites when switching from local to published dependencies, matching the user experience documented in `/docs/getting-started/quickstart.md`.

### VI. Git Subtree Publishing Strategy
APX MUST use **git subtree** (not copy/snapshot) for publishing to canonical repos, preserving commit history and authorship.

**Rules:**
- `apx publish` uses `git subtree split` to extract schema subdirectories
- Commit history, authors, and timestamps are preserved in canonical repo
- PRs to canonical repo show full git history for auditability
- Tag format: app repo uses `proto/<domain>/<api>/v1/v1.2.3`, canonical repo uses `proto/<domain>/<api>/v1.2.3`

**Rationale:** History preservation enables debugging API evolution, maintains authorship credit, and provides transparent audit trails. This aligns with documented publishing workflow in `/docs/publishing/`.

### VII. Multi-Format Schema Support with Consistent Tooling
Protocol Buffers are primary, but APX MUST support OpenAPI, Avro, JSON Schema, and Parquet with format-specific validation.

**Rules:**
- **Protocol Buffers**: buf integration for linting, breaking changes, and code generation
- **OpenAPI**: spectral linting, oasdiff breaking detection
- **Avro**: compatibility mode validation (BACKWARD default)
- **JSON Schema**: jsonschema-diff for breaking changes
- **Parquet**: conservative additive nullable column policy
- All formats share unified CLI: `apx lint`, `apx breaking`, `apx gen <lang>`

**Rationale:** Organizations use diverse schema formats. Unified tooling reduces cognitive load and ensures consistent governance across all API types.

## Testing Requirements

### Unit Testing Standards
Every `internal/` package MUST have comprehensive unit tests:

**Coverage targets:**
- `internal/schema/*`: 90%+ (core business logic)
- `internal/detector/*`: 85%+ (project detection)
- `internal/interactive/*`: 80%+ (user interaction flows)
- `internal/config/*`: 90%+ (configuration parsing)

**Testing patterns:**
- Table-driven tests for multiple input scenarios
- Mock external dependencies (git, file system when appropriate)
- Test error paths explicitly
- Use `testify/require` for assertions

### Integration Testing with Testscript
CLI commands MUST have testscript-based integration tests:

**Required coverage:**
- All CLI commands with common flag combinations
- Error handling and validation failures
- File system operations (init, gen, publish workflows)
- Configuration loading and validation

**Location:** `testdata/script/*.txt`

**Example:**
```testscript
# Test apx init creates correct structure
exec apx init proto payment/v1
exists internal/apis/proto/payment/v1
grep 'package.*payment.v1' internal/apis/proto/payment/v1/*.proto
```

### GitHub Integration Testing with Gitea
APX MUST validate GitHub workflows using Gitea as a test double:

**Required scenarios:**
- Create PR to canonical repo via `apx publish`
- Tag creation and validation in both app and canonical repos
- CODEOWNERS enforcement
- Protected branch/tag patterns
- Merge conflict detection and resolution

**Infrastructure:**
- Gitea container spun up per test suite
- Clean state for each test (isolated repositories)
- API token authentication
- Webhook simulation for CI triggers

**Rationale:** APX's core value is GitHub integration. Testing against real git operations prevents production failures and validates documented workflows.

## Architecture & Code Organization

### Package Structure (NON-NEGOTIABLE)
```
cmd/apx/
├── main.go              # Entry point only (~85 lines)
├── commands/            # CLI command definitions
│   ├── init.go         # ~150 lines max
│   ├── lint.go
│   ├── breaking.go
│   ├── publish.go
│   ├── gen.go
│   ├── config.go
│   ├── semver.go
│   ├── catalog.go
│   ├── policy.go
│   └── common.go       # Shared utilities
internal/
├── schema/             # Schema operations
│   └── init.go         # Schema initialization
├── detector/           # Project detection
│   └── project.go      # Git, language detection
├── interactive/        # User interaction
│   └── setup.go        # Survey prompts
├── config/             # Configuration
│   └── config.go       # apx.yaml parsing
├── ui/                 # User interface
│   └── ui.go           # Output formatting
└── publisher/          # Publishing logic (TODO)
    ├── subtree.go      # Git subtree operations
    └── pr.go           # GitHub PR creation
```

**Rules:**
- CLI logic stays in `cmd/apx/commands/`
- Business logic goes in `internal/` packages
- Each command file <200 lines
- Packages are independently testable
- No circular dependencies

### Dependency Management
- Use standard library where sufficient
- Third-party dependencies require justification
- Pin versions in `go.mod`
- External tools (buf, spectral) installed via `scripts/install-tools.sh`

## Development Workflow

### Before Starting Work
1. Read relevant `/docs/` pages for feature context
2. Write tests capturing expected behavior
3. Get test review/approval from team
4. Ensure tests fail (Red)
5. Implement minimal code to pass tests (Green)
6. Refactor while keeping tests green

### Code Review Requirements
- All changes require review from at least one maintainer
- Reviewers verify:
  - [ ] Tests exist and cover new code paths
  - [ ] Documentation updated if user-facing changes
  - [ ] Code follows architecture patterns
  - [ ] Error handling is comprehensive
  - [ ] No performance regressions

### Pull Request Checklist
- [ ] Tests pass locally (`make test`)
- [ ] Integration tests pass (`make test-integration`)
- [ ] Documentation updated in `/docs/`
- [ ] CHANGELOG.md updated (if applicable)
- [ ] No linter warnings (`make lint`)
- [ ] Code coverage maintained or improved

## Quality Gates

### Pre-Merge Requirements
- **All tests passing**: Unit, integration, and Gitea-based tests
- **Code review approved**: At least one maintainer approval
- **Documentation updated**: User-facing changes reflected in `/docs/`
- **No regressions**: Performance tests show no degradation
- **Linters clean**: `golangci-lint`, `buf lint`, format checks pass

### Release Criteria
- All quality gates passed
- Integration tests validated against real GitHub (staging)
- Documentation generated and published
- CHANGELOG.md updated with user-facing changes
- Backward compatibility verified (unless major version bump)

## Governance

This constitution supersedes all other development practices and guidelines. All code, tests, and documentation must comply with these principles.

**Amendment Process:**
1. Propose change with rationale and impact analysis
2. Team discussion and consensus
3. Update constitution with new version number
4. Migrate existing code if needed
5. Update dependent templates and documentation

**Compliance:**
- All PRs are reviewed against this constitution
- Violations block merge until resolved
- Complexity exceptions require written justification
- Use `.specify/templates/` for task planning aligned with principles

**Version Control:**
- Constitution changes increment version following semver:
  - MAJOR: Backward-incompatible principle changes (e.g., removing a core principle)
  - MINOR: New principles or expanded guidance (e.g., adding cross-platform path requirement)
  - PATCH: Clarifications, typo fixes, non-semantic refinements

**Version**: 1.1.0 | **Ratified**: 2025-11-21 | **Last Amended**: 2025-11-22

## Changelog

### v1.1.0 (2025-11-22)
- **ADDED**: Section II "Cross-Platform Path Operations" as non-negotiable requirement
- Ensures all path operations work identically on Unix and Windows systems
- Mandates use of `filepath` package and normalization with `filepath.ToSlash()` for git/config operations
- Requires Windows CI testing for all path-related code
- Renumbered subsequent sections (Test-First Development is now III, etc.)
