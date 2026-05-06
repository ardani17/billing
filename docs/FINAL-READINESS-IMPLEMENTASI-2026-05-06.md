# Final Readiness Implementasi

Tanggal: 2026-05-06

## Scope

Dokumen ini adalah audit penutup untuk pekerjaan dari `.kiro/specs/project-audit-completion` dan `.kiro/specs/financial-completion`.

Tetap dikecualikan sesuai arahan:

- MikroTik
- OLT
- Map / FTTH mapping

## Status Akhir

Project sudah siap untuk diuji pengguna pada scope Billing Core, Keuangan, Settings aktif, Reporting, dan Notifikasi.

Yang sudah ditutup pada putaran terakhir:

- Route `/reports/reconciliation` ditambahkan untuk rekonsiliasi finance.
- Menu sidebar menampilkan `Pengeluaran` dan `Rekonsiliasi`.
- Halaman `/expenses` sekarang memakai app shell, filter periode, reload, dan konfirmasi hapus.
- Expense create/update/delete sekarang menulis `audit_logs` dengan action `expense.created`, `expense.updated`, dan `expense.deleted`.
- Report page sekarang memaksa refetch saat filter berubah.
- Build production web sudah berhasil.
- Backend billing-api dan notification service sudah lulus test.
- Smoke route production Next sudah lulus untuk route inti.

## Rekonsiliasi Keuangan

Route: `/reports/reconciliation`

Data yang disatukan:

- Revenue report
- Aging / piutang
- Payment report
- Voucher revenue dan reseller margin
- Profit-loss
- Expense periode
- Credit note impact dari invoice yang tersedia
- Debit note impact dari customer invoice yang tersedia

Fungsi halaman:

- Filter periode.
- Filter area untuk report revenue/payment/aging/profit-loss.
- Kartu tagihan terukur, pembayaran diterima, piutang akhir, dan net collection.
- Breakdown revenue, piutang, credit note, debit note, payment, expense, reseller margin, dan voucher revenue.
- Daftar anomali dasar seperti piutang terbuka, mismatch revenue, atau keterbatasan scope expense.
- Export memakai endpoint report export aktif.

Catatan penting:

- Expense saat ini masih tenant-wide karena tabel `expenses` belum memiliki `area_id`. Halaman rekonsiliasi menampilkan catatan ini saat area filter dipakai.
- Ini bukan bug runtime. Ini batasan skema yang perlu migration baru jika nanti ingin expense per area/cabang.

## Catatan `area_id`

Tempat user client bisa membuat dan memilih area saat ini:

- Master area dibuat dari `/customers/areas`, lalu disimpan ke endpoint `POST /api/v1/areas`.
- Saat membuat pelanggan di `/customers/new`, user memilih field `Area`; form mengirim `area_id` ke payload customer.
- Laporan utama `/reports` memakai `area_id` sebagai filter report.
- Rekonsiliasi `/reports/reconciliation` juga memakai pilihan area sebagai filter revenue, aging, payment, dan profit-loss.

Batasan saat ini:

- User belum bisa menginput `area_id` pada expense/pengeluaran.
- Tabel `expenses`, DTO expense, repository, dan form `/expenses` belum memiliki field `area_id`.
- Jika nanti expense harus per area/cabang, perlu spec lanjutan: tambah migration `expenses.area_id`, update DTO/API, update form expense, update `SumByCategory`, dan sesuaikan profit-loss/reconciliation agar expense ikut filter area.

## Expense dan Profit-Loss

Status:

- Expense CRUD sudah live via `/api/v1/expenses`.
- Expense UI memakai filter periode yang sama dengan profit-loss.
- Profit-loss mengambil expense dari `expenseRepo.SumByCategory`.
- Expense update/delete sudah masuk audit log.
- Create expense juga ditambahkan audit log agar jejak perubahan lengkap.

Endpoint terkait:

