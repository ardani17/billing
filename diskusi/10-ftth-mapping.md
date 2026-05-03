# 10 — FTTH Visual Mapping

---

## Konsep

Peta interaktif untuk memvisualisasikan seluruh jaringan fiber optik ISP: dari OLT di NOC sampai ONT di rumah pelanggan. Membantu teknisi merencanakan pemasangan, troubleshoot gangguan, dan memantau kapasitas jaringan secara visual.

```
Hierarki Jaringan FTTH:
  OLT (di NOC/POP)
    └── Kabel Backbone
          └── ODP / Splitter (di tiang/gedung)
                └── Kabel Drop
                      └── ONT (di rumah pelanggan)
```

### Tech Stack Peta
| Komponen | Teknologi | Alasan |
|---|---|---|
| Map Library | **Leaflet.js** | Open source, ringan, mobile-friendly |
| Tile Server | **OpenStreetMap** | Gratis, tidak perlu API key |
| React Wrapper | **react-leaflet** | Integrasi dengan Next.js |
| Drawing | **Leaflet.draw** | Plugin untuk gambar polyline, marker |
| Clustering | **Leaflet.markercluster** | Grouping marker saat zoom out |

---

## Layout Halaman Peta (`/network-map`)

### Desktop: Split View

```
╔══════════════════════════════════════════════════════════════════════════╗
║  Dashboard > Peta Jaringan                                               ║
║                                                                          ║
║  ┌────────────────────────────────────────────┬─────────────────────────┐║
║  │                                            │ 📋 Detail Panel         │║
║  │                                            │                         │║
║  │              [PETA LEAFLET]                │ ODP-01-A                │║
║  │                                            │ Jl. Merdeka No. 5      │║
║  │         📡 OLT-01                          │ Splitter: 1:8          │║
║  │          │                                 │ Terpakai: 7/8 (87%)    │║
║  │          ├── 🔵 ODP-01-A                   │                         │║
║  │          │    ├── 🟢 ONT Ahmad             │ ONT Terhubung:          │║
║  │          │    ├── 🟢 ONT Budi              │ 🟢 Ahmad R. (-18 dBm)  │║
║  │          │    ├── 🔴 ONT Citra (LOS)       │ 🟢 Budi S. (-22 dBm)  │║
║  │          │    └── 🟢 ONT Dewi              │ 🔴 Citra D. (LOS)     │║
║  │          │                                 │ 🟢 Dewi A. (-19 dBm)  │║
║  │          ├── 🔵 ODP-01-B                   │ ...                     │║
║  │          │    └── ...                      │                         │║
║  │          │                                 │ [Lihat Detail ODP]      │║
║  │                                            │ [Tambah ONT]            │║
║  │  [🔍] [📍] [✏️] [📐] [🗑️]  [Layer ▼]    │ [Edit Lokasi]           │║
║  └────────────────────────────────────────────┴─────────────────────────┘║
╚══════════════════════════════════════════════════════════════════════════╝
```

### Mobile: Full Screen + Floating Controls

```
╔══════════════════════════╗
║  ☰  Peta Jaringan   🔍  ║
╠══════════════════════════╣
║                          ║
║     [PETA FULL SCREEN]   ║
║                          ║
║  📡 OLT-01               ║
║   │                      ║
║   ├── 🔵 ODP-01-A        ║
║   │    ├── 🟢 Ahmad      ║
║   │    └── 🔴 Citra      ║
║   │                      ║
║                          ║
║  [📍] [✏️] [📐]         ║
║                          ║
╠══════════════════════════╣
║  ▲ Swipe up for details  ║
║  ODP-01-A • 7/8 terpakai ║
╚══════════════════════════╝
```

- Tap node → bottom sheet muncul dengan detail
- Swipe up → full detail panel
- Floating toolbar di bawah untuk drawing tools

---

## Node Types (Marker di Peta)

### Ikon & Warna per Tipe Node

| Node | Ikon | Warna | Ukuran | Keterangan |
|---|---|---|---|---|
| **OLT** | 📡 Tower | Biru tua | Besar | Pusat jaringan |
| **ODP / Splitter** | 🔵 Kotak | Biru | Sedang | Titik distribusi |
| **ONT (Online)** | 🟢 Bulat | Hijau | Kecil | Pelanggan aktif, signal normal |
| **ONT (Weak Signal)** | 🟡 Bulat | Kuning | Kecil | Signal lemah (-25 s/d -27 dBm) |
| **ONT (Offline/LOS)** | 🔴 Bulat | Merah | Kecil | Offline atau Loss of Signal |
| **ONT (Pending)** | ⚪ Bulat | Abu-abu | Kecil | Belum diaktivasi |

### Garis Koneksi (Polyline)

| Koneksi | Warna | Style | Keterangan |
|---|---|---|---|
| OLT → ODP (Backbone) | Biru tua | Solid, tebal (4px) | Kabel backbone utama |
| ODP → ONT (Drop) | Hijau | Solid, tipis (2px) | Kabel drop ke pelanggan |
| ODP → ONT (Offline) | Merah | Dashed, tipis (2px) | Pelanggan offline |

### Clustering (Zoom Out)
- Saat zoom out, ONT yang berdekatan di-cluster menjadi 1 marker dengan angka
- Contoh: cluster "45" berarti ada 45 ONT di area tersebut
- Warna cluster: hijau (semua online), kuning (ada yang weak), merah (ada yang offline)
- Klik cluster → zoom in ke area tersebut

---

## Label & Keterangan di Node (Custom Annotation)

