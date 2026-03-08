# Import Model Gaps — Implementation Plan

This plan addresses the five gaps identified in `importmodel_gap.md`, ordered by dependency chain and value.

---

## Phase 1: Catalog Schema Update (Gap 3 prerequisite)

**Goal:** Extend `catalog.Module` to support identity fields so publish, show, and search can use them.

### Tasks

1. **Extend `catalog.Module` struct** in `internal/catalog/generator.go`:
   ```go
   type Module struct {
       ID               string   `yaml:"id"`                          // e.g. "proto/payments/ledger/v1"
       Name             string   `yaml:"name"`                        // backward compat
       Format           string   `yaml:"format"`
       Domain           string   `yaml:"domain,omitempty"`
       APILine          string   `yaml:"api_line,omitempty"`          // e.g. "v1"
       Description      string   `yaml:"description,omitempty"`
       Version          string   `yaml:"version"`
       LatestStable     string   `yaml:"latest_stable,omitempty"`
       LatestPrerelease string   `yaml:"latest_prerelease,omitempty"`
       Lifecycle        string   `yaml:"lifecycle,omitempty"`
       Path             string   `yaml:"path"`
       Tags             []string `yaml:"tags,omitempty"`
       Owners           []string `yaml:"owners,omitempty"`
   }
   ```

2. **Backward compatibility:** we don't need to maintain backward compatibility for `Name` since it's not currently used in a critical way. We can deprecate it in favor of `ID`.

3. **Update `Generator.Scan()`** to populate new fields when scanning filesystem.

4. **Update `SearchModules()`** to support filtering by `lifecycle`, `domain`, `api_line`.

5. **Update search command** to add `--lifecycle` and `--domain` flags and display new fields.

6. **Tests:**
   - Unit tests for new Module fields in catalog load/save
   - Update `testdata/script/search_catalog.txt` with identity-aware catalog entries

**Files:** `internal/catalog/generator.go`, `internal/catalog/search.go`, `cmd/apx/commands/search.go`, `internal/catalog/search_test.go`

**Estimate:** 1 session

---

## Phase 2: Catalog Recording on Publish (Gap 3)

**Goal:** When `apx publish` completes, update `catalog.yaml` with the published API's identity, version, and lifecycle.

### Approach: tag-based regeneration (preferred over incremental update)

Instead of editing `catalog.yaml` during publish (concurrency-unsafe), publish creates a git tag. Then a separate `apx catalog generate` command regenerates the catalog from tags.

### Tasks

1. **Add `apx catalog generate` command** in `cmd/apx/commands/catalog.go`:
   - Scan git tags matching `<format>/<domain>/<name>/<line>/v*`
   - Parse each tag into API identity + version
   - Determine latest stable and latest prerelease per API line
   - Write `catalog/catalog.yaml`

2. **Add `catalog generate` to publish post-step (optional):**
   - After `publishWithIdentity()` creates the tag, optionally run `catalog generate` if `--update-catalog` is passed
   - Default: don't auto-update (CI handles it)

3. **Read lifecycle from apx.yaml or flag:**
   - Publish already writes lifecycle via `--lifecycle` flag
   - Store lifecycle in a tag annotation or a sidecar file (`catalog/lifecycle/<api-id>.yaml`)
   - Simpler: read lifecycle from apx.yaml's `api.lifecycle` field (already exists in config)

4. **Tests:**
   - Testscript: `catalog_generate.txt` — set up repo with tags, run `apx catalog generate`, verify catalog.yaml output
   - Unit test: tag parsing, version sorting, catalog assembly

**Files:** `cmd/apx/commands/catalog.go`, `cmd/apx/commands/root.go`, `internal/catalog/generator.go`

**Estimate:** 1–2 sessions

---

## Phase 3: `apx show` Command (Gap 4)

**Goal:** `apx show proto/payments/ledger/v1` prints full identity + catalog data.

### Tasks

1. **Add `apx show` command** in `cmd/apx/commands/show.go`:
   - Takes an API ID as argument
   - Resolves source repo from flag or `apx.yaml`
   - Calls `BuildIdentityBlock()` for derived fields (module, import, tag)
   - Reads `catalog.yaml` for catalog fields (latest stable, latest prerelease, owners, lifecycle)
   - Merges and prints

2. **Output format:**
   ```
   API: proto/payments/ledger/v1
   Format: proto
   Domain: payments
   Name: ledger
   Line: v1
   Lifecycle: beta
   Latest stable: none
   Latest prerelease: v1.0.0-beta.1
   Go module: github.com/acme/apis/proto/payments/ledger
   Go import: github.com/acme/apis/proto/payments/ledger/v1
   Owners: team-payments
   ```

3. **Graceful degradation:** If no catalog is available, show only derived fields (same as `inspect identity`). Print a note: "Run `apx catalog generate` for release data."

4. **JSON support:** `apx --json show proto/payments/ledger/v1`

5. **Register command** in `root.go`.

6. **Tests:**
   - Testscript: `show.txt` — with and without catalog, JSON output, bad API ID
   - Update docs: `cli-reference/core-commands.md`

**Files:** `cmd/apx/commands/show.go`, `cmd/apx/commands/root.go`

**Estimate:** 1 session

**Depends on:** Phase 1 (catalog schema), Phase 2 (catalog data)

---

## Phase 4: `go_package` Validation (Gap 1)

**Goal:** During publish, validate that `.proto` files' `option go_package` matches the derived import path.

### Approach: lightweight regex extraction (no full proto parser)

### Tasks

