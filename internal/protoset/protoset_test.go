package protoset

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/infobloxopen/apx/internal/opclass"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ledgerFile is a synthetic API file. It imports google/protobuf/timestamp.proto
// and has one service with a unary, a server-streaming, a client-streaming and
// a bidirectional method.
func ledgerFile() *descriptorpb.FileDescriptorProto {
	msg := func(name string, fields ...*descriptorpb.FieldDescriptorProto) *descriptorpb.DescriptorProto {
		return &descriptorpb.DescriptorProto{Name: proto.String(name), Field: fields}
	}
	field := func(name string, num int32, typ descriptorpb.FieldDescriptorProto_Type, typeName string) *descriptorpb.FieldDescriptorProto {
		f := &descriptorpb.FieldDescriptorProto{
			Name: proto.String(name), Number: proto.Int32(num), Type: typ.Enum(),
			Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), JsonName: proto.String(name),
		}
		if typeName != "" {
			f.TypeName = proto.String(typeName)
		}
		return f
	}
	method := func(name, in, out string, clientStream, serverStream bool) *descriptorpb.MethodDescriptorProto {
		m := &descriptorpb.MethodDescriptorProto{
			Name: proto.String(name), InputType: proto.String(in), OutputType: proto.String(out),
		}
		if clientStream {
			m.ClientStreaming = proto.Bool(true)
		}
		if serverStream {
			m.ServerStreaming = proto.Bool(true)
		}
		return m
	}
	const pkg = ".acme.ledger.v1."
	return &descriptorpb.FileDescriptorProto{
		Name:       proto.String("acme/ledger/v1/ledger.proto"),
		Package:    proto.String("acme.ledger.v1"),
		Syntax:     proto.String("proto3"),
		Dependency: []string{"google/protobuf/timestamp.proto"},
		MessageType: []*descriptorpb.DescriptorProto{
			msg("Entry", field("id", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING, ""),
				field("posted_at", 2, descriptorpb.FieldDescriptorProto_TYPE_MESSAGE, ".google.protobuf.Timestamp")),
			msg("GetEntryRequest", field("id", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING, "")),
			msg("ListEntriesRequest"),
			msg("CreateEntryRequest", field("entry", 1, descriptorpb.FieldDescriptorProto_TYPE_MESSAGE, pkg+"Entry")),
		},
		Service: []*descriptorpb.ServiceDescriptorProto{{
			Name: proto.String("Ledger"),
			Method: []*descriptorpb.MethodDescriptorProto{
				method("GetEntry", pkg+"GetEntryRequest", pkg+"Entry", false, false),
				method("ListEntries", pkg+"ListEntriesRequest", pkg+"Entry", false, true),
				method("CreateEntry", pkg+"CreateEntryRequest", pkg+"Entry", false, false),
				method("UploadEntries", pkg+"Entry", pkg+"Entry", true, false),
				method("SyncEntries", pkg+"Entry", pkg+"Entry", true, true),
			},
		}},
		SourceCodeInfo: &descriptorpb.SourceCodeInfo{Location: []*descriptorpb.SourceCodeInfo_Location{{
			Path: []int32{4, 0}, Span: []int32{1, 0, 9}, LeadingComments: proto.String(" An entry.\n"),
		}}},
	}
}

// importedServiceFile is a synthetic dependency with a service of its own, as
// google/longrunning/operations.proto is for a real API.
func importedServiceFile() *descriptorpb.FileDescriptorProto {
	return &descriptorpb.FileDescriptorProto{
		Name:        proto.String("acme/ops/v1/ops.proto"),
		Package:     proto.String("acme.ops.v1"),
		Syntax:      proto.String("proto3"),
		MessageType: []*descriptorpb.DescriptorProto{{Name: proto.String("Op")}},
		Service: []*descriptorpb.ServiceDescriptorProto{{
			Name: proto.String("Operations"),
			Method: []*descriptorpb.MethodDescriptorProto{{
				Name: proto.String("GetOp"), InputType: proto.String(".acme.ops.v1.Op"), OutputType: proto.String(".acme.ops.v1.Op"),
			}},
		}},
	}
}

func timestampFile() *descriptorpb.FileDescriptorProto {
	return protodesc.ToFileDescriptorProto(timestamppb.File_google_protobuf_timestamp_proto)
}

// markImport adds the Buf image extension (field 8042, ImageFileExtension with
// is_import, field 1) to f, as `buf build -o x.binpb` writes it.
func markImport(f *descriptorpb.FileDescriptorProto, isImport bool) {
	var inner []byte
	inner = protowire.AppendTag(inner, 1, protowire.VarintType)
	inner = protowire.AppendVarint(inner, protowire.EncodeBool(isImport))
	var ext []byte
	ext = protowire.AppendTag(ext, bufImageExtensionField, protowire.BytesType)
	ext = protowire.AppendBytes(ext, inner)
	r := f.ProtoReflect()
	r.SetUnknown(append(r.GetUnknown(), ext...))
}

