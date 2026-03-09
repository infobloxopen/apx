package language

import _ "embed"

//go:embed rust_doc/code_generation.md
var rustCodeGenDoc string

//go:embed rust_doc/dev_workflow.md
var rustDevWorkflowDoc string

func (r *rustPlugin) DocMeta() DocMeta {
	return DocMeta{
		SupportMatrix: map[string]string{
			"published_artifact": "Cargo crate",
			"local_overlay":      "`cargo path dep`",
			"resolution":         "Cargo dependency",
			"codegen":            "`tonic-build` / `prost-build`",
			"dev_command":        "`cargo build`",
			"unlink_hint":        "Update `Cargo.toml`",
			"tier":               "Tier 2",
		},
		IdentityRows: []IdentityRow{
			{CoordType: "Crate", DerivedValue: "acme-payments-ledger-v1-proto"},
			{CoordType: "Rust mod", DerivedValue: "acme_payments::ledger::v1"},
		},
		PathMappings: []PathMapping{
			{
				APXPath:     "proto/payments/ledger/v1",
				TargetCoord: "acme-payments-ledger-v1-proto",
				Description: "Cargo crate name",
			},
		},
		Sections: map[string]string{
			"code_generation": rustCodeGenDoc,
			"dev_workflow":    rustDevWorkflowDoc,
		},
	}
}
