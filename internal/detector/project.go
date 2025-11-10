package detector

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ProjectDefaults holds default values for project initialization
type ProjectDefaults struct {
	Org       string
	Repo      string
	Languages []string
}

// GetSmartDefaults detects default values based on the current directory context
func GetSmartDefaults() (*ProjectDefaults, error) {
	defaults := &ProjectDefaults{
		Org:       "your-org-name",
		Repo:      "your-apis-repo",
		Languages: []string{"go"}, // default to go
	}

	// Try to detect org from git remote
	if org, err := DetectOrgFromGit(); err == nil && org != "" {
		defaults.Org = org
	}

	// Try to detect repo name from current directory
	if repo, err := DetectRepoName(); err == nil && repo != "" {
		defaults.Repo = repo
	}

	// Try to detect languages from project files
	if languages, err := DetectLanguages(); err == nil && len(languages) > 0 {
		defaults.Languages = languages
	}

	return defaults, nil
}

// DetectOrgFromGit tries to extract organization name from git remotes
func DetectOrgFromGit() (string, error) {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Look for github.com in the path
	if strings.Contains(cwd, "github.com") {
		parts := strings.Split(cwd, string(filepath.Separator))
		for i, part := range parts {
			if part == "github.com" && i+1 < len(parts) {
				return parts[i+1], nil
			}
		}
	}

	// TODO: Could also try parsing git remote origin URL
	return "", fmt.Errorf("could not detect org from git")
}

// DetectRepoName tries to extract repository name from current directory or git
func DetectRepoName() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Use the current directory name as repo name
	return filepath.Base(cwd), nil
}

// DetectLanguages tries to detect target languages from project files
func DetectLanguages() ([]string, error) {
	var languages []string

	// Check for Go
	if _, err := os.Stat("go.mod"); err == nil {
		languages = append(languages, "go")
	}

	// Check for Python
	if _, err := os.Stat("requirements.txt"); err == nil {
		languages = append(languages, "python")
	} else if _, err := os.Stat("pyproject.toml"); err == nil {
		languages = append(languages, "python")
	}

	// Check for Java
	if _, err := os.Stat("pom.xml"); err == nil {
		languages = append(languages, "java")
	} else if _, err := os.Stat("build.gradle"); err == nil {
		languages = append(languages, "java")
	}

	// Default to go if nothing detected
	if len(languages) == 0 {
		languages = []string{"go"}
	}

	return languages, nil
}

// IsInteractive checks if we're running in an interactive terminal
func IsInteractive() bool {
	return os.Getenv("CI") == "" && os.Getenv("TERM") != "dumb"
}
