package config

import (
	"strings"
	"testing"
)

// normalizeGitURL must never embed a credential — the returned URL is persisted
// in the cache checkout's .git/config, so a token there would leak.
func TestNormalizeGitURL_NeverEmbedsToken(t *testing.T) {
	t.Setenv("APX_GITHUB_TOKEN", "secret-token-value")

	cases := []struct{ in, want string }{
		{"github.com/acme/apis", "https://github.com/acme/apis.git"},
		{"github.com/acme/apis.git", "https://github.com/acme/apis.git"},
		{"https://github.com/acme/apis.git", "https://github.com/acme/apis.git"},
		{"git@github.com:acme/apis.git", "git@github.com:acme/apis.git"},
		{"/local/bare/repo.git", "/local/bare/repo.git"},
	}
	for _, c := range cases {
		got := normalizeGitURL(c.in)
		if got != c.want {
			t.Errorf("normalizeGitURL(%q) = %q, want %q", c.in, got, c.want)
		}
		if strings.Contains(got, "secret-token-value") {
			t.Errorf("normalizeGitURL(%q) leaked the token into the URL: %q", c.in, got)
		}
	}
}

// gitAuthArgs supplies the token as a transient -c http.extraHeader (not in the
// URL), only for github.com https clones, and only when a token is present.
func TestGitAuthArgs(t *testing.T) {
	t.Setenv("APX_GITHUB_TOKEN", "secret-token-value")

	// Non-github URL: no auth args regardless of token.
	if got := gitAuthArgs("https://gitlab.com/acme/apis.git"); got != nil {
		t.Errorf("gitAuthArgs(non-github) = %v, want nil", got)
	}
	if got := gitAuthArgs("/local/bare/repo.git"); got != nil {
		t.Errorf("gitAuthArgs(local path) = %v, want nil", got)
	}

	// github.com https with a token: a -c http.extraHeader arg carrying the
	// base64(x-access-token:TOKEN) credential, and the -c flag precedes it.
	got := gitAuthArgs("https://github.com/acme/apis.git")
	if len(got) != 2 || got[0] != "-c" {
		t.Fatalf("gitAuthArgs(github) = %v, want [-c http.extraHeader=...]", got)
	}
	if !strings.HasPrefix(got[1], "http.extraHeader=AUTHORIZATION: Basic ") {
		t.Fatalf("gitAuthArgs header malformed: %q", got[1])
	}
	// The raw token must not appear verbatim (it is base64-encoded).
	if strings.Contains(got[1], "secret-token-value") {
		t.Errorf("gitAuthArgs embedded the raw token: %q", got[1])
	}
}
