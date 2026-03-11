#!/usr/bin/env bash
# demo-catalog-site.sh — Generates a demo catalog with 45 APIs across
# 5 formats, 12 domains, and all lifecycle states, then launches
# the catalog site explorer locally.
#
# Theme: MegaMart — a full e-commerce + warehouse platform
#   - OpenAPI (REST) for external-facing APIs
#   - Proto (gRPC) for internal service-to-service
#   - Avro for inter-service event messaging
#   - Parquet for reporting data lake schemas
#   - JSON Schema for configuration
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

# ── Step 2: Write a 45-API demo catalog ─────────────────────────────────────
echo "▸ Writing demo catalog.yaml..."
mkdir -p "$DEMO_DIR"
cat > "$DEMO_DIR/catalog.yaml" <<'CATALOG'
version: 1
org: megamart
repo: apis
import_root: go.megamart.dev/apis

modules:
  # ━━━ OpenAPI — REST, external-facing ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

  # ─── IAM domain ────────────────────────────────────────────────────
  - id: openapi/iam/users/v1
    format: openapi
    domain: iam
    api_line: v1
    description: "User registration, profiles, password resets, and account lifecycle"
    version: v1.8.0
    latest_stable: v1.8.0
    lifecycle: stable
    path: openapi/iam/users/v1
    tags: [iam, users, registration, auth]
    owners: [iam-team]

  - id: openapi/iam/groups/v1
    format: openapi
    domain: iam
    api_line: v1
    description: "User group management — create, assign members, and list groups"
    version: v1.3.0
    latest_stable: v1.3.0
    lifecycle: stable
    path: openapi/iam/groups/v1
    tags: [iam, groups, organization]
    owners: [iam-team]

  - id: openapi/iam/access-policies/v1
    format: openapi
    domain: iam
    api_line: v1
    description: "RBAC access policies — define rules binding roles to resources"
    version: v1.5.0
    latest_stable: v1.5.0
    lifecycle: stable
    path: openapi/iam/access-policies/v1
    tags: [iam, rbac, policies, authorization]
    owners: [iam-team, security-team]

  - id: openapi/iam/compartments/v1
    format: openapi
    domain: iam
    api_line: v1
    description: "Resource compartments for multi-tenancy isolation"
    version: v1.0.0-beta.2
    latest_prerelease: v1.0.0-beta.2
    lifecycle: beta
    path: openapi/iam/compartments/v1
    tags: [iam, compartments, multi-tenancy]
    owners: [iam-team]

  - id: openapi/iam/service-accounts/v1
    format: openapi
    domain: iam
    api_line: v1
    description: "Machine-to-machine service accounts and API key management"
    version: v1.1.0
    latest_stable: v1.1.0
    lifecycle: stable
    path: openapi/iam/service-accounts/v1
    tags: [iam, service-accounts, api-keys]
    owners: [iam-team]

  # ─── Catalog domain ────────────────────────────────────────────────
  - id: openapi/catalog/products/v1
    format: openapi
    domain: catalog
    api_line: v1
    description: "Legacy product listings — superseded by products/v2"
    version: v1.9.0
    latest_stable: v1.9.0
    lifecycle: deprecated
    path: openapi/catalog/products/v1
    tags: [catalog, products, legacy, deprecated]
    owners: [catalog-team]

  - id: openapi/catalog/products/v2
    format: openapi
    domain: catalog
    api_line: v2
    description: "Product catalog with variants, rich attributes, and localization"
    version: v2.3.0
    latest_stable: v2.3.0
    lifecycle: stable
    path: openapi/catalog/products/v2
    tags: [catalog, products, variants]
    owners: [catalog-team]

  - id: openapi/catalog/categories/v1
    format: openapi
    domain: catalog
    api_line: v1
    description: "Category taxonomy — hierarchical tree of product categories"
    version: v1.2.0
    latest_stable: v1.2.0
    lifecycle: stable
    path: openapi/catalog/categories/v1
    tags: [catalog, categories, taxonomy]
    owners: [catalog-team]

  - id: openapi/catalog/images/v1
    format: openapi
    domain: catalog
    api_line: v1
    description: "Product image upload, transformation, thumbnails, and CDN URLs"
    version: v1.4.0
    latest_stable: v1.4.0
    lifecycle: stable
    path: openapi/catalog/images/v1
    tags: [catalog, images, media, cdn]
    owners: [catalog-team, media-team]

  - id: openapi/catalog/skus/v1
    format: openapi
    domain: catalog
    api_line: v1
    description: "SKU management — inventory units, pricing tiers, and barcodes"
    version: v1.1.0
    latest_stable: v1.1.0
    lifecycle: stable
    path: openapi/catalog/skus/v1
    tags: [catalog, skus, pricing, inventory]
    owners: [catalog-team]

  - id: openapi/catalog/search/v0
    format: openapi
    domain: catalog
    api_line: v0
    description: "Full-text product search with facets and autocomplete"
    version: v0.3.0-alpha.1
    latest_prerelease: v0.3.0-alpha.1
    lifecycle: experimental
    path: openapi/catalog/search/v0
    tags: [catalog, search, experimental]
    owners: [search-team]

  # ─── Orders domain ─────────────────────────────────────────────────
  - id: openapi/orders/checkout/v1
    format: openapi
    domain: orders
    api_line: v1
    description: "Shopping cart checkout — address validation, payment, order creation"
    version: v1.6.0
    latest_stable: v1.6.0
    lifecycle: stable
    path: openapi/orders/checkout/v1
    tags: [orders, checkout, cart]
    owners: [orders-team]

  - id: openapi/orders/orders/v1
    format: openapi
    domain: orders
    api_line: v1
    description: "Legacy order management — replaced by orders/v2"
    version: v1.12.0
    latest_stable: v1.12.0
    lifecycle: sunset
    path: openapi/orders/orders/v1
    tags: [orders, legacy, sunset]
    owners: [orders-team]

  - id: openapi/orders/orders/v2
    format: openapi
    domain: orders
    api_line: v2
    description: "Order management with partial fulfillment, split shipments, and status tracking"
    version: v2.1.0
    latest_stable: v2.1.0
    lifecycle: stable
    path: openapi/orders/orders/v2
    tags: [orders, fulfillment]
    owners: [orders-team]

  - id: openapi/orders/returns/v1
    format: openapi
    domain: orders
    api_line: v1
    description: "Returns, refunds, and exchange processing"
    version: v1.2.0
    latest_stable: v1.2.0
    lifecycle: stable
    path: openapi/orders/returns/v1
    tags: [orders, returns, refunds]
    owners: [orders-team]

  # ─── Shipping domain ───────────────────────────────────────────────
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

  - id: openapi/shipping/rates/v1
    format: openapi
    domain: shipping
    api_line: v1
    description: "Carrier rate quotes, cost comparison, and delivery estimates"
    version: v1.0.0
    latest_stable: v1.0.0
    lifecycle: stable
    path: openapi/shipping/rates/v1
    tags: [shipping, rates, carriers]
    owners: [logistics-team]

  - id: openapi/shipping/labels/v1
    format: openapi
    domain: shipping
    api_line: v1
    description: "Shipping label generation, void, and reprint"
    version: v1.1.0
    latest_stable: v1.1.0
    lifecycle: stable
    path: openapi/shipping/labels/v1
    tags: [shipping, labels, carriers]
    owners: [logistics-team]

  # ─── Notifications domain ──────────────────────────────────────────
  - id: openapi/notifications/email/v1
    format: openapi
    domain: notifications
    api_line: v1
    description: "Transactional email — templates, welcome emails, password resets, order confirmations"
    version: v1.3.0
    latest_stable: v1.3.0
    lifecycle: stable
    path: openapi/notifications/email/v1
    tags: [notifications, email, templates]
    owners: [notifications-team]

  # ━━━ Proto — gRPC, internal service-to-service ━━━━━━━━━━━━━━━━━━━━

  # ─── Payments domain ───────────────────────────────────────────────
  - id: proto/payments/ledger/v1
    format: proto
    domain: payments
    api_line: v1
    description: "Core ledger service — accounts, balances, and double-entry transactions"
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
    description: "Dispute resolution — chargebacks, evidence submission, arbitration"
    version: v1.0.0-beta.3
    latest_prerelease: v1.0.0-beta.3
    lifecycle: beta
    path: proto/payments/disputes/v1
    tags: [payments, disputes, compliance]
    owners: [payments-team, compliance-team]

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

  # ─── Identity domain ───────────────────────────────────────────────
  - id: proto/identity/users/v1
    format: proto
    domain: identity
    api_line: v1
    description: "Internal user accounts, profile management, and authentication tokens"
    version: v1.12.0
    latest_stable: v1.12.0
    lifecycle: stable
    path: proto/identity/users/v1
    tags: [identity, users, auth]
    owners: [iam-team]

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
    owners: [iam-team]

  # ─── Inventory domain ──────────────────────────────────────────────
  - id: proto/inventory/stock/v1
    format: proto
    domain: inventory
    api_line: v1
    description: "Stock levels, reservations, adjustments, and cycle counts"
    version: v1.5.0
    latest_stable: v1.5.0
    lifecycle: stable
    path: proto/inventory/stock/v1
    tags: [inventory, stock, warehouse]
    owners: [inventory-team]

  - id: proto/inventory/allocation/v1
    format: proto
    domain: inventory
    api_line: v1
    description: "Inventory allocation for orders — reserve, commit, and release"
    version: v1.2.0
    latest_stable: v1.2.0
    lifecycle: stable
    path: proto/inventory/allocation/v1
    tags: [inventory, allocation, orders]
    owners: [inventory-team]

  # ─── Warehouse domain ──────────────────────────────────────────────
  - id: proto/warehouse/picking/v1
    format: proto
    domain: warehouse
    api_line: v1
    description: "Pick lists, bin locations, wave planning, and pick confirmation"
    version: v1.3.0
    latest_stable: v1.3.0
    lifecycle: stable
    path: proto/warehouse/picking/v1
    tags: [warehouse, picking, wms]
    owners: [warehouse-team]

  - id: proto/warehouse/receiving/v1
    format: proto
    domain: warehouse
    api_line: v1
    description: "Inbound receiving — purchase order receipt, inspection, and putaway"
    version: v1.1.0
    latest_stable: v1.1.0
    lifecycle: stable
    path: proto/warehouse/receiving/v1
    tags: [warehouse, receiving, inbound]
    owners: [warehouse-team]

  - id: proto/warehouse/transfers/v1
    format: proto
    domain: warehouse
    api_line: v1
    description: "Inter-warehouse inventory transfers and transit tracking"
    version: v1.0.0-beta.1
    latest_prerelease: v1.0.0-beta.1
    lifecycle: beta
    path: proto/warehouse/transfers/v1
    tags: [warehouse, transfers, logistics]
    owners: [warehouse-team]

  # ─── Fulfillment domain ────────────────────────────────────────────
  - id: proto/fulfillment/routing/v1
    format: proto
    domain: fulfillment
    api_line: v1
    description: "Order routing — assign orders to optimal warehouse based on proximity and stock"
    version: v1.4.0
    latest_stable: v1.4.0
    lifecycle: stable
    path: proto/fulfillment/routing/v1
    tags: [fulfillment, routing, optimization]
    owners: [fulfillment-team]

  - id: proto/fulfillment/packing/v1
    format: proto
    domain: fulfillment
    api_line: v1
    description: "Packing verification, box selection, weight capture, and seal confirmation"
    version: v1.2.0
    latest_stable: v1.2.0
    lifecycle: stable
    path: proto/fulfillment/packing/v1
    tags: [fulfillment, packing, shipping]
    owners: [fulfillment-team]

  # ─── Google domain (external) ──────────────────────────────────────
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

  # ━━━ Avro — inter-service event messaging ━━━━━━━━━━━━━━━━━━━━━━━━━

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

  - id: avro/events/order-placed/v1
    format: avro
    domain: events
    api_line: v1
    description: "Order placed event — emitted when checkout completes"
    version: v1.2.0
    latest_stable: v1.2.0
    lifecycle: stable
    path: avro/events/order-placed/v1
    tags: [events, orders, checkout]
    owners: [orders-team, data-platform-team]

  - id: avro/events/order-shipped/v1
    format: avro
    domain: events
    api_line: v1
    description: "Order shipped event — emitted when a shipment leaves the warehouse"
    version: v1.1.0
    latest_stable: v1.1.0
    lifecycle: stable
    path: avro/events/order-shipped/v1
    tags: [events, orders, shipping]
    owners: [fulfillment-team, data-platform-team]

  - id: avro/events/inventory-updated/v1
    format: avro
    domain: events
    api_line: v1
    description: "Inventory change event — stock adjustments, receipts, and depletions"
    version: v1.0.0
    latest_stable: v1.0.0
    lifecycle: stable
    path: avro/events/inventory-updated/v1
    tags: [events, inventory, stock]
    owners: [inventory-team, data-platform-team]

  - id: avro/events/user-registered/v1
    format: avro
    domain: events
    api_line: v1
    description: "User registration event — triggers welcome email and analytics"
    version: v1.0.0-beta.1
    latest_prerelease: v1.0.0-beta.1
    lifecycle: beta
    path: avro/events/user-registered/v1
    tags: [events, users, registration]
    owners: [iam-team, data-platform-team]

  # ━━━ Parquet — reporting data lake schemas ━━━━━━━━━━━━━━━━━━━━━━━━

  - id: parquet/reporting/orders/v1
    format: parquet
    domain: reporting
    api_line: v1
    description: "Order fact table for the analytics data lake"
    version: v1.6.0
    latest_stable: v1.6.0
    lifecycle: stable
    path: parquet/reporting/orders/v1
    tags: [reporting, orders, analytics, datalake]
    owners: [data-platform-team]

  - id: parquet/reporting/revenue/v1
    format: parquet
    domain: reporting
    api_line: v1
    description: "Revenue summary table — daily aggregates by product, region, and channel"
    version: v1.2.0
    latest_stable: v1.2.0
    lifecycle: stable
    path: parquet/reporting/revenue/v1
    tags: [reporting, revenue, analytics]
    owners: [data-platform-team]

  - id: parquet/reporting/inventory-snapshot/v1
    format: parquet
    domain: reporting
    api_line: v1
    description: "Daily inventory snapshot — stock levels by warehouse and SKU"
    version: v1.1.0
    latest_stable: v1.1.0
    lifecycle: stable
    path: parquet/reporting/inventory-snapshot/v1
    tags: [reporting, inventory, warehouse]
    owners: [data-platform-team]

  - id: parquet/reporting/customer-activity/v1
    format: parquet
    domain: reporting
    api_line: v1
    description: "Customer activity analytics — sessions, conversions, lifetime value"
    version: v1.0.0-beta.1
    latest_prerelease: v1.0.0-beta.1
    lifecycle: beta
    path: parquet/reporting/customer-activity/v1
    tags: [reporting, customers, analytics]
    owners: [data-platform-team]

  # ━━━ JSON Schema — configuration ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

  - id: jsonschema/config/feature-flags/v1
    format: jsonschema
    domain: config
    api_line: v1
    description: "Feature flag configuration — rollouts, targeting rules, overrides"
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
CATALOG

