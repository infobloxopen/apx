package opclass

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestClassify(t *testing.T) {
	tests := []struct {
		name     string
		in       Input
		want     Class
		wantRule Rule
		wantErr  bool
	}{
		// Explicit extension.
		{"explicit wins over everything", Input{Method: "GET", Path: "/a/{id}", OperationID: "ListA", AIPMethod: "Get", Explicit: "write"}, Write, RuleExplicit, false},
		{"explicit invalid", Input{Method: "GET", Path: "/a", Explicit: "Read"}, "", "", true},
		{"explicit unknown", Input{Method: "GET", Path: "/a", Explicit: "stream"}, "", "", true},

		// Rule 1: x-aip-method.
		{"aip Get", Input{Method: "POST", Path: "/a", OperationID: "ListA", AIPMethod: "Get"}, Read, RuleAIP, false},
		{"aip List", Input{Method: "GET", Path: "/a/{id}", AIPMethod: "List"}, List, RuleAIP, false},
		{"aip Create", Input{Method: "GET", Path: "/a", AIPMethod: "Create"}, Write, RuleAIP, false},
		{"aip Update", Input{Method: "GET", Path: "/a", AIPMethod: "Update"}, Write, RuleAIP, false},
		{"aip Delete", Input{Method: "GET", Path: "/a", AIPMethod: "Delete"}, Write, RuleAIP, false},
		{"aip BatchGet", Input{Method: "POST", Path: "/a:batchGet", AIPMethod: "BatchGet"}, Read, RuleAIP, false},
		{"aip Search", Input{Method: "POST", Path: "/a:search", AIPMethod: "Search"}, List, RuleAIP, false},
		{"aip Custom falls through to the name", Input{Method: "POST", Path: "/a", OperationID: "Things_List", AIPMethod: "Custom"}, List, RuleName, false},
		{"aip unknown falls through to the shape", Input{Method: "GET", Path: "/a/{id}", OperationID: "Things_Export", AIPMethod: "Export"}, Read, RuleShape, false},

		// Rule 2: atlas names in the published form.
		{"atlas list with a repeated service noun", Input{Method: "POST", Path: "/x", OperationID: "Accounts_AccountListUsers"}, List, RuleName, false},
		{"atlas List", Input{Method: "POST", Path: "/x", OperationID: "AccountOrganizationGroups_List"}, List, RuleName, false},
		{"atlas Read", Input{Method: "POST", Path: "/x", OperationID: "AccountOrganizationGroups_Read"}, Read, RuleName, false},
		{"atlas Create", Input{Method: "GET", Path: "/x", OperationID: "AccountOrganizationGroups_Create"}, Write, RuleName, false},
		{"atlas Update", Input{Method: "GET", Path: "/x", OperationID: "AccountOrganizationGroups_Update"}, Write, RuleName, false},
		{"atlas Patch", Input{Method: "GET", Path: "/x", OperationID: "AccountOrganizationGroups_Patch"}, Write, RuleName, false},
		{"atlas Delete", Input{Method: "GET", Path: "/x", OperationID: "AccountOrganizationGroups_Delete"}, Write, RuleName, false},
		{"atlas verb last", Input{Method: "POST", Path: "/x", OperationID: "Users_UserRead"}, Read, RuleName, false},
		{"atlas plural ies", Input{Method: "POST", Path: "/x", OperationID: "Policies_PolicyListRules"}, List, RuleName, false},
		{"service word that is a verb is not used", Input{Method: "OPTIONS", Path: "/x", OperationID: "List_Verify"}, Custom, RuleDefault, false},

		// Rule 2: camelCase and other forms.
		{"camel list", Input{Method: "POST", Path: "/x", OperationID: "listDashboards"}, List, RuleName, false},
		{"camel get", Input{Method: "POST", Path: "/x", OperationID: "getDashboard"}, Read, RuleName, false},
		{"camel verb last", Input{Method: "POST", Path: "/x", OperationID: "dashboardDelete"}, Write, RuleName, false},
		{"get with a List suffix", Input{Method: "POST", Path: "/x", OperationID: "GetUserList"}, List, RuleName, false},
		{"snake case", Input{Method: "POST", Path: "/x", OperationID: "list_users"}, List, RuleName, false},
		{"kebab case", Input{Method: "GET", Path: "/x", OperationID: "users-create"}, Write, RuleName, false},
		{"acronym", Input{Method: "POST", Path: "/x", OperationID: "GetHTTPRoute"}, Read, RuleName, false},
		{"single verb", Input{Method: "POST", Path: "/x", OperationID: "Get"}, Read, RuleName, false},
		{"Blacklist is one word", Input{Method: "OPTIONS", Path: "/x", OperationID: "Blacklist"}, Custom, RuleDefault, false},
		{"Security_Blacklist is not list", Input{Method: "POST", Path: "/blacklist", OperationID: "Security_Blacklist"}, Write, RuleShape, false},
		{"checklist is one word", Input{Method: "OPTIONS", Path: "/x", OperationID: "checklist"}, Custom, RuleDefault, false},
		{"Getaway is one word", Input{Method: "OPTIONS", Path: "/x", OperationID: "Getaway"}, Custom, RuleDefault, false},
		{"verb in the middle does not match", Input{Method: "OPTIONS", Path: "/x", OperationID: "bulkDeleteUsers"}, Custom, RuleDefault, false},

		// Rule 3: verb and path shape.
		{"GET item", Input{Method: "GET", Path: "/users/{id}"}, Read, RuleShape, false},
		{"GET item with a trailing slash", Input{Method: "get", Path: "/users/{id}/"}, Read, RuleShape, false},
		{"GET collection", Input{Method: "GET", Path: "/users"}, List, RuleShape, false},
		{"GET sub-collection", Input{Method: "GET", Path: "/users/{id}/keys", OperationID: "userKeys"}, List, RuleShape, false},
		{"HEAD item", Input{Method: "HEAD", Path: "/users/{id}"}, Read, RuleShape, false},
		{"GET mixed segment", Input{Method: "GET", Path: "/files/{name}.json"}, List, RuleShape, false},
		{"POST", Input{Method: "POST", Path: "/users"}, Write, RuleShape, false},
		{"PUT", Input{Method: "PUT", Path: "/users/{id}"}, Write, RuleShape, false},
		{"PATCH", Input{Method: "patch", Path: "/users/{id}"}, Write, RuleShape, false},
		{"DELETE", Input{Method: "DELETE", Path: "/users/{id}"}, Write, RuleShape, false},
		{"POST custom verb", Input{Method: "POST", Path: "/users/{id}:disable"}, Write, RuleShape, false},

		// Rule 4: custom.
		{"GET custom verb", Input{Method: "GET", Path: "/users/{id}:export"}, Custom, RuleDefault, false},
		{"GET root", Input{Method: "GET", Path: "/"}, Custom, RuleDefault, false},
		{"OPTIONS", Input{Method: "OPTIONS", Path: "/users"}, Custom, RuleDefault, false},
		{"TRACE", Input{Method: "TRACE", Path: "/users/{id}"}, Custom, RuleDefault, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, rule, err := Classify(tt.in)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Classify() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want || rule != tt.wantRule {
				t.Errorf("Classify() = %q by %q, want %q by %q", got, rule, tt.want, tt.wantRule)
			}
		})
	}
}

