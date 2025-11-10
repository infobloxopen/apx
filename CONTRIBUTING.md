# Contributing to APX

This document explains how to contribute to the APX CLI tool and how to run the comprehensive test suite.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Making Changes](#making-changes)
- [Testing](#testing)
- [Submitting Changes](#submitting-changes)
- [Code Style](#code-style)
- [Architecture](#architecture)

## Code of Conduct

This project adheres to the [Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code.

## Getting Started

1. Fork the repository on GitHub
2. Clone your fork: `git clone https://github.com/YOUR_USERNAME/apx.git`
3. Add upstream remote: `git remote add upstream https://github.com/infobloxopen/apx.git`

## Development Setup

### Prerequisites

- Go 1.25 or later
- Git
- Make

### Install Dependencies

```bash
# Install external tools
./scripts/install-tools.sh

# Install Go dependencies
go mod download
```

### Build the Project

```bash
make build
```

### Run Tests

```bash
# Unit tests
make test

# Integration tests
make test-integration

# All tests
make test-all
```

## Making Changes

### Branch Naming

Use descriptive branch names:

- `feature/add-avro-support`
- `fix/breaking-change-detection`
- `docs/update-readme`

### Commit Messages

Follow the [Conventional Commits](https://conventionalcommits.org/) specification:

```
type(scope): description

[optional body]

[optional footer]
```

Examples:
- `feat(semver): add support for pre-release versions`
- `fix(config): handle missing config file gracefully`
- `docs(readme): update installation instructions`

### Types

- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring
- `test`: Adding or updating tests
- `chore`: Maintenance tasks

## Testing

### Unit Tests

Write unit tests for all new functionality:

```bash
go test ./internal/...
```

### Integration Tests

Integration tests validate end-to-end functionality:

```bash
go test ./tests/integration/...
```

### Test Coverage

Maintain high test coverage:

```bash
make coverage
```

### Test Fixtures

Place test fixtures in `testdata/` directories:

```
internal/protoval/testdata/
├── valid/
│   ├── user.proto
│   └── payment.proto
└── invalid/
    └── broken.proto
```

## Submitting Changes

### Pull Request Process

1. Update your fork:
   ```bash
   git fetch upstream
   git rebase upstream/main
   ```

2. Create a feature branch:
   ```bash
   git checkout -b feature/your-feature
   ```

3. Make your changes and commit:
   ```bash
   git add .
   git commit -m "feat(scope): your change description"
   ```

4. Push to your fork:
   ```bash
   git push origin feature/your-feature
   ```

5. Create a Pull Request on GitHub

### Pull Request Guidelines

- Fill out the PR template completely
- Include tests for new functionality
- Update documentation as needed
- Ensure CI checks pass
- Request review from maintainers

### PR Title Format

Use the same format as commit messages:

```
feat(semver): add support for pre-release versions
```

## Code Style

### Go Code Style

Follow standard Go conventions:

- Use `gofmt` for formatting
- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use meaningful variable and function names
- Add comments for exported functions

### Code Analysis

Run code analysis before submitting:

```bash
make lint  # Runs go vet
```

### Import Organization

Organize imports in groups:

```go
import (
    // Standard library
    "context"
    "fmt"
    "os"

    // Third-party packages
    "github.com/urfave/cli/v2"
    "gopkg.in/yaml.v3"

    // Internal packages
    "github.com/infobloxopen/apx/internal/config"
    "github.com/infobloxopen/apx/internal/ui"
)
```

## Architecture

### Project Structure

```
apx/
├── cmd/apx/               # CLI entry point
├── internal/              # Internal packages
│   ├── config/           # Configuration management
│   ├── execx/            # External tool execution
│   ├── protoval/         # Protocol Buffer validation
│   ├── openapival/       # OpenAPI validation
│   ├── policy/           # Policy enforcement
│   ├── semver/           # Semantic versioning
│   ├── gitx/             # Git operations
│   ├── gen/              # Code generation
│   ├── catalog/          # Module discovery
│   └── ui/               # User interface
├── scripts/              # Helper scripts
└── testdata/             # Test fixtures
```

### Key Principles

1. **Separation of Concerns**: Each package has a single responsibility
2. **Dependency Injection**: Pass dependencies explicitly
3. **Error Handling**: Return descriptive errors
4. **Testability**: Design for easy testing
5. **Configuration**: Make behavior configurable

### Adding New Schema Support

To add support for a new schema format:

1. Create a validation package in `internal/`
2. Implement the `Validator` interface
3. Add configuration options
4. Update module discovery logic
5. Add tests and documentation

### Error Handling

Use structured errors with context:

```go
if err != nil {
    return fmt.Errorf("failed to parse config file %s: %w", path, err)
}
```

### Logging

Use the internal UI package for consistent output:

```go
ui.Debug("Processing file %s", filename)
ui.Info("Validation completed successfully")
ui.Warning("Deprecated option used: %s", option)
ui.Error("Validation failed: %v", err)
```

## Documentation

### Code Documentation

- Document all exported functions and types
- Use examples in godoc comments
- Keep comments up to date

### User Documentation

- Update README.md for user-facing changes
- Add examples for new features
- Update configuration reference

### API Documentation

Generate API documentation:

```bash
make docs
```

## Release Process

Releases are automated via GitHub Actions:

1. Create a git tag: `git tag v1.2.3`
2. Push the tag: `git push origin v1.2.3`
3. GitHub Actions builds and publishes releases

## Getting Help

- Join our [Discussions](https://github.com/infobloxopen/apx/discussions)
- Open an [Issue](https://github.com/infobloxopen/apx/issues)
- Review existing documentation

## Recognition

Contributors are recognized in:

- GitHub contributors list
- Release notes
- Project documentation

Thank you for contributing to APX! 🚀