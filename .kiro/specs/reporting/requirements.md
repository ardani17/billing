# Requirements Document

## Introduction

Dokumen ini mendefinisikan requirements untuk **Reporting & Analytics** di platform ISPBoss. Modul laporan mengagregasi data dari seluruh modul lain (pelanggan, billing, paket, MikroTik, OLT, notifikasi) dan menyajikannya dalam bentuk grafik, tabel, dan ringkasan yang actionable. Laporan membantu pemilik ISP mengambil keputusan bisnis dan operasional berdasarkan data.

Reporting & Analytics terdiri dari dua komponen utama:
- **Backend** (`services/billing-api`): REST API untuk generate laporan, aggregasi data, export, jadwal otomatis, pengeluaran, target KPI, forecasting, dan custom report builder
- **Frontend** (`apps/web`): Halaman laporan interaktif dengan 4 kategori tab (Keuangan, Pelanggan, Jaringan, Operasional), filter global, grafik, perbandingan periode, dan export

Modul ini dirancang untuk **graceful degradation** — laporan tetap berfungsi meskipun beberapa modul (MikroTik, OLT, Notifikasi) belum aktif atau service sedang down. Data yang tidak tersedia disembunyikan atau ditampilkan dari cache terakhir.

## Glossary

- **Report_API**: REST API di Billing API (`/v1/reports/*`) yang menyediakan endpoint untuk generate, query, dan export laporan
- **Report_Page**: Halaman laporan interaktif di frontend (`/reports`) yang menampilkan 4 kategori laporan dengan filter global
- **Expense_API**: REST API di Billing API (`/v1/expenses/*`) untuk input dan manajemen pengeluaran manual
- **Report_Scheduler**: Background job (asynq) yang menjalankan generate laporan terjadwal dan mengirim hasilnya via email/WhatsApp
- **Export_Worker**: Background job (asynq) yang memproses export laporan ke PDF/Excel secara asinkron
- **KPI_Target**: Konfigurasi target bisnis per tenant (pendapatan, collection rate, churn rate, SLA uptime) yang ditampilkan sebagai progress bar di laporan
- **Aging_Report**: Laporan piutang yang mengelompokkan tunggakan berdasarkan umur (1-7 hari, 8-14 hari, 15-30 hari, 30+ hari)
- **Collection_Rate**: Persentase invoice yang terbayar dalam satu periode dibandingkan total invoice yang jatuh tempo
- **ARPU**: Average Revenue Per User — rata-rata pendapatan per pelanggan aktif dalam satu periode
- **CLV**: Customer Lifetime Value — estimasi total pendapatan dari satu pelanggan selama masa berlangganan
- **Churn_Rate**: Persentase pelanggan yang berhenti berlangganan dalam satu periode dibandingkan total pelanggan aktif
- **Forecasting_Engine**: Komponen yang menghitung proyeksi 3 bulan ke depan menggunakan linear regression berdasarkan data historis 6 bulan
- **Period_Comparison**: Fitur perbandingan dua periode (MoM, YoY, QoQ, custom) dengan delta dan insight otomatis
- **Custom_Report_Builder**: Fitur untuk membuat laporan custom dengan memilih metrik dan dimensi, menyimpan sebagai template, dan menjadwalkan
- **Expense_Category**: Kategori pengeluaran yang bisa dikonfigurasi per tenant (bandwidth, gaji, sewa tiang, listrik, perangkat, notifikasi, lainnya)
- **Dashboard_Widget**: Komponen ringkasan metrik kunci yang ditampilkan di halaman utama dashboard
- **Tenant**: Organisasi ISP yang menggunakan platform ISPBoss (multi-tenant SaaS)
- **Billing_API**: Go microservice (`services/billing-api/`) yang menangani pelanggan, invoice, pembayaran, dan reporting
- **Network_Service**: Go microservice (`services/network-service/`) yang menangani MikroTik, OLT, dan monitoring jaringan

## Requirements

### Requirement 1: Backend — Endpoint Laporan Keuangan (Ringkasan Pendapatan)

**User Story:** Sebagai pemilik ISP, saya ingin melihat ringkasan pendapatan bulanan yang dipecah per sumber (tagihan bulanan, voucher, biaya pasang, denda, lainnya), sehingga saya bisa memahami komposisi pendapatan bisnis saya.

#### Acceptance Criteria

1. THE Report_API SHALL expose GET /v1/reports/financial/revenue accepting query parameters: period_start (date), period_end (date), compare_start (date, nullable), compare_end (date, nullable), area_id (nullable), package_id (nullable) and returning a revenue summary object
2. THE Report_API SHALL return revenue breakdown by source: monthly_subscription (dari invoice item type 'monthly'), voucher_sales (dari voucher yang terjual), installation_fees (dari invoice item type 'installation'), late_fees (dari invoice item type 'penalty'), and other (sisa item types)
3. WHEN compare_start and compare_end parameters are provided, THE Report_API SHALL return both current period and comparison period data with calculated delta (absolute and percentage) per source
4. THE Report_API SHALL return a monthly revenue trend array for the last 12 months containing: month, total_revenue, monthly_subscription, voucher_sales, and other_revenue per month
5. WHEN area_id parameter is provided, THE Report_API SHALL filter revenue data to only include invoices and payments from customers in the specified area
6. WHEN package_id parameter is provided, THE Report_API SHALL filter revenue data to only include invoices and payments from customers with the specified package


### Requirement 2: Backend — Endpoint Laporan Piutang / Aging Report

**User Story:** Sebagai pemilik ISP, saya ingin melihat laporan piutang yang dikelompokkan berdasarkan umur tunggakan, sehingga saya bisa memprioritaskan penagihan dan memantau collection rate.

#### Acceptance Criteria

