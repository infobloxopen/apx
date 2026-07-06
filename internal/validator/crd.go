package validator

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// CRDValidator lints and breaking-checks Kubernetes CustomResourceDefinitions.
//
// A CRD is an OpenAPI v3 structural schema wrapped in a Kubernetes envelope
// (its spec.versions[].schema.openAPIV3Schema). This validator implements the
// Kubernetes-specific rules that raw OpenAPI tooling (oasdiff/spectral) does
// not know about: structural-schema constraints for lint, and served-version
// compatibility for breaking analysis. Like the Avro and JSON Schema
// validators it is pure Go (yaml.v3 only) — no external toolchain and no
// Kubernetes apiextensions-apiserver dependency.
//
// Scope: the declared contract (schema, GVK, served/storage versions,
// deprecation). Runtime behavior — conversion webhooks, admission control,
// defaulting — is out of scope.
type CRDValidator struct {
	resolver *ToolchainResolver
}

// NewCRDValidator creates a new CRD validator.
func NewCRDValidator(resolver *ToolchainResolver) *CRDValidator {
	return &CRDValidator{resolver: resolver}
}

// ---------------------------------------------------------------------------
// CRD document model (subset of apiextensions.k8s.io/v1)
// ---------------------------------------------------------------------------

type crdDoc struct {
	APIVersion string  `yaml:"apiVersion"`
	Kind       string  `yaml:"kind"`
	Metadata   crdMeta `yaml:"metadata"`
	Spec       crdSpec `yaml:"spec"`
}

type crdMeta struct {
	Name string `yaml:"name"`
}

type crdSpec struct {
	Group    string       `yaml:"group"`
	Names    crdNames     `yaml:"names"`
	Scope    string       `yaml:"scope"`
	Versions []crdVersion `yaml:"versions"`
}

type crdNames struct {
	Kind       string   `yaml:"kind"`
	ListKind   string   `yaml:"listKind"`
	Plural     string   `yaml:"plural"`
	Singular   string   `yaml:"singular"`
	ShortNames []string `yaml:"shortNames"`
}

type crdVersion struct {
	Name       string     `yaml:"name"`
	Served     *bool      `yaml:"served"`
	Storage    *bool      `yaml:"storage"`
	Deprecated bool       `yaml:"deprecated"`
	Schema     crdVSchema `yaml:"schema"`
}

func (v crdVersion) isServed() bool  { return v.Served != nil && *v.Served }
func (v crdVersion) isStorage() bool { return v.Storage != nil && *v.Storage }

type crdVSchema struct {
	OpenAPIV3Schema *jsonSchema `yaml:"openAPIV3Schema"`
}

// jsonSchema is the subset of a structural OpenAPI v3 schema that CRD lint and
// breaking analysis inspect. Polymorphic fields (items, additionalProperties)
// use dedicated types so a CRD like `additionalProperties: false` parses.
type jsonSchema struct {
	Type                 string                 `yaml:"type"`
	Format               string                 `yaml:"format"`
	Description          string                 `yaml:"description"`
	Required             []string               `yaml:"required"`
	Properties           map[string]*jsonSchema `yaml:"properties"`
	Items                *jsonSchema            `yaml:"items"`
	AdditionalProperties *schemaOrBool          `yaml:"additionalProperties"`
	Enum                 []interface{}          `yaml:"enum"`
	Default              interface{}            `yaml:"default"`
	Nullable             bool                   `yaml:"nullable"`

	// numeric / string / array constraints
	Minimum          *float64 `yaml:"minimum"`
	Maximum          *float64 `yaml:"maximum"`
	ExclusiveMinimum bool     `yaml:"exclusiveMinimum"`
	ExclusiveMaximum bool     `yaml:"exclusiveMaximum"`
	MinLength        *int64   `yaml:"minLength"`
	MaxLength        *int64   `yaml:"maxLength"`
	Pattern          string   `yaml:"pattern"`
	MinItems         *int64   `yaml:"minItems"`
	MaxItems         *int64   `yaml:"maxItems"`

	// logical junctors (value validation, not structure)
	AllOf []*jsonSchema `yaml:"allOf"`
	AnyOf []*jsonSchema `yaml:"anyOf"`
	OneOf []*jsonSchema `yaml:"oneOf"`
	Not   *jsonSchema   `yaml:"not"`

	// Kubernetes extensions
	XPreserveUnknownFields *bool         `yaml:"x-kubernetes-preserve-unknown-fields"`
	XIntOrString           bool          `yaml:"x-kubernetes-int-or-string"`
	XEmbeddedResource      bool          `yaml:"x-kubernetes-embedded-resource"`
	XListType              string        `yaml:"x-kubernetes-list-type"`
	XListMapKeys           []string      `yaml:"x-kubernetes-list-map-keys"`
	XValidations           []xValidation `yaml:"x-kubernetes-validations"`
}

