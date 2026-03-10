package schema

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// ExtractOpenAPI parses an OpenAPI (or Swagger 2.0) spec file and returns
// a summarized view of its paths, operations, and schemas.
func ExtractOpenAPI(filePath string) (*OpenAPISpec, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", filePath, err)
	}

	var doc map[string]interface{}

	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".json":
		if err := json.Unmarshal(data, &doc); err != nil {
			return nil, fmt.Errorf("parsing JSON %s: %w", filePath, err)
		}
	default: // .yaml, .yml
		if err := yaml.Unmarshal(data, &doc); err != nil {
			return nil, fmt.Errorf("parsing YAML %s: %w", filePath, err)
		}
	}

	spec := &OpenAPISpec{}

	// Extract info.
	if info, ok := mapGet[map[string]interface{}](doc, "info"); ok {
		spec.Title, _ = mapGet[string](info, "title")
		spec.Version, _ = mapGet[string](info, "version")
		spec.Description, _ = mapGet[string](info, "description")
	}

	// Extract paths.
	if paths, ok := mapGet[map[string]interface{}](doc, "paths"); ok {
		spec.Paths = extractOpenAPIPaths(paths)
	}

	// Extract schemas — OpenAPI 3.x uses components.schemas, Swagger 2.x uses definitions.
	if components, ok := mapGet[map[string]interface{}](doc, "components"); ok {
		if schemas, ok := mapGet[map[string]interface{}](components, "schemas"); ok {
			spec.Schemas = extractOpenAPISchemas(schemas)
		}
	} else if defs, ok := mapGet[map[string]interface{}](doc, "definitions"); ok {
		spec.Schemas = extractOpenAPISchemas(defs)
	}

	return spec, nil
}

var httpMethods = []string{"get", "post", "put", "delete", "patch", "head", "options"}

func extractOpenAPIPaths(paths map[string]interface{}) []OpenAPIPath {
	// Sort path keys for deterministic output.
	keys := sortedKeys(paths)

	var result []OpenAPIPath
	for _, pathStr := range keys {
		pathObj, ok := mapGet[map[string]interface{}](paths, pathStr)
		if !ok {
			continue
		}

		apiPath := OpenAPIPath{Path: pathStr}

		for _, method := range httpMethods {
			opObj, ok := mapGet[map[string]interface{}](pathObj, method)
			if !ok {
				continue
			}

			op := OpenAPIOperation{
				Method: strings.ToUpper(method),
			}
			op.Summary, _ = mapGet[string](opObj, "summary")
			op.OperationID, _ = mapGet[string](opObj, "operationId")
			op.Description, _ = mapGet[string](opObj, "description")

			// Summarize parameters.
			if params, ok := opObj["parameters"]; ok {
				if paramSlice, ok := params.([]interface{}); ok {
					for _, p := range paramSlice {
						if pm, ok := p.(map[string]interface{}); ok {
							in, _ := mapGet[string](pm, "in")
							name, _ := mapGet[string](pm, "name")
							if in != "" && name != "" {
								op.Parameters = append(op.Parameters, in+": "+name)
							}
						}
					}
				}
			}

			// Summarize responses.
			if responses, ok := mapGet[map[string]interface{}](opObj, "responses"); ok {
				respKeys := sortedKeys(responses)
				for _, code := range respKeys {
					if respObj, ok := mapGet[map[string]interface{}](responses, code); ok {
						desc, _ := mapGet[string](respObj, "description")
						if desc != "" {
							op.Responses = append(op.Responses, code+": "+desc)
						} else {
							op.Responses = append(op.Responses, code)
						}
					}
				}
			}

			apiPath.Operations = append(apiPath.Operations, op)
		}

		if len(apiPath.Operations) > 0 {
			result = append(result, apiPath)
		}
	}

	return result
}

func extractOpenAPISchemas(schemas map[string]interface{}) []OpenAPISchema {
	keys := sortedKeys(schemas)

	var result []OpenAPISchema
	for _, name := range keys {
		schemaObj, ok := mapGet[map[string]interface{}](schemas, name)
		if !ok {
			continue
		}

		s := OpenAPISchema{Name: name}
		s.Type, _ = mapGet[string](schemaObj, "type")

		// Extract required set.
		required := make(map[string]bool)
		if reqArr, ok := schemaObj["required"]; ok {
			if arr, ok := reqArr.([]interface{}); ok {
				for _, v := range arr {
					if str, ok := v.(string); ok {
						required[str] = true
					}
				}
			}
		}

		// Extract properties.
		if props, ok := mapGet[map[string]interface{}](schemaObj, "properties"); ok {
			propKeys := sortedKeys(props)
			for _, propName := range propKeys {
				propObj, ok := mapGet[map[string]interface{}](props, propName)
				if !ok {
					continue
				}
				prop := OpenAPIProperty{
					Name:     propName,
					Type:     openAPIPropertyType(propObj),
					Required: required[propName],
				}
				prop.Description, _ = mapGet[string](propObj, "description")
				s.Properties = append(s.Properties, prop)
			}
		}

		result = append(result, s)
	}

	return result
}

// openAPIPropertyType derives a human-readable type string from a property schema.
func openAPIPropertyType(propObj map[string]interface{}) string {
	if t, ok := mapGet[string](propObj, "type"); ok {
		if t == "array" {
			if items, ok := mapGet[map[string]interface{}](propObj, "items"); ok {
				if itemType, ok := mapGet[string](items, "type"); ok {
					return "array<" + itemType + ">"
				}
				if ref, ok := mapGet[string](items, "$ref"); ok {
					return "array<" + refName(ref) + ">"
				}
			}
			return "array"
		}
		if t == "string" {
			if format, ok := mapGet[string](propObj, "format"); ok {
				return "string(" + format + ")"
			}
		}
		if t == "integer" {
			if format, ok := mapGet[string](propObj, "format"); ok {
				return format // int32, int64
			}
		}
		return t
	}
	if ref, ok := mapGet[string](propObj, "$ref"); ok {
		return refName(ref)
	}
	return ""
}

// refName extracts the schema name from a $ref like "#/components/schemas/Foo".
func refName(ref string) string {
	parts := strings.Split(ref, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ref
}

// ── helpers ────────────────────────────────────────────────────────────────

// mapGet performs a typed map lookup with generics.
func mapGet[T any](m map[string]interface{}, key string) (T, bool) {
	v, ok := m[key]
	if !ok {
		var zero T
		return zero, false
	}
	t, ok := v.(T)
	return t, ok
}

func sortedKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
