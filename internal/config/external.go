package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Import mode constants control how downstream consumers reference external API schemas.
const (
	ImportModePreserve = "preserve" // Keep original upstream import paths
	ImportModeRewrite  = "rewrite"  // Rewrite to internal canonical paths
)

// Origin constants classify the provenance of an API.
const (
	OriginExternal = "external" // Registered external API, upstream-preserving
	OriginForked   = "forked"   // Forked external API, internalized
)

// Error sentinels for external API operations.
var (
	ErrExternalDuplicateID   = fmt.Errorf("duplicate external API ID")
	ErrExternalPathConflict  = fmt.Errorf("managed path conflicts with existing module")
	ErrExternalNotFound      = fmt.Errorf("external API not found")
	ErrExternalNotExternal   = fmt.Errorf("API is not an external API")
	ErrExternalAlreadyTarget = fmt.Errorf("API is already at target classification")
	ErrExternalInvalidMode   = fmt.Errorf("invalid import mode")
	ErrExternalInvalidOrigin = fmt.Errorf("invalid origin")
)

// ExternalRegistration represents a registered external API in apx.yaml.
type ExternalRegistration struct {
	ID           string   `yaml:"id"`
	ManagedRepo  string   `yaml:"managed_repo"`
	ManagedPath  string   `yaml:"managed_path"`
	UpstreamRepo string   `yaml:"upstream_repo"`
	UpstreamPath string   `yaml:"upstream_path"`
	ImportMode   string   `yaml:"import_mode,omitempty"`
	Origin       string   `yaml:"origin,omitempty"`
	Description  string   `yaml:"description,omitempty"`
	Lifecycle    string   `yaml:"lifecycle,omitempty"`
	Version      string   `yaml:"version,omitempty"`
	Owners       []string `yaml:"owners,omitempty"`
	Tags         []string `yaml:"tags,omitempty"`
}

// Validate checks that the external registration has all required fields and
// valid values. It applies defaults for import_mode and origin if not set.
func (r *ExternalRegistration) Validate() error {
	// Validate API ID
	if _, err := ParseAPIID(r.ID); err != nil {
		return fmt.Errorf("invalid API ID: %w", err)
	}

	if r.ManagedRepo == "" {
		return fmt.Errorf("managed_repo is required")
	}
	if r.ManagedPath == "" {
		return fmt.Errorf("managed_path is required")
	}
	if r.UpstreamRepo == "" {
		return fmt.Errorf("upstream_repo is required")
	}
	if r.UpstreamPath == "" {
		return fmt.Errorf("upstream_path is required")
	}

	// Validate managed_path: no leading/trailing slashes, no ..
	if strings.HasPrefix(r.ManagedPath, "/") || strings.HasSuffix(r.ManagedPath, "/") {
		return fmt.Errorf("managed_path must not have leading or trailing slashes")
	}
	if strings.Contains(r.ManagedPath, "..") {
		return fmt.Errorf("managed_path must not contain '..'")
	}

	// Apply and validate import_mode
	if r.ImportMode == "" {
		r.ImportMode = ImportModePreserve
	}
	if r.ImportMode != ImportModePreserve && r.ImportMode != ImportModeRewrite {
		return fmt.Errorf("%w: must be %q or %q", ErrExternalInvalidMode, ImportModePreserve, ImportModeRewrite)
	}

	// Apply and validate origin
	if r.Origin == "" {
		r.Origin = OriginExternal
	}
	if r.Origin != OriginExternal && r.Origin != OriginForked {
		return fmt.Errorf("%w: must be %q or %q", ErrExternalInvalidOrigin, OriginExternal, OriginForked)
	}

	// Validate lifecycle if set
	if r.Lifecycle != "" {
		if err := ValidateLifecycle(r.Lifecycle); err != nil {
			return err
		}
	}

	// Validate repo URLs (allow shorthand like github.com/org/repo)
	if err := validateRepoURL(r.ManagedRepo); err != nil {
		return fmt.Errorf("invalid managed_repo URL: %w", err)
	}
	if err := validateRepoURL(r.UpstreamRepo); err != nil {
		return fmt.Errorf("invalid upstream_repo URL: %w", err)
	}

	return nil
}

