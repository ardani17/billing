# 03 — Dashboard Layout & Navbar (`app.ispboss.id/dashboard`)

**Tujuan:** Pusat kontrol utama untuk operator ISP.

---

## Layout Desktop

```
╔══════════════════════════════════════════════════════════════════════════╗
║ ┌──────────┬─────────────────────────────────────────────────────────┐  ║
║ │          │  TOPBAR                                                │  ║
║ │          │  🔍 Cari pelanggan, invoice...   ☀️🌙  🔔  👤         │  ║
║ │          ├─────────────────────────────────────────────────────────┤  ║
║ │ SIDEBAR  │                                                        │  ║
║ │          │  ┌──────────────┬──────────────┬──────────────┐        │  ║
║ │ 🔵ISPBoss│  │ Total Plgn   │ Pendapatan   │ Tunggakan    │        │  ║
║ │          │  │ 847          │ Rp 12.5jt    │ Rp 3.2jt     │        │  ║
║ │ Dashboard│  └──────────────┴──────────────┴──────────────┘        │  ║
║ │ Pelanggan│                                                        │  ║
║ │ Paket    │  ┌──────────────────────┬───────────────────┐          │  ║
║ │ Reseller │  │ Grafik Pendapatan    │ Pelanggan Baru    │          │  ║
║ │ ━━━━━━━━ │  │ 6 Bulan Terakhir     │ Minggu Ini        │          │  ║
║ │ Invoice  │  └──────────────────────┴───────────────────┘          │  ║
║ │ Pembayaran│                                                        │  ║
║ │ Voucher  │  ┌──────────────────────┬───────────────────┐          │  ║
║ │ ━━━━━━━━ │  │ Invoice Terbaru      │ Aktivitas Terkini │          │  ║
║ │ MikroTik │  └──────────────────────┴───────────────────┘          │  ║
║ │ OLT      │                                                        │  ║
║ │ Peta     │  ┌──────────────────────────────────────────┐          │  ║
║ │ ━━━━━━━━ │  │ Status Router (full width)               │          │  ║
║ │ Notif    │  └──────────────────────────────────────────┘          │  ║
║ │ Laporan  │                                                        │  ║
║ │ ━━━━━━━━ │                                                        │  ║
║ │ Settings │                                                        │  ║
║ │ Bantuan  │                                                        │  ║
║ │ «Collapse│                                                        │  ║
║ └──────────┴─────────────────────────────────────────────────────────┘  ║
╚══════════════════════════════════════════════════════════════════════════╝
```

## Layout Mobile

```
╔══════════════════════════╗
║  ☰ ISPBoss        🔔 👤  ║
╠══════════════════════════╣
║  [Total Plgn: 847]       ║
║  [Pendapatan: Rp 12.5jt] ║
║  [Tunggakan: Rp 3.2jt]   ║
║  [Grafik Pendapatan]      ║
║  [Invoice Terbaru]        ║
╠══════════════════════════╣
║  📊  👥  💰  🔧  ⋯       ║
║  Home Plgn Bill MT More   ║
╚══════════════════════════╝
```

---

## Sidebar

### Responsive Behavior
| Layar | Sidebar |
|---|---|
| Desktop ≥ 1280px (`xl`) | Expanded (240px) — icon + teks |
| Laptop 1024-1279px (`lg`) | Collapsed (64px) — icon only, hover tooltip |
| Tablet & Mobile < 1024px | Hidden — hamburger buka drawer overlay |

- Tombol collapse/expand di bawah sidebar
- Preferensi disimpan di `localStorage`
- Transisi smooth 200ms

### Warna Sidebar (Ikut Tema)
| Elemen | Light | Dark |
|---|---|---|
| Background | #FFFFFF | #0F172A |
| Border kanan | #E2E8F0 | #334155 |
| Menu text | #64748B | #94A3B8 |
| Menu hover bg | #F1F5F9 | #1E293B |
| Menu active bg | #EFF6FF | #1E3A8A/20 |
| Menu active text | #2563EB | #3B82F6 |
| Active left border | #2563EB | #3B82F6 |