1. THE Report_API SHALL expose GET /v1/reports/financial/aging accepting query parameters: period_end (date), area_id (nullable), package_id (nullable) and returning an aging report object
2. THE Report_API SHALL group outstanding invoices into aging buckets: 1-7 days, 8-14 days, 15-30 days, and 30+ days, each containing total_amount and customer_count
3. THE Report_API SHALL calculate and return collection_rate as the percentage of invoices paid within the current period compared to total invoices due
4. THE Report_API SHALL calculate and return average_days_to_pay as the mean number of days between invoice due date and payment date for paid invoices in the period
5. THE Report_API SHALL return a top_debtors array (max 10) containing customer_id, customer_name, total_outstanding, and months_overdue, ordered by total_outstanding descending
6. THE Report_API SHALL return a receivables_trend array for the last 6 months containing: month and total_outstanding per month
7. FOR ALL aging report responses, THE Report_API SHALL ensure that the sum of amounts across all aging buckets equals the total_outstanding value (invariant)

### Requirement 3: Backend — Endpoint Laporan Pembayaran per Metode

**User Story:** Sebagai pemilik ISP, saya ingin melihat distribusi pembayaran berdasarkan metode (tunai, transfer, Xendit, QRIS), sehingga saya bisa memahami preferensi pembayaran pelanggan.

#### Acceptance Criteria

1. THE Report_API SHALL expose GET /v1/reports/financial/payments accepting query parameters: period_start (date), period_end (date), area_id (nullable), package_id (nullable) and returning a payment distribution object
2. THE Report_API SHALL return payment breakdown by method containing: method_name, total_amount, transaction_count, and percentage of total for each payment method used in the period
3. THE Report_API SHALL return a daily_payments array for the specified period containing: date, total_amount, and transaction_count per day
4. THE Report_API SHALL identify and return peak_payment_date as the date with the highest total payment amount in the period

### Requirement 4: Backend — Endpoint Laporan Pendapatan Voucher

**User Story:** Sebagai pemilik ISP, saya ingin melihat laporan penjualan voucher yang dipecah per paket dan per reseller, sehingga saya bisa mengoptimalkan strategi penjualan voucher.

#### Acceptance Criteria

1. THE Report_API SHALL expose GET /v1/reports/financial/vouchers accepting query parameters: period_start (date), period_end (date) and returning a voucher revenue report object
2. THE Report_API SHALL return voucher sales breakdown by package containing: package_name, total_revenue, voucher_count, and percentage of total per voucher package
3. THE Report_API SHALL return voucher sales breakdown by reseller containing: reseller_name, total_revenue, voucher_count, and reseller_margin per reseller
4. THE Report_API SHALL return total_reseller_margin as the sum of all reseller margins (sell_price - reseller_price) for the period

### Requirement 5: Backend — Endpoint Laporan Laba Rugi Sederhana

**User Story:** Sebagai pemilik ISP, saya ingin melihat laporan laba rugi sederhana yang menggabungkan pendapatan otomatis dari billing dengan pengeluaran yang diinput manual, sehingga saya bisa memantau profitabilitas bisnis.

#### Acceptance Criteria

1. THE Report_API SHALL expose GET /v1/reports/financial/profit-loss accepting query parameters: period_start (date), period_end (date), compare_start (date, nullable), compare_end (date, nullable) and returning a profit-loss report object
2. THE Report_API SHALL calculate total_revenue automatically from billing data: monthly subscriptions, voucher sales, installation fees, late fees, and other income
3. THE Report_API SHALL calculate total_expenses from manually inputted expense records for the period, grouped by expense category
4. THE Report_API SHALL calculate net_profit as total_revenue minus total_expenses and profit_margin as net_profit divided by total_revenue expressed as percentage
5. WHEN compare_start and compare_end parameters are provided, THE Report_API SHALL return both current and comparison period profit-loss data with calculated delta per line item

### Requirement 6: Backend — CRUD Pengeluaran (Expense Management)

**User Story:** Sebagai admin ISP, saya ingin menginput pengeluaran bisnis secara manual dengan kategori yang bisa dikonfigurasi, sehingga data pengeluaran tersedia untuk laporan laba rugi.

#### Acceptance Criteria

1. THE Expense_API SHALL expose POST /v1/expenses accepting: category_id, amount (integer, dalam Rupiah), description, expense_date, is_recurring (boolean), recurring_day (nullable, 1-28 untuk tanggal auto-repeat bulanan) and returning the created expense with HTTP 201
2. THE Expense_API SHALL expose GET /v1/expenses accepting query parameters: period_start (date), period_end (date), category_id (nullable) and returning a list of expenses ordered by expense_date descending
3. THE Expense_API SHALL expose PUT /v1/expenses/:id accepting category_id, amount, description, expense_date, is_recurring, recurring_day and returning the updated expense
4. THE Expense_API SHALL expose DELETE /v1/expenses/:id performing a soft delete and returning HTTP 204
5. THE Expense_API SHALL expose GET /v1/expenses/categories returning the list of expense categories for the tenant
6. THE Expense_API SHALL expose POST /v1/expenses/categories accepting name and returning the created category with HTTP 201
7. THE Expense_API SHALL expose PUT /v1/expenses/categories/:id accepting name and returning the updated category
8. THE Expense_API SHALL expose DELETE /v1/expenses/categories/:id performing a soft delete — IF the category has expenses, THEN THE Expense_API SHALL reject deletion with an error
9. THE Expense_API SHALL provide default expense categories for new tenants: Bandwidth/Upstream, Gaji Karyawan, Sewa Tiang/Infrastruktur, Listrik & Operasional, Perangkat, Notifikasi, Lainnya
10. WHEN is_recurring is true and recurring_day is set, THE Report_Scheduler SHALL auto-create the expense on the specified day each month


### Requirement 7: Backend — Endpoint Laporan Pelanggan (Pertumbuhan & Distribusi)

**User Story:** Sebagai pemilik ISP, saya ingin melihat pertumbuhan pelanggan, distribusi per paket/area/status, dan metrik ARPU/CLV, sehingga saya bisa memahami dinamika basis pelanggan.

#### Acceptance Criteria

