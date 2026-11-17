package optable

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"math/rand"
	"strings"
	"testing"

	"github.com/infobloxopen/apx/internal/opclass"
	"gopkg.in/yaml.v3"
)

const ledgerProto = "proto/payments/ledger/v1"

// grpcDocument is the classified document of a gRPC API, as package protoset
// gives it: POST /<package>.<Service>/<Method>, operationId <Service>_<Method>.
func grpcDocument() *opclass.Document {
	return &opclass.Document{Operations: []opclass.Operation{
		{Method: "POST", Path: "/payments.ledger.v1.Ledger/GetEntry", OperationID: "Ledger_GetEntry", Class: opclass.Read, Rule: opclass.RuleName},
		{Method: "POST", Path: "/payments.ledger.v1.Ledger/WatchEntries", OperationID: "Ledger_WatchEntries", Class: opclass.Write, Streaming: true, Rule: opclass.RuleShape},
	}}
}

func grpcAPI(set []byte) API {
	sum := sha256.Sum256(set)
	return API{Document: grpcDocument(), Descriptor: &Descriptor{Set: set, Digest: "sha256:" + hex.EncodeToString(sum[:])}}
}

func TestDescriptorConfigMapName(t *testing.T) {
	if got := DescriptorConfigMapName(ledgerProto); got != "optable-4def501dd8-descriptors" {
		t.Errorf("DescriptorConfigMapName = %s", got)
	}
	if !strings.HasPrefix(DescriptorConfigMapName(ledgerProto), ConfigMapName(ledgerProto)) {
		t.Error("the descriptor ConfigMap name does not start with the operation-table ConfigMap name")
	}
}

