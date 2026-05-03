# 08 — Integrasi MikroTik

---

## Konsep Integrasi

Network Service (Golang) berkomunikasi dengan router MikroTik via **RouterOS API** (library `go-routeros`). Semua perintah ke router dijalankan secara **async** melalui Redis queue, bukan langsung dari request HTTP user.

```
Billing API                          Network Service
  │                                    │
  ├── Event: customer.activated ──────►├── Buat user PPPoE di router
  ├── Event: customer.isolated ───────►├── Disable user, redirect walled garden
  ├── Event: customer.unblocked ──────►├── Enable user, hapus redirect
  ├── Event: customer.suspended ──────►├── Hapus user dari router
  ├── Event: customer.terminated ─────►├── Hapus user dari router
  ├── Event: package.changed ─────────►├── Update bandwidth profile
  │                                    │
  │         Redis Queue (asynq)        │
  │                                    │
  │◄── Event: mikrotik.command_result ─┤── Lapor hasil (sukses/gagal)
  │◄── Event: mikrotik.router_offline ─┤── Router tidak bisa dihubungi
  │◄── Event: mikrotik.sync_failed ────┤── Sinkronisasi gagal
```

### Dukungan Versi RouterOS
| Versi | Status | Keterangan |
|---|---|---|
| RouterOS v6 (6.x) | ✅ Didukung | Mayoritas ISP masih pakai v6 |
| RouterOS v7 (7.x) | ✅ Didukung | Versi terbaru, beberapa API berbeda |

- Kode **terpisah** untuk v6 dan v7 (adapter pattern per versi)
- Auto-detect versi saat tambah router
- Perbedaan utama: path API queue, beberapa parameter baru di v7

### Connection Pool + Lazy Connect + Event-Driven Warm-Up

Strategi koneksi ke router menggunakan **Connection Pool dengan Lazy Connect dan Predictive Warm-Up**:

```
┌─────────────────────────────────────────────────────────────┐
│  Per Router (via VPN IP)                                     │
│                                                              │
│  ┌─── Command Pool (Lazy, 3-5 koneksi) ──────────────────┐  │
│  │  Untuk: CRUD user, isolir, sync, backup                │  │
│  │  Koneksi dibuat saat dibutuhkan (lazy)                 │  │
│  │  Idle timeout: 5 menit → tutup otomatis                │  │
│  │  Max lifetime: 1 jam → tutup & buat baru               │  │
│  │  Health ping: setiap 30 detik, buang yang mati         │  │
│  └────────────────────────────────────────────────────────┘  │
│                                                              │
│  ┌─── Monitor Connection (Persistent, 1 koneksi) ────────┐  │
│  │  Untuk: traffic real-time, CPU, RAM, uptime             │  │
│  │  Selalu terhubung selama router online                  │  │
│  │  Auto-reconnect jika putus (backoff: 5s, 15s, 30s, 60s)│  │
│  │  Terpisah dari command pool                             │  │
│  └────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

**Kenapa tidak persistent connection untuk semua?**
- 1.000 tenant × 3 router × 5 koneksi = **15.000 koneksi persistent** → terlalu boros
- MikroTik punya limit concurrent API connection (default 20 per router)
- Lazy connect: mayoritas router idle 90% waktu, koneksi hanya dibuat saat ada perintah

**Kenapa tidak connect-on-demand murni?**
- Setiap connect baru butuh ~200-500ms (TCP handshake + API auth)
- Isolir massal 100 pelanggan → 100 × 300ms = 30 detik (terlalu lambat)
- Pool reuse koneksi existing → perintah langsung dikirim tanpa overhead

**Event-Driven Warm-Up (Predictive Pool):**
```
Saat ada event burst masuk (misal isolir massal 50 pelanggan di MK-01):
  │
  ▼
Sebelum proses perintah:
  → Deteksi: "ada 50 perintah untuk MK-01 dalam antrian"
  → Warm-up: buka 5 koneksi sekaligus ke MK-01 (paralel)
  → Proses 50 perintah via 5 koneksi (10 perintah per koneksi)
  → Setelah selesai: koneksi kembali ke idle pool
  → 5 menit tidak ada perintah baru → tutup koneksi
```

**Lifecycle Koneksi:**
```
Koneksi baru dibuat (lazy / warm-up)
  │
  ▼
Masuk ke pool (status: idle)
  │
  ├── Ada perintah → status: busy → kirim perintah → selesai → idle
  │
  ├── Health ping setiap 30 detik
  │     → Gagal? → buang dari pool, buat baru saat dibutuhkan
  │
  ├── Idle > 5 menit → tutup otomatis
  │
  └── Lifetime > 1 jam → tutup, buat baru saat dibutuhkan
      (mencegah stale connection dari memory leak di RouterOS)
