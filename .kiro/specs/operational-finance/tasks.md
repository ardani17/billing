# Implementation Plan: Keuangan Operasional

## Overview

Implementasi dilakukan bertahap agar fitur yang sudah ada tidak rusak. Tahap pertama hanya memunculkan dan merapikan pengeluaran. Tahap kedua membangun inventaris. Tahap ketiga membangun cashflow dan integrasinya ke laporan/dashboard.

## Tasks

- [x] 1. Navigasi dan halaman pengeluaran
  - [x] 1.1 Tambahkan grup sidebar "Keuangan" di `apps/web/app/components/app-shell.tsx`.
  - [x] 1.2 Tambahkan item Pengeluaran menuju `/expenses`.
  - [x] 1.3 Rapikan halaman `/expenses` agar konsisten dengan AppShell, PageHeader, Section, DataTable.
  - [x] 1.4 Tambahkan filter periode dan kategori di `/expenses`.
  - [x] 1.5 Verifikasi `/expenses` berjalan pada tenant Billing-only.

- [x] 2. Pengeluaran backend enhancement
  - [x] 2.1 Audit entity expense existing dan migration existing.
  - [x] 2.2 Tambahkan field opsional payment_method, vendor_name, reference_number, attachment_url bila belum ada.
  - [x] 2.3 Pastikan delete memakai soft delete dan audit log existing.
  - [x] 2.4 Tambahkan endpoint/filter untuk period_start, period_end, category_id.
  - [x] 2.5 Verifikasi handler/usecase expense tetap lolos dalam paket test billing-api.

- [x] 3. Database inventaris
  - [x] 3.1 Buat migration `inventory_items`.
  - [x] 3.2 Buat migration `inventory_assets`.
  - [x] 3.3 Buat migration `inventory_movements`.
  - [x] 3.4 Tambahkan index tenant/status/location.
  - [x] 3.5 Tambahkan RLS policy tenant isolation.

- [x] 4. Domain dan repository inventaris
  - [x] 4.1 Buat domain operational finance.
  - [x] 4.2 Buat DTO request/response inventory.
  - [x] 4.3 Buat repository inventory dengan query terstruktur pgx.
  - [x] 4.4 Buat repository wrapper inventory.
  - [x] 4.5 Tambahkan invariant usecase: stock tidak boleh negatif.

- [x] 5. Usecase inventaris
  - [x] 5.1 Implement CRUD inventory item.
  - [x] 5.2 Implement CRUD inventory asset.
  - [x] 5.3 Implement stock movement purchase/install/return/transfer/adjustment/damaged/lost.
  - [x] 5.4 Implement assign asset to customer.
  - [x] 5.5 Implement optional expense creation for purchase movement.
  - [x] 5.6 Tambahkan audit log untuk mutasi stok dan assignment aset.

- [x] 6. Handler dan routes inventaris
  - [x] 6.1 Tambahkan InventoryHandler.
  - [x] 6.2 Register `/api/v1/inventory/items`.
  - [x] 6.3 Register `/api/v1/inventory/assets`.
  - [x] 6.4 Register `/api/v1/inventory/movements`.
  - [x] 6.5 Register `/api/v1/inventory/stock`.
  - [x] 6.6 Tambahkan RBAC finance/inventory permission.

- [x] 7. Frontend inventaris
  - [x] 7.1 Buat route `/inventory`.
  - [x] 7.2 Buat tab Ringkasan, Barang, Aset Serial, Mutasi Stok dengan stok menipis di Ringkasan.
  - [x] 7.3 Buat form master barang.
  - [x] 7.4 Buat form stok masuk/purchase.
  - [x] 7.5 Buat aksi assign/return/damaged untuk aset serial.
  - [x] 7.6 Tambahkan item Inventaris ke sidebar Keuangan.
  - [x] 7.7 Pastikan UI responsive tanpa horizontal overflow lewat build dan smoke halaman.

- [x] 8. Backend cashflow
  - [x] 8.1 Buat Cashflow domain DTO.
  - [x] 8.2 Buat aggregation repository untuk cash-in/cash-out.
  - [x] 8.3 Implement `GET /v1/cashflow/summary`.
  - [x] 8.4 Implement `GET /v1/cashflow/transactions`.
  - [x] 8.5 Implement `GET /v1/cashflow/trend`.
  - [x] 8.6 Implement export CSV.
  - [x] 8.7 Tambahkan invariant usecase: opening + cash_in - cash_out = closing estimate.

- [x] 9. Frontend cashflow
  - [x] 9.1 Buat route `/cashflow`.
  - [x] 9.2 Buat summary cards.
  - [x] 9.3 Buat trend chart.
  - [x] 9.4 Buat breakdown kategori.
  - [x] 9.5 Buat transaction table dan filter periode.
  - [x] 9.6 Tambahkan item Arus Kas ke sidebar Keuangan.

- [x] 10. Integrasi laporan/dashboard
  - [x] 10.1 Tambahkan link dari ProfitLoss expense section ke `/expenses`.
  - [x] 10.2 Tambahkan entry/link Cashflow di financial reports.
  - [x] 10.3 Tambahkan widget cashflow ringkas di dashboard jika data tersedia.
  - [x] 10.4 Pastikan cashflow tidak memanggil network-service.

- [x] 11. Verification
  - [x] 11.1 Run `go test ./internal/domain ./internal/repository ./internal/usecase ./internal/handler` di billing-api.
  - [x] 11.2 Run `npm.cmd --prefix apps/web run build`.
  - [x] 11.3 Smoke `/expenses`, `/inventory`, `/cashflow`.
  - [x] 11.4 Smoke tenant Billing-only dengan MikroTik/Fiber inactive.

- [x] 12. Hardening spec gap audit
  - [x] 12.1 Wire metadata expense sampai repository dan form/table UI.
  - [x] 12.2 Tambahkan audit log expense create/update/delete.
  - [x] 12.3 Sesuaikan RBAC expense untuk kasir dan kunci kategori tetap admin-only.
  - [x] 12.4 Batasi inventory write untuk operator, teknisi, dan kasir.
  - [x] 12.5 Redact cost inventory untuk role tanpa finance permission penuh.
  - [x] 12.6 Paksa mutasi item serial memakai `asset_id`.
  - [x] 12.7 Tambahkan action lost/RMA/retired dan sinkronkan movement ke status/lokasi asset.
  - [x] 12.8 Tambahkan manual cashflow + filter category/search/source/direction.
  - [x] 12.9 Masukkan voucher direct sale dan inventory purchase tanpa expense ke cashflow.
  - [x] 12.10 Verifikasi `go test ./...` billing-api dan `npm.cmd --workspace @ispboss/web run build`.
