# Buf Issues

Troubleshooting guide for Protocol Buffers tooling issues with Buf.

## Installation & Setup

### `buf: command not found`

```
exec: "buf": executable file not found in $PATH
```

**Fix:** Install Buf via APX or manually:

```bash
# Preferred — uses pinned version from apx.lock
apx fetch

# Or install globally
brew install bufbuild/buf/buf

# Verify
buf --version
```

### Wrong Buf version

```
Warning: buf version 1.28.0 does not match pinned version 1.47.2
```

**Fix:** APX pins tool versions in `apx.lock`. Use `apx fetch` to download the correct version, which is cached in `.apx-tools/`.

---

## Lint Failures

### `PACKAGE_VERSION_SUFFIX`

```
proto/payments/ledger/v1/ledger.proto:3:1:
  Package name "payments.ledger" should have a version suffix.
```

**Fix:** Include the version in the proto package:
```protobuf
package payments.ledger.v1;  // ✔
// not:
package payments.ledger;     // ✘
```

### `FIELD_LOWER_SNAKE_CASE`

```
Field name "userName" should be lower_snake_case.
```

**Fix:** Use snake_case for field names:
```protobuf
string user_name = 1;  // ✔
// not:
string userName = 1;   // ✘
```

### `SERVICE_SUFFIX`

```
Service name "Ledger" should have suffix "Service".
```

**Fix:**
```protobuf
service LedgerService { ... }  // ✔
```

### `RPC_REQUEST_RESPONSE_UNIQUE`

```
RPC "GetUser" request type "GetUserRequest" is also used by RPC "ListUsers".
```

**Fix:** Each RPC should have unique request and response types.

### Disabling specific lint rules

Configure exceptions in `buf.yaml`:

```yaml
version: v2
lint:
  use:
    - DEFAULT
  except:
    - FIELD_LOWER_SNAKE_CASE    # if you have legacy fields
```

> **Note:** APX's `apx policy check` may enforce required lint rules that override `buf.yaml` exceptions.

---

## Breaking Change Failures

### `FIELD_NO_DELETE`

```
Previously present field "2" with name "email" on message "User" was deleted.
```

**Fix:** Don't remove fields — mark them as reserved instead:
```protobuf
message User {
  string name = 1;
  reserved 2;              // was: string email = 2;
  reserved "email";
  string email_address = 3;  // new field with new number
}
```

### `FIELD_SAME_TYPE`

```
Field "1" with name "id" on message "User" changed type from "string" to "int64".
```

**Fix:** Field types cannot change. Add a new field with the new type and deprecate the old one.

### `ENUM_VALUE_NO_DELETE`

```
Previously present enum value "2" with name "STATUS_ACTIVE" was deleted.
```

**Fix:** Reserve deleted enum values:
```protobuf
enum Status {
  STATUS_UNSPECIFIED = 0;
  reserved 2;  // was STATUS_ACTIVE
  STATUS_ENABLED = 3;
}
```

### False positive breaking changes

If `apx breaking` reports changes that aren't actually breaking:

```bash
# Check the baseline reference
apx breaking --against origin/main --verbose

# Verify you're comparing against the right baseline
git log --oneline origin/main..HEAD
```

Common causes of false positives:
- Comparing against the wrong baseline branch
- `buf.yaml` version change (v1 vs v2) causes structural differences
- Reordering fields (field numbers didn't change, but file diff triggers detection)

---

## `go_package` Warnings

### `go_package does not match canonical path`

```
Warning: go_package "github.com/myorg/myrepo/gen/proto/payments/ledger/v1"
  does not match canonical path "github.com/acme-corp/apis/proto/payments/ledger/v1"
```

**Cause:** The `go_package` option in your `.proto` files doesn't match the import path that consumers will use after publishing to the canonical repo.

**Fix:** Update the `go_package` option:
```protobuf
option go_package = "github.com/acme-corp/apis/proto/payments/ledger/v1";
```

The canonical path is derived from:
- `canonical_repo` in `apx.yaml` (or `--canonical-repo` flag)
- The API ID path

Check what APX expects:
```bash
apx inspect identity proto/payments/ledger/v1
apx explain go-path proto/payments/ledger/v1
```

---

## `buf.yaml` Configuration

### v1 vs v2 format

Buf v1 and v2 have different `buf.yaml` schemas. APX works with both but prefers v2:

```yaml
# v2 (recommended)
version: v2
lint:
  use:
    - DEFAULT
breaking:
  use:
    - FILE

# v1 (legacy)
version: v1
lint:
  use:
    - DEFAULT
breaking:
  use:
    - FILE
```

### Missing `buf.yaml`

If no `buf.yaml` exists, Buf uses defaults. APX recommends creating one for consistent behavior:

```bash
buf config init
```

### Dependencies in `buf.yaml`

If your protos import well-known types or other Buf modules:

```yaml
version: v2
deps:
  - buf.build/googleapis/googleapis
  - buf.build/grpc/grpc
```

Then run:
```bash
buf dep update
```

---

## Performance

### Slow lint/breaking checks

**Cause:** Buf scans the entire proto tree by default.

**Fix:** Scope checks to a specific path:
```bash
apx lint proto/payments/ledger/v1/
apx breaking proto/payments/ledger/v1/ --against HEAD^
```

---

## Debugging

```bash
# Verbose output from APX (shows buf invocations)
apx lint --verbose

# Run buf directly for deeper debugging
buf lint proto/payments/ledger/v1/ --error-format=json
buf breaking proto/payments/ledger/v1/ --against .git#branch=main

# Check buf configuration
buf config ls-lint-rules
buf config ls-breaking-rules
```

## See Also

- [Validation Commands](../cli-reference/validation-commands.md) — `apx lint`, `apx breaking` reference
- [Code Generation Troubleshooting](code-generation.md) — `buf generate` issues
- [Common Errors](common-errors.md) — general error reference
