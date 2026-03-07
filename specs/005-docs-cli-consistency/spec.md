# Feature Specification: Docs-CLI Consistency

**Feature Branch**: `005-docs-cli-consistency`
**Created**: 2026-03-07
**Status**: Draft
**Input**: User description: "Make APX documentation and CLI behavior fully consistent so developers can trust the product. The feature should ensure that documented commands, flags, examples, config formats, lockfile formats, and catalog examples match the actual implementation and current release. When docs describe unsupported behavior, the system must either implement it or clearly mark it as future work. Success means a developer can follow the public getting-started and publishing docs end-to-end without hitting mismatches, dead commands, or incorrect examples."

## User Scenarios & Testing *(mandatory)*

### User Story 1 — Quickstart End-to-End (Priority: P1)

A developer installs APX, follows the getting-started quickstart guide, and successfully initializes a schema repo, lints a schema, checks for breaking changes, computes a version suggestion, searches the catalog, and adds a dependency — without encountering any unknown-command errors, wrong flag names, or incorrect examples.

**Why this priority**: The quickstart is the first thing a new user reads. If it fails, the product appears broken and untrustworthy. Every command, flag, and example in the quickstart must match the running CLI.

**Independent Test**: Walk through every shell command in `docs/getting-started/quickstart.md` against the compiled binary. Each command must either succeed or produce the documented error output — never "unknown command" or "unknown flag".

**Acceptance Scenarios**:

1. **Given** APX is installed and the quickstart guide is open, **When** a developer copies and runs every `apx` command in the guide (substitute their own repo names), **Then** no command produces "unknown command", "unknown flag", or "unknown shorthand flag" errors.
2. **Given** the quickstart documents `apx breaking`, **When** a developer runs that command, **Then** the documented flags and arguments match the CLI's required and optional flags.
3. **Given** the quickstart documents `apx search <query>`, **When** a developer runs `apx search payments`, **Then** the command succeeds (one positional arg, not two).
4. **Given** the quickstart references `apx version suggest`, **When** a developer runs it, **Then** the command is either renamed in docs to match `apx semver suggest` or an alias exists so the documented command works.

---

### User Story 2 — CLI Reference Accuracy (Priority: P1)

A developer consults the CLI reference page to discover available commands, flags, argument counts, and environment variables. Every item listed exists in the binary, every required flag is marked as such, and no undocumented commands or flags are missing.

**Why this priority**: The CLI reference is the single source of truth for tool behavior. Inaccurate references cause CI scripts to fail and erode developer trust.

**Independent Test**: For each command listed in `docs/cli-reference/index.md`, run `apx <command> --help` and confirm descriptions, flags, and argument counts match. For each flag listed in the docs, confirm it exists. For each environment variable listed, confirm it is read by the binary.

**Acceptance Scenarios**:

1. **Given** the CLI reference lists a command (e.g., `apx lint`, `apx breaking`), **When** a developer runs `apx <command> --help`, **Then** the flags, required args, and description match the docs.
2. **Given** the CLI reference lists environment variables (`APX_CONFIG`, `APX_VERBOSE`, etc.), **When** a developer sets them and runs APX, **Then** documented variables are honored or the docs clearly mark unimplemented ones as "Planned — not yet available".
3. **Given** a command exists in the compiled binary (e.g., `apx catalog build`, `apx config init`), **When** a developer searches the CLI reference, **Then** the command is documented.

---

### User Story 3 — Config File Format Consistency (Priority: P1)

A developer initializes a repo with `apx init`, edits the generated `apx.yaml`, and then runs `apx lint`, `apx gen`, and `apx publish`. The config format produced by `init` is accepted by every other command without parse errors, unknown-field warnings, or type mismatches.

**Why this priority**: The config file is the nucleus of every workflow. If `init` generates YAML that the rest of the CLI rejects, the entire product is unusable.

**Independent Test**: Run `apx init canonical proto my.module.v1`, then run `apx config validate` on the generated `apx.yaml`. Repeat for `apx init app`. Both must pass. Also confirm that the documented config schema in the docs matches the struct fields in `internal/config/config.go`.

**Acceptance Scenarios**:

1. **Given** a user runs `apx init canonical proto my.module.v1`, **When** the generated `apx.yaml` is loaded by `apx config validate`, **Then** it passes validation with no errors.
2. **Given** the docs show a sample `apx.yaml` with specific fields, **When** a developer copies that sample and runs `apx config validate`, **Then** it passes validation — or the docs note which fields are illustrative future examples.
3. **Given** the `apx.example.yaml` file in the repo root, **When** it is loaded by `apx config validate`, **Then** it passes validation.

