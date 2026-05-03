# 05 — Manajemen Paket Internet

---

## Jenis Paket

Ada 2 jenis paket yang berbeda:

| Jenis | Tipe Pelanggan | Billing | Contoh |
|---|---|---|---|
| **Paket PPPoE/Static** | Pelanggan tetap (rumahan/kantor) | Bulanan | Pro 50M — Rp 350.000/bulan |
| **Paket Hotspot/Voucher** | End-user via reseller | Per voucher (harian/jam) | 1 Hari 5M — Rp 3.000 |

---

## Halaman Daftar Paket (`/packages`)

```
╔══════════════════════════════════════════════════════════════════════╗
║  Dashboard > Paket Internet                                         ║
║                                                                      ║
║  Paket Internet                                    [+ Tambah Paket]  ║
║                                                                      ║
║  [PPPoE/Static]  [Hotspot/Voucher]     ← Tab jenis paket            ║
║                                                                      ║
║  ┌─────────────────────────────────────────────────────────────────┐  ║
║  │ 🔍 Cari nama paket...        Filter: [Semua Status ▼] [Reset] │  ║
║  └─────────────────────────────────────────────────────────────────┘  ║
║                                                                      ║
║  TAB: PPPoE/Static                                                   ║
║  ┌──────────────┬────────┬────────┬────────┬──────────┬──────┬────┐  ║
║  │ Nama Paket   │Download│Upload  │ Harga  │ Pelanggan│Status│Aksi│  ║
║  ├──────────────┼────────┼────────┼────────┼──────────┼──────┼────┤  ║
║  │ Basic 10M    │ 10 Mbps│ 5 Mbps │ Rp150rb│ 320 plgn │🟢Aktif│ ⋯ │  ║
║  │ Pro 50M      │ 50 Mbps│ 25 Mbps│ Rp350rb│ 412 plgn │🟢Aktif│ ⋯ │  ║
║  │ Ultra 100M   │100 Mbps│ 50 Mbps│ Rp750rb│ 87 plgn  │🟢Aktif│ ⋯ │  ║
║  └──────────────┴────────┴────────┴────────┴──────────┴──────┴────┘  ║
║                                                                      ║
║  TAB: Hotspot/Voucher                                                ║
║  ┌──────────────┬────────┬────────┬────────┬──────────┬──────┬────┐  ║
║  │ Nama Paket   │Download│Upload  │Harga   │Harga     │Durasi│Aksi│  ║
║  │              │        │        │Jual    │Reseller  │      │    │  ║
║  ├──────────────┼────────┼────────┼────────┼──────────┼──────┼────┤  ║
║  │ 1 Hari 5M    │ 5 Mbps │ 3 Mbps │ Rp3.000│ Rp2.000  │1 hari│ ⋯ │  ║
║  │ 3 Hari 10M   │ 10 Mbps│ 5 Mbps │ Rp8.000│ Rp5.500  │3 hari│ ⋯ │  ║
║  │ 7 Hari 10M   │ 10 Mbps│ 5 Mbps │Rp15.000│ Rp11.000 │7 hari│ ⋯ │  ║
║  └──────────────┴────────┴────────┴────────┴──────────┴──────┴────┘  ║
╚══════════════════════════════════════════════════════════════════════╝
```

### Mobile Card List
```
PPPoE:
┌──────────────────────────┐
│ Pro 50M            �Aktif│
│ ↓50 Mbps  ↑25 Mbps      │
│ Rp 350.000/bulan         │
│ 412 pelanggan      [⋯]  │
└──────────────────────────┘

Voucher:
┌──────────────────────────┐
│ 1 Hari 5M          🟢Aktif│
│ ↓5 Mbps  ↑3 Mbps        │
│ Jual: Rp3.000            │
│ Reseller: Rp2.000  [⋯]  │
└──────────────────────────┘
```

---

## Form Tambah Paket PPPoE/Static (`/packages/new?type=pppoe`)

