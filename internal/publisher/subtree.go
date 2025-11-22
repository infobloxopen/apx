package publisher

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// SubtreePublisher handles git subtree split and push operations
type SubtreePublisher struct {
	repoPath string
}

// NewSubtreePublisher creates a new subtree publisher
func NewSubtreePublisher(repoPath string) *SubtreePublisher {
	return &SubtreePublisher{
		repoPath: repoPath,
	}
}

// Split performs a git subtree split for the specified module directory
func (p *SubtreePublisher) Split(moduleDir, branch string) (string, error) {
	absPath, err := filepath.Abs(filepath.Join(p.repoPath, moduleDir))
	if err != nil {
		return "", fmt.Errorf("failed to resolve module path: %w", err)
	}

	// Get relative path from repo root
	relPath := strings.TrimPrefix(absPath, p.repoPath+"/")

	// Run git subtree split
	cmd := exec.Command("git", "subtree", "split", "--prefix="+relPath, "-b", branch)
	cmd.Dir = p.repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git subtree split failed: %w\nOutput: %s", err, string(output))
	}

	commitHash := strings.TrimSpace(string(output))
	return commitHash, nil
}

// Push pushes the split subtree to a remote repository
func (p *SubtreePublisher) Push(branch, remoteURL string) error {
	cmd := exec.Command("git", "push", remoteURL, branch+":main")
	cmd.Dir = p.repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git push failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// PublishModule performs a complete publish: split and push
func (p *SubtreePublisher) PublishModule(moduleDir, remoteURL, version string) (string, error) {
	branchName := fmt.Sprintf("publish/%s", strings.ReplaceAll(moduleDir, "/", "-"))

	// Split subtree
	commitHash, err := p.Split(moduleDir, branchName)
	if err != nil {
		return "", fmt.Errorf("subtree split failed: %w", err)
	}

	// Push to remote
	if err := p.Push(branchName, remoteURL); err != nil {
		return "", fmt.Errorf("push failed: %w", err)
	}

	return commitHash, nil
}