func TestBuildGRPCAPI(t *testing.T) {
	set := []byte("synthetic descriptor set")
	// One route serves the gRPC API and the REST gateway of the same service.
	b := binding("payments",
		BindingAPI{Ref: ledgerProto, Version: "v1.2.0", TargetRefs: []TargetRef{ingress("ledger")}},
		BindingAPI{Ref: "openapi/payments/ledger/v1", Version: "v1.2.0", TargetRefs: []TargetRef{ingress("ledger")}},
	)
	apis := map[string]API{
		ledgerProto:                  grpcAPI(set),
		"openapi/payments/ledger/v1": {Document: spec(t, "/v1", "GET /entries/{id} Ledger_GetEntry")},
	}
	res, err := Build([]*Binding{b}, apis, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := res.Err(); err != nil {
		t.Fatalf("the gRPC paths and the REST paths are in conflict: %v", err)
	}
	var grpcTable, restTable *Table
	for i := range res.Tables {
		switch res.Tables[i].API {
		case ledgerProto:
			grpcTable = &res.Tables[i]
		case "openapi/payments/ledger/v1":
			restTable = &res.Tables[i]
		}
	}
	if grpcTable == nil || restTable == nil {
		t.Fatalf("tables = %+v", res.Tables)
	}
	want := TableDescriptor{ConfigMap: "optable-4def501dd8-descriptors", Key: DescriptorDataKey, Digest: apis[ledgerProto].Descriptor.Digest}
	if grpcTable.Descriptor == nil || *grpcTable.Descriptor != want {
		t.Errorf("gRPC table descriptor = %+v, want %+v", grpcTable.Descriptor, want)
	}
	if restTable.Descriptor != nil {
		t.Errorf("the OpenAPI table has a descriptor: %+v", restTable.Descriptor)
	}
	if got := grpcTable.Operations[0]; got.Method != "POST" || got.Path != "/payments.ledger.v1.Ledger/GetEntry" || got.OperationID != "Ledger_GetEntry" {
		t.Errorf("gRPC operation = %+v", got)
	}
	if !grpcTable.Operations[1].Streaming {
		t.Error("the streaming flag of WatchEntries is lost")
	}

	// The JSON of the gRPC table has the descriptor; the JSON of the OpenAPI table does not.
	data, err := grpcTable.JSON()
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	d, ok := decoded["descriptor"].(map[string]any)
	if !ok || d["configMap"] != want.ConfigMap || d["key"] != DescriptorDataKey || d["digest"] != want.Digest {
		t.Errorf("descriptor JSON = %v", decoded["descriptor"])
	}
	if data, err = restTable.JSON(); err != nil || bytes.Contains(data, []byte(`"descriptor"`)) {
		t.Errorf("the OpenAPI table JSON has a descriptor (err %v)", err)
	}

	// The route has the REST and the gRPC operations of the two APIs.
	ops := map[string]string{}
	for _, op := range res.RouteTable.Routes[0].Operations {
		ops[op.Method+" "+op.Path] = op.API
	}
	if ops["POST /payments.ledger.v1.Ledger/GetEntry"] != ledgerProto || ops["GET /v1/entries/{id}"] != "openapi/payments/ledger/v1" {
		t.Errorf("route operations = %v", ops)
	}
}

func TestBuildGRPCExternalPrefix(t *testing.T) {
	for _, prefix := range []string{"", "/"} {
		b := binding("payments", BindingAPI{Ref: ledgerProto, Version: "v1.0.0", ExternalPrefix: prefix, TargetRefs: []TargetRef{ingress("ledger")}})
		res, err := Build([]*Binding{b}, map[string]API{ledgerProto: grpcAPI([]byte("x"))}, "")
		if err != nil {
			t.Fatalf("externalPrefix %q: %v", prefix, err)
		}
		if got := res.RouteTable.Routes[0].Operations[0].Path; got != "/payments.ledger.v1.Ledger/GetEntry" {
			t.Errorf("externalPrefix %q: path = %s", prefix, got)
		}
	}
	b := binding("payments", BindingAPI{Ref: ledgerProto, Version: "v1.0.0", ExternalPrefix: "/api", TargetRefs: []TargetRef{ingress("ledger")}})
	_, err := Build([]*Binding{b}, map[string]API{ledgerProto: grpcAPI([]byte("x"))}, "")
	if err == nil || !strings.Contains(err.Error(), `externalPrefix "/api"`) || !strings.Contains(err.Error(), "gRPC") {
		t.Errorf("externalPrefix /api on a gRPC API: err = %v", err)
	}
}

func TestDescriptorConfigMap(t *testing.T) {
	set := bytes.Repeat([]byte("descriptor bytes "), 100)
	b := binding("payments", BindingAPI{Ref: ledgerProto, Version: "v1.2.0", TargetRefs: []TargetRef{ingress("ledger")}})
	res, err := Build([]*Binding{b}, map[string]API{ledgerProto: grpcAPI(set)}, "")
	if err != nil {
		t.Fatal(err)
	}
	table := &res.Tables[0]
	manifest, err := table.DescriptorConfigMap(set, "payments")
	if err != nil {
		t.Fatal(err)
	}
	var cm struct {
		APIVersion string `yaml:"apiVersion"`
		Kind       string `yaml:"kind"`
		Metadata   struct {
			Name        string            `yaml:"name"`
			Namespace   string            `yaml:"namespace"`
			Labels      map[string]string `yaml:"labels"`
			Annotations map[string]string `yaml:"annotations"`
		} `yaml:"metadata"`
		Data       map[string]string `yaml:"data"`
		BinaryData map[string]string `yaml:"binaryData"`
	}
	if err := yaml.Unmarshal(manifest, &cm); err != nil {
		t.Fatal(err)
	}
	if cm.APIVersion != "v1" || cm.Kind != "ConfigMap" || cm.Metadata.Name != "optable-4def501dd8-descriptors" || cm.Metadata.Namespace != "payments" {
		t.Errorf("manifest = %+v", cm)
	}
	if cm.Metadata.Labels[LabelDescriptorSet] != "true" || len(cm.Metadata.Labels) != 1 {
		t.Errorf("labels = %v", cm.Metadata.Labels)
	}
	wantAnn := map[string]string{AnnotationAPI: ledgerProto, AnnotationAPIVersion: "v1.2.0", AnnotationDescriptorDigest: table.Descriptor.Digest}
	for k, v := range wantAnn {
		if cm.Metadata.Annotations[k] != v {
			t.Errorf("annotation %s = %q, want %q", k, cm.Metadata.Annotations[k], v)
		}
	}
	if len(cm.Data) != 0 || len(cm.BinaryData) != 1 {
		t.Errorf("data = %v, binaryData keys = %d", cm.Data, len(cm.BinaryData))
	}
	zipped, err := base64.StdEncoding.DecodeString(cm.BinaryData[DescriptorDataKey])
	if err != nil {
		t.Fatal(err)
	}
	got, err := Gunzip(zipped)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, set) {
		t.Error("the gzip data is not the descriptor set")
	}
	again, err := table.DescriptorConfigMap(set, "payments")
	if err != nil || !bytes.Equal(again, manifest) {
		t.Errorf("a second encode gives other bytes (err %v)", err)
	}
	noNamespace, err := table.DescriptorConfigMap(set, "")
	if err != nil || bytes.Contains(noNamespace, []byte("namespace:")) {
		t.Errorf("no namespace: err %v\n%s", err, noNamespace)
	}

	// The set must be the one that the table names.
	if _, err := table.DescriptorConfigMap([]byte("another set"), ""); err == nil || !strings.Contains(err.Error(), "digest") {
		t.Errorf("another set: err = %v", err)
	}
	// An OpenAPI table has no descriptor.
	open := &Table{API: "openapi/a/v1", Version: "v1.0.0"}
	if _, err := open.DescriptorConfigMap(set, ""); err == nil {
		t.Error("an OpenAPI table gives a descriptor ConfigMap")
	}
}

func TestDescriptorConfigMapSizeLimit(t *testing.T) {
	// Random bytes do not compress: 800 KiB stay above the limit after gzip.
	big := make([]byte, 800*1024)
	rand.New(rand.NewSource(1)).Read(big)
	b := binding("payments", BindingAPI{Ref: ledgerProto, Version: "v1.0.0", TargetRefs: []TargetRef{ingress("ledger")}})
	res, err := Build([]*Binding{b}, map[string]API{ledgerProto: grpcAPI(big)}, "")
	if err != nil {
		t.Fatal(err)
	}
	_, err = res.Tables[0].DescriptorConfigMap(big, "")
	if err == nil || !strings.Contains(err.Error(), "KiB") || !strings.Contains(err.Error(), ledgerProto) {
		t.Errorf("an oversize set: err = %v", err)
	}
}
