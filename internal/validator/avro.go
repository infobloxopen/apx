package validator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// AvroValidator handles Avro schema validation
type AvroValidator struct {
	resolver          *ToolchainResolver
	compatibilityMode string // BACKWARD, FORWARD, FULL, NONE
}

// NewAvroValidator creates a new Avro validator
func NewAvroValidator(resolver *ToolchainResolver) *AvroValidator {
	return &AvroValidator{
		resolver:          resolver,
		compatibilityMode: "BACKWARD",
	}
}

// SetCompatibilityMode sets the compatibility checking mode
func (v *AvroValidator) SetCompatibilityMode(mode string) {
	v.compatibilityMode = mode
}

// avroSchema represents an Avro record schema.
type avroSchema struct {
	Type      string      `json:"type"`
	Name      string      `json:"name"`
	Namespace string      `json:"namespace"`
	Fields    []avroField `json:"fields"`
}

// avroField represents a field within an Avro record.
type avroField struct {
	Name    string          `json:"name"`
	Type    json.RawMessage `json:"type"`
	Default json.RawMessage `json:"default"`
	Doc     string          `json:"doc"`
}

// hasDefault returns true if the field has a default value defined.
func (f avroField) hasDefault() bool {
	return len(f.Default) > 0 && string(f.Default) != "null" || isNullableUnion(f.Type)
}

// isNullableUnion returns true if the type is a union that includes "null"
// as its first element (the Avro convention for optional fields).
func isNullableUnion(raw json.RawMessage) bool {
	var union []json.RawMessage
	if err := json.Unmarshal(raw, &union); err != nil || len(union) == 0 {
		return false
	}
	var first string
	_ = json.Unmarshal(union[0], &first)
	return first == "null"
}

// validAvroTypes is the set of primitive Avro type names.
var validAvroTypes = map[string]bool{
	"null": true, "boolean": true, "int": true, "long": true,
	"float": true, "double": true, "bytes": true, "string": true,
	"record": true, "enum": true, "array": true, "map": true,
	"union": true, "fixed": true,
}

// Lint validates Avro schema syntax using native Go parsing.
func (v *AvroValidator) Lint(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}

	// Must be valid JSON
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("invalid JSON in Avro schema: %w", err)
	}

	// Must have a "type" field
	typeRaw, ok := raw["type"]
	if !ok {
		return fmt.Errorf("avro schema missing required 'type' field")
	}
	var typeName string
	if err := json.Unmarshal(typeRaw, &typeName); err != nil {
		return fmt.Errorf("'type' must be a string: %w", err)
	}
	if !validAvroTypes[typeName] {
		return fmt.Errorf("unknown avro type: %q", typeName)
	}

	// Record schemas must have a "name" and valid "fields"
	if typeName == "record" {
		nameRaw, ok := raw["name"]
		if !ok {
			return fmt.Errorf("avro record schema missing required 'name' field")
		}
		var name string
		if err := json.Unmarshal(nameRaw, &name); err != nil || name == "" {
			return fmt.Errorf("'name' must be a non-empty string")
		}

		fieldsRaw, ok := raw["fields"]
		if !ok {
			return fmt.Errorf("avro record schema missing required 'fields' array")
		}
		var fields []json.RawMessage
		if err := json.Unmarshal(fieldsRaw, &fields); err != nil {
			return fmt.Errorf("'fields' must be an array: %w", err)
		}
		for i, fRaw := range fields {
			var f map[string]json.RawMessage
			if err := json.Unmarshal(fRaw, &f); err != nil {
				return fmt.Errorf("field[%d] is not an object: %w", i, err)
			}
			if _, ok := f["name"]; !ok {
				return fmt.Errorf("field[%d] missing required 'name'", i)
			}
			if _, ok := f["type"]; !ok {
				return fmt.Errorf("field[%d] missing required 'type'", i)
			}
		}
	}

	return nil
}

// Breaking checks Avro schema compatibility between two versions.
// path is the new schema; against is the old/baseline schema.
func (v *AvroValidator) Breaking(path, against string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}
	absAgainst, err := filepath.Abs(against)
	if err != nil {
		return fmt.Errorf("failed to resolve against path: %w", err)
	}

	newData, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("reading new schema %s: %w", path, err)
	}
	oldData, err := os.ReadFile(absAgainst)
	if err != nil {
		return fmt.Errorf("reading old schema %s: %w", against, err)
	}

	var newSchema, oldSchema avroSchema
	if err := json.Unmarshal(newData, &newSchema); err != nil {
		return fmt.Errorf("parsing new schema: %w", err)
	}
	if err := json.Unmarshal(oldData, &oldSchema); err != nil {
		return fmt.Errorf("parsing old schema: %w", err)
	}

	mode := v.compatibilityMode
	if mode == "" {
		mode = "BACKWARD"
	}

	switch strings.ToUpper(mode) {
	case "BACKWARD":
		return checkAvroBackward(newSchema, oldSchema)
	case "FORWARD":
		return checkAvroBackward(oldSchema, newSchema)
	case "FULL":
		if err := checkAvroBackward(newSchema, oldSchema); err != nil {
			return err
		}
		return checkAvroBackward(oldSchema, newSchema)
	case "NONE":
		return nil
	default:
		return fmt.Errorf("unknown compatibility mode: %s", v.compatibilityMode)
	}
}

// checkAvroBackward verifies that reader can read data written by writer.
// Avro BACKWARD compatibility: new schema (reader) can read data written by
// old schema (writer).
func checkAvroBackward(reader, writer avroSchema) error {
	if reader.Type != "record" || writer.Type != "record" {
		// Non-record types: skip structural field check
		return nil
	}

	writerFields := make(map[string]avroField, len(writer.Fields))
	for _, f := range writer.Fields {
		writerFields[f.Name] = f
	}
	readerFields := make(map[string]avroField, len(reader.Fields))
	for _, f := range reader.Fields {
		readerFields[f.Name] = f
	}

	var violations []string

	// For each reader field: if not in writer it must have a default so the
	// reader knows what value to use for records written without it.
	for _, rf := range reader.Fields {
		if _, inWriter := writerFields[rf.Name]; !inWriter {
			if !rf.hasDefault() {
				violations = append(violations, fmt.Sprintf(
					"field %q added to new schema without a default value "+
						"(cannot read old data that is missing this field)",
					rf.Name))
			}
		}
	}

	// For shared fields: types must match (simplified: compare JSON representation).
	for _, rf := range reader.Fields {
		wf, inWriter := writerFields[rf.Name]
		if !inWriter {
			continue
		}
		if string(rf.Type) != string(wf.Type) {
			violations = append(violations, fmt.Sprintf(
				"field %q type changed from %s to %s "+
					"(type changes are not backward compatible)",
				rf.Name, string(wf.Type), string(rf.Type)))
		}
	}

	if len(violations) > 0 {
		return fmt.Errorf("avro backward compatibility violations:\n  %s",
			strings.Join(violations, "\n  "))
	}
	return nil
}
