# Implementation Plan: 010 Go client generator

**Spec**: `spec.md`. **Branch**: `010-go-client-generation`.

## Architecture

Add a third `client.Generator` alongside angular, and lift the hardcoded npm build/publish
behind optional interfaces so the go generator supplies its own compile-verify + git-tag publish.

```
internal/client/
  client.go     ── Generator (existing) + NEW optional Builder, Publisher interfaces
  registry.go   ── (unchanged) Register/Get/Names
  angular.go    ── (unchanged output) ng-openapi-gen orchestration  [template]
  goclient.go   ── NEW: goGenerator implements Generator (+ Builder + Publisher)
                   orchestrates `go run oapi-codegen@<pinned>` → types+client → go.mod/doc.go

cmd/apx/commands/client.go
  buildClientPackage/publishClientPackage ── refactor to: if gen implements Builder/Publisher
                                             use it, else the existing npm path (angular unchanged)
```

- **oapi-codegen orchestration** mirrors angular's shape: pinned version const; preflight
  `exec.LookPath("go")`; write a temp `oapi-codegen` config YAML (package name, `generate:
  [types, client]`); `exec.CommandContext(ctx, "go", "run",
  "github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@"+ver, "-config", cfg, specAbs)`;
  capture output; error tail via the existing `tail(...)` helper.
- **go.mod emission**: `module <--package>` + a `go` directive + the oapi-codegen runtime
  require (`github.com/oapi-codegen/runtime`). `--package` = module path for `--generator go`.
- **Builder** (`go build ./...` in OutputDir) and **Publisher** (record `go-module` artifact via
  `internal/publisher/record`; git-tag publish reusing `internal/publisher/tags.go` precedent;
  honor `--dry-run`). Both optional interfaces, type-asserted in the command layer.
- **de driver (FR-8)**: thin — extend the existing `de api publish --client` shell-out to pass
  `--generator go`. Kept minimal; substantive logic stays in apx.

## Test strategy (functional + e2e)

- Unit: goGenerator config/args construction; `--package`→module-path handling; unknown
  `x-aip-*` extension tolerated.
- **e2e (real tool + real compile)**: generate from a checked-in enriched OpenAPI fixture (the
  toy spec shape from feature 044; if 044's golden isn't vendored into apx, add a minimal
  enriched fixture under `testdata/`), run the go `Builder` → assert `go build` clean and the
  expected typed methods/enum/required exist in the output (grep/parse the generated file).
  This exercises the real `oapi-codegen` and real `go build`, not a stub.
- Backward-compat: existing angular tests (`internal/client/client_test.go`) stay green; a test
  asserts angular build/publish still routes to npm (Builder/Publisher NOT implemented by
  angular, so the fallback path is taken).
- Dry-run publish records a `go-module` artifact.

## Scope gate

Everything traces to an FR. No CLI/TF/Python. No real token-gated publish (dry-run/record is the
bar). The Builder/Publisher refactor is the minimum to de-hardcode npm — no speculative backends.
`oapi-codegen` is orchestrated, not reimplemented (apx charter).