type xValidation struct {
	Rule    string `yaml:"rule"`
	Message string `yaml:"message"`
}

func (s *jsonSchema) preservesUnknown() bool {
	return s.XPreserveUnknownFields != nil && *s.XPreserveUnknownFields
}

// schemaOrBool models the additionalProperties field, which may be a boolean
// or a nested schema.
type schemaOrBool struct {
	Bool   *bool
	Schema *jsonSchema
}

func (sb *schemaOrBool) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind == yaml.ScalarNode {
		var b bool
		if err := node.Decode(&b); err == nil {
			sb.Bool = &b
			return nil
		}
	}
	var s jsonSchema
	if err := node.Decode(&s); err != nil {
		return err
	}
	sb.Schema = &s
	return nil
}

func (sb *schemaOrBool) allowsAll() bool {
	return sb != nil && sb.Bool != nil && *sb.Bool
}

// ---------------------------------------------------------------------------
// CRD loading
// ---------------------------------------------------------------------------

// crdVersionRe matches a Kubernetes API version string: v<major> with an
// optional alpha/beta maturity suffix (e.g. v1, v2, v1alpha1, v2beta3).
var crdVersionRe = regexp.MustCompile(`^v[1-9][0-9]*((alpha|beta)[1-9][0-9]*)?$`)

// IsCRDVersion reports whether s is a valid Kubernetes API version string.
func IsCRDVersion(s string) bool {
	return crdVersionRe.MatchString(s)
}

// crdVersionMaturity returns "alpha", "beta", or "ga" for a Kubernetes version
// string. Alpha versions carry no compatibility guarantee under the Kubernetes
// deprecation policy.
func crdVersionMaturity(name string) string {
	switch {
	case strings.Contains(name, "alpha"):
		return "alpha"
	case strings.Contains(name, "beta"):
		return "beta"
	default:
		return "ga"
	}
}

// loadCRD reads and parses a single CRD manifest. When path is a directory it
// loads the first *.yaml/*.yml file that is a CustomResourceDefinition.
func loadCRD(path string) (*crdDoc, string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, "", fmt.Errorf("failed to resolve path: %w", err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, "", fmt.Errorf("reading %s: %w", path, err)
	}
	if info.IsDir() {
		file, findErr := findCRDFile(abs)
		if findErr != nil {
			return nil, "", findErr
		}
		abs = file
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, "", fmt.Errorf("reading %s: %w", abs, err)
	}
	doc, err := parseCRD(data)
	if err != nil {
		return nil, abs, fmt.Errorf("%s: %w", abs, err)
	}
	return doc, abs, nil
}

func parseCRD(data []byte) (*crdDoc, error) {
	var doc crdDoc
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("invalid YAML: %w", err)
	}
	if doc.Kind != "CustomResourceDefinition" {
		return nil, fmt.Errorf("not a CustomResourceDefinition (kind=%q)", doc.Kind)
	}
	return &doc, nil
}

// findCRDFile returns the first CRD manifest in a directory tree.
func findCRDFile(dir string) (string, error) {
	var found string
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil || info.IsDir() || found != "" {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}
		if data, err := os.ReadFile(path); err == nil && LooksLikeCRD(data) {
			found = path
		}
		return nil
	})
	if found == "" {
		return "", fmt.Errorf("no CustomResourceDefinition file found in %s", dir)
	}
	return found, nil
}

// LooksLikeCRD reports whether a YAML document is a Kubernetes CRD, used by
// format detection. It checks for the apiextensions group and the CRD kind
// without a full parse so it stays cheap.
func LooksLikeCRD(data []byte) bool {
	var head struct {
		APIVersion string `yaml:"apiVersion"`
		Kind       string `yaml:"kind"`
	}
	if err := yaml.Unmarshal(data, &head); err != nil {
		return false
	}
	return head.Kind == "CustomResourceDefinition" &&
		strings.HasPrefix(head.APIVersion, "apiextensions.k8s.io/")
}

