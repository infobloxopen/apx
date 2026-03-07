# Research: Canonical APX Configuration Model

**Feature**: 006-canonical-config-schema  
**Date**: 2026-03-07

## Research Questions

### 1. How to implement strict YAML unmarshalling that rejects unknown keys in Go with yaml.v3?

**Decision**: Use a two-pass approach — first unmarshal into `yaml.Node` tree, then walk the tree comparing keys against a whitelist derived from the schema definition. Do NOT rely on `yaml.v3`'s struct tags alone because `yaml.v3` silently ignores unknown keys by default and has no `DisallowUnknownFields` equivalent (unlike `encoding/json`).

**Rationale**: The `yaml.v3` library (`gopkg.in/yaml.v3`) does not provide a built-in strict mode. The only reliable way to reject unknown keys is to:
1. Unmarshal into `*yaml.Node` to get the raw key-value tree with line numbers.
2. Walk the node tree and compare each mapping key against the set of allowed keys for that section.
3. Collect all unknown keys as `ValidationError` items with field paths and line numbers.

This approach also enables reporting _all_ errors at once rather than failing on the first one, which matches FR-003 ("report all schema violations").

**Alternatives considered**:
- **Unmarshal into `map[string]interface{}` then compare**: Loses line number information and type granularity. Rejected because error messages without line numbers are significantly less useful.
- **Use `encoding/json` with `DisallowUnknownFields` after converting YAML→JSON**: Adds a conversion step and loses YAML-specific error context (line/column). Rejected for unnecessary complexity.
- **Switch to a schema specification library (e.g., JSON Schema validator for YAML)**: Adds a large external dependency. Rejected per constraint of no new external deps for validation.

---

### 2. How should the schema version registry be structured?

**Decision**: Define a `SchemaVersion` struct that captures the version number, a map of allowed fields with metadata (type, required, default, description, enum values), and a set of migration steps from the previous version. Store versions as a Go slice in `internal/config/schema.go`, keyed by version integer. The current schema version is a package-level constant.

**Rationale**: A Go-native registry (not an external file) keeps the schema definition co-located with the validation code, ensures compile-time type safety, and avoids the need to ship/load additional data files. Since there is currently only `version: 1` and the product is pre-stable, the registry starts with a single entry and grows as versions are added.

**Alternatives considered**:
- **JSON Schema file shipped alongside the binary**: Requires a JSON Schema validator dependency and file distribution. Rejected for added complexity.
- **Code-generated from a DSL or spec file**: Over-engineering for the current scale. Can be revisited if schema versions proliferate. Rejected for now.

---

### 3. How should migration transformations be represented?

**Decision**: Each migration step is a function `func(node *yaml.Node) ([]Change, error)` that takes a raw YAML tree, applies transformations in place, and returns a list of `Change` structs describing what was modified. Migrations are chained: to go from version N to current, apply migrations N→N+1, N+1→N+2, etc.

**Rationale**: Using `yaml.Node` as the migration medium preserves comments and formatting in the user's file. Function-based migrations are flexible enough to handle renames, additions, removals, and type coercions. The `Change` return value supports FR-008 (human-readable summary).

**Alternatives considered**:
- **Declarative migration rules (rename: old→new, add: field=default)**: Simpler for basic cases but cannot handle complex transformations (e.g., restructuring nested sections). Rejected because it limits future expressiveness.
- **Unmarshal old version → marshal new version**: Destroys comments and formatting. Rejected because users care about their YAML formatting.

---

### 4. How to detect unknown keys at arbitrary nesting depth?

**Decision**: Build a recursive field-definition tree where each node specifies its allowed children. During validation, walk the `yaml.Node` tree depth-first. At each mapping node, look up the parent's field definition and check whether each key is in the allowed set. Unknown keys produce errors with full dotted paths (e.g., `policy.openapi.unknown_field`).

**Rationale**: The current `apx.yaml` has three levels of nesting (top → section → sub-section, e.g., `policy.openapi.spectral_ruleset`). A recursive approach handles arbitrary depth without special cases.

**Alternatives considered**:
- **Flat key validation (top-level only)**: Misses unknown keys in nested sections. Rejected because spec FR-010 says "unrecognized top-level keys" but the spirit extends to all levels for safety.

---

### 5. How should `apx.yaml` emitted by `init` be unified across code paths?

**Decision**: Consolidate all three YAML-emitting functions (`config.Init()`, `schema.Initializer.createConfigWithDefaults()`, `schema.AppScaffolder.generateApxYaml()`) to use a single function that marshals the `Config` struct. The struct becomes the single source of truth. A `DefaultConfig()` factory function produces a valid `Config` with sensible defaults, and each init mode customizes specific fields before marshalling.

**Rationale**: Three independent string-template emitters is the root cause of the problem described in the spec — they can (and do) diverge. Using struct marshalling guarantees the output matches the schema definition because the struct _is_ the schema definition. This also makes the init output automatically pass validation.

**Alternatives considered**:
- **Keep separate templates but validate them in tests**: Fixes symptoms but not the root cause. Still three places to update when fields change. Rejected.
- **Use a YAML template file embedded in the binary**: Adds embed complexity and still requires synchronization with the struct. Rejected for marginal benefit.

---

### 6. How should validation errors be structured for both human and machine consumption?

**Decision**: Each validation error is a `ValidationError` struct with fields: `Field` (dotted path), `Kind` (enum: missing, invalid_type, unknown_key, invalid_value, deprecated), `Message` (human sentence), `Line` (from yaml.Node, 0 if unavailable), and `Hint` (suggested fix). The `apx config validate` command renders these as human-readable lines by default and as JSON when `--json` is passed.

**Rationale**: Aligns with FR-003 (field path + nature + expected value), SC-001 (remediation hint), and the constitution's `--json` flag convention.

**Alternatives considered**:
- **Simple string errors**: Loses structured information for tooling and CI. Rejected.

---

### 7. What is the backup strategy for `apx config migrate`?

**Decision**: Before writing the migrated file, copy the original to `apx.yaml.bak`. If `apx.yaml.bak` already exists, append a timestamp suffix (e.g., `apx.yaml.bak.20260307T143000`). This is simple, transparent, and requires no additional dependencies.

**Rationale**: FR-007 requires a non-destructive backup. A `.bak` file is the most discoverable convention for developers. Timestamped fallback prevents accidental overwrites of previous backups.

**Alternatives considered**:
- **Write to stdout only, never modify the file**: Forces users to manually redirect and replace. Rejected for poor UX.
- **Git stash or commit**: Assumes git is available and the repo is clean. Rejected because APX can operate outside git repos.

---

### 8. How should deprecated fields be tracked?

**Decision**: The field-definition metadata includes an optional `DeprecatedSince` (version int) and `Replacement` (field path string). During validation, if a deprecated field is present, emit a warning (not error) using `ui.Warning()`. The warning message names the deprecated field and the replacement.

**Rationale**: FR-011 requires warning-not-error semantics. Tracking deprecation metadata in the schema definition keeps it co-located with the field definition and ensures the migration code can also use it.

---

## Summary

All research questions are resolved. No external dependencies are needed. The approach relies entirely on `gopkg.in/yaml.v3`'s `Node` API for strict validation and comment-preserving migration, with Go-native schema definitions in `internal/config/schema.go`.
