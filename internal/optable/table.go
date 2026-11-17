package optable

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/infobloxopen/apx/internal/opclass"
)

// The schema strings of the two outputs.
const (
	TableSchema      = "apx.infoblox.com/operation-table/v1"
	RouteTableSchema = "traefik.infoblox.com/route-table/v1"
)

// The audiences of an API.
const (
	AudiencePublic   = "public"
	AudienceInternal = "internal"
)

// Operation is one row of an operation table.
type Operation struct {
	// Method is the HTTP method in upper case.
	Method string `json:"method"`
	// Path is the external path template: the path that the client sends.
	Path        string        `json:"path"`
	OperationID string        `json:"operationId"`
	Class       opclass.Class `json:"class"`
	Streaming   bool          `json:"streaming"`
}

// Table is the operation table of one API. Operations is sorted by Path, then
// by Method.
type Table struct {
	Schema     string      `json:"schema"`
	API        string      `json:"api"`
	Version    string      `json:"version"`
	Audience   string      `json:"audience"`
	Operations []Operation `json:"operations"`
	// Descriptor is set for a gRPC API only: where its descriptor set is.
	Descriptor *TableDescriptor `json:"descriptor,omitempty"`
}

// TableDescriptor names the descriptor set of a gRPC API: the ConfigMap, its
// binaryData key and the SHA-256 digest of the set before gzip.
type TableDescriptor struct {
	ConfigMap string `json:"configMap"`
	Key       string `json:"key"`
	Digest    string `json:"digest"`
}

// Counts returns the number of operations in each class.
func (t *Table) Counts() map[opclass.Class]int {
	out := make(map[opclass.Class]int, 4)
	for _, op := range t.Operations {
		out[op.Class]++
	}
	return out
}

// RouteOperation is one resolved operation of a route.
type RouteOperation struct {
	Method    string `json:"method"`
	Path      string `json:"path"`
	API       string `json:"api"`
	Operation string `json:"operation"`
	Streaming bool   `json:"streaming"`
}

// Route is one Ingress with its resolved operations. Each method and path
// shape appears one time.
type Route struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	// DefaultAPI is the API with the highest precedence on the route: the
	// first public API in binding order, otherwise the first API.
	DefaultAPI string           `json:"defaultApi"`
	Operations []RouteOperation `json:"operations"`
}

// RouteTable is the resolved table of all routes, sorted by namespace, then by
// name.
type RouteTable struct {
	Schema string  `json:"schema"`
	Routes []Route `json:"routes"`
}

// Conflict is one overlap between two APIs on one route that the precedence
// rules do not resolve.
type Conflict struct {
	Namespace string `json:"namespace"`
	Route     string `json:"route"`
	Method    string `json:"method"`
	Path      string `json:"path"`
	API       string `json:"api"`
	OtherAPI  string `json:"otherApi"`
	// OtherPath is set when the second API writes the same path shape with
	// other variable names.
	OtherPath string `json:"otherPath,omitempty"`
	Reason    string `json:"reason"`
}

func (c Conflict) String() string {
	route := c.Route
	if c.Namespace != "" {
		route = c.Namespace + "/" + c.Route
	}
	path := c.Path
	if c.OtherPath != "" {
		path += " and " + c.OtherPath
	}
	return fmt.Sprintf("route %s: %s %s: API %s and API %s are in conflict: %s", route, c.Method, path, c.API, c.OtherAPI, c.Reason)
}

// ConflictError is the error of a render that has conflicts.
type ConflictError struct {
	Conflicts []Conflict
}

func (e *ConflictError) Error() string {
	lines := make([]string, 0, len(e.Conflicts)+1)
	lines = append(lines, fmt.Sprintf("%d operation conflict(s) between the APIs of the binding:", len(e.Conflicts)))
	for _, c := range e.Conflicts {
		lines = append(lines, "  "+c.String())
	}
	return strings.Join(lines, "\n")
}

// API is the contract of one API id: its classified document and its audience.
type API struct {
	Document *opclass.Document
	// Audience is "public" or "internal". Empty means internal.
	Audience string
	// Descriptor is set for a gRPC API only (package protoset gives it).
	Descriptor *Descriptor
}

// Descriptor is the canonical descriptor set of a gRPC API and its digest
// ("sha256:" and the hex SHA-256 of Set).
type Descriptor struct {
	Set    []byte
	Digest string
}

// Result is the output of Build.
type Result struct {
	// Tables has one operation table for each API, sorted by API id.
	Tables     []Table
	RouteTable RouteTable
	Conflicts  []Conflict
}

