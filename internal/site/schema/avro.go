package schema

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// avroRaw is the raw JSON structure for Avro schema parsing.
type avroRaw struct {
	Type      string          `json:"type"`
	Name      string          `json:"name"`
	Namespace string          `json:"namespace"`
	Doc       string          `json:"doc"`
	Fields    []avroFieldRaw  `json:"fields"`
	Symbols   []string        `json:"symbols"` // enum
	Items     json.RawMessage `json:"items"`   // array items
}

type avroFieldRaw struct {
	Name    string          `json:"name"`
	Type    json.RawMessage `json:"type"`
	Default json.RawMessage `json:"default"`
	Doc     string          `json:"doc"`
}

// ExtractAvro parses an Avro schema file (.avsc) and returns its structure.
func ExtractAvro(filePath string) (*AvroSchema, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", filePath, err)
	}

	var raw avroRaw
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", filePath, err)
	}

	schema := &AvroSchema{
		Type:      raw.Type,
		Name:      raw.Name,
		Namespace: raw.Namespace,
		Doc:       raw.Doc,
		Symbols:   raw.Symbols,
	}

	for _, f := range raw.Fields {
		field := AvroField{
			Name: f.Name,
			Type: stringifyAvroType(f.Type),
			Doc:  f.Doc,
		}
		if len(f.Default) > 0 {
			field.Default = string(f.Default)
		}
		schema.Fields = append(schema.Fields, field)
	}

	return schema, nil
}

// stringifyAvroType converts a raw JSON type into a human-readable string.
func stringifyAvroType(raw json.RawMessage) string {
	if len(raw) == 0 {
		return "unknown"
	}

	// Try simple string: "string", "int", "long", etc.
	var simple string
	if err := json.Unmarshal(raw, &simple); err == nil {
		return simple
	}

	// Try union (array of types): ["null", "string"]
	var union []json.RawMessage
	if err := json.Unmarshal(raw, &union); err == nil && len(union) > 0 {
		parts := make([]string, 0, len(union))
		for _, u := range union {
			parts = append(parts, stringifyAvroType(u))
		}
		return "union<" + strings.Join(parts, ", ") + ">"
	}

	// Try complex type (map/record/array/enum object)
	var complex struct {
		Type  string          `json:"type"`
		Name  string          `json:"name"`
		Items json.RawMessage `json:"items"`
	}
	if err := json.Unmarshal(raw, &complex); err == nil {
		switch complex.Type {
		case "record":
			if complex.Name != "" {
				return "record<" + complex.Name + ">"
			}
			return "record"
		case "array":
			return "array<" + stringifyAvroType(complex.Items) + ">"
		case "map":
			return "map<" + stringifyAvroType(complex.Items) + ">"
		case "enum":
			if complex.Name != "" {
				return "enum<" + complex.Name + ">"
			}
			return "enum"
		default:
			if complex.Type != "" {
				return complex.Type
			}
		}
	}

	return "complex"
}
