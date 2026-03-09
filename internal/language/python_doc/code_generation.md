### Python

APX scaffolds Python namespace packages with PEP 625 distribution names
and `pkgutil`-based namespace packages.

```bash
apx gen python
```

This creates overlays in `.apx/overlays/python/` with:
- `pyproject.toml` — distribution metadata with the derived dist name
- `__init__.py` hierarchy using `pkgutil.extend_path` for namespace packages
- Generated protobuf/schema code

**Key characteristics:**
- Distribution name: `{org}-{domain}-{name}-{line}` (e.g. `acme-payments-ledger-v1`)
- Import path: `{org}_apis.{domain}.{name}.{line}` (e.g. `acme_apis.payments.ledger.v1`)
- Editable install via `apx link python` (runs `pip install -e`)
- Requires `org` in `apx.yaml` for package naming
