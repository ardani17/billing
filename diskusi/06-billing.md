# 06 — Billing & Invoice

---

## Konsep Billing

ISPBoss menangani 2 model billing yang berbeda sesuai jenis paket:

| Model | Jenis Paket | Mekanisme |
|---|---|---|
| **Billing Bulanan** | PPPoE/Static | Invoice otomatis setiap bulan, bayar manual/gateway |
| **Billing Voucher** | Hotspot/Voucher | Prepaid via saldo reseller, tidak ada invoice |

> Dokumen ini fokus pada **Billing Bulanan** untuk pelanggan PPPoE/Static. Billing voucher sudah dibahas di dokumen 05 (Manajemen Paket).

---

## Siklus Billing Bulanan

```
Tanggal Generate Invoice (configurable per tenant, default: H-5 jatuh tempo)
  │
  ▼
Invoice Digenerate (status: Belum Bayar)
  │
  ├── Notifikasi dikirim ke pelanggan (WA/SMS/Email)
  │
  ▼
Tanggal Jatuh Tempo (sesuai setting per pelanggan, tgl 1-28)
  │
  ├── Belum bayar? → Reminder bertingkat (lihat jadwal di bawah)
  │
  ▼
Grace Period (configurable, default: 7 hari setelah jatuh tempo)
  │
  ├── Masih belum bayar?
  │     → Isolir otomatis (redirect ke walled garden)
  │     → Notifikasi isolir dikirim
  │
  ▼
Batas Toleransi (configurable, default: 30 hari setelah jatuh tempo)
  │
  ├── Masih belum bayar?
  │     → Status pelanggan: Suspend
  │     → Koneksi dimatikan total (bukan redirect)
  │     → Notifikasi suspend dikirim
  │
  ▼
Pembayaran Diterima (kapan saja dalam siklus)
  │
  ├── Invoice → Lunas
  ├── Jika sedang isolir → Buka isolir otomatis
  ├── Jika sedang suspend → Aktifkan kembali
  ├── Notifikasi konfirmasi pembayaran
  └── Reset quota (jika paket ada quota)
```

---

## Reminder Bertingkat (Configurable)

Jadwal pengiriman notifikasi otomatis untuk mengurangi tunggakan:

| Waktu | Tipe | Pesan |
|---|---|---|
| H-5 (saat invoice terbit) | Invoice Baru | "Invoice bulan {periode} sudah terbit. Total: Rp {jumlah}" |
| H-1 (sehari sebelum jatuh tempo) | Reminder | "Besok jatuh tempo. Segera bayar tagihan Rp {jumlah}" |
| H+1 (sehari setelah jatuh tempo) | Peringatan | "Tagihan sudah lewat jatuh tempo. Bayar sebelum {tanggal_isolir}" |
| H+3 | Peringatan Keras | "Peringatan terakhir! Internet akan diisolir dalam {sisa_hari} hari" |
| H+7 (saat isolir) | Notifikasi Isolir | "Internet Anda diisolir karena tunggakan. Bayar untuk mengaktifkan kembali" |

- Jadwal reminder **configurable per tenant** di Settings > Billing > Reminder
- Admin bisa tambah/hapus/ubah jadwal reminder
- Setiap reminder bisa diaktifkan/nonaktifkan secara individual
- Channel pengiriman mengikuti setting notifikasi tenant (WA/SMS/Email)
- Template pesan bisa dikustomisasi per tenant (lihat dokumen 07)

---

## Konfigurasi Billing per Tenant

Diatur di **Settings > Billing** (lihat dokumen 12):

| Setting | Default | Keterangan |
|---|---|---|
| Tanggal Generate Invoice | H-5 jatuh tempo | Berapa hari sebelum jatuh tempo invoice digenerate |
| Grace Period | 7 hari | Toleransi setelah jatuh tempo sebelum isolir |
| Batas Toleransi | 30 hari | Setelah ini, pelanggan di-suspend total |
| Auto-Isolir | Aktif | Otomatis isolir setelah grace period |
| Auto-Buka Isolir | Aktif | Otomatis buka isolir setelah pembayaran |
| Denda Keterlambatan | Nonaktif | Opsional, nominal tetap atau persentase |
| Pajak/PPN | Nonaktif | Opsional, persentase (default 11%) |
| Nomor Invoice Prefix | INV | Format: {PREFIX}-{YYYY}-{MM}-{SEQ} |
| Metode Pembayaran Aktif | Manual | Manual, Xendit, Midtrans (bisa multiple) |
| Payment Link Expiry | 7 hari | Masa berlaku payment link setelah generate |
| Tagihan Pelanggan Baru | Prorate | Prorate bulan pertama atau bulan penuh |
| Perhitungan Hari per Bulan | 30 hari tetap | Selalu 30 hari untuk simplicity prorate |
| Timezone Cron Job | WIB (UTC+7) | Timezone untuk semua cron job billing. Configurable: WIB/WITA/WIT |


---

## Halaman Daftar Invoice (`/invoices`)

### Layout Desktop

```
╔══════════════════════════════════════════════════════════════════════════╗
║  Dashboard > Invoice                                                     ║
║                                                                          ║
║  Invoice                                          [+ Buat Invoice Manual]║
║                                                                          ║
║  ┌────────────┬────────────┬────────────┬────────────┐                   ║
║  │ 📄 Total   │ ✅ Lunas    │ ⏳ Belum    │ 🔴 Terlambat│                   ║
║  │ 847        │ 720        │ 87         │ 40         │                   ║
║  │Rp294.5jt   │Rp252.0jt   │Rp30.5jt    │Rp12.0jt    │                   ║
║  └────────────┴────────────┴────────────┴────────────┘                   ║
║                                                                          ║
║  ┌───────────────────────────────────────────────────────────────────┐    ║
║  │ 🔍 Cari no invoice, nama pelanggan...                             │    ║
║  ├───────────────────────────────────────────────────────────────────┤    ║
║  │ Filter: [Status ▼] [Periode ▼] [Paket ▼] [Area ▼] [Reset]       │    ║
║  └───────────────────────────────────────────────────────────────────┘    ║
║                                                                          ║
║  ┌──────────────┬───────────┬──────────┬─────────┬────────┬──────┬────┐  ║
║  │ No Invoice   │ Pelanggan │ Periode  │ Jumlah  │ Status │ Bayar│Aksi│  ║
║  ├──────────────┼───────────┼──────────┼─────────┼────────┼──────┼────┤  ║
║  │ INV-2026-04  │ Ahmad R.  │ Apr 2026 │ Rp350rb │ ✅Lunas │ Xendit│ ⋯ │  ║
║  │ -001         │ PLG-001   │          │         │        │      │    │  ║
║  ├──────────────┼───────────┼──────────┼─────────┼────────┼──────┼────┤  ║
║  │ INV-2026-04  │ Budi S.   │ Apr 2026 │ Rp150rb │ 🔴Telat │ -    │ ⋯ │  ║
║  │ -002         │ PLG-002   │          │ +Rp10rb │ 15 hari│      │    │  ║
║  ├──────────────┼───────────┼──────────┼─────────┼────────┼──────┼────┤  ║
║  │ INV-2026-04  │ Citra D.  │ Apr 2026 │ Rp350rb │ ⏳Belum │ -    │ ⋯ │  ║
║  │ -003         │ PLG-003   │          │         │ 5 hari │      │    │  ║
║  └──────────────┴───────────┴──────────┴─────────┴────────┴──────┴────┘  ║
║                                                                          ║
║  ◀ 1 2 3 ... 85 ▶                            10 / 25 / 50 per halaman   ║
╚══════════════════════════════════════════════════════════════════════════╝
```

