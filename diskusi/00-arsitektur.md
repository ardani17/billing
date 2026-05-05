# 00 — Arsitektur & Keputusan Teknis

## Gambaran Umum

Aplikasi web billing untuk RT/RW Net dan ISP yang akan dijual sebagai **SaaS platform** bernama **ISPBoss** (`ispboss.id`). Setiap pembeli (operator ISP) adalah satu tenant yang bisa mengelola pelanggan, jaringan, dan billing mereka sendiri.

---

## Identitas Produk

### Nama Produk: ISPBoss
- **Domain utama:** `ispboss.id`
- **Landing page:** `ispboss.id` (public, SEO optimized)
- **Dashboard app:** `app.ispboss.id` (login required)
- **API:** `api.ispboss.id`
- **Tagline:** "Kelola ISP Kamu Dari Satu Dashboard"

### Konfigurasi Nama Produk (Dinamis)
```
SITE_NAME=ISPBoss
SITE_DOMAIN=ispboss.id
SITE_TAGLINE=Kelola ISP Kamu Dari Satu Dashboard
SITE_DESCRIPTION=Platform billing dan manajemen jaringan all-in-one untuk ISP dan RT/RW Net
```

### Free Trial
- **3 hari free trial** untuk semua tier
- Tidak perlu kartu kredit untuk mulai trial
- Setelah 3 hari: akun di-freeze, data tetap tersimpan 30 hari

---

## Fitur Utama

| No | Fitur | Deskripsi |
|---|---|---|
| 1 | Manajemen Pelanggan | Data pelanggan, assignment paket, status |
| 2 | Manajemen Paket | CRUD paket internet, profile bandwidth |
| 3 | Billing Core | Invoice otomatis, pembayaran manual + gateway, notifikasi, laporan, reseller/voucher |
| 4 | Add-on MikroTik | PPPoE/Hotspot, isolir otomatis, monitoring, VPN, backup (v6 & v7) |
| 5 | Add-on OLT + Peta Jaringan | Multi-brand OLT, provisioning ONT, ODP, peta interaktif, topologi OLT-ODP-ONT |
| 6 | White Label | Logo, warna, domain kustom per tenant |

---

## Deployment Model
- **SaaS** — di-host oleh penyedia, tenant tinggal pakai

---

## Arsitektur

- **Multi-service architecture** dengan **API-first approach**
- Frontend dan backend **100% terpisah**, berkomunikasi hanya via REST API
- Setiap service terpisah dan independen
- Clean architecture pattern (domain/usecase/repository/handler)

### Tech Stack
| Service | Teknologi | Alasan |
|---|---|---|
| Frontend | **Next.js** (SSR/CSR) | Dashboard, UI only — tidak ada business logic |
| **Semua Backend** | **Golang** | Satu bahasa, performa tinggi, konsisten |
| Database | **PostgreSQL** | Relational data |
| Cache/Queue | **Redis** | Caching, message queue, background job |
| API Gateway | **Nginx / Traefik** | Routing, rate limiting, SSL termination |

> **Full Golang Backend.** Tidak perlu campur bahasa. ~300k rps, goroutine untuk concurrency, satu toolchain.

### Golang Framework & Library
| Kebutuhan | Library |
|---|---|
| HTTP Framework | **Fiber** atau **Gin** |
| Database | **pgx** + **sqlc** |
| Migration | **golang-migrate** |
| Auth | **golang-jwt** |
| MikroTik | **go-routeros** |
| SNMP (OLT) | **gosnmp** |
| SSH (OLT) | **golang.org/x/crypto/ssh** |
| WebSocket | **gorilla/websocket** |
| PDF Invoice | **maroto** atau **gofpdf** |
| Queue/Worker | **asynq** (Redis-based) |
| Logging | **zerolog** |
| Validation | **go-playground/validator** |
| Config | **viper** |
| Testing | **testify** + **gomock** |
| Swagger | **swaggo/swag** |

---

## Arsitektur Diagram

