package validator

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ToolchainResolver finds and validates external tools
type ToolchainResolver struct {
	bundlePath         string
	offlineMode        bool
	checksumValidation bool
}

// ToolchainResolverOption configures the resolver
type ToolchainResolverOption func(*ToolchainResolver)

// WithBundlePath sets the offline bundle directory
func WithBundlePath(path string) ToolchainResolverOption {
	return func(r *ToolchainResolver) {
		r.bundlePath = path
	}
}

// WithOfflineMode enables offline operation
func WithOfflineMode(offline bool) ToolchainResolverOption {
	return func(r *ToolchainResolver) {
		r.offlineMode = offline
	}
}

// WithChecksumValidation enables checksum verification
func WithChecksumValidation(validate bool) ToolchainResolverOption {
	return func(r *ToolchainResolver) {
		r.checksumValidation = validate
	}
}

// NewToolchainResolver creates a new toolchain resolver
func NewToolchainResolver(opts ...ToolchainResolverOption) *ToolchainResolver {
	r := &ToolchainResolver{
		bundlePath:         "",
		offlineMode:        false,
		checksumValidation: false,
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// ResolveTool finds the path to a tool binary
func (r *ToolchainResolver) ResolveTool(name, version string) (string, error) {
	// Try offline bundle first if configured
	if r.bundlePath != "" {
		bundlePath := filepath.Join(r.bundlePath, name)
		if _, err := os.Stat(bundlePath); err == nil {
			return bundlePath, nil
		}
	}

	// Fall back to PATH lookup
	if !r.offlineMode {
		path, err := exec.LookPath(name)
		if err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("tool not found: %s (version %s)", name, version)
}

// ToolRef represents a tool reference with version and checksum
type ToolRef struct {
	Version  string `yaml:"version"`
	Checksum string `yaml:"checksum"`
}

// ToolchainProfile represents the apx.lock file
type ToolchainProfile struct {
	Version int                `yaml:"version"`
	Tools   map[string]ToolRef `yaml:"tools"`
}

// LoadToolchainProfile loads apx.lock
func LoadToolchainProfile(path string) (*ToolchainProfile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read toolchain profile: %w", err)
	}

	var profile ToolchainProfile
	if err := yaml.Unmarshal(data, &profile); err != nil {
		return nil, fmt.Errorf("failed to parse toolchain profile: %w", err)
	}

	return &profile, nil
}

// ValidateTool checks if a tool version matches the profile
func (p *ToolchainProfile) ValidateTool(name, version string) error {
	ref, ok := p.Tools[name]
	if !ok {
		return fmt.Errorf("tool %s not found in profile", name)
	}

	if ref.Version != version {
		return fmt.Errorf("tool %s version mismatch: expected %s, got %s", name, ref.Version, version)
	}

	return nil
}

// ComputeChecksum calculates SHA256 checksum of a file
func ComputeChecksum(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("failed to compute checksum: %w", err)
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