// validateRepoURL checks that a repository URL is well-formed.
// It accepts either full URLs (https://github.com/org/repo) or
// shorthand (github.com/org/repo).
func validateRepoURL(repoURL string) error {
	if repoURL == "" {
		return fmt.Errorf("URL is empty")
	}

	// If it looks like a full URL, parse it
	if strings.Contains(repoURL, "://") {
		u, err := url.Parse(repoURL)
		if err != nil {
			return fmt.Errorf("malformed URL: %w", err)
		}
		if u.Host == "" {
			return fmt.Errorf("URL must have a host")
		}
		return nil
	}

	// Shorthand form: must have at least host/path
	parts := strings.SplitN(repoURL, "/", 3)
	if len(parts) < 2 {
		return fmt.Errorf("expected host/path format, got %q", repoURL)
	}
	if !strings.Contains(parts[0], ".") {
		return fmt.Errorf("host %q must contain a dot", parts[0])
	}

	return nil
}

// AddExternal adds an external API registration to the config.
// It validates the registration and checks for duplicate IDs and path conflicts.
func AddExternal(cfg *Config, reg *ExternalRegistration, existingModulePaths []string) error {
	if err := reg.Validate(); err != nil {
		return err
	}

	// Check for duplicate ID in existing external registrations
	for _, existing := range cfg.ExternalAPIs {
		if existing.ID == reg.ID {
			return fmt.Errorf("%w: %s", ErrExternalDuplicateID, reg.ID)
		}
	}

	// Check for managed_path conflicts with existing module paths
	for _, p := range existingModulePaths {
		if p == reg.ManagedPath {
			return fmt.Errorf("%w: %s conflicts with existing module at %s",
				ErrExternalPathConflict, reg.ManagedPath, p)
		}
	}

	cfg.ExternalAPIs = append(cfg.ExternalAPIs, *reg)
	return nil
}

// RemoveExternal removes an external API registration by ID.
func RemoveExternal(cfg *Config, apiID string) error {
	for i, reg := range cfg.ExternalAPIs {
		if reg.ID == apiID {
			cfg.ExternalAPIs = append(cfg.ExternalAPIs[:i], cfg.ExternalAPIs[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("%w: %s", ErrExternalNotFound, apiID)
}

// FindExternalByID looks up an external registration by API ID.
func FindExternalByID(cfg *Config, apiID string) (*ExternalRegistration, error) {
	for i := range cfg.ExternalAPIs {
		if cfg.ExternalAPIs[i].ID == apiID {
			return &cfg.ExternalAPIs[i], nil
		}
	}
	return nil, fmt.Errorf("%w: %s", ErrExternalNotFound, apiID)
}

// ListExternals returns all external API registrations, optionally filtered by origin.
func ListExternals(cfg *Config, originFilter string) []ExternalRegistration {
	if originFilter == "" {
		return cfg.ExternalAPIs
	}
	var result []ExternalRegistration
	for _, reg := range cfg.ExternalAPIs {
		if reg.Origin == originFilter {
			result = append(result, reg)
		}
	}
	return result
}

// TransitionExternal changes an external API's origin between "external" and "forked".
// When transitioning to forked, import_mode changes to "rewrite".
// When transitioning to external, import_mode changes to "preserve".
func TransitionExternal(cfg *Config, apiID string, targetOrigin string) error {
	if targetOrigin != OriginExternal && targetOrigin != OriginForked {
		return fmt.Errorf("%w: must be %q or %q", ErrExternalInvalidOrigin, OriginExternal, OriginForked)
	}

	reg, err := FindExternalByID(cfg, apiID)
	if err != nil {
		return err
	}

	if reg.Origin == targetOrigin {
		return fmt.Errorf("%w: %s is already classified as %q",
			ErrExternalAlreadyTarget, apiID, targetOrigin)
	}

	reg.Origin = targetOrigin
	switch targetOrigin {
	case OriginForked:
		reg.ImportMode = ImportModeRewrite
	case OriginExternal:
		reg.ImportMode = ImportModePreserve
	}

	return nil
}

// SaveConfig writes the config back to the specified path as YAML.
func SaveConfig(cfg *Config, configPath string) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	return os.WriteFile(configPath, data, 0644)
}