### Menu Sidebar
| Group | Menu | Icon (Phosphor) | Badge |
|---|---|---|---|
| **Utama** | Dashboard | SquaresFour | - |
| | Pelanggan | Users | - |
| | Paket Internet | Package | - |
| | Reseller | Storefront | - |
| **Billing** | Invoice | Receipt | Jumlah unpaid |
| | Pembayaran | CreditCard | Hari ini |
| | Voucher | Ticket | - |
| **Network** | MikroTik | WifiHigh | Router offline |
| | OLT | Broadcast | - |
| | Peta Jaringan | MapTrifold | - |
| **Komunikasi** | Notifikasi | ChatCircleDots | Pending |
| **Laporan** | Laporan | ChartLineUp | - |
| **Sistem** | Pengaturan | GearSix | - |
| | Bantuan | Question | - |

### Halaman Bantuan (`/help`)
- **Panduan Pengguna**: dokumentasi fitur per modul (inline help)
- **FAQ**: pertanyaan umum per kategori
- **Troubleshooting**: panduan per modul (MikroTik, OLT, billing)
- **Hubungi Support**: form tiket ke tim ISPBoss
- **Changelog**: riwayat update fitur ISPBoss
- **Video Tutorial**: link ke video panduan (YouTube/embed)
- Konten dikelola oleh tim ISPBoss (bukan per tenant)

### Menu per Role (RBAC)
| Menu | Super Admin | Tenant Admin | Operator | Teknisi | Kasir | Reseller |
|---|---|---|---|---|---|---|
| Dashboard | ✅ (semua tenant) | ✅ | ✅ | ✅ | ✅ | ✅ (reseller) |
| Pelanggan | ✅ | ✅ | ✅ | ❌ | 👁️ | ❌ |
| Paket | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |
| Reseller | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |
| Invoice | ✅ | ✅ | ✅ | ❌ | ✅ | ❌ |
| Pembayaran | ✅ | ✅ | ✅ | ❌ | ✅ | ❌ |
| Voucher | ✅ | ✅ | ✅ | ❌ | ❌ | ✅ (beli/print) |
| MikroTik | ✅ | ✅ | ✅ | ✅ | ❌ | ❌ |
| OLT | ✅ | ✅ | ❌ | ✅ | ❌ | ❌ |
| Peta | ✅ | ✅ | ❌ | ✅ | ❌ | ❌ |
| Notifikasi | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |
| Laporan | ✅ | ✅ | 👁️ | ❌ | ❌ | ❌ |
| Pengaturan | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ |
| Bantuan | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |

Menu tidak diizinkan = **hidden** (bukan disabled).

### Detail Role

| Role | Deskripsi | Scope |
|---|---|---|
| **Super Admin** | Tim ISPBoss. Akses semua tenant, manage subscription, support. | Lintas tenant |
| **Tenant Admin** | Pemilik ISP. Full access tenant sendiri, kelola user, settings. | 1 tenant |
| **Operator** | Staff operasional. Kelola pelanggan, billing, notifikasi. Tidak bisa akses settings. | 1 tenant |
| **Teknisi** | Staff teknis. Kelola MikroTik, OLT, peta. Tidak bisa akses billing. | 1 tenant |
| **Kasir** | Staff keuangan. Input pembayaran, lihat pelanggan (read-only). | 1 tenant |
| **Reseller** | Mitra penjual voucher. Dashboard terpisah, beli/print voucher, deposit. | 1 tenant |

> **Super Admin** adalah role internal ISPBoss (bukan tenant). Super Admin bisa: lihat semua tenant, manage subscription, impersonate tenant admin untuk troubleshooting, akses audit log global. Super Admin **tidak bisa** edit data pelanggan tenant secara langsung (harus impersonate dulu).

---

## Topbar

### Desktop
```
🔍 Cari pelanggan, invoice, router...    ☀️/🌙   🔔(3)  👤
```
| Elemen | Fungsi |
|---|---|
| Global Search | Cari pelanggan, invoice, router — dropdown real-time |
| Toggle Tema | Switch light/dark |
| Bell | Badge count, dropdown notifikasi terbaru |
| Avatar | Dropdown: nama, role, Profil, Settings, Logout |

