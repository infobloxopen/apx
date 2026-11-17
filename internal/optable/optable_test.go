package optable

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/infobloxopen/apx/internal/opclass"
	"gopkg.in/yaml.v3"
)

var update = flag.Bool("update", false, "rewrite the golden files")

func TestJoinPath(t *testing.T) {
	tests := []struct {
		name, prefix, path, want string
	}{
		{"prefix and path", "/api/v1", "/dashboards/{id}", "/api/v1/dashboards/{id}"},
		{"no prefix", "", "/api/v1/dashboards", "/api/v1/dashboards"},
		{"root prefix", "/", "/dashboards", "/dashboards"},
		{"trailing slash on prefix", "/v2/", "/accounts/groups", "/v2/accounts/groups"},
		{"double slashes", "//api//v1/", "//widgets///{widgetId}/", "/api/v1/widgets/{widgetId}"},
		{"trailing slash on path", "/v2", "/accounts/", "/v2/accounts"},
		{"path without leading slash", "/v2", "accounts", "/v2/accounts"},
		{"root path", "/v2", "/", "/v2"},
		{"root of all", "", "/", "/"},
		{"empty", "", "", "/"},
		{"custom verb stays", "/v1", "/users/{id}:export", "/v1/users/{id}:export"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := JoinPath(tt.prefix, tt.path); got != tt.want {
				t.Errorf("JoinPath(%q, %q) = %q, want %q", tt.prefix, tt.path, got, tt.want)
			}
		})
	}
}

