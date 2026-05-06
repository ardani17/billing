# 15 - Super Admin / Owner Console

## Tujuan

Super Admin adalah role internal ISPBoss untuk owner/pengelola platform SaaS. Role ini tidak mewakili tenant ISP tertentu. Super Admin dipakai untuk melihat kondisi semua tenant, mengelola subscription dan add-on, membantu support, melakukan impersonate secara terkontrol, serta membaca audit global.

Super Admin tidak boleh mengedit data operasional tenant secara langsung. Jika perlu troubleshooting data tenant, Super Admin harus masuk melalui mekanisme impersonate tenant admin dengan alasan yang tercatat di audit log.

---

## Kondisi UI Saat Ini

Yang sudah tersedia:

- Overview platform.
- Daftar tenant.
- Detail tenant.
- Subscription tenant.
- Support page.
- Service health.
- Audit global.
- Platform settings.

Yang sudah live-backed:

- `/api/v1/admin/platform/overview`
- `/api/v1/admin/platform/tenants`
- `/api/v1/admin/platform/tenants/:id`
- `/api/v1/admin/platform/subscriptions`
- `/api/v1/admin/platform/support`
- `/api/v1/admin/platform/health`
- `/api/v1/admin/platform/audit`
- `/api/v1/admin/impersonate`
- `/api/v1/admin/stop-impersonate`

Catatan: beberapa endpoint masih tahap awal. Subscription masih dihitung dari plan tenant, support masih kosong, dan health masih sederhana.

---

## Ruang Lingkup Super Admin

### 1. Dashboard Owner

Dashboard Super Admin harus menampilkan kondisi bisnis dan operasional SaaS:

- Tenant aktif, trial, suspended, cancelled.
- MRR platform.
- Tenant baru periode berjalan.
- Trial yang akan habis.
- Subscription overdue.
- Upgrade request.
- Tenant bermasalah berdasarkan health, invoice terbuka, modul error, atau aktivitas terakhir.
- Service health ringkas.
- Audit global terbaru.

### 2. Tenant Management

Super Admin harus bisa mengelola tenant:

- Lihat semua tenant lintas platform.
- Filter/search tenant berdasarkan nama, domain, plan, status, health.
- Lihat detail tenant.
- Buat tenant manual jika diperlukan oleh tim ISPBoss.
- Edit profil tenant platform: nama tenant, domain, owner, status.
- Activate, suspend, resume, cancel tenant.
- Reset owner tenant admin.
- Verifikasi domain tenant.
- Lihat ringkasan modul aktif: Billing Core, MikroTik, Fiber Network.

### 3. Subscription dan Add-on Entitlement

Super Admin harus bisa mengelola paket komersial tenant:

| Paket / Add-on | Flag | Catatan |
|---|---|---|
| Billing Core | `billing_core` | Selalu aktif untuk tenant aktif |
| MikroTik | `mikrotik` | Add-on berbayar terpisah |
| OLT + Peta Jaringan | `fiber_network` | Satu add-on gabungan, tidak dipisah |

Fitur yang dibutuhkan:

- Lihat current plan tenant.
- Lihat tanggal mulai, renewal, trial expiry, status subscription.
- Aktifkan/nonaktifkan add-on MikroTik.
- Aktifkan/nonaktifkan add-on Fiber Network.
- Ubah plan.
- Catat alasan perubahan.
- Simpan audit entitlement changes.
- Preserve data add-on saat add-on dinonaktifkan.

### 4. Upgrade Request

Tenant Admin tidak boleh mengaktifkan add-on berbayar sendiri. Tenant Admin hanya mengirim upgrade request. Super Admin memproses request tersebut.

Status request:

- `pending`
- `approved`
- `rejected`
- `cancelled`

Minimal data:

- Tenant.
- Modul/plan yang diminta.
- Pesan dari tenant.
- Status.
- Diproses oleh.
- Alasan approve/reject.
- Timestamp.

