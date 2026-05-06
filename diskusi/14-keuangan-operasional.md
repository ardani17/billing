# 14 - Keuangan Operasional

## Tujuan

Modul Keuangan Operasional melengkapi Billing Core agar tenant ISP bisa mencatat biaya operasional, mengelola inventaris/perangkat, dan membaca arus kas harian tanpa harus memakai software akuntansi penuh.

Dokumen ini menutup gap yang ditemukan pada diskusi sebelumnya:

- Pengeluaran operasional sudah dibahas di dokumen laporan, tetapi menu dan alur operasionalnya belum jelas.
- Inventaris/aset belum dibahas sebagai modul sendiri.
- Cashflow/arus kas belum dibahas sebagai laporan berbeda dari laba rugi.

## Prinsip Produk

ISPBoss tetap bukan software akuntansi lengkap. Fitur ini berfokus pada kebutuhan praktis ISP kecil-menengah:

- tahu uang masuk dan uang keluar;
- tahu stok ONT/router/kabel/splitter;
- tahu perangkat dipasang ke pelanggan, ODP, gudang, atau teknisi;
- tahu pembelian perangkat ikut tercatat sebagai pengeluaran;
- tahu kas operasional cukup atau tidak untuk periode berjalan.

## Ruang Lingkup

### 1. Pengeluaran Operasional

Pengeluaran adalah pencatatan uang keluar tenant.

Contoh:

- bandwidth/upstream;
- gaji admin, teknisi, kasir;
- sewa tiang atau infrastruktur;
- listrik/NOC/kantor;
- beli ONT/router/kabel/splitter;
- biaya notifikasi WA/SMS/email;
- biaya transport teknisi;
- biaya lain-lain.

Status saat ini:

- Backend dan UI dasar `/expenses` sudah ada.
- Kategori pengeluaran sudah ada.
- Belum ada menu sidebar untuk membuka halaman pengeluaran.
- Perlu dirapikan agar menjadi bagian dari navigasi Keuangan/Billing.

### 2. Inventaris dan Aset

Inventaris mencatat barang fisik yang dimiliki atau dikelola ISP.

Jenis barang:

- ONT/ONU;
- router;
- OLT spare;
- switch;
- kabel fiber;
- splitter;
- ODP/ODC material;
- konektor, patchcord, adaptor;
- tools teknisi.

Inventaris dibedakan menjadi:

- **Stok habis pakai**: kabel, konektor, material kecil.
- **Perangkat bernomor seri**: ONT, router, switch, OLT module.
- **Aset operasional**: laptop admin, tools teknisi, perangkat NOC.

Lokasi barang:

- gudang;
- teknisi;
- pelanggan;
- POP/NOC;
- ODP/ODC;
- rusak/RMA;
- hilang.

Alur utama:

1. Admin membuat master item.
2. Admin input stok masuk dari pembelian.
3. Jika pembelian memakai kas, sistem bisa membuat pengeluaran otomatis.
4. Teknisi/admin mengeluarkan stok untuk instalasi pelanggan atau perbaikan.
5. Perangkat serial bisa ditautkan ke pelanggan, ONU/ONT, router, atau titik jaringan.
6. Perangkat bisa dikembalikan, dipindah lokasi, rusak, atau dihapus dari stok aktif.

### 3. Cashflow / Arus Kas

Cashflow adalah ringkasan uang masuk dan keluar berdasarkan tanggal transaksi kas.

Cashflow berbeda dari laba rugi:

| Topik | Cashflow | Laba Rugi |
|---|---|---|
| Fokus | kas masuk/keluar | pendapatan dan biaya |
| Dasar tanggal | tanggal pembayaran/transaksi | periode pendapatan/biaya |
| Invoice belum dibayar | tidak masuk kas | bisa masuk piutang |
| Deposit reseller | kas masuk, tetapi bukan revenue final | bukan pendapatan sampai voucher terjual/dipakai sesuai kebijakan |
| Pembelian stok | kas keluar | bisa menjadi biaya langsung atau aset/stok |

Sumber kas masuk:

- pembayaran invoice pelanggan;
- pembayaran biaya pasang;
- pembayaran denda;
- penjualan voucher langsung;
- deposit reseller;
- pemasukan manual lain.

