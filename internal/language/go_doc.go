package language

import _ "embed"

//go:embed go_doc/code_generation.md
var goCodeGenDoc string

//go:embed go_doc/dev_workflow.md
var goDevWorkflowDoc string

func (g *goPlugin) DocMeta() DocMeta {
	return DocMeta{
		SupportMatrix: map[string]string{
			"published_artifact": "Go module",
			"local_overlay":      "`go.work use`",
			"resolution":         "go.work -> go.mod",
			"codegen":            "`apx gen go`",
			"dev_command":        "`apx sync`",
			"unlink_hint":        "`go get ...`",
			"tier":               "**Tier 1**",
		},
		IdentityRows: []IdentityRow{
			{CoordType: "Go module", DerivedValue: "github.com/acme/apis/proto/payments/ledger"},
			{CoordType: "Go import", DerivedValue: "github.com/acme/apis/proto/payments/ledger/v1"},
		},
		PathMappings: []PathMapping{
			{
				APXPath:     "proto/payments/ledger/v1",
				TargetCoord: "github.com/<org>/apis/proto/payments/ledger",
				Description: "Go module path (v0/v1: no suffix)",
			},
			{
				APXPath:     "proto/payments/ledger/v2",
				TargetCoord: "github.com/<org>/apis/proto/payments/ledger/v2",
				Description: "Go module path (v2+: major suffix)",
			},
		},
		Sections: map[string]string{
			"code_generation": goCodeGenDoc,
			"dev_workflow":    goDevWorkflowDoc,
		},
	}
}