1. THE Report_API SHALL expose GET /v1/reports/customers/growth accepting query parameters: period_start (date), period_end (date), compare_start (date, nullable), compare_end (date, nullable) and returning a customer growth report object
2. THE Report_API SHALL return: total_active (pelanggan aktif saat ini), new_customers (pelanggan baru dalam periode), churned_customers (pelanggan berhenti dalam periode), and net_growth (new minus churned)
3. THE Report_API SHALL return a monthly_growth_trend array for the last 12 months containing: month, total_active, new_customers, and churned_customers per month
4. THE Report_API SHALL calculate and return arpu (Average Revenue Per User) as total revenue divided by average active customers in the period
5. THE Report_API SHALL calculate and return clv (Customer Lifetime Value) as ARPU multiplied by average customer lifetime in months
6. THE Report_API SHALL calculate and return churn_rate as churned_customers divided by total_active at the start of the period, expressed as percentage
7. WHEN compare_start and compare_end parameters are provided, THE Report_API SHALL return both current and comparison period data with calculated delta for each metric

### Requirement 8: Backend — Endpoint Laporan Distribusi Pelanggan

**User Story:** Sebagai pemilik ISP, saya ingin melihat distribusi pelanggan berdasarkan paket, area, status, dan metode koneksi, sehingga saya bisa mengidentifikasi segmen pelanggan terbesar.

#### Acceptance Criteria

1. THE Report_API SHALL expose GET /v1/reports/customers/distribution accepting query parameters: period_end (date) and returning distribution breakdowns
2. THE Report_API SHALL return distribution_by_package array containing: package_id, package_name, customer_count, and percentage per package
3. THE Report_API SHALL return distribution_by_area array containing: area_id, area_name, customer_count, and percentage per area
4. THE Report_API SHALL return distribution_by_status object containing customer counts for each status: aktif, pending, isolir, suspend, berhenti
5. THE Report_API SHALL return distribution_by_connection_method array containing: connection_method, customer_count, and percentage per method (pppoe, hotspot, dhcp_binding, static)

### Requirement 9: Backend — Endpoint Laporan Churn Analysis

**User Story:** Sebagai pemilik ISP, saya ingin menganalisis pelanggan yang berhenti berlangganan berdasarkan alasan, paket, dan area, sehingga saya bisa mengambil tindakan untuk mengurangi churn.

#### Acceptance Criteria

1. THE Report_API SHALL expose GET /v1/reports/customers/churn accepting query parameters: period_start (date), period_end (date) and returning a churn analysis report object
2. THE Report_API SHALL return churned_customers_count and churn_rate for the period
3. THE Report_API SHALL return churn_by_reason array containing: reason, count, and percentage per churn reason (pindah rumah, harga mahal, pindah ISP lain, kualitas jaringan, tidak diketahui)
4. THE Report_API SHALL return churn_by_package array containing: package_name, churned_count, and churn_rate per package
5. THE Report_API SHALL return churn_by_area array containing: area_name, churned_count, and churn_rate per area
6. THE Report_API SHALL return average_lifetime_months as the mean subscription duration of churned customers in the period

### Requirement 10: Backend — Endpoint Laporan Pendapatan per Area

**User Story:** Sebagai pemilik ISP, saya ingin melihat pendapatan, piutang, dan ARPU per area, sehingga saya bisa mengidentifikasi area paling menguntungkan dan area yang perlu perhatian.

#### Acceptance Criteria

1. THE Report_API SHALL expose GET /v1/reports/financial/revenue-by-area accepting query parameters: period_start (date), period_end (date) and returning a revenue-by-area report object
2. THE Report_API SHALL return an areas array containing per area: area_id, area_name, customer_count, total_revenue, total_outstanding, and arpu
3. THE Report_API SHALL return total row with aggregated values across all areas
4. THE Report_API SHALL identify and return most_profitable_area (highest ARPU with low outstanding) and attention_needed_area (highest outstanding relative to revenue)
5. WHEN a user requests drill-down for a specific area, THE Report_API SHALL return customer-level detail for that area via GET /v1/reports/financial/revenue-by-area/:area_id/customers

### Requirement 11: Backend — Endpoint Laporan Jaringan (Uptime Router)

**User Story:** Sebagai pemilik ISP, saya ingin melihat uptime dan status setiap router MikroTik, sehingga saya bisa memantau SLA dan mengidentifikasi router bermasalah.

#### Acceptance Criteria

1. THE Report_API SHALL expose GET /v1/reports/network/uptime accepting query parameters: period_start (date), period_end (date), router_id (nullable) and returning a router uptime report object
2. THE Report_API SHALL return per router: router_id, router_name, uptime_percentage, total_downtime_minutes, reboot_count, and status_label (Excellent ≥99.9%, Good ≥99.5%, Fair ≥98%, Poor <98%)
3. THE Report_API SHALL return sla_target (from KPI settings) and routers_below_sla array listing routers with uptime below the SLA target
4. WHEN router_id parameter is provided, THE Report_API SHALL return a downtime_timeline array for the specified router containing: start_time, end_time, duration_minutes, and cause (if known)
5. IF the MikroTik module is not active for the tenant, THEN THE Report_API SHALL return an empty response with a module_inactive flag instead of an error

### Requirement 12: Backend — Endpoint Laporan Traffic Jaringan

**User Story:** Sebagai pemilik ISP, saya ingin melihat laporan traffic per router dan per pelanggan, sehingga saya bisa memantau penggunaan bandwidth dan mengidentifikasi pelanggan over-use.

#### Acceptance Criteria

1. THE Report_API SHALL expose GET /v1/reports/network/traffic accepting query parameters: period_start (date), period_end (date), router_id (nullable) and returning a traffic report object
2. THE Report_API SHALL return total_download_bytes, total_upload_bytes, total_traffic_bytes, peak_traffic_bps, peak_traffic_time, and average_traffic_bps for the period
3. THE Report_API SHALL return traffic_by_router array containing: router_id, router_name, download_bytes, upload_bytes, and percentage of total per router
4. THE Report_API SHALL return top_customers array (max 10) containing: customer_id, customer_name, package_name, download_bytes, upload_bytes, and over_use_flag (true if traffic significantly exceeds package bandwidth expectation)
5. IF the MikroTik module is not active for the tenant, THEN THE Report_API SHALL return an empty response with a module_inactive flag

