# Feature Specification: Go client generator (`apx client --generator go`)

**Feature Branch**: `010-go-client-generation`
**Created**: 2026-07-02
**Status**: Draft
**Initiative**: WS-024 (out-of-the-box CLI & Terraform surfaces) — **P0 fast-follow** (after the
devedge-sdk contract-enrichment keystone, feature 044)

## Context

apx already generates and publishes a TypeScript/Angular API client (WS-020) by orchestrating
`ng-openapi-gen` behind a clean `client.Generator` interface + registry
(`internal/client/`). WS-024 needs a **Go client** too: it is what the generated CLI (P1) and
the Terraform provider (P2) call to actually reach the service. The proposal (hub
`specs/cli-and-terraform-seam-proposal.md`, G7d/§8) names it the keystone that "both the CLI
and the TF provider sit on."

The devedge-sdk keystone (feature 044) makes a service's OpenAPI v3 **lossless** — native
`required`/`readOnly`/`writeOnly`/`enum` plus `x-aip-*` extensions. This feature adds an apx
generator that reads that enriched spec and emits a typed, compilable Go client module. It
proves the "one contract, many surfaces" thesis at the first Go surface: enriched OpenAPI in,
Go client out.

apx's charter is **orchestrate, don't reimplement** (memory `apx-vs-buf-codegen-boundary`): the
generator drives `oapi-codegen` (the standard OpenAPI-3 → Go client tool), exactly as the
angular generator drives `ng-openapi-gen`. It does not hand-write a code generator.

**Non-goals** (later WS-024 phases / deferred): the CLI, the Terraform provider, and any
per-app client repo convention. This feature emits + verifies + (dry-run) publishes the Go
client artifact; where a consumer hosts it is a P1/P2 decision. No Python/Java client. No
change to the TS/angular path's observable output.

## Ground truth (verified)

