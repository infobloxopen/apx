package schema

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractProto_BasicFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ledger.proto")
	os.WriteFile(path, []byte(`syntax = "proto3";

package payments.ledger.v1;

import "google/protobuf/timestamp.proto";

option go_package = "github.com/acme/apis/proto/payments/ledger/v1;ledgerpb";

// LedgerService manages financial ledger entries.
service LedgerService {
  // CreateEntry creates a new ledger entry.
  rpc CreateEntry(CreateEntryRequest) returns (CreateEntryResponse);

  // ListEntries lists ledger entries with pagination.
  rpc ListEntries(ListEntriesRequest) returns (ListEntriesResponse);
}

// Entry represents a single ledger entry.
message Entry {
  string id = 1;
  string account_id = 2; // owning account
  int64 amount_cents = 3;
  string currency = 4;
  google.protobuf.Timestamp created_at = 5;
}

// CreateEntryRequest is the request to create an entry.
message CreateEntryRequest {
  string account_id = 1;
  int64 amount_cents = 2;
  string currency = 3;
}

message CreateEntryResponse {
  Entry entry = 1;
}

message ListEntriesRequest {
  string account_id = 1;
  int32 page_size = 2;
  string page_token = 3;
}

message ListEntriesResponse {
  repeated Entry entries = 1;
  string next_page_token = 2;
}
`), 0o644)

	proto, err := ExtractProto(path)
	require.NoError(t, err)

	assert.Equal(t, "proto3", proto.Syntax)
	assert.Equal(t, "payments.ledger.v1", proto.Package)
	require.Len(t, proto.Imports, 1)
	assert.Equal(t, "google/protobuf/timestamp.proto", proto.Imports[0])

	require.Len(t, proto.Options, 1)
	assert.Equal(t, "go_package", proto.Options[0].Name)

	// Service.
	require.Len(t, proto.Services, 1)
	svc := proto.Services[0]
	assert.Equal(t, "LedgerService", svc.Name)
	assert.Contains(t, svc.Comment, "manages financial ledger entries")
	require.Len(t, svc.Methods, 2)
	assert.Equal(t, "CreateEntry", svc.Methods[0].Name)
	assert.Equal(t, "CreateEntryRequest", svc.Methods[0].InputType)
	assert.Equal(t, "CreateEntryResponse", svc.Methods[0].OutputType)
	assert.Contains(t, svc.Methods[0].Comment, "creates a new ledger entry")

	// Messages.
	require.Len(t, proto.Messages, 5)
	entry := proto.Messages[0]
	assert.Equal(t, "Entry", entry.Name)
	assert.Contains(t, entry.Comment, "single ledger entry")
	require.Len(t, entry.Fields, 5)
	assert.Equal(t, "id", entry.Fields[0].Name)
	assert.Equal(t, 1, entry.Fields[0].Number)
	assert.Equal(t, "string", entry.Fields[0].Type)

	// Check repeated field.
	listResp := proto.Messages[4]
	assert.Equal(t, "ListEntriesResponse", listResp.Name)
	assert.Equal(t, "repeated", listResp.Fields[0].Label)
}

func TestExtractProto_StreamingRPCs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stream.proto")
	os.WriteFile(path, []byte(`syntax = "proto3";
package streaming.v1;

service StreamService {
  rpc ServerStream(Request) returns (stream Response);
  rpc ClientStream(stream Request) returns (Response);
  rpc BiDiStream(stream Request) returns (stream Response);
}

message Request { string id = 1; }
message Response { string data = 1; }
`), 0o644)

	proto, err := ExtractProto(path)
	require.NoError(t, err)

	require.Len(t, proto.Services, 1)
	methods := proto.Services[0].Methods
	require.Len(t, methods, 3)

	assert.Equal(t, "ServerStream", methods[0].Name)
	assert.False(t, methods[0].ClientStreaming)
	assert.True(t, methods[0].ServerStreaming)

	assert.Equal(t, "ClientStream", methods[1].Name)
	assert.True(t, methods[1].ClientStreaming)
	assert.False(t, methods[1].ServerStreaming)

	assert.Equal(t, "BiDiStream", methods[2].Name)
	assert.True(t, methods[2].ClientStreaming)
	assert.True(t, methods[2].ServerStreaming)
}

