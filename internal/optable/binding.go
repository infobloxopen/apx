package optable

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/infobloxopen/apx/internal/pathlint"
	"gopkg.in/yaml.v3"
)

// The identity of the document that carries the route binding: an AppContract.
// The route binding is the optional routes object of each entry of
// spec.provides.apis.
const (
	BindingKind    = "AppContract"
	BindingGroup   = "appkit.infoblox.dev"
	BindingVersion = "v1alpha1"
	// BindingAPIVersion is the preferred apiVersion. BindingAPIVersions lists
	// each accepted value.
	BindingAPIVersion = BindingGroup + "/" + BindingVersion
	// ReplacedBindingKind is the kind that carried the binding before. A
	// document of this kind is an error.
	ReplacedBindingKind = "RouteAPIBinding"
	// RefScheme starts the ref of an entry: apx://<api-id>@<version>.
	RefScheme = "apx://"
	// MaxTargetRefs is the largest number of targetRefs of one entry.
	MaxTargetRefs = 16
)

// bindingGroups are the API groups of an AppContract that this package reads.
var bindingGroups = []string{BindingGroup, "appkit.platform.infoblox.com"}

// BindingAPIVersions returns each accepted apiVersion of an AppContract.
func BindingAPIVersions() []string {
	out := make([]string, len(bindingGroups))
	for i, g := range bindingGroups {
		out[i] = g + "/" + BindingVersion
	}
	return out
}

func validBindingAPIVersion(v string) bool {
	for _, ok := range BindingAPIVersions() {
		if v == ok {
			return true
		}
	}
	return false
}

// The route kind that a targetRef can name, and its API group.
const (
	TargetKindIngress  = "Ingress"
	TargetGroupIngress = "networking.k8s.io"
)

const replacedKindMessage = "RouteAPIBinding is replaced by the routes object of an AppContract (spec.provides.apis[].routes)"

// Binding is the route binding of one AppContract. It links the routes of one
// namespace to the APIs that they serve. Spec.APIs holds only the entries of
// spec.provides.apis that have a routes object.
type Binding struct {
	APIVersion string          `yaml:"apiVersion" json:"apiVersion"`
	Kind       string          `yaml:"kind" json:"kind"`
	Metadata   BindingMetadata `yaml:"metadata" json:"metadata"`
	Spec       BindingSpec     `yaml:"spec" json:"spec"`
}

// BindingMetadata is the part of the object metadata that this package reads.
type BindingMetadata struct {
	Name      string `yaml:"name" json:"name"`
	Namespace string `yaml:"namespace,omitempty" json:"namespace,omitempty"`
}

// BindingSpec lists the APIs of the binding.
type BindingSpec struct {
	APIs []BindingAPI `yaml:"apis" json:"apis"`
}

// BindingAPI pins one release of one API and names the routes that serve it.
// It is one entry of spec.provides.apis that has a routes object.
type BindingAPI struct {
	// Ref is the API id, for example "openapi/payments/ledger/v1". It is the
	// part of the entry ref between "apx://" and "@".
	Ref string `yaml:"ref" json:"ref"`
	// Version is the pinned release, for example "v1.0.0". It is the part of
	// the entry ref after "@".
	Version string `yaml:"version" json:"version"`
	// ExternalPrefix is optional. When it is set, it replaces the base path of
	// the API contract. Use "/" when the client path has no prefix.
	ExternalPrefix string `yaml:"externalPrefix,omitempty" json:"externalPrefix,omitempty"`
	// Audience is optional: "public" or "internal". When it is set, it is the
	// audience of the API.
	Audience string `yaml:"audience,omitempty" json:"audience,omitempty"`
	// TargetRefs are the routes of this entry. They are in the namespace of
	// the contract.
	TargetRefs []TargetRef `yaml:"targetRefs" json:"targetRefs"`

	// at is the path of the entry in the contract, for example
	// "spec.provides.apis[2]". Entries without routes are skipped, so the
	// position in BindingSpec.APIs can be a different number.
	at string
}

// TargetRef names one route in the namespace of the contract. It has no
// namespace field.
type TargetRef struct {
	Group string `yaml:"group" json:"group"`
	Kind  string `yaml:"kind" json:"kind"`
	Name  string `yaml:"name" json:"name"`
}

