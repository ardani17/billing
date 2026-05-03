# 07 — Notifikasi (WhatsApp, SMS, Email)

---

## Konsep Notifikasi

Notification Service adalah service terpisah (Golang) yang menangani semua pengiriman pesan ke pelanggan dan admin. Service ini menerima event dari Billing API via Redis queue dan mengirim pesan melalui provider yang dikonfigurasi tenant.

```
Billing API / Network Service
  │
  ├── Event: invoice.created
  ├── Event: payment.received
  ├── Event: customer.isolated
  ├── Event: customer.activated
  └── Event: ...
  │
  ▼
Redis Queue (asynq)
  │
  ▼
Notification Service
  │
  ├── Resolve template + variabel
  ├── Pilih channel (WA/SMS/Email)
  ├── Kirim via provider adapter
  └── Log hasil pengiriman
```

---

## Channel Notifikasi

| Channel | Provider | Prioritas | Keterangan |
|---|---|---|---|
| **WhatsApp** | Fonnte, WaBlas, WA Business API | 🥇 Utama | Paling umum di Indonesia, open rate tinggi |
| **SMS** | Zenziva, Twilio, Nexmo | 🥈 Fallback | Jika WA gagal atau pelanggan tidak punya WA |
| **Email** | SMTP, Mailgun, SendGrid | 🥉 Opsional | Untuk invoice PDF, laporan, pelanggan korporat |

### Prioritas Pengiriman (Configurable)
```
Default: WhatsApp → SMS (fallback) → Email (fallback)

Tenant bisa atur:
  - Hanya WhatsApp
  - WhatsApp + SMS fallback
  - WhatsApp + Email fallback
  - Semua channel sekaligus (broadcast)
  - Per tipe notifikasi (misal: invoice via Email, reminder via WA)
```

---

## Adapter Pattern (Multi-Provider)

Setiap channel menggunakan adapter pattern agar bisa ganti provider tanpa ubah business logic:

```
┌─────────────────────────────────────────────────────┐
│ NotificationService                                  │
│                                                     │
│  ┌─── WhatsApp Adapter ──────────────────────────┐  │
│  │  interface: WAProvider                         │  │
│  │  ├── FonnteAdapter                            │  │
│  │  ├── WaBlasAdapter                            │  │
│  │  └── WABusinessAPIAdapter                     │  │
│  └───────────────────────────────────────────────┘  │
│                                                     │
│  ┌─── SMS Adapter ───────────────────────────────┐  │
│  │  interface: SMSProvider                        │  │
│  │  ├── ZenzivaAdapter                           │  │
│  │  ├── TwilioAdapter                            │  │
│  │  └── NexmoAdapter                             │  │
│  └───────────────────────────────────────────────┘  │
│                                                     │
│  ┌─── Email Adapter ────────────────────────────┐   │
│  │  interface: EmailProvider                     │   │
│  │  ├── SMTPAdapter                              │   │
│  │  ├── MailgunAdapter                           │   │
│  │  └── SendGridAdapter                          │   │
│  └───────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────┘
```

- Tenant pilih provider mana yang mau dipakai di Settings
- API key / credential disimpan terenkripsi per tenant
- Bisa ganti provider tanpa downtime


---

## Template Notifikasi

### Daftar Template Bawaan (Default)

| ID | Nama Template | Trigger | Channel Default |
|---|---|---|---|
| `invoice_new` | Invoice Baru | Invoice digenerate | WA + Email |
| `reminder_h1` | Reminder H-1 | 1 hari sebelum jatuh tempo | WA |
| `reminder_h_plus1` | Reminder H+1 | 1 hari setelah jatuh tempo | WA |
| `reminder_h_plus3` | Peringatan Terakhir | 3 hari setelah jatuh tempo | WA + SMS |
| `isolir_notice` | Notifikasi Isolir | Saat pelanggan diisolir | WA + SMS |
| `suspend_notice` | Notifikasi Suspend | Saat pelanggan di-suspend | WA + SMS |
| `payment_confirm` | Konfirmasi Pembayaran | Pembayaran diterima | WA |
| `unblock_notice` | Buka Isolir | Isolir dibuka setelah bayar | WA |
| `welcome` | Selamat Datang | Pelanggan baru diaktivasi | WA + Email |
| `package_change` | Ganti Paket | Upgrade/downgrade paket | WA |
| `price_change` | Perubahan Harga | Harga paket berubah | WA + Email |
| `maintenance` | Info Maintenance | Jadwal maintenance jaringan | WA |
| `custom` | Pesan Custom | Admin kirim manual | WA/SMS/Email |

### Variabel Template

Variabel yang bisa dipakai di semua template:

| Variabel | Contoh | Keterangan |
|---|---|---|
| `{nama}` | Ahmad Rizki | Nama pelanggan |
| `{id_pelanggan}` | PLG-001 | ID pelanggan |
| `{nama_isp}` | ISPBoss Net | Nama ISP tenant |
| `{telepon_isp}` | 0812-xxxx-xxxx | Telepon ISP |
| `{paket}` | Pro 50M | Nama paket |
| `{harga}` | Rp 350.000 | Harga paket |
| `{periode}` | April 2026 | Periode tagihan |
| `{no_invoice}` | INV-2026-04-001 | Nomor invoice |
| `{total_tagihan}` | Rp 388.500 | Total tagihan |
| `{jatuh_tempo}` | 5 April 2026 | Tanggal jatuh tempo |
| `{sisa_hari}` | 3 | Sisa hari sebelum isolir |
| `{terlambat_hari}` | 15 | Jumlah hari terlambat |
| `{link_bayar}` | https://pay.xendit.co/... | Link pembayaran online |
| `{link_bayar_short}` | https://isb.id/p/abc12 | Short URL payment link (untuk SMS) |
| `{tanggal_bayar}` | 3 April 2026 | Tanggal pembayaran |
| `{metode_bayar}` | VA BCA | Metode pembayaran |
| `{jumlah_bayar}` | Rp 388.500 | Jumlah yang dibayar |