### Requirement 13: Backend — Endpoint Laporan Signal Quality OLT

**User Story:** Sebagai pemilik ISP, saya ingin melihat laporan kualitas signal ONT dan ringkasan alarm OLT, sehingga saya bisa melakukan maintenance proaktif pada jaringan fiber.

#### Acceptance Criteria

1. THE Report_API SHALL expose GET /v1/reports/network/signal-quality accepting query parameters: period_start (date), period_end (date), olt_id (nullable) and returning a signal quality report object
2. THE Report_API SHALL return signal distribution: normal_count (signal ≥ -20 dBm), warning_count (-20 to -25 dBm), weak_count (-25 to -27 dBm), critical_count (< -27 dBm), and total_ont_count
3. THE Report_API SHALL return average_signal_dbm across all ONTs
4. THE Report_API SHALL return degrading_onts array listing ONTs whose signal has dropped more than 2 dB in the last 30 days, containing: customer_name, customer_id, current_signal_dbm, signal_change_db
5. THE Report_API SHALL return alarm_summary containing: total_alarms, alarms_by_type (LOS, Dying Gasp, Signal Degraded, etc.) with count, average_duration_minutes, and resolved_percentage per type
6. IF the OLT module is not active for the tenant, THEN THE Report_API SHALL return an empty response with a module_inactive flag


### Requirement 14: Backend — Endpoint Laporan Kapasitas Jaringan

**User Story:** Sebagai pemilik ISP, saya ingin melihat kapasitas router dan ODP, sehingga saya bisa merencanakan penambahan perangkat sebelum kapasitas penuh.

#### Acceptance Criteria

1. THE Report_API SHALL expose GET /v1/reports/network/capacity and returning a capacity report object
2. THE Report_API SHALL return router_capacity array containing per router: router_id, router_name, current_customers, max_capacity, usage_percentage, and estimated_full_date (projected date when capacity reaches 100% based on current growth rate)
3. THE Report_API SHALL return odp_capacity array containing per ODP: odp_id, odp_name, used_ports, total_ports, usage_percentage, and status_label (OK <75%, Hampir Penuh ≥75%, Penuh 100%)
4. THE Report_API SHALL return recommendations array containing actionable suggestions for routers or ODPs approaching capacity (usage ≥ 80%)
5. IF the MikroTik or OLT module is not active, THEN THE Report_API SHALL omit the corresponding capacity section and include a module_inactive flag

### Requirement 15: Backend — Endpoint Laporan Operasional (Aktivitas Admin)

**User Story:** Sebagai pemilik ISP, saya ingin melihat aktivitas admin dan operator, sehingga saya bisa memantau produktivitas tim dan audit operasional.

#### Acceptance Criteria

1. THE Report_API SHALL expose GET /v1/reports/operational/activity accepting query parameters: period_start (date), period_end (date) and returning an admin activity report object
2. THE Report_API SHALL return per_user_activity array containing: user_id, user_name, role, login_days (jumlah hari login), action_count (total aksi), and last_active_at per user
3. THE Report_API SHALL return top_actions array containing: action_type (catat pembayaran, edit pelanggan, kirim notifikasi, isolir/buka isolir, etc.), count, and percentage of total

### Requirement 16: Backend — Endpoint Laporan Notifikasi

**User Story:** Sebagai admin ISP, saya ingin melihat statistik pengiriman notifikasi per channel dan per template, sehingga saya bisa memantau efektivitas dan biaya notifikasi.

#### Acceptance Criteria

1. THE Report_API SHALL expose GET /v1/reports/operational/notifications accepting query parameters: period_start (date), period_end (date) and returning a notification statistics report object
2. THE Report_API SHALL return totals: total_sent, total_delivered, total_failed, success_rate (percentage), and total_cost
3. THE Report_API SHALL return per_channel array containing: channel (whatsapp, sms, email), sent_count, delivered_count, failed_count, success_rate, and cost per channel
4. THE Report_API SHALL return per_template array containing: template_name, sent_count per notification template
5. IF the Notification module is not active, THEN THE Report_API SHALL return an empty response with a module_inactive flag

### Requirement 17: Backend — Endpoint Laporan Sync MikroTik & OLT

**User Story:** Sebagai admin ISP, saya ingin melihat status sinkronisasi antara database dan perangkat jaringan, sehingga saya bisa mengidentifikasi masalah sync dan orphan users.

#### Acceptance Criteria

1. THE Report_API SHALL expose GET /v1/reports/operational/sync accepting query parameters: period_start (date), period_end (date) and returning a sync status report object
2. THE Report_API SHALL return mikrotik_sync per router: router_id, router_name, sync_ok_count, sync_failed_count, orphan_user_count, and pending_sync_count
3. THE Report_API SHALL return olt_sync per OLT: olt_id, olt_name, sync_ok_count, sync_failed_count, and unmanaged_ont_count
4. THE Report_API SHALL return overall sync_success_rate as the percentage of successful syncs across all devices
5. IF the MikroTik or OLT module is not active, THEN THE Report_API SHALL omit the corresponding sync section and include a module_inactive flag

### Requirement 18: Backend — Export Laporan (PDF, Excel, CSV)

**User Story:** Sebagai pemilik ISP, saya ingin mengexport laporan ke PDF (dengan branding tenant), Excel, dan CSV, sehingga saya bisa mencetak, mengarsip, atau menganalisis data lebih lanjut.

#### Acceptance Criteria