Setiap node di peta bisa punya **label yang tampil langsung** (tanpa perlu klik) dan **keterangan detail** (custom fields):

### Label di Peta (Always Visible)

```
Zoom level tinggi (dekat):

  📡 OLT-01 Pusat                    ← label nama selalu tampil
     ZTE C320 • 245 ONT

  🔵 ODP-01-A                        ← label nama + info ringkas
     1:8 • 7/8 • Pool: 10.10.1.0/24

  🟢 Ahmad R.                        ← label nama pelanggan
     Pro 50M • -18 dBm

Zoom level rendah (jauh):
  📡 OLT-01                          ← hanya nama singkat
  🔵 ODP-01-A                        ← hanya nama
  (ONT di-cluster, label hidden)
```

### Setting Label per Tipe Node

Admin bisa atur informasi apa yang tampil di label:

```
╔══════════════════════════════════════════════════════════════╗
║  Settings > Peta > Label Node                                ║
║                                                              ║
║  Label OLT:                                                  ║
║  ☑ Nama OLT                                                 ║
║  ☑ Brand & Model                                            ║
║  ☑ Jumlah ONT                                               ║
║  ☐ IP Address                                               ║
║  ☐ Uptime                                                   ║
║                                                              ║
║  Label ODP:                                                  ║
║  ☑ Nama ODP                                                 ║
║  ☑ Tipe Splitter (1:8, 1:16)                                ║
║  ☑ Kapasitas (terpakai/total)                               ║
║  ☑ IP Pool / Subnet                                         ║
║  ☐ PON Port                                                 ║
║  ☐ VLAN                                                     ║
║  ☐ Catatan Custom                                           ║
║                                                              ║
║  Label ONT:                                                  ║
║  ☑ Nama Pelanggan                                           ║
║  ☑ Paket Internet                                           ║
║  ☐ Signal (dBm)                                             ║
║  ☐ ID Pelanggan                                             ║
║  ☐ IP Address                                               ║
║  ☐ SN ONT                                                   ║
║                                                              ║
║  Tampilkan label mulai zoom level: [15 ▼]                   ║
║  (semakin kecil = label tampil dari lebih jauh)              ║
║                                                              ║
║                                          [Simpan]            ║
╚══════════════════════════════════════════════════════════════╝
```

### Keterangan Custom per Node (Custom Fields)

Setiap node bisa punya **catatan/keterangan tambahan** yang diisi manual oleh admin/teknisi:

```
Klik ODP-01-A → Detail Panel → tab "Keterangan":

╔══════════════════════════════════════════════════════════════╗
║  Keterangan — ODP-01-A                                       ║
║                                                              ║
║  ┌─── Info Jaringan ─────────────────────────────────────┐   ║
║  │  IP Pool / Subnet    : [10.10.1.0/24_________]        │   ║
║  │  VLAN                : [100___]                        │   ║
║  │  Gateway             : [10.10.1.1__________]          │   ║
║  │  PON Port            : 0/1 (auto dari data OLT)       │   ║
║  │  Tipe Kabel Backbone : [Single Mode 12 Core__]        │   ║
║  │  Panjang Kabel       : [1.2 km] (auto dari polyline)  │   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║  ┌─── Info Fisik ────────────────────────────────────────┐   ║
║  │  Lokasi Detail       : [Tiang PLN No. 45, depan      │   ║
║  │                         masjid Al-Ikhlas RT 03/05]    │   ║
║  │  Tipe Tiang           : [Tiang PLN ▼]                 │   ║
║  │  Ketinggian           : [6 meter___]                  │   ║
║  │  Akses                : [Butuh tangga 4m__]           │   ║
║  │  Foto                 : [📎 Upload foto] 3 foto       │   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║  ┌─── Catatan Bebas ─────────────────────────────────────┐   ║
║  │  [ODP ini sering kena air hujan karena posisi miring. │   ║
║  │   Sudah dipasang pelindung tambahan 15 Apr 2026.      │   ║
║  │   Kabel core 5 sudah putus, pakai core 7 sebagai     │   ║
║  │   pengganti.]                                         │   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║  Terakhir diedit: Teknisi Andi, 20 Apr 2026 14:30           ║
║                                                              ║
║                                          [Simpan]            ║
╚══════════════════════════════════════════════════════════════╝
```

### Field Keterangan per Tipe Node

**OLT:**
| Field | Tipe | Keterangan |
|---|---|---|
| IP Management | Text | IP untuk akses management OLT |
| Uplink | Text | Info uplink ke backbone (misal: "FO 24 core ke POP Utama") |
| UPS | Text | Info UPS (misal: "APC 3000VA, backup 2 jam") |
| Ruangan | Text | Lokasi di dalam gedung |
| Foto | Upload | Foto OLT dan ruangan |
| Catatan | Textarea | Catatan bebas |

**ODP / Splitter:**
| Field | Tipe | Keterangan |
|---|---|---|
| IP Pool / Subnet | Text | Subnet yang dialokasikan untuk ODP ini |
| VLAN | Number | VLAN ID |
| Gateway | Text | IP gateway |
| Tipe Kabel Backbone | Text | Single mode, jumlah core |
| Panjang Kabel | Auto/Manual | Otomatis dari polyline, bisa override manual |
| Lokasi Detail | Text | Deskripsi lokasi fisik yang detail |
| Tipe Tiang | Dropdown | Tiang PLN / Tiang Telkom / Tiang Sendiri / Dinding Gedung |
| Ketinggian | Text | Ketinggian ODP dari tanah |
| Akses | Text | Cara akses (tangga, mobil crane, dll) |
| Foto | Upload | Foto ODP dan sekitarnya (max 5 foto) |
| Catatan | Textarea | Catatan bebas (riwayat perbaikan, masalah, dll) |

