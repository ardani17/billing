# 04 — Manajemen Pelanggan

---

## Halaman Daftar Pelanggan (`/customers`)

### Layout Desktop

```
╔══════════════════════════════════════════════════════════════════════╗
║  Dashboard > Pelanggan                                              ║
║                                                                      ║
║  Pelanggan                                        [+ Tambah Pelanggan]║
║                                                                      ║
║  ┌─────────────────────────────────────────────────────────────────┐  ║
║  │ 🔍 Cari nama, ID, alamat, telepon...                           │  ║
║  ├─────────────────────────────────────────────────────────────────┤  ║
║  │ Filter: [Status ▼] [Paket ▼] [Area ▼] [Jatuh Tempo ▼] [Reset]  │  ║
║  └─────────────────────────────────────────────────────────────────┘  ║
║                                                                      ║
║  Menampilkan 847 pelanggan                    [Export ▼] [Import]    ║
║                                                                      ║
║  ┌────┬──────────┬───────────┬──────────┬────────┬────────┬───────┐  ║
║  │ No │ Nama     │ ID Plgn   │ Paket    │ Status │ Tagihan│ Aksi  │  ║
║  ├────┼──────────┼───────────┼──────────┼────────┼────────┼───────┤  ║
║  │ 1  │ Ahmad R. │ PLG-001   │ Pro 50M  │ 🟢Aktif│ Lunas  │ ⋯     │  ║
║  │ 2  │ Budi S.  │ PLG-002   │ Basic 10M│ 🔴Isolir│ Rp150rb│ ⋯     │  ║
║  │ 3  │ Citra D. │ PLG-003   │ Pro 50M  │ 🟢Aktif│ Lunas  │ ⋯     │  ║
║  │ 4  │ Dewi A.  │ PLG-004   │ Basic 10M│ 🟡Pending│Rp150rb│ ⋯     │  ║
║  │ 5  │ Eko P.   │ PLG-005   │ Ultra100M│ ⚫Berhenti│ -    │ ⋯     │  ║
║  └────┴──────────┴───────────┴──────────┴────────┴────────┴───────┘  ║
║                                                                      ║
║  ◀ 1 2 3 ... 85 ▶                          10 / 25 / 50 per halaman ║
╚══════════════════════════════════════════════════════════════════════╝
```

### Layout Mobile (Card List)

```
┌──────────────────────────┐
│ Ahmad Rizki        🟢Aktif│
│ PLG-001 • Pro 50Mbps     │
│ Area: RT 03/05 Sukamaju  │
│ Tagihan: Lunas     [⋯]  │
├──────────────────────────┤
│ Budi Santoso      🔴Isolir│
│ PLG-002 • Basic 10Mbps   │
│ Area: RT 01/02 Mekarjaya │
│ Tagihan: Rp 150.000 [⋯] │
└──────────────────────────┘
```

### Kolom Tabel
| Kolom | Deskripsi | Sortable |
|---|---|---|
| No | Nomor urut | - |
| Nama | Nama lengkap pelanggan | ✅ |
| ID Pelanggan | Format `PLG-001`, auto-increment per tenant | ✅ |
| Paket | Nama paket internet | ✅ |
| Status | Aktif, Isolir, Pending, Berhenti | ✅ |
| Area | Wilayah/group area pelanggan | ✅ |
| Tagihan | Status tagihan bulan ini | ✅ |
| Aksi | Dropdown: Detail, Edit, Isolir/Aktifkan, Hapus | - |

### Status Pelanggan
| Status | Warna | Arti |
|---|---|---|
| 🟢 Aktif | Green | Internet aktif, tagihan lunas |
| 🟡 Pending | Amber | Baru daftar, menunggu aktivasi |
| 🔴 Isolir | Red | Internet dimatikan karena tunggakan (redirect ke walled garden) |
| 🟣 Suspend | Purple | Koneksi dimatikan total setelah lewat batas toleransi (30+ hari tunggakan) |
| ⚫ Berhenti | Gray | Pelanggan berhenti berlangganan |

### Filter & Search
| Filter | Opsi |
|---|---|
| Search | Cari nama, ID, alamat, telepon (debounce 300ms) |
| Status | Semua, Aktif, Pending, Isolir, Suspend, Berhenti |
| Paket | Dropdown semua paket yang tersedia |
| Area | Dropdown semua area/wilayah |
| Jatuh Tempo | Semua, Hari ini, Minggu ini, Bulan ini, Sudah lewat |
| Reset | Hapus semua filter |

