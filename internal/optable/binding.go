package optable

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/infobloxopen/apx/internal/pathlint"
	"gopkg.in/yaml.v3"
)

// The identity of a RouteAPIBinding document.
const (
	BindingAPIVersion = "traefik.infoblox.com/v1alpha1"
	BindingKind       = "RouteAPIBinding"
)

// The route kind that a targetRef can name, and its API group.
const (
	TargetKindIngress  = "Ingress"
	TargetGroupIngress = "networking.k8s.io"
)

// Binding is one RouteAPIBinding. It links the routes of one namespace to the
// APIs that they serve.
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
type BindingAPI struct {
	// Ref is the API id, for example "openapi/payments/ledger/v1".
	Ref string `yaml:"ref" json:"ref"`
	// Version is the pinned release, for example "v1.0.0".
	Version string `yaml:"version" json:"version"`
	// ExternalPrefix is optional. When it is set, it replaces the base path of
	// the API contract. Use "/" when the client path has no prefix.
	ExternalPrefix string `yaml:"externalPrefix,omitempty" json:"externalPrefix,omitempty"`
	// TargetRefs are the routes of this entry. They are in the namespace of
	// the binding.
	TargetRefs []TargetRef `yaml:"targetRefs" json:"targetRefs"`
}

// TargetRef names one route in the namespace of the binding. It has no
// namespace field.
type TargetRef struct {
	Group string `yaml:"group" json:"group"`
	Kind  string `yaml:"kind" json:"kind"`
	Name  string `yaml:"name" json:"name"`
}

// rawBinding is the decode target. Metadata and the top level accept the
// fields that a rendered manifest can carry; the spec is strict.
type rawBinding struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name      string `yaml:"name"`
		Namespace string `yaml:"namespace"`
	} `yaml:"metadata"`
	Spec yaml.Node `yaml:"spec"`
}

type rawSpec struct {
	APIs []struct {
		Ref            string `yaml:"ref"`
		Version        string `yaml:"version"`
		ExternalPrefix string `yaml:"externalPrefix"`
		TargetRefs     []struct {
			Group     string  `yaml:"group"`
			Kind      string  `yaml:"kind"`
			Name      string  `yaml:"name"`
			Namespace *string `yaml:"namespace"`
		} `yaml:"targetRefs"`
	} `yaml:"apis"`
}

// ParseBinding decodes and validates one RouteAPIBinding document. A document
// of another kind or apiVersion is an error.
func ParseBinding(data []byte) (*Binding, error) {
	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		return nil, fmt.Errorf("parsing binding: %w", err)
	}
	if node.Kind == 0 {
		return nil, errors.New("empty binding document")
	}
	return decodeBinding(&node)
}

func decodeBinding(node *yaml.Node) (*Binding, error) {
	var raw rawBinding
	if err := node.Decode(&raw); err != nil {
		return nil, fmt.Errorf("parsing binding: %w", err)
	}
	b := &Binding{APIVersion: raw.APIVersion, Kind: raw.Kind}
	b.Metadata.Name = raw.Metadata.Name
	b.Metadata.Namespace = raw.Metadata.Namespace
	if b.Kind != BindingKind {
		return nil, fmt.Errorf("unsupported kind %q (expected %s)", b.Kind, BindingKind)
	}
	if b.APIVersion != BindingAPIVersion {
		return nil, fmt.Errorf("%s %q: unsupported apiVersion %q (expected %s)", BindingKind, b.Metadata.Name, b.APIVersion, BindingAPIVersion)
	}

	// The spec is strict: an unknown field is an error. yaml.Node has no
	// strict decode, so the spec is encoded again and read with KnownFields.
	var spec rawSpec
	if raw.Spec.Kind != 0 {
		specBytes, err := yaml.Marshal(&raw.Spec)
		if err != nil {
			return nil, fmt.Errorf("%s %q: %w", BindingKind, b.Metadata.Name, err)
		}
		dec := yaml.NewDecoder(bytes.NewReader(specBytes))
		dec.KnownFields(true)
		if err := dec.Decode(&spec); err != nil && err != io.EOF {
			if strings.Contains(err.Error(), "field namespace not found") {
				return nil, fmt.Errorf("%s %q: a namespace field is not permitted in the spec; the binding and its routes are in one namespace", BindingKind, b.Metadata.Name)
			}
			return nil, fmt.Errorf("%s %q: spec: %w", BindingKind, b.Metadata.Name, err)
		}
	}
	for i, api := range spec.APIs {
		entry := BindingAPI{Ref: api.Ref, Version: api.Version, ExternalPrefix: api.ExternalPrefix}
		for j, ref := range api.TargetRefs {
			if ref.Namespace != nil {
				return nil, fmt.Errorf("%s %q: spec.apis[%d].targetRefs[%d]: a namespace field is not permitted; the binding and its routes are in one namespace",
					BindingKind, b.Metadata.Name, i, j)
			}
			entry.TargetRefs = append(entry.TargetRefs, TargetRef{Group: ref.Group, Kind: ref.Kind, Name: ref.Name})
		}
		b.Spec.APIs = append(b.Spec.APIs, entry)
	}
	if err := b.Validate(); err != nil {
		return nil, err
	}
	return b, nil
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
	if b.APIVersion != BindingAPIVersion {
		fail("unsupported apiVersion %q (expected %s)", b.APIVersion, BindingAPIVersion)
	}
	if b.Metadata.Name == "" {
		fail("metadata.name is required")
	}
	if len(b.Spec.APIs) == 0 {
		fail("spec.apis is empty")
	}
	versions := map[string]string{}
	for i, api := range b.Spec.APIs {
		at := fmt.Sprintf("spec.apis[%d]", i)
		if api.Ref == "" {
			fail("%s.ref is required", at)
		}
		if api.Version == "" {
			fail("%s.version is required", at)
		}
		if prev, ok := versions[api.Ref]; ok && prev != api.Version && api.Ref != "" {
			fail("%s: API %s is pinned at two versions (%s and %s)", at, api.Ref, prev, api.Version)
		}
		versions[api.Ref] = api.Version
		if p := api.ExternalPrefix; p != "" && !strings.HasPrefix(p, "/") {
			fail("%s.externalPrefix %q must start with \"/\"", at, p)
		}
		if len(api.TargetRefs) == 0 {
			fail("%s.targetRefs is empty", at)
		}
		seen := map[TargetRef]bool{}
		for j, ref := range api.TargetRefs {
			rat := fmt.Sprintf("%s.targetRefs[%d]", at, j)
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

// ParseBindings reads each RouteAPIBinding of a manifest stream. A document of
// another kind is skipped, so the rendered output of a chart is a valid input.
// source names the input in an error.
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
		if head.Kind != BindingKind {
			if head.Kind != "" {
				kinds = append(kinds, head.Kind)
			}
			continue
		}
		b, err := decodeBinding(&node)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", source, err)
		}
		out = append(out, b)
	}
	if len(out) == 0 {
		if len(kinds) > 0 {
			return nil, fmt.Errorf("%s: no %s found (kinds in the input: %s)", source, BindingKind, strings.Join(kinds, ", "))
		}
		return nil, fmt.Errorf("%s: no %s found", source, BindingKind)
	}
	return out, nil
}

// LoadBindings reads the bindings of a YAML file or of a Helm chart directory.
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
