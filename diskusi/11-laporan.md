# 11 — Reporting & Analytics

---

## Konsep

Modul laporan menyajikan data dari semua modul lain (pelanggan, billing, paket, MikroTik, OLT) dalam bentuk grafik, tabel, dan ringkasan yang actionable. Laporan membantu pemilik ISP mengambil keputusan bisnis dan operasional.

### Kategori Laporan

| Kategori | Ikon | Deskripsi |
|---|---|---|
| **Keuangan** | 💰 | Pendapatan, piutang, pembayaran, voucher |
| **Pelanggan** | 👥 | Pertumbuhan, churn, distribusi paket/area |
| **Jaringan** | 📡 | Uptime router, traffic, signal quality, alarm |
| **Operasional** | ⚙️ | Aktivitas admin, notifikasi, sync status |

---

## Halaman Laporan (`/reports`)

### Layout Desktop

```
╔══════════════════════════════════════════════════════════════════════════╗
║  Dashboard > Laporan                                                     ║
║                                                                          ║
║  Laporan & Analytics                                                     ║
║                                                                          ║
║  ┌─── Filter Global ────────────────────────────────────────────────┐    ║
║  │  Periode: [April 2026 ▼]  Bandingkan: [Maret 2026 ▼]           │    ║
║  │  Area: [Semua ▼]  Paket: [Semua ▼]  [Terapkan]  [Reset]       │    ║
║  └──────────────────────────────────────────────────────────────────┘    ║
║                                                                          ║
║  [💰 Keuangan]  [👥 Pelanggan]  [📡 Jaringan]  [⚙️ Operasional]       ║
║                                                                          ║
║  ┌──────────────────────────────────────────────────────────────────┐    ║
║  │                                                                  │    ║
║  │  (Konten laporan sesuai tab yang dipilih)                        │    ║
║  │                                                                  │    ║
║  └──────────────────────────────────────────────────────────────────┘    ║
║                                                                          ║
║  [📥 Export PDF]  [📥 Export Excel]  [📅 Jadwalkan Laporan]             ║
╚══════════════════════════════════════════════════════════════════════════╝
```

### Filter Global
Semua laporan bisa difilter dengan:

| Filter | Opsi |
|---|---|
| Periode | Hari ini, Minggu ini, Bulan ini, Kuartal, Tahun, Custom range |
| Bandingkan | Periode sebelumnya (untuk melihat trend/perubahan) |
| Area | Semua atau pilih area tertentu |
| Paket | Semua atau pilih paket tertentu |
| Router | Semua atau pilih router tertentu (untuk laporan jaringan) |


---

## 💰 Laporan Keuangan

### 1. Ringkasan Pendapatan

```
┌────────────────────────────────────────────────────────────────┐
│ Pendapatan — April 2026                    vs Maret 2026       │
│                                                                │
│ ┌──────────────┬──────────────┬──────────────┬───────────────┐ │
│ │ Total        │ Bulanan      │ Voucher      │ Lainnya       │ │
│ │ Rp 282.2jt   │ Rp 252.0jt   │ Rp 15.2jt    │ Rp 15.0jt    │ │
│ │ ↑ 5.2%       │ ↑ 4.8%       │ ↑ 12.3%      │ ↑ 2.1%       │ │
│ └──────────────┴──────────────┴──────────────┴───────────────┘ │
│                                                                │
│ Grafik Pendapatan (12 bulan):                                  │
│ 300jt ┤                                              ██        │
│ 250jt ┤                              ██  ██  ██  ██  ██        │
│ 200jt ┤              ██  ██  ██  ██  ██  ██  ██  ██  ██        │
│ 150jt ┤  ██  ██  ██  ██  ██  ██  ██  ██  ██  ██  ██  ██        │
│       └──May──Jun──Jul──Aug──Sep──Oct──Nov──Dec──Jan──Feb──Mar──Apr│
│                                                                │
│ ■ Bulanan  ■ Voucher  ■ Lainnya                               │
└────────────────────────────────────────────────────────────────┘
```

### 2. Laporan Piutang / Tunggakan (Aging Report)

```
┌────────────────────────────────────────────────────────────────┐
│ Aging Report — April 2026                                      │
│                                                                │
│ ┌──────────────┬──────────┬──────────┬──────────┬────────────┐ │
│ │ 1-7 hari     │ 8-14 hari│ 15-30 hari│ 30+ hari│ Total      │ │
│ ├──────────────┼──────────┼──────────┼──────────┼────────────┤ │
│ │ Rp 12.5jt    │ Rp 8.2jt │ Rp 5.8jt │ Rp 3.5jt│ Rp 30.0jt  │ │
│ │ 35 plgn      │ 22 plgn  │ 18 plgn  │ 12 plgn │ 87 plgn    │ │
│ └──────────────┴──────────┴──────────┴──────────┴────────────┘ │
│                                                                │
│ Collection Rate: 85.6% (target: 90%)                           │
│ Rata-rata Waktu Bayar: 3.2 hari sebelum jatuh tempo           │
│                                                                │
│ Top 10 Tunggakan Terbesar:                                     │
│ 1. Budi S. (PLG-002) — Rp 1.050.000 (3 bulan)                │
│ 2. Eko P. (PLG-005) — Rp 750.000 (2 bulan)                   │
│ ...                                                            │
│                                                                │
│ Trend Piutang (6 bulan):                                       │
│ Nov: 22jt → Dec: 25jt → Jan: 24jt → Feb: 28jt → Mar: 29jt → Apr: 30jt│
└────────────────────────────────────────────────────────────────┘
```

### 3. Laporan Pembayaran

