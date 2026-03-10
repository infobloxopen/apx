package schema

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

// ExtractJSONSchema parses a JSON Schema document and extracts its structure.
func ExtractJSONSchema(filePath string) (*JSONSchemaDoc, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", filePath, err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", filePath, err)
	}

	doc := &JSONSchemaDoc{}

	doc.SchemaURI = jsonString(raw["$schema"])
	doc.Title = jsonString(raw["title"])
	doc.Description = jsonString(raw["description"])
	doc.Type = jsonString(raw["type"])

	// Extract required set.
	required := make(map[string]bool)
	if r, ok := raw["required"]; ok {
		var arr []string
		if json.Unmarshal(r, &arr) == nil {
			for _, name := range arr {
				required[name] = true
			}
		}
	}

	// Extract properties.
	if props, ok := raw["properties"]; ok {
		doc.Properties = extractJSONSchemaProps(props, required, 0)
	}

	return doc, nil
}

const maxJSONSchemaDepth = 3

// extractJSONSchemaProps recursively extracts properties from a JSON Schema "properties" object.
func extractJSONSchemaProps(raw json.RawMessage, required map[string]bool, depth int) []JSONSchemaProp {
	if depth >= maxJSONSchemaDepth {
		return nil
	}

	var propsMap map[string]json.RawMessage
	if err := json.Unmarshal(raw, &propsMap); err != nil {
		return nil
	}

	// Sort keys for deterministic output.
	keys := make([]string, 0, len(propsMap))
	for k := range propsMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var result []JSONSchemaProp
	for _, name := range keys {
		propRaw := propsMap[name]
		var propObj map[string]json.RawMessage
		if json.Unmarshal(propRaw, &propObj) != nil {
			continue
		}

		prop := JSONSchemaProp{
			Name:        name,
			Type:        jsonSchemaTypeString(propObj),
			Description: jsonString(propObj["description"]),
			Required:    required[name],
		}

		// Recurse into nested object properties.
		if nestedProps, ok := propObj["properties"]; ok {
			nestedReq := make(map[string]bool)
			if r, ok := propObj["required"]; ok {
				var arr []string
				if json.Unmarshal(r, &arr) == nil {
					for _, n := range arr {
						nestedReq[n] = true
					}
				}
			}
			prop.Properties = extractJSONSchemaProps(nestedProps, nestedReq, depth+1)
		}

		result = append(result, prop)
	}

	return result
}

// jsonSchemaTypeString extracts a human-readable type from a property schema.
func jsonSchemaTypeString(propObj map[string]json.RawMessage) string {
	// Simple "type" field.
	if t := jsonString(propObj["type"]); t != "" {
		// Check for array items type.
		if t == "array" {
			if items, ok := propObj["items"]; ok {
				var itemObj map[string]json.RawMessage
				if json.Unmarshal(items, &itemObj) == nil {
					if itemType := jsonString(itemObj["type"]); itemType != "" {
						return "array<" + itemType + ">"
					}
				}
			}
		}
		return t
	}

	// "enum" keyword.
	if e, ok := propObj["enum"]; ok {
		var vals []json.RawMessage
		if json.Unmarshal(e, &vals) == nil && len(vals) > 0 {
			return "enum"
		}
	}

	// "$ref" keyword.
	if ref := jsonString(propObj["$ref"]); ref != "" {
		return "$ref"
	}

	return ""
}

// jsonString extracts a string from a JSON value, returning "" if not a string.
func jsonString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	return ""
}
