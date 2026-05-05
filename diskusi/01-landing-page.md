# 01 — Landing Page (`ispboss.id`)

**Tujuan:** Meyakinkan calon pembeli (operator ISP) untuk mendaftar.

---

## Struktur Halaman

### A. Navbar (Sticky)
```
┌─────────────────────────────────────────────────────────────┐
│  [Logo ISPBoss]   Fitur   Harga   FAQ   ☀️🌙 [Masuk] [Coba Gratis]  │
└─────────────────────────────────────────────────────────────┘
```
- Logo + nama di kiri
- Link navigasi di tengah
- Toggle tema (sun/moon), "Masuk" (outline), "Coba Gratis 3 Hari" (filled blue) di kanan
- Sticky on scroll dengan backdrop blur
- Mobile: hamburger menu

### B. Hero Section (Split Layout)
```
┌──────────────────────────────────────────────────────────┐
│                                                          │
│  Kelola ISP Kamu              ┌────────────────────┐     │
│  Dari Satu Dashboard          │  [Screenshot/      │     │
│                               │   Mockup Dashboard] │     │
│  Platform billing dan         │                    │     │
│  manajemen jaringan           └────────────────────┘     │
│  all-in-one untuk ISP                                    │
│                                                          │
│  [Coba Gratis 3 Hari]  [Lihat Fitur ↓]                  │
│                                                          │
│  ✓ Tanpa kartu kredit  ✓ Setup 5 menit  ✓ Cancel kapan saja │
└──────────────────────────────────────────────────────────┘
```
- Headline besar di kiri, mockup dashboard di kanan
- Background: subtle gradient biru ke putih

### C. Fitur Utama (Bento Grid)
```
┌────────────────┬────────────────┬──────────┐
│  Billing       │  MikroTik      │  OLT     │
│  Otomatis      │  Integration   │  Multi-  │
│  (besar)       │  (besar)       │  Brand   │
├────────────────┼────────────────┤  (kecil) │
│  Notifikasi    │  FTTH Visual   ├──────────┤
│  WA/SMS/Email  │  Mapping       │  White   │
│  (kecil)       │  (kecil)       │  Label   │
└────────────────┴────────────────┴──────────┘
```
- Bento grid asimetris, icon Phosphor + judul + 1-2 kalimat
- Hover: subtle lift + shadow. Mobile: stack vertikal

### D. Cara Kerja (3 Step)
```
①                    ②                    ③
Daftar &             Hubungkan            Kelola
Setup Akun           Router & Perangkat   Pelanggan & Billing
```
- Horizontal 3 step dengan garis penghubung. Mobile: vertikal

### E. Pricing Section
```
┌────────────────┬────────────────┬──────────────────────┐
│ Billing Core   │ + MikroTik     │ + OLT + Peta Jaringan│
│ Billing lengkap│ RouterOS add-on│ Fiber network add-on │
│ Mulai paket A  │ Upgrade add-on │ Upgrade add-on       │
│ [Coba Billing] │ [Tambah]       │ [Tambah]             │
└────────────────┴────────────────┴──────────────────────┘
```
- Harga tetap bisa memiliki tier jumlah pelanggan (Starter/Growth/Pro/Enterprise), tetapi bundling fitur dipisah menjadi Billing Core, Add-on MikroTik, dan Add-on OLT + Peta Jaringan.
- Notifikasi, laporan, payment gateway, dan reseller/voucher masuk Billing Core, bukan add-on terpisah.
- Mobile: stack vertikal dengan checklist add-on.

### F. Testimonial / Social Proof
- Carousel kutipan ISP, auto-scroll pelan

### G. FAQ (Accordion)
- "Apakah bisa pakai domain sendiri?"
- "Support merek OLT apa saja?"
- "Apakah data saya aman?"
- "Bagaimana cara migrasi dari billing lama?"
- "Apakah ada batasan jumlah router?"
- "Bagaimana jika trial habis?"

### H. CTA Section
```
┌──────────────────────────────────────────────┐
│   Siap Kelola ISP Kamu Lebih Efisien?        │
│   [Coba Gratis 3 Hari →]                     │
└──────────────────────────────────────────────┘
```
- Background biru gelap / gradient

### I. Footer
```
ISPBoss              Produk        Perusahaan
Kelola ISP Kamu      Fitur         Tentang Kami
Dari Satu Dashboard  Harga         Blog / Kontak
[Social Icons]       API Docs      Kebijakan Privasi
© 2026 ISPBoss
```

---

## Tema Warna (Light & Dark)

### Light Mode
```
Background utama     : #FFFFFF
Background section   : #F8FAFC (alternating)
Hero gradient        : #EFF6FF → #FFFFFF
Teks heading         : #0F172A
Teks paragraf        : #64748B
CTA primary          : #2563EB, hover #1D4ED8
Card                 : #FFFFFF, border #E2E8F0, blue tinted shadow
Navbar               : white/80 + backdrop-blur
Footer               : #0F172A, teks #94A3B8
CTA section          : gradient #1E3A8A → #2563EB
```

### Dark Mode
```
Background utama     : #0F172A
Background section   : #1E293B (alternating)
Hero gradient        : #172554 → #0F172A
Teks heading         : #F8FAFC
Teks paragraf        : #94A3B8
CTA primary          : #3B82F6, hover #60A5FA
Card                 : #1E293B, border #334155, blue tinted shadow
Navbar               : Slate-900/80 + backdrop-blur
Footer               : #020617, teks #64748B
CTA section          : gradient #1E3A8A → #172554
```

### Toggle
- Default: system preference (`prefers-color-scheme`)
- Override via sun/moon icon di navbar
- Simpan di `localStorage`, transisi 200ms

---

## SEO Strategy

| Elemen | Implementasi |
|---|---|
| Title | "ISPBoss - Platform Billing & Manajemen Jaringan untuk ISP" |
| Meta description | "Mulai dari sistem billing ISP lengkap, lalu tambah modul MikroTik dan OLT + Peta Jaringan sesuai kebutuhan. Coba gratis 3 hari." |
| H1 | "Kelola ISP Kamu Dari Satu Dashboard" |
| URL | Clean: `/fitur`, `/harga`, `/faq` |
| Open Graph | Title, description, image preview |
| Structured data | Organization, Product, FAQ (JSON-LD) |
| Sitemap | Auto-generate `sitemap.xml` |
| Performance | LCP < 2.5s, FID < 100ms, CLS < 0.1 |
| Image | WebP, lazy loading, alt text |
| Keywords | "billing ISP", "billing RT RW Net", "manajemen MikroTik", "software ISP Indonesia" |
