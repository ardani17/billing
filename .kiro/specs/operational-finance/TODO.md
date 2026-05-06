# TODO Keuangan Operasional

## Confirmed

- [x] Pengeluaran operasional sudah dibahas di diskusi 11.
- [x] Endpoint `/v1/expenses/*` sudah direncanakan di arsitektur.
- [x] UI `/expenses` sudah ada tetapi belum masuk sidebar.
- [x] Inventaris belum punya diskusi/spec lengkap.
- [x] Cashflow belum punya diskusi/spec lengkap.
- [x] Modul ini harus menjadi bagian Billing Core dan tidak bergantung pada MikroTik/OLT.

## Found Gaps

- [x] Sidebar belum punya grup/menu Keuangan.
- [x] `/expenses` belum mudah ditemukan dari navbar.
- [x] `/expenses` belum sepenuhnya konsisten dengan halaman live lain.
- [x] Inventaris belum punya backend.
- [x] Inventaris belum punya UI.
- [x] Cashflow belum punya backend aggregation.
- [x] Cashflow belum punya UI.
- [x] Laporan laba rugi belum memberi jalur jelas ke pengelolaan expense.
- [x] Dashboard belum punya widget cashflow operasional.
- [x] RBAC finance/inventory perlu dipastikan.

## Recommended Work Order

1. Tampilkan Pengeluaran di sidebar dan rapikan UI existing.
2. Lengkapi expense filter/field/audit yang kurang.
3. Bangun inventaris backend.
4. Bangun inventaris UI.
5. Bangun cashflow backend.
6. Bangun cashflow UI.
7. Integrasikan ke laporan dan dashboard.
8. Smoke Billing-only.

## Acceptance Smoke

- [x] Admin bisa membuka Pengeluaran dari sidebar.
- [x] Admin bisa tambah/edit/hapus pengeluaran.
- [x] Admin bisa membuat master item inventaris.
- [x] Admin bisa input stok masuk.
- [x] Admin bisa membuat expense terkait dari mutasi stok via API.
- [x] Admin bisa assign/return/damage ONT/router ke pelanggan secara manual via API.
- [x] Perubahan item/aset/mutasi inventaris masuk audit log tenant.
- [x] Admin bisa melihat cashflow periode berjalan.
- [x] Billing-only tenant tidak error saat MikroTik/Fiber inactive.

## Hardening 2026-05-06

- [x] Metadata expense `payment_method`, `vendor_name`, `reference_number`, `attachment_url` dipakai repository dan UI.
- [x] Expense create/update/delete menulis audit log tenant.
- [x] RBAC expense membuka CRUD untuk kasir, tetapi kategori expense tetap admin-only.
- [x] Operator/teknisi/kasir tidak bisa POST/PUT/DELETE inventory.
- [x] Biaya inventory disembunyikan dari role operasional tanpa akses finance penuh.
- [x] Item `track_serial=true` wajib memakai `asset_id` saat mutasi stok manual.
- [x] Mutasi `install`, `return`, `damaged`, `lost`, `rma`, `retired` dapat mengubah status/lokasi aset serial.
- [x] UI inventory menyediakan aksi lost, RMA, retired.
- [x] Cashflow mencakup manual income/out, voucher direct sale, inventory purchase tanpa expense, dan expense operasional.
- [x] Cashflow transaction filter mendukung direction, source, category, dan search.
