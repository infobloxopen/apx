// Package schema extracts documentation-level structure from API schema files.
// Each format (proto, openapi, avro, jsonschema, parquet) has a dedicated
// extractor that produces typed output for the catalog site's detail panel.
package schema

// SchemaDetail holds extracted schema content for one API module.
// A module directory may contain multiple schema files.
type SchemaDetail struct {
	Files []SchemaFile `json:"files"`
}

// SchemaFile represents parsed content from one schema file.
// Exactly one format-specific field will be non-nil.
type SchemaFile struct {
	Filename   string         `json:"filename"`
	RawContent string         `json:"raw_content,omitempty"` // original file text for source viewer
	Proto      *ProtoFile     `json:"proto,omitempty"`
	OpenAPI    *OpenAPISpec   `json:"openapi,omitempty"`
	Avro       *AvroSchema    `json:"avro,omitempty"`
	JSONSchema *JSONSchemaDoc `json:"jsonschema,omitempty"`
	Parquet    *ParquetSchema `json:"parquet,omitempty"`
}

// ── Proto ──────────────────────────────────────────────────────────────────

// ProtoFile represents the documentation-level content of a .proto file.
type ProtoFile struct {
	Syntax   string         `json:"syntax"`
	Package  string         `json:"package"`
	Imports  []string       `json:"imports,omitempty"`
	Options  []ProtoOption  `json:"options,omitempty"`
	Services []ProtoService `json:"services,omitempty"`
	Messages []ProtoMessage `json:"messages,omitempty"`
	Enums    []ProtoEnum    `json:"enums,omitempty"`
}

// ProtoOption is a top-level file option (e.g. go_package).
type ProtoOption struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// ProtoService is a gRPC service definition.
type ProtoService struct {
	Name    string     `json:"name"`
	Comment string     `json:"comment,omitempty"`
	Methods []ProtoRPC `json:"methods"`
}

// ProtoRPC is a single RPC method within a service.
type ProtoRPC struct {
	Name            string `json:"name"`
	InputType       string `json:"input_type"`
	OutputType      string `json:"output_type"`
	Comment         string `json:"comment,omitempty"`
	ClientStreaming bool   `json:"client_streaming,omitempty"`
	ServerStreaming bool   `json:"server_streaming,omitempty"`
}

// ProtoMessage is a message definition, potentially with nested types.
type ProtoMessage struct {
	Name    string         `json:"name"`
	Comment string         `json:"comment,omitempty"`
	Fields  []ProtoField   `json:"fields"`
	Nested  []ProtoMessage `json:"nested,omitempty"`
	Enums   []ProtoEnum    `json:"enums,omitempty"`
}

// ProtoField is a single field within a message.
type ProtoField struct {
	Name    string `json:"name"`
	Number  int    `json:"number"`
	Type    string `json:"type"`
	Label   string `json:"label,omitempty"` // repeated, optional, required, map
	Comment string `json:"comment,omitempty"`
}

// ProtoEnum is an enum definition.
type ProtoEnum struct {
	Name    string           `json:"name"`
	Comment string           `json:"comment,omitempty"`
	Values  []ProtoEnumValue `json:"values"`
}

// ProtoEnumValue is a single enum constant.
type ProtoEnumValue struct {
	Name   string `json:"name"`
	Number int    `json:"number"`
}

// ── OpenAPI ────────────────────────────────────────────────────────────────

// OpenAPISpec is a summarized view of an OpenAPI (or Swagger) spec.
type OpenAPISpec struct {
	Title       string          `json:"title,omitempty"`
	Version     string          `json:"version,omitempty"`
	Description string          `json:"description,omitempty"`
	Paths       []OpenAPIPath   `json:"paths,omitempty"`
	Schemas     []OpenAPISchema `json:"schemas,omitempty"`
}

// OpenAPIPath groups all operations for one URL path.
type OpenAPIPath struct {
	Path       string             `json:"path"`
	Operations []OpenAPIOperation `json:"operations"`
}

// OpenAPIOperation is a single HTTP operation (e.g. GET /users).
type OpenAPIOperation struct {
	Method       string   `json:"method"` // GET, POST, PUT, DELETE, PATCH
	Summary      string   `json:"summary,omitempty"`
	OperationID  string   `json:"operation_id,omitempty"`
	Description  string   `json:"description,omitempty"`
	Parameters   []string `json:"parameters,omitempty"`    // "query: limit", "path: id"
	Responses    []string `json:"responses,omitempty"`     // "200: OK", "404: Not Found"
	RequestBody  string   `json:"request_body,omitempty"`  // schema type name, e.g. "CreateUserRequest"
	ResponseBody string   `json:"response_body,omitempty"` // schema type name, e.g. "User"
}

// OpenAPISchema is a component/definition schema.
type OpenAPISchema struct {
	Name       string            `json:"name"`
	Type       string            `json:"type,omitempty"`
	Properties []OpenAPIProperty `json:"properties,omitempty"`
}

// OpenAPIProperty is a single property within a schema.
type OpenAPIProperty struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Required    bool   `json:"required,omitempty"`
	Description string `json:"description,omitempty"`
}

// ── Avro ───────────────────────────────────────────────────────────────────

// AvroSchema represents an Avro record or enum schema.
type AvroSchema struct {
	Type      string      `json:"type"`
	Name      string      `json:"name"`
	Namespace string      `json:"namespace,omitempty"`
	Doc       string      `json:"doc,omitempty"`
	Fields    []AvroField `json:"fields,omitempty"`
	Symbols   []string    `json:"symbols,omitempty"` // for enum types
}

// AvroField is a single field within an Avro record.
type AvroField struct {
	Name    string `json:"name"`
	Type    string `json:"type"` // stringified: "string", "union<null, string>", etc.
	Default string `json:"default,omitempty"`
	Doc     string `json:"doc,omitempty"`
}

// ── JSON Schema ────────────────────────────────────────────────────────────

// JSONSchemaDoc is a summarized view of a JSON Schema document.
type JSONSchemaDoc struct {
	Title       string           `json:"title,omitempty"`
	Description string           `json:"description,omitempty"`
	SchemaURI   string           `json:"schema_uri,omitempty"`
	Type        string           `json:"type,omitempty"`
	Properties  []JSONSchemaProp `json:"properties,omitempty"`
}

// JSONSchemaProp is a property within a JSON Schema, with optional nesting.
type JSONSchemaProp struct {
	Name        string           `json:"name"`
	Type        string           `json:"type,omitempty"`
	Description string           `json:"description,omitempty"`
	Required    bool             `json:"required,omitempty"`
	Properties  []JSONSchemaProp `json:"properties,omitempty"` // nested objects
}

// ── Parquet ────────────────────────────────────────────────────────────────

// ParquetSchema represents a Parquet message definition.
type ParquetSchema struct {
	MessageName string          `json:"message_name"`
	Columns     []ParquetColumn `json:"columns"`
}

// ParquetColumn is a single column within a Parquet message.
type ParquetColumn struct {
	Name       string `json:"name"`
	PhysType   string `json:"phys_type"`
	Repetition string `json:"repetition"` // required, optional, repeated
	Annotation string `json:"annotation,omitempty"`
}
