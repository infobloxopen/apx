package schema

import (
	"fmt"
	"os"
	"strings"

	"github.com/infobloxopen/apx/internal/config"
	"github.com/infobloxopen/apx/internal/detector"
	"github.com/infobloxopen/apx/internal/ui"
)

// InitOptions holds options for schema initialization
type InitOptions struct {
	Kind       string
	ModulePath string
	Defaults   *detector.ProjectDefaults
}

// Initializer handles schema initialization logic
type Initializer struct {
	// Could include config, logger, etc.
}

// NewInitializer creates a new schema initializer
func NewInitializer() *Initializer {
	return &Initializer{}
}

// Initialize creates a new schema module
func (i *Initializer) Initialize(opts InitOptions) error {
	// Validate kind
	validKinds := []string{"proto", "openapi", "avro", "jsonschema", "parquet"}
	if !contains(validKinds, opts.Kind) {
		return fmt.Errorf("invalid kind '%s'. Supported kinds: %v", opts.Kind, validKinds)
	}

	ui.Info("Initializing %s module: %s", opts.Kind, opts.ModulePath)

	// Create apx.yaml config if it doesn't exist
	if _, err := os.Stat("apx.yaml"); os.IsNotExist(err) {
		ui.Info("Creating apx.yaml configuration file...")
		if err := i.createConfigWithDefaults(opts.Defaults); err != nil {
			return fmt.Errorf("failed to create apx.yaml: %w", err)
		}
	}

	// Create appropriate directory structure based on schema type
	return i.createSchemaFiles(opts)
}

func (i *Initializer) createSchemaFiles(opts InitOptions) error {
	var schemaDir, schemaPath string
	var fileName, schemaContent string
	var additionalFiles []string

	switch opts.Kind {
	case "proto":
		// Use internal directory to prevent vendoring
		schemaDir = "internal/proto"
		if err := os.MkdirAll(schemaDir, 0755); err != nil {
			return fmt.Errorf("failed to create proto directory: %w", err)
		}

		// Create buf.yaml for buf tooling
		bufConfig := fmt.Sprintf(`version: v1
name: buf.build/%s/%s
lint:
  use:
    - DEFAULT
  except:
    - UNARY_RPC
breaking:
  use:
    - FILE
`, opts.Defaults.Org, opts.Defaults.Repo)

		if err := os.WriteFile("buf.yaml", []byte(bufConfig), 0644); err != nil {
			return fmt.Errorf("failed to create buf.yaml: %w", err)
		}
		additionalFiles = append(additionalFiles, "buf.yaml")

		// Create buf.gen.yaml for code generation
		bufGenConfig := `version: v1
plugins:
  - plugin: buf.build/protocolbuffers/go
    out: gen/go
    opt: paths=source_relative
  - plugin: buf.build/grpc/go
    out: gen/go
    opt: paths=source_relative
`
		if err := os.WriteFile("buf.gen.yaml", []byte(bufGenConfig), 0644); err != nil {
			return fmt.Errorf("failed to create buf.gen.yaml: %w", err)
		}
		additionalFiles = append(additionalFiles, "buf.gen.yaml")

		// Create proto file
		fileName = "example.proto"
		schemaPath = fmt.Sprintf("%s/%s", schemaDir, fileName)
		schemaContent = fmt.Sprintf(`syntax = "proto3";

package %s;

option go_package = "%s";

// Example message
message ExampleMessage {
  string id = 1;
  string name = 2;
  int64 timestamp = 3;
}

// Example service
service ExampleService {
  rpc GetExample(GetExampleRequest) returns (ExampleMessage);
}

message GetExampleRequest {
  string id = 1;
}
`, opts.ModulePath, opts.ModulePath)

	default:
		// For non-proto schemas, use schemas directory
		schemaDir = "schemas"
		if err := os.MkdirAll(schemaDir, 0755); err != nil {
			return fmt.Errorf("failed to create schemas directory: %w", err)
		}

		fileName = getSchemaFileName(opts.Kind)
		schemaPath = fmt.Sprintf("%s/%s", schemaDir, fileName)
		schemaContent = getSchemaContent(opts.Kind, opts.ModulePath)
	}

	if err := os.WriteFile(schemaPath, []byte(schemaContent), 0644); err != nil {
		return fmt.Errorf("failed to create schema file: %w", err)
	}

	// Display success message
	ui.Success("Successfully initialized %s module!", opts.Kind)
	ui.Info("Created files:")
	ui.Info("  - apx.yaml (configuration)")

	for _, file := range additionalFiles {
		ui.Info("  - %s", file)
	}

	ui.Info("  - %s (example schema)", schemaPath)
	ui.Info("")
	ui.Info("Next steps:")
	ui.Info("  1. Edit %s to define your schema", schemaPath)
	if opts.Kind == "proto" {
		ui.Info("  2. Run 'buf lint' to validate your proto files")
		ui.Info("  3. Run 'buf generate' to generate code")
		ui.Info("  4. Run 'apx config validate' to check your configuration")
	} else {
		ui.Info("  2. Run 'apx lint' to validate your schema")
		ui.Info("  3. Run 'apx config validate' to check your configuration")
	}

	return nil
}

