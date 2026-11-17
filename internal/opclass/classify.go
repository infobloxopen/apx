package opclass

import (
	"fmt"
	"strings"
	"unicode"
)

// Extension names that this package reads or writes.
const (
	ExtensionClass     = "x-operation-class"
	ExtensionAIPMethod = "x-aip-method"
	ExtensionStreaming = "x-streaming"
)

// Class is the operation class.
type Class string

// The four operation classes.
const (
	Read   Class = "read"
	List   Class = "list"
	Write  Class = "write"
	Custom Class = "custom"
)

// Classes returns the valid classes in a fixed order.
func Classes() []Class { return []Class{Read, List, Write, Custom} }

// ParseClass validates s as an operation class. The match is exact: the
// extension value is lower case.
func ParseClass(s string) (Class, error) {
	switch c := Class(s); c {
	case Read, List, Write, Custom:
		return c, nil
	}
	return "", fmt.Errorf("invalid %s %q (expected read, list, write or custom)", ExtensionClass, s)
}

// Rule names the rule that decided a class.
type Rule string

// The rules, in the order in which Classify applies them.
const (
	RuleExplicit Rule = "explicit" // x-operation-class on the operation
	RuleAIP      Rule = "aip"      // x-aip-method other than Custom
	RuleName     Rule = "name"     // verb word in the operationId
	RuleShape    Rule = "shape"    // HTTP verb and path shape
	RuleDefault  Rule = "default"  // nothing matched: custom
)

// Input describes one operation for Classify. Only Method and Path are
// required; an empty field means that the document does not have the value.
type Input struct {
	Method      string // HTTP method, any case
	Path        string // path template, for example "/users/{id}"
	OperationID string
	AIPMethod   string // value of x-aip-method
	Explicit    string // value of x-operation-class
}

// Classify returns the class of one operation and the rule that decided it. The
// only error is an invalid explicit x-operation-class.
func Classify(in Input) (Class, Rule, error) {
	if in.Explicit != "" {
		c, err := ParseClass(in.Explicit)
		if err != nil {
			return "", "", err
		}
		return c, RuleExplicit, nil
	}
	if c, ok := classFromAIP(in.AIPMethod); ok {
		return c, RuleAIP, nil
	}
	if c, ok := classFromName(in.OperationID); ok {
		return c, RuleName, nil
	}
	if c, ok := classFromShape(in.Method, in.Path); ok {
		return c, RuleShape, nil
	}
	return Custom, RuleDefault, nil
}

func classFromAIP(method string) (Class, bool) {
	switch strings.ToLower(strings.TrimSpace(method)) {
	case "get", "batchget":
		return Read, true
	case "list", "search":
		return List, true
	case "create", "update", "delete", "undelete", "purge",
		"batchcreate", "batchupdate", "batchdelete":
		return Write, true
	}
	return "", false
}

func verbClass(word string) (Class, bool) {
	switch strings.ToLower(word) {
	case "list":
		return List, true
	case "read", "get":
		return Read, true
	case "create", "update", "patch", "delete":
		return Write, true
	}
	return "", false
}

func classFromName(id string) (Class, bool) {
	method, service := id, ""
	if i := strings.LastIndex(id, "_"); i >= 0 && i+1 < len(id) && unicode.IsUpper(rune(id[i+1])) {
		service, method = id[:i], id[i+1:]
	}
	words := splitWords(method)
	if service != "" {
		words = dropServicePrefix(words, splitWords(service))
	}
	if len(words) == 0 {
		return "", false
	}
	first, last := words[0], words[len(words)-1]
	if c, ok := verbClass(first); ok {
		if c == Read && len(words) > 1 && strings.EqualFold(last, "list") {
			return List, true
		}
		return c, true
	}
	return verbClass(last)
}

// dropServicePrefix removes the leading method words that repeat the leading
// service words. It always keeps one method word.
func dropServicePrefix(method, service []string) []string {
	n := 0
	for n < len(method)-1 && n < len(service) && sameNoun(method[n], service[n]) {
		n++
	}
	return method[n:]
}

// sameNoun reports whether a equals b, where b can carry a plural ending.
func sameNoun(a, b string) bool {
	a, b = strings.ToLower(a), strings.ToLower(b)
	if a == b || a+"s" == b || a+"es" == b {
		return true
	}
	return strings.HasSuffix(a, "y") && strings.TrimSuffix(a, "y")+"ies" == b
}

// splitWords splits an identifier into words. See the package documentation
// for the boundaries.
func splitWords(s string) []string {
	var words []string
	var cur []rune
	flush := func() {
		if len(cur) > 0 {
			words = append(words, string(cur))
			cur = cur[:0]
		}
	}
	r := []rune(s)
	for i, c := range r {
		if !unicode.IsLetter(c) && !unicode.IsDigit(c) {
			flush()
			continue
		}
		if unicode.IsUpper(c) && len(cur) > 0 {
			prev := r[i-1]
			nextLower := i+1 < len(r) && unicode.IsLower(r[i+1])
			if unicode.IsLower(prev) || unicode.IsDigit(prev) || (unicode.IsUpper(prev) && nextLower) {
				flush()
			}
		}
		cur = append(cur, c)
	}
	flush()
	return words
}

func classFromShape(method, path string) (Class, bool) {
	switch strings.ToUpper(strings.TrimSpace(method)) {
	case "POST", "PUT", "PATCH", "DELETE":
		return Write, true
	case "GET", "HEAD":
		last := lastSegment(path)
		switch {
		case last == "":
			return "", false
		case isVariable(last):
			return Read, true
		case strings.Contains(afterVariables(last), ":"):
			return "", false
		}
		return List, true
	}
	return "", false
}

func lastSegment(path string) string {
	path = strings.TrimRight(path, "/")
	return path[strings.LastIndex(path, "/")+1:]
}

// isVariable reports whether seg is exactly one path variable, "{name}".
func isVariable(seg string) bool {
	return len(seg) > 2 && strings.HasPrefix(seg, "{") && strings.HasSuffix(seg, "}") &&
		strings.Count(seg, "{") == 1
}

// afterVariables returns seg without its "{...}" parts, so that a colon inside
// a variable name is not read as a custom verb.
func afterVariables(seg string) string {
	var b strings.Builder
	depth := 0
	for _, c := range seg {
		switch {
		case c == '{':
			depth++
		case c == '}' && depth > 0:
			depth--
		case depth == 0:
			b.WriteRune(c)
		}
	}
	return b.String()
}
