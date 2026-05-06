# Production Readiness Spec

Tanggal: 2026-05-06

## Scope

Dokumen ini menjadi daftar kerja sampai ISPBoss siap production untuk scope aplikasi inti:

- Billing core
- Pelanggan, area, paket internet, invoice, pembayaran
- Voucher dan reseller
- Notifikasi
- Reporting, rekonsiliasi, cashflow, expense, inventory
- Settings tenant
- Super Admin / owner aplikasi
- Landing page publik
- Deployment, security, backup, observability

Dikecualikan dari implementasi production pada dokumen ini:

- MikroTik
- OLT
- Fitur map yang bergantung pada perangkat OLT/fiber lapangan

## Definition of Done Production

Aplikasi baru bisa disebut siap production jika semua poin berikut terpenuhi:

- Tidak ada bypass authentication development pada environment production.
- Secret production wajib eksplisit dan aplikasi gagal start jika memakai default development.
- Semua endpoint penting punya RBAC backend dan diuji per role.
- Build web, test backend, dan smoke route berjalan otomatis di CI.
- Migration database punya prosedur up, backup sebelum migrasi, dan rollback plan.
- Payment gateway, notification provider, dan webhook diuji end-to-end pada staging.
- Backup dan restore Postgres pernah diuji.
- Error penting bisa terlihat dari log, healthcheck, dan alert.
- Flow harian tenant admin bisa berjalan tanpa mock untuk scope billing-only.

## 1. Security Production Gate

### Masalah

Frontend proxy masih memiliki fallback development token agar UI lokal mudah diuji. Ini aman untuk development, tetapi tidak boleh aktif di production.

### Tasks

- Nonaktifkan dev JWT otomatis saat `NODE_ENV=production` atau `APP_ENV=production`.
- Tambahkan flag eksplisit `ISPBOSS_ENABLE_DEV_AUTH=true` hanya untuk development.
- Jika tidak ada session token pada production, proxy harus mengembalikan `401 AUTH_REQUIRED`.
- Super Admin API proxy harus memakai token session asli di production.
- Dokumentasikan env auth development dan production.

### Acceptance Criteria

- Development lokal tetap bisa dipakai tanpa login penuh jika flag development aktif/default.
- Production tidak membuat token palsu.
- Build web berhasil.

## 2. Secret dan Environment Hardening

### Masalah

Docker compose development masih menyediakan fallback default untuk DB password, JWT secret, dan encryption key. Untuk production, default seperti ini berbahaya.

### Tasks

- Buat template `.env.production.example` tanpa secret nyata.
- Tandai nilai yang wajib diganti sebelum deploy.
- Tambahkan validasi service agar `APP_ENV=production` menolak secret development umum.
- Dokumentasikan checklist env production.

### Acceptance Criteria

- Developer punya template env production yang jelas.
- Aplikasi production gagal start jika memakai `change-me-to-a-strong-secret` atau secret kosong.

## 3. CI/CD Release Gate

### Masalah

Belum ada pipeline GitHub Actions di repo root.

### Tasks

- Tambahkan workflow CI.
- Jalankan `npm ci`.
- Jalankan build web.
- Jalankan `go test ./...` untuk `billing-api`, `notification`, dan service lain yang tidak bergantung perangkat nyata.
- Simpan artefak/log jika gagal.

### Acceptance Criteria

- PR atau push ke `main` menjalankan test dan build otomatis.
- Failure menghentikan release.

## 4. Migration, Backup, dan Restore

### Masalah

Migration sudah ada, tetapi prosedur production belum menjadi command/runbook yang aman.

### Tasks

- Buat runbook migration production.
- Buat script backup Postgres manual.
- Buat script restore rehearsal untuk staging/local.
- Tambahkan urutan deploy: backup, migrate, healthcheck, smoke test.

### Acceptance Criteria

- Operator bisa melakukan backup sebelum migrasi.
- Restore bisa diuji pada database kosong/staging.
- Ada rollback plan per release.

## 5. RBAC dan Module Entitlement

### Masalah

Role dan module gating sudah ada sebagian, tetapi perlu matrix test agar billing-only tidak tergantung MikroTik/OLT.

### Tasks

- Buat test matrix role: super admin, tenant admin, operator, kasir, reseller.
- Pastikan tenant billing-only tetap bisa membuat pelanggan manual tanpa module network.
- Pastikan menu/module MikroTik dan OLT tidak membuat error jika entitlement tidak aktif.
- Pastikan data finance/cost tidak muncul untuk role tanpa permission finance.

### Acceptance Criteria

- Semua role utama punya smoke path.
- Billing-only berjalan end-to-end tanpa perangkat jaringan.

## 6. Billing dan Payment Edge Cases

### Masalah

Flow invoice dan pembayaran sudah berjalan, tetapi produksi perlu tahan kasus partial, double payment, multi-bulan, void/refund, dan webhook duplicate.

### Tasks