Sumber kas keluar:

- pengeluaran operasional;
- pembelian inventaris/perangkat;
- refund;
- withdraw reseller;
- pengeluaran manual lain.

Laporan cashflow minimal:

- saldo awal periode;
- total kas masuk;
- total kas keluar;
- net cashflow;
- saldo akhir estimasi;
- breakdown per kategori;
- tren harian/mingguan/bulanan;
- daftar transaksi kas terbaru.

## Navigasi UI

Sidebar perlu menampilkan pengelolaan keuangan operasional.

Rekomendasi struktur:

```
Billing
  Invoice
  Pembayaran
  Voucher

Keuangan
  Pengeluaran
  Inventaris
  Arus Kas
```

Alternatif jika sidebar ingin tetap pendek:

```
Billing
  Invoice
  Pembayaran
  Voucher
  Pengeluaran
  Inventaris
  Arus Kas
```

Rekomendasi saya: buat grup baru **Keuangan**, karena invoice/pembayaran/voucher adalah transaksi billing, sedangkan pengeluaran/inventaris/cashflow adalah operasional bisnis.

Mobile bottom nav tidak perlu menambah semua menu. Menu Keuangan bisa diakses dari sidebar mobile atau halaman More/Pengaturan.

## Role dan Hak Akses

| Role | Pengeluaran | Inventaris | Cashflow |
|---|---|---|---|
| Owner | full access | full access | full access |
| Admin | full access | full access | lihat/export |
| Kasir | create/update pengeluaran | lihat stok | lihat cashflow |
| Operator | lihat terbatas | mutasi stok terbatas | tidak wajib |
| Teknisi | tidak wajib | ambil/kembali perangkat | tidak wajib |

Semua create/update/delete harus masuk audit log tenant.

Catatan hardening implementasi:

- kategori pengeluaran tetap admin-only karena mengubah struktur pembukuan;
- operator, teknisi, dan kasir tidak boleh membuat/mengubah/menghapus master inventory lewat endpoint umum;
- biaya default barang, biaya beli aset, dan unit cost mutasi tidak boleh diekspos ke role operasional tanpa akses finance penuh;
- kasir boleh mencatat pengeluaran, tetapi pencatatan kas manual tetap dibatasi ke admin/owner.

## Backend API

Pengeluaran yang sudah direncanakan:

```text
GET    /v1/expenses
POST   /v1/expenses
GET    /v1/expenses/:id
PUT    /v1/expenses/:id
DELETE /v1/expenses/:id

GET    /v1/expenses/categories
POST   /v1/expenses/categories
PUT    /v1/expenses/categories/:id
DELETE /v1/expenses/categories/:id
```

Inventaris yang perlu dibuat:

```text
GET    /v1/inventory/items
POST   /v1/inventory/items
GET    /v1/inventory/items/:id
PUT    /v1/inventory/items/:id
DELETE /v1/inventory/items/:id

GET    /v1/inventory/stock
POST   /v1/inventory/movements
GET    /v1/inventory/movements

GET    /v1/inventory/assets
POST   /v1/inventory/assets
PUT    /v1/inventory/assets/:id
POST   /v1/inventory/assets/:id/assign
POST   /v1/inventory/assets/:id/return
POST   /v1/inventory/assets/:id/mark-damaged
POST   /v1/inventory/assets/:id/mark-lost
POST   /v1/inventory/assets/:id/mark-rma
POST   /v1/inventory/assets/:id/retire
```

Cashflow yang perlu dibuat:

```text
GET /v1/cashflow/summary
GET /v1/cashflow/transactions?direction=&source=&category=&search=
GET /v1/cashflow/trend
GET /v1/cashflow/categories
GET /v1/cashflow/export
POST /v1/cashflow/manual
```

## Data Model Ringkas

### expenses

Sudah ada di implementasi reporting. Perlu ditinjau ulang untuk memastikan field berikut cukup:

- tenant_id;
- category_id;
- amount;
- description;
- expense_date;
- is_recurring;
- recurring_day;
- created_by_id;
- deleted_at.

Tambahan yang disarankan:

- payment_method;
- vendor_name;
- reference_number;
- attachment_url;
- inventory_movement_id nullable.