**ONT / Pelanggan:**
| Field | Tipe | Keterangan |
|---|---|---|
| IP Address | Auto | Dari DHCP/PPPoE (otomatis) |
| ODP Port | Number | Port berapa di ODP |
| Tipe Kabel Drop | Text | Misal: "FO 2 core, 50 meter" |
| Lokasi ONT | Text | Misal: "Di atas kulkas, ruang tamu" |
| Foto | Upload | Foto instalasi ONT |
| Catatan | Textarea | Catatan teknisi |

### Foto di Node

Setiap node bisa punya **foto** yang di-upload oleh teknisi:

```
┌──────────────────────────────────────────┐
│ 📷 Foto — ODP-01-A (3 foto)             │
│                                          │
│ ┌────────┐ ┌────────┐ ┌────────┐        │
│ │ Foto 1 │ │ Foto 2 │ │ Foto 3 │        │
│ │ ODP    │ │ Label  │ │ Kabel  │        │
│ │ tampak │ │ ODP    │ │ masuk  │        │
│ │ depan  │ │        │ │        │        │
│ └────────┘ └────────┘ └────────┘        │
│                                          │
│ [📎 Upload Foto Baru]                   │
│                                          │
│ Diupload oleh: Teknisi Andi             │
│ Tanggal: 15 Apr 2026                    │
└──────────────────────────────────────────┘
```

- Max 5 foto per node
- Foto di-compress otomatis (max 1 MB per foto)
- Berguna untuk dokumentasi instalasi dan troubleshooting
- Teknisi bisa upload langsung dari HP saat di lapangan

---

## Interaksi Peta

### Klik Node → Detail Panel

**Klik OLT:**
```
┌─────────────────────────┐
│ 📡 OLT-01 Pusat         │
│ ZTE C320 • 🟢 Online    │
│ PON: 8 port • ONT: 245  │
│ Alarm: 0                 │
│                          │
│ [Lihat Detail OLT]      │
│ [Lihat ONT di OLT ini]  │
└─────────────────────────┘
```

**Klik ODP:**
```
┌─────────────────────────┐
│ 🔵 ODP-01-A             │
│ Splitter 1:8 • 7/8 port │
│ Jl. Merdeka No. 5       │
│                          │
│ ONT: 🟢5 🟡1 🔴1        │
│                          │
│ [Lihat Detail ODP]      │
│ [Tambah ONT ke ODP ini] │
│ [Edit Lokasi]            │
└─────────────────────────┘
```

**Klik ONT:**
```
┌─────────────────────────┐
│ 🟢 Ahmad Rizki (PLG-001)│
│ Pro 50M • Signal: -18 dBm│
│ ONT: ZTEG12345678        │
│ ODP: ODP-01-A Port 3    │
│                          │
│ [Lihat Pelanggan]        │
│ [Lihat Detail ONT]      │
│ [Edit Lokasi]            │
└─────────────────────────┘
```

### Drawing Tools (Toolbar)

| Tool | Ikon | Fungsi |
|---|---|---|
| Search | 🔍 | Cari pelanggan/ODP/OLT di peta |
| Add Marker | 📍 | Tambah node baru (ODP/ONT) di peta |
| Draw Line | ✏️ | Gambar jalur kabel fiber |
| Measure | 📐 | Ukur jarak antar 2 titik |
| Delete | 🗑️ | Hapus node atau garis |
| Layer | Layer ▼ | Toggle layer: OLT, ODP, ONT, Kabel, Satellite |

### Search di Peta

```
╔══════════════════════════════════════════════════════════════╗
║  🔍 Cari di peta...                                         ║
║                                                              ║
║  Hasil:                                                      ║
║  👤 Ahmad Rizki (PLG-001) — Jl. Merdeka No. 10             ║
║  🔵 ODP-01-A — Jl. Merdeka No. 5                           ║
║  📡 OLT-01 Pusat — Gedung utama lt.1                       ║
║                                                              ║
║  Klik hasil → peta zoom & center ke lokasi                   ║
╚══════════════════════════════════════════════════════════════╝
```

- Search by: nama pelanggan, ID pelanggan, nama ODP, nama OLT, SN ONT, alamat
- Autocomplete real-time
- Klik hasil → peta zoom ke lokasi, marker di-highlight

---

## Layer Control

Admin bisa toggle visibility per layer:

| Layer | Default | Keterangan |
|---|---|---|
| OLT | ✅ Visible | Marker OLT |
| ODP / Splitter | ✅ Visible | Marker ODP |
| ONT (Online) | ✅ Visible | Marker ONT online |
| ONT (Offline) | ✅ Visible | Marker ONT offline |
| Kabel Backbone | ✅ Visible | Garis OLT → ODP |
| Kabel Drop | ❌ Hidden | Garis ODP → ONT (terlalu banyak, toggle manual) |
| Area / Wilayah | ❌ Hidden | Polygon area pelanggan |
| Satellite | ❌ Hidden | Tile satellite (Google/Bing) |
| Heatmap Signal | ❌ Hidden | Heatmap kualitas signal per area |

### Heatmap Signal Quality

```
┌──────────────────────────────────────────┐
│  Heatmap Signal Quality                  │
│                                          │
│  ████████ Signal bagus (-8 s/d -20 dBm) │
│  ████████ Signal sedang (-20 s/d -25)   │
│  ████████ Signal lemah (-25 s/d -27)    │
│  ████████ Signal kritis (< -27 dBm)     │
│                                          │
│  Berdasarkan rata-rata signal ONT        │
│  per area/cluster                        │
└──────────────────────────────────────────┘
```