### 5. Impersonate Tenant Admin

Impersonate harus dipakai untuk troubleshooting, bukan untuk bypass audit.

Flow:

1. Super Admin buka detail tenant.
2. Pilih tenant admin target.
3. Isi alasan impersonate.
4. Sistem membuat token impersonation.
5. UI menampilkan banner mode impersonate.
6. Semua aksi tenant tercatat dengan `impersonator_id`.
7. Super Admin bisa stop impersonate dan kembali ke console.

Aturan:

- Hanya boleh impersonate user role `tenant_admin`.
- Tidak boleh impersonate super admin lain.
- Alasan wajib diisi.
- Start/stop impersonate wajib masuk audit global.

### 6. Support Console

Support harus menjadi tempat tim ISPBoss menangani tiket lintas tenant.

Minimal fitur:

- Daftar tiket lintas tenant.
- Filter tenant, status, priority, assigned agent.
- Detail tiket.
- Komentar internal.
- Assign tiket ke tim.
- Ubah status.
- Link cepat ke tenant detail.
- Jika perlu, start impersonate dari tiket dengan alasan otomatis.

Status tiket:

- `open`
- `in_progress`
- `waiting_tenant`
- `resolved`
- `closed`

Priority:

- `low`
- `normal`
- `high`
- `urgent`

### 7. Service Health

Health Super Admin harus lebih dari tabel statis.

Yang perlu dipantau:

- Billing API.
- PostgreSQL.
- Redis.
- Network Service.
- Notification Service.
- Queue/asynq worker.
- Payment gateway status.
- Notification failure rate.
- Database latency.
- Error rate.
- Last successful cron/report/reminder.

Untuk tahap awal, cukup tampilkan status live dan detail error terakhir jika ada.

### 8. Audit Global

Audit global harus bisa dipakai untuk investigasi.

Fitur yang dibutuhkan:

- Filter tenant.
- Filter actor.
- Filter action.
- Filter date range.
- Search entity ID.
- Detail event/payload.
- Export CSV.
- Tandai event sensitif: impersonate, entitlement change, tenant suspend, owner reset, security policy change.

### 9. Platform Settings

Settings Super Admin harus mengelola konfigurasi platform, bukan hanya tabel statis.

Minimal:

- Plan defaults.
- Harga plan.
- Included modules.
- Trial days.
- Support contact.
- Security policy: MFA required, impersonate reason required, audit retention.
- Default tenant limits: customer limit, router limit, OLT limit, reseller limit.

---

## Navigasi Super Admin

Menu yang dibutuhkan:

- Overview
- Tenants
- Subscriptions
- Upgrade Requests
- Support
- Service Health
- Audit Global
- Platform Settings

Mobile Super Admin harus tetap bisa membuka semua menu. Jika bottom nav dibatasi, gunakan menu More/drawer.

---

## Prioritas Implementasi

1. Entitlement UI untuk paket dan add-on tenant.
2. Tenant lifecycle actions: edit, activate, suspend, resume, cancel.
3. Impersonate flow lengkap dengan alasan dan audit.
4. Subscription lifecycle dan upgrade request.
5. Support ticket console.
6. Audit global filter/search/detail.
7. Health monitoring yang lebih nyata.
8. Platform settings persistence.

---

## Acceptance Criteria

- Super Admin dapat melihat semua tenant dan detail tenant.
- Super Admin dapat mengubah plan dan add-on tenant.
- Perubahan add-on langsung mempengaruhi menu/API tenant sesuai module gating.
- Billing Core tetap berjalan walaupun MikroTik/Fiber Network tidak aktif.
- Setiap perubahan entitlement tercatat di audit.
- Super Admin dapat impersonate tenant admin dengan alasan.
- Mode impersonate terlihat jelas dan bisa dihentikan.
- Super Admin dapat melihat dan memfilter audit global.
- Super Admin dapat melihat status service utama.
- Semua menu Super Admin dapat diakses di mobile.