### Contoh Template WhatsApp

**Invoice Baru (`invoice_new`):**
```
Halo {nama} 👋

Invoice bulan {periode} sudah terbit.

📄 No: {no_invoice}
💰 Total: {total_tagihan}
📅 Jatuh Tempo: {jatuh_tempo}

Bayar sekarang: {link_bayar}

Terima kasih 🙏
{nama_isp}
```

**Reminder H+1 (`reminder_h_plus1`):**
```
Halo {nama},

Tagihan bulan {periode} sebesar {total_tagihan} sudah lewat jatuh tempo.

Segera bayar sebelum {sisa_hari} hari lagi untuk menghindari isolir.

Bayar: {link_bayar}

{nama_isp} • {telepon_isp}
```

**Notifikasi Isolir (`isolir_notice`):**
```
⚠️ {nama}, layanan internet Anda diisolir karena tunggakan {terlambat_hari} hari.

Tagihan: {total_tagihan}
Bayar sekarang untuk mengaktifkan kembali: {link_bayar}

Hubungi kami: {telepon_isp}
{nama_isp}
```

**Konfirmasi Pembayaran (`payment_confirm`):**
```
✅ Pembayaran diterima!

Pelanggan: {nama} ({id_pelanggan})
Invoice: {no_invoice}
Jumlah: {jumlah_bayar}
Metode: {metode_bayar}
Tanggal: {tanggal_bayar}

Terima kasih atas pembayaran Anda 🙏
{nama_isp}
```

**Selamat Datang (`welcome`):**
```
Selamat datang di {nama_isp}! 🎉

Halo {nama}, layanan internet Anda sudah aktif.

📦 Paket: {paket}
💰 Harga: {harga}/bulan
📅 Jatuh Tempo: Setiap tanggal {jatuh_tempo}

Jika ada kendala, hubungi kami di {telepon_isp}.

Selamat menikmati internet cepat! 🚀
{nama_isp}
```

### Contoh Template SMS (Ringkas, Max 160 Karakter)

SMS punya keterbatasan: max 160 karakter (ASCII) atau 70 karakter (Unicode/emoji). Template SMS harus **terpisah dan lebih ringkas** dari WA.

**Invoice Baru (`invoice_new`):**
```
{nama_isp}: Invoice {periode} Rp{total_tagihan} jatuh tempo {jatuh_tempo}. Bayar: {link_bayar_short}
```

**Reminder H+1 (`reminder_h_plus1`):**
```
{nama_isp}: Tagihan {periode} Rp{total_tagihan} sudah lewat. Bayar sebelum isolir: {link_bayar_short}
```

**Notifikasi Isolir (`isolir_notice`):**
```
{nama_isp}: Internet diisolir. Tagihan Rp{total_tagihan}. Bayar: {link_bayar_short} atau hub {telepon_isp}
```

**Konfirmasi Pembayaran (`payment_confirm`):**
```
{nama_isp}: Pembayaran Rp{jumlah_bayar} diterima untuk {no_invoice}. Terima kasih!
```

> **Catatan:** Variabel `{link_bayar_short}` adalah versi pendek dari payment link. Sistem otomatis generate short URL (misal via bit.ly atau custom shortener) untuk SMS.


---

## Halaman Kelola Template (`/notifications/templates`)

### Mobile Layout

```
╔══════════════════════════╗
║  ☰  Notifikasi      🔍  ║
╠══════════════════════════╣
║                          ║
║  [Template] [Log] [Broadcast]║
║  ← swipe tab →           ║
║                          ║
║  ┌──────────────────────┐║
║  │ Invoice Baru   🟢Aktif│║
║  │ Otomatis • WA+Email  │║
║  │ Terakhir: 847x  [⋯]  │║
║  ├──────────────────────┤║
║  │ Reminder H-1   🟢Aktif│║
║  │ Otomatis • WA         │║
║  │ Terakhir: 87x   [⋯]  │║
║  ├──────────────────────┤║
║  │ Promo Ramadhan ⚫Off  │║
║  │ Manual • WA           │║
║  │ Terakhir: 0x    [⋯]  │║
║  └──────────────────────┘║
║                          ║
║  [+ Buat Template]       ║
╚══════════════════════════╝
```

### Desktop Layout

```
╔══════════════════════════════════════════════════════════════════════╗
║  Dashboard > Notifikasi > Template                                   ║
║                                                                      ║
║  Template Notifikasi                            [+ Buat Template]    ║
║                                                                      ║
║  ┌──────────────────┬──────────┬──────────┬────────┬──────┬───────┐  ║
║  │ Nama Template    │ Trigger  │ Channel  │ Status │ Terakhir│Aksi │  ║
║  ├──────────────────┼──────────┼──────────┼────────┼────────┼──────┤  ║
║  │ Invoice Baru     │ Otomatis │ WA+Email │ 🟢Aktif│ 847x   │ ⋯   │  ║
║  │ Reminder H-1     │ Otomatis │ WA       │ 🟢Aktif│ 87x    │ ⋯   │  ║
║  │ Notifikasi Isolir│ Otomatis │ WA+SMS   │ 🟢Aktif│ 40x    │ ⋯   │  ║
║  │ Konfirmasi Bayar │ Otomatis │ WA       │ 🟢Aktif│ 720x   │ ⋯   │  ║
║  │ Info Maintenance │ Manual   │ WA       │ 🟢Aktif│ 3x     │ ⋯   │  ║
║  │ Promo Ramadhan   │ Manual   │ WA       │ ⚫Nonaktif│ 0x   │ ⋯   │  ║
║  └──────────────────┴──────────┴──────────┴────────┴────────┴──────┘  ║
╚══════════════════════════════════════════════════════════════════════╝
```

