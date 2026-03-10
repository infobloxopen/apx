package config

import (
	"fmt"
	"sort"
	"strings"
)

// FieldType represents the type of a configuration field.
type FieldType int

const (
	TypeInt    FieldType = iota // integer
	TypeString                  // string
	TypeBool                    // boolean
	TypeList                    // list/sequence
	TypeMap                     // map with dynamic keys → struct values
	TypeStruct                  // fixed-key mapping
)

// String returns the human-readable name for a FieldType.
func (ft FieldType) String() string {
	switch ft {
	case TypeInt:
		return "integer"
	case TypeString:
		return "string"
	case TypeBool:
		return "boolean"
	case TypeList:
		return "list"
	case TypeMap:
		return "map"
	case TypeStruct:
		return "struct"
	default:
		return "unknown"
	}
}

// FieldDef describes a single configuration field and its constraints.
type FieldDef struct {
	Name            string              // YAML key name
	Type            FieldType           // Expected YAML type
	Required        bool                // Whether the field must be present
	Default         interface{}         // Default value when omitted (nil = no default)
	Description     string              // Human-readable description
	EnumValues      []string            // If non-nil, value must be one of these
	Pattern         string              // If non-empty, value must match this pattern description
	Children        map[string]FieldDef // For TypeStruct: allowed child fields
	ItemDef         *FieldDef           // For TypeList: schema for each list element
	DeprecatedSince int                 // Schema version when deprecated (0 = active)
	Replacement     string              // Dotted path of replacement field
}

// DeprecatedField tracks a field that is deprecated in a given schema version.
type DeprecatedField struct {
	Field       string // Dotted field path
	Since       int    // Version in which it was deprecated
	Replacement string // Dotted path of replacement field
}

// SchemaVersion defines the complete allowed structure for apx.yaml at a version.
type SchemaVersion struct {
	Version        int                 // Schema version number
	Fields         map[string]FieldDef // Top-level fields and recursive definitions
	RequiredFields []string            // Top-level required field names
	Deprecated     []DeprecatedField   // Deprecated fields at this version
}

// MigrationFunc transforms a YAML node tree from version N to N+1.
// It returns the list of changes applied.
type MigrationFunc func(data []byte) (result []byte, changes []Change, err error)

// Change represents a single transformation applied during migration.
type Change struct {
	Action string `json:"action"` // added, removed, renamed, changed_default
	Field  string `json:"field"`  // Dotted field path affected
	Detail string `json:"detail"` // Human-readable explanation
}

// SchemaRegistry holds all known schema versions and migration chain.
type SchemaRegistry struct {
	Versions       map[int]SchemaVersion // All defined schema versions
	CurrentVersion int                   // Latest schema version supported
	Migrations     map[int]MigrationFunc // Migration from version N → N+1, keyed by source
}

// CurrentSchemaVersion is the latest schema version supported by this binary.
const CurrentSchemaVersion = 1

// Registry is the package-level schema registry singleton.
var Registry = buildRegistry()

func buildRegistry() *SchemaRegistry {
	v1 := buildV1Schema()
	return &SchemaRegistry{
		Versions:       map[int]SchemaVersion{1: v1},
		CurrentVersion: CurrentSchemaVersion,
		Migrations:     map[int]MigrationFunc{},
	}
}