### Mobile
```
☰    ISPBoss / Logo    🔔 👤
```
- Hamburger buka sidebar drawer
- Search di dalam drawer atau halaman terpisah

### Warna Topbar
| Elemen | Light | Dark |
|---|---|---|
| Background | #FFFFFF | #0F172A |
| Border bawah | #E2E8F0 | #334155 |
| Search bg | #F1F5F9 | #1E293B |
| Icon | #64748B | #94A3B8 |
| Badge notif | #DC2626 | #EF4444 |

---

## Bottom Nav (Mobile Only)

```
📊        👥        💰       🔧       ⋯
Home    Pelanggan  Billing  MikroTik  More
```
- "More" buka bottom sheet menu lengkap
- Active: filled icon + Blue 600/500
- Inactive: outline icon + Slate 400
- Disesuaikan per role

---

## Dashboard Widgets

### Baris 1 — Stat Cards (4 kolom desktop, 2 tablet, 1 mobile)
| Widget | Data | Icon | Warna |
|---|---|---|---|
| Total Pelanggan | Pelanggan aktif | Users | Blue |
| Pendapatan Bulan Ini | Pembayaran diterima | CurrencyCircleDollar | Green |
| Tunggakan | Invoice belum bayar | Warning | Amber |
| Router Online | Router MikroTik online | WifiHigh | Green/Red |

Setiap card: angka besar (Geist Mono), label, % perubahan (↑↓), sparkline mini.

### Baris 2 — Grafik (60/40)
| Widget | Ukuran | Deskripsi |
|---|---|---|
| Grafik Pendapatan | 60% | Line chart 6 bulan |
| Pelanggan Baru | 40% | List 5 terbaru |

### Baris 3 — Tabel & Aktivitas (60/40)
| Widget | Ukuran | Deskripsi |
|---|---|---|
| Invoice Terbaru | 60% | 5 invoice terakhir |
| Aktivitas Terkini | 40% | Log: bayar, baru, offline |

### Baris 4 — Network Status (full width, jika MikroTik aktif)
| Widget | Deskripsi |
|---|---|
| Status Router | List router: nama, IP, status, uptime, user aktif |

### Widget Behavior
- Loading skeleton saat data belum siap
- Empty state jika belum ada data
- Modul non-aktif → widget hidden
- Auto-refresh 30 detik via SWR
- Klik → navigasi ke halaman detail

### Widget Data Source Mapping

| Widget | Source Service | API Endpoint | Update | Cache TTL |
|---|---|---|---|---|
| Total Pelanggan | Billing API | `/v1/customers/stats` | Real-time | 5 menit |
| Pendapatan Bulan Ini | Billing API | `/v1/reports/revenue/current` | Real-time | 1 menit |
| Tunggakan | Billing API | `/v1/reports/aging/summary` | Real-time | 5 menit |
| Router Online | Network Service | `/v1/mikrotik/status/summary` | Real-time | 30 detik |
| Grafik Pendapatan | Billing API | `/v1/reports/revenue/chart` | Hourly | 1 jam |
| Pelanggan Baru | Billing API | `/v1/customers/recent` | Real-time | 5 menit |
| Invoice Terbaru | Billing API | `/v1/invoices/recent` | Real-time | 1 menit |
| Aktivitas Terkini | Billing API | `/v1/audit-log/recent` | Real-time | 30 detik |
| Status Router | Network Service | `/v1/mikrotik/status/list` | Real-time | 30 detik |
- Modul non-aktif → widget hidden
- Auto-refresh 30 detik via SWR
- Klik → navigasi ke halaman detail

### Dashboard per Role
| Role | Widget |
|---|---|
| Tenant Admin | Semua |
| Operator | Semua |
| Teknisi | Router Online, Status Router, Aktivitas Network |
| Kasir | Pendapatan, Tunggakan, Invoice Terbaru, Pembayaran Hari Ini |
