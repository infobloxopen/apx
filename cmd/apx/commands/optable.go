package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/infobloxopen/apx/internal/catalog"
	"github.com/infobloxopen/apx/internal/opclass"
	"github.com/infobloxopen/apx/internal/optable"
	"github.com/spf13/cobra"
)

func newOptableCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "optable",
		Short: "Render the operation tables of the APIs that a route binding names",
		Long: `Optable works with the operation table of an API: the list of its operations
with the external path template, the operationId, the operation class and the
streaming flag of each one.`,
	}
	cmd.AddCommand(newOptableRenderCmd())
	return cmd
}

func newOptableRenderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "render",
		Short: "Write one operation-table ConfigMap for each API of a RouteAPIBinding",
		Long: `Render reads one or more RouteAPIBinding documents
(apiVersion traefik.infoblox.com/v1alpha1) and the OpenAPI contract of each API
that they pin. For each API it writes a Kubernetes ConfigMap manifest with the
operation table. With --route-table it also writes the route table of the
routes that the bindings target.

A --binding argument is a YAML file or a Helm chart directory. A chart
directory is rendered with 'helm template'; a file needs no helm binary.
Documents of other kinds in the input are skipped.

The contract of each API comes from --spec <api-id>=<file>, or from
--specs-dir, a directory with the layout of an API repository: the single
OpenAPI file in <dir>/<api-id>/. Check out that directory at the pinned
version; render does not download a release.

The external path template of an operation is the externalPrefix of the
binding entry when it is set, otherwise the base path of the contract, joined
with the path of the operation.

The audience of an API is --audience <api-id>=public|internal, otherwise the
audience of the catalog entry (--catalog, or catalog/catalog.yaml in
--specs-dir), otherwise internal.

More than one API can be bound to one route. For each method and path, a
literal segment wins over a {name} segment, then a public API wins over an
internal API. Any other overlap is a conflict, and so is one path with two
operationId values. A conflict makes the command fail and no file is written.

Examples:
  apx optable render --binding binding.yaml \
      --spec openapi/payments/ledger/v1=ledger.yaml --out chart/files
  apx optable render --binding helm/payments --specs-dir ../apis \
      --out rendered --route-table --namespace payments
  apx optable render --binding binding.yaml --specs-dir ../apis \
      --out rendered --route-table-configmap route-table --format json`,
		Args: cobra.NoArgs,
		RunE: optableRenderAction,
	}
	cmd.Flags().StringArray("binding", nil, "RouteAPIBinding YAML file or chart directory (repeatable, required)")
	cmd.Flags().StringArray("spec", nil, "<api-id>=<file>: the OpenAPI file of one API (repeatable)")
	cmd.Flags().String("specs-dir", "", "directory with the layout of an API repository; the contract of an API is in <dir>/<api-id>/")
	cmd.Flags().StringArray("audience", nil, "<api-id>=public|internal: the audience of one API (repeatable)")
	cmd.Flags().String("catalog", "", "path or URL of a catalog file that gives the audience of each API")
	cmd.Flags().String("out", "", "output directory (required)")
	cmd.Flags().Bool("route-table", false, "also write route-table.json")
	cmd.Flags().String("route-table-configmap", "", "also write a ConfigMap with this name that holds the gzip route table (implies --route-table)")
	cmd.Flags().String("namespace", "", "metadata.namespace of the manifests, and the namespace of a binding that has none")
	cmd.Flags().String("format", "text", "Output format of the summary: text or json")
	cmd.Flags().StringArray("helm-set", nil, "key=value passed to 'helm template --set' for a chart directory (repeatable)")
	cmd.Flags().String("release-name", "optable", "helm release name used when rendering a chart directory")
	_ = cmd.MarkFlagRequired("binding")
	_ = cmd.MarkFlagRequired("out")
	return cmd
}

// optableSummary is the JSON output of `apx optable render`.
type optableSummary struct {
	APIs      []optableAPISummary   `json:"apis"`
	Routes    []optableRouteSummary `json:"routes"`
	Conflicts []optable.Conflict    `json:"conflicts"`
	// Files are the files that the command wrote, relative to --out.
	Files []string `json:"files"`
}

type optableAPISummary struct {
	API        string         `json:"api"`
	Version    string         `json:"version"`
	Audience   string         `json:"audience"`
	Spec       string         `json:"spec"`
	ConfigMap  string         `json:"configMap"`
	Operations int            `json:"operations"`
	Classes    map[string]int `json:"classes"`
}

type optableRouteSummary struct {
	Namespace  string `json:"namespace"`
	Name       string `json:"name"`
	DefaultAPI string `json:"defaultApi"`
	Operations int    `json:"operations"`
}