echo "  ✓ 45 APIs across 5 formats, 12 domains, 6 lifecycle states"

# ── Step 3: Create schema files for each API ────────────────────────────────
echo "▸ Writing schema files..."

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Proto schemas (internal gRPC services)
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

# Proto: payments/ledger/v1
mkdir -p "$DEMO_DIR/proto/payments/ledger/v1"
cat > "$DEMO_DIR/proto/payments/ledger/v1/ledger.proto" <<'EOF'
syntax = "proto3";

package payments.ledger.v1;

import "google/protobuf/timestamp.proto";

option go_package = "go.megamart.dev/apis/proto/payments/ledger/v1;ledgerpb";

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

option go_package = "go.megamart.dev/apis/proto/payments/invoices/v2;invoicespb";

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

option go_package = "go.megamart.dev/apis/proto/payments/disputes/v1;disputespb";

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

option go_package = "go.megamart.dev/apis/proto/identity/users/v1;userspb";

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

option go_package = "go.megamart.dev/apis/proto/identity/roles/v1;rolespb";

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

option go_package = "go.megamart.dev/apis/proto/payments/charges/v1;chargespb";

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

# Proto: inventory/stock/v1
mkdir -p "$DEMO_DIR/proto/inventory/stock/v1"
cat > "$DEMO_DIR/proto/inventory/stock/v1/stock.proto" <<'EOF'
syntax = "proto3";

package inventory.stock.v1;

option go_package = "go.megamart.dev/apis/proto/inventory/stock/v1;stockpb";

// StockService manages stock levels, reservations, and adjustments.
service StockService {
  // GetStockLevel returns the current stock for a SKU at a warehouse.
  rpc GetStockLevel(GetStockLevelRequest) returns (StockLevel);

  // ListStockLevels returns stock across all warehouses for a SKU.
  rpc ListStockLevels(ListStockLevelsRequest) returns (ListStockLevelsResponse);

  // AdjustStock applies a manual stock adjustment (receiving, damage, cycle count).
  rpc AdjustStock(AdjustStockRequest) returns (StockLevel);

  // ReserveStock places a soft hold on inventory for an order.
  rpc ReserveStock(ReserveStockRequest) returns (Reservation);

  // ReleaseReservation cancels a previously held reservation.
  rpc ReleaseReservation(ReleaseReservationRequest) returns (ReleaseReservationResponse);
}

// StockLevel represents current inventory for a SKU at a location.
message StockLevel {
  string sku_id = 1;
  string warehouse_id = 2;
  int32 on_hand = 3;          // physically present
  int32 reserved = 4;         // held for orders
  int32 available = 5;        // on_hand - reserved
  int32 incoming = 6;         // in transit from suppliers
  string updated_at = 7;
}

// Reservation is a soft hold on stock for a pending order.
message Reservation {
  string id = 1;
  string sku_id = 2;
  string warehouse_id = 3;
  string order_id = 4;
  int32 quantity = 5;
  string status = 6; // held, committed, released, expired
  string expires_at = 7;
}

// AdjustmentReason classifies why stock was adjusted.
enum AdjustmentReason {
  ADJUSTMENT_REASON_UNSPECIFIED = 0;
  ADJUSTMENT_REASON_RECEIVING = 1;
  ADJUSTMENT_REASON_DAMAGE = 2;
  ADJUSTMENT_REASON_CYCLE_COUNT = 3;
  ADJUSTMENT_REASON_RETURN = 4;
  ADJUSTMENT_REASON_TRANSFER = 5;
}

message GetStockLevelRequest {
  string sku_id = 1;
  string warehouse_id = 2;
}

message ListStockLevelsRequest {
  string sku_id = 1;
  int32 page_size = 2;
  string page_token = 3;
}

message ListStockLevelsResponse {
  repeated StockLevel levels = 1;
  string next_page_token = 2;
}

message AdjustStockRequest {
  string sku_id = 1;
  string warehouse_id = 2;
  int32 quantity_delta = 3;    // positive = add, negative = remove
  AdjustmentReason reason = 4;
  string reference_id = 5;     // PO number, return ID, etc.
  string notes = 6;
}

message ReserveStockRequest {
  string sku_id = 1;
  string warehouse_id = 2;
  string order_id = 3;
  int32 quantity = 4;
}

message ReleaseReservationRequest { string reservation_id = 1; }
message ReleaseReservationResponse {}
EOF

# Proto: inventory/allocation/v1
mkdir -p "$DEMO_DIR/proto/inventory/allocation/v1"
cat > "$DEMO_DIR/proto/inventory/allocation/v1/allocation.proto" <<'EOF'
syntax = "proto3";

package inventory.allocation.v1;

option go_package = "go.megamart.dev/apis/proto/inventory/allocation/v1;allocationpb";

// AllocationService allocates inventory to orders across warehouses.
service AllocationService {
  // AllocateOrder determines which warehouse(s) can fulfill an order.
  rpc AllocateOrder(AllocateOrderRequest) returns (AllocationPlan);

  // CommitAllocation confirms an allocation after packing.
  rpc CommitAllocation(CommitAllocationRequest) returns (CommitAllocationResponse);

  // GetAllocationPlan retrieves an existing plan.
  rpc GetAllocationPlan(GetAllocationPlanRequest) returns (AllocationPlan);
}

// AllocationPlan describes how an order's items are split across warehouses.
message AllocationPlan {
  string id = 1;
  string order_id = 2;
  repeated AllocationLine lines = 3;
  string status = 4; // planned, committed, partial, failed
  string created_at = 5;
}

// AllocationLine maps a SKU quantity to a specific warehouse.
message AllocationLine {
  string sku_id = 1;
  string warehouse_id = 2;
  int32 quantity = 3;
  string reservation_id = 4;
}

message AllocateOrderRequest {
  string order_id = 1;
  repeated OrderItem items = 2;
  string shipping_address_zip = 3;  // for proximity routing
}

message OrderItem {
  string sku_id = 1;
  int32 quantity = 2;
}

message CommitAllocationRequest { string allocation_plan_id = 1; }
message CommitAllocationResponse {}
message GetAllocationPlanRequest { string id = 1; }
EOF

# Proto: warehouse/picking/v1
mkdir -p "$DEMO_DIR/proto/warehouse/picking/v1"
cat > "$DEMO_DIR/proto/warehouse/picking/v1/picking.proto" <<'EOF'
syntax = "proto3";

package warehouse.picking.v1;

option go_package = "go.megamart.dev/apis/proto/warehouse/picking/v1;pickingpb";

// PickingService manages pick lists and bin-level operations.
service PickingService {
  // CreatePickList generates a pick list from allocation lines.
  rpc CreatePickList(CreatePickListRequest) returns (PickList);

  // GetPickList retrieves a pick list by ID.
  rpc GetPickList(GetPickListRequest) returns (PickList);

  // ConfirmPick marks an item as picked from its bin.
  rpc ConfirmPick(ConfirmPickRequest) returns (PickItem);

  // CompletePickList finalizes all picks and transitions to packing.
  rpc CompletePickList(CompletePickListRequest) returns (PickList);

  // ListPickLists returns pick lists for a warehouse.
  rpc ListPickLists(ListPickListsRequest) returns (ListPickListsResponse);
}

// PickList is a set of items to be collected from warehouse bins.
message PickList {
  string id = 1;
  string warehouse_id = 2;
  string order_id = 3;
  string allocation_plan_id = 4;
  repeated PickItem items = 5;
  PickListStatus status = 6;
  string assigned_to = 7;     // picker user ID
  string created_at = 8;
  string completed_at = 9;
}

