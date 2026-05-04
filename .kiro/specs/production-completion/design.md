# Design: Production Completion

## Principles

- Semua integrasi perangkat harus manual/on-demand kecuali fitur scheduler secara eksplisit diaktifkan.
- Jangan tampilkan data palsu pada modul operasional; lebih baik empty state jujur.
- Semua write path penting harus punya audit log atau event log.
- Semua modul baru dibuat kecil dan terpisah: domain, repository, usecase, handler, web proxy, UI panel.
- Default development harus aman untuk CHR test: scheduler mati, command dangerous ditolak, dan koneksi ditutup setelah aksi.

## Execution Order

### Phase 1 - MikroTik Backup/Firmware

Tambahkan manual export backup `.rsc`, metadata backup, list/download/delete backup, firmware/system package read, dan UI submenu Backup/Firmware. Restore disiapkan dengan guard konfirmasi, tetapi eksekusi restore harus dibatasi command aman terlebih dahulu.

### Phase 2 - MikroTik Bulk Actions

Tambahkan job table dan status model untuk bulk sync, backup, firmware check, dan export status. Semua bulk action harus asynchronous dan menampilkan progress.

### Phase 3 - Tenant Settings Persistence

Lengkapi backend settings untuk billing, invoice, profile ISP, MikroTik defaults, voucher, map, OLT, localization, audit log tenant, dan subscription.

### Phase 4 - Notification Production

Finalisasi provider credential, template default, manual send, broadcast, reminder invoice, retry, dan monitoring log.

### Phase 5 - Tenant Admin End-to-End

Smoke test customer -> package -> invoice -> payment -> payment link -> PPPoE/Hotspot -> isolir/unisolir -> report.

### Phase 6 - OLT Real Validation

Hubungkan OLT real saat perangkat tersedia, validasi SNMP/CLI, provisioning ONT, alarm, capacity, dan map relation.

### Phase 7 - Super Admin Hardening

Lengkapi support ticket, subscription lifecycle, tenant suspend/reactivate, impersonation safeguards, dan audit global.

### Phase 8 - Production Hardening

RBAC audit, security hardening, backup/restore database, env checklist, error boundary, responsive QA, dan dokumentasi deploy.

## Verification

- Setiap phase minimal menjalankan test service terkait.
- Frontend wajib `npm.cmd --workspace @ispboss/web run build`.
- Integrasi MikroTik wajib smoke test ke CHR untuk command aman.
- Setelah build Next, dev server harus dibersihkan `.next` dan direstart untuk menghindari stale chunk runtime.
