# 12 — Pengaturan (Settings)

---

## Konsep

Halaman Settings adalah pusat konfigurasi tenant. Semua pengaturan yang disebut di dokumen 00-11 dikonfigurasi di sini. Settings dikelompokkan dalam menu sidebar terpisah agar mudah dinavigasi.

### Akses Settings
| Role | Akses |
|---|---|
| Tenant Admin | Full access semua settings |
| Operator | Hanya profil sendiri |
| Teknisi | Hanya profil sendiri |
| Kasir | Hanya profil sendiri |
| Reseller | Tidak ada akses settings (punya dashboard sendiri) |

---

## Layout Settings (`/settings`)

### Desktop Layout

```
╔══════════════════════════════════════════════════════════════════════════╗
║  Dashboard > Pengaturan                                                  ║
║                                                                          ║
║  ┌──────────────────┬───────────────────────────────────────────────────┐║
║  │ Menu Settings    │ Konten                                            │║
║  │                  │                                                   │║
║  │ 🏢 Profil ISP    │ (sesuai menu yang dipilih)                        │║
║  │ 🎨 White Label   │                                                   │║
║  │ 👥 User & Role   │                                                   │║
║  │ 💰 Billing       │                                                   │║
║  │ 💳 Payment Gateway│                                                   │║
║  │ 📱 Notifikasi    │                                                   │║
║  │ 📡 MikroTik      │                                                   │║
║  │ 📡 OLT           │                                                   │║
║  │ 🗺️ Peta          │                                                   │║
║  │ 📊 Laporan       │                                                   │║
║  │ 🔐 Keamanan      │                                                   │║
║  │ 📦 Subscription  │                                                   │║
║  │ 📋 Audit Log     │                                                   │║
║  └──────────────────┴───────────────────────────────────────────────────┘║
╚══════════════════════════════════════════════════════════════════════════╝
```

### Mobile Layout

```
╔══════════════════════════╗
║  ☰  Pengaturan           ║
╠══════════════════════════╣
║                          ║
║  ┌──────────────────────┐║
║  │ 🏢 Profil ISP     →  │║
║  ├──────────────────────┤║
║  │ 🎨 White Label    →  │║
║  ├──────────────────────┤║
║  │ 👥 User & Role    →  │║
║  ├──────────────────────┤║
║  │ 💰 Billing        →  │║
║  ├──────────────────────┤║
║  │ 💳 Payment Gateway →  │║
║  ├──────────────────────┤║
║  │ 📱 Notifikasi     →  │║
║  ├──────────────────────┤║
║  │ 📡 MikroTik       →  │║
║  ├──────────────────────┤║
║  │ 📡 OLT            →  │║
║  ├──────────────────────┤║
║  │ 🗺️ Peta           →  │║
║  ├──────────────────────┤║
║  │ 📊 Laporan        →  │║
║  ├──────────────────────┤║
║  │ 🔐 Keamanan       →  │║
║  ├──────────────────────┤║
║  │ 📦 Subscription   →  │║
║  ├──────────────────────┤║
║  │ 📋 Audit Log      →  │║
║  └──────────────────────┘║
╚══════════════════════════╝

Tap menu → navigasi ke halaman setting
← Back untuk kembali ke daftar menu
Form setting: full-width, scroll vertikal
```


---

## 🏢 Profil ISP (`/settings/profile`)

```
╔══════════════════════════════════════════════════════════════╗
║  Profil ISP                                                  ║
║                                                              ║
║  ┌─── Informasi Perusahaan ──────────────────────────────┐   ║
║  │  Nama ISP *              No. Telepon *                │   ║
║  │  [ISPBoss Net________]  [0812-xxxx-xxxx___]          │   ║
║  │                                                       │   ║
║  │  Email                   Website                      │   ║
║  │  [admin@ispboss.net_]   [www.ispboss.net__]          │   ║
║  │                                                       │   ║
║  │  Alamat                                               │   ║
║  │  [Jl. Raya No. 1, Kota Depok, Jawa Barat]           │   ║
║  │                                                       │   ║
║  │  NPWP (opsional)                                      │   ║
║  │  [12.345.678.9-012.000]                              │   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║  ┌─── Timezone ──────────────────────────────────────────┐   ║
║  │  Timezone *: [WIB (UTC+7) ▼]                         │   ║
║  │  Opsi: WIB (UTC+7) / WITA (UTC+8) / WIT (UTC+9)     │   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║  Data ini digunakan di: invoice PDF, walled garden,          ║
║  notifikasi, laporan.                                        ║
║                                                              ║
║                                          [Simpan]            ║
╚══════════════════════════════════════════════════════════════╝
```