- Overlay heatmap di atas peta
- Warna berdasarkan rata-rata signal ONT di area tersebut
- Membantu identifikasi area dengan kualitas jaringan buruk

---

## Tambah Node dari Peta

### Tambah ODP Baru

```
Klik tool 📍 → klik lokasi di peta → form muncul:

╔══════════════════════════════════════════════════════════════╗
║  Tambah ODP di Lokasi Ini                                    ║
║                                                              ║
║  Koordinat: -6.914744, 107.609810                            ║
║                                                              ║
║  Nama ODP *: [ODP-03-A_________]                            ║
║  Tipe Splitter *: ○ 1:4  ● 1:8  ○ 1:16  ○ 1:32            ║
║  OLT *: [OLT-01 Pusat ▼]                                   ║
║  PON Port *: [0/5 ▼]                                        ║
║  Alamat: [Tiang listrik depan RT 03/05]                     ║
║                                                              ║
║                    [Batal]  [Simpan ODP]                     ║
╚══════════════════════════════════════════════════════════════╝
```

### Gambar Jalur Kabel

```
Klik tool ✏️ → klik titik-titik di peta untuk gambar polyline:

  📡 OLT ──── titik 1 ──── titik 2 ──── 🔵 ODP

Setelah selesai gambar:
╔══════════════════════════════════════════════════════════════╗
║  Jalur Kabel Baru                                            ║
║                                                              ║
║  Dari: OLT-01 Pusat                                         ║
║  Ke: ODP-03-A                                                ║
║  Panjang: 1.2 km (dihitung otomatis)                        ║
║  Tipe *: ● Backbone  ○ Drop                                 ║
║  Jumlah Core: [12___]                                        ║
║  Keterangan: [Lewat jalan utama, tiang PLN]                 ║
║                                                              ║
║                    [Batal]  [Simpan Jalur]                   ║
╚══════════════════════════════════════════════════════════════╝
```

### Ukur Jarak

```
Klik tool 📐 → klik 2 titik di peta:

  Titik A ─────── 2.3 km ─────── Titik B

Berguna untuk estimasi:
  - Panjang kabel yang dibutuhkan
  - Apakah jarak masih dalam range GPON (~20 km)
  - Estimasi signal loss berdasarkan jarak
```

---

## Edit Lokasi Node

Semua node (OLT, ODP, ONT) bisa dipindah lokasinya di peta:

```
Klik node → [Edit Lokasi] → drag marker ke posisi baru → [Simpan]
```

- Koordinat otomatis terupdate di database
- Berguna saat:
  - Lokasi awal tidak akurat (GPS error saat input)
  - Pelanggan pindah rumah tapi masih di area yang sama
  - ODP dipindah ke tiang lain

---

## Topologi View (Non-Map)

Selain peta geografis, ada view topologi hierarki:

```
╔══════════════════════════════════════════════════════════════╗
║  Topologi Jaringan — OLT-01 Pusat                            ║
║                                                              ║
║  📡 OLT-01 Pusat (ZTE C320)                                 ║
║  ├── Port 0/1 (64 ONT)                                      ║
║  │   ├── 🔵 ODP-01-A (1:8, 7/8 terpakai)                   ║
║  │   │   ├── 🟢 Ahmad R. (PLG-001) -18 dBm                 ║
║  │   │   ├── 🟢 Budi S. (PLG-002) -22 dBm                  ║
║  │   │   ├── 🔴 Citra D. (PLG-003) LOS                     ║
║  │   │   ├── 🟢 Dewi A. (PLG-004) -19 dBm                  ║
║  │   │   └── ... (3 lagi)                                    ║
║  │   ├── 🔵 ODP-01-B (1:16, 10/16 terpakai)                ║
║  │   │   └── ...                                             ║
║  │   └── 🔵 ODP-01-C (1:8, 5/8 terpakai)                   ║
║  │       └── ...                                             ║
║  ├── Port 0/2 (58 ONT)                                      ║
║  │   └── ...                                                 ║
║  └── Port 0/3 (45 ONT)                                      ║
║      └── ...                                                 ║
║                                                              ║
║  [Peta Geografis]  [Topologi]     ← toggle view             ║
╚══════════════════════════════════════════════════════════════╝
```

- Tree view hierarki: OLT → PON Port → ODP → ONT
- Klik node → navigasi ke detail
- Warna sesuai status (hijau/kuning/merah)
- Bisa collapse/expand per level
- Berguna untuk melihat struktur jaringan tanpa perlu peta

---

## Data Sumber

Peta mengambil data dari modul lain:

| Data | Sumber | Keterangan |
|---|---|---|
| Lokasi OLT | Dokumen 09 (OLT) | Koordinat dari form tambah OLT |
| Lokasi ODP | Dokumen 09 (OLT) | Koordinat dari form tambah ODP |
| Lokasi ONT/Pelanggan | Dokumen 04 (Pelanggan) | Koordinat GPS wajib saat tambah pelanggan |
| Status ONT | Dokumen 09 (OLT) | Online/offline/signal dari SNMP monitoring |
| Jalur kabel | FTTH Mapping | Digambar manual oleh admin/teknisi di peta |
| Area/Wilayah | Dokumen 04 (Pelanggan) | Polygon area dari kelola area |

