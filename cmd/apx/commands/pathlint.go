package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/infobloxopen/apx/internal/optable"
	"github.com/infobloxopen/apx/internal/pathlint"
	"github.com/spf13/cobra"
)

func newPathlintCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pathlint",
		Short: "Reconcile chart ingress paths against published spec paths",
		Long: `Pathlint compares the HTTP paths a service's rendered Kubernetes Ingress
actually exposes against the paths its published OpenAPI/Swagger spec
declares, and reports coverage in three buckets:

  [1] undeclared ingress surface — reachable in the cluster, in no spec
  [2] spec paths not reachable via any ingress rule — declared but unrouted
  [3] matched — ingress rules backed by a real spec path

Per WS-035 decision R4 this is a metric, not a hard gate: pass --warn-only to
always exit 0 (report unchanged) and surface drift as a warning. Without
--warn-only the command exits non-zero when bucket [1] or [2] is non-empty.

An --ingress argument may be a chart directory (rendered via 'helm template')
or an already-rendered manifest YAML file (no helm binary required).

The external path of a spec path is the base path of the spec (servers[0].url
or basePath) plus the path. When the client path does not start with that base
path, replace it: --external-prefix sets one prefix for every spec, and
--binding takes the externalPrefix of the matching API from a RouteAPIBinding
(a YAML file or a chart directory). With --binding, write a spec as
<api-id>=<file>; a plain file is accepted when the binding has one API. An API
without externalPrefix keeps the base path of its spec.

Examples:
  apx pathlint --ingress rendered.yaml --spec identity.swagger.json
  apx pathlint --ingress helm/identity --spec v1.json --spec v2.json --warn-only
  apx pathlint --ingress rendered.yaml --spec spec.yaml --out pathlint-report.txt
  apx pathlint --ingress rendered.yaml --spec spec.yaml --external-prefix /api/v2
  apx pathlint --ingress helm/identity --binding helm/identity \
      --spec openapi/iam/identity/v2=identity.yaml --format json`,
		RunE: pathlintAction,
	}
	cmd.Flags().StringArray("ingress", nil, "chart directory or rendered manifest YAML (repeatable, required)")
	cmd.Flags().StringArray("spec", nil, "OpenAPI v3 or Swagger v2 spec file, or <api-id>=<file> with --binding (repeatable, required)")
	cmd.Flags().String("external-prefix", "", "external path prefix that replaces the base path of every spec")
	cmd.Flags().String("binding", "", "RouteAPIBinding YAML file or chart directory; the externalPrefix of the matching API replaces the base path of its spec")
	cmd.Flags().String("format", "text", "Output format of the report: text or json")
	cmd.Flags().StringArray("helm-set", nil, "key=value passed to 'helm template --set' for chart-directory inputs (repeatable)")
	cmd.Flags().String("release-name", "pathlint", "helm release name used when rendering a chart directory")
	cmd.Flags().Bool("warn-only", false, "print the report but always exit 0, even if drift is found")
	cmd.Flags().String("out", "", "write the report to this file instead of stdout")
	_ = cmd.MarkFlagRequired("ingress")
	_ = cmd.MarkFlagRequired("spec")
	return cmd
}

