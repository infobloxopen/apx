package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// ValidateFile reads an apx.yaml file and validates it against the canonical schema.
// It returns a ValidationResult containing all errors and warnings.
func ValidateFile(path string) (*ValidationResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	return ValidateBytes(data)
}

// ValidateBytes validates raw YAML bytes against the canonical schema.
func ValidateBytes(data []byte) (*ValidationResult, error) {
	node, err := parseYAMLNode(data)
	if err != nil {
		return &ValidationResult{
			Valid: false,
			Errors: []*ValidationError{{
				Field:   "",
				Kind:    ErrInvalidType,
				Message: fmt.Sprintf("YAML syntax error: %v", err),
				Hint:    "fix the YAML syntax and try again",
			}},
		}, nil
	}

	// Handle empty document
	if node == nil || node.Kind == 0 || (node.Kind == yaml.DocumentNode && (len(node.Content) == 0 || node.Content[0].Kind == yaml.ScalarNode && node.Content[0].Value == "")) {
		return &ValidationResult{
			Valid: false,
			Errors: []*ValidationError{{
				Field:   "",
				Kind:    ErrMissing,
				Message: "configuration file is empty",
				Hint:    "add at least 'version: 1', 'org:', and 'repo:' to apx.yaml",
			}},
		}, nil
	}

	// Unwrap document node to get the root mapping
	root := node
	if root.Kind == yaml.DocumentNode {
		if len(root.Content) == 0 {
			return &ValidationResult{
				Valid: false,
				Errors: []*ValidationError{{
					Field:   "",
					Kind:    ErrMissing,
					Message: "configuration file is empty",
					Hint:    "add at least 'version: 1', 'org:', and 'repo:' to apx.yaml",
				}},
			}, nil
		}
		root = root.Content[0]
	}

	if root.Kind != yaml.MappingNode {
		return &ValidationResult{
			Valid: false,
			Errors: []*ValidationError{{
				Field:   "",
				Kind:    ErrInvalidType,
				Message: "expected a YAML mapping at the top level",
				Line:    root.Line,
				Hint:    "apx.yaml must be a YAML mapping (key: value pairs)",
			}},
		}, nil
	}

	// Extract version to look up schema
	version, versionLine, err := extractVersion(root)
	if err != nil {
		return &ValidationResult{
			Valid: false,
			Errors: []*ValidationError{{
				Field:   "version",
				Kind:    ErrMissing,
				Message: "field 'version' is required",
				Hint:    "add 'version: 1' to apx.yaml",
			}},
		}, nil
	}

	schema, ok := Registry.Versions[version]
	if !ok {
		hint := fmt.Sprintf("this APX binary supports schema versions 1–%d", Registry.CurrentVersion)
		msg := fmt.Sprintf("unsupported schema version %d", version)
		if version > Registry.CurrentVersion {
			hint = fmt.Sprintf("upgrade APX to handle schema version %d (this binary supports up to %d)", version, Registry.CurrentVersion)
			msg = fmt.Sprintf("schema version %d is newer than this APX binary supports (max: %d)", version, Registry.CurrentVersion)
		}
		return &ValidationResult{
			Valid: false,
			Errors: []*ValidationError{{
				Field:   "version",
				Kind:    ErrInvalidValue,
				Message: msg,
				Line:    versionLine,
				Hint:    hint,
			}},
		}, nil
	}

	// Walk the node tree against the schema
	result := &ValidationResult{}
	walkNode(root, "", schema.Fields, result)

	// Check required top-level fields
	presentKeys := collectMappingKeys(root)
	for _, req := range schema.RequiredFields {
		if _, found := presentKeys[req]; !found {
			result.Errors = append(result.Errors, &ValidationError{
				Field:   req,
				Kind:    ErrMissing,
				Message: fmt.Sprintf("field '%s' is required", req),
				Hint:    fmt.Sprintf("add '%s: <value>' to apx.yaml", req),
			})
		}
	}

	result.Valid = len(result.Errors) == 0
	return result, nil
}

// parseYAMLNode parses YAML bytes into a *yaml.Node tree.
func parseYAMLNode(data []byte) (*yaml.Node, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	return &doc, nil
}

