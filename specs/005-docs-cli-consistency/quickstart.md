# Quickstart: Implementing Docs-CLI Consistency

**Feature**: 005-docs-cli-consistency | **Branch**: `005-docs-cli-consistency`

## Prerequisites

- Go 1.26.1+ installed
- APX repo cloned and on branch `005-docs-cli-consistency`
- Python 3.x with `pip install -r docs/requirements.txt` (for Sphinx builds)

## Step 1: Fix the Config Generation Bug

The most critical code change — `apx init app` generates invalid YAML.

**File**: `internal/schema/app.go`, function `generateApxYaml` (line ~92)

**Current** (broken):
```go
content := fmt.Sprintf(`kind: %s
module: %s
org: %s
version: v1
`, format, moduleName, s.org)
```

**Fix**: Generate Config-struct-compatible YAML:
```go
content := fmt.Sprintf(`version: 1
org: %s
repo: %s
module_roots:
  - %s
`, s.org, s.repo, format)
```

Note: The `AppScaffolder` struct may need a `repo` field added. Check the constructor.

**Verify**:
```bash
cd $(mktemp -d)
apx init app --org=testorg --non-interactive internal/apis/proto/payments/ledger
apx config validate  # Must exit 0
```

## Step 2: Fix the Existing Doc-Parity Test

**File**: `cmd/apx/commands/doc_parity_test.go`

Fix the search example (2 args → 1 arg):
```go
// Change:
{"search", []string{"apx", "search", "payments", "ledger"}}
// To:
{"search", []string{"apx", "search", "payments"}}
```

**Verify**:
```bash
go test ./cmd/apx/commands/ -run TestDocParity -v
```

## Step 3: Add Comprehensive Parity Tests

Add to `doc_parity_test.go`:
1. `TestDocParity_AllCommandsExist` — every registered command is findable
2. `TestDocParity_AllFlagsExist` — every documented flag exists on its command
3. `TestDocParity_RequiredFlags` — required flags are enforced
4. `TestDocParity_ConfigRoundtrip` — init generates valid config

See [contracts/doc-parity-test-contract.md](contracts/doc-parity-test-contract.md) for exact patterns.

## Step 4: Rewrite CLI Reference

**File**: `docs/cli-reference/index.md`

Rebuild from the data model in [data-model.md](data-model.md):
- One section per command with `Use`, flags table, examples
- Include the exit-code table
- Mark `APX_VERBOSE`, `APX_USE_CONTAINER`, `APX_CACHE_DIR` as "Planned"

## Step 5: Fix Quickstart Guide

**File**: `docs/getting-started/quickstart.md`

Key fixes:
- `apx version suggest` → `apx semver suggest`
- `apx search payments ledger` → `apx search payments`
- Add `--against` to all `breaking` examples
- Fix config format examples to match Config struct

## Step 6: Fix Publishing & Workflow Docs

**Files**: `docs/publishing/index.md`, `docs/dependencies/index.md`, `docs/app-repos/index.md`

- Replace all non-existent commands with "Planned" admonitions
- Fix CI templates to use real commands/flags
- Fix lockfile format to match `LockFile` struct (map-based, not list)

## Step 7: Fix Broken Links

Run Sphinx to find all broken references:
```bash
cd docs && sphinx-build -W -b html . _build 2>&1 | grep "toctree"
```

For each broken reference, either:
- Create a minimal stub page with a "Coming soon" note
- Remove the toctree entry if the topic isn't planned

## Step 8: Clean Up Stale References

Search and replace:
```bash
grep -rn "urfave\|survey\|cli/v2" docs/ README.md INTERACTIVE_INIT.md
```

Remove or replace all matches.

## Step 9: Validate Everything

```bash
# Go tests (including parity)
go test ./cmd/apx/commands/ -run TestDocParity -v

# Full test suite
go test ./...

# Sphinx build (warnings as errors)
cd docs && sphinx-build -W -b html . _build

# Manual: walk through quickstart end-to-end
```

## Key Files Reference

| File | Role |
|------|------|
| `internal/schema/app.go` | Config generation (fix here) |
| `internal/config/config.go` | Config struct (source of truth) |
| `internal/config/dependencies.go` | Lockfile struct (source of truth) |
| `cmd/apx/commands/doc_parity_test.go` | Parity tests (extend here) |
| `cmd/apx/commands/root.go` | Command registration (read only) |
| `docs/cli-reference/index.md` | CLI reference (rewrite) |
| `docs/getting-started/quickstart.md` | Quickstart (fix) |
| `apx.example.yaml` | Example config (verify only) |