func TestSplitWords(t *testing.T) {
	tests := map[string][]string{
		"listDashboards":            {"list", "Dashboards"},
		"Accounts_AccountListUsers": {"Accounts", "Account", "List", "Users"},
		"GetHTTPRoute":              {"Get", "HTTP", "Route"},
		"getV2Thing":                {"get", "V2", "Thing"},
		"list-users.v1":             {"list", "users", "v1"},
		"Blacklist":                 {"Blacklist"},
		"":                          nil,
	}
	for in, want := range tests {
		if got := splitWords(in); !reflect.DeepEqual(got, want) {
			t.Errorf("splitWords(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestParseClass(t *testing.T) {
	for _, c := range Classes() {
		if got, err := ParseClass(string(c)); err != nil || got != c {
			t.Errorf("ParseClass(%q) = %q, %v", c, got, err)
		}
	}
	for _, bad := range []string{"", "READ", "stream"} {
		if _, err := ParseClass(bad); err == nil {
			t.Errorf("ParseClass(%q) must fail", bad)
		}
	}
}

func TestLoadFixtures(t *testing.T) {
	tests := []struct {
		file     string
		basePath string
		want     []Operation
	}{
		{
			file:     "atlas.yaml",
			basePath: "/api/identity/v1",
			want: []Operation{
				{"GET", "/accounts/{id}/users", "Accounts_AccountListUsers", List, false, RuleName},
				{"GET", "/blacklist", "Security_Blacklist", List, false, RuleShape},
				{"GET", "/books/{id}", "Library_Fetch", Read, false, RuleAIP},
				{"GET", "/events", "Events_Watch", Custom, true, RuleExplicit},
				{"GET", "/groups", "AccountOrganizationGroups_List", List, false, RuleName},
				{"POST", "/groups", "AccountOrganizationGroups_Create", Write, false, RuleName},
				{"DELETE", "/groups/{id}", "AccountOrganizationGroups_Delete", Write, false, RuleName},
				{"GET", "/groups/{id}", "AccountOrganizationGroups_Read", Read, false, RuleName},
				{"PATCH", "/groups/{id}", "AccountOrganizationGroups_Patch", Write, false, RuleName},
				{"PUT", "/groups/{id}", "AccountOrganizationGroups_Update", Write, false, RuleName},
				{"OPTIONS", "/session/verify", "Sessions_Verify", Custom, false, RuleDefault},
			},
		},
		{
			file:     "camel.json",
			basePath: "/api/v1",
			want: []Operation{
				{"GET", "/dashboards", "listDashboards", List, false, RuleName},
				{"POST", "/dashboards", "importDashboard", Write, false, RuleShape},
				{"GET", "/dashboards/{id}", "getDashboard", Read, false, RuleName},
				{"HEAD", "/dashboards/{id}", "", Read, false, RuleShape},
				{"GET", "/dashboards/{id}/rows", "streamRows", List, true, RuleShape},
				{"GET", "/dashboards/{id}:export", "exportDashboard", Custom, true, RuleDefault},
				{"GET", "/healthz", "", List, false, RuleShape},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			doc, err := Load(filepath.Join("testdata", tt.file))
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if doc.BasePath != tt.basePath {
				t.Errorf("BasePath = %q, want %q", doc.BasePath, tt.basePath)
			}
			if !reflect.DeepEqual(doc.Operations, tt.want) {
				t.Errorf("Operations:\n got %+v\nwant %+v", doc.Operations, tt.want)
			}
			total := 0
			for _, n := range doc.Counts() {
				total += n
			}
			if total != len(tt.want) {
				t.Errorf("Counts() sums to %d, want %d", total, len(tt.want))
			}
		})
	}
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		name, doc, wantErr string
	}{
		{"not yaml", "a: [", "parsing spec"},
		{"empty", "", "empty document"},
		{"root is a list", "- a\n", "not an object"},
		{"no paths", "openapi: 3.0.3\n", `"paths"`},
		{"invalid class", "paths:\n  /a:\n    get:\n      x-operation-class: fast\n", `GET /a: invalid x-operation-class "fast"`},
		{"class is not a string", "paths:\n  /a:\n    get:\n      x-operation-class: [read]\n", "not a string"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse([]byte(tt.doc))
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Parse() error = %v, want it to contain %q", err, tt.wantErr)
			}
		})
	}
	if _, err := Load(filepath.Join("testdata", "missing.yaml")); err == nil {
		t.Error("Load of a missing file must fail")
	}
}

