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

	// buf v2 rejects a path contained by a module as a positional build input
	// ("you must provide the workspace or module as the input, and filter to
	// this path using --path"). Run buf from the workspace/module root and
	// select the target schema dir with --path.
	root, rel, err := bufRootAndPath(absPath)
	if err != nil {
		return err
	}

	args := []string{"lint"}
	if rel != "." {
		args = append(args, "--path", rel)
	}
	cmd := exec.Command(bufPath, args...)
	cmd.Dir = root
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

	// As with Lint, buf v2 needs the workspace/module as the input and the
	// target selected via --path rather than a positional subdir input.
	root, rel, err := bufRootAndPath(absPath)
	if err != nil {
		return err
	}

	args := []string{"breaking", "--against", againstArg}
	if rel != "." {
		args = append(args, "--path", rel)
	}
	cmd := exec.Command(bufPath, args...)
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("buf breaking failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// isGitRef returns true if the string looks like a git ref (e.g. origin/main, HEAD~1).
// bufRootAndPath locates the buf workspace/module root at or above absPath
// (the nearest ancestor directory containing buf.work.yaml or buf.yaml) and
// returns that root together with absPath expressed relative to it (slash-
// separated). buf v2 requires the workspace/module as the build input and
// selects targets via --path, so a schema subdirectory contained by a module
// cannot be passed as a positional input directly. When the schema dir is
// itself the root, rel is "." and callers omit --path.
func bufRootAndPath(absPath string) (root, rel string, err error) {
	dir := absPath
	if info, statErr := os.Stat(absPath); statErr != nil || !info.IsDir() {
		dir = filepath.Dir(absPath)
	}
	for {
		for _, name := range []string{"buf.work.yaml", "buf.work.yml", "buf.yaml", "buf.yml"} {
			if _, statErr := os.Stat(filepath.Join(dir, name)); statErr == nil {
				r, relErr := filepath.Rel(dir, absPath)
				if relErr != nil {
					return "", "", fmt.Errorf("computing path relative to buf root: %w", relErr)
				}
				return dir, filepath.ToSlash(r), nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", "", fmt.Errorf("no buf workspace/module config (buf.yaml or buf.work.yaml) found at or above %s", absPath)
		}
		dir = parent
	}
}

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