// ---------------------------------------------------------------------------
// Lint
// ---------------------------------------------------------------------------

// validCRDTypes is the set of OpenAPI types allowed in a structural schema.
var validCRDTypes = map[string]bool{
	"object": true, "array": true, "string": true,
	"integer": true, "number": true, "boolean": true,
}

// Lint validates a CRD manifest against Kubernetes structural-schema rules and
// CRD conventions. It collects all violations and returns them together.
func (v *CRDValidator) Lint(path string) error {
	doc, file, err := loadCRD(path)
	if err != nil {
		return err
	}

	var violations []string
	add := func(format string, args ...interface{}) {
		violations = append(violations, fmt.Sprintf(format, args...))
	}

	// --- Envelope ---
	if doc.APIVersion != "apiextensions.k8s.io/v1" {
		if strings.HasPrefix(doc.APIVersion, "apiextensions.k8s.io/v1beta1") {
			add("apiVersion apiextensions.k8s.io/v1beta1 is not supported; migrate the CRD to apiextensions.k8s.io/v1")
		} else {
			add("apiVersion must be apiextensions.k8s.io/v1, got %q", doc.APIVersion)
		}
	}

	spec := doc.Spec
	if spec.Group == "" {
		add("spec.group is required")
	} else if !strings.Contains(spec.Group, ".") {
		add("spec.group %q should be a DNS subdomain (e.g. example.com)", spec.Group)
	}
	if spec.Names.Kind == "" {
		add("spec.names.kind is required")
	}
	if spec.Names.Plural == "" {
		add("spec.names.plural is required")
	} else if spec.Names.Plural != strings.ToLower(spec.Names.Plural) {
		add("spec.names.plural %q must be lowercase", spec.Names.Plural)
	}
	if spec.Scope != "Namespaced" && spec.Scope != "Cluster" {
		add("spec.scope must be Namespaced or Cluster, got %q", spec.Scope)
	}
	// metadata.name must be <plural>.<group>
	if spec.Names.Plural != "" && spec.Group != "" {
		want := spec.Names.Plural + "." + spec.Group
		if doc.Metadata.Name != want {
			add("metadata.name must be %q (<plural>.<group>), got %q", want, doc.Metadata.Name)
		}
	}

	// --- Versions ---
	if len(spec.Versions) == 0 {
		add("spec.versions must declare at least one version")
	}
	storageCount := 0
	servedCount := 0
	seenNames := map[string]bool{}
	for i, ver := range spec.Versions {
		if ver.Name == "" {
			add("spec.versions[%d].name is required", i)
		} else {
			if !IsCRDVersion(ver.Name) {
				add("spec.versions[%d].name %q is not a valid Kubernetes version (want v<major>[alpha|beta<n>])", i, ver.Name)
			}
			if seenNames[ver.Name] {
				add("spec.versions[%d].name %q is duplicated", i, ver.Name)
			}
			seenNames[ver.Name] = true
		}
		if ver.isStorage() {
			storageCount++
		}
		if ver.isServed() {
			servedCount++
		}
		// Structural schema is required per version in apiextensions v1.
		if ver.Schema.OpenAPIV3Schema == nil {
			add("spec.versions[%d] (%s) is missing schema.openAPIV3Schema", i, ver.Name)
			continue
		}
		root := ver.Schema.OpenAPIV3Schema
		if root.Type != "object" {
			add("spec.versions[%d] (%s): root schema type must be object, got %q", i, ver.Name, root.Type)
		}
		ctx := &structuralCtx{version: ver.Name, add: add}
		checkStructural(root, fmt.Sprintf("versions[%d](%s).openAPIV3Schema", i, ver.Name), ctx)
	}
	if len(spec.Versions) > 0 {
		if storageCount == 0 {
			add("exactly one version must set storage: true (found none)")
		} else if storageCount > 1 {
			add("exactly one version must set storage: true (found %d)", storageCount)
		}
		if servedCount == 0 {
			add("at least one version must set served: true")
		}
	}

	if len(violations) > 0 {
		sort.Strings(violations)
		return fmt.Errorf("CRD lint errors in %s:\n  - %s", file, strings.Join(violations, "\n  - "))
	}
	return nil
}

type structuralCtx struct {
	version string
	add     func(string, ...interface{})
}

