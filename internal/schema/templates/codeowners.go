package templates

import "fmt"

// GenerateCodeowners generates a CODEOWNERS file for the canonical repo
func GenerateCodeowners(org string) string {
	return fmt.Sprintf(`# Code ownership for API schemas
# See: https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/customizing-your-repository/about-code-owners

# Default owners for all files
* @%s/api-owners

# Proto schemas
/proto/ @%s/proto-owners

# OpenAPI schemas
/openapi/ @%s/openapi-owners

# Avro schemas
/avro/ @%s/avro-owners

# JSON Schema
/jsonschema/ @%s/jsonschema-owners

# Parquet schemas
/parquet/ @%s/parquet-owners
`, org, org, org, org, org, org)
}
