# Tasks: 010 Go client generator

Model routing: `[S]` mechanical, `[C]` cross-cutting/design. Gate: all `[X]`, apx build+tests
green, real `oapi-codegen` + `go build` e2e observed, angular path unregressed.

- [ ] **T010-1 [C]** Add optional `Builder` and `Publisher` interfaces to
  `internal/client/client.go` (mirroring `internal/language/plugin.go:68-96`, type-asserted).
  Refactor `buildClientPackage`/`publishClientPackage` (`cmd/apx/commands/client.go:358-408`) to
  use them if the resolved generator implements them, else the existing npm path. No observable
  change for angular.
- [ ] **T010-2 [C]** New `internal/client/goclient.go`: `goGenerator` implements `Generator`
  (`Name()=="go"`, `init()` registers). `Generate` orchestrates
  `go run …/oapi-codegen@<pinnedConst>` (preflight `go` on PATH; temp config YAML with
  `generate: [types, client]`, package name; spec abs+stat), emits `<pkg>.gen.go` + `go.mod`
  (module = `--package`) + `doc.go`, returns `Result`. Fail-loud on tool/spec errors (FM-1/FM-2).
- [ ] **T010-3 [S]** `--package`→Go-module-path handling for `--generator go` (spec resolution in
  `client.go`); derive a default module path when empty; `--scope` ignored for go. Doc the rule.
- [ ] **T010-4 [C]** goGenerator implements `Builder` (`go build ./...` in OutputDir; error ⇒
  fail, FM-3) and `Publisher` (record `go-module` artifact via `internal/publisher/record`;
  git-tag publish reusing `internal/publisher/tags.go`; honor `--dry-run` default true).
- [ ] **T010-5 [S]** Enriched-OpenAPI fixture under apx `testdata/` (a toy spec carrying
  `enum`/`required`/`readOnly` + a couple `x-aip-*` extensions), for the e2e. If feature 044's
  golden is straightforward to vendor, prefer that shape.
- [ ] **T010-6 [C]** e2e test: generate from the fixture with the real tool, run the go
  `Builder`, assert `go build` clean + a typed method per RPC + an enum type + a required field;
  assert `x-aip-*` presence does not break generation (AC-1/2/3).
- [ ] **T010-7 [S]** Backward-compat test: angular still registered; angular build/publish routes
  to npm (Builder/Publisher not implemented by angular ⇒ fallback). Existing
  `internal/client/client_test.go` green (AC-5/FM-4).
- [ ] **T010-8 [S]** `apx client publish --generator go --dry-run` records `go-module` artifact,
  no token needed (AC-4). Command help lists `go` (AC-6).
- [ ] **T010-9 [S]** devedge thin driver (FR-8): extend `de api publish --client` to pass
  `--generator go`. (Separate repo `/Users/dgarcia/go/src/github.com/infobloxopen/devedge`; keep
  minimal — do LAST, after apx is green.)
- [ ] **T010-10 [S]** `go build ./... && <apx test target>` green; `apx` help lists `go`; update
  `docs/cli-reference` client docs to mention `--generator go`. Deterministic.
