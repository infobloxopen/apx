package opclass

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAnnotate(t *testing.T) {
	tests := []struct {
		name  string
		in    string
		want  string
		added int
	}{
		{
			name: "block style keeps comments, quoting and order",
			in: `openapi: 3.0.3   # keep me
paths:
  /users:
    # list them
    get:
      operationId: 'listUsers'  # the id
      responses:
        "200": {description: ok}
    post:
      summary: no id here
      responses:
        "201": {description: ok}
  /users/{id}:
    get:
      x-operation-class: custom
      operationId: getUser
    delete:
      operationId: >-
        folded
      description: last
`,
			want: `openapi: 3.0.3   # keep me
paths:
  /users:
    # list them
    get:
      operationId: 'listUsers'  # the id
      x-operation-class: list
      responses:
        "200": {description: ok}
    post:
      x-operation-class: write
      summary: no id here
      responses:
        "201": {description: ok}
  /users/{id}:
    get:
      x-operation-class: custom
      operationId: getUser
    delete:
      x-operation-class: write
      operationId: >-
        folded
      description: last
`,
			added: 3,
		},
		{
			name:  "CRLF line endings",
			in:    "paths:\r\n  /a:\r\n    get:\r\n      operationId: getA\r\n      summary: s\r\n",
			want:  "paths:\r\n  /a:\r\n    get:\r\n      operationId: getA\r\n      x-operation-class: read\r\n      summary: s\r\n",
			added: 1,
		},
		{
			name:  "no final newline",
			in:    "paths:\n  /a/{id}:\n    get:\n      operationId: getA",
			want:  "paths:\n  /a/{id}:\n    get:\n      operationId: getA\n      x-operation-class: read",
			added: 1,
		},
		{
			name:  "flow style on one line",
			in:    "paths:\n  /a: {get: {operationId: listA}, post: {}}\n",
			want:  "paths:\n  /a: {get: {\"x-operation-class\": \"list\", operationId: listA}, post: {\"x-operation-class\": \"write\"}}\n",
			added: 2,
		},
		{
			name: "pretty JSON",
			in: `{
  "paths": {
    "/a/{id}": {
      "get": {
        "operationId": "getA",
        "responses": {}
      }
    }
  }
}
`,
			want: `{
  "paths": {
    "/a/{id}": {
      "get": {
        "x-operation-class": "read",
        "operationId": "getA",
        "responses": {}
      }
    }
  }
}
`,
			added: 1,
		},
		{
			name:  "non-ASCII text before a flow operation",
			in:    "paths:\n  \"/café\": {get: {operationId: listCafé}}\n",
			want:  "paths:\n  \"/café\": {get: {\"x-operation-class\": \"list\", operationId: listCafé}}\n",
			added: 1,
		},
		{
			name: "an aliased operation is not changed",
			in: `paths:
  /a:
    get: &op
      operationId: listA
  /b:
    get: *op
`,
			want: `paths:
  /a:
    get: &op
      operationId: listA
      x-operation-class: list
  /b:
    get: *op
`,
			added: 1,
		},
		{
			name:  "nothing to add",
			in:    "paths:\n  /a:\n    get:\n      x-operation-class: read\n",
			want:  "paths:\n  /a:\n    get:\n      x-operation-class: read\n",
			added: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, added, err := Annotate([]byte(tt.in))
			if err != nil {
				t.Fatalf("Annotate: %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("Annotate output:\n%q\nwant:\n%q", got, tt.want)
			}
			if added != tt.added {
				t.Errorf("added = %d, want %d", added, tt.added)
			}
			// A second run changes nothing.
			again, addedAgain, err := Annotate(got)
			if err != nil {
				t.Fatalf("second Annotate: %v", err)
			}
			if string(again) != string(got) || addedAgain != 0 {
				t.Errorf("second Annotate added %d and changed the document", addedAgain)
			}
		})
	}
}

func TestAnnotateErrors(t *testing.T) {
	for name, in := range map[string]string{
		"invalid explicit class": "paths:\n  /a:\n    get:\n      x-operation-class: nope\n",
		"no paths":               "openapi: 3.0.3\n",
		"not yaml":               "a: [",
	} {
		if _, _, err := Annotate([]byte(in)); err == nil {
			t.Errorf("%s: Annotate must fail", name)
		}
	}
}

// TestAnnotateFixtures checks that each fixture gains the extension on every
// operation, that the classes do not move, and that only lines are added.
func TestAnnotateFixtures(t *testing.T) {
	for _, file := range []string{"atlas.yaml", "camel.json"} {
		t.Run(file, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("testdata", file))
			if err != nil {
				t.Fatal(err)
			}
			before, err := Parse(data)
			if err != nil {
				t.Fatal(err)
			}
			out, added, err := Annotate(data)
			if err != nil {
				t.Fatalf("Annotate: %v", err)
			}
			after, err := Parse(out)
			if err != nil {
				t.Fatal(err)
			}
			missing := 0
			for _, op := range before.Operations {
				if op.Rule != RuleExplicit {
					missing++
				}
			}
			if added != missing {
				t.Errorf("added = %d, want %d", added, missing)
			}
			for i, op := range after.Operations {
				if op.Rule != RuleExplicit || op.Class != before.Operations[i].Class {
					t.Errorf("%s %s: class %q by %q, want %q by explicit", op.Method, op.Path, op.Class, op.Rule, before.Operations[i].Class)
				}
			}
			src := strings.SplitAfter(string(data), "\n")
			i := 0
			for _, line := range strings.SplitAfter(string(out), "\n") {
				if i < len(src) && line == src[i] {
					i++
				} else if !strings.Contains(line, ExtensionClass) {
					t.Fatalf("Annotate changed a line that it did not add: %q", line)
				}
			}
			if i != len(src) {
				t.Errorf("Annotate dropped source lines: matched %d of %d", i, len(src))
			}
		})
	}
}