func TestExtractProto_NestedMessage(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested.proto")
	os.WriteFile(path, []byte(`syntax = "proto3";
package test.v1;

message Outer {
  string id = 1;

  message Inner {
    string value = 1;
  }

  Inner nested = 2;
}
`), 0o644)

	proto, err := ExtractProto(path)
	require.NoError(t, err)

	require.Len(t, proto.Messages, 1)
	outer := proto.Messages[0]
	assert.Equal(t, "Outer", outer.Name)
	require.Len(t, outer.Nested, 1)
	assert.Equal(t, "Inner", outer.Nested[0].Name)
	require.Len(t, outer.Nested[0].Fields, 1)
	assert.Equal(t, "value", outer.Nested[0].Fields[0].Name)
}

func TestExtractProto_Enum(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "enums.proto")
	os.WriteFile(path, []byte(`syntax = "proto3";
package test.v1;

// Status represents processing state.
enum Status {
  STATUS_UNKNOWN = 0;
  STATUS_PENDING = 1;
  STATUS_ACTIVE = 2;
  STATUS_CLOSED = 3;
}
`), 0o644)

	proto, err := ExtractProto(path)
	require.NoError(t, err)

	require.Len(t, proto.Enums, 1)
	enum := proto.Enums[0]
	assert.Equal(t, "Status", enum.Name)
	assert.Contains(t, enum.Comment, "processing state")
	require.Len(t, enum.Values, 4)
	assert.Equal(t, "STATUS_UNKNOWN", enum.Values[0].Name)
	assert.Equal(t, 0, enum.Values[0].Number)
	assert.Equal(t, "STATUS_CLOSED", enum.Values[3].Name)
	assert.Equal(t, 3, enum.Values[3].Number)
}

func TestExtractProto_MapField(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "maps.proto")
	os.WriteFile(path, []byte(`syntax = "proto3";
package test.v1;

message Config {
  map<string, string> labels = 1;
  string name = 2;
}
`), 0o644)

	proto, err := ExtractProto(path)
	require.NoError(t, err)

	require.Len(t, proto.Messages, 1)
	fields := proto.Messages[0].Fields
	require.Len(t, fields, 2)
	assert.Equal(t, "labels", fields[0].Name)
	assert.Equal(t, "map<string, string>", fields[0].Type)
	assert.Equal(t, "map", fields[0].Label)
}

func TestExtractProto_InlineComment(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "inline.proto")
	os.WriteFile(path, []byte(`syntax = "proto3";
package test.v1;

message Thing {
  string id = 1; // unique identifier
  string name = 2; // display name
}
`), 0o644)

	proto, err := ExtractProto(path)
	require.NoError(t, err)

	require.Len(t, proto.Messages, 1)
	fields := proto.Messages[0].Fields
	require.Len(t, fields, 2)
	assert.Equal(t, "unique identifier", fields[0].Comment)
	assert.Equal(t, "display name", fields[1].Comment)
}

func TestExtractProto_SingleLineMessage(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "single.proto")
	os.WriteFile(path, []byte(`syntax = "proto3";
package test.v1;

service TestService {
  rpc Get(GetRequest) returns (GetResponse);
}

message GetRequest { string id = 1; }
message GetResponse { string name = 1; }
`), 0o644)

	proto, err := ExtractProto(path)
	require.NoError(t, err)

	require.Len(t, proto.Messages, 2)
	assert.Equal(t, "GetRequest", proto.Messages[0].Name)
	require.Len(t, proto.Messages[0].Fields, 1)
	assert.Equal(t, "id", proto.Messages[0].Fields[0].Name)
	assert.Equal(t, "string", proto.Messages[0].Fields[0].Type)
	assert.Equal(t, 1, proto.Messages[0].Fields[0].Number)

	assert.Equal(t, "GetResponse", proto.Messages[1].Name)
	require.Len(t, proto.Messages[1].Fields, 1)
	assert.Equal(t, "name", proto.Messages[1].Fields[0].Name)
}

func TestExtractProto_FileNotFound(t *testing.T) {
	_, err := ExtractProto("/nonexistent/path.proto")
	assert.Error(t, err)
}

func TestExtractProto_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.proto")
	os.WriteFile(path, []byte(``), 0o644)

	proto, err := ExtractProto(path)
	require.NoError(t, err)
	assert.Equal(t, "", proto.Syntax)
	assert.Empty(t, proto.Messages)
}