// rawContract is the decode target of an AppContract. The decode is not
// strict: only the routes object of an entry rejects an unknown field.
type rawContract struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name      string `yaml:"name"`
		Namespace string `yaml:"namespace"`
	} `yaml:"metadata"`
	Spec struct {
		Provides struct {
			APIs []struct {
				Ref    yaml.Node `yaml:"ref"`
				Routes yaml.Node `yaml:"routes"`
			} `yaml:"apis"`
		} `yaml:"provides"`
	} `yaml:"spec"`
}

// rawRoutes is the strict decode target of a routes object.
type rawRoutes struct {
	TargetRefs []struct {
		Group     string  `yaml:"group"`
		Kind      string  `yaml:"kind"`
		Name      string  `yaml:"name"`
		Namespace *string `yaml:"namespace"`
	} `yaml:"targetRefs"`
	ExternalPrefix string `yaml:"externalPrefix"`
	Audience       string `yaml:"audience"`
	// These fields are for other readers. This package reads and ignores them.
	Profile           yaml.Node `yaml:"profile"`
	OperationProfiles yaml.Node `yaml:"operationProfiles"`
	TableRef          yaml.Node `yaml:"tableRef"`
}

// absent reports that a field is not in the document or is null.
func absent(n *yaml.Node) bool {
	return n.Kind == 0 || (n.Kind == yaml.ScalarNode && n.Tag == "!!null")
}

// ParseBinding decodes and validates the route binding of one AppContract
// document. A document of another kind or apiVersion is an error, and so is a
// contract where no entry has routes.
func ParseBinding(data []byte) (*Binding, error) {
	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		return nil, fmt.Errorf("parsing binding: %w", err)
	}
	if node.Kind == 0 {
		return nil, errors.New("empty binding document")
	}
	b, err := decodeBinding(&node)
	if err != nil {
		return nil, err
	}
	if b == nil {
		return nil, fmt.Errorf("%s: no entry of spec.provides.apis has routes", contractName(&node))
	}
	if _, err := RouteAudiences([]*Binding{b}); err != nil {
		return nil, err
	}
	return b, nil
}

// contractName names a document in an error: its kind and metadata.name.
func contractName(node *yaml.Node) string {
	var head struct {
		Kind     string `yaml:"kind"`
		Metadata struct {
			Name string `yaml:"name"`
		} `yaml:"metadata"`
	}
	_ = node.Decode(&head)
	return fmt.Sprintf("%s %q", head.Kind, head.Metadata.Name)
}

// decodeBinding reads one document. It returns nil without an error for an
// AppContract where no entry has routes.
func decodeBinding(node *yaml.Node) (*Binding, error) {
	var head struct {
		APIVersion string `yaml:"apiVersion"`
		Kind       string `yaml:"kind"`
	}
	if err := node.Decode(&head); err != nil {
		return nil, fmt.Errorf("parsing binding: %w", err)
	}
	if head.Kind == ReplacedBindingKind {
		return nil, fmt.Errorf("%s: %s", contractName(node), replacedKindMessage)
	}
	if head.Kind != BindingKind {
		return nil, fmt.Errorf("unsupported kind %q (expected %s)", head.Kind, BindingKind)
	}
	name := contractName(node)
	if !validBindingAPIVersion(head.APIVersion) {
		return nil, fmt.Errorf("%s: unsupported apiVersion %q (expected %s)", name, head.APIVersion, strings.Join(BindingAPIVersions(), " or "))
	}
	var raw rawContract
	if err := node.Decode(&raw); err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	b := &Binding{APIVersion: raw.APIVersion, Kind: raw.Kind}
	b.Metadata.Name = raw.Metadata.Name
	b.Metadata.Namespace = raw.Metadata.Namespace

	for i := range raw.Spec.Provides.APIs {
		api := &raw.Spec.Provides.APIs[i]
		if absent(&api.Routes) {
			continue // an entry without routes has no binding
		}
		at := fmt.Sprintf("spec.provides.apis[%d]", i)
		entry := BindingAPI{at: at}
		var err error
		if entry.Ref, entry.Version, err = parseRef(&api.Ref); err != nil {
			return nil, fmt.Errorf("%s: %s: %w", name, at, err)
		}

		// The routes object is strict: an unknown field is an error. yaml.Node
		// has no strict decode, so the object is encoded again and read with
		// KnownFields.
		routesBytes, err := yaml.Marshal(&api.Routes)
		if err != nil {
			return nil, fmt.Errorf("%s: %s.routes: %w", name, at, err)
		}
		var routes rawRoutes
		dec := yaml.NewDecoder(bytes.NewReader(routesBytes))
		dec.KnownFields(true)
		if err := dec.Decode(&routes); err != nil && err != io.EOF {
			if strings.Contains(err.Error(), "field namespace not found") {
				return nil, fmt.Errorf("%s: %s.routes: a namespace field is not permitted; the contract and its routes are in one namespace", name, at)
			}
			if fields := unknownFields(err); len(fields) > 0 {
				return nil, fmt.Errorf("%s: %s.routes: unknown field %s", name, at, strings.Join(fields, ", "))
			}
			return nil, fmt.Errorf("%s: %s.routes: %w", name, at, err)
		}
		entry.ExternalPrefix = routes.ExternalPrefix
		entry.Audience = routes.Audience
		for j, ref := range routes.TargetRefs {
			if ref.Namespace != nil {
				return nil, fmt.Errorf("%s: %s.routes.targetRefs[%d]: a namespace field is not permitted; the contract and its routes are in one namespace",
					name, at, j)
			}
			entry.TargetRefs = append(entry.TargetRefs, TargetRef{Group: ref.Group, Kind: ref.Kind, Name: ref.Name})
		}
		b.Spec.APIs = append(b.Spec.APIs, entry)
	}
	if len(b.Spec.APIs) == 0 {
		return nil, nil
	}
	if err := b.Validate(); err != nil {
		return nil, err
	}
	return b, nil
}

