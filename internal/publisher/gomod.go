package publisher

import (
	"fmt"
	"strings"
)

// GenerateGoMod creates a minimal go.mod file for a published API module.
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
