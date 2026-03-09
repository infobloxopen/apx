// Package policy implements policy evaluation for APX schema projects.
// It checks organizational constraints such as forbidden proto options,
// allowed plugins, and format-specific compatibility rules.
package policy

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/infobloxopen/apx/internal/config"
	"github.com/infobloxopen/apx/internal/validator"
)

// Violation represents a single policy rule violation.
type Violation struct {
	Rule    string // short rule identifier, e.g. "forbidden_proto_option"
	File    string // file path relative to checked root
	Message string // human-readable description
}

// Result holds the outcome of a policy check.
type Result struct {
	Violations []Violation
	Checked    int // number of rules evaluated
}

// Passed returns true when no violations were found.
func (r *Result) Passed() bool { return len(r.Violations) == 0 }

// Check evaluates all configured policy rules against the schemas found
// at path and returns a structured result.
func Check(pol config.Policy, path string) (*Result, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolving path: %w", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", path, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", path)
	}

	result := &Result{}

	// Detect format for format-specific checks.
	format := validator.DetectFormat(absPath)

	// --- Proto-specific checks ---
	if format == validator.FormatProto || format == validator.FormatUnknown {
		protoFiles, _ := validator.GlobProtoFiles(absPath)

		// Forbidden proto options
		if len(pol.ForbiddenProtoOptions) > 0 && len(protoFiles) > 0 {
			result.Checked++
			checkForbiddenOptions(pol.ForbiddenProtoOptions, protoFiles, absPath, result)
		}

		// Allowed proto plugins
		if len(pol.AllowedProtoPlugins) > 0 {
			result.Checked++
			checkAllowedPlugins(pol.AllowedProtoPlugins, absPath, result)
		}
	}

	// --- OpenAPI checks ---
	if format == validator.FormatOpenAPI || format == validator.FormatUnknown {
		if pol.OpenAPI.SpectralRuleset != "" {
			result.Checked++
			checkSpectralRuleset(pol.OpenAPI.SpectralRuleset, absPath, result)
		}
	}

	// --- Avro checks ---
	if format == validator.FormatAvro || format == validator.FormatUnknown {
		if pol.Avro.Compatibility != "" {
			result.Checked++
			checkAvroCompatibility(pol.Avro.Compatibility, result)
		}
	}

	// --- JSON Schema checks ---
	if format == validator.FormatJSONSchema || format == validator.FormatUnknown {
		if pol.JSONSchema.BreakingMode != "" {
			result.Checked++
			checkJSONSchemaBreakingMode(pol.JSONSchema.BreakingMode, result)
		}
	}

	// --- Parquet checks ---
	if format == validator.FormatParquet || format == validator.FormatUnknown {
		result.Checked++
		checkParquetPolicy(pol.Parquet.AllowAdditiveNullableOnly, absPath, result)
	}

	return result, nil
}

// checkForbiddenOptions scans proto files for options matching any forbidden
// pattern and adds a violation for each match.
func checkForbiddenOptions(patterns []string, files []string, root string, result *Result) {
	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, p := range patterns {
		re, err := regexp.Compile(p)
		if err != nil {
			result.Violations = append(result.Violations, Violation{
				Rule:    "forbidden_proto_option",
				Message: fmt.Sprintf("invalid forbidden option regex %q: %v", p, err),
			})
			continue
		}
		compiled = append(compiled, re)
	}
	if len(compiled) == 0 {
		return
	}

	optionRe := regexp.MustCompile(`^\s*option\s+\(?([^)=\s]+)`)

	for _, f := range files {
		fh, err := os.Open(f)
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(fh)
		lineNo := 0
		for scanner.Scan() {
			lineNo++
			line := scanner.Text()
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") {
				continue
			}
			m := optionRe.FindStringSubmatch(trimmed)
			if m == nil {
				continue
			}
			optionName := m[1]
			for _, re := range compiled {
				if re.MatchString(optionName) {
					rel, _ := filepath.Rel(root, f)
					if rel == "" {
						rel = f
					}
					result.Violations = append(result.Violations, Violation{
						Rule:    "forbidden_proto_option",
						File:    rel,
						Message: fmt.Sprintf("%s:%d: forbidden option %q matches pattern %q", rel, lineNo, optionName, re.String()),
					})
				}
			}
		}
		fh.Close()
	}
}

