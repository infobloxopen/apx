# APX Interactive Initialization

The `apx init` command now supports both interactive and non-interactive modes for setting up new schema modules with smart defaults and customizable configuration.

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

When run without `--non-interactive`, APX guides you through configuration. You can run it with or without arguments:

```bash
# Full interactive mode - prompts for everything
apx init

# Interactive with pre-specified schema type and module
apx init proto my.service.v1

# Output:
🚀 Welcome to APX initialization!
Let's set up your configuration with some questions...

? What type of schema do you want to create? proto
? Module path/name: (my.service.v1)

📋 Schema Configuration:
   Type: proto
   Module: my.service.v1

? Organization name: (infobloxopen) 
? Repository name: (apx) 
? Target languages (select all that apply): [go, python, java]

Configuration complete! 🎉
```

### ⚡ Non-Interactive Mode

For automation or when you want to use defaults. **Note: Non-interactive mode requires both schema type and module path arguments.**

```bash
# Use all defaults (requires both arguments)
apx init --non-interactive proto my.service.v1

# Override specific defaults
apx init --non-interactive \
  --org "mycompany" \
  --repo "awesome-apis" \
  --languages "go,python,java" \
  proto my.service.v1

# ❌ This will fail - requires both arguments in non-interactive mode
apx init --non-interactive
```

## Command Options

| Flag | Description | Example |
|------|-------------|---------|
| `--non-interactive` | Skip interactive prompts | `--non-interactive` |
| `--org VALUE` | Set organization name | `--org "acme-corp"` |
| `--repo VALUE` | Set repository name | `--repo "my-apis"` |
| `--languages VALUE` | Set target languages (comma-separated) | `--languages "go,python,java"` |

## Examples

### Example 1: Full Interactive Mode

```bash
# No arguments - completely interactive
apx init

# Will prompt for:
# 1. Schema type (proto, openapi, avro, jsonschema, parquet)  
# 2. Module path/name (with smart defaults based on schema type)
# 3. Organization name (auto-detected from directory)
# 4. Repository name (auto-detected from directory)
# 5. Target languages (auto-detected from project files)
```

### Example 2: Smart Defaults in GitHub Repository

```bash
# In /Users/dev/go/src/github.com/infobloxopen/my-apis/
apx init --non-interactive proto user.v1

# Generates apx.yaml with:
# org: infobloxopen
# repo: my-apis
# languages: [go] (detected from go.mod)
```

### Example 2: Multi-Language Project

```bash
# Custom configuration for multiple languages
apx init --non-interactive \
  --org "tech-startup" \
  --repo "platform-apis" \
  --languages "go,python,java" \
  proto platform.users.v1
```

### Example 3: Different Schema Types

```bash
# Interactive mode - will prompt for schema type
apx init

# Non-interactive with specific schema types
apx init --non-interactive openapi my-rest-api
apx init --non-interactive jsonschema my-data-schema  
apx init --non-interactive avro my-event-schema

# Partial interactive - specify schema, prompted for module path and config
apx init proto
# Will prompt for module path and configuration
```

## Generated Configuration

The generated `apx.yaml` includes:

- ✅ **Custom org/repo names** (from flags or smart detection)
- ✅ **Language-specific configurations** with appropriate tools and plugins
- ✅ **Sensible policy defaults** for each schema type
- ✅ **Modern tool versions** (buf, spectral, etc.)
- ✅ **Best practice settings** for CI/CD and publishing

### Go Configuration
```yaml
language_targets:
  go:
    enabled: true
    plugins:
      - name: protoc-gen-go
        version: v1.64.0
      - name: protoc-gen-go-grpc
        version: v1.5.0
```

### Python Configuration
```yaml
  python:
    enabled: true
    tool: grpcio-tools
    version: 1.64.0
```

### Java Configuration
```yaml
  java:
    enabled: true
    plugins:
      - name: protoc-gen-grpc-java
        version: 1.68.1
```

## Advanced Usage

### Usage Patterns

APX init supports several usage patterns:

```bash
# 🎯 Full interactive (recommended for first-time users)
apx init

# 🚀 Quick interactive with schema type pre-selected  
apx init proto

# ⚡ Non-interactive automation (requires both arguments)
apx init --non-interactive proto my.service.v1

# 🔧 Non-interactive with custom settings
apx init --non-interactive --org "company" --repo "apis" proto service.v1
```

### Flag Order Matters
Due to urfave/cli parsing, flags must come before arguments:

```bash
# ✅ Correct
apx init --non-interactive --org "company" proto service.v1

# ❌ Incorrect  
apx init proto service.v1 --non-interactive --org "company"
```

### Interactive Environment Detection
Interactive mode is automatically disabled in:
- CI environments (`CI=true`)
- Non-TTY terminals (`TERM=dumb`)

You can force non-interactive mode with `--non-interactive`.

## Default Module Path Suggestions

When running in full interactive mode (`apx init`), APX suggests sensible defaults based on the schema type:

| Schema Type | Default Module Path | Use Case |
|-------------|-------------------|----------|
| `proto` | `com.example.service.v1` | gRPC services, protobuf APIs |
| `openapi` | `my-api` | REST APIs, HTTP services |
| `avro` | `com.example.events` | Event streaming, data pipelines |
| `jsonschema` | `com.example.schema` | Data validation, configuration |
| `parquet` | `com.example.data` | Data analytics, columnar storage |

These defaults can be customized during the interactive prompts or overridden with command-line arguments.

## Integration with Existing Workflows

The enhanced `apx init` command is perfect for:

- **Local development**: Interactive setup with intelligent defaults
- **CI/CD pipelines**: Non-interactive mode with explicit configuration  
- **Team onboarding**: Consistent project setup across different environments
- **Polyglot projects**: Automatic multi-language support detection
- **Exploratory development**: Quick schema type selection without pre-planning