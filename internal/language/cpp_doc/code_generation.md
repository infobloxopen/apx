### C++

C++ APIs are published as Conan packages using conventional naming derived
from the API identity.

Conan package references are derived as:
- Package name: `{org}-{domain}-{name}-{line}-proto`
- C++ namespace: `{org}::{domain}::{name}::{line}`

**Key characteristics:**
- Conan reference: `{org}-{domain}-{name}-{line}-proto`
- C++ namespace: `{org}::{domain}::{name}::{line}`
- Consumer adds dependency to `conanfile.txt` or `conanfile.py`
- Local dev via `conan editable add` for development iteration
- Requires `org` in `apx.yaml` for Conan reference derivation
