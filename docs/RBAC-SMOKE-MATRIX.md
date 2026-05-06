# RBAC Smoke Matrix

Tanggal: 2026-05-06

Scope matrix ini adalah route operasional non-MikroTik/OLT.

## Role

- Super Admin: owner aplikasi/SaaS.
- Tenant Admin: admin ISP/client tenant.
- Operator: operasional pelanggan dan laporan read-only pada area tertentu.
- Kasir: pembayaran, invoice read, expense operasional sesuai kebijakan tenant.
- Reseller: portal voucher reseller.

## Matrix Minimum

| Area | Super Admin | Tenant Admin | Operator | Kasir | Reseller |
| --- | --- | --- | --- | --- | --- |
| Dashboard tenant | Impersonate | Read | Read terbatas | Read finance/payment | Tidak |
| Pelanggan | Impersonate | CRUD | Read/update operasional | Read | Tidak |
| Area pelanggan | Impersonate | CRUD | CRUD operasional | Tidak | Tidak |
| Paket internet | Impersonate | CRUD | Read | Read | Tidak |
| Invoice | Impersonate | CRUD/admin action | Read | Read/catat bayar | Tidak |
| Pembayaran | Impersonate | Read/admin action | Tidak | Catat bayar/read | Tidak |
| Voucher tenant | Impersonate | CRUD/generate/print | Read | Read | Tidak |
| Reseller admin | Impersonate | CRUD/saldo/deposit | Tidak | Tidak | Tidak |
| Portal reseller | Tidak | Tidak | Tidak | Tidak | Login/action sendiri |
| Pengeluaran | Impersonate | CRUD | Tidak | CRUD sesuai spec | Tidak |
| Inventory | Impersonate | CRUD | Read-only | Read-only tanpa cost | Tidak |
| Cashflow | Impersonate | Read/export | Tidak | Read | Tidak |
| Reports | Impersonate | Read/export/admin setting | Read | Read | Tidak |
| Settings | Impersonate | Manage | Tidak | Tidak | Tidak |
| Super Admin console | Full | Tidak | Tidak | Tidak | Tidak |

## Smoke Procedure

1. Login sebagai setiap role.
2. Buka route utama sesuai matrix.
3. Coba satu action yang diizinkan dan satu action yang dilarang.
4. Pastikan response dilarang adalah `403` atau UI menyembunyikan aksi.
5. Pastikan role tanpa finance permission tidak menerima cost/cashflow sensitif.
6. Pastikan tenant tanpa module MikroTik/OLT tetap bisa menjalankan billing-only.

## Acceptance

- Tidak ada role yang bisa melewati RBAC backend hanya karena tombol UI disembunyikan.
- Billing-only tidak error ketika module MikroTik/OLT tidak aktif.
- Super Admin route tidak bisa diakses oleh token tenant biasa.