---

## 🎨 White Label (`/settings/branding`)

```
╔══════════════════════════════════════════════════════════════╗
║  White Label / Branding                                      ║
║                                                              ║
║  ┌─── Logo ──────────────────────────────────────────────┐   ║
║  │  Logo Utama (untuk sidebar, invoice, notifikasi)      │   ║
║  │  [📎 Upload] atau [🗑️ Hapus]                          │   ║
║  │  ┌────────┐  Format: PNG/SVG, max 500 KB              │   ║
║  │  │ [LOGO] │  Rekomendasi: 200x60 px                   │   ║
║  │  └────────┘                                           │   ║
║  │                                                       │   ║
║  │  Favicon                                              │   ║
║  │  [📎 Upload]  Format: ICO/PNG, 32x32 px              │   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║  ┌─── Warna ─────────────────────────────────────────────┐   ║
║  │  Warna Primer: [#2563EB 🎨]  (default: Blue 600)     │   ║
║  │  Preview:                                             │   ║
║  │  ┌──────────────────────────────────────────────────┐ │   ║
║  │  │  [Tombol Primer]  [Link]  [Sidebar Active]       │ │   ║
║  │  └──────────────────────────────────────────────────┘ │   ║
║  │                                                       │   ║
║  │  ☐ Gunakan warna custom (override tema default)       │   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║  ┌─── Custom Domain (Fase Lanjut) ───────────────────────┐   ║
║  │  Domain Custom: [billing.ispboss.net___]              │   ║
║  │  Status: ⚫ Belum dikonfigurasi                       │   ║
║  │                                                       │   ║
║  │  Instruksi:                                           │   ║
║  │  1. Tambahkan CNAME record di DNS Anda:               │   ║
║  │     billing.ispboss.net → app.ispboss.id              │   ║
║  │  2. Klik [Verifikasi Domain]                          │   ║
║  │  3. SSL otomatis di-generate (Let's Encrypt)          │   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║                                          [Simpan]            ║
╚══════════════════════════════════════════════════════════════╝
```

White label digunakan di:
- Sidebar dashboard (logo)
- Invoice PDF (logo, nama, alamat)
- Walled garden (logo, nama, warna)
- Notifikasi WA/Email (nama ISP)
- Voucher print (logo, nama)
- Hotspot login page (logo, nama, warna)

---

## 👥 User & Role Management (`/settings/users`)

### Daftar User

```
╔══════════════════════════════════════════════════════════════╗
║  User & Role                                [+ Tambah User]  ║
║                                                              ║
║  ┌──────────────┬──────────┬──────────────┬────────┬──────┐  ║
║  │ Nama         │ Role     │ Email        │ Status │ Aksi │  ║
║  ├──────────────┼──────────┼──────────────┼────────┼──────┤  ║
║  │ Budi Santoso │ Admin    │ budi@isp.net │ 🟢Aktif│ ⋯    │  ║
║  │ Ani Rahayu   │ Kasir    │ ani@isp.net  │ 🟢Aktif│ ⋯    │  ║
║  │ Andi Pratama │ Teknisi  │ andi@isp.net │ 🟢Aktif│ ⋯    │  ║
║  │ Dewi Lestari │ Operator │ dewi@isp.net │ 🟢Aktif│ ⋯    │  ║
║  │ Eko Prasetyo │ Operator │ eko@isp.net  │ ⚫Nonaktif│ ⋯  │  ║
║  └──────────────┴──────────┴──────────────┴────────┴──────┘  ║
╚══════════════════════════════════════════════════════════════╝
```

### Form Tambah User

```
╔══════════════════════════════════════════════════════════════╗
║  Tambah User Baru                                            ║
║                                                              ║
║  Nama Lengkap *: [___________________]                      ║
║  Email *:        [___________________]                      ║
║  No. Telepon:    [+62________________]                      ║
║  Password *:     [••••••••••      👁️]                       ║
║                                                              ║
║  Role *:                                                     ║
║  ○ Tenant Admin — Full access                                ║
║  ○ Operator — Operasional harian (pelanggan, billing)        ║
║  ○ Teknisi — Network (MikroTik, OLT, peta)                  ║
║  ○ Kasir — Input pembayaran saja                             ║
║                                                              ║
║  Notifikasi yang diterima:                                   ║
║  ☑ Pembayaran besar (> Rp 1jt)                              ║
║  ☐ Pelanggan baru                                           ║
║  ☐ Router offline                                           ║
║  ☐ Alarm OLT                                                ║
║                                                              ║
║                    [Batal]  [Simpan User]                    ║
╚══════════════════════════════════════════════════════════════╝
```