// buildV1Schema constructs the complete field-definition tree for schema version 1.
func buildV1Schema() SchemaVersion {
	pluginItem := FieldDef{
		Name:        "plugin",
		Type:        TypeStruct,
		Description: "Plugin name and version",
		Children: map[string]FieldDef{
			"name": {
				Name:        "name",
				Type:        TypeString,
				Required:    true,
				Description: "Plugin name",
			},
			"version": {
				Name:        "version",
				Type:        TypeString,
				Required:    true,
				Description: "Plugin version",
			},
		},
	}

	languageTargetValue := FieldDef{
		Name:        "language_target",
		Type:        TypeStruct,
		Description: "Code generation target for a language",
		Children: map[string]FieldDef{
			"enabled": {
				Name:        "enabled",
				Type:        TypeBool,
				Description: "Whether this language target is active",
				Default:     false,
			},
			"tool": {
				Name:        "tool",
				Type:        TypeString,
				Description: "Tool name (e.g., grpcio-tools)",
			},
			"version": {
				Name:        "version",
				Type:        TypeString,
				Description: "Tool version",
			},
			"plugins": {
				Name:        "plugins",
				Type:        TypeList,
				Description: "List of plugin name/version maps",
				ItemDef:     &pluginItem,
			},
		},
	}

	fields := map[string]FieldDef{
		"version": {
			Name:        "version",
			Type:        TypeInt,
			Required:    true,
			Description: "Schema version number",
		},
		"org": {
			Name:        "org",
			Type:        TypeString,
			Required:    true,
			Description: "GitHub organization name",
		},
		"repo": {
			Name:        "repo",
			Type:        TypeString,
			Required:    true,
			Description: "Canonical API repository name",
		},
		"import_root": {
			Name:        "import_root",
			Type:        TypeString,
			Description: "Custom public Go import prefix (e.g. go.acme.dev/apis). Overrides source.repo for Go module/import paths.",
		},
		"site_url": {
			Name:        "site_url",
			Type:        TypeString,
			Description: "Custom domain for the catalog site (e.g. apis.internal.infoblox.dev). Defaults to {org}.github.io/{repo}.",
		},
		"catalog_url": {
			Name:        "catalog_url",
			Type:        TypeString,
			Description: "Remote catalog URL for dependency discovery. Used by apx search and apx show when --catalog is not specified.",
		},
		"catalog_registries": {
			Name:        "catalog_registries",
			Type:        TypeList,
			Description: "OCI catalog registries for API discovery. Each entry maps to ghcr.io/<org>/<repo>-catalog:latest.",
			ItemDef: &FieldDef{
				Name:        "catalog_registry",
				Type:        TypeStruct,
				Description: "A GHCR-hosted catalog registry reference",
				Children: map[string]FieldDef{
					"org":  {Name: "org", Type: TypeString, Required: true, Description: "GitHub organization"},
					"repo": {Name: "repo", Type: TypeString, Required: true, Description: "Canonical API repository name"},
				},
			},
		},
		"module_roots": {
			Name:        "module_roots",
			Type:        TypeList,
			Description: "Directories containing schema modules",
			Default:     []string{"proto"},
			ItemDef: &FieldDef{
				Name: "module_root",
				Type: TypeString,
			},
		},
		"language_targets": {
			Name:        "language_targets",
			Type:        TypeMap,
			Description: "Code generation targets keyed by language",
			ItemDef:     &languageTargetValue,
		},
		"policy": {
			Name:        "policy",
			Type:        TypeStruct,
			Description: "Validation policy settings",
			Children: map[string]FieldDef{
				"forbidden_proto_options": {
					Name:        "forbidden_proto_options",
					Type:        TypeList,
					Description: "Regex patterns for forbidden proto options",
					ItemDef:     &FieldDef{Name: "pattern", Type: TypeString},
				},
				"allowed_proto_plugins": {
					Name:        "allowed_proto_plugins",
					Type:        TypeList,
					Description: "Allowed protoc plugin names",
					ItemDef:     &FieldDef{Name: "plugin", Type: TypeString},
				},
				"openapi": {
					Name:        "openapi",
					Type:        TypeStruct,
					Description: "OpenAPI-specific policy",
					Children: map[string]FieldDef{
						"spectral_ruleset": {
							Name:        "spectral_ruleset",
							Type:        TypeString,
							Description: "Path to Spectral ruleset file",
						},
					},
				},
				"avro": {
					Name:        "avro",
					Type:        TypeStruct,
					Description: "Avro-specific policy",
					Children: map[string]FieldDef{
						"compatibility": {
							Name:        "compatibility",
							Type:        TypeString,
							Description: "Avro compatibility mode",
							Default:     "BACKWARD",
							EnumValues:  []string{"BACKWARD", "FORWARD", "FULL", "NONE"},
						},
					},
				},
				"jsonschema": {
					Name:        "jsonschema",
					Type:        TypeStruct,
					Description: "JSON Schema policy",
					Children: map[string]FieldDef{
						"breaking_mode": {
							Name:        "breaking_mode",
							Type:        TypeString,
							Description: "Breaking change detection mode",
							Default:     "strict",
							EnumValues:  []string{"strict", "lenient"},
						},
					},
				},
				"parquet": {
					Name:        "parquet",
					Type:        TypeStruct,
					Description: "Parquet policy",
					Children: map[string]FieldDef{
						"allow_additive_nullable_only": {
							Name:        "allow_additive_nullable_only",
							Type:        TypeBool,
							Description: "Whether to restrict to additive nullable columns",
							Default:     true,
						},
					},
				},
			},
		},
		"release": {
			Name:        "release",
			Type:        TypeStruct,
			Description: "Release configuration",
			Children: map[string]FieldDef{
				"tag_format": {
					Name:        "tag_format",
					Type:        TypeString,
					Description: "Tag pattern; must contain {version}",
					Default:     "{subdir}/v{version}",
					Pattern:     "must contain {version}",
				},
				"ci_only": {
					Name:        "ci_only",
					Type:        TypeBool,
					Description: "Restrict releasing to CI environments",
					Default:     true,
				},
			},
		},
		"tools": {
			Name:        "tools",
			Type:        TypeStruct,
			Description: "Pinned tool versions",
			Children: map[string]FieldDef{
				"buf": {
					Name:        "buf",
					Type:        TypeStruct,
					Description: "Buf CLI settings",
					Children: map[string]FieldDef{
						"version": {Name: "version", Type: TypeString, Description: "Buf CLI version"},
					},
				},
				"oasdiff": {
					Name:        "oasdiff",
					Type:        TypeStruct,
					Description: "oasdiff settings",
					Children: map[string]FieldDef{
						"version": {Name: "version", Type: TypeString, Description: "oasdiff version"},
					},
				},
				"spectral": {
					Name:        "spectral",
					Type:        TypeStruct,
					Description: "Spectral settings",
					Children: map[string]FieldDef{
						"version": {Name: "version", Type: TypeString, Description: "Spectral version"},
					},
				},
				"avrotool": {
					Name:        "avrotool",
					Type:        TypeStruct,
					Description: "Avro tools settings",
					Children: map[string]FieldDef{
						"version": {Name: "version", Type: TypeString, Description: "Avro tools version"},
					},
				},
				"jsonschemadiff": {
					Name:        "jsonschemadiff",
					Type:        TypeStruct,
					Description: "JSON Schema diff settings",
					Children: map[string]FieldDef{
						"version": {Name: "version", Type: TypeString, Description: "JSON Schema diff version"},
					},
				},
			},
		},
		"execution": {
			Name:        "execution",
			Type:        TypeStruct,
			Description: "Execution environment settings",
			Children: map[string]FieldDef{
				"mode": {
					Name:        "mode",
					Type:        TypeString,
					Description: "Where tools run",
					Default:     "local",
					EnumValues:  []string{"local", "container"},
				},
				"container_image": {
					Name:        "container_image",
					Type:        TypeString,
					Description: "Container image when mode=container",
				},
			},
		},
		"api": {
			Name:        "api",
			Type:        TypeStruct,
			Description: "Canonical API identity",
			Children: map[string]FieldDef{
				"id": {
					Name:        "id",
					Type:        TypeString,
					Description: "Full API identifier (format/domain/name/line)",
					Pattern:     "<format>/<domain>/<name>/<line>",
				},
				"format": {
					Name:        "format",
					Type:        TypeString,
					Description: "Schema format",
					EnumValues:  []string{"proto", "openapi", "avro", "jsonschema", "parquet"},
				},
				"domain": {
					Name:        "domain",
					Type:        TypeString,
					Description: "Business domain for the API",
				},
				"name": {
					Name:        "name",
					Type:        TypeString,
					Description: "API name within the domain",
				},
				"line": {
					Name:        "line",
					Type:        TypeString,
					Description: "API compatibility line (e.g. v1, v2)",
					Pattern:     "v<major>",
				},
				"lifecycle": {
					Name:        "lifecycle",
					Type:        TypeString,
					Description: "Maturity/support state of this API line",
					EnumValues:  []string{"experimental", "preview", "beta", "stable", "deprecated", "sunset"},
				},
			},
		},
		"source": {
			Name:        "source",
			Type:        TypeStruct,
			Description: "Canonical source repository identity",
			Children: map[string]FieldDef{
				"repo": {
					Name:        "repo",
					Type:        TypeString,
					Description: "Canonical source repository (e.g. github.com/acme/apis)",
				},
				"path": {
					Name:        "path",
					Type:        TypeString,
					Description: "Path within the canonical repo (derived from api.id)",
				},
			},
		},
		"releases": {
			Name:        "releases",
			Type:        TypeStruct,
			Description: "Release version tracking",
			Children: map[string]FieldDef{
				"current": {
					Name:        "current",
					Type:        TypeString,
					Description: "Current release version (SemVer)",
					Pattern:     "v<major>.<minor>.<patch>[-prerelease]",
				},
			},
		},
		"languages": {
			Name:        "languages",
			Type:        TypeMap,
			Description: "Derived language-specific coordinates keyed by language",
			ItemDef: &FieldDef{
				Name:        "language_coords",
				Type:        TypeStruct,
				Description: "Language-specific module and import paths",
				Children: map[string]FieldDef{
					"module": {
						Name:        "module",
						Type:        TypeString,
						Description: "Module/package path for the language",
					},
					"import": {
						Name:        "import",
						Type:        TypeString,
						Description: "Import path for the language",
					},
				},
			},
		},
		"external_apis": {
			Name:        "external_apis",
			Type:        TypeList,
			Description: "Registered external API sources",
			ItemDef: &FieldDef{
				Name:        "external_registration",
				Type:        TypeStruct,
				Description: "An external API registration entry",
				Children: map[string]FieldDef{
					"id":            {Name: "id", Type: TypeString, Required: true, Description: "Canonical API ID (format/domain/name/line)"},
					"managed_repo":  {Name: "managed_repo", Type: TypeString, Required: true, Description: "Internal repo hosting curated snapshots"},
					"managed_path":  {Name: "managed_path", Type: TypeString, Required: true, Description: "Filesystem path in managed repo"},
					"upstream_repo": {Name: "upstream_repo", Type: TypeString, Required: true, Description: "Original external repository URL"},
					"upstream_path": {Name: "upstream_path", Type: TypeString, Required: true, Description: "Path in upstream repository"},
					"import_mode":   {Name: "import_mode", Type: TypeString, Description: "Import path handling", EnumValues: []string{"preserve", "rewrite"}},
					"origin":        {Name: "origin", Type: TypeString, Description: "API classification", EnumValues: []string{"external", "forked"}},
					"description":   {Name: "description", Type: TypeString, Description: "Human-readable description"},
					"lifecycle":     {Name: "lifecycle", Type: TypeString, Description: "Lifecycle state", EnumValues: []string{"experimental", "preview", "beta", "stable", "deprecated", "sunset"}},
					"version":       {Name: "version", Type: TypeString, Description: "Current managed snapshot version"},
					"owners":        {Name: "owners", Type: TypeList, Description: "Team or individual owners", ItemDef: &FieldDef{Name: "owner", Type: TypeString}},
					"tags":          {Name: "tags", Type: TypeList, Description: "Searchable tags", ItemDef: &FieldDef{Name: "tag", Type: TypeString}},
				},
			},
		},
	}

	return SchemaVersion{
		Version:        1,
		Fields:         fields,
		RequiredFields: []string{"version", "org", "repo"},
		Deprecated:     nil,
	}
}