### Layout Mobile (Card List)

```
┌──────────────────────────────┐
│ INV-2026-04-001       ✅Lunas│
│ Ahmad Rizki • PLG-001        │
│ Periode: April 2026          │
│ Rp 350.000        via Xendit │
├──────────────────────────────┤
│ INV-2026-04-002    🔴Terlambat│
│ Budi Santoso • PLG-002       │
│ Periode: April 2026          │
│ Rp 150.000 + Denda Rp 10.000│
│ Terlambat 15 hari      [⋯]  │
└──────────────────────────────┘
```

### Status Invoice
| Status | Warna | Arti |
|---|---|---|
| ⏳ Belum Bayar | Amber | Invoice sudah terbit, belum jatuh tempo |
| 🔴 Terlambat | Red | Sudah lewat jatuh tempo, belum bayar |
| ✅ Lunas | Green | Sudah dibayar penuh |
| 💰 Bayar Sebagian | Blue | Sudah bayar tapi belum lunas |
| ❌ Batal | Gray | Invoice dibatalkan oleh admin |
| 🔄 Prorate | Purple | Invoice prorate (upgrade/downgrade paket) |

### Filter & Search
| Filter | Opsi |
|---|---|
| Search | Cari no invoice, nama pelanggan, ID pelanggan |
| Status | Semua, Belum Bayar, Terlambat, Lunas, Bayar Sebagian, Batal |
| Periode | Bulan & tahun (dropdown) |
| Paket | Dropdown semua paket PPPoE |
| Area | Dropdown semua area |
| Reset | Hapus semua filter |


---

## Detail Invoice (`/invoices/:id`)

```
╔══════════════════════════════════════════════════════════════════╗
║  Dashboard > Invoice > INV-2026-04-001                           ║
║                                                                  ║
║  ┌──────────────────────────────────────────────────────────┐    ║
║  │  INVOICE                                                  │    ║
║  │  No: INV-2026-04-001              Status: ✅ Lunas        │    ║
║  │                                                          │    ║
║  │  ┌─── Dari ──────────────┬─── Untuk ─────────────────┐  │    ║
║  │  │ ISPBoss Net           │ Ahmad Rizki                │  │    ║
║  │  │ Jl. Raya No. 1       │ PLG-001                    │  │    ║
║  │  │ Telp: 0812-xxx       │ Jl. Merdeka No. 10         │  │    ║
║  │  │                       │ Telp: 0812-345-6789       │  │    ║
║  │  └───────────────────────┴───────────────────────────┘  │    ║
║  │                                                          │    ║
║  │  Periode: April 2026                                     │    ║
║  │  Jatuh Tempo: 5 April 2026                               │    ║
║  │  Tanggal Bayar: 3 April 2026                             │    ║
║  │                                                          │    ║
║  │  ┌──────────────────────────┬─────────┬────────────────┐ │    ║
║  │  │ Item                     │ Qty     │ Jumlah         │ │    ║
║  │  ├──────────────────────────┼─────────┼────────────────┤ │    ║
║  │  │ Paket Pro 50M (Apr 2026)│ 1 bulan │ Rp 350.000     │ │    ║
║  │  ├──────────────────────────┼─────────┼────────────────┤ │    ║
║  │  │ Subtotal                 │         │ Rp 350.000     │ │    ║
║  │  │ PPN 11%                  │         │ Rp  38.500     │ │    ║
║  │  │ Denda keterlambatan      │         │ Rp       0     │ │    ║
║  │  ├──────────────────────────┼─────────┼────────────────┤ │    ║
║  │  │ **TOTAL**                │         │ **Rp 388.500** │ │    ║
║  │  └──────────────────────────┴─────────┴────────────────┘ │    ║
║  │                                                          │    ║
║  │  Riwayat Pembayaran:                                     │    ║
║  │  • 3 Apr 2026 — Rp 388.500 via Xendit (VA BCA)          │    ║
║  │                                                          │    ║
║  │  [Download PDF]  [Kirim ke Pelanggan]  [Cetak]           │    ║
║  └──────────────────────────────────────────────────────────┘    ║
║                                                                  ║
║  [Edit Invoice]  [Batalkan Invoice]  [Catat Pembayaran]          ║
╚══════════════════════════════════════════════════════════════════╝
```

### Item Invoice
Invoice bisa berisi beberapa item (line items):

| Tipe Item | Contoh | Kapan Muncul |
|---|---|---|
| Tagihan Bulanan | Paket Pro 50M — Rp 350.000 | Setiap bulan (otomatis) |
| Biaya Pasang | Biaya Pasang — Rp 500.000 | Invoice pertama saja (jika ada) |
| Prorate Upgrade | Selisih Pro→Ultra (15 hari) — Rp 200.000 | Saat upgrade paket |
| Prorate Credit | Kredit Basic→Pro (15 hari) — -Rp 75.000 | Saat downgrade paket |
| Denda | Denda keterlambatan — Rp 10.000 | Jika setting denda aktif |
| PPN | PPN 11% — Rp 38.500 | Jika setting pajak aktif |
| Item Custom | Biaya tambahan kabel — Rp 50.000 | Ditambahkan manual oleh admin |

### Nomor Invoice
- Format: `{PREFIX}-{YYYY}-{MM}-{SEQ}`
- Contoh: `INV-2026-04-001`
- Prefix configurable per tenant (default: INV)
- Sequence auto-increment per bulan per tenant
- Unik per tenant


---

## Generate Invoice Otomatis

### Proses Generate (Background Job)

```
Cron job harian (jam 00:01)
  │
  ▼
Cari semua pelanggan aktif yang:
  - Jatuh tempo dalam H-{generate_days} hari
  - Belum punya invoice untuk periode ini
  │
  ▼
Untuk setiap pelanggan:
  ├── Buat invoice dengan item: tagihan bulanan
  ├── Tambahkan PPN (jika setting aktif)
  ├── Tambahkan denda (jika ada tunggakan sebelumnya & setting aktif)
  ├── Simpan invoice (status: Belum Bayar)
  ├── Kirim notifikasi ke pelanggan (WA/SMS/Email)
  └── Jika payment gateway aktif → generate payment link
```

### Invoice Manual
Admin bisa buat invoice manual untuk kasus khusus:
- Biaya pasang terpisah
- Tagihan tambahan (kabel, perangkat, dll)
- Invoice custom untuk pelanggan tertentu

