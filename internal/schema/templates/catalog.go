package templates

import "fmt"

// GenerateCatalog generates a catalog.yaml for the canonical repo
func GenerateCatalog(org, repo string) string {
	return fmt.Sprintf(`version: 1
org: %s
repo: %s
modules: []
`, org, repo)
}

// GenerateReadme generates a README.md for the canonical repo
func GenerateReadme(org, repo string) string {
	return fmt.Sprintf(`# %s/%s

Canonical API Schema Repository

## Overview

This repository contains the canonical API schemas for %s, organized by schema format:

- **proto/** - Protocol Buffer definitions
- **openapi/** - OpenAPI specifications
- **avro/** - Avro schemas
- **jsonschema/** - JSON Schema definitions
- **parquet/** - Parquet schema definitions

## Structure

Each schema format has its own directory. Schemas are organized by domain/service within each directory.

## Usage

### For Producers (Publishing APIs)

1. Create your schema in the appropriate format directory
2. Run lint checks: `+"`apx lint <path/to/schema>`"+`
3. Check for breaking changes: `+"`apx breaking <path/to/schema> --against <previous-version>`"+`
4. Publish: `+"`apx publish <path/to/schema> --version <semver>`"+`

### For Consumers (Using APIs)

1. Search for schemas: `+"`apx search <query>`"+`
2. Add dependency: `+"`apx add <schema-module>`"+`
3. Generate code: `+"`apx gen go`"+` (or python, java)
4. Sync overlays: `+"`apx sync`"+`

## Branch Protection

This repository should have the following branch protection rules on `+"`main`"+`:

- Require pull request reviews
- Require status checks (lint, breaking change detection)
- Require CODEOWNERS review
- Restrict who can push to main

## Contributing

See CONTRIBUTING.md for contribution guidelines.
`, org, repo, org)
}
