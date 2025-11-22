package validator

import (
	"fmt"
	"os/exec"
	"path/filepath"
)

// OpenAPIValidator handles OpenAPI schema validation
type OpenAPIValidator struct {
	resolver *ToolchainResolver
}

// NewOpenAPIValidator creates a new OpenAPI validator
func NewOpenAPIValidator(resolver *ToolchainResolver) *OpenAPIValidator {
	return &OpenAPIValidator{resolver: resolver}
}

// Lint runs spectral lint on OpenAPI specs
func (v *OpenAPIValidator) Lint(path string) error {
	spectralPath, err := v.resolver.ResolveTool("spectral", "v6.11.0")
	if err != nil {
		return fmt.Errorf("failed to resolve spectral: %w", err)
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	cmd := exec.Command(spectralPath, "lint", absPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("spectral lint failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// Breaking runs oasdiff to detect breaking changes
func (v *OpenAPIValidator) Breaking(path, against string) error {
	oasdiffPath, err := v.resolver.ResolveTool("oasdiff", "v1.9.6")
	if err != nil {
		return fmt.Errorf("failed to resolve oasdiff: %w", err)
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	cmd := exec.Command(oasdiffPath, "breaking", against, absPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("oasdiff breaking failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}