1. THE Report_API SHALL expose POST /v1/reports/export accepting: report_type (revenue, aging, payments, vouchers, profit_loss, customer_growth, distribution, churn, revenue_by_area, uptime, traffic, signal_quality, capacity, activity, notifications, sync), format (pdf, xlsx, csv), filters (period_start, period_end, area_id, package_id, router_id) and returning a job_id for async processing
2. WHEN format is 'csv', THE Report_API SHALL generate the CSV file synchronously and return the file directly in the response
3. WHEN format is 'pdf' or 'xlsx', THE Export_Worker SHALL process the export asynchronously via asynq background job
4. WHEN a PDF export is generated, THE Export_Worker SHALL include tenant branding: logo, ISP name, and tagline in the header, and generation timestamp with ISPBoss attribution in the footer
5. WHEN a PDF export is generated, THE Export_Worker SHALL render charts as images embedded in the PDF document
6. THE Report_API SHALL expose GET /v1/reports/export/:job_id returning the export job status (pending, processing, completed, failed) and download URL when completed
7. WHEN an export job is completed, THE Report_API SHALL notify the requesting user via WebSocket event
8. IF an export job fails, THEN THE Export_Worker SHALL retry once automatically — IF the retry also fails, THEN THE Export_Worker SHALL mark the job as failed and notify the admin

### Requirement 19: Backend — Jadwal Laporan Otomatis

**User Story:** Sebagai pemilik ISP, saya ingin menjadwalkan laporan untuk digenerate dan dikirim otomatis secara harian/mingguan/bulanan, sehingga saya menerima laporan rutin tanpa harus login.

#### Acceptance Criteria

1. THE Report_API SHALL expose POST /v1/reports/schedules accepting: report_type, schedule_type (daily, weekly, monthly), format (pdf, xlsx), recipients (array of {type: 'email'|'whatsapp', address: string}), and filters and returning the created schedule with HTTP 201
2. THE Report_API SHALL expose GET /v1/reports/schedules returning all active report schedules for the tenant
3. THE Report_API SHALL expose PUT /v1/reports/schedules/:id accepting updated schedule configuration and returning the updated schedule
4. THE Report_API SHALL expose DELETE /v1/reports/schedules/:id deactivating the schedule and returning HTTP 204
5. WHEN schedule_type is 'daily', THE Report_Scheduler SHALL generate the report every day at 07:00 tenant local time
6. WHEN schedule_type is 'weekly', THE Report_Scheduler SHALL generate the report every Monday at 07:00 tenant local time
7. WHEN schedule_type is 'monthly', THE Report_Scheduler SHALL generate the report on the 1st of each month at 07:00 tenant local time
8. WHEN a scheduled report is generated, THE Report_Scheduler SHALL send the report via the configured channels: email (PDF attachment) and/or WhatsApp (download link)
9. THE Report_API SHALL retain generated scheduled reports for 12 months, after which they are automatically deleted


### Requirement 20: Backend — Target KPI Setting

**User Story:** Sebagai pemilik ISP, saya ingin mengatur target bisnis (pendapatan, collection rate, churn rate, SLA uptime), sehingga laporan menampilkan progress terhadap target sebagai pembanding.

#### Acceptance Criteria

1. THE Report_API SHALL expose GET /v1/reports/kpi-targets returning the current KPI targets for the tenant
2. THE Report_API SHALL expose PUT /v1/reports/kpi-targets accepting: monthly_revenue_target (integer, Rupiah), collection_rate_target (percentage), max_receivables (integer, Rupiah), new_customers_monthly_target (integer), max_churn_rate (percentage), total_customers_target (integer, akhir tahun), sla_uptime_target (percentage), max_active_alarms (integer), min_signal_quality_percentage (percentage of ONTs with normal signal) and returning the updated targets
3. WHEN KPI targets are set, THE Report_API SHALL include target values and progress indicators in all relevant report responses: revenue reports include monthly_revenue_target, aging reports include collection_rate_target, customer reports include churn_rate_target and new_customers_target, uptime reports include sla_uptime_target
4. THE Report_API SHALL calculate progress_percentage for each KPI as current_value divided by target_value and return a status_label: tercapai (≥100%), hampir (≥80%), di_bawah_target (<80%)
5. WHEN a tenant has no KPI targets configured, THE Report_API SHALL return reports without target comparisons (targets are optional)

### Requirement 21: Backend — Forecasting / Proyeksi

**User Story:** Sebagai pemilik ISP, saya ingin melihat proyeksi pendapatan dan pertumbuhan pelanggan 3 bulan ke depan, sehingga saya bisa merencanakan bisnis berdasarkan trend.

#### Acceptance Criteria

1. THE Report_API SHALL expose GET /v1/reports/forecast returning a forecast report object based on the last 6 months of historical data
2. THE Forecasting_Engine SHALL calculate projections for the next 3 months using simple linear regression for: monthly_revenue, total_active_customers, and total_receivables
3. THE Report_API SHALL return per projected month: month, projected_revenue, projected_customers, and projected_receivables
4. WHEN KPI targets are set, THE Report_API SHALL return estimated_target_date indicating when each KPI target is projected to be reached based on the current trend
5. THE Report_API SHALL include a disclaimer flag indicating that projections are estimates based on linear trend and actual results may differ due to seasonal factors, promotions, or market changes
6. IF fewer than 3 months of historical data are available, THEN THE Report_API SHALL return an insufficient_data flag instead of projections

### Requirement 22: Backend — Perbandingan Antar Periode

**User Story:** Sebagai pemilik ISP, saya ingin membandingkan metrik bisnis antara dua periode (MoM, YoY, QoQ, custom), sehingga saya bisa melihat trend dan perubahan performa.

#### Acceptance Criteria

1. THE Report_API SHALL expose GET /v1/reports/comparison accepting: comparison_type (mom, yoy, qoq, custom), base_period_start (date), base_period_end (date), compare_period_start (date, required for custom), compare_period_end (date, required for custom) and returning a comparison report object
2. WHEN comparison_type is 'mom', THE Report_API SHALL automatically set the comparison period to the previous month
3. WHEN comparison_type is 'yoy', THE Report_API SHALL automatically set the comparison period to the same month in the previous year
4. WHEN comparison_type is 'qoq', THE Report_API SHALL automatically set the comparison period to the previous quarter
5. THE Report_API SHALL return a metrics array containing per metric: metric_name, base_value, compare_value, delta_absolute, delta_percentage, and trend (improving, declining, stable) for: pendapatan, pelanggan_aktif, arpu, collection_rate, churn_rate, piutang, and router_uptime
6. THE Report_API SHALL return an insights array containing 3-5 auto-generated text insights based on the largest deltas (e.g., "Pendapatan naik 56.3% dibanding periode sebelumnya", "ARPU turun 4.0% — kemungkinan karena banyak pelanggan Basic baru")