```
┌────────────────────────────────────────────────────────────────┐
│ Pembayaran — April 2026                                        │
│                                                                │
│ Per Metode:                                                    │
│ ┌──────────────┬──────────┬──────────┬──────────┐              │
│ │ Tunai        │ Transfer │ Xendit   │ QRIS     │              │
│ │ Rp 85.0jt    │ Rp 62.0jt│ Rp 80.0jt│ Rp 25.0jt│              │
│ │ 34%          │ 25%      │ 32%      │ 10%      │              │
│ └──────────────┴──────────┴──────────┴──────────┘              │
│                                                                │
│ Grafik Pembayaran Harian (bulan ini):                          │
│ ▁▂▃▅▆█▇▅▆▇█▆▅▃▂▁▂▃▅▆█▇▅▃▂▁▂▃▅                               │
│ 1  3  5  7  9  11 13 15 17 19 21 23 25 27 29                  │
│                                                                │
│ Puncak pembayaran: tanggal 5 (jatuh tempo mayoritas)           │
└────────────────────────────────────────────────────────────────┘
```

### 4. Laporan Pendapatan Voucher

```
┌────────────────────────────────────────────────────────────────┐
│ Pendapatan Voucher — April 2026                                │
│                                                                │
│ Total: Rp 15.2jt (1,520 voucher terjual)                      │
│                                                                │
│ Per Paket:                                                     │
│ ┌──────────────┬──────────┬──────────┬──────────┐              │
│ │ 1 Hari 5M    │ 3 Hari 10M│ 7 Hari 10M│ Lainnya │              │
│ │ Rp 4.5jt     │ Rp 5.8jt  │ Rp 3.9jt  │ Rp 1.0jt│              │
│ │ 750 vcr      │ 420 vcr   │ 260 vcr   │ 90 vcr  │              │
│ └──────────────┴──────────┴──────────┴──────────┘              │
│                                                                │
│ Per Reseller:                                                  │
│ 1. Toko Adi — Rp 6.2jt (620 vcr)                             │
│ 2. Warnet Budi — Rp 4.5jt (450 vcr)                          │
│ 3. Cafe Citra — Rp 2.8jt (280 vcr)                           │
│                                                                │
│ Margin Reseller Total: Rp 3.8jt                                │
└────────────────────────────────────────────────────────────────┘
```

### 5. Laporan Laba Rugi Sederhana

```
┌────────────────────────────────────────────────────────────────┐
│ Laba Rugi Sederhana — April 2026                               │
│                                                                │
│ PENDAPATAN:                                                    │
│   Tagihan Bulanan              Rp 252.000.000                  │
│   Penjualan Voucher            Rp  15.200.000                  │
│   Biaya Pasang                 Rp   8.500.000                  │
│   Denda Keterlambatan          Rp   1.200.000                  │
│   Lainnya                      Rp   5.300.000                  │
│   ─────────────────────────────────────────                    │
│   Total Pendapatan             Rp 282.200.000                  │
│                                                                │
│ PENGELUARAN (input manual):                                    │
│   Bandwidth / Upstream         Rp  45.000.000                  │
│   Sewa Tiang / Infrastruktur   Rp  12.000.000                  │
│   Gaji Karyawan                Rp  35.000.000                  │
│   Listrik & Operasional        Rp   8.000.000                  │
│   Notifikasi (WA/SMS)          Rp     264.000                  │
│   Lainnya                      Rp   5.000.000                  │
│   ─────────────────────────────────────────                    │
│   Total Pengeluaran            Rp 105.264.000                  │
│                                                                │
│ LABA BERSIH                    Rp 176.936.000                  │
│ Margin: 62.7%                                                  │
└────────────────────────────────────────────────────────────────┘
```

> **Catatan:** Pengeluaran diinput manual oleh admin (ISPBoss bukan software akuntansi). Pendapatan otomatis dari data billing.


---

## 👥 Laporan Pelanggan

### 1. Pertumbuhan Pelanggan

```
┌────────────────────────────────────────────────────────────────┐
│ Pertumbuhan Pelanggan — 2026                                   │
│                                                                │
│ ┌──────────────┬──────────────┬──────────────┬───────────────┐ │
│ │ Total Aktif  │ Baru (bulan) │ Churn (bulan)│ Net Growth    │ │
│ │ 847          │ +38          │ -5           │ +33           │ │
│ │              │ ↑ 12% vs Mar │ ↓ 2% vs Mar  │ ↑ 15% vs Mar │ │
│ └──────────────┴──────────────┴──────────────┴───────────────┘ │
│                                                                │
│ Grafik Pertumbuhan (12 bulan):                                 │
│ 900 ┤                                                    ██    │
│ 800 ┤                              ██  ██  ██  ██  ██  ██      │
│ 700 ┤              ██  ██  ██  ██                              │
│ 600 ┤  ██  ██  ██                                              │
│     └──May──Jun──Jul──Aug──Sep──Oct──Nov──Dec──Jan──Feb──Mar──Apr│
│                                                                │
│ ■ Total Aktif  ── Baru  ── Churn                              │
│                                                                │
│ Churn Rate: 0.6% (target: < 2%)                               │
│ Average Revenue Per User (ARPU): Rp 333.000                   │
│ Customer Lifetime Value (CLV): Rp 6.660.000 (20 bulan avg)    │
└────────────────────────────────────────────────────────────────┘
```

### 2. Distribusi Pelanggan

```
┌────────────────────────────────────────────────────────────────┐
│ Distribusi Pelanggan — April 2026                              │
│                                                                │
│ Per Paket:                          Per Status:                │
│ ┌──────────┬──────┬───────┐        ┌──────────┬──────┐        │
│ │ Basic 10M│ 320  │ 38%   │        │ 🟢 Aktif │ 720  │        │
│ │ Pro 50M  │ 412  │ 49%   │        │ 🟡 Pending│ 15  │        │
│ │ Ultra100M│ 87   │ 10%   │        │ 🔴 Isolir │ 87  │        │
│ │ Lainnya  │ 28   │ 3%    │        │ 🟣 Suspend│ 5   │        │
│ └──────────┴──────┴───────┘        │ ⚫ Berhenti│ 25  │        │
│                                    └──────────┴──────┘        │
│ Per Area:                           Per Metode Koneksi:        │
│ ┌──────────┬──────┬───────┐        ┌──────────┬──────┐        │
│ │ Sukamaju │ 245  │ 29%   │        │ PPPoE    │ 720  │        │
│ │ Mekarjaya│ 180  │ 21%   │        │ DHCP     │ 85   │        │
│ │ Cimanggis│ 150  │ 18%   │        │ Static   │ 30   │        │
│ │ Lainnya  │ 272  │ 32%   │        │ Hotspot  │ 12   │        │
│ └──────────┴──────┴───────┘        └──────────┴──────┘        │
└────────────────────────────────────────────────────────────────┘
```

