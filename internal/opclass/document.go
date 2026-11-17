package opclass

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Operation is one operation of an OpenAPI document with its class.
type Operation struct {
	// Method is the HTTP method in upper case.
	Method string `json:"method"`
	// Path is the path template exactly as the document writes it, without the
	// base path of the document.
	Path string `json:"path"`
	// OperationID is empty when the document does not set it.
	OperationID string `json:"operationId"`
	Class       Class  `json:"class"`
	Streaming   bool   `json:"streaming"`
	// Rule is the rule that decided Class.
	Rule Rule `json:"rule"`
}

// Document is the classified view of one OpenAPI 3.x or Swagger 2 document.
type Document struct {
	// BasePath is the path component of servers[0].url (OpenAPI 3.x) or the
	// basePath (Swagger 2), without a trailing slash. The external path of an
	// operation is BasePath + Operation.Path.
	BasePath string
	// Operations is sorted by Path, then by Method.
	Operations []Operation
}

// Counts returns the number of operations in each class.
func (d *Document) Counts() map[Class]int {
	out := make(map[Class]int, 4)
	for _, op := range d.Operations {
		out[op.Class]++
	}
	return out
}

// Load reads and classifies the document at path (YAML or JSON).
func Load(path string) (*Document, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading spec: %w", err)
	}
	doc, err := Parse(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return doc, nil
}

// Parse classifies each operation of an OpenAPI document (YAML or JSON). It
// reads the first YAML document only and does not follow a $ref on a path
// item. It returns an error when the document has no "paths" object or when an
// explicit x-operation-class is not valid.
func Parse(data []byte) (*Document, error) {
	root, err := parseRoot(data)
	if err != nil {
		return nil, err
	}
	refs, err := walk(root)
	if err != nil {
		return nil, err
	}
	doc := &Document{BasePath: basePath(root), Operations: make([]Operation, 0, len(refs))}
	var errs []error
	for _, ref := range refs {
		op, opErr := ref.operation(root)
		if opErr != nil {
			errs = append(errs, opErr)
			continue
		}
		doc.Operations = append(doc.Operations, op)
	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	sort.SliceStable(doc.Operations, func(i, j int) bool {
		a, b := doc.Operations[i], doc.Operations[j]
		if a.Path != b.Path {
			return a.Path < b.Path
		}
		return a.Method < b.Method
	})
	return doc, nil
}

// ErrNoPaths is the error of Parse and Load, wrapped, for a document that has
// no top-level "paths" object.
var ErrNoPaths = errors.New(`no top-level "paths" object found`)

// httpMethods are the operation keys of a path item.
var httpMethods = map[string]bool{
	"get": true, "put": true, "post": true, "delete": true,
	"options": true, "head": true, "patch": true, "trace": true,
}

// opRef points at one operation mapping in the YAML tree.
type opRef struct {
	method  string
	path    string
	node    *yaml.Node // the operation mapping, with an alias resolved
	aliased bool       // the document reaches the mapping through an alias
}

func parseRoot(data []byte) (*yaml.Node, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parsing spec: %w", err)
	}
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return nil, errors.New("empty document")
	}
	root := resolve(doc.Content[0])
	if root.Kind != yaml.MappingNode {
		return nil, errors.New("the document root is not an object")
	}
	return root, nil
}

func walk(root *yaml.Node) ([]opRef, error) {
	paths := resolve(mapGet(root, "paths"))
	if paths == nil || paths.Kind != yaml.MappingNode {
		return nil, ErrNoPaths
	}
	var refs []opRef
	for i := 0; i+1 < len(paths.Content); i += 2 {
		item := resolve(paths.Content[i+1])
		if item == nil || item.Kind != yaml.MappingNode {
			continue
		}
		for j := 0; j+1 < len(item.Content); j += 2 {
			key := strings.ToLower(item.Content[j].Value)
			if !httpMethods[key] {
				continue
			}
			raw := item.Content[j+1]
			node := resolve(raw)
			if node == nil || node.Kind != yaml.MappingNode {
				continue
			}
			refs = append(refs, opRef{
				method:  strings.ToUpper(key),
				path:    paths.Content[i].Value,
				node:    node,
				aliased: raw.Kind == yaml.AliasNode,
			})
		}
	}
	return refs, nil
}

