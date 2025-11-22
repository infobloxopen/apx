package templates

// GenerateBufYaml generates a buf.yaml configuration for canonical repo
func GenerateBufYaml() string {
	return `version: v2
modules:
  - path: proto
breaking:
  use:
    - FILE
lint:
  use:
    - STANDARD
`
}

// GenerateBufWorkYaml generates a buf.work.yaml workspace configuration for canonical repo
func GenerateBufWorkYaml() string {
	return `version: v2
directories:
  - proto
  - openapi
  - avro
  - jsonschema
  - parquet
`
}
