package githubauth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestServer(t *testing.T, handler http.Handler) (*Client, func()) {
	t.Helper()
	server := httptest.NewServer(handler)
	orig := APIBaseURL
	APIBaseURL = server.URL
	client := NewClient("test-token-123")
	return client, func() {
		APIBaseURL = orig
		server.Close()
	}
}

func TestClient_Get(t *testing.T) {
	client, cleanup := setupTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/repos/org/repo", r.URL.Path)
		assert.Equal(t, "Bearer test-token-123", r.Header.Get("Authorization"))
		assert.Equal(t, "application/vnd.github+json", r.Header.Get("Accept"))
		assert.Equal(t, "2022-11-28", r.Header.Get("X-GitHub-Api-Version"))
		w.WriteHeader(200)
		fmt.Fprint(w, `{"id": 42}`)
	}))
	defer cleanup()

	body, status, err := client.Get("/repos/org/repo")
	require.NoError(t, err)
	assert.Equal(t, 200, status)
	assert.Contains(t, string(body), `"id"`)
}

func TestClient_Post(t *testing.T) {
	client, cleanup := setupTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var reqBody map[string]string
		json.NewDecoder(r.Body).Decode(&reqBody)
		assert.Equal(t, "main", reqBody["base"])

		w.WriteHeader(201)
		fmt.Fprint(w, `{"number": 1}`)
	}))
	defer cleanup()

	body, status, err := client.Post("/repos/org/repo/pulls", map[string]string{
		"title": "test PR",
		"head":  "feature",
		"base":  "main",
	})
	require.NoError(t, err)
	assert.Equal(t, 201, status)
	assert.Contains(t, string(body), `"number"`)
}

func TestClient_Put(t *testing.T) {
	client, cleanup := setupTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PUT", r.Method)
		w.WriteHeader(204)
	}))
	defer cleanup()

	_, status, err := client.Put("/repos/org/repo/branches/main/protection", map[string]bool{
		"enforce_admins": true,
	})
	require.NoError(t, err)
	assert.Equal(t, 204, status)
}

func TestClient_Delete(t *testing.T) {
	client, cleanup := setupTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		w.WriteHeader(204)
	}))
	defer cleanup()

	_, status, err := client.Delete("/repos/org/repo")
	require.NoError(t, err)
	assert.Equal(t, 204, status)
}

func TestClient_GetPaginated_Array(t *testing.T) {
	page := 0
	client, cleanup := setupTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page++
		if page == 1 {
			w.Header().Set("Link", fmt.Sprintf(`<%s/next>; rel="next"`, APIBaseURL))
			fmt.Fprint(w, `[{"id":1},{"id":2}]`)
		} else {
			fmt.Fprint(w, `[{"id":3}]`)
		}
	}))
	defer cleanup()

	items, err := client.GetPaginated("/orgs/org/packages")
	require.NoError(t, err)
	assert.Len(t, items, 3)
}

func TestClient_GetPaginated_WrappedObject(t *testing.T) {
	client, cleanup := setupTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"total_count":2,"installations":[{"id":1},{"id":2}]}`)
	}))
	defer cleanup()

	items, err := client.GetPaginated("/orgs/org/installations")
	require.NoError(t, err)
	assert.Len(t, items, 2)
}

func TestClient_Token(t *testing.T) {
	c := NewClient("my-secret-token")
	assert.Equal(t, "my-secret-token", c.Token())
}

func TestParseNextLink(t *testing.T) {
	tests := []struct {
		header string
		want   string
	}{
		{
			header: `<https://api.github.com/repos?page=2>; rel="next", <https://api.github.com/repos?page=5>; rel="last"`,
			want:   "https://api.github.com/repos?page=2",
		},
		{
			header: `<https://api.github.com/repos?page=5>; rel="last"`,
			want:   "",
		},
		{
			header: "",
			want:   "",
		},
	}

	for _, tt := range tests {
		got := parseNextLink(tt.header)
		assert.Equal(t, tt.want, got)
	}
}

func TestEnsureToken_CachedToken(t *testing.T) {
	restore := withTempConfigDir(t)
	defer restore()

	// Pre-cache a token.
	require.NoError(t, SaveToken("myorg", &Token{AccessToken: "cached-token"}))

	tok, err := EnsureToken("myorg")
	require.NoError(t, err)
	assert.Equal(t, "cached-token", tok)
}

func TestEnsureToken_NoClientID(t *testing.T) {
	restore := withTempConfigDir(t)
	defer restore()

	_, err := EnsureToken("noorg")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no GitHub user app configured")
}