- Tambahkan regression test partial payment.
- Tambahkan regression test pembayaran beberapa invoice/bulan.
- Tambahkan handling overpayment/credit carry-forward.
- Pastikan invoice detail bisa dibuka dan receipt jelas.
- Uji webhook duplicate dan expired payment.

### Acceptance Criteria

- Pembayaran sebagian tidak menutup invoice sebelum lunas.
- Pembayaran dobel tidak menggandakan saldo secara salah.
- Pembayaran beberapa bulan tercatat sebagai satu transaksi operasional yang mudah diaudit.

## 7. Notification Production

### Masalah

Notification service sudah ada, tetapi provider nyata perlu hardening.

### Tasks

- Finalisasi template invoice reminder, paid, overdue, broadcast.
- Uji provider WhatsApp/email nyata pada staging.
- Tambahkan retry monitoring dan resend manual.
- Pastikan log delivery bisa difilter tenant, channel, status.

### Acceptance Criteria

- Tenant bisa test send manual.
- Failure provider tercatat dan bisa diulang.

## 8. Reseller dan Voucher Operations

### Masalah

Portal reseller sudah berkembang, tetapi perlu pengujian produksi harian.

### Tasks

- Uji login reseller, saldo, beli voucher, cetak voucher, transaksi, deposit.
- Pastikan stok voucher dikelola dari tenant admin.
- Pastikan audit saldo reseller ada saat top-up/debit.
- Tambahkan filter dan pagination untuk data besar.

### Acceptance Criteria

- Reseller bisa bekerja harian tanpa akses admin.
- Tenant admin bisa rekonsiliasi saldo dan voucher.

## 9. Finance, Inventory, dan Cashflow

### Masalah

Expense, inventory, dan cashflow sudah tersedia, tetapi production perlu audit trail, approval, dan attachment yang rapi.

### Tasks

- Pastikan expense create/update/delete masuk audit log.
- Pastikan metadata expense dipakai di query dan UI.
- Tambahkan approval policy untuk pengeluaran besar.
- Pastikan asset serial wajib untuk item `track_serial=true`.
- Pastikan cashflow mencakup income manual, payment, expense, voucher, dan koreksi.

### Acceptance Criteria

- Cashflow dapat dipakai sebagai ringkasan operasional.
- Inventory serial tidak bisa masuk/keluar tanpa jejak asset.

## 10. Super Admin Production

### Masalah

Console owner sudah ada, tetapi perlu fitur SaaS production.

### Tasks

- Tenant lifecycle: create, suspend, reactivate, impersonate.
- Module entitlement: billing-only, billing+MikroTik, billing+OLT/map.
- Subscription SaaS: paket, masa aktif, invoice owner, reminder tenant.
- Health monitoring platform.
- Audit global.

### Acceptance Criteria

- Owner bisa mengelola tenant tanpa akses database langsung.
- Tenant tanpa add-on tidak melihat/menjalankan fitur add-on.

## 11. Observability dan Support

### Masalah

Healthcheck ada, tetapi belum cukup untuk operasi production.

### Tasks

- Standarkan structured log.
- Tambahkan error tracking atau minimal log aggregation target.
- Buat endpoint/status dashboard untuk dependency utama.
- Buat runbook incident dasar.

### Acceptance Criteria

- Error 5xx bisa dilacak ke service dan request.
- Admin tahu jika payment/notification/database bermasalah.

## 12. Codebase Cleanup

### Masalah

Beberapa file terlalu besar dan ada artefak binary di workspace.

### Tasks

- Pecah file frontend besar per modul.
- Pecah handler/repository backend besar secara bertahap.
- Cek binary besar apakah tracked Git.
- Hapus binary tracked atau tambahkan ke `.gitignore`.

### Acceptance Criteria

- Repo tidak menyimpan build binary yang tidak perlu.
- Modul lebih mudah dirawat tanpa mengubah behavior.

## Todo Eksekusi

| No | Area | Status |
| --- | --- | --- |
| 1 | Security production gate | Done - dev JWT terkunci saat production, proxy mengembalikan 401 tanpa session |
| 2 | Secret dan environment hardening | In progress - env template production dan validasi secret service sudah ditambahkan |
| 3 | CI/CD release gate | Done - GitHub Actions build web dan Go test ditambahkan |
| 4 | Migration, backup, restore | In progress - runbook dan script backup/restore awal sudah ditambahkan |
| 5 | RBAC dan entitlement | In progress - smoke matrix role sudah dibuat di `docs/RBAC-SMOKE-MATRIX.md` |
| 6 | Billing/payment regression | In progress - test backend existing untuk partial/FIFO/overpayment/void terverifikasi lewat `go test ./...` |
| 7 | Notification production | Pending |
| 8 | Reseller/voucher operations | Pending |
| 9 | Finance/inventory/cashflow | Pending |
| 10 | Super Admin production | Pending |
| 11 | Observability/support | In progress - runbook monitoring minimum sudah dibuat |
| 12 | Codebase cleanup | In progress - `.gitattributes` dan ignore backup sudah ditambahkan; binary besar dipastikan tidak tracked |
