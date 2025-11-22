# Research Notes: Docs-Aligned APX Experience

## Decision Records

### Decision: Implement top-level CLI using `urfave/cli/v2` command structure mirroring `/docs` workflows
- **Rationale**: `urfave/cli/v2` already powers existing commands (see `cmd/apx/main.go`). Aligning command definitions and flag usage with the documented getting-started steps lets us wire behavior without replacing the CLI framework and ensures help text stays consistent with current patterns.
- **Alternatives considered**: Rewriting CLI orchestration with Cobra (rejected—breaks existing commands and diverges from docs); building a custom dispatcher (rejected—adds maintenance burden, offers no benefit over urfave/cli).

### Decision: Use `survey/v2` for interactive flows with deterministic prompts
- **Rationale**: Survey is already the interactive abstraction in `internal/interactive`. Reusing it allows us to deliver the documented guided setup while keeping prompt wording stable for doc parity tests.
- **Alternatives considered**: Custom prompt handling via fmt/scan (rejected—harder to test, inconsistent UX); promptui (rejected—would introduce new dependency and theming differences).

### Decision: Route validation and breaking checks through a new `internal/validator` facade per schema format
- **Rationale**: The spec requires format-specific tooling (buf, spectral, oasdiff, avro compat, jsonschema-diff, parquet rules). A dedicated package encapsulates command dispatch, tool discovery, and output normalization so CLI commands can remain thin and coverage can be targeted at format adapters.
- **Alternatives considered**: Embedding tool calls directly in each command (rejected—duplicates logic and complicates testing); invoking shell scripts (rejected—harder to mock and reason about in Go unit tests).

### Decision: Establish doc-parity tests via golden fixtures derived from `/docs/getting-started/quickstart.md`
- **Rationale**: Documentation-first constitution mandates CLI outputs match docs. Capturing canonical transcripts in `testdata/golden/` and diffing command output ensures changes that diverge from documentation fail during TDD.
- **Alternatives considered**: Manual review of docs each release (rejected—non-deterministic, high effort); embedding Markdown parsing in runtime (rejected—parsing overhead in CLI, unnecessary complexity).

### Decision: Use `testscript` suites for end-to-end flows covering canonical repo bootstrap, schema publish, and consumer overlay
- **Rationale**: `testscript` already underpins existing CLI tests. It excels at orchestrating filesystem operations, git commands, and CLI invocations exactly like the quickstart. Each user story maps to a standalone script ensuring independent validation.
- **Alternatives considered**: go test table-driven integration harness (rejected—harder to express multi-step shell flows); Bats shell tests (rejected—introduces bash dependency, inconsistent with Go tooling).

### Decision: Simulate GitHub interactions with Gitea in integration tests while retaining production compatibility with GitHub Enterprise
- **Rationale**: Constitution requires Gitea-backed integration validation. Gitea provides API compatibility for PRs/tags while remaining self-hostable, matching the "self hosted with GitHub" goal by mirroring enterprise workflows locally.
- **Alternatives considered**: Mocking GitHub APIs (rejected—insufficient coverage of git subtree behavior); running against live GitHub (rejected—unstable, requires network access, violates offline constraint).

### Decision: Treat external toolchains (buf, spectral, oasdiff, etc.) as pluggable executables resolved via `apx fetch`
- **Rationale**: `/docs/dependencies/index.md` prescribes script-driven installation. Encapsulating tool lookup within `internal/toolchain` (or extending existing helpers) allows offline bundles and version locking via `apx.lock`.
- **Alternatives considered**: Embedding libraries for each format (rejected—heavy vendoring, diverges from documented workflow); relying on host-installed binaries without management (rejected—breaks portability requirement).

### Decision: Enforce canonical import overlays through enhancements to `apx sync` backed by go.work management helpers
- **Rationale**: Quickstart describes deterministic overlays. Centralizing overlay operations prevents command-specific drift and keeps generated code uncommitted.
- **Alternatives considered**: Let each command manipulate go.work independently (rejected—risk of inconsistent state); require users to run go work manually (rejected—violates developer experience-first principle).

## Follow-Up Questions

- Do we need additional schema formats beyond those listed (proto, openapi, avro, jsonschema, parquet) for the initial milestone? (Assumed NO per spec.)
- Are there organization-specific defaults (e.g., buf lint templates) that must remain configurable per `apx.yaml`? (Plan assumes yes; will surface via config loader design.)

## References

- `/docs/getting-started/quickstart.md`
- `/docs/publishing/index.md`
- `/docs/dependencies/index.md`
- `.specify/memory/constitution.md`
