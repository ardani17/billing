# Production Runbook

Tanggal: 2026-05-06

## Scope

Runbook ini untuk operasi production billing core ISPBoss. MikroTik dan OLT tetap dikelola pada runbook add-on masing-masing.

## Pre-Deploy Checklist

- Pastikan `APP_ENV=production`.
- Pastikan `ISPBOSS_ENABLE_DEV_AUTH=false`.
- Pastikan `JWT_SECRET` bukan default development dan minimal 32 karakter.
- Pastikan `DB_PASSWORD` bukan default development.
- Pastikan `DB_SSL_MODE=require`, `verify-ca`, atau `verify-full`.
- Pastikan payment gateway key dan webhook IP provider sudah diisi jika gateway diaktifkan.
- Pastikan backup database terbaru berhasil dibuat dan bisa dibaca.
- Pastikan CI build/test hijau.

Preflight env:

```powershell
.\scripts\check-production-env.ps1 -EnvPath .\docker\.env.production
```

## Backup Database

Jalankan dari root repo pada host yang bisa mengakses container Postgres:

```powershell
.\scripts\backup-postgres.ps1 -ContainerName ispboss-postgres -Database ispboss -User ispboss
```

Output default disimpan ke folder `backups/`.

## Restore Rehearsal

Restore rehearsal harus dilakukan ke database staging atau database restore, bukan database production aktif.

```powershell
.\scripts\restore-postgres.ps1 -BackupPath .\backups\ispboss-ispboss-YYYYMMDD-HHMMSS.dump -Database ispboss_restore
```

Setelah restore:

- Cek jumlah tenant.
- Cek jumlah customer.
- Cek invoice dan payment sample.
- Jalankan smoke API terhadap database restore jika memungkinkan.

## Migration Sequence

1. Freeze perubahan operasional besar pada tenant.
2. Jalankan backup database.
3. Simpan checksum/nama file backup pada catatan release.
4. Jalankan migration.
5. Restart service.
6. Cek `/healthz` setiap service.
7. Jalankan smoke route web.
8. Cek log error 5xx.
9. Buka kembali operasional.

## Rollback Plan

Jika migration gagal sebelum aplikasi dipakai:

1. Stop service aplikasi.
2. Restore backup terakhir ke database production sesuai prosedur infrastruktur.
3. Deploy image/app versi sebelumnya.
4. Jalankan smoke test.
5. Catat insiden dan blokir release yang gagal.

Jika migration berhasil tetapi data sudah berubah setelah release, jangan restore langsung tanpa analisa. Buat patch migration atau koreksi data terkontrol.

## Smoke Test Minimum

- Login tenant admin.
- Buka dashboard.
- Buka pelanggan.
- Buat pelanggan manual billing-only.
- Buat paket bulanan.
- Generate invoice.
- Catat pembayaran manual.
- Buka report revenue dan reconciliation.
- Buka reseller admin dan portal reseller jika fitur digunakan tenant.
- Buka settings billing, payment, notification, users.
- Buka Super Admin overview dan tenant detail.

## Monitoring Minimum

- HTTP healthcheck service.
- Log error 5xx per service.
- Queue notification pending/failed.
- Webhook payment failed.
- Database disk usage.
- Backup success/failure.