### Pagination
- Default 25 per halaman
- Opsi: 10 / 25 / 50
- Tampilkan total pelanggan

### Quick Stats (Di Atas Tabel)
```
[🟢 720 Aktif] [🟡 15 Pending] [🔴 87 Isolir] [🟣 5 Suspend] [⚫ 25 Berhenti]
```
- Klik salah satu → langsung filter tabel ke status tersebut
- Mempercepat kerja operator tanpa perlu buka dropdown filter

### Bulk Action (Aksi Massal)
Checkbox di setiap baris tabel. Saat ada yang dicentang, muncul toolbar:
```
┌──────────────────────────────────────────────────────────────┐
│ ☑ 12 dipilih   [Isolir Massal] [Kirim Notifikasi] [Export] [Hapus] │
└──────────────────────────────────────────────────────────────┘
```
| Bulk Action | Deskripsi |
|---|---|
| Isolir Massal | Suspend semua yang dipilih (trigger MikroTik) |
| Aktifkan Massal | Buka isolir semua yang dipilih |
| Kirim Notifikasi | Kirim pesan ke semua yang dipilih |
| Ganti Paket Massal | Pindahkan semua ke paket lain |
| Export Terpilih | Export hanya yang dicentang |
| Hapus Massal | Soft delete semua yang dipilih |
| Edit Massal | Update area, jatuh tempo, atau catatan untuk semua yang dipilih |

Semua bulk action butuh konfirmasi dialog.

---

## Form Tambah/Edit Pelanggan (`/customers/new` atau `/customers/:id/edit`)

### Layout

```
╔══════════════════════════════════════════════════════════════╗
║  Dashboard > Pelanggan > Tambah Pelanggan                    ║
║                                                              ║
║  Tambah Pelanggan Baru                                       ║
║                                                              ║
║  ┌─── Data Pribadi ──────────────────────────────────────┐   ║
║  │                                                       │   ║
║  │  Nama Lengkap *          No. Telepon / WA *           │   ║
║  │  [___________________]   [+62______________]          │   ║
║  │                                                       │   ║
║  │  Email                                                │   ║
║  │  [___________________]                                │   ║
║  │                                                       │   ║
║  │  Alamat Pemasangan *                                  │   ║
║  │  [___________________________________________]        │   ║
║  │                                                       │   ║
║  │  Area / Wilayah *                                     │   ║
║  │  [Pilih Area ▼] atau [+ Buat Area Baru]              │   ║
║  │                                                       │   ║
║  │  Koordinat GPS *                                      │   ║
║  │  [Lat: _________]  [Lng: _________]  [📍 Pilih Map]  │   ║
║  │  (klik "Pilih Map" untuk pin lokasi di peta)          │   ║
║  │                                                       │   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║  ┌─── Data Layanan ──────────────────────────────────────┐   ║
║  │                                                       │   ║
║  │  Paket Internet *        Tanggal Aktivasi *           │   ║
║  │  [Pilih Paket ▼]        [26/04/2026]                  │   ║
║  │                                                       │   ║
║  │  Tanggal Jatuh Tempo *   Metode Koneksi *             │   ║
║  │  [Setiap tanggal ▼]     [PPPoE ▼]                    │   ║
║  │  Opsi: PPPoE / Hotspot / DHCP Binding / Static        │   ║
║  │                                                       │   ║
║  │  Username PPPoE          Password PPPoE               │   ║
║  │  [___________________]   [___________________]        │   ║
║  │  [🔄 Auto-generate]                                   │   ║
║  │                                                       │   ║
║  │  MAC Address (untuk DHCP Binding)                     │   ║
║  │  [__:__:__:__:__:__]                                  │   ║
║  │                                                       │   ║
║  │  Router MikroTik         ODP / Port OLT               │   ║
║  │  [Pilih Router ▼]       [___________________]        │   ║
║  │                                                       │   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║  ┌─── Catatan ───────────────────────────────────────────┐   ║
║  │  [___________________________________________]        │   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║                              [Batal]  [Simpan Pelanggan]     ║
╚══════════════════════════════════════════════════════════════╝
```