// checkStructural walks a schema node enforcing the core Kubernetes structural
// schema rules: every specified node has a type (unless it is
// x-kubernetes-preserve-unknown-fields or x-kubernetes-int-or-string), types
// are valid, arrays specify items, and properties/additionalProperties are not
// combined at the same level.
func checkStructural(s *jsonSchema, path string, ctx *structuralCtx) {
	if s == nil {
		return
	}

	// int-or-string and preserve-unknown-fields are structural escape hatches:
	// a node with either need not (and, for int-or-string, must not) carry a type.
	escape := s.XIntOrString || s.preservesUnknown()

	if s.Type == "" {
		if !escape && len(s.AllOf) == 0 && len(s.AnyOf) == 0 && len(s.OneOf) == 0 {
			ctx.add("%s: missing type (every structural schema node needs a type, or x-kubernetes-preserve-unknown-fields / x-kubernetes-int-or-string)", path)
		}
	} else if !validCRDTypes[s.Type] {
		ctx.add("%s: invalid type %q", path, s.Type)
	}

	if s.Type == "object" {
		if len(s.Properties) > 0 && s.AdditionalProperties != nil && s.AdditionalProperties.Schema != nil {
			ctx.add("%s: properties and additionalProperties (schema form) are mutually exclusive in a structural schema", path)
		}
		for name, prop := range s.Properties {
			checkStructural(prop, path+".properties."+name, ctx)
		}
		if s.AdditionalProperties != nil && s.AdditionalProperties.Schema != nil {
			checkStructural(s.AdditionalProperties.Schema, path+".additionalProperties", ctx)
		}
	}

	if s.Type == "array" {
		if s.Items == nil && !escape {
			ctx.add("%s: array must specify items", path)
		}
		checkStructural(s.Items, path+".items", ctx)
	}

	// Logical junctors are value validation; recurse for completeness but do
	// not require a type at the junctor level.
	for i, sub := range s.AllOf {
		checkStructural(sub, fmt.Sprintf("%s.allOf[%d]", path, i), ctx)
	}
	for i, sub := range s.AnyOf {
		checkStructural(sub, fmt.Sprintf("%s.anyOf[%d]", path, i), ctx)
	}
	for i, sub := range s.OneOf {
		checkStructural(sub, fmt.Sprintf("%s.oneOf[%d]", path, i), ctx)
	}
}

// ---------------------------------------------------------------------------
// Breaking analysis (Kubernetes served-version compatibility)
// ---------------------------------------------------------------------------

// Breaking checks for backward-incompatible changes between two CRD manifests.
// path is the new CRD; against is the old/baseline CRD (a file or directory).
//
// The Kubernetes served-version compatibility rules differ from raw OpenAPI
// diffing:
//   - Removing a served version is breaking.
//   - Within a served version you cannot remove a field, narrow its type, add a
//     required field, tighten a constraint, remove enum values, or drop
//     x-kubernetes-preserve-unknown-fields.
//   - Alpha versions carry NO compatibility guarantee (Kubernetes deprecation
//     policy), so changes to an alpha version are never flagged.
//   - Additive changes (new optional field, new served version, relaxed
//     constraint) are NOT breaking.
//
// Conversion-webhook and storage-migration behavior is out of scope (the
// catalog captures the declared contract, not runtime behavior).
func (v *CRDValidator) Breaking(path, against string) error {
	newDoc, _, err := loadCRD(path)
	if err != nil {
		return fmt.Errorf("reading new CRD: %w", err)
	}
	oldDoc, _, err := loadCRD(against)
	if err != nil {
		// No baseline (e.g. first release, or a git-tag path that is not a
		// file) — nothing to compare against.
		if errors.Is(err, os.ErrNotExist) || strings.Contains(err.Error(), "no CustomResourceDefinition") {
			return nil
		}
		return fmt.Errorf("reading baseline CRD: %w", err)
	}

	var breaking []string
	add := func(format string, args ...interface{}) {
		breaking = append(breaking, fmt.Sprintf(format, args...))
	}

	oldVersions := map[string]crdVersion{}
	for _, ver := range oldDoc.Spec.Versions {
		oldVersions[ver.Name] = ver
	}
	newVersions := map[string]crdVersion{}
	for _, ver := range newDoc.Spec.Versions {
		newVersions[ver.Name] = ver
	}

	// Removing a served version is breaking (unless it was an alpha version).
	for name, oldVer := range oldVersions {
		if !oldVer.isServed() {
			continue
		}
		if crdVersionMaturity(name) == "alpha" {
			continue
		}
		if _, ok := newVersions[name]; !ok {
			add("served version %s was removed", name)
			continue
		}
		if newVer := newVersions[name]; !newVer.isServed() {
			add("version %s is no longer served", name)
		}
	}

	// Per-version schema compatibility, for versions present in both.
	for name, oldVer := range oldVersions {
		newVer, ok := newVersions[name]
		if !ok {
			continue
		}
		// Alpha versions have no compatibility guarantee.
		if crdVersionMaturity(name) == "alpha" {
			continue
		}
		// Only guard versions that were served in the baseline.
		if !oldVer.isServed() {
			continue
		}
		if oldVer.Schema.OpenAPIV3Schema == nil || newVer.Schema.OpenAPIV3Schema == nil {
			continue
		}
		diffSchema(
			oldVer.Schema.OpenAPIV3Schema,
			newVer.Schema.OpenAPIV3Schema,
			fmt.Sprintf("%s", name),
			add,
		)
	}

	if len(breaking) > 0 {
		sort.Strings(breaking)
		return fmt.Errorf("breaking changes detected (served-version incompatibility):\n  - %s",
			strings.Join(breaking, "\n  - "))
	}
	return nil
}