### Aksi User
| Aksi | Deskripsi | Konfirmasi |
|---|---|---|
| Edit | Edit data dan role user | Tidak |
| Nonaktifkan | User tidak bisa login | Ya |
| Aktifkan | User bisa login kembali | Tidak |
| Reset Password | Kirim link reset via email | Ya |
| Hapus | Hapus user permanen | Ya — ketik nama |
| Lihat Aktivitas | Lihat log aktivitas user | Tidak |


---

## 💰 Billing Settings (`/settings/billing`)

Semua konfigurasi billing dari dokumen 06:

```
╔══════════════════════════════════════════════════════════════╗
║  Konfigurasi Billing                                         ║
║                                                              ║
║  ┌─── Generate Invoice ──────────────────────────────────┐   ║
║  │  Generate invoice H-[5] sebelum jatuh tempo           │   ║
║  │  Nomor Invoice Prefix: [INV___]                       │   ║
║  │  Tagihan Pelanggan Baru: ● Prorate  ○ Bulan Penuh    │   ║
║  │  Perhitungan Hari: ● 30 hari tetap  ○ Hari aktual    │   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║  ┌─── Isolir & Toleransi ────────────────────────────────┐   ║
║  │  Grace Period: [7] hari setelah jatuh tempo           │   ║
║  │  Batas Toleransi (suspend): [30] hari                 │   ║
║  │  Auto-Isolir: ● Aktif  ○ Nonaktif                    │   ║
║  │  Auto-Buka Isolir: ● Aktif  ○ Nonaktif               │   ║
║  │  Buka isolir saat bayar sebagian: ○ Ya  ● Tidak       │   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║  ┌─── Denda & Pajak ────────────────────────────────────┐    ║
║  │  Denda Keterlambatan: ○ Aktif  ● Nonaktif            │    ║
║  │  Tipe: ○ Nominal [Rp___]  ○ Persentase [__]%         │    ║
║  │        ○ Harian [Rp___/hari]  Max: [Rp___]           │    ║
║  │                                                       │   ║
║  │  PPN/Pajak: ○ Aktif  ● Nonaktif                      │    ║
║  │  Persentase: [11] %                                   │    ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║  ┌─── Pelanggan Berhenti ────────────────────────────────┐   ║
║  │  ● Aktif sampai akhir periode (default)               │   ║
║  │  ○ Refund prorate                                     │   ║
║  │  ○ Potong langsung                                    │   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║  ┌─── Diskon ────────────────────────────────────────────┐   ║
║  │  Max Diskon Kombinasi: [50] %                         │   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║  ┌─── Reminder Bertingkat ───────────────────────────────┐   ║
║  │  ☑ H-5: Invoice terbit                                │   ║
║  │  ☑ H-1: Reminder sebelum jatuh tempo                  │   ║
║  │  ☑ H+1: Peringatan lewat jatuh tempo                  │   ║
║  │  ☑ H+3: Peringatan terakhir                           │   ║
║  │  ☑ H+7: Notifikasi isolir                             │   ║
║  │  [+ Tambah Reminder Custom]                           │   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║                                          [Simpan]            ║
╚══════════════════════════════════════════════════════════════╝
```

---

## 💳 Payment Gateway (`/settings/payment`)

```
╔══════════════════════════════════════════════════════════════╗
║  Payment Gateway                                             ║
║                                                              ║
║  ┌─── Xendit ────────────────────────────────────────────┐   ║
║  │  ☑ Aktifkan Xendit                                    │   ║
║  │  API Key: [••••••••••••••••      👁️]                  │   ║
║  │  Callback Token: [••••••••••••••]                     │   ║
║  │  Webhook URL: https://api.ispboss.id/v1/webhooks/xendit│   ║
║  │  Status: 🟢 Terhubung                                │   ║
║  │  [Test Koneksi]                                       │   ║
║  │                                                       │   ║
║  │  Channel Aktif:                                       │   ║
║  │  ☑ Virtual Account (BCA, BNI, BRI, Mandiri, Permata) │   ║
║  │  ☑ QRIS                                              │   ║
║  │  ☑ E-wallet (OVO, GoPay, DANA, ShopeePay)           │   ║
║  │  ☐ Kartu Kredit/Debit                                │   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║  ┌─── Midtrans ──────────────────────────────────────────┐   ║
║  │  ☐ Aktifkan Midtrans                                  │   ║
║  │  Server Key: [••••••••••]                             │   ║
║  │  Client Key: [••••••••••]                             │   ║
║  │  Status: ⚫ Belum dikonfigurasi                       │   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║  ┌─── Pengaturan Umum ───────────────────────────────────┐   ║
║  │  Payment Link Expiry: [7] hari                        │   ║
║  │  Mode: ● Production  ○ Sandbox (testing)              │   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║                                          [Simpan]            ║
╚══════════════════════════════════════════════════════════════╝
```

