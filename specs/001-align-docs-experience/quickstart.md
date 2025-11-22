# Feature Quickstart: Docs-Aligned APX Experience

This guide walks contributors through validating the documentation-driven workflows defined in the feature specification. It assumes you have read `/docs/getting-started/quickstart.md` and understand the APX constitution.

## Prerequisites
- Go 1.24 installed locally
- `buf`, `spectral`, `oasdiff`, `avro-tools`, `jsonschema-diff`, and `parquet-tools` available via `apx fetch`
- Docker (for spinning up the Gitea-based integration harness)
- GitHub Enterprise Server or Gitea access tokens for testing publish flows

## Environment Setup
1. **Install toolchain**
   ```bash
   make install-tools   # wraps scripts/install-tools.sh
   apx fetch            # hydrates binaries into ./bin
   ```
2. **Configure go.work overlays**
   ```bash
   apx sync             # ensures canonical overlay state matches docs
   ```
3. **Launch test dependencies**
   ```bash
   make up-gitea        # (target to be added) starts disposable Gitea for integration tests
   ```

## Validating User Story 1 (Canonical Bootstrap)
```bash
rm -rf /tmp/apx-canonical && mkdir -p /tmp/apx-canonical
cd /tmp/apx-canonical
git init
apx init canonical --org=myorg --non-interactive
```
- Compare generated files with documentation fixtures in `docs/canonical-repo/structure.md`.
- Run `go test ./internal/schema -run TestCanonicalScaffold` (to be authored) to verify structural parity.

## Validating User Story 2 (Author & Publish)
```bash
cd /tmp && rm -rf app-repo && mkdir app-repo && cd app-repo
git init
apx init app internal/apis/proto/payments/ledger --org=myorg
apx lint
apx breaking --against main
git tag proto/payments/ledger/v1/v1.2.3
apx publish --module-path=internal/apis/proto/payments/ledger/v1 --canonical-repo=github.com/myorg/apis
```
- Inspect `testdata/script/publish_ledger.txt` for expected transcripts.
- Verify Gitea receives subtree PR with preserved history.

## Validating User Story 3 (Consumer Overlays)
```bash
cd /tmp && rm -rf consumer && mkdir consumer && cd consumer
git init
apx search ledger
apx add proto/payments/ledger/v1@v1.2.3
apx gen go
apx sync
go test ./...
apx unlink proto/payments/ledger/v1
```
- Confirm go.work entries match patterns from the quickstart documentation.
- Ensure generated code resides under `internal/gen/` and remains untracked.

## Test Matrix
- `go test ./...` (unit coverage for `internal/*` packages)
- `go test ./cmd/apx -run TestHelpParity` (CLI doc parity tests)
- `go test ./tests/integration -run TestPublishWorkflow` (Gitea-backed flows)
- `go test ./testdata/script -run TestScripts` (testscript scenarios)

## Troubleshooting
- **Tool mismatch**: Re-run `apx fetch` and commit updated `apx.lock`.
- **Doc parity failure**: Regenerate golden outputs via `go test ./cmd/apx -run TestRecordGolden -update` and update `/docs/` simultaneously.
- **Gitea auth errors**: Reset tokens using `make reset-gitea` (to be scripted) and reconfigure credentials in `.env`.