// diffSchema compares an old and new schema node and records breaking changes.
// It walks required fields, properties, items, and per-field constraints.
func diffSchema(oldS, newS *jsonSchema, path string, add func(string, ...interface{})) {
	if oldS == nil || newS == nil {
		return
	}

	// Type narrowing: a changed type is breaking. int-or-string relaxations
	// (adding the extension) are additive and handled below.
	if oldS.Type != "" && newS.Type != "" && oldS.Type != newS.Type {
		add("%s: type changed from %q to %q", path, oldS.Type, newS.Type)
	}

	// Dropping preserve-unknown-fields tightens the contract.
	if oldS.preservesUnknown() && !newS.preservesUnknown() {
		add("%s: x-kubernetes-preserve-unknown-fields was removed (previously accepted fields may now be pruned/rejected)", path)
	}

	// New required fields are breaking.
	oldReq := stringSet(oldS.Required)
	for _, r := range newS.Required {
		if !oldReq[r] {
			add("%s.%s: became required", path, r)
		}
	}

	// Removed properties are breaking; recurse into shared ones.
	for name, oldProp := range oldS.Properties {
		newProp, ok := newS.Properties[name]
		if !ok {
			// Removal only matters if the parent did not open up to unknown fields.
			if !newS.preservesUnknown() {
				add("%s.%s: field was removed", path, name)
			}
			continue
		}
		diffSchema(oldProp, newProp, path+"."+name, add)
	}

	// Items (array element schema).
	if oldS.Items != nil && newS.Items != nil {
		diffSchema(oldS.Items, newS.Items, path+"[]", add)
	}

	// additionalProperties schema.
	if oldS.AdditionalProperties != nil && oldS.AdditionalProperties.Schema != nil &&
		newS.AdditionalProperties != nil && newS.AdditionalProperties.Schema != nil {
		diffSchema(oldS.AdditionalProperties.Schema, newS.AdditionalProperties.Schema, path+"{}", add)
	}

	diffConstraints(oldS, newS, path, add)
}

// diffConstraints records tightened validation constraints, which reject
// payloads that the old schema accepted.
func diffConstraints(oldS, newS *jsonSchema, path string, add func(string, ...interface{})) {
	// String length.
	if tightenedUpperInt(oldS.MaxLength, newS.MaxLength) {
		add("%s: maxLength tightened (%s → %s)", path, i64s(oldS.MaxLength), i64s(newS.MaxLength))
	}
	if tightenedLowerInt(oldS.MinLength, newS.MinLength) {
		add("%s: minLength tightened (%s → %s)", path, i64s(oldS.MinLength), i64s(newS.MinLength))
	}
	// Array length.
	if tightenedUpperInt(oldS.MaxItems, newS.MaxItems) {
		add("%s: maxItems tightened (%s → %s)", path, i64s(oldS.MaxItems), i64s(newS.MaxItems))
	}
	if tightenedLowerInt(oldS.MinItems, newS.MinItems) {
		add("%s: minItems tightened (%s → %s)", path, i64s(oldS.MinItems), i64s(newS.MinItems))
	}
	// Numeric bounds.
	if tightenedUpperFloat(oldS.Maximum, newS.Maximum) {
		add("%s: maximum tightened (%s → %s)", path, f64s(oldS.Maximum), f64s(newS.Maximum))
	}
	if tightenedLowerFloat(oldS.Minimum, newS.Minimum) {
		add("%s: minimum tightened (%s → %s)", path, f64s(oldS.Minimum), f64s(newS.Minimum))
	}
	// Pattern: adding or changing a regex can reject previously-valid values.
	if oldS.Pattern != newS.Pattern && newS.Pattern != "" {
		if oldS.Pattern == "" {
			add("%s: pattern %q was added", path, newS.Pattern)
		} else {
			add("%s: pattern changed (%q → %q)", path, oldS.Pattern, newS.Pattern)
		}
	}
	// Enum: adding an enum where none existed, or removing values, is breaking.
	diffEnum(oldS, newS, path, add)
}

