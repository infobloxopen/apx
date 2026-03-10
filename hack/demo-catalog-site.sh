#!/usr/bin/env bash
# demo-catalog-site.sh — Generates a demo catalog with 15 APIs across
# multiple formats, domains, lifecycles, and versions, then launches
# the catalog site explorer locally.
#
# Usage:
#   ./hack/demo-catalog-site.sh          # build apx, generate site, open browser
#   ./hack/demo-catalog-site.sh --no-open  # same but skip browser auto-open

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
DEMO_DIR="$SCRIPT_DIR/demo-catalog-site"

NO_OPEN="${1:-}"

# ── Step 1: Build apx from source ──────────────────────────────────────────
echo "▸ Building apx..."
cd "$REPO_ROOT"
GOTOOLCHAIN=auto go build -o "$DEMO_DIR/apx" ./cmd/apx
echo "  ✓ Binary: $DEMO_DIR/apx"

# ── Step 2: Write a 15-API demo catalog ─────────────────────────────────────
echo "▸ Writing demo catalog.yaml..."
mkdir -p "$DEMO_DIR"
cat > "$DEMO_DIR/catalog.yaml" <<'CATALOG'
version: 1
org: acme
repo: apis
import_root: go.acme.dev/apis

modules:
  # ─── Proto APIs (Payments domain) ───────────────────────────────────
  - id: proto/payments/ledger/v1
    format: proto
    domain: payments
    api_line: v1
    description: "Core ledger service — manages accounts, balances, and double-entry transactions"
    version: v1.8.2
    latest_stable: v1.8.2
    lifecycle: stable
    path: proto/payments/ledger/v1
    tags: [payments, ledger, finops]
    owners: [payments-team]

  - id: proto/payments/invoices/v2
    format: proto
    domain: payments
    api_line: v2
    description: "Invoice generation, lifecycle tracking, and PDF rendering"
    version: v2.1.0
    latest_stable: v2.1.0
    latest_prerelease: v2.2.0-rc.1
    lifecycle: stable
    path: proto/payments/invoices/v2
    tags: [payments, invoices, billing]
    owners: [payments-team]

  - id: proto/payments/disputes/v1
    format: proto
    domain: payments
    api_line: v1
    description: "Dispute resolution workflow — chargebacks, evidence submission, arbitration"
    version: v1.0.0-beta.3
    latest_prerelease: v1.0.0-beta.3
    lifecycle: beta
    path: proto/payments/disputes/v1
    tags: [payments, disputes, compliance]
    owners: [payments-team, compliance-team]

  # ─── Proto APIs (Identity domain) ───────────────────────────────────
  - id: proto/identity/users/v1
    format: proto
    domain: identity
    api_line: v1
    description: "User registration, profile management, and authentication tokens"
    version: v1.12.0
    latest_stable: v1.12.0
    lifecycle: stable
    path: proto/identity/users/v1
    tags: [identity, users, auth]
    owners: [identity-team]

  - id: proto/identity/roles/v1
    format: proto
    domain: identity
    api_line: v1
    description: "RBAC role definitions, permission grants, and policy evaluation"
    version: v1.3.1
    latest_stable: v1.3.1
    lifecycle: stable
    path: proto/identity/roles/v1
    tags: [identity, rbac, authorization]
    owners: [identity-team]

  # ─── OpenAPI APIs ───────────────────────────────────────────────────
  - id: openapi/billing/subscriptions/v1
    format: openapi
    domain: billing
    api_line: v1
    description: "Subscription plans, trials, renewals, and cancellation flows"
    version: v1.5.0
    latest_stable: v1.5.0
    lifecycle: stable
    path: openapi/billing/subscriptions/v1
    tags: [billing, subscriptions, saas]
    owners: [billing-team]

  - id: openapi/billing/metering/v1
    format: openapi
    domain: billing
    api_line: v1
    description: "Usage metering, rate aggregation, and overage tracking"
    version: v1.0.0-alpha.2
    latest_prerelease: v1.0.0-alpha.2
    lifecycle: experimental
    path: openapi/billing/metering/v1
    tags: [billing, metering, usage]
    owners: [billing-team]

  - id: openapi/shipping/tracking/v1
    format: openapi
    domain: shipping
    api_line: v1
    description: "Shipment tracking, carrier integration, and delivery ETAs"
    version: v1.2.0
    latest_stable: v1.2.0
    lifecycle: stable
    path: openapi/shipping/tracking/v1
    tags: [shipping, tracking, logistics]
    owners: [logistics-team]

  # ─── Avro event schemas ────────────────────────────────────────────
  - id: avro/events/clicks/v1
    format: avro
    domain: events
    api_line: v1
    description: "Clickstream events — page views, button clicks, and navigation"
    version: v1.4.0
    latest_stable: v1.4.0
    lifecycle: stable
    path: avro/events/clicks/v1
    tags: [events, clickstream, analytics]
    owners: [data-platform-team]

  - id: avro/events/transactions/v1
    format: avro
    domain: events
    api_line: v1
    description: "Financial transaction events for the data warehouse pipeline"
    version: v1.1.0
    latest_stable: v1.1.0
    lifecycle: stable
    path: avro/events/transactions/v1
    tags: [events, transactions, data-warehouse]
    owners: [data-platform-team]

  # ─── JSON Schema configs ───────────────────────────────────────────
  - id: jsonschema/config/feature-flags/v1
    format: jsonschema
    domain: config
    api_line: v1
    description: "Feature flag configuration schema — rollouts, targeting rules, overrides"
    version: v1.0.0
    latest_stable: v1.0.0
    lifecycle: stable
    path: jsonschema/config/feature-flags/v1
    tags: [config, feature-flags, rollout]
    owners: [platform-team]

  - id: jsonschema/config/notifications/v1
    format: jsonschema
    domain: config
    api_line: v1
    description: "Notification routing and template configuration"
    version: v1.0.0-beta.1
    latest_prerelease: v1.0.0-beta.1
    lifecycle: beta
    path: jsonschema/config/notifications/v1
    tags: [config, notifications, email]
    owners: [platform-team]

  # ─── Parquet data schemas ──────────────────────────────────────────
  - id: parquet/warehouse/orders/v1
    format: parquet
    domain: warehouse
    api_line: v1
    description: "Order fact table schema for the analytics data lake"
    version: v1.6.0
    latest_stable: v1.6.0
    lifecycle: stable
    path: parquet/warehouse/orders/v1
    tags: [warehouse, orders, analytics, datalake]
    owners: [data-platform-team]

  # ─── External / deprecated APIs ────────────────────────────────────
  - id: proto/google/pubsub/v1
    format: proto
    domain: google
    api_line: v1
    description: "Google Cloud Pub/Sub — managed publish-subscribe messaging"
    version: v1.0.0
    latest_stable: v1.0.0
    lifecycle: stable
    path: proto/google/pubsub/v1
    origin: external
    upstream_repo: github.com/googleapis/googleapis
    upstream_path: google/pubsub/v1
    import_mode: preserve
    tags: [google, pubsub, messaging, external]
    owners: [platform-team]

  - id: proto/payments/charges/v1
    format: proto
    domain: payments
    api_line: v1
    description: "Legacy charge processing — superseded by ledger/v1"
    version: v1.9.0
    latest_stable: v1.9.0
    lifecycle: deprecated
    path: proto/payments/charges/v1
    tags: [payments, legacy, deprecated]
    owners: [payments-team]