```

**Konfigurasi Pool per Router:**
| Setting | Default | Keterangan |
|---|---|---|
| Min Pool Size | 0 | Lazy: tidak ada koneksi saat idle |
| Max Pool Size | 5 | Maksimal koneksi bersamaan per router |
| Idle Timeout | 5 menit | Tutup koneksi yang tidak dipakai |
| Max Lifetime | 1 jam | Tutup & buat baru (anti stale) |
| Health Ping | 30 detik | Cek koneksi masih hidup |
| Warm-Up Threshold | 10 perintah | Jika antrian > 10, warm-up pool ke max |
| Connect Timeout | 5 detik | Timeout saat buat koneksi baru |
| Command Timeout | 30 detik | Timeout per perintah ke router |

- Setiap router memiliki **connection pool** (default 5 koneksi persistent)
- Mengurangi overhead connect/disconnect untuk perintah frequent
- Auto-reconnect jika koneksi terputus (max 3 retry)
- **Rate limiting per router**: max 10 perintah/detik (mencegah router kewalahan)
- Perintah yang melebihi limit → masuk antrian, diproses sesuai urutan
- **Prioritas perintah**:

| Prioritas | Perintah | Keterangan |
|---|---|---|
| 🔴 Tinggi | Isolir, buka isolir, disconnect | Terkait billing, harus cepat |
| 🟡 Sedang | Buat user, update profile, hapus user | Operasional normal |
| 🟢 Rendah | Sync, monitoring, backup | Bisa ditunda |

---

## Halaman Daftar Router (`/mikrotik`)

### Layout Desktop

```
╔══════════════════════════════════════════════════════════════════════════╗
║  Dashboard > MikroTik                                                    ║
║                                                                          ║
║  Router MikroTik                                    [+ Tambah Router]    ║
║                                                                          ║
║  ┌────────────┬────────────┬────────────┬────────────┐                   ║
║  │ 📡 Total   │ 🟢 Online  │ 🔴 Offline │ ⚠️ Pending  │                   ║
║  │ 5 router   │ 4          │ 0          │ 1 sync     │                   ║
║  └────────────┴────────────┴────────────┴────────────┘                   ║
║                                                                          ║
║  ┌──────────┬──────────────┬────────┬────────┬────────┬──────┬────────┐  ║
║  │ Nama     │ IP Address   │ Versi  │ Uptime │ User   │Status│ Aksi   │  ║
║  │          │              │ ROS    │        │ Aktif  │      │        │  ║
║  ├──────────┼──────────────┼────────┼────────┼────────┼──────┼────────┤  ║
║  │ MK-01    │ 192.168.1.1  │ v6.49  │ 45 hari│ 320    │🟢 On │ ⋯      │  ║
║  │ Pusat    │ :8728        │        │        │        │      │        │  ║
║  ├──────────┼──────────────┼────────┼────────┼────────┼──────┼────────┤  ║
║  │ MK-02    │ 192.168.1.2  │ v7.14  │ 12 hari│ 412    │🟢 On │ ⋯      │  ║
║  │ Cabang A │ :8728        │        │        │        │      │        │  ║
║  ├──────────┼──────────────┼────────┼────────┼────────┼──────┼────────┤  ║
║  │ MK-03    │ 10.0.0.1     │ v6.49  │ -      │ 0      │🔴 Off│ ⋯      │  ║
║  │ Cadangan │ :8728        │        │        │        │      │        │  ║
║  └──────────┴──────────────┴────────┴────────┴────────┴──────┴────────┘  ║
╚══════════════════════════════════════════════════════════════════════════╝
```

### Status Router
| Status | Warna | Arti |
|---|---|---|
| 🟢 Online | Green | Terhubung, API bisa diakses |
| 🔴 Offline | Red | Tidak bisa dihubungi |
| 🟡 Maintenance | Amber | Sedang maintenance (diset manual oleh admin) |
| ⚠️ Pending Sync | Orange | Ada perintah yang belum tersinkronisasi |

### Layout Mobile (Card List)

```
┌──────────────────────────────┐
│ 📡 MK-01 Pusat        🟢 On │
│ 192.168.1.1:8728 • v6.49    │
│ Uptime: 45 hari • CPU: 15%  │
│ User: 320 aktif        [⋯]  │
├──────────────────────────────┤
│ 📡 MK-02 Cabang A     🟢 On │
│ 192.168.1.2:8728 • v7.14    │
│ Uptime: 12 hari • CPU: 22%  │
│ User: 412 aktif        [⋯]  │
├──────────────────────────────┤
│ 📡 MK-03 Cadangan    🔴 Off │
│ 10.0.0.1:8728 • v6.49       │
│ Terakhir online: 2 jam lalu │
│ User: 0                [⋯]  │
└──────────────────────────────┘
```

- Teknisi sering akses dari HP saat di lapangan
- Card menampilkan info paling penting: nama, status, IP, uptime, jumlah user
- Tap card → buka detail router


---

## Form Tambah Router (`/mikrotik/new`)

```
╔══════════════════════════════════════════════════════════════╗
║  Dashboard > MikroTik > Tambah Router                        ║
║                                                              ║
║  ┌─── Informasi Router ──────────────────────────────────┐   ║
║  │  Nama Router *           Lokasi / Keterangan          │   ║
║  │  [MK-01 Pusat________]  [Gedung utama lt.2_______]   │   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║  ┌─── Koneksi ───────────────────────────────────────────┐   ║
║  │  IP Address / Hostname *   Port API *                 │   ║
║  │  [192.168.1.1_________]   [8728___]                   │   ║
║  │                                                       │   ║
║  │  Username *                Password *                 │   ║
║  │  [admin______________]   [••••••••••      👁️]         │   ║
║  │                                                       │   ║
║  │  ☐ Gunakan SSL (port 8729)                            │   ║
║  │                                                       │   ║
║  │  [🔍 Test Koneksi]                                    │   ║
║  │  Status: ⏳ Belum ditest                              │   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║  ┌─── Hasil Auto-Detect (setelah test koneksi) ──────────┐   ║
║  │  RouterOS Version: v6.49.10                           │   ║
║  │  Board: RB750Gr3 (hEX)                                │   ║
║  │  CPU: 2 core, Load: 15%                               │   ║
║  │  RAM: 256 MB (used: 45%)                              │   ║
║  │  Uptime: 45 hari 3 jam                                │   ║
║  │  Identity: MikroTik-Pusat                             │   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║  ┌─── Pengaturan ────────────────────────────────────────┐   ║
║  │  Tipe Layanan *                                       │   ║
║  │  ☑ PPPoE Server                                       │   ║
║  │  ☐ Hotspot Server                                     │   ║
║  │  ☐ DHCP Binding                                       │   ║
║  │  ☐ Static IP                                          │   ║
║  │                                                       │   ║
║  │  Health Check Interval                                │   ║
║  │  [60] detik  (cek koneksi berkala)                    │   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║                              [Batal]  [Simpan Router]        ║
╚══════════════════════════════════════════════════════════════╝
```

### Field Router
| Field | Wajib | Keterangan |
|---|---|---|
| Nama Router | ✅ | Nama identifikasi, unik per tenant |
| Lokasi / Keterangan | ❌ | Deskripsi lokasi fisik router |
| IP Address / Hostname | ✅ | IP publik atau hostname yang bisa diakses dari server ISPBoss |
| Port API | ✅ | Default 8728 (non-SSL) atau 8729 (SSL) |
| Username | ✅ | Username RouterOS API |
| Password | ✅ | Password RouterOS API (disimpan terenkripsi) |
| SSL | ❌ | Default non-SSL. Aktifkan jika router support SSL API |
| Tipe Layanan | ✅ | PPPoE / Hotspot / DHCP Binding / Static (bisa multiple) |
| Health Check Interval | ❌ | Default 60 detik |

### Test Koneksi
Saat klik "Test Koneksi":
1. Coba connect ke RouterOS API
2. Jika berhasil → auto-detect: versi ROS, board, CPU, RAM, uptime, identity
3. Jika gagal → tampilkan error: "Connection refused", "Auth failed", "Timeout"
4. Hasil auto-detect ditampilkan di section "Hasil Auto-Detect"

### Keamanan Koneksi
- ISPBoss server harus bisa mengakses IP router (via VPN, port forwarding, atau IP publik)
- Credential disimpan **terenkripsi** (AES-256) di database
- Rekomendasi: buat user API khusus di MikroTik dengan permission terbatas (bukan admin full)
- Panduan setup user API ditampilkan saat tambah router:
```
Buat user API di MikroTik:
/user add name=ispboss password=xxx group=full
/user set ispboss address=<IP_SERVER_ISPBOSS>
```


---

## Detail Router (`/mikrotik/:id`)

```
╔══════════════════════════════════════════════════════════════════╗
║  Dashboard > MikroTik > MK-01 Pusat                              ║
║                                                                  ║
║  ┌──────────────────────────────────────────────────────────┐    ║
║  │  📡 MK-01 Pusat                          🟢 Online       │    ║
║  │  192.168.1.1:8728 • RouterOS v6.49 • RB750Gr3 (hEX)     │    ║
║  │  Uptime: 45 hari 3 jam • CPU: 15% • RAM: 45%            │    ║
║  │                                                          │    ║
║  │  [Edit]  [Sync Sekarang]  [Reboot]  [⋯ Lainnya]         │    ║
║  └──────────────────────────────────────────────────────────┘    ║
║                                                                  ║
║  ┌─ Tab ────────────────────────────────────────────────────┐    ║
║  │ [PPPoE Users] [Hotspot] [DHCP] [Queue] [Traffic]      │    ║
║  │ [Interfaces] [IP Pool] [Firewall] [Log] [Terminal]   │    ║
║  └──────────────────────────────────────────────────────────┘    ║
╚══════════════════════════════════════════════════════════════════╝
```

### Tab PPPoE Users

```
┌──────────────────────────────────────────────────────────────────┐
│ PPPoE Users — MK-01 Pusat                    [+ Tambah Manual]   │
│                                                                  │
│ 🔍 Cari username...    Filter: [Status ▼] [Profile ▼] [Reset]  │
│                                                                  │
│ ┌──────────────┬──────────┬──────────┬────────┬────────┬──────┐  │
│ │ Username     │ Profile  │ IP       │ Uptime │ Status │ Aksi │  │
│ ├──────────────┼──────────┼──────────┼────────┼────────┼──────┤  │
│ │ ahmad-plg001 │ Pro-50M  │ 10.10.1.5│ 3h 15m │ 🟢 On  │ ⋯   │  │
│ │ budi-plg002  │ Basic-10M│ -        │ -      │ 🔴 Off │ ⋯   │  │
│ │ citra-plg003 │ Pro-50M  │ 10.10.1.8│ 1d 5h  │ 🟢 On  │ ⋯   │  │
│ └──────────────┴──────────┴──────────┴────────┴────────┴──────┘  │
│                                                                  │
│ Total: 320 users • Online: 285 • Offline: 35                    │
└──────────────────────────────────────────────────────────────────┘
```

Aksi per PPPoE user:
| Aksi | Deskripsi | Konfirmasi |
|---|---|---|
| Disconnect | Putuskan koneksi aktif (user bisa reconnect) | Ya |
| Disable | Nonaktifkan user (tidak bisa connect) | Ya |
| Enable | Aktifkan kembali user yang disabled | Tidak |
| Remove | Hapus user dari router | Ya |
| Reset Counter | Reset traffic counter | Tidak |
| Lihat Pelanggan | Navigasi ke detail pelanggan di billing | Tidak |

### Tab Hotspot (jika tipe layanan = Hotspot)

```
┌──────────────────────────────────────────────────────────────────┐
│ Hotspot Users — MK-01 Pusat                                      │
│                                                                  │
│ ┌──────────────┬──────────┬──────────┬────────┬────────┬──────┐  │
│ │ Username     │ Profile  │ IP       │ Uptime │ Bytes  │ Aksi │  │
│ ├──────────────┼──────────┼──────────┼────────┼────────┼──────┤  │
│ │ ISP-AB12CD   │ 1Hari-5M │ 10.0.0.5 │ 5h 30m │ 2.1 GB │ ⋯   │  │
│ │ ISP-EF56GH   │ 3Hari-10M│ 10.0.0.8 │ 1d 2h  │ 8.5 GB │ ⋯   │  │
│ └──────────────┴──────────┴──────────┴────────┴────────┴──────┘  │
│                                                                  │
│ Active: 45 • Expired: 120                                        │
└──────────────────────────────────────────────────────────────────┘
```

### Tab Queue (Bandwidth Management)

```
┌──────────────────────────────────────────────────────────────────┐
│ Simple Queue — MK-01 Pusat                                       │
│                                                                  │
│ ┌──────────────┬──────────┬──────────┬──────────┬──────────────┐ │
│ │ Name         │ Target   │ Max Limit│ Burst    │ Bytes (↓/↑)  │ │
│ ├──────────────┼──────────┼──────────┼──────────┼──────────────┤ │
│ │ ahmad-plg001 │ 10.10.1.5│ 50M/25M  │ 60M/30M  │ 15G / 3.2G  │ │
│ │ budi-plg002  │ 10.10.1.6│ 10M/5M   │ -        │ 5.1G / 800M │ │
│ └──────────────┴──────────┴──────────┴──────────┴──────────────┘ │
│                                                                  │
│ Queue Type: ○ Simple Queue  ○ Queue Tree  (read-only, dari router)│
└──────────────────────────────────────────────────────────────────┘
```

### Tab Traffic (Monitoring Real-time)

```
┌──────────────────────────────────────────────────────────────────┐
│ Traffic Monitor — MK-01 Pusat                                    │
│                                                                  │
│ Interface: [ether1-WAN ▼]    Refresh: [Auto 5s ▼]              │
│                                                                  │
│ Download: 245.8 Mbps  ████████████████████░░░░░  (49% of 500M)  │
│ Upload:    82.3 Mbps  ██████░░░░░░░░░░░░░░░░░░░  (16% of 500M)  │
│                                                                  │
│ Traffic 24 Jam:                                                  │
│ ↓ ▁▂▃▅▆█▇▅▆▇█▆▅▃▂▁▂▃▅▆█▇▅                                    │
│ ↑ ▁▁▂▂▃▃▄▃▃▄▅▄▃▂▁▁▂▂▃▃▄▃▃                                    │
│   00:00    06:00    12:00    18:00    sekarang                   │
│                                                                  │
│ Total Hari Ini: ↓ 1.2 TB  ↑ 350 GB                             │
└──────────────────────────────────────────────────────────────────┘
```

- Data traffic via RouterOS API `/interface/monitor-traffic`
- Refresh otomatis setiap 5 detik (configurable)
- Bisa pilih interface mana yang mau dimonitor
- Grafik 24 jam disimpan di Redis (time-series data)

### Tab Interfaces

```
┌──────────────────────────────────────────────────────────────────┐
│ Interfaces — MK-01 Pusat                                         │
│                                                                  │
│ ┌──────────────┬──────────┬──────────┬──────────┬──────────────┐ │
│ │ Name         │ Type     │ Status   │ TX Rate  │ RX Rate      │ │
│ ├──────────────┼──────────┼──────────┼──────────┼──────────────┤ │
│ │ ether1-WAN   │ Ethernet │ 🟢 Up    │ 82.3 Mbps│ 245.8 Mbps   │ │
│ │ ether2-LAN   │ Ethernet │ 🟢 Up    │ 245.8 Mbps│ 82.3 Mbps   │ │
│ │ pppoe-server │ PPPoE    │ 🟢 Up    │ -        │ -            │ │
│ │ wlan1        │ Wireless │ 🔴 Down  │ -        │ -            │ │
│ └──────────────┴──────────┴──────────┴──────────┴──────────────┘ │
└──────────────────────────────────────────────────────────────────┘
```

### Tab IP Pool

```
┌──────────────────────────────────────────────────────────────────┐
│ IP Pool — MK-01 Pusat                                            │
│                                                                  │
│ ┌──────────────┬──────────────────┬──────────┬──────────────────┐ │
│ │ Name         │ Range            │ Used     │ Available        │ │
│ ├──────────────┼──────────────────┼──────────┼──────────────────┤ │
│ │ pool-pppoe   │ 10.10.1.2-10.254│ 285/253  │ ⚠️ 87% penuh    │ │
│ │ pool-hotspot │ 10.20.1.2-10.254│ 45/253   │ 82% tersedia     │ │
│ └──────────────┴──────────────────┴──────────┴──────────────────┘ │
│                                                                  │
│ ⚠️ Pool "pool-pppoe" hampir penuh! Pertimbangkan expand range.  │
└──────────────────────────────────────────────────────────────────┘
```

- Warning otomatis jika pool > 80% terpakai
- Notifikasi ke admin jika pool > 90%

### Tab Log (Router Log)

```
┌──────────────────────────────────────────────────────────────────┐
│ Router Log — MK-01 Pusat                    [Refresh] [Export]   │
│                                                                  │
│ Filter: [Semua ▼]  🔍 Cari...                                  │
│                                                                  │
│ 14:30:15 system,info  user admin logged in from 10.0.0.100      │
│ 14:28:03 pppoe,info   ahmad-plg001 logged in from <pppoe-1>    │
│ 14:25:11 pppoe,info   budi-plg002 logged out                   │
│ 14:20:00 system,info  ISPBoss sync completed (320 users)        │
│ 13:15:22 pppoe,error  citra-plg003 auth failed                 │
└──────────────────────────────────────────────────────────────────┘
```

### Tab Terminal (Advanced)

```
┌──────────────────────────────────────────────────────────────────┐
│ Terminal — MK-01 Pusat                    ⚠️ Mode Advanced      │
│                                                                  │
│ Hanya untuk Tenant Admin & Teknisi.                              │
│ Perintah dicatat di audit log.                                   │
│                                                                  │
│ [admin@MikroTik-Pusat] > /ppp secret print                     │
│ # NAME          SERVICE  PROFILE    REMOTE-ADDRESS               │
│ 0 ahmad-plg001  pppoe    Pro-50M    10.10.1.5                   │
│ 1 budi-plg002   pppoe    Basic-10M                              │
│ 2 citra-plg003  pppoe    Pro-50M    10.10.1.8                   │
│                                                                  │
│ [admin@MikroTik-Pusat] > _                                      │
└──────────────────────────────────────────────────────────────────┘
```

- **Hanya Tenant Admin & Teknisi** yang bisa akses
- Semua perintah **dicatat di audit log** (siapa, kapan, perintah apa)
- **Blacklist perintah berbahaya**: `/system reset`, `/system shutdown`, `/user remove`, `/file remove`
- Perintah yang di-blacklist → ditolak dengan pesan "Perintah ini tidak diizinkan dari dashboard"


---

## Manajemen PPPoE dari Dashboard

### Alur Pelanggan Baru → Buat User PPPoE

```
Event: customer.activated (dari Billing API)
  │
  ▼
