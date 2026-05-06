# Requirements Document

## Introduction

Dokumen ini mendefinisikan requirements untuk modul **Keuangan Operasional** ISPBoss. Modul ini mencakup pengeluaran operasional, inventaris/aset, dan cashflow/arus kas. Fokusnya adalah kebutuhan praktis tenant ISP untuk mengelola biaya, stok perangkat, dan posisi kas tanpa menjadikan ISPBoss sebagai software akuntansi penuh.

Modul ini harus berjalan dalam paket Billing Core tanpa ketergantungan pada add-on MikroTik atau Fiber Network.

## Glossary

- **Expense**: Catatan pengeluaran operasional tenant.
- **Expense_Category**: Kategori pengeluaran, misalnya bandwidth, gaji, listrik, perangkat.
- **Inventory_Item**: Master barang, misalnya ONT, router, kabel, splitter.
- **Inventory_Asset**: Unit perangkat bernomor seri, misalnya ONT dengan serial number atau MAC address.
- **Inventory_Movement**: Catatan mutasi stok atau perpindahan aset.
- **Cashflow**: Laporan uang masuk dan uang keluar berdasarkan tanggal transaksi kas.
- **Cash_In**: Uang masuk, seperti pembayaran invoice atau deposit reseller.
- **Cash_Out**: Uang keluar, seperti expense, refund, atau withdraw reseller.
- **Billing_Core**: Paket utama ISPBoss tanpa add-on MikroTik dan Fiber Network.

## Requirements

### Requirement 1: Sidebar dan Navigasi Keuangan

**User Story:** Sebagai admin tenant, saya ingin menu Pengeluaran, Inventaris, dan Arus Kas terlihat di sidebar, sehingga saya bisa mengelola keuangan operasional tanpa mencari route tersembunyi.

#### Acceptance Criteria

1. THE Web_App SHALL add a visible sidebar group named "Keuangan".
2. THE Keuangan group SHALL include links to `/expenses`, `/inventory`, and `/cashflow`.
3. THE existing `/expenses` page SHALL be reachable from the sidebar.
4. THE mobile layout SHALL keep the bottom nav compact and expose Keuangan through the mobile sidebar or More flow.
5. THE navigation SHALL remain available when MikroTik and Fiber Network modules are inactive.

### Requirement 2: Pengeluaran Operasional

**User Story:** Sebagai owner/admin ISP, saya ingin mencatat pengeluaran operasional, sehingga laporan laba rugi dan cashflow bisa membaca biaya bisnis.

#### Acceptance Criteria

1. THE Expense_API SHALL support create, list, update, and delete expense records by tenant.
2. THE Expense_API SHALL support configurable expense categories per tenant.
3. THE Expense_Page SHALL support adding, editing, deleting, filtering, and viewing total expenses.
4. THE Expense_Page SHALL support recurring expense configuration for monthly costs.
5. THE Expense entity SHOULD support optional payment_method, vendor_name, reference_number, and attachment_url fields.
6. WHEN an expense is deleted, THE system SHALL soft-delete it and preserve audit trace.
7. THE profit/loss report SHALL include expenses as cost lines.

### Requirement 3: Inventaris Master Barang

**User Story:** Sebagai admin ISP, saya ingin membuat master barang seperti ONT, router, kabel, dan splitter, sehingga stok perangkat bisa dikelola rapi.

#### Acceptance Criteria

1. THE Inventory_API SHALL expose CRUD endpoints for inventory items.
2. THE Inventory_Item SHALL include name, category, unit, track_serial, min_stock, default_cost, and is_active.
3. THE Inventory_Page SHALL show item list with stock summary, minimum stock warning, and item status.
4. WHEN track_serial is true, THE system SHALL require per-unit asset records for stock-in operations.
5. WHEN track_serial is false, THE system SHALL manage quantity-based stock.

### Requirement 4: Inventaris Aset Bernomor Seri

**User Story:** Sebagai admin/teknisi ISP, saya ingin melacak ONT/router per serial number, sehingga saya tahu perangkat terpasang di pelanggan mana atau berada di lokasi mana.

#### Acceptance Criteria

1. THE Inventory_API SHALL expose endpoints for inventory assets.
2. THE Inventory_Asset SHALL include item_id, serial_number, mac_address, status, location_type, location_id, assigned_customer_id, purchase_cost, purchase_date, and warranty_until.
3. THE system SHALL prevent duplicate serial_number per tenant for active assets.
4. THE system SHALL allow assigning an asset to a customer.
5. THE system SHALL allow returning an asset from customer to warehouse or technician.
6. THE system SHALL allow marking an asset as damaged, lost, RMA, or retired.
7. IF Fiber Network module is inactive, THE UI SHALL hide ODP/ONT location options but still allow warehouse, technician, and customer locations.