// PickItem is a single line item to pick.
message PickItem {
  string sku_id = 1;
  string bin_location = 2;    // e.g. "A-12-3"
  int32 quantity_requested = 3;
  int32 quantity_picked = 4;
  bool confirmed = 5;
}

// PickListStatus tracks the state of a pick list.
enum PickListStatus {
  PICK_LIST_STATUS_UNSPECIFIED = 0;
  PICK_LIST_STATUS_PENDING = 1;
  PICK_LIST_STATUS_IN_PROGRESS = 2;
  PICK_LIST_STATUS_COMPLETED = 3;
  PICK_LIST_STATUS_CANCELLED = 4;
}

message CreatePickListRequest {
  string warehouse_id = 1;
  string order_id = 2;
  string allocation_plan_id = 3;
  repeated PickItem items = 4;
}
message GetPickListRequest { string id = 1; }
message ConfirmPickRequest {
  string pick_list_id = 1;
  string sku_id = 2;
  int32 quantity_picked = 3;
}
message CompletePickListRequest { string id = 1; }
message ListPickListsRequest {
  string warehouse_id = 1;
  PickListStatus status = 2;
  int32 page_size = 3;
  string page_token = 4;
}
message ListPickListsResponse {
  repeated PickList pick_lists = 1;
  string next_page_token = 2;
}
EOF

# Proto: warehouse/receiving/v1
mkdir -p "$DEMO_DIR/proto/warehouse/receiving/v1"
cat > "$DEMO_DIR/proto/warehouse/receiving/v1/receiving.proto" <<'EOF'
syntax = "proto3";

package warehouse.receiving.v1;

option go_package = "go.megamart.dev/apis/proto/warehouse/receiving/v1;receivingpb";

// ReceivingService handles inbound shipments from suppliers.
service ReceivingService {
  // CreateReceipt opens a new receiving record against a purchase order.
  rpc CreateReceipt(CreateReceiptRequest) returns (Receipt);

  // RecordLineItem records a line item as received and inspected.
  rpc RecordLineItem(RecordLineItemRequest) returns (ReceiptLineItem);

  // CompleteReceipt finalizes receiving and triggers putaway.
  rpc CompleteReceipt(CompleteReceiptRequest) returns (Receipt);

  // GetReceipt retrieves a receiving record.
  rpc GetReceipt(GetReceiptRequest) returns (Receipt);
}

// Receipt is an inbound receiving record.
message Receipt {
  string id = 1;
  string warehouse_id = 2;
  string purchase_order_id = 3;
  string supplier_id = 4;
  repeated ReceiptLineItem line_items = 5;
  string status = 6;   // open, completed, cancelled
  string received_at = 7;
}

// ReceiptLineItem is a single product line in a receipt.
message ReceiptLineItem {
  string sku_id = 1;
  int32 expected_quantity = 2;
  int32 received_quantity = 3;
  int32 damaged_quantity = 4;
  string bin_location = 5;      // putaway location
  InspectionStatus inspection = 6;
}

// InspectionStatus tracks QA inspection.
enum InspectionStatus {
  INSPECTION_STATUS_UNSPECIFIED = 0;
  INSPECTION_STATUS_PENDING = 1;
  INSPECTION_STATUS_PASSED = 2;
  INSPECTION_STATUS_FAILED = 3;
}

message CreateReceiptRequest {
  string warehouse_id = 1;
  string purchase_order_id = 2;
  string supplier_id = 3;
}
message RecordLineItemRequest {
  string receipt_id = 1;
  string sku_id = 2;
  int32 received_quantity = 3;
  int32 damaged_quantity = 4;
  string bin_location = 5;
}
message CompleteReceiptRequest { string id = 1; }
message GetReceiptRequest { string id = 1; }
EOF

# Proto: warehouse/transfers/v1
mkdir -p "$DEMO_DIR/proto/warehouse/transfers/v1"
cat > "$DEMO_DIR/proto/warehouse/transfers/v1/transfers.proto" <<'EOF'
syntax = "proto3";

package warehouse.transfers.v1;

option go_package = "go.megamart.dev/apis/proto/warehouse/transfers/v1;transferspb";

// TransferService manages inter-warehouse inventory transfers.
service TransferService {
  // CreateTransfer initiates a transfer between warehouses.
  rpc CreateTransfer(CreateTransferRequest) returns (Transfer);

  // ShipTransfer marks a transfer as shipped from the source.
  rpc ShipTransfer(ShipTransferRequest) returns (Transfer);

  // ReceiveTransfer marks a transfer as received at the destination.
  rpc ReceiveTransfer(ReceiveTransferRequest) returns (Transfer);

  // GetTransfer retrieves a transfer by ID.
  rpc GetTransfer(GetTransferRequest) returns (Transfer);

  // ListTransfers returns transfers for a warehouse.
  rpc ListTransfers(ListTransfersRequest) returns (ListTransfersResponse);
}

// Transfer represents an inter-warehouse stock movement.
message Transfer {
  string id = 1;
  string source_warehouse_id = 2;
  string destination_warehouse_id = 3;
  repeated TransferLine lines = 4;
  TransferStatus status = 5;
  string tracking_number = 6;
  string created_at = 7;
  string shipped_at = 8;
  string received_at = 9;
}

// TransferLine is a single SKU in a transfer.
message TransferLine {
  string sku_id = 1;
  int32 quantity_sent = 2;
  int32 quantity_received = 3;
}

// TransferStatus tracks the lifecycle of a transfer.
enum TransferStatus {
  TRANSFER_STATUS_UNSPECIFIED = 0;
  TRANSFER_STATUS_DRAFT = 1;
  TRANSFER_STATUS_SHIPPED = 2;
  TRANSFER_STATUS_IN_TRANSIT = 3;
  TRANSFER_STATUS_RECEIVED = 4;
  TRANSFER_STATUS_CANCELLED = 5;
}

message CreateTransferRequest {
  string source_warehouse_id = 1;
  string destination_warehouse_id = 2;
  repeated TransferLine lines = 3;
}
message ShipTransferRequest {
  string id = 1;
  string tracking_number = 2;
}
message ReceiveTransferRequest { string id = 1; }
message GetTransferRequest { string id = 1; }
message ListTransfersRequest {
  string warehouse_id = 1;
  TransferStatus status = 2;
  int32 page_size = 3;
  string page_token = 4;
}
message ListTransfersResponse {
  repeated Transfer transfers = 1;
  string next_page_token = 2;
}
EOF

# Proto: fulfillment/routing/v1
mkdir -p "$DEMO_DIR/proto/fulfillment/routing/v1"
cat > "$DEMO_DIR/proto/fulfillment/routing/v1/routing.proto" <<'EOF'
syntax = "proto3";

package fulfillment.routing.v1;

option go_package = "go.megamart.dev/apis/proto/fulfillment/routing/v1;routingpb";

// RoutingService assigns orders to the optimal warehouse for fulfillment.
service RoutingService {
  // RouteOrder determines the best warehouse(s) for an order.
  rpc RouteOrder(RouteOrderRequest) returns (RoutingDecision);

  // GetRoutingDecision retrieves a past routing decision.
  rpc GetRoutingDecision(GetRoutingDecisionRequest) returns (RoutingDecision);

  // ListWarehouses returns warehouses eligible for fulfillment.
  rpc ListWarehouses(ListWarehousesRequest) returns (ListWarehousesResponse);
}

// RoutingDecision describes the selected warehouse and reasoning.
message RoutingDecision {
  string id = 1;
  string order_id = 2;
  string selected_warehouse_id = 3;
  repeated WarehouseCandidate candidates = 4;
  string reason = 5;    // proximity, stock_availability, cost
  string decided_at = 6;
}

// WarehouseCandidate is a warehouse evaluated during routing.
message WarehouseCandidate {
  string warehouse_id = 1;
  string name = 2;
  double distance_km = 3;
  bool has_full_stock = 4;
  double estimated_cost = 5;
  int32 score = 6;
}

// Warehouse is a fulfillment center.
message Warehouse {
  string id = 1;
  string name = 2;
  string address = 3;
  string zip_code = 4;
  string region = 5;
  bool active = 6;
  repeated string capabilities = 7; // standard, refrigerated, hazmat, oversized
}

message RouteOrderRequest {
  string order_id = 1;
  repeated OrderItem items = 2;
  string shipping_zip = 3;
  string shipping_speed = 4;  // standard, express, overnight
}

message OrderItem {
  string sku_id = 1;
  int32 quantity = 2;
}

message GetRoutingDecisionRequest { string id = 1; }
message ListWarehousesRequest {
  string region = 1;
  bool active_only = 2;
}
message ListWarehousesResponse {
  repeated Warehouse warehouses = 1;
}
EOF

# Proto: fulfillment/packing/v1
mkdir -p "$DEMO_DIR/proto/fulfillment/packing/v1"
cat > "$DEMO_DIR/proto/fulfillment/packing/v1/packing.proto" <<'EOF'
syntax = "proto3";

package fulfillment.packing.v1;

option go_package = "go.megamart.dev/apis/proto/fulfillment/packing/v1;packingpb";

// PackingService manages the packing stage of order fulfillment.
service PackingService {
  // CreatePackSession starts a new packing session from a pick list.
  rpc CreatePackSession(CreatePackSessionRequest) returns (PackSession);

  // ScanItem records a scanned item during packing verification.
  rpc ScanItem(ScanItemRequest) returns (ScanItemResponse);

  // SelectBox chooses the appropriate box size for the shipment.
  rpc SelectBox(SelectBoxRequest) returns (BoxRecommendation);

  // SealAndWeigh records final weight and seals the package.
  rpc SealAndWeigh(SealAndWeighRequest) returns (PackSession);

  // GetPackSession retrieves a packing session.
  rpc GetPackSession(GetPackSessionRequest) returns (PackSession);
}

// PackSession tracks the packing of a single shipment.
message PackSession {
  string id = 1;
  string pick_list_id = 2;
  string order_id = 3;
  string packer_id = 4;
  repeated PackedItem items = 5;
  string box_type = 6;
  double weight_kg = 7;
  PackSessionStatus status = 8;
  string started_at = 9;
  string completed_at = 10;
}

// PackedItem is a verified item in the box.
message PackedItem {
  string sku_id = 1;
  int32 quantity = 2;
  bool verified = 3;
}

// BoxRecommendation suggests a box size.
message BoxRecommendation {
  string box_type = 1;         // small, medium, large, oversized
  double length_cm = 2;
  double width_cm = 3;
  double height_cm = 4;
  double max_weight_kg = 5;
}

// PackSessionStatus tracks packing progress.
enum PackSessionStatus {
  PACK_SESSION_STATUS_UNSPECIFIED = 0;
  PACK_SESSION_STATUS_IN_PROGRESS = 1;
  PACK_SESSION_STATUS_SEALED = 2;
  PACK_SESSION_STATUS_CANCELLED = 3;
}

