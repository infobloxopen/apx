package validator

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

// ParquetValidator handles Parquet schema validation.
// APX represents Parquet schemas as message-notation text files (.parquet),
// using the same schema language that parquet-tools outputs.
//
// Example schema file:
//
//	message user {
//	  required binary id (STRING);
//	  required binary name (STRING);
//	  optional int32 age;
//	}
type ParquetValidator struct {
	resolver                  *ToolchainResolver
	allowAdditiveNullableOnly bool
}

// NewParquetValidator creates a new Parquet validator
func NewParquetValidator(resolver *ToolchainResolver) *ParquetValidator {
	return &ParquetValidator{
		resolver:                  resolver,
		allowAdditiveNullableOnly: true,
	}
}

// SetAdditiveNullableOnlyPolicy sets whether only additive nullable columns are allowed
func (v *ParquetValidator) SetAdditiveNullableOnlyPolicy(allow bool) {
	v.allowAdditiveNullableOnly = allow
}

// parquetMessage is a parsed Parquet message schema.
type parquetMessage struct {
	Name    string
	Columns []parquetColumn
}

// parquetColumn is a single column definition in a Parquet schema.
type parquetColumn struct {
	Repetition string // required, optional, repeated
	PhysType   string // int32, int64, float, double, binary, boolean, etc.
	Name       string
	Annotation string // logical type annotation, e.g. STRING, DATE
}

// validParquetRepetitions is the set of valid repetition levels.
var validParquetRepetitions = map[string]bool{
	"required": true, "optional": true, "repeated": true,
}

// validParquetTypes is the set of valid Parquet physical types.
var validParquetTypes = map[string]bool{
	"boolean": true, "int32": true, "int64": true, "int96": true,
	"float": true, "double": true, "binary": true,
	"fixed_len_byte_array": true,
}

// validParquetAnnotations is the set of valid Parquet logical type annotations.
var validParquetAnnotations = map[string]bool{
	"STRING": true, "UTF8": true, "DATE": true,
	"TIMESTAMP_MILLIS": true, "TIMESTAMP_MICROS": true,
	"DECIMAL": true,
	"INT_8":   true, "INT_16": true, "INT_32": true, "INT_64": true,
	"UINT_8": true, "UINT_16": true, "UINT_32": true, "UINT_64": true,
	"JSON": true, "BSON": true,
	"MAP": true, "LIST": true,
	"ENUM": true, "UUID": true,
	"TIME_MILLIS": true, "TIME_MICROS": true,
	"INTERVAL": true,
}

// columnLineRe matches a flat column definition line:
//
//	<repetition> <type> <name> [(<annotation>)];
var columnLineRe = regexp.MustCompile(
	`^\s*(required|optional|repeated)\s+([\w_]+)\s+([\w_]+)(?:\s*\(([^)]+)\))?\s*;`)

// messageHeaderRe matches the opening line of a message schema.
var messageHeaderRe = regexp.MustCompile(`^\s*message\s+([\w_]+)\s*\{`)

// groupHeaderRe matches a nested group definition line.
var groupHeaderRe = regexp.MustCompile(`^\s*(required|optional|repeated)\s+group\s+\w+`)

// snakeCaseRe matches valid snake_case identifiers.
var snakeCaseRe = regexp.MustCompile(`^[a-z][a-z0-9]*(_[a-z0-9]+)*$`)

// isSnakeCase returns true if the string is a valid snake_case identifier.
func isSnakeCase(s string) bool {
	return snakeCaseRe.MatchString(s)
}

// isMessageNameValid checks that message names use PascalCase or snake_case.
func isMessageNameValid(name string) bool {
	if len(name) == 0 {
		return false
	}
	// Allow PascalCase (starts with uppercase) or snake_case (starts with lowercase)
	if unicode.IsUpper(rune(name[0])) {
		// PascalCase: all alphanumeric
		for _, r := range name {
			if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
				return false
			}
		}
		return true
	}
	return isSnakeCase(name)
}

