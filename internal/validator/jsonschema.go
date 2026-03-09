package validator

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// JSONSchemaValidator handles JSON Schema validation
type JSONSchemaValidator struct {
	resolver *ToolchainResolver
}

// NewJSONSchemaValidator creates a new JSON Schema validator
func NewJSONSchemaValidator(resolver *ToolchainResolver) *JSONSchemaValidator {
	return &JSONSchemaValidator{resolver: resolver}
}

// recognizedSchemaDrafts is the set of well-known JSON Schema draft URIs.
var recognizedSchemaDrafts = map[string]bool{
	"http://json-schema.org/draft-04/schema#":        true,
	"http://json-schema.org/draft-04/schema":         true,
	"http://json-schema.org/draft-06/schema#":        true,
	"http://json-schema.org/draft-06/schema":         true,
	"http://json-schema.org/draft-07/schema#":        true,
	"http://json-schema.org/draft-07/schema":         true,
	"https://json-schema.org/draft/2019-09/schema":   true,
	"https://json-schema.org/draft/2020-12/schema":   true,
}

// validJSONSchemaTypes is the set of primitive JSON Schema type names.
var validJSONSchemaTypes = map[string]bool{
	"null": true, "boolean": true, "integer": true, "number": true,
	"string": true, "array": true, "object": true,
}

// Lint validates JSON Schema syntax using native Go parsing.
func (v *JSONSchemaValidator) Lint(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}

	// Must be valid JSON
	var schema map[string]json.RawMessage
	if err := json.Unmarshal(data, &schema); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	// $schema keyword, if present, must be a recognized draft URI
	if raw, ok := schema["$schema"]; ok {
		var schemaURI string
		if err := json.Unmarshal(raw, &schemaURI); err != nil {
			return fmt.Errorf("'$schema' must be a string")
		}
		if !recognizedSchemaDrafts[schemaURI] {
			return fmt.Errorf("unrecognized '$schema' URI: %s", schemaURI)
		}
	}

	// type keyword, if present, must be a valid type string or array of type strings
	if raw, ok := schema["type"]; ok {
		if err := validateJSONSchemaType(raw); err != nil {
			return fmt.Errorf("invalid 'type': %w", err)
		}
	}

	// properties, if present, must be an object
	if raw, ok := schema["properties"]; ok {
		var props map[string]json.RawMessage
		if err := json.Unmarshal(raw, &props); err != nil {
			return fmt.Errorf("'properties' must be an object: %w", err)
		}
	}

	// required, if present, must be an array of strings
	if raw, ok := schema["required"]; ok {
		var req []string
		if err := json.Unmarshal(raw, &req); err != nil {
			return fmt.Errorf("'required' must be an array of strings: %w", err)
		}
	}

	// items, if present, must be an object (or array for tuple validation)
	if raw, ok := schema["items"]; ok {
		var obj map[string]json.RawMessage
		if err := json.Unmarshal(raw, &obj); err != nil {
			// Also allow array form for tuple validation
			var arr []json.RawMessage
			if err2 := json.Unmarshal(raw, &arr); err2 != nil {
				return fmt.Errorf("'items' must be a schema object or array of schema objects")
			}
		}
	}

	return nil
}

// validateJSONSchemaType validates the value of a JSON Schema "type" keyword.
// It accepts either a single type string or an array of type strings.
func validateJSONSchemaType(raw json.RawMessage) error {
	// Try single string form
	var single string
	if err := json.Unmarshal(raw, &single); err == nil {
		if !validJSONSchemaTypes[single] {
			return fmt.Errorf("%q is not a valid JSON Schema type", single)
		}
		return nil
	}

	// Try array form
	var multi []string
	if err := json.Unmarshal(raw, &multi); err != nil {
		return fmt.Errorf("must be a string or array of strings")
	}
	for _, t := range multi {
		if !validJSONSchemaTypes[t] {
			return fmt.Errorf("%q is not a valid JSON Schema type", t)
		}
	}
	return nil
}

// Breaking runs jsonschema-diff to detect breaking changes.
// path is the new schema; against is the old/baseline schema.
func (v *JSONSchemaValidator) Breaking(path, against string) error {
	jsDiffPath, err := v.resolver.ResolveTool("jsonschema-diff", "0.3.0")
	if err != nil {
		return fmt.Errorf("failed to resolve jsonschema-diff: %w", err)
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	cmd := exec.Command(jsDiffPath, against, absPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("jsonschema-diff failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}
