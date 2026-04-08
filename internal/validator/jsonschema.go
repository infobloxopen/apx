package validator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	"http://json-schema.org/draft-04/schema#":      true,
	"http://json-schema.org/draft-04/schema":       true,
	"http://json-schema.org/draft-06/schema#":      true,
	"http://json-schema.org/draft-06/schema":       true,
	"http://json-schema.org/draft-07/schema#":      true,
	"http://json-schema.org/draft-07/schema":       true,
	"https://json-schema.org/draft/2019-09/schema": true,
	"https://json-schema.org/draft/2020-12/schema": true,
}

// validJSONSchemaTypes is the set of primitive JSON Schema type names.
var validJSONSchemaTypes = map[string]bool{
	"null": true, "boolean": true, "integer": true, "number": true,
	"string": true, "array": true, "object": true,
}

// Lint validates JSON Schema syntax using native Go parsing. If path is a
// directory, all *.json files under it are linted recursively.
func (v *JSONSchemaValidator) Lint(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}

	if info.IsDir() {
		return v.lintDir(absPath)
	}
	return v.lintFile(absPath)
}

// lintDir walks a directory and lints every *.json file.
func (v *JSONSchemaValidator) lintDir(dir string) error {
	found := false
	err := filepath.Walk(dir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() || filepath.Ext(path) != ".json" {
			return nil
		}
		found = true
		if lintErr := v.lintFile(path); lintErr != nil {
			return fmt.Errorf("%s: %w", path, lintErr)
		}
		return nil
	})
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("no .json files found in %s", dir)
	}
	return nil
}

// lintFile validates a single JSON Schema file.
func (v *JSONSchemaValidator) lintFile(absPath string) error {
	data, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("reading %s: %w", absPath, err)
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

// Breaking detects backward-incompatible changes between two JSON Schema
// files using native Go comparison. No external tools required.
//
// A change is breaking if a payload valid under the old schema could be
// rejected by the new schema. Detected breaking changes:
//   - Property removed from "properties"
//   - Property type changed (e.g., string → integer)
//   - New field added to "required"
//   - Type of the root schema changed
func (v *JSONSchemaValidator) Breaking(path, against string) error {
	oldSchema, err := loadJSONSchemaFile(against)
	if err != nil {
		// Baseline doesn't exist — new schema, nothing to compare.
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("reading baseline schema: %w", err)
	}

	newSchema, err := loadJSONSchemaFile(path)
	if err != nil {
		return fmt.Errorf("reading new schema: %w", err)
	}

	var breaking []string

	// Check root type change.
	if oldType, newType := jsonSchemaType(oldSchema), jsonSchemaType(newSchema); oldType != "" && newType != "" && oldType != newType {
		breaking = append(breaking, fmt.Sprintf("root type changed from %q to %q", oldType, newType))
	}

	// Check properties.
	oldProps := jsonSchemaProperties(oldSchema)
	newProps := jsonSchemaProperties(newSchema)
	for name := range oldProps {
		if _, exists := newProps[name]; !exists {
			breaking = append(breaking, fmt.Sprintf("property %q removed", name))
			continue
		}
		oldPropType := jsonSchemaType(oldProps[name])
		newPropType := jsonSchemaType(newProps[name])
		if oldPropType != "" && newPropType != "" && oldPropType != newPropType {
			breaking = append(breaking, fmt.Sprintf("property %q type changed from %q to %q", name, oldPropType, newPropType))
		}
	}

	// Check required fields added.
	oldRequired := jsonSchemaRequired(oldSchema)
	newRequired := jsonSchemaRequired(newSchema)
	for field := range newRequired {
		if !oldRequired[field] {
			breaking = append(breaking, fmt.Sprintf("field %q added to required", field))
		}
	}

	if len(breaking) > 0 {
		return fmt.Errorf("breaking changes detected:\n  - %s", strings.Join(breaking, "\n  - "))
	}
	return nil
}

// loadJSONSchemaFile reads and parses a JSON Schema file.
func loadJSONSchemaFile(path string) (map[string]json.RawMessage, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}
	var schema map[string]json.RawMessage
	if err := json.Unmarshal(data, &schema); err != nil {
		return nil, fmt.Errorf("invalid JSON in %s: %w", path, err)
	}
	return schema, nil
}

// jsonSchemaType extracts the "type" string from a schema object.
// Returns empty string if not present or not a simple string type.
func jsonSchemaType(schema map[string]json.RawMessage) string {
	raw, ok := schema["type"]
	if !ok {
		return ""
	}
	var t string
	if err := json.Unmarshal(raw, &t); err != nil {
		return ""
	}
	return t
}

// jsonSchemaProperties extracts the "properties" map from a schema object.
func jsonSchemaProperties(schema map[string]json.RawMessage) map[string]map[string]json.RawMessage {
	raw, ok := schema["properties"]
	if !ok {
		return nil
	}
	var props map[string]json.RawMessage
	if err := json.Unmarshal(raw, &props); err != nil {
		return nil
	}
	result := make(map[string]map[string]json.RawMessage, len(props))
	for name, propRaw := range props {
		var prop map[string]json.RawMessage
		if err := json.Unmarshal(propRaw, &prop); err == nil {
			result[name] = prop
		}
	}
	return result
}

// jsonSchemaRequired extracts the "required" array as a set.
func jsonSchemaRequired(schema map[string]json.RawMessage) map[string]bool {
	raw, ok := schema["required"]
	if !ok {
		return nil
	}
	var req []string
	if err := json.Unmarshal(raw, &req); err != nil {
		return nil
	}
	set := make(map[string]bool, len(req))
	for _, r := range req {
		set[r] = true
	}
	return set
}
