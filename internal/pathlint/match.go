package pathlint

import "strings"

// isWildcardSeg reports whether a path segment stands for "any value":
// an OpenAPI/gRPC-gateway template variable like "{id}" or "{id.value}",
// or a literal "*" (used internally for regex-derived ingress wildcards).
func isWildcardSeg(s string) bool {
	if s == "*" {
		return true
	}
	if len(s) >= 2 && strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}") {
		return true
	}
	return false
}

func segEqual(a, b string) bool {
	return a == b || isWildcardSeg(a) || isWildcardSeg(b)
}

// matches reports whether an ingress rule would route a request to the
// given spec path. Segment-wise, per the Kubernetes Ingress Prefix
// semantics: a Prefix rule matches its own path and any deeper path that
// shares it as a "/"-bounded prefix; an Exact rule (or a regex-derived
// rule where no wildcard tail was recognized) only matches the identical
// path shape.
func matches(rule Rule, sp SpecPath) bool {
	is, ss := rule.Segments, sp.Segments
	if len(ss) < len(is) {
		return false
	}
	for i := range is {
		if !segEqual(is[i], ss[i]) {
			return false
		}
	}
	if len(ss) == len(is) {
		return true
	}
	return rule.Wildcard
}