---

## 📱 Notifikasi Settings (`/settings/notifications`)

Konfigurasi provider notifikasi dari dokumen 07 (sudah detail di dokumen 07, di sini ringkasan):

| Setting | Lokasi Detail |
|---|---|
| Provider WhatsApp (Fonnte/WaBlas/WA Business API) | Dokumen 07 |
| Provider SMS (Zenziva/Twilio/Nexmo) | Dokumen 07 |
| Provider Email (SMTP/Mailgun/SendGrid) | Dokumen 07 |
| Prioritas Channel | Dokumen 07 |
| Quiet Hours | Dokumen 07 |
| Rate Limit Broadcast | Dokumen 07 |
| Template Notifikasi | Dokumen 07 (`/notifications/templates`) |

---

## 📡 MikroTik Settings (`/settings/mikrotik`)

```
╔══════════════════════════════════════════════════════════════╗
║  Konfigurasi MikroTik                                        ║
║                                                              ║
║  ┌─── Bandwidth Method ──────────────────────────────────┐   ║
║  │  ● PPPoE Profile rate-limit (default, paling umum)    │   ║
║  │  ○ Simple Queue (monitoring per user)                 │   ║
║  │  ○ Queue Tree + PCQ (ISP besar)                       │   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║  ┌─── Isolir Method ────────────────────────────────────┐    ║
║  │  ● DNS Redirect (default, paling efektif)            │    ║
║  │  ○ HTTP Redirect only                                │    ║
║  │  ○ Block All + Whitelist                             │    ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║  ┌─── Walled Garden ────────────────────────────────────┐    ║
║  │  Hosting: ● ISPBoss (default)  ○ Custom URL          │    ║
║  │  Custom URL: [https://billing.myisp.net/walled]      │    ║
║  │                                                       │   ║
║  │  Pesan Custom (tampil di walled garden):              │   ║
║  │  [Tagihan Anda belum dibayar. Segera bayar untuk_____]│   ║
║  │  [mengaktifkan kembali layanan internet Anda.________]│   ║
║  │                                                       │   ║
║  │  Tampilkan di walled garden:                          │   ║
║  │  ☑ Tombol bayar (payment gateway)                    │   ║
║  │  ☑ Info kontak admin (telepon/WA)                    │   ║
║  │  ☑ Detail tagihan (jumlah, periode)                  │   ║
║  │  ☐ Nomor rekening bank                               │   ║
║  │                                                       │   ║
║  │  Redirect setelah bayar: [/thank-you___]             │   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║  ┌─── Username PPPoE Format ─────────────────────────────┐   ║
║  │  Format: [{nama-depan}-{id-pelanggan}]                │   ║
║  │  Preview: ahmad-plg001                                │   ║
║  │  Variabel: {nama-depan}, {nama-belakang}, {id},       │   ║
║  │            {telepon}, {custom}                        │   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║  ┌─── Lainnya ───────────────────────────────────────────┐   ║
║  │  Health Check Interval Default: [60] detik            │   ║
║  │  Sync Interval: [15] menit                            │   ║
║  │  Auto Port Migration: ○ Ya  ● Tidak (tanya admin)    │   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║                                          [Simpan]            ║
╚══════════════════════════════════════════════════════════════╝
```

---

## 📡 OLT Settings (`/settings/olt`)

