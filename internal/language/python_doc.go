package language

import _ "embed"

//go:embed python_doc/code_generation.md
var pythonCodeGenDoc string

//go:embed python_doc/dev_workflow.md
var pythonDevWorkflowDoc string

func (p *pythonPlugin) DocMeta() DocMeta {
	return DocMeta{
		SupportMatrix: map[string]string{
			"published_artifact": "Python wheel",
			"local_overlay":      "`pip install -e`",
			"resolution":         "pkgutil namespace",
			"codegen":            "`apx gen python`",
			"dev_command":        "`apx link python`",
			"unlink_hint":        "`pip install ...`",
			"tier":               "Tier 2",
		},
		IdentityRows: []IdentityRow{
			{CoordType: "Py dist", DerivedValue: "acme-payments-ledger-v1"},
			{CoordType: "Py import", DerivedValue: "acme_apis.payments.ledger.v1"},
		},
		PathMappings: []PathMapping{
			{
				APXPath:     "proto/payments/ledger/v1",
				TargetCoord: "acme-payments-ledger-v1",
				Description: "Python distribution name",
			},
		},
		Sections: map[string]string{
			"code_generation": pythonCodeGenDoc,
			"dev_workflow":    pythonDevWorkflowDoc,
		},
	}
}
