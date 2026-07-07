package pathlint

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// k8sIngressDoc is a narrow view of a Kubernetes Ingress manifest: just
// enough to recover host + path + pathType. Any other kind decodes into
// an empty struct and is skipped.
type k8sIngressDoc struct {
	Kind     string `yaml:"kind"`
	Metadata struct {
		Name string `yaml:"name"`
	} `yaml:"metadata"`
	Spec struct {
		Rules []struct {
			Host string `yaml:"host"`
			HTTP struct {
				Paths []struct {
					Path     string `yaml:"path"`
					PathType string `yaml:"pathType"`
				} `yaml:"paths"`
			} `yaml:"http"`
		} `yaml:"rules"`
	} `yaml:"spec"`
}

// normalizeIngressPath turns a raw Ingress path + pathType into a Rule
// (Path/Segments/Wildcard only; Host/Source filled in by the caller).
//
// Handles three shapes seen in real charts:
//   - plain literal + pathType: Prefix|Exact  (standard k8s Ingress)
//   - plain literal + pathType: ImplementationSpecific with no regex
//     marker (assumed prefix-like; see README limits)
//   - nginx regex-capture paths like "/api/atlas-tagging/v2/?(.*)"
//     (common with nginx.ingress.kubernetes.io/rewrite-target) — the
//     "(.*)" capture is stripped and treated as an explicit wildcard.
func normalizeIngressPath(raw, pathType string) Rule {
	p := raw
	wildcard := false
	if idx := strings.Index(p, "(.*)"); idx >= 0 {
		p = p[:idx]
		wildcard = true
	}
	p = strings.TrimSuffix(p, "?")

	switch strings.ToLower(pathType) {
	case "prefix":
		wildcard = true
	case "exact":
		// keep whatever the regex-stripping above decided (normally false)
	default: // "ImplementationSpecific" or unset
		if !wildcard {
			wildcard = true // conservative default; see README limits
		}
	}

	segs := splitClean(p)
	return Rule{Path: joinSegments(segs), Segments: segs, Wildcard: wildcard}
}

// decodeIngressManifest scans a (possibly multi-document) rendered
// manifest and returns one Rule per (rule, path) pair found in every
// Ingress object. source is a label used for report provenance.
func decodeIngressManifest(text, source string) []Rule {
	dec := yaml.NewDecoder(strings.NewReader(text))
	var rules []Rule
	found := 0
	for {
		var doc k8sIngressDoc
		err := dec.Decode(&doc)
		if err == io.EOF {
			break
		}
		if err != nil {
			continue // best-effort: skip a document this narrow struct can't decode
		}
		if !strings.EqualFold(doc.Kind, "Ingress") {
			continue
		}
		found++
		name := doc.Metadata.Name
		for _, r := range doc.Spec.Rules {
			host := r.Host
			if host == "" {
				host = "*"
			}
			for _, p := range r.HTTP.Paths {
				rule := normalizeIngressPath(p.Path, p.PathType)
				rule.Host = host
				rule.Source = fmt.Sprintf("%s (%s)", name, source)
				rules = append(rules, rule)
			}
		}
	}
	if found == 0 {
		fmt.Fprintf(os.Stderr, "warning: no Ingress resources found in %s\n", source)
	}
	return rules
}

// renderHelmChart shells out to `helm template` with default values plus
// any --helm-set overrides. Returns rendered manifest YAML.
func renderHelmChart(dir, release string, sets []string) (string, error) {
	args := []string{"template", release, dir}
	for _, s := range sets {
		args = append(args, "--set", s)
	}
	cmd := exec.Command("helm", args...)
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%v: %s", err, strings.TrimSpace(errBuf.String()))
	}
	return out.String(), nil
}

// loadIngressFromValuesFallback is a best-effort, template-free fallback
// used only when `helm template` cannot render the chart (e.g. a
// required value has no safe default and none was supplied via
// --helm-set). It parses values.yaml as plain data and collects every
// "paths: [...]" list it finds anywhere in the tree, tagging each with
// the nearest sibling "name" field if present. It CANNOT resolve
// Go-template expressions, so any path built from a template variable
// (rare in practice — ingress paths are almost always literal strings)
// will not be recovered. Always assumed Prefix (matches how every
// observed chart's ingress template hard-codes pathType for such lists).
func loadIngressFromValuesFallback(dir string) ([]Rule, error) {
	data, err := os.ReadFile(filepath.Join(dir, "values.yaml"))
	if err != nil {
		return nil, err
	}
	var root interface{}
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, err
	}
	var rules []Rule
	var walk func(node interface{}, nameHint string)
	walk = func(node interface{}, nameHint string) {
		switch v := node.(type) {
		case map[string]interface{}:
			name := nameHint
			if n, ok := v["name"].(string); ok {
				name = n
			}
			if rawPaths, ok := v["paths"].([]interface{}); ok {
				for _, rp := range rawPaths {
					s, ok := rp.(string)
					if !ok {
						continue
					}
					rule := normalizeIngressPath(s, "Prefix")
					rule.Host = "*"
					rule.Source = fmt.Sprintf("%s (values.yaml, raw fallback)", name)
					rules = append(rules, rule)
				}
			}
			for k, sub := range v {
				walk(sub, k)
			}
		case []interface{}:
			for _, item := range v {
				walk(item, nameHint)
			}
		}
	}
	walk(root, filepath.Base(dir))
	return rules, nil
}

// loadIngressInput dispatches on whether the given --ingress argument is
// a chart directory or an already-rendered manifest file. A rendered
// manifest file is parsed directly and requires no helm binary.
func loadIngressInput(path, release string, helmSets []string) ([]Rule, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("cannot stat %s: %w", path, err)
	}
	if !info.IsDir() {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		return decodeIngressManifest(string(data), path), nil
	}

	out, err := renderHelmChart(path, release, helmSets)
	if err == nil {
		fmt.Fprintf(os.Stderr, "ingress source %s: rendered via helm template\n", path)
		return decodeIngressManifest(out, fmt.Sprintf("helm template %s", path)), nil
	}
	fmt.Fprintf(os.Stderr,
		"warning: helm template failed for %s (%v)\n"+
			"         falling back to raw values.yaml parsing — this is heuristic and does\n"+
			"         NOT evaluate Go-template expressions; pass --helm-set to supply the\n"+
			"         missing required value(s) and get a real render instead.\n", path, err)
	rules, ferr := loadIngressFromValuesFallback(path)
	if ferr != nil {
		return nil, fmt.Errorf("helm template failed (%v) and values.yaml fallback also failed: %w", err, ferr)
	}
	return rules, nil
}