CATALOG

echo "  ✓ 15 APIs across 5 formats, 7 domains, 5 lifecycle states"

# ── Step 3: Create schema files for each API ────────────────────────────────
echo "▸ Writing schema files..."

# Proto: payments/ledger/v1
mkdir -p "$DEMO_DIR/proto/payments/ledger/v1"
cat > "$DEMO_DIR/proto/payments/ledger/v1/ledger.proto" <<'EOF'
syntax = "proto3";

package payments.ledger.v1;

import "google/protobuf/timestamp.proto";

option go_package = "go.acme.dev/apis/proto/payments/ledger/v1;ledgerpb";

// LedgerService manages financial ledger entries and account balances.
service LedgerService {
  // CreateEntry records a new double-entry transaction.
  rpc CreateEntry(CreateEntryRequest) returns (CreateEntryResponse);

  // GetEntry retrieves a single ledger entry by ID.
  rpc GetEntry(GetEntryRequest) returns (Entry);

  // ListEntries returns paginated entries for an account.
  rpc ListEntries(ListEntriesRequest) returns (ListEntriesResponse);

  // GetBalance returns the current balance for an account.
  rpc GetBalance(GetBalanceRequest) returns (Balance);
}

// Entry represents a single ledger entry (one side of a double-entry).
message Entry {
  string id = 1;                          // unique entry ID
  string account_id = 2;                  // owning account
  string contra_account_id = 3;           // other side of the entry
  int64 amount_cents = 4;                 // signed amount in cents
  string currency = 5;                    // ISO 4217 currency code
  string description = 6;                 // human-readable description
  google.protobuf.Timestamp created_at = 7;
  map<string, string> metadata = 8;       // arbitrary key-value pairs
}

