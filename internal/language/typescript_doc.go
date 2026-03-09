package language

import _ "embed"

//go:embed typescript_doc/code_generation.md
var typescriptCodeGenDoc string

//go:embed typescript_doc/dev_workflow.md
var typescriptDevWorkflowDoc string

func (t *typescriptPlugin) DocMeta() DocMeta {
	return DocMeta{
		SupportMatrix: map[string]string{
			"published_artifact": "npm package",
			"local_overlay":      "`npm link`",
			"resolution":         "npm/yarn",
			"codegen":            "protoc + ts-proto",
			"dev_command":        "`npm link`",
			"unlink_hint":        "`npm install ...`",
			"tier":               "Tier 2",
		},
		IdentityRows: []IdentityRow{
			{CoordType: "npm", DerivedValue: "@acme/payments-ledger-v1-proto"},
		},
		PathMappings: []PathMapping{
			{
				APXPath:     "proto/payments/ledger/v1",
				TargetCoord: "@acme/payments-ledger-v1-proto",
				Description: "Scoped npm package name",
			},
		},
		Sections: map[string]string{
			"code_generation": typescriptCodeGenDoc,
			"dev_workflow":    typescriptDevWorkflowDoc,
		},
	}
}