func (r opRef) operation(root *yaml.Node) (Operation, error) {
	in := Input{
		Method:      r.method,
		Path:        r.path,
		OperationID: scalar(mapGet(r.node, "operationId")),
		AIPMethod:   scalar(mapGet(r.node, ExtensionAIPMethod)),
		Explicit:    scalar(mapGet(r.node, ExtensionClass)),
	}
	if n := mapGet(r.node, ExtensionClass); n != nil && in.Explicit == "" {
		return Operation{}, fmt.Errorf("%s %s: %s is empty or not a string", r.method, r.path, ExtensionClass)
	}
	class, rule, err := Classify(in)
	if err != nil {
		return Operation{}, fmt.Errorf("%s %s: %w", r.method, r.path, err)
	}
	return Operation{
		Method:      r.method,
		Path:        r.path,
		OperationID: in.OperationID,
		Class:       class,
		Streaming:   streams(root, r.node),
		Rule:        rule,
	}, nil
}

// streams reports whether the operation declares a streaming response.
func streams(root, op *yaml.Node) bool {
	if n := resolve(mapGet(op, ExtensionStreaming)); n != nil && n.Kind == yaml.ScalarNode && n.Tag == "!!bool" {
		if strings.EqualFold(n.Value, "true") {
			return true
		}
	}
	responses := resolve(mapGet(op, "responses"))
	if responses != nil && responses.Kind == yaml.MappingNode {
		for i := 1; i < len(responses.Content); i += 2 {
			content := resolve(mapGet(resolve(responses.Content[i]), "content"))
			if content == nil || content.Kind != yaml.MappingNode {
				continue
			}
			for j := 0; j < len(content.Content); j += 2 {
				if streamingMediaType(content.Content[j].Value) {
					return true
				}
			}
		}
	}
	// Swagger 2: the operation's "produces", or the document's when absent.
	produces := resolve(mapGet(op, "produces"))
	if produces == nil {
		produces = resolve(mapGet(root, "produces"))
	}
	if produces != nil && produces.Kind == yaml.SequenceNode {
		for _, n := range produces.Content {
			if streamingMediaType(n.Value) {
				return true
			}
		}
	}
	return false
}

func streamingMediaType(s string) bool {
	if i := strings.Index(s, ";"); i >= 0 {
		s = s[:i]
	}
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "text/event-stream", "application/x-ndjson":
		return true
	}
	return false
}

func basePath(root *yaml.Node) string {
	raw := scalar(mapGet(root, "basePath"))
	if servers := resolve(mapGet(root, "servers")); servers != nil && servers.Kind == yaml.SequenceNode && len(servers.Content) > 0 {
		raw = scalar(mapGet(resolve(servers.Content[0]), "url"))
		if i := strings.Index(raw, "://"); i >= 0 {
			rest := raw[i+3:]
			if j := strings.Index(rest, "/"); j >= 0 {
				raw = rest[j:]
			} else {
				raw = ""
			}
		}
	}
	if i := strings.IndexAny(raw, "?#"); i >= 0 {
		raw = raw[:i]
	}
	if !strings.HasPrefix(raw, "/") {
		return ""
	}
	return strings.TrimRight(raw, "/")
}

func resolve(n *yaml.Node) *yaml.Node {
	for n != nil && n.Kind == yaml.AliasNode {
		n = n.Alias
	}
	return n
}

func mapGet(n *yaml.Node, key string) *yaml.Node {
	n = resolve(n)
	if n == nil || n.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(n.Content); i += 2 {
		if n.Content[i].Value == key {
			return n.Content[i+1]
		}
	}
	return nil
}

func scalar(n *yaml.Node) string {
	n = resolve(n)
	if n == nil || n.Kind != yaml.ScalarNode || n.Tag == "!!null" {
		return ""
	}
	return n.Value
}