### 3. Laporan Churn (Pelanggan Berhenti)

```
┌────────────────────────────────────────────────────────────────┐
│ Analisis Churn — April 2026                                    │
│                                                                │
│ Pelanggan berhenti bulan ini: 5                                │
│                                                                │
│ Alasan Berhenti:                                               │
│ ┌──────────────────────┬──────┬───────┐                        │
│ │ Pindah rumah         │ 2    │ 40%   │                        │
│ │ Harga terlalu mahal  │ 1    │ 20%   │                        │
│ │ Pindah ke ISP lain   │ 1    │ 20%   │                        │
│ │ Tidak diketahui      │ 1    │ 20%   │                        │
│ └──────────────────────┴──────┴───────┘                        │
│                                                                │
│ Rata-rata lama berlangganan sebelum churn: 8.5 bulan           │
│ Paket paling banyak churn: Basic 10M (3 dari 5)               │
│ Area paling banyak churn: Mekarjaya (2 dari 5)                │
└────────────────────────────────────────────────────────────────┘
```


---

## 📡 Laporan Jaringan

### 1. Uptime & Status Router

```
┌────────────────────────────────────────────────────────────────┐
│ Uptime Router — April 2026                                     │
│                                                                │
│ ┌──────────┬──────────┬──────────┬──────────┬────────────────┐ │
│ │ Router   │ Uptime % │ Downtime │ Reboot   │ Status         │ │
│ ├──────────┼──────────┼──────────┼──────────┼────────────────┤ │
│ │ MK-01    │ 99.98%   │ 8 menit  │ 0        │ 🟢 Excellent   │ │
│ │ MK-02    │ 99.85%   │ 65 menit │ 1 (PLN)  │ 🟢 Good        │ │
│ │ MK-03    │ 98.50%   │ 10.8 jam │ 3        │ 🟡 Fair        │ │
│ └──────────┴──────────┴──────────┴──────────┴────────────────┘ │
│                                                                │
│ SLA Target: 99.5%                                              │
│ Router di bawah SLA: MK-03 (98.50%) ⚠️                       │
│                                                                │
│ Timeline Downtime MK-03:                                       │
│ ──────█──────────────█████──────────█──────────────────────    │
│ 1 Apr          10 Apr          20 Apr          30 Apr          │
│ (8 menit)      (9 jam PLN)     (1.8 jam)                      │
└────────────────────────────────────────────────────────────────┘
```

### 2. Traffic Report

```
┌────────────────────────────────────────────────────────────────┐
│ Traffic Report — April 2026                                    │
│                                                                │
│ Total Traffic:                                                 │
│ ┌──────────────┬──────────────┬──────────────┐                 │
│ │ Download     │ Upload       │ Total        │                 │
│ │ 38.5 TB      │ 8.2 TB       │ 46.7 TB      │                 │
│ │ ↑ 8% vs Mar  │ ↑ 5% vs Mar  │ ↑ 7% vs Mar  │                 │
│ └──────────────┴──────────────┴──────────────┘                 │
│                                                                │
│ Peak Traffic: 1.8 Gbps (Minggu, 20:00)                        │
│ Average Traffic: 650 Mbps                                      │
│                                                                │
│ Traffic per Router:                                            │
│ MK-01: ████████████████████ 18.5 TB (40%)                     │
│ MK-02: ██████████████████████████ 22.0 TB (47%)               │
│ MK-03: ██████ 6.2 TB (13%)                                    │
│                                                                │
│ Top 10 Pelanggan (traffic terbesar):                           │
│ 1. Ahmad R. (PLG-001) — 850 GB (Pro 50M)                     │
│ 2. Budi S. (PLG-002) — 620 GB (Basic 10M) ⚠️ over-use       │
│ ...                                                            │
└────────────────────────────────────────────────────────────────┘
```

### 3. Signal Quality Report (OLT)

```
┌────────────────────────────────────────────────────────────────┐
│ Signal Quality — April 2026                                    │
│                                                                │
│ ┌──────────────┬──────────┬──────────┬──────────┬────────────┐ │
│ │ 🟢 Normal    │ 🟡 Warning│ ⚠️ Weak  │ 🔴 Critical│ Total   │ │
│ │ 210 (86%)    │ 20 (8%)  │ 10 (4%)  │ 5 (2%)   │ 245 ONT  │ │
│ └──────────────┴──────────┴──────────┴──────────┴────────────┘ │
│                                                                │
│ Rata-rata Signal: -20.5 dBm                                   │
│ ONT dengan signal memburuk (trend turun 30 hari):              │
│ 1. Citra D. (PLG-003) — -28.1 dBm (turun 3 dB dalam 30 hari)│
│ 2. Fajar R. (PLG-008) — -26.5 dBm (turun 2 dB)              │
│                                                                │
│ Alarm bulan ini: 45 total (32 LOS, 8 Dying Gasp, 5 lainnya)  │
└────────────────────────────────────────────────────────────────┘
```

### 4. Laporan Alarm OLT

```
┌────────────────────────────────────────────────────────────────┐
│ Alarm Summary — April 2026                                     │
│                                                                │
│ ┌──────────────────┬──────────┬──────────┬──────────┐          │
│ │ Tipe Alarm       │ Jumlah   │ Avg Durasi│ Resolved │          │
│ ├──────────────────┼──────────┼──────────┼──────────┤          │
│ │ ONT LOS          │ 32       │ 2.5 jam  │ 30 (94%) │          │
│ │ Dying Gasp       │ 8        │ 45 menit │ 8 (100%) │          │
│ │ Signal Degraded  │ 5        │ ongoing  │ 2 (40%)  │          │
│ └──────────────────┴──────────┴──────────┴──────────┘          │
│                                                                │
│ Area dengan alarm terbanyak: Mekarjaya (15 alarm)              │
│ Penyebab utama LOS: PLN mati (60%), kabel putus (25%),        │
│                      konektor lepas (15%)                      │
└────────────────────────────────────────────────────────────────┘
```