### Requirement 23: Backend — Custom Report Builder

**User Story:** Sebagai admin ISP, saya ingin membuat laporan custom dengan memilih metrik dan dimensi sendiri, menyimpan sebagai template, dan menjadwalkan, sehingga saya bisa mendapatkan laporan yang sesuai kebutuhan spesifik.

#### Acceptance Criteria

1. THE Report_API SHALL expose POST /v1/reports/custom/preview accepting: metrics (array, max 3, from: customer_count, revenue, outstanding, arpu, collection_rate, churn_rate, traffic_gb, average_signal_dbm), group_by (primary dimension from: area, package, month, status, connection_method, router), sub_group_by (secondary dimension, nullable), period_start (date), period_end (date), display_type (table, bar_chart, line_chart, pie_chart) and returning the report data
2. THE Report_API SHALL expose POST /v1/reports/custom/templates accepting: name, metrics, group_by, sub_group_by, display_type, and default_period_range and returning the saved template with HTTP 201
3. THE Report_API SHALL expose GET /v1/reports/custom/templates returning all saved custom report templates for the tenant
4. THE Report_API SHALL expose DELETE /v1/reports/custom/templates/:id deleting the template and returning HTTP 204
5. THE Report_API SHALL enforce a maximum of 3 metrics and 2 dimensions (group_by + sub_group_by) per custom report to maintain readability
6. THE Custom_Report_Builder SHALL support scheduling custom reports using the same schedule mechanism as built-in reports (Requirement 19)

### Requirement 24: Backend — Dashboard Widget Data

**User Story:** Sebagai pemilik ISP, saya ingin melihat metrik kunci di halaman utama dashboard, sehingga saya bisa memantau kesehatan bisnis secara sekilas tanpa membuka halaman laporan.

#### Acceptance Criteria

1. THE Report_API SHALL expose GET /v1/reports/dashboard returning a dashboard widget data object optimized for fast loading (target < 500ms)
2. THE Report_API SHALL return widget data: total_active_customers (with trend vs previous month), monthly_revenue (with percentage of target), total_receivables (with customer count), routers_online_count and routers_offline_count, collection_rate (with target), churn_rate (with target), and arpu
3. WHEN KPI targets are set, THE Report_API SHALL include target values and progress indicators for each applicable widget
4. THE Report_API SHALL cache dashboard widget data in Redis with a TTL of 5 minutes to ensure fast response times
5. WHEN underlying data changes (payment received, customer created, etc.), THE Report_API SHALL invalidate the relevant dashboard cache entries


### Requirement 25: Backend — Graceful Degradation

**User Story:** Sebagai pemilik ISP, saya ingin laporan tetap berfungsi meskipun beberapa modul tidak aktif atau service sedang down, sehingga saya selalu bisa mengakses data yang tersedia.

#### Acceptance Criteria

1. WHEN the MikroTik module is not active for a tenant, THE Report_API SHALL hide network reports (uptime, traffic) and return module_inactive flag — financial and customer reports SHALL remain fully functional
2. WHEN the OLT module is not active for a tenant, THE Report_API SHALL hide signal quality and alarm reports and return module_inactive flag — other reports SHALL remain fully functional
3. WHEN the Notification module is not active for a tenant, THE Report_API SHALL hide notification statistics and return module_inactive flag — other reports SHALL remain fully functional
4. WHEN the Network Service is temporarily unavailable, THE Report_API SHALL return cached network report data (max 1 hour old) with a stale_data flag and last_updated timestamp
5. WHEN no data exists for a period (new tenant), THE Report_API SHALL return an empty_state response with a descriptive message: "Belum ada data untuk periode ini. Data akan muncul setelah ada pelanggan aktif."
6. IF a report generation fails, THEN THE Report_API SHALL return a partial response with available data and an errors array listing which sections could not be generated

### Requirement 26: Frontend — Halaman Laporan dengan Tab Kategori dan Filter Global

**User Story:** Sebagai pemilik ISP, saya ingin halaman laporan dengan 4 tab kategori (Keuangan, Pelanggan, Jaringan, Operasional) dan filter global, sehingga saya bisa menavigasi dan memfilter semua laporan dari satu halaman.

#### Acceptance Criteria

1. THE Report_Page SHALL render at route `/reports` with 4 category tabs: Keuangan (💰), Pelanggan (👥), Jaringan (📡), Operasional (⚙️)
2. THE Report_Page SHALL display a global filter bar at the top with: Periode (dropdown: Hari ini, Minggu ini, Bulan ini, Kuartal, Tahun, Custom range), Bandingkan (dropdown: periode sebelumnya for comparison), Area (dropdown: Semua or specific area), Paket (dropdown: Semua or specific package), and Terapkan/Reset buttons
3. WHEN the Jaringan tab is active, THE Report_Page SHALL display an additional Router filter dropdown
4. WHEN filters are applied and Terapkan is clicked, THE Report_Page SHALL reload all visible report components with the new filter parameters
5. THE Report_Page SHALL persist the selected tab and filter state in the URL query parameters so that the page can be bookmarked and shared
6. THE Report_Page SHALL display export buttons at the bottom: Export PDF, Export Excel, and Jadwalkan Laporan
7. WHEN displayed on mobile, THE Report_Page SHALL use a swipeable horizontal tab bar for categories and a collapsible filter section

### Requirement 27: Frontend — Visualisasi Laporan Keuangan

**User Story:** Sebagai pemilik ISP, saya ingin melihat laporan keuangan dengan grafik pendapatan, tabel aging, distribusi pembayaran, dan laba rugi, sehingga saya bisa memahami kesehatan keuangan bisnis secara visual.

#### Acceptance Criteria