```
╔══════════════════════════════════════════════════════════════╗
║  Buat Invoice Manual                                         ║
║                                                              ║
║  Pelanggan *                                                 ║
║  [Cari pelanggan... ▼]                                      ║
║                                                              ║
║  Jatuh Tempo *                                               ║
║  [dd/mm/yyyy]                                                ║
║                                                              ║
║  Item Invoice:                                               ║
║  ┌──────────────────────────┬─────────┬──────────┬────────┐  ║
║  │ Deskripsi                │ Qty     │ Harga    │ Aksi   │  ║
║  ├──────────────────────────┼─────────┼──────────┼────────┤  ║
║  │ Biaya pasang baru        │ 1       │ Rp500.000│ 🗑️     │  ║
║  │ Kabel tambahan 50m       │ 1       │ Rp 75.000│ 🗑️     │  ║
║  └──────────────────────────┴─────────┴──────────┴────────┘  ║
║  [+ Tambah Item]                                             ║
║                                                              ║
║  Subtotal:  Rp 575.000                                       ║
║  PPN 11%:   Rp  63.250                                       ║
║  Total:     Rp 638.250                                       ║
║                                                              ║
║  Catatan (opsional):                                         ║
║  [___________________________________________]               ║
║                                                              ║
║                    [Batal]  [Simpan & Kirim]  [Simpan Draft] ║
╚══════════════════════════════════════════════════════════════╝
```


---

## Pembayaran

### Metode Pembayaran

| Metode | Tipe | Keterangan |
|---|---|---|
| **Manual (Tunai)** | Offline | Admin/kasir catat pembayaran tunai |
| **Manual (Transfer)** | Offline | Admin/kasir catat transfer bank, bisa upload bukti |
| **Xendit** | Online | VA, QRIS, e-wallet, kartu kredit |
| **Midtrans** | Online | VA, QRIS, e-wallet, kartu kredit |

> Tenant bisa aktifkan lebih dari satu payment gateway sekaligus.

### Catat Pembayaran Manual

```
╔══════════════════════════════════════════════════════════════╗
║  Catat Pembayaran — INV-2026-04-002                          ║
║                                                              ║
║  Pelanggan: Budi Santoso (PLG-002)                           ║
║  Total Tagihan: Rp 160.000 (termasuk denda Rp 10.000)       ║
║  Sudah Dibayar: Rp 0                                         ║
║  Sisa: Rp 160.000                                            ║
║                                                              ║
║  Jumlah Bayar *                                              ║
║  [Rp ___________]  [Bayar Penuh: Rp 160.000]                ║
║                                                              ║
║  Metode *                                                    ║
║  ○ Tunai  ○ Transfer Bank  ○ Lainnya                         ║
║                                                              ║
║  Tanggal Bayar *                                             ║
║  [dd/mm/yyyy]  (default: hari ini)                           ║
║                                                              ║
║  Bukti Transfer (opsional)                                   ║
║  [📎 Upload gambar]                                          ║
║                                                              ║
║  Catatan                                                     ║
║  [Transfer BCA jam 15:30]                                    ║
║                                                              ║
║                              [Batal]  [Simpan Pembayaran]    ║
╚══════════════════════════════════════════════════════════════╝
```

### Pembayaran Sebagian (Partial Payment)
- Pelanggan boleh bayar sebagian dari total tagihan
- Status invoice berubah ke 💰 **Bayar Sebagian**
- Sisa tagihan tetap tercatat
- Bisa bayar lagi sampai lunas
- Isolir **tetap berlaku** sampai lunas penuh (configurable: admin bisa pilih buka isolir saat bayar sebagian)

### Overpayment (Kelebihan Bayar)
Jika pelanggan membayar lebih dari total tagihan:
- Kelebihan otomatis menjadi **kredit saldo pelanggan**
- Kredit dipotong dari invoice bulan berikutnya
- Ditampilkan di detail pelanggan: "Kredit: Rp {jumlah}"
- Admin bisa refund manual jika pelanggan minta
- Contoh: Tagihan Rp 350.000, bayar Rp 400.000 → kredit Rp 50.000
- Invoice bulan depan: Rp 350.000 - Rp 50.000 = Rp 300.000

### Multi-Invoice Tunggakan
Pelanggan bisa punya lebih dari 1 invoice terlambat. Aturan pembayaran:
- Pembayaran otomatis dialokasikan ke **invoice terlama dulu** (FIFO)
- Admin bisa override dan pilih invoice tertentu saat catat pembayaran manual
- Tombol **"Bayar Semua Tunggakan"** untuk lunasi semua invoice sekaligus
- Di payment gateway: generate 1 payment link untuk total semua tunggakan

```
Contoh: Pelanggan punya 3 invoice terlambat
  INV-2026-02: Rp 350.000 (terlambat 60 hari)
  INV-2026-03: Rp 350.000 (terlambat 30 hari)
  INV-2026-04: Rp 350.000 (terlambat 5 hari)

  Bayar Rp 400.000:
    → INV-2026-02: Lunas (Rp 350.000)
    → INV-2026-03: Bayar Sebagian (Rp 50.000 dari Rp 350.000)
    → INV-2026-04: Belum Bayar
```

### Alur Pembayaran Online (Payment Gateway)

```
Invoice digenerate
  │
  ▼
Sistem buat payment link via Xendit/Midtrans
  ├── Virtual Account (BCA, BNI, BRI, Mandiri, Permata)
  ├── QRIS
  ├── E-wallet (OVO, GoPay, DANA, ShopeePay)
  └── Kartu Kredit/Debit
  │
  ▼
Payment link dikirim ke pelanggan via notifikasi
  │
  ▼
Pelanggan bayar via channel pilihan
  │
  ▼
Webhook dari payment gateway
  │
  ├── Verifikasi signature webhook
  ├── Cocokkan dengan invoice
  ├── Update status invoice → Lunas
  ├── Jika isolir → buka isolir otomatis
  ├── Kirim notifikasi konfirmasi
  └── Log transaksi
```

### Webhook Handler
| Event | Aksi |
|---|---|
| `payment.paid` | Update invoice → Lunas, buka isolir, kirim notifikasi |
| `payment.expired` | Log saja, invoice tetap Belum Bayar |
| `payment.failed` | Log saja, kirim notifikasi gagal bayar |

### Keamanan Webhook
- Verifikasi **callback token / signature** dari payment gateway
- Endpoint webhook hanya menerima IP dari whitelist Xendit/Midtrans
- **Idempotency**: cek apakah payment sudah diproses sebelumnya (hindari double processing)
- Log semua webhook request (termasuk yang gagal verifikasi)

### Payment Link Expiry
- Payment link memiliki **masa berlaku 7 hari** (configurable per tenant)
- Setelah expired:
  - Status payment link → Expired
  - Invoice tetap Belum Bayar (tidak berubah)
  - **Auto-regenerate** payment link baru saat kirim reminder berikutnya
  - Pelanggan juga bisa minta link baru via walled garden
- Jika pelanggan bayar via transfer manual setelah link expired → admin catat manual seperti biasa

### Concurrency & Double Payment Prevention
Untuk mencegah double payment (admin catat manual bersamaan dengan webhook masuk):
- Setiap invoice memiliki **optimistic locking** (version field)
- Sebelum update status invoice, cek versi terakhir
- Jika versi sudah berubah (sudah diproses oleh proses lain) → tolak dan log sebagai duplikat
- Webhook handler menggunakan **idempotency key** dari payment gateway
- Jika terdeteksi double payment → tandai sebagai overpayment, masuk ke kredit saldo pelanggan
- Admin mendapat notifikasi jika terjadi double payment untuk review manual


---

## Isolir & Buka Isolir Otomatis

### Alur Isolir Otomatis

