package pathlint

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// loadSpec reads one OpenAPI v3 or Swagger v2 document (JSON or YAML —
// yaml.v3 parses both) and expands every path key into its full
// external form: base (servers[0].url's path component, or basePath) +
// the path key itself.
func loadSpec(path string) ([]SpecPath, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc map[string]interface{}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	base := ""
	switch {
	case isOpenAPIv3(doc):
		base = openAPIv3Base(doc)
	case isSwaggerV2(doc):
		if bp, ok := doc["basePath"].(string); ok {
			base = bp
		}
	default:
		return nil, fmt.Errorf("%s: unrecognized spec format (expected \"openapi: 3.x\" or \"swagger: 2.0\")", path)
	}

	pathsRaw, ok := doc["paths"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("%s: no top-level \"paths\" object found", path)
	}

	name := filepath.Base(path)
	out := make([]SpecPath, 0, len(pathsRaw))
	for p := range pathsRaw {
		full := joinPath(base, p)
		segs := splitClean(full)
		out = append(out, SpecPath{Path: joinSegments(segs), Segments: segs, Spec: name, Base: base})
	}
	return out, nil
}

func isOpenAPIv3(doc map[string]interface{}) bool {
	v, ok := doc["openapi"].(string)
	return ok && strings.HasPrefix(v, "3")
}

func isSwaggerV2(doc map[string]interface{}) bool {
	v, ok := doc["swagger"].(string)
	return ok && strings.HasPrefix(v, "2")
}

// openAPIv3Base extracts the path component of servers[0].url. Servers
// may be a full absolute URL ("https://host/api/v1") or a bare path
// ("/api/v1"); either way we only care about the path.
func openAPIv3Base(doc map[string]interface{}) string {
	servers, ok := doc["servers"].([]interface{})
	if !ok || len(servers) == 0 {
		return ""
	}
	first, ok := servers[0].(map[string]interface{})
	if !ok {
		return ""
	}
	raw, ok := first["url"].(string)
	if !ok {
		return ""
	}
	if strings.HasPrefix(raw, "/") {
		return raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return u.Path
}

func joinPath(base, p string) string {
	b := strings.TrimSuffix(base, "/")
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return b + p
}