```
╔══════════════════════════════════════════════════════════════╗
║  Dashboard > Paket Internet > Tambah Paket PPPoE             ║
║                                                              ║
║  ┌─── Informasi Paket ───────────────────────────────────┐   ║
║  │  Nama Paket *            Harga per Bulan (Rp) *       │   ║
║  │  [___________________]   [___________________]        │   ║
║  │  Deskripsi                                            │   ║
║  │  [___________________________________________]        │   ║
║  │  Biaya Pasang (Rp)  — sekali bayar di awal            │   ║
║  │  [___________________]  (0 jika gratis)               │   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║  ┌─── Bandwidth ─────────────────────────────────────────┐   ║
║  │  Download (Mbps) *       Upload (Mbps) *              │   ║
║  │  [___________________]   [___________________]        │   ║
║  │                                                       │   ║
║  │  Tipe Bandwidth *                                     │   ║
║  │  ○ Dedicated (bandwidth dijamin)                      │   ║
║  │  ● Shared / Up-to (bandwidth maksimal)                │   ║
║  │                                                       │   ║
║  │  ☐ Aktifkan Burst                                     │   ║
║  │  Burst Download    Burst Upload     Burst Threshold   │   ║
║  │  [_______ Mbps]    [_______ Mbps]   [_______ Mbps]   │   ║
║  │  Burst Time [_______ detik]                           │   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║  ┌─── Quota ─────────────────────────────────────────────┐   ║
║  │  Tipe Quota *                                         │   ║
║  │  ● Unlimited (tanpa batas)                            │   ║
║  │  ○ Quota Bulanan                                      │   ║
║  │  ○ FUP (Fair Usage Policy)                            │   ║
║  │                                                       │   ║
║  │  (jika Quota Bulanan):                                │   ║
║  │  Quota (GB) [_____]  Setelah habis:                   │   ║
║  │    ○ Turunkan ke [___] Mbps                           │   ║
║  │    ○ Matikan koneksi                                  │   ║
║  │                                                       │   ║
║  │  (jika FUP):                                          │   ║
║  │  Batas FUP (GB) [_____]  Setelah FUP:                 │   ║
║  │    Turunkan ke [___] Mbps                             │   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║  ┌─── MikroTik Profile (jika modul aktif) ───────────────┐   ║
║  │  Nama Profile MikroTik                                │   ║
║  │  [___________________] (auto dari nama paket)         │   ║
║  │  Address Pool          Parent Queue (opsional)        │   ║
║  │  [Pilih Pool ▼]       [Pilih Queue ▼]                │   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║  Status: ● Aktif  ○ Nonaktif                                 ║
║                              [Batal]  [Simpan Paket]         ║
╚══════════════════════════════════════════════════════════════╝
```

### Field Paket PPPoE
| Field | Wajib | Keterangan |
|---|---|---|
| Nama Paket | ✅ | Unik per tenant |
| Deskripsi | ❌ | Penjelasan paket |
| Harga per Bulan | ✅ | Dalam Rupiah |
| Biaya Pasang | ❌ | Sekali bayar di awal, default 0 (gratis). Harga custom per pelanggan saat aktivasi |
| Download (Mbps) | ✅ | Bandwidth download |
| Upload (Mbps) | ✅ | Bandwidth upload |
| Tipe Bandwidth | ✅ | Dedicated / Shared (Up-to) |
| Burst Download | ❌ | MikroTik burst feature |
| Burst Upload | ❌ | MikroTik burst feature |
| Burst Threshold | ❌ | Batas sebelum burst berhenti |
| Burst Time | ❌ | Durasi burst (detik) |
| Tipe Quota | ✅ | Unlimited / Quota Bulanan / FUP |
| Quota (GB) | Conditional | Wajib jika tipe = Quota/FUP |
| Aksi setelah habis | Conditional | Turunkan speed / matikan |
| Nama Profile MikroTik | ❌ | Auto-generate dari nama paket. Hidden jika modul MikroTik belum aktif |
| Address Pool | ❌ | IP pool. Hidden jika modul MikroTik belum aktif |
| Parent Queue | ❌ | Queue hierarchy. Hidden jika modul MikroTik belum aktif |
| Status | ✅ | Aktif / Nonaktif |

---

## Form Tambah Paket Hotspot/Voucher (`/packages/new?type=voucher`)

