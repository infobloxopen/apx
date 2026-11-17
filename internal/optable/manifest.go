package optable

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"

	"gopkg.in/yaml.v3"
)

// The names that the manifests use.
const (
	// LabelOperationTable marks the ConfigMap of an operation table.
	LabelOperationTable = "apx.infoblox.com/operation-table"
	// AnnotationAPI holds the API id. An API id has slashes, so it cannot be a
	// label value.
	AnnotationAPI = "apx.infoblox.com/api"
	// AnnotationAPIVersion holds the pinned release of the API.
	AnnotationAPIVersion = "apx.infoblox.com/api-version"
	// TableDataKey is the data key of an operation-table ConfigMap.
	TableDataKey = "operations.json"
	// RouteTableFile is the file name of the route table.
	RouteTableFile = "route-table.json"
	// RouteTableDataKey is the binaryData key of a route-table ConfigMap.
	RouteTableDataKey = "route-table.json.gz"
)

// ConfigMapName returns the name of the operation-table ConfigMap of an API:
// "optable-" and the first 10 hex digits of the SHA-256 of the API id.
func ConfigMapName(apiID string) string {
	sum := sha256.Sum256([]byte(apiID))
	return "optable-" + hex.EncodeToString(sum[:])[:10]
}

// JSON encodes the table as the value of the data key operations.json.
func (t *Table) JSON() ([]byte, error) { return encodeJSON(t) }

// JSON encodes the route table as the file route-table.json.
func (rt *RouteTable) JSON() ([]byte, error) { return encodeJSON(rt) }

func encodeJSON(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ConfigMap returns the ConfigMap manifest of the operation table. namespace
// is optional.
func (t *Table) ConfigMap(namespace string) ([]byte, error) {
	data, err := t.JSON()
	if err != nil {
		return nil, err
	}
	meta := []*yaml.Node{str("name"), str(ConfigMapName(t.API))}
	if namespace != "" {
		meta = append(meta, str("namespace"), str(namespace))
	}
	meta = append(meta,
		str("labels"), mapping(str(LabelOperationTable), quoted("true")),
		str("annotations"), mapping(
			str(AnnotationAPI), str(t.API),
			str(AnnotationAPIVersion), str(t.Version),
		),
	)
	return encodeYAML(mapping(
		str("apiVersion"), str("v1"),
		str("kind"), str("ConfigMap"),
		str("metadata"), mapping(meta...),
		str("data"), mapping(str(TableDataKey), literal(string(data))),
	))
}

// ConfigMap returns a ConfigMap manifest that holds the route table as gzip
// data under the binaryData key route-table.json.gz. namespace is optional.
func (rt *RouteTable) ConfigMap(name, namespace string) ([]byte, error) {
	data, err := rt.JSON()
	if err != nil {
		return nil, err
	}
	zipped, err := Gzip(data)
	if err != nil {
		return nil, err
	}
	meta := []*yaml.Node{str("name"), str(name)}
	if namespace != "" {
		meta = append(meta, str("namespace"), str(namespace))
	}
	return encodeYAML(mapping(
		str("apiVersion"), str("v1"),
		str("kind"), str("ConfigMap"),
		str("metadata"), mapping(meta...),
		str("binaryData"), mapping(str(RouteTableDataKey), str(base64.StdEncoding.EncodeToString(zipped))),
	))
}

// Gzip compresses data with no file name and no modification time, so the
// output depends only on the input.
func Gzip(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	zw, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return nil, err
	}
	if _, err := zw.Write(data); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Gunzip is the inverse of Gzip.
func Gunzip(data []byte) ([]byte, error) {
	zr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer func() { _ = zr.Close() }()
	return io.ReadAll(zr)
}

func encodeYAML(node *yaml.Node) ([]byte, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(node); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func str(s string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: s}
}

func quoted(s string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: s, Style: yaml.DoubleQuotedStyle}
}

func literal(s string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: s, Style: yaml.LiteralStyle}
}

func mapping(pairs ...*yaml.Node) *yaml.Node {
	return &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Content: pairs}
}
