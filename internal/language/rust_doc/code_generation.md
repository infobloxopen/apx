### Rust

Rust APIs are published as Cargo crates using conventional naming derived
from the API identity.

Crate names are derived as:
- Crate: `{org}-{domain}-{name}-{line}-proto`
- Rust module: `{org}_{domain}::{name}::{line}`

**Key characteristics:**
- Crate name: `{org}-{domain}-{name}-{line}-proto`
- Rust module path: `{org}_{domain}::{name}::{line}`
- Consumer adds `[dependencies]` entry to `Cargo.toml`
- Local dev via path dependencies for development iteration
- Requires `org` in `apx.yaml` for crate name derivation