// DefaultConfig returns a valid Config populated with version 1 defaults.
// All init paths should use this as the single source of truth.
func DefaultConfig() *Config {
	return &Config{
		Version: CurrentSchemaVersion,
		Org:     "your-org-name",
		Repo:    "apis",
		ModuleRoots: []string{
			"proto",
			"openapi",
			"avro",
			"jsonschema",
			"parquet",
		},
		LanguageTargets: map[string]LanguageTarget{
			"go": {
				Enabled: true,
				Plugins: []map[string]string{
					{"name": "protoc-gen-go", "version": "v1.64.0"},
					{"name": "protoc-gen-go-grpc", "version": "v1.5.0"},
				},
			},
		},
		Policy: Policy{
			ForbiddenProtoOptions: []string{`^gorm\.`},
			AllowedProtoPlugins:   []string{"protoc-gen-go", "protoc-gen-go-grpc"},
			OpenAPI: struct {
				SpectralRuleset string `yaml:"spectral_ruleset,omitempty"`
			}{SpectralRuleset: ".spectral.yaml"},
			Avro: struct {
				Compatibility string `yaml:"compatibility,omitempty"`
			}{Compatibility: "BACKWARD"},
			JSONSchema: struct {
				BreakingMode string `yaml:"breaking_mode,omitempty"`
			}{BreakingMode: "strict"},
			Parquet: struct {
				AllowAdditiveNullableOnly bool `yaml:"allow_additive_nullable_only,omitempty"`
			}{AllowAdditiveNullableOnly: true},
		},
		Release: ReleaseConfig{
			TagFormat: "{subdir}/v{version}",
			CIOnly:    true,
		},
		Tools: Tools{
			Buf: struct {
				Version string `yaml:"version"`
			}{Version: "v1.45.0"},
			OASDiff: struct {
				Version string `yaml:"version"`
			}{Version: "v1.9.6"},
			Spectral: struct {
				Version string `yaml:"version"`
			}{Version: "v6.11.0"},
			AvroTool: struct {
				Version string `yaml:"version"`
			}{Version: "1.11.3"},
			JSONSchemaDiff: struct {
				Version string `yaml:"version"`
			}{Version: "0.3.0"},
		},
		Execution: Execution{
			Mode:           "local",
			ContainerImage: "",
		},
	}
}