// Err returns a *ConflictError when the result has conflicts, otherwise nil.
func (r *Result) Err() error {
	if len(r.Conflicts) == 0 {
		return nil
	}
	return &ConflictError{Conflicts: r.Conflicts}
}

// Build computes the operation table of each API of the bindings and the
// route table of their routes. apis maps an API id to its contract; every API
// of the bindings must be in it. defaultNamespace is the namespace of a
// binding that has no metadata.namespace.
//
// A conflict is not an error of Build: the caller reads Result.Conflicts or
// Result.Err.
func Build(bindings []*Binding, apis map[string]API, defaultNamespace string) (*Result, error) {
	type bound struct {
		api    string
		prefix string
	}
	type routeKey struct{ namespace, name string }

	var errs []error
	versions := map[string]string{}
	prefixes := map[string][]string{} // API id -> effective prefixes, in binding order
	routes := map[routeKey][]bound{}
	var apiOrder []string

	for _, b := range bindings {
		namespace := b.Metadata.Namespace
		if namespace == "" {
			namespace = defaultNamespace
		}
		for _, entry := range b.Spec.APIs {
			api, ok := apis[entry.Ref]
			if !ok || api.Document == nil {
				errs = append(errs, fmt.Errorf("API %s: no API contract was given", entry.Ref))
				continue
			}
			if err := validateAudience(api.Audience); err != nil {
				errs = append(errs, fmt.Errorf("API %s: %w", entry.Ref, err))
				continue
			}
			if prev, seen := versions[entry.Ref]; !seen {
				versions[entry.Ref] = entry.Version
				apiOrder = append(apiOrder, entry.Ref)
			} else if prev != entry.Version {
				errs = append(errs, fmt.Errorf("API %s is pinned at two versions (%s and %s)", entry.Ref, prev, entry.Version))
				continue
			}
			prefix := entry.ExternalPrefix
			if api.Descriptor != nil && prefix != "" && JoinPath(prefix, "") != "/" {
				// A gRPC client always sends /<package>.<Service>/<Method>.
				errs = append(errs, fmt.Errorf("API %s: externalPrefix %q: a gRPC path has no prefix (use \"/\" or no externalPrefix)", entry.Ref, entry.ExternalPrefix))
				continue
			}
			if prefix == "" {
				prefix = api.Document.BasePath
			}
			prefix = JoinPath(prefix, "")
			if !contains(prefixes[entry.Ref], prefix) {
				prefixes[entry.Ref] = append(prefixes[entry.Ref], prefix)
			}
			for _, ref := range entry.TargetRefs {
				key := routeKey{namespace, ref.Name}
				item := bound{api: entry.Ref, prefix: prefix}
				dup := false
				for _, have := range routes[key] {
					dup = dup || have == item
				}
				if !dup {
					routes[key] = append(routes[key], item)
				}
			}
		}
	}

	for _, id := range apiOrder {
		for _, op := range apis[id].Document.Operations {
			if op.OperationID == "" {
				errs = append(errs, fmt.Errorf("API %s: %s %s has no operationId", id, op.Method, op.Path))
			}
		}
	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}

	res := &Result{RouteTable: RouteTable{Schema: RouteTableSchema, Routes: []Route{}}}

	sorted := append([]string(nil), apiOrder...)
	sort.Strings(sorted)
	for _, id := range sorted {
		t := Table{Schema: TableSchema, API: id, Version: versions[id], Audience: audienceOf(apis[id]), Operations: []Operation{}}
		if d := apis[id].Descriptor; d != nil {
			t.Descriptor = &TableDescriptor{ConfigMap: DescriptorConfigMapName(id), Key: DescriptorDataKey, Digest: d.Digest}
		}
		seen := map[string]bool{}
		for _, prefix := range prefixes[id] {
			for _, op := range apis[id].Document.Operations {
				row := Operation{
					Method:      strings.ToUpper(op.Method),
					Path:        JoinPath(prefix, op.Path),
					OperationID: op.OperationID,
					Class:       op.Class,
					Streaming:   op.Streaming,
				}
				if k := row.Method + " " + row.Path; !seen[k] {
					seen[k] = true
					t.Operations = append(t.Operations, row)
				}
			}
		}
		sort.SliceStable(t.Operations, func(i, j int) bool {
			a, b := t.Operations[i], t.Operations[j]
			if a.Path != b.Path {
				return a.Path < b.Path
			}
			return a.Method < b.Method
		})
		res.Tables = append(res.Tables, t)
	}

	keys := make([]routeKey, 0, len(routes))
	for k := range routes {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].namespace != keys[j].namespace {
			return keys[i].namespace < keys[j].namespace
		}
		return keys[i].name < keys[j].name
	})
	for _, key := range keys {
		route := Route{Namespace: key.namespace, Name: key.name, Operations: []RouteOperation{}}
		var cands []candidate
		for order, item := range routes[key] {
			api := apis[item.api]
			if route.DefaultAPI == "" {
				route.DefaultAPI = item.api
			}
			for _, op := range api.Document.Operations {
				cands = append(cands, candidate{
					RouteOperation: RouteOperation{
						Method:    strings.ToUpper(op.Method),
						Path:      JoinPath(item.prefix, op.Path),
						API:       item.api,
						Operation: op.OperationID,
						Streaming: op.Streaming,
					},
					audience: audienceOf(api),
					order:    order,
				})
			}
		}
		for _, item := range routes[key] {
			if audienceOf(apis[item.api]) == AudiencePublic {
				route.DefaultAPI = item.api
				break
			}
		}
		ops, conflicts := resolve(cands)
		for i := range conflicts {
			conflicts[i].Namespace, conflicts[i].Route = key.namespace, key.name
		}
		route.Operations = append(route.Operations, ops...)
		res.Conflicts = append(res.Conflicts, conflicts...)
		res.RouteTable.Routes = append(res.RouteTable.Routes, route)
	}
	return res, nil
}