```
╔══════════════════════════════════════════════════════════════╗
║  Dashboard > Paket Internet > Tambah Paket Voucher           ║
║                                                              ║
║  ┌─── Informasi Paket ───────────────────────────────────┐   ║
║  │  Nama Paket *                                         │   ║
║  │  [___________________]                                │   ║
║  │  Deskripsi                                            │   ║
║  │  [___________________________________________]        │   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║  ┌─── Harga ─────────────────────────────────────────────┐   ║
║  │  Harga Jual (Rp) *      Harga Reseller (Rp) *        │   ║
║  │  [___________________]   [___________________]        │   ║
║  │  Margin reseller: Rp 1.000 (otomatis dihitung)        │   ║
║  │  ⚠️ Harga reseller harus lebih kecil dari harga jual  │   ║
║  │  Minimum margin: Rp 500                               │   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║  ┌─── Durasi & Bandwidth ────────────────────────────────┐   ║
║  │  Durasi *                                             │   ║
║  │  [___] ○ Jam  ○ Hari  ○ Minggu  ○ Bulan               │   ║
║  │                                                       │   ║
║  │  Download (Mbps) *       Upload (Mbps) *              │   ║
║  │  [___________________]   [___________________]        │   ║
║  │                                                       │   ║
║  │  ☐ Aktifkan Burst (sama seperti PPPoE)                │   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║  ┌─── Quota ─────────────────────────────────────────────┐   ║
║  │  ● Unlimited  ○ Quota (MB/GB)                         │   ║
║  │  Quota [_____] ○ MB ○ GB                              │   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║  ┌─── MikroTik Hotspot Profile (jika modul aktif) ───────┐   ║
║  │  Nama Profile Hotspot                                 │   ║
║  │  [___________________] (auto dari nama paket)         │   ║
║  │  Shared Users (max device bersamaan)                   │   ║
║  │  [___] (default: 1)                                   │   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║  Status: ● Aktif  ○ Nonaktif                                 ║
║                              [Batal]  [Simpan Paket]         ║
╚══════════════════════════════════════════════════════════════╝
```

### Field Paket Voucher
| Field | Wajib | Keterangan |
|---|---|---|
| Nama Paket | ✅ | Contoh: "1 Hari 5M" |
| Deskripsi | ❌ | Penjelasan |
| Harga Jual | ✅ | Harga end-user |
| Harga Reseller | ✅ | Harga beli reseller. **Wajib < Harga Jual, minimum margin Rp 500** |
| Durasi | ✅ | Angka + satuan (jam/hari/minggu/bulan) |
| Download (Mbps) | ✅ | Bandwidth download |
| Upload (Mbps) | ✅ | Bandwidth upload |
| Burst | ❌ | Sama seperti PPPoE |
| Tipe Quota | ✅ | Unlimited / Quota |
| Quota | Conditional | Dalam MB atau GB |
| Nama Profile Hotspot | ❌ | Hidden jika modul MikroTik belum aktif |
| Shared Users | ❌ | Max device bersamaan, default 1 |
| Status | ✅ | Aktif / Nonaktif |

---

## Manajemen Voucher (`/vouchers`)

### Generate Voucher

```
╔══════════════════════════════════════════════════════════════╗
║  Dashboard > Voucher > Generate                              ║
║                                                              ║
║  Generate Voucher Baru                                       ║
║                                                              ║
║  Paket Voucher *                                             ║
║  [Pilih Paket ▼]                                            ║
║                                                              ║
║  Jumlah Voucher *                                            ║
║  [___] voucher                                               ║
║                                                              ║
║  Format Kode Voucher *                                       ║
║  ○ Angka saja         (contoh: 847291)                       ║
║  ○ Huruf saja         (contoh: ABCDEF)                       ║
║  ● Gabungan           (contoh: AB12CD)                       ║
║                                                              ║
║  Panjang Kode *                                              ║
║  [6] karakter   (min 6, max 16)                              ║
║                                                              ║
║  Prefix (opsional)                                           ║
║  [____]  (contoh: "ISP-" → ISP-AB12CD)                      ║
║                                                              ║
║  Preview: ISP-AB12CD, ISP-XY34ZW, ISP-MN56PQ                ║
║                                                              ║
║                    [Batal]  [Generate 50 Voucher]            ║
╚══════════════════════════════════════════════════════════════╝
```

