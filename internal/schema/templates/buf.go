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
