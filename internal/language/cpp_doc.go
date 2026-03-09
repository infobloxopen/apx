package language

import _ "embed"

//go:embed cpp_doc/code_generation.md
var cppCodeGenDoc string

//go:embed cpp_doc/dev_workflow.md
var cppDevWorkflowDoc string

func (c *cppPlugin) DocMeta() DocMeta {
	return DocMeta{
		SupportMatrix: map[string]string{
			"published_artifact": "Conan package",
			"local_overlay":      "`conan editable`",
			"resolution":         "Conan dependency",
			"codegen":            "`protoc` + `protoc-gen-grpc`",
			"dev_command":        "`conan install`",
			"unlink_hint":        "Update `conanfile`",
			"tier":               "Tier 2",
		},
		IdentityRows: []IdentityRow{
			{CoordType: "Conan", DerivedValue: "acme-payments-ledger-v1-proto"},
			{CoordType: "C++ ns", DerivedValue: "acme::payments::ledger::v1"},
		},
		PathMappings: []PathMapping{
			{
				APXPath:     "proto/payments/ledger/v1",
				TargetCoord: "acme-payments-ledger-v1-proto",
				Description: "Conan package reference",
			},
		},
		Sections: map[string]string{
			"code_generation": cppCodeGenDoc,
			"dev_workflow":    cppDevWorkflowDoc,
		},
	}
}