- `GET /api/v1/expenses`
- `POST /api/v1/expenses`
- `PUT /api/v1/expenses/:id`
- `DELETE /api/v1/expenses/:id`
- `GET /api/v1/reports/financial/profit-loss`

RBAC:

- Expense route dibatasi `tenant_admin`.
- Report financial read dapat diakses `tenant_admin`, `operator`, dan `kasir`.
- Report admin/export/settings dibatasi `tenant_admin`.

## Audit Settings

| Route | Status | Keterangan |
| --- | --- | --- |
| `/settings` | Live | Index settings membaca module capability. |
| `/settings/users` | Live | Terhubung ke `/api/v1/settings/users`. |
| `/settings/payment` | Live | Terhubung ke payment gateway settings. |
| `/settings/billing` | Live | Terhubung ke `GET/PUT /api/v1/settings/billing`. |
| `/settings/reports` | Live | KPI target, schedule, dan custom report template. |
| `/settings/notifications` | Live | Terhubung ke notification-service config dan template. |
| `/settings/security` | Live | Change password. |
| `/settings/branding` | Live lokal | Client branding tersedia pada frontend. |
| `/settings/audit-log` | Deferred | Belum ada endpoint list audit log global tenant. |
| `/settings/invoice` | Deferred | Digabung sementara ke `/settings/billing`. |
| `/settings/localization` | Deferred | Belum ada endpoint persistence terpisah. |
| `/settings/profile` | Deferred | Belum ada endpoint profile tenant terpisah. |
| `/settings/subscription` | Deferred | Belum ada endpoint subscription tenant pada billing core. |
| `/settings/voucher` | Deferred | Belum ada endpoint voucher settings terpisah. |
| `/settings/mikrotik` | Excluded | Ditunda sesuai scope. |
| `/settings/olt` | Excluded | Ditunda sesuai scope. |
| `/settings/map` | Excluded | Ditunda sesuai scope. |

Halaman deferred sengaja tidak menampilkan data palsu. UI generic menampilkan status bahwa endpoint persistence belum tersedia.

## Permission Matrix

| Area / Action | Super Admin | Tenant Admin | Operator | Kasir | Reseller |
| --- | --- | --- | --- | --- | --- |
| Admin platform `/api/v1/admin/*` | Ya | Tidak | Tidak | Tidak | Tidak |
| Customer read | Bypass | Ya | Ya | Ya | Tidak |
| Customer create/update | Bypass | Ya | Ya | Tidak | Tidak |
| Customer delete/bulk delete | Bypass | Ya | Tidak | Tidak | Tidak |
| Area CRUD | Bypass | Ya | Ya | Tidak | Tidak |
| Package read | Bypass | Ya | Ya | Ya | Tidak |
| Package admin actions | Bypass | Ya | Tidak | Tidak | Tidak |
| Invoice read | Bypass | Ya | Ya | Ya | Tidak |
| Invoice create/edit/payment-facing write | Bypass | Ya | Tidak | Ya | Tidak |
| Invoice cancel, reminder, bulk cancel, bulk PDF | Bypass | Ya | Tidak | Tidak | Tidak |
| Credit note / debit note | Bypass | Ya | Tidak | Tidak | Tidak |
| Payment read/write | Bypass | Ya | Tidak | Ya | Tidak |
| Payment void/import/admin actions | Bypass | Ya | Tidak | Tidak | Tidak |
| Expense CRUD | Bypass | Ya | Tidak | Tidak | Tidak |
| Report read | Bypass | Ya | Ya | Ya | Tidak |
| Report export, KPI, schedule, custom template | Bypass | Ya | Tidak | Tidak | Tidak |
| Billing settings | Bypass | Ya | Tidak | Tidak | Tidak |
| Payment gateway settings | Bypass | Ya | Tidak | Tidak | Tidak |
| Notification config/template | Bypass | Ya | Tidak | Tidak | Tidak |
| Reseller portal actions | Tidak | Tidak | Tidak | Tidak | Ya |