---

## ⚙️ Laporan Operasional

### 1. Aktivitas Admin & Operator

```
┌────────────────────────────────────────────────────────────────┐
│ Aktivitas User — April 2026                                    │
│                                                                │
│ ┌──────────────┬──────────┬──────────┬──────────┬────────────┐ │
│ │ User         │ Role     │ Login    │ Aksi     │ Terakhir   │ │
│ ├──────────────┼──────────┼──────────┼──────────┼────────────┤ │
│ │ Admin Budi   │ Admin    │ 28 hari  │ 1,250    │ Hari ini   │ │
│ │ Kasir Ani    │ Kasir    │ 25 hari  │ 890      │ Hari ini   │ │
│ │ Teknisi Andi │ Teknisi  │ 22 hari  │ 450      │ Kemarin    │ │
│ │ Operator Dewi│ Operator │ 20 hari  │ 680      │ Hari ini   │ │
│ └──────────────┴──────────┴──────────┴──────────┴────────────┘ │
│                                                                │
│ Top Aksi:                                                      │
│ 1. Catat pembayaran — 720x                                    │
│ 2. Edit pelanggan — 150x                                      │
│ 3. Kirim notifikasi — 2,450x                                  │
│ 4. Isolir/buka isolir — 127x                                  │
└────────────────────────────────────────────────────────────────┘
```

### 2. Laporan Notifikasi

```
┌────────────────────────────────────────────────────────────────┐
│ Notifikasi — April 2026                                        │
│                                                                │
│ ┌──────────────┬──────────┬──────────┬──────────┐              │
│ │ Total Kirim  │ Terkirim │ Gagal    │ Biaya    │              │
│ │ 2,450        │ 2,380    │ 52       │ Rp 264rb │              │
│ │              │ 97.1%    │ 2.1%     │          │              │
│ └──────────────┴──────────┴──────────┴──────────┘              │
│                                                                │
│ Per Channel:                                                   │
│ WhatsApp: 2,100 (97.5% success) — Rp 210rb                   │
│ SMS: 52 (94.2% success) — Rp 26rb                             │
│ Email: 298 (99.5% success) — Rp 28rb                          │
│                                                                │
│ Per Template:                                                  │
│ Invoice Baru: 847x • Konfirmasi Bayar: 720x • Reminder: 450x │
└────────────────────────────────────────────────────────────────┘
```

### 3. Laporan Sync MikroTik & OLT

```
┌────────────────────────────────────────────────────────────────┐
│ Sync Status — April 2026                                       │
│                                                                │
│ MikroTik:                                                      │
│ ┌──────────┬──────────┬──────────┬──────────┬────────────────┐ │
│ │ Router   │ Sync OK  │ Failed   │ Orphan   │ Pending        │ │
│ ├──────────┼──────────┼──────────┼──────────┼────────────────┤ │
│ │ MK-01    │ 2,880    │ 3        │ 2        │ 0              │ │
│ │ MK-02    │ 2,880    │ 5        │ 0        │ 1              │ │
│ └──────────┴──────────┴──────────┴──────────┴────────────────┘ │
│                                                                │
│ OLT:                                                           │
│ ┌──────────┬──────────┬──────────┬──────────┐                  │
│ │ OLT      │ Sync OK  │ Failed   │ Unmanaged│                  │
│ ├──────────┼──────────┼──────────┼──────────┤                  │
│ │ OLT-01   │ 1,440    │ 2        │ 3        │                  │
│ └──────────┴──────────┴──────────┴──────────┘                  │
│                                                                │
│ Sync success rate: 99.8%                                       │
└────────────────────────────────────────────────────────────────┘
```

### 4. Laporan Kapasitas Jaringan

```
┌────────────────────────────────────────────────────────────────┐
│ Kapasitas Jaringan — April 2026                                │
│                                                                │
│ Pelanggan per Router:                                          │
│ ┌──────────┬──────────┬──────────┬──────────┬────────────────┐ │
│ │ Router   │ Pelanggan│ Kapasitas│ Terpakai │ Estimasi Penuh │ │
│ ├──────────┼──────────┼──────────┼──────────┼────────────────┤ │
│ │ MK-01    │ 320      │ 500      │ 64%      │ ~5 bulan lagi  │ │
│ │ MK-02    │ 412      │ 500      │ 82% ⚠️  │ ~2 bulan lagi  │ │
│ └──────────┴──────────┴──────────┴──────────┴────────────────┘ │
│                                                                │
│ Pelanggan per ODP:                                             │
│ ┌──────────┬──────────┬──────────┬──────────┐                  │
│ │ ODP      │ Terpakai │ Kapasitas│ Status   │                  │
│ ├──────────┼──────────┼──────────┼──────────┤                  │
│ │ ODP-01-A │ 7/8      │ 87%      │ ⚠️ Hampir│                  │
│ │ ODP-01-B │ 10/16    │ 63%      │ 🟢 OK    │                  │
│ │ ODP-02-A │ 3/8      │ 38%      │ 🟢 OK    │                  │
│ └──────────┴──────────┴──────────┴──────────┘                  │
│                                                                │
│ Rekomendasi:                                                   │
│ ⚠️ MK-02 mendekati kapasitas. Pertimbangkan tambah router.   │
│ ⚠️ ODP-01-A hampir penuh. Pasang ODP baru di area Sukamaju.  │
└────────────────────────────────────────────────────────────────┘
```

### 5. Laporan Traffic per Pelanggan

