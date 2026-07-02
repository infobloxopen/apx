package catalog

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// resourceOptionRe matches the head of a `google.api.resource` message option,
// e.g. `option (google.api.resource) = {`. The trailing brace may sit on the
// same line or a later line; we only anchor on the option name here.
var resourceOptionRe = regexp.MustCompile(`\(\s*google\.api\.resource\s*\)\s*=\s*\{`)

// typeFieldRe extracts the `type:` value from inside a resource option block,
// e.g. `type: "iam.example.com/User"`.
var typeFieldRe = regexp.MustCompile(`\btype\s*:\s*"([^"]+)"`)

// ScanResourceTypes walks the .proto files under dir and returns the AIP-122
// resource types declared via `option (google.api.resource) = { type: "..." }`.
//
// It is a source-level scanner, not a proto compiler: apx indexes existing
// annotations and does not generate or transform code (lifecycle-not-codegen).
// Comments are stripped so annotations inside `//` or `/* */` are ignored.
// The result is deduplicated and sorted. A missing dir yields an empty result
// (not an error) so generation can index only the modules present on disk.
func ScanResourceTypes(dir string) ([]string, error) {
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("stat proto dir %s: %w", dir, err)
	}
	if !info.IsDir() {
		return nil, nil
	}

	seen := make(map[string]bool)
	err = filepath.Walk(dir, func(path string, fi os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if fi.IsDir() {
			return nil
		}
		if strings.ToLower(filepath.Ext(path)) != ".proto" {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("read proto %s: %w", path, readErr)
		}
		for _, t := range extractResourceTypes(string(data)) {
			seen[t] = true
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	types := make([]string, 0, len(seen))
	for t := range seen {
		types = append(types, t)
	}
	sort.Strings(types)
	return types, nil
}

// extractResourceTypes returns the resource type strings declared in a single
// proto source. It strips comments first, then finds each
// `(google.api.resource) = { ... }` block and reads its `type:` field.
func extractResourceTypes(src string) []string {
	src = stripProtoComments(src)

	var types []string
	locs := resourceOptionRe.FindAllStringIndex(src, -1)
	for _, loc := range locs {
		// Scan from the opening brace to its matching close brace, tracking
		// nesting so a `type:` in an inner block still resolves to the option.
		block, ok := braceBlock(src, loc[1]-1) // loc[1]-1 points at the '{'
		if !ok {
			continue
		}
		if m := typeFieldRe.FindStringSubmatch(block); m != nil {
			t := strings.TrimSpace(m[1])
			if t != "" {
				types = append(types, t)
			}
		}
	}
	return types
}

// braceBlock returns the substring between the '{' at openIdx and its matching
// '}', tracking nested braces. openIdx must point at a '{'. The returned string
// excludes the outer braces. ok is false if no matching brace is found.
func braceBlock(s string, openIdx int) (string, bool) {
	if openIdx < 0 || openIdx >= len(s) || s[openIdx] != '{' {
		return "", false
	}
	depth := 0
	for i := openIdx; i < len(s); i++ {
		switch s[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return s[openIdx+1 : i], true
			}
		}
	}
	return "", false
}

// stripProtoComments removes // line comments and /* */ block comments while
// preserving string literals (so a "//" inside a quoted value survives).
func stripProtoComments(src string) string {
	var b strings.Builder
	b.Grow(len(src))

	const (
		normal = iota
		inString
		inLineComment
		inBlockComment
	)
	state := normal

	for i := 0; i < len(src); i++ {
		c := src[i]
		var next byte
		if i+1 < len(src) {
			next = src[i+1]
		}

		switch state {
		case normal:
			switch {
			case c == '"':
				state = inString
				b.WriteByte(c)
			case c == '/' && next == '/':
				state = inLineComment
				i++ // consume second '/'
			case c == '/' && next == '*':
				state = inBlockComment
				i++ // consume '*'
			default:
				b.WriteByte(c)
			}
		case inString:
			b.WriteByte(c)
			if c == '\\' && i+1 < len(src) {
				// keep escaped char verbatim
				b.WriteByte(next)
				i++
			} else if c == '"' {
				state = normal
			}
		case inLineComment:
			if c == '\n' {
				state = normal
				b.WriteByte(c) // keep newline for line structure
			}
		case inBlockComment:
			if c == '*' && next == '/' {
				state = normal
				i++ // consume '/'
			}
		}
	}
	return b.String()
}