### Form Edit Template

```
╔══════════════════════════════════════════════════════════════╗
║  Edit Template — Invoice Baru                                ║
║                                                              ║
║  Nama Template *                                             ║
║  [Invoice Baru___________]                                   ║
║                                                              ║
║  Channel *                                                   ║
║  ☑ WhatsApp  ☐ SMS  ☑ Email                                 ║
║                                                              ║
║  ┌─── Pesan WhatsApp ────────────────────────────────────┐   ║
║  │  Halo {nama} 👋                                       │   ║
║  │                                                       │   ║
║  │  Invoice bulan {periode} sudah terbit.                │   ║
║  │  📄 No: {no_invoice}                                  │   ║
║  │  💰 Total: {total_tagihan}                            │   ║
║  │  📅 Jatuh Tempo: {jatuh_tempo}                        │   ║
║  │                                                       │   ║
║  │  Bayar sekarang: {link_bayar}                         │   ║
║  │                                                       │   ║
║  │  Terima kasih 🙏                                      │   ║
║  │  {nama_isp}                                           │   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║  Variabel tersedia:                                          ║
║  [nama] [id_pelanggan] [paket] [harga] [periode]            ║
║  [no_invoice] [total_tagihan] [jatuh_tempo] [link_bayar]    ║
║  (klik untuk insert)                                         ║
║                                                              ║
║  ┌─── Subject Email ─────────────────────────────────────┐   ║
║  │  Invoice {periode} — {nama_isp}                       │   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║  ┌─── Body Email (HTML) ─────────────────────────────────┐   ║
║  │  [Rich text editor dengan template HTML]              │   ║
║  │  Bisa attach invoice PDF otomatis                     │   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║  [Preview]  [Kirim Test]           [Batal]  [Simpan]         ║
╚══════════════════════════════════════════════════════════════╝
```

### Fitur Template
- **Preview**: Lihat hasil template dengan data contoh
- **Kirim Test**: Kirim ke nomor/email admin untuk test
- **Variabel klik-insert**: Klik variabel untuk insert ke posisi cursor
- **Template bawaan** tidak bisa dihapus, tapi bisa diedit isinya
- **Template custom** bisa dibuat untuk kebutuhan khusus (promo, info, dll)
- **Version history**: Simpan 10 versi terakhir per template, bisa rollback ke versi sebelumnya
- **Character counter**: Untuk template SMS, tampilkan jumlah karakter real-time (warning jika > 160)


---

## Pengiriman Otomatis vs Manual

### Notifikasi Otomatis
Dikirim oleh sistem berdasarkan event/trigger:

| Event | Template | Timing |
|---|---|---|
| Invoice digenerate | `invoice_new` | Langsung setelah generate |
| H-1 jatuh tempo | `reminder_h1` | Cron job harian |
| H+1 jatuh tempo | `reminder_h_plus1` | Cron job harian |
| H+3 jatuh tempo | `reminder_h_plus3` | Cron job harian |
| Pelanggan diisolir | `isolir_notice` | Langsung setelah isolir |
| Pelanggan di-suspend | `suspend_notice` | Langsung setelah suspend |
| Pembayaran diterima | `payment_confirm` | Langsung setelah bayar |
| Isolir dibuka | `unblock_notice` | Langsung setelah buka isolir |
| Pelanggan baru aktif | `welcome` | Langsung setelah aktivasi |
| Ganti paket | `package_change` | Langsung setelah ganti |
| Harga paket berubah | `price_change` | 30 hari sebelum berlaku |

- Setiap notifikasi otomatis bisa **diaktifkan/nonaktifkan** per template
- Jadwal reminder mengikuti konfigurasi di dokumen 06 (Billing)

### Notifikasi Manual
Admin kirim pesan langsung ke pelanggan:

**Kirim ke 1 pelanggan:**
- Dari detail pelanggan → tombol "Kirim Notifikasi"
- Pilih template atau tulis pesan custom
- Pilih channel (WA/SMS/Email)

**Kirim ke beberapa pelanggan (Bulk):**
- Dari tabel pelanggan → centang → "Kirim Notifikasi"
- Pilih template
- Proses async (background job)

---

## Broadcast Massal (`/notifications/broadcast`)

Untuk kirim pesan ke banyak pelanggan sekaligus (promo, info maintenance, pengumuman):

```
╔══════════════════════════════════════════════════════════════╗
║  Dashboard > Notifikasi > Broadcast                          ║
║                                                              ║
║  Broadcast Baru                                              ║
║                                                              ║
║  Nama Broadcast *                                            ║
║  [Info Maintenance 28 April_____]                            ║
║                                                              ║
║  Penerima *                                                  ║
║  ○ Semua pelanggan aktif                                     ║
║  ○ Filter berdasarkan:                                       ║
║    ☑ Paket: [Pro 50M ▼]                                     ║
║    ☐ Area: [Pilih Area ▼]                                   ║
║    ☐ Status: [Pilih Status ▼]                                ║
║  ○ Pilih manual (dari tabel pelanggan)                       ║
║                                                              ║
║  Jumlah penerima: 412 pelanggan                              ║
║                                                              ║
║  Template *                                                  ║
║  [Pilih Template ▼] atau [Tulis Custom]                     ║
║                                                              ║
║  Channel *                                                   ║
║  ☑ WhatsApp  ☐ SMS  ☐ Email                                 ║
║                                                              ║
║  Jadwal Kirim *                                              ║
║  ● Kirim sekarang                                            ║
║  ○ Jadwalkan: [dd/mm/yyyy] [hh:mm]                          ║
║                                                              ║
║  [Preview]  [Kirim Test]     [Batal]  [Kirim Broadcast]     ║
╚══════════════════════════════════════════════════════════════╝
```