message CreatePackSessionRequest {
  string pick_list_id = 1;
  string packer_id = 2;
}
message ScanItemRequest {
  string pack_session_id = 1;
  string barcode = 2;
}
message ScanItemResponse {
  string sku_id = 1;
  bool match = 2;
  string error_message = 3;
}
message SelectBoxRequest {
  string pack_session_id = 1;
}
message SealAndWeighRequest {
  string pack_session_id = 1;
  double weight_kg = 2;
}
message GetPackSessionRequest { string id = 1; }
EOF

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# OpenAPI schemas (external REST APIs)
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

# OpenAPI: iam/users/v1
mkdir -p "$DEMO_DIR/openapi/iam/users/v1"
cat > "$DEMO_DIR/openapi/iam/users/v1/users.yaml" <<'EOF'
openapi: "3.0.3"
info:
  title: Users API
  version: "1.8.0"
  description: User registration, profile management, password resets, and account lifecycle.
paths:
  /users:
    get:
      summary: List users
      operationId: listUsers
      parameters:
        - name: email
          in: query
          description: Filter by email address
        - name: status
          in: query
          description: "Filter by status: active, suspended, pending"
        - name: limit
          in: query
          schema: { type: integer }
        - name: page_token
          in: query
      responses:
        "200":
          description: Paginated list of users
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/User'
    post:
      summary: Register a new user
      operationId: createUser
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/CreateUserRequest'
      responses:
        "201":
          description: User created
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/User'
        "409":
          description: Email already registered
  /users/{id}:
    get:
      summary: Get user by ID
      operationId: getUser
      parameters:
        - name: id
          in: path
          required: true
      responses:
        "200":
          description: User details
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/User'
        "404":
          description: User not found
    patch:
      summary: Update user profile
      operationId: updateUser
      parameters:
        - name: id
          in: path
          required: true
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/UpdateUserRequest'
      responses:
        "200":
          description: Updated user
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/User'
    delete:
      summary: Deactivate user account
      operationId: deactivateUser
      parameters:
        - name: id
          in: path
          required: true
      responses:
        "204":
          description: User deactivated
  /users/{id}/password-reset:
    post:
      summary: Initiate password reset
      operationId: requestPasswordReset
      parameters:
        - name: id
          in: path
          required: true
      responses:
        "202":
          description: Password reset email sent
  /users/{id}/verify-email:
    post:
      summary: Verify email address
      operationId: verifyEmail
      parameters:
        - name: id
          in: path
          required: true
      responses:
        "200":
          description: Email verified
components:
  schemas:
    User:
      type: object
      required: [id, email, status]
      properties:
        id:
          type: string
          description: Unique user identifier
        email:
          type: string
          format: email
          description: User email address
        display_name:
          type: string
          description: Display name
        avatar_url:
          type: string
          format: uri
        status:
          type: string
          description: "Account status: active, suspended, pending_verification"
        email_verified:
          type: boolean
        created_at:
          type: string
          format: date-time
        last_login_at:
          type: string
          format: date-time
    CreateUserRequest:
      type: object
      required: [email, password, display_name]
      properties:
        email:
          type: string
          format: email
        password:
          type: string
          format: password
          minLength: 8
        display_name:
          type: string
    UpdateUserRequest:
      type: object
      properties:
        display_name:
          type: string
        avatar_url:
          type: string
          format: uri
EOF

# OpenAPI: iam/groups/v1
mkdir -p "$DEMO_DIR/openapi/iam/groups/v1"
cat > "$DEMO_DIR/openapi/iam/groups/v1/groups.yaml" <<'EOF'
openapi: "3.0.3"
info:
  title: Groups API
  version: "1.3.0"
  description: User group management — create groups, assign members, and list memberships.
paths:
  /groups:
    get:
      summary: List groups
      operationId: listGroups
      parameters:
        - name: limit
          in: query
        - name: page_token
          in: query
      responses:
        "200":
          description: List of groups
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/Group'
    post:
      summary: Create a group
      operationId: createGroup
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/CreateGroupRequest'
      responses:
        "201":
          description: Group created
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Group'
  /groups/{id}:
    get:
      summary: Get group by ID
      operationId: getGroup
      parameters:
        - name: id
          in: path
          required: true
      responses:
        "200":
          description: Group details
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Group'
    delete:
      summary: Delete a group
      operationId: deleteGroup
      parameters:
        - name: id
          in: path
          required: true
      responses:
        "204":
          description: Group deleted
  /groups/{id}/members:
    get:
      summary: List group members
      operationId: listGroupMembers
      parameters:
        - name: id
          in: path
          required: true
      responses:
        "200":
          description: List of members
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/GroupMember'
    post:
      summary: Add member to group
      operationId: addGroupMember
      parameters:
        - name: id
          in: path
          required: true
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/AddGroupMemberRequest'
      responses:
        "201":
          description: Member added
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/GroupMember'
  /groups/{id}/members/{user_id}:
    delete:
      summary: Remove member from group
      operationId: removeGroupMember
      parameters:
        - name: id
          in: path
          required: true
        - name: user_id
          in: path
          required: true
      responses:
        "204":
          description: Member removed
components:
  schemas:
    Group:
      type: object
      required: [id, name]
      properties:
        id:
          type: string
        name:
          type: string
          description: Group display name
        description:
          type: string
        member_count:
          type: integer
        created_at:
          type: string
          format: date-time
    GroupMember:
      type: object
      properties:
        user_id:
          type: string
        role:
          type: string
          description: "Role within group: admin, member"
        added_at:
          type: string
          format: date-time
    CreateGroupRequest:
      type: object
      required: [name]
      properties:
        name:
          type: string
          description: Group display name
        description:
          type: string
    AddGroupMemberRequest:
      type: object
      required: [user_id]
      properties:
        user_id:
          type: string
        role:
          type: string
          description: "Role within group: admin, member"
EOF

# OpenAPI: iam/access-policies/v1
mkdir -p "$DEMO_DIR/openapi/iam/access-policies/v1"
cat > "$DEMO_DIR/openapi/iam/access-policies/v1/access-policies.yaml" <<'EOF'
openapi: "3.0.3"
info:
  title: Access Policies API
  version: "1.5.0"
  description: RBAC access policies — define rules binding principals to permissions on resources.
paths:
  /policies:
    get:
      summary: List access policies
      operationId: listPolicies
      parameters:
        - name: principal_type
          in: query
          description: "Filter by principal: user, group, service_account"
        - name: resource_type
          in: query
      responses:
        "200":
          description: List of policies
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/AccessPolicy'
    post:
      summary: Create an access policy
      operationId: createPolicy
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/CreatePolicyRequest'
      responses:
        "201":
          description: Policy created
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/AccessPolicy'
  /policies/{id}:
    get:
      summary: Get policy by ID
      operationId: getPolicy
      parameters:
        - name: id
          in: path
          required: true
      responses:
        "200":
          description: Policy details
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/AccessPolicy'
    put:
      summary: Replace a policy
      operationId: updatePolicy
      parameters:
        - name: id
          in: path
          required: true
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/UpdatePolicyRequest'
      responses:
        "200":
          description: Policy updated
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/AccessPolicy'
    delete:
      summary: Delete a policy
      operationId: deletePolicy
      parameters:
        - name: id
          in: path
          required: true
      responses:
        "204":
          description: Policy deleted
  /policies/evaluate:
    post:
      summary: Evaluate whether an action is allowed
      operationId: evaluateAccess
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/EvaluateRequest'
      responses:
        "200":
          description: Access decision
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/EvaluateResponse'
components:
  schemas:
    AccessPolicy:
      type: object
      required: [id, principal, resource, actions, effect]
      properties:
        id:
          type: string
        principal:
          $ref: "#/components/schemas/Principal"
        resource:
          type: string
          description: "Resource pattern (e.g., orders/*, catalog/products/123)"
        actions:
          type: array
          items: { type: string }
          description: "Allowed actions: read, write, delete, admin"
        effect:
          type: string
          description: "Policy effect: allow or deny"
        conditions:
          type: object
          description: Optional conditions (IP range, time window)
    Principal:
      type: object
      properties:
        type:
          type: string
          description: "Principal type: user, group, service_account"
        id:
          type: string
          description: Principal identifier
    EvaluateRequest:
      type: object
      required: [principal, resource, action]
      properties:
        principal:
          $ref: "#/components/schemas/Principal"
        resource:
          type: string
        action:
          type: string
    EvaluateResponse:
      type: object
      properties:
        allowed:
          type: boolean
        matched_policy_id:
          type: string
    CreatePolicyRequest:
      type: object
      required: [principal, resource, actions, effect]
      properties:
        principal:
          $ref: "#/components/schemas/Principal"
        resource:
          type: string
          description: "Resource pattern (e.g., orders/*, catalog/products/123)"
        actions:
          type: array
          items: { type: string }
        effect:
          type: string
          description: "Policy effect: allow or deny"
        conditions:
          type: object
          description: Optional conditions (IP range, time window)
    UpdatePolicyRequest:
      type: object
      required: [principal, resource, actions, effect]
      properties:
        principal:
          $ref: "#/components/schemas/Principal"
        resource:
          type: string
        actions:
          type: array
          items: { type: string }
        effect:
          type: string
        conditions:
          type: object
EOF

# OpenAPI: iam/compartments/v1
mkdir -p "$DEMO_DIR/openapi/iam/compartments/v1"
cat > "$DEMO_DIR/openapi/iam/compartments/v1/compartments.yaml" <<'EOF'
openapi: "3.0.3"
info:
  title: Compartments API
  version: "1.0.0-beta.2"
  description: Resource compartments for multi-tenancy isolation. Compartments form a hierarchy for organizing and isolating resources.
paths:
  /compartments:
    get:
      summary: List compartments
      operationId: listCompartments
      parameters:
        - name: parent_id
          in: query
          description: Filter by parent compartment
      responses:
        "200":
          description: List of compartments
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/Compartment'
    post:
      summary: Create a compartment
      operationId: createCompartment
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/CreateCompartmentRequest'
      responses:
        "201":
          description: Compartment created
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Compartment'
  /compartments/{id}:
    get:
      summary: Get compartment by ID
      operationId: getCompartment
      parameters:
        - name: id
          in: path
          required: true
      responses:
        "200":
          description: Compartment details
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Compartment'
    patch:
      summary: Update compartment
      operationId: updateCompartment
      parameters:
        - name: id
          in: path
          required: true
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/UpdateCompartmentRequest'
      responses:
        "200":
          description: Compartment updated
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Compartment'
    delete:
      summary: Delete compartment
      operationId: deleteCompartment
      parameters:
        - name: id
          in: path
          required: true
      responses:
        "204":
          description: Compartment deleted
components:
  schemas:
    Compartment:
      type: object
      required: [id, name]
      properties:
        id:
          type: string
        name:
          type: string
          description: Display name
        description:
          type: string
        parent_id:
          type: string
          description: Parent compartment ID (null for root)
        path:
          type: string
          description: "Full hierarchy path (e.g., /root/engineering/backend)"
        created_at:
          type: string
          format: date-time
    CreateCompartmentRequest:
      type: object
      required: [name]
      properties:
        name:
          type: string
          description: Display name
        description:
          type: string
        parent_id:
          type: string
          description: Parent compartment ID (null for root)
    UpdateCompartmentRequest:
      type: object
      properties:
        name:
          type: string
        description:
          type: string
EOF

