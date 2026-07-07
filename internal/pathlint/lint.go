package pathlint

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

// Report holds the result of reconciling ingress rules against spec paths.
type Report struct {
	IngressInputs []string
	SpecInputs    []string
	Rules         []Rule
	Specs         []SpecPath

	ruleCovered    []bool
	specCovered    []bool
	matchedExample []string

	MatchedCount int
	Undeclared   int
	Unreachable  int
}

// Analyze loads every ingress source and spec file, computes coverage, and
// returns a Report. releaseName is used when an --ingress argument is a chart
// directory (rendered via `helm template`); rendered-manifest files need no
// helm binary.
func Analyze(ingressInputs, specInputs, helmSets []string, releaseName string) (*Report, error) {
	r := &Report{IngressInputs: ingressInputs, SpecInputs: specInputs}

	for _, in := range ingressInputs {
		rs, err := loadIngressInput(in, releaseName, helmSets)
		if err != nil {
			return nil, err
		}
		r.Rules = append(r.Rules, rs...)
	}
	for _, sp := range specInputs {
		ps, err := loadSpec(sp)
		if err != nil {
			return nil, err
		}
		r.Specs = append(r.Specs, ps...)
	}

	r.ruleCovered = make([]bool, len(r.Rules))
	r.specCovered = make([]bool, len(r.Specs))
	// For bucket 3 we report, per matched ingress rule, one representative
	// spec path — enough to show *why* it's considered declared without
	// dumping the full cross product.
	r.matchedExample = make([]string, len(r.Rules))

	for i := range r.Rules {
		for j := range r.Specs {
			if matches(r.Rules[i], r.Specs[j]) {
				r.ruleCovered[i] = true
				r.specCovered[j] = true
				r.MatchedCount++
				if r.matchedExample[i] == "" {
					r.matchedExample[i] = r.Specs[j].Path + " (" + r.Specs[j].Spec + ")"
				}
			}
		}
	}
	for i := range r.Rules {
		if !r.ruleCovered[i] {
			r.Undeclared++
		}
	}
	for j := range r.Specs {
		if !r.specCovered[j] {
			r.Unreachable++
		}
	}
	return r, nil
}

// Drifted reports whether the ingress surface and spec paths disagree — any
// undeclared ingress path or any unreachable spec path.
func (r *Report) Drifted() bool {
	return r.Undeclared > 0 || r.Unreachable > 0
}

// Write renders the human/machine-readable coverage report. When warnOnly is
// true a drifted result is reported as WARN (and callers exit 0) rather than
// FAIL — the R4 metric-not-gate posture.
func (r *Report) Write(w io.Writer, warnOnly bool) {
	fmt.Fprintln(w, "pathlint report")
	fmt.Fprintln(w, "===============")
	fmt.Fprintf(w, "ingress inputs : %s\n", strings.Join(r.IngressInputs, ", "))
	fmt.Fprintf(w, "spec inputs    : %s\n", strings.Join(r.SpecInputs, ", "))
	fmt.Fprintf(w, "ingress rules  : %d\n", len(r.Rules))
	fmt.Fprintf(w, "spec paths     : %d\n\n", len(r.Specs))

	type row struct {
		key  string
		line string
	}

	var bucket1 []row
	for i, rule := range r.Rules {
		if !r.ruleCovered[i] {
			kind := "exact"
			if rule.Wildcard {
				kind = "prefix"
			}
			bucket1 = append(bucket1, row{rule.Path, fmt.Sprintf("  %-45s %-6s host=%-10s %s", rule.Path, kind, rule.Host, rule.Source)})
		}
	}
	sort.Slice(bucket1, func(i, j int) bool { return bucket1[i].key < bucket1[j].key })

	fmt.Fprintf(w, "[1] UNDECLARED INGRESS SURFACE — reachable in the cluster, absent from every spec (%d)\n", r.Undeclared)
	if r.Undeclared == 0 {
		fmt.Fprintln(w, "  (none)")
	}
	for _, b := range bucket1 {
		fmt.Fprintln(w, b.line)
	}
	fmt.Fprintln(w)

	var bucket2 []row
	for j, sp := range r.Specs {
		if !r.specCovered[j] {
			bucket2 = append(bucket2, row{sp.Path, fmt.Sprintf("  %-45s spec=%s base=%q", sp.Path, sp.Spec, sp.Base)})
		}
	}
	sort.Slice(bucket2, func(i, j int) bool { return bucket2[i].key < bucket2[j].key })

	fmt.Fprintf(w, "[2] SPEC PATHS NOT REACHABLE VIA ANY INGRESS RULE — declared but dead/unrouted (%d)\n", r.Unreachable)
	if r.Unreachable == 0 {
		fmt.Fprintln(w, "  (none)")
	}
	for _, b := range bucket2 {
		fmt.Fprintln(w, b.line)
	}
	fmt.Fprintln(w)

	matchedRules := 0
	var bucket3 []string
	for i, rule := range r.Rules {
		if r.ruleCovered[i] {
			matchedRules++
			bucket3 = append(bucket3, fmt.Sprintf("  %-45s -> %s", rule.Path, r.matchedExample[i]))
		}
	}
	sort.Strings(bucket3)
	fmt.Fprintf(w, "[3] MATCHED — %d/%d ingress rule(s) confirmed declared, %d/%d spec path(s) confirmed routed (%d pairs)\n",
		matchedRules, len(r.Rules), len(r.Specs)-r.Unreachable, len(r.Specs), r.MatchedCount)
	for _, l := range bucket3 {
		fmt.Fprintln(w, l)
	}
	fmt.Fprintln(w)

	status := "PASS"
	if r.Drifted() {
		status = "FAIL"
		if warnOnly {
			status = "WARN (--warn-only)"
		}
	}
	fmt.Fprintf(w, "RESULT: %s (undeclared=%d unreachable=%d matched_pairs=%d)\n", status, r.Undeclared, r.Unreachable, r.MatchedCount)
}
