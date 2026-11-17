// Package protoset reads the descriptor set of a gRPC API and gives its
// operations in the classified form of package opclass.
//
// # Input
//
// The input is the output of `buf build -o <file>`: a Buf image, in the binary
// form or in the JSON form. A Buf image is wire compatible with a
// google.protobuf.FileDescriptorSet, and it marks each file that the API
// imports (the file extension buf.alpha.image.v1.ImageFileExtension, field
// 8042, with is_import). A plain FileDescriptorSet, for example from
// `protoc --include_imports --descriptor_set_out`, is also accepted; it has no
// marks.
//
// # Operations
//
// Each method of each service of a file that is not an import is one
// operation. A plain set has no marks, so each of its files counts.
//
//   - Method is POST: each gRPC call is an HTTP/2 POST.
//   - Path is /<package>.<Service>/<Method>, the path that a gRPC client sends.
//   - OperationID is <Service>_<Method>, the form that a gRPC gateway gives to
//     the same method in OpenAPI. opclass reads that form.
//   - Class comes from opclass.Classify with the method, the path and the
//     operationId. A name with no verb word falls to the shape rule, and POST
//     gives write.
//   - Streaming is true when the client or the server streams.
//
// # Canonical set
//
// The set must be complete: each import of each file must be in it, so that a
// reader can decode each request and response message. Set.Canonical is the
// set that a reader gets: a binary FileDescriptorSet with the files sorted by
// name, without source code info and without the Buf marks. Other unknown
// fields, such as option extensions that this binary does not know, stay.
// Set.Digest is the SHA-256 of Set.Canonical.
package protoset

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/infobloxopen/apx/internal/opclass"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

// bufImageExtensionField is the field number of ImageFile.buf_extension in a
// Buf image (buf.alpha.image.v1). Its message has is_import as field 1.
const bufImageExtensionField protowire.Number = 8042

// ErrNoServices is the error, wrapped, of a descriptor set whose files (other
// than the imports) declare no service.
var ErrNoServices = errors.New("the descriptor set has no service")

// Set is one parsed descriptor set.
type Set struct {
	// Document holds the operations. BasePath is empty: a gRPC path has no
	// prefix.
	Document *opclass.Document
	// Canonical is the binary FileDescriptorSet that a reader gets.
	Canonical []byte
	// Digest is "sha256:" and the hex SHA-256 of Canonical.
	Digest string
}

// Load reads the descriptor set at path. A file that ends in .json (any case)
// is the JSON form; any other file is the binary form.
func Load(path string) (*Set, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading the descriptor set: %w", err)
	}
	set, err := Parse(data, strings.EqualFold(filepath.Ext(path), ".json"))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", filepath.ToSlash(path), err)
	}
	return set, nil
}

// Parse reads a descriptor set in the binary form, or in the JSON form when
// jsonForm is true.
func Parse(data []byte, jsonForm bool) (*Set, error) {
	var (
		fds     *descriptorpb.FileDescriptorSet
		imports map[string]bool
		err     error
	)
	if jsonForm {
		fds, imports, err = decodeJSON(data)
	} else {
		fds, imports, err = decodeBinary(data)
	}
	if err != nil {
		return nil, err
	}
	if len(fds.File) == 0 {
		return nil, errors.New("the descriptor set has no file")
	}
	files, err := protodesc.NewFiles(fds)
	if err != nil {
		return nil, fmt.Errorf("the descriptor set is not complete or not valid: %w", err)
	}
	doc, err := operations(fds, files, imports)
	if err != nil {
		return nil, err
	}
	canonical, err := canonicalize(fds)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(canonical)
	return &Set{Document: doc, Canonical: canonical, Digest: "sha256:" + hex.EncodeToString(sum[:])}, nil
}

// decodeBinary decodes the binary form. imports is nil when no file has a Buf
// mark (a plain set).
func decodeBinary(data []byte) (*descriptorpb.FileDescriptorSet, map[string]bool, error) {
	fds := &descriptorpb.FileDescriptorSet{}
	if err := proto.Unmarshal(data, fds); err != nil {
		return nil, nil, fmt.Errorf("decoding the descriptor set: %w", err)
	}
	var imports map[string]bool
	for _, f := range fds.File {
		isImport, marked, err := bufMark(f.ProtoReflect().GetUnknown())
		if err != nil {
			return nil, nil, fmt.Errorf("decoding the descriptor set: file %s: %w", f.GetName(), err)
		}
		if marked {
			if imports == nil {
				imports = map[string]bool{}
			}
			imports[f.GetName()] = isImport
		}
	}
	return fds, imports, nil
}

// bufMark reads the Buf image extension from the unknown fields of a file.
func bufMark(unknown []byte) (isImport, marked bool, err error) {
	for len(unknown) > 0 {
		num, typ, n := protowire.ConsumeTag(unknown)
		if n < 0 {
			return false, false, protowire.ParseError(n)
		}
		unknown = unknown[n:]
		if num == bufImageExtensionField && typ == protowire.BytesType {
			inner, m := protowire.ConsumeBytes(unknown)
			if m < 0 {
				return false, false, protowire.ParseError(m)
			}
			isImport, err = bufIsImport(inner)
			if err != nil {
				return false, false, err
			}
			marked = true
			unknown = unknown[m:]
			continue
		}
		m := protowire.ConsumeFieldValue(num, typ, unknown)
		if m < 0 {
			return false, false, protowire.ParseError(m)
		}
		unknown = unknown[m:]
	}
	return isImport, marked, nil
}