Network Service menerima event
  │
  ├── Cari router yang di-assign ke pelanggan
  ├── Cari profile yang sesuai dengan paket pelanggan
  │
  ▼
Kirim perintah ke RouterOS API:
  v6: /ppp/secret/add
  v7: /ppp/secret/add (sama, tapi beberapa param berbeda)
  │
  ├── name = {username_pppoe}
  ├── password = {password_pppoe}
  ├── service = pppoe
  ├── profile = {nama_profile_dari_paket}
  ├── remote-address = (dari pool, opsional)
  ├── comment = "ISPBoss:{customer_id}:{tenant_id}"
  │
  ▼
Hasil:
  ├── Sukses → Log, update status pelanggan
  └── Gagal → Retry mechanism (lihat dokumen 06), notifikasi admin
```

### Alur Isolir (Billing → MikroTik)

```
Event: customer.isolated (dari Billing API)
  │
  ▼
Network Service:
  │
  ├── PPPoE: /ppp/secret/set {user} disabled=yes
  ├── Disconnect active session: /ppp/active/remove {session}
  ├── Tambah firewall rule redirect ke walled garden:
  │     /ip/firewall/nat/add chain=dstnat
  │       src-address={remote_ip}
  │       dst-port=80,443
  │       action=dst-nat to-addresses={walled_garden_ip}
  │       comment="ISPBoss:isolir:{customer_id}"
  │
  ▼
Hasil → Event: mikrotik.command_result
```

### Alur Buka Isolir (Billing → MikroTik)

```
Event: customer.unblocked (dari Billing API)
  │
  ▼
Network Service:
  │
  ├── PPPoE: /ppp/secret/set {user} disabled=no
  ├── Hapus firewall rule redirect:
  │     /ip/firewall/nat/remove [find comment="ISPBoss:isolir:{customer_id}"]
  ├── Reset quota (jika paket ada quota):
  │     Update queue limit / reset counter
  │
  ▼
Hasil → Event: mikrotik.command_result
```

### Alur Suspend (Hapus User)

```
Event: customer.suspended (dari Billing API)
  │
  ▼
Network Service:
  │
  ├── Disconnect active session
  ├── Hapus user PPPoE: /ppp/secret/remove {user}
  ├── Hapus queue: /queue/simple/remove [find name={user}]
  ├── Hapus firewall rule (jika ada)
  │
  ▼
Hasil → Event: mikrotik.command_result
```

### Alur Upgrade/Downgrade Paket

```
Event: package.changed (dari Billing API)
  │
  ▼
Network Service:
  │
  ├── Update profile: /ppp/secret/set {user} profile={new_profile}
  ├── Update queue (jika simple queue):
  │     /queue/simple/set [find name={user}]
  │       max-limit={new_download}/{new_upload}
  │       burst-limit={burst_download}/{burst_upload}
  │       burst-threshold={threshold}
  │       burst-time={burst_time}
  ├── Disconnect & reconnect (agar profile baru berlaku langsung)
  │
  ▼
Hasil → Event: mikrotik.command_result
```


---

## Profile & Bandwidth Management

### PPPoE Profile
Setiap paket internet di ISPBoss otomatis membuat PPPoE Profile di MikroTik:

```
Paket "Pro 50M" di ISPBoss → Profile "Pro-50M" di MikroTik

/ppp/profile/add
  name=Pro-50M
  local-address=gateway
  remote-address=pool-pppoe
  rate-limit=50M/25M                    ← download/upload
  burst-limit=60M/30M                   ← burst (jika aktif)
  burst-threshold=40M/20M
  burst-time=10s/10s
  only-one=yes                          ← 1 session per user
  address-list=active-pppoe             ← untuk firewall
```

### Bandwidth Management Strategy
| Metode | Kapan Dipakai | Keterangan |
|---|---|---|
| **PPPoE Profile rate-limit** | Default | Paling sederhana, bandwidth per user |
| **Simple Queue** | Jika perlu monitoring per user | Queue otomatis dibuat per user, bisa lihat traffic |
| **Queue Tree + PCQ** | ISP besar, bandwidth sharing | Lebih efisien untuk banyak user, tapi setup lebih kompleks |

- Default: **PPPoE Profile rate-limit** (paling umum di RT/RW Net)
- Admin bisa pilih metode di Settings > MikroTik > Bandwidth Method
- Jika pakai Simple Queue → ISPBoss otomatis buat/update/hapus queue per pelanggan

### Sinkronisasi Profile
Saat admin membuat/edit paket di ISPBoss:
1. Sistem cek apakah profile sudah ada di router
2. Jika belum → buat profile baru
3. Jika sudah → update parameter profile
4. Sinkronisasi ke **semua router** yang terdaftar (async, parallel)

---

## Walled Garden (Halaman Tagihan Isolir)

### Arsitektur Walled Garden

```
Pelanggan diisolir
  │
  ▼
MikroTik redirect HTTP/HTTPS ke walled garden server
  │
  ▼
Walled Garden Page (di-serve oleh ISPBoss atau MikroTik hotspot)
  │
  ├── Tampilkan info tagihan pelanggan
  ├── Tombol bayar (jika payment gateway aktif)
  └── Info kontak admin
