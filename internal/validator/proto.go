package validator

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
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

	dir, targetArgs, err := bufTargetArgs(absPath)
	if err != nil {
		return err
	}
	cmd := exec.Command(bufPath, append([]string{"lint"}, targetArgs...)...)
	cmd.Dir = dir
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

	// Convert git refs (HEAD~1, origin/main, release tags) to buf's
	// .git#ref= form. Release tags routinely contain slashes
	// (proto/payments/ledger/v1.0.0), so "contains a slash" cannot mean
	// "is a path" — treat against as a path only when it exists on disk,
	// and pin it to an absolute path since buf runs from the module root.
	againstArg := against
	if strings.HasPrefix(against, ".git#") {
		// already a buf-style reference — leave as-is
	} else if _, statErr := os.Stat(against); statErr == nil {
		if abs, absErr := filepath.Abs(against); absErr == nil {
			againstArg = abs
		}
	} else {
		againstArg = ".git#ref=" + against
	}

	dir, targetArgs, err := bufTargetArgs(absPath)
	if err != nil {
		return err
	}
	args := append([]string{"breaking", "--against", againstArg}, targetArgs...)
	cmd := exec.Command(bufPath, args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("buf breaking failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// bufTargetArgs computes the buf working directory and input/selector args for
// validating the proto schema at absPath. buf v2 wants a workspace or module as
// the build input, with sub-targets selected via --path:
//   - absPath is the workspace root -> run in root, no selector
//   - absPath is a declared module  -> run in root, positional <module> input
//   - absPath is inside a module    -> run in root, --path <rel> selector
func bufTargetArgs(absPath string) (dir string, args []string, err error) {
	root, rel, err := bufRootAndPath(absPath)
	if err != nil {
		return "", nil, err
	}
	if rel == "." {
		return root, nil, nil
	}
	for _, m := range bufModulePaths(root) {
		if rel == m {
			return root, []string{rel}, nil
		}
	}
	return root, []string{"--path", rel}, nil
}

// bufModulePaths returns module directory paths (slash-separated, relative to
// root) declared by the buf workspace config: buf.work.yaml "directories" or
// buf.yaml v2 "modules[].path". Returns nil when none are declared (e.g. a v1
// single-module buf.yaml whose module is the workspace root itself).
func bufModulePaths(root string) []string {
	for _, name := range []string{"buf.work.yaml", "buf.work.yml"} {
		if dirs := bufWorkDirectories(filepath.Join(root, name)); dirs != nil {
			return dirs
		}
	}
	for _, name := range []string{"buf.yaml", "buf.yml"} {
		if mods := bufYAMLModulePaths(filepath.Join(root, name)); mods != nil {
			return mods
		}
	}
	return nil
}

func bufYAMLModulePaths(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var cfg struct {
		Modules []struct {
			Path string `yaml:"path"`
		} `yaml:"modules"`
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil
	}
	var out []string
	for _, m := range cfg.Modules {
		if m.Path != "" {
			out = append(out, filepath.ToSlash(filepath.Clean(m.Path)))
		}
	}
	return out
}

func bufWorkDirectories(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var cfg struct {
		Directories []string `yaml:"directories"`
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil
	}
	var out []string
	for _, d := range cfg.Directories {
		if d != "" {
			out = append(out, filepath.ToSlash(filepath.Clean(d)))
		}
	}
	return out
}

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