> **Catatan:** Koordinat GPS pelanggan sudah **wajib** di dokumen 04. Ini memastikan semua pelanggan bisa ditampilkan di peta.

---

## Export & Import Peta

### Import KML/KMZ

Untuk ISP yang sudah punya data mapping di Google Earth atau GIS lain:

```
╔══════════════════════════════════════════════════════════════╗
║  Import Data Peta                                            ║
║                                                              ║
║  Format yang didukung:                                       ║
║  ● KML (.kml)  ○ KMZ (.kmz)  ○ GeoJSON (.geojson)          ║
║                                                              ║
║  [📎 Upload file]                                            ║
║                                                              ║
║  Preview:                                                    ║
║  ┌──────────────────────────────────────────────────────────┐║
║  │  [PETA PREVIEW]                                          │║
║  │                                                          │║
║  │  Ditemukan:                                              │║
║  │  • 12 Placemark (titik) → bisa jadi ODP atau ONT        │║
║  │  • 5 LineString (garis) → jalur kabel                    │║
║  │  • 2 Polygon (area) → area/wilayah                       │║
║  └──────────────────────────────────────────────────────────┘║
║                                                              ║
║  Mapping Tipe:                                               ║
║  ┌──────────────────┬──────────────────────────────────────┐ ║
║  │ Data di KML      │ Import sebagai                       │ ║
║  ├──────────────────┼──────────────────────────────────────┤ ║
║  │ Placemark "ODP-" │ [ODP / Splitter ▼]                   │ ║
║  │ Placemark "PLG-" │ [ONT / Pelanggan ▼]                  │ ║
║  │ Placemark lainnya│ [ODP / Splitter ▼]                   │ ║
║  │ LineString       │ [Jalur Kabel ▼]                      │ ║
║  │ Polygon          │ [Area / Wilayah ▼]                   │ ║
║  └──────────────────┴──────────────────────────────────────┘ ║
║                                                              ║
║  ☑ Auto-match nama dengan pelanggan existing                 ║
║  ☐ Overwrite data lokasi yang sudah ada                      ║
║                                                              ║
║                    [Batal]  [Import 19 item]                 ║
╚══════════════════════════════════════════════════════════════╝
```

**Fitur Import:**
- Support **KML, KMZ, dan GeoJSON**
- KMZ otomatis di-extract (KMZ = ZIP berisi KML + gambar)
- Preview di peta sebelum import
- **Mapping tipe otomatis**: detect dari nama/folder di KML (misal folder "ODP" → import sebagai ODP)
- **Auto-match pelanggan**: jika nama di KML cocok dengan nama pelanggan di ISPBoss → link otomatis
- Bisa pilih overwrite atau skip jika data sudah ada
- Proses async untuk file besar (> 100 item)

### Export KML/KMZ

```
╔══════════════════════════════════════════════════════════════╗
║  Export Data Peta                                            ║
║                                                              ║
║  Format *                                                    ║
║  ● KML (.kml)  ○ KMZ (.kmz)  ○ GeoJSON (.geojson)          ║
║  ○ CSV (.csv)  ○ PDF (cetak peta)  ○ PNG (screenshot)       ║
║                                                              ║
║  Data yang di-export:                                        ║
║  ☑ OLT (3 titik)                                            ║
║  ☑ ODP / Splitter (25 titik)                                ║
║  ☑ ONT / Pelanggan (847 titik)                              ║
║  ☑ Jalur Kabel (15 garis)                                   ║
║  ☐ Area / Wilayah (8 polygon)                               ║
║                                                              ║
║  Opsi KML/KMZ:                                              ║
║  ☑ Sertakan ikon custom per tipe node                       ║
║  ☑ Sertakan info detail di description (nama, signal, paket)║
║  ☑ Organisasi dalam folder per tipe                         ║
║  ☐ Sertakan foto (hanya KMZ)                                ║
║                                                              ║
║                    [Batal]  [Export]                          ║
╚══════════════════════════════════════════════════════════════╝
```

**Fitur Export:**
- **KML**: bisa dibuka di Google Earth, Google Maps, QGIS
- **KMZ**: KML + ikon custom dalam 1 file ZIP (lebih kecil, lebih portable)
- **GeoJSON**: untuk developer/integrasi GIS
- Organisasi dalam **folder per tipe** (OLT, ODP, ONT, Kabel) di KML
- Description per node berisi: nama, tipe, status, signal, paket, alamat
- Ikon custom per tipe node (OLT = tower, ODP = kotak, ONT = bulat)
- Proses async untuk data besar

### Export Lainnya

| Format | Kegunaan |
|---|---|
| PNG/JPG | Screenshot area peta yang terlihat |
| CSV/Excel | Daftar semua node dengan koordinat (untuk spreadsheet) |
| PDF (A3/A4) | Cetak peta dengan legenda, untuk dokumentasi fisik |

---

## Reverse Geocoding (Klik Peta → Alamat Lengkap)

Klik di mana saja di peta untuk mendapatkan alamat lengkap secara otomatis:

```
Klik di peta (koordinat: -6.914744, 107.609810)
  │
  ▼
Reverse Geocoding API (Nominatim / OpenStreetMap):
  │
  ▼
┌──────────────────────────────────────────┐
│ 📍 Alamat Ditemukan                      │
│                                          │
│ Jl. Merdeka No. 10                       │
│ Kel. Sukamaju, Kec. Cimanggis           │
│ Kota Depok, Jawa Barat 16451            │
│                                          │
│ Koordinat: -6.914744, 107.609810         │
│                                          │
│ [Salin Alamat]  [Gunakan untuk:]         │
│   ○ Tambah ODP di sini                   │
│   ○ Update lokasi pelanggan              │
│   ○ Tandai titik saja                    │
└──────────────────────────────────────────┘
```

