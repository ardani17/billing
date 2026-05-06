# Design Document - Keuangan Operasional

## Overview

Keuangan Operasional terdiri dari tiga area:

- Pengeluaran: melanjutkan Expense API/UI yang sudah ada.
- Inventaris: modul baru untuk master item, aset serial, dan mutasi stok.
- Cashflow: laporan uang masuk/keluar yang berbeda dari laba rugi.

Modul ini berada di `services/billing-api` karena semua sumber utama berada di domain billing: invoice, payment, reseller transaction, expense, dan inventory purchase. Frontend berada di `apps/web`.

## Current State

Yang sudah ada:

- Route frontend `/expenses`.
- Komponen `ExpenseForm`, `ExpenseTable`, `CategoryManager`.
- Domain/backend expense dan expense category.
- Expense digunakan oleh reporting/profit-loss.

Gap:

- `/expenses` belum masuk sidebar.
- Belum ada `/inventory`.
- Belum ada `/cashflow`.
- Belum ada domain inventory.
- Belum ada cashflow aggregation API.

## Navigation Design

Sidebar menambah grup baru:

```text
Keuangan
  Pengeluaran -> /expenses
  Inventaris  -> /inventory
  Arus Kas    -> /cashflow
```

Menu ini tidak memakai module gate MikroTik/Fiber. Keuangan adalah bagian Billing Core.

## Backend Architecture

```mermaid
graph TD
  UI[apps/web] --> ExpenseAPI[/v1/expenses]
  UI --> InventoryAPI[/v1/inventory]
  UI --> CashflowAPI[/v1/cashflow]

  ExpenseAPI --> ExpenseUsecase
  InventoryAPI --> InventoryUsecase
  CashflowAPI --> CashflowUsecase

  ExpenseUsecase --> ExpenseRepo
  InventoryUsecase --> InventoryRepo
  CashflowUsecase --> PaymentRepo
  CashflowUsecase --> ExpenseRepo
  CashflowUsecase --> ResellerTxRepo
  CashflowUsecase --> InventoryRepo

  ExpenseRepo --> DB[(PostgreSQL)]
  InventoryRepo --> DB
  PaymentRepo --> DB
  ResellerTxRepo --> DB
```

## Domain Model

### Expense

Existing entity remains the source of cash-out and profit/loss cost lines.

Recommended additional fields:

- payment_method;
- vendor_name;
- reference_number;
- attachment_url;
- inventory_movement_id.

### InventoryItem

```go
type InventoryItem struct {
    ID          string
    TenantID    string
    Name        string
    Category    string
    Unit        string
    TrackSerial bool
    MinStock    int
    DefaultCost int64
    IsActive    bool
}
```

### InventoryAsset

```go
type InventoryAsset struct {
    ID                 string
    TenantID           string
    ItemID             string
    SerialNumber       string
    MacAddress         string
    Status             AssetStatus
    LocationType       LocationType
    LocationID         string
    AssignedCustomerID string
    PurchaseCost       int64
    PurchaseDate       *time.Time
    WarrantyUntil      *time.Time
}
```

### InventoryMovement

```go
type InventoryMovement struct {
    ID               string
    TenantID         string
    ItemID           string
    AssetID          string
    MovementType     MovementType
    Quantity         int
    FromLocationType string
    FromLocationID   string
    ToLocationType   string
    ToLocationID     string
    CustomerID       string
    ExpenseID        string
    Notes            string
    CreatedByID      string
}
```

### Cashflow

Cashflow can be calculated from source tables instead of storing every transaction in a dedicated table.

Source mapping:

| Source | Direction | Notes |
|---|---|---|
| payments | cash_in | customer invoice payments |
| reseller_transactions deposit | cash_in | deposit is cash-in but not automatically revenue |
| reseller_transactions withdraw | cash_out | reseller withdrawal |
| expenses | cash_out | operational expense |
| inventory purchase movement | cash_out | if not already linked to expense |
| refunds/credit notes | cash_out | when implemented |
| manual income | cash_in | optional future table |

## API Design

### Expenses

Existing `/v1/expenses/*` remains.

### Inventory

```text
GET    /v1/inventory/items
POST   /v1/inventory/items
GET    /v1/inventory/items/:id
PUT    /v1/inventory/items/:id
DELETE /v1/inventory/items/:id

GET    /v1/inventory/assets
POST   /v1/inventory/assets
GET    /v1/inventory/assets/:id
PUT    /v1/inventory/assets/:id
POST   /v1/inventory/assets/:id/assign
POST   /v1/inventory/assets/:id/return
POST   /v1/inventory/assets/:id/mark-damaged

GET    /v1/inventory/movements
POST   /v1/inventory/movements
GET    /v1/inventory/stock
```

### Cashflow

```text
GET /v1/cashflow/summary
GET /v1/cashflow/transactions
GET /v1/cashflow/trend
GET /v1/cashflow/export
```

## Frontend Pages

### `/expenses`

Existing page should be integrated into AppShell visual style and sidebar.

Expected controls:

- period filter;
- category filter;
- add/edit modal or inline form;
- category manager;
- recurring marker;
- total expense summary.

### `/inventory`

Tabs:

- Ringkasan;
- Barang;
- Aset Serial;
- Mutasi Stok;
- Stok Menipis.

Expected controls:

- add item;
- stock-in purchase;
- assign asset to customer;
- return asset;
- mark damaged/lost;
- filter by category, location, status.

### `/cashflow`

Sections:

- summary cards: opening balance, cash-in, cash-out, net, closing estimate;
- trend chart;
- category breakdown;
- transaction table;
- CSV export.

## RBAC

Finance amounts are sensitive. Owner/admin have full access. Kasir can manage expense and view cashflow. Operator/teknisi can be limited to inventory movement tasks.

## Multi-Tenant and Add-on Compatibility

All queries must filter tenant_id. No endpoint may require MikroTik or Fiber Network. Optional ODP/ONT location selectors appear only when fiber_network is active.

## Test Strategy

- Backend unit tests for cashflow aggregation invariants.
- Repository tests for inventory stock cannot go negative.
- Handler tests for RBAC and tenant isolation.
- Frontend build test.
- Billing-only smoke: mikrotik=false, fiber_network=false, `/expenses`, `/inventory`, `/cashflow` load without network calls.