---

### User Story 4 — Publishing and CI Workflow (Priority: P2)

A developer follows the publishing docs to set up CI pipelines for schema validation, breaking-change detection, version management, and publishing. Every CI template command works with the current release.

**Why this priority**: CI integration is the core value proposition (automated governance). Broken CI templates mean blocked releases and support escalations.

**Independent Test**: Extract every `apx` command from all CI template blocks in the docs, run them against the binary (with appropriate config), and confirm each produces a zero or documented non-zero exit code — never "unknown command / flag".

**Acceptance Scenarios**:

1. **Given** a CI template in the publishing docs, **When** a developer pastes it into their pipeline, **Then** every `apx` command in the template is valid (exists, has correct flags).
2. **Given** the docs reference `apx fetch --ci`, **When** a developer runs it, **Then** either the `--ci` flag exists or the docs use the actual flags (`--output`, `--verify`).
3. **Given** the docs reference `apx tag subdir` and `apx packages publish`, **When** those commands don't exist, **Then** the docs either remove them or mark them as "Planned — not yet available".

---

### User Story 5 — Dependency and Lockfile Workflows (Priority: P2)

A developer follows the dependency docs to search for schemas, add a dependency, generate code, and understand the lockfile format. The documented lockfile structure matches what APX reads and writes.

**Why this priority**: Incorrect lockfile documentation leads to manual edits that corrupt the file, causing cascading failures across dependent teams.

**Independent Test**: Run `apx add <dep>`, inspect the generated `apx-lock.yaml`, and confirm its structure matches the format described in the docs. Also confirm documented commands for updating and listing dependencies exist or are clearly marked as planned.

**Acceptance Scenarios**:

1. **Given** the dependency docs describe the lockfile format, **When** a developer compares it to an actual `apx-lock.yaml` produced by the CLI, **Then** field names, nesting, and types match.
2. **Given** the docs describe `apx update` and `apx upgrade` commands, **When** those commands don't exist yet, **Then** the docs either remove them or clearly mark them as "Planned".
3. **Given** the docs describe `apx list apis` and `apx show`, **When** those commands don't exist yet, **Then** the docs either remove them or clearly mark them as "Planned".

---

### User Story 6 — Broken Internal Links Resolved (Priority: P3)

A developer navigates the published documentation site and clicks cross-reference links. No link leads to a 404 page.

**Why this priority**: Broken links are a signal of abandonment. They erode confidence but don't block functionality, so they are lower priority than command mismatches.

**Independent Test**: Run a Sphinx/MkDocs build and confirm zero broken-link warnings. Alternatively, spider the generated site and confirm all internal hrefs resolve.

**Acceptance Scenarios**:

1. **Given** the documentation contains toctree references to sub-pages, **When** the documentation is built, **Then** no "toctree contains reference to nonexisting document" warnings are emitted.
2. **Given** a developer clicks an internal link on the docs site, **When** the target page doesn't exist, **Then** the link is either removed, pointed to an existing page, or the target page is created.

---

### Edge Cases

- What happens when docs reference a future command that doesn't exist yet? The docs must use a clear, consistent marker (e.g., *"Planned — not yet available"* callout) rather than presenting it as working.
- What happens when a flag is optional in code but shown without explanation in docs? Docs must indicate whether it's required or optional, and show the default value.
- What happens when `apx init` generates a config with fields that differ from what `apx config validate` expects? The generated config must always pass validation; this is a code fix, not a docs fix.
- What happens when exit codes differ between docs and implementation? A single canonical table of exit codes must be defined and all doc pages must reference it.

## Requirements *(mandatory)*

### Functional Requirements

#### Commands & Flags

- **FR-001**: Every `apx` command shown in any documentation file MUST either exist in the compiled binary or be wrapped in a "Planned — not yet available" callout block.
- **FR-002**: Every required flag (e.g., `--against` on `apx breaking`) MUST be shown in all documentation examples that use that command.
- **FR-003**: Command names in documentation MUST match the CLI exactly. Where the CLI uses `apx semver suggest`, docs MUST NOT say `apx version suggest`.
- **FR-004**: Positional argument counts in documentation examples MUST match the `cobra.Args` validation in the command definition (e.g., `apx search` takes at most 1 arg, not 2).
- **FR-005**: All commands present in the compiled binary (including `apx catalog build`, `apx config init`, `apx config validate`) MUST be documented in the CLI reference.
- **FR-006**: All flags present in the compiled binary (including `--against`, `--dry-run`, `--create-pr`, `--clean`, `--manifest`, `--output`, `--verify`, `--skip-git`) MUST be documented for their respective commands.