// extractVersion reads the "version" field from a root mapping node.
// Returns the version integer, the line number, and any error.
func extractVersion(root *yaml.Node) (int, int, error) {
	if root.Kind != yaml.MappingNode {
		return 0, 0, fmt.Errorf("root is not a mapping")
	}

	for i := 0; i < len(root.Content)-1; i += 2 {
		keyNode := root.Content[i]
		valNode := root.Content[i+1]
		if keyNode.Value == "version" {
			var version int
			if err := valNode.Decode(&version); err != nil {
				return 0, valNode.Line, fmt.Errorf("version is not an integer")
			}
			return version, valNode.Line, nil
		}
	}
	return 0, 0, fmt.Errorf("version field not found")
}

// walkNode recursively validates a YAML mapping node against a field-definition map.
func walkNode(node *yaml.Node, prefix string, fields map[string]FieldDef, result *ValidationResult) {
	if node.Kind != yaml.MappingNode {
		return
	}

	for i := 0; i < len(node.Content)-1; i += 2 {
		keyNode := node.Content[i]
		valNode := node.Content[i+1]
		key := keyNode.Value
		path := key
		if prefix != "" {
			path = prefix + "." + key
		}

		fd, known := fields[key]
		if !known {
			result.Errors = append(result.Errors, &ValidationError{
				Field:   path,
				Kind:    ErrUnknownKey,
				Message: fmt.Sprintf("unknown field '%s'", path),
				Line:    keyNode.Line,
				Hint:    fmt.Sprintf("remove '%s' from apx.yaml; see 'apx config validate --help' for valid fields", key),
			})
			continue
		}

		// Check for deprecated fields
		if fd.DeprecatedSince > 0 {
			msg := fmt.Sprintf("field '%s' is deprecated since version %d", path, fd.DeprecatedSince)
			hint := fmt.Sprintf("remove '%s' from apx.yaml", key)
			if fd.Replacement != "" {
				msg += fmt.Sprintf("; use '%s' instead", fd.Replacement)
				hint = fmt.Sprintf("replace '%s' with '%s'", key, fd.Replacement)
			}
			result.Warnings = append(result.Warnings, &ValidationError{
				Field:   path,
				Kind:    ErrDeprecated,
				Message: msg,
				Line:    keyNode.Line,
				Hint:    hint,
			})
		}

		// Type checking
		if !checkType(valNode, fd) {
			result.Errors = append(result.Errors, &ValidationError{
				Field:   path,
				Kind:    ErrInvalidType,
				Message: fmt.Sprintf("field '%s' expects %s, got %s", path, fd.Type, yamlNodeTypeStr(valNode)),
				Line:    valNode.Line,
				Hint:    fmt.Sprintf("change '%s' to a %s value", key, fd.Type),
			})
			continue
		}

		// Enum validation for scalar string fields
		if fd.Type == TypeString && len(fd.EnumValues) > 0 {
			validateEnum(valNode, path, fd, result)
		}

		// Pattern validation
		if fd.Type == TypeString && fd.Pattern != "" {
			validatePattern(valNode, path, fd, result)
		}

		// Recurse into struct children
		if fd.Type == TypeStruct && valNode.Kind == yaml.MappingNode && len(fd.Children) > 0 {
			walkNode(valNode, path, fd.Children, result)
			// Check required children
			checkRequiredChildren(valNode, path, fd.Children, result)
		}

		// Recurse into map values (dynamic keys → struct values)
		if fd.Type == TypeMap && valNode.Kind == yaml.MappingNode && fd.ItemDef != nil {
			for j := 0; j < len(valNode.Content)-1; j += 2 {
				mapKeyNode := valNode.Content[j]
				mapValNode := valNode.Content[j+1]
				mapPath := path + "." + mapKeyNode.Value

				if fd.ItemDef.Type == TypeStruct && mapValNode.Kind == yaml.MappingNode && len(fd.ItemDef.Children) > 0 {
					walkNode(mapValNode, mapPath, fd.ItemDef.Children, result)
					checkRequiredChildren(mapValNode, mapPath, fd.ItemDef.Children, result)
				}
			}
		}

		// Validate list items
		if fd.Type == TypeList && valNode.Kind == yaml.SequenceNode && fd.ItemDef != nil {
			for j, item := range valNode.Content {
				itemPath := fmt.Sprintf("%s[%d]", path, j)
				if fd.ItemDef.Type == TypeStruct && item.Kind == yaml.MappingNode && len(fd.ItemDef.Children) > 0 {
					walkNode(item, itemPath, fd.ItemDef.Children, result)
					checkRequiredChildren(item, itemPath, fd.ItemDef.Children, result)
				} else if fd.ItemDef.Type == TypeString && item.Kind != yaml.ScalarNode {
					result.Errors = append(result.Errors, &ValidationError{
						Field:   itemPath,
						Kind:    ErrInvalidType,
						Message: fmt.Sprintf("field '%s' expects string, got %s", itemPath, yamlNodeTypeStr(item)),
						Line:    item.Line,
					})
				}
			}
		}
	}
}

