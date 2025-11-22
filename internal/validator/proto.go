package validator

import (
	"fmt"
	"os/exec"
	"path/filepath"
)

// ProtoValidator handles Protocol Buffer schema validation
type ProtoValidator struct {
	resolver *ToolchainResolver
}

// NewProtoValidator creates a new Protocol Buffer validator
func NewProtoValidator(resolver *ToolchainResolver) *ProtoValidator {
	return &ProtoValidator{resolver: resolver}
}

// Lint runs buf lint on proto files
func (v *ProtoValidator) Lint(path string) error {
	bufPath, err := v.resolver.ResolveTool("buf", "v1.45.0")
	if err != nil {
		return fmt.Errorf("failed to resolve buf: %w", err)
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	cmd := exec.Command(bufPath, "lint", absPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("buf lint failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// Breaking runs buf breaking change detection
func (v *ProtoValidator) Breaking(path, against string) error {
	bufPath, err := v.resolver.ResolveTool("buf", "v1.45.0")
	if err != nil {
		return fmt.Errorf("failed to resolve buf: %w", err)
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	cmd := exec.Command(bufPath, "breaking", absPath, "--against", against)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("buf breaking failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}
