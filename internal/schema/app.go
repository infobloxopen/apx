package schema

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// AppScaffolder creates application repository structure with schema modules
type AppScaffolder struct {
	modulePath string
	org        string
}

// NewAppScaffolder creates a new app scaffolder
func NewAppScaffolder(modulePath, org string) *AppScaffolder {
	return &AppScaffolder{
		modulePath: modulePath,
		org:        org,
	}
}

// Generate creates the app repository structure
func (s *AppScaffolder) Generate(baseDir string) error {
	if s.modulePath == "" {
		return fmt.Errorf("module path is required")
	}
	if s.org == "" {
		return fmt.Errorf("organization is required")
	}

	// Create module directory
	moduleDir := filepath.Join(baseDir, s.modulePath)
	if err := os.MkdirAll(moduleDir, 0755); err != nil {
		return fmt.Errorf("failed to create module directory: %w", err)
	}

	// Detect format from path
	format := detectFormatFromPath(s.modulePath)

	// Generate apx.yaml at workspace root
	if err := s.generateApxYaml(baseDir, format); err != nil {
		return fmt.Errorf("failed to generate apx.yaml: %w", err)
	}

	// Generate example schema file based on format
	if err := s.generateExampleSchema(moduleDir, format); err != nil {
		return fmt.Errorf("failed to generate example schema: %w", err)
	}

	// Generate root .gitignore if it doesn't exist
	gitignorePath := filepath.Join(baseDir, ".gitignore")
	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		if err := s.generateGitignore(gitignorePath); err != nil {
			return fmt.Errorf("failed to generate .gitignore: %w", err)
		}
	}

	// Generate buf.work.yaml for workspace configuration
	bufWorkPath := filepath.Join(baseDir, "buf.work.yaml")
	if _, err := os.Stat(bufWorkPath); os.IsNotExist(err) {
		if err := s.generateBufWorkYaml(bufWorkPath, s.modulePath); err != nil {
			return fmt.Errorf("failed to generate buf.work.yaml: %w", err)
		}
	}

	return nil
}

func detectFormatFromPath(path string) string {
	if strings.Contains(path, "/proto/") {
		return "proto"
	}
	if strings.Contains(path, "/openapi/") {
		return "openapi"
	}
	if strings.Contains(path, "/avro/") {
		return "avro"
	}
	if strings.Contains(path, "/jsonschema/") {
		return "jsonschema"
	}
	if strings.Contains(path, "/parquet/") {
		return "parquet"
	}
	return "proto" // default
}

func (s *AppScaffolder) generateApxYaml(baseDir, format string) error {
	moduleName := extractModuleName(s.modulePath, format)

	content := fmt.Sprintf(`kind: %s
module: %s
org: %s
version: v1
`, format, moduleName, s.org)

	apxYamlPath := filepath.Join(baseDir, "apx.yaml")
	return os.WriteFile(apxYamlPath, []byte(content), 0644)
}

func extractModuleName(path, format string) string {
	// Extract module name from path
	// Example: internal/apis/proto/payments/ledger/v1 -> payments.ledger.v1
	parts := strings.Split(path, "/")

	// Find the format directory and extract everything after it
	var moduleParts []string
	foundFormat := false
	for _, part := range parts {
		if part == format {
			foundFormat = true
			continue
		}
		if foundFormat && part != "" {
			moduleParts = append(moduleParts, part)
		}
	}

	if format == "proto" {
		// Proto uses dot notation
		return strings.Join(moduleParts, ".")
	}
	// Others use slash notation
	return strings.Join(moduleParts, "/")
}

