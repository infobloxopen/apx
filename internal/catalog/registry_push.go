package catalog

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// PushOptions configures catalog publishing to a registry.
type PushOptions struct {
	Org  string // GitHub org
	Repo string // canonical repo name
	Host string // registry host (empty → ghcr.io)
	Tag  string // image tag (empty → "latest")

	// GHTokenFn overrides the default gh-auth-token function.
	GHTokenFn func() (string, error)

	// HTTPClient overrides the default HTTP client.
	HTTPClient *http.Client
}

func (o *PushOptions) host() string {
	if o.Host != "" {
		return o.Host
	}
	return ghcrHost
}

func (o *PushOptions) tag() string {
	if o.Tag != "" {
		return o.Tag
	}
	return "latest"
}

func (o *PushOptions) imageRef() string {
	return fmt.Sprintf("%s/%s%s", o.Org, o.Repo, CatalogImageSuffix)
}

func (o *PushOptions) httpClient() *http.Client {
	if o.HTTPClient != nil {
		return o.HTTPClient
	}
	return http.DefaultClient
}

func (o *PushOptions) ghToken() (string, error) {
	if o.GHTokenFn != nil {
		return o.GHTokenFn()
	}
	return ghAuthToken()
}

// PushCatalog pushes a catalog to the OCI registry as a single-layer artifact.
func PushCatalog(cat *Catalog, opts PushOptions) error {
	token, err := opts.ghToken()
	if err != nil {
		return fmt.Errorf("registry auth: %w", err)
	}

	ref := opts.imageRef()
	host := opts.host()
	tag := opts.tag()
	client := opts.httpClient()

	// 1. Marshal catalog to YAML
	yamlData, err := yaml.Marshal(cat)
	if err != nil {
		return fmt.Errorf("marshal catalog: %w", err)
	}

	// 2. Create tar.gz layer containing catalog.yaml
	layerData, err := createTarGz("catalog.yaml", yamlData)
	if err != nil {
		return fmt.Errorf("create layer: %w", err)
	}
	layerDigest := fmt.Sprintf("sha256:%x", sha256.Sum256(layerData))
	layerSize := int64(len(layerData))

	// 3. Create an empty config blob (OCI convention)
	configData := []byte("{}")
	configDigest := fmt.Sprintf("sha256:%x", sha256.Sum256(configData))
	configSize := int64(len(configData))

	// 4. Upload the layer blob
	if err := uploadBlob(client, host, ref, token, layerDigest, layerData); err != nil {
		return fmt.Errorf("upload layer: %w", err)
	}

	// 5. Upload the config blob
	if err := uploadBlob(client, host, ref, token, configDigest, configData); err != nil {
		return fmt.Errorf("upload config: %w", err)
	}

	// 6. Build and push the manifest
	manifest := ociManifest{
		SchemaVersion: 2,
		MediaType:     "application/vnd.oci.image.manifest.v1+json",
		Config: ociDescriptor{
			MediaType: "application/vnd.oci.image.config.v1+json",
			Digest:    configDigest,
			Size:      configSize,
		},
		Layers: []ociDescriptor{
			{
				MediaType: "application/vnd.oci.image.layer.v1.tar+gzip",
				Digest:    layerDigest,
				Size:      layerSize,
				Annotations: map[string]string{
					"org.opencontainers.image.title": "catalog.yaml",
				},
			},
		},
		Annotations: map[string]string{
			"org.opencontainers.image.source":  fmt.Sprintf("https://github.com/%s/%s", cat.Org, cat.Repo),
			"org.opencontainers.image.created": time.Now().UTC().Format(time.RFC3339),
			"dev.apx.catalog.org":              cat.Org,
			"dev.apx.catalog.repo":             cat.Repo,
			"dev.apx.catalog.module_count":     fmt.Sprintf("%d", len(cat.Modules)),
		},
	}

	if err := pushManifest(client, host, ref, tag, token, &manifest); err != nil {
		return fmt.Errorf("push manifest: %w", err)
	}

	return nil
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

// uploadBlob uploads a blob to the registry using the monolithic POST+PUT flow.
func uploadBlob(client *http.Client, host, ref, token, digest string, data []byte) error {
	// Start upload — POST to get an upload URL
	postURL := fmt.Sprintf("https://%s/v2/%s/blobs/uploads/", host, ref)
	req, err := http.NewRequest("POST", postURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("initiate blob upload: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("blob upload init returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	// Get the upload location
	location := resp.Header.Get("Location")
	if location == "" {
		return fmt.Errorf("blob upload init did not return Location header")
	}

	// Complete upload — PUT with digest query parameter
	sep := "?"
	if strings.Contains(location, "?") {
		sep = "&"
	}
	putURL := location + sep + "digest=" + digest

	putReq, err := http.NewRequest("PUT", putURL, bytes.NewReader(data))
	if err != nil {
		return err
	}
	putReq.Header.Set("Authorization", "Bearer "+token)
	putReq.Header.Set("Content-Type", "application/octet-stream")
	putReq.ContentLength = int64(len(data))

	putResp, err := client.Do(putReq)
	if err != nil {
		return fmt.Errorf("complete blob upload: %w", err)
	}
	defer putResp.Body.Close()

	if putResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(putResp.Body)
		return fmt.Errorf("blob upload PUT returned HTTP %d: %s", putResp.StatusCode, string(body))
	}

	return nil
}

// pushManifest pushes an OCI manifest to the registry.
func pushManifest(client *http.Client, host, ref, tag, token string, manifest *ociManifest) error {
	body, err := json.Marshal(manifest)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://%s/v2/%s/manifests/%s", host, ref, tag)
	req, err := http.NewRequest("PUT", url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/vnd.oci.image.manifest.v1+json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("push manifest: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("manifest PUT returned HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}