// Account holds the running balance for a ledger account.
message Account {
  string id = 1;
  string name = 2;
  string type = 3; // asset, liability, revenue, expense
  int64 balance_cents = 4;
  string currency = 5;
}

// Balance is the current balance snapshot for an account.
message Balance {
  string account_id = 1;
  int64 available_cents = 2;
  int64 pending_cents = 3;
  string currency = 4;
}

// EntryStatus tracks processing state.
enum EntryStatus {
  ENTRY_STATUS_UNKNOWN = 0;
  ENTRY_STATUS_PENDING = 1;
  ENTRY_STATUS_POSTED = 2;
  ENTRY_STATUS_REVERSED = 3;
}

message CreateEntryRequest {
  string debit_account_id = 1;
  string credit_account_id = 2;
  int64 amount_cents = 3;
  string currency = 4;
  string description = 5;
}

message CreateEntryResponse {
  Entry debit_entry = 1;
  Entry credit_entry = 2;
}

message GetEntryRequest { string id = 1; }

message ListEntriesRequest {
  string account_id = 1;
  int32 page_size = 2;
  string page_token = 3;
}

message ListEntriesResponse {
  repeated Entry entries = 1;
  string next_page_token = 2;
}

message GetBalanceRequest { string account_id = 1; }
EOF

# Proto: payments/invoices/v2
mkdir -p "$DEMO_DIR/proto/payments/invoices/v2"
cat > "$DEMO_DIR/proto/payments/invoices/v2/invoices.proto" <<'EOF'
syntax = "proto3";

package payments.invoices.v2;

option go_package = "go.acme.dev/apis/proto/payments/invoices/v2;invoicespb";

// InvoiceService manages invoice lifecycle from creation to payment.
service InvoiceService {
  rpc CreateInvoice(CreateInvoiceRequest) returns (Invoice);
  rpc GetInvoice(GetInvoiceRequest) returns (Invoice);
  rpc ListInvoices(ListInvoicesRequest) returns (ListInvoicesResponse);
  rpc FinalizeInvoice(FinalizeInvoiceRequest) returns (Invoice);
  rpc VoidInvoice(VoidInvoiceRequest) returns (Invoice);
  // GeneratePDF returns a stream of PDF bytes.
  rpc GeneratePDF(GeneratePDFRequest) returns (stream PDFChunk);
}

message Invoice {
  string id = 1;
  string customer_id = 2;
  repeated LineItem line_items = 3;
  int64 subtotal_cents = 4;
  int64 tax_cents = 5;
  int64 total_cents = 6;
  string currency = 7;
  string status = 8; // draft, finalized, paid, void
  string due_date = 9;
}

