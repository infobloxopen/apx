package publisher

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// IdempotencyResult describes whether a release is new, already done, or conflicting.
type IdempotencyResult string

const (
	// ReleaseNew means no previous release exists for this version.
	ReleaseNew IdempotencyResult = "new"

	// ReleaseAlreadyPublished means the exact same content was already
	// published at this version. Safe to skip.
	ReleaseAlreadyPublished IdempotencyResult = "already-published"

	// ReleaseConflict means this version was already published with
	// different content. The caller must choose a different version.
	ReleaseConflict IdempotencyResult = "conflict"
)

// CheckIdempotency determines whether a release at the given tag is new,
// already published, or conflicting.
//
// It checks:
//  1. Whether the git tag already exists
//  2. If it does, whether the content hash matches
//
// The contentDir is the directory containing the files to be published.
// The repoPath is the root of the git repository.
func CheckIdempotency(repoPath, tag, contentDir string) (IdempotencyResult, error) {
	tm := NewTagManager(repoPath, "")

	exists, err := tm.TagExists(tag)
	if err != nil {
		return "", fmt.Errorf("checking tag existence: %w", err)
	}

	if !exists {
		return ReleaseNew, nil
	}

	// Tag exists. Compute current content hash and compare with what's at the tag.
	currentHash, err := HashDirectory(contentDir)
	if err != nil {
		return "", fmt.Errorf("hashing content directory: %w", err)
	}

	tagHash, err := HashGitTreeAtTag(repoPath, tag, contentDir)
	if err != nil {
		// If we can't read the tag's content (e.g. not a reachable tree),
		// treat it as a conflict to be safe.
		return ReleaseConflict, nil
	}

	if currentHash == tagHash {
		return ReleaseAlreadyPublished, nil
	}

	return ReleaseConflict, nil
}

// HashDirectory computes a deterministic SHA-256 hash of a directory's file
// contents. Files are sorted lexicographically by relative path to ensure
// determinism. Only regular files are hashed.
func HashDirectory(dir string) (string, error) {
	h := sha256.New()

	var files []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(dir, path)
		if relErr != nil {
			return relErr
		}
		files = append(files, rel)
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("walking directory %s: %w", dir, err)
	}

	sort.Strings(files)

	for _, f := range files {
		// Write filename as boundary
		fmt.Fprintf(h, "file:%s\n", f)

		data, err := os.ReadFile(filepath.Join(dir, f))
		if err != nil {
			return "", fmt.Errorf("reading %s: %w", f, err)
		}
		h.Write(data)
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// HashGitTreeAtTag computes the same hash but from the git tree at a given tag.
// The subDir parameter scopes the hash to a subdirectory within the tree.
func HashGitTreeAtTag(repoPath, tag, subDir string) (string, error) {
	// Use git ls-tree to list files at the tag, scoped to subDir
	relDir, err := filepath.Rel(repoPath, subDir)
	if err != nil {
		relDir = subDir
	}
	relDir = filepath.ToSlash(relDir)
	if relDir == "." {
		relDir = ""
	}

	args := []string{"ls-tree", "-r", "--name-only", tag}
	if relDir != "" {
		args = append(args, "--", relDir)
	}

	out, err := gitCommand(repoPath, args...)
	if err != nil {
		return "", fmt.Errorf("listing tree at %s: %w", tag, err)
	}

	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) == 0 || (len(lines) == 1 && lines[0] == "") {
		return "", fmt.Errorf("no files found at tag %s in %s", tag, relDir)
	}

	sort.Strings(lines)

	h := sha256.New()
	for _, line := range lines {
		var relPath string
		if relDir != "" {
			relPath = strings.TrimPrefix(line, relDir+"/")
		} else {
			relPath = line
		}
		fmt.Fprintf(h, "file:%s\n", relPath)

		content, err := gitCommand(repoPath, "show", tag+":"+line)
		if err != nil {
			return "", fmt.Errorf("reading %s at %s: %w", line, tag, err)
		}
		h.Write([]byte(content))
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// gitCommand runs a git command and returns its stdout.
func gitCommand(repoPath string, args ...string) (string, error) {
	cmd := newGitCmd(repoPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %w\n%s", strings.Join(args, " "), err, string(output))
	}
	return string(output), nil
}

// newGitCmd creates an exec.Cmd for git in the given repo directory.
func newGitCmd(repoPath string, args ...string) *exec.Cmd {
	cmd := exec.Command("git", args...)
	cmd.Dir = repoPath
	return cmd
}
