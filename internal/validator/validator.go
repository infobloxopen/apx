package validator

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
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

// DetectFormat detects the schema format from a file or directory path.
// When path is a directory it walks the tree looking for the first file
// with a recognizable schema extension.
func DetectFormat(path string) SchemaFormat {
	info, err := os.Stat(path)
	if err == nil && info.IsDir() {
		return detectFormatFromDir(path)
	}
	return detectFormatFromFile(path)
}

// detectFormatFromFile detects schema format from a single file path.
func detectFormatFromFile(path string) SchemaFormat {
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

// detectFormatFromDir walks a directory tree and returns the format of
// the first schema file found. It stops as soon as a file with a
// recognized extension is encountered.
func detectFormatFromDir(dir string) SchemaFormat {
	var found SchemaFormat
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if f := detectFormatFromFile(path); f != FormatUnknown {
			found = f
			return filepath.SkipAll
		}
		return nil
	})
	if found == "" {
		return FormatUnknown
	}
	return found
}

// DetectFormatFromModuleRoots infers the schema format from the
// configured module_roots. Each root is expected to end with a format
// segment (e.g. "internal/apis/proto" → proto). If all roots agree on
// one format it is returned; otherwise FormatUnknown.
func DetectFormatFromModuleRoots(roots []string) SchemaFormat {
	if len(roots) == 0 {
		return FormatUnknown
	}
	var detected SchemaFormat
	for _, root := range roots {
		seg := strings.ToLower(filepath.Base(root))
		var f SchemaFormat
		switch seg {
		case "proto", "protobuf":
			f = FormatProto
		case "openapi", "swagger":
			f = FormatOpenAPI
		case "avro":
			f = FormatAvro
		case "jsonschema":
			f = FormatJSONSchema
		case "parquet":
			f = FormatParquet
		default:
			continue
		}
		if detected == "" {
			detected = f
		} else if detected != f {
			return FormatUnknown // ambiguous
		}
	}
	if detected == "" {
		return FormatUnknown
	}
	return detected
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
