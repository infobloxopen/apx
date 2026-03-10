package schema

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var (
	protoSyntaxRe  = regexp.MustCompile(`^\s*syntax\s*=\s*"(proto[23])"\s*;`)
	protoPackageRe = regexp.MustCompile(`^\s*package\s+([\w.]+)\s*;`)
	protoImportRe  = regexp.MustCompile(`^\s*import\s+(?:public\s+)?"([^"]+)"\s*;`)
	protoOptionRe  = regexp.MustCompile(`^\s*option\s+([\w.]+)\s*=\s*"([^"]+)"\s*;`)
	protoServiceRe = regexp.MustCompile(`^\s*service\s+(\w+)\s*\{?`)
	protoMessageRe = regexp.MustCompile(`^\s*message\s+(\w+)\s*\{?`)
	protoEnumRe    = regexp.MustCompile(`^\s*enum\s+(\w+)\s*\{?`)
	protoRPCRe     = regexp.MustCompile(`^\s*rpc\s+(\w+)\s*\(\s*(stream\s+)?(\w[\w.]*)\s*\)\s*returns\s*\(\s*(stream\s+)?(\w[\w.]*)\s*\)`)
	protoFieldRe   = regexp.MustCompile(`^\s*(repeated\s+|optional\s+|required\s+)?(?:map\s*<\s*(\w+)\s*,\s*(\w[\w.]*)\s*>|(\w[\w.]*(?:\.\w+)*))\s+(\w+)\s*=\s*(\d+)`)
	protoEnumValRe = regexp.MustCompile(`^\s*(\w+)\s*=\s*(-?\d+)`)
)

// ExtractProto parses a .proto file and extracts its documentation-level
// structure: syntax, package, imports, services, messages, and enums.
// This is a best-effort line-based parser, not a full protobuf compiler.
func ExtractProto(filePath string) (*ProtoFile, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", filePath, err)
	}
	defer f.Close()

	result := &ProtoFile{}
	var lines []string

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning %s: %w", filePath, err)
	}

	p := &protoParser{lines: lines, pos: 0, result: result}
	p.parseTopLevel()

	return result, nil
}

type protoParser struct {
	lines  []string
	pos    int
	result *ProtoFile
}

func (p *protoParser) parseTopLevel() {
	for p.pos < len(p.lines) {
		line := p.lines[p.pos]
		trimmed := strings.TrimSpace(line)

		// Collect leading comment.
		comment := p.collectComment()

		if p.pos >= len(p.lines) {
			break
		}

		line = p.lines[p.pos]
		trimmed = strings.TrimSpace(line)

		// Skip blank lines.
		if trimmed == "" {
			p.pos++
			continue
		}

		// Skip single-line comments that weren't followed by a declaration.
		if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") {
			p.pos++
			continue
		}

		if m := protoSyntaxRe.FindStringSubmatch(line); m != nil {
			p.result.Syntax = m[1]
			p.pos++
			continue
		}

		if m := protoPackageRe.FindStringSubmatch(line); m != nil {
			p.result.Package = m[1]
			p.pos++
			continue
		}

		if m := protoImportRe.FindStringSubmatch(line); m != nil {
			p.result.Imports = append(p.result.Imports, m[1])
			p.pos++
			continue
		}

		if m := protoOptionRe.FindStringSubmatch(line); m != nil {
			p.result.Options = append(p.result.Options, ProtoOption{Name: m[1], Value: m[2]})
			p.pos++
			continue
		}

		if m := protoServiceRe.FindStringSubmatch(line); m != nil {
			svc := p.parseService(m[1], comment)
			p.result.Services = append(p.result.Services, svc)
			continue
		}

		if m := protoMessageRe.FindStringSubmatch(line); m != nil {
			msg := p.parseMessage(m[1], comment)
			p.result.Messages = append(p.result.Messages, msg)
			continue
		}

		if m := protoEnumRe.FindStringSubmatch(line); m != nil {
			enum := p.parseEnum(m[1], comment)
			p.result.Enums = append(p.result.Enums, enum)
			continue
		}

		p.pos++
	}
}

