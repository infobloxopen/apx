### Go Development Loop

1. `apx add <api-id>` — add dependency
2. `apx gen go` — generate Go code with canonical imports
3. `apx sync` — update `go.work` to include overlays
4. Edit and test locally — Go toolchain resolves via `go.work`
5. `apx unlink <api-id>` — switch to released module