```
┌────────────────────────────────────────────────────────────────┐
│ Traffic per Pelanggan — April 2026                             │
│                                                                │
│ Top 10 (traffic terbesar):                                     │
│ ┌──────────────┬──────────┬──────────┬──────────┬────────────┐ │
│ │ Pelanggan    │ Paket    │ Download │ Upload   │ Status     │ │
│ ├──────────────┼──────────┼──────────┼──────────┼────────────┤ │
│ │ Ahmad R.     │ Pro 50M  │ 850 GB   │ 120 GB   │ 🟢 Normal  │ │
│ │ Budi S.      │ Basic 10M│ 620 GB   │ 80 GB    │ ⚠️ Over-use│ │
│ │ Citra D.     │ Pro 50M  │ 580 GB   │ 95 GB    │ 🟢 Normal  │ │
│ └──────────────┴──────────┴──────────┴──────────┴────────────┘ │
│                                                                │
│ Over-use: pelanggan Basic 10M tapi traffic setara Pro 50M     │
│ Rekomendasi: tawarkan upgrade paket ke pelanggan over-use     │
└────────────────────────────────────────────────────────────────┘
```

---

## Export Laporan

### Format Export

| Format | Kegunaan | Proses |
|---|---|---|
| **PDF** | Cetak, kirim ke pemilik ISP, arsip | Async (background job) |
| **Excel (.xlsx)** | Analisis lanjutan, pivot table | Async |
| **CSV** | Import ke software lain | Langsung (sinkron) |

### Template PDF Laporan

```
┌──────────────────────────────────────────────────────────┐
│  [LOGO]  ISPBoss Net                                     │
│  Laporan Pendapatan — April 2026                         │
│  ─────────────────────────────────────────────────────── │
│                                                          │
│  Ringkasan:                                              │
│  Total Pendapatan: Rp 282.200.000                        │
│  Collection Rate: 85.6%                                  │
│  Pelanggan Aktif: 847                                    │
│                                                          │
│  [Grafik Pendapatan 12 Bulan]                            │
│                                                          │
│  [Tabel Detail per Paket]                                │
│  [Tabel Detail per Area]                                 │
│                                                          │
│  ─────────────────────────────────────────────────────── │
│  Digenerate: 28 Apr 2026 14:30                           │
│  ISPBoss — Kelola ISP Kamu Dari Satu Dashboard           │
└──────────────────────────────────────────────────────────┘
```

- Branding tenant (logo, nama ISP)
- Grafik di-render sebagai gambar di PDF
- Generate async → notifikasi saat selesai → download link

---

## Jadwal Laporan Otomatis

Admin bisa jadwalkan laporan untuk digenerate dan dikirim otomatis:

```
╔══════════════════════════════════════════════════════════════╗
║  Jadwalkan Laporan                                           ║
║                                                              ║
║  Laporan *: [Ringkasan Pendapatan ▼]                        ║
║                                                              ║
║  Jadwal *:                                                   ║
║  ○ Harian (setiap jam 07:00)                                ║
║  ● Mingguan (setiap Senin jam 07:00)                        ║
║  ○ Bulanan (setiap tanggal 1 jam 07:00)                     ║
║                                                              ║
║  Format *: ● PDF  ○ Excel                                   ║
║                                                              ║
║  Kirim ke:                                                   ║
║  ☑ Email: admin@ispboss.net                                 ║
║  ☑ WhatsApp: +6281234567890                                 ║
║                                                              ║
║  Penerima tambahan:                                          ║
║  [+ Tambah email/WA]                                        ║
║                                                              ║
║                    [Batal]  [Simpan Jadwal]                  ║
╚══════════════════════════════════════════════════════════════╝
```

| Laporan | Jadwal Rekomendasi |
|---|---|
| Ringkasan Pendapatan | Bulanan (tanggal 1) |
| Aging Report | Mingguan (Senin) |
| Pertumbuhan Pelanggan | Bulanan |
| Uptime Router | Mingguan |
| Signal Quality | Mingguan |
| Laba Rugi | Bulanan |

- Laporan digenerate otomatis oleh background job
- Dikirim via email (PDF attachment) dan/atau WA (link download)
- Riwayat laporan tersimpan 12 bulan

---

## Dashboard Analytics (Widget di Halaman Utama)

Beberapa metrik kunci ditampilkan di dashboard utama (dokumen 03):

| Widget | Data | Update |
|---|---|---|
| Total Pelanggan | Aktif, trend bulan ini | Real-time |
| Pendapatan Bulan Ini | Total diterima, % dari target | Real-time |
| Tunggakan | Total piutang, jumlah pelanggan | Real-time |
| Router Online | Online/offline, alert | Real-time |
| Collection Rate | % invoice terbayar bulan ini | Harian |
| Churn Rate | % pelanggan berhenti bulan ini | Harian |
| ARPU | Rata-rata pendapatan per pelanggan | Bulanan |

---

## Target / KPI Setting

Admin bisa set target bisnis yang ditampilkan di laporan sebagai pembanding:

```
╔══════════════════════════════════════════════════════════════╗
║  Settings > Laporan > Target KPI                             ║
║                                                              ║
║  ┌─── Target Keuangan ───────────────────────────────────┐   ║
║  │  Pendapatan Bulanan    : [Rp 300.000.000]             │   ║
║  │  Collection Rate       : [90] %                       │   ║
║  │  Max Piutang           : [Rp 25.000.000]              │   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║  ┌─── Target Pelanggan ──────────────────────────────────┐   ║
║  │  Pelanggan Baru / Bulan: [40]                         │   ║
║  │  Max Churn Rate        : [2] %                        │   ║
║  │  Target Total Pelanggan: [1000] (akhir tahun)         │   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║  ┌─── Target Jaringan ───────────────────────────────────┐   ║
║  │  SLA Uptime Router     : [99.5] %                     │   ║
║  │  Max Alarm Aktif       : [5]                          │   ║
║  │  Min Signal Quality    : [85] % ONT normal            │   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║                                          [Simpan Target]     ║
╚══════════════════════════════════════════════════════════════╝
```

### Tampilan Target di Laporan