// collectComment gathers consecutive // comment lines and returns the text.
// It advances p.pos past the comment block.
func (p *protoParser) collectComment() string {
	var lines []string
	for p.pos < len(p.lines) {
		trimmed := strings.TrimSpace(p.lines[p.pos])
		if strings.HasPrefix(trimmed, "//") {
			text := strings.TrimPrefix(trimmed, "//")
			text = strings.TrimPrefix(text, " ")
			lines = append(lines, text)
			p.pos++
		} else {
			break
		}
	}
	return strings.Join(lines, " ")
}

func (p *protoParser) parseService(name, comment string) ProtoService {
	svc := ProtoService{Name: name, Comment: comment}

	// Advance past the opening line (which may or may not have '{').
	declLine := p.lines[p.pos]
	p.pos++
	depth := 1
	if !strings.Contains(declLine, "{") {
		// The '{' might be on the next line.
		for p.pos < len(p.lines) {
			if strings.Contains(p.lines[p.pos], "{") {
				declLine = p.lines[p.pos]
				depth = 1
				p.pos++
				break
			}
			p.pos++
		}
	}

	// Handle single-line services (rare but possible): service Foo { }
	if strings.Contains(declLine, "{") && strings.Contains(declLine, "}") {
		return svc
	}

	for p.pos < len(p.lines) && depth > 0 {
		line := p.lines[p.pos]
		trimmed := strings.TrimSpace(line)

		depth += strings.Count(line, "{") - strings.Count(line, "}")
		if depth <= 0 {
			p.pos++
			break
		}

		// Collect RPC-level comment.
		rpcComment := ""
		if strings.HasPrefix(trimmed, "//") {
			rpcComment = p.collectComment()
			if p.pos >= len(p.lines) {
				break
			}
			line = p.lines[p.pos]
			trimmed = strings.TrimSpace(line)
		}

		if m := protoRPCRe.FindStringSubmatch(line); m != nil {
			rpc := ProtoRPC{
				Name:            m[1],
				ClientStreaming: strings.TrimSpace(m[2]) == "stream",
				InputType:       m[3],
				ServerStreaming: strings.TrimSpace(m[4]) == "stream",
				OutputType:      m[5],
				Comment:         rpcComment,
			}
			// Also check for inline trailing comment.
			if rpc.Comment == "" {
				if idx := strings.Index(line, "//"); idx >= 0 {
					rpc.Comment = strings.TrimSpace(strings.TrimPrefix(line[idx:], "//"))
				}
			}
			svc.Methods = append(svc.Methods, rpc)

			// Skip brace depth for inline `{ ... }` after the rpc.
			depth += strings.Count(line, "{") - strings.Count(line, "}")
		}

		p.pos++
	}

	return svc
}