### Batasan Generate Voucher
- **Maksimal 500 voucher per batch** (generate langsung, sinkron)
- Di atas 500: proses **async** via background job, admin mendapat notifikasi setelah selesai
- Mekanisme **collision avoidance**: jika kode yang di-generate sudah ada, sistem retry otomatis (max 3x per kode). Jika tetap gagal, skip dan laporkan jumlah yang gagal.
- Minimum panjang kode: **6 karakter** (untuk mengurangi risiko duplikat)

### Daftar Voucher (`/vouchers`)

```
┌─────────────────────────────────────────────────────────────────┐
│ Voucher                              [Generate Voucher]         │
│                                                                 │
│ Filter: [Semua Paket ▼] [Semua Status ▼] [Semua Reseller ▼]   │
│                                                                 │
│ ┌──────────┬──────────┬──────────┬──────────┬────────┬───────┐  │
│ │ Kode     │ Paket    │ Reseller │ Status   │ Dipakai│ Aksi  │  │
│ ├──────────┼──────────┼──────────┼──────────┼────────┼───────┤  │
│ │ ISP-AB12 │ 1Hari 5M │ -        │ 🟢Tersedia│ -      │ ⋯     │  │
│ │ ISP-CD34 │ 1Hari 5M │ Toko Adi │ 🔵Terjual │ -      │ ⋯     │  │
│ │ ISP-EF56 │ 3Hari 10M│ Toko Adi │ 🟡Aktif  │ 2/3 hari│ ⋯    │  │
│ │ ISP-GH78 │ 1Hari 5M │ Toko Budi│ ⚫Expired │ Selesai│ ⋯     │  │
│ └──────────┴──────────┴──────────┴──────────┴────────┴───────┘  │
└─────────────────────────────────────────────────────────────────┘
```

### Status Voucher
| Status | Warna | Arti |
|---|---|---|
| 🟢 Tersedia | Green | Belum dijual, siap dipakai |
| 🔵 Terjual | Blue | Sudah dibeli reseller, belum dipakai end-user |
| 🟡 Aktif | Amber | Sedang dipakai (durasi berjalan) |
| ⚫ Selesai | Gray | Durasi habis |
| 🔴 Expired | Red | Masa berlaku habis sebelum dipakai (saldo dikembalikan ke reseller) |
| ❌ Void | Dark | Dibatalkan oleh admin |

### Masa Berlaku Voucher (Sebelum Dipakai)
Voucher yang sudah dibeli reseller memiliki **masa berlaku 90 hari** sejak tanggal pembelian. Jika dalam 90 hari voucher belum dipakai oleh end-user:
- Status berubah menjadi 🔴 **Expired**
- **Saldo dikembalikan otomatis** ke reseller (refund penuh)
- Proses expired dijalankan oleh background job (cron harian)
- Admin bisa mengubah masa berlaku default di pengaturan tenant

> **Catatan:** Masa berlaku ini hanya berlaku untuk voucher yang belum dipakai. Voucher yang sudah aktif (sedang dipakai end-user) tetap berjalan sesuai durasi paket.

### Snapshot Harga Voucher
Saat reseller membeli voucher, harga **di-snapshot pada saat pembelian**:
- Jika admin mengubah harga paket setelahnya, voucher yang sudah dibeli **tidak terpengaruh**
- Harga yang tercatat di voucher = harga saat transaksi pembelian
- Ini berlaku untuk harga jual maupun harga reseller

### Print Voucher (PDF)
- Pilih voucher yang mau dicetak (checkbox)
- Generate PDF dengan layout voucher kecil (bisa dipotong)
- Format per voucher:
```
┌─────────────────────────┐
│  ISPBoss WiFi            │
│  ─────────────────────── │
│  Kode: ISP-AB12CD        │
│  Paket: 1 Hari 5Mbps     │
│  Harga: Rp 3.000         │
│  Berlaku s/d: 25 Jul 2026│
│  ─────────────────────── │
│  Hubungi: 0812-xxxx-xxxx │
└─────────────────────────┘
```
- 8-12 voucher per halaman A4
- Branding tenant (logo, nama ISP) di setiap voucher