1. WHEN the Keuangan tab is active, THE Report_Page SHALL display sections: Ringkasan Pendapatan, Piutang/Aging, Pembayaran per Metode, Pendapatan Voucher, Laba Rugi Sederhana, and Pendapatan per Area
2. THE Report_Page SHALL render the Ringkasan Pendapatan section with: 4 summary cards (Total, Bulanan, Voucher, Lainnya) each showing amount and delta percentage vs comparison period, and a 12-month stacked bar chart showing revenue by source
3. THE Report_Page SHALL render the Aging Report section with: 4 aging bucket cards (1-7 hari, 8-14 hari, 15-30 hari, 30+ hari) each showing amount and customer count, collection rate with target progress bar, average days to pay, top 10 debtors table, and 6-month receivables trend line chart
4. THE Report_Page SHALL render the Pembayaran section with: payment method distribution cards showing amount and percentage per method, and a daily payment bar chart for the current period
5. THE Report_Page SHALL render the Laba Rugi section with: revenue line items (auto from billing), expense line items (from manual input), net profit, and profit margin percentage
6. WHEN KPI targets are set, THE Report_Page SHALL display progress bars with color coding: hijau (tercapai ≥100%), kuning (hampir ≥80%), merah (di bawah <80%)

### Requirement 28: Frontend — Visualisasi Laporan Pelanggan

**User Story:** Sebagai pemilik ISP, saya ingin melihat laporan pelanggan dengan grafik pertumbuhan, distribusi, dan analisis churn, sehingga saya bisa memahami dinamika basis pelanggan secara visual.

#### Acceptance Criteria

1. WHEN the Pelanggan tab is active, THE Report_Page SHALL display sections: Pertumbuhan Pelanggan, Distribusi Pelanggan, and Analisis Churn
2. THE Report_Page SHALL render the Pertumbuhan section with: 4 summary cards (Total Aktif, Baru, Churn, Net Growth) each showing count and delta vs comparison period, a 12-month line chart showing total active customers with new and churned overlays, and ARPU, CLV, and Churn Rate metrics
3. THE Report_Page SHALL render the Distribusi section with: pie/donut charts for distribution by package, area, status, and connection method
4. THE Report_Page SHALL render the Churn Analysis section with: churned count, churn rate, breakdown by reason (bar chart), breakdown by package and area (tables), and average lifetime before churn

### Requirement 29: Frontend — Visualisasi Laporan Jaringan

**User Story:** Sebagai pemilik ISP, saya ingin melihat laporan jaringan dengan uptime router, traffic, signal quality, alarm, dan kapasitas, sehingga saya bisa memantau kesehatan infrastruktur secara visual.

#### Acceptance Criteria

1. WHEN the Jaringan tab is active, THE Report_Page SHALL display sections: Uptime Router, Traffic, Signal Quality, Alarm Summary, and Kapasitas Jaringan
2. THE Report_Page SHALL render the Uptime section with: a table showing per router uptime percentage, downtime, reboot count, and status label with color coding, SLA target line, and routers below SLA highlighted
3. THE Report_Page SHALL render the Traffic section with: total traffic summary cards (download, upload, total) with delta, peak traffic info, traffic distribution per router (horizontal bar chart), and top 10 customers by traffic table
4. THE Report_Page SHALL render the Signal Quality section with: signal distribution cards (Normal, Warning, Weak, Critical) with counts and percentages, average signal, degrading ONTs list, and alarm summary table by type
5. THE Report_Page SHALL render the Kapasitas section with: router capacity table with usage percentage and estimated full date, ODP capacity table with port usage and status, and recommendations list
6. WHEN the MikroTik or OLT module is not active, THE Report_Page SHALL hide the corresponding sections and display a message: "Modul [nama] belum aktif. Aktifkan modul untuk melihat laporan ini."

### Requirement 30: Frontend — Visualisasi Laporan Operasional

**User Story:** Sebagai pemilik ISP, saya ingin melihat laporan operasional dengan aktivitas admin, statistik notifikasi, dan status sync, sehingga saya bisa memantau operasional harian.

#### Acceptance Criteria

1. WHEN the Operasional tab is active, THE Report_Page SHALL display sections: Aktivitas Admin, Notifikasi, and Sync Status
2. THE Report_Page SHALL render the Aktivitas section with: a table showing per user login days, action count, role, and last active date, and top actions list with counts
3. THE Report_Page SHALL render the Notifikasi section with: summary cards (Total Kirim, Terkirim, Gagal, Biaya), per channel breakdown (WhatsApp, SMS, Email) with success rate and cost, and per template usage counts
4. THE Report_Page SHALL render the Sync Status section with: MikroTik sync table per router (sync OK, failed, orphan, pending), OLT sync table per OLT (sync OK, failed, unmanaged), and overall sync success rate


### Requirement 31: Frontend — Perbandingan Periode dan Insight Otomatis

**User Story:** Sebagai pemilik ISP, saya ingin membandingkan metrik bisnis antara dua periode secara side-by-side dengan insight otomatis, sehingga saya bisa memahami perubahan performa dengan cepat.

#### Acceptance Criteria

1. THE Report_Page SHALL provide a comparison view accessible from the Bandingkan filter with options: MoM, YoY, QoQ, and Custom
2. THE Report_Page SHALL render a side-by-side comparison table showing: metric name, base period value, comparison period value, delta absolute, delta percentage, and trend indicator (arrow up green for improving, arrow down red for declining)
3. THE Report_Page SHALL display 3-5 auto-generated insight cards below the comparison table, each containing a text description of a significant change and its possible cause
4. THE Report_Page SHALL color-code delta values: green for improvements, red for declines, gray for stable (delta < 1%)

### Requirement 32: Frontend — Forecasting Visualization

**User Story:** Sebagai pemilik ISP, saya ingin melihat proyeksi pendapatan dan pelanggan 3 bulan ke depan dalam grafik, sehingga saya bisa merencanakan bisnis berdasarkan trend visual.

#### Acceptance Criteria