func marshalSet(t *testing.T, files ...*descriptorpb.FileDescriptorProto) []byte {
	t.Helper()
	data, err := proto.Marshal(&descriptorpb.FileDescriptorSet{File: files})
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func paths(doc *opclass.Document) []string {
	out := make([]string, len(doc.Operations))
	for i, op := range doc.Operations {
		out[i] = op.Method + " " + op.Path
	}
	return out
}

func TestParseOperations(t *testing.T) {
	set, err := Parse(marshalSet(t, timestampFile(), ledgerFile()), false)
	if err != nil {
		t.Fatal(err)
	}
	if set.Document.BasePath != "" {
		t.Errorf("BasePath = %q, want empty", set.Document.BasePath)
	}
	want := []opclass.Operation{
		{Method: "POST", Path: "/acme.ledger.v1.Ledger/CreateEntry", OperationID: "Ledger_CreateEntry", Class: opclass.Write, Rule: opclass.RuleName},
		{Method: "POST", Path: "/acme.ledger.v1.Ledger/GetEntry", OperationID: "Ledger_GetEntry", Class: opclass.Read, Rule: opclass.RuleName},
		{Method: "POST", Path: "/acme.ledger.v1.Ledger/ListEntries", OperationID: "Ledger_ListEntries", Class: opclass.List, Streaming: true, Rule: opclass.RuleName},
		{Method: "POST", Path: "/acme.ledger.v1.Ledger/SyncEntries", OperationID: "Ledger_SyncEntries", Class: opclass.Write, Streaming: true, Rule: opclass.RuleShape},
		{Method: "POST", Path: "/acme.ledger.v1.Ledger/UploadEntries", OperationID: "Ledger_UploadEntries", Class: opclass.Write, Streaming: true, Rule: opclass.RuleShape},
	}
	if len(set.Document.Operations) != len(want) {
		t.Fatalf("operations = %v, want %d", paths(set.Document), len(want))
	}
	for i, op := range set.Document.Operations {
		if op != want[i] {
			t.Errorf("operation %d = %+v, want %+v", i, op, want[i])
		}
	}
}

func TestBufImportMarks(t *testing.T) {
	ops, ledger, ts := importedServiceFile(), ledgerFile(), timestampFile()
	ledger.Dependency = append(ledger.Dependency, "acme/ops/v1/ops.proto")

	// A plain set has no marks: each service gives operations.
	plain, err := Parse(marshalSet(t, ts, ops, ledger), false)
	if err != nil {
		t.Fatal(err)
	}
	if got := paths(plain.Document); len(got) != 6 || got[0] != "POST /acme.ledger.v1.Ledger/CreateEntry" || got[5] != "POST /acme.ops.v1.Operations/GetOp" {
		t.Errorf("plain set operations = %v", got)
	}

	// A Buf image marks the imports: only the API's own files give operations.
	markImport(ts, true)
	markImport(ops, true)
	markImport(ledger, false)
	image, err := Parse(marshalSet(t, ts, ops, ledger), false)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range paths(image.Document) {
		if strings.Contains(p, "acme.ops.v1") {
			t.Errorf("an imported service gives an operation: %s", p)
		}
	}
	if len(image.Document.Operations) != 5 {
		t.Errorf("image operations = %v, want the 5 Ledger methods", paths(image.Document))
	}
	// The marks are not in the canonical set, so a reader gets a plain set.
	var canon descriptorpb.FileDescriptorSet
	if err := proto.Unmarshal(image.Canonical, &canon); err != nil {
		t.Fatal(err)
	}
	for _, f := range canon.File {
		if len(f.ProtoReflect().GetUnknown()) != 0 {
			t.Errorf("file %s keeps unknown fields in the canonical set", f.GetName())
		}
	}
	if len(canon.File) != 3 {
		t.Errorf("canonical set has %d files, want 3 (the imports stay: a reader decodes their messages)", len(canon.File))
	}
}

func TestCanonicalSet(t *testing.T) {
	a, err := Parse(marshalSet(t, timestampFile(), ledgerFile()), false)
	if err != nil {
		t.Fatal(err)
	}
	// Another file order and a Buf mark give the same canonical bytes and digest.
	ts, ledger := timestampFile(), ledgerFile()
	markImport(ts, true)
	markImport(ledger, false)
	b, err := Parse(marshalSet(t, ledger, ts), false)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a.Canonical, b.Canonical) || a.Digest != b.Digest {
		t.Fatal("the canonical set depends on the file order or on the Buf marks")
	}
	sum := sha256.Sum256(a.Canonical)
	if a.Digest != "sha256:"+hex.EncodeToString(sum[:]) {
		t.Errorf("Digest = %s, want the SHA-256 of Canonical", a.Digest)
	}
	var canon descriptorpb.FileDescriptorSet
	if err := proto.Unmarshal(a.Canonical, &canon); err != nil {
		t.Fatal(err)
	}
	if canon.File[0].GetName() != "acme/ledger/v1/ledger.proto" || canon.File[1].GetName() != "google/protobuf/timestamp.proto" {
		t.Errorf("canonical files are not sorted by name: %s, %s", canon.File[0].GetName(), canon.File[1].GetName())
	}
	for _, f := range canon.File {
		if f.SourceCodeInfo != nil {
			t.Errorf("file %s keeps its source code info", f.GetName())
		}
	}
	// A reader can build the files from the canonical set and find each message type.
	files, err := protodesc.NewFiles(&canon)
	if err != nil {
		t.Fatalf("the canonical set does not build: %v", err)
	}
	if _, err := files.FindDescriptorByName("acme.ledger.v1.CreateEntryRequest"); err != nil {
		t.Errorf("the request type of CreateEntry is not in the set: %v", err)
	}
}

