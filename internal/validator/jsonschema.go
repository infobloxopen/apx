package validator

import (
	"fmt"
	"os/exec"
	"path/filepath"
)

// JSONSchemaValidator handles JSON Schema validation
type JSONSchemaValidator struct {
	resolver *ToolchainResolver
}

// NewJSONSchemaValidator creates a new JSON Schema validator
func NewJSONSchemaValidator(resolver *ToolchainResolver) *JSONSchemaValidator {
	return &JSONSchemaValidator{resolver: resolver}
}

// Lint validates JSON Schema syntax
func (v *JSONSchemaValidator) Lint(path string) error {
	// JSON Schema validation using jsonschema CLI or similar
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	// Placeholder: actual implementation would use jsonschema validator
	_ = absPath
	return nil
}

// Breaking runs jsonschema-diff to detect breaking changes
func (v *JSONSchemaValidator) Breaking(path, against string) error {
	jsDiffPath, err := v.resolver.ResolveTool("jsonschema-diff", "0.3.0")
	if err != nil {
		return fmt.Errorf("failed to resolve jsonschema-diff: %w", err)
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	cmd := exec.Command(jsDiffPath, against, absPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("jsonschema-diff failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}