### Fitur Broadcast
- **Filter penerima**: berdasarkan paket, area, status, atau pilih manual
- **Jadwal kirim**: langsung atau dijadwalkan (scheduled)
- **Preview**: lihat pesan final sebelum kirim
- **Kirim test**: kirim ke admin dulu untuk verifikasi
- **Proses async**: background job, tidak blocking UI
- **Progress tracking**: lihat progress pengiriman real-time
- **Rate limiting**: max 50 pesan/detik (mencegah throttle dari provider)
- **Pause & Resume**: broadcast yang sedang berjalan bisa di-pause dan dilanjutkan
- **Cancel**: broadcast yang dijadwalkan tapi belum mulai bisa dibatalkan
- **Exclude opt-out**: pelanggan yang opt-out dari broadcast otomatis di-skip

### Riwayat Broadcast

```
┌──────────────────────────────────────────────────────────────────┐
│ Riwayat Broadcast                                                │
│                                                                  │
│ ┌──────────────────┬──────────┬──────────┬──────────┬──────────┐ │
│ │ Nama             │ Tanggal  │ Penerima │ Terkirim │ Gagal    │ │
│ ├──────────────────┼──────────┼──────────┼──────────┼──────────┤ │
│ │ Info Maintenance │ 28/04/26 │ 412      │ 408      │ 4        │ │
│ │ Promo Ramadhan   │ 01/03/26 │ 847      │ 830      │ 17       │ │
│ │ Upgrade Jaringan │ 15/02/26 │ 200      │ 198      │ 2        │ │
│ └──────────────────┴──────────┴──────────┴──────────┴──────────┘ │
└──────────────────────────────────────────────────────────────────┘
```


---

## Log Pengiriman (`/notifications/logs`)

### Halaman Log

```
╔══════════════════════════════════════════════════════════════════════════╗
║  Dashboard > Notifikasi > Log Pengiriman                                 ║
║                                                                          ║
║  Log Pengiriman                                                          ║
║                                                                          ║
║  ┌────────────┬────────────┬────────────┬────────────┐                   ║
║  │ 📨 Total   │ ✅ Terkirim │ ❌ Gagal    │ ⏳ Pending  │                   ║
║  │ 2,450      │ 2,380      │ 52         │ 18         │                   ║
║  │ Bulan ini  │ 97.1%      │ 2.1%       │ 0.7%       │                   ║
║  └────────────┴────────────┴────────────┴────────────┘                   ║
║                                                                          ║
║  ┌───────────────────────────────────────────────────────────────────┐    ║
║  │ 🔍 Cari pelanggan, no telepon...                                  │    ║
║  ├───────────────────────────────────────────────────────────────────┤    ║
║  │ Filter: [Channel ▼] [Status ▼] [Template ▼] [Periode ▼] [Reset]  │    ║
║  └───────────────────────────────────────────────────────────────────┘    ║
║                                                                          ║
║  ┌──────────┬───────────┬──────────┬──────────┬────────┬──────┬───────┐  ║
║  │ Waktu    │ Pelanggan │ Channel  │ Template │ Status │ Retry│ Detail│  ║
║  ├──────────┼───────────┼──────────┼──────────┼────────┼──────┼───────┤  ║
║  │ 28/04/26 │ Ahmad R.  │ 📱 WA    │ Invoice  │ ✅Sent │ -    │ 👁️    │  ║
║  │ 14:30    │ PLG-001   │          │ Baru     │        │      │       │  ║
║  ├──────────┼───────────┼──────────┼──────────┼────────┼──────┼───────┤  ║
║  │ 28/04/26 │ Budi S.   │ 📱 WA    │ Reminder │ ❌Fail │ 3/3  │ 👁️    │  ║
║  │ 08:00    │ PLG-002   │          │ H+1      │        │      │       │  ║
║  ├──────────┼───────────┼──────────┼──────────┼────────┼──────┼───────┤  ║
║  │ 28/04/26 │ Citra D.  │ 📧 Email │ Invoice  │ ✅Sent │ -    │ 👁️    │  ║
║  │ 00:05    │ PLG-003   │          │ Baru     │        │      │       │  ║
║  └──────────┴───────────┴──────────┴──────────┴────────┴──────┴───────┘  ║
║                                                                          ║
║  ◀ 1 2 3 ... ▶                                10 / 25 / 50 per halaman  ║
╚══════════════════════════════════════════════════════════════════════════╝
```

### Status Pengiriman
| Status | Warna | Arti |
|---|---|---|
| ⏳ Pending | Amber | Dalam antrian, belum dikirim |
| 📤 Sending | Blue | Sedang dikirim ke provider |
| ✅ Sent | Green | Berhasil dikirim ke provider |
| ✅✅ Delivered | Dark Green | Terkirim ke perangkat penerima (jika provider support) |
| 📖 Read | Purple | Dibaca oleh penerima (jika provider support, WA only) |
| ❌ Failed | Red | Gagal dikirim setelah semua retry |
| 🔄 Retrying | Orange | Sedang retry setelah gagal |

### Detail Log (Klik 👁️)