func (s *AppScaffolder) generateExampleSchema(moduleDir, format string) error {
	moduleName := extractModuleName(s.modulePath, format)

	switch format {
	case "proto":
		return s.generateProtoExample(moduleDir, moduleName)
	case "openapi":
		return s.generateOpenAPIExample(moduleDir, moduleName)
	case "avro":
		return s.generateAvroExample(moduleDir, moduleName)
	case "jsonschema":
		return s.generateJSONSchemaExample(moduleDir, moduleName)
	case "parquet":
		return s.generateParquetExample(moduleDir, moduleName)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

func (s *AppScaffolder) generateProtoExample(moduleDir, moduleName string) error {
	// Extract the base name for the file
	parts := strings.Split(moduleName, ".")
	baseName := "service" // default
	if len(parts) >= 2 {
		baseName = parts[len(parts)-2] // e.g., "ledger" from "payments.ledger.v1"
	} else if len(parts) == 1 {
		baseName = parts[0]
	}

	content := fmt.Sprintf(`syntax = "proto3";

package %s;

// Example message definition
message %sRequest {
  string id = 1;
}

message %sResponse {
  string result = 1;
}

// Example service definition
service %sService {
  rpc Get%s(%sRequest) returns (%sResponse);
}
`, moduleName, capitalize(baseName), capitalize(baseName),
		capitalize(baseName), capitalize(baseName), capitalize(baseName), capitalize(baseName))

	protoPath := filepath.Join(moduleDir, baseName+".proto")
	return os.WriteFile(protoPath, []byte(content), 0644)
}

func (s *AppScaffolder) generateOpenAPIExample(moduleDir, moduleName string) error {
	// Extract the base name for the file
	parts := strings.Split(moduleName, "/")
	baseName := parts[0]

	content := fmt.Sprintf(`openapi: 3.0.3
info:
  title: %s API
  version: 1.0.0
  description: Example API specification

paths:
  /%s:
    get:
      summary: List %s
      responses:
        '200':
          description: Successful response
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/%s'

components:
  schemas:
    %s:
      type: object
      properties:
        id:
          type: string
        name:
          type: string
`, capitalize(baseName), baseName, baseName, capitalize(baseName), capitalize(baseName))

	yamlPath := filepath.Join(moduleDir, baseName+".yaml")
	return os.WriteFile(yamlPath, []byte(content), 0644)
}

func (s *AppScaffolder) generateAvroExample(moduleDir, moduleName string) error {
	parts := strings.Split(moduleName, "/")
	baseName := parts[0]

	content := fmt.Sprintf(`{
  "type": "record",
  "name": "%s",
  "namespace": "%s",
  "fields": [
    {"name": "id", "type": "string"},
    {"name": "timestamp", "type": "long"}
  ]
}
`, capitalize(baseName), s.org)

	avroPath := filepath.Join(moduleDir, baseName+".avsc")
	return os.WriteFile(avroPath, []byte(content), 0644)
}

func (s *AppScaffolder) generateJSONSchemaExample(moduleDir, moduleName string) error {
	parts := strings.Split(moduleName, "/")
	baseName := parts[0]

	content := fmt.Sprintf(`{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "$id": "https://%s.com/schemas/%s.json",
  "title": "%s",
  "type": "object",
  "properties": {
    "id": {
      "type": "string"
    },
    "name": {
      "type": "string"
    }
  },
  "required": ["id"]
}
`, s.org, baseName, capitalize(baseName))

	jsonPath := filepath.Join(moduleDir, baseName+".json")
	return os.WriteFile(jsonPath, []byte(content), 0644)
}

func (s *AppScaffolder) generateParquetExample(moduleDir, moduleName string) error {
	parts := strings.Split(moduleName, "/")
	baseName := parts[0]

	content := fmt.Sprintf(`message %s {
  required binary id (UTF8);
  optional int64 timestamp;
  optional binary data (UTF8);
}
`, baseName)

	parquetPath := filepath.Join(moduleDir, baseName+".parquet")
	return os.WriteFile(parquetPath, []byte(content), 0644)
}

func (s *AppScaffolder) generateGitignore(path string) error {
	content := `# Generated code
/internal/gen/

# Build artifacts
bin/
dist/
*.exe
*.dll
*.so
*.dylib

# IDE
.idea/
.vscode/
*.swp
*.swo
*~

# OS
.DS_Store
Thumbs.db

# Dependency directories
vendor/
node_modules/

# Test coverage
coverage.out
*.test

# APX lock file (commit this)
# apx.lock
`
	return os.WriteFile(path, []byte(content), 0644)
}

func (s *AppScaffolder) generateBufWorkYaml(path, modulePath string) error {
	// Extract the base path for buf workspace (up to the format directory)
	// Example: internal/apis/proto/payments/ledger/v1 -> internal/apis/proto/payments/ledger
	parts := strings.Split(modulePath, "/")
	var baseParts []string
	for _, part := range parts {
		if strings.HasPrefix(part, "v") && len(part) > 1 {
			// Skip version directory
			break
		}
		baseParts = append(baseParts, part)
	}
	workspacePath := strings.Join(baseParts, "/")

	content := fmt.Sprintf(`version: v2
directories:
  - %s
`, workspacePath)
	return os.WriteFile(path, []byte(content), 0644)
}

func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