### Penggunaan Reverse Geocoding

| Konteks | Cara Kerja |
|---|---|
| **Tambah ODP dari peta** | Klik lokasi → alamat otomatis terisi di form ODP |
| **Tambah pelanggan** | Klik "📍 Pilih Map" di form pelanggan → klik peta → alamat + koordinat otomatis terisi |
| **Edit lokasi** | Drag marker ke posisi baru → alamat otomatis terupdate |
| **Survei lapangan** | Teknisi klik lokasi di peta → dapat alamat lengkap untuk navigasi |

### Provider Reverse Geocoding

| Provider | Biaya | Rate Limit | Keterangan |
|---|---|---|---|
| **Nominatim (OSM)** | Gratis | 1 req/detik | Default, cukup untuk mayoritas kasus |
| **Google Geocoding** | Berbayar | 50 req/detik | Lebih akurat untuk Indonesia, opsional |
| **Mapbox** | Freemium | 10 req/detik | Alternatif |

- Default: **Nominatim** (gratis, tidak perlu API key)
- Admin bisa switch ke Google Geocoding di Settings jika butuh akurasi lebih tinggi
- **Caching**: hasil geocoding di-cache 30 hari (koordinat yang sama → alamat yang sama)
- **Fallback**: jika Nominatim gagal → tampilkan koordinat saja tanpa alamat

---

## Offline Mode (Teknisi di Lapangan)

Teknisi sering bekerja di area tanpa sinyal internet. Peta harus tetap bisa dipakai offline:

### Download Area untuk Offline

```
╔══════════════════════════════════════════════════════════════╗
║  Download Peta Offline                                       ║
║                                                              ║
║  Pilih area yang mau di-download:                            ║
║  ┌──────────────────────────────────────────────────────────┐║
║  │  [PETA — gambar kotak area yang mau di-download]         │║
║  │                                                          │║
║  │  ┌─────────────────────┐                                 │║
║  │  │  Area terpilih      │                                 │║
║  │  │  (drag untuk resize)│                                 │║
║  │  └─────────────────────┘                                 │║
║  └──────────────────────────────────────────────────────────┘║
║                                                              ║
║  Zoom level: [12 ▼] sampai [18 ▼]                          ║
║  Estimasi ukuran: ~45 MB                                     ║
║                                                              ║
║  Data yang di-cache:                                         ║
║  ☑ Tile peta (OpenStreetMap)                                ║
║  ☑ Data node (OLT, ODP, ONT)                               ║
║  ☑ Jalur kabel                                              ║
║  ☑ Foto node (thumbnail saja)                               ║
║                                                              ║
║                    [Batal]  [Download]                        ║
╚══════════════════════════════════════════════════════════════╝
```

### Cara Kerja Offline

```
Sebelum ke lapangan:
  Teknisi download area peta + data node
  → Tersimpan di browser (IndexedDB / Service Worker cache)

Di lapangan (tanpa internet):
  → Peta tetap bisa dibuka (dari cache)
  → Data node tetap bisa dilihat
  → Bisa tambah/edit node (tersimpan lokal)
  → Bisa upload foto (tersimpan lokal)
  → Indikator: "⚡ Mode Offline"

Kembali online:
  → Auto-sync perubahan ke server
  → Upload foto yang tertunda
  → Resolve conflict jika ada (last-write-wins atau tanya user)
  → Indikator: "✅ Tersinkronisasi"
```

- Menggunakan **Service Worker** + **IndexedDB** untuk cache
- Max cache per area: **100 MB** (configurable)
- Cache expired setelah **7 hari** (harus re-download untuk data terbaru)
- Perubahan offline ditandai dengan ikon ⚡ sampai tersinkronisasi

---

## Navigasi & Lokasi Saya

### Tombol "Lokasi Saya"

```
Klik tombol 📍 (My Location) di toolbar peta:
  → Browser minta izin GPS
  → Marker biru berkedip muncul di posisi teknisi
  → Peta center ke posisi teknisi
  → Update posisi real-time (GPS tracking)
```

### Navigasi ke Node

```
Klik node (misal ODP-01-A) → Detail Panel:

┌─────────────────────────┐
│ 🔵 ODP-01-A             │
│ Jl. Merdeka No. 5       │
│                          │
│ 📍 Jarak dari saya: 1.2 km│
│                          │
│ [🗺️ Navigasi]           │
│   → Buka Google Maps     │
│   → Buka Waze            │
│   → Petunjuk arah di peta│
└─────────────────────────┘
```

- **Navigasi eksternal**: buka Google Maps atau Waze dengan koordinat tujuan (deep link)
- **Navigasi di peta**: tampilkan garis lurus dari posisi saya ke node tujuan + jarak
- **Jarak otomatis**: setiap node menampilkan jarak dari posisi teknisi saat ini

---

## Filter Peta Berdasarkan Status

Selain layer control, ada filter cepat berdasarkan status:

```
┌──────────────────────────────────────────────────────────────┐
│ Filter Peta:                                                  │
│                                                              │
│ Status ONT:  [Semua ▼]  [Online] [Offline] [Weak Signal]    │
│ Status Billing: [Semua ▼] [Aktif] [Isolir] [Pending]        │
│ Paket:       [Semua ▼]                                      │
│ Area:        [Semua ▼]                                      │
│ ODP:         [Semua ▼]                                      │
│                                                              │
│ Menampilkan: 87 dari 847 pelanggan                           │
│ [Reset Filter]                                               │
└──────────────────────────────────────────────────────────────┘
```