```
Cron job harian (jam 01:00)
  │
  ▼
Cari invoice dengan status Belum Bayar/Terlambat
  yang sudah lewat grace period
  │
  ▼
Untuk setiap invoice:
  ├── Update status pelanggan → Isolir
  ├── Kirim perintah ke MikroTik (jika modul aktif):
  │     ├── PPPoE: disable user, redirect ke walled garden
  │     └── Static: tambah firewall rule redirect
  ├── Kirim notifikasi isolir ke pelanggan
  └── Log: "Isolir otomatis — tunggakan {X} hari"
```

### Retry Mechanism (MikroTik Command)
Jika perintah ke MikroTik gagal (router offline, timeout, koneksi terputus):

| Percobaan | Interval | Aksi jika gagal |
|---|---|---|
| 1 (pertama) | Langsung | Retry |
| 2 | 5 menit | Retry |
| 3 | 30 menit | Retry |
| 4 | 2 jam | Retry |
| 5 (terakhir) | 6 jam | Tandai sebagai **"Pending Sync"**, notifikasi ke admin |

- Status pelanggan di database **tetap diupdate** (isolir/aktif) meskipun MikroTik gagal
- Background job **periodic sync** (setiap 15 menit) mencoba sinkronisasi ulang semua yang "Pending Sync"
- Admin bisa trigger **manual sync** dari dashboard
- Indikator visual di tabel pelanggan: ⚠️ "Belum sinkron ke router" jika status DB ≠ status MikroTik

### Walled Garden (Halaman Tagihan)
Pelanggan yang diisolir di-redirect ke halaman tagihan:

```
┌──────────────────────────────────────────┐
│                                          │
│         ⚠️ Layanan Internet Anda         │
│            Sementara Dihentikan          │
│                                          │
│  Tagihan Anda untuk periode April 2026   │
│  belum dibayar.                          │
│                                          │
│  Total Tagihan: Rp 350.000              │
│  Jatuh Tempo: 5 April 2026              │
│  Terlambat: 15 hari                      │
│                                          │
│  ┌────────────────────────────────────┐  │
│  │  [Bayar Sekarang via QRIS/VA]     │  │
│  └────────────────────────────────────┘  │
│                                          │
│  Atau hubungi admin:                     │
│  📱 0812-xxxx-xxxx (WhatsApp)           │
│                                          │
│  ISPBoss Net — Kelola ISP Kamu           │
│  Dari Satu Dashboard                     │
└──────────────────────────────────────────┘
```

- Halaman ini di-serve oleh MikroTik (hotspot login page / walled garden)
- Branding tenant (logo, nama ISP, warna)
- Tombol bayar langsung ke payment link (jika gateway aktif)
- Jika gateway tidak aktif, tampilkan info kontak admin saja

### Alur Buka Isolir Otomatis

```
Pembayaran diterima (manual atau webhook)
  │
  ▼
Invoice status → Lunas
  │
  ▼
Cek: pelanggan sedang isolir?
  ├── Ya:
  │     ├── Kirim perintah ke MikroTik: enable user
  │     ├── Hapus firewall rule redirect
  │     ├── Update status pelanggan → Aktif
  │     ├── Reset quota (jika paket ada quota)
  │     ├── Kirim notifikasi: "Internet Anda sudah aktif kembali"
  │     └── Log: "Buka isolir otomatis — pembayaran diterima"
  └── Tidak: selesai
```

### Batas Toleransi (Suspend Total)

```
Cron job harian (jam 02:00)
  │
  ▼
Cari pelanggan isolir yang sudah lewat batas toleransi
  (default: 30 hari setelah jatuh tempo)
  │
  ▼
Untuk setiap pelanggan:
  ├── Update status → Suspend
  ├── MikroTik: hapus user PPPoE (bukan disable, tapi remove)
  ├── Kirim notifikasi: "Layanan dihentikan, hubungi admin"
  └── Log: "Suspend — tunggakan {X} hari"
```

> **Catatan:** Pelanggan suspend bisa diaktifkan kembali oleh admin secara manual. Perlu bayar semua tunggakan + biaya aktivasi ulang (opsional, configurable).


---

## Prorate Billing (Upgrade/Downgrade Paket)

### Upgrade Paket

```
Contoh: Pelanggan upgrade dari Basic 10M (Rp 150.000) ke Pro 50M (Rp 350.000)
Tanggal upgrade: 15 April 2026
Jatuh tempo: 5 Mei 2026 (sisa 20 hari dari 30 hari)

Perhitungan:
  Sisa nilai Basic  = Rp 150.000 × (20/30) = Rp 100.000 (kredit)
  Nilai Pro sisa     = Rp 350.000 × (20/30) = Rp 233.333 (tagihan)
  Selisih            = Rp 233.333 - Rp 100.000 = Rp 133.333

  → Generate invoice prorate: Rp 133.333 (dibulatkan ke Rp 133.500)
  → Bandwidth langsung diupdate ke Pro 50M
  → Invoice bulan depan: Rp 350.000 (harga penuh Pro)
```

### Downgrade Paket

```
Contoh: Pelanggan downgrade dari Pro 50M (Rp 350.000) ke Basic 10M (Rp 150.000)
Tanggal downgrade: 15 April 2026
Jatuh tempo: 5 Mei 2026 (sisa 20 hari dari 30 hari)

Perhitungan:
  Sisa nilai Pro     = Rp 350.000 × (20/30) = Rp 233.333 (kredit)
  Nilai Basic sisa   = Rp 150.000 × (20/30) = Rp 100.000 (tagihan)
  Selisih            = Rp 100.000 - Rp 233.333 = -Rp 133.333

  → Kredit Rp 133.333 diterapkan ke invoice bulan depan
  → Bandwidth langsung diupdate ke Basic 10M
  → Invoice bulan depan: Rp 150.000 - Rp 133.333 = Rp 16.667 (dibulatkan Rp 17.000)
```

### Pembulatan
- Semua perhitungan prorate dibulatkan ke **Rp 500 terdekat ke atas**
- Contoh: Rp 133.333 → Rp 133.500
- Kredit dibulatkan ke **Rp 500 terdekat ke bawah** (menguntungkan pelanggan)
- Perhitungan hari selalu menggunakan **30 hari tetap** per bulan (bukan hari aktual) untuk konsistensi

### Pelanggan Berhenti Tengah Bulan

```
Contoh: Pelanggan berhenti tanggal 15 April 2026
Paket: Pro 50M (Rp 350.000/bulan)
Jatuh tempo: setiap tanggal 5
Invoice April sudah terbit dan lunas

Opsi 1: Tanpa refund (default)
  → Layanan tetap aktif sampai akhir periode (5 Mei)
  → Setelah periode habis, tidak generate invoice baru
  → Paling sederhana, umum di industri ISP

Opsi 2: Refund prorate (configurable)
  Sisa hari yang tidak dipakai = 20 hari (15 Apr → 5 Mei)
  Refund = Rp 350.000 × (20/30) = Rp 233.333 → Rp 233.000
  → Generate credit note
  → Admin proses refund manual

Opsi 3: Potong langsung (configurable)
  → Layanan langsung dimatikan saat berhenti
  → Tidak ada refund
  → Cocok untuk pelanggan bermasalah
```

Setting di tenant: **Pelanggan Berhenti** → Aktif sampai akhir periode / Refund prorate / Potong langsung

---

## Credit Note & Debit Note

### Credit Note (Nota Kredit)
Dokumen formal untuk pengurangan tagihan:

| Kapan Diterbitkan | Contoh |
|---|---|
| Invoice dibatalkan setelah lunas | Pelanggan sudah bayar, ternyata invoice salah |
| Refund prorate pelanggan berhenti | Pelanggan berhenti tengah bulan, ada sisa periode |
| Overpayment yang di-refund | Kelebihan bayar dikembalikan |
| Kompensasi gangguan layanan | Internet mati 3 hari, admin beri kompensasi |

- Format nomor: `CN-{YYYY}-{MM}-{SEQ}` (contoh: CN-2026-04-001)
- Referensi ke invoice terkait
- Bisa diterapkan sebagai kredit saldo atau refund tunai
- Generate PDF (format mirip invoice)

### Debit Note (Nota Debit)
Dokumen formal untuk tagihan tambahan di luar invoice reguler:

| Kapan Diterbitkan | Contoh |
|---|---|
| Biaya tambahan setelah invoice terbit | Pemasangan kabel tambahan |
| Penggantian perangkat | ONT rusak, ganti baru |
| Biaya aktivasi ulang | Pelanggan suspend, mau aktif lagi |

- Format nomor: `DN-{YYYY}-{MM}-{SEQ}`
- Bisa digabung ke invoice berikutnya atau berdiri sendiri

---

## Diskon & Promo (Opsional)

### Tipe Diskon

| Tipe | Contoh | Penerapan |
|---|---|---|
| **Diskon Pelanggan** | Pelanggan loyal diskon 10% | Per pelanggan, berlaku terus sampai dicabut |
| **Promo Periode** | Gratis 1 bulan untuk pelanggan baru | Otomatis, berlaku dalam rentang tanggal tertentu |
| **Diskon Bundling** | Bayar 6 bulan, gratis 1 bulan | Saat pembayaran di muka |
| **Diskon Referral** | Ajak teman, dapat diskon Rp 50.000 | Otomatis saat referral aktif |

### Penerapan Diskon di Invoice
- Diskon ditampilkan sebagai **item terpisah** (nilai negatif) di invoice
- Dihitung sebelum PPN (diskon mengurangi subtotal)
- Bisa dikombinasikan (pelanggan loyal + promo periode), tapi ada **max diskon** (configurable, default 50%)
- Admin bisa beri diskon manual per invoice

```
  Paket Pro 50M (Apr 2026)    1 bln    Rp  350.000
  Diskon Pelanggan Loyal 10%           -Rp   35.000
  ─────────────────────────────────────────────────
  Subtotal                              Rp  315.000
  PPN 11%                               Rp   34.650
  ─────────────────────────────────────────────────
  TOTAL                                 Rp  349.650
```

---

## Audit Trail Pembayaran & Invoice

Semua operasi billing dicatat dalam audit log (append-only):

```
INV-2026-04-002 — Audit Trail:
┌──────────────────┬──────────────────────────────────┬──────────────┐
│ Waktu            │ Aksi                             │ Oleh         │
├──────────────────┼──────────────────────────────────┼──────────────┤
│ 2026-04-01 00:01 │ Invoice digenerate otomatis       │ System       │
│ 2026-04-01 00:02 │ Notifikasi invoice dikirim (WA)   │ System       │
│ 2026-04-04 10:00 │ Reminder H-1 dikirim (WA)        │ System       │
│ 2026-04-06 08:00 │ Reminder H+1 dikirim (WA)        │ System       │
│ 2026-04-12 01:00 │ Isolir otomatis                   │ System       │
│ 2026-04-15 14:30 │ Pembayaran Rp 160.000 dicatat     │ Kasir Ani    │
│ 2026-04-15 14:30 │ Denda Rp 10.000 di-waive          │ Admin Budi   │
│ 2026-04-15 14:31 │ Buka isolir otomatis              │ System       │
│ 2026-04-15 14:31 │ Notifikasi konfirmasi bayar (WA)  │ System       │
└──────────────────┴──────────────────────────────────┴──────────────┘
```

- Log tidak bisa dihapus atau diubah (append-only)
- Mencatat: siapa, kapan, apa yang dilakukan
- Penting untuk rekonsiliasi keuangan dan audit
- Bisa difilter per invoice, per pelanggan, per user admin

---

## Halaman Daftar Pembayaran (`/payments`)

Halaman khusus untuk kasir yang menangani banyak pembayaran per hari:

### Layout Desktop

```
╔══════════════════════════════════════════════════════════════════════════╗
║  Dashboard > Pembayaran                                                  ║
║                                                                          ║
║  Pembayaran                                       [+ Catat Pembayaran]   ║
║                                                                          ║
║  ┌────────────┬────────────┬────────────┬────────────┐                   ║
║  │ 💰 Hari Ini│ 📊 Bulan Ini│ 🏦 Tunai   │ 💳 Online  │                   ║
║  │ Rp 4.2jt   │ Rp 252.0jt │ Rp 1.8jt   │ Rp 2.4jt   │                   ║
║  │ 12 trx     │ 720 trx    │ 5 trx      │ 7 trx      │                   ║
║  └────────────┴────────────┴────────────┴────────────┘                   ║
║                                                                          ║
║  ┌───────────────────────────────────────────────────────────────────┐    ║
║  │ 🔍 Cari pelanggan, no invoice...                                  │    ║
║  ├───────────────────────────────────────────────────────────────────┤    ║
║  │ Filter: [Metode ▼] [Periode ▼] [Dicatat Oleh ▼] [Reset]         │    ║
║  └───────────────────────────────────────────────────────────────────┘    ║
║                                                                          ║
║  ┌──────────┬───────────┬──────────────┬─────────┬────────┬──────┬────┐  ║
║  │ Tanggal  │ Pelanggan │ Invoice      │ Jumlah  │ Metode │ Oleh │Aksi│  ║
║  ├──────────┼───────────┼──────────────┼─────────┼────────┼──────┼────┤  ║
║  │ 28/04/26 │ Ahmad R.  │ INV-2026-04  │ Rp388.5k│ VA BCA │ Xendit│ ⋯ │  ║
║  │ 14:30    │ PLG-001   │ -001         │         │        │      │    │  ║
║  ├──────────┼───────────┼──────────────┼─────────┼────────┼──────┼────┤  ║
║  │ 28/04/26 │ Dewi A.   │ INV-2026-04  │ Rp150.0k│ Tunai  │ Kasir│ ⋯ │  ║
║  │ 13:15    │ PLG-004   │ -004         │         │        │ Ani  │    │  ║
║  └──────────┴───────────┴──────────────┴─────────┴────────┴──────┴────┘  ║
║                                                                          ║
║  ◀ 1 2 3 ... ▶                                10 / 25 / 50 per halaman  ║
╚══════════════════════════════════════════════════════════════════════════╝
```

### Quick Payment (Bayar Cepat)
Untuk kasir yang menerima banyak pembayaran tunai:

```
╔══════════════════════════════════════════════════════════════╗
║  Bayar Cepat                                                 ║
║                                                              ║
║  🔍 Cari pelanggan (nama, ID, telepon)...                    ║
║  [Ahmad Rizki — PLG-001 — Pro 50M]  ← autocomplete          ║
║                                                              ║
║  Invoice terbuka:                                            ║
║  ☑ INV-2026-04-001  Apr 2026  Rp 388.500  ⏳ Belum Bayar    ║
║  ☐ INV-2026-03-001  Mar 2026  Rp 388.500  🔴 Terlambat      ║
║                                                              ║
║  Total dipilih: Rp 388.500                                   ║
║  Metode: ○ Tunai  ● Transfer  ○ Lainnya                      ║
║                                                              ║
║                              [Batal]  [Bayar Rp 388.500]    ║
╚══════════════════════════════════════════════════════════════╝
```

- Autocomplete pelanggan (search by nama, ID, telepon)
- Langsung tampilkan invoice yang belum bayar
- Checkbox untuk pilih invoice mana yang mau dibayar
- 1 klik bayar — minimal input untuk kasir yang sibuk

### Struk / Kwitansi Pembayaran
Untuk pembayaran tunai di tempat, kasir butuh bukti terima yang ringkas (bukan invoice PDF):

```
┌──────────────────────────────┐
│  ISPBoss Net                 │
│  ─────────────────────────── │
│  KWITANSI PEMBAYARAN         │
│  No: PAY-2026-04-0012        │
│  Tanggal: 28 Apr 2026 14:30  │
│  ─────────────────────────── │
│  Pelanggan: Ahmad Rizki      │
│  ID: PLG-001                 │
│  Invoice: INV-2026-04-001    │
│  ─────────────────────────── │
│  Jumlah: Rp 388.500          │
│  Metode: Tunai               │
│  Kasir: Ani                  │
│  ─────────────────────────── │
│  Terima kasih!               │
└──────────────────────────────┘
```

- Format kecil untuk **thermal printer** (58mm / 80mm)
- Otomatis muncul dialog cetak setelah simpan pembayaran
- Bisa cetak ulang dari riwayat pembayaran
- Nomor kwitansi: `PAY-{YYYY}-{MM}-{SEQ}`

### Void / Reversal Pembayaran
Jika kasir salah catat pembayaran:

| Aksi | Siapa | Dampak |
|---|---|---|
| Void Pembayaran | **Hanya Admin** (bukan kasir) | Status invoice kembali ke sebelumnya. Jika sudah buka isolir → isolir kembali |
| Alasan wajib diisi | — | Untuk audit trail |
| Batas waktu void | 24 jam | Setelah 24 jam, harus pakai credit note |

```
Void Pembayaran — PAY-2026-04-0012
  Alasan: Salah input, seharusnya PLG-002 bukan PLG-001
  [Konfirmasi Void]

  Dampak:
  → INV-2026-04-001: Lunas → Belum Bayar
  → Pelanggan PLG-001: Aktif → Isolir (jika sebelumnya isolir)
  → Kredit saldo: dikembalikan (jika ada overpayment)
```

---

## Pembayaran di Muka (Prepaid Bulanan)

Pelanggan PPPoE bisa bayar beberapa bulan sekaligus:

### Mekanisme
- Admin pilih pelanggan → "Bayar di Muka" → pilih jumlah bulan (3/6/12)
- Sistem generate **1 invoice gabungan** untuk semua periode
- Diskon bundling otomatis diterapkan (jika setting aktif)

```
Contoh: Bayar 6 bulan di muka
Paket: Pro 50M (Rp 350.000/bulan)
Diskon bundling: Gratis 1 bulan (bayar 5, dapat 6)

Invoice:
  Paket Pro 50M × 6 bulan          Rp 2.100.000
  Diskon Bundling (gratis 1 bln)  -Rp   350.000
  ─────────────────────────────────────────────
  Subtotal                          Rp 1.750.000
  PPN 11%                           Rp   192.500
  ─────────────────────────────────────────────
  TOTAL                             Rp 1.942.500

Periode tercakup: Mei 2026 — Oktober 2026
→ Selama 6 bulan, tidak generate invoice otomatis
→ Invoice otomatis mulai lagi November 2026
```

### Edge Case Prepaid
| Skenario | Handling |
|---|---|
| Upgrade di tengah prepaid | Hitung selisih prorate untuk sisa bulan prepaid. Generate invoice prorate |
| Downgrade di tengah prepaid | Kredit selisih untuk sisa bulan. Diterapkan setelah prepaid habis |
| Berhenti di tengah prepaid | Refund prorate sisa bulan (jika setting refund aktif). Generate credit note |
| Perubahan harga paket | Tidak berpengaruh — harga sudah di-lock saat pembayaran prepaid |

---

## Recurring Item (Tagihan Tambahan Berulang)

Beberapa ISP menagih item tambahan secara rutin selain paket internet:

### Contoh Recurring Item
| Item | Harga | Keterangan |
|---|---|---|
| Sewa ONT | Rp 25.000/bulan | Perangkat milik ISP yang dipinjamkan |
| Sewa Router | Rp 15.000/bulan | Router WiFi milik ISP |
| IP Public Static | Rp 50.000/bulan | Tambahan IP public |
| Maintenance Fee | Rp 10.000/bulan | Biaya perawatan jaringan |

### Penerapan
- Recurring item di-assign **per pelanggan** (bukan per paket)
- Otomatis ditambahkan ke invoice bulanan sebagai line item terpisah
- Bisa diaktifkan/nonaktifkan per pelanggan
- Tanggal mulai & tanggal selesai (opsional, untuk sewa sementara)

```
Invoice INV-2026-04-001:
  Paket Pro 50M (Apr 2026)    1 bln    Rp  350.000
  Sewa ONT                    1 bln    Rp   25.000
  IP Public Static             1 bln    Rp   50.000
  ─────────────────────────────────────────────────
  Subtotal                              Rp  425.000
  PPN 11%                               Rp   46.750
  ─────────────────────────────────────────────────
  TOTAL                                 Rp  471.750
```

### Kelola Recurring Item
- Admin bisa tambah/edit/hapus recurring item dari **detail pelanggan > tab Layanan**
- Atau dari menu **Settings > Recurring Item** untuk template item yang sering dipakai

---

## Aging Report (Laporan Umur Piutang)

Mengelompokkan piutang berdasarkan umur tunggakan untuk keputusan penagihan:

```
┌────────────────────────────────────────────────────────────────┐
│ Aging Report — April 2026                                      │
│                                                                │
│ ┌──────────────┬──────────┬──────────┬──────────┬────────────┐ │
│ │ 1-7 hari     │ 8-14 hari│ 15-30 hari│ 30+ hari│ Total      │ │
│ ├──────────────┼──────────┼──────────┼──────────┼────────────┤ │
│ │ Rp 12.5jt    │ Rp 8.2jt │ Rp 5.8jt │ Rp 3.5jt│ Rp 30.0jt  │ │
│ │ 35 plgn      │ 22 plgn  │ 18 plgn  │ 12 plgn │ 87 plgn    │ │
│ │ 🟡 Grace     │ 🟠 Warning│ 🔴 Isolir │ ⚫ Suspend│            │ │
│ └──────────────┴──────────┴──────────┴──────────┴────────────┘ │
│                                                                │
│ Trend Piutang (3 bulan terakhir):                              │
│ Feb: Rp 25.0jt → Mar: Rp 28.0jt → Apr: Rp 30.0jt (↑ 7.1%)  │
│                                                                │
│ Pelanggan dengan tunggakan terbesar:                           │
│ 1. Budi S. (PLG-002) — Rp 1.050.000 (3 bulan)                │
│ 2. Eko P. (PLG-005) — Rp 750.000 (2 bulan)                   │
│ 3. Fajar R. (PLG-008) — Rp 500.000 (2 bulan)                 │
│                                                                │
│ [Export PDF]  [Export Excel]                                    │
└────────────────────────────────────────────────────────────────┘
```