func TestCustomOptionsStay(t *testing.T) {
	// An option extension that the reader does not know (google.api.http, for
	// example) is an unknown field of the options message. It must stay.
	ledger := ledgerFile()
	var opt []byte
	opt = protowire.AppendTag(opt, 72295728, protowire.BytesType) // the field number of google.api.http
	opt = protowire.AppendBytes(opt, []byte("synthetic"))
	m := ledger.Service[0].Method[0]
	m.Options = &descriptorpb.MethodOptions{}
	m.Options.ProtoReflect().SetUnknown(opt)
	set, err := Parse(marshalSet(t, timestampFile(), ledger), false)
	if err != nil {
		t.Fatal(err)
	}
	var canon descriptorpb.FileDescriptorSet
	if err := proto.Unmarshal(set.Canonical, &canon); err != nil {
		t.Fatal(err)
	}
	got := canon.File[0].Service[0].Method[0].GetOptions().ProtoReflect().GetUnknown()
	if !bytes.Equal(got, opt) {
		t.Errorf("the custom method option is lost: %x", got)
	}
}

func TestJSONForm(t *testing.T) {
	binary, err := Parse(marshalSet(t, timestampFile(), ledgerFile()), false)
	if err != nil {
		t.Fatal(err)
	}
	ops := importedServiceFile()
	ledger := ledgerFile()
	ledger.Dependency = append(ledger.Dependency, "acme/ops/v1/ops.proto")
	data, err := protojson.Marshal(&descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{timestampFile(), ops, ledger}})
	if err != nil {
		t.Fatal(err)
	}
	// The JSON form of a Buf image has bufExtension on each file.
	text := strings.Replace(string(data), `"name":"acme/ops/v1/ops.proto"`, `"name":"acme/ops/v1/ops.proto","bufExtension":{"isImport":true}`, 1)
	text = strings.Replace(text, `"name":"google/protobuf/timestamp.proto"`, `"name":"google/protobuf/timestamp.proto","bufExtension":{"isImport":true}`, 1)
	if !strings.Contains(text, "bufExtension") {
		t.Fatal("the test did not add the Buf marks")
	}
	set, err := Parse([]byte(text), true)
	if err != nil {
		t.Fatal(err)
	}
	if got := paths(set.Document); len(got) != 5 {
		t.Errorf("JSON image operations = %v, want the 5 Ledger methods", got)
	}
	if strings.Join(paths(set.Document), ",") != strings.Join(paths(binary.Document), ",") {
		t.Error("the JSON form and the binary form give other operations")
	}
}

func TestParseErrors(t *testing.T) {
	noService := ledgerFile()
	noService.Service = nil
	cases := []struct {
		name string
		data []byte
		json bool
		want string
	}{
		{"not a descriptor set", []byte("not protobuf \xff\xff"), false, "decoding the descriptor set"},
		{"not JSON", []byte("{"), true, "decoding the descriptor set"},
		{"empty", nil, false, "no file"},
		{"missing import", marshalSet(t, ledgerFile()), false, "not complete"},
		{"no service", marshalSet(t, timestampFile(), noService), false, ErrNoServices.Error()},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := Parse(c.data, c.json)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("err = %v, want it to contain %q", err, c.want)
			}
		})
	}
	if _, err := Parse(marshalSet(t, timestampFile(), noService), false); !errors.Is(err, ErrNoServices) {
		t.Errorf("a set with no service does not wrap ErrNoServices: %v", err)
	}
}

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "ledger.binpb")
	if err := os.WriteFile(bin, marshalSet(t, timestampFile(), ledgerFile()), 0o644); err != nil {
		t.Fatal(err)
	}
	data, err := protojson.Marshal(&descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{timestampFile(), ledgerFile()}})
	if err != nil {
		t.Fatal(err)
	}
	js := filepath.Join(dir, "ledger.JSON")
	if err := os.WriteFile(js, data, 0o644); err != nil {
		t.Fatal(err)
	}
	a, err := Load(bin)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Load(js)
	if err != nil {
		t.Fatal(err)
	}
	if a.Digest != b.Digest {
		t.Error("the binary file and the JSON file of one set give two digests")
	}
	if _, err := Load(filepath.Join(dir, "missing.binpb")); err == nil || !strings.Contains(err.Error(), "reading the descriptor set") {
		t.Errorf("a missing file: err = %v", err)
	}
	if _, err := Load(js + ".bad"); err == nil {
		t.Error("a missing JSON file gives no error")
	}
}