Catatan:

- `super_admin` memakai bypass RBAC dari middleware untuk route tenant tertentu, sedangkan route admin platform tetap dibatasi explicit super admin.
- Frontend sekarang menampilkan entry utama untuk halaman aktif, tetapi enforcement utama tetap berada di backend route.

## Verifikasi Notifikasi

Flow yang diverifikasi dari kode dan test:

- `InvoiceActionUsecase.BulkReminder` publish event `invoice.reminder`.
- Payload memakai `InvoiceReminderPayload` berisi invoice, tenant, customer, total, dan due date.
- Notification service mendaftarkan handler `invoice.reminder` pada `EventConsumer`.
- `EventConsumer` decode queue envelope lalu meneruskan ke `DeliveryPipeline.ProcessEvent`.
- `DeliveryPipeline` resolve template berdasarkan `event_type`.
- Default template untuk `invoice.reminder` tersedia di seed notification.
- Pipeline membuat `NotificationLog` dengan status `sent`, `failed`, `pending`, atau `skipped`.
- Endpoint UI notification tersedia untuk config, template, log, test send, manual send, dan resend.

Catatan produksi:

- Channel WhatsApp/SMS tetap bergantung credential provider tenant.
- Jika template atau credential belum aktif, pipeline akan skip atau mencatat failure sesuai jalur yang ada.

## UI Smoke Test

Smoke production Next dilakukan setelah `npm.cmd --workspace @ispboss/web run build`, menggunakan `next start` pada port sementara dan request HTTP ke route inti.

Semua route berikut mengembalikan HTTP 200:

- `/dashboard`
- `/customers`
- `/customers/new`
- `/packages`
- `/packages/new`
- `/notifications`
- `/reports`
- `/reports/reconciliation`
- `/expenses`
- `/settings`
- `/settings/billing`
- `/settings/reports`
- `/settings/notifications`
- `/settings/payment`
- `/settings/security`
- `/settings/users`

## Verifikasi Command

Berhasil:

- `go test ./...` di `services/billing-api`
- `go test ./...` di `services/notification`
- `npm.cmd --workspace @ispboss/web run build`
- Smoke route production Next pada 16 route inti

Playwright:

- `npm.cmd ls @playwright/test --depth=0` di worktree ini masih kosong, jadi smoke browser diganti dengan route smoke production Next.

## Sinkronisasi Diskusi

| Dokumen | Status akhir scope aktif |
| --- | --- |
| `00-arsitektur.md` | Terverifikasi build/test dan service layout. |
| `01-landing-page.md` | Route auth/landing ada, tidak menjadi fokus perubahan. |
| `02-auth.md` | RBAC diaudit dan matrix dibuat. |
| `03-dashboard-layout.md` | App shell dan route inti lulus smoke. |
| `04-pelanggan.md` | Customer route aktif dan masuk smoke. |
| `05-paket.md` | Package route aktif dan masuk smoke. |
| `06-billing.md` | Invoice, payment, credit/debit note, recurring item, bulk PDF, dan UI finance utama ditutup. |
| `07-notifikasi.md` | Flow reminder ke notification service diverifikasi dari producer, worker, template, dan log pipeline. |
| `08-mikrotik.md` | Excluded. |
| `09-olt.md` | Excluded. |
| `10-ftth-mapping.md` | Excluded. |
| `11-laporan.md` | Report settings dan rekonsiliasi finance ditambahkan. |
| `12-settings.md` | Settings aktif diaudit, live/deferred/excluded dicatat. |

## Kesimpulan

Checklist implementasi aktif sudah ditutup untuk scope saat ini. Sisa yang tercatat sebagai deferred bukan pekerjaan yang tertinggal pada scope ini, melainkan modul/endpoint yang memang belum menjadi prioritas atau sengaja dikecualikan.