func diffEnum(oldS, newS *jsonSchema, path string, add func(string, ...interface{})) {
	oldEnum := enumSet(oldS.Enum)
	newEnum := enumSet(newS.Enum)
	if len(oldEnum) == 0 && len(newEnum) > 0 {
		add("%s: enum constraint was added (previously any value was allowed)", path)
		return
	}
	if len(oldEnum) == 0 {
		return
	}
	for val := range oldEnum {
		if !newEnum[val] {
			add("%s: enum value %s was removed", path, val)
		}
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func stringSet(in []string) map[string]bool {
	set := make(map[string]bool, len(in))
	for _, s := range in {
		set[s] = true
	}
	return set
}

func enumSet(vals []interface{}) map[string]bool {
	set := make(map[string]bool, len(vals))
	for _, v := range vals {
		set[fmt.Sprintf("%v", v)] = true
	}
	return set
}

// tightenedUpperInt reports whether an upper bound (maxLength/maxItems) was
// lowered: present-and-smaller, or newly introduced where none existed.
func tightenedUpperInt(oldV, newV *int64) bool {
	if newV == nil {
		return false
	}
	if oldV == nil {
		return true // adding an upper bound tightens
	}
	return *newV < *oldV
}

// tightenedLowerInt reports whether a lower bound (minLength/minItems) was
// raised: present-and-larger, or newly introduced.
func tightenedLowerInt(oldV, newV *int64) bool {
	if newV == nil {
		return false
	}
	if oldV == nil {
		return true
	}
	return *newV > *oldV
}

func tightenedUpperFloat(oldV, newV *float64) bool {
	if newV == nil {
		return false
	}
	if oldV == nil {
		return true
	}
	return *newV < *oldV
}

func tightenedLowerFloat(oldV, newV *float64) bool {
	if newV == nil {
		return false
	}
	if oldV == nil {
		return true
	}
	return *newV > *oldV
}

func i64s(v *int64) string {
	if v == nil {
		return "unset"
	}
	return fmt.Sprintf("%d", *v)
}

func f64s(v *float64) string {
	if v == nil {
		return "unset"
	}
	return strings.TrimSuffix(strings.TrimRight(fmt.Sprintf("%f", *v), "0"), ".")
}

// CRDInfo carries the catalog-facing facts extracted from a CRD manifest.
type CRDInfo struct {
	Group              string
	Kind               string
	Plural             string
	Scope              string
	ServedVersions     []string
	StorageVersion     string
	DeprecatedVersions []string
}

// ExtractCRDInfo parses a CRD file (or directory) and returns its GVK and
// served/storage facts for the catalog. It is used at catalog-generation time,
// mirroring how proto resource types are indexed.
func ExtractCRDInfo(path string) (*CRDInfo, error) {
	doc, _, err := loadCRD(path)
	if err != nil {
		return nil, err
	}
	info := &CRDInfo{
		Group:  doc.Spec.Group,
		Kind:   doc.Spec.Names.Kind,
		Plural: doc.Spec.Names.Plural,
		Scope:  doc.Spec.Scope,
	}
	for _, ver := range doc.Spec.Versions {
		if ver.isServed() {
			info.ServedVersions = append(info.ServedVersions, ver.Name)
		}
		if ver.isStorage() {
			info.StorageVersion = ver.Name
		}
		if ver.Deprecated {
			info.DeprecatedVersions = append(info.DeprecatedVersions, ver.Name)
		}
	}
	return info, nil
}
