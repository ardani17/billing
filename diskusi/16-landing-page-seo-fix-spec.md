# 16 - Landing Page SEO Fix Spec

## Tujuan

Memperbaiki landing page publik ISPBoss agar root domain, metadata, structured data, sitemap, robots, link publik, dan social preview siap untuk indexing search engine dan sharing link.

Spec ini berasal dari audit kode pada 6 Mei 2026 terhadap:

- `apps/web/app/page.tsx`
- `apps/web/app/layout.tsx`
- `apps/web/app/landing/page.tsx`
- `apps/web/app/landing/LandingPageClient.tsx`
- struktur route dan asset publik di `apps/web/app` dan `apps/web/public`

## Ringkasan Temuan

| Prioritas | Area | Masalah | Dampak SEO |
|---|---|---|---|
| P1 | Root URL | `/` masih halaman chooser internal, bukan landing publik utama | Domain utama bisa terindeks sebagai halaman tipis, bukan halaman marketing |
| P1 | Canonical metadata | Landing belum punya `metadataBase`, canonical, `openGraph.url`, dan site name | Rawan duplikasi URL dan social metadata tidak absolut |
| P1 | Robots dan sitemap | Belum ada `robots.ts` dan `sitemap.ts` | Crawler tidak mendapat daftar URL publik resmi dan route app/private bisa menjadi noise |
| P2 | Footer links | `/contact` dan `/privacy` belum ada; White label mengarah ke `/settings/branding` | 404, crawl error, atau crawler masuk area aplikasi |
| P2 | FAQ | Sebagian besar jawaban FAQ tidak muncul di initial HTML dan belum ada `FAQPage` JSON-LD | Long-tail SEO dan peluang rich result berkurang |
| P2 | Social preview | Belum ada OG image, Twitter metadata, favicon, icon, atau manifest | Preview saat dibagikan tidak konsisten dan brand signal lemah |

## Scope Perbaikan

### In scope

- Membuat root domain `/` menjadi landing publik utama atau melakukan permanent redirect ke `/landing`.
- Menambahkan metadata SEO lengkap untuk landing page.
- Menambahkan `robots.ts`, `sitemap.ts`, dan manifest/icon dasar.
- Membuat route publik yang ditautkan dari landing: `/contact` dan `/privacy`.
- Mengubah link White label agar mengarah ke konten publik, bukan halaman settings internal.
- Memperluas JSON-LD: `Organization`, `SoftwareApplication` atau `Product`, `FAQPage`, dan `WebSite`.
- Memastikan FAQ tetap bisa dibaca crawler walaupun UI tetap accordion.
- Menyiapkan OG/Twitter image yang konsisten dengan brand ISPBoss.

### Out of scope

- Implementasi blog, knowledge base, atau programmatic SEO multi-kota.
- Integrasi analytics, Search Console, atau tag manager.
- Perubahan backend billing, auth, dashboard, MikroTik, OLT, dan map.
- Rework visual besar landing page di luar kebutuhan SEO.

## Keputusan Produk

### Root URL

Rekomendasi utama: `/` harus menjadi landing page publik.

Pilihan implementasi:

1. Pindahkan isi `apps/web/app/landing` ke `apps/web/app/page.tsx`, lalu biarkan `/landing` redirect ke `/`.
2. Atau, jadikan `apps/web/app/page.tsx` redirect permanen ke `/landing`.

Pilihan 1 lebih kuat untuk SEO karena canonical utama menjadi root domain. Pilihan 2 lebih cepat dan tetap aman jika deploy sudah memakai `/landing` di iklan atau materi promosi.

Acceptance:

- Membuka `/` menampilkan konten marketing utama, bukan chooser internal.
- Tidak ada dua URL publik yang sama-sama canonical untuk konten landing.
- Jika `/landing` tetap ada, canonical harus menunjuk ke URL yang dipilih sebagai utama.

## Metadata Spec

Tambahkan metadata global di `apps/web/app/layout.tsx`:

- `metadataBase`: domain produksi, misalnya `https://ispboss.id`.
- `applicationName`: `ISPBoss`.
- `creator` dan `publisher`: `ISPBoss`.
- `formatDetection`: nonaktifkan auto telephone/email jika tidak dibutuhkan.
- `icons`: favicon, app icon, apple touch icon.
- `manifest`: `/manifest.webmanifest`.

Tambahkan metadata landing:

- `title`: `ISPBoss - Platform Billing dan Manajemen Jaringan untuk ISP`
- `description`: `Kelola billing ISP, pelanggan, invoice, pembayaran, MikroTik, OLT, notifikasi, dan peta FTTH dari satu dashboard. Coba gratis 3 hari.`
- `alternates.canonical`: `/` atau `/landing` sesuai keputusan root.
- `openGraph.type`: `website`
- `openGraph.locale`: `id_ID`
- `openGraph.siteName`: `ISPBoss`
- `openGraph.url`: canonical URL utama
- `openGraph.images`: `/opengraph-image`
- `twitter.card`: `summary_large_image`
- `twitter.title`, `twitter.description`, `twitter.images`
- `robots`: index dan follow untuk landing publik.

Acceptance:

- Source HTML berisi title, description, canonical, OG title, OG description, OG image, Twitter card, dan language `id`.
- URL OG image absolut ketika dirender production.
- Metadata tidak mengarah ke localhost.

## Robots dan Sitemap Spec

Tambahkan `apps/web/app/robots.ts`.

Rules:

- Allow: `/`, `/landing`, `/register`, `/login`, `/forgot-password`, `/verify-email`, `/contact`, `/privacy`.
- Disallow route aplikasi operasional:
  - `/dashboard`
  - `/customers`
  - `/packages`
  - `/invoices`
  - `/payments`
  - `/reports`
  - `/settings`
  - `/super-admin`
  - `/mikrotik`
  - `/olt`
  - `/network-map`
  - `/api`

Tambahkan `apps/web/app/sitemap.ts`.

URL minimum:

- `/`
- `/landing` hanya jika route ini tetap dipakai publik
- `/register`
- `/login`
- `/forgot-password`
- `/contact`
- `/privacy`

Set `changeFrequency` dan `priority`:

- `/`: daily, priority 1.0
- `/register`: monthly, priority 0.7
- `/contact`: monthly, priority 0.6
- `/privacy`: yearly, priority 0.3

Acceptance:

- `/robots.txt` bisa dibuka dan menunjuk ke `/sitemap.xml`.
- `/sitemap.xml` hanya memuat URL publik yang aman untuk indexing.
- Route dashboard/internal tidak masuk sitemap.

## Link Publik Spec

Perbaiki footer landing:

- `/contact`: buat halaman kontak publik sederhana.
- `/privacy`: buat halaman kebijakan privasi publik.
- `White label`: ubah ke anchor publik, misalnya `#white-label`, atau buat section publik di landing.

Acceptance:

- Tidak ada link footer landing yang 404.
- Tidak ada link dari landing publik yang membawa crawler ke halaman settings tenant.
- Link CTA utama tetap ke `/register`.

## Structured Data Spec

Pindahkan JSON-LD agar dirender dari server component jika memungkinkan. Jika tetap di client component, pastikan payload lengkap dan tidak bergantung pada state.

JSON-LD minimum:

1. `Organization`
   - `name`: `ISPBoss`
   - `url`: domain produksi
   - `logo`: URL logo/icon publik
   - `contactPoint`: jika nomor/email support sudah final

2. `SoftwareApplication`
   - `name`: `ISPBoss`
   - `applicationCategory`: `BusinessApplication`
   - `operatingSystem`: `Web`
   - `description`
   - `offers` untuk harga mulai dari Starter

3. `FAQPage`
   - Semua pertanyaan FAQ yang tampil di halaman.
   - Semua jawaban harus tersedia di JSON-LD dan HTML.

4. `WebSite`
   - `name`: `ISPBoss`
   - `url`: domain produksi

Acceptance:

- JSON-LD valid ketika diuji dengan Rich Results Test atau schema validator.
- FAQ schema berisi semua item FAQ, bukan hanya item yang sedang terbuka.
- Tidak ada data palsu seperti rating/review jika belum benar-benar ada.

## FAQ dan Konten Spec

FAQ tetap boleh accordion, tetapi semua jawaban harus ada di DOM atau tersedia sebagai structured data lengkap.

Tambahkan/rapikan FAQ yang lebih dekat ke intent pencarian:

- `Apa itu software billing ISP?`
- `Apakah ISPBoss cocok untuk RT/RW Net?`
- `Apakah bisa integrasi MikroTik PPPoE?`
- `Apakah mendukung OLT dan jaringan FTTH?`
- `Bagaimana migrasi dari billing lama?`
- `Apakah trial perlu kartu kredit?`

Tambahkan keyword natural di copy, bukan stuffing:

