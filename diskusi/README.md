# Diskusi Proyek: ISPBoss — ISP Billing SaaS Platform

## Tanggal Mulai Diskusi: 26 April 2026

Dokumen diskusi dipecah per topik agar mudah di-review dan di-maintain.

## Daftar File Diskusi

| No | File | Topik | Status |
|---|---|---|---|
| 00 | [00-arsitektur.md](./00-arsitektur.md) | Arsitektur, tech stack, prinsip kode, identitas produk | ✅ Selesai |
| 01 | [01-landing-page.md](./01-landing-page.md) | Landing page UI/UX, SEO, tema warna | ✅ Selesai |
| 02 | [02-auth.md](./02-auth.md) | Register, login, lupa password, Google OAuth | ✅ Selesai |
| 03 | [03-dashboard-layout.md](./03-dashboard-layout.md) | Sidebar, topbar, bottom nav, dashboard widgets | ✅ Selesai |
| 04 | [04-pelanggan.md](./04-pelanggan.md) | CRUD pelanggan, tabel, form, filter, area, import/export | ✅ Selesai |
| 05 | [05-paket.md](./05-paket.md) | Paket PPPoE & Voucher, reseller, generate voucher, print PDF | ✅ Selesai |
| 06 | [06-billing.md](./06-billing.md) | Invoice, pembayaran, payment gateway | 🔲 Belum |
| 07 | [07-notifikasi.md](./07-notifikasi.md) | WhatsApp, SMS, Email | 🔲 Belum |
| 08 | [08-mikrotik.md](./08-mikrotik.md) | RouterOS v6/v7, PPPoE, isolir, monitoring | 🔲 Belum |
| 09 | [09-olt.md](./09-olt.md) | Multi-brand OLT, provisioning ONT | 🔲 Belum |
| 10 | [10-ftth-mapping.md](./10-ftth-mapping.md) | Peta interaktif, topologi jaringan | 🔲 Belum |
| 11 | [11-laporan.md](./11-laporan.md) | Reporting & analytics | 🔲 Belum |
| 12 | [12-settings.md](./12-settings.md) | Pengaturan tenant, white label, profil | 🔲 Belum |
| 13 | [13-billing-core-standalone.md](./13-billing-core-standalone.md) | Billing core tanpa add-on MikroTik/OLT | Belum |
| 14 | [14-keuangan-operasional.md](./14-keuangan-operasional.md) | Pengeluaran, inventaris, dan cashflow operasional | Baru |
| 15 | [15-super-admin.md](./15-super-admin.md) | Super Admin owner console, tenant, subscription, support, audit global | Baru |
| 16 | [16-landing-page-seo-fix-spec.md](./16-landing-page-seo-fix-spec.md) | Rangkuman audit dan spec perbaikan SEO landing page | Selesai |

## Alur Kerja

```
Diskusi per modul (detail sampai tuntas)
  → Review & finalisasi diskusi
  → Buat spec dari diskusi yang sudah final
  → Eksekusi spec → kode
  → Testing → deploy production
```