// validateEnum checks if a scalar value is one of the allowed enum values.
func validateEnum(valNode *yaml.Node, path string, fd FieldDef, result *ValidationResult) {
	val := valNode.Value
	for _, allowed := range fd.EnumValues {
		if val == allowed {
			return
		}
	}
	result.Errors = append(result.Errors, &ValidationError{
		Field:   path,
		Kind:    ErrInvalidValue,
		Message: fmt.Sprintf("field '%s' must be one of [%s], got '%s'", path, strings.Join(fd.EnumValues, ", "), val),
		Line:    valNode.Line,
		Hint:    fmt.Sprintf("change '%s' to one of: %s", fd.Name, strings.Join(fd.EnumValues, ", ")),
	})
}

// validatePattern checks the value against a pattern constraint.
// For tag_format, the pattern is "must contain {version}".
func validatePattern(valNode *yaml.Node, path string, fd FieldDef, result *ValidationResult) {
	val := valNode.Value

	// Special-case: "must contain {version}" pattern for tag_format
	if fd.Pattern == "must contain {version}" {
		if !strings.Contains(val, "{version}") {
			result.Errors = append(result.Errors, &ValidationError{
				Field:   path,
				Kind:    ErrInvalidValue,
				Message: fmt.Sprintf("field '%s' %s, got '%s'", path, fd.Pattern, val),
				Line:    valNode.Line,
				Hint:    fmt.Sprintf("ensure '%s' contains the '{version}' placeholder", fd.Name),
			})
		}
	}
}

// checkType returns true if the YAML node matches the expected FieldType.
func checkType(node *yaml.Node, fd FieldDef) bool {
	switch fd.Type {
	case TypeInt:
		if node.Kind != yaml.ScalarNode {
			return false
		}
		// yaml.v3 uses tag !!int for integers
		return node.Tag == "!!int"
	case TypeString:
		if node.Kind != yaml.ScalarNode {
			return false
		}
		// Accept !!str, !!null (empty values), and untagged scalars
		return node.Tag == "!!str" || node.Tag == "" || node.Tag == "!!null"
	case TypeBool:
		if node.Kind != yaml.ScalarNode {
			return false
		}
		return node.Tag == "!!bool"
	case TypeList:
		return node.Kind == yaml.SequenceNode
	case TypeMap:
		return node.Kind == yaml.MappingNode
	case TypeStruct:
		return node.Kind == yaml.MappingNode
	}
	return false
}

// yamlNodeTypeStr returns a human-readable type name for a yaml.Node.
func yamlNodeTypeStr(node *yaml.Node) string {
	switch node.Kind {
	case yaml.ScalarNode:
		switch node.Tag {
		case "!!int":
			return "integer"
		case "!!bool":
			return "boolean"
		case "!!float":
			return "float"
		case "!!null":
			return "null"
		default:
			return "string"
		}
	case yaml.MappingNode:
		return "mapping"
	case yaml.SequenceNode:
		return "list"
	case yaml.AliasNode:
		return "alias"
	default:
		return "unknown"
	}
}

// collectMappingKeys returns the set of keys present in a mapping node.
func collectMappingKeys(node *yaml.Node) map[string]bool {
	keys := make(map[string]bool)
	if node.Kind != yaml.MappingNode {
		return keys
	}
	for i := 0; i < len(node.Content)-1; i += 2 {
		keys[node.Content[i].Value] = true
	}
	return keys
}

// checkRequiredChildren looks for missing required fields within a struct node.
func checkRequiredChildren(node *yaml.Node, prefix string, children map[string]FieldDef, result *ValidationResult) {
	present := collectMappingKeys(node)
	for name, fd := range children {
		if fd.Required && !present[name] {
			path := name
			if prefix != "" {
				path = prefix + "." + name
			}
			result.Errors = append(result.Errors, &ValidationError{
				Field:   path,
				Kind:    ErrMissing,
				Message: fmt.Sprintf("field '%s' is required", path),
				Hint:    fmt.Sprintf("add '%s: <value>' under '%s'", name, prefix),
			})
		}
	}
}
