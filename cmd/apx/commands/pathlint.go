package commands

import (
	"fmt"
	"io"
	"os"

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

Examples:
  apx pathlint --ingress rendered.yaml --spec identity.swagger.json
  apx pathlint --ingress helm/identity --spec v1.json --spec v2.json --warn-only
  apx pathlint --ingress rendered.yaml --spec spec.yaml --out pathlint-report.txt`,
		RunE: pathlintAction,
	}
	cmd.Flags().StringArray("ingress", nil, "chart directory or rendered manifest YAML (repeatable, required)")
	cmd.Flags().StringArray("spec", nil, "OpenAPI v3 or Swagger v2 spec file (repeatable, required)")
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

	report, err := pathlint.Analyze(ingressInputs, specInputs, helmSets, release)
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
	report.Write(out, warnOnly)

	if report.Drifted() && !warnOnly {
		return fmt.Errorf("pathlint: drift found (undeclared=%d unreachable=%d)", report.Undeclared, report.Unreachable)
	}
	return nil
}