message LineItem {
  string description = 1;
  int32 quantity = 2;
  int64 unit_price_cents = 3;
  int64 amount_cents = 4;
}

message CreateInvoiceRequest {
  string customer_id = 1;
  repeated LineItem line_items = 2;
  string currency = 3;
  string due_date = 4;
}

message GetInvoiceRequest { string id = 1; }
message ListInvoicesRequest {
  string customer_id = 1;
  int32 page_size = 2;
  string page_token = 3;
}
message ListInvoicesResponse {
  repeated Invoice invoices = 1;
  string next_page_token = 2;
}
message FinalizeInvoiceRequest { string id = 1; }
message VoidInvoiceRequest { string id = 1; }
message GeneratePDFRequest { string invoice_id = 1; }
message PDFChunk { bytes data = 1; }
EOF

# Proto: payments/disputes/v1
mkdir -p "$DEMO_DIR/proto/payments/disputes/v1"
cat > "$DEMO_DIR/proto/payments/disputes/v1/disputes.proto" <<'EOF'
syntax = "proto3";

package payments.disputes.v1;

option go_package = "go.acme.dev/apis/proto/payments/disputes/v1;disputespb";

// DisputeService handles chargeback disputes and evidence management.
service DisputeService {
  rpc FileDispute(FileDisputeRequest) returns (Dispute);
  rpc GetDispute(GetDisputeRequest) returns (Dispute);
  rpc SubmitEvidence(SubmitEvidenceRequest) returns (Dispute);
  rpc AcceptDispute(AcceptDisputeRequest) returns (Dispute);
}

message Dispute {
  string id = 1;
  string transaction_id = 2;
  string reason = 3;
  int64 amount_cents = 4;
  string status = 5; // open, evidence_submitted, won, lost, accepted
  repeated Evidence evidence = 6;
}

message Evidence {
  string id = 1;
  string type = 2; // receipt, shipping_proof, correspondence
  string url = 3;
  string description = 4;
}

message FileDisputeRequest {
  string transaction_id = 1;
  string reason = 2;
  int64 amount_cents = 3;
}

message GetDisputeRequest { string id = 1; }
message SubmitEvidenceRequest {
  string dispute_id = 1;
  Evidence evidence = 2;
}
message AcceptDisputeRequest { string id = 1; }
EOF

# Proto: identity/users/v1
mkdir -p "$DEMO_DIR/proto/identity/users/v1"
cat > "$DEMO_DIR/proto/identity/users/v1/users.proto" <<'EOF'
syntax = "proto3";

package identity.users.v1;

option go_package = "go.acme.dev/apis/proto/identity/users/v1;userspb";

// UserService manages user accounts and authentication.
service UserService {
  rpc CreateUser(CreateUserRequest) returns (User);
  rpc GetUser(GetUserRequest) returns (User);
  rpc UpdateProfile(UpdateProfileRequest) returns (User);
  rpc DeleteUser(DeleteUserRequest) returns (DeleteUserResponse);
  rpc Authenticate(AuthenticateRequest) returns (AuthToken);
}

message User {
  string id = 1;
  string email = 2;
  string display_name = 3;
  string avatar_url = 4;
  bool email_verified = 5;
  string created_at = 6;
}

message AuthToken {
  string access_token = 1;
  string refresh_token = 2;
  int64 expires_in = 3;
}

message CreateUserRequest {
  string email = 1;
  string password = 2;
  string display_name = 3;
}

message GetUserRequest { string id = 1; }
message UpdateProfileRequest {
  string id = 1;
  string display_name = 2;
  string avatar_url = 3;
}
message DeleteUserRequest { string id = 1; }
message DeleteUserResponse {}
message AuthenticateRequest {
  string email = 1;
  string password = 2;
}
EOF

# Proto: identity/roles/v1
mkdir -p "$DEMO_DIR/proto/identity/roles/v1"
cat > "$DEMO_DIR/proto/identity/roles/v1/roles.proto" <<'EOF'
syntax = "proto3";