- `client.Generator` (`internal/client/client.go:55-63`): `Name() string`,
  `Generate(ctx, GenerateContext) (Result, error)`. `GenerateContext` (L17-38):
  `SpecPath, OutputDir, PackageName, Scope, PackageVersion`. `Result` (L42-52):
  `PackageDir, PackageName, Files`. Registry `internal/client/registry.go` (`Register` in each
  generator's `init()`); help lists `client.Names()` at `cmd/apx/commands/client.go:48`.
- Template to mirror: `internal/client/angular.go` — pinned tool version const (`angular.go:17`),
  `exec.LookPath` preflight + `exec.CommandContext(ctx, "npx", "--yes", "ng-openapi-gen@"+ver, …)`
  (L31-58), spec abs+stat check (L35-41), package scaffolding `writePackageJSON`/`tsconfigJSON`/
  `renderReadme` (L153-262), barrel fixup `ensureProviderExported` (L271-293).
- Build/publish is hardcoded npm in the command layer, NOT abstracted:
  `buildClientPackage` (`cmd/apx/commands/client.go:358-383`, `npm install`+`npm run build`),
  `publishClientPackage` (L388-408, `npm publish` + `--dry-run`). Abstraction precedent to
  mirror: `internal/language/plugin.go:68-96` (`Scaffolder`/`PostGenHook`/… optional interfaces
  consumed by type-assertion).
- `apx client` commands (`cmd/apx/commands/client.go`): `generate`/`publish`; shared flags
  `--input --from --output --scope --package --generator`(default `typescript-angular`, L18)
  `--version` (`addClientResolutionFlags` L33-41); generate adds `--build --clean`; publish adds
  `--dry-run`(default true) `--record`. Resolves `gen := client.Get(name)` (L140) then
  `gen.Generate(...)`.
- Spec resolution reused unchanged: `MaterializeSpec` (`internal/config/depsrc.go:39`) +
  `resolveFromDependency` (`client.go:294-319`) — the `--from` hot-loop for unreleased specs.
- Artifact recording is a free-form `Type string` (`internal/publisher/record.go:69-74`,
  `AddArtifact`); `go-module` already used (`release.go:1133-1140`); Go modules publish by
  **git tag** (`internal/publisher/tags.go`), not a registry — no `npm publish` analog.
- Fully greenfield: zero existing `oapi-codegen`/`go-client` references in the repo.

## Requirements

- **FR-1**: A new generator (`internal/client/goclient.go`, `package client`) MUST implement
  `client.Generator` with `Name() == "go"` and self-register in `init()`. `apx client generate
  --generator go` and `client.Names()` in help MUST list it.
- **FR-2**: `Generate` MUST orchestrate `oapi-codegen` (the standard tool), invoked with a
  **pinned** version via `go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@<ver>`
  (const, overridable later — the Go analog of angular's `npx --yes …@ver`), so no pre-installed
  binary is required. Preflight MUST verify `go` is on PATH and fail loud with an actionable
  message if the tool cannot be fetched/run.
- **FR-3**: `Generate` MUST read the OpenAPI spec at `gc.SpecPath` (abs + existence check, as
  angular does), configure `oapi-codegen` to emit **types + a typed client**, and produce a
  compilable Go module in `gc.OutputDir`: the generated `<pkg>.gen.go`, a `go.mod` (module path
  from the resolution flags — see FR-4), and a short `doc.go`/README. `Result.Files` MUST list
  them.
- **FR-4**: For `--generator go`, `--package` is interpreted as the **Go module path** (e.g.
  `github.com/acme/widgets-client`) and `--scope` is ignored (npm-only). If `--package` is empty,
  derive a sensible module path from the spec/api id and document the derivation. (No change to
  how angular reads `--scope`/`--package`.)
- **FR-5**: The enriched-OpenAPI signals from feature 044 MUST be honored where `oapi-codegen`
  supports them natively: `enum` → Go enum/const types; `required` → non-omitempty/required
  fields; `readOnly`/`writeOnly` respected per `oapi-codegen` semantics. `x-aip-*` extensions
  MUST NOT break generation (unknown extensions ignored) — the generated client works on any
  valid OpenAPI, and is *better-typed* on an enriched one.
- **FR-6**: Add optional `Builder` and `Publisher` interfaces to `internal/client` (mirroring
  `internal/language` optional interfaces, consumed by type-assertion). `buildClientPackage` /
  `publishClientPackage` in the command layer MUST be refactored to: use the generator's
  `Builder`/`Publisher` if it implements them, else fall back to the existing npm path. The go
  generator's `Builder` runs `go build ./...` in the output module (compile-verify);
  its `Publisher` records a `go-module` artifact and performs a git-tag-based publish, honoring
  `--dry-run` (default true).
- **FR-7 (backward-compat, load-bearing)**: The TS/angular `generate` and `publish` observable
  behavior MUST be **unchanged** — same generated output for the same input, same npm build/publish
  calls. A test asserts the angular generator still registers and its build/publish still routes to
  the npm path. (WS-020 must not regress; memory notes WS-020 hardened to "backward-compat
  byte-identical".)
- **FR-8**: `de api publish` (devedge) gains a thin driver so the Go client is drivable
  end-to-end (`de api publish --client go` or equivalent). This MAY be a minimal shell-out change
  reusing the existing `de api publish --client` plumbing; the substantive generator lives in apx.
  (Small; if it grows, split to P1.)

## Acceptance Criteria

- **AC-1**: `apx client generate --generator go --input <enriched toy openapi> --package
  github.com/example/toy-client --output <dir>` emits a module that `go build ./...` compiles
  clean (verified by the go `Builder`, not just asserted).
- **AC-2**: The generated client has a typed method for each toy RPC (Create/Get/List/Update/
  Delete) and Go types for each resource; an `allowed_values`/`enum` field becomes a Go
  enum/const type; a `required` field is generated as required (non-omitempty).
- **AC-3**: Feeding the **enriched** toy spec (with `x-aip-*` extensions present) generates
  without error — unknown extensions are ignored, no crash, no lossy failure.
- **AC-4**: `apx client publish --generator go --dry-run` records a `go-module` artifact
  (`internal/publisher/record`) and completes without a write token (dry-run verified, matching
  the WS-020 publish posture).
- **AC-5 (backward-compat)**: `apx client generate/publish` with the default
  `typescript-angular` generator produces the same output and takes the same npm build/publish
  path as before this feature (regression test green).
- **AC-6**: The full apx test suite + build are green; `apx` help lists `go` among client
  generators.

## Failure Modes (fail-loud, not silent)

- **FM-1 — Tooling unavailable**: `go` not on PATH, or `oapi-codegen@<ver>` cannot be fetched/run
  → `Generate` returns a clear error naming the tool + how to fix; no partial/broken module left
  claimed as success.
- **FM-2 — Missing/invalid spec**: `gc.SpecPath` absent or not valid OpenAPI → fail loud (reuse
  the angular stat/abs pattern; surface `oapi-codegen`'s parse error tail).
- **FM-3 — Uncompilable output**: if the generated client does not `go build`, the `Builder`
  returns a non-nil error and `publish` MUST NOT proceed (never publish a broken client).
- **FM-4 — Backward-compat break**: any change to the angular path's output/build/publish is a
  hard test failure (AC-5 guards it).

## Out of scope (later / deferred)

- The CLI (P1), Terraform provider (P2), and a per-app Go-client *repo* convention.
- A full git-tag publish to a public module (dry-run/record is the keystone bar; real publish is
  gated on repo/token like WS-020's TS publish).
- Python/Java clients; a protoc-based Go client (oapi-codegen from OpenAPI is the ratified path,
  D1 — one enriched-OpenAPI interchange).
