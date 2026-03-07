# Feature Specification: Canonical APX Configuration Model

**Feature Branch**: `006-canonical-config-schema`  
**Created**: 2026-03-07  
**Status**: Draft  
**Input**: User description: "Create a single canonical APX configuration model so developers only have one valid structure for apx.yaml. The feature should eliminate conflicting shapes in documentation and implementation, support clear validation errors, and provide a stable schema that can evolve with explicit versioning. Success means new and existing users can understand, validate, and migrate apx.yaml files without ambiguity."

## User Scenarios & Testing *(mandatory)*

### User Story 1 — Validate an Existing apx.yaml Against the Canonical Schema (Priority: P1)

A developer who already has an `apx.yaml` in their repository runs a single validation command and immediately learns whether their file conforms to the canonical schema. If it does not, they receive specific, field-level error messages that name the offending field, describe the violation, and indicate the expected value or type. They can fix all issues without consulting source code.

**Why this priority**: This is the primary day-to-day interaction with the schema. Without reliable, actionable validation errors, every other improvement is theoretical — developers will still encounter silent misconfiguration at runtime.

**Independent Test**: Given a set of intentionally malformed `apx.yaml` files (missing required fields, wrong types, unknown keys, incompatible versions), run `apx config validate` against each. Every violation must produce an error message that identifies the field and describes the problem. Fully valid files must produce no errors. This story is independently demonstrable as a standalone CLI invocation.

**Acceptance Scenarios**:

1. **Given** a valid `apx.yaml` with `version: 1` and all required fields, **When** a developer runs `apx config validate`, **Then** the command exits successfully with no error output.
2. **Given** an `apx.yaml` that omits the required `org` field, **When** a developer runs `apx config validate`, **Then** the command fails and the error message names `org` as the missing field.
3. **Given** an `apx.yaml` that contains an unrecognized top-level key (e.g., `foobar: true`), **When** a developer runs `apx config validate`, **Then** the command fails and the error message identifies `foobar` as an unknown field.
4. **Given** an `apx.yaml` where `publishing.tag_format` contains an invalid placeholder pattern, **When** a developer runs `apx config validate`, **Then** the error message names `publishing.tag_format` and describes the valid placeholder syntax.
5. **Given** `apx.example.yaml` in the repository root, **When** a developer runs `apx config validate --config apx.example.yaml`, **Then** the command exits successfully.

---

### User Story 2 — Initialize a Repository with a Conforming apx.yaml (Priority: P1)

A developer runs `apx init` and the generated `apx.yaml` is immediately accepted by `apx config validate` with no modifications. The generated file contains all required fields, uses the canonical structure, and matches the documented schema definition.

**Why this priority**: The `init` command is the entry point for new users. If the generated file does not conform to the canonical schema, the product undermines its own contract from the very first step.

**Independent Test**: Run `apx init` (both `canonical` and `app` modes) in a clean directory. Load each generated `apx.yaml` through `apx config validate`. Both must pass without error.

**Acceptance Scenarios**:

1. **Given** a clean directory, **When** a developer runs `apx init canonical proto my.org/schemav1`, **Then** the generated `apx.yaml` passes `apx config validate` without modification.
2. **Given** a clean directory, **When** a developer runs `apx init app`, **Then** the generated `apx.yaml` passes `apx config validate` without modification.
3. **Given** a successfully generated `apx.yaml`, **When** a developer opens it, **Then** every field present is documented in the canonical schema reference with a description of its purpose.

---

### User Story 3 — Migrate an Older apx.yaml to the Current Schema Version (Priority: P2)

A developer with an existing project on a previous schema version runs a migration command (or follows migration guidance) to bring their `apx.yaml` up to the current version. The process is non-destructive, produces a valid file, and explains every change made.

**Why this priority**: Users with existing repos must be able to adopt the canonical schema without manual archaeology. A migration path protects existing investments and enables the schema to evolve safely over time.

**Independent Test**: Given an `apx.yaml` with `version: 1` that predates a specific structural change, run `apx config migrate`. The output file must pass `apx config validate` and a changelog of changes must be displayed or written to the terminal.

**Acceptance Scenarios**:

1. **Given** an `apx.yaml` at `version: 1` that is missing fields added in a later version, **When** a developer runs `apx config migrate`, **Then** the command adds missing fields with documented defaults and sets `version` to the current value.
2. **Given** an `apx.yaml` with fields that have been renamed in the current schema, **When** a developer runs `apx config migrate`, **Then** values are moved to the new field names and obsolete keys are removed.
3. **Given** a migration that makes no changes (the file is already current), **When** a developer runs `apx config migrate`, **Then** the command exits cleanly and reports that no migration was needed.
4. **Given** a migration is run, **When** changes are applied, **Then** the original file is backed up before being modified.

---

### User Story 4 — Understand the Schema from Documentation (Priority: P2)

A developer who has never used APX reads the configuration reference in the docs and gains a complete, unambiguous picture of every field, its type, whether it is required or optional, its default value, and the valid range of values. They can compose a valid `apx.yaml` from scratch using only the documentation.

**Why this priority**: Documentation is the primary interface between the product and developers who are not yet using it. If the docs and the implementation conflict, developers write broken files and trust erodes. This story is independently valuable once the schema is stable.

**Independent Test**: Given the published schema reference docs, follow them to hand-craft an `apx.yaml` from scratch for a proto-only project and for a multi-format project. Run `apx config validate` on both. Both must pass. Any field described as optional must be omittable without validation failure.

**Acceptance Scenarios**:

1. **Given** the schema reference documentation, **When** a developer writes an `apx.yaml` by hand using only the docs as a guide, **Then** `apx config validate` accepts it.
2. **Given** the schema reference marks a field as optional, **When** a developer omits that field, **Then** `apx config validate` does not reject the file for the omitted field.
3. **Given** the schema reference specifies an enumerated set of valid values for a field (e.g., `execution.mode`), **When** a developer supplies a value outside that set, **Then** validation fails with an error naming the field and listing the accepted values.
4. **Given** the docs describe a field as deprecated, **When** a developer uses it, **Then** validation emits a warning (not an error) and points to the replacement field.

---

### Edge Cases

- What happens when `apx.yaml` is completely empty or contains only whitespace?
- What happens when the `version` field is absent — is the file rejected outright, or is a default assumed?
- How does the system behave when `version` references a schema version higher than the installed APX binary supports?
- What happens when conflicting values exist under the same key parsed from environment variable overrides and the file?
- How are unknown keys under a known section (e.g., an unrecognized sub-key under `policy`) treated — hard error or warning?
- What happens when `module_roots` is an empty list?

---

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST define exactly one canonical structure for `apx.yaml` per schema version; no alternative layouts or field aliases that produce different behavior for the same intent.
- **FR-002**: The system MUST expose a `validate` subcommand (e.g., `apx config validate`) that loads an `apx.yaml` and reports all schema violations before any operation is performed.
- **FR-003**: Validation errors MUST identify the specific field path (e.g., `publishing.tag_format`), the nature of the violation (missing, wrong type, invalid value, unknown key), and the expected value or constraint.
- **FR-004**: The schema MUST include an explicit `version` integer field; the system MUST reject files with an unrecognized version value and provide a clear error.
- **FR-005**: All fields in the schema MUST be documented with their name, type, required/optional status, default value, and accepted values or pattern.
- **FR-006**: The `apx init` command MUST generate an `apx.yaml` that passes `apx config validate` without modification, for every supported init mode.
- **FR-007**: The system MUST expose a `migrate` subcommand (e.g., `apx config migrate`) that upgrades an `apx.yaml` from an older schema version to the current version non-destructively, backing up the original file.
- **FR-008**: `apx config migrate` MUST output a human-readable summary of every change it applies.
- **FR-009**: When no changes are required during migration, the system MUST report a clean status without modifying any files.
- **FR-010**: The canonical schema MUST treat unrecognized top-level keys as hard validation errors to prevent silently accepted typos.
- **FR-011**: Deprecated fields MUST produce a warning during validation (not a hard error) and the warning MUST name the replacement field.
- **FR-012**: The schema reference documentation MUST be generated from or verified against the authoritative schema definition to prevent drift between docs and implementation.
- **FR-013**: The system MUST validate that required fields (`version`, `org`, `repo`) are present and non-empty in every `apx.yaml`; absence of any required field is a hard error.
- **FR-014**: Users MUST be able to point any APX command to an alternate config file via a flag or environment variable; the same validation rules apply regardless of the config file path.

### Key Entities

- **Schema Version**: An integer that identifies the structural contract of `apx.yaml`. Each version maps to a defined set of allowed fields, required fields, and type constraints. Versions are monotonically increasing.
- **Configuration File (`apx.yaml`)**: The single file that controls APX behavior for a repository. Contains identity fields (org, repo), module layout, language generation targets, schema-specific policies, publishing settings, tool pin versions, and execution mode.
- **Validation Result**: The output of the `validate` subcommand — either a success signal or an ordered list of `ValidationError` items. Each item carries a field path, violation kind, and remediation hint.
- **Migration Plan**: The set of transformations that `apx config migrate` would apply to bring an `apx.yaml` from version N to the current version. Can be previewed (dry-run) before writing.
- **Canonical Schema Definition**: The authoritative, versioned specification of `apx.yaml` field structure. All other artifacts (documentation, generated init templates, validation logic) are derived from or verified against this single definition.

---

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A developer encountering their first validation error can identify the offending field and understand the fix without reading source code — measured by all error messages including field path and remediation hint.
- **SC-002**: 100% of files generated by `apx init` pass `apx config validate` without any manual edits, across all supported init modes.
- **SC-003**: A developer can migrate an existing `apx.yaml` from any prior schema version to the current version in under 2 minutes using the `migrate` command alone, with no manual file editing required.
- **SC-004**: The schema reference documentation has zero fields that contradict `apx config validate` behavior — confirmed by an automated or manual audit at release time.
- **SC-005**: Zero previously valid `apx.yaml` files (those passing validation on the prior version) are silently invalidated by a schema version upgrade — migration either fixes them automatically or produces an actionable error.
- **SC-006**: Developers can write a syntactically valid `apx.yaml` from scratch using only the documentation, and the file passes `apx config validate` on the first attempt, at a success rate of at least 80% in user testing or documented walkthroughs.

---

## Assumptions

- The `version` field already exists in the current `apx.yaml` structure (`version: 1`); this feature formalizes the contract around it rather than introducing it from scratch.
- Schema migration need only support upgrading forward (older → current); downgrading to a prior version is out of scope.
- Environment variable overrides (if any) follow the same field-naming conventions as the file and are subject to the same type constraints.
- A dry-run flag for `apx config migrate` is desirable but can be delivered as a follow-on; the initial implementation writes files with a mandatory backup.
- Documentation generation or verification automation is a quality gate, not a new user-facing feature; it may run as part of CI rather than as a separate CLI command.
