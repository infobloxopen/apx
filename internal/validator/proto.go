package validator

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
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
	bufPath, err := v.resolver.ResolveTool("buf", "v1.66.1")
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
	bufPath, err := v.resolver.ResolveTool("buf", "v1.66.1")
	if err != nil {
		return fmt.Errorf("failed to resolve buf: %w", err)
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	// Convert git refs (e.g. HEAD~1, origin/main) to buf's .git#ref= format.
	// If against is already a path or buf-style reference, leave it as-is.
	againstArg := against
	if !strings.Contains(against, "/") || isGitRef(against) {
		againstArg = ".git#ref=" + against
	}

	cmd := exec.Command(bufPath, "breaking", absPath, "--against", againstArg)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("buf breaking failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// isGitRef returns true if the string looks like a git ref (e.g. origin/main, HEAD~1).
func isGitRef(s string) bool {
	if strings.HasPrefix(s, "HEAD") {
		return true
	}
	if strings.HasPrefix(s, "origin/") || strings.HasPrefix(s, "upstream/") {
		return true
	}
	return false
}

// goPackageRe matches: option go_package = "path;alias"; or option go_package = "path";
var goPackageRe = regexp.MustCompile(`^\s*option\s+go_package\s*=\s*"([^"]+)"\s*;`)

// ExtractGoPackage reads a .proto file and extracts the go_package option value.
// Returns the import path, an optional alias, and any error.
//
// Handles the standard forms:
//
//	option go_package = "github.com/acme/apis/proto/payments/ledger/v1";
//	option go_package = "github.com/acme/apis/proto/payments/ledger/v1;ledgerpb";
func ExtractGoPackage(protoPath string) (importPath string, alias string, err error) {
	f, err := os.Open(protoPath)
	if err != nil {
		return "", "", fmt.Errorf("opening proto file: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		// Skip comments
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") {
			continue
		}

		m := goPackageRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		value := m[1]

		// Split on semicolon for alias: "path;alias"
		if idx := strings.Index(value, ";"); idx >= 0 {
			return value[:idx], value[idx+1:], nil
		}
		return value, "", nil
	}
	if err := scanner.Err(); err != nil {
		return "", "", fmt.Errorf("reading proto file: %w", err)
	}

	// No go_package option found — not an error, just empty
	return "", "", nil
}

// GlobProtoFiles returns all .proto files under the given directory.
func GlobProtoFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".proto") {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

// CheckGoPackageCanonical scans proto files under dir and returns warnings for
// any go_package option that contains "apis-go" instead of the canonical "apis"
// import root. Returns a slice of human-readable warning strings (empty if all
// files are clean).
func CheckGoPackageCanonical(dir string) []string {
	files, err := GlobProtoFiles(dir)
	if err != nil || len(files) == 0 {
		return nil
	}

	var warnings []string
	for _, f := range files {
		importPath, _, err := ExtractGoPackage(f)
		if err != nil || importPath == "" {
			continue
		}
		if strings.Contains(importPath, "/apis-go/") {
			rel, _ := filepath.Rel(dir, f)
			if rel == "" {
				rel = f
			}
			warnings = append(warnings, fmt.Sprintf(
				"%s: go_package uses deprecated 'apis-go' import root (%s). "+
					"Use 'apis' instead: replace '/apis-go/' with '/apis/' in go_package.",
				rel, importPath,
			))
		}
	}
	return warnings
}