func TestShape(t *testing.T) {
	tests := []struct{ path, want string }{
		{"/a/{id}", "/a/{}"},
		{"/a/{name}/b", "/a/{}/b"},
		{"/a/b", "/a/b"},
		{"/", "/"},
		{"/a/{id}:export", "/a/{id}:export"},
	}
	for _, tt := range tests {
		if got := shape(tt.path); got != tt.want {
			t.Errorf("shape(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestConfigMapName(t *testing.T) {
	got := ConfigMapName("openapi/eor/bi-server/v1")
	if !strings.HasPrefix(got, "optable-") || len(got) != len("optable-")+10 {
		t.Fatalf("ConfigMapName = %q", got)
	}
	if got != ConfigMapName("openapi/eor/bi-server/v1") {
		t.Error("ConfigMapName is not stable")
	}
	if got == ConfigMapName("openapi/eor/bi-server/v2") {
		t.Error("two API ids give one name")
	}
}

// spec builds a small OpenAPI document. Each operation is "METHOD path id".
func spec(t *testing.T, base string, ops ...string) *opclass.Document {
	t.Helper()
	paths := map[string]map[string]any{}
	for _, op := range ops {
		f := strings.Fields(op)
		item := paths[f[1]]
		if item == nil {
			item = map[string]any{}
			paths[f[1]] = item
		}
		o := map[string]any{"responses": map[string]any{"200": map[string]any{"description": "ok"}}}
		if len(f) > 2 {
			o["operationId"] = f[2]
		}
		item[strings.ToLower(f[0])] = o
	}
	doc := map[string]any{"openapi": "3.0.3", "paths": paths}
	if base != "" {
		doc["servers"] = []any{map[string]any{"url": base}}
	}
	data, err := yaml.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	out, err := opclass.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func ingress(name string) TargetRef {
	return TargetRef{Group: TargetGroupIngress, Kind: TargetKindIngress, Name: name}
}

func binding(namespace string, apis ...BindingAPI) *Binding {
	return &Binding{
		APIVersion: BindingAPIVersion, Kind: BindingKind,
		Metadata: BindingMetadata{Name: "b", Namespace: namespace},
		Spec:     BindingSpec{APIs: apis},
	}
}

func TestBuildPrecedence(t *testing.T) {
	type api struct {
		id, audience, base, prefix string
		ops                        []string
	}
	tests := []struct {
		name string
		apis []api
		// want is "METHOD path api operation" for each resolved operation.
		want         []string
		wantDefault  string
		wantConflict []string // substrings of the one conflict
	}{
		{
			name: "one API",
			apis: []api{{id: "a/v1", base: "/v1", ops: []string{"GET /things listThings", "GET /things/{id} getThing"}}},
			want: []string{
				"GET /v1/things a/v1 listThings",
				"GET /v1/things/{id} a/v1 getThing",
			},
			wantDefault: "a/v1",
		},
		{
			name: "literal over variable keeps both templates",
			apis: []api{
				{id: "pub/v2", audience: "public", base: "/v2", ops: []string{"GET /accounts/{id} Accounts_Read"}},
				{id: "int/v2", audience: "internal", base: "/v2", ops: []string{"GET /accounts/groups Groups_List"}},
			},
			want: []string{
				"GET /v2/accounts/groups int/v2 Groups_List",
				"GET /v2/accounts/{id} pub/v2 Accounts_Read",
			},
			wantDefault: "pub/v2",
		},
		{
			name: "public over internal on a shared path",
			apis: []api{
				{id: "int/v2", audience: "internal", base: "/v2", ops: []string{"GET /accounts Accounts_List", "POST /accounts Accounts_Create"}},
				{id: "pub/v2", audience: "public", base: "/v2", ops: []string{"GET /accounts Accounts_List"}},
			},
			want: []string{
				"GET /v2/accounts pub/v2 Accounts_List",
				"POST /v2/accounts int/v2 Accounts_Create",
			},
			wantDefault: "pub/v2",
		},
		{
			name: "identical public and internal paths go to the public API",
			apis: []api{
				{id: "int/v2", audience: "internal", base: "/v2", ops: []string{"GET /accounts Accounts_List", "GET /accounts/{account_id} Accounts_Read"}},
				{id: "pub/v2", audience: "public", base: "/v2", ops: []string{"GET /accounts Accounts_List", "GET /accounts/{id} Accounts_Read"}},
			},
			want: []string{
				"GET /v2/accounts pub/v2 Accounts_List",
				"GET /v2/accounts/{id} pub/v2 Accounts_Read",
			},
			wantDefault: "pub/v2",
		},
		{
			name: "same audience is a conflict",
			apis: []api{
				{id: "a/v1", audience: "internal", base: "/v1", ops: []string{"GET /things listThings"}},
				{id: "b/v1", audience: "internal", base: "/v1", ops: []string{"GET /things listThings"}},
			},
			wantDefault:  "a/v1",
			wantConflict: []string{"route ns/web", "GET /v1/things", "API a/v1", "API b/v1", "same audience (internal)"},
		},
		{
			name: "two public APIs are a conflict",
			apis: []api{
				{id: "a/v1", audience: "public", base: "/v1", ops: []string{"GET /things listThings"}},
				{id: "b/v1", audience: "public", base: "/v1", ops: []string{"GET /things listThings"}},
			},
			wantDefault:  "a/v1",
			wantConflict: []string{"GET /v1/things", "API a/v1", "API b/v1", "same audience (public)"},
		},
		{
			name: "different operationId is a conflict, also for public and internal",
			apis: []api{
				{id: "pub/v1", audience: "public", base: "/v1", ops: []string{"GET /things/{id} getThing"}},
				{id: "int/v1", audience: "internal", base: "/v1", ops: []string{"GET /things/{name} readThing"}},
			},
			wantDefault:  "pub/v1",
			wantConflict: []string{"GET /v1/things/{id} and /v1/things/{name}", "API pub/v1", "API int/v1", "operationId values are different (getThing and readThing)"},
		},
		{
			name: "another method on one path is no overlap",
			apis: []api{
				{id: "a/v1", base: "/v1", ops: []string{"GET /things listThings"}},
				{id: "b/v1", base: "/v1", ops: []string{"POST /things createThing"}},
			},
			want: []string{
				"GET /v1/things a/v1 listThings",
				"POST /v1/things b/v1 createThing",
			},
			wantDefault: "a/v1",
		},
		{
			name: "externalPrefix replaces the base path",
			apis: []api{
				{id: "a/v1", base: "/internal/v1", prefix: "/api/v1/", ops: []string{"GET /things listThings"}},
				{id: "b/v1", base: "/v9", prefix: "/", ops: []string{"GET /other listOther"}},
			},
			want: []string{
				"GET /api/v1/things a/v1 listThings",
				"GET /other b/v1 listOther",
			},
			wantDefault: "a/v1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apis := map[string]API{}
			var entries []BindingAPI
			for _, a := range tt.apis {
				apis[a.id] = API{Document: spec(t, a.base, a.ops...), Audience: a.audience}
				entries = append(entries, BindingAPI{Ref: a.id, Version: "v1.0.0", ExternalPrefix: a.prefix, TargetRefs: []TargetRef{ingress("web")}})
			}
			res, err := Build([]*Binding{binding("ns", entries...)}, apis, "")
			if err != nil {
				t.Fatal(err)
			}
			if len(res.RouteTable.Routes) != 1 {
				t.Fatalf("routes = %d, want 1", len(res.RouteTable.Routes))
			}
			route := res.RouteTable.Routes[0]
			if route.Namespace != "ns" || route.Name != "web" {
				t.Errorf("route = %s/%s", route.Namespace, route.Name)
			}
			if route.DefaultAPI != tt.wantDefault {
				t.Errorf("defaultApi = %q, want %q", route.DefaultAPI, tt.wantDefault)
			}
			if tt.wantConflict != nil {
				if len(res.Conflicts) != 1 {
					t.Fatalf("conflicts = %v, want 1", res.Conflicts)
				}
				if res.Err() == nil {
					t.Fatal("Err() = nil with a conflict")
				}
				msg := res.Err().Error()
				for _, sub := range tt.wantConflict {
					if !strings.Contains(msg, sub) {
						t.Errorf("conflict %q does not contain %q", msg, sub)
					}
				}
				return
			}
			if err := res.Err(); err != nil {
				t.Fatalf("unexpected conflict: %v", err)
			}
			var got []string
			for _, op := range route.Operations {
				got = append(got, strings.Join([]string{op.Method, op.Path, op.API, op.Operation}, " "))
			}
			if strings.Join(got, "\n") != strings.Join(tt.want, "\n") {
				t.Errorf("operations:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(tt.want, "\n"))
			}
		})
	}
}

func TestBuildRoutesAreSeparate(t *testing.T) {
	// Two internal APIs with one path do not conflict when no route has both.
	apis := map[string]API{
		"a/v1": {Document: spec(t, "/v1", "GET /things listThings")},
		"b/v1": {Document: spec(t, "/v1", "GET /things listThings")},
	}
	b := binding("ns",
		BindingAPI{Ref: "a/v1", Version: "v1.0.0", TargetRefs: []TargetRef{ingress("one")}},
		BindingAPI{Ref: "b/v1", Version: "v1.0.0", TargetRefs: []TargetRef{ingress("two")}},
	)
	res, err := Build([]*Binding{b}, apis, "")
	if err != nil || res.Err() != nil {
		t.Fatalf("err=%v conflicts=%v", err, res.Err())
	}
	if len(res.RouteTable.Routes) != 2 || res.RouteTable.Routes[0].Name != "one" || res.RouteTable.Routes[1].DefaultAPI != "b/v1" {
		t.Errorf("routes = %+v", res.RouteTable.Routes)
	}
}

func TestBuildTwoPrefixesForOneAPI(t *testing.T) {
	apis := map[string]API{"a/v1": {Document: spec(t, "/v1", "GET /things listThings")}}
	b := binding("",
		BindingAPI{Ref: "a/v1", Version: "v1.0.0", TargetRefs: []TargetRef{ingress("one")}},
		BindingAPI{Ref: "a/v1", Version: "v1.0.0", ExternalPrefix: "/legacy", TargetRefs: []TargetRef{ingress("two")}},
	)
	res, err := Build([]*Binding{b}, apis, "fallback")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Tables) != 1 || len(res.Tables[0].Operations) != 2 {
		t.Fatalf("tables = %+v", res.Tables)
	}
	if p := res.Tables[0].Operations[0].Path; p != "/legacy/things" {
		t.Errorf("first path = %q", p)
	}
	if res.Tables[0].Audience != AudienceInternal {
		t.Errorf("audience = %q, want the default internal", res.Tables[0].Audience)
	}
	for _, r := range res.RouteTable.Routes {
		if r.Namespace != "fallback" || len(r.Operations) != 1 {
			t.Errorf("route = %+v", r)
		}
	}
}

func TestBuildErrors(t *testing.T) {
	doc := spec(t, "/v1", "GET /things listThings")
	tests := []struct {
		name     string
		bindings []*Binding
		apis     map[string]API
		want     string
	}{
		{
			name:     "no contract",
			bindings: []*Binding{binding("ns", BindingAPI{Ref: "a/v1", Version: "v1.0.0", TargetRefs: []TargetRef{ingress("web")}})},
			apis:     map[string]API{},
			want:     "API a/v1: no API contract was given",
		},
		{
			name:     "no operationId",
			bindings: []*Binding{binding("ns", BindingAPI{Ref: "a/v1", Version: "v1.0.0", TargetRefs: []TargetRef{ingress("web")}})},
			apis:     map[string]API{"a/v1": {Document: spec(t, "/v1", "GET /things")}},
			want:     "API a/v1: GET /things has no operationId",
		},
		{
			name: "two versions in two bindings",
			bindings: []*Binding{
				binding("ns", BindingAPI{Ref: "a/v1", Version: "v1.0.0", TargetRefs: []TargetRef{ingress("web")}}),
				binding("ns", BindingAPI{Ref: "a/v1", Version: "v1.1.0", TargetRefs: []TargetRef{ingress("web")}}),
			},
			apis: map[string]API{"a/v1": {Document: doc}},
			want: "API a/v1 is pinned at two versions (v1.0.0 and v1.1.0)",
		},
		{
			name:     "bad audience",
			bindings: []*Binding{binding("ns", BindingAPI{Ref: "a/v1", Version: "v1.0.0", TargetRefs: []TargetRef{ingress("web")}})},
			apis:     map[string]API{"a/v1": {Document: doc, Audience: "partner"}},
			want:     `invalid audience "partner"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Build(tt.bindings, tt.apis, "")
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("err = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestParseBinding(t *testing.T) {
	const good = `apiVersion: traefik.infoblox.com/v1alpha1
kind: RouteAPIBinding
metadata: {name: eor, namespace: eor-prod, labels: {a: b}}
spec:
  apis:
    - ref: openapi/eor/bi-server/v1
      version: v1.0.0
      externalPrefix: /api/v1
      targetRefs:
        - {group: networking.k8s.io, kind: Ingress, name: eor}
`
	b, err := ParseBinding([]byte(good))
	if err != nil {
		t.Fatal(err)
	}
	if b.Metadata.Name != "eor" || b.Metadata.Namespace != "eor-prod" || len(b.Spec.APIs) != 1 {
		t.Fatalf("binding = %+v", b)
	}
	if a := b.Spec.APIs[0]; a.Ref != "openapi/eor/bi-server/v1" || a.Version != "v1.0.0" || a.ExternalPrefix != "/api/v1" || a.TargetRefs[0] != ingress("eor") {
		t.Errorf("api = %+v", a)
	}

	tests := []struct {
		name string
		edit func(string) string
		want string
	}{
		{"unknown kind", replace("kind: RouteAPIBinding", "kind: RouteBinding"), `unsupported kind "RouteBinding"`},
		{"unknown apiVersion", replace("v1alpha1", "v1beta1"), `unsupported apiVersion "traefik.infoblox.com/v1beta1"`},
		{"other group", replace("traefik.infoblox.com/v1alpha1", "gateway.networking.k8s.io/v1alpha1"), "unsupported apiVersion"},
		{"targetRef namespace", replace("name: eor}", "name: eor, namespace: other}"), "targetRefs[0]: a namespace field is not permitted"},
		{"targetRef kind HTTPRoute", replace("kind: Ingress", "kind: HTTPRoute"), `kind "HTTPRoute" is not supported (expected Ingress)`},
		{"targetRef group", replace("group: networking.k8s.io", "group: extensions"), `group "extensions" is not valid for kind Ingress`},
		{"targetRef without name", replace(", name: eor}", "}"), "targetRefs[0].name is required"},
		{"no targetRefs", replace("        - {group: networking.k8s.io, kind: Ingress, name: eor}\n", "        []\n"), "targetRefs is empty"},
		{"no ref", replace("ref: openapi/eor/bi-server/v1", "ref: \"\""), "spec.apis[0].ref is required"},
		{"no version", replace("      version: v1.0.0\n", ""), "spec.apis[0].version is required"},
		{"relative prefix", replace("externalPrefix: /api/v1", "externalPrefix: api/v1"), `must start with "/"`},
		{"unknown spec field", replace("      version: v1.0.0\n", "      version: v1.0.0\n      audience: public\n"), "field audience not found"},
		{"no name", replace("name: eor, namespace: eor-prod, ", ""), "metadata.name is required"},
		{"no apis", func(string) string {
			return "apiVersion: traefik.infoblox.com/v1alpha1\nkind: RouteAPIBinding\nmetadata: {name: eor}\nspec: {apis: []}\n"
		}, "spec.apis is empty"},
		{"repeated targetRef", replace("        - {group: networking.k8s.io, kind: Ingress, name: eor}\n",
			"        - {group: networking.k8s.io, kind: Ingress, name: eor}\n        - {group: networking.k8s.io, kind: Ingress, name: eor}\n"), `repeats Ingress "eor"`},
		{"empty document", func(string) string { return "" }, "empty binding document"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := tt.edit(good)
			if doc == good {
				t.Fatal("the edit changed nothing")
			}
			_, err := ParseBinding([]byte(doc))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("err = %v, want %q", err, tt.want)
			}
		})
	}
}

func replace(old, new string) func(string) string {
	return func(s string) string { return strings.Replace(s, old, new, 1) }
}

func TestParseBindingsStream(t *testing.T) {
	bs, err := LoadBindings(filepath.Join("testdata", "binding.yaml"), "test", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(bs) != 2 || bs[0].Metadata.Name != "identity" || bs[1].Metadata.Name != "reports" {
		t.Fatalf("bindings = %+v", bs)
	}

	_, err = ParseBindings([]byte("apiVersion: v1\nkind: Service\nmetadata: {name: x}\n"), "in.yaml")
	if err == nil || !strings.Contains(err.Error(), "in.yaml: no RouteAPIBinding found (kinds in the input: Service)") {
		t.Errorf("err = %v", err)
	}
	_, err = ParseBindings([]byte("kind: RouteAPIBinding\napiVersion: other/v1\nmetadata: {name: x}\n"), "in.yaml")
	if err == nil || !strings.Contains(err.Error(), "unsupported apiVersion") {
		t.Errorf("err = %v", err)
	}
	if _, err := LoadBindings(filepath.Join("testdata", "missing.yaml"), "test", nil); err == nil {
		t.Error("a missing file gives no error")
	}
}

// fixture builds the result of the bindings in testdata/binding.yaml.
func fixture(t *testing.T) *Result {
	t.Helper()
	bindings, err := LoadBindings(filepath.Join("testdata", "binding.yaml"), "test", nil)
	if err != nil {
		t.Fatal(err)
	}
	load := func(name string) *opclass.Document {
		doc, err := opclass.Load(filepath.Join("testdata", name))
		if err != nil {
			t.Fatal(err)
		}
		return doc
	}
	apis := map[string]API{
		"openapi/iam/identity/v2":          {Document: load("identity-public.openapi.yaml"), Audience: AudiencePublic},
		"openapi/iam/identity-internal/v2": {Document: load("identity-internal.openapi.yaml"), Audience: AudienceInternal},
		"openapi/reports/server/v1":        {Document: load("reports.openapi.yaml")},
	}
	res, err := Build(bindings, apis, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := res.Err(); err != nil {
		t.Fatal(err)
	}
	return res
}

// golden compares data with a golden file. Line endings of the golden file
// are normalized, because a Windows checkout can convert them.
func golden(t *testing.T, name string, data []byte) {
	t.Helper()
	path := filepath.Join("testdata", "golden", name)
	if *update {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want = bytes.ReplaceAll(want, []byte("\r\n"), []byte("\n"))
	if !bytes.Equal(data, want) {
		t.Errorf("%s differs from the golden file.\ngot:\n%s\nwant:\n%s", name, data, want)
	}
}

func TestGoldenOutput(t *testing.T) {
	res := fixture(t)
	if len(res.Tables) != 3 {
		t.Fatalf("tables = %d, want 3", len(res.Tables))
	}
	for i := range res.Tables {
		tbl := &res.Tables[i]
		data, err := tbl.ConfigMap("")
		if err != nil {
			t.Fatal(err)
		}
		golden(t, ConfigMapName(tbl.API)+".yaml", data)

		// The manifest is valid YAML and its data key holds the table.
		var cm struct {
			Kind     string `yaml:"kind"`
			Metadata struct {
				Name        string            `yaml:"name"`
				Labels      map[string]string `yaml:"labels"`
				Annotations map[string]string `yaml:"annotations"`
			} `yaml:"metadata"`
			Data map[string]string `yaml:"data"`
		}
		if err := yaml.Unmarshal(data, &cm); err != nil {
			t.Fatal(err)
		}
		if cm.Kind != "ConfigMap" || cm.Metadata.Name != ConfigMapName(tbl.API) ||
			cm.Metadata.Labels[LabelOperationTable] != "true" ||
			cm.Metadata.Annotations[AnnotationAPI] != tbl.API ||
			cm.Metadata.Annotations[AnnotationAPIVersion] != tbl.Version {
			t.Errorf("manifest metadata = %+v", cm)
		}
		var back Table
		if err := json.Unmarshal([]byte(cm.Data[TableDataKey]), &back); err != nil {
			t.Fatal(err)
		}
		if back.Schema != TableSchema || back.API != tbl.API || len(back.Operations) != len(tbl.Operations) {
			t.Errorf("table in the manifest = %+v", back)
		}
	}
	rt, err := res.RouteTable.JSON()
	if err != nil {
		t.Fatal(err)
	}
	golden(t, RouteTableFile, rt)
}

func TestOutputIsDeterministic(t *testing.T) {
	render := func() []byte {
		res := fixture(t)
		var all bytes.Buffer
		for i := range res.Tables {
			data, err := res.Tables[i].ConfigMap("ns")
			if err != nil {
				t.Fatal(err)
			}
			all.Write(data)
		}
		cm, err := res.RouteTable.ConfigMap("route-table", "ns")
		if err != nil {
			t.Fatal(err)
		}
		all.Write(cm)
		return all.Bytes()
	}
	first := render()
	for i := 0; i < 5; i++ {
		if !bytes.Equal(first, render()) {
			t.Fatal("two renders of one input differ")
		}
	}
	if !bytes.Contains(first, []byte("  namespace: ns\n")) {
		t.Error("the namespace is not in the manifests")
	}
}

func TestOperationsAreSorted(t *testing.T) {
	res := fixture(t)
	for _, tbl := range res.Tables {
		for i := 1; i < len(tbl.Operations); i++ {
			a, b := tbl.Operations[i-1], tbl.Operations[i]
			if a.Path > b.Path || (a.Path == b.Path && a.Method >= b.Method) {
				t.Errorf("%s: %s %s is before %s %s", tbl.API, a.Method, a.Path, b.Method, b.Path)
			}
		}
	}
}

func TestRouteTableConfigMapRoundTrip(t *testing.T) {
	res := fixture(t)
	want, err := res.RouteTable.JSON()
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := res.RouteTable.ConfigMap("route-table", "traefik")
	if err != nil {
		t.Fatal(err)
	}
	var cm struct {
		APIVersion string `yaml:"apiVersion"`
		Kind       string `yaml:"kind"`
		Metadata   struct {
			Name      string `yaml:"name"`
			Namespace string `yaml:"namespace"`
		} `yaml:"metadata"`
		BinaryData map[string]string `yaml:"binaryData"`
	}
	if err := yaml.Unmarshal(manifest, &cm); err != nil {
		t.Fatal(err)
	}
	if cm.APIVersion != "v1" || cm.Kind != "ConfigMap" || cm.Metadata.Name != "route-table" || cm.Metadata.Namespace != "traefik" {
		t.Errorf("manifest = %+v", cm)
	}
	zipped, err := base64.StdEncoding.DecodeString(cm.BinaryData[RouteTableDataKey])
	if err != nil {
		t.Fatal(err)
	}
	got, err := Gunzip(zipped)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("round trip:\n%s\nwant:\n%s", got, want)
	}
	var rt RouteTable
	if err := json.Unmarshal(got, &rt); err != nil {
		t.Fatal(err)
	}
	if rt.Schema != RouteTableSchema || len(rt.Routes) != 3 {
		t.Errorf("route table = %+v", rt)
	}
}