### Field
| Field | Wajib | Keterangan |
|---|---|---|
| Nama Lengkap | ✅ | Min 3 karakter |
| No. Telepon/WA | ✅ | Format +62, untuk notifikasi. Unik **per tenant** (bukan global — satu nomor bisa jadi pelanggan di ISP berbeda) |
| Email | ❌ | Opsional |
| Alamat Pemasangan | ✅ | Alamat lengkap |
| Area / Wilayah | ✅ | Pilih dari dropdown atau buat baru |
| Koordinat GPS | ✅ | **Wajib** — untuk FTTH mapping & geotag pemasangan |
| Paket Internet | ✅ | Dropdown dari daftar paket |
| Tanggal Aktivasi | ✅ | Default: hari ini |
| Tanggal Jatuh Tempo | ✅ | Setiap tanggal berapa (1-28) |
| Metode Koneksi | ✅ | PPPoE / Hotspot / DHCP Binding / Static IP |
| Username PPPoE | Conditional | Wajib jika PPPoE, bisa auto-generate. Format default: `{nama-depan}-{id-pelanggan}` → `ahmad-plg001`. Tenant bisa custom format di settings |
| Password PPPoE | Conditional | Wajib jika PPPoE, bisa auto-generate |
| MAC Address | Conditional | Wajib jika DHCP Binding. Format: `AA:BB:CC:DD:EE:FF`. Untuk static lease di DHCP server MikroTik |
| Router MikroTik | ❌ | Pilih dari daftar router (dropdown, jika modul MikroTik aktif). Relasi ke Device Registry |
| ODP / Port OLT | ❌ | Pilih dari daftar ODP (dropdown, jika modul OLT aktif). Relasi ke Device Registry |
| Catatan | ❌ | Free text |

### Pilih Koordinat GPS via Map
Klik tombol "📍 Pilih Map" membuka modal peta:
```
┌──────────────────────────────────────────┐
│  Pilih Lokasi Pemasangan                 │
│  ┌──────────────────────────────────┐    │
│  │                                  │    │
│  │         [Peta Leaflet]           │    │
│  │                                  │    │
│  │            📍                    │    │
│  │                                  │    │
│  │                                  │    │
│  └──────────────────────────────────┘    │
│  Lat: -6.914744   Lng: 107.609810        │
│  🔍 Cari alamat...                       │
│                                          │
│                    [Batal]  [Pilih Lokasi]│
└──────────────────────────────────────────┘
```
- Klik di peta untuk pin lokasi
- Bisa search alamat (geocoding)
- Koordinat otomatis terisi di form

---

## Halaman Detail Pelanggan (`/customers/:id`)

```
╔══════════════════════════════════════════════════════════════════╗
║  Dashboard > Pelanggan > Ahmad Rizki                             ║
║                                                                  ║
║  ┌──────────────────────────────────────────────────────────┐    ║
║  │  👤 Ahmad Rizki                          🟢 Aktif        │    ║
║  │  PLG-001 • Pro 50Mbps • Area: RT 03/05 Sukamaju          │    ║
║  │  📱 +62 812-3456-7890  •  📧 ahmad@email.com             │    ║
║  │  📍 Jl. Merdeka No. 10, Kel. Sukamaju, Kec. Cimanggis    │    ║
║  │                                                          │    ║
║  │  [Edit]  [Isolir]  [Kirim Notifikasi]  [⋯ Lainnya]      │    ║
║  └──────────────────────────────────────────────────────────┘    ║
║                                                                  ║
║  ┌─ Tab ────────────────────────────────────────────────────┐    ║
║  │  [Ringkasan]  [Invoice]  [Pembayaran]  [Network]  [Log] │    ║
║  └──────────────────────────────────────────────────────────┘    ║
║                                                                  ║
║  TAB: Ringkasan                                                  ║
║  ┌──────────────┬──────────────┬──────────────┬──────────────┐   ║
║  │ Paket        │ Jatuh Tempo  │ Sisa Hari    │ Total Bayar  │   ║
║  │ Pro 50Mbps   │ Tgl 5/bulan  │ 9 hari lagi  │ Rp 2.1jt    │   ║
║  │ Rp 350.000   │              │              │ (6 bulan)    │   ║
║  └──────────────┴──────────────┴──────────────┴──────────────┘   ║
║  Kredit Saldo: Rp 50.000 (dari overpayment)                     ║
║                                                                  ║
║  Informasi Layanan:                                              ║
║  Metode: PPPoE • Username: ahmad-plg001                          ║
║  Router: MK-01 (192.168.1.1) • ODP: ODP-05-A Port 3             ║
║  Aktivasi: 15 Oktober 2025 • Berlangganan: 6 bulan              ║
║  Koordinat: -6.914744, 107.609810 [Lihat di Peta]               ║
║                                                                  ║
║  Riwayat Paket:                                                  ║
║  • Pro 50Mbps (sejak 26 Apr 2026) ← sekarang                    ║
║  • Basic 10Mbps (15 Okt 2025 — 25 Apr 2026)                     ║
║                                                                  ║
║  Catatan: Rumah warna biru, sebelah masjid                       ║
╚══════════════════════════════════════════════════════════════════╝
```