### Bulk Action Voucher (Admin)
| Aksi | Deskripsi | Syarat |
|---|---|---|
| Bulk Print | Cetak beberapa voucher sekaligus ke PDF | Pilih via checkbox |
| Bulk Void | Batalkan voucher yang belum terjual | Status = Tersedia |
| Bulk Assign | Assign voucher ke reseller tertentu | Status = Tersedia |
| Export CSV | Download daftar voucher ke CSV | Semua status |

### Audit Trail Voucher
Setiap voucher memiliki log lifecycle lengkap:
```
ISP-AB12CD — Audit Trail:
┌──────────────────┬──────────────────┬──────────────┐
│ Waktu            │ Aksi             │ Oleh         │
├──────────────────┼──────────────────┼──────────────┤
│ 2026-04-20 10:00 │ Generated        │ Admin Budi   │
│ 2026-04-20 14:30 │ Sold to reseller │ Toko Adi     │
│ 2026-04-22 09:15 │ Used by end-user │ MAC: xx:xx   │
│ 2026-04-23 09:15 │ Expired (durasi) │ System       │
└──────────────────┴──────────────────┴──────────────┘
```
- Log disimpan untuk keperluan rekonsiliasi keuangan
- Tidak bisa dihapus (append-only)

---

## Manajemen Reseller (`/resellers`)

### Daftar Reseller

```
╔══════════════════════════════════════════════════════════════════╗
║  Dashboard > Reseller                                            ║
║                                                                  ║
║  Reseller                                    [+ Tambah Reseller] ║
║                                                                  ║
║  ┌──────────────┬──────────────┬──────────┬──────────┬────────┐  ║
║  │ Nama         │ Telepon      │ Saldo    │ Terjual  │ Aksi   │  ║
║  ├──────────────┼──────────────┼──────────┼──────────┼────────┤  ║
║  │ Toko Adi     │ 0812-xxx     │ Rp150.000│ 245 vcr  │ ⋯      │  ║
║  │ Warnet Budi  │ 0813-xxx     │ Rp 50.000│ 120 vcr  │ ⋯      │  ║
║  │ Cafe Citra   │ 0857-xxx     │ Rp 0     │ 89 vcr   │ ⋯      │  ║
║  └──────────────┴──────────────┴──────────┴──────────┴────────┘  ║
╚══════════════════════════════════════════════════════════════════╝
```

### Form Tambah Reseller
| Field | Wajib | Keterangan |
|---|---|---|
| Nama Reseller | ✅ | Nama toko/orang |
| No. Telepon/WA | ✅ | Untuk kontak & login |
| Email | ❌ | Opsional |
| Alamat | ❌ | Lokasi reseller |
| Password | ✅ | Untuk login dashboard reseller |
| Saldo Awal | ❌ | Default 0 |
| Limit Pembelian Harian | ❌ | Max pembelian voucher per hari (0 = tanpa batas). Mencegah penyalahgunaan jika akun diretas |

### Status Reseller
| Status | Arti | Dampak |
|---|---|---|
| 🟢 Aktif | Beroperasi normal | Bisa login, beli voucher, print |
| 🟡 Suspended | Ditangguhkan sementara | Tidak bisa beli voucher baru. Voucher existing tetap berlaku. Tidak bisa login dashboard |
| ⚫ Nonaktif | Berhenti kerjasama | Tidak bisa login. Voucher tersedia di-void. Saldo bisa di-refund manual oleh admin |

### Aksi Reseller
| Aksi | Deskripsi | Konfirmasi |
|---|---|---|
| Edit | Buka form edit data reseller | Tidak |
| Suspend | Tangguhkan sementara | Ya |
| Aktifkan | Aktifkan kembali dari suspend | Tidak |
| Nonaktifkan | Berhenti kerjasama permanen | Ya — ketik nama reseller |
| Reset Password | Kirim password baru via WA | Ya |

### Detail Reseller
Tab: Ringkasan, Voucher, Transaksi, Deposit