// unknownFieldRE finds the field name in a strict decode error of yaml.v3.
var unknownFieldRE = regexp.MustCompile(`field (\S+) not found in type`)

// unknownFields returns the quoted names of the fields that a strict decode
// rejected. The line numbers of that error are not those of the input, so the
// caller does not print the error itself.
func unknownFields(err error) []string {
	var out []string
	for _, m := range unknownFieldRE.FindAllStringSubmatch(err.Error(), -1) {
		out = append(out, strconv.Quote(m[1]))
	}
	return out
}

// parseRef splits the ref of an entry that has routes:
// apx://<api-id>@<version>.
func parseRef(node *yaml.Node) (apiID, version string, err error) {
	if absent(node) {
		return "", "", errors.New("ref is required")
	}
	if node.Kind != yaml.ScalarNode {
		return "", "", errors.New("ref must be a string")
	}
	ref := node.Value
	if ref == "" {
		return "", "", errors.New("ref is required")
	}
	rest, ok := strings.CutPrefix(ref, RefScheme)
	if !ok {
		return "", "", fmt.Errorf("ref %q: routes needs a ref with the %s scheme (%s<api-id>@<version>)", ref, RefScheme, RefScheme)
	}
	at := strings.LastIndex(rest, "@")
	if at < 0 || at == len(rest)-1 {
		return "", "", fmt.Errorf("ref %q: routes needs a pinned ref (%s<api-id>@<version>)", ref, RefScheme)
	}
	if at == 0 {
		return "", "", fmt.Errorf("ref %q has no API id (%s<api-id>@<version>)", ref, RefScheme)
	}
	return rest[:at], rest[at+1:], nil
}

// entryPath is the path of entry i in an error.
func (b *Binding) entryPath(i int) string {
	if at := b.Spec.APIs[i].at; at != "" {
		return at
	}
	return fmt.Sprintf("spec.provides.apis[%d]", i)
}