```
                        ┌─────────────────────┐
                        │   API Gateway        │
                        │   (Nginx/Traefik)    │
                        │   api.ispboss.id     │
                        └──────────┬──────────┘
                                   │
              ┌────────────────────┼────────────────────┐
              │                    │                     │
    ┌─────────▼────────┐ ┌────────▼─────────┐ ┌────────▼─────────┐
    │  Billing API     │ │  Network Service  │ │  Notification    │
    │  (Golang)        │ │  (Golang)         │ │  Service         │
    │                  │ │                   │ │  (Golang)        │
    │  - Customer      │ │  - MikroTik API   │ │                  │
    │  - Invoice       │ │  - OLT (SNMP/SSH) │ │  - WhatsApp      │
    │  - Payment       │ │  - Monitoring     │ │  - SMS           │
    │  - Package       │ │  - FTTH Mapping   │ │  - Email         │
    │  - Auth/RBAC     │ │                   │ │                  │
    │  - White Label   │ │                   │ │                  │
    │  - Reporting     │ │                   │ │                  │
    └────────┬─────────┘ └────────┬──────────┘ └──────────────────┘
             │                    │
    ┌────────▼────────────────────▼───────┐
    │  PostgreSQL (Multi-tenant)           │
    │  + Redis (Cache/Queue)               │
    └──────────────────────────────────────┘

    ┌──────────────────────────────────────┐
    │  Frontend (Next.js)                  │
    │  app.ispboss.id                      │
    │  - SSR/CSR UI only                   │
    │  - Consume API dari api.ispboss.id   │
    │  - Auth via JWT token dari API       │
    └──────────────────────────────────────┘
```

### Domain & API Structure
```
ispboss.id              → Landing page (Next.js)
app.ispboss.id          → Dashboard (Next.js)
app.ispboss.id/reseller → Dashboard Reseller (Next.js)
api.ispboss.id          → API Gateway

  # Billing API
  /v1/auth/*            → Login, register, forgot password, JWT
  /v1/customers/*       → CRUD pelanggan, import/export
  /v1/areas/*           → CRUD area/wilayah pelanggan
  /v1/packages/*        → CRUD paket PPPoE & voucher
  /v1/invoices/*        → CRUD invoice, generate, prorate
  /v1/payments/*        → Catat pembayaran, riwayat, quick pay
  /v1/vouchers/*        → Generate, list, validate, bulk action
  /v1/resellers/*       → CRUD reseller, saldo, deposit
  /v1/credit-notes/*    → Credit note (refund, kompensasi)
  /v1/debit-notes/*     → Debit note (tagihan tambahan)
  /v1/tenants/*         → Profil tenant, subscription, modul
  /v1/reports/*         → Generate laporan, export, jadwal
  /v1/expenses/*        → Input pengeluaran (untuk laba rugi)
  /v1/audit-log/*       → Audit log (read-only)
  /v1/settings/*        → Konfigurasi billing, branding, user

  # Reseller API (subset, role-restricted)
  /v1/reseller/dashboard  → Ringkasan saldo, terjual, voucher
  /v1/reseller/vouchers/* → Beli, list, print voucher
  /v1/reseller/deposit/*  → Top-up saldo, riwayat deposit
  /v1/reseller/history/*  → Riwayat transaksi

  # Network Service
  /v1/mikrotik/*        → CRUD router, PPPoE user, queue, traffic
  /v1/mikrotik/vpn/*    → VPN tunnel management
  /v1/olt/*             → CRUD OLT, ONT, provisioning, alarm
  /v1/odp/*             → CRUD ODP/splitter
  /v1/devices/*         → Device Registry (semua perangkat)
  /v1/network-map/*     → FTTH mapping, node, jalur kabel

  # Notification Service
  /v1/notifications/*           → Kirim notifikasi, log
  /v1/notifications/templates/* → CRUD template notifikasi
  /v1/notifications/broadcast/* → Broadcast massal, jadwal
```

### Komunikasi Antar Service
| Komunikasi | Metode |
|---|---|
| Frontend → Backend | REST API via `api.ispboss.id` |
| Antar Backend Service | Event Queue via Redis |
| Backend → Frontend (realtime) | WebSocket / SSE |

### Event Contract (Redis Queue)
Daftar event yang dikirim antar service:

| Event | Publisher | Consumer | Deskripsi |
|---|---|---|---|
| `customer.created` | Billing API | Network, Notification | Pelanggan baru dibuat |
| `customer.activated` | Billing API | Network, Notification | Pelanggan diaktivasi → buat user PPPoE + provisioning ONT |
| `customer.isolated` | Billing API | Network, Notification | Pelanggan diisolir → disable PPPoE + redirect walled garden |
| `customer.suspended` | Billing API | Network, Notification | Pelanggan di-suspend → hapus user PPPoE dari router |
| `customer.unblocked` | Billing API | Network, Notification | Isolir dibuka → enable PPPoE + hapus redirect |
| `customer.terminated` | Billing API | Network, Notification | Pelanggan berhenti → hapus PPPoE + decommission ONT |
| `package.changed` | Billing API | Network, Notification | Upgrade/downgrade → update profile + reconnect |
| `package.price_changed` | Billing API | Notification | Harga paket berubah → notifikasi pelanggan terdampak |
| `invoice.created` | Billing API | Notification | Invoice baru → kirim notifikasi + payment link |
| `invoice.reminder` | Billing API | Notification | Reminder H-1/H+1/H+3 → kirim notifikasi |
| `payment.received` | Billing API | Network, Notification | Pembayaran diterima → buka isolir + konfirmasi |
| `payment.gateway_webhook` | Billing API | Notification | Webhook dari Xendit/Midtrans |
| `mikrotik.router_offline` | Network Service | Notification | Router offline 3x gagal → notifikasi teknisi |
| `mikrotik.router_online` | Network Service | Notification | Router kembali online → notifikasi teknisi |
| `mikrotik.unexpected_reboot` | Network Service | Notification | Uptime reset terdeteksi → notifikasi teknisi |
| `mikrotik.sync_failed` | Network Service | Notification | Sinkronisasi gagal → notifikasi admin |
| `mikrotik.command_result` | Network Service | Billing API | Hasil perintah (sukses/gagal) → update status pelanggan |
| `mikrotik.pool_warning` | Network Service | Notification | IP pool > 80% → notifikasi admin |
| `olt.alarm` | Network Service | Notification | Alarm OLT (LOS, dying gasp, dll) → notifikasi teknisi |
| `olt.ont_detected` | Network Service | Notification | ONT baru terdeteksi (unregistered) → notifikasi teknisi |
| `notification.sent` | Notification | Billing API | Notifikasi berhasil dikirim → log |
| `notification.failed` | Notification | Billing API | Notifikasi gagal → log + alert jika banyak gagal |
| `vpn.tunnel_down` | Network Service | Notification | VPN tunnel putus → notifikasi admin |
| `vpn.tunnel_up` | Network Service | Notification | VPN tunnel kembali terhubung |

### Event Payload Schema

Semua event menggunakan format JSON:

```json
{
  "event_type": "customer.isolated",
  "tenant_id": "t-001",
  "timestamp": "2026-04-28T14:30:00+07:00",
  "correlation_id": "uuid-v4",
  "payload": {
    // isi tergantung event_type
  }
}
```

Contoh payload per event:

```json
// customer.activated
"payload": {
  "customer_id": "PLG-001",
  "name": "Ahmad Rizki",
  "package_id": "pkg-pro-50m",
  "connection_method": "pppoe",
  "pppoe_username": "ahmad-plg001",
  "pppoe_password": "encrypted",
  "router_id": "mk-01",
  "olt_id": "olt-01",
  "ont_sn": "ZTEG12345678"
}

// payment.received
"payload": {
  "customer_id": "PLG-001",
  "invoice_id": "INV-2026-04-001",
  "amount": 388500,
  "method": "xendit_va_bca",
  "is_full_payment": true,
  "customer_status": "isolated"
}

// mikrotik.command_result
"payload": {
  "router_id": "mk-01",
  "customer_id": "PLG-001",
  "command": "ppp_secret_set_disabled",
  "success": true,
  "error": null,
  "execution_time_ms": 150
}
```

### Device Registry
Semua perangkat jaringan (router MikroTik, OLT, ODP) disimpan dalam Device Registry di Network Service:

```
/v1/devices/*
  ├── /v1/devices/routers      → Daftar router MikroTik
  ├── /v1/devices/olts         → Daftar OLT
  ├── /v1/devices/odps         → Daftar ODP (splitter)
  └── /v1/devices/onts         → Daftar ONT (per pelanggan)
```

Relasi perangkat ke pelanggan:
```
Pelanggan → Router MikroTik (PPPoE user)
Pelanggan → OLT → ODP → ONT (FTTH path)
```

