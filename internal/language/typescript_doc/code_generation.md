### TypeScript

TypeScript APIs are published as scoped npm packages with a `-proto` suffix.

```bash
npm install @{org}/{domain}-{name}-{line}-proto
```

**Key characteristics:**
- npm package: `@{org}/{domain}-{name}-{line}-proto`
- Import path equals the npm package name
- Consumer installs via `npm install` or `yarn add`
- Local dev via `npm link` for development iteration
- Requires `org` in `apx.yaml` for package scoping