```
╔══════════════════════════════════════════════════════════════╗
║  Konfigurasi OLT                                             ║
║                                                              ║
║  ┌─── Signal Threshold ──────────────────────────────────┐   ║
║  │  Normal:   -8 s/d [-25] dBm                          │   ║
║  │  Warning:  [-25] s/d [-27] dBm                       │   ║
║  │  Weak:     [-27] s/d [-30] dBm                       │   ║
║  │  Critical: < [-30] dBm                               │   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║  ┌─── VLAN Strategy ────────────────────────────────────┐    ║
║  │  ● Single VLAN (default, semua pelanggan 1 VLAN)     │    ║
║  │  ○ Per Paket                                         │    ║
║  │  ○ Per ODP                                           │    ║
║  │  ○ Per Pelanggan                                     │    ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║  ┌─── Lainnya ───────────────────────────────────────────┐   ║
║  │  Health Check Interval: [300] detik (5 menit)         │   ║
║  │  Sync Interval: [30] menit                            │   ║
║  │  Auto-Provisioning: ○ Aktif  ● Nonaktif              │   ║
║  │  Auto Port Migration: ○ Ya  ● Tidak (tanya admin)    │   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║                                          [Simpan]            ║
╚══════════════════════════════════════════════════════════════╝
```

---

## 🎫 Voucher Settings (`/settings/voucher`)

```
╔══════════════════════════════════════════════════════════════╗
║  Konfigurasi Voucher                                         ║
║                                                              ║
║  ┌─── Format Kode ───────────────────────────────────────┐   ║
║  │  Panjang Kode Default: [6] karakter (min 6, max 16)   │   ║
║  │  Format Default: ● Gabungan  ○ Angka  ○ Huruf        │   ║
║  │  Prefix Default: [ISP-___]                            │   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║  ┌─── Masa Berlaku ─────────────────────────────────────┐    ║
║  │  Masa berlaku voucher (sebelum dipakai): [90] hari   │    ║
║  │  Setelah expired: ● Refund saldo reseller  ○ Hangus  │    ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║  ┌─── Generate ──────────────────────────────────────────┐   ║
║  │  Max generate per batch (sinkron): [500]              │   ║
║  │  Di atas limit → proses async                         │   ║
║  │  Collision retry: [3] kali per kode                   │   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║                                          [Simpan]            ║
╚══════════════════════════════════════════════════════════════╝
```

---

## 🌐 Lokalisasi (`/settings/localization`)

```
╔══════════════════════════════════════════════════════════════╗
║  Lokalisasi                                                  ║
║                                                              ║
║  ┌─── Format ────────────────────────────────────────────┐   ║
║  │  Format Tanggal: ● DD/MM/YYYY  ○ MM/DD/YYYY          │   ║
║  │  Format Mata Uang: Rp (Indonesia Rupiah)              │   ║
║  │  Pemisah Ribuan: ● Titik (1.000.000)  ○ Koma         │   ║
║  │  Bahasa Interface: ● Bahasa Indonesia  ○ English      │   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║                                          [Simpan]            ║
╚══════════════════════════════════════════════════════════════╝
```

---

## 📄 Invoice Customization (`/settings/invoice`)

```
╔══════════════════════════════════════════════════════════════╗
║  Kustomisasi Invoice                                         ║
║                                                              ║
║  ┌─── Footer Invoice ────────────────────────────────────┐   ║
║  │  Teks Footer (tampil di bawah invoice PDF):           │   ║
║  │  [Terima kasih atas pembayaran Anda.________________] │   ║
║  │  [Pembayaran bisa dilakukan via transfer bank________]│   ║
║  │  [atau scan QRIS di invoice ini.____________________] │   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║  ┌─── Email Invoice ────────────────────────────────────┐    ║
║  │  Email Signature:                                    │    ║
║  │  [Salam,_________________________________________]   │    ║
║  │  [Tim ISPBoss Net________________________________]   │    ║
║  │  [0812-xxxx-xxxx________________________________]   │    ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║  ┌─── Nomor Rekening (tampil di invoice) ────────────────┐   ║
║  │  [+ Tambah Rekening]                                  │   ║
║  │  ┌──────────┬──────────────┬──────────────────────┐   │   ║
║  │  │ Bank     │ No. Rekening │ Atas Nama            │   │   ║
║  │  ├──────────┼──────────────┼──────────────────────┤   │   ║
║  │  │ BCA      │ 123-456-789  │ PT ISPBoss Net       │   │   ║
║  │  │ BRI      │ 987-654-321  │ PT ISPBoss Net       │   │   ║
║  │  └──────────┴──────────────┴──────────────────────┘   │   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║                                          [Simpan]            ║
╚══════════════════════════════════════════════════════════════╝
```

---

## 🗺️ Peta Settings (`/settings/map`)