| Filter | Kegunaan |
|---|---|
| ONT Offline saja | Troubleshooting — lihat semua pelanggan yang bermasalah |
| ONT Weak Signal | Maintenance preventif — identifikasi sebelum putus |
| Billing Isolir | Penagihan lapangan — kunjungi pelanggan yang diisolir |
| Billing Pending | Verifikasi pemasangan baru |
| Per Paket | Analisis distribusi pelanggan per paket |
| Per ODP | Lihat semua pelanggan di 1 ODP |

---

## Estimasi Optical Loss Budget

Kalkulasi loss budget untuk perencanaan jalur fiber baru:

```
╔══════════════════════════════════════════════════════════════╗
║  Kalkulator Loss Budget                                      ║
║                                                              ║
║  ┌─── Parameter ─────────────────────────────────────────┐   ║
║  │  Jarak OLT → ODP (km)     : [1.2___] (auto dari peta)│   ║
║  │  Jarak ODP → ONT (km)     : [0.3___] (auto dari peta)│   ║
║  │  Jumlah Splitter           : [1___]                   │   ║
║  │  Tipe Splitter             : [1:8 ▼]                  │   ║
║  │  Jumlah Konektor           : [4___]                   │   ║
║  │  Jumlah Sambungan (Splice) : [2___]                   │   ║
║  │  SFP TX Power (dBm)       : [+3.0__] (auto dari OLT) │   ║
║  │  ONT Sensitivity (dBm)    : [-28.0_] (default GPON)   │   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║  ┌─── Perhitungan ───────────────────────────────────────┐   ║
║  │  Fiber loss (1.5 km × 0.35 dB/km)  : 0.53 dB         │   ║
║  │  Splitter loss (1:8)                : 10.5 dB         │   ║
║  │  Konektor loss (4 × 0.5 dB)        : 2.0 dB          │   ║
║  │  Splice loss (2 × 0.1 dB)          : 0.2 dB          │   ║
║  │  Safety margin                      : 3.0 dB          │   ║
║  │  ─────────────────────────────────────────────         │   ║
║  │  Total Loss                         : 16.23 dB        │   ║
║  │  Budget Tersedia (TX - Sensitivity) : 31.0 dB         │   ║
║  │  Sisa Margin                        : 14.77 dB        │   ║
║  │                                                       │   ║
║  │  Status: ✅ FEASIBLE (margin cukup)                   │   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║  Estimasi signal di ONT: -13.23 dBm (🟢 Normal)            ║
║                                                              ║
║                                          [Tutup]             ║
╚══════════════════════════════════════════════════════════════╝
```

### Parameter Loss Default
| Komponen | Loss | Keterangan |
|---|---|---|
| Fiber (per km) | 0.35 dB/km | Single mode 1310nm |
| Splitter 1:4 | 7.0 dB | |
| Splitter 1:8 | 10.5 dB | |
| Splitter 1:16 | 13.5 dB | |
| Splitter 1:32 | 17.0 dB | |
| Konektor (per buah) | 0.5 dB | SC/APC connector |
| Splice (per buah) | 0.1 dB | Fusion splice |
| Safety margin | 3.0 dB | Cadangan untuk aging & repair |

- Jarak bisa **auto-fill dari peta** (jika jalur kabel sudah digambar)
- SFP TX power bisa **auto-fill dari data OLT** (jika SFP monitoring aktif)
- Bisa diakses dari toolbar peta atau dari form provisioning ONT
- Membantu teknisi memutuskan apakah jalur baru feasible **sebelum** pasang kabel

---

## Riwayat Perubahan Peta

### Undo/Redo

- **Ctrl+Z** (undo) dan **Ctrl+Y** (redo) untuk aksi terakhir di peta
- Berlaku untuk: tambah/hapus/pindah node, gambar/hapus jalur kabel
- Buffer 20 aksi terakhir per session

### Riwayat per Node

```
┌──────────────────────────────────────────────────────────────────┐
│ Riwayat Perubahan — ODP-01-A                                     │
│                                                                  │
│ ┌──────────────────┬──────────────────────────────┬────────────┐ │
│ │ Waktu            │ Perubahan                    │ Oleh       │ │
│ ├──────────────────┼──────────────────────────────┼────────────┤ │
│ │ 28/04/26 14:30   │ Lokasi dipindah (drag)       │ Teknisi Andi│ │
│ │ 20/04/26 10:00   │ Foto ditambahkan (3 foto)    │ Teknisi Andi│ │
│ │ 15/04/26 09:00   │ IP Pool diubah ke 10.10.2.0  │ Admin Budi │ │
│ │ 01/04/26 08:00   │ ODP dibuat                   │ Admin Budi │ │
│ └──────────────────┴──────────────────────────────┴────────────┘ │
│                                                                  │
│ [Restore ke versi sebelumnya]                                    │
└──────────────────────────────────────────────────────────────────┘
```

### Soft Delete & Restore
- Node yang dihapus **tidak langsung hilang** — masuk ke "Trash" (soft delete)
- Bisa di-restore dalam **30 hari**
- Setelah 30 hari → permanent delete
- Admin bisa lihat daftar node terhapus di Settings > Peta > Trash

---

## Share & Embed Peta

### Share Link (Read-Only)

