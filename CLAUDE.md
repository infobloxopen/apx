# APX — Claude Code Instructions

## Critical Rules

### specs/ directory is READ-ONLY
**NEVER edit any file under `specs/` under any circumstances.**

The `specs/` directory contains feature specifications that are instructions/source material only. They are not code targets. Do not modify, update, reformat, or touch them in any way unless the user explicitly references running a spec kit agent for a specific spec.

If you find specs/ files modified, run: `git checkout -- specs/`

### Build
```bash
GOTOOLCHAIN=auto go build ./...
GOTOOLCHAIN=auto go test ./...
```

### Vocabulary
- `apx release` is the release pipeline (NOT `apx publish` — that was removed)
- Lifecycle values: `experimental`, `beta` (canonical), `stable`, `deprecated`, `sunset`
- `preview` is accepted as a backward-compatible alias for `beta` only
