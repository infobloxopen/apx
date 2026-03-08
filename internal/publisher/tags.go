package publisher

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// TagManager handles tag creation and validation
type TagManager struct {
	repoPath  string
	tagFormat string
}

// NewTagManager creates a new tag manager
func NewTagManager(repoPath, tagFormat string) *TagManager {
	if tagFormat == "" {
		tagFormat = "{subdir}/v{version}"
	}
	return &TagManager{
		repoPath:  repoPath,
		tagFormat: tagFormat,
	}
}

// FormatTag formats a tag using the configured format
func (m *TagManager) FormatTag(subdir, version string) string {
	tag := m.tagFormat
	tag = strings.ReplaceAll(tag, "{subdir}", subdir)
	tag = strings.ReplaceAll(tag, "{version}", version)
	return tag
}

// ValidateVersion checks if a version string is valid semver
func (m *TagManager) ValidateVersion(version string) error {
	// Require 'v' prefix for version strings
	if len(version) == 0 || version[0] != 'v' {
		return fmt.Errorf("invalid version format: %s (expected semver like v1.2.3 with 'v' prefix)", version)
	}

	// Simplified semver validation
	semverRegex := regexp.MustCompile(`^v(\d+)\.(\d+)\.(\d+)(-[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?(\+[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?$`)
	if !semverRegex.MatchString(version) {
		return fmt.Errorf("invalid version format: %s (expected semver like v1.2.3)", version)
	}
	return nil
}

// CreateTag creates a git tag
func (m *TagManager) CreateTag(tag, message, commitHash string) error {
	cmd := exec.Command("git", "tag", "-a", tag, "-m", message, commitHash)
	cmd.Dir = m.repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create tag: %w\nOutput: %s", err, string(output))
	}
	return nil
}

// PushTag pushes a tag to remote
func (m *TagManager) PushTag(tag, remote string) error {
	if remote == "" {
		remote = "origin"
	}

	cmd := exec.Command("git", "push", remote, tag)
	cmd.Dir = m.repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to push tag: %w\nOutput: %s", err, string(output))
	}
	return nil
}

// TagExists checks if a tag already exists
func (m *TagManager) TagExists(tag string) (bool, error) {
	cmd := exec.Command("git", "tag", "-l", tag)
	cmd.Dir = m.repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("failed to list tags: %w", err)
	}

	tags := strings.TrimSpace(string(output))
	return tags != "", nil
}

// ListTags returns all tags matching the given pattern (glob).
// If pattern is empty, all tags are returned.
func (m *TagManager) ListTags(pattern string) ([]string, error) {
	args := []string{"tag", "-l"}
	if pattern != "" {
		args = append(args, pattern)
	}
	cmd := exec.Command("git", args...)
	cmd.Dir = m.repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to list tags: %w\nOutput: %s", err, string(output))
	}

	raw := strings.TrimSpace(string(output))
	if raw == "" {
		return nil, nil
	}
	tags := strings.Split(raw, "\n")

	// Trim whitespace from each tag
	result := make([]string, 0, len(tags))
	for _, t := range tags {
		t = strings.TrimSpace(t)
		if t != "" {
			result = append(result, t)
		}
	}
	return result, nil
}

// ListVersionsForAPI lists all tags that match an API ID prefix and extracts
// the version part. For example, given prefix "proto/payments/ledger/v1",
// a tag "proto/payments/ledger/v1/v1.2.3" yields "v1.2.3".
func (m *TagManager) ListVersionsForAPI(apiID string) ([]string, error) {
	pattern := apiID + "/v*"
	tags, err := m.ListTags(pattern)
	if err != nil {
		return nil, err
	}

	prefix := apiID + "/"
	var versions []string
	for _, tag := range tags {
		if !strings.HasPrefix(tag, prefix) {
			continue
		}
		ver := strings.TrimPrefix(tag, prefix)
		versions = append(versions, ver)
	}
	return versions, nil
}

// CreateAndPushTag creates and pushes a module tag
func (m *TagManager) CreateAndPushTag(subdir, version, commitHash string) (string, error) {
	// Validate version
	if err := m.ValidateVersion(version); err != nil {
		return "", err
	}

	// Format tag
	tag := m.FormatTag(subdir, version)

	// Check if tag exists
	exists, err := m.TagExists(tag)
	if err != nil {
		return "", err
	}
	if exists {
		return "", fmt.Errorf("tag already exists: %s", tag)
	}

	// Create tag
	message := fmt.Sprintf("Release %s version %s", subdir, version)
	if err := m.CreateTag(tag, message, commitHash); err != nil {
		return "", err
	}

	// Push tag
	if err := m.PushTag(tag, ""); err != nil {
		return tag, fmt.Errorf("tag created locally but push failed: %w", err)
	}

	return tag, nil
}