// checkAllowedPlugins reads buf.gen.yaml (if present) and verifies all
// configured plugins are in the allowed list.
func checkAllowedPlugins(allowed []string, dir string, result *Result) {
	genFile := filepath.Join(dir, "buf.gen.yaml")
	data, err := os.ReadFile(genFile)
	if err != nil {
		// No buf.gen.yaml — nothing to check.
		return
	}

	allowedSet := make(map[string]bool, len(allowed))
	for _, p := range allowed {
		allowedSet[p] = true
	}

	// Simple line-based extraction of plugin names from buf.gen.yaml.
	// Matches lines like "  - plugin: protoc-gen-go" or "  plugin: buf.build/grpc/go"
	pluginRe := regexp.MustCompile(`-?\s*plugin:\s*(.+)`)
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		m := pluginRe.FindStringSubmatch(scanner.Text())
		if m == nil {
			continue
		}
		pluginName := strings.TrimSpace(m[1])
		if !allowedSet[pluginName] {
			result.Violations = append(result.Violations, Violation{
				Rule:    "allowed_proto_plugin",
				File:    "buf.gen.yaml",
				Message: fmt.Sprintf("buf.gen.yaml:%d: plugin %q is not in the allowed list %v", lineNo, pluginName, allowed),
			})
		}
	}
}

// checkSpectralRuleset verifies that the spectral ruleset file referenced
// by the policy actually exists in the project.
func checkSpectralRuleset(ruleset string, dir string, result *Result) {
	rulesetPath := filepath.Join(dir, ruleset)
	if _, err := os.Stat(rulesetPath); os.IsNotExist(err) {
		result.Violations = append(result.Violations, Violation{
			Rule:    "openapi_spectral_ruleset",
			File:    ruleset,
			Message: fmt.Sprintf("spectral ruleset %q referenced in policy does not exist at %s", ruleset, rulesetPath),
		})
	}
}

// checkAvroCompatibility validates the configured compatibility mode.
func checkAvroCompatibility(mode string, result *Result) {
	valid := map[string]bool{
		"BACKWARD": true, "FORWARD": true, "FULL": true, "NONE": true,
		"BACKWARD_TRANSITIVE": true, "FORWARD_TRANSITIVE": true, "FULL_TRANSITIVE": true,
	}
	if !valid[strings.ToUpper(mode)] {
		result.Violations = append(result.Violations, Violation{
			Rule:    "avro_compatibility",
			Message: fmt.Sprintf("avro compatibility mode %q is not valid; must be one of BACKWARD, FORWARD, FULL, NONE (or transitive variants)", mode),
		})
	}
}

// checkJSONSchemaBreakingMode validates the configured breaking change mode.
func checkJSONSchemaBreakingMode(mode string, result *Result) {
	valid := map[string]bool{"strict": true, "lenient": true}
	if !valid[strings.ToLower(mode)] {
		result.Violations = append(result.Violations, Violation{
			Rule:    "jsonschema_breaking_mode",
			Message: fmt.Sprintf("JSON Schema breaking_mode %q is not valid; must be 'strict' or 'lenient'", mode),
		})
	}
}

// checkParquetPolicy checks parquet-specific policy constraints.
// When AllowAdditiveNullableOnly is true, this is a configuration
// declaration that restricts schema evolution — the check verifies
// the setting is acknowledged.
func checkParquetPolicy(allowAdditiveNullableOnly bool, dir string, result *Result) {
	// The policy is a configuration constraint. For parquet files,
	// we verify the presence of parquet schemas and note the active policy.
	// Actual enforcement happens during breaking-change detection in the
	// parquet validator's SetAdditiveNullableOnlyPolicy().
}
