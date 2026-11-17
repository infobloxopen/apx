package opclass

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"gopkg.in/yaml.v3"
)

// Annotate adds x-operation-class to each operation of the document that lacks
// it and returns the new document with the number of operations that it
// changed. An operation that already has the extension keeps its value.
//
// Annotate does not re-encode the document. It inserts text at the positions
// that the YAML parser reports, so every other byte stays as it was: comments,
// key order, quoting, indentation and line endings (LF or CRLF).
//
//   - Block style: one new line, after the operationId line when the id is a
//     one-line scalar, otherwise before the first key of the operation.
//   - Flow style and JSON: a quoted member, as the first member of the object.
//
// Limits: an operation that the document reaches through a YAML alias is not
// changed (the count excludes it); an x-operation-class that arrives through a
// merge key ("<<") is not seen, so the operation gets its own; only the first
// YAML document of a stream is read.
func Annotate(data []byte) ([]byte, int, error) {
	root, err := parseRoot(data)
	if err != nil {
		return nil, 0, err
	}
	refs, err := walk(root)
	if err != nil {
		return nil, 0, err
	}
	src := newSource(data)
	var edits []edit
	seen := map[*yaml.Node]bool{}
	for _, ref := range refs {
		op, opErr := ref.operation(root)
		if opErr != nil {
			return nil, 0, opErr
		}
		if op.Rule == RuleExplicit || ref.aliased || seen[ref.node] {
			continue
		}
		seen[ref.node] = true
		e, edErr := src.editFor(ref.node, op.Class)
		if edErr != nil {
			return nil, 0, fmt.Errorf("%s %s: %w", ref.method, ref.path, edErr)
		}
		edits = append(edits, e)
	}
	if len(edits) == 0 {
		return data, 0, nil
	}
	sort.SliceStable(edits, func(i, j int) bool { return edits[i].offset < edits[j].offset })
	var out bytes.Buffer
	pos := 0
	for _, e := range edits {
		out.Write(data[pos:e.offset])
		out.WriteString(e.text)
		pos = e.offset
	}
	out.Write(data[pos:])

	// The result must parse to the same operations with the same classes.
	before, err := Parse(data)
	if err != nil {
		return nil, 0, err
	}
	after, err := Parse(out.Bytes())
	if err != nil {
		return nil, 0, fmt.Errorf("the annotated document does not parse: %w", err)
	}
	if len(before.Operations) != len(after.Operations) {
		return nil, 0, fmt.Errorf("the annotated document has %d operations, expected %d", len(after.Operations), len(before.Operations))
	}
	for i, b := range before.Operations {
		a := after.Operations[i]
		if a.Method != b.Method || a.Path != b.Path || a.OperationID != b.OperationID || a.Class != b.Class || a.Streaming != b.Streaming {
			return nil, 0, fmt.Errorf("%s %s: the annotated document differs from the source", b.Method, b.Path)
		}
	}
	return out.Bytes(), len(edits), nil
}

type edit struct {
	offset int
	text   string
}

// source maps the line and column of a YAML node to a byte offset.
type source struct {
	data  []byte
	lines []int // byte offset of the start of each line
}

func newSource(data []byte) *source {
	s := &source{data: data, lines: []int{0}}
	for i, b := range data {
		if b == '\n' {
			s.lines = append(s.lines, i+1)
		}
	}
	return s
}

// line returns the bytes of the 1-based line, with its line ending.
func (s *source) line(n int) ([]byte, int, error) {
	if n < 1 || n > len(s.lines) {
		return nil, 0, fmt.Errorf("line %d is outside the document", n)
	}
	start := s.lines[n-1]
	end := len(s.data)
	if n < len(s.lines) {
		end = s.lines[n]
	}
	return s.data[start:end], start, nil
}

// offset converts a 1-based line and a 1-based column (in characters) to a
// byte offset.
func (s *source) offset(line, col int) (int, error) {
	text, start, err := s.line(line)
	if err != nil {
		return 0, err
	}
	off := 0
	for i := 1; i < col; i++ {
		if off >= len(text) {
			return 0, fmt.Errorf("column %d is outside line %d", col, line)
		}
		_, size := utf8.DecodeRune(text[off:])
		off += size
	}
	return start + off, nil
}

func eolOf(line []byte) string {
	if bytes.HasSuffix(line, []byte("\r\n")) {
		return "\r\n"
	}
	return "\n"
}

func (s *source) editFor(op *yaml.Node, class Class) (edit, error) {
	if op.Style&yaml.FlowStyle != 0 {
		return s.flowEdit(op, class)
	}
	if len(op.Content) < 2 {
		return edit{}, fmt.Errorf("the operation has no keys")
	}
	first := op.Content[0]
	firstLine, firstStart, err := s.line(first.Line)
	if err != nil {
		return edit{}, err
	}
	keyOff, err := s.offset(first.Line, first.Column)
	if err != nil {
		return edit{}, err
	}
	indent := string(s.data[firstStart:keyOff])
	if strings.Trim(indent, " ") != "" {
		return edit{}, fmt.Errorf("line %d: the first key of the operation does not start its line", first.Line)
	}
	eol := eolOf(firstLine)
	text := indent + ExtensionClass + ": " + string(class)

	// After the operationId line, when the id is a one-line scalar.
	for i := 0; i+1 < len(op.Content); i += 2 {
		k, v := op.Content[i], op.Content[i+1]
		if k.Value != "operationId" || v.Kind != yaml.ScalarNode || v.Line != k.Line ||
			v.Style&(yaml.LiteralStyle|yaml.FoldedStyle) != 0 || strings.ContainsAny(v.Value, " \t\r\n") {
			continue
		}
		idLine, idStart, lineErr := s.line(k.Line)
		if lineErr != nil {
			return edit{}, lineErr
		}
		if !bytes.HasSuffix(idLine, []byte("\n")) {
			return edit{offset: idStart + len(idLine), text: eol + text}, nil
		}
		return edit{offset: idStart + len(idLine), text: text + eolOf(idLine)}, nil
	}
	return edit{offset: firstStart, text: text + eol}, nil
}

func (s *source) flowEdit(op *yaml.Node, class Class) (edit, error) {
	open, err := s.offset(op.Line, op.Column)
	if err != nil {
		return edit{}, err
	}
	if open >= len(s.data) || s.data[open] != '{' {
		return edit{}, fmt.Errorf("line %d: expected the start of an object", op.Line)
	}
	member := fmt.Sprintf("%q: %q", ExtensionClass, string(class))
	if len(op.Content) == 0 {
		return edit{offset: open + 1, text: member}, nil
	}
	first := op.Content[0]
	if first.Line == op.Line {
		return edit{offset: open + 1, text: member + ", "}, nil
	}
	// A pretty-printed object: a new line with the indentation of the first key.
	firstLine, firstStart, err := s.line(first.Line)
	if err != nil {
		return edit{}, err
	}
	keyOff, err := s.offset(first.Line, first.Column)
	if err != nil {
		return edit{}, err
	}
	indent := string(s.data[firstStart:keyOff])
	if strings.Trim(indent, " \t") != "" {
		return edit{offset: open + 1, text: member + ", "}, nil
	}
	return edit{offset: firstStart, text: indent + member + "," + eolOf(firstLine)}, nil
}
