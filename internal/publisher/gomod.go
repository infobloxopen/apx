package publisher

import (
	"fmt"
	"path/filepath"
	"strings"
)

// GenerateGoMod creates a minimal go.mod file for a released API module.
//
// The generated go.mod contains only the module directive and go version.
// CI should run `go mod tidy` to populate require directives.
//
// Parameters:
//   - modulePath: full Go module path (e.g. "github.com/acme/apis/proto/payments/ledger")
//   - goVersion: Go version string (e.g. "1.21")
func GenerateGoMod(modulePath string, goVersion string) ([]byte, error) {
	if modulePath == "" {
		return nil, fmt.Errorf("module path is required")
	}
	if goVersion == "" {
		goVersion = "1.21"
	}
	// Strip "go" prefix if accidentally passed (e.g. "go1.21" → "1.21")
	goVersion = strings.TrimPrefix(goVersion, "go")

	content := fmt.Sprintf("module %s\n\ngo %s\n", modulePath, goVersion)
	return []byte(content), nil
}

// goModTargetDir returns the directory that must hold the generated go.mod for
// a released Go module, given destDir — the canonical version directory that
// holds the version's sources (e.g. ".../iam-identity/v2") — and the module
// path.
//
// Placement follows Go semantic-import-versioning:
//
//   - v2+ module paths carry a "/vN" suffix and are rooted IN their own version
//     directory, so the go.mod belongs in destDir and its module path matches
//     its directory (".../iam-identity/v2/go.mod" → "module .../iam-identity/v2").
//   - v0/v1 module paths have no version suffix (Go rejects "/v0" and "/v1"
//     suffixes) and are rooted at the family root, destDir's parent. Their code
//     still lives in the vN/ subtree and imports as <module>/vN
//     (".../iam-identity/go.mod" → "module .../iam-identity", sources under
//     ".../iam-identity/v1/").
//
// Deriving the location from the module path — rather than unconditionally
// using destDir's parent — keeps every release PR confined to its own version
// subtree. Two concurrent releases of different major versions of one family
// therefore never both write the shared family-root go.mod, which is the
// add/add conflict fixed in apx#27.
func goModTargetDir(destDir, module string) string {
	verSeg := filepath.Base(destDir) // the version segment, e.g. "v1", "v2"
	if strings.HasSuffix(module, "/"+verSeg) {
		return destDir
	}
	return filepath.Dir(destDir)
}

// ParseGoModModule extracts the module path from an existing go.mod file's contents.
// Returns the module path or an error if the module directive is not found.
func ParseGoModModule(content []byte) (string, error) {
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			mod := strings.TrimPrefix(line, "module ")
			mod = strings.TrimSpace(mod)
			return mod, nil
		}
	}
	return "", fmt.Errorf("no module directive found in go.mod")
}