```

### Opsi Implementasi Walled Garden
| Opsi | Keterangan | Pro | Kontra |
|---|---|---|---|
| **MikroTik Hotspot Page** | HTML di-upload ke router | Tidak perlu server tambahan | Sulit update, tidak dinamis |
| **External Walled Garden** | Redirect ke URL ISPBoss | Dinamis, data real-time | Perlu whitelist domain di firewall |

**Rekomendasi: External Walled Garden** — redirect ke `walled.{tenant_domain}` atau `app.ispboss.id/walled/{tenant_id}/{customer_id}`

### Firewall Rules untuk Walled Garden

```
# Whitelist domain walled garden & payment gateway
/ip firewall address-list add list=walled-garden-allowed address=app.ispboss.id
/ip firewall address-list add list=walled-garden-allowed address=xendit.co
/ip firewall address-list add list=walled-garden-allowed address=midtrans.com

# Redirect pelanggan isolir ke walled garden
/ip firewall nat add chain=dstnat
  src-address-list=isolated-customers
  dst-address-list=!walled-garden-allowed
  protocol=tcp dst-port=80
  action=dst-nat to-addresses={walled_garden_ip} to-ports=80
  comment="ISPBoss:walled-garden"
```

- ISPBoss otomatis mengelola address-list `isolated-customers` (tambah saat isolir, hapus saat buka)
- Whitelist domain payment gateway agar pelanggan bisa bayar langsung

### Keterbatasan HTTPS Redirect
MikroTik **tidak bisa intercept HTTPS** (port 443) tanpa SSL certificate yang valid. Pelanggan yang buka website HTTPS akan dapat error "connection not secure", bukan redirect ke walled garden.

**Solusi yang diterapkan:**
1. **DNS Redirect** (rekomendasi utama): MikroTik set DNS pelanggan isolir ke DNS server ISPBoss yang resolve semua domain ke IP walled garden
2. **Block all + whitelist**: Block semua traffic kecuali walled garden dan payment gateway. Pelanggan yang buka website apapun akan timeout, lalu buka browser → redirect HTTP ke walled garden
3. **Kombinasi**: DNS redirect + HTTP redirect + block HTTPS (kecuali whitelist)

```
# Opsi 1: DNS Redirect (paling efektif)
/ip firewall nat add chain=dstnat
  src-address-list=isolated-customers
  protocol=udp dst-port=53
  action=dst-nat to-addresses={ispboss_dns_ip}
  comment="ISPBoss:dns-redirect"

# Opsi 2: Block all + whitelist
/ip firewall filter add chain=forward
  src-address-list=isolated-customers
  dst-address-list=!walled-garden-allowed
  action=drop
  comment="ISPBoss:block-isolated"
```

- Admin bisa pilih metode di **Settings > MikroTik > Isolir Method**
- Default: DNS Redirect (paling user-friendly)

---

## Sinkronisasi Database ↔ Router

### Periodic Sync (Background Job)

```
Cron job setiap 15 menit:
  │
  ▼
Untuk setiap router online:
  │
  ├── Ambil semua PPPoE user dari router (/ppp/secret/print)
  ├── Bandingkan dengan data di database ISPBoss
  │
  ├── User ada di router tapi tidak di DB:
  │     → Tandai sebagai "Orphan" (user tidak dikelola ISPBoss)
  │     → Tampilkan di dashboard untuk review admin
  │
  ├── User ada di DB tapi tidak di router:
  │     → Tandai sebagai "Missing" (perlu dibuat ulang)
  │     → Auto-create jika pelanggan status Aktif
  │
  ├── User ada di keduanya tapi data berbeda:
  │     → Tandai sebagai "Out of Sync"
  │     → Auto-fix: update router sesuai data DB (DB = source of truth)
  │
  └── Semua cocok → status "Synced" ✅
```

### Sync Status per Router

```
┌──────────────────────────────────────────────────────────────────┐
│ Sync Status — MK-01 Pusat                                        │
│                                                                  │
│ Terakhir sync: 28 Apr 2026 14:30 (5 menit lalu)                │
│ Status: ✅ Synced (320/320 users cocok)                          │
│                                                                  │
│ ┌──────────────┬──────────┬──────────┬──────────┬──────────────┐ │
│ │ Synced       │ Orphan   │ Missing  │ Out of Sync│ Pending    │ │
│ │ 320          │ 2        │ 0        │ 0          │ 1          │ │
│ └──────────────┴──────────┴──────────┴──────────┴──────────────┘ │
│                                                                  │
│ Orphan Users (tidak dikelola ISPBoss):                           │
│ • test-user-1 (profile: default)                                │
│ • backup-admin (profile: admin)                                  │
│                                                                  │
│ [Sync Sekarang]  [Auto-Fix All]  [Export Laporan Sync]           │
└──────────────────────────────────────────────────────────────────┘
```

### Conflict Resolution
- **Database = Source of Truth** — jika ada perbedaan, data di database yang benar
- Orphan user di router **tidak dihapus otomatis** (bisa jadi user manual admin)
- Admin bisa pilih: "Import ke ISPBoss" atau "Hapus dari Router" untuk orphan
- Semua perubahan sync dicatat di audit log

---

## Health Check & Monitoring

### Health Check per Router
- Cek koneksi ke RouterOS API setiap **60 detik** (configurable)
- Jika gagal 3x berturut-turut → status router → 🔴 Offline
- Kirim event `mikrotik.router_offline` → notifikasi ke Teknisi
- Jika kembali online → kirim event `mikrotik.router_online` → notifikasi ke Teknisi
- Log semua status change

### Deteksi Unexpected Reboot
- Setiap health check, simpan **uptime terakhir** per router
- Jika uptime tiba-tiba jauh lebih kecil dari sebelumnya → **unexpected reboot detected**
- Contoh: uptime sebelumnya 45 hari, sekarang 3 menit → router reboot sendiri
- Kirim notifikasi ke Teknisi: "⚠️ Router MK-01 reboot tidak terduga (uptime reset dari 45 hari ke 3 menit)"
- Log event dengan timestamp untuk analisis pola (power failure, overload, dll)

### Monitoring Metrics (via RouterOS API)
| Metric | API Path | Interval |
|---|---|---|
| CPU Load | `/system/resource` | 30 detik |
| RAM Usage | `/system/resource` | 30 detik |
| Uptime | `/system/resource` | 5 menit |
| Interface Traffic | `/interface/monitor-traffic` | 5 detik (real-time) |
| Active PPPoE Sessions | `/ppp/active/print count-only` | 30 detik |
| Queue Stats | `/queue/simple/print` | 1 menit |
| IP Pool Usage | `/ip/pool/used/print count-only` | 5 menit |

- Data disimpan di Redis (time-series, retention 7 hari)
- Grafik di dashboard: CPU, RAM, traffic, active sessions
- Alert jika: CPU > 90%, RAM > 90%, pool > 90%

---

## Hotspot Integration

### Alur Voucher → Hotspot User

```
Reseller beli voucher (dari dokumen 05)
  │
  ▼
End-user masukkan kode voucher di halaman login hotspot
  │
  ▼
MikroTik hotspot server validasi ke ISPBoss API:
  GET /v1/vouchers/validate?code={kode}&mac={mac_address}
  │
  ▼
ISPBoss API:
  ├── Cek kode valid & status = Terjual
  ├── Aktivasi voucher (status → Aktif)
  ├── Return: profile, bandwidth, durasi, quota
  │
  ▼
MikroTik buat hotspot user:
  /ip/hotspot/user/add
    name={kode_voucher}
    password={kode_voucher}
    profile={hotspot_profile}
    limit-uptime={durasi}
    limit-bytes-total={quota}
    comment="ISPBoss:voucher:{voucher_id}"
  │
  ▼
