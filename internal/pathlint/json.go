package pathlint

import "sort"

// JSONReport is the machine-readable form of a Report (--format json).
type JSONReport struct {
	IngressInputs []string          `json:"ingressInputs"`
	SpecInputs    []string          `json:"specInputs"`
	IngressRules  int               `json:"ingressRules"`
	SpecPaths     int               `json:"specPaths"`
	Undeclared    []JSONIngressRule `json:"undeclared"`
	Unreachable   []JSONSpecPath    `json:"unreachable"`
	Matched       []JSONMatch       `json:"matched"`
	MatchedPairs  int               `json:"matchedPairs"`
	// Result is PASS, WARN or FAIL.
	Result string `json:"result"`
}

// JSONIngressRule is an ingress rule that no spec path declares.
type JSONIngressRule struct {
	Path   string `json:"path"`
	Kind   string `json:"kind"` // exact or prefix
	Host   string `json:"host"`
	Source string `json:"source"`
}

// JSONSpecPath is a spec path that no ingress rule routes.
type JSONSpecPath struct {
	Path string `json:"path"`
	Spec string `json:"spec"`
	Base string `json:"base"`
}

// JSONMatch is an ingress rule with one spec path that declares it.
type JSONMatch struct {
	Path    string `json:"path"`
	Example string `json:"example"`
}

// JSON returns the report as a JSONReport. With warnOnly, a drifted result is
// WARN and not FAIL.
func (r *Report) JSON(warnOnly bool) JSONReport {
	out := JSONReport{
		IngressInputs: r.IngressInputs,
		SpecInputs:    r.SpecInputs,
		IngressRules:  len(r.Rules),
		SpecPaths:     len(r.Specs),
		Undeclared:    []JSONIngressRule{},
		Unreachable:   []JSONSpecPath{},
		Matched:       []JSONMatch{},
		MatchedPairs:  r.MatchedCount,
		Result:        "PASS",
	}
	for i, rule := range r.Rules {
		if r.ruleCovered[i] {
			out.Matched = append(out.Matched, JSONMatch{Path: rule.Path, Example: r.matchedExample[i]})
			continue
		}
		kind := "exact"
		if rule.Wildcard {
			kind = "prefix"
		}
		out.Undeclared = append(out.Undeclared, JSONIngressRule{Path: rule.Path, Kind: kind, Host: rule.Host, Source: rule.Source})
	}
	for j, sp := range r.Specs {
		if !r.specCovered[j] {
			out.Unreachable = append(out.Unreachable, JSONSpecPath{Path: sp.Path, Spec: sp.Spec, Base: sp.Base})
		}
	}
	sort.SliceStable(out.Undeclared, func(i, j int) bool { return out.Undeclared[i].Path < out.Undeclared[j].Path })
	sort.SliceStable(out.Unreachable, func(i, j int) bool {
		a, b := out.Unreachable[i], out.Unreachable[j]
		if a.Path != b.Path {
			return a.Path < b.Path
		}
		return a.Spec < b.Spec
	})
	sort.SliceStable(out.Matched, func(i, j int) bool { return out.Matched[i].Path < out.Matched[j].Path })
	if r.Drifted() {
		out.Result = "FAIL"
		if warnOnly {
			out.Result = "WARN"
		}
	}
	return out
}