### Top-Up Saldo Reseller
**Manual (oleh admin):**
```
Top-Up Saldo — Toko Adi
Saldo saat ini: Rp 150.000
Jumlah top-up: [Rp ___________]
Catatan: [Transfer BCA 15:30]
[Top-Up]
```

**Otomatis (via payment gateway):**
- Reseller bisa top-up sendiri dari dashboard reseller
- Pilih nominal → bayar via Xendit (VA, QRIS, e-wallet)
- Saldo otomatis bertambah setelah pembayaran dikonfirmasi

---

## Dashboard Reseller (`app.ispboss.id/reseller`)

Reseller punya login dan dashboard sendiri yang terpisah dari dashboard admin.

```
╔══════════════════════════════════════════════════════════════╗
║  ┌──────────┬────────────────────────────────────────────┐   ║
║  │ SIDEBAR  │  TOPBAR                                    │   ║
║  │ (mini)   │  Toko Adi          Saldo: Rp 150.000  👤  │   ║
║  │          ├────────────────────────────────────────────┤   ║
║  │ 🏠 Home  │                                            │   ║
║  │ 🎫 Beli  │  ┌────────────┬────────────┬────────────┐  │   ║
║  │ 📋 Voucher│  │ Saldo      │ Terjual    │ Voucher    │  │   ║
║  │ 💰 Deposit│  │ Rp 150.000 │ Hari Ini:12│ Tersedia:45│  │   ║
║  │ 📊 Riwayat│  └────────────┴────────────┴────────────┘  │   ║
║  │          │                                            │   ║
║  │          │  Beli Voucher Cepat:                        │   ║
║  │          │  ┌────────────┬────────┬──────┬──────────┐  │   ║
║  │          │  │ Paket      │ Harga  │ Qty  │ Aksi     │  │   ║
║  │          │  │ 1 Hari 5M  │ Rp2.000│ [__] │ [Beli]   │  │   ║
║  │          │  │ 3 Hari 10M │ Rp5.500│ [__] │ [Beli]   │  │   ║
║  │          │  │ 7 Hari 10M │Rp11.000│ [__] │ [Beli]   │  │   ║
║  │          │  └────────────┴────────┴──────┴──────────┘  │   ║
║  │          │                                            │   ║
║  │          │  Voucher Tersedia:                          │   ║
║  │          │  ┌──────────┬──────────┬──────────┐        │   ║
║  │          │  │ ISP-AB12 │ 1Hari 5M │ [Print]  │        │   ║
║  │          │  │ ISP-CD34 │ 1Hari 5M │ [Print]  │        │   ║
║  │          │  │ ISP-EF56 │ 3Hari 10M│ [Print]  │        │   ║
║  │          │  └──────────┴──────────┴──────────┘        │   ║
║  │          │  [Print Semua]                              │   ║
║  └──────────┴────────────────────────────────────────────┘   ║
╚══════════════════════════════════════════════════════════════╝
```

### Menu Dashboard Reseller
| Menu | Fungsi |
|---|---|
| Home | Ringkasan saldo, terjual hari ini, voucher tersedia |
| Beli Voucher | Beli voucher baru (potong saldo) |
| Voucher Saya | List semua voucher (tersedia, terjual, aktif, selesai) + print |
| Deposit | Top-up saldo (manual request / payment gateway) |
| Riwayat | Riwayat transaksi (beli voucher, deposit, dll) |

### Alur Beli Voucher (Reseller)
```
Reseller pilih paket + jumlah
  → Cek status reseller aktif?
      → Tidak: "Akun Anda ditangguhkan, hubungi admin"
  → Cek limit pembelian harian?
      → Melebihi: "Batas pembelian harian tercapai"
  → Cek saldo cukup?
      → Tidak: "Saldo tidak cukup, silakan deposit"
      → Ya: potong saldo → generate voucher code
             → Snapshot harga saat pembelian
  → Voucher muncul di "Voucher Saya"
  → Reseller print PDF → jual ke end-user
```

### Keamanan Dashboard Reseller
- Login via **No. Telepon + Password**
- **OTP via WhatsApp** untuk login dari device baru (opsional, bisa diaktifkan admin)
- **Session management**: reseller bisa lihat device aktif dan logout dari semua device
- **Auto-logout** setelah 24 jam tidak aktif
- **Rate limiting** login: max 5 percobaan gagal → lock 15 menit

