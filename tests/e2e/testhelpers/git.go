// Copyright 2025 Infoblox Inc.
// SPDX-License-Identifier: Apache-2.0

package testhelpers

import (
"context"
"fmt"
"os"
"os/exec"
"path/filepath"
"strings"
)

// GitRepo wraps git operations for testing
type GitRepo struct {
	Path      string
	RemoteURL string
}

// Clone clones a git repository to a local path
func Clone(ctx context.Context, url, path string) (*GitRepo, error) {
	cmd := exec.CommandContext(ctx, "git", "clone", url, path)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to clone %s: %w", url, err)
	}

	return &GitRepo{
		Path:      path,
		RemoteURL: url,
	}, nil
}

// Init initializes a new git repository
func InitRepo(path string) (*GitRepo, error) {
	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	cmd := exec.Command("git", "init", path)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to initialize git repo: %w", err)
	}

	return &GitRepo{Path: path}, nil
}

// SetRemote sets the remote URL for the repository
func (r *GitRepo) SetRemote(name, url string) error {
	cmd := exec.Command("git", "-C", r.Path, "remote", "add", name, url)
	if err := cmd.Run(); err != nil {
		// Try to set-url if remote already exists
		cmd = exec.Command("git", "-C", r.Path, "remote", "set-url", name, url)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set remote: %w", err)
		}
	}
	r.RemoteURL = url
	return nil
}

// WriteFile writes content to a file in the repository
func (r *GitRepo) WriteFile(relativePath, content string) error {
	fullPath := filepath.Join(r.Path, relativePath)
	dir := filepath.Dir(fullPath)
	
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// Add stages files for commit
func (r *GitRepo) Add(files ...string) error {
	args := append([]string{"-C", r.Path, "add"}, files...)
	cmd := exec.Command("git", args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add files: %w", err)
	}
	return nil
}

// Commit creates a commit with the given message
func (r *GitRepo) Commit(message string) error {
	cmd := exec.Command("git", "-C", r.Path, "commit", "-m", message)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}
	return nil
}

// Tag creates a tag at the current commit
func (r *GitRepo) Tag(tagName string) error {
	cmd := exec.Command("git", "-C", r.Path, "tag", tagName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create tag: %w", err)
	}
	return nil
}

// Push pushes commits to remote
func (r *GitRepo) Push(remote, branch string) error {
	cmd := exec.Command("git", "-C", r.Path, "push", remote, branch)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to push: %w\nOutput: %s", err, output)
	}
	return nil
}

// PushTags pushes tags to remote
func (r *GitRepo) PushTags(remote string) error {
	cmd := exec.Command("git", "-C", r.Path, "push", remote, "--tags")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to push tags: %w\nOutput: %s", err, output)
	}
	return nil
}

// CreateBranch creates a new branch
func (r *GitRepo) CreateBranch(branchName string) error {
	cmd := exec.Command("git", "-C", r.Path, "checkout", "-b", branchName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create branch: %w", err)
	}
	return nil
}

// Checkout switches to a branch or commit
func (r *GitRepo) Checkout(ref string) error {
	cmd := exec.Command("git", "-C", r.Path, "checkout", ref)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to checkout %s: %w", ref, err)
	}
	return nil
}

// GetCurrentCommit returns the current commit SHA
func (r *GitRepo) GetCurrentCommit() (string, error) {
	cmd := exec.Command("git", "-C", r.Path, "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current commit: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// GetLog returns git log output
func (r *GitRepo) GetLog(format string, maxCount int) (string, error) {
	args := []string{"-C", r.Path, "log"}
	if format != "" {
		args = append(args, fmt.Sprintf("--pretty=format:%s", format))
	}
	if maxCount > 0 {
		args = append(args, fmt.Sprintf("-n%d", maxCount))
	}

	cmd := exec.Command("git", args...)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get log: %w", err)
	}
	return string(output), nil
}

// ConfigureUser sets git user config for the repository
func (r *GitRepo) ConfigureUser(name, email string) error {
	cmd := exec.Command("git", "-C", r.Path, "config", "user.name", name)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set user.name: %w", err)
	}

	cmd = exec.Command("git", "-C", r.Path, "config", "user.email", email)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set user.email: %w", err)
	}

	return nil
}
