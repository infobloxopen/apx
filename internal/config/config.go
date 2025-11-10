package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the APX configuration
type Config struct {
	Version         int                       `yaml:"version"`
	Org             string                    `yaml:"org"`
	Repo            string                    `yaml:"repo"`
	ModuleRoots     []string                  `yaml:"module_roots"`
	LanguageTargets map[string]LanguageTarget `yaml:"language_targets"`
	Policy          Policy                    `yaml:"policy"`
	Publishing      Publishing                `yaml:"publishing"`
	Tools           Tools                     `yaml:"tools"`
	Execution       Execution                 `yaml:"execution"`
}

// LanguageTarget represents configuration for a target language
type LanguageTarget struct {
	Enabled bool                `yaml:"enabled"`
	Tool    string              `yaml:"tool,omitempty"`
	Version string              `yaml:"version,omitempty"`
	Plugins []map[string]string `yaml:"plugins,omitempty"`
}

// Policy represents policy configuration
type Policy struct {
	ForbiddenProtoOptions []string `yaml:"forbidden_proto_options,omitempty"`
	AllowedProtoPlugins   []string `yaml:"allowed_proto_plugins,omitempty"`
	OpenAPI               struct {
		SpectralRuleset string `yaml:"spectral_ruleset,omitempty"`
	} `yaml:"openapi,omitempty"`
	Avro struct {
		Compatibility string `yaml:"compatibility,omitempty"`
	} `yaml:"avro,omitempty"`
	JSONSchema struct {
		BreakingMode string `yaml:"breaking_mode,omitempty"`
	} `yaml:"jsonschema,omitempty"`
	Parquet struct {
		AllowAdditiveNullableOnly bool `yaml:"allow_additive_nullable_only,omitempty"`
	} `yaml:"parquet,omitempty"`
}

// Publishing represents publishing configuration
type Publishing struct {
	TagFormat string `yaml:"tag_format"`
	CIOnly    bool   `yaml:"ci_only"`
}

// Tools represents tool configuration
type Tools struct {
	Buf struct {
		Version string `yaml:"version"`
	} `yaml:"buf"`
	OASDiff struct {
		Version string `yaml:"version"`
	} `yaml:"oasdiff"`
	Spectral struct {
		Version string `yaml:"version"`
	} `yaml:"spectral"`
	AvroTool struct {
		Version string `yaml:"version"`
	} `yaml:"avrotool"`
	JSONSchemaDiff struct {
		Version string `yaml:"version"`
	} `yaml:"jsonschemadiff"`
}

// Execution represents execution configuration
type Execution struct {
	Mode           string `yaml:"mode"`
	ContainerImage string `yaml:"container_image"`
}

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error in field '%s': %s", e.Field, e.Message)
}

// IsValidationError checks if an error is a validation error
func IsValidationError(err error) bool {
	_, ok := err.(*ValidationError)
	return ok
}

// Load loads configuration from the specified file
func Load(configPath string) (*Config, error) {
	if configPath == "" {
		configPath = "apx.yaml"
	}

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found: %s", configPath)
	}

	// Read the file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate configuration
	if err := validateConfig(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// Init creates a default configuration file
func Init() error {
	configPath := "apx.yaml"

	// Check if file already exists
	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("apx.yaml already exists")
	}

	// Create default configuration
	defaultConfig := `version: 1
org: your-org-name
repo: your-apis-repo
module_roots:
  - proto
  - openapi
  - avro
  - jsonschema
  - parquet
language_targets:
  go:
    enabled: true
    plugins:
      - name: protoc-gen-go
        version: v1.64.0
      - name: protoc-gen-go-grpc
        version: v1.5.0
policy:
  forbidden_proto_options:
    - "^gorm\\."
  allowed_proto_plugins:
    - protoc-gen-go
    - protoc-gen-go-grpc
  openapi:
    spectral_ruleset: ".spectral.yaml"
  avro:
    compatibility: "BACKWARD"
  jsonschema:
    breaking_mode: "strict"
  parquet:
    allow_additive_nullable_only: true
publishing:
  tag_format: "{subdir}/v{version}"
  ci_only: true
tools:
  buf:
    version: v1.45.0
  oasdiff:
    version: v1.9.6
  spectral:
    version: v6.11.0
  avrotool:
    version: "1.11.3"
  jsonschemadiff:
    version: "0.3.0"
execution:
  mode: "local"
  container_image: ""
`

	return os.WriteFile(configPath, []byte(defaultConfig), 0644)
}

// validateConfig validates the configuration
func validateConfig(config *Config) error {
	if config.Version == 0 {
		return &ValidationError{Field: "version", Message: "version is required"}
	}

	if config.Org == "" {
		return &ValidationError{Field: "org", Message: "org is required"}
	}

	if config.Repo == "" {
		return &ValidationError{Field: "repo", Message: "repo is required"}
	}

	return nil
}

// GetConfigPath returns the path to the configuration file
func GetConfigPath() string {
	if path := os.Getenv("APX_CONFIG"); path != "" {
		return path
	}
	return filepath.Join(".", "apx.yaml")
}