func (p *protoParser) parseMessage(name, comment string) ProtoMessage {
	msg := ProtoMessage{Name: name, Comment: comment}

	declLine := p.lines[p.pos]
	p.pos++
	depth := 1
	if !strings.Contains(declLine, "{") {
		for p.pos < len(p.lines) {
			if strings.Contains(p.lines[p.pos], "{") {
				declLine = p.lines[p.pos]
				depth = 1
				p.pos++
				break
			}
			p.pos++
		}
	}

	// Handle single-line messages like: message Foo { string id = 1; }
	if strings.Contains(declLine, "{") && strings.Contains(declLine, "}") {
		// Extract content between { and }.
		openIdx := strings.Index(declLine, "{")
		closeIdx := strings.LastIndex(declLine, "}")
		if openIdx >= 0 && closeIdx > openIdx {
			inner := strings.TrimSpace(declLine[openIdx+1 : closeIdx])
			// Try to parse each semicolon-separated statement as a field.
			for _, stmt := range strings.Split(inner, ";") {
				stmt = strings.TrimSpace(stmt)
				if stmt == "" {
					continue
				}
				if m := protoFieldRe.FindStringSubmatch(stmt); m != nil {
					label := strings.TrimSpace(m[1])
					var fieldType string
					if m[2] != "" {
						fieldType = "map<" + m[2] + ", " + m[3] + ">"
						label = "map"
					} else {
						fieldType = m[4]
					}
					fieldName := m[5]
					fieldNumber, _ := strconv.Atoi(m[6])
					field := ProtoField{
						Name:   fieldName,
						Number: fieldNumber,
						Type:   fieldType,
						Label:  label,
					}
					if idx := strings.Index(stmt, "//"); idx >= 0 {
						field.Comment = strings.TrimSpace(strings.TrimPrefix(stmt[idx:], "//"))
					}
					msg.Fields = append(msg.Fields, field)
				}
			}
		}
		return msg
	}

	for p.pos < len(p.lines) && depth > 0 {
		line := p.lines[p.pos]
		trimmed := strings.TrimSpace(line)

		// Collect field-level comment.
		fieldComment := ""
		if strings.HasPrefix(trimmed, "//") {
			fieldComment = p.collectComment()
			if p.pos >= len(p.lines) {
				break
			}
			line = p.lines[p.pos]
			trimmed = strings.TrimSpace(line)
		}

		// Check for nested message.
		if m := protoMessageRe.FindStringSubmatch(line); m != nil {
			nested := p.parseMessage(m[1], fieldComment)
			msg.Nested = append(msg.Nested, nested)
			continue
		}

		// Check for nested enum.
		if m := protoEnumRe.FindStringSubmatch(line); m != nil {
			enum := p.parseEnum(m[1], fieldComment)
			msg.Enums = append(msg.Enums, enum)
			continue
		}

		depth += strings.Count(line, "{") - strings.Count(line, "}")
		if depth <= 0 {
			p.pos++
			break
		}

		// Try to parse a field.
		if m := protoFieldRe.FindStringSubmatch(line); m != nil {
			label := strings.TrimSpace(m[1])
			var fieldType string
			if m[2] != "" {
				// map<K, V> field.
				fieldType = "map<" + m[2] + ", " + m[3] + ">"
				label = "map"
			} else {
				fieldType = m[4]
			}
			fieldName := m[5]
			fieldNumber, _ := strconv.Atoi(m[6])

			field := ProtoField{
				Name:    fieldName,
				Number:  fieldNumber,
				Type:    fieldType,
				Label:   label,
				Comment: fieldComment,
			}
			// Check for inline trailing comment.
			if field.Comment == "" {
				if idx := strings.Index(line, "//"); idx >= 0 {
					field.Comment = strings.TrimSpace(strings.TrimPrefix(line[idx:], "//"))
				}
			}
			msg.Fields = append(msg.Fields, field)
		}

		// Handle oneof blocks — skip the oneof keyword, parse inner fields.
		if strings.HasPrefix(trimmed, "oneof ") {
			// oneof adds a nesting level; inner fields parsed on next iterations.
		}

		p.pos++
	}

	return msg
}

func (p *protoParser) parseEnum(name, comment string) ProtoEnum {
	enum := ProtoEnum{Name: name, Comment: comment}

	declLine := p.lines[p.pos]
	p.pos++
	depth := 1
	if !strings.Contains(declLine, "{") {
		for p.pos < len(p.lines) {
			if strings.Contains(p.lines[p.pos], "{") {
				declLine = p.lines[p.pos]
				depth = 1
				p.pos++
				break
			}
			p.pos++
		}
	}

	// Handle single-line enums (rare): enum Foo { }
	if strings.Contains(declLine, "{") && strings.Contains(declLine, "}") {
		return enum
	}

	for p.pos < len(p.lines) && depth > 0 {
		line := p.lines[p.pos]
		trimmed := strings.TrimSpace(line)

		depth += strings.Count(line, "{") - strings.Count(line, "}")
		if depth <= 0 {
			p.pos++
			break
		}

		// Skip comments, options, reserved.
		if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "option ") || strings.HasPrefix(trimmed, "reserved ") {
			p.pos++
			continue
		}

		if m := protoEnumValRe.FindStringSubmatch(trimmed); m != nil {
			num, _ := strconv.Atoi(m[2])
			enum.Values = append(enum.Values, ProtoEnumValue{
				Name:   m[1],
				Number: num,
			})
		}

		p.pos++
	}

	return enum
}