1. **Add `ExtractGoPackage()` in `internal/validator/proto.go`:**
   ```go
   // ExtractGoPackage reads a .proto file and extracts the go_package option.
   // Returns (import_path, alias, error).
   // Handles: option go_package = "path/to/pkg;alias";
   func ExtractGoPackage(protoPath string) (string, string, error)
   ```
   parse using https://github.com/bufbuild/protocompile
   

2. **Add `ValidateGoPackage()` in `internal/config/identity.go`:**
   ```go
   // ValidateGoPackage checks that a go_package value matches the derived import path.
   func ValidateGoPackage(goPackage string, expectedImport string) error
   ```
   - Strip alias suffix (`;alias`)
   - Compare path portion to `expectedImport`

3. **Wire into `publishWithIdentity()`:**
   - After deriving identity, glob `source.Path/**/*.proto`
   - For each proto file, extract `go_package` and validate
   - On mismatch: warning (not error) with `--strict` flag to make it an error

4. **Handle edge cases:**
   - Files without `go_package` → skip (buf managed mode may handle it)
   - Multiple files with different `go_package` → warn about the mismatched ones
   - Non-proto API formats → skip validation entirely

5. **Tests:**
   - Unit: `TestExtractGoPackage` (valid, with alias, missing, malformed)
   - Unit: `TestValidateGoPackage` (match, mismatch, alias handling)
   - Integration: testscript with proto files containing correct/incorrect `go_package`

**Files:** `internal/validator/proto.go`, `internal/config/identity.go`, `cmd/apx/commands/publish.go`

**Estimate:** 1 session

---

## Phase 5: `go.mod` Generation (Gap 2)

**Goal:** During publish, create or validate the subdirectory `go.mod` in the canonical repo.

### Approach: generate minimal `go.mod`, let CI run `go mod tidy`

### Tasks

1. **Add `GenerateGoMod()` in `internal/publisher/gomod.go`:**
   ```go
   // GenerateGoMod creates a minimal go.mod file for a published API module.
   func GenerateGoMod(modulePath string, goVersion string) ([]byte, error)
   ```
   Generates:
   ```
   module github.com/acme/apis/proto/payments/ledger

   go 1.21
   ```
   No `require` entries — those come from `go mod tidy` in CI.

2. **Compute `go.mod` placement:**
   - v1: `<format>/<domain>/<name>/go.mod` (module root above the `/v1/` package dir)
   - v2+: `<format>/<domain>/<name>/v<N>/go.mod` (module root = package dir)

3. **Add `DeriveGoModDir()` in `internal/config/identity.go`:**
   ```go
   func DeriveGoModDir(api *APIIdentity) string
   ```

4. **Wire into `publishWithIdentity()`:**
   - Before subtree split, check if `go.mod` exists at the derived location
   - If missing: generate minimal `go.mod` and write it
   - If exists: validate `module` directive matches `DeriveGoModule()`
   - On mismatch: error

5. **`--skip-gomod` flag:** Allow skipping for non-Go workflows.

6. **Tests:**
   - Unit: `TestGenerateGoMod`, `TestDeriveGoModDir`
   - Testscript: publish with go.mod validation

**Files:** `internal/publisher/gomod.go`, `internal/config/identity.go`, `cmd/apx/commands/publish.go`

**Estimate:** 1–2 sessions

**Depends on:** Phase 4 (both are publish validation steps)

---

## Phase 6: API ID Argument for lint/breaking (Gap 5)

**Goal:** `apx lint proto/payments/ledger/v1` resolves the API ID to a filesystem path.

### Tasks

1. **Add `ResolveAPIPath()` in `internal/config/resolve.go`:**
   ```go
   // ResolveAPIPath resolves an API ID to a filesystem path using module_roots from config.
   // If the argument is already a valid filesystem path, returns it unchanged.
   func ResolveAPIPath(arg string, cfg *Config) (string, error)
   ```
   Logic:
   - If `os.Stat(arg)` succeeds → it's a path, return as-is
   - If `ParseAPIID(arg)` succeeds → search `module_roots` for matching directory
   - Fall back to checking common patterns: `internal/apis/<arg>`, `schemas/<arg>`, `<arg>`

2. **Wire into `lintAction()` and `breakingAction()`:**
   - Before passing path to validator, run `ResolveAPIPath()`
   - Transparent: existing path arguments still work

3. **Tests:**
   - Unit: `TestResolveAPIPath` (path input, API ID input, not found)
   - Testscript: `lint_proto.txt` updates — add case with API ID syntax

**Files:** `internal/config/resolve.go`, `cmd/apx/commands/lint.go`, `cmd/apx/commands/breaking.go`

**Estimate:** 1 session

---

## Dependency Graph

```
Phase 1 (catalog schema)
  ↓
Phase 2 (catalog generate)
  ↓
Phase 3 (apx show)

Phase 4 (go_package validation)  ← independent
  ↓
Phase 5 (go.mod generation)

Phase 6 (lint/breaking API ID)  ← independent
```

Phases 1–3 are a chain. Phases 4–5 are a chain. Phase 6 is independent.
All three chains can be worked in parallel.

## Priority Order

| Priority | Phase | Value | Risk |
|----------|-------|-------|------|
| 1 | Phase 1 — Catalog schema | Unlocks show + search improvements | Low — additive struct changes |
| 2 | Phase 6 — lint/breaking API ID | High UX value, simple | Low — path resolution only |
| 3 | Phase 2 — Catalog generate | Enables release tracking | Medium — git tag parsing |
| 4 | Phase 3 — apx show | Developer-facing discovery | Low — composition of existing code |
| 5 | Phase 4 — go_package validation | Guardrail for correctness | Medium — proto parsing edge cases |
| 6 | Phase 5 — go.mod generation | Full publish automation | Medium — Go module ecosystem |

**Total estimate:** 6–8 sessions across all phases.