```
┌──────────────────────────────────────────────────────────┐
│ Detail Notifikasi                                        │
│                                                          │
│ Pelanggan: Budi Santoso (PLG-002)                        │
│ Telepon: +6281345678901                                  │
│ Channel: WhatsApp (Fonnte)                               │
│ Template: Reminder H+1                                   │
│ Trigger: Otomatis (cron billing)                         │
│                                                          │
│ Pesan:                                                   │
│ ┌──────────────────────────────────────────────────────┐ │
│ │ Halo Budi Santoso,                                   │ │
│ │                                                      │ │
│ │ Tagihan bulan April 2026 sebesar Rp 150.000 sudah   │ │
│ │ lewat jatuh tempo. Segera bayar sebelum 3 hari lagi │ │
│ │ untuk menghindari isolir.                            │ │
│ │                                                      │ │
│ │ Bayar: https://pay.xendit.co/abc123                  │ │
│ │                                                      │ │
│ │ ISPBoss Net • 0812-xxxx-xxxx                        │ │
│ └──────────────────────────────────────────────────────┘ │
│                                                          │
│ Timeline:                                                │
│ • 28/04/26 08:00 — Pending (masuk antrian)               │
│ • 28/04/26 08:00 — Sending (kirim ke Fonnte)             │
│ • 28/04/26 08:01 — Failed (timeout dari Fonnte)          │
│ • 28/04/26 08:06 — Retrying (retry 1/3)                  │
│ • 28/04/26 08:06 — Failed (nomor tidak terdaftar WA)     │
│ • 28/04/26 08:11 — Retrying (retry 2/3, fallback SMS)    │
│ • 28/04/26 08:11 — Failed (saldo SMS habis)              │
│ • 28/04/26 08:16 — Failed (max retry reached)            │
│                                                          │
│ Error: Nomor tidak terdaftar di WhatsApp, saldo SMS habis│
│                                                          │
│ [Kirim Ulang]  [Kirim via Channel Lain]  [Tutup]         │
└──────────────────────────────────────────────────────────┘
```

### Retry Mechanism
| Percobaan | Interval | Aksi |
|---|---|---|
| 1 (pertama) | Langsung | Kirim via channel utama |
| 2 | 5 menit | Retry channel utama |
| 3 | 5 menit | Fallback ke channel berikutnya (WA → SMS → Email) |
| Semua gagal | — | Tandai Failed, admin bisa kirim ulang manual |

- Max retry: **3 kali** per notifikasi
- Fallback otomatis ke channel berikutnya setelah retry habis
- Admin bisa **kirim ulang manual** dari detail log
- Notifikasi gagal yang banyak → alert ke admin

### Deduplication (Pencegahan Duplikat)
Mencegah notifikasi duplikat jika event dikirim 2x (retry dari billing API, cron overlap):
- **Dedup key**: `{tenant_id}:{pelanggan_id}:{template_id}:{periode}`
- Contoh: `t001:plg001:invoice_new:2026-04` → hanya 1 invoice_new per pelanggan per periode
- Window dedup: **1 jam** (notifikasi dengan key yang sama dalam 1 jam dianggap duplikat)
- Duplikat yang terdeteksi → log sebagai "Skipped (duplicate)" tanpa kirim

### Deteksi Nomor Tidak Aktif
Jika pengiriman ke nomor yang sama gagal berturut-turut:
- **3x gagal berturut-turut** ke nomor yang sama (lintas notifikasi berbeda) → tandai nomor sebagai ⚠️ **"Suspect"**
- Tampilkan warning di detail pelanggan: "Nomor WA mungkin tidak aktif — terakhir gagal {tanggal}"
- Notifikasi ke admin: "5 pelanggan dengan nomor suspect bulan ini"
- Admin bisa update nomor pelanggan atau tandai sebagai "Verified" setelah konfirmasi


---

## Konfigurasi Provider per Tenant (`/settings/notifications`)

### Halaman Konfigurasi

```
╔══════════════════════════════════════════════════════════════╗
║  Settings > Notifikasi                                       ║
║                                                              ║
║  ┌─── WhatsApp ──────────────────────────────────────────┐   ║
║  │  Provider *                                           │   ║
║  │  ● Fonnte  ○ WaBlas  ○ WA Business API               │   ║
║  │                                                       │   ║
║  │  API Token *                                          │   ║
║  │  [••••••••••••••••••••]  [👁️ Show]                    │   ║
║  │                                                       │   ║
║  │  Nomor Pengirim                                       │   ║
║  │  [+628123456789_______]                               │   ║
║  │                                                       │   ║
║  │  Status: 🟢 Terhubung (saldo: 12.500 pesan)          │   ║
║  │  [Test Kirim]  [Cek Saldo]                            │   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║  ┌─── SMS ───────────────────────────────────────────────┐   ║
║  │  ☐ Aktifkan SMS (sebagai fallback)                    │   ║
║  │                                                       │   ║
║  │  Provider: [Zenziva ▼]                                │   ║
║  │  API Key: [••••••••••]                                │   ║
║  │  User Key: [••••••••••]                               │   ║
║  │                                                       │   ║
║  │  Status: ⚫ Belum dikonfigurasi                       │   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║  ┌─── Email ─────────────────────────────────────────────┐   ║
║  │  ☐ Aktifkan Email                                     │   ║
║  │                                                       │   ║
║  │  Provider: ○ SMTP  ○ Mailgun  ○ SendGrid              │   ║
║  │  SMTP Host: [smtp.gmail.com___]                       │   ║
║  │  SMTP Port: [587___]                                  │   ║
║  │  Username: [admin@ispboss.net_]                       │   ║
║  │  Password: [••••••••••]                               │   ║
║  │  From Name: [ISPBoss Net______]                       │   ║
║  │  From Email: [billing@ispboss.net]                    │   ║
║  │                                                       │   ║
║  │  Status: ⚫ Belum dikonfigurasi                       │   ║
║  │  [Test Kirim]                                         │   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║  ┌─── Pengaturan Umum ───────────────────────────────────┐   ║
║  │  Prioritas Channel:                                   │   ║
║  │  1. [WhatsApp ▼]  2. [SMS ▼]  3. [Email ▼]          │   ║
║  │  (drag & drop untuk ubah urutan)                      │   ║
║  │                                                       │   ║
║  │  Jam Kirim Notifikasi:                                │   ║
║  │  Mulai: [07:00]  Sampai: [21:00]                     │   ║
║  │  (notifikasi di luar jam ini ditunda ke jam mulai)    │   ║
║  │                                                       │   ║
║  │  Rate Limit Broadcast:                                │   ║
║  │  [50] pesan per detik                                 │   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║                                          [Simpan Pengaturan] ║
╚══════════════════════════════════════════════════════════════╝
```

