// Package pathlint reconciles the HTTP paths a service's rendered Kubernetes
// Ingress actually exposes against the paths its published OpenAPI/Swagger spec
// declares. It reports coverage as a metric (undeclared / unreachable /
// matched) rather than a hard boolean gate — see WS-035 decision R4.
package pathlint

// Rule is one normalized externally-reachable path pattern taken from a
// rendered Kubernetes Ingress object.
type Rule struct {
	Path     string   // normalized path, e.g. "/v2/current_user" or "/"
	Segments []string // path split on "/", empty entries dropped
	Wildcard bool     // true = this rule also matches everything *under* Path (Prefix semantics)
	Host     string   // ingress rule host, "*" if unknown/unset
	Source   string   // provenance string for the report (object name + origin)
}

// SpecPath is one fully-expanded external path a spec claims to serve,
// i.e. (servers[0].url path | basePath) + the spec's relative path key.
type SpecPath struct {
	Path     string
	Segments []string
	Spec     string // spec file the path came from (base name)
	Base     string // the base path applied (servers url path, or swagger basePath)
}

func splitClean(p string) []string {
	// Trim leading/trailing slashes then split; collapse to nil for root.
	start, end := 0, len(p)
	for start < end && p[start] == '/' {
		start++
	}
	for end > start && p[end-1] == '/' {
		end--
	}
	p = p[start:end]
	if p == "" {
		return nil
	}
	return splitOnSlash(p)
}

func splitOnSlash(p string) []string {
	var out []string
	last := 0
	for i := 0; i < len(p); i++ {
		if p[i] == '/' {
			out = append(out, p[last:i])
			last = i + 1
		}
	}
	out = append(out, p[last:])
	return out
}

func joinSegments(segs []string) string {
	if len(segs) == 0 {
		return "/"
	}
	out := ""
	for _, s := range segs {
		out += "/" + s
	}
	return out
}