package identity.roles.v1;

option go_package = "go.acme.dev/apis/proto/identity/roles/v1;rolespb";

// RoleService manages RBAC roles and permission assignments.
service RoleService {
  rpc CreateRole(CreateRoleRequest) returns (Role);
  rpc GetRole(GetRoleRequest) returns (Role);
  rpc AssignRole(AssignRoleRequest) returns (AssignRoleResponse);
  rpc CheckPermission(CheckPermissionRequest) returns (CheckPermissionResponse);
}

message Role {
  string id = 1;
  string name = 2;
  string description = 3;
  repeated Permission permissions = 4;
}

message Permission {
  string resource = 1; // e.g. "invoices", "users"
  string action = 2;   // e.g. "read", "write", "delete"
}

message CreateRoleRequest {
  string name = 1;
  string description = 2;
  repeated Permission permissions = 3;
}
message GetRoleRequest { string id = 1; }
message AssignRoleRequest {
  string user_id = 1;
  string role_id = 2;
}
message AssignRoleResponse {}
message CheckPermissionRequest {
  string user_id = 1;
  string resource = 2;
  string action = 3;
}
message CheckPermissionResponse {
  bool allowed = 1;
}
EOF

# Proto: google/pubsub/v1 (external)
mkdir -p "$DEMO_DIR/proto/google/pubsub/v1"
cat > "$DEMO_DIR/proto/google/pubsub/v1/pubsub.proto" <<'EOF'
syntax = "proto3";

package google.pubsub.v1;

// Publisher manages topic publishing.
service Publisher {
  rpc Publish(PublishRequest) returns (PublishResponse);
}

// Subscriber manages message consumption.
service Subscriber {
  rpc Pull(PullRequest) returns (PullResponse);
  rpc StreamingPull(stream StreamingPullRequest) returns (stream StreamingPullResponse);
  rpc Acknowledge(AcknowledgeRequest) returns (AcknowledgeResponse);
}

message PubsubMessage {
  bytes data = 1;
  map<string, string> attributes = 2;
  string message_id = 3;
  string publish_time = 4;
}

message Topic {
  string name = 1;
  map<string, string> labels = 2;
}

message PublishRequest {
  string topic = 1;
  repeated PubsubMessage messages = 2;
}
message PublishResponse { repeated string message_ids = 1; }
message PullRequest {
  string subscription = 1;
  int32 max_messages = 2;
}
message PullResponse { repeated ReceivedMessage received_messages = 1; }
message ReceivedMessage {
  string ack_id = 1;
  PubsubMessage message = 2;
}
message StreamingPullRequest { string subscription = 1; }
message StreamingPullResponse { repeated ReceivedMessage received_messages = 1; }
message AcknowledgeRequest {
  string subscription = 1;
  repeated string ack_ids = 2;
}
message AcknowledgeResponse {}
EOF

# Proto: payments/charges/v1 (deprecated)
mkdir -p "$DEMO_DIR/proto/payments/charges/v1"
cat > "$DEMO_DIR/proto/payments/charges/v1/charges.proto" <<'EOF'
syntax = "proto3";

package payments.charges.v1;

option go_package = "go.acme.dev/apis/proto/payments/charges/v1;chargespb";

// ChargeService is DEPRECATED. Use LedgerService instead.
service ChargeService {
  rpc CreateCharge(CreateChargeRequest) returns (Charge);
  rpc GetCharge(GetChargeRequest) returns (Charge);
  rpc RefundCharge(RefundChargeRequest) returns (Charge);
}

message Charge {
  string id = 1;
  string customer_id = 2;
  int64 amount_cents = 3;
  string currency = 4;
  string status = 5; // pending, captured, refunded, failed
}

message CreateChargeRequest {
  string customer_id = 1;
  int64 amount_cents = 2;
  string currency = 3;
  string payment_method_id = 4;
}
message GetChargeRequest { string id = 1; }
message RefundChargeRequest {
  string charge_id = 1;
  int64 amount_cents = 2; // partial refund amount; 0 = full refund
}
EOF