func bufIsImport(msg []byte) (bool, error) {
	isImport := false
	for len(msg) > 0 {
		num, typ, n := protowire.ConsumeTag(msg)
		if n < 0 {
			return false, protowire.ParseError(n)
		}
		msg = msg[n:]
		if num == 1 && typ == protowire.VarintType {
			v, m := protowire.ConsumeVarint(msg)
			if m < 0 {
				return false, protowire.ParseError(m)
			}
			isImport = protowire.DecodeBool(v)
			msg = msg[m:]
			continue
		}
		m := protowire.ConsumeFieldValue(num, typ, msg)
		if m < 0 {
			return false, protowire.ParseError(m)
		}
		msg = msg[m:]
	}
	return isImport, nil
}

// jsonMarks is the part of the JSON form of a Buf image that has the marks.
// protojson accepts the JSON name and the proto name of a field, so both are
// read.
type jsonMarks struct {
	File []struct {
		Name     string    `json:"name"`
		Ext      *jsonMark `json:"bufExtension"`
		ExtSnake *jsonMark `json:"buf_extension"`
	} `json:"file"`
}

type jsonMark struct {
	IsImport      *bool `json:"isImport"`
	IsImportSnake *bool `json:"is_import"`
}

// decodeJSON decodes the JSON form. The Buf marks are not fields of
// FileDescriptorProto, so protojson drops them, and a second pass reads them.
func decodeJSON(data []byte) (*descriptorpb.FileDescriptorSet, map[string]bool, error) {
	fds := &descriptorpb.FileDescriptorSet{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(data, fds); err != nil {
		return nil, nil, fmt.Errorf("decoding the descriptor set: %w", err)
	}
	var marks jsonMarks
	if err := json.Unmarshal(data, &marks); err != nil {
		return nil, nil, fmt.Errorf("decoding the descriptor set: %w", err)
	}
	var imports map[string]bool
	for _, f := range marks.File {
		m := f.Ext
		if m == nil {
			m = f.ExtSnake
		}
		if m == nil {
			continue
		}
		if imports == nil {
			imports = map[string]bool{}
		}
		v := m.IsImport
		if v == nil {
			v = m.IsImportSnake
		}
		imports[f.Name] = v != nil && *v
	}
	return fds, imports, nil
}

// operations lists the methods of the files that are not imports.
func operations(fds *descriptorpb.FileDescriptorSet, files *protoregistry.Files, imports map[string]bool) (*opclass.Document, error) {
	doc := &opclass.Document{Operations: []opclass.Operation{}}
	for _, f := range fds.File {
		if imports != nil && imports[f.GetName()] {
			continue
		}
		fd, err := files.FindFileByPath(f.GetName())
		if err != nil {
			return nil, fmt.Errorf("file %s: %w", f.GetName(), err)
		}
		services := fd.Services()
		for i := 0; i < services.Len(); i++ {
			s := services.Get(i)
			methods := s.Methods()
			for j := 0; j < methods.Len(); j++ {
				doc.Operations = append(doc.Operations, operation(s, methods.Get(j)))
			}
		}
	}
	if len(doc.Operations) == 0 {
		return nil, ErrNoServices
	}
	sort.SliceStable(doc.Operations, func(i, j int) bool {
		a, b := doc.Operations[i], doc.Operations[j]
		if a.Path != b.Path {
			return a.Path < b.Path
		}
		return a.Method < b.Method
	})
	return doc, nil
}

func operation(s protoreflect.ServiceDescriptor, m protoreflect.MethodDescriptor) opclass.Operation {
	op := opclass.Operation{
		Method:      "POST",
		Path:        "/" + string(s.FullName()) + "/" + string(m.Name()),
		OperationID: string(s.Name()) + "_" + string(m.Name()),
		Streaming:   m.IsStreamingClient() || m.IsStreamingServer(),
	}
	// Classify fails only on an explicit class, and a gRPC method has none.
	op.Class, op.Rule, _ = opclass.Classify(opclass.Input{Method: op.Method, Path: op.Path, OperationID: op.OperationID})
	return op
}

// canonicalize returns the deterministic binary form of the set that a reader
// gets: files sorted by name, no source code info, no Buf mark.
func canonicalize(fds *descriptorpb.FileDescriptorSet) ([]byte, error) {
	out := &descriptorpb.FileDescriptorSet{}
	for _, f := range fds.File {
		c := proto.Clone(f).(*descriptorpb.FileDescriptorProto)
		c.SourceCodeInfo = nil
		r := c.ProtoReflect()
		unknown, err := withoutField(r.GetUnknown(), bufImageExtensionField)
		if err != nil {
			return nil, fmt.Errorf("file %s: %w", f.GetName(), err)
		}
		r.SetUnknown(unknown)
		out.File = append(out.File, c)
	}
	sort.Slice(out.File, func(i, j int) bool { return out.File[i].GetName() < out.File[j].GetName() })
	data, err := proto.MarshalOptions{Deterministic: true}.Marshal(out)
	if err != nil {
		return nil, fmt.Errorf("encoding the canonical descriptor set: %w", err)
	}
	return data, nil
}

// withoutField returns the unknown fields without each occurrence of num.
func withoutField(unknown []byte, num protowire.Number) (protoreflect.RawFields, error) {
	var out []byte
	for len(unknown) > 0 {
		n, typ, tagLen := protowire.ConsumeTag(unknown)
		if tagLen < 0 {
			return nil, protowire.ParseError(tagLen)
		}
		valLen := protowire.ConsumeFieldValue(n, typ, unknown[tagLen:])
		if valLen < 0 {
			return nil, protowire.ParseError(valLen)
		}
		if n != num {
			out = append(out, unknown[:tagLen+valLen]...)
		}
		unknown = unknown[tagLen+valLen:]
	}
	return out, nil
}