### Pengaturan Penting
| Setting | Default | Keterangan |
|---|---|---|
| Provider WA | Fonnte | Provider WhatsApp yang dipakai |
| Fallback SMS | Nonaktif | Aktifkan jika mau fallback ke SMS |
| Fallback Email | Nonaktif | Aktifkan jika mau fallback ke Email |
| Prioritas Channel | WA → SMS → Email | Urutan pengiriman dan fallback |
| Jam Kirim | 07:00 — 21:00 | Notifikasi di luar jam ini ditunda (quiet hours) |
| Rate Limit | 50/detik | Mencegah throttle dari provider |

### Quiet Hours (Jam Tenang)
- Notifikasi otomatis **tidak dikirim** di luar jam yang ditentukan
- Default: 07:00 — 21:00 (WIB, sesuai timezone tenant)
- Notifikasi yang seharusnya dikirim di luar jam → masuk antrian, dikirim saat jam mulai
- **Exception**: notifikasi pembayaran diterima dan buka isolir tetap dikirim kapan saja (pelanggan menunggu konfirmasi)

### Saldo & Monitoring Provider
- Tampilkan **sisa saldo** provider (jika API support)
- **Alert ke admin** jika saldo di bawah threshold (misal < 500 pesan)
- **Health check** berkala ke provider (setiap 5 menit)
- Jika provider down → otomatis fallback ke channel berikutnya


---

## Notifikasi ke Admin (Internal)

Selain notifikasi ke pelanggan, sistem juga kirim notifikasi ke admin/operator:

| Event | Penerima | Channel |
|---|---|---|
| Saldo provider WA/SMS rendah | Tenant Admin | WA + Email |
| Provider down / error rate tinggi | Tenant Admin | Email |
| Pembayaran besar diterima (> threshold) | Tenant Admin, Kasir | WA |
| Double payment terdeteksi | Tenant Admin | WA + Email |
| Pelanggan baru mendaftar | Operator | WA |
| Broadcast selesai | Pengirim broadcast | WA |
| Generate invoice selesai | Tenant Admin | WA |
| Banyak notifikasi gagal (> 10% error rate) | Tenant Admin | Email |
| Router MikroTik offline | Teknisi | WA |
| Sync MikroTik gagal (pending sync) | Teknisi | WA |

- Admin bisa atur notifikasi mana yang mau diterima di **Settings > Profil > Notifikasi**
- Setiap role bisa punya preferensi notifikasi berbeda

---

## Notifikasi In-App (Dashboard)

Selain WA/SMS/Email, ada notifikasi di dalam dashboard:

```
┌──────────────────────────────────────────┐
│ 🔔 Notifikasi                    [Semua] │
│ ─────────────────────────────────────── │
│ 🟢 5 menit lalu                         │
│ Pembayaran Rp 388.500 dari Ahmad Rizki   │
│ via Xendit (VA BCA)                      │
│ ─────────────────────────────────────── │
│ 🟡 15 menit lalu                        │
│ 3 pelanggan diisolir otomatis            │
│ ─────────────────────────────────────── │
│ 🔴 1 jam lalu                           │
│ Saldo Fonnte tinggal 200 pesan!          │
│ ─────────────────────────────────────── │
│ ⚪ 2 jam lalu                           │
│ Broadcast "Info Maintenance" selesai     │
│ 408/412 terkirim                         │
└──────────────────────────────────────────┘
```

- Bell icon di topbar dengan badge jumlah unread
- Klik notifikasi → navigasi ke halaman terkait
- Mark as read / mark all as read
- Realtime via **WebSocket / SSE**
- Tersimpan 30 hari, setelah itu auto-delete

---

## Opt-Out / Unsubscribe Pelanggan

Pelanggan bisa memilih untuk tidak menerima notifikasi tertentu:

### Kategori Notifikasi
| Kategori | Bisa Opt-Out? | Contoh |
|---|---|---|
| **Transaksional** | ❌ Tidak | Invoice, konfirmasi bayar, isolir, buka isolir, welcome |
| **Reminder** | ✅ Ya | Reminder H-1, H+1, H+3 |
| **Promosi** | ✅ Ya | Broadcast promo, info diskon |
| **Informasi** | ✅ Ya | Info maintenance, pengumuman umum |

### Mekanisme Opt-Out
- **WhatsApp**: Pelanggan balas "STOP" → sistem otomatis tandai opt-out untuk kategori promosi & informasi
- **Email**: Link "Unsubscribe" di footer setiap email → halaman preferensi notifikasi
- **SMS**: Balas "STOP" → opt-out dari SMS
- Admin bisa lihat dan override preferensi opt-out per pelanggan di detail pelanggan

> **Catatan:** Notifikasi transaksional (invoice, isolir, konfirmasi bayar) **tidak bisa** di-opt-out karena terkait langsung dengan layanan.