// parseParquetSchema parses a Parquet message schema text file.
func parseParquetSchema(path string) (*parquetMessage, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", path, err)
	}
	defer f.Close()

	var msg parquetMessage
	var violations []string
	foundHeader := false
	lineNum := 0
	depth := 0
	seen := make(map[string]int) // column name → line number

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		lineNum++
		trimmed := strings.TrimSpace(line)

		// Skip blank lines and comments
		if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "#") {
			continue
		}

		if !foundHeader {
			m := messageHeaderRe.FindStringSubmatch(line)
			if m == nil {
				return nil, fmt.Errorf("line %d: expected 'message <name> {', got: %s", lineNum, trimmed)
			}
			msg.Name = m[1]
			if !isMessageNameValid(msg.Name) {
				violations = append(violations, fmt.Sprintf(
					"line %d: message name %q should be PascalCase or snake_case", lineNum, msg.Name))
			}
			foundHeader = true
			depth = 1
			continue
		}

		// Track nesting depth for group fields
		openCount := strings.Count(line, "{")
		closeCount := strings.Count(line, "}")
		depth += openCount - closeCount

		if depth <= 0 {
			// Closing brace of the top-level message
			break
		}

		// Only parse top-level (depth==1) columns; skip nested group contents
		if depth > 1 {
			continue
		}

		// Skip group headers (lines containing '{')
		if groupHeaderRe.MatchString(trimmed) || trimmed == "}" {
			continue
		}

		m := columnLineRe.FindStringSubmatch(line)
		if m == nil {
			// Closing brace is not an unrecognized line
			if trimmed == "}" {
				continue
			}
			violations = append(violations, fmt.Sprintf(
				"line %d: unrecognized column definition: %s", lineNum, trimmed))
			continue
		}

		col := parquetColumn{
			Repetition: m[1],
			PhysType:   m[2],
			Name:       m[3],
			Annotation: m[4],
		}

		if !validParquetRepetitions[col.Repetition] {
			return nil, fmt.Errorf("line %d: invalid repetition %q", lineNum, col.Repetition)
		}
		if !validParquetTypes[col.PhysType] {
			violations = append(violations, fmt.Sprintf(
				"line %d: unknown physical type %q for column %q", lineNum, col.PhysType, col.Name))
			continue
		}

		// Validate logical type annotation
		if col.Annotation != "" && !validParquetAnnotations[strings.TrimSpace(col.Annotation)] {
			violations = append(violations, fmt.Sprintf(
				"line %d: unknown logical type annotation %q for column %q", lineNum, col.Annotation, col.Name))
		}

		// Check column naming convention (snake_case)
		if !isSnakeCase(col.Name) {
			violations = append(violations, fmt.Sprintf(
				"line %d: column name %q should be snake_case", lineNum, col.Name))
		}

		// Check for duplicate column names
		if prevLine, exists := seen[col.Name]; exists {
			violations = append(violations, fmt.Sprintf(
				"line %d: duplicate column name %q (first defined on line %d)", lineNum, col.Name, prevLine))
		}
		seen[col.Name] = lineNum

		msg.Columns = append(msg.Columns, col)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	if !foundHeader {
		return nil, fmt.Errorf("no 'message' declaration found in %s", path)
	}

	// Empty message check
	if len(msg.Columns) == 0 {
		violations = append(violations, "message has no columns")
	}

	if len(violations) > 0 {
		return nil, fmt.Errorf("parquet lint errors:\n  %s", strings.Join(violations, "\n  "))
	}

	return &msg, nil
}

// Lint validates Parquet schema syntax using the native message-notation parser.
func (v *ParquetValidator) Lint(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}
	_, err = parseParquetSchema(absPath)
	return err
}

// Breaking checks for breaking changes between two Parquet schemas.
// path is the new schema; against is the old/baseline schema.
//
// Under the default policy (allowAdditiveNullableOnly=true):
//   - New optional columns are allowed (additive nullable)
//   - New required columns are breaking (old data lacks values for them)
//   - Removed columns are breaking
//   - Type changes are breaking
//   - required → optional is allowed (relaxing the constraint)
//   - optional → required is breaking
func (v *ParquetValidator) Breaking(path, against string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}
	absAgainst, err := filepath.Abs(against)
	if err != nil {
		return fmt.Errorf("failed to resolve against path: %w", err)
	}

	newMsg, err := parseParquetSchema(absPath)
	if err != nil {
		return fmt.Errorf("parsing new schema: %w", err)
	}
	oldMsg, err := parseParquetSchema(absAgainst)
	if err != nil {
		return fmt.Errorf("parsing old schema: %w", err)
	}

	oldCols := make(map[string]parquetColumn, len(oldMsg.Columns))
	for _, c := range oldMsg.Columns {
		oldCols[c.Name] = c
	}
	newCols := make(map[string]parquetColumn, len(newMsg.Columns))
	for _, c := range newMsg.Columns {
		newCols[c.Name] = c
	}

	var violations []string

	for _, nc := range newMsg.Columns {
		oc, existed := oldCols[nc.Name]
		if !existed {
			// New column: optional is additive-nullable (OK); required is breaking
			if nc.Repetition == "required" {
				violations = append(violations, fmt.Sprintf(
					"column %q added as required (old data has no values for it; add as optional instead)",
					nc.Name))
			}
			continue
		}

		// Type change
		if nc.PhysType != oc.PhysType {
			violations = append(violations, fmt.Sprintf(
				"column %q physical type changed from %s to %s",
				nc.Name, oc.PhysType, nc.PhysType))
		}

		// optional → required is breaking
		if oc.Repetition == "optional" && nc.Repetition == "required" {
			violations = append(violations, fmt.Sprintf(
				"column %q changed from optional to required (old data may contain null values)",
				nc.Name))
		}

		// Annotation (logical type) change is breaking
		if nc.Annotation != oc.Annotation {
			violations = append(violations, fmt.Sprintf(
				"column %q annotation changed from %q to %q (logical type change affects deserialization)",
				nc.Name, oc.Annotation, nc.Annotation))
		}
	}

	// Removed columns
	for _, oc := range oldMsg.Columns {
		if _, exists := newCols[oc.Name]; !exists {
			violations = append(violations, fmt.Sprintf(
				"column %q removed (readers depending on this column will break)",
				oc.Name))
		}
	}

	if len(violations) > 0 {
		return fmt.Errorf("parquet schema breaking changes:\n  %s",
			strings.Join(violations, "\n  "))
	}
	return nil
}
