# Requirements Document

## Introduction

Spec ini menutup gap modul keuangan berdasarkan audit implementasi pada 2026-05-06. Fokusnya adalah membuat fitur finance yang sudah ada di backend menjadi siap pakai secara operasional melalui UI, settings persistence, workflow rekonsiliasi, dan verifikasi build/test.

Spec ini tidak mencakup MikroTik, OLT, dan Map.

## Requirements

### Requirement 1: Billing settings live persistence

**User Story:** Sebagai tenant admin, saya ingin mengatur konfigurasi billing dari UI agar pajak, denda, jatuh tempo, prefix invoice, dan reminder bisa dikelola tanpa perubahan database manual.

#### Acceptance Criteria

1. WHEN admin membuka `/settings/billing` THEN sistem SHALL menampilkan data billing settings dari API.
2. WHEN admin menyimpan perubahan billing settings THEN sistem SHALL memvalidasi dan menyimpan perubahan ke database.
3. WHEN tax diaktifkan THEN sistem SHALL menyimpan tax rate dan menggunakan nilai tersebut pada invoice baru yang menerapkan pajak.
4. WHEN penalty diaktifkan THEN sistem SHALL menyimpan rule penalty, grace period, dan maksimum denda.
5. WHEN request dilakukan oleh user tanpa permission settings finance THEN sistem SHALL menolak dengan status authorization yang sesuai.

### Requirement 2: Invoice financial operations UI

**User Story:** Sebagai admin finance, saya ingin mengelola operasi invoice lanjutan dari UI agar pekerjaan harian tidak bergantung pada API manual.

#### Acceptance Criteria

1. WHEN admin membuka halaman invoice THEN sistem SHALL menyediakan action untuk create, edit, cancel, PDF, reminder, dan export sesuai permission.
2. WHEN admin membuat invoice prepaid THEN sistem SHALL menyediakan flow khusus prepaid invoice.
3. WHEN admin memilih banyak invoice THEN sistem SHALL menyediakan bulk action untuk reminder, cancel, export, dan PDF.
4. WHEN bulk PDF dijalankan THEN sistem SHALL menghasilkan file valid untuk seluruh invoice terpilih.
5. WHEN invoice memakai tax THEN sistem SHALL menampilkan subtotal, tax, discount, penalty, dan total secara jelas.

### Requirement 3: Credit note and debit note workflow

**User Story:** Sebagai admin finance, saya ingin membuat dan melacak credit note serta debit note agar koreksi tagihan tercatat rapi.

#### Acceptance Criteria

1. WHEN admin membuka detail invoice THEN sistem SHALL menampilkan riwayat credit note dan debit note terkait.
2. WHEN admin membuat credit note THEN sistem SHALL mengurangi saldo/tagihan sesuai aturan backend.
3. WHEN admin membuat debit note THEN sistem SHALL menambah saldo/tagihan sesuai aturan backend.
4. WHEN credit/debit note dibuat THEN sistem SHALL mencatat audit log berisi user, alasan, nominal, dan target invoice/customer.
5. WHEN data credit/debit note dibutuhkan untuk laporan THEN sistem SHALL dapat diambil melalui API list/detail.

### Requirement 4: Customer recurring item management

**User Story:** Sebagai admin finance, saya ingin mengatur recurring item per customer agar biaya tambahan bulanan bisa otomatis masuk invoice.

#### Acceptance Criteria

1. WHEN admin membuka detail customer THEN sistem SHALL menampilkan recurring item customer.
2. WHEN admin menambah recurring item THEN sistem SHALL menyimpan nama, nominal, periode, tanggal mulai, dan status aktif.
3. WHEN invoice bulanan dibuat THEN sistem SHALL memasukkan recurring item aktif sesuai periode.
4. WHEN recurring item dinonaktifkan THEN sistem SHALL tidak memasukkannya ke invoice baru.
5. WHEN recurring item berubah THEN sistem SHALL mencatat audit log.

### Requirement 5: Payment operations completion

**User Story:** Sebagai admin finance, saya ingin menjalankan quick payment, multi-invoice payment, receipt, proof, void, dan import dari UI agar pencatatan pembayaran lengkap.