#### Config Format

- **FR-007**: The `apx.yaml` generated by `apx init canonical` MUST pass `apx config validate` without errors.
- **FR-008**: The `apx.yaml` generated by `apx init app` MUST pass `apx config validate` without errors.
- **FR-009**: The documented `apx.yaml` schema in the CLI reference and quickstart MUST match the fields in `internal/config/config.go`'s `Config` struct.
- **FR-010**: `apx.example.yaml` in the repository root MUST pass `apx config validate`.

#### Lockfile & Dependencies

- **FR-011**: The documented `apx-lock.yaml` format MUST match the `LockFile` struct fields in `internal/config/dependencies.go` (map-based `Dependencies`, with `Repo`, `Ref`, `Modules` fields).
- **FR-012**: Commands that don't exist (`apx update`, `apx upgrade`, `apx list`, `apx show`) MUST either be removed from docs or clearly marked as planned.

#### CI Templates

- **FR-013**: Every `apx` command in CI template code blocks MUST be a valid, currently-working command with correct flags.
- **FR-014**: CI templates that reference non-existent commands/flags (`apx fetch --ci`, `apx tag subdir`, `apx packages publish`, `apx version verify`) MUST be updated to use current commands or removed.

#### Cross-References & Links

- **FR-015**: All toctree and cross-reference targets in the docs MUST resolve to existing files. Broken link targets MUST be removed or have stub pages created.

#### Stale Content

- **FR-016**: All references to the previous CLI framework (urfave/cli) MUST be removed from documentation.
- **FR-017**: Environment variables listed in docs (`APX_VERBOSE`, `APX_USE_CONTAINER`, `APX_CACHE_DIR`) MUST either be implemented in code OR marked as "Planned" in docs.

#### Exit Codes

- **FR-018**: A single canonical exit-code table MUST be defined and consistently referenced across all documentation pages. The table MUST match the actual exit codes produced by the CLI.

### Key Entities

- **Documentation Page**: A markdown file in `docs/` that describes user-facing behavior (commands, config, workflows, examples).
- **CLI Command**: A cobra command registered in `cmd/apx/commands/root.go` with a name, flags, arg constraints, and help text.
- **Config Schema**: The set of fields in `internal/config/config.go` `Config` struct that defines what `apx.yaml` accepts.
- **Lockfile Schema**: The set of fields in `internal/config/dependencies.go` `LockFile` struct that defines what `apx-lock.yaml` contains.
- **CI Template**: A YAML code block in documentation meant to be copied into a CI pipeline file.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A developer can follow the quickstart guide end-to-end (every documented command) without encountering "unknown command", "unknown flag", or argument-count errors. Pass rate: 100%.
- **SC-002**: Every command listed in the CLI reference matches `apx <command> --help` output for name, description, flags, and argument count. Match rate: 100%.
- **SC-003**: Running `apx config validate` on the config generated by `apx init canonical` and `apx init app` succeeds with zero errors.
- **SC-004**: Every CI template in the docs produces valid (non-"unknown command/flag") output when each command is tested against the binary.
- **SC-005**: Building the documentation site produces zero "nonexisting document" or broken-link warnings.
- **SC-006**: Zero references to "urfave/cli", "survey", or other replaced dependencies remain in user-facing docs.
- **SC-007**: The documented lockfile format, when written as a YAML file and parsed by the CLI lockfile reader, produces no errors.
- **SC-008**: `apx.example.yaml` passes `apx config validate`.
- **SC-009**: No command or flag exists in the compiled binary that is absent from the CLI reference docs.
- **SC-010**: Commands that are planned but not yet implemented are marked with a consistent "Planned" callout visible to users, not presented as working.

## Assumptions

- The CLI reference and getting-started docs are the primary developer-facing documentation. Internal specs files are not considered user-facing.
- Where a mismatch exists between documentation and implementation, the implementation is assumed to be the source of truth unless the documented behavior is trivially implementable (e.g., adding a command alias).
- Commands documented but not yet implemented (`apx list`, `apx show`, `apx update`, `apx upgrade`, `apx tag subdir`, `apx packages publish`) will be marked as "Planned" rather than implemented in this feature — unless they can be trivially wired up from existing internal code.
- The nine non-existent commands represent future roadmap work; this feature is about documentation accuracy, not implementing missing features.
- `start.md` and `next.md` are internal files, not published documentation, so they are not in scope for user-facing consistency fixes.
- This feature covers docs changes and minimal code fixes (e.g., making `init`-generated configs parseable). It does NOT cover implementing new CLI commands.
