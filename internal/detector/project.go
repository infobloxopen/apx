package detector

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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
		Repo:      "apis",
		Languages: []string{"go"}, // default to go
	}

	// Try to detect org and repo from git remote origin URL
	if org, repo, err := DetectFromGitRemote("origin"); err == nil {
		if org != "" {
			defaults.Org = org
		}
		if repo != "" {
			defaults.Repo = repo
		}
	} else {
		// Fall back to filesystem path detection for org
		if org, err := detectOrgFromPath(); err == nil && org != "" {
			defaults.Org = org
		}
		// Fall back to directory name for repo
		if repo, err := detectRepoFromDir(); err == nil && repo != "" {
			defaults.Repo = repo
		}
	}

	// Try to detect languages from project files
	if languages, err := DetectLanguages(); err == nil && len(languages) > 0 {
		defaults.Languages = languages
	}

	return defaults, nil
}

// gitRemoteRegexps matches common git remote URL formats:
//
//	git@github.com:Org/repo.git       → Org, repo
//	ssh://git@github.com/Org/repo.git → Org, repo
//	https://github.com/Org/repo.git   → Org, repo
//	https://github.com/Org/repo       → Org, repo
var gitRemoteRegexps = []*regexp.Regexp{
	// SSH shorthand: git@host:org/repo.git
	regexp.MustCompile(`^[^@]+@[^:]+:([^/]+)/([^/]+?)(?:\.git)?$`),
	// SSH or HTTPS URL: scheme://host/org/repo.git
	regexp.MustCompile(`^(?:https?|ssh|git)://[^/]+/([^/]+)/([^/]+?)(?:\.git)?$`),
}

// DetectFromGitRemote parses the git remote URL for the given remote name
// and extracts the org (owner) and repo name.
func DetectFromGitRemote(remoteName string) (org, repo string, err error) {
	url, err := gitRemoteURL(remoteName)
	if err != nil {
		return "", "", err
	}
	return ParseGitRemoteURL(url)
}

// ParseGitRemoteURL extracts org and repo from a git remote URL string.
// Supports SSH shorthand (git@host:org/repo.git), SSH URLs, and HTTPS URLs.
func ParseGitRemoteURL(url string) (org, repo string, err error) {
	url = strings.TrimSpace(url)
	for _, re := range gitRemoteRegexps {
		m := re.FindStringSubmatch(url)
		if m != nil {
			return m[1], m[2], nil
		}
	}
	return "", "", fmt.Errorf("could not parse git remote URL: %s", url)
}

// gitRemoteURL runs `git remote get-url <name>` and returns the URL.
var gitRemoteURL = func(remoteName string) (string, error) {
	cmd := exec.Command("git", "remote", "get-url", remoteName)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get git remote %q: %w", remoteName, err)
	}
	return strings.TrimSpace(string(out)), nil
}

// detectOrgFromPath falls back to inspecting the filesystem path for a
// github.com/<org> pattern (works in GOPATH-style layouts).
func detectOrgFromPath() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	if strings.Contains(cwd, "github.com") {
		parts := strings.Split(cwd, string(filepath.Separator))
		for i, part := range parts {
			if part == "github.com" && i+1 < len(parts) {
				return parts[i+1], nil
			}
		}
	}

	return "", fmt.Errorf("could not detect org from path")
}

// detectRepoFromDir returns the current directory name as the repo name.
func detectRepoFromDir() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Base(cwd), nil
}

// DetectOrgFromGit is kept for backward compatibility. It tries git remote
// first, then falls back to filesystem path inspection.
func DetectOrgFromGit() (string, error) {
	if org, _, err := DetectFromGitRemote("origin"); err == nil && org != "" {
		return org, nil
	}
	return detectOrgFromPath()
}

// DetectRepoName detects the repository name from git remote origin,
// falling back to the current directory name.
func DetectRepoName() (string, error) {
	if _, repo, err := DetectFromGitRemote("origin"); err == nil && repo != "" {
		return repo, nil
	}
	return detectRepoFromDir()
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