# OpenAPI: iam/service-accounts/v1
mkdir -p "$DEMO_DIR/openapi/iam/service-accounts/v1"
cat > "$DEMO_DIR/openapi/iam/service-accounts/v1/service-accounts.yaml" <<'EOF'
openapi: "3.0.3"
info:
  title: Service Accounts API
  version: "1.1.0"
  description: Machine-to-machine service accounts and API key management.
paths:
  /service-accounts:
    get:
      summary: List service accounts
      operationId: listServiceAccounts
      responses:
        "200":
          description: List of service accounts
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/ServiceAccount'
    post:
      summary: Create a service account
      operationId: createServiceAccount
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/CreateServiceAccountRequest'
      responses:
        "201":
          description: Service account created
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ServiceAccount'
  /service-accounts/{id}:
    get:
      summary: Get service account
      operationId: getServiceAccount
      parameters:
        - name: id
          in: path
          required: true
      responses:
        "200":
          description: Service account details
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ServiceAccount'
    delete:
      summary: Delete service account
      operationId: deleteServiceAccount
      parameters:
        - name: id
          in: path
          required: true
      responses:
        "204":
          description: Deleted
  /service-accounts/{id}/keys:
    get:
      summary: List API keys
      operationId: listApiKeys
      parameters:
        - name: id
          in: path
          required: true
      responses:
        "200":
          description: API keys for this service account
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/ApiKey'
    post:
      summary: Create API key
      operationId: createApiKey
      parameters:
        - name: id
          in: path
          required: true
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/CreateApiKeyRequest'
      responses:
        "201":
          description: API key created (secret shown once)
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ApiKey'
  /service-accounts/{id}/keys/{key_id}:
    delete:
      summary: Revoke API key
      operationId: revokeApiKey
      parameters:
        - name: id
          in: path
          required: true
        - name: key_id
          in: path
          required: true
      responses:
        "204":
          description: API key revoked
components:
  schemas:
    ServiceAccount:
      type: object
      required: [id, name]
      properties:
        id:
          type: string
        name:
          type: string
          description: Service account name
        description:
          type: string
        active:
          type: boolean
        created_at:
          type: string
          format: date-time
    ApiKey:
      type: object
      properties:
        id:
          type: string
        prefix:
          type: string
          description: Key prefix for identification (first 8 chars)
        created_at:
          type: string
          format: date-time
        expires_at:
          type: string
          format: date-time
        last_used_at:
          type: string
          format: date-time
    CreateServiceAccountRequest:
      type: object
      required: [name]
      properties:
        name:
          type: string
          description: Service account name
        description:
          type: string
    CreateApiKeyRequest:
      type: object
      properties:
        expires_at:
          type: string
          format: date-time
          description: Optional expiration date
EOF

# OpenAPI: catalog/products/v1 (deprecated)
mkdir -p "$DEMO_DIR/openapi/catalog/products/v1"
cat > "$DEMO_DIR/openapi/catalog/products/v1/products.yaml" <<'EOF'
openapi: "3.0.3"
info:
  title: Products API (Deprecated)
  version: "1.9.0"
  description: "DEPRECATED: Legacy product listings. Use catalog/products/v2 for variants and rich attributes."
paths:
  /products:
    get:
      summary: List products
      operationId: listProducts
      deprecated: true
      parameters:
        - name: category
          in: query
        - name: limit
          in: query
      responses:
        "200":
          description: List of products
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/Product'
  /products/{id}:
    get:
      summary: Get product by ID
      operationId: getProduct
      deprecated: true
      parameters:
        - name: id
          in: path
          required: true
      responses:
        "200":
          description: Product details
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Product'
        "404":
          description: Not found
components:
  schemas:
    Product:
      type: object
      required: [id, name, price_cents]
      properties:
        id:
          type: string
        name:
          type: string
          description: Product name
        description:
          type: string
        price_cents:
          type: integer
          format: int64
          description: Price in cents
        category:
          type: string
        in_stock:
          type: boolean
EOF

# OpenAPI: catalog/products/v2
mkdir -p "$DEMO_DIR/openapi/catalog/products/v2"
cat > "$DEMO_DIR/openapi/catalog/products/v2/products.yaml" <<'EOF'
openapi: "3.0.3"
info:
  title: Products API
  version: "2.3.0"
  description: Product catalog with variants, rich attributes, localization, and multi-image support.
paths:
  /products:
    get:
      summary: List products
      operationId: listProducts
      parameters:
        - name: category_id
          in: query
        - name: brand
          in: query
        - name: min_price
          in: query
          schema: { type: integer }
        - name: max_price
          in: query
          schema: { type: integer }
        - name: in_stock
          in: query
          schema: { type: boolean }
        - name: limit
          in: query
        - name: page_token
          in: query
      responses:
        "200":
          description: Paginated product list
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/Product'
    post:
      summary: Create a product
      operationId: createProduct
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/CreateProductRequest'
      responses:
        "201":
          description: Product created
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Product'
  /products/{id}:
    get:
      summary: Get product by ID
      operationId: getProduct
      parameters:
        - name: id
          in: path
          required: true
        - name: locale
          in: query
          description: Locale for localized content (e.g., en-US, es-MX)
      responses:
        "200":
          description: Product details with variants
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Product'
        "404":
          description: Not found
    put:
      summary: Update product
      operationId: updateProduct
      parameters:
        - name: id
          in: path
          required: true
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/UpdateProductRequest'
      responses:
        "200":
          description: Updated product
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Product'
    delete:
      summary: Archive product
      operationId: archiveProduct
      parameters:
        - name: id
          in: path
          required: true
      responses:
        "204":
          description: Product archived
  /products/{id}/variants:
    get:
      summary: List product variants
      operationId: listVariants
      parameters:
        - name: id
          in: path
          required: true
      responses:
        "200":
          description: List of variants
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/Variant'
    post:
      summary: Add a variant
      operationId: createVariant
      parameters:
        - name: id
          in: path
          required: true
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/CreateVariantRequest'
      responses:
        "201":
          description: Variant created
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Variant'
components:
  schemas:
    Product:
      type: object
      required: [id, name, status]
      properties:
        id:
          type: string
        name:
          type: string
          description: Product display name
        description:
          type: string
        brand:
          type: string
        category_id:
          type: string
        status:
          type: string
          description: "Product status: active, draft, archived"
        attributes:
          type: object
          description: Key-value product attributes (color, material, etc.)
        images:
          type: array
          items:
            $ref: "#/components/schemas/ProductImage"
        variants:
          type: array
          items:
            $ref: "#/components/schemas/Variant"
        created_at:
          type: string
          format: date-time
    Variant:
      type: object
      required: [id, sku_id, price_cents]
      properties:
        id:
          type: string
        sku_id:
          type: string
        name:
          type: string
          description: "Variant label (e.g., Large / Blue)"
        price_cents:
          type: integer
          format: int64
        compare_at_price_cents:
          type: integer
          format: int64
          description: Original price for showing discounts
        attributes:
          type: object
          description: "Variant-specific attributes (size, color)"
        in_stock:
          type: boolean
    ProductImage:
      type: object
      properties:
        url:
          type: string
          format: uri
        alt_text:
          type: string
        position:
          type: integer
          description: Display order
    CreateProductRequest:
      type: object
      required: [name]
      properties:
        name:
          type: string
          description: Product display name
        description:
          type: string
        brand:
          type: string
        category_id:
          type: string
        attributes:
          type: object
          description: Key-value product attributes (color, material, etc.)
    UpdateProductRequest:
      type: object
      properties:
        name:
          type: string
        description:
          type: string
        brand:
          type: string
        category_id:
          type: string
        status:
          type: string
          description: "Product status: active, draft, archived"
        attributes:
          type: object
    CreateVariantRequest:
      type: object
      required: [sku_id, price_cents]
      properties:
        sku_id:
          type: string
        name:
          type: string
          description: "Variant label (e.g., Large / Blue)"
        price_cents:
          type: integer
          format: int64
        compare_at_price_cents:
          type: integer
          format: int64
        attributes:
          type: object
          description: "Variant-specific attributes (size, color)"
EOF

# OpenAPI: catalog/categories/v1
mkdir -p "$DEMO_DIR/openapi/catalog/categories/v1"
cat > "$DEMO_DIR/openapi/catalog/categories/v1/categories.yaml" <<'EOF'
openapi: "3.0.3"
info:
  title: Categories API
  version: "1.2.0"
  description: Hierarchical category taxonomy for organizing products.
paths:
  /categories:
    get:
      summary: List top-level categories
      operationId: listCategories
      responses:
        "200":
          description: Category tree
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/Category'
    post:
      summary: Create a category
      operationId: createCategory
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/CreateCategoryRequest'
      responses:
        "201":
          description: Category created
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Category'
  /categories/{id}:
    get:
      summary: Get category with children
      operationId: getCategory
      parameters:
        - name: id
          in: path
          required: true
        - name: depth
          in: query
          description: How many levels of children to include
          schema: { type: integer }
      responses:
        "200":
          description: Category with nested children
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Category'
    patch:
      summary: Update category
      operationId: updateCategory
      parameters:
        - name: id
          in: path
          required: true
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/UpdateCategoryRequest'
      responses:
        "200":
          description: Updated category
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Category'
    delete:
      summary: Delete category
      operationId: deleteCategory
      parameters:
        - name: id
          in: path
          required: true
      responses:
        "204":
          description: Category deleted
components:
  schemas:
    Category:
      type: object
      required: [id, name, slug]
      properties:
        id:
          type: string
        name:
          type: string
          description: Category display name
        slug:
          type: string
          description: URL-safe slug
        description:
          type: string
        parent_id:
          type: string
        image_url:
          type: string
          format: uri
        product_count:
          type: integer
          description: Number of products in this category
        children:
          type: array
          items:
            $ref: "#/components/schemas/Category"
    CreateCategoryRequest:
      type: object
      required: [name, slug]
      properties:
        name:
          type: string
          description: Category display name
        slug:
          type: string
          description: URL-safe slug
        description:
          type: string
        parent_id:
          type: string
        image_url:
          type: string
          format: uri
    UpdateCategoryRequest:
      type: object
      properties:
        name:
          type: string
        slug:
          type: string
        description:
          type: string
        parent_id:
          type: string
        image_url:
          type: string
          format: uri
EOF

# OpenAPI: catalog/images/v1
mkdir -p "$DEMO_DIR/openapi/catalog/images/v1"
cat > "$DEMO_DIR/openapi/catalog/images/v1/images.yaml" <<'EOF'
openapi: "3.0.3"
info:
  title: Images API
  version: "1.4.0"
  description: Product image upload, transformation, thumbnail generation, and CDN delivery.
