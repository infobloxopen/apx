package publisher

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// PullRequestCreator handles GitHub/Gitea PR creation
type PullRequestCreator struct {
	baseURL string
	token   string
	org     string
	repo    string
	client  *http.Client
}

// NewPullRequestCreator creates a new PR creator
func NewPullRequestCreator(baseURL, token, org, repo string) *PullRequestCreator {
	return &PullRequestCreator{
		baseURL: baseURL,
		token:   token,
		org:     org,
		repo:    repo,
		client:  &http.Client{},
	}
}

// PRRequest represents a pull request creation request
type PRRequest struct {
	Title string `json:"title"`
	Head  string `json:"head"`
	Base  string `json:"base"`
	Body  string `json:"body,omitempty"`
}

// PRResponse represents a pull request response
type PRResponse struct {
	Number  int    `json:"number"`
	HTMLURL string `json:"html_url"`
	State   string `json:"state"`
}

// CreatePR creates a pull request
func (c *PullRequestCreator) CreatePR(title, head, base, body string) (*PRResponse, error) {
	// Prepare request
	prReq := PRRequest{
		Title: title,
		Head:  head,
		Base:  base,
		Body:  body,
	}

	jsonData, err := json.Marshal(prReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal PR request: %w", err)
	}

	// GitHub/Gitea API endpoint
	url := fmt.Sprintf("%s/repos/%s/%s/pulls", c.baseURL, c.org, c.repo)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "token "+c.token)

	// Send request
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("PR creation failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse response
	var prResp PRResponse
	if err := json.Unmarshal(bodyBytes, &prResp); err != nil {
		return nil, fmt.Errorf("failed to parse PR response: %w", err)
	}

	return &prResp, nil
}

// CreateModulePR creates a PR for a module publish
func (c *PullRequestCreator) CreateModulePR(module, version, branch string) (*PRResponse, error) {
	title := fmt.Sprintf("Publish %s version %s", module, version)
	body := fmt.Sprintf("Automated publish of module `%s` at version `%s`", module, version)

	return c.CreatePR(title, branch, "main", body)
}