- billing ISP
- software ISP Indonesia
- billing RT/RW Net
- manajemen MikroTik
- billing PPPoE
- manajemen OLT
- peta jaringan FTTH

Acceptance:

- H1 hanya satu dan menjelaskan produk.
- H2 mencakup fitur, cara kerja, harga, white label, FAQ, dan CTA.
- Konten tetap natural dibaca manusia.

## Asset dan Social Preview Spec

Tambahkan asset minimal:

- `apps/web/app/opengraph-image.tsx` atau file image statis untuk OG.
- `apps/web/app/twitter-image.tsx` jika berbeda dari OG.
- `apps/web/app/icon.tsx` atau favicon statis.
- `apps/web/app/apple-icon.tsx` atau apple touch icon statis.
- `apps/web/app/manifest.ts` atau `manifest.webmanifest`.

OG image direction:

- Ukuran 1200 x 630.
- Menampilkan brand `ISPBoss`, headline singkat, dan visual dashboard/network.
- Tidak memakai teks kecil berlebihan.
- Kontras tinggi dan aman saat dicrop platform sosial.

Acceptance:

- `/opengraph-image` mengembalikan image valid.
- Preview WhatsApp, Facebook, LinkedIn, dan X/Twitter minimal punya title, description, dan large image.
- Favicon tampil di browser tab.

## Performance dan Accessibility Checks

Target:

- LCP kurang dari 2.5 detik pada koneksi normal.
- CLS kurang dari 0.1.
- HTML awal berisi konten utama landing, bukan skeleton kosong.
- Tombol mobile menu punya `aria-expanded`.
- FAQ accordion punya `aria-expanded` dan relasi kontrol jika memungkinkan.

Acceptance:

- `npm.cmd --workspace @ispboss/web run build` sukses.
- Lighthouse SEO minimal 95 untuk landing.
- Lighthouse Accessibility minimal 90 untuk landing.
- Tidak ada 404 untuk link internal landing.

## Urutan Eksekusi Rekomendasi

1. Putuskan canonical utama: `/` atau `/landing`.
2. Ubah root URL sesuai keputusan.
3. Lengkapi metadata global dan metadata landing.
4. Tambah robots dan sitemap.
5. Tambah `/contact`, `/privacy`, dan section/link White label publik.
6. Lengkapi JSON-LD dan FAQ.
7. Tambah OG image, favicon/icon, dan manifest.
8. Jalankan build dan cek HTML output.
9. Jalankan Lighthouse atau Playwright smoke untuk `/`, `/robots.txt`, `/sitemap.xml`, `/contact`, dan `/privacy`.

## Status Implementasi

Selesai dikerjakan pada 6 Mei 2026.

Keputusan yang dipakai:

- `/` menjadi landing page publik utama.
- `/landing` melakukan permanent redirect ke `/`.
- Canonical utama memakai `https://ispboss.id/`.
- Sitemap hanya memuat route publik yang aman untuk indexing.
- Route aplikasi operasional tetap tidak dimasukkan sitemap dan diblokir di `robots.txt`.

Verifikasi yang sudah dijalankan:

- `npm.cmd --workspace @ispboss/web run build` sukses.
- `PLAYWRIGHT_PORT=3100 npm.cmd run test:e2e` sukses.
- Live dev check: `/` status 200, `/landing` status 308 ke `/`, `/robots.txt` status 200, `/sitemap.xml` status 200, `/contact` status 200, `/privacy` status 200, `/opengraph-image` status 200 image/png, `/icon` status 200 image/png.
- HTML `/` berisi canonical, OG image, Twitter card, `FAQPage` JSON-LD, dan jawaban FAQ.
- Lighthouse terhadap production server lokal: SEO 100 dan Accessibility 95.

## Checklist Done

- [x] `/` adalah landing utama atau redirect permanen ke canonical landing.
- [x] Canonical URL eksplisit.
- [x] Metadata OG/Twitter lengkap.
- [x] `robots.txt` tersedia.
- [x] `sitemap.xml` tersedia.
- [x] `/contact` tersedia.
- [x] `/privacy` tersedia.
- [x] Footer tidak menuju route private atau 404.
- [x] FAQ lengkap di HTML/JSON-LD.
- [x] JSON-LD tersedia dan berisi `Organization`, `WebSite`, `SoftwareApplication`, dan `FAQPage`.
- [x] OG image dan favicon tersedia.
- [x] Build web sukses.
- [x] Smoke test landing page sukses.
- [x] Lighthouse SEO minimal 95.
- [x] Lighthouse Accessibility minimal 90.
