package schema

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

var (
	parquetMsgRe = regexp.MustCompile(`^\s*message\s+([\w_]+)\s*\{`)
	parquetColRe = regexp.MustCompile(
		`^\s*(required|optional|repeated)\s+([\w_]+)\s+([\w_]+)(?:\s*\(([^)]+)\))?\s*;`)
)

// ExtractParquet parses a Parquet message-notation schema file.
func ExtractParquet(filePath string) (*ParquetSchema, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", filePath, err)
	}
	defer f.Close()

	var result ParquetSchema
	foundHeader := false
	depth := 0

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Skip blank lines and comments.
		if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "#") {
			continue
		}

		if !foundHeader {
			m := parquetMsgRe.FindStringSubmatch(line)
			if m == nil {
				continue // skip lines before message header
			}
			result.MessageName = m[1]
			foundHeader = true
			depth = 1
			continue
		}

		// Track nesting depth.
		depth += strings.Count(line, "{") - strings.Count(line, "}")

		if depth <= 0 {
			break // end of top-level message
		}

		// Only parse top-level columns (depth == 1).
		if depth > 1 {
			continue
		}

		if m := parquetColRe.FindStringSubmatch(trimmed); m != nil {
			result.Columns = append(result.Columns, ParquetColumn{
				Repetition: m[1],
				PhysType:   m[2],
				Name:       m[3],
				Annotation: m[4],
			})
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning %s: %w", filePath, err)
	}

	if !foundHeader {
		return nil, fmt.Errorf("no message definition found in %s", filePath)
	}

	return &result, nil
}
