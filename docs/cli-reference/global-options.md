# Global Options

These flags are available on every APX command.

## Flags

| Flag | Shorthand | Type | Default | Description |
|------|-----------|------|---------|-------------|
| `--quiet` | `-q` | bool | `false` | Suppress non-essential output |
| `--verbose` | | bool | `false` | Enable verbose output with additional detail |
| `--json` | | bool | `false` | Format output as JSON (supported by most commands) |
| `--no-color` | | bool | `false` | Disable colored terminal output |
| `--config` | | string | `apx.yaml` | Path to the APX configuration file |

---

## `--quiet`

Suppresses informational messages, leaving only errors and the primary output. Useful in CI pipelines where you only want actionable output.

```bash
apx lint --quiet
apx release prepare proto/payments/ledger/v1 --version v1.0.0 -q
```

---

## `--verbose`

Enables additional diagnostic output, including resolved tool paths, config values, and intermediate steps.

```bash
apx lint --verbose
apx gen go --verbose
```

`--quiet` and `--verbose` are mutually exclusive. If both are set, `--quiet` wins.

---

## `--json`

Formats output as JSON for machine consumption. Supported by commands that produce structured data:

```bash
apx --json show proto/payments/ledger/v1
apx --json search payments
apx --json config validate
apx --json release inspect
apx --json release history proto/payments/ledger/v1
apx --json semver suggest --against HEAD^
apx --json inspect identity proto/payments/ledger/v1
apx --json explain go-path proto/payments/ledger/v1
```

:::{note}
`--json` is a persistent flag on the root command. It can be placed before or after the subcommand: `apx --json show ...` or `apx show --json ...` both work.
:::

---

## `--no-color`

Disables ANSI color codes in terminal output. Automatically enabled when stdout is not a TTY (e.g. piped output or CI environments).

```bash
apx lint --no-color
apx release prepare --no-color proto/payments/ledger/v1 --version v1.0.0
```

---

## `--config`

Specifies a custom path to the APX configuration file. Defaults to `apx.yaml` in the current directory.

```bash
apx lint --config /path/to/custom/apx.yaml
apx config validate --config staging-apx.yaml
```

This is useful when:
- Running APX from a directory other than the repo root
- Managing multiple configurations (e.g. staging vs production)
- Testing config changes without modifying the default file

---

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | General error |
| `6` | Configuration validation error |

The exit code `6` is returned specifically by `apx config validate` when the configuration is invalid, allowing CI scripts to distinguish config errors from other failures.

---

## Environment Variables

| Variable | Equivalent flag | Description |
|----------|----------------|-------------|
| `APX_CONFIG` | `--config` | Path to configuration file |
| `APX_VERBOSE` | `--verbose` | Enable verbose output |
| `APX_QUIET` | `--quiet` | Suppress non-essential output |
| `APX_JSON` | `--json` | Format output as JSON |
| `HTTP_PROXY` / `HTTPS_PROXY` | — | Proxy settings for network operations |
| `NO_COLOR` | `--no-color` | Disable color output (standard convention) |