```
Pendapatan April 2026:
  Rp 282.2jt / Rp 300jt target
  ████████████████████████░░░░  94.1%  🟡 Hampir tercapai

Collection Rate:
  85.6% / 90% target
  ████████████████████░░░░░░░░  85.6%  🔴 Di bawah target

Pelanggan Baru:
  38 / 40 target
  ████████████████████████████░  95.0%  🟡 Hampir tercapai
```

- Progress bar visual di setiap metrik yang punya target
- Warna: 🟢 tercapai (≥100%), 🟡 hampir (≥80%), 🔴 di bawah (<80%)
- Target bisa diubah kapan saja, berlaku untuk periode berikutnya

---

## Laporan Pendapatan per Area

```
┌────────────────────────────────────────────────────────────────┐
│ Pendapatan per Area — April 2026                               │
│                                                                │
│ ┌──────────────┬──────────┬──────────┬──────────┬────────────┐ │
│ │ Area         │ Pelanggan│ Pendapatan│ Piutang  │ ARPU       │ │
│ ├──────────────┼──────────┼──────────┼──────────┼────────────┤ │
│ │ Sukamaju     │ 245      │ Rp 85.7jt│ Rp 8.5jt │ Rp 350rb   │ │
│ │ Mekarjaya    │ 180      │ Rp 54.0jt│ Rp 9.2jt │ Rp 300rb   │ │
│ │ Cimanggis    │ 150      │ Rp 52.5jt│ Rp 5.3jt │ Rp 350rb   │ │
│ │ Bojong Gede  │ 120      │ Rp 42.0jt│ Rp 4.0jt │ Rp 350rb   │ │
│ │ Lainnya      │ 152      │ Rp 48.0jt│ Rp 3.0jt │ Rp 316rb   │ │
│ ├──────────────┼──────────┼──────────┼──────────┼────────────┤ │
│ │ **Total**    │ **847**  │**Rp282jt**│**Rp30jt**│ **Rp333rb**│ │
│ └──────────────┴──────────┴──────────┴──────────┴────────────┘ │
│                                                                │
│ Area paling menguntungkan: Sukamaju (ARPU Rp 350rb, piutang rendah)│
│ Area perlu perhatian: Mekarjaya (piutang tinggi Rp 9.2jt)     │
│                                                                │
│ Grafik Pendapatan per Area (pie chart):                        │
│ ■ Sukamaju 30%  ■ Mekarjaya 19%  ■ Cimanggis 19%             │
│ ■ Bojong Gede 15%  ■ Lainnya 17%                              │
└────────────────────────────────────────────────────────────────┘
```

- Pendapatan, piutang, dan ARPU per area
- Identifikasi area paling menguntungkan dan area bermasalah
- Bisa drill-down: klik area → lihat detail pelanggan di area tersebut

---

## Forecasting / Proyeksi

Berdasarkan data historis 6 bulan terakhir, sistem memberikan proyeksi sederhana (linear projection):

```
┌────────────────────────────────────────────────────────────────┐
│ Proyeksi — Berdasarkan Trend 6 Bulan Terakhir                 │
│                                                                │
│ Pendapatan:                                                    │
│ ┌──────────────────────────────────────────────────────────┐   │
│ │ Mei 2026 (proyeksi): Rp 290jt (↑ 2.8% dari April)      │   │
│ │ Jun 2026 (proyeksi): Rp 298jt                           │   │
│ │ Target Rp 300jt tercapai: ~Juli 2026                    │   │
│ │                                                          │   │
│ │ Aktual ────────── Proyeksi - - - - -                    │   │
│ │ 300jt ┤                          - - - ✓ target         │   │
│ │ 280jt ┤                    ██ - -                        │   │
│ │ 260jt ┤              ██  ██                              │   │
│ │ 240jt ┤        ██  ██                                    │   │
│ │       └──Nov──Dec──Jan──Feb──Mar──Apr──May──Jun──Jul     │   │
│ └──────────────────────────────────────────────────────────┘   │
│                                                                │
│ Pelanggan:                                                     │
│ Proyeksi akhir 2026: 1,043 pelanggan (growth rate 33/bulan)   │
│ Target 1,000 tercapai: ~November 2026                          │
│                                                                │
│ Piutang:                                                       │
│ Jika collection rate tetap 85.6%:                              │
│ Proyeksi piutang Mei: Rp 31.5jt (↑ 5%)                       │
│ Rekomendasi: tingkatkan collection rate ke 90% untuk           │
│ menurunkan piutang ke Rp 25jt                                  │
│                                                                │
│ ⚠️ Proyeksi berdasarkan linear trend. Hasil aktual bisa       │
│ berbeda karena faktor musiman, promo, atau perubahan pasar.    │
└────────────────────────────────────────────────────────────────┘
```

- Proyeksi **3 bulan ke depan** berdasarkan trend 6 bulan terakhir
- Metode: linear regression sederhana (bukan AI/ML)
- Tampilkan kapan target KPI akan tercapai
- Disclaimer: proyeksi bukan jaminan, hanya estimasi

---

## Perbandingan Antar Periode (Side-by-Side)

```
┌────────────────────────────────────────────────────────────────┐
│ Perbandingan: April 2026 vs April 2025 (Year-over-Year)       │
│                                                                │
│ ┌──────────────────┬──────────────┬──────────────┬───────────┐ │
│ │ Metrik           │ Apr 2025     │ Apr 2026     │ Delta     │ │
│ ├──────────────────┼──────────────┼──────────────┼───────────┤ │
│ │ Pendapatan       │ Rp 180.5jt   │ Rp 282.2jt   │ ↑ 56.3%  │ │
│ │ Pelanggan Aktif  │ 520          │ 847          │ ↑ 62.9%  │ │
│ │ ARPU             │ Rp 347rb     │ Rp 333rb     │ ↓ 4.0%   │ │
│ │ Collection Rate  │ 88.2%        │ 85.6%        │ ↓ 2.6pp  │ │
│ │ Churn Rate       │ 1.2%         │ 0.6%         │ ↓ 0.6pp  │ │
│ │ Piutang          │ Rp 15.0jt    │ Rp 30.0jt    │ ↑ 100%   │ │
│ │ Router Uptime    │ 99.2%        │ 99.5%        │ ↑ 0.3pp  │ │
│ └──────────────────┴──────────────┴──────────────┴───────────┘ │
│                                                                │
│ Insight:                                                       │
│ ✅ Pendapatan dan pelanggan naik signifikan (+56%, +63%)       │
│ ⚠️ ARPU turun 4% — mungkin karena banyak pelanggan Basic baru │
│ ⚠️ Piutang naik 100% — perlu perhatian collection              │
│ ✅ Churn rate membaik (1.2% → 0.6%)                           │
└────────────────────────────────────────────────────────────────┘
```