func pathlintAction(cmd *cobra.Command, _ []string) error {
	ingressInputs, _ := cmd.Flags().GetStringArray("ingress")
	specInputs, _ := cmd.Flags().GetStringArray("spec")
	helmSets, _ := cmd.Flags().GetStringArray("helm-set")
	release, _ := cmd.Flags().GetString("release-name")
	warnOnly, _ := cmd.Flags().GetBool("warn-only")
	outPath, _ := cmd.Flags().GetString("out")
	externalPrefix, _ := cmd.Flags().GetString("external-prefix")
	bindingPath, _ := cmd.Flags().GetString("binding")
	format, _ := cmd.Flags().GetString("format")
	if format != "text" && format != "json" {
		return fmt.Errorf("invalid --format %q (expected text or json)", format)
	}

	specs, err := pathlintSpecs(specInputs, externalPrefix, bindingPath, release, helmSets)
	if err != nil {
		return err
	}
	report, err := pathlint.AnalyzeSpecs(ingressInputs, specs, helmSets, release)
	if err != nil {
		return err
	}

	out := io.Writer(os.Stdout)
	if outPath != "" {
		f, createErr := os.Create(outPath)
		if createErr != nil {
			return createErr
		}
		defer f.Close()
		out = f
	}
	if format == "json" {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report.JSON(warnOnly)); err != nil {
			return err
		}
	} else {
		report.Write(out, warnOnly)
	}

	if report.Drifted() && !warnOnly {
		return fmt.Errorf("pathlint: drift found (undeclared=%d unreachable=%d)", report.Undeclared, report.Unreachable)
	}
	return nil
}

// pathlintSpecs gives each --spec value its external prefixes. Without
// --external-prefix and --binding, a spec keeps its base path.
func pathlintSpecs(specInputs []string, externalPrefix, bindingPath, release string, helmSets []string) ([]pathlint.SpecInput, error) {
	if externalPrefix != "" && bindingPath != "" {
		return nil, errors.New("--external-prefix and --binding cannot be used together")
	}
	if externalPrefix != "" && !strings.HasPrefix(externalPrefix, "/") {
		return nil, fmt.Errorf("invalid --external-prefix %q (must start with \"/\")", externalPrefix)
	}
	specs := make([]pathlint.SpecInput, 0, len(specInputs))
	if bindingPath == "" {
		for _, sp := range specInputs {
			in := pathlint.SpecInput{Path: sp}
			if externalPrefix != "" {
				in.ExternalPrefixes = []string{externalPrefix}
			}
			specs = append(specs, in)
		}
		return specs, nil
	}

	bindings, err := optable.LoadBindings(bindingPath, release, helmSets)
	if err != nil {
		return nil, err
	}
	// The entries of each API, in binding order. One API can be in two
	// entries with different prefixes.
	entries := map[string][]optable.BindingAPI{}
	var ids []string
	for _, b := range bindings {
		for _, entry := range b.Spec.APIs {
			if _, ok := entries[entry.Ref]; !ok {
				ids = append(ids, entry.Ref)
			}
			entries[entry.Ref] = append(entries[entry.Ref], entry)
		}
	}
	for _, sp := range specInputs {
		id, file := "", sp
		// A value that is a file is a plain path, also when it has "=".
		if _, statErr := os.Stat(sp); statErr != nil {
			if left, right, ok := strings.Cut(sp, "="); ok && left != "" && right != "" {
				id, file = left, right
			}
		}
		if id == "" {
			if len(ids) != 1 {
				return nil, fmt.Errorf("--spec %s: the binding has %d APIs; write the spec as <api-id>=<file>", sp, len(ids))
			}
			id = ids[0]
		}
		list, ok := entries[id]
		if !ok {
			return nil, fmt.Errorf("--spec %s: API %s is not in the binding (APIs: %s)", sp, id, strings.Join(ids, ", "))
		}
		in := pathlint.SpecInput{Path: file}
		keepsBase := false
		var prefixes []string
		for _, entry := range list {
			if entry.ExternalPrefix == "" {
				keepsBase = true
				continue
			}
			dup := false
			for _, p := range prefixes {
				dup = dup || p == entry.ExternalPrefix
			}
			if !dup {
				prefixes = append(prefixes, entry.ExternalPrefix)
			}
		}
		if keepsBase && len(prefixes) > 0 {
			return nil, fmt.Errorf("--spec %s: API %s has entries with and without externalPrefix; pathlint cannot use both", sp, id)
		}
		in.ExternalPrefixes = prefixes
		specs = append(specs, in)
	}
	return specs, nil
}
