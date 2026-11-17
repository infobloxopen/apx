package optable

import "strings"

// JoinPath joins a prefix and an operation path into one normalized external
// path template. Empty segments are dropped, so the result has single slashes
// and no trailing slash. The root is "/".
func JoinPath(prefix, path string) string {
	segs := append(segments(prefix), segments(path)...)
	if len(segs) == 0 {
		return "/"
	}
	return "/" + strings.Join(segs, "/")
}

// segments splits a path on "/" and drops the empty segments.
func segments(path string) []string {
	var out []string
	for _, s := range strings.Split(path, "/") {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

// isVar reports whether a segment is a path variable: the whole segment is
// "{name}".
func isVar(seg string) bool {
	return len(seg) >= 2 && strings.HasPrefix(seg, "{") && strings.HasSuffix(seg, "}")
}

// shape returns the path with each variable segment replaced by "{}". Two
// templates with the same shape match the same requests.
func shape(path string) string {
	segs := segments(path)
	if len(segs) == 0 {
		return "/"
	}
	for i, s := range segs {
		if isVar(s) {
			segs[i] = "{}"
		}
	}
	return "/" + strings.Join(segs, "/")
}
