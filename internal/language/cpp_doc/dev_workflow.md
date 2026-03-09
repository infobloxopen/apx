### C++ Development Loop

1. Add `{org}-{domain}-{name}-{line}-proto` to your `conanfile`
2. `conan install .` — resolve dependencies
3. Include `{org}/{domain}/{name}/{line}` headers in C++ code
4. Local development via `conan editable add` for overlay resolution