paths:
  /images:
    post:
      summary: Upload an image
      operationId: uploadImage
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/UploadImageRequest'
      responses:
        "201":
          description: Image uploaded and processing started
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Image'
  /images/{id}:
    get:
      summary: Get image metadata
      operationId: getImage
      parameters:
        - name: id
          in: path
          required: true
      responses:
        "200":
          description: Image metadata with CDN URLs
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Image'
    delete:
      summary: Delete an image
      operationId: deleteImage
      parameters:
        - name: id
          in: path
          required: true
      responses:
        "204":
          description: Image deleted
  /images/{id}/transform:
    post:
      summary: Request an image transformation
      operationId: transformImage
      parameters:
        - name: id
          in: path
          required: true
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/TransformImageRequest'
      responses:
        "202":
          description: Transformation queued
  /products/{product_id}/images:
    get:
      summary: List images for a product
      operationId: listProductImages
      parameters:
        - name: product_id
          in: path
          required: true
      responses:
        "200":
          description: Product images
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/Image'
components:
  schemas:
    Image:
      type: object
      required: [id, original_url, status]
      properties:
        id:
          type: string
        product_id:
          type: string
        original_url:
          type: string
          format: uri
        thumbnail_url:
          type: string
          format: uri
        medium_url:
          type: string
          format: uri
        large_url:
          type: string
          format: uri
        alt_text:
          type: string
        width:
          type: integer
        height:
          type: integer
        file_size_bytes:
          type: integer
        content_type:
          type: string
          description: "MIME type (image/jpeg, image/png, image/webp)"
        status:
          type: string
          description: "Processing status: pending, ready, failed"
        created_at:
          type: string
          format: date-time
    UploadImageRequest:
      type: object
      required: [product_id, original_url]
      properties:
        product_id:
          type: string
        original_url:
          type: string
          format: uri
        alt_text:
          type: string
    TransformImageRequest:
      type: object
      required: [operations]
      properties:
        operations:
          type: array
          items:
            type: object
            properties:
              type:
                type: string
                description: "Operation type: resize, crop, rotate, watermark"
              params:
                type: object
                description: Operation-specific parameters
EOF

# OpenAPI: catalog/skus/v1
mkdir -p "$DEMO_DIR/openapi/catalog/skus/v1"
cat > "$DEMO_DIR/openapi/catalog/skus/v1/skus.yaml" <<'EOF'
openapi: "3.0.3"
info:
  title: SKUs API
  version: "1.1.0"
  description: SKU management — inventory units, pricing tiers, barcodes, and weight/dimensions.
paths:
  /skus:
    get:
      summary: List SKUs
      operationId: listSkus
      parameters:
        - name: product_id
          in: query
        - name: active
          in: query
          schema: { type: boolean }
        - name: limit
          in: query
        - name: page_token
          in: query
      responses:
        "200":
          description: Paginated SKU list
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/SKU'
    post:
      summary: Create a SKU
      operationId: createSku
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/CreateSkuRequest'
      responses:
        "201":
          description: SKU created
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/SKU'
  /skus/{id}:
    get:
      summary: Get SKU by ID
      operationId: getSku
      parameters:
        - name: id
          in: path
          required: true
      responses:
        "200":
          description: SKU details
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/SKU'
    patch:
      summary: Update SKU
      operationId: updateSku
      parameters:
        - name: id
          in: path
          required: true
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/UpdateSkuRequest'
      responses:
        "200":
          description: Updated SKU
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/SKU'
  /skus/{id}/pricing:
    put:
      summary: Set pricing tiers
      operationId: setPricing
      parameters:
        - name: id
          in: path
          required: true
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/SetPricingRequest'
      responses:
        "200":
          description: Pricing updated
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/SKU'
components:
  schemas:
    SKU:
      type: object
      required: [id, product_id, price_cents]
      properties:
        id:
          type: string
          description: Unique SKU identifier
        product_id:
          type: string
        barcode:
          type: string
          description: UPC or EAN barcode
        price_cents:
          type: integer
          format: int64
        cost_cents:
          type: integer
          format: int64
          description: Unit cost for margin calculation
        weight_grams:
          type: integer
        length_cm:
          type: number
        width_cm:
          type: number
        height_cm:
          type: number
        active:
          type: boolean
        pricing_tiers:
          type: array
          items:
            $ref: "#/components/schemas/PricingTier"
    PricingTier:
      type: object
      properties:
        min_quantity:
          type: integer
        price_cents:
          type: integer
          format: int64
          description: Price per unit at this tier
    CreateSkuRequest:
      type: object
      required: [product_id, price_cents]
      properties:
        product_id:
          type: string
        barcode:
          type: string
          description: UPC or EAN barcode
        price_cents:
          type: integer
          format: int64
        cost_cents:
          type: integer
          format: int64
        weight_grams:
          type: integer
        length_cm:
          type: number
        width_cm:
          type: number
        height_cm:
          type: number
    UpdateSkuRequest:
      type: object
      properties:
        barcode:
          type: string
        price_cents:
          type: integer
          format: int64
        cost_cents:
          type: integer
          format: int64
        weight_grams:
          type: integer
        active:
          type: boolean
    SetPricingRequest:
      type: object
      required: [tiers]
      properties:
        tiers:
          type: array
          items:
            $ref: "#/components/schemas/PricingTier"
EOF

# OpenAPI: catalog/search/v0
mkdir -p "$DEMO_DIR/openapi/catalog/search/v0"
cat > "$DEMO_DIR/openapi/catalog/search/v0/search.yaml" <<'EOF'
openapi: "3.0.3"
info:
  title: Product Search API
  version: "0.3.0-alpha.1"
  description: "EXPERIMENTAL: Full-text product search with facets, autocomplete, and relevance ranking."
paths:
  /search:
    get:
      summary: Search products
      operationId: searchProducts
      parameters:
        - name: q
          in: query
          required: true
          description: Search query string
        - name: category
          in: query
        - name: min_price
          in: query
          schema: { type: integer }
        - name: max_price
          in: query
          schema: { type: integer }
        - name: sort
          in: query
          description: "Sort: relevance, price_asc, price_desc, newest"
        - name: limit
          in: query
        - name: offset
          in: query
      responses:
        "200":
          description: Search results with facets
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/SearchResult'
  /search/autocomplete:
    get:
      summary: Autocomplete suggestions
      operationId: autocomplete
      parameters:
        - name: q
          in: query
          required: true
          description: Partial query for suggestions
        - name: limit
          in: query
          schema: { type: integer, default: 5 }
      responses:
        "200":
          description: Autocomplete suggestions
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/AutocompleteSuggestion'
  /search/facets:
    get:
      summary: Get available facets for a query
      operationId: getFacets
      parameters:
        - name: q
          in: query
      responses:
        "200":
          description: Facet counts
          content:
            application/json:
              schema:
                type: object
                description: Facet counts by category, brand, price range
components:
  schemas:
    SearchResult:
      type: object
      properties:
        total_count:
          type: integer
        results:
          type: array
          items:
            $ref: "#/components/schemas/SearchHit"
        facets:
          type: object
          description: Facet counts by category, brand, price range
    SearchHit:
      type: object
      properties:
        product_id:
          type: string
        name:
          type: string
        description:
          type: string
        price_cents:
          type: integer
        image_url:
          type: string
        score:
          type: number
          description: Relevance score
    AutocompleteSuggestion:
      type: object
      properties:
        text:
          type: string
        type:
          type: string
          description: "Suggestion type: product, category, brand"
EOF

# OpenAPI: orders/checkout/v1
mkdir -p "$DEMO_DIR/openapi/orders/checkout/v1"
cat > "$DEMO_DIR/openapi/orders/checkout/v1/checkout.yaml" <<'EOF'
openapi: "3.0.3"
info:
  title: Checkout API
  version: "1.6.0"
  description: Shopping cart checkout flow — cart management, address validation, payment, and order creation.
paths:
  /carts/{cart_id}:
    get:
      summary: Get cart contents
      operationId: getCart
      parameters:
        - name: cart_id
          in: path
          required: true
      responses:
        "200":
          description: Cart with items and totals
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Cart'
  /carts/{cart_id}/items:
    post:
      summary: Add item to cart
      operationId: addCartItem
      parameters:
        - name: cart_id
          in: path
          required: true
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/AddCartItemRequest'
      responses:
        "200":
          description: Updated cart
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Cart'
    delete:
      summary: Remove item from cart
      operationId: removeCartItem
      parameters:
        - name: cart_id
          in: path
          required: true
      responses:
        "200":
          description: Updated cart
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Cart'
  /carts/{cart_id}/checkout:
    post:
      summary: Submit checkout
      operationId: checkout
      parameters:
        - name: cart_id
          in: path
          required: true
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/CheckoutRequest'
      responses:
        "201":
          description: Order created
        "400":
          description: Validation errors (out of stock, invalid address)
        "402":
          description: Payment failed
  /checkout/validate-address:
    post:
      summary: Validate a shipping address
      operationId: validateAddress
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/Address'
      responses:
        "200":
          description: Address validation result
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/AddressValidationResult'
components:
  schemas:
    Cart:
      type: object
      required: [id, items]
      properties:
        id:
          type: string
        user_id:
          type: string
        items:
          type: array
          items:
            $ref: "#/components/schemas/CartItem"
        subtotal_cents:
          type: integer
        tax_cents:
          type: integer
        shipping_cents:
          type: integer
        total_cents:
          type: integer
        currency:
          type: string
    CartItem:
      type: object
      properties:
        sku_id:
          type: string
        product_name:
          type: string
        quantity:
          type: integer
        unit_price_cents:
          type: integer
        line_total_cents:
          type: integer
    CheckoutRequest:
      type: object
      required: [shipping_address, payment_method_id]
      properties:
        shipping_address:
          $ref: "#/components/schemas/Address"
        billing_address:
          $ref: "#/components/schemas/Address"
        payment_method_id:
          type: string
        promo_code:
          type: string
    Address:
      type: object
      required: [line1, city, state, zip, country]
      properties:
        line1:
          type: string
        line2:
          type: string
        city:
          type: string
        state:
          type: string
        zip:
          type: string
        country:
          type: string
    AddCartItemRequest:
      type: object
      required: [sku_id, quantity]
      properties:
        sku_id:
          type: string
        quantity:
          type: integer
    AddressValidationResult:
      type: object
      properties:
        valid:
          type: boolean
        normalized_address:
          $ref: "#/components/schemas/Address"
        suggestions:
          type: array
          items:
            $ref: "#/components/schemas/Address"
EOF

# OpenAPI: orders/orders/v1 (sunset)
mkdir -p "$DEMO_DIR/openapi/orders/orders/v1"
cat > "$DEMO_DIR/openapi/orders/orders/v1/orders.yaml" <<'EOF'
openapi: "3.0.3"
info:
  title: Orders API (Sunset)
  version: "1.12.0"
  description: "SUNSET: Legacy order management. Use orders/v2 for partial fulfillment and split shipments."
paths:
  /orders:
    get:
      summary: List orders
      operationId: listOrders
      deprecated: true
      parameters:
        - name: customer_id
          in: query
        - name: status
          in: query
      responses:
        "200":
          description: List of orders
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/Order'
  /orders/{id}:
    get:
      summary: Get order by ID
      operationId: getOrder
      deprecated: true
      parameters:
        - name: id
          in: path
          required: true
      responses:
        "200":
          description: Order details
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Order'
components:
  schemas:
    Order:
      type: object
      required: [id, customer_id, status, total_cents]
      properties:
        id:
          type: string
        customer_id:
          type: string
        status:
          type: string
        total_cents:
          type: integer
        currency:
          type: string
        created_at:
          type: string
          format: date-time
EOF

# OpenAPI: orders/orders/v2
mkdir -p "$DEMO_DIR/openapi/orders/orders/v2"
cat > "$DEMO_DIR/openapi/orders/orders/v2/orders.yaml" <<'EOF'
openapi: "3.0.3"
info:
  title: Orders API
  version: "2.1.0"
  description: Enhanced order management with partial fulfillment, split shipments, and detailed status tracking.
