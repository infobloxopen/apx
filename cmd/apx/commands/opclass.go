package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/infobloxopen/apx/internal/opclass"
	"github.com/infobloxopen/apx/internal/ui"
	"github.com/spf13/cobra"
)

func newOpclassCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "opclass <spec>",
		Short: "Classify the operations of an OpenAPI spec (read, list, write, custom)",
		Long: `Opclass gives each operation of an OpenAPI 3.x or Swagger 2 document an
operation class: read, list, write or custom. The class is the value of the
extension x-operation-class.

An x-operation-class that is already on an operation always wins. Without it,
the first rule that matches decides:

  1. x-aip-method, when it is present and is not Custom
  2. a verb word at the start or the end of the operationId
     (List, Read, Get, Create, Update, Patch, Delete)
  3. the HTTP verb and the path shape: GET /things/{id} is read, GET /things
     is list, and POST, PUT, PATCH and DELETE are write
  4. otherwise custom

The streaming column is true when a response has the content type
text/event-stream or application/x-ndjson, or when x-streaming is true.

With --write, opclass adds x-operation-class to each operation that lacks it.
It inserts lines and does not re-encode the file: comments, key order, quoting
and line endings stay as they are. It never changes x-aip-method.

Examples:
  apx opclass openapi/payments/ledger/v1/ledger.yaml
  apx opclass ledger.yaml --format json
  apx opclass ledger.yaml --write`,
		Args: cobra.ExactArgs(1),
		RunE: opclassAction,
	}
	cmd.Flags().String("format", "text", "Output format: text or json")
	cmd.Flags().Bool("write", false, "Add x-operation-class to each operation of the file that lacks it")
	return cmd
}

// opclassReport is the JSON output of `apx opclass`.
type opclassReport struct {
	Spec       string              `json:"spec"`
	BasePath   string              `json:"basePath"`
	Operations []opclass.Operation `json:"operations"`
	Counts     map[string]int      `json:"counts"`
	Added      *int                `json:"added,omitempty"`
}

func opclassAction(cmd *cobra.Command, args []string) error {
	specPath := args[0]
	format, _ := cmd.Flags().GetString("format")
	if asJSON, _ := cmd.Flags().GetBool("json"); asJSON {
		format = "json"
	}
	if format != "text" && format != "json" {
		return fmt.Errorf("invalid --format %q (expected text or json)", format)
	}

	report := opclassReport{Spec: specPath}
	if write, _ := cmd.Flags().GetBool("write"); write {
		added, err := opclassWrite(specPath)
		if err != nil {
			return err
		}
		report.Added = &added
	}

	doc, err := opclass.Load(specPath)
	if err != nil {
		return err
	}
	report.BasePath = doc.BasePath
	report.Operations = doc.Operations
	report.Counts = map[string]int{}
	counts := doc.Counts()
	for _, c := range opclass.Classes() {
		report.Counts[string(c)] = counts[c]
	}

	out := cmd.OutOrStdout()
	if format == "json" {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(report)
	}
	writeOpclassText(out, report)
	if report.Added != nil {
		ui.Success("Added %s to %d operations in %s", opclass.ExtensionClass, *report.Added, specPath)
	}
	return nil
}

// opclassWrite annotates the file in place and keeps its permissions.
func opclassWrite(specPath string) (int, error) {
	data, err := os.ReadFile(specPath)
	if err != nil {
		return 0, fmt.Errorf("reading spec: %w", err)
	}
	out, added, err := opclass.Annotate(data)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", specPath, err)
	}
	if added == 0 {
		return 0, nil
	}
	mode := os.FileMode(0o644)
	if info, statErr := os.Stat(specPath); statErr == nil {
		mode = info.Mode().Perm()
	}
	if err := os.WriteFile(specPath, out, mode); err != nil {
		return 0, fmt.Errorf("writing spec: %w", err)
	}
	return added, nil
}

func writeOpclassText(out io.Writer, report opclassReport) {
	tw := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "METHOD\tPATH\tOPERATION ID\tCLASS\tSTREAMING\tRULE")
	for _, op := range report.Operations {
		id := op.OperationID
		if id == "" {
			id = "-"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n", op.Method, op.Path, id, op.Class, strconv.FormatBool(op.Streaming), op.Rule)
	}
	_ = tw.Flush()
	parts := make([]string, 0, 4)
	for _, c := range opclass.Classes() {
		parts = append(parts, fmt.Sprintf("%s=%d", c, report.Counts[string(c)]))
	}
	fmt.Fprintf(out, "\n%d operations: %s\n", len(report.Operations), strings.Join(parts, " "))
}
