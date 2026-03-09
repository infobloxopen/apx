### Rust Development Loop

1. Add `{org}-{domain}-{name}-{line}-proto` to `[dependencies]` in `Cargo.toml`
2. `cargo build` — compile with generated code
3. `use {org}_{domain}::{name}::{line}::*` in Rust code
4. Local development via path dependency for overlay resolution