- Klik per kelompok umur → lihat daftar pelanggan
- Bisa bulk action dari sini (kirim reminder, isolir massal)
- Detail lengkap di dokumen **11 — Reporting & Analytics**

---

## Perubahan Harga Paket — Dampak ke Billing

Jika admin mengubah harga paket:

| Skenario | Handling |
|---|---|
| Invoice sudah terbit (bulan ini) | **Tidak berubah** — tetap harga lama |
| Invoice bulan depan | **Harga baru** otomatis diterapkan |
| Pelanggan prepaid | **Tidak berubah** — harga sudah di-lock |
| Notifikasi | Kirim notifikasi ke semua pelanggan paket tersebut: "Harga paket {nama} berubah mulai {bulan}" |

### Grace Period Perubahan Harga
- Perubahan harga **berlaku mulai periode berikutnya** (bukan langsung)
- Admin bisa set tanggal efektif perubahan harga
- Sistem otomatis kirim notifikasi ke pelanggan terdampak **30 hari sebelum** harga baru berlaku
- Pelanggan bisa downgrade/berhenti sebelum harga baru berlaku

---

## Denda Keterlambatan (Opsional)

Jika diaktifkan di settings:

| Setting | Opsi | Contoh |
|---|---|---|
| Tipe Denda | Nominal tetap | Rp 10.000 per invoice terlambat |
| Tipe Denda | Persentase | 5% dari total tagihan |
| Tipe Denda | Harian | Rp 2.000 per hari keterlambatan |
| Max Denda | Nominal | Rp 50.000 (batas atas denda) |

- Denda dihitung otomatis saat pembayaran dicatat
- Ditampilkan sebagai item terpisah di invoice
- Admin bisa **hapus/waive denda** secara manual per invoice

---

## Pajak / PPN (Opsional)

Jika diaktifkan di settings:
- Default: 11% (PPN Indonesia)
- Persentase configurable
- Ditampilkan sebagai item terpisah di invoice
- Dihitung dari subtotal (sebelum denda)

---

## Riwayat Pembayaran per Pelanggan

Tersedia di tab **Pembayaran** pada halaman detail pelanggan:

```
┌──────────────────────────────────────────────────────────────────┐
│ Riwayat Pembayaran — Ahmad Rizki (PLG-001)                       │
│                                                                  │
│ ┌──────────┬──────────────┬──────────┬──────────┬──────────────┐ │
│ │ Tanggal  │ Invoice      │ Jumlah   │ Metode   │ Status       │ │
│ ├──────────┼──────────────┼──────────┼──────────┼──────────────┤ │
│ │ 03/04/26 │ INV-2026-04  │ Rp388.500│ Xendit VA│ ✅ Berhasil  │ │
│ │ 02/03/26 │ INV-2026-03  │ Rp388.500│ Tunai    │ ✅ Berhasil  │ │
│ │ 05/02/26 │ INV-2026-02  │ Rp388.500│ Transfer │ ✅ Berhasil  │ │
│ │ 10/01/26 │ INV-2026-01  │ Rp388.500│ QRIS     │ ✅ Berhasil  │ │
│ └──────────┴──────────────┴──────────┴──────────┴──────────────┘ │
│                                                                  │
│ Total dibayar (2026): Rp 1.554.000                               │
│ Rata-rata keterlambatan: 1.2 hari                                │
└──────────────────────────────────────────────────────────────────┘
```


---

## Invoice PDF

### Template PDF Invoice

```
┌──────────────────────────────────────────────────────────┐
│  [LOGO]  ISPBoss Net                                     │
│  Jl. Raya No. 1, Kota Depok                             │
│  Telp: 0812-xxxx-xxxx                                   │
│  ─────────────────────────────────────────────────────── │
│                                                          │
│  INVOICE                                                 │
│  No: INV-2026-04-001                                     │
│  Tanggal: 1 April 2026                                   │
│  Jatuh Tempo: 5 April 2026                               │
│  Status: LUNAS                                           │
│                                                          │
│  Kepada:                                                 │
│  Ahmad Rizki (PLG-001)                                   │
│  Jl. Merdeka No. 10, Kel. Sukamaju                      │
│  Telp: 0812-345-6789                                     │
│  ─────────────────────────────────────────────────────── │
│                                                          │
│  Deskripsi                    Qty      Harga             │
│  ─────────────────────────────────────────────────────── │
│  Paket Pro 50M (Apr 2026)    1 bln    Rp  350.000       │
│  ─────────────────────────────────────────────────────── │
│  Subtotal                              Rp  350.000       │
│  PPN 11%                               Rp   38.500       │
│  Denda                                 Rp        0       │
│  ─────────────────────────────────────────────────────── │
│  TOTAL                                 Rp  388.500       │
│  ─────────────────────────────────────────────────────── │
│                                                          │
│  Pembayaran:                                             │
│  3 Apr 2026 — Rp 388.500 via VA BCA                     │
│                                                          │
│  ─────────────────────────────────────────────────────── │
│  Terima kasih atas pembayaran Anda.                      │
│  ISPBoss Net — Kelola ISP Kamu Dari Satu Dashboard       │
└──────────────────────────────────────────────────────────┘
```

- Branding tenant (logo, nama, alamat, kontak)
- Generate via library **maroto** atau **gofpdf** (Golang)
- Bisa download dari detail invoice
- Bisa dikirim langsung ke pelanggan via notifikasi (attachment WA/Email)

---

## Bulk Action Invoice

| Aksi | Deskripsi | Syarat |
|---|---|---|
| Bulk Kirim Reminder | Kirim notifikasi reminder ke semua yang belum bayar | Status = Belum Bayar / Terlambat |
| Bulk Download PDF | Download beberapa invoice sekaligus (ZIP) | Pilih via checkbox |
| Bulk Batalkan | Batalkan beberapa invoice | Status = Belum Bayar |
| Export CSV/Excel | Download daftar invoice ke CSV/Excel | Semua status |
| Bulk Catat Pembayaran | Upload CSV pembayaran (pelanggan, jumlah, metode, tanggal) | Untuk input banyak pembayaran sekaligus |

---

## Rekonsiliasi Keuangan

### Ringkasan Bulanan (di Dashboard)

```
┌────────────────────────────────────────────────────────────┐
│ Ringkasan Billing — April 2026                             │
│                                                            │
│ ┌──────────────┬──────────────┬──────────────┬───────────┐ │
│ │ Total Tagihan│ Sudah Bayar  │ Belum Bayar  │ Terlambat │ │
│ │ Rp 294.5jt   │ Rp 252.0jt   │ Rp 30.5jt    │ Rp 12.0jt│ │
│ │ 847 invoice  │ 720 invoice  │ 87 invoice   │ 40 invoice│ │
│ └──────────────┴──────────────┴──────────────┴───────────┘ │
│                                                            │
│ Collection Rate: 85.6%                                     │
│ Rata-rata Waktu Bayar: 3.2 hari sebelum jatuh tempo       │
│                                                            │
│ Pendapatan Voucher (bulan ini): Rp 15.2jt                  │
│ Total Pendapatan: Rp 267.2jt                               │
└────────────────────────────────────────────────────────────┘
```