func TestParseSkipsNonOperations(t *testing.T) {
	doc, err := Parse([]byte(`paths:
  /a:
    summary: not an operation
    parameters: []
    get:
    post: {}
  /b: null
  x-internal: true
`))
	if err != nil {
		t.Fatal(err)
	}
	want := []Operation{{"POST", "/a", "", Write, false, RuleShape}}
	if !reflect.DeepEqual(doc.Operations, want) {
		t.Errorf("Operations = %+v, want %+v", doc.Operations, want)
	}
}

func TestBasePath(t *testing.T) {
	tests := map[string]string{
		"servers: [{url: /api/v1}]":                             "/api/v1",
		"servers: [{url: 'https://{env}.example.com/api/v2/'}]": "/api/v2",
		"servers: [{url: 'https://example.com'}]":               "",
		"servers: [{url: 'https://a.example/x'}, {url: /y}]":    "/x",
		"basePath: /legacy/":                                    "/legacy",
		"openapi: 3.0.3":                                        "",
	}
	for head, want := range tests {
		doc, err := Parse([]byte(head + "\npaths: {}\n"))
		if err != nil {
			t.Fatalf("%s: %v", head, err)
		}
		if doc.BasePath != want {
			t.Errorf("%s: BasePath = %q, want %q", head, doc.BasePath, want)
		}
	}
}