### Requirement 5: Mutasi Stok dan Perangkat

**User Story:** Sebagai admin/teknisi ISP, saya ingin mencatat mutasi stok masuk, keluar, pindah, rusak, atau hilang, sehingga stok fisik dapat direkonsiliasi.

#### Acceptance Criteria

1. THE Inventory_API SHALL expose endpoints to create and list inventory movements.
2. THE Inventory_Movement SHALL support movement types: purchase, install, return, transfer, adjustment, damaged, lost.
3. WHEN a purchase movement is created with cost, THE system SHOULD offer creating an expense automatically.
4. WHEN an install movement is linked to a customer, THE stock SHALL decrease or asset location SHALL move to customer.
5. WHEN a return movement is created, THE stock SHALL increase or asset location SHALL move back to warehouse/technician.
6. THE system SHALL reject movements that make quantity stock negative.
7. Every movement SHALL record created_by_id and timestamp.

### Requirement 6: Cashflow Summary

**User Story:** Sebagai owner ISP, saya ingin melihat arus kas masuk dan keluar, sehingga saya bisa mengetahui kesehatan kas operasional.

#### Acceptance Criteria

1. THE Cashflow_API SHALL expose GET `/v1/cashflow/summary` with period_start and period_end.
2. THE Cashflow_API SHALL return opening_balance, total_cash_in, total_cash_out, net_cashflow, and closing_balance_estimate.
3. THE Cashflow_API SHALL include cash-in from customer payments, installation payments, voucher sales/direct payments, reseller deposits, and manual income.
4. THE Cashflow_API SHALL include cash-out from expenses, inventory purchases, refunds, reseller withdraws, and manual cash-out.
5. THE Cashflow_Page SHALL show cashflow summary cards and category breakdown.
6. THE Cashflow_Page SHALL clearly distinguish cashflow from profit/loss.

### Requirement 7: Cashflow Transactions and Trend

**User Story:** Sebagai owner/kasir, saya ingin melihat daftar transaksi kas dan tren harian, sehingga saya bisa memeriksa sumber kenaikan atau penurunan kas.

#### Acceptance Criteria

1. THE Cashflow_API SHALL expose GET `/v1/cashflow/transactions`.
2. THE Cashflow_API SHALL expose GET `/v1/cashflow/trend`.
3. THE transactions endpoint SHALL support filters: period, type, source, category, search.
4. THE trend endpoint SHALL return daily or monthly cash-in, cash-out, and net values.
5. THE Cashflow_Page SHALL show a responsive trend chart and transaction table.
6. THE Cashflow_Page SHALL support export to CSV at minimum.

### Requirement 8: Reporting Integration

**User Story:** Sebagai owner ISP, saya ingin pengeluaran, inventaris, dan cashflow terhubung ke laporan, sehingga laporan keuangan lebih lengkap.

#### Acceptance Criteria

1. THE Report_Page SHALL link to Expense_Page from profit/loss expense section.
2. THE Profit_Loss report SHALL continue using expenses as cost input.
3. THE Financial report SHALL add a Cashflow section or link to Cashflow_Page.
4. THE Dashboard SHOULD show an operational cash widget when data is available.
5. THE Custom Report Builder MAY include expense and cashflow datasets.

### Requirement 9: RBAC dan Audit

**User Story:** Sebagai owner, saya ingin akses keuangan operasional dibatasi dan diaudit, sehingga perubahan kas dan stok dapat dipertanggungjawabkan.

#### Acceptance Criteria

1. THE system SHALL restrict full finance access to owner/admin roles.
2. THE kasir role SHALL be allowed to manage expenses and view cashflow.
3. THE teknisi/operator role SHALL only access inventory movements assigned to operational workflow.
4. THE system SHALL write audit logs for expense create/update/delete, inventory item changes, asset assignment, and stock movement.
5. THE system SHALL not expose cost/cashflow values to roles without finance permission.

### Requirement 10: Billing-Only Compatibility

**User Story:** Sebagai tenant yang hanya membeli paket Billing Core, saya ingin pengeluaran, inventaris, dan cashflow tetap berjalan, sehingga aplikasi tidak bergantung pada add-on jaringan.

#### Acceptance Criteria

1. THE module SHALL work when mikrotik=false and fiber_network=false.
2. THE inventory UI SHALL support manual customer assignment without MikroTik or OLT data.
3. THE cashflow report SHALL not call network-service.
4. THE sidebar SHALL not hide Keuangan based on add-on module entitlement.
5. THE build and smoke tests SHALL pass for Billing-only tenant.