### Tab Detail
| Tab | Konten |
|---|---|
| Ringkasan | Info paket, jatuh tempo, total bayar, kredit saldo, info layanan, catatan |
| Invoice | Tabel riwayat invoice pelanggan ini (nomor, periode, jumlah, status) |
| Pembayaran | Tabel riwayat pembayaran (tanggal, jumlah, metode, bukti) |
| Layanan | Recurring item (sewa ONT, IP public, dll), diskon aktif, preferensi notifikasi |
| Network | Status PPPoE, traffic real-time, router info (jika MikroTik aktif) |
| Log | Riwayat perubahan: aktivasi, isolir, ganti paket, edit data, dll |

### Tab Network (jika MikroTik aktif)
```
┌──────────────────────────────────────────────────────────┐
│ Router: MK-01 (192.168.1.1)     Status PPPoE: 🟢 Online │
│ Username: ahmad-plg001          Uptime: 3 hari 5 jam     │
│ IP Address: 10.10.1.25          MAC: AA:BB:CC:DD:EE:FF   │
│ Download: 45.2 Mbps             Upload: 12.8 Mbps        │
│ Bandwidth Limit: 50M/50M                                 │
│                                                          │
│ Traffic Hari Ini:                                        │
│ ▁▂▃▅▆█▇▅▆▇█▆▅▃▂▁▂▃▅▆█▇▅                               │
│ 00:00        06:00        12:00        18:00    sekarang │
│                                                          │
│ [Disconnect] [Reset Counter] [Ubah Bandwidth]            │
└──────────────────────────────────────────────────────────┘
```

### Tab Log (Audit Trail)
```
┌──────────────────────────────────────────────────────────┐
│ 26 Apr 2026 14:30 — Admin mengubah paket dari Basic 10M │
│                      ke Pro 50M                          │
│ 20 Apr 2026 09:15 — Sistem melakukan isolir otomatis     │
│                      (tunggakan 15 hari)                 │
│ 05 Apr 2026 10:00 — Invoice INV-2026-04-001 digenerate   │
│ 05 Mar 2026 11:22 — Pembayaran Rp 350.000 via Xendit    │
│ 15 Okt 2025 08:00 — Pelanggan diaktivasi oleh Admin     │
└──────────────────────────────────────────────────────────┘
```

---

## Fitur Area / Wilayah

Pelanggan bisa dikelompokkan per area untuk memudahkan manajemen.

### Halaman Kelola Area (`/customers/areas`)
```
┌──────────────────────────────────────────────────────┐
│ Area / Wilayah                      [+ Tambah Area]  │
│                                                      │
│ ┌──────────────┬──────────────┬──────────┬────────┐  │
│ │ Nama Area    │ Jumlah Plgn  │ ODP      │ Aksi   │  │
│ ├──────────────┼──────────────┼──────────┼────────┤  │
│ │ RT 03/05     │ 45 pelanggan │ ODP-05-A │ ⋯      │  │
│ │ Sukamaju     │              │          │        │  │
│ ├──────────────┼──────────────┼──────────┼────────┤  │
│ │ RT 01/02     │ 32 pelanggan │ ODP-01-B │ ⋯      │  │
│ │ Mekarjaya    │              │          │        │  │
│ └──────────────┴──────────────┴──────────┴────────┘  │
└──────────────────────────────────────────────────────┘
```

### Data Area
| Field | Wajib | Keterangan |
|---|---|---|
| Nama Area | ✅ | Contoh: "RT 03/05 Sukamaju" |
| Deskripsi | ❌ | Keterangan tambahan |
| ODP Terkait | ❌ | Link ke ODP di FTTH mapping |
| Koordinat Pusat | ❌ | Untuk centering peta area |

