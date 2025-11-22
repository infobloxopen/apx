package validator

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

// ErrNotImplemented is returned when a validator method is not yet implemented
var ErrNotImplemented = errors.New("validator not implemented")

// SchemaFormat represents supported schema formats
type SchemaFormat string

const (
	FormatProto      SchemaFormat = "proto"
	FormatOpenAPI    SchemaFormat = "openapi"
	FormatAvro       SchemaFormat = "avro"
	FormatJSONSchema SchemaFormat = "jsonschema"
	FormatParquet    SchemaFormat = "parquet"
	FormatUnknown    SchemaFormat = "unknown"
)

// Validator provides a unified interface for schema validation
type Validator struct {
	resolver         *ToolchainResolver
	protoValidator   *ProtoValidator
	oasValidator     *OpenAPIValidator
	avroValidator    *AvroValidator
	jsonValidator    *JSONSchemaValidator
	parquetValidator *ParquetValidator
}

// NewValidator creates a new validator with the specified toolchain resolver
func NewValidator(resolver *ToolchainResolver) *Validator {
	return &Validator{
		resolver:         resolver,
		protoValidator:   NewProtoValidator(resolver),
		oasValidator:     NewOpenAPIValidator(resolver),
		avroValidator:    NewAvroValidator(resolver),
		jsonValidator:    NewJSONSchemaValidator(resolver),
		parquetValidator: NewParquetValidator(resolver),
	}
}

// DetectFormat detects the schema format from file path and content
func DetectFormat(path string) SchemaFormat {
	ext := strings.ToLower(filepath.Ext(path))
	base := strings.ToLower(filepath.Base(path))

	// Proto files
	if ext == ".proto" {
		return FormatProto
	}

	// OpenAPI files
	if strings.Contains(base, "openapi") || strings.Contains(base, "swagger") {
		return FormatOpenAPI
	}
	if ext == ".yaml" || ext == ".yml" || ext == ".json" {
		// Could be OpenAPI, Avro, or JSON Schema - need more context
		// For now, check directory structure
		dir := filepath.Dir(path)
		if strings.Contains(dir, "openapi") {
			return FormatOpenAPI
		}
		if strings.Contains(dir, "avro") {
			return FormatAvro
		}
		if strings.Contains(dir, "jsonschema") {
			return FormatJSONSchema
		}
	}

	// Avro files
	if ext == ".avsc" || ext == ".avdl" || ext == ".avpr" {
		return FormatAvro
	}

	// Parquet files
	if ext == ".parquet" || strings.Contains(base, "parquet") {
		return FormatParquet
	}

	return FormatUnknown
}

// Lint validates a schema file for syntax and style issues
func (v *Validator) Lint(path string, format SchemaFormat) error {
	if format == FormatUnknown {
		format = DetectFormat(path)
	}

	switch format {
	case FormatProto:
		return v.protoValidator.Lint(path)
	case FormatOpenAPI:
		return v.oasValidator.Lint(path)
	case FormatAvro:
		return v.avroValidator.Lint(path)
	case FormatJSONSchema:
		return v.jsonValidator.Lint(path)
	case FormatParquet:
		return v.parquetValidator.Lint(path)
	default:
		return fmt.Errorf("unsupported schema format for file: %s", path)
	}
}

// Breaking checks for breaking changes between two schema versions
func (v *Validator) Breaking(path, against string, format SchemaFormat) error {
	if format == FormatUnknown {
		format = DetectFormat(path)
	}

	switch format {
	case FormatProto:
		return v.protoValidator.Breaking(path, against)
	case FormatOpenAPI:
		return v.oasValidator.Breaking(path, against)
	case FormatAvro:
		return v.avroValidator.Breaking(path, against)
	case FormatJSONSchema:
		return v.jsonValidator.Breaking(path, against)
	case FormatParquet:
		return v.parquetValidator.Breaking(path, against)
	default:
		return fmt.Errorf("unsupported schema format for file: %s", path)
	}
}

// SetAvroCompatibilityMode sets the Avro compatibility checking mode
func (v *Validator) SetAvroCompatibilityMode(mode string) {
	v.avroValidator.SetCompatibilityMode(mode)
}

// SetParquetAdditiveNullableOnly sets the Parquet schema evolution policy
func (v *Validator) SetParquetAdditiveNullableOnly(allow bool) {
	v.parquetValidator.SetAdditiveNullableOnlyPolicy(allow)
}
