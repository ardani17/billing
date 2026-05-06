# 13 - Billing Core Standalone

Dokumen ini memperjelas batas modul setelah keputusan paket komersial:

- Billing Core selalu aktif.
- Add-on MikroTik bersifat opsional.
- Add-on OLT + Peta Jaringan bersifat opsional dan memakai module flag `fiber_network`.

Tujuan utamanya: aplikasi Billing Core harus tetap siap dipakai walaupun tenant tidak membeli dua add-on jaringan.

---

## Prinsip Produk

Billing Core adalah aplikasi utama untuk operasional admin ISP:

- pelanggan
- paket
- invoice
- pembayaran manual
- payment gateway
- reminder dan notifikasi
- reseller/voucher
- laporan
- settings dasar

MikroTik dan Fiber Network tidak boleh menjadi syarat agar Billing Core berjalan.

Jika add-on tidak aktif:

- Menu add-on disembunyikan.
- Field add-on disembunyikan.
- Widget add-on disembunyikan.
- API add-on mengembalikan `MODULE_NOT_ENABLED`.
- Flow Billing Core tetap berhasil.
- Event teknis jaringan menjadi no-op aman.

---

## Customer Billing-Only

Tenant Billing Core harus bisa input pelanggan secara manual.

Field yang wajib untuk Billing Core:

- nama pelanggan
- telepon
- alamat
- area atau wilayah
- paket billing
- tanggal aktivasi
- tanggal jatuh tempo
- status pelanggan

Field yang opsional untuk Billing Core:

- email
- catatan
- koordinat
- referensi layanan manual

Field yang hanya muncul jika Add-on MikroTik aktif:

- metode PPPoE, Hotspot, DHCP Binding, Static IP
- router MikroTik
- username/password PPPoE
- MAC address
- profile/router mapping
- aksi teknis disconnect, reset, sync, isolir router

Field yang hanya muncul jika Add-on OLT + Peta Jaringan aktif:

- OLT
- ODP
- ONT/ONU
- port ODP
- koordinat wajib untuk map
- link ke peta jaringan

Keputusan implementasi:

- Tambahkan metode koneksi netral `manual`.
- Default form pelanggan untuk tenant Billing-only adalah `manual`.
- `router_id`, PPPoE credential, MAC address, ODP, ONT/ONU, dan koordinat tidak boleh wajib untuk Billing Core.

---

## Package Billing-Only

Paket bulanan Billing Core tidak boleh dipaksa disebut PPPoE.

Billing Core butuh paket:

- paket bulanan
- harga bulanan
- biaya pasang
- status aktif/nonaktif
- data FUP/kuota jika dipakai sebagai informasi billing
- voucher/reseller sebagai produk Billing Core

Field yang hanya muncul jika Add-on MikroTik aktif:

- MikroTik profile
- address pool
- parent queue
- burst
- hotspot profile
- shared users

Keputusan implementasi:

- Tambahkan tipe paket netral seperti `monthly`, atau minimal ubah tampilan UI agar `pppoe` lama ditampilkan sebagai paket bulanan saat MikroTik tidak aktif.
- Voucher tetap bagian Billing Core secara komersial.
- Provisioning voucher ke Hotspot MikroTik hanya berjalan jika Add-on MikroTik aktif.

---

## Billing Dan Isolir

Billing Core memiliki status finansial dan status layanan pelanggan.
MikroTik memiliki aksi teknis enforcement.

Jika Add-on MikroTik tidak aktif:

- Auto-isolir mengubah status billing/pelanggan.
- Reminder dan notifikasi tetap berjalan.
- Walled garden billing page tetap bisa dipakai sebagai halaman tagihan.
- Sistem tidak membuat pending router sync.
- Sistem tidak memanggil RouterOS.
- Sistem tidak publish event teknis yang wajib dieksekusi network-service.

Jika Add-on MikroTik aktif:

- Status isolir/unisolir dapat membuat pending sync.
- Aksi teknis RouterOS tetap on-demand/manual sesuai aturan modul MikroTik.
- Tidak boleh ada polling/login RouterOS terus menerus.

---

## Dashboard Dan Report

Dashboard Billing Core selalu tampil:

- pelanggan aktif
- pendapatan bulan ini
- piutang
- collection rate
- invoice terbaru
- pembayaran terbaru

Widget MikroTik hanya tampil jika `mikrotik` aktif.

Widget OLT/Peta hanya tampil jika `fiber_network` aktif.

Report Billing Core selalu tampil:

- laporan keuangan
- laporan pelanggan
- laporan pembayaran
- laporan reseller/voucher
- laporan notifikasi
- laporan audit operasional

Report jaringan hanya tampil sesuai add-on:

- MikroTik: uptime, traffic, session, router health
- Fiber Network: OLT signal, alarm, ONT/ONU, kapasitas ODP, peta jaringan

---

## Import Export

Template import Billing-only:

```csv
nama,telepon,email,alamat,area,paket,tanggal_aktivasi,tanggal_jatuh_tempo,status,catatan
Ahmad Rizki,+6281234567890,ahmad@email.com,Jl. Merdeka No. 10,RT 03/05,Pro 50M,2026-05-05,5,aktif,Rumah biru
```

Kolom tambahan jika MikroTik aktif:

```csv
connection_method,router_id,pppoe_username,pppoe_password,mac_address
```

Kolom tambahan jika Fiber Network aktif:

```csv
olt_id,odp_id,onu_id,odp_port,latitude,longitude
```

---

## Audit Implementasi Saat Ini

Hasil smoke Billing-only saat `mikrotik` inactive dan `fiber_network` tidak aktif:

- `/api/billing/customers?page_size=10` -> 200
- `/api/billing/packages?page_size=10` -> 200
- `/api/billing/invoices?page_size=10` -> 200
- `/api/billing/payments?page_size=10` -> 200
- `/api/billing/reports/dashboard` -> 200

Gap yang masih harus dikerjakan:

- Customer UI masih default PPPoE.
- Customer backend belum punya mode `manual`.
- Koordinat masih berasal dari desain awal yang wajib untuk mapping.
- Import pelanggan masih membawa kolom PPPoE/router/ODP.
- Package type masih `pppoe` untuk paket bulanan.
- Package UI masih menampilkan MikroTik profile.
- Isolir/pending sync perlu no-router path saat MikroTik nonaktif.
- Dashboard masih memanggil summary MikroTik dan OLT.
- Proxy MikroTik tertentu masih mengubah `MODULE_NOT_ENABLED` menjadi 502.
- Report perlu konsisten memakai `fiber_network` untuk OLT + Peta Jaringan.

---

## Urutan Pengerjaan

1. Customer standalone mode.
2. Package standalone mode.
3. Billing isolir no-router behavior.
4. Import/export capability-aware templates.
5. Dashboard/report capability cleanup.
6. Smoke test kombinasi entitlement.

