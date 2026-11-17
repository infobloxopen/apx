package validator

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/infobloxopen/apx/internal/opclass"
)

// CheckOperationIDs fails when an operation of the OpenAPI document at specFile
// has no operationId, or when an operationId repeats inside the document. It
// also fails on an invalid x-operation-class. It is native Go: it does not need
// spectral or a ruleset. A document without a "paths" object passes.
func CheckOperationIDs(specFile string) error {
	doc, err := opclass.Load(specFile)
	if err != nil {
		if errors.Is(err, opclass.ErrNoPaths) {
			return nil
		}
		return fmt.Errorf("operationId check: %w", err)
	}
	var problems []string
	byID := map[string][]string{}
	for _, op := range doc.Operations {
		where := op.Method + " " + op.Path
		if op.OperationID == "" {
			problems = append(problems, where+": operationId is missing")
			continue
		}
		byID[op.OperationID] = append(byID[op.OperationID], where)
	}
	ids := make([]string, 0, len(byID))
	for id, uses := range byID {
		if len(uses) > 1 {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	for _, id := range ids {
		problems = append(problems, fmt.Sprintf("operationId %q is not unique: %s", id, strings.Join(byID[id], ", ")))
	}
	if len(problems) == 0 {
		return nil
	}
	return fmt.Errorf("operationId check failed for %s (%d problems):\n  %s",
		specFile, len(problems), strings.Join(problems, "\n  "))
}

// CompareOperationIDs reports the operationIds of the base document that the
// revision renames or removes. Operations are matched by HTTP method and path;
// the names of path variables do not matter. A finding is:
//
//   - an operation that stays on the same method and path with another
//     operationId, or with none;
//   - an operation that is gone, when no operation of the revision has its
//     operationId.
//
// A new operationId on an operation that had none is not a finding. When one
// of the documents does not parse, there is no comparison and no error: the
// diff tool reports a document that does not load.
func CompareOperationIDs(baseFile, revFile string) error {
	base, err := opclass.Load(baseFile)
	if err != nil {
		return nil
	}
	rev, err := opclass.Load(revFile)
	if err != nil {
		return nil
	}
	revByKey := make(map[string]string, len(rev.Operations))
	revIDs := make(map[string]bool, len(rev.Operations))
	for _, op := range rev.Operations {
		revByKey[operationKey(op)] = op.OperationID
		if op.OperationID != "" {
			revIDs[op.OperationID] = true
		}
	}
	var problems []string
	for _, op := range base.Operations {
		if op.OperationID == "" {
			continue
		}
		where := op.Method + " " + op.Path
		now, kept := revByKey[operationKey(op)]
		switch {
		case kept && now == op.OperationID:
		case kept && now == "":
			problems = append(problems, fmt.Sprintf("%s: operationId %q was removed", where, op.OperationID))
		case kept:
			problems = append(problems, fmt.Sprintf("%s: operationId %q was renamed to %q", where, op.OperationID, now))
		case !revIDs[op.OperationID]:
			problems = append(problems, fmt.Sprintf("%s: operationId %q was removed with its operation", where, op.OperationID))
		}
	}
	if len(problems) == 0 {
		return nil
	}
	return fmt.Errorf("operationId changes (%d); a renamed or removed operationId breaks generated clients and per-operation history:\n  %s",
		len(problems), strings.Join(problems, "\n  "))
}

// operationKey is the method and the path with each variable name removed.
func operationKey(op opclass.Operation) string {
	var b strings.Builder
	depth := 0
	for _, c := range op.Path {
		switch {
		case c == '{':
			if depth == 0 {
				b.WriteString("{}")
			}
			depth++
		case c == '}' && depth > 0:
			depth--
		case depth == 0:
			b.WriteRune(c)
		}
	}
	return op.Method + " " + b.String()
}