User terkoneksi → durasi mulai berjalan
```

### Hotspot Login Page (Custom)
- ISPBoss menyediakan **template login page** yang bisa di-upload ke MikroTik
- Branding tenant (logo, nama ISP, warna)
- Input field: kode voucher
- Responsive (mobile-friendly)
- Bisa custom HTML/CSS dari dashboard ISPBoss

---

## Aksi Router

| Aksi | Deskripsi | Konfirmasi | Role |
|---|---|---|---|
| Edit | Edit data koneksi router | Tidak | Admin, Teknisi |
| Sync Sekarang | Trigger sinkronisasi manual | Tidak | Admin, Teknisi |
| Auto-Fix All | Perbaiki semua out-of-sync | Ya | Admin |
| Reboot | Reboot router via API | Ya — ketik nama router | Admin |
| Maintenance Mode | Set status maintenance (skip health check) | Ya | Admin, Teknisi |
| Hapus | Hapus router dari ISPBoss (tidak hapus config di router) | Ya — ketik nama router | Admin |
| Backup Config | Download backup config router (.rsc) | Tidak | Admin, Teknisi |
| Migrasi Pelanggan | Pindahkan semua user ke router lain | Ya — pilih router tujuan | Admin |
| Terminal | Akses terminal RouterOS dari dashboard | — | Admin, Teknisi |

### Bulk Action Router
Jika tenant punya banyak router (5-20+), tersedia aksi massal:

| Aksi | Deskripsi |
|---|---|
| Bulk Sync | Sinkronisasi semua router sekaligus |
| Bulk Backup | Backup config semua router |
| Bulk Firmware Check | Cek versi firmware semua router, tampilkan yang outdated |
| Export Status | Download laporan status semua router (CSV/PDF) |

- Checkbox di tabel daftar router → toolbar bulk action muncul
- Proses async, progress tracking per router

---

## Troubleshooting Guide (In-Dashboard Help)

Panel bantuan yang muncul di setiap halaman MikroTik untuk membantu teknisi:

### Masalah Umum & Solusi

| Masalah | Kemungkinan Penyebab | Solusi |
|---|---|---|
| Router tidak bisa dihubungi | IP salah, port tertutup, router mati, VPN putus | Cek IP & port, cek VPN status, ping router dari server |
| Test koneksi gagal "Auth failed" | Username/password salah, user belum dibuat | Cek credential, buat user API di router |
| Test koneksi gagal "Timeout" | Firewall block, IP tidak reachable | Cek firewall router, pastikan port 8728 terbuka, cek VPN |
| User PPPoE tidak bisa connect | Profile salah, password salah, pool habis | Cek profile, reset password, cek IP pool usage |
| Isolir tidak bekerja | Firewall rule tidak terbuat, DNS redirect gagal | Cek tab Firewall, cek address-list, manual sync |
| Bandwidth tidak sesuai paket | Queue tidak terbuat, profile belum sync | Cek tab Queue, trigger sync manual |
| Sync gagal terus | Router offline, credential berubah, API port tertutup | Cek status router, test koneksi ulang |

- Tampilkan sebagai **collapsible help panel** di sidebar kanan (ikon ❓)
- Setiap halaman menampilkan troubleshooting yang relevan dengan konteks halaman tersebut
- Link ke dokumentasi lengkap di help center

### Migrasi Router

Untuk memindahkan pelanggan dari router lama ke router baru (misal upgrade hardware):

```
╔══════════════════════════════════════════════════════════════╗
║  Migrasi Router                                              ║
║                                                              ║
║  Router Sumber: MK-01 Pusat (320 pelanggan)                  ║
║  Router Tujuan: [Pilih Router ▼]                            ║
║                                                              ║
║  Pelanggan yang akan dipindah:                               ║
║  ☑ Semua pelanggan (320)                                     ║
║  ○ Pilih manual:                                             ║
║    ☑ ahmad-plg001 (Pro 50M)                                  ║
║    ☑ budi-plg002 (Basic 10M)                                 ║
║    ☐ citra-plg003 (Pro 50M)                                  ║
║                                                              ║
║  Preview:                                                    ║
║  • 320 user PPPoE akan dibuat di router tujuan               ║
║  • 320 user PPPoE akan dihapus dari router sumber            ║
║  • Profile akan disinkronisasi ke router tujuan              ║
║  • Pelanggan akan disconnect sementara (~30 detik)           ║
║                                                              ║
║  ⚠️ Proses ini akan memutus koneksi pelanggan sementara.    ║
║  Disarankan dilakukan di luar jam sibuk.                     ║
║                                                              ║
║                    [Batal]  [Mulai Migrasi]                  ║
╚══════════════════════════════════════════════════════════════╝
```

Alur migrasi:
1. Buat semua profile di router tujuan (jika belum ada)
2. Buat user PPPoE di router tujuan (batch, async)
3. Disconnect user di router sumber
4. Hapus user di router sumber
5. Update database: pelanggan → router tujuan
6. Verifikasi: cek semua user bisa connect di router baru
7. Laporan migrasi: X berhasil, Y gagal

- Proses async (background job), progress tracking real-time
- Jika ada yang gagal → rollback per user (buat ulang di router sumber)
- Notifikasi ke admin setelah selesai

### Scheduled Backup Config

Selain backup manual, ada backup otomatis berkala:

| Setting | Default | Keterangan |
|---|---|---|
| Auto-Backup | Aktif | Backup config otomatis |
| Jadwal | Mingguan (Minggu 02:00) | Configurable per router |
| Retensi | 10 versi terakhir | Backup lama otomatis dihapus |
| Format | .rsc (RouterOS script) | Bisa di-restore via terminal |

```
┌──────────────────────────────────────────────────────────────────┐
│ Backup History — MK-01 Pusat                                     │
│                                                                  │
│ ┌──────────────────┬──────────┬──────────┬──────────────────────┐ │
│ │ Tanggal          │ Ukuran   │ Tipe     │ Aksi                 │ │
│ ├──────────────────┼──────────┼──────────┼──────────────────────┤ │
│ │ 27 Apr 2026 02:00│ 45 KB    │ Auto     │ [Download] [Restore] │ │
│ │ 20 Apr 2026 02:00│ 44 KB    │ Auto     │ [Download] [Restore] │ │
│ │ 15 Apr 2026 14:30│ 44 KB    │ Manual   │ [Download] [Restore] │ │
│ └──────────────────┴──────────┴──────────┴──────────────────────┘ │
│                                                                  │
│ [Backup Sekarang]                                                │
└──────────────────────────────────────────────────────────────────┘
```

- Backup disimpan di storage ISPBoss (bukan di router)
- Restore: upload .rsc ke router via API (konfirmasi wajib, hanya Admin)
- Notifikasi ke admin jika backup gagal

### Firmware Version Tracking

```
┌──────────────────────────────────────────────────────────────────┐
│ Firmware — MK-01 Pusat                                           │
│                                                                  │
│ Versi saat ini: RouterOS v6.49.10                                │
│ Versi terbaru:  RouterOS v6.49.17                                │
│ Status: ⚠️ Update tersedia (7 versi di belakang)                │
│                                                                  │
│ Board: RB750Gr3 (hEX)                                           │
│ Architecture: mmips                                              │
│ Firmware: 6.49.10                                                │
│ Factory Firmware: 6.44.6                                         │
│                                                                  │
│ ⚠️ Firmware sudah outdated > 6 bulan.                           │
│ Disarankan update ke versi terbaru.                              │
│                                                                  │
│ Catatan: ISPBoss TIDAK melakukan auto-update firmware.           │
│ Update firmware harus dilakukan manual oleh teknisi.             │
└──────────────────────────────────────────────────────────────────┘
```

- Cek versi terbaru via MikroTik upgrade server (periodik, 1x sehari)
- **Tidak ada auto-update** (terlalu berisiko, bisa brick router)
- Warning di daftar router jika firmware outdated > 6 bulan
- Notifikasi ke teknisi jika ada critical security update

---

## Keamanan

### Keamanan Koneksi
- Credential router disimpan **terenkripsi AES-256** di database
- Koneksi ke router via **VPN tunnel** (rekomendasi) atau IP publik dengan firewall
- SSL API (port 8729) direkomendasikan jika router support
- User API di MikroTik sebaiknya **bukan admin** — buat user khusus dengan permission terbatas

### Keamanan Development
- Default `NETWORK_MODE=mock` — tidak pernah konek ke router production saat development
- Mock adapter mengembalikan response simulasi
- Integration test menggunakan **MikroTik CHR** (Cloud Hosted Router, VM gratis dari MikroTik)
- Credential di environment variable, **TIDAK** di kode/Git

### Audit Trail
Semua perintah ke router dicatat:
```
┌──────────────────┬──────────────────────────────────┬──────────────┐
│ Waktu            │ Perintah                         │ Oleh         │
├──────────────────┼──────────────────────────────────┼──────────────┤
│ 28/04/26 14:30   │ /ppp/secret/add ahmad-plg001     │ System (event)│
│ 28/04/26 14:25   │ /ppp/secret/set budi disabled=yes│ System (isolir)│
│ 28/04/26 14:20   │ /system/resource/print           │ Teknisi Andi │
│ 28/04/26 14:15   │ /ppp/active/remove session-5     │ Admin Budi   │
└──────────────────┴──────────────────────────────────┴──────────────┘
```

---

## VPN Tunnel Management (PPP / WireGuard)

### Kenapa ISPBoss Perlu Menyediakan VPN?

ISPBoss adalah SaaS yang di-host di cloud, sementara router MikroTik dan OLT ada di lokasi tenant (on-premise). Masalah utama:

| Masalah | Dampak | Solusi VPN |
|---|---|---|
| Tenant tidak punya IP publik | ISPBoss tidak bisa akses router | VPN tunnel dari router ke server ISPBoss |
| IP publik dinamis (berubah-ubah) | Koneksi putus setiap IP berubah | VPN tunnel tetap stabil meskipun IP berubah |
| Latency tinggi ke API | Perintah isolir/buka isolir lambat | VPN mengurangi hop, koneksi lebih stabil |
| Keamanan koneksi | RouterOS API tanpa enkripsi (port 8728) | VPN mengenkripsi semua traffic |
| Multi-site ISP | Banyak lokasi router, sulit manage IP | 1 VPN tunnel per site, semua terhubung |

### Arsitektur VPN

```
┌─────────────────────────────────────────────────────────────┐
│  ISPBoss Cloud (VPN Server)                                  │
│  vpn.ispboss.id                                              │
│                                                              │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │
│  │ VPN Hub     │  │ Billing API │  │ Network Svc │         │
│  │ (WireGuard/ │──│             │──│             │         │
│  │  L2TP/PPTP) │  │             │  │             │         │
│  └──────┬──────┘  └─────────────┘  └─────────────┘         │
│         │                                                    │
└─────────┼────────────────────────────────────────────────────┘
          │ VPN Tunnel (encrypted)
          │
    ┌─────┼──────────────────────────────────┐
    │     │                                  │
    │  ┌──▼──────┐  ┌──────────┐  ┌────────┐│
    │  │ MikroTik│  │ MikroTik │  │  OLT   ││
    │  │ Router 1│  │ Router 2 │  │        ││
    │  └─────────┘  └──────────┘  └────────┘│
    │                                        │
    │  Lokasi Tenant (on-premise)            │
    └────────────────────────────────────────┘
