package schema

import (
	"os"
	"path/filepath"
	"strings"
)

// formatExtensions maps schema formats to the file extensions to scan.
var formatExtensions = map[string][]string{
	"proto":      {".proto"},
	"openapi":    {".yaml", ".yml", ".json"},
	"avro":       {".avsc", ".json"},
	"jsonschema": {".json"},
	"parquet":    {".parquet"},
}

// ExtractSchema scans the directory at modulePath for schema files matching
// the given format and returns a SchemaDetail with extracted content.
// Returns nil if modulePath is empty, no schema files are found, or all
// extractions fail. This is intentionally lenient — the site still works
// without schema content.
func ExtractSchema(modulePath, format string) *SchemaDetail {
	if modulePath == "" || format == "" {
		return nil
	}

	exts, ok := formatExtensions[format]
	if !ok {
		return nil
	}

	var files []string
	for _, ext := range exts {
		pattern := filepath.Join(modulePath, "*"+ext)
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}
		files = append(files, matches...)
	}

	if len(files) == 0 {
		return nil
	}

	var schemaFiles []SchemaFile
	for _, filePath := range files {
		sf := SchemaFile{
			Filename: filepath.Base(filePath),
		}

		var extracted bool
		switch format {
		case "proto":
			if result, err := ExtractProto(filePath); err == nil {
				sf.Proto = result
				extracted = true
			}
		case "openapi":
			if result, err := ExtractOpenAPI(filePath); err == nil {
				sf.OpenAPI = result
				extracted = true
			}
		case "avro":
			if result, err := ExtractAvro(filePath); err == nil {
				sf.Avro = result
				extracted = true
			}
		case "jsonschema":
			if result, err := ExtractJSONSchema(filePath); err == nil {
				sf.JSONSchema = result
				extracted = true
			}
		case "parquet":
			if result, err := ExtractParquet(filePath); err == nil {
				sf.Parquet = result
				extracted = true
			}
		}

		if extracted {
			// Store raw file content for source viewer (skip binary formats).
			if format != "parquet" {
				if raw, err := os.ReadFile(filePath); err == nil {
					sf.RawContent = string(raw)
				}
			}
			schemaFiles = append(schemaFiles, sf)
		}
	}

	if len(schemaFiles) == 0 {
		return nil
	}

	return &SchemaDetail{Files: schemaFiles}
}

// IsSchemaFormat returns true if the given format is supported for schema extraction.
func IsSchemaFormat(format string) bool {
	_, ok := formatExtensions[strings.ToLower(format)]
	return ok
}