# OpenAPI: billing/subscriptions/v1
mkdir -p "$DEMO_DIR/openapi/billing/subscriptions/v1"
cat > "$DEMO_DIR/openapi/billing/subscriptions/v1/subscriptions.yaml" <<'EOF'
openapi: "3.0.3"
info:
  title: Subscription API
  version: "1.5.0"
  description: Manage subscription plans, trials, renewals, and cancellations.
paths:
  /subscriptions:
    get:
      summary: List subscriptions
      operationId: listSubscriptions
      parameters:
        - name: customer_id
          in: query
        - name: status
          in: query
        - name: limit
          in: query
      responses:
        "200":
          description: List of subscriptions
    post:
      summary: Create a subscription
      operationId: createSubscription
      responses:
        "201":
          description: Subscription created
        "400":
          description: Invalid request
  /subscriptions/{id}:
    get:
      summary: Get subscription by ID
      parameters:
        - name: id
          in: path
      responses:
        "200":
          description: Subscription details
        "404":
          description: Not found
    delete:
      summary: Cancel subscription
      parameters:
        - name: id
          in: path
      responses:
        "200":
          description: Subscription cancelled
  /plans:
    get:
      summary: List available plans
      responses:
        "200":
          description: List of plans
components:
  schemas:
    Subscription:
      type: object
      required: [id, customer_id, plan_id, status]
      properties:
        id:
          type: string
          description: Unique subscription ID
        customer_id:
          type: string
          description: Customer this subscription belongs to
        plan_id:
          type: string
          description: Plan the customer is subscribed to
        status:
          type: string
          description: "Current status: active, trialing, past_due, canceled"
        current_period_end:
          type: string
          format: date-time
          description: End of the current billing period
        cancel_at_period_end:
          type: boolean
    Plan:
      type: object
      required: [id, name, amount_cents]
      properties:
        id:
          type: string
        name:
          type: string
          description: Display name of the plan
        amount_cents:
          type: integer
          format: int64
          description: Price per billing period in cents
        interval:
          type: string
          description: "Billing interval: month, year"
        trial_days:
          type: integer
EOF

# OpenAPI: billing/metering/v1
mkdir -p "$DEMO_DIR/openapi/billing/metering/v1"
cat > "$DEMO_DIR/openapi/billing/metering/v1/metering.yaml" <<'EOF'
openapi: "3.0.3"
info:
  title: Usage Metering API
  version: "1.0.0-alpha.2"
  description: Record and aggregate usage data for consumption-based billing.
paths:
  /usage:
    post:
      summary: Report usage event
      operationId: reportUsage
      responses:
        "202":
          description: Usage event accepted
  /usage/aggregate:
    get:
      summary: Get aggregated usage
      operationId: getAggregatedUsage
      parameters:
        - name: customer_id
          in: query
        - name: metric
          in: query
        - name: start_date
          in: query
        - name: end_date
          in: query
      responses:
        "200":
          description: Aggregated usage data
components:
  schemas:
    UsageEvent:
      type: object
      required: [customer_id, metric, quantity]
      properties:
        customer_id:
          type: string
        metric:
          type: string
          description: "Usage metric name (e.g., api_calls, storage_gb)"
        quantity:
          type: number
          description: Quantity consumed
        timestamp:
          type: string
          format: date-time
    UsageAggregate:
      type: object
      properties:
        customer_id:
          type: string
        metric:
          type: string
        total:
          type: number
        period_start:
          type: string
          format: date-time
        period_end:
          type: string
          format: date-time
EOF

# OpenAPI: shipping/tracking/v1
mkdir -p "$DEMO_DIR/openapi/shipping/tracking/v1"
cat > "$DEMO_DIR/openapi/shipping/tracking/v1/tracking.yaml" <<'EOF'
openapi: "3.0.3"
info:
  title: Shipment Tracking API
  version: "1.2.0"
  description: Track shipments, get delivery ETAs, and list carriers.
paths:
  /shipments/{id}/track:
    get:
      summary: Get tracking info for a shipment
      parameters:
        - name: id
          in: path
      responses:
        "200":
          description: Tracking information
        "404":
          description: Shipment not found
  /shipments:
    get:
      summary: List shipments
      parameters:
        - name: order_id
          in: query
        - name: status
          in: query
      responses:
        "200":
          description: List of shipments
  /carriers:
    get:
      summary: List available carriers
      responses:
        "200":
          description: List of carriers
components:
  schemas:
    Shipment:
      type: object
      required: [id, carrier, status]
      properties:
        id:
          type: string
          description: Shipment ID
        order_id:
          type: string
        carrier:
          type: string
          description: Carrier code (ups, fedex, usps, dhl)
        tracking_number:
          type: string
        status:
          type: string
          description: "Status: pending, in_transit, delivered, exception"
        estimated_delivery:
          type: string
          format: date
        events:
          type: array
          items:
            $ref: "#/components/schemas/TrackingEvent"
    TrackingEvent:
      type: object
      properties:
        timestamp:
          type: string
          format: date-time
        location:
          type: string
        description:
          type: string
        status:
          type: string
EOF

# Avro: events/clicks/v1
mkdir -p "$DEMO_DIR/avro/events/clicks/v1"
cat > "$DEMO_DIR/avro/events/clicks/v1/clicks.avsc" <<'EOF'
{
  "type": "record",
  "name": "ClickEvent",
  "namespace": "com.acme.events.clicks",
  "doc": "Clickstream event capturing user interactions with UI elements.",
  "fields": [
    {"name": "eventId", "type": "string", "doc": "Unique event identifier (UUID)"},
    {"name": "userId", "type": ["null", "string"], "default": null, "doc": "Authenticated user ID, null for anonymous"},
    {"name": "sessionId", "type": "string", "doc": "Browser session identifier"},
    {"name": "timestamp", "type": "long", "doc": "Event timestamp in milliseconds since epoch"},
    {"name": "pageUrl", "type": "string", "doc": "Full URL of the page"},
    {"name": "elementId", "type": ["null", "string"], "default": null, "doc": "DOM element ID that was clicked"},
    {"name": "elementType", "type": "string", "doc": "Element type: button, link, image, form, other"},
    {"name": "metadata", "type": {"type": "map", "values": "string"}, "doc": "Additional key-value metadata"}
  ]
}
EOF

# Avro: events/transactions/v1
mkdir -p "$DEMO_DIR/avro/events/transactions/v1"
cat > "$DEMO_DIR/avro/events/transactions/v1/transactions.avsc" <<'EOF'
{
  "type": "record",
  "name": "TransactionEvent",
  "namespace": "com.acme.events.transactions",
  "doc": "Financial transaction events for the data warehouse pipeline.",
  "fields": [
    {"name": "transactionId", "type": "string", "doc": "Unique transaction ID"},
    {"name": "accountId", "type": "string", "doc": "Source account"},
    {"name": "amountCents", "type": "long", "doc": "Signed amount in cents"},
    {"name": "currency", "type": "string", "doc": "ISO 4217 currency code"},
    {"name": "type", "type": {"type": "enum", "name": "TransactionType", "symbols": ["DEBIT", "CREDIT", "TRANSFER", "REFUND"]}, "doc": "Transaction classification"},
    {"name": "status", "type": "string", "doc": "Processing status: pending, posted, failed, reversed"},
    {"name": "timestamp", "type": "long", "doc": "Event timestamp in milliseconds"},
    {"name": "merchantId", "type": ["null", "string"], "default": null, "doc": "Merchant ID for card transactions"}
  ]
}
EOF

# JSON Schema: config/feature-flags/v1
mkdir -p "$DEMO_DIR/jsonschema/config/feature-flags/v1"
cat > "$DEMO_DIR/jsonschema/config/feature-flags/v1/feature-flags.json" <<'EOF'
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "Feature Flag Configuration",
  "description": "Schema for defining feature flags with rollout rules and targeting.",
  "type": "object",
  "properties": {
    "name": {
      "type": "string",
      "description": "Unique flag identifier (kebab-case)"
    },
    "description": {
      "type": "string",
      "description": "Human-readable description of the flag"
    },
    "enabled": {
      "type": "boolean",
      "description": "Global kill switch for this flag"
    },
    "rolloutPercentage": {
      "type": "number",
      "description": "Percentage of users who see this flag (0-100)"
    },
    "targeting": {
      "type": "object",
      "description": "Targeting rules for specific user segments",
      "properties": {
        "allowList": {
          "type": "array",
          "items": {"type": "string"},
          "description": "User IDs that always see this flag"
        },
        "denyList": {
          "type": "array",
          "items": {"type": "string"},
          "description": "User IDs that never see this flag"
        },
        "attributes": {
          "type": "object",
          "description": "Attribute-based targeting rules",
          "properties": {
            "plan": {"type": "string", "description": "Required subscription plan"},
            "region": {"type": "string", "description": "Required region code"}
          }
        }
      }
    },
    "variants": {
      "type": "array",
      "items": {"type": "string"},
      "description": "A/B test variant names"
    }
  },
  "required": ["name", "enabled"]
}
EOF

# JSON Schema: config/notifications/v1
mkdir -p "$DEMO_DIR/jsonschema/config/notifications/v1"
cat > "$DEMO_DIR/jsonschema/config/notifications/v1/notifications.json" <<'EOF'
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "Notification Configuration",
  "description": "Routing and template configuration for notifications.",
  "type": "object",
  "properties": {
    "channel": {
      "type": "string",
      "description": "Delivery channel: email, sms, push, slack, webhook"
    },
    "templateId": {
      "type": "string",
      "description": "Reference to a notification template"
    },
    "recipients": {
      "type": "array",
      "items": {"type": "string"},
      "description": "List of recipient addresses or user IDs"
    },
    "schedule": {
      "type": "object",
      "description": "Delivery schedule configuration",
      "properties": {
        "immediate": {"type": "boolean", "description": "Send immediately"},
        "delay_seconds": {"type": "integer", "description": "Delay before sending"},
        "batch_window": {"type": "string", "description": "Batching window (e.g., 5m, 1h)"}
      },
      "required": ["immediate"]
    },
    "priority": {
      "type": "string",
      "description": "Delivery priority: low, normal, high, critical"
    }
  },
  "required": ["channel", "templateId", "recipients"]
}
EOF

# Parquet: warehouse/orders/v1
mkdir -p "$DEMO_DIR/parquet/warehouse/orders/v1"
cat > "$DEMO_DIR/parquet/warehouse/orders/v1/orders.parquet" <<'EOF'
message order_fact {
  required binary order_id (STRING);
  required binary customer_id (STRING);
  required int64 order_date (TIMESTAMP_MILLIS);
  required double total_amount;
  required double tax_amount;
  optional double discount_amount;
  required binary currency (STRING);
  required binary status (STRING);
  optional binary shipping_carrier (STRING);
  optional int32 item_count;
  required binary region (STRING);
  optional binary promo_code (STRING);
}
EOF

echo "  ✓ Schema files created for all 15 APIs"

# ── Step 4: Generate and serve ──────────────────────────────────────────────
echo "▸ Launching catalog site on http://localhost:10451 ..."
echo "  Press Ctrl+C to stop"
echo ""

SERVE_FLAGS=(--catalog "$DEMO_DIR/catalog.yaml" --dir "$DEMO_DIR")
if [ "$NO_OPEN" = "--no-open" ]; then
  SERVE_FLAGS+=(--no-open)
fi

"$DEMO_DIR/apx" catalog site serve "${SERVE_FLAGS[@]}"