paths:
  /orders:
    get:
      summary: List orders
      operationId: listOrders
      parameters:
        - name: customer_id
          in: query
        - name: status
          in: query
        - name: created_after
          in: query
          schema: { type: string, format: date-time }
        - name: limit
          in: query
        - name: page_token
          in: query
      responses:
        "200":
          description: Paginated orders
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/Order'
  /orders/{id}:
    get:
      summary: Get order by ID
      operationId: getOrder
      parameters:
        - name: id
          in: path
          required: true
      responses:
        "200":
          description: Order with line items and shipments
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Order'
    patch:
      summary: Update order
      operationId: updateOrder
      parameters:
        - name: id
          in: path
          required: true
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/UpdateOrderRequest'
      responses:
        "200":
          description: Updated order
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Order'
  /orders/{id}/cancel:
    post:
      summary: Cancel an order
      operationId: cancelOrder
      parameters:
        - name: id
          in: path
          required: true
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/CancelOrderRequest'
      responses:
        "200":
          description: Order cancelled
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Order'
        "409":
          description: Order already shipped
  /orders/{id}/shipments:
    get:
      summary: List shipments for an order
      operationId: listOrderShipments
      parameters:
        - name: id
          in: path
          required: true
      responses:
        "200":
          description: List of shipments
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/OrderShipment'
components:
  schemas:
    Order:
      type: object
      required: [id, customer_id, status, total_cents]
      properties:
        id:
          type: string
        customer_id:
          type: string
        status:
          type: string
          description: "Status: pending, processing, partially_shipped, shipped, delivered, cancelled"
        line_items:
          type: array
          items:
            $ref: "#/components/schemas/OrderLineItem"
        subtotal_cents:
          type: integer
        tax_cents:
          type: integer
        shipping_cents:
          type: integer
        discount_cents:
          type: integer
        total_cents:
          type: integer
        currency:
          type: string
        shipping_address:
          type: object
        shipments:
          type: array
          items:
            $ref: "#/components/schemas/OrderShipment"
        created_at:
          type: string
          format: date-time
    OrderLineItem:
      type: object
      properties:
        sku_id:
          type: string
        product_name:
          type: string
        quantity:
          type: integer
        quantity_fulfilled:
          type: integer
          description: Quantity already shipped
        unit_price_cents:
          type: integer
        line_total_cents:
          type: integer
    OrderShipment:
      type: object
      properties:
        shipment_id:
          type: string
        carrier:
          type: string
        tracking_number:
          type: string
        status:
          type: string
        items:
          type: array
          items:
            type: object
            properties:
              sku_id:
                type: string
              quantity:
                type: integer
    UpdateOrderRequest:
      type: object
      properties:
        shipping_address:
          type: object
        status:
          type: string
    CancelOrderRequest:
      type: object
      properties:
        reason:
          type: string
          description: Cancellation reason
EOF

# OpenAPI: orders/returns/v1
mkdir -p "$DEMO_DIR/openapi/orders/returns/v1"
cat > "$DEMO_DIR/openapi/orders/returns/v1/returns.yaml" <<'EOF'
openapi: "3.0.3"
info:
  title: Returns API
  version: "1.2.0"
  description: Returns, refunds, and exchange processing.
paths:
  /returns:
    get:
      summary: List return requests
      operationId: listReturns
      parameters:
        - name: order_id
          in: query
        - name: status
          in: query
        - name: limit
          in: query
      responses:
        "200":
          description: List of returns
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/ReturnRequest'
    post:
      summary: Create a return request
      operationId: createReturn
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/CreateReturnRequest'
      responses:
        "201":
          description: Return request created
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ReturnRequest'
  /returns/{id}:
    get:
      summary: Get return details
      operationId: getReturn
      parameters:
        - name: id
          in: path
          required: true
      responses:
        "200":
          description: Return details
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ReturnRequest'
  /returns/{id}/approve:
    post:
      summary: Approve a return
      operationId: approveReturn
      parameters:
        - name: id
          in: path
          required: true
      responses:
        "200":
          description: Return approved, refund initiated
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ReturnRequest'
  /returns/{id}/reject:
    post:
      summary: Reject a return
      operationId: rejectReturn
      parameters:
        - name: id
          in: path
          required: true
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/RejectReturnRequest'
      responses:
        "200":
          description: Return rejected
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ReturnRequest'
components:
  schemas:
    ReturnRequest:
      type: object
      required: [id, order_id, status]
      properties:
        id:
          type: string
        order_id:
          type: string
        status:
          type: string
          description: "Status: pending, approved, rejected, refunded, exchanged"
        reason:
          type: string
          description: "Reason: defective, wrong_item, not_as_described, changed_mind"
        items:
          type: array
          items:
            $ref: "#/components/schemas/ReturnItem"
        refund_amount_cents:
          type: integer
        created_at:
          type: string
          format: date-time
    ReturnItem:
      type: object
      properties:
        sku_id:
          type: string
        quantity:
          type: integer
        condition:
          type: string
          description: "Item condition: unopened, used, damaged"
    CreateReturnRequest:
      type: object
      required: [order_id, reason, items]
      properties:
        order_id:
          type: string
        reason:
          type: string
          description: "Reason: defective, wrong_item, not_as_described, changed_mind"
        items:
          type: array
          items:
            $ref: "#/components/schemas/ReturnItem"
    RejectReturnRequest:
      type: object
      properties:
        reason:
          type: string
          description: Reason for rejection
EOF

# OpenAPI: shipping/tracking/v1
mkdir -p "$DEMO_DIR/openapi/shipping/tracking/v1"
cat > "$DEMO_DIR/openapi/shipping/tracking/v1/tracking.yaml" <<'EOF'
openapi: "3.0.3"
info:
  title: Shipment Tracking API
  version: "1.2.0"
  description: Track shipments, get delivery ETAs, and list carrier events.
paths:
  /shipments/{id}/track:
    get:
      summary: Get tracking info for a shipment
      operationId: trackShipment
      parameters:
        - name: id
          in: path
          required: true
      responses:
        "200":
          description: Tracking information with events
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Shipment'
        "404":
          description: Shipment not found
  /shipments:
    get:
      summary: List shipments
      operationId: listShipments
      parameters:
        - name: order_id
          in: query
        - name: status
          in: query
        - name: carrier
          in: query
      responses:
        "200":
          description: List of shipments
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/Shipment'
  /carriers:
    get:
      summary: List available carriers
      operationId: listCarriers
      responses:
        "200":
          description: List of supported carriers
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/Carrier'
components:
  schemas:
    Shipment:
      type: object
      required: [id, carrier, status]
      properties:
        id:
          type: string
        order_id:
          type: string
        carrier:
          type: string
          description: "Carrier code: ups, fedex, usps, dhl"
        tracking_number:
          type: string
        status:
          type: string
          description: "Status: pending, in_transit, out_for_delivery, delivered, exception"
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
    Carrier:
      type: object
      properties:
        code:
          type: string
          description: "Carrier code: ups, fedex, usps, dhl"
        name:
          type: string
          description: Carrier display name
        service_levels:
          type: array
          items:
            type: string
          description: Available service levels
EOF

# OpenAPI: shipping/rates/v1
mkdir -p "$DEMO_DIR/openapi/shipping/rates/v1"
cat > "$DEMO_DIR/openapi/shipping/rates/v1/rates.yaml" <<'EOF'
openapi: "3.0.3"
info:
  title: Shipping Rates API
  version: "1.0.0"
  description: Carrier rate quotes, cost comparison, and delivery time estimates.
paths:
  /rates/quote:
    post:
      summary: Get shipping rate quotes
      operationId: getRateQuotes
      description: Returns rates from all available carriers for the given package and destination.
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/RateQuoteRequest'
      responses:
        "200":
          description: List of rate quotes sorted by price
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/RateQuote'
        "400":
          description: Invalid address or dimensions
  /rates/carriers:
    get:
      summary: List carriers and service levels
      operationId: listCarrierServices
      responses:
        "200":
          description: Carriers with available service levels
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/CarrierService'
components:
  schemas:
    RateQuoteRequest:
      type: object
      required: [origin_zip, destination_zip, weight_grams]
      properties:
        origin_zip:
          type: string
        destination_zip:
          type: string
        weight_grams:
          type: integer
        length_cm:
          type: number
        width_cm:
          type: number
        height_cm:
          type: number
        service_level:
          type: string
          description: "Optional filter: standard, express, overnight"
    RateQuote:
      type: object
      properties:
        carrier:
          type: string
        service_level:
          type: string
        price_cents:
          type: integer
        currency:
          type: string
        estimated_days:
          type: integer
          description: Estimated business days to delivery
        guaranteed:
          type: boolean
          description: Whether delivery date is guaranteed
    CarrierService:
      type: object
      properties:
        carrier:
          type: string
          description: Carrier code
        name:
          type: string
          description: Carrier display name
        service_levels:
          type: array
          items:
            type: object
            properties:
              code:
                type: string
              name:
                type: string
              estimated_days:
                type: integer
EOF

# OpenAPI: shipping/labels/v1
mkdir -p "$DEMO_DIR/openapi/shipping/labels/v1"
cat > "$DEMO_DIR/openapi/shipping/labels/v1/labels.yaml" <<'EOF'
openapi: "3.0.3"
info:
  title: Shipping Labels API
  version: "1.1.0"
  description: Generate, void, and reprint shipping labels for carriers.
paths:
  /labels:
    post:
      summary: Create a shipping label
      operationId: createLabel
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/CreateLabelRequest'
      responses:
        "201":
          description: Label created with PDF URL
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Label'
    get:
      summary: List labels
      operationId: listLabels
      parameters:
        - name: shipment_id
          in: query
        - name: carrier
          in: query
      responses:
        "200":
          description: List of labels
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/Label'
  /labels/{id}:
    get:
      summary: Get label details
      operationId: getLabel
      parameters:
        - name: id
          in: path
          required: true
      responses:
        "200":
          description: Label details with download URL
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Label'
  /labels/{id}/void:
    post:
      summary: Void a label
      operationId: voidLabel
      parameters:
        - name: id
          in: path
          required: true
      responses:
        "200":
          description: Label voided, carrier notified
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Label'
  /labels/{id}/reprint:
    post:
      summary: Reprint a label
      operationId: reprintLabel
      parameters:
        - name: id
          in: path
          required: true
      responses:
        "200":
          description: New PDF URL generated
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Label'
components:
  schemas:
    Label:
      type: object
      required: [id, carrier, tracking_number]
      properties:
        id:
          type: string
        shipment_id:
          type: string
        carrier:
          type: string
        service_level:
          type: string
        tracking_number:
          type: string
        label_url:
          type: string
          format: uri
          description: PDF download URL
        status:
          type: string
          description: "Status: active, voided"
        cost_cents:
          type: integer
        created_at:
          type: string
          format: date-time
    CreateLabelRequest:
      type: object
      required: [shipment_id, carrier, service_level, from_address, to_address, weight_grams]
      properties:
        shipment_id:
          type: string
        carrier:
          type: string
        service_level:
          type: string
        from_address:
          type: object
        to_address:
          type: object
        weight_grams:
          type: integer
        dimensions:
          type: object
          properties:
            length_cm: { type: number }
            width_cm: { type: number }
            height_cm: { type: number }
EOF

