# Design Document

## Overview

Financial completion menyatukan backend finance yang sudah tersedia dengan UI operasional dan settings persistence. Desain ini mempertahankan struktur monorepo saat ini:

- Backend utama: `services/billing-api`
- Frontend admin: `apps/web`
- Migration database: `services/billing-api/migrations`
- Spec tracking: `.kiro/specs/financial-completion`

Fokus desain adalah menutup gap tanpa mengubah ulang domain yang sudah ada.

## Architecture

### Backend

Backend harus memakai layer yang sudah ada:

- Handler HTTP di `services/billing-api/internal/handler`
- Usecase di `services/billing-api/internal/usecase`
- Repository di `services/billing-api/internal/repository`
- Domain model di `services/billing-api/internal/domain`

Endpoint baru atau penyempurnaan endpoint harus dipasang pada router existing, mengikuti pola route `/api/v1`.

### Frontend

Frontend harus mengikuti route dan komponen existing di `apps/web/app`.

Prioritas UI:

- `/settings/billing`
- `/settings/reports`
- Invoice list/detail action
- Payment operation action
- Customer detail recurring item
- Finance reconciliation dashboard

Komponen yang saat ini masih generic placeholder harus diganti dengan live page apabila fitur membutuhkan persistence nyata.

## API Design

| Capability | API expectation |
| --- | --- |
| Billing settings | `GET /api/v1/settings/billing`, `PUT /api/v1/settings/billing` |
| Report settings | `GET /api/v1/settings/reports`, `PUT /api/v1/settings/reports` atau endpoint resource per KPI/schedule/template |
| Credit note list/detail | `GET /api/v1/credit-notes`, `GET /api/v1/credit-notes/:id` jika belum tersedia |
| Debit note list/detail | `GET /api/v1/debit-notes`, `GET /api/v1/debit-notes/:id` jika belum tersedia |
| Recurring item customer | Reuse nested customer recurring item endpoint existing |
| Reconciliation | `GET /api/v1/reports/reconciliation` atau extend report finance endpoint existing |
| Bulk PDF invoice | Reuse endpoint bulk PDF existing, tetapi implementasi harus menghasilkan PDF final |

Jika endpoint create sudah ada, jangan membuat endpoint duplikat. Tambahkan list/detail hanya jika UI membutuhkan data yang belum tersedia.

## Data Model

Desain harus memanfaatkan tabel dan field yang sudah ada terlebih dahulu:

- `billing_settings`
- `expense_categories`
- `expenses`
- `report_schedules`
- `report_jobs`
- `custom_report_templates`
- `kpi_targets`
- invoice/payment related tables
- credit/debit note related tables jika sudah tersedia

Migration baru hanya dibuat apabila field yang dibutuhkan benar-benar belum ada.

## UI Design

### Settings billing

Halaman `/settings/billing` harus menjadi form live dengan section:

- Invoice numbering and prefix
- Tax settings
- Penalty settings
- Due date and grace period
- Reminder defaults
- Payment link defaults jika sudah ada di backend

Form harus memiliki loading, dirty state, save success, validation error, dan authorization error.

### Invoice operations

Invoice page harus mendukung:

- Create invoice
- Create prepaid invoice
- Edit invoice
- Cancel invoice
- Download PDF
- Send reminder
- Bulk select
- Bulk reminder
- Bulk cancel
- Bulk export
- Bulk PDF
- Credit note / debit note entry point dari detail invoice

### Payment operations

Payment page harus mendukung:

- Quick payment by customer
- Multi-invoice payment
- Pay-all
- Receipt download/print
- Proof upload/view
- Void payment
- Import payment result screen

### Customer recurring item

Customer detail harus memiliki tab atau section recurring item:

- List recurring item
- Add/edit recurring item
- Activate/deactivate
- Show next billing inclusion

### Reconciliation

Reconciliation dashboard harus berisi:

- Period filter
- Area/cabang filter
- Invoice issued
- Payment collected
- Outstanding
- Expense
- Voucher impact
- Credit/debit note impact
- Net collection
- Anomaly list
- Export action

### Report settings

Halaman `/settings/reports` harus mengelola:

- KPI target
- Report schedule
- Custom report template
- Job history minimal untuk schedule result

## Permissions

Operasi berikut harus dibatasi role finance admin atau tenant admin:

- Billing settings update
- Report settings update
- Cancel invoice
- Bulk cancel invoice
- Credit note
- Debit note
- Void payment
- Expense delete
- Report schedule create/update/delete

Read-only finance user dapat melihat invoice, payment, expense, dan report tanpa action destruktif.

## Error Handling

Frontend harus menampilkan error yang bisa ditindaklanjuti:

- Validation error per field
- Permission denied
- Network/server error
- Empty state
- Partial success untuk import dan bulk action

Backend harus mengembalikan response terstruktur sesuai pola API existing.

## Testing Strategy

Backend:

- Unit/usecase test untuk billing settings save/load.
- Handler test untuk endpoint settings dan report controls jika pola test tersedia.
- Regression test untuk credit/debit note dan recurring item jika menyentuh logic invoice.
- Test bulk PDF minimal memastikan response file valid dan tidak placeholder.

Frontend:

- Build check workspace web.
- Component/page smoke test jika test framework tersedia.
- Manual smoke test:
  - Save billing settings.
  - Create prepaid invoice.
  - Create credit/debit note.
  - Add recurring item.
  - Quick payment.
  - Void payment.
  - Add expense.
  - Open profit-loss.
  - Open reconciliation.
  - Save report settings.

## Migration Plan

1. Audit ulang tabel existing sebelum menambah migration.
2. Implement backend gaps terkecil lebih dulu: settings billing/report and list/detail credit/debit note.
3. Implement UI live pages.
4. Finalkan bulk PDF dan reconciliation.
5. Jalankan test backend dan build frontend.
6. Update dokumen audit dengan hasil implementasi.