### Opsi Perbandingan
| Tipe | Contoh |
|---|---|
| Month-over-Month (MoM) | April 2026 vs Maret 2026 |
| Year-over-Year (YoY) | April 2026 vs April 2025 |
| Quarter-over-Quarter (QoQ) | Q1 2026 vs Q4 2025 |
| Custom | Pilih 2 periode bebas |

- Delta ditampilkan dalam **persentase** dan **nilai absolut**
- Warna: hijau (membaik), merah (memburuk), abu-abu (stabil)
- **Insight otomatis**: sistem generate 3-5 insight berdasarkan delta terbesar

---

## Custom Report Builder

Admin bisa buat laporan custom yang tidak ada di template bawaan:

```
╔══════════════════════════════════════════════════════════════╗
║  Custom Report Builder                                       ║
║                                                              ║
║  Nama Laporan: [Pendapatan per Area per Paket___]           ║
║                                                              ║
║  Metrik (pilih 1 atau lebih):                                ║
║  ☑ Jumlah Pelanggan                                         ║
║  ☑ Pendapatan                                               ║
║  ☐ Piutang                                                  ║
║  ☐ ARPU                                                     ║
║  ☐ Collection Rate                                          ║
║  ☐ Churn Rate                                               ║
║  ☐ Traffic (GB)                                             ║
║  ☐ Signal Rata-rata (dBm)                                   ║
║                                                              ║
║  Group By (dimensi):                                         ║
║  ● Area                                                      ║
║  ○ Paket                                                     ║
║  ○ Bulan                                                     ║
║  ○ Status                                                    ║
║  ○ Metode Koneksi                                            ║
║  ○ Router                                                    ║
║                                                              ║
║  Sub-Group (opsional):                                       ║
║  [Paket ▼]                                                  ║
║                                                              ║
║  Periode: [Januari 2026] sampai [April 2026]                ║
║                                                              ║
║  Tampilan: ● Tabel  ○ Grafik Bar  ○ Grafik Line  ○ Pie     ║
║                                                              ║
║  [Preview]  [Simpan sebagai Template]  [Export]               ║
╚══════════════════════════════════════════════════════════════╝
```

### Hasil Custom Report

```
┌────────────────────────────────────────────────────────────────┐
│ Pendapatan per Area per Paket — Jan-Apr 2026                   │
│                                                                │
│ ┌──────────────┬──────────┬──────────┬──────────┬────────────┐ │
│ │ Area         │ Basic 10M│ Pro 50M  │ Ultra100M│ Total      │ │
│ ├──────────────┼──────────┼──────────┼──────────┼────────────┤ │
│ │ Sukamaju     │ Rp 120jt │ Rp 180jt │ Rp 42jt  │ Rp 342jt   │ │
│ │ Mekarjaya    │ Rp 95jt  │ Rp 100jt │ Rp 20jt  │ Rp 215jt   │ │
│ │ Cimanggis    │ Rp 80jt  │ Rp 110jt │ Rp 18jt  │ Rp 208jt   │ │
│ │ ...          │ ...      │ ...      │ ...      │ ...        │ │
│ └──────────────┴──────────┴──────────┴──────────┴────────────┘ │
│                                                                │
│ [Export PDF]  [Export Excel]                                    │
└────────────────────────────────────────────────────────────────┘
```

- Laporan custom bisa **disimpan sebagai template** untuk dipakai ulang
- Bisa dijadwalkan (sama seperti laporan bawaan)
- Max 3 metrik + 2 dimensi per laporan (untuk menjaga keterbacaan)
- Template custom tersimpan per tenant

---

## Mobile Layout Laporan

```
╔══════════════════════════╗
║  ☰  Laporan         🔍  ║
╠══════════════════════════╣
║                          ║
║  Periode: [Apr 2026 ▼]  ║
║                          ║
║  [💰] [👥] [📡] [⚙️]    ║
║  ← swipe tab kategori → ║
║                          ║
║  ┌──────────────────────┐║
║  │ Pendapatan           │║
║  │ Rp 282.2jt           │║
║  │ ████████████████░░ 94%│║
║  │ Target: Rp 300jt     │║
║  └──────────────────────┘║
║  ┌──────────────────────┐║
║  │ Collection Rate      │║
║  │ 85.6%                │║
║  │ ████████████████░░░░ │║
║  │ Target: 90%     🔴   │║
║  └──────────────────────┘║
║  ┌──────────────────────┐║
║  │ Pelanggan Aktif      │║
║  │ 847 (+33 bulan ini)  │║
║  │ ↑ 4.1% vs Mar       │║
║  └──────────────────────┘║
║                          ║
║  [Grafik Pendapatan →]   ║
║  (horizontal scroll)     ║
║                          ║
║  [📥 Export]             ║
╚══════════════════════════╝
```

- **Card-based layout** untuk ringkasan metrik
- **Swipe horizontal** antar tab kategori
- **Grafik responsive**: horizontal scroll jika terlalu lebar
- **Progress bar** dengan target KPI di setiap card
- Tombol export di bawah

---

## Pengeluaran (Input Manual)

ISPBoss bukan software akuntansi, tapi menyediakan input pengeluaran sederhana untuk laporan laba rugi:

```
╔══════════════════════════════════════════════════════════════╗
║  Pengeluaran — April 2026                  [+ Tambah]        ║
║                                                              ║
║  ┌──────────────────────┬──────────┬──────────┬────────────┐ ║
║  │ Kategori             │ Jumlah   │ Tanggal  │ Aksi       │ ║
║  ├──────────────────────┼──────────┼──────────┼────────────┤ ║
║  │ Bandwidth/Upstream   │ Rp 45jt  │ 01/04/26 │ ⋯          │ ║
║  │ Gaji Karyawan        │ Rp 35jt  │ 01/04/26 │ ⋯          │ ║
║  │ Sewa Tiang           │ Rp 12jt  │ 05/04/26 │ ⋯          │ ║
║  │ Listrik              │ Rp 8jt   │ 10/04/26 │ ⋯          │ ║
║  │ Beli ONT 10 unit     │ Rp 5jt   │ 15/04/26 │ ⋯          │ ║
║  └──────────────────────┴──────────┴──────────┴────────────┘ ║
║                                                              ║
║  Total Pengeluaran: Rp 105.264.000                           ║
╚══════════════════════════════════════════════════════════════╝
```

### Kategori Pengeluaran (Default)
| Kategori | Keterangan |
|---|---|
| Bandwidth / Upstream | Biaya bandwidth ke provider |
| Gaji Karyawan | Gaji admin, teknisi, kasir |
| Sewa Tiang / Infrastruktur | Sewa tiang PLN/Telkom |
| Listrik & Operasional | Listrik NOC, internet kantor |
| Perangkat | Beli ONT, router, kabel, dll |
| Notifikasi | Biaya WA/SMS (auto dari modul 07) |
| Lainnya | Pengeluaran tidak terkategori |

- Kategori bisa ditambah/edit oleh admin
- Pengeluaran recurring (misal gaji, sewa) bisa diset **auto-repeat** bulanan
- Biaya notifikasi otomatis diambil dari data modul 07

---

## Graceful Degradation

Jika data source tidak tersedia (modul dinonaktifkan atau service down):

| Situasi | Handling |
|---|---|
| Modul MikroTik nonaktif | Laporan jaringan (uptime, traffic) → hidden. Laporan keuangan & pelanggan tetap tampil |
| Modul OLT nonaktif | Laporan signal quality & alarm → hidden. Laporan lainnya tetap tampil |
| Network Service down | Laporan jaringan → tampilkan data terakhir yang di-cache + banner "Data terakhir: {waktu}" |
| Billing API down | Laporan keuangan → tampilkan data terakhir yang di-cache + banner "Data mungkin belum terbaru" |
| Data belum ada (tenant baru) | Tampilkan empty state: "Belum ada data untuk periode ini. Data akan muncul setelah ada pelanggan aktif." |
| Export gagal (PDF/Excel) | Retry otomatis 1x. Jika tetap gagal → notifikasi admin: "Export gagal, coba lagi nanti" |

- Laporan **tidak pernah error/crash** — selalu tampilkan sesuatu (data cache, empty state, atau pesan informatif)
- Data cache untuk laporan: **1 jam** (jika service down, data terakhir tetap bisa dilihat)
- Grafik yang tidak bisa di-render → tampilkan tabel sebagai fallback

---

## Integrasi dengan Modul Lain

| Modul | Data untuk Laporan |
|---|---|
| **Pelanggan (04)** | Jumlah pelanggan, distribusi paket/area/status, churn |
| **Paket (05)** | Pendapatan per paket, distribusi pelanggan per paket |
| **Billing (06)** | Pendapatan, piutang, pembayaran, collection rate, aging |
| **Notifikasi (07)** | Jumlah kirim, success rate, biaya per channel |
| **MikroTik (08)** | Uptime router, traffic, active sessions |
| **OLT (09)** | Signal quality, alarm, ONT status |
| **FTTH Mapping (10)** | Coverage area, kapasitas ODP |
| **Settings (12)** | Konfigurasi target (SLA, collection rate target) |

---

## Keputusan

| Keputusan | Detail |
|---|---|
| Kategori laporan | 4 kategori: Keuangan, Pelanggan, Jaringan, Operasional |
| Filter global | Periode, bandingkan, area, paket, router |
| Laporan keuangan | ✅ Pendapatan, piutang/aging, pembayaran, voucher, laba rugi sederhana |
| Laporan pelanggan | ✅ Pertumbuhan, distribusi, churn analysis (alasan, paket, area) |
| Laporan jaringan | ✅ Uptime router, traffic, signal quality, alarm summary |
| Laporan operasional | ✅ Aktivitas admin, notifikasi, sync status |
| Export | ✅ PDF (branding tenant, grafik), Excel, CSV. Generate async |
| Jadwal otomatis | ✅ Harian/mingguan/bulanan, kirim via email + WA |
| Pengeluaran | ✅ Input manual, kategori configurable, auto-repeat, biaya notifikasi otomatis |
| Laba rugi | ✅ Sederhana (pendapatan otomatis, pengeluaran manual). Bukan software akuntansi |
| Dashboard widget | ✅ Metrik kunci di halaman utama (real-time) |
| ARPU & CLV | ✅ Rata-rata pendapatan per user, customer lifetime value |
| Pendapatan per area | ✅ Pendapatan, piutang, ARPU per area. Identifikasi area menguntungkan vs bermasalah |
| Target / KPI | ✅ Set target pendapatan, collection rate, churn, SLA. Progress bar visual di laporan |
| Forecasting | ✅ Proyeksi 3 bulan ke depan (linear regression), kapan target tercapai |
| Perbandingan periode | ✅ MoM, YoY, QoQ, custom. Side-by-side delta + insight otomatis |
| Custom report builder | ✅ Pilih metrik + dimensi, simpan template, jadwalkan. Max 3 metrik + 2 dimensi |
| Mobile layout | ✅ Card-based, swipe tab, grafik horizontal scroll, progress bar KPI |
| Churn analysis | ✅ Rate, alasan, paket/area paling banyak churn |
| Data retention | Laporan tersimpan 12 bulan, data mentah sesuai modul masing-masing |
| Generate | Async via background job (asynq), notifikasi saat selesai |