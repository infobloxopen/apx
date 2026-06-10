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
	// assetName returns the release asset filename for a version/os/arch
	// combination, or "" when that platform is not published. Asset naming
	// is tool-specific (each project's release pipeline picks its own OS and
	// arch spellings), so it cannot be derived from a shared pattern.
	assetName func(version, goos, goarch string) string
	// binaryName inside the archive. If empty, defaults to the tool name.
	binaryName string
}

// toolRegistry maps tool names to their download specs.
var toolRegistry = map[string]toolSpec{
	"buf": {
		repo:       "bufbuild/buf",
		assetName:  bufAssetName,
		binaryName: "buf",
	},
	"oasdiff": {
		repo:       "Tufin/oasdiff",
		assetName:  oasdiffAssetName,
		binaryName: "oasdiff",
	},
}

// bufAssetName maps to buf's release assets: buf-{OS}-{arch}[.tar.gz|.exe].
// buf spells arm64 as "aarch64" on Linux but "arm64" on Darwin/Windows, and
// ships Windows binaries as bare .exe files rather than archives.
func bufAssetName(_, goos, goarch string) string {
	var osName string
	switch goos {
	case "darwin":
		osName = "Darwin"
	case "linux":
		osName = "Linux"
	case "windows":
		osName = "Windows"
	default:
		return ""
	}

	var arch string
	switch goarch {
	case "amd64":
		arch = "x86_64"
	case "arm64":
		arch = "arm64"
		if goos == "linux" {
			arch = "aarch64"
		}
	default:
		return ""
	}

	if goos == "windows" {
		return fmt.Sprintf("buf-%s-%s.exe", osName, arch)
	}
	return fmt.Sprintf("buf-%s-%s.tar.gz", osName, arch)
}

// oasdiffAssetName maps to oasdiff's release assets:
// oasdiff_{version}_{os}_{arch}.tar.gz with Go-style arch names, except
// macOS which ships a single universal "darwin_all" binary.
func oasdiffAssetName(version, goos, goarch string) string {
	v := strings.TrimPrefix(version, "v")
	switch goos {
	case "darwin":
		return fmt.Sprintf("oasdiff_%s_darwin_all.tar.gz", v)
	case "linux", "windows":
		if goarch != "amd64" && goarch != "arm64" {
			return ""
		}
		return fmt.Sprintf("oasdiff_%s_%s_%s.tar.gz", v, goos, goarch)
	default:
		return ""
	}
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
	archive := spec.assetName(version, goos, goarch)
	if archive == "" {
		return "", fmt.Errorf("%s has no published release asset for %s/%s", name, goos, goarch)
	}
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