```
╔══════════════════════════════════════════════════════════════╗
║  Konfigurasi Peta                                            ║
║                                                              ║
║  ┌─── Geocoding Provider ────────────────────────────────┐   ║
║  │  ● Nominatim / OpenStreetMap (gratis)                 │   ║
║  │  ○ Google Geocoding (berbayar, lebih akurat)          │   ║
║  │    API Key: [••••••••••]                              │   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║  ┌─── Label Node ────────────────────────────────────────┐   ║
║  │  (konfigurasi label per tipe node — lihat dokumen 10) │   ║
║  │  [Konfigurasi Label →]                                │   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║  ┌─── Default Map Center ────────────────────────────────┐   ║
║  │  Latitude:  [-6.914744___]                            │   ║
║  │  Longitude: [107.609810__]                            │   ║
║  │  Zoom Level: [13___]                                  │   ║
║  │  [📍 Pilih dari Peta]                                │   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║                                          [Simpan]            ║
╚══════════════════════════════════════════════════════════════╝
```

---

## 📊 Laporan Settings (`/settings/reports`)

```
╔══════════════════════════════════════════════════════════════╗
║  Konfigurasi Laporan                                         ║
║                                                              ║
║  ┌─── Target KPI ────────────────────────────────────────┐   ║
║  │  (konfigurasi target — lihat dokumen 11)              │   ║
║  │  [Konfigurasi Target KPI →]                           │   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║  ┌─── Jadwal Laporan Otomatis ───────────────────────────┐   ║
║  │  (daftar jadwal — lihat dokumen 11)                   │   ║
║  │  [Kelola Jadwal Laporan →]                            │   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║  ┌─── Kategori Pengeluaran ──────────────────────────────┐   ║
║  │  (kelola kategori — lihat dokumen 11)                 │   ║
║  │  [Kelola Kategori Pengeluaran →]                      │   ║
║  └───────────────────────────────────────────────────────┘   ║
╚══════════════════════════════════════════════════════════════╝
```


---

## 🔐 Keamanan (`/settings/security`)

### Profil & Password

```
╔══════════════════════════════════════════════════════════════╗
║  Keamanan Akun                                               ║
║                                                              ║
║  ┌─── Ubah Password ────────────────────────────────────┐    ║
║  │  Password Lama:  [••••••••••      👁️]                │    ║
║  │  Password Baru:  [••••••••••      👁️]                │    ║
║  │  Konfirmasi:     [••••••••••      👁️]                │    ║
║  │  [Ubah Password]                                     │    ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║  ┌─── Two-Factor Authentication (2FA) ───────────────────┐   ║
║  │  Status: ⚫ Belum diaktifkan                          │   ║
║  │  [Aktifkan 2FA]                                       │   ║
║  │                                                       │   ║
║  │  Metode: Google Authenticator / Authy                 │   ║
║  │  Scan QR code → masukkan kode 6 digit → selesai      │   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║  ┌─── Session Management ────────────────────────────────┐   ║
║  │  Device aktif:                                        │   ║
║  │  ┌──────────────┬──────────────┬──────────┬────────┐  │   ║
║  │  │ Device       │ IP Address   │ Terakhir │ Aksi   │  │   ║
║  │  ├──────────────┼──────────────┼──────────┼────────┤  │   ║
║  │  │ Chrome Win10 │ 103.xx.xx.xx │ Sekarang │ (ini)  │  │   ║
║  │  │ Safari iPhone│ 182.xx.xx.xx │ 2 jam lalu│[Logout]│  │   ║
║  │  │ Firefox Linux│ 36.xx.xx.xx  │ 3 hari   │[Logout]│  │   ║
║  │  └──────────────┴──────────────┴──────────┴────────┘  │   ║
║  │  [Logout Semua Device Lain]                           │   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║  ┌─── API Key (untuk integrasi) ─────────────────────────┐   ║
║  │  API Key: [••••••••••••••••••••      👁️]  [Regenerate]│   ║
║  │  Dibuat: 15 Jan 2026                                  │   ║
║  │  Terakhir dipakai: 28 Apr 2026                        │   ║
║  │                                                       │   ║
║  │  ⚠️ Regenerate akan membatalkan key lama.             │   ║
║  └───────────────────────────────────────────────────────┘   ║
╚══════════════════════════════════════════════════════════════╝
```

---

## 📦 Subscription (`/settings/subscription`)