### Laporan Detail
Detail laporan keuangan dibahas di dokumen **11 — Reporting & Analytics**.

---

## Pelanggan Baru — Invoice Pertama

### Skenario Aktivasi Tengah Bulan

```
Contoh: Pelanggan baru aktivasi 20 April 2026
Paket: Pro 50M (Rp 350.000/bulan)
Jatuh tempo: setiap tanggal 5
Biaya pasang: Rp 500.000

Opsi 1: Prorate bulan pertama (default)
  Sisa April = 15 hari (20 Apr → 5 Mei)
  Tagihan prorate = Rp 350.000 × (15/30) = Rp 175.000
  Invoice pertama:
    - Biaya pasang: Rp 500.000
    - Prorate Apr-Mei: Rp 175.000
    - Total: Rp 675.000
  Invoice bulan depan (Mei): Rp 350.000 (penuh)

Opsi 2: Bulan penuh (configurable)
  Invoice pertama:
    - Biaya pasang: Rp 500.000
    - Tagihan April: Rp 350.000 (penuh)
    - Total: Rp 850.000
  Invoice bulan depan (Mei): Rp 350.000 (penuh)
```

Setting di tenant: **Tagihan Pelanggan Baru** → Prorate / Bulan Penuh

---

## Aksi Invoice

| Aksi | Deskripsi | Konfirmasi |
|---|---|---|
| Lihat Detail | Buka halaman detail invoice | Tidak |
| Download PDF | Download invoice sebagai PDF | Tidak |
| Kirim ke Pelanggan | Kirim invoice via WA/Email | Ya — pilih channel |
| Catat Pembayaran | Buka form catat pembayaran manual | Tidak |
| Edit | Edit item invoice (hanya jika belum bayar) | Tidak |
| Batalkan | Batalkan invoice, status → Batal | Ya — ketik nomor invoice |
| Kirim Reminder | Kirim reminder pembayaran | Ya |

---

## Integrasi dengan Modul Lain

| Modul | Integrasi |
|---|---|
| **Pelanggan (04)** | Tab Invoice & Pembayaran di detail pelanggan |
| **Paket (05)** | Harga paket → item invoice. Prorate saat upgrade/downgrade |
| **Notifikasi (07)** | Kirim invoice, reminder, konfirmasi bayar, notifikasi isolir |
| **MikroTik (08)** | Isolir/buka isolir otomatis, walled garden |
| **Laporan (11)** | Data invoice & pembayaran → laporan keuangan |
| **Settings (12)** | Konfigurasi billing, payment gateway, pajak, denda |

---

## Keputusan

| Keputusan | Detail |
|---|---|
| Model billing | 2 model: Bulanan (PPPoE) dan Prepaid/Voucher (Hotspot) |
| Generate invoice | Otomatis via cron job, H-5 jatuh tempo (configurable) |
| Invoice manual | ✅ Ada — untuk biaya pasang, tagihan custom |
| Nomor invoice | Format `{PREFIX}-{YYYY}-{MM}-{SEQ}`, prefix configurable |
| Status invoice | 6 status: Belum Bayar, Terlambat, Lunas, Bayar Sebagian, Batal, Prorate |
| Pembayaran manual | ✅ Tunai, transfer bank, upload bukti |
| Payment gateway | ✅ Xendit & Midtrans, bisa aktif bersamaan |
| Payment link expiry | ✅ Default 7 hari, auto-regenerate saat kirim reminder |
| Webhook | Verifikasi signature, idempotency, IP whitelist |
| Concurrency | ✅ Optimistic locking, idempotency key, double payment → kredit saldo |
| Pembayaran sebagian | ✅ Diizinkan, isolir tetap berlaku sampai lunas (configurable) |
| Overpayment | ✅ Kelebihan bayar → kredit saldo pelanggan, dipotong invoice berikutnya |
| Multi-invoice tunggakan | ✅ FIFO (invoice terlama duluan), bisa bayar semua sekaligus |
| Reminder bertingkat | ✅ H-5, H-1, H+1, H+3, H+7 (configurable per tenant) |
| Grace period | Default 7 hari, configurable per tenant |
| Batas toleransi | Default 30 hari, setelah itu suspend total |
| Auto-isolir | ✅ Default aktif, configurable |
| Auto-buka isolir | ✅ Default aktif, setelah pembayaran diterima |
| Retry MikroTik | ✅ 5x retry dengan backoff, periodic sync 15 menit, manual sync |
| Walled garden | ✅ Halaman tagihan dengan tombol bayar langsung |
| Prorate | ✅ Untuk upgrade/downgrade, pembulatan Rp 500, selalu 30 hari/bulan |
| Pelanggan baru | Prorate bulan pertama (default) atau bulan penuh (configurable) |
| Pelanggan berhenti | ✅ 3 opsi: aktif sampai akhir periode (default) / refund prorate / potong langsung |
| Credit note | ✅ Format CN-{YYYY}-{MM}-{SEQ}, untuk pembatalan, refund, kompensasi |
| Debit note | ✅ Format DN-{YYYY}-{MM}-{SEQ}, untuk tagihan tambahan |
| Diskon & promo | ✅ 4 tipe: pelanggan, periode, bundling, referral. Max diskon configurable |
| Denda keterlambatan | ❌ Default nonaktif. Opsi: nominal tetap, persentase, harian. Max denda configurable |
| PPN/Pajak | ❌ Default nonaktif. Persentase configurable (default 11%) |
| Invoice PDF | ✅ Generate via Golang, branding tenant, download & kirim |
| Biaya pasang | Masuk ke invoice pertama sebagai item terpisah |
| Biaya aktivasi ulang | ✅ Opsional, untuk pelanggan yang di-suspend lalu aktif kembali |
| Bulk action | ✅ Bulk reminder, download PDF, batalkan, export |
| Halaman pembayaran | ✅ `/payments` — daftar semua pembayaran + quick payment untuk kasir |
| Struk/kwitansi | ✅ Format thermal printer (58/80mm), cetak otomatis setelah bayar, nomor PAY-{YYYY}-{MM}-{SEQ} |
| Void pembayaran | ✅ Hanya admin, batas 24 jam, alasan wajib, rollback status invoice & isolir |
| Pembayaran di muka | ✅ Bayar 3/6/12 bulan sekaligus, 1 invoice gabungan, diskon bundling otomatis |
| Recurring item | ✅ Tagihan tambahan berulang per pelanggan (sewa ONT, IP public, dll), otomatis masuk invoice |
| Aging report | ✅ Piutang per kelompok umur (1-7, 8-14, 15-30, 30+ hari), trend, top debitur |
| Perubahan harga paket | ✅ Berlaku mulai periode berikutnya, notifikasi 30 hari sebelum, invoice existing tidak berubah |
| Timezone | ✅ Configurable per tenant (WIB/WITA/WIT), default WIB |
| Audit trail | ✅ Log lengkap semua operasi billing (append-only) |
| Rekonsiliasi | ✅ Ringkasan bulanan, collection rate, gabungan voucher + bulanan |
| Perhitungan hari | Selalu 30 hari tetap per bulan untuk konsistensi prorate |