# APX Interactive Initialization

The `apx init` command supports both interactive and non-interactive modes for setting up new API repositories with smart defaults.

## Features

### 🤖 Smart Defaults Detection

APX automatically detects project context and suggests intelligent defaults:

- **Organization Name**: Extracted from GitHub paths (e.g., `/Users/user/go/src/github.com/company/repo` → `company`)
- **Repository Name**: Extracted from current directory name
- **Target Languages**: Auto-detected from project files:
  - Go: presence of `go.mod`
  - Python: presence of `requirements.txt` or `pyproject.toml`
  - Java: presence of `pom.xml` or `build.gradle`

### 🎯 Interactive Mode (Default)

When run without `--non-interactive`, APX guides you through configuration:

```bash
# Initialize a canonical repo interactively
apx init canonical

# Initialize an app module interactively
apx init app internal/apis/proto

# Example interactive session for apx init canonical:
? Organization name (mycompany):
? Repository name (apis):

✓ Created buf.yaml
✓ Created catalog.yaml
✓ Initialized git repository
```

### ⚡ Non-Interactive Mode

For automation and CI/CD. Provide all required flags explicitly:

```bash
# Initialize canonical repo non-interactively
apx init canonical --org mycompany --repo apis --non-interactive

# Initialize app repo non-interactively
apx init app --org mycompany --repo myapp --non-interactive internal/apis/proto
```

## Command Options

**`apx init canonical`**

| Flag | Description |
|------|-------------|
| `--org VALUE` | GitHub organization name |
| `--repo VALUE` | Repository name (default: `apis`) |
| `--skip-git` | Skip `git init` |
| `--non-interactive` | Disable interactive prompts |

**`apx init app`**

| Flag | Description |
|------|-------------|
| `--org VALUE` | GitHub organization name |
| `--repo VALUE` | App repository name |
| `--non-interactive` | Disable interactive prompts |

## Examples

### Example 1: Interactive canonical repo setup

```bash
# In a fresh directory
apx init canonical
# Prompts for: org, repo name
```

### Example 2: Non-interactive canonical repo (CI/CD)

```bash
apx init canonical --org mycompany --repo apis --non-interactive
```

### Example 3: Interactive app module setup

```bash
# In your app repo
apx init app internal/apis/proto
# Prompts for: org, repo
```

### Example 4: Non-interactive app module setup

```bash
apx init app --org mycompany --repo myapp --non-interactive internal/apis/proto
```

## Generated Configuration

`apx init app` creates `apx.yaml` in the repo root:

```yaml
version: 1
org: mycompany
repo: myapp
module_roots:
  - internal/apis/proto
```

`apx init canonical` creates `buf.yaml`, `catalog.yaml`, `CODEOWNERS`, and CI workflow templates.

## Advanced Usage

### Usage Patterns

```bash
# Interactive canonical repo setup
apx init canonical

# Interactive app module setup
apx init app internal/apis/proto

# Non-interactive with all flags
apx init canonical --org mycompany --repo apis --non-interactive
apx init app --org mycompany --repo myapp --non-interactive internal/apis/proto
```

### Interactive Environment Detection
Interactive mode is automatically disabled in:
- CI environments (`CI=true`)
- Non-TTY terminals (`TERM=dumb`)

You can force non-interactive mode with `--non-interactive`.

## Module Path Conventions

The `<module-path>` argument to `apx init app` should point to the root of your schema tree within the app repo:

| Schema Format | Typical Module Path |
|---|---|
| Protocol Buffers | `internal/apis/proto` |
| OpenAPI | `internal/apis/openapi` |
| Avro | `internal/apis/avro` |
| JSON Schema | `internal/apis/jsonschema` |
| Parquet | `internal/apis/parquet` |

APX discovers schema files by walking the directory tree starting at this path.

## Integration with Existing Workflows

The enhanced `apx init` command is perfect for:

- **Local development**: Interactive setup with intelligent defaults
- **CI/CD pipelines**: Non-interactive mode with explicit configuration  
- **Team onboarding**: Consistent project setup across different environments
- **Polyglot projects**: Automatic multi-language support detection
- **Exploratory development**: Quick schema type selection without pre-planning