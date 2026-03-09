package language

import _ "embed"

//go:embed java_doc/code_generation.md
var javaCodeGenDoc string

//go:embed java_doc/dev_workflow.md
var javaDevWorkflowDoc string

func (j *javaPlugin) DocMeta() DocMeta {
	return DocMeta{
		SupportMatrix: map[string]string{
			"published_artifact": "Maven JAR",
			"local_overlay":      "`mvn install`",
			"resolution":         "Maven dependency",
			"codegen":            "protobuf-maven-plugin",
			"dev_command":        "`mvn generate-sources`",
			"unlink_hint":        "Update `pom.xml`",
			"tier":               "Tier 2",
		},
		IdentityRows: []IdentityRow{
			{CoordType: "Maven", DerivedValue: "com.acme.apis:payments-ledger-v1-proto"},
			{CoordType: "Java pkg", DerivedValue: "com.acme.apis.payments.ledger.v1"},
		},
		PathMappings: []PathMapping{
			{
				APXPath:     "proto/payments/ledger/v1",
				TargetCoord: "com.acme.apis:payments-ledger-v1-proto",
				Description: "Maven coordinates (groupId:artifactId)",
			},
		},
		Sections: map[string]string{
			"code_generation": javaCodeGenDoc,
			"dev_workflow":    javaDevWorkflowDoc,
		},
	}
}