# OpenAPI: notifications/email/v1
mkdir -p "$DEMO_DIR/openapi/notifications/email/v1"
cat > "$DEMO_DIR/openapi/notifications/email/v1/email.yaml" <<'EOF'
openapi: "3.0.3"
info:
  title: Email API
  version: "1.3.0"
  description: Transactional email service — templates, welcome emails, password resets, and order confirmations.
paths:
  /emails/send:
    post:
      summary: Send a transactional email
      operationId: sendEmail
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/SendEmailRequest'
      responses:
        "202":
          description: Email queued for delivery
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/EmailDeliveryStatus'
        "400":
          description: Invalid template or recipient
  /templates:
    get:
      summary: List email templates
      operationId: listTemplates
      responses:
        "200":
          description: List of templates
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/EmailTemplate'
    post:
      summary: Create a template
      operationId: createTemplate
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/CreateTemplateRequest'
      responses:
        "201":
          description: Template created
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/EmailTemplate'
  /templates/{id}:
    get:
      summary: Get template by ID
      operationId: getTemplate
      parameters:
        - name: id
          in: path
          required: true
      responses:
        "200":
          description: Template details
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/EmailTemplate'
    put:
      summary: Update a template
      operationId: updateTemplate
      parameters:
        - name: id
          in: path
          required: true
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/UpdateTemplateRequest'
      responses:
        "200":
          description: Template updated
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/EmailTemplate'
  /emails/{id}/status:
    get:
      summary: Get delivery status
      operationId: getEmailStatus
      parameters:
        - name: id
          in: path
          required: true
      responses:
        "200":
          description: Delivery status
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/EmailDeliveryStatus'
components:
  schemas:
    SendEmailRequest:
      type: object
      required: [template_id, to, variables]
      properties:
        template_id:
          type: string
          description: "Template key: welcome, password_reset, order_confirmation, shipping_update"
        to:
          type: string
          format: email
        variables:
          type: object
          description: Template variables (user_name, order_id, reset_link, etc.)
        priority:
          type: string
          description: "Priority: low, normal, high"
    EmailTemplate:
      type: object
      required: [id, name, subject]
      properties:
        id:
          type: string
        name:
          type: string
          description: Template name
        subject:
          type: string
          description: Email subject template
        body_html:
          type: string
          description: HTML body template
        body_text:
          type: string
          description: Plain text fallback
        variables:
          type: array
          items: { type: string }
          description: Required template variables
    EmailDeliveryStatus:
      type: object
      properties:
        id:
          type: string
        status:
          type: string
          description: "Delivery status: queued, sent, delivered, bounced, failed"
        sent_at:
          type: string
          format: date-time
        delivered_at:
          type: string
          format: date-time
    CreateTemplateRequest:
      type: object
      required: [name, subject, body_html]
      properties:
        name:
          type: string
          description: Template name
        subject:
          type: string
          description: Email subject template
        body_html:
          type: string
          description: HTML body template
        body_text:
          type: string
          description: Plain text fallback
        variables:
          type: array
          items: { type: string }
          description: Required template variables
    UpdateTemplateRequest:
      type: object
      properties:
        name:
          type: string
        subject:
          type: string
        body_html:
          type: string
        body_text:
          type: string
        variables:
          type: array
          items: { type: string }
EOF

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Avro schemas (inter-service event messaging)
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

# Avro: events/clicks/v1
mkdir -p "$DEMO_DIR/avro/events/clicks/v1"
cat > "$DEMO_DIR/avro/events/clicks/v1/clicks.avsc" <<'EOF'
{
  "type": "record",
  "name": "ClickEvent",
  "namespace": "com.megamart.events.clicks",
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
  "namespace": "com.megamart.events.transactions",
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

# Avro: events/order-placed/v1
mkdir -p "$DEMO_DIR/avro/events/order-placed/v1"
cat > "$DEMO_DIR/avro/events/order-placed/v1/order-placed.avsc" <<'EOF'
{
  "type": "record",
  "name": "OrderPlacedEvent",
  "namespace": "com.megamart.events.orders",
  "doc": "Emitted when a customer completes checkout and an order is created.",
  "fields": [
    {"name": "eventId", "type": "string", "doc": "Unique event identifier"},
    {"name": "orderId", "type": "string", "doc": "Order ID"},
    {"name": "customerId", "type": "string", "doc": "Customer who placed the order"},
    {"name": "totalCents", "type": "long", "doc": "Order total in cents"},
    {"name": "currency", "type": "string", "doc": "ISO 4217 currency code"},
    {"name": "itemCount", "type": "int", "doc": "Number of line items"},
    {"name": "shippingZip", "type": "string", "doc": "Destination zip code"},
    {"name": "promoCode", "type": ["null", "string"], "default": null, "doc": "Applied promo code"},
    {"name": "timestamp", "type": "long", "doc": "Event timestamp in milliseconds"}
  ]
}
EOF

# Avro: events/order-shipped/v1
mkdir -p "$DEMO_DIR/avro/events/order-shipped/v1"
cat > "$DEMO_DIR/avro/events/order-shipped/v1/order-shipped.avsc" <<'EOF'
{
  "type": "record",
  "name": "OrderShippedEvent",
  "namespace": "com.megamart.events.orders",
  "doc": "Emitted when a shipment leaves the warehouse for delivery.",
  "fields": [
    {"name": "eventId", "type": "string", "doc": "Unique event identifier"},
    {"name": "orderId", "type": "string", "doc": "Order ID"},
    {"name": "shipmentId", "type": "string", "doc": "Shipment ID"},
    {"name": "carrier", "type": "string", "doc": "Carrier code (ups, fedex, usps, dhl)"},
    {"name": "trackingNumber", "type": "string", "doc": "Carrier tracking number"},
    {"name": "warehouseId", "type": "string", "doc": "Origin warehouse"},
    {"name": "itemCount", "type": "int", "doc": "Number of items in this shipment"},
    {"name": "timestamp", "type": "long", "doc": "Event timestamp in milliseconds"}
  ]
}
EOF

# Avro: events/inventory-updated/v1
mkdir -p "$DEMO_DIR/avro/events/inventory-updated/v1"
cat > "$DEMO_DIR/avro/events/inventory-updated/v1/inventory-updated.avsc" <<'EOF'
{
  "type": "record",
  "name": "InventoryUpdatedEvent",
  "namespace": "com.megamart.events.inventory",
  "doc": "Emitted when stock levels change — receiving, sales, adjustments, or transfers.",
  "fields": [
    {"name": "eventId", "type": "string", "doc": "Unique event identifier"},
    {"name": "skuId", "type": "string", "doc": "SKU that changed"},
    {"name": "warehouseId", "type": "string", "doc": "Warehouse where change occurred"},
    {"name": "previousOnHand", "type": "int", "doc": "Previous on-hand quantity"},
    {"name": "newOnHand", "type": "int", "doc": "New on-hand quantity"},
    {"name": "reason", "type": {"type": "enum", "name": "ChangeReason", "symbols": ["SALE", "RECEIVING", "RETURN", "ADJUSTMENT", "TRANSFER_IN", "TRANSFER_OUT", "DAMAGE"]}, "doc": "Reason for the change"},
    {"name": "referenceId", "type": ["null", "string"], "default": null, "doc": "Related entity ID (order, receipt, transfer)"},
    {"name": "timestamp", "type": "long", "doc": "Event timestamp in milliseconds"}
  ]
}
EOF

# Avro: events/user-registered/v1
mkdir -p "$DEMO_DIR/avro/events/user-registered/v1"
cat > "$DEMO_DIR/avro/events/user-registered/v1/user-registered.avsc" <<'EOF'
{
  "type": "record",
  "name": "UserRegisteredEvent",
  "namespace": "com.megamart.events.users",
  "doc": "Emitted when a new user completes registration. Triggers welcome email and analytics.",
  "fields": [
    {"name": "eventId", "type": "string", "doc": "Unique event identifier"},
    {"name": "userId", "type": "string", "doc": "New user ID"},
    {"name": "email", "type": "string", "doc": "User email address"},
    {"name": "displayName", "type": "string", "doc": "Display name"},
    {"name": "registrationSource", "type": "string", "doc": "Source: web, mobile_app, api"},
    {"name": "referralCode", "type": ["null", "string"], "default": null, "doc": "Referral code if applicable"},
    {"name": "timestamp", "type": "long", "doc": "Event timestamp in milliseconds"}
  ]
}
EOF

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# JSON Schema configs
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

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

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Parquet schemas (reporting data lake)
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

# Parquet: reporting/orders/v1
mkdir -p "$DEMO_DIR/parquet/reporting/orders/v1"
cat > "$DEMO_DIR/parquet/reporting/orders/v1/orders.parquet" <<'EOF'
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

# Parquet: reporting/revenue/v1
mkdir -p "$DEMO_DIR/parquet/reporting/revenue/v1"
cat > "$DEMO_DIR/parquet/reporting/revenue/v1/revenue.parquet" <<'EOF'
message revenue_summary {
  required int64 report_date (DATE);
  required binary product_id (STRING);
  required binary category_id (STRING);
  required binary region (STRING);
  required binary channel (STRING);
  required int64 units_sold;
  required double gross_revenue;
  required double net_revenue;
  required double tax_collected;
  required double shipping_revenue;
  optional double refund_amount;
  required double cost_of_goods;
  required double gross_margin;
}
EOF

# Parquet: reporting/inventory-snapshot/v1
mkdir -p "$DEMO_DIR/parquet/reporting/inventory-snapshot/v1"
cat > "$DEMO_DIR/parquet/reporting/inventory-snapshot/v1/inventory-snapshot.parquet" <<'EOF'
message inventory_snapshot {
  required int64 snapshot_date (DATE);
  required binary sku_id (STRING);
  required binary warehouse_id (STRING);
  required binary warehouse_name (STRING);
  required int32 on_hand_quantity;
  required int32 reserved_quantity;
  required int32 available_quantity;
  required int32 incoming_quantity;
  optional double unit_cost;
  required double total_value;
  required binary reorder_status (STRING);
  optional int32 days_of_supply;
}
EOF

# Parquet: reporting/customer-activity/v1
mkdir -p "$DEMO_DIR/parquet/reporting/customer-activity/v1"
cat > "$DEMO_DIR/parquet/reporting/customer-activity/v1/customer-activity.parquet" <<'EOF'
message customer_activity {
  required binary customer_id (STRING);
  required int64 activity_date (DATE);
  required int32 page_views;
  required int32 product_views;
  required int32 add_to_cart_count;
  required int32 orders_placed;
  optional double total_spent;
  required binary acquisition_channel (STRING);
  optional int64 first_purchase_date (DATE);
  optional int64 last_purchase_date (DATE);
  required double lifetime_value;
  required int32 days_since_last_order;
  required binary segment (STRING);
}
EOF

echo "  ✓ Schema files created for all 45 APIs"

# ── Step 4: Generate and serve ──────────────────────────────────────────────
echo "▸ Launching catalog site on http://localhost:10451 ..."
echo "  Press Ctrl+C to stop"
echo ""

SERVE_FLAGS=(--catalog "$DEMO_DIR/catalog.yaml" --dir "$DEMO_DIR")
if [ "$NO_OPEN" = "--no-open" ]; then
  SERVE_FLAGS+=(--no-open)
fi

"$DEMO_DIR/apx" catalog site serve "${SERVE_FLAGS[@]}"