1. THE Report_Page SHALL display a Proyeksi section in the Keuangan tab showing a line chart with: actual data (solid line) for the last 6 months and projected data (dashed line) for the next 3 months
2. THE Report_Page SHALL display projected values per month: projected revenue, projected customers, and projected receivables
3. WHEN KPI targets are set, THE Report_Page SHALL display a horizontal target line on the chart and indicate the estimated month when the target will be reached
4. THE Report_Page SHALL display a disclaimer: "Proyeksi berdasarkan linear trend 6 bulan terakhir. Hasil aktual bisa berbeda karena faktor musiman, promo, atau perubahan pasar."
5. WHEN fewer than 3 months of historical data are available, THE Report_Page SHALL display a message: "Data historis belum cukup untuk proyeksi. Minimal 3 bulan data diperlukan."

### Requirement 33: Frontend — Custom Report Builder UI

**User Story:** Sebagai admin ISP, saya ingin antarmuka visual untuk membuat laporan custom dengan memilih metrik dan dimensi, sehingga saya bisa membuat laporan sesuai kebutuhan tanpa bantuan teknis.

#### Acceptance Criteria

1. THE Report_Page SHALL provide a Custom Report Builder accessible from a "Buat Laporan Custom" button
2. THE Custom_Report_Builder SHALL display a form with: Nama Laporan (text input), Metrik (checkbox list, max 3: Jumlah Pelanggan, Pendapatan, Piutang, ARPU, Collection Rate, Churn Rate, Traffic GB, Signal Rata-rata dBm), Group By (radio: Area, Paket, Bulan, Status, Metode Koneksi, Router), Sub-Group (optional dropdown), Periode (date range picker), and Tampilan (radio: Tabel, Grafik Bar, Grafik Line, Pie)
3. THE Custom_Report_Builder SHALL provide a Preview button that renders the report data in the selected display format without saving
4. THE Custom_Report_Builder SHALL provide a "Simpan sebagai Template" button that saves the configuration for reuse
5. THE Custom_Report_Builder SHALL provide an Export button that triggers export in PDF or Excel format
6. THE Custom_Report_Builder SHALL display saved templates in a list with options to: load, edit, delete, and schedule

### Requirement 34: Frontend — Halaman Pengeluaran

**User Story:** Sebagai admin ISP, saya ingin halaman untuk menginput dan mengelola pengeluaran bisnis, sehingga data pengeluaran tersedia untuk laporan laba rugi.

#### Acceptance Criteria

1. THE Report_Page SHALL provide an Pengeluaran page accessible from the navigation menu showing a list of expenses for the selected period
2. THE Report_Page SHALL display expenses in a table with columns: Kategori, Jumlah, Tanggal, Keterangan, Recurring indicator, and action menu (edit, delete)
3. THE Report_Page SHALL provide a "Tambah Pengeluaran" button that opens a form with: Kategori (dropdown from configured categories), Jumlah (currency input), Keterangan (text), Tanggal (date picker), Recurring toggle (with day-of-month selector when enabled)
4. THE Report_Page SHALL display total expenses for the selected period at the bottom of the table
5. THE Report_Page SHALL provide a link to manage expense categories (add, edit, delete)

### Requirement 35: Frontend — Dashboard Widget

**User Story:** Sebagai pemilik ISP, saya ingin melihat metrik kunci di halaman utama dashboard, sehingga saya bisa memantau kesehatan bisnis secara sekilas.

#### Acceptance Criteria

1. THE Dashboard_Widget SHALL display on the main dashboard page (`/`) a grid of metric cards: Total Pelanggan Aktif (with trend), Pendapatan Bulan Ini (with target progress), Tunggakan (with customer count), Router Online/Offline (with alert count), Collection Rate (with target), Churn Rate (with target), and ARPU
2. WHEN KPI targets are set, THE Dashboard_Widget SHALL display progress bars on applicable cards with color coding: hijau (tercapai), kuning (hampir), merah (di bawah target)
3. THE Dashboard_Widget SHALL auto-refresh data every 5 minutes without full page reload
4. WHEN a user clicks on a dashboard widget, THE Dashboard_Widget SHALL navigate to the corresponding detailed report section in the Report_Page
5. WHEN displayed on mobile, THE Dashboard_Widget SHALL render cards in a single-column stacked layout with touch-friendly sizing (min 44x44px touch targets)

### Requirement 36: Frontend — Mobile Responsive Layout

**User Story:** Sebagai pemilik ISP, saya ingin mengakses laporan dari perangkat mobile dengan layout yang nyaman, sehingga saya bisa memantau bisnis dari mana saja.

#### Acceptance Criteria

1. THE Report_Page SHALL use a card-based layout on mobile (viewport < 768px) for all summary metrics, replacing side-by-side cards with stacked single-column cards
2. THE Report_Page SHALL use horizontal swipe for category tabs on mobile instead of a full tab bar
3. THE Report_Page SHALL make charts horizontally scrollable on mobile when the chart width exceeds the viewport
4. THE Report_Page SHALL collapse the global filter bar into an expandable section on mobile, showing only the active period by default
5. THE Report_Page SHALL ensure all interactive elements have a minimum touch target of 44x44px and text is at least 16px on mobile

### Requirement 37: Frontend — Jadwal Laporan UI

**User Story:** Sebagai pemilik ISP, saya ingin antarmuka untuk menjadwalkan laporan otomatis, sehingga saya bisa mengatur pengiriman laporan rutin tanpa harus login.

#### Acceptance Criteria

1. THE Report_Page SHALL provide a "Jadwalkan Laporan" button that opens a schedule configuration dialog
2. THE Report_Page SHALL display the schedule form with: Laporan (dropdown of available report types), Jadwal (radio: Harian, Mingguan, Bulanan), Format (radio: PDF, Excel), Kirim ke (checkboxes: Email with address input, WhatsApp with phone input), and Penerima tambahan (repeatable email/phone inputs)
3. THE Report_Page SHALL display a list of active schedules with: report name, schedule type, format, recipients, and action buttons (edit, delete)
4. WHEN a schedule is created or updated, THE Report_Page SHALL display a confirmation message with the next scheduled generation time