// GenerateSchemaDoc walks the FieldDef tree for the current schema version
// and returns a Markdown reference table documenting every field.
func GenerateSchemaDoc() string {
	schema, ok := Registry.Versions[CurrentSchemaVersion]
	if !ok {
		return "No schema found.\n"
	}

	var sb strings.Builder
	sb.WriteString("# Configuration Reference (`apx.yaml`)\n\n")
	sb.WriteString(fmt.Sprintf("Schema version: **%d**\n\n", schema.Version))
	sb.WriteString("| YAML Path | Type | Required | Default | Allowed Values | Description |\n")
	sb.WriteString("|-----------|------|----------|---------|----------------|-------------|\n")

	writeFieldRows(&sb, "", schema.Fields)

	return sb.String()
}

// writeFieldRows recursively writes Markdown table rows for each field.
func writeFieldRows(sb *strings.Builder, prefix string, fields map[string]FieldDef) {
	// Sort keys for deterministic output
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		fd := fields[key]
		path := key
		if prefix != "" {
			path = prefix + "." + key
		}

		req := "no"
		if fd.Required {
			req = "yes"
		}

		def := ""
		if fd.Default != nil {
			def = fmt.Sprintf("`%v`", fd.Default)
		}

		enum := ""
		if len(fd.EnumValues) > 0 {
			enum = strings.Join(fd.EnumValues, ", ")
		}

		desc := fd.Description
		if fd.DeprecatedSince > 0 {
			desc += fmt.Sprintf(" *(deprecated since v%d; use `%s`)*", fd.DeprecatedSince, fd.Replacement)
		}

		sb.WriteString(fmt.Sprintf("| `%s` | %s | %s | %s | %s | %s |\n",
			path, fd.Type, req, def, enum, desc))

		// Recurse into children for struct types
		if fd.Type == TypeStruct && len(fd.Children) > 0 {
			writeFieldRows(sb, path, fd.Children)
		}

		// For map types, document the value schema under <key> placeholder
		if fd.Type == TypeMap && fd.ItemDef != nil && fd.ItemDef.Type == TypeStruct {
			mapPrefix := path + ".<key>"
			sb.WriteString(fmt.Sprintf("| `%s` | %s | %s | %s | %s | %s |\n",
				mapPrefix, fd.ItemDef.Type, "", "", "", fd.ItemDef.Description))
			if len(fd.ItemDef.Children) > 0 {
				writeFieldRows(sb, mapPrefix, fd.ItemDef.Children)
			}
		}
	}
}