```
╔══════════════════════════════════════════════════════════════╗
║  Subscription ISPBoss                                        ║
║                                                              ║
║  ┌─── Paket Saat Ini ───────────────────────────────────┐    ║
║  │  Paket: Growth (101-500 pelanggan)                   │    ║
║  │  Harga: Rp 350.000/bulan                             │    ║
║  │  Pelanggan saat ini: 847 / 500 ⚠️ Melebihi limit    │    ║
║  │  Berlaku sampai: 5 Mei 2026                          │    ║
║  │  Status: 🟢 Aktif                                    │    ║
║  │                                                       │   ║
║  │  ⚠️ Pelanggan Anda melebihi limit paket Growth.      │   ║
║  │  Upgrade ke Pro untuk menghindari pembatasan.         │   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║  ┌─── Upgrade Paket ────────────────────────────────────┐    ║
║  │  ┌──────────┬──────────┬──────────────┬──────────┐   │    ║
║  │  │ Starter  │ Growth   │ Pro ⭐       │Enterprise│   │    ║
║  │  │ 0-100    │ 101-500  │ 501-2000     │ 2000+    │   │    ║
║  │  │ Rp150rb  │ Rp350rb  │ Rp750rb      │ Custom   │   │    ║
║  │  │          │ (saat ini)│ [Upgrade]   │[Hubungi] │   │    ║
║  │  └──────────┴──────────┴──────────────┴──────────┘   │    ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║  ┌─── Riwayat Pembayaran ISPBoss ────────────────────────┐   ║
║  │  ┌──────────┬──────────┬──────────┬──────────────────┐│   ║
║  │  │ Tanggal  │ Paket    │ Jumlah   │ Status           ││   ║
║  │  ├──────────┼──────────┼──────────┼──────────────────┤│   ║
║  │  │ 05/04/26 │ Growth   │ Rp 350rb │ ✅ Lunas         ││   ║
║  │  │ 05/03/26 │ Growth   │ Rp 350rb │ ✅ Lunas         ││   ║
║  │  │ 05/02/26 │ Starter  │ Rp 150rb │ ✅ Lunas         ││   ║
║  │  └──────────┴──────────┴──────────┴──────────────────┘│   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║  ┌─── Modul Aktif ──────────────────────────────────────┐    ║
║  │  ☑ Pelanggan (always on)                             │    ║
║  │  ☑ Billing (always on)                               │    ║
║  │  ☑ MikroTik                    [Nonaktifkan]         │    ║
║  │  ☐ OLT                         [Aktifkan]            │    ║
║  │  ☐ FTTH Mapping                 [Aktifkan]            │    ║
║  │  ☑ Notifikasi                  [Nonaktifkan]         │    ║
║  │  ☑ Laporan (always on)                               │    ║
║  └───────────────────────────────────────────────────────┘   ║
╚══════════════════════════════════════════════════════════════╝
```

### Module Registry
Sesuai dokumen 00, modul bisa diaktifkan/nonaktifkan per tenant:

| Modul | Default | Bisa Dinonaktifkan? |
|---|---|---|
| Core (auth, tenant) | Always on | ❌ |
| Pelanggan | Always on | ❌ |
| Paket | Always on | ❌ |
| Billing | Always on | ❌ |
| MikroTik | Enabled | ✅ |
| OLT | Disabled | ✅ |
| FTTH Mapping | Disabled | ✅ |
| Notifikasi | Enabled | ✅ |
| Laporan | Always on | ❌ |

- Modul yang dinonaktifkan → menu hidden, widget hidden, event diabaikan
- Modul yang diaktifkan setelah ada data → trigger initial sync wizard (MikroTik/OLT)

---

## 📋 Audit Log (`/settings/audit-log`)

```
╔══════════════════════════════════════════════════════════════════════════╗
║  Audit Log                                                               ║
║                                                                          ║
║  ┌───────────────────────────────────────────────────────────────────┐    ║
║  │ 🔍 Cari...  Filter: [User ▼] [Aksi ▼] [Modul ▼] [Periode ▼]    │    ║
║  └───────────────────────────────────────────────────────────────────┘    ║
║                                                                          ║
║  ┌──────────┬──────────────┬──────────────────────────────┬────────────┐ ║
║  │ Waktu    │ User         │ Aksi                         │ Modul      │ ║
║  ├──────────┼──────────────┼──────────────────────────────┼────────────┤ ║
║  │ 14:30    │ Admin Budi   │ Edit pelanggan PLG-001       │ Pelanggan  │ ║
║  │ 14:25    │ System       │ Isolir otomatis PLG-002      │ Billing    │ ║
║  │ 14:20    │ Kasir Ani    │ Catat pembayaran Rp 388.500  │ Pembayaran │ ║
║  │ 14:15    │ Teknisi Andi │ /ppp/active/remove session-5 │ MikroTik   │ ║
║  │ 14:10    │ System       │ Sync MK-01 completed (320 ok)│ MikroTik   │ ║
║  │ 14:05    │ Admin Budi   │ Login dari Chrome Win10      │ Auth       │ ║
║  │ 14:00    │ System       │ Invoice INV-2026-04-001 gen  │ Billing    │ ║
║  └──────────┴──────────────┴──────────────────────────────┴────────────┘ ║
║                                                                          ║
║  ◀ 1 2 3 ... ▶                                10 / 25 / 50 per halaman  ║
║                                                                          ║
║  [📥 Export CSV]                                                         ║
╚══════════════════════════════════════════════════════════════════════════╝
```

