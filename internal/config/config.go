package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the APX configuration
type Config struct {
	Version           int                       `yaml:"version"`
	Org               string                    `yaml:"org"`
	Repo              string                    `yaml:"repo"`
	ImportRoot        string                    `yaml:"import_root,omitempty"`        // optional public import root (e.g. go.acme.dev/apis)
	SiteURL           string                    `yaml:"site_url,omitempty"`           // custom domain for the catalog site (e.g. apis.internal.infoblox.dev)
	CatalogURL        string                    `yaml:"catalog_url,omitempty"`        // remote catalog URL for discovery
	CatalogRegistries []CatalogRegistry         `yaml:"catalog_registries,omitempty"` // OCI catalog registries for discovery
	ModuleRoots       []string                  `yaml:"module_roots"`
	LanguageTargets   map[string]LanguageTarget `yaml:"language_targets"`
	Policy            Policy                    `yaml:"policy"`
	Release           ReleaseConfig             `yaml:"release"`
	Tools             Tools                     `yaml:"tools"`
	Execution         Execution                 `yaml:"execution"`
	API               *APIIdentity              `yaml:"api,omitempty"`
	Source            *SourceIdentity           `yaml:"source,omitempty"`
	Releases          *ReleaseInfo              `yaml:"releases,omitempty"`
	Languages         map[string]LanguageCoords `yaml:"languages,omitempty"`
	ExternalAPIs      []ExternalRegistration    `yaml:"external_apis,omitempty"`
	APISources        []APISource               `yaml:"api_sources,omitempty"`
}

// APIIdentity describes the canonical identity of an API.
type APIIdentity struct {
	ID        string `yaml:"id" json:"id"`               // e.g. "proto/payments/ledger/v1"
	Format    string `yaml:"format" json:"format"`       // e.g. "proto", "openapi", "avro"
	Domain    string `yaml:"domain" json:"domain"`       // e.g. "payments"
	Name      string `yaml:"name" json:"name"`           // e.g. "ledger"
	Line      string `yaml:"line" json:"line"`           // e.g. "v0", "v1", "v2"
	Lifecycle string `yaml:"lifecycle" json:"lifecycle"` // experimental, beta, stable, deprecated, sunset
}

// SourceIdentity describes where the canonical source lives.
type SourceIdentity struct {
	Repo string `yaml:"repo" json:"repo"` // e.g. "github.com/acme/apis"
	Path string `yaml:"path" json:"path"` // e.g. "proto/payments/ledger/v1"
}

// ReleaseInfo tracks the current and latest releases for an API line.
type ReleaseInfo struct {
	Current string `yaml:"current" json:"current"` // e.g. "v1.0.0-beta.1"
}