```

### Protokol VPN yang Didukung

| Protokol | Keterangan | Rekomendasi |
|---|---|---|
| **WireGuard** | Modern, cepat, ringan, enkripsi kuat | 🥇 Utama (RouterOS v7+) |
| **L2TP/IPSec** | Stabil, didukung semua versi RouterOS | 🥈 Fallback (v6 & v7) |
| **PPTP** | Lama, kurang aman, tapi paling mudah setup | 🥉 Legacy (jika tidak ada opsi lain) |
| **SSTP** | Bisa lewat firewall/NAT ketat | Alternatif untuk jaringan restricted |
| **OpenVPN** | Open source, fleksibel | Alternatif jika WireGuard tidak tersedia |

- Default rekomendasi: **WireGuard** untuk RouterOS v7, **L2TP/IPSec** untuk RouterOS v6
- Tenant bisa pilih protokol sesuai kebutuhan

### Halaman VPN Management (`/mikrotik/vpn`)

```
╔══════════════════════════════════════════════════════════════════════════╗
║  Dashboard > MikroTik > VPN Tunnel                                       ║
║                                                                          ║
║  VPN Tunnel                                       [+ Setup VPN Baru]     ║
║                                                                          ║
║  ┌────────────┬────────────┬────────────┬────────────┐                   ║
║  │ 🔗 Total   │ 🟢 Connected│ 🔴 Disconn │ ⏳ Pending  │                   ║
║  │ 5 tunnel   │ 4          │ 0          │ 1          │                   ║
║  └────────────┴────────────┴────────────┴────────────┘                   ║
║                                                                          ║
║  ┌──────────┬──────────────┬──────────┬────────┬────────┬──────┬──────┐  ║
║  │ Nama     │ Router       │ Protokol │ IP VPN │ Uptime │Latency│Aksi │  ║
║  ├──────────┼──────────────┼──────────┼────────┼────────┼──────┼──────┤  ║
║  │ VPN-01   │ MK-01 Pusat  │ WireGuard│ 10.99. │ 45 hari│ 12ms │ ⋯   │  ║
║  │          │              │          │ 0.2    │        │      │      │  ║
║  ├──────────┼──────────────┼──────────┼────────┼────────┼──────┼──────┤  ║
║  │ VPN-02   │ MK-02 Cabang │ L2TP     │ 10.99. │ 12 hari│ 25ms │ ⋯   │  ║
║  │          │              │          │ 0.3    │        │      │      │  ║
║  ├──────────┼──────────────┼──────────┼────────┼────────┼──────┼──────┤  ║
║  │ VPN-03   │ OLT-ZTE      │ L2TP     │ 10.99. │ 30 hari│ 18ms │ ⋯   │  ║
║  │          │              │          │ 0.4    │        │      │      │  ║
║  └──────────┴──────────────┴──────────┴────────┴────────┴──────┴──────┘  ║
╚══════════════════════════════════════════════════════════════════════════╝
```

### Setup VPN Wizard

```
╔══════════════════════════════════════════════════════════════╗
║  Setup VPN Tunnel — Langkah 1 dari 3                         ║
║                                                              ║
║  Pilih Protokol VPN *                                        ║
║  ● WireGuard (rekomendasi untuk RouterOS v7+)                ║
║  ○ L2TP/IPSec (untuk RouterOS v6 & v7)                      ║
║  ○ PPTP (legacy, kurang aman)                                ║
║  ○ SSTP (untuk jaringan restricted)                          ║
║                                                              ║
║  Router yang akan dihubungkan *                              ║
║  [Pilih Router ▼] atau [Belum terdaftar — setup manual]     ║
║                                                              ║
║                                    [Batal]  [Lanjut →]       ║
╚══════════════════════════════════════════════════════════════╝
```

```
╔══════════════════════════════════════════════════════════════╗
║  Setup VPN Tunnel — Langkah 2 dari 3                         ║
║                                                              ║
║  Konfigurasi VPN sudah digenerate:                           ║
║                                                              ║
║  ┌─── Server ISPBoss (sudah dikonfigurasi otomatis) ─────┐   ║
║  │  Endpoint: vpn.ispboss.id:51820                       │   ║
║  │  Public Key: aBcDeFgHiJkLmNoPqRsTuVwXyZ...           │   ║
║  │  IP VPN Server: 10.99.0.1/24                          │   ║
║  │  IP VPN Client: 10.99.0.2/32                          │   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║  ┌─── Konfigurasi untuk Router MikroTik ─────────────────┐   ║
║  │  Salin script ini dan jalankan di terminal MikroTik:   │   ║
║  │                                                       │   ║
║  │  /interface/wireguard/add                             │   ║
║  │    name=ispboss-vpn                                   │   ║
║  │    listen-port=51820                                  │   ║
║  │    private-key="xYzAbCdEfGhIjKlMnOpQrStUvWxYz..."   │   ║
║  │                                                       │   ║
║  │  /interface/wireguard/peers/add                       │   ║
║  │    interface=ispboss-vpn                              │   ║
║  │    public-key="aBcDeFgHiJkLmNoPqRsTuVwXyZ..."       │   ║
║  │    endpoint-address=vpn.ispboss.id                    │   ║
║  │    endpoint-port=51820                                │   ║
║  │    allowed-address=10.99.0.0/24                       │   ║
║  │    persistent-keepalive=25                            │   ║
║  │                                                       │   ║
║  │  /ip/address/add                                      │   ║
║  │    address=10.99.0.2/24                               │   ║
║  │    interface=ispboss-vpn                              │   ║
║  │                                                       │   ║
║  │  [📋 Salin Script]  [📥 Download .rsc]                │   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║  Atau jika router sudah terdaftar & online:                  ║
║  [🔧 Auto-Configure via API] (otomatis setup di router)     ║
║                                                              ║
║                              [← Kembali]  [Lanjut →]        ║
╚══════════════════════════════════════════════════════════════╝
```

```
╔══════════════════════════════════════════════════════════════╗
║  Setup VPN Tunnel — Langkah 3 dari 3                         ║
║                                                              ║
║  Verifikasi Koneksi                                          ║
║                                                              ║
║  [🔍 Test Koneksi VPN]                                      ║
║                                                              ║
║  Status: 🟢 Terhubung!                                      ║
║  Latency: 12ms                                               ║
║  IP VPN Router: 10.99.0.2                                    ║
║  Handshake: 5 detik lalu                                     ║
║                                                              ║
║  ✅ VPN tunnel berhasil dibuat.                              ║
║  Router MK-01 Pusat sekarang bisa diakses via 10.99.0.2     ║
║                                                              ║
║  ☑ Gunakan IP VPN sebagai koneksi utama ke router ini        ║
║    (update IP router dari 192.168.1.1 ke 10.99.0.2)         ║
║                                                              ║
║                                          [Selesai]           ║
╚══════════════════════════════════════════════════════════════╝
```

### Fitur VPN
| Fitur | Keterangan |
|---|---|
| Auto-generate config | ISPBoss generate key pair dan config otomatis |
| Auto-configure | Jika router sudah online, setup VPN via API (tanpa manual) |
| Script download | Download .rsc script untuk setup manual di router |
| Health monitoring | Cek status VPN setiap 30 detik (ping, handshake) |
| Auto-reconnect | Jika VPN putus, MikroTik auto-reconnect (persistent-keepalive) |
| Latency monitoring | Tampilkan latency real-time per tunnel |
| Multi-tunnel | 1 tenant bisa punya banyak tunnel (1 per site/router) |

### IP Addressing VPN
- Subnet VPN per tenant: `10.99.{tenant_seq}.0/24`
- Server ISPBoss: `10.99.X.1`
- Client (router/OLT): `10.99.X.2`, `10.99.X.3`, dst.
- Max 253 perangkat per tenant (cukup untuk mayoritas ISP)
- Jika butuh lebih → expand ke subnet /23 atau /22

### Keamanan VPN
- **Key pair** di-generate per tunnel, private key tidak pernah dikirim via internet
- **Pre-shared key** (PSK) opsional untuk lapisan keamanan tambahan
- **Firewall di VPN server**: hanya izinkan traffic ke port RouterOS API (8728/8729) dan SNMP (161)
- **Rate limiting**: mencegah abuse tunnel
- **Audit log**: semua koneksi VPN dicatat

### Bandwidth Monitoring & Cap per Tunnel
- Tampilkan **bandwidth usage real-time** per tunnel (TX/RX)
- **Bandwidth cap per tenant** untuk mencegah 1 tenant monopoli VPN server:

| Tier Tenant | Bandwidth Cap VPN |
|---|---|
| Starter | 10 Mbps |
| Growth | 50 Mbps |
| Pro | 200 Mbps |
| Enterprise | Custom / Unlimited |

- Traffic yang melebihi cap → throttle (bukan drop)
- Dashboard: grafik bandwidth per tunnel (24 jam)
- Alert ke admin ISPBoss jika total bandwidth VPN server > 80%

### VPN Server High Availability
VPN server ISPBoss harus highly available karena semua tenant bergantung padanya:

```
Tenant Router
  │
  ├── Primary:   vpn1.ispboss.id (Jakarta)
  ├── Secondary: vpn2.ispboss.id (Surabaya)
  │
  └── MikroTik auto-failover:
        Jika vpn1 tidak bisa dihubungi 30 detik
        → otomatis switch ke vpn2
        → notifikasi ke admin ISPBoss