func optableRenderAction(cmd *cobra.Command, _ []string) error {
	bindingInputs, _ := cmd.Flags().GetStringArray("binding")
	specFlags, _ := cmd.Flags().GetStringArray("spec")
	specsDir, _ := cmd.Flags().GetString("specs-dir")
	audienceFlags, _ := cmd.Flags().GetStringArray("audience")
	catalogFlag, _ := cmd.Flags().GetString("catalog")
	outDir, _ := cmd.Flags().GetString("out")
	routeTable, _ := cmd.Flags().GetBool("route-table")
	routeTableConfigMap, _ := cmd.Flags().GetString("route-table-configmap")
	namespace, _ := cmd.Flags().GetString("namespace")
	helmSets, _ := cmd.Flags().GetStringArray("helm-set")
	release, _ := cmd.Flags().GetString("release-name")
	format, _ := cmd.Flags().GetString("format")
	if asJSON, _ := cmd.Flags().GetBool("json"); asJSON {
		format = "json"
	}
	if format != "text" && format != "json" {
		return fmt.Errorf("invalid --format %q (expected text or json)", format)
	}
	if routeTableConfigMap != "" {
		routeTable = true
	}

	specFiles, err := parseAPIPairs("spec", specFlags)
	if err != nil {
		return err
	}
	audiences, err := parseAPIPairs("audience", audienceFlags)
	if err != nil {
		return err
	}
	for id, a := range audiences {
		if a != optable.AudiencePublic && a != optable.AudienceInternal {
			return fmt.Errorf("invalid --audience %s=%s (expected %s or %s)", id, a, optable.AudiencePublic, optable.AudienceInternal)
		}
	}

	var bindings []*optable.Binding
	for _, in := range bindingInputs {
		bs, loadErr := optable.LoadBindings(in, release, helmSets)
		if loadErr != nil {
			return loadErr
		}
		bindings = append(bindings, bs...)
	}

	catalogAudiences, err := optableCatalogAudiences(catalogFlag, specsDir)
	if err != nil {
		return err
	}

	apis := map[string]optable.API{}
	specOf := map[string]string{}
	var errs []error
	for _, b := range bindings {
		for _, entry := range b.Spec.APIs {
			if _, done := apis[entry.Ref]; done {
				continue
			}
			specPath, findErr := optableSpecPath(entry.Ref, specFiles, specsDir)
			if findErr != nil {
				errs = append(errs, findErr)
				continue
			}
			doc, loadErr := opclass.Load(specPath)
			if loadErr != nil {
				errs = append(errs, fmt.Errorf("API %s: %w", entry.Ref, loadErr))
				continue
			}
			audience := audiences[entry.Ref]
			if audience == "" {
				audience = catalogAudiences[entry.Ref]
			}
			if audience == "" {
				audience = optable.AudienceInternal
			}
			apis[entry.Ref] = optable.API{Document: doc, Audience: audience}
			specOf[entry.Ref] = filepath.ToSlash(specPath)
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	res, err := optable.Build(bindings, apis, namespace)
	if err != nil {
		return err
	}
	if routeTable {
		for _, r := range res.RouteTable.Routes {
			if r.Namespace == "" {
				return fmt.Errorf("route %s has no namespace: set metadata.namespace on the binding or pass --namespace", r.Name)
			}
		}
	}

	summary := optableSummary{
		APIs:      []optableAPISummary{},
		Routes:    []optableRouteSummary{},
		Conflicts: res.Conflicts,
		Files:     []string{},
	}
	if summary.Conflicts == nil {
		summary.Conflicts = []optable.Conflict{}
	}
	for i := range res.Tables {
		t := &res.Tables[i]
		classes := map[string]int{}
		counts := t.Counts()
		for _, c := range opclass.Classes() {
			classes[string(c)] = counts[c]
		}
		summary.APIs = append(summary.APIs, optableAPISummary{
			API: t.API, Version: t.Version, Audience: t.Audience, Spec: specOf[t.API],
			ConfigMap: optable.ConfigMapName(t.API), Operations: len(t.Operations), Classes: classes,
		})
	}
	for _, r := range res.RouteTable.Routes {
		summary.Routes = append(summary.Routes, optableRouteSummary{
			Namespace: r.Namespace, Name: r.Name, DefaultAPI: r.DefaultAPI, Operations: len(r.Operations),
		})
	}

	out := cmd.OutOrStdout()
	if conflictErr := res.Err(); conflictErr != nil {
		// A conflict writes no file. The JSON summary still lists the conflicts.
		if format == "json" {
			if encErr := writeOptableJSON(out, summary); encErr != nil {
				return encErr
			}
		}
		return conflictErr
	}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}
	write := func(name string, data []byte) error {
		if err := os.WriteFile(filepath.Join(outDir, name), data, 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", name, err)
		}
		summary.Files = append(summary.Files, name)
		return nil
	}
	for i := range res.Tables {
		t := &res.Tables[i]
		data, encErr := t.ConfigMap(namespace)
		if encErr != nil {
			return encErr
		}
		if err := write(optable.ConfigMapName(t.API)+".yaml", data); err != nil {
			return err
		}
	}
	if routeTable {
		data, encErr := res.RouteTable.JSON()
		if encErr != nil {
			return encErr
		}
		if err := write(optable.RouteTableFile, data); err != nil {
			return err
		}
	}
	if routeTableConfigMap != "" {
		data, encErr := res.RouteTable.ConfigMap(routeTableConfigMap, namespace)
		if encErr != nil {
			return encErr
		}
		if err := write("route-table-configmap.yaml", data); err != nil {
			return err
		}
	}

	if format == "json" {
		return writeOptableJSON(out, summary)
	}
	writeOptableText(out, summary, outDir)
	return nil
}

func writeOptableJSON(out io.Writer, summary optableSummary) error {
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(summary)
}

func writeOptableText(out io.Writer, summary optableSummary, outDir string) {
	for _, a := range summary.APIs {
		parts := make([]string, 0, 4)
		for _, c := range opclass.Classes() {
			parts = append(parts, fmt.Sprintf("%s=%d", c, a.Classes[string(c)]))
		}
		fmt.Fprintf(out, "%s %s (%s): %d operations: %s -> %s\n",
			a.API, a.Version, a.Audience, a.Operations, strings.Join(parts, " "), a.ConfigMap)
	}
	for _, r := range summary.Routes {
		name := r.Name
		if r.Namespace != "" {
			name = r.Namespace + "/" + r.Name
		}
		fmt.Fprintf(out, "route %s: %d operations, default API %s\n", name, r.Operations, r.DefaultAPI)
	}
	fmt.Fprintf(out, "wrote %d file(s) to %s: %s\n", len(summary.Files), filepath.ToSlash(outDir), strings.Join(summary.Files, ", "))
}

// parseAPIPairs reads repeated "<api-id>=<value>" flag values. An API id has no
// "=", so the first "=" is the separator.
func parseAPIPairs(flag string, values []string) (map[string]string, error) {
	out := make(map[string]string, len(values))
	for _, v := range values {
		id, value, ok := strings.Cut(v, "=")
		if !ok || id == "" || value == "" {
			return nil, fmt.Errorf("invalid --%s %q (expected <api-id>=<value>)", flag, v)
		}
		if _, dup := out[id]; dup {
			return nil, fmt.Errorf("--%s is given two times for API %s", flag, id)
		}
		out[id] = value
	}
	return out, nil
}

// optableSpecPath finds the OpenAPI file of an API: the --spec value, else the
// single OpenAPI file in <specs-dir>/<api-id>/.
func optableSpecPath(apiID string, specFiles map[string]string, specsDir string) (string, error) {
	if p, ok := specFiles[apiID]; ok {
		return p, nil
	}
	if specsDir == "" {
		return "", fmt.Errorf("API %s: no API contract: pass --spec %s=<file> or --specs-dir <dir>", apiID, apiID)
	}
	dir := filepath.Join(specsDir, filepath.FromSlash(apiID))
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("API %s: %w", apiID, err)
	}
	var found []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		switch strings.ToLower(filepath.Ext(e.Name())) {
		case ".yaml", ".yml", ".json":
		default:
			continue
		}
		p := filepath.Join(dir, e.Name())
		if _, loadErr := opclass.Load(p); loadErr != nil {
			if errors.Is(loadErr, opclass.ErrNoPaths) {
				continue // not an OpenAPI document
			}
			return "", fmt.Errorf("API %s: %w", apiID, loadErr)
		}
		found = append(found, p)
	}
	switch len(found) {
	case 0:
		return "", fmt.Errorf("API %s: no OpenAPI file in %s", apiID, filepath.ToSlash(dir))
	case 1:
		return found[0], nil
	}
	names := make([]string, len(found))
	for i, p := range found {
		names[i] = filepath.Base(p)
	}
	return "", fmt.Errorf("API %s: %d OpenAPI files in %s (%s): pass --spec %s=<file>",
		apiID, len(found), filepath.ToSlash(dir), strings.Join(names, ", "), apiID)
}

// optableCatalogAudiences reads the audience of each catalog module. The
// catalog is the --catalog value, else catalog/catalog.yaml in --specs-dir
// when that file is there. Without a catalog the result is empty.
func optableCatalogAudiences(catalogFlag, specsDir string) (map[string]string, error) {
	out := map[string]string{}
	path := catalogFlag
	if path == "" {
		if specsDir == "" {
			return out, nil
		}
		path = filepath.Join(specsDir, "catalog", "catalog.yaml")
		if _, err := os.Stat(path); err != nil {
			return out, nil
		}
	}
	cat, err := catalog.SourceFor(path).Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load catalog from %s: %w", path, err)
	}
	for _, m := range cat.Modules {
		if m.Audience == "" {
			continue
		}
		id := m.ID
		if id == "" {
			id = m.Name
		}
		out[id] = m.Audience
	}
	return out, nil
}