- Setiap perangkat punya: `id`, `tenant_id`, `name`, `type`, `ip_address`, `credentials` (encrypted), `status`, `last_seen`
- Credential perangkat **terenkripsi** di database, tidak plain text
- Status perangkat: Online, Offline, Maintenance, Unknown

---

## Multi-Tenant Strategy
- **Shared DB, shared schema** dengan `tenant_id` di setiap tabel
- Target: 1.000 tenant × 1.000 pelanggan = **1 juta pelanggan**
- Indexing pada `tenant_id`
- Opsi shard ke DB terpisah untuk tenant besar

### Multi-Tenant Isolation (Detail Implementasi)

```
Request masuk (JWT token)
  │
  ▼
API Middleware: Extract tenant_id dari JWT claims
  │
  ▼
Inject tenant_id ke request context (ctx)
  │
  ▼
Repository layer: SEMUA query WAJIB filter WHERE tenant_id = ctx.TenantID
  │
  ▼
Database: Row Level Security (RLS) sebagai safety net
```

**Lapisan keamanan:**

| Lapisan | Mekanisme | Tujuan |
|---|---|---|
| **1. JWT Token** | `tenant_id` di-embed dalam JWT claims saat login | Identifikasi tenant |
| **2. API Middleware** | Extract `tenant_id` dari token, inject ke context | Propagasi tenant ke semua layer |
| **3. Repository** | Semua query wajib pakai `WHERE tenant_id = ?` | Filter data per tenant |
| **4. RLS (PostgreSQL)** | Policy: `USING (tenant_id = current_setting('app.tenant_id'))` | Safety net jika developer lupa filter |
| **5. Audit** | Log semua query dengan tenant_id | Deteksi anomali |

**RLS Policy (PostgreSQL):**
```sql
-- Set tenant_id di session setiap request
SET app.tenant_id = '{tenant_id_from_jwt}';

-- Policy di setiap tabel
ALTER TABLE customers ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON customers
  USING (tenant_id = current_setting('app.tenant_id')::uuid);
```

**Aturan:**
- Developer **DILARANG** query tanpa filter `tenant_id`
- RLS adalah **safety net terakhir**, bukan pengganti filter di kode
- Super Admin bisa bypass RLS (untuk support/troubleshooting)
- Credential perangkat (MikroTik, OLT) terenkripsi per tenant (AES-256)
- File upload (foto, backup) disimpan di path terpisah per tenant

---

## Prinsip Kode
- **Maksimal 200 baris per file**
- Clean architecture — separation of concerns
- Modular dan scalable
- **Komentar kode WAJIB berbahasa Indonesia**
- Nama variabel dan fungsi tetap **bahasa Inggris**

### Contoh Komentar Golang
```go
// BuatUserPPPoE membuat user PPPoE baru di router MikroTik.
// Mengembalikan error jika koneksi ke router gagal.
func (s *MikroTikService) BuatUserPPPoE(ctx context.Context, user PPPoEUser) error {
    // Validasi input sebelum kirim ke router
    if err := user.Validate(); err != nil {
        return fmt.Errorf("validasi gagal: %w", err)
    }
    // Kirim perintah ke RouterOS API
    reply, err := s.client.Run("/ppp/secret/add", ...)
}
```

### Contoh Komentar TypeScript
```typescript
/**
 * Hook untuk mengambil daftar pelanggan dari API.
 * Menggunakan SWR untuk caching dan revalidasi otomatis.
 */
export function usePelanggan(tenantId: string, filters?: FilterPelanggan) { ... }
```

### Contoh Komentar SQL
```sql
-- Tabel pelanggan: menyimpan data semua pelanggan ISP per tenant.
-- Field tenant_id digunakan untuk isolasi data antar tenant.
CREATE TABLE customers ( ... );
```

---

## Strategi Anti-Lemot (Performance)

**Response Time Target:**
| Operasi | Target |
|---|---|
| GET list | < 100ms |
| GET detail | < 50ms |
| POST create/update | < 200ms |
| Dashboard load | < 500ms |
| Generate laporan | Async |
| Kirim notifikasi massal | Async |
| Sync MikroTik | Async |

**Caching:** Redis cache → invalidate saat data berubah
**Async:** Background job via asynq, notify via WebSocket
**DB:** Composite index, partial index, connection pooling, read replica
**Frontend:** SWR/React Query, optimistic UI, lazy loading, SSR

---

## API Testing Strategy

