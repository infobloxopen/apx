package validator

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// toolSpec describes how to download a tool from GitHub releases.
type toolSpec struct {
	// Org/Repo on GitHub (e.g. "bufbuild/buf").
	repo string
	// archivePattern is a Go template-style pattern for the archive filename.
	// Supported placeholders: {version}, {os}, {arch}.
	// If empty, the tool is not downloadable.
	archivePattern string
	// binaryName inside the archive. If empty, defaults to the tool name.
	binaryName string
}

// toolRegistry maps tool names to their download specs.
var toolRegistry = map[string]toolSpec{
	"buf": {
		repo:           "bufbuild/buf",
		archivePattern: "buf-{OS}-{ARCH}.tar.gz",
		binaryName:     "buf",
	},
	"oasdiff": {
		repo:           "Tufin/oasdiff",
		archivePattern: "oasdiff_{VERSION}_{os}_{arch}.tar.gz",
		binaryName:     "oasdiff",
	},
}

// normalizeOS returns the OS string used in release asset names.
func normalizeOS(tool, goos string) string {
	switch goos {
	case "darwin":
		return "Darwin"
	case "linux":
		return "Linux"
	case "windows":
		return "Windows"
	default:
		return goos
	}
}

// normalizeArch returns the architecture string used in release asset names.
func normalizeArch(tool, goarch string) string {
	switch goarch {
	case "amd64":
		return "x86_64"
	case "arm64":
		return "aarch64"
	default:
		return goarch
	}
}

// expandPattern replaces placeholders in an archive pattern.
func expandPattern(pattern, version, goos, goarch string) string {
	osTitle := normalizeOS("", goos)
	archTitle := normalizeArch("", goarch)

	r := strings.NewReplacer(
		"{version}", strings.TrimPrefix(version, "v"),
		"{VERSION}", strings.TrimPrefix(version, "v"),
		"{os}", strings.ToLower(osTitle),
		"{OS}", osTitle,
		"{arch}", strings.ToLower(archTitle),
		"{ARCH}", archTitle,
	)
	return r.Replace(pattern)
}

// cacheDir returns the directory where downloaded tools are cached.
func cacheDir(name, version string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.TempDir()
	}
	return filepath.Join(home, ".apx", "tools", name, version)
}

// downloadTool downloads a tool binary from GitHub releases and caches it.
// Returns the path to the cached binary.
func downloadTool(name, version string) (string, error) {
	spec, ok := toolRegistry[name]
	if !ok {
		return "", fmt.Errorf("no download source registered for tool: %s", name)
	}

	dir := cacheDir(name, version)
	binName := spec.binaryName
	if binName == "" {
		binName = name
	}
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}
	binPath := filepath.Join(dir, binName)

	// Already cached?
	if _, err := os.Stat(binPath); err == nil {
		return binPath, nil
	}

	goos := runtime.GOOS
	goarch := runtime.GOARCH
	archive := expandPattern(spec.archivePattern, version, goos, goarch)
	url := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", spec.repo, version, archive)

	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("downloading %s: %w", name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("downloading %s: HTTP %d from %s", name, resp.StatusCode, url)
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("creating cache directory: %w", err)
	}

	if strings.HasSuffix(archive, ".tar.gz") || strings.HasSuffix(archive, ".tgz") {
		if err := extractFromTarGz(resp.Body, binName, binPath); err != nil {
			return "", fmt.Errorf("extracting %s from archive: %w", name, err)
		}
	} else {
		// Direct binary download
		if err := downloadToFile(resp.Body, binPath); err != nil {
			return "", fmt.Errorf("saving %s binary: %w", name, err)
		}
	}

	if err := os.Chmod(binPath, 0755); err != nil {
		return "", fmt.Errorf("making %s executable: %w", name, err)
	}

	return binPath, nil
}

// extractFromTarGz extracts a single file from a tar.gz archive.
func extractFromTarGz(r io.Reader, targetName, destPath string) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("opening gzip: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading tar: %w", err)
		}

		// Match by base name — archives may include directory prefixes.
		if filepath.Base(hdr.Name) == targetName && hdr.Typeflag == tar.TypeReg {
			out, err := os.Create(destPath)
			if err != nil {
				return fmt.Errorf("creating output file: %w", err)
			}
			defer out.Close()

			if _, err := io.Copy(out, tr); err != nil {
				return fmt.Errorf("writing output file: %w", err)
			}
			return nil
		}
	}

	return fmt.Errorf("binary %q not found in archive", targetName)
}

// downloadToFile writes a reader to a file.
func downloadToFile(r io.Reader, path string) error {
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, r)
	return err
}
