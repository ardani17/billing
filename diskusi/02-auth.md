# 02 — Auth (Register, Login, Lupa Password)

---

## Halaman Register (`ispboss.id/register`)

**Tujuan:** Pendaftaran tenant baru (pemilik ISP).
**Layout:** Split screen — kiri branding/social proof, kanan form. Mobile: form full width.

```
┌────────────────────────────┬─────────────────────────────┐
│                            │                             │
│  (Ilustrasi/Branding)      │  🔵 ISPBoss                │
│                            │  Buat Akun Baru             │
│  "Bergabung dengan         │                             │
│   ratusan ISP yang         │  Nama Lengkap               │
│   sudah percaya ISPBoss"   │  [___________________]      │
│                            │  Email                      │
│  ✓ 500+ ISP aktif          │  [___________________]      │
│  ✓ 50.000+ pelanggan       │  Nomor WhatsApp             │
│  ✓ 99.9% uptime            │  [+62________________]      │
│                            │  Nama ISP / Perusahaan      │
│                            │  [___________________]      │
│                            │  Password                   │
│                            │  [••••••••          👁️]     │
│                            │  Konfirmasi Password        │
│                            │  [••••••••          👁️]     │
│                            │                             │
│                            │  ── atau daftar dengan ──   │
│                            │  [G  Daftar via Google]     │
│                            │                             │
│                            │  ☐ Setuju Syarat & Privasi  │
│                            │  [Daftar Sekarang]          │
│                            │  Sudah punya akun? Masuk    │
└────────────────────────────┴─────────────────────────────┘
```

### Field Register
| Field | Tipe | Validasi |
|---|---|---|
| Nama Lengkap | Text | Wajib, min 3 karakter |
| Email | Email | Wajib, format valid, unik |
| Nomor WhatsApp | Phone | Wajib, format +62 |
| Nama ISP / Perusahaan | Text | Wajib, jadi nama tenant |
| Password | Password | Wajib, min 8 karakter |
| Konfirmasi Password | Password | Wajib, harus sama |
| Google OAuth | Button | Alternatif, auto-fill nama & email |

### Alur Register
```
User isi form / klik "Daftar via Google"
  → Validasi input
  → Kirim email verifikasi (link, expired 24 jam)
  → Tampilkan halaman "Cek Email Kamu"
  → User klik link verifikasi
  → Email terverifikasi → auto-login → redirect ke dashboard
  → Paket default: Starter (trial 3 hari)
```

### Halaman "Cek Email Kamu"
```
┌──────────────────────────────────────┐
│         📧                           │
│   Cek Email Kamu                     │
│   Kami sudah kirim link verifikasi   │
│   ke ahmad@gmail.com                 │
│   Tidak terima email?                │
│   [Kirim Ulang] (cooldown 60 detik) │
└──────────────────────────────────────┘
```

---

## Halaman Login (`app.ispboss.id/login`)

**Tujuan:** Semua user masuk ke dashboard.
**Layout:** Split screen, sama seperti register.

```
┌────────────────────────────┬─────────────────────────────┐
│                            │                             │
│  (Ilustrasi/Branding)      │  🔵 ISPBoss                │
│  "Kelola ISP Kamu          │  Masuk ke Dashboard         │
│   Dari Satu Dashboard"     │                             │
│                            │  Email                      │
│                            │  [___________________]      │
│                            │  Password                   │
│                            │  [••••••••          👁️]     │
│                            │                             │
│                            │  ☐ Ingat saya  Lupa password?│
│                            │  [Masuk]                    │
│                            │                             │
│                            │  ── atau masuk dengan ──    │
│                            │  [G  Masuk via Google]      │
│                            │                             │
│                            │  Belum punya akun?          │
│                            │  Coba Gratis 3 Hari         │
└────────────────────────────┴─────────────────────────────┘
```

### Alur Login
```
User isi email + password / klik "Masuk via Google"
  → Validasi credential
  → Cek email terverifikasi?
      → Belum: "Verifikasi email dulu" + kirim ulang
      → Sudah: generate JWT token
  → Redirect sesuai role:
      - Tenant Admin → /dashboard
      - Operator → /dashboard
      - Teknisi → /network
      - Kasir → /payments
```

---

## Halaman Lupa Password (`app.ispboss.id/forgot-password`)

```
┌──────────────────────────────────────┐
│  🔵 ISPBoss                         │
│  Lupa Password?                      │
│  Masukkan email, kami kirim link reset│
│  Email [____________________]        │
│  [Kirim Link Reset]                  │
│  ← Kembali ke login                 │
└──────────────────────────────────────┘
```

**Alur:** Kirim email → klik link (expired 1 jam) → set password baru → auto-login.

---

## Keputusan Auth

| Keputusan | Detail |
|---|---|
| Verifikasi email | **Wajib** sebelum bisa pakai |
| Google OAuth | **Ya** — daftar & login, auto-verifikasi email |
| Onboarding wizard | **Tidak** — langsung ke dashboard |
| Paket default | Starter (trial 3 hari) |
| Remember me | JWT: 7 hari (checked) vs 24 jam (unchecked) |
| Rate limit login | Max 5 gagal → lock 15 menit |
| Lupa password | Link reset via email, expired 1 jam |
| 2FA | Fase lanjut (Google Authenticator) |
| Session management | Lihat & logout device lain dari settings |