// LanguageCoords describes language-specific derived coordinates for an API.
type LanguageCoords struct {
	Module string `yaml:"module,omitempty" json:"module,omitempty"` // e.g. "github.com/acme/apis/proto/payments/ledger"
	Import string `yaml:"import,omitempty" json:"import,omitempty"` // e.g. "github.com/acme/apis/proto/payments/ledger/v1"
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

// ReleaseConfig represents release configuration
type ReleaseConfig struct {
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

// CatalogRegistry identifies a GHCR-hosted catalog to query for API discovery.
// The catalog image is derived as ghcr.io/<org>/<repo>-catalog:latest.
type CatalogRegistry struct {
	Org  string `yaml:"org" json:"org"`   // GitHub org (e.g. "acme")
	Repo string `yaml:"repo" json:"repo"` // canonical repo name (e.g. "apis")
}

// LockFile represents the apx.lock file structure for dependency pinning
type LockFile struct {
	Version      int                       `yaml:"version"`
	Toolchains   map[string]ToolchainLock  `yaml:"toolchains"`
	Dependencies map[string]DependencyLock `yaml:"dependencies"`
}

// ToolchainLock represents a locked toolchain version
type ToolchainLock struct {
	Version  string `yaml:"version"`
	Checksum string `yaml:"checksum,omitempty"`
	Path     string `yaml:"path,omitempty"` // For offline bundles
}

// DependencyLock represents a locked schema dependency
type DependencyLock struct {
	Repo         string   `yaml:"repo"`
	Ref          string   `yaml:"ref"`
	Modules      []string `yaml:"modules"`
	Origin       string   `yaml:"origin,omitempty"`
	UpstreamRepo string   `yaml:"upstream_repo,omitempty"`
	UpstreamPath string   `yaml:"upstream_path,omitempty"`
	ImportMode   string   `yaml:"import_mode,omitempty"`
}

// ErrorKind classifies the type of a validation error.
type ErrorKind string

const (
	ErrMissing      ErrorKind = "missing"
	ErrInvalidType  ErrorKind = "invalid_type"
	ErrInvalidValue ErrorKind = "invalid_value"
	ErrUnknownKey   ErrorKind = "unknown_key"
	ErrDeprecated   ErrorKind = "deprecated"
)

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string    `json:"field"`
	Kind    ErrorKind `json:"kind"`
	Message string    `json:"message"`
	Line    int       `json:"line,omitempty"`
	Hint    string    `json:"hint,omitempty"`
}

func (e *ValidationError) Error() string {
	if e.Line > 0 {
		return fmt.Sprintf("line %d: %s", e.Line, e.Message)
	}
	return e.Message
}

// MarshalJSON implements json.Marshaler for ValidationError.
func (e *ValidationError) MarshalJSON() ([]byte, error) {
	type alias struct {
		Field   string    `json:"field"`
		Kind    ErrorKind `json:"kind"`
		Message string    `json:"message"`
		Line    int       `json:"line,omitempty"`
		Hint    string    `json:"hint,omitempty"`
	}
	return json.Marshal(&alias{
		Field:   e.Field,
		Kind:    e.Kind,
		Message: e.Message,
		Line:    e.Line,
		Hint:    e.Hint,
	})
}

// ValidationResult aggregates the outcome of validating an entire apx.yaml file.
type ValidationResult struct {
	Errors   []*ValidationError `json:"errors"`
	Warnings []*ValidationError `json:"warnings"`
	Valid    bool               `json:"valid"`
}

// MarshalJSON implements json.Marshaler for ValidationResult.
func (r *ValidationResult) MarshalJSON() ([]byte, error) {
	type alias struct {
		Valid    bool               `json:"valid"`
		Errors   []*ValidationError `json:"errors"`
		Warnings []*ValidationError `json:"warnings"`
	}
	errs := r.Errors
	if errs == nil {
		errs = []*ValidationError{}
	}
	warns := r.Warnings
	if warns == nil {
		warns = []*ValidationError{}
	}
	return json.Marshal(&alias{
		Valid:    r.Valid,
		Errors:   errs,
		Warnings: warns,
	})
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
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate against the canonical schema
	result, err := ValidateFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}
	if !result.Valid {
		// Return the first error for backward compatibility
		return nil, result.Errors[0]
	}

	return &cfg, nil
}

// LoadRaw reads and parses the config YAML without schema validation.
// This is useful for extracting basic fields (like Org) from configs that
// may have been modified by other tools (e.g. apx add) and don't pass
// strict validation.
func LoadRaw(configPath string) (*Config, error) {
	if configPath == "" {
		configPath = "apx.yaml"
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}
	return &cfg, nil
}

// Init creates a default configuration file
func Init() error {
	configPath := "apx.yaml"

	// Check if file already exists
	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("apx.yaml already exists")
	}

	// Use canonical DefaultConfig + MarshalConfigString for guaranteed schema compliance
	cfg := DefaultConfig()
	content, err := MarshalConfigString(cfg)
	if err != nil {
		return fmt.Errorf("failed to generate default config: %w", err)
	}

	return os.WriteFile(configPath, []byte(content), 0644)
}

// GetConfigPath returns the path to the configuration file
func GetConfigPath() string {
	if path := os.Getenv("APX_CONFIG"); path != "" {
		return path
	}
	return filepath.Join(".", "apx.yaml")
}