#### Acceptance Criteria

1. WHEN admin membuka payment page THEN sistem SHALL menyediakan quick payment untuk customer dan invoice terbuka.
2. WHEN customer punya beberapa invoice unpaid THEN sistem SHALL mendukung multi-invoice payment dan pay-all.
3. WHEN payment dibuat THEN sistem SHALL menyediakan receipt yang bisa dicetak atau diunduh.
4. WHEN proof dibutuhkan THEN sistem SHALL mendukung upload dan view proof.
5. WHEN payment perlu dibatalkan THEN sistem SHALL menyediakan void flow dengan alasan dan audit log.
6. WHEN admin mengimpor payment THEN sistem SHALL menampilkan hasil sukses/gagal per baris.

### Requirement 6: Financial reconciliation dashboard

**User Story:** Sebagai finance owner, saya ingin melihat rekonsiliasi invoice, payment, voucher, expense, dan koreksi agar kondisi kas dan piutang dapat dipantau.

#### Acceptance Criteria

1. WHEN user membuka dashboard rekonsiliasi THEN sistem SHALL menampilkan invoice issued, payment collected, outstanding, expense, voucher usage, credit note, debit note, dan net collection.
2. WHEN periode diubah THEN sistem SHALL menghitung ulang seluruh metrik untuk periode tersebut.
3. WHEN ada selisih atau anomali THEN sistem SHALL menampilkan daftar item yang perlu ditinjau.
4. WHEN user memilih area/cabang THEN sistem SHALL memfilter metrik berdasarkan area/cabang.
5. WHEN data diekspor THEN sistem SHALL menghasilkan export CSV/XLSX sesuai filter aktif.

### Requirement 7: Report settings and admin controls

**User Story:** Sebagai tenant admin, saya ingin mengatur KPI, jadwal laporan, dan custom report agar laporan keuangan bisa disesuaikan dengan kebutuhan perusahaan.

#### Acceptance Criteria

1. WHEN admin membuka `/settings/reports` THEN sistem SHALL menampilkan KPI target, report schedule, dan custom report template.
2. WHEN admin menyimpan KPI target THEN sistem SHALL memakai target tersebut pada laporan KPI.
3. WHEN admin membuat jadwal report THEN sistem SHALL membuat schedule aktif untuk report terkait.
4. WHEN admin membuat custom report template THEN sistem SHALL menyimpan definisi field, filter, dan format export.
5. WHEN schedule report berjalan THEN sistem SHALL mencatat job result sukses/gagal.

### Requirement 8: Expense and profit-loss integration

**User Story:** Sebagai finance owner, saya ingin expense terhubung ke laporan laba rugi agar kondisi profit bisa dibaca dari sistem.

#### Acceptance Criteria

1. WHEN expense dibuat THEN sistem SHALL mengelompokkan expense berdasarkan kategori.
2. WHEN expense recurring aktif THEN sistem SHALL membuat expense sesuai jadwal.
3. WHEN laporan profit-loss dibuka THEN sistem SHALL memasukkan revenue, discount, tax, voucher impact, expense, dan net profit.
4. WHEN user memfilter periode THEN sistem SHALL memakai periode yang sama untuk revenue dan expense.
5. WHEN expense diubah atau dihapus THEN sistem SHALL mencatat audit log.

### Requirement 9: Verification and release readiness

**User Story:** Sebagai maintainer, saya ingin seluruh perubahan finance dapat dites dan dibuild agar aman sebelum production.

#### Acceptance Criteria

1. WHEN backend berubah THEN `go test ./...` untuk service terkait SHALL berhasil.
2. WHEN frontend berubah THEN build web workspace SHALL berhasil.
3. WHEN dependency frontend belum tersedia THEN dokumentasi setup SHALL menjelaskan command install yang diperlukan.
4. WHEN fitur finance selesai THEN smoke test SHALL mencakup settings billing, invoice, payment, expense, report, dan rekonsiliasi.
5. WHEN spec selesai THEN dokumen audit SHALL diperbarui atau ditautkan ke hasil implementasi.