```
╔══════════════════════════════════════════════════════════════╗
║  Share Peta                                                  ║
║                                                              ║
║  Link: https://app.ispboss.id/map/share/abc123xyz            ║
║  [📋 Salin Link]                                            ║
║                                                              ║
║  Pengaturan:                                                 ║
║  Akses: ● Siapa saja dengan link  ○ Perlu password          ║
║  Expired: [7 hari ▼]  (atau: tidak expired)                 ║
║                                                              ║
║  Data yang ditampilkan:                                      ║
║  ☑ OLT                                                      ║
║  ☑ ODP / Splitter                                           ║
║  ☐ ONT / Pelanggan (sembunyikan data pelanggan)             ║
║  ☑ Jalur Kabel                                              ║
║  ☐ Signal / Status (sembunyikan data teknis)                ║
║                                                              ║
║                    [Batal]  [Buat Link]                      ║
╚══════════════════════════════════════════════════════════════╝
```

### Embed (iFrame)

```html
<iframe 
  src="https://app.ispboss.id/map/embed/abc123xyz" 
  width="800" height="600" 
  frameborder="0">
</iframe>
```

- Peta read-only (tidak bisa edit)
- Admin pilih data apa yang ditampilkan (bisa sembunyikan data pelanggan untuk privasi)
- Link bisa di-set expired (7 hari, 30 hari, atau tidak expired)
- Bisa dilindungi password
- Berguna untuk: presentasi ke investor, share ke partner, embed di website ISP

---

## Graceful Degradation

Jika modul OLT belum aktif:
- Peta tetap bisa dipakai untuk **plotting lokasi pelanggan** (dari koordinat GPS di dokumen 04)
- Marker ONT ditampilkan tanpa data signal (hanya lokasi)
- OLT dan ODP tidak ditampilkan
- Drawing tools tetap tersedia untuk perencanaan jalur kabel

Jika modul MikroTik belum aktif:
- Peta tetap bisa dipakai
- Router MikroTik tidak ditampilkan di peta

---

## Integrasi dengan Modul Lain

| Modul | Integrasi |
|---|---|
| **Pelanggan (04)** | Koordinat GPS pelanggan → marker ONT di peta. "Lihat di Peta" dari detail pelanggan |
| **Paket (05)** | Warna marker bisa dibedakan per paket (opsional) |
| **MikroTik (08)** | Marker router MikroTik di peta (opsional) |
| **OLT (09)** | Marker OLT, ODP, status ONT (signal, online/offline) |
| **Laporan (11)** | Heatmap signal quality, coverage report per area |

---

## Keputusan

| Keputusan | Detail |
|---|---|
| Map library | Leaflet.js + OpenStreetMap (gratis, open source) |
| React wrapper | react-leaflet untuk integrasi Next.js |
| Drawing tools | Leaflet.draw untuk gambar jalur kabel dan tambah node |
| Clustering | Leaflet.markercluster untuk grouping saat zoom out |
| Label node | ✅ Always-visible label di peta, configurable per tipe node (nama, IP pool, kapasitas, signal) |
| Keterangan custom | ✅ Custom fields per node: IP pool, VLAN, gateway, tipe kabel, lokasi detail, catatan bebas |
| Foto node | ✅ Max 5 foto per node, upload dari HP, auto-compress 1 MB |
| Layout desktop | Split view: peta (70%) + detail panel (30%) |
| Layout mobile | Full screen peta + bottom sheet detail + floating toolbar |
| Node types | 6 tipe: OLT, ODP, ONT Online, ONT Weak, ONT Offline, ONT Pending |
| Garis koneksi | Backbone (biru, tebal) dan Drop (hijau/merah, tipis) |
| Layer control | ✅ Toggle per layer: OLT, ODP, ONT, kabel, area, satellite, heatmap |
| Heatmap signal | ✅ Overlay heatmap kualitas signal per area |
| Search | ✅ Cari pelanggan, ODP, OLT, SN ONT, alamat → zoom ke lokasi |
| Drawing | ✅ Tambah ODP, gambar jalur kabel, ukur jarak dari peta |
| Edit lokasi | ✅ Drag & drop marker untuk update koordinat |
| Topologi view | ✅ Tree view hierarki OLT → PON → ODP → ONT (non-map) |
| Export | ✅ KML, KMZ, GeoJSON, CSV, PNG, PDF. Folder per tipe, ikon custom, description detail |
| Import | ✅ KML, KMZ, GeoJSON. Preview sebelum import, auto-match pelanggan, mapping tipe otomatis |
| Reverse geocoding | ✅ Klik peta → alamat lengkap otomatis. Nominatim (default, gratis), Google Geocoding (opsional) |
| Geocoding cache | ✅ Cache 30 hari, fallback ke koordinat jika gagal |
| Graceful degradation | ✅ Peta tetap bisa dipakai tanpa modul OLT/MikroTik |
| Offline mode | ✅ Download area peta + data node, edit offline, auto-sync saat online. Service Worker + IndexedDB |
| Navigasi | ✅ Lokasi saya (GPS), jarak ke node, navigasi ke Google Maps/Waze |
| Filter status | ✅ Filter by ONT status (online/offline/weak), billing status (aktif/isolir/pending), paket, area, ODP |
| Loss budget calculator | ✅ Kalkulasi optical loss (fiber, splitter, konektor, splice), auto-fill dari peta & OLT |
| Riwayat perubahan | ✅ Undo/redo (20 aksi), riwayat per node, soft delete + restore 30 hari |
| Share & embed | ✅ Share link read-only (expired, password), embed iframe, pilih data yang ditampilkan |
| Data sumber | Koordinat dari pelanggan (04), status dari OLT (09), jalur kabel dari FTTH mapping |