```

- **2 VPN endpoint** di lokasi berbeda (geo-redundancy)
- MikroTik dikonfigurasi dengan **failover peer** (primary + secondary)
- Auto-failover di sisi client (MikroTik script atau WireGuard multi-peer)
- Notifikasi ke semua tenant jika VPN server maintenance terjadwal
- **SLA VPN**: 99.9% uptime

### Manfaat untuk Tenant
```
Tanpa VPN:
  Tenant harus punya IP publik per router
  Atau setup port forwarding manual
  Koneksi tidak terenkripsi (RouterOS API plain text)
  IP dinamis → koneksi sering putus

Dengan VPN ISPBoss:
  ✅ Tidak perlu IP publik
  ✅ Tidak perlu port forwarding
  ✅ Koneksi terenkripsi end-to-end
  ✅ IP VPN stabil (tidak berubah)
  ✅ Latency lebih rendah (direct tunnel)
  ✅ Semua perangkat (MikroTik + OLT) bisa diakses via 1 tunnel
  ✅ Setup 5 menit via wizard
```

---

## DHCP Server Management

Beberapa ISP menggunakan DHCP untuk distribusi IP ke pelanggan (terutama untuk jaringan hotspot, static-binding, atau pelanggan tanpa PPPoE).

### Tab DHCP (Detail Router)

```
┌──────────────────────────────────────────────────────────────────┐
│ DHCP Server — MK-01 Pusat                                        │
│                                                                  │
│ [Servers]  [Leases]  [Static Bindings]  [Networks]              │
│                                                                  │
│ DHCP Servers:                                                    │
│ ┌──────────────┬──────────┬──────────┬──────────┬──────────────┐ │
│ │ Name         │ Interface│ Pool     │ Leases   │ Status       │ │
│ ├──────────────┼──────────┼──────────┼──────────┼──────────────┤ │
│ │ dhcp-lan     │ ether2   │ pool-lan │ 45/253   │ 🟢 Enabled   │ │
│ │ dhcp-hotspot │ wlan1    │ pool-hs  │ 120/253  │ 🟢 Enabled   │ │
│ └──────────────┴──────────┴──────────┴──────────┴──────────────┘ │
│                                                                  │
│ Active Leases:                                                   │
│ ┌──────────────┬──────────────┬──────────────┬────────┬───────┐  │
│ │ IP Address   │ MAC Address  │ Hostname     │ Status │ Aksi  │  │
│ ├──────────────┼──────────────┼──────────────┼────────┼───────┤  │
│ │ 10.20.1.5    │ AA:BB:CC:11  │ iPhone-Ahmad │ 🟢Bound│ ⋯     │  │
│ │ 10.20.1.8    │ DD:EE:FF:22  │ Laptop-Budi  │ 🟢Bound│ ⋯     │  │
│ │ 10.20.1.15   │ 11:22:33:44  │ Android-Citra│ ⏳Wait │ ⋯     │  │
│ └──────────────┴──────────────┴──────────────┴────────┴───────┘  │
│                                                                  │
│ Total Leases: 45 aktif / 253 tersedia                            │
│ ⚠️ Pool "pool-hs" 47% terpakai                                  │
└──────────────────────────────────────────────────────────────────┘
```

### DHCP Static Binding (MAC-IP Binding)

Untuk pelanggan yang butuh IP tetap via DHCP (bukan PPPoE, bukan full static):

```
┌──────────────────────────────────────────────────────────────────┐
│ Static Bindings — MK-01 Pusat                [+ Tambah Binding]  │
│                                                                  │
│ ┌──────────────┬──────────────┬──────────────┬────────┬───────┐  │
│ │ IP Address   │ MAC Address  │ Pelanggan    │ Server │ Aksi  │  │
│ ├──────────────┼──────────────┼──────────────┼────────┼───────┤  │
│ │ 10.20.1.100  │ AA:BB:CC:11  │ Ahmad (PLG-1)│ dhcp-lan│ ⋯    │  │
│ │ 10.20.1.101  │ DD:EE:FF:22  │ Budi (PLG-2) │ dhcp-lan│ ⋯    │  │
│ └──────────────┴──────────────┴──────────────┴────────┴───────┘  │
└──────────────────────────────────────────────────────────────────┘
```

- Static binding = MAC address pelanggan selalu dapat IP yang sama
- Berguna untuk pelanggan yang butuh IP tetap tapi tidak pakai PPPoE
- ISPBoss bisa **auto-create static binding** saat pelanggan diaktivasi (jika MAC address diisi)

### Alur Pelanggan DHCP (Non-PPPoE)

```
Pelanggan baru (metode: DHCP Binding)
  │
  ▼
Admin input MAC address pelanggan di form pelanggan
  │
  ▼
Network Service:
  ├── Buat DHCP static lease:
  │     /ip/dhcp-server/lease/add
  │       address={ip_dari_pool}
  │       mac-address={mac_pelanggan}
  │       server={dhcp_server}
  │       comment="ISPBoss:{customer_id}:{tenant_id}"
  │
  ├── Buat simple queue (bandwidth limit):
  │     /queue/simple/add
  │       name={customer_id}
  │       target={ip_address}
  │       max-limit={download}/{upload}
  │
  ├── Tambah address-list:
  │     /ip/firewall/address-list/add
  │       list=active-dhcp address={ip_address}
  │
  ▼
Isolir pelanggan DHCP:
  ├── Disable lease: /ip/dhcp-server/lease/set {lease} disabled=yes
  ├── Pindah ke address-list isolated-customers
  ├── Redirect ke walled garden (sama seperti PPPoE/Static)
```

### DHCP Network Settings (Read-Only)

```
┌──────────────────────────────────────────────────────────────────┐
│ DHCP Networks — MK-01 Pusat                                      │
│                                                                  │
│ ┌──────────────┬──────────────┬──────────────┬──────────────────┐ │
│ │ Address      │ Gateway      │ DNS          │ Domain           │ │
│ ├──────────────┼──────────────┼──────────────┼──────────────────┤ │
│ │ 10.20.1.0/24 │ 10.20.1.1    │ 8.8.8.8,    │ isp.local        │ │
│ │              │              │ 8.8.4.4     │                  │ │
│ └──────────────┴──────────────┴──────────────┴──────────────────┘ │
│                                                                  │
│ ℹ️ Network settings hanya bisa diedit via Terminal atau Winbox.  │
└──────────────────────────────────────────────────────────────────┘
```

- DHCP network settings ditampilkan **read-only** di dashboard
- Untuk edit → gunakan tab Terminal atau Winbox langsung
- ISPBoss tidak mengelola DHCP server config (hanya lease/binding)

---

## Static IP Management

Untuk pelanggan dengan koneksi Static IP (bukan PPPoE/Hotspot):

### Alur Pelanggan Static IP

```
Pelanggan baru (metode: Static IP)
  │
  ▼
Admin assign IP dari pool static:
  ├── IP Address: 10.30.1.5/24
  ├── Gateway: 10.30.1.1
  ├── DNS: 8.8.8.8, 8.8.4.4
  │
  ▼
Network Service:
  ├── Tambah ARP entry: /ip/arp/add address=10.30.1.5 mac-address={mac}
  ├── Tambah address-list: /ip/firewall/address-list/add list=active-static address=10.30.1.5
  ├── Buat simple queue (bandwidth limit):
  │     /queue/simple/add name={customer_id} target=10.30.1.5
  │       max-limit={download}/{upload}
  │
  ▼
Isolir pelanggan static:
  ├── Hapus dari address-list active-static
  ├── Tambah ke address-list isolated-customers
  ├── Firewall rule block/redirect (sama seperti PPPoE)