### Kegunaan Area
- Filter pelanggan per area di tabel
- Grouping di laporan (pendapatan per area)
- Referensi saat assign ODP di FTTH mapping
- Memudahkan teknisi cari pelanggan di lapangan

---

## Import / Export Pelanggan

### Export
- **Format:** CSV dan Excel (.xlsx)
- **Opsi export:**
  - Semua pelanggan
  - Pelanggan terfilter (sesuai filter aktif)
  - Pilih kolom yang mau di-export
- **Proses:** Async (background job) → download link saat selesai

### Import
- **Format:** CSV dan Excel (.xlsx)
- **Alur import:**
```
Upload file
  → Preview data (tabel 10 baris pertama)
  → Mapping kolom (cocokkan kolom file dengan field sistem)
  → Validasi (tampilkan error per baris jika ada)
  → Konfirmasi import
  → Proses async (background job)
  → Tampilkan hasil: X berhasil, Y gagal (download error log)
```

### Template Import
Sediakan template CSV/Excel yang bisa didownload:
```csv
nama,telepon,email,alamat,area,paket,metode_koneksi,username_pppoe,password_pppoe,latitude,longitude,tanggal_aktivasi,tanggal_jatuh_tempo
Ahmad Rizki,+6281234567890,ahmad@email.com,Jl. Merdeka No. 10,RT 03/05 Sukamaju,Pro 50M,pppoe,ahmad-001,pass123,-6.914744,107.609810,2025-10-15,5
```

### Validasi Import
| Validasi | Aksi |
|---|---|
| Nama kosong | Error, skip baris |
| Telepon duplikat | Warning, tetap import (bisa jadi update) |
| Paket tidak ditemukan | Error, skip baris |
| Format koordinat salah | Warning, import tanpa koordinat |
| Format tanggal salah | Error, skip baris |

---

## Aksi Pelanggan

### Dari Tabel (Menu ⋯)
| Aksi | Deskripsi | Konfirmasi |
|---|---|---|
| Detail | Buka halaman detail | Tidak |
| Edit | Buka form edit | Tidak |
| Isolir | Suspend internet (trigger MikroTik) | Ya — "Yakin isolir Ahmad Rizki?" |
| Aktifkan | Buka isolir (trigger MikroTik) | Ya |
| Kirim Notifikasi | Kirim pesan WA/SMS/Email | Ya — pilih template |
| Hapus | Soft delete pelanggan | Ya — ketik nama pelanggan untuk konfirmasi (mencegah salah hapus) |

### Dari Detail Pelanggan
Sama seperti di atas, plus:
| Aksi | Deskripsi |
|---|---|
| Ganti Paket | Ubah paket internet (trigger update bandwidth di MikroTik) |
| Lihat di Peta | Buka FTTH map centered di lokasi pelanggan |
| Reset PPPoE | Reset koneksi PPPoE di MikroTik |

---

## Keputusan

| Keputusan | Detail |
|---|---|
| ID Pelanggan | Format `PLG-001`, auto-increment per tenant |
| Area/Wilayah | ✅ Ada — grouping pelanggan per area |
| Import/Export | ✅ CSV & Excel, async processing |
| Foto KTP | ❌ Tidak perlu |
| Koordinat GPS | ✅ **Wajib** — untuk FTTH mapping & geotag pemasangan |
| Breadcrumb | ✅ Ada — Dashboard > Pelanggan > Nama |
| Soft delete | ✅ Data tidak benar-benar dihapus, bisa di-restore |
| Bulk action | ✅ Isolir massal, notifikasi massal, ganti paket massal |
| Quick stats | ✅ Ringkasan status di atas tabel, klik untuk filter |
| Telepon unik | Per tenant (bukan global) |
| Username PPPoE format | Default `{nama-depan}-{id}`, customizable di settings |
| Konfirmasi hapus | Ketik nama pelanggan (seperti GitHub delete repo) |
| Riwayat paket | ✅ Tampil di tab Ringkasan |
| Status Suspend | ✅ Ditambahkan — koneksi dimatikan total setelah 30+ hari tunggakan |
| Kredit saldo | ✅ Tampil di tab Ringkasan — dari overpayment, downgrade, credit note |
| Tab Layanan | ✅ Recurring item, diskon aktif, preferensi notifikasi per pelanggan |