---

## Anti-Spam / Throttle per Pelanggan

Mencegah pelanggan menerima terlalu banyak pesan dalam waktu singkat:

| Setting | Default | Keterangan |
|---|---|---|
| Max pesan per pelanggan per hari | 5 | Tidak termasuk konfirmasi bayar & buka isolir |
| Cooldown antar pesan | 30 menit | Minimum jarak antar pesan ke pelanggan yang sama |
| Exception | Konfirmasi bayar, buka isolir | Selalu dikirim tanpa throttle |

### Alur Throttle
```
Notifikasi masuk untuk pelanggan X
  │
  ├── Cek: sudah berapa pesan hari ini?
  │     → ≥ 5: skip, log "Throttled (daily limit)"
  │
  ├── Cek: kapan pesan terakhir ke pelanggan ini?
  │     → < 30 menit: tunda ke 30 menit setelah pesan terakhir
  │
  ├── Cek: pelanggan opt-out untuk kategori ini?
  │     → Ya: skip, log "Skipped (opt-out)"
  │
  └── Kirim notifikasi
```

---

## Biaya / Cost Tracking

Setiap pengiriman notifikasi dicatat biayanya untuk monitoring:

### Estimasi Biaya per Provider
| Provider | Channel | Biaya per Pesan | Keterangan |
|---|---|---|---|
| Fonnte | WA | ~Rp 50-100 | Tergantung paket |
| WaBlas | WA | ~Rp 75-150 | Tergantung paket |
| Zenziva | SMS | ~Rp 350-500 | Per SMS (160 char) |
| SMTP | Email | Gratis | Self-hosted |
| Mailgun | Email | ~Rp 15-30 | Per email |

### Dashboard Biaya Notifikasi

```
┌────────────────────────────────────────────────────────────┐
│ Biaya Notifikasi — April 2026                              │
│                                                            │
│ ┌──────────────┬──────────────┬──────────────┬───────────┐ │
│ │ 📱 WhatsApp  │ 📨 SMS       │ 📧 Email     │ Total     │ │
│ │ 2,380 pesan  │ 52 pesan     │ 847 email    │           │ │
│ │ Rp 238.000   │ Rp 26.000    │ Rp 0 (SMTP)  │ Rp 264.000│ │
│ └──────────────┴──────────────┴──────────────┴───────────┘ │
│                                                            │
│ Biaya per pelanggan: Rp 311/bulan                          │
│ Trend: Mar Rp 245.000 → Apr Rp 264.000 (↑ 7.8%)         │
└────────────────────────────────────────────────────────────┘
```

- Admin bisa set **biaya per pesan** per provider di settings
- Biaya dihitung otomatis berdasarkan log pengiriman
- Membantu admin memilih provider yang paling cost-effective

---

## Statistik & Analytics Notifikasi

### Dashboard Analytics (`/notifications/analytics`)

```
┌────────────────────────────────────────────────────────────┐
│ Analytics Notifikasi — April 2026                          │
│                                                            │
│ Delivery Rate:                                             │
│ ┌──────────────┬──────────────┬──────────────┐             │
│ │ 📱 WhatsApp  │ 📨 SMS       │ 📧 Email     │             │
│ │ 97.1%        │ 94.2%        │ 99.5%        │             │
│ └──────────────┴──────────────┴──────────────┘             │
│                                                            │
│ Trend Pengiriman (6 bulan):                                │
│ ▁▂▃▅▆█  WA: naik 12%                                     │
│ ▁▁▁▁▁▁  SMS: stabil                                      │
│ ▁▂▃▄▅▆  Email: naik 8%                                   │
│                                                            │
│ Top 5 Nomor Gagal (sering fail):                           │
│ 1. +6281xxx (PLG-045) — 12x gagal (suspect)               │
│ 2. +6285xxx (PLG-102) — 8x gagal (suspect)                │
│ 3. +6812xxx (PLG-078) — 5x gagal                          │
│                                                            │
│ Template Paling Sering Dipakai:                            │
│ 1. Invoice Baru — 847x                                    │
│ 2. Konfirmasi Bayar — 720x                                │
│ 3. Reminder H-1 — 87x                                     │
│                                                            │
│ [Export PDF]  [Export Excel]                                │
└────────────────────────────────────────────────────────────┘
```

- **Delivery rate** per channel
- **Trend pengiriman** per bulan (grafik)
- **Top failed numbers** — nomor yang sering gagal
- **Template usage** — template mana yang paling sering dipakai
- **Read rate** (jika provider support, WA only)
- Detail lengkap di dokumen **11 — Reporting & Analytics**

---

## Webhook Inbound (Balasan Pelanggan)

Beberapa provider WA mendukung inbound message (pelanggan membalas pesan):

### Alur Inbound
```
Pelanggan balas pesan WA
  │
  ▼
Provider kirim webhook ke Notification Service
  │
  ▼
Parsing pesan:
  ├── "STOP" → Proses opt-out, balas konfirmasi
  ├── "BAYAR" → Kirim payment link terbaru
  └── Pesan lain → Simpan sebagai "Reply", notifikasi ke admin
```

### Tampilan di Dashboard
- Balasan pelanggan muncul di **log notifikasi** sebagai "↩️ Reply"
- Admin/operator mendapat **notifikasi in-app** bahwa ada balasan
- Klik reply → lihat pesan asli + balasan pelanggan
- Admin bisa **balas langsung** dari dashboard (manual reply)