func (i *Initializer) createConfigWithDefaults(defaults *detector.ProjectDefaults) error {
	if defaults == nil {
		return config.Init() // fallback to default config creation
	}

	// Build language targets based on detected/selected languages
	languageTargets := make(map[string]interface{})

	for _, lang := range defaults.Languages {
		switch lang {
		case "go":
			languageTargets["go"] = map[string]interface{}{
				"enabled": true,
				"plugins": []map[string]string{
					{"name": "protoc-gen-go", "version": "v1.64.0"},
					{"name": "protoc-gen-go-grpc", "version": "v1.5.0"},
				},
			}
		case "python":
			languageTargets["python"] = map[string]interface{}{
				"enabled": true,
				"tool":    "grpcio-tools",
				"version": "1.64.0",
			}
		case "java":
			languageTargets["java"] = map[string]interface{}{
				"enabled": true,
				"plugins": []map[string]string{
					{"name": "protoc-gen-grpc-java", "version": "1.68.1"},
				},
			}
		}
	}

	configContent := fmt.Sprintf(`version: 1
org: %s
repo: %s
module_roots:
  - proto
  - openapi
  - avro
  - jsonschema
  - parquet
language_targets:
%s
policy:
  forbidden_proto_options:
    - "^gorm\\."
  allowed_proto_plugins:
    - protoc-gen-go
    - protoc-gen-go-grpc
  openapi:
    spectral_ruleset: ".spectral.yaml"
  avro:
    compatibility: "BACKWARD"
  jsonschema:
    breaking_mode: "strict"
  parquet:
    allow_additive_nullable_only: true
publishing:
  tag_format: "{subdir}/v{version}"
  ci_only: true
tools:
  buf:
    version: v1.45.0
  oasdiff:
    version: v1.9.6
  spectral:
    version: v6.11.0
  avrotool:
    version: "1.11.3"
  jsonschemadiff:
    version: "0.3.0"
execution:
  mode: "local"
  container_image: ""
`, defaults.Org, defaults.Repo, formatLanguageTargets(languageTargets))

	if _, err := os.Stat("apx.yaml"); err == nil {
		return fmt.Errorf("apx.yaml already exists")
	}

	return os.WriteFile("apx.yaml", []byte(configContent), 0644)
}

// formatLanguageTargets formats the language targets section for YAML
func formatLanguageTargets(targets map[string]interface{}) string {
	var result strings.Builder

	for lang, config := range targets {
		result.WriteString(fmt.Sprintf("  %s:\n", lang))

		if langConfig, ok := config.(map[string]interface{}); ok {
			if enabled, ok := langConfig["enabled"].(bool); ok {
				result.WriteString(fmt.Sprintf("    enabled: %t\n", enabled))
			}

			if tool, ok := langConfig["tool"].(string); ok {
				result.WriteString(fmt.Sprintf("    tool: %s\n", tool))
			}

			if version, ok := langConfig["version"].(string); ok {
				result.WriteString(fmt.Sprintf("    version: %s\n", version))
			}

			if plugins, ok := langConfig["plugins"].([]map[string]string); ok {
				result.WriteString("    plugins:\n")
				for _, plugin := range plugins {
					result.WriteString(fmt.Sprintf("      - name: %s\n", plugin["name"]))
					result.WriteString(fmt.Sprintf("        version: %s\n", plugin["version"]))
				}
			}
		}
	}

	return result.String()
}

// getSchemaFileName returns the appropriate filename for each schema type
func getSchemaFileName(kind string) string {
	switch kind {
	case "openapi":
		return "example.yaml"
	case "avro":
		return "example.avsc"
	case "jsonschema":
		return "example.json"
	case "parquet":
		return "example.parquet.schema"
	default:
		return "example.schema"
	}
}

// getSchemaContent returns the appropriate schema content for each type
func getSchemaContent(kind, modulePath string) string {
	switch kind {
	case "openapi":
		return fmt.Sprintf(`openapi: 3.0.3
info:
  title: %s API
  description: Example OpenAPI specification
  version: 1.0.0
  contact:
    name: API Support
    email: support@example.com

servers:
  - url: https://api.example.com/v1
    description: Production server

paths:
  /examples:
    get:
      summary: List examples
      operationId: listExamples
      responses:
        '200':
          description: List of examples
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/Example'

  /examples/{id}:
    get:
      summary: Get example by ID
      operationId: getExample
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
      responses:
        '200':
          description: Example details
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Example'
        '404':
          description: Example not found

components:
  schemas:
    Example:
      type: object
      required:
        - id
        - name
      properties:
        id:
          type: string
          description: Unique identifier
        name:
          type: string
          description: Example name
        timestamp:
          type: integer
          format: int64
          description: Creation timestamp
`, modulePath)

	case "avro":
		return `{
  "type": "record",
  "name": "ExampleRecord",
  "namespace": "com.example",
  "doc": "Example Avro schema",
  "fields": [
    {
      "name": "id",
      "type": "string",
      "doc": "Unique identifier"
    },
    {
      "name": "name",
      "type": "string",
      "doc": "Record name"
    },
    {
      "name": "timestamp",
      "type": "long",
      "doc": "Creation timestamp"
    }
  ]
}`

	case "jsonschema":
		return `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://example.com/schemas/example.json",
  "title": "Example Schema",
  "description": "An example JSON Schema",
  "type": "object",
  "properties": {
    "id": {
      "type": "string",
      "description": "Unique identifier"
    },
    "name": {
      "type": "string",
      "description": "Example name"
    },
    "timestamp": {
      "type": "integer",
      "description": "Creation timestamp"
    }
  },
  "required": ["id", "name"],
  "additionalProperties": false
}`

	case "parquet":
		return `message ExampleSchema {
  required binary id (UTF8);
  required binary name (UTF8);
  optional int64 timestamp;
}`

	default:
		return "# Example schema content"
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