### Data yang Dicatat
| Kategori | Contoh Aksi |
|---|---|
| Auth | Login, logout, gagal login, reset password |
| Pelanggan | Tambah, edit, hapus, ganti paket, isolir, buka isolir |
| Billing | Generate invoice, catat pembayaran, void, batalkan invoice |
| MikroTik | Semua perintah ke router (add/set/remove user, reboot) |
| OLT | Provisioning ONT, decommission, reboot ONT |
| Notifikasi | Kirim notifikasi, broadcast |
| Settings | Ubah konfigurasi, tambah/hapus user |

- Log **append-only** (tidak bisa dihapus atau diedit)
- Retention: **12 bulan** (setelah itu auto-archive)
- Filter by: user, aksi, modul, periode
- Export ke CSV untuk audit eksternal

---

## Integrasi dengan Modul Lain

Dokumen 12 adalah **pusat konfigurasi** yang direferensikan oleh semua modul:

| Modul | Setting yang Dikonfigurasi |
|---|---|
| **Arsitektur (00)** | Module registry, RBAC roles |
| **Auth (02)** | 2FA, session management, API key |
| **Pelanggan (04)** | Username PPPoE format, area management |
| **Paket (05)** | Voucher masa berlaku, reseller settings |
| **Billing (06)** | Grace period, denda, pajak, reminder, payment gateway |
| **Notifikasi (07)** | Provider WA/SMS/Email, quiet hours, template |
| **MikroTik (08)** | Bandwidth method, isolir method, sync interval |
| **OLT (09)** | Signal threshold, VLAN strategy, auto-provisioning |
| **FTTH Mapping (10)** | Geocoding provider, label node, default map center |
| **Laporan (11)** | Target KPI, jadwal laporan, kategori pengeluaran |

---

## Keputusan

| Keputusan | Detail |
|---|---|
| Akses settings | Hanya Tenant Admin. User lain hanya bisa edit profil sendiri |
| Profil ISP | ✅ Nama, alamat, telepon, email, NPWP, timezone |
| White label | ✅ Logo, favicon, warna primer, custom domain (fase lanjut) |
| User management | ✅ CRUD user, 4 role (Admin, Operator, Teknisi, Kasir), preferensi notifikasi per user |
| Billing settings | ✅ Semua konfigurasi dari dokumen 06 (grace period, denda, pajak, reminder, dll) |
| Payment gateway | ✅ Xendit + Midtrans, API key terenkripsi, sandbox mode, channel selection |
| Notifikasi settings | ✅ Referensi ke dokumen 07 (provider, template, quiet hours) |
| MikroTik settings | ✅ Bandwidth method, isolir method, PPPoE format, sync interval |
| OLT settings | ✅ Signal threshold, VLAN strategy, auto-provisioning |
| Peta settings | ✅ Geocoding provider, label node, default map center |
| Laporan settings | ✅ Target KPI, jadwal laporan, kategori pengeluaran |
| Keamanan | ✅ Ubah password, 2FA (Google Authenticator), session management, API key |
| Subscription | ✅ Lihat paket saat ini, upgrade, riwayat pembayaran, module registry |
| Module registry | ✅ Aktifkan/nonaktifkan modul per tenant (MikroTik, OLT, FTTH, Notifikasi) |
| Audit log | ✅ Append-only, 12 bulan retention, filter, export CSV |
| Custom domain | ✅ Fase lanjut — CNAME + auto SSL (Let's Encrypt) |
| Voucher settings | ✅ Format kode, masa berlaku, max generate, collision retry |
| Lokalisasi | ✅ Format tanggal, mata uang, pemisah ribuan, bahasa interface |
| Invoice customization | ✅ Footer custom, email signature, nomor rekening bank |
| Walled garden config | ✅ Hosting (ISPBoss/custom), pesan custom, tampilkan tombol bayar/kontak/rekening |