// Validate checks the binding. It returns every problem in one error.
func (b *Binding) Validate() error {
	var errs []error
	fail := func(format string, args ...any) {
		errs = append(errs, fmt.Errorf("%s %q: "+format, append([]any{BindingKind, b.Metadata.Name}, args...)...))
	}
	if b.Kind != BindingKind {
		fail("unsupported kind %q (expected %s)", b.Kind, BindingKind)
	}
	if !validBindingAPIVersion(b.APIVersion) {
		fail("unsupported apiVersion %q (expected %s)", b.APIVersion, strings.Join(BindingAPIVersions(), " or "))
	}
	if b.Metadata.Name == "" {
		fail("metadata.name is required")
	}
	if len(b.Spec.APIs) == 0 {
		fail("no entry of spec.provides.apis has routes")
	}
	versions := map[string]string{}
	for i, api := range b.Spec.APIs {
		at := b.entryPath(i)
		if api.Ref == "" {
			fail("%s.ref is required", at)
		}
		if api.Version == "" {
			fail("%s.ref: routes needs a pinned ref (%s<api-id>@<version>)", at, RefScheme)
		}
		if prev, ok := versions[api.Ref]; ok && prev != api.Version && api.Ref != "" {
			fail("%s: API %s is pinned at two versions (%s and %s)", at, api.Ref, prev, api.Version)
		}
		versions[api.Ref] = api.Version
		if p := api.ExternalPrefix; p != "" && !strings.HasPrefix(p, "/") {
			fail("%s.routes.externalPrefix %q must start with \"/\"", at, p)
		}
		if a := api.Audience; a != "" && a != AudiencePublic && a != AudienceInternal {
			fail("%s.routes.audience %q is not valid (expected %s or %s)", at, a, AudiencePublic, AudienceInternal)
		}
		if len(api.TargetRefs) == 0 {
			fail("%s.routes.targetRefs is empty", at)
		}
		if n := len(api.TargetRefs); n > MaxTargetRefs {
			fail("%s.routes.targetRefs has %d entries (the maximum is %d)", at, n, MaxTargetRefs)
		}
		seen := map[TargetRef]bool{}
		for j, ref := range api.TargetRefs {
			rat := fmt.Sprintf("%s.routes.targetRefs[%d]", at, j)
			if ref.Kind != TargetKindIngress {
				fail("%s.kind %q is not supported (expected %s)", rat, ref.Kind, TargetKindIngress)
			} else if ref.Group != TargetGroupIngress {
				fail("%s.group %q is not valid for kind %s (expected %s)", rat, ref.Group, TargetKindIngress, TargetGroupIngress)
			}
			if ref.Name == "" {
				fail("%s.name is required", rat)
			}
			if seen[ref] {
				fail("%s repeats %s %q", rat, ref.Kind, ref.Name)
			}
			seen[ref] = true
		}
	}
	return errors.Join(errs...)
}

// RouteAudiences returns the audience that the routes objects give to each
// API: API id to "public" or "internal". An API without routes.audience is not
// in the result. Two entries of one API with different audiences are an error.
func RouteAudiences(bindings []*Binding) (map[string]string, error) {
	out := map[string]string{}
	var errs []error
	for _, b := range bindings {
		for i, api := range b.Spec.APIs {
			if api.Audience == "" {
				continue
			}
			prev, ok := out[api.Ref]
			if ok && prev != api.Audience {
				errs = append(errs, fmt.Errorf("%s %q: %s.routes.audience: API %s has two audiences (%s and %s)",
					BindingKind, b.Metadata.Name, b.entryPath(i), api.Ref, prev, api.Audience))
				continue
			}
			out[api.Ref] = api.Audience
		}
	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return out, nil
}

// ParseBindings reads the route binding of each AppContract of a manifest
// stream. A document of another kind is skipped, so the rendered output of a
// chart is a valid input. An AppContract where no entry has routes gives no
// binding. A RouteAPIBinding document is an error. source names the input in
// an error.
func ParseBindings(data []byte, source string) ([]*Binding, error) {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	var out []*Binding
	var kinds []string
	for {
		var node yaml.Node
		err := dec.Decode(&node)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("%s: %w", source, err)
		}
		var head struct {
			Kind string `yaml:"kind"`
		}
		if err := node.Decode(&head); err != nil {
			continue // not an object: a list or a scalar document
		}
		if head.Kind != "" {
			kinds = append(kinds, head.Kind)
		}
		if head.Kind != BindingKind && head.Kind != ReplacedBindingKind {
			continue
		}
		b, err := decodeBinding(&node)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", source, err)
		}
		if b != nil {
			out = append(out, b)
		}
	}
	if len(out) == 0 {
		if len(kinds) > 0 {
			return nil, fmt.Errorf("%s: no %s with a routes object found (kinds in the input: %s)", source, BindingKind, strings.Join(kinds, ", "))
		}
		return nil, fmt.Errorf("%s: no %s with a routes object found", source, BindingKind)
	}
	if _, err := RouteAudiences(out); err != nil {
		return nil, fmt.Errorf("%s: %w", source, err)
	}
	return out, nil
}

// LoadBindings reads the route bindings of the AppContract documents of a YAML
// file or of a Helm chart directory.
// A chart directory is rendered with "helm template"; a file needs no helm
// binary.
func LoadBindings(path, release string, helmSets []string) ([]*Binding, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("binding %s: %w", path, err)
	}
	if !info.IsDir() {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("binding %s: %w", path, err)
		}
		return ParseBindings(data, path)
	}
	rendered, err := pathlint.RenderChart(path, release, helmSets)
	if err != nil {
		return nil, fmt.Errorf("helm template %s: %w", path, err)
	}
	return ParseBindings([]byte(rendered), "helm template "+path)
}