```

### IP Pool Static
- Pool terpisah dari PPPoE (misal: `pool-static: 10.30.1.2-10.30.1.254`)
- Admin assign IP manual per pelanggan (bukan auto dari pool)
- IP ditampilkan di detail pelanggan dan bisa diedit

---

## Tab Firewall (Detail Router)

Tab Firewall menampilkan firewall rules yang dikelola ISPBoss di router:

```
┌──────────────────────────────────────────────────────────────────┐
│ Firewall Rules — MK-01 Pusat                                     │
│                                                                  │
│ [ISPBoss Managed]  [All Rules]     ← toggle view                │
│                                                                  │
│ ISPBoss Managed Rules:                                           │
│ ┌────┬──────────────────────────────────┬──────────┬───────────┐ │
│ │ #  │ Comment                          │ Chain    │ Action    │ │
│ ├────┼──────────────────────────────────┼──────────┼───────────┤ │
│ │ 1  │ ISPBoss:walled-garden            │ dstnat   │ dst-nat   │ │
│ │ 2  │ ISPBoss:dns-redirect             │ dstnat   │ dst-nat   │ │
│ │ 3  │ ISPBoss:isolir:PLG-002           │ dstnat   │ dst-nat   │ │
│ │ 4  │ ISPBoss:isolir:PLG-008           │ dstnat   │ dst-nat   │ │
│ │ 5  │ ISPBoss:block-isolated           │ forward  │ drop      │ │
│ └────┴──────────────────────────────────┴──────────┴───────────┘ │
│                                                                  │
│ ⚠️ Rules yang dikelola ISPBoss tidak boleh diedit manual        │
│ di router. Perubahan akan di-revert saat sync berikutnya.       │
│                                                                  │
│ Address Lists:                                                   │
│ ┌──────────────────────┬──────────┬──────────────────────────┐   │
│ │ List Name            │ Count    │ Keterangan               │   │
│ ├──────────────────────┼──────────┼──────────────────────────┤   │
│ │ isolated-customers   │ 12       │ Pelanggan yang diisolir  │   │
│ │ active-pppoe         │ 285      │ PPPoE user aktif         │   │
│ │ active-static        │ 15       │ Static IP user aktif     │   │
│ │ walled-garden-allowed│ 5        │ Domain whitelist isolir  │   │
│ └──────────────────────┴──────────┴──────────────────────────┘   │
└──────────────────────────────────────────────────────────────────┘
```

- **View "ISPBoss Managed"**: hanya tampilkan rules yang dibuat oleh ISPBoss (dikenali dari comment prefix `ISPBoss:`)
- **View "All Rules"**: tampilkan semua rules di router (read-only)
- Admin **tidak bisa edit firewall dari dashboard** — hanya bisa lihat (read-only)
- Untuk edit firewall manual → gunakan tab Terminal atau Winbox langsung

---

## Initial Sync Wizard (Aktivasi Modul MikroTik)

Jika tenant mengaktifkan modul MikroTik setelah sudah punya pelanggan existing:

```
╔══════════════════════════════════════════════════════════════╗
║  Setup MikroTik — Langkah 1 dari 3                           ║
║                                                              ║
║  Selamat datang di modul MikroTik! 🎉                       ║
║                                                              ║
║  Anda sudah punya 847 pelanggan. Mari hubungkan              ║
║  pelanggan Anda ke router MikroTik.                          ║
║                                                              ║
║  Langkah 1: Tambah Router                                    ║
║  [+ Tambah Router Pertama]                                   ║
║                                                              ║
║  Langkah 2: Mapping Pelanggan ke Router                      ║
║  (setelah router ditambahkan)                                ║
║                                                              ║
║  Langkah 3: Sinkronisasi                                     ║
║  (setelah mapping selesai)                                   ║
║                                                              ║
║                                          [Lewati untuk nanti]║
╚══════════════════════════════════════════════════════════════╝
```

### Langkah 2: Mapping Pelanggan ke Router

```
╔══════════════════════════════════════════════════════════════╗
║  Setup MikroTik — Langkah 2: Mapping Pelanggan               ║
║                                                              ║
║  Router: MK-01 Pusat (192.168.1.1)                           ║
║                                                              ║
║  Opsi Mapping:                                               ║
║                                                              ║
║  ○ Auto-detect: Cocokkan username PPPoE di router            ║
║    dengan username pelanggan di ISPBoss                      ║
║                                                              ║
║  ○ Import dari router: Ambil semua user PPPoE dari           ║
║    router dan buat pelanggan baru di ISPBoss                 ║
║                                                              ║
║  ○ Manual: Assign pelanggan ke router satu per satu          ║
║                                                              ║
║  [Mulai Mapping]                                             ║
╚══════════════════════════════════════════════════════════════╝
```

### Hasil Auto-Detect Mapping

```
┌──────────────────────────────────────────────────────────────────┐
│ Hasil Mapping — MK-01 Pusat                                      │
│                                                                  │
│ ┌──────────────┬──────────────┬──────────┬──────────────────────┐ │
│ │ Status       │ Jumlah       │ Detail   │ Aksi                 │ │
│ ├──────────────┼──────────────┼──────────┼──────────────────────┤ │
│ │ ✅ Matched   │ 310          │ Username │ Otomatis terhubung   │ │
│ │              │              │ cocok    │                      │ │
│ │ ⚠️ Unmatched │ 10           │ Username │ [Mapping Manual]     │ │
│ │ (ISPBoss)    │              │ tidak ada│                      │ │
│ │              │              │ di router│                      │ │
│ │ ❓ Orphan    │ 5            │ Ada di   │ [Import] [Abaikan]   │ │
│ │ (Router)     │              │ router,  │                      │ │
│ │              │              │ tidak di │                      │ │
│ │              │              │ ISPBoss  │                      │ │
│ └──────────────┴──────────────┴──────────┴──────────────────────┘ │
│                                                                  │
│ [Selesaikan Mapping]  [Ulangi]                                   │
└──────────────────────────────────────────────────────────────────┘
```

- Wizard hanya muncul **sekali** saat pertama kali modul MikroTik diaktifkan
- Bisa di-skip dan dilakukan nanti dari menu Settings
- Setelah mapping selesai → trigger full sync

---

## Integrasi dengan Modul Lain

| Modul | Integrasi |
|---|---|
| **Pelanggan (04)** | Tab Network di detail pelanggan, assign router |
| **Paket (05)** | Profile MikroTik dari paket, burst settings |
| **Billing (06)** | Isolir/buka isolir otomatis, walled garden, retry mechanism |
| **Notifikasi (07)** | Alert router offline, sync gagal ke teknisi |
| **OLT (09)** | Pelanggan FTTH: MikroTik untuk bandwidth, OLT untuk layer fisik. VPN tunnel juga untuk akses OLT |
| **FTTH Mapping (10)** | Visualisasi router di peta jaringan |
| **Laporan (11)** | Traffic per router, uptime, active sessions |
| **Settings (12)** | Konfigurasi bandwidth method, health check interval |

---

## Keputusan

| Keputusan | Detail |
|---|---|
| Library | `go-routeros` (Golang) untuk komunikasi RouterOS API |
| Versi RouterOS | ✅ Support v6 dan v7, kode terpisah (adapter pattern) |
| Auto-detect | ✅ Versi, board, CPU, RAM, uptime saat tambah router |
| Connection pool | ✅ Lazy Connect + Event-Driven Warm-Up. Min 0, max 5 per router. Idle 5 menit, lifetime 1 jam |
| Monitor connection | ✅ 1 koneksi persistent per router untuk real-time monitoring (terpisah dari pool) |
| Warm-up threshold | Antrian > 10 perintah → warm-up pool ke max size (prediktif) |
| Rate limiting perintah | ✅ Max 10/detik per router, prioritas (tinggi/sedang/rendah) |
| Koneksi | Via RouterOS API (port 8728/8729), bukan SSH |
| Credential | Terenkripsi AES-256, bukan plain text |
| Async command | ✅ Semua perintah via Redis queue, bukan langsung dari HTTP request |
| PPPoE management | ✅ CRUD user, enable/disable, disconnect dari dashboard |
| Hotspot management | ✅ Validasi voucher via API, custom login page |
| Bandwidth method | 3 opsi: Profile rate-limit (default), Simple Queue, Queue Tree + PCQ |
| Profile sync | ✅ Otomatis sinkronisasi profile saat paket dibuat/diedit |
| Isolir | Disable user + firewall redirect ke walled garden |
| Buka isolir | Enable user + hapus firewall redirect |
| Suspend | Hapus user dari router (bukan disable) |
| Walled garden | ✅ External (redirect ke URL ISPBoss), dinamis, tombol bayar langsung |
| HTTPS redirect | ✅ 3 solusi: DNS redirect (default), block+whitelist, kombinasi. Configurable per tenant |
| Isolir method | Configurable: DNS Redirect (default), HTTP Redirect, Block All + Whitelist |
| Sinkronisasi | ✅ Periodic sync 15 menit, detect orphan/missing/out-of-sync |
| Conflict resolution | Database = source of truth. Orphan tidak dihapus otomatis |
| Health check | ✅ Setiap 60 detik, 3x gagal → offline, notifikasi ke teknisi |
| Monitoring | ✅ CPU, RAM, traffic, sessions, pool usage. Data di Redis 7 hari |
| IP Pool warning | ✅ Alert jika pool > 80%, notifikasi jika > 90% |
| Terminal | ✅ Akses terminal dari dashboard, blacklist perintah berbahaya, audit log |
| Backup config | ✅ Manual + auto backup mingguan, retensi 10 versi, restore via API |
| Firmware tracking | ✅ Cek versi terbaru harian, warning jika outdated > 6 bulan, tidak auto-update |
| Migrasi router | ✅ Pindahkan pelanggan antar router, batch async, rollback per user jika gagal |
| Static IP | ✅ Assign IP manual, pool terpisah, ARP entry + address-list + queue |
| DHCP Server | ✅ Tab DHCP di detail router: servers, leases, static bindings, networks |
| DHCP Binding | ✅ Auto-create static lease saat pelanggan diaktivasi (MAC-IP binding) |
| Tab Firewall | ✅ Read-only, tampilkan ISPBoss managed rules dan address-list |
| Multi-router per pelanggan | ❌ Fase awal: 1 pelanggan = 1 router. Multi-router = fitur lanjutan |
| Initial sync wizard | ✅ Wizard saat pertama kali aktifkan modul, auto-detect mapping pelanggan ke router |
| Reboot | ✅ Via API, konfirmasi ketik nama router |
| Development | Mock adapter default, MikroTik CHR untuk integration test |
| VPN tunnel | ✅ ISPBoss sebagai VPN server. WireGuard (v7), L2TP/IPSec (v6), PPTP (legacy), SSTP |
| VPN auto-config | ✅ Generate config otomatis, auto-configure via API, download .rsc script |
| VPN monitoring | ✅ Status, latency, handshake. Health check 30 detik, auto-reconnect |
| VPN keamanan | Key pair per tunnel, PSK opsional, firewall whitelist port, audit log |
| VPN bandwidth | ✅ Monitoring per tunnel, bandwidth cap per tier tenant, throttle jika melebihi |
| VPN high availability | ✅ 2 endpoint (geo-redundancy), auto-failover di client, SLA 99.9% |
| VPN addressing | Subnet per tenant 10.99.{seq}.0/24, max 253 perangkat |
| Unexpected reboot | ✅ Deteksi uptime reset, notifikasi ke teknisi, log untuk analisis pola |
| Bulk action router | ✅ Bulk sync, backup, firmware check, export status |
| Troubleshooting guide | ✅ In-dashboard help panel, kontekstual per halaman |
| Mobile layout | ✅ Card list untuk daftar router, optimized untuk teknisi di lapangan |
| Audit trail | ✅ Semua perintah ke router dicatat (append-only) |