**Layer 1:** Swagger UI (auto-generate dari kode, test dari browser)
**Layer 2:** Bruno (pengganti Postman, collection di Git)
**Layer 3:** Go automated test (testify, testcontainers-go, httptest, rapid)

---

## Keamanan Perangkat Saat Development (KRITIS)

**DILARANG** menjalankan perintah langsung ke perangkat OLT/MikroTik production saat development.

- Default `NETWORK_MODE=mock`
- Mock/stub untuk development & unit test
- MikroTik CHR (VM) untuk integration test
- Credential di environment variable, TIDAK di kode/Git

---

## Monorepo Structure (Turborepo)
```
apps/
  web/              → Next.js (frontend only)
services/
  billing-api/      → Golang
  network-service/  → Golang
  notification/     → Golang
pkg/
  database/         → Shared DB, migration, tenant resolver
  auth/             → Shared JWT, RBAC
  tenant/           → Shared tenant context
  queue/            → Shared Redis/asynq
  logger/           → Shared zerolog
packages/
  ui/               → Shared React UI (shadcn/ui)
  types/            → Shared TypeScript types
  config/           → Shared eslint, tsconfig
api-tests/          → Bruno collection
docker/             → Dockerfile + docker-compose
```

---

## Hosting
- Awal: **VPS** + Docker Compose
- Scale: **Kubernetes** di AWS/GCP saat 100+ tenant
- Cloud-ready dari awal (stateless, env vars)

---

## Pricing Model
| Tier | Pelanggan | Harga/Bulan |
|---|---|---|
| Starter | 0-100 | Rp 150.000 |
| Growth | 101-500 | Rp 350.000 |
| Pro | 501-2.000 | Rp 750.000 |
| Enterprise | 2.000+ | Custom |

---

## Keamanan & Operasional

### Multi-Tenant Security
- Row Level Security (RLS) di PostgreSQL
- Setiap query WAJIB filter `tenant_id`
- Audit log operasi sensitif

### RBAC
| Role | Akses |
|---|---|
| Super Admin | Semua tenant |
| Tenant Admin | Full access tenant sendiri |
| Operator | Operasional harian |
| Teknisi | Network/OLT/MikroTik |
| Kasir | Input pembayaran |
| Reseller | Dashboard reseller: beli voucher, print, deposit, riwayat |

### Offline Resilience (Detail)
Strategi saat perangkat (MikroTik/OLT) tidak bisa dihubungi:

```
Perintah gagal (router/OLT offline)
  │
  ▼
Masuk retry queue (asynq):
  Retry 1: langsung
  Retry 2: 5 menit
  Retry 3: 30 menit
  Retry 4: 2 jam
  Retry 5: 6 jam
  │
  ▼
Semua retry gagal (max 24 jam):
  → Status: "Pending Sync"
  → Notifikasi ke admin/teknisi
  → Database tetap diupdate (DB = source of truth untuk MikroTik)
  │
  ▼
Perangkat kembali online:
  → Periodic sync mendeteksi perbedaan
  → Auto-fix: sinkronisasi DB → perangkat
  → Log: "Sync recovered — {X} perintah tertunda berhasil dieksekusi"
```

**Conflict resolution:**
- **MikroTik**: Database = source of truth. Jika data berbeda, update router sesuai DB.
- **OLT**: OLT = source of truth untuk data fisik (SN, port, signal). DB diupdate sesuai OLT.
- **Orphan user** (ada di router tapi tidak di DB): tidak dihapus otomatis, tampilkan untuk review admin.
- Conflict resolution

### Database Migration (Zero-Downtime)
- Backward compatible, rollback plan
- `golang-migrate` dengan file SQL bernomor

### Backup & Disaster Recovery
- Automated daily backup, PITR
- RPO < 1 jam, RTO < 4 jam
- Multi-region backup

### Rate Limiting per Tenant
| Tier | Rate Limit |
|---|---|
| Starter | 60 req/menit |
| Growth | 200 req/menit |
| Pro | 500 req/menit |
| Enterprise | Custom |

### Monitoring & Alerting
- Health check per service
- Prometheus + Grafana
- Alerting ke Telegram/WA/Email
- Centralized logging (Grafana Loki)

### Monitoring Metrics & Alert Thresholds

| Metric | Source | Threshold | Alert Channel |
|---|---|---|---|
| API Response Time | Prometheus | > 500ms (warning), > 2s (critical) | Telegram ops |
| API Error Rate | Prometheus | > 1% (warning), > 5% (critical) | Telegram ops |
| DB Connection Pool | Prometheus | > 80% used (warning) | Telegram ops |
| Redis Queue Depth | Prometheus | > 1000 jobs pending (warning) | Telegram ops |
| Disk Usage | Prometheus | > 80% (warning), > 90% (critical) | Email ops |
| Router Offline | Network Service | 3x gagal health check | WA teknisi tenant |
| OLT Offline | Network Service | 3x gagal SNMP ping | WA teknisi tenant |
| VPN Tunnel Down | Network Service | Disconnect > 1 menit | WA admin tenant |
| Notifikasi Error Rate | Notification Service | > 10% gagal | Email admin tenant |
| Invoice Generate Gagal | Billing API | Any failure | WA admin tenant |

**Escalation:**
- Alert tidak di-acknowledge dalam 15 menit → escalate ke level berikutnya
- Level 1: Teknisi/Operator tenant → Level 2: Tenant Admin → Level 3: Tim ISPBoss

---

## UI/UX Global

### Tema: Light & Dark Mode dengan Biru
| Token | Light | Dark |
|---|---|---|
| Primary | #2563EB (Blue 600) | #3B82F6 (Blue 500) |
| Background | #FFFFFF | #0F172A (Slate 900) |
| Surface | #F8FAFC (Slate 50) | #1E293B (Slate 800) |
| Border | #E2E8F0 (Slate 200) | #334155 (Slate 700) |
| Text Primary | #0F172A | #F8FAFC |
| Text Secondary | #64748B | #94A3B8 |

### Design Philosophy (taste-skill)
- DESIGN_VARIANCE: 5, MOTION_INTENSITY: 4, VISUAL_DENSITY: 6
- Typography: Geist + Geist Mono
- Icons: Phosphor Icons
- Components: shadcn/ui (dikustomisasi)
- Animasi: Framer Motion (spring physics)

### Responsive (Mobile-First)
- Breakpoints: default < 640, sm ≥ 640, md ≥ 768, lg ≥ 1024, xl ≥ 1280, 2xl ≥ 1536
- Touch target min 44x44px, font min 16px mobile
- `min-h-[100dvh]` bukan `h-screen`

---

## Alur Bisnis Penting

### Customer Status State Machine

```
                    ┌──────────┐
                    │ Pending  │ ← Pelanggan baru didaftarkan
                    └────┬─────┘
                         │ Admin aktivasi
                         ▼
                    ┌──────────┐
          ┌────────│  Aktif   │◄────────────────────┐
          │        └────┬─────┘                     │
          │             │                           │
          │             │ Invoice belum bayar        │ Pembayaran diterima
          │             │ > grace period (7 hari)    │ (auto buka isolir)
          │             ▼                           │
          │        ┌──────────┐                     │
          │        │  Isolir  │─────────────────────┘
          │        └────┬─────┘
          │             │
          │             │ Belum bayar > batas toleransi (30 hari)
          │             ▼
          │        ┌──────────┐
          │        │ Suspend  │ ← Koneksi dimatikan total, user dihapus dari router
          │        └────┬─────┘
          │             │ Admin reaktivasi manual (bayar tunggakan + biaya aktivasi)
          │             │
          │             ▼
          │        ┌──────────┐
          └───────►│ Berhenti │ ← Admin terminasi (dari status apapun)
                   └──────────┘
```

| Transisi | Trigger | Aksi di MikroTik | Aksi di OLT | Notifikasi |
|---|---|---|---|---|
| Pending → Aktif | Admin aktivasi | Buat user PPPoE | Provisioning ONT | Welcome |
| Aktif → Isolir | Invoice > grace period | Disable user, redirect walled garden | Tidak ada | Isolir notice |
| Isolir → Aktif | Pembayaran diterima | Enable user, hapus redirect | Tidak ada | Buka isolir |
| Isolir → Suspend | Invoice > batas toleransi | Hapus user dari router | Tidak ada | Suspend notice |
| Suspend → Aktif | Admin manual + bayar | Buat ulang user PPPoE | Tidak ada | Reaktivasi |
| Any → Berhenti | Admin terminasi | Hapus user dari router | Decommission ONT | - |

### Invoice Status (terpisah dari Customer Status)

```
Belum Bayar → Terlambat → Lunas
                ↓
           Bayar Sebagian → Lunas
                
Belum Bayar → Batal (admin cancel)
Prorate (invoice khusus upgrade/downgrade)
```

> **Penting:** Customer status dan invoice status adalah 2 hal berbeda. Customer bisa Aktif tapi punya invoice Terlambat (masih dalam grace period). Customer Isolir pasti punya invoice Terlambat.

### Terminologi Standar

| Istilah | Definisi | Level |
|---|---|---|
| **Isolir** | Internet di-redirect ke walled garden, user PPPoE disabled | Customer status (MikroTik) |
| **Suspend** | Internet dimatikan total, user PPPoE dihapus dari router | Customer status (MikroTik) |
| **Disable** | Aksi teknis di MikroTik: `/ppp/secret/set disabled=yes` | MikroTik action (= isolir) |
| **Remove** | Aksi teknis di MikroTik: `/ppp/secret/remove` | MikroTik action (= suspend) |
| **Decommission** | Hapus ONT dari OLT | OLT action (saat pelanggan berhenti) |
| **Void** | Batalkan voucher/invoice/pembayaran | Billing action |

### Billing + MikroTik
```
Pelanggan baru aktif    → Buat user PPPoE/Hotspot
Pembayaran diterima     → Enable user, reset quota
Jatuh tempo belum bayar → Redirect ke walled garden
Lewat grace period      → Disable/suspend user
Upgrade/downgrade       → Update bandwidth profile
Pelanggan berhenti      → Hapus user dari MikroTik
```

### White Label
```
Tenant daftar → Setup branding → Pelanggan akses via custom domain
```

---

## Strategi Pengembangan Bertahap

### Product Packaging & Module Registry (per tenant)

ISPBoss dijual sebagai paket SaaS modular. Paket dasar adalah **Billing Core**.
Yang dijual terpisah hanya **Add-on MikroTik** dan **Add-on OLT + Peta Jaringan**.
Notifikasi, laporan, payment gateway, reseller/voucher, dan settings dasar adalah bagian dari Billing Core.

| Paket Komersial | Module Flag | Isi |
|---|---|---|
| Billing Core | `billing_core` | Auth, tenant, pelanggan, paket, invoice, pembayaran, payment gateway, notifikasi, laporan, reseller/voucher, settings |
| Add-on MikroTik | `mikrotik` | Router, PPPoE/Hotspot, isolir teknis, session, traffic, VPN, backup/firmware, sync |
| Add-on OLT + Peta Jaringan | `fiber_network` | OLT, ONT, ODP, provisioning, alarm, peta jaringan, FTTH mapping, topologi fiber |

```
billing_core -> always
mikrotik -> optional add-on
fiber_network -> optional add-on (OLT + Peta Jaringan)
```

### Prinsip: Loose Coupling + Event-Driven + Graceful Degradation
- NoOp/Stub implementation untuk modul belum aktif
- Event diabaikan jika penerima belum ada
- Menu/widget hidden untuk modul non-aktif
- API untuk add-on nonaktif mengembalikan error aman `MODULE_NOT_ENABLED`, bukan crash
- Billing Core tetap berjalan penuh meskipun MikroTik atau Fiber Network tidak dibeli
- Event billing ke MikroTik hanya diproses jika add-on MikroTik aktif
- Event OLT/Peta hanya diproses jika add-on Fiber Network aktif

### Urutan Pengembangan
```
billing_core (wajib)
  |-- customer/package/invoice/payment/notification/reporting/settings
  |-- mikrotik (optional add-on)
  `-- fiber_network (optional add-on: OLT + Peta Jaringan)
```

---

## Rencana Spec (Urutan Final)

| No | Spec | Deskripsi |
|---|---|---|
| 1 | Project Foundation | Monorepo, DB, auth, RBAC, tenant, white label |
| 2 | Customer & Package | CRUD pelanggan, paket internet |
| 3 | Billing Core | Invoice otomatis, pembayaran, payment gateway, notifikasi, laporan |
| 4 | MikroTik Add-on | RouterOS v6/v7, PPPoE, isolir, monitoring |
| 5 | Fiber Network Add-on | OLT + Peta Jaringan, provisioning ONT, topologi |
| 6 | Reporting & Analytics | Laporan keuangan, dashboard |
