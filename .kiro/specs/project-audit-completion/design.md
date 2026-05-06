# Design Document

## Overview

Project audit completion adalah hardening layer untuk area non-keuangan yang sudah diaudit. Tujuannya bukan membangun ulang fitur yang sudah ada, tetapi memastikan route, settings, permission, smoke test, dan dokumentasi status benar-benar siap dipakai.

Spec ini tidak mencakup MikroTik, OLT, Map, dan detail finance lanjutan.

## Scope

In scope:

- Frontend dependency/build verification.
- Settings page readiness.
- Permission matrix untuk action sensitif.
- Browser smoke test workflow utama.
- Notification integration verification.
- Sinkronisasi status dokumen `diskusi`.

Out of scope:

- MikroTik.
- OLT.
- Map / mapping.
- Financial completion detail, karena ditangani oleh spec `financial-completion`.

## Implementation Areas

### Frontend build readiness

Verifikasi harus dimulai dari dependency web workspace. Jika `next` tidak tersedia, tentukan apakah root install belum dijalankan atau workspace dependency belum benar.

Dokumentasikan command setup yang valid, lalu jalankan build.

### Settings readiness

Audit seluruh route di `apps/web/app/settings`:

- `users`
- `payment`
- `notifications`
- `security`
- `branding`
- `billing`
- `invoice`
- `localization`
- `voucher`
- `subscription`
- `audit-log`

Halaman yang masih generic harus diputuskan:

- Diubah menjadi live page.
- Ditandai sebagai pending/deferred.
- Dipindahkan ke spec lanjutan yang sesuai.

### Permission matrix

Buat matriks role/action untuk:

- User management.
- Settings update.
- Customer destructive action.
- Package destructive action.
- Notification settings.
- Report export/settings.
- Finance destructive action, direferensikan ke spec finance.

Backend harus menggunakan guard existing bila sudah tersedia. Frontend harus menyembunyikan atau men-disable action sesuai permission.

### Smoke test

Smoke test dilakukan setelah build/frontend dependency sehat.

Minimal workflow:

- Login.
- Dashboard render.
- Customer list/detail/create/edit.
- Package list/create/edit.
- Notification page.
- Report page.
- Settings page utama.

Jika browser automation belum tersedia, hasil manual smoke test dicatat di dokumen audit.

### Notification integration

Verifikasi integrasi dari billing reminder ke notification service:

- Event/job dibuat.
- Template dipilih.
- Channel aktif/nonaktif dihormati.
- Delivery status dan failure reason tercatat.

## Validation

Backend validation:

- `go test ./...` untuk service yang berubah.

Frontend validation:

- Install dependency jika dibutuhkan.
- `npm.cmd --workspace @ispboss/web run build`.
- Smoke test browser atau manual route check.

Documentation validation:

- Update dokumen audit.
- Update status `diskusi` jika tim memutuskan dokumen tersebut menjadi sumber status aktif.
