package githubauth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// APIBaseURL is the base URL for the GitHub REST API.
// Override in tests to point at httptest.Server.
var APIBaseURL = "https://api.github.com"

// Client is an authenticated GitHub REST API client.
type Client struct {
	token      string
	httpClient *http.Client
}

// NewClient creates a Client from a raw access token.
func NewClient(token string) *Client {
	return &Client{
		token:      token,
		httpClient: http.DefaultClient,
	}
}

// Token returns the raw access token (e.g. for GHCR Bearer auth).
func (c *Client) Token() string { return c.token }

// Get performs a GET request against the GitHub API.
func (c *Client) Get(path string) ([]byte, int, error) {
	return c.do("GET", path, nil)
}

// Post performs a POST request with a JSON body.
func (c *Client) Post(path string, body interface{}) ([]byte, int, error) {
	return c.doJSON("POST", path, body)
}

// Put performs a PUT request with a JSON body.
func (c *Client) Put(path string, body interface{}) ([]byte, int, error) {
	return c.doJSON("PUT", path, body)
}

// Patch performs a PATCH request with a JSON body.
func (c *Client) Patch(path string, body interface{}) ([]byte, int, error) {
	return c.doJSON("PATCH", path, body)
}

// Delete performs a DELETE request.
func (c *Client) Delete(path string) ([]byte, int, error) {
	return c.do("DELETE", path, nil)
}

// GetPaginated follows GitHub's Link-header pagination and returns all
// items from every page, concatenated into a single slice.
// The endpoint must return a JSON array at the top level, OR a JSON object
// with a list field (the function auto-detects both forms).
func (c *Client) GetPaginated(path string) ([]json.RawMessage, error) {
	var all []json.RawMessage
	nextURL := c.fullURL(path)

	for nextURL != "" {
		req, err := http.NewRequest("GET", nextURL, nil)
		if err != nil {
			return nil, err
		}
		c.setHeaders(req)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode >= 400 {
			return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
		}

		// Try to unmarshal as array first.
		var items []json.RawMessage
		if err := json.Unmarshal(body, &items); err != nil {
			// GitHub sometimes wraps arrays in an object
			// e.g. { "installations": [...] } or { "secrets": [...] }.
			// Find the first array field.
			var obj map[string]json.RawMessage
			if err2 := json.Unmarshal(body, &obj); err2 != nil {
				return nil, fmt.Errorf("failed to parse paginated response: %w", err)
			}
			for _, v := range obj {
				if json.Unmarshal(v, &items) == nil && len(items) > 0 {
					break
				}
			}
		}
		all = append(all, items...)

		nextURL = parseNextLink(resp.Header.Get("Link"))
	}

	return all, nil
}

// do performs an HTTP request with the given method and optional body reader.
func (c *Client) do(method, path string, body io.Reader) ([]byte, int, error) {
	req, err := http.NewRequest(method, c.fullURL(path), body)
	if err != nil {
		return nil, 0, err
	}
	c.setHeaders(req)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	return respBody, resp.StatusCode, nil
}

// doJSON marshals body to JSON and performs the request.
func (c *Client) doJSON(method, path string, body interface{}) ([]byte, int, error) {
	var r io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to marshal request body: %w", err)
		}
		r = bytes.NewReader(data)
	}
	return c.do(method, path, r)
}

// setHeaders applies standard GitHub API headers.
func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
}

// fullURL converts a relative path (e.g. "/repos/o/r") to the full API URL.
// If the path is already a full URL (starts with http), it is returned as-is.
func (c *Client) fullURL(path string) string {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return APIBaseURL + path
}

// parseNextLink extracts the "next" URL from a GitHub Link header.
// Returns "" if there is no next page.
func parseNextLink(header string) string {
	if header == "" {
		return ""
	}
	for _, part := range strings.Split(header, ",") {
		part = strings.TrimSpace(part)
		if strings.Contains(part, `rel="next"`) {
			start := strings.Index(part, "<")
			end := strings.Index(part, ">")
			if start >= 0 && end > start {
				return part[start+1 : end]
			}
		}
	}
	return ""
}

// --- EnsureToken: high-level helper ---

// EnsureToken returns a valid access token for the given org, loading from
// cache or triggering the device flow if needed.
func EnsureToken(org string) (string, error) {
	tok, err := LoadToken(org)
	if err != nil {
		return "", fmt.Errorf("failed to load cached token: %w", err)
	}
	if tok != nil {
		return tok.AccessToken, nil
	}

	// No cached token — need to do device flow.
	clientID, err := ReadCache(org, "user-app-client-id")
	if err != nil {
		return "", fmt.Errorf("failed to read client ID: %w", err)
	}
	if clientID == "" {
		return "", fmt.Errorf("no GitHub user app configured for org %q — run `apx init canonical --setup-github` first", org)
	}

	tok, err = DeviceFlowLogin(clientID)
	if err != nil {
		return "", fmt.Errorf("device flow login failed: %w", err)
	}

	if err := SaveToken(org, tok); err != nil {
		return "", fmt.Errorf("failed to save token: %w", err)
	}

	return tok.AccessToken, nil
}