Field metadata tersebut harus dibaca/ditulis oleh repository, tampil di form/table UI, dan tetap ikut audit log create/update/delete.

### inventory_items

Master barang:

- id;
- tenant_id;
- name;
- category;
- unit;
- track_serial;
- min_stock;
- default_cost;
- is_active.

### inventory_assets

Unit perangkat bernomor seri:

- id;
- tenant_id;
- item_id;
- serial_number;
- mac_address;
- status;
- location_type;
- location_id;
- assigned_customer_id;
- purchase_cost;
- purchase_date;
- warranty_until.

### inventory_movements

Mutasi stok:

- id;
- tenant_id;
- item_id;
- asset_id nullable;
- movement_type: purchase, install, return, transfer, adjustment, damaged, lost;
- quantity;
- from_location_type;
- from_location_id;
- to_location_type;
- to_location_id;
- customer_id nullable;
- expense_id nullable;
- notes;
- created_by_id.

### cashflow_transactions

Tidak wajib sebagai tabel fisik jika bisa diagregasi dari payment, expense, reseller transaction, dan refund. Namun view/materialized view bisa dipertimbangkan untuk performa.

## Integrasi Modul

| Modul | Integrasi |
|---|---|
| Pelanggan | perangkat bisa ditautkan ke pelanggan |
| Invoice/Pembayaran | pembayaran menjadi cash-in |
| Voucher/Reseller | deposit reseller cash-in, withdraw cash-out |
| Pengeluaran | menjadi cash-out |
| Inventaris | pembelian stok dapat membuat expense otomatis |
| Laporan | laba rugi memakai expense; cashflow memakai transaksi kas |
| OLT/Peta Jaringan | aset ONT/ODP bisa ditautkan jika add-on fiber aktif |

## Graceful Degradation

- Billing-only tetap bisa memakai pengeluaran, inventaris, dan cashflow.
- Jika MikroTik tidak aktif, inventaris tetap bisa menautkan perangkat manual ke pelanggan.
- Jika fiber_network tidak aktif, lokasi ODP/ONT disembunyikan, tetapi gudang/teknisi/pelanggan tetap aktif.
- Jika reporting belum lengkap, halaman cashflow tetap menampilkan summary dari payment dan expense.

## Keputusan

| Keputusan | Detail |
|---|---|
| Nama modul | Keuangan Operasional |
| Menu utama | Tambah grup sidebar Keuangan |
| Halaman minimal | Pengeluaran, Inventaris, Arus Kas |
| Pengeluaran | Lanjutkan UI/API yang sudah ada dan munculkan di navbar |
| Inventaris | Modul baru untuk item, stok, aset serial, mutasi |
| Cashflow | Modul/laporan baru, beda dari laba rugi |
| Akuntansi penuh | Tidak termasuk ledger, jurnal umum, neraca, pajak |
| Billing-only | Wajib berjalan tanpa MikroTik/OLT |

## Status Implementasi 2026-05-06

Sudah dikerjakan:

- Sidebar grup Keuangan dengan menu Pengeluaran, Inventaris, dan Arus Kas.
- `/expenses` dirapikan agar masuk AppShell dan memiliki filter periode/kategori.
- Backend inventaris: item, aset serial, mutasi stok, stock summary, assign, return, mark-damaged.
- Database inventaris: `inventory_items`, `inventory_assets`, `inventory_movements`, index tenant, dan RLS.
- Audit log tenant untuk item inventaris, aset serial, assignment, return/damaged, dan mutasi stok.
- UI `/inventory`: ringkasan stok rendah, master barang, aset serial, assign/return/damaged, mutasi stok.
- Mutasi stok purchase dapat membuat expense terkait jika kategori expense dipilih.
- Backend cashflow: summary, transactions, trend, export CSV.
- UI `/cashflow`: summary card, trend, breakdown, transaction table, export CSV.
- Dashboard menampilkan widget cashflow operasional.
- Laporan laba rugi memberi shortcut ke Pengeluaran dan Arus Kas.

Verifikasi:

- `go test ./internal/domain ./internal/repository ./internal/usecase ./internal/handler`
- `npm.cmd --prefix apps/web run build`
- Smoke lokal `/expenses`, `/inventory`, `/cashflow`, `/dashboard`, `/reports`