---

## Graceful Degradation (Modul MikroTik Belum Aktif)

Semua field terkait MikroTik **hidden** jika modul belum aktif:
- Profile MikroTik → hidden
- Address Pool → hidden
- Parent Queue → hidden
- Burst → hidden
- Hotspot Profile → hidden
- Shared Users → hidden

Paket tetap bisa dibuat dan dipakai untuk billing. Saat modul MikroTik diaktifkan nanti, admin bisa edit paket untuk menambahkan konfigurasi MikroTik.

---

## Upgrade/Downgrade Paket (Referensi ke Billing)

Pelanggan PPPoE/Static bisa pindah paket (upgrade/downgrade):
- **Proses**: Admin pilih pelanggan → ganti paket → sistem hitung prorate
- **Prorate billing**: dihitung proporsional berdasarkan sisa hari di periode berjalan
- **Efek langsung**: bandwidth profile di MikroTik langsung diupdate (jika modul aktif)
- **Detail implementasi prorate** → lihat dokumen **06 — Billing & Invoice**

> ⚠️ Fitur ini bergantung pada modul billing. Detail perhitungan prorate, generate invoice selisih, dan alur pembayaran dibahas di dokumen billing.

---

## Aksi Paket

| Aksi | Deskripsi | Konfirmasi |
|---|---|---|
| Edit | Buka form edit | Tidak |
| Nonaktifkan | Paket tidak muncul saat tambah pelanggan baru. Pelanggan existing tetap jalan | Ya |
| Aktifkan | Paket muncul kembali | Tidak |
| Hapus | Hanya bisa jika 0 pelanggan yang pakai. Jika masih ada → tampilkan "Nonaktifkan saja" | Ya — ketik nama paket |
| Duplikat | Buat paket baru dengan data copy dari paket ini | Tidak |

---

## Keputusan

| Keputusan | Detail |
|---|---|
| Jenis paket | 2 jenis: PPPoE/Static (bulanan) dan Hotspot/Voucher (durasi) |
| Harga reseller | ✅ Ada — harga beli reseller lebih murah dari harga jual. **Validasi: reseller < jual, min margin Rp 500** |
| Biaya pasang | ✅ Field terpisah, sekali bayar, harga custom per pelanggan |
| Quota | 3 opsi: Unlimited, Quota Bulanan, FUP |
| Burst | ✅ Integrasikan semua fitur burst MikroTik |
| Paket nonaktif | Pelanggan existing tetap jalan, paket tidak muncul untuk pelanggan baru |
| Voucher code format | Customizable: angka/huruf/gabungan, panjang **6-16** (min 6), prefix opsional |
| Voucher collision | Retry otomatis max 3x per kode, skip & laporkan jika gagal |
| Generate limit | Max 500 per batch (sinkron). Di atas 500 → async + notifikasi |
| Print voucher | ✅ PDF, 8-12 per halaman A4, branding tenant, **tanggal berlaku** |
| Voucher masa berlaku | ✅ **90 hari** sejak dibeli reseller. Expired → saldo refund otomatis. Configurable per tenant |
| Snapshot harga | ✅ Harga di-snapshot saat pembelian. Perubahan harga paket tidak mempengaruhi voucher yang sudah dibeli |
| MikroTik fields | Hidden jika modul belum aktif (graceful degradation) |
| Manajemen reseller | ✅ Ada — login sendiri, saldo, beli voucher, print. **Status: Aktif/Suspended/Nonaktif** |
| Keamanan reseller | OTP WA (opsional), session management, auto-logout 24 jam, rate limiting login |
| Limit pembelian reseller | ✅ Opsional — admin bisa set max pembelian harian per reseller |
| Deposit reseller | Manual (admin top-up) + otomatis (payment gateway) |
| Bulk action voucher | ✅ Bulk print, void, assign, export CSV |
| Audit trail voucher | ✅ Log lifecycle lengkap (append-only), untuk rekonsiliasi keuangan |
| Upgrade/downgrade paket | ✅ Ada — prorate billing, detail di dokumen 06 (Billing) |