// candidate is one operation of one API that is bound to a route.
type candidate struct {
	RouteOperation
	audience string
	order    int // position of the API in the binding
}

// resolve applies the precedence rules to the operations of one route. It
// returns one operation for each method and path shape, sorted by path and
// then by method, and the overlaps that the rules do not resolve.
func resolve(cands []candidate) ([]RouteOperation, []Conflict) {
	groups := map[string][]candidate{}
	var order []string
	for _, c := range cands {
		// Rule 1 (a literal segment wins over a variable) needs no selection
		// here: templates with a different shape are different groups, and the
		// matcher prefers the literal segment.
		k := c.Method + " " + shape(c.Path)
		if _, ok := groups[k]; !ok {
			order = append(order, k)
		}
		groups[k] = append(groups[k], c)
	}

	var out []RouteOperation
	var conflicts []Conflict
	for _, k := range order {
		group := dedupe(groups[k])
		winner, conflict := pick(group)
		if conflict != nil {
			conflicts = append(conflicts, *conflict)
			continue
		}
		out = append(out, winner.RouteOperation)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Path != out[j].Path {
			return out[i].Path < out[j].Path
		}
		return out[i].Method < out[j].Method
	})
	sort.SliceStable(conflicts, func(i, j int) bool {
		a, b := conflicts[i], conflicts[j]
		if a.Path != b.Path {
			return a.Path < b.Path
		}
		return a.Method < b.Method
	})
	return out, conflicts
}

// dedupe drops a candidate that repeats an earlier one exactly.
func dedupe(group []candidate) []candidate {
	var out []candidate
	for _, c := range group {
		dup := false
		for _, have := range out {
			dup = dup || have.RouteOperation == c.RouteOperation
		}
		if !dup {
			out = append(out, c)
		}
	}
	return out
}

// pick selects the operation of one method and path shape.
func pick(group []candidate) (candidate, *Conflict) {
	if len(group) == 1 {
		return group[0], nil
	}
	conflict := func(a, b candidate, reason string) *Conflict {
		c := &Conflict{Method: a.Method, Path: a.Path, API: a.API, OtherAPI: b.API, Reason: reason}
		if b.Path != a.Path {
			c.OtherPath = b.Path
		}
		return c
	}
	first := group[0]
	for _, c := range group[1:] {
		if c.Operation != first.Operation {
			return candidate{}, conflict(first, c, fmt.Sprintf("the operationId values are different (%s and %s)", first.Operation, c.Operation))
		}
	}
	// Rule 2: public wins over internal.
	var public []candidate
	for _, c := range group {
		if c.audience == AudiencePublic {
			public = append(public, c)
		}
	}
	switch len(public) {
	case 1:
		return public[0], nil
	case 0:
		return candidate{}, conflict(group[0], group[1], "the two APIs have the same audience (internal)")
	default:
		return candidate{}, conflict(public[0], public[1], "the two APIs have the same audience (public)")
	}
}

func audienceOf(api API) string {
	if api.Audience == AudiencePublic {
		return AudiencePublic
	}
	return AudienceInternal
}

func validateAudience(audience string) error {
	switch audience {
	case "", AudiencePublic, AudienceInternal:
		return nil
	}
	return fmt.Errorf("invalid audience %q (expected %s or %s)", audience, AudiencePublic, AudienceInternal)
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