### Auto-Reply (Opsional)
| Keyword | Respon Otomatis |
|---|---|
| STOP | "Anda telah berhenti menerima pesan promo dari {nama_isp}. Untuk berlangganan kembali, balas START" |
| BAYAR | "Berikut link pembayaran Anda: {link_bayar}. Total: {total_tagihan}" |
| INFO | "Paket Anda: {paket} ({harga}/bln). Status: {status}. Hubungi {telepon_isp} untuk info lebih lanjut" |
| Lainnya | "Terima kasih telah menghubungi {nama_isp}. Admin kami akan segera merespon." |

- Auto-reply bisa diaktifkan/nonaktifkan per tenant
- Keyword dan respon bisa dikustomisasi

---

## Graceful Degradation (Modul Notifikasi Belum Dikonfigurasi)

Jika tenant belum mengkonfigurasi provider notifikasi:
- Semua trigger notifikasi tetap berjalan
- Pesan masuk ke **antrian** tapi tidak dikirim
- Status di log: ⚠️ "Provider belum dikonfigurasi"
- Banner di dashboard: "Konfigurasi notifikasi untuk mulai mengirim pesan ke pelanggan"
- Billing dan fitur lain **tetap berjalan normal** tanpa notifikasi

### Fallback Jika Semua Provider Gagal

Jika provider sudah dikonfigurasi tapi semua gagal (saldo habis, provider down, API error):

```
Kirim notifikasi
  │
  ├── WA gagal → retry 3x → fallback SMS
  ├── SMS gagal → retry 3x → fallback Email
  ├── Email gagal → retry 3x
  │
  ▼
Semua channel gagal:
  ├── Status: ❌ "Semua channel gagal"
  ├── Notifikasi in-app ke admin: "⚠️ Notifikasi gagal dikirim ke {pelanggan}"
  ├── Pesan tetap tersimpan di log (bisa kirim ulang manual)
  ├── Jika error rate > 10% dalam 1 jam:
  │     → Alert ke Tenant Admin via in-app notification
  │     → Email darurat ke admin (jika email masih bisa)
  │     → Banner di dashboard: "⚠️ Banyak notifikasi gagal. Cek konfigurasi provider."
  └── Billing tetap berjalan normal (isolir tetap jalan meskipun notifikasi gagal)
```

**Penting:** Kegagalan notifikasi **tidak boleh** menghentikan proses billing. Invoice tetap digenerate, isolir tetap berjalan, pembayaran tetap diproses — hanya notifikasi ke pelanggan yang tertunda.

---

## Integrasi dengan Modul Lain

| Modul | Integrasi |
|---|---|
| **Pelanggan (04)** | Kirim notifikasi dari detail pelanggan, bulk kirim dari tabel |
| **Paket (05)** | Notifikasi perubahan harga paket ke pelanggan terdampak |
| **Billing (06)** | Invoice, reminder bertingkat, konfirmasi bayar, isolir, buka isolir |
| **MikroTik (08)** | Alert router offline, sync gagal ke teknisi |
| **Settings (12)** | Konfigurasi provider, template, preferensi notifikasi admin |

---

## Keputusan

| Keputusan | Detail |
|---|---|
| Channel | 3 channel: WhatsApp (utama), SMS (fallback), Email (opsional) |
| Provider WA | Fonnte, WaBlas, WA Business API — adapter pattern, ganti tanpa downtime |
| Provider SMS | Zenziva, Twilio, Nexmo — adapter pattern |
| Provider Email | SMTP, Mailgun, SendGrid — adapter pattern |
| Prioritas | Configurable per tenant, default: WA → SMS → Email |
| Template | ✅ Template bawaan + custom. Variabel dinamis, preview, test kirim, version history (10 versi) |
| Template SMS | ✅ Terpisah dari WA, max 160 karakter, short URL otomatis, character counter |
| Notifikasi otomatis | ✅ Trigger dari event billing, pelanggan, network |
| Notifikasi manual | ✅ Kirim ke 1 pelanggan atau bulk dari tabel |
| Broadcast | ✅ Filter penerima, jadwal kirim, progress tracking, rate limiting, pause/resume/cancel |
| Log pengiriman | ✅ Status lengkap (pending → sent → delivered → read → failed) |
| Retry | 3x retry, fallback otomatis ke channel berikutnya |
| Deduplication | ✅ Dedup key per pelanggan+template+periode, window 1 jam |
| Deteksi nomor tidak aktif | ✅ 3x gagal berturut-turut → tandai suspect, warning di detail pelanggan |
| Opt-out / unsubscribe | ✅ Pelanggan bisa opt-out dari reminder, promo, info. Transaksional tidak bisa opt-out |
| Anti-spam throttle | ✅ Max 5 pesan/hari per pelanggan, cooldown 30 menit, exception untuk konfirmasi bayar |
| Cost tracking | ✅ Biaya per pesan per provider, dashboard biaya bulanan, biaya per pelanggan |
| Analytics | ✅ Delivery rate, trend, top failed numbers, template usage, read rate |
| Webhook inbound | ✅ Terima balasan pelanggan, auto-reply (STOP/BAYAR/INFO), notifikasi ke admin |
| Quiet hours | ✅ Default 07:00-21:00, exception untuk konfirmasi bayar & buka isolir |
| Saldo monitoring | ✅ Cek saldo provider, alert jika rendah, health check berkala |
| Notifikasi admin | ✅ Internal notification untuk event penting (saldo rendah, error, pembayaran besar) |
| In-app notification | ✅ Bell icon, realtime via WebSocket, tersimpan 30 hari |
| Graceful degradation | ✅ Billing tetap jalan tanpa notifikasi jika provider belum dikonfigurasi |
| Credential storage | Terenkripsi per tenant, tidak plain text |
| Rate limiting | Default 50 pesan/detik untuk broadcast |
| Arsitektur | Service terpisah (Golang), terima event via Redis queue (asynq) |