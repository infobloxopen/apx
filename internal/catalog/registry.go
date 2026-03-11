package catalog

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/infobloxopen/apx/pkg/githubauth"
)

// ghcrHost is the default GHCR registry host.
const ghcrHost = "ghcr.io"

// CatalogImageSuffix is appended to the repo name to form the GHCR package name.
// For canonical repo "apis" → GHCR package "apis-catalog".
const CatalogImageSuffix = "-catalog"

// RegistrySource pulls catalog.yaml from an OCI artifact on GHCR.
type RegistrySource struct {
	Org  string // GitHub org (e.g. "acme")
	Repo string // canonical repo name (e.g. "apis")
	Host string // registry host (empty → ghcr.io)
	Tag  string // image tag (empty → "latest")

	// GHTokenFn overrides the default gh-auth-token function.
	// Exists for testability — production code leaves this nil.
	GHTokenFn func() (string, error)

	// HTTPClient overrides the default HTTP client.
	// Exists for testability — production code leaves this nil.
	HTTPClient *http.Client
}

// Load pulls the catalog artifact from the registry and returns the catalog.
func (r *RegistrySource) Load() (*Catalog, error) {
	token, err := r.ghToken()
	if err != nil {
		return nil, fmt.Errorf("registry auth: %w", err)
	}

	ref := r.imageRef()
	tag := r.tag()
	client := r.httpClient()

	// 1. Pull the manifest
	manifest, err := r.pullManifest(client, ref, tag, token)
	if err != nil {
		return nil, err
	}

	// 2. Find the catalog layer
	if len(manifest.Layers) == 0 {
		return nil, fmt.Errorf("OCI manifest for %s:%s has no layers", ref, tag)
	}

	// Use the first layer — our artifact has a single data layer
	layerDigest := manifest.Layers[0].Digest
	layerMediaType := manifest.Layers[0].MediaType

	// 3. Pull the blob
	data, err := r.pullBlob(client, ref, layerDigest, token)
	if err != nil {
		return nil, err
	}

	// 4. Extract catalog.yaml from the blob
	return r.extractCatalog(data, layerMediaType)
}

// Name returns a human-readable identifier.
func (r *RegistrySource) Name() string {
	// OCI image references must be lowercase.
	return fmt.Sprintf("%s/%s/%s%s:%s",
		r.host(),
		strings.ToLower(r.Org),
		strings.ToLower(r.Repo),
		CatalogImageSuffix,
		r.tag())
}

// ---------------------------------------------------------------------------
// OCI manifest types (minimal subset)
// ---------------------------------------------------------------------------

// ociManifest is a minimal OCI image manifest.
type ociManifest struct {
	SchemaVersion int               `json:"schemaVersion"`
	MediaType     string            `json:"mediaType"`
	Config        ociDescriptor     `json:"config"`
	Layers        []ociDescriptor   `json:"layers"`
	Annotations   map[string]string `json:"annotations,omitempty"`
}

// ociDescriptor describes a content-addressable blob.
type ociDescriptor struct {
	MediaType   string            `json:"mediaType"`
	Digest      string            `json:"digest"`
	Size        int64             `json:"size"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

func (r *RegistrySource) host() string {
	if r.Host != "" {
		return r.Host
	}
	return ghcrHost
}

func (r *RegistrySource) tag() string {
	if r.Tag != "" {
		return r.Tag
	}
	return "latest"
}

func (r *RegistrySource) imageRef() string {
	// OCI image references must be lowercase.
	return fmt.Sprintf("%s/%s%s",
		strings.ToLower(r.Org),
		strings.ToLower(r.Repo),
		CatalogImageSuffix)
}

func (r *RegistrySource) httpClient() *http.Client {
	if r.HTTPClient != nil {
		return r.HTTPClient
	}
	return http.DefaultClient
}

// ghToken gets a GitHub token for GHCR authentication.
func (r *RegistrySource) ghToken() (string, error) {
	if r.GHTokenFn != nil {
		return r.GHTokenFn()
	}
	return ghAuthToken()
}

// ghAuthToken returns a GitHub token for GHCR auth.
// Uses the githubauth package (device flow + token cache) instead of `gh auth token`.
var ghAuthToken = ghAuthTokenReal

func ghAuthTokenReal() (string, error) {
	org, err := githubauth.DetectOrg()
	if err != nil {
		return "", fmt.Errorf("cannot detect GitHub org: %w", err)
	}
	token, err := githubauth.EnsureToken(org)
	if err != nil {
		return "", fmt.Errorf("GitHub auth failed: %w", err)
	}
	return token, nil
}

// pullManifest fetches the OCI manifest for the given image reference and tag.
func (r *RegistrySource) pullManifest(client *http.Client, ref, tag, token string) (*ociManifest, error) {
	url := fmt.Sprintf("https://%s/v2/%s/manifests/%s", r.host(), ref, tag)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create manifest request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	// Accept both OCI and Docker manifest types
	req.Header.Set("Accept", strings.Join([]string{
		"application/vnd.oci.image.manifest.v1+json",
		"application/vnd.docker.distribution.manifest.v2+json",
	}, ", "))

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch manifest %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("manifest %s returned HTTP %d", url, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read manifest body: %w", err)
	}

	var m ociManifest
	if err := json.Unmarshal(body, &m); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	return &m, nil
}

// pullBlob downloads a blob by digest from the registry.
func (r *RegistrySource) pullBlob(client *http.Client, ref, digest, token string) ([]byte, error) {
	url := fmt.Sprintf("https://%s/v2/%s/blobs/%s", r.host(), ref, digest)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create blob request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch blob %s: %w", digest, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("blob %s returned HTTP %d", digest, resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read blob body: %w", err)
	}
	return data, nil
}

// extractCatalog extracts catalog.yaml from the blob data.
// If the media type indicates gzip/tar, it decompresses and extracts.
// Otherwise it tries to parse the raw bytes as YAML directly.
func (r *RegistrySource) extractCatalog(data []byte, mediaType string) (*Catalog, error) {
	// Try to decompress as tar.gz first (OCI layer convention)
	if isGzipped(data) {
		yamlData, err := extractFromTarGz(data, "catalog.yaml")
		if err == nil {
			data = yamlData
		}
		// If extraction fails, fall through to try raw YAML
	}

	var cat Catalog
	if err := yaml.Unmarshal(data, &cat); err != nil {
		return nil, fmt.Errorf("parse catalog from OCI layer: %w", err)
	}
	return &cat, nil
}

// isGzipped checks if data starts with the gzip magic number.
func isGzipped(data []byte) bool {
	return len(data) >= 2 && data[0] == 0x1f && data[1] == 0x8b
}

// createTarGz creates a tar.gz archive containing a single file.
func createTarGz(name string, data []byte) ([]byte, error) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	hdr := &tar.Header{
		Name: name,
		Mode: 0644,
		Size: int64(len(data)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return nil, err
	}
	if _, err := tw.Write(data); err != nil {
		return nil, err
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	if err := gw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// extractFromTarGz extracts a named file from a tar.gz archive.
func extractFromTarGz(data []byte, name string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		// Match by exact name or by base name
		if hdr.Name == name || strings.HasSuffix(hdr.Name, "/"+name) {
			return io.ReadAll(tr)
		}
	}
	return nil, fmt.Errorf("file %q not found in tar archive", name)
}
