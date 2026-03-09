### Go

APX manages Go overlays through `go.work` files, providing seamless local
development without `replace` directives.

```bash
apx gen go
```

This creates local overlays in `.apx/overlays/go/` and updates `go.work`
to include them. Generated code uses canonical import paths so it works
identically in local development and after release.

**Key characteristics:**
- Module path follows Go major version conventions (no suffix for v0/v1, `/vN` for v2+)
- Import path always includes the line version (`/v1`, `/v2`, etc.)
- `go.work` overlays enable local development without `replace` directives
- `apx sync` keeps `go.work` in sync after adding/removing dependencies
