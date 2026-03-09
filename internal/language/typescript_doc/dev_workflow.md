### TypeScript Development Loop

1. `npm install @{org}/{domain}-{name}-{line}-proto` — add dependency
2. `import { ... } from '@{org}/{domain}-{name}-{line}-proto'` — import in code
3. Local dev via `npm link` for development iteration
4. `apx unlink <api-id>` — switch back to released npm package
