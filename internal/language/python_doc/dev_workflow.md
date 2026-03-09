### Python Development Loop

1. `apx add <api-id>` — add dependency
2. `apx gen python` — generate Python code with namespace packages
3. `apx link python` — run `pip install -e` in active virtualenv
4. `from {org}_apis.{domain}.{name}.{line} import ...` — import generated code
5. `apx unlink <api-id>` — switch to released package via `pip install`
