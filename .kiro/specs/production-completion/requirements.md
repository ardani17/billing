# Requirements: Production Completion

## Scope

Spec ini merangkum hasil audit aplikasi ISPBoss setelah modul reporting selesai dan MikroTik live integration mulai berjalan. Tujuannya adalah mengubah aplikasi dari local pilot menjadi sistem yang siap dipakai tenant ISP secara operasional.

## Review Findings

### R1 - Core Tenant Admin

- Pelanggan, area, paket, invoice, pembayaran, gateway, reseller, voucher, dashboard, dan reporting sudah punya backend dan UI live API.
- Data lokal masih kecil/demo sehingga perlu smoke test end-to-end dengan data operasional yang lebih realistis.
- Beberapa endpoint list sensitif terhadap query param yang tidak sesuai; frontend utama sudah memakai pola yang benar.

### R2 - MikroTik

- Mode network-service sudah `live`.
- Health checker dan PPPoE sync scheduler default nonaktif, sehingga tidak login API berkala saat idle.
- Router CHR real sudah tested untuk SSL API, PPPoE, Hotspot, DHCP, Static IP, Walled Garden, Terminal read-only, dan audit.
- Sisa gap MikroTik adalah backup/restore, firmware check, bulk action, tenant settings lookup untuk metode isolir, dan hardening test.

### R3 - OLT

- Backend dan UI OLT sudah ada.
- Data lokal OLT masih kosong.
- Adapter beberapa brand masih berupa fondasi/placeholder sehingga perlu real-device validation sebelum production.

### R4 - Notification

- Service notification hidup dengan config, template, log, manual send, resend, dan provider Fonnte/SMTP/Zenziva.
- Belum ada bukti credential provider produksi dan real send/retry/reminder/broadcast diuji end-to-end.

### R5 - Settings

- Settings yang sudah live: users, payment gateway, notification, security password.
- Settings yang masih generic/belum persist penuh: billing, invoice, profile ISP, MikroTik defaults, voucher, map, OLT, localization, audit log tenant, subscription.

### R6 - Super Admin

- Super Admin UI/API live untuk overview, tenants, subscriptions, health, audit.
- Support ticket belum punya backend tabel khusus.
- Subscription SaaS, tenant lifecycle, dan owner workflow masih perlu hardening.

### R7 - Frontend

- Halaman operasional utama banyak sudah API-backed.
- Legacy `mock-data.ts` dan `module-pages.tsx` masih ada untuk halaman lama/bantuan dan harus dibersihkan bertahap.
- Mobile responsiveness sudah banyak diperbaiki, tapi belum ada visual QA menyeluruh.

### R8 - Production Readiness

- Semua service utama lulus `go test ./...` dan health check lokal.
- Perlu env production checklist, secret rotation, backup/restore, RBAC audit, error handling form, dan dokumentasi deploy.

## Acceptance Criteria

1. Semua modul Tenant Admin utama tidak bergantung pada mock data untuk workflow operasional.
2. MikroTik dapat dipakai real dengan operasi on-demand, audit lengkap, backup/restore aman, dan bulk job terkontrol.
3. Settings penting tenant tersimpan di backend dan dipakai oleh flow terkait.
4. Notification dapat mengirim real provider dengan template, retry, log, dan broadcast/reminder.
5. Super Admin punya data live untuk tenant, subscription, health, audit, dan support minimal.
6. Aplikasi punya smoke test end-to-end dan dokumentasi deploy/operasional.
