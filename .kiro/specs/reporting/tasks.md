# Implementation Plan: Reporting & Analytics

## Overview

Implementasi bottom-up modul Reporting & Analytics untuk ISPBoss. Dimulai dari database migrations (6 tabel baru), domain entities (DTOs, pure functions, errors), property tests, sqlc queries (CRUD + aggregasi kompleks), repository wrappers, usecase layer (8 usecase), HTTP handlers (9 handler), route registration + wiring di main.go, background workers (3 worker), NetworkServiceClient untuk cross-service calls, dan frontend components (halaman laporan, pengeluaran, dashboard widget, chart components). Setiap task membangun di atas task sebelumnya dan bisa divalidasi secara independen. Backend menggunakan Go (Fiber v2, sqlc, pgx, asynq, rapid untuk PBT). Frontend menggunakan TypeScript/Next.js App Router dengan Recharts untuk chart. Semua komentar dalam Bahasa Indonesia. Maksimal 200 baris per file Go.

## Tasks

- [x] 1. Database migrations
  - [x] 1.1 Buat migration: tabel expense_categories
    - Buat `services/billing-api/migrations/XXXXXX_create_expense_categories.up.sql` — tabel `expense_categories` dengan kolom: `id` UUID PK DEFAULT gen_random_uuid(), `tenant_id` UUID NOT NULL FK tenants(id), `name` VARCHAR(255) NOT NULL, `is_default` BOOLEAN NOT NULL DEFAULT false, `deleted_at` TIMESTAMPTZ, `created_at` TIMESTAMPTZ NOT NULL DEFAULT now(), `updated_at` TIMESTAMPTZ NOT NULL DEFAULT now(), UNIQUE constraint pada `(tenant_id, name) WHERE deleted_at IS NULL`, RLS policy `expense_categories_tenant_isolation`
    - Buat file `.down.sql` — drop policy, constraint, dan tabel
    - _Requirements: 6.5, 6.6, 6.9_

  - [x] 1.2 Buat migration: tabel expenses
    - Buat `services/billing-api/migrations/XXXXXX_create_expenses.up.sql` — tabel `expenses` dengan kolom: `id` UUID PK, `tenant_id` UUID NOT NULL FK tenants(id), `category_id` UUID NOT NULL FK expense_categories(id), `amount` BIGINT NOT NULL CHECK (amount > 0), `description` TEXT NOT NULL DEFAULT '', `expense_date` DATE NOT NULL, `is_recurring` BOOLEAN NOT NULL DEFAULT false, `recurring_day` INTEGER CHECK (recurring_day >= 1 AND recurring_day <= 28), `created_by_id` UUID NOT NULL FK users(id), `deleted_at` TIMESTAMPTZ, `created_at` TIMESTAMPTZ NOT NULL DEFAULT now(), `updated_at` TIMESTAMPTZ NOT NULL DEFAULT now(), index `idx_expenses_tenant_period` pada `(tenant_id, expense_date) WHERE deleted_at IS NULL`, RLS policy `expenses_tenant_isolation`
    - Buat file `.down.sql`
    - _Requirements: 6.1, 6.2, 6.3, 6.4, 6.10_

  - [x] 1.3 Buat migration: tabel kpi_targets
    - Buat `services/billing-api/migrations/XXXXXX_create_kpi_targets.up.sql` — tabel `kpi_targets` dengan kolom: `id` UUID PK, `tenant_id` UUID NOT NULL UNIQUE FK tenants(id), `monthly_revenue_target` BIGINT, `collection_rate_target` NUMERIC(5,2), `max_receivables` BIGINT, `new_customers_monthly_target` INTEGER, `max_churn_rate` NUMERIC(5,2), `total_customers_target` INTEGER, `sla_uptime_target` NUMERIC(5,2), `max_active_alarms` INTEGER, `min_signal_quality_percentage` NUMERIC(5,2), `created_at` TIMESTAMPTZ NOT NULL DEFAULT now(), `updated_at` TIMESTAMPTZ NOT NULL DEFAULT now(), RLS policy `kpi_targets_tenant_isolation`
    - Buat file `.down.sql`
    - _Requirements: 20.1, 20.2_

  - [x] 1.4 Buat migration: tabel report_schedules
    - Buat `services/billing-api/migrations/XXXXXX_create_report_schedules.up.sql` — tabel `report_schedules` dengan kolom: `id` UUID PK, `tenant_id` UUID NOT NULL FK tenants(id), `report_type` VARCHAR(50) NOT NULL, `schedule_type` VARCHAR(20) NOT NULL CHECK IN ('daily','weekly','monthly'), `format` VARCHAR(10) NOT NULL CHECK IN ('pdf','xlsx'), `recipients` JSONB NOT NULL DEFAULT '[]', `filters` JSONB NOT NULL DEFAULT '{}', `is_active` BOOLEAN NOT NULL DEFAULT true, `created_by_id` UUID NOT NULL FK users(id), `created_at` TIMESTAMPTZ, `updated_at` TIMESTAMPTZ, index `idx_report_schedules_tenant` pada `(tenant_id) WHERE is_active = true`, RLS policy
    - Buat file `.down.sql`
    - _Requirements: 19.1, 19.2, 19.3, 19.4_

  - [x] 1.5 Buat migration: tabel report_jobs
    - Buat `services/billing-api/migrations/XXXXXX_create_report_jobs.up.sql` — tabel `report_jobs` dengan kolom: `id` UUID PK, `tenant_id` UUID NOT NULL FK tenants(id), `report_type` VARCHAR(50) NOT NULL, `format` VARCHAR(10) NOT NULL, `filters` JSONB NOT NULL DEFAULT '{}', `status` VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK IN ('pending','processing','completed','failed'), `download_url` TEXT, `error` TEXT, `requested_by` UUID NOT NULL FK users(id), `created_at` TIMESTAMPTZ, `updated_at` TIMESTAMPTZ, index `idx_report_jobs_tenant_status` pada `(tenant_id, status)`, RLS policy
    - Buat file `.down.sql`
    - _Requirements: 18.1, 18.6_

  - [x] 1.6 Buat migration: tabel custom_report_templates
    - Buat `services/billing-api/migrations/XXXXXX_create_custom_report_templates.up.sql` — tabel `custom_report_templates` dengan kolom: `id` UUID PK, `tenant_id` UUID NOT NULL FK tenants(id), `name` VARCHAR(255) NOT NULL, `metrics` JSONB NOT NULL DEFAULT '[]', `group_by` VARCHAR(50) NOT NULL, `sub_group_by` VARCHAR(50), `display_type` VARCHAR(20) NOT NULL CHECK IN ('table','bar_chart','line_chart','pie_chart'), `default_period_range` VARCHAR(20), `created_by_id` UUID NOT NULL FK users(id), `created_at` TIMESTAMPTZ, `updated_at` TIMESTAMPTZ, index `idx_custom_report_templates_tenant`, RLS policy
    - Buat file `.down.sql`
    - _Requirements: 23.2, 23.3, 23.4_


- [x] 2. Domain entities — Report DTOs, Types, dan Errors
  - [x] 2.1 Buat domain/report.go dengan semua Report DTOs dan types
    - Buat `services/billing-api/internal/domain/report.go` dengan: `ReportFilter` struct, `RevenueSource`, `RevenueDelta`, `MonthlyRevenueTrend`, `RevenueReport`, `AgingBucket`, `TopDebtor`, `ReceivablesTrend`, `AgingReport`, `PaymentMethodBreakdown`, `DailyPayment`, `PaymentReport`, `VoucherByPackage`, `VoucherByReseller`, `VoucherRevenueReport`, `ProfitLossLineItem`, `ProfitLossReport`, `CustomerGrowthReport`, `MonthlyGrowthTrend`, `DistributionItem`, `CustomerDistributionReport`, `ChurnByReason`, `ChurnAnalysisReport`, `AreaRevenue`, `RevenueByAreaReport`, `ComparisonType` constants (MoM, YoY, QoQ, Custom), `ComparisonMetric`, `ComparisonReport`, `ForecastMonth`, `ForecastReport`, `DashboardData`
    - Semua struct sesuai desain dokumen, komentar dalam Bahasa Indonesia
    - _Requirements: 1.1, 1.2, 2.1, 3.1, 4.1, 5.1, 7.1, 8.1, 9.1, 10.1, 22.1, 24.1_

  - [x] 2.2 Buat domain/report_network.go dengan Network Report DTOs
    - Buat `services/billing-api/internal/domain/report_network.go` dengan: `RouterUptimeItem`, `DowntimeEvent`, `UptimeReport`, `TrafficReport`, `RouterTraffic`, `CustomerTraffic`, `SignalQualityReport`, `DegradingONT`, `AlarmTypeSummary`, `CapacityReport`, `RouterCapacity`, `ODPCapacity`
    - _Requirements: 11.1, 12.1, 13.1, 14.1_

  - [x] 2.3 Buat domain/report_operational.go dengan Operational Report DTOs
    - Buat `services/billing-api/internal/domain/report_operational.go` dengan: `UserActivity`, `ActionSummary`, `ActivityReport`, `NotificationReport`, `ChannelStats`, `TemplateStats`, `SyncReport`, `RouterSyncStatus`, `OLTSyncStatus`
    - _Requirements: 15.1, 16.1, 17.1_

  - [x] 2.4 Buat domain/expense.go dengan Expense entity, ExpenseCategory, dan errors
    - Buat `services/billing-api/internal/domain/expense.go` dengan: `Expense` struct, `ExpenseCategory` struct, `DefaultExpenseCategories` slice (7 kategori default), `KPITarget` struct, `ScheduleType` constants, `ReportSchedule` struct, `Recipient` struct, `ReportJobStatus` constants, `ReportJob` struct, `CustomReportTemplate` struct, domain errors (`ErrExpenseNotFound`, `ErrExpenseCategoryNotFound`, `ErrCategoryHasExpenses`, `ErrCategoryNameDuplicate`, `ErrReportScheduleNotFound`, `ErrReportJobNotFound`, `ErrTemplateNotFound`, `ErrKPITargetNotFound`, `ErrInsufficientData`, `ErrInvalidReportType`, `ErrInvalidExportFormat`, `ErrMaxMetricsExceeded`)
    - _Requirements: 6.1, 6.5, 6.8, 6.9, 18.1, 19.1, 20.1, 23.2_

  - [x] 2.5 Buat domain/expense_dto.go dengan Expense request/response DTOs
    - Buat `services/billing-api/internal/domain/expense_dto.go` dengan: `CreateExpenseRequest`, `UpdateExpenseRequest`, `CreateScheduleRequest`, `UpdateScheduleRequest`, `UpdateKPITargetRequest`, `CreateTemplateRequest`, `ExportRequest`
    - Validasi menggunakan go-playground/validator tags
    - _Requirements: 6.1, 6.3, 6.6, 6.7, 18.1, 19.1, 20.2, 23.1_

  - [x] 2.6 Buat domain/forecast.go dengan LinearRegression pure function dan helpers
    - Buat `services/billing-api/internal/domain/forecast.go` dengan: `DataPoint` struct, `LinearRegressionResult` struct, `LinearRegression(points []DataPoint) LinearRegressionResult` pure function, `Predict(result LinearRegressionResult, x float64) float64`, `CalculateComparisonDelta(baseValue, compareValue float64) (deltaAbs, deltaPct float64, trend string)`, `GenerateInsights(metrics []ComparisonMetric) []string`, `formatPercentage(pct float64) string` helper
    - Invarian LinearRegression: Predict(result, x) == Slope*x + Intercept; jika semua Y sama → slope == 0; jika 2 titik → R² == 1.0
    - _Requirements: 21.2, 22.5, 22.6_

- [x] 3. Domain entities — Repository interfaces
  - [x] 3.1 Tambahkan repository interfaces ke domain/repository.go
    - Append ke `services/billing-api/internal/domain/repository.go`: `ExpenseRepository` interface (Create, GetByID, Update, SoftDelete, List, ListRecurring, SumByCategory), `ExpenseCategoryRepository` interface (Create, GetByID, Update, SoftDelete, List, NameExists, ExpenseCount, CreateDefaults), `KPITargetRepository` interface (GetByTenant, Upsert), `ReportScheduleRepository` interface (Create, GetByID, Update, Deactivate, ListByTenant, ListDue), `ReportJobRepository` interface (Create, GetByID, UpdateStatus, CleanupOld), `CustomReportTemplateRepository` interface (Create, GetByID, Delete, ListByTenant), `ReportAggregationRepository` interface (GetRevenueSummary, GetMonthlyRevenueTrend, GetAgingReport, GetPaymentDistribution, GetVoucherRevenue, GetRevenueByArea, GetCustomerGrowth, GetMonthlyGrowthTrend, GetCustomerDistribution, GetChurnAnalysis, GetAdminActivity, GetDashboardData, GetCustomReportData, GetMonthlyRevenueHistory, GetMonthlyCustomerHistory, GetMonthlyReceivablesHistory), `NetworkServiceClient` interface (GetUptimeReport, GetTrafficReport, GetSignalQualityReport, GetCapacityReport, GetSyncReport, GetNotificationReport)
    - _Requirements: 6.1, 6.5, 11.1, 12.1, 13.1, 14.1, 15.1, 16.1, 17.1, 18.1, 19.1, 20.1, 23.1, 24.1_

  - [x] 3.2 Tambahkan usecase interfaces ke domain/repository.go
    - Append ke `services/billing-api/internal/domain/repository.go`: `ReportUsecase` interface, `ExpenseUsecase` interface, `ScheduleUsecase` interface, `KPITargetUsecase` interface, `CustomReportTemplateUsecase` interface — sesuai desain dokumen
    - _Requirements: 1.1, 5.1, 6.1, 7.1, 18.1, 19.1, 20.1, 21.1, 22.1, 23.1, 24.1_


- [x] 4. Property tests untuk pure domain functions
  - [x] 4.1 Tulis property test: Sum invariant aging bucket (Property 1)
    - **Property 1: Invariant jumlah aging bucket sama dengan total outstanding**
    - Di `services/billing-api/internal/domain/report_test.go`, gunakan `rapid.Check` untuk verifikasi bahwa untuk set invoice outstanding yang dikelompokkan ke aging buckets (1-7, 8-14, 15-30, 30+), sum `total_amount` semua bucket == `total_outstanding`. Berlaku juga untuk revenue breakdown (sum sources == total) dan area revenue (sum area == total row).
    - **Validates: Requirements 2.7, 1.2, 10.3**

  - [x] 4.2 Tulis property test: Kalkulasi delta perbandingan (Property 2)
    - **Property 2: Kalkulasi delta perbandingan periode**
    - Di `services/billing-api/internal/domain/forecast_test.go`, gunakan `rapid.Check` untuk verifikasi bahwa `CalculateComparisonDelta(base, compare)` menghasilkan: `delta_absolute == base - compare`, `delta_percentage == (delta_absolute / |compare|) * 100` (atau 0 jika compare == 0), `trend == "stable"` jika |pct| < 1, `"improving"` jika pct > 0, `"declining"` jika pct < 0.
    - **Validates: Requirements 1.3, 5.5, 7.7, 22.5**

  - [x] 4.3 Tulis property test: Klasifikasi aging bucket (Property 4)
    - **Property 4: Klasifikasi aging bucket berdasarkan umur tunggakan**
    - Di `services/billing-api/internal/domain/report_test.go`, gunakan `rapid.Check` untuk verifikasi bahwa invoice dengan overdue days tertentu masuk ke bucket yang benar: 1-7 → bucket "1-7 hari", 8-14 → "8-14 hari", 15-30 → "15-30 hari", >30 → "30+ hari".
    - **Validates: Requirements 2.2**

  - [x] 4.4 Tulis property test: Distribusi persentase berjumlah 100% (Property 5)
    - **Property 5: Distribusi persentase selalu berjumlah 100%**
    - Di `services/billing-api/internal/domain/report_test.go`, gunakan `rapid.Check` untuk verifikasi bahwa untuk set items yang didistribusikan, sum semua `percentage` mendekati 100% (toleransi ±0.1%). Setiap `percentage == item_amount / total_amount * 100`.
    - **Validates: Requirements 3.2, 4.2, 8.2, 8.3, 8.5**

  - [x] 4.5 Tulis property test: Kalkulasi laba rugi dan margin (Property 6)
    - **Property 6: Kalkulasi laba rugi dan margin**
    - Di `services/billing-api/internal/domain/report_test.go`, gunakan `rapid.Check` untuk verifikasi bahwa `net_profit == total_revenue - total_expenses` dan `profit_margin == net_profit / total_revenue * 100` (atau 0 jika revenue == 0).
    - **Validates: Requirements 5.4, 4.4**

  - [x] 4.6 Tulis property test: Metrik pelanggan (Property 7)
    - **Property 7: Kalkulasi metrik pelanggan (net growth, ARPU, CLV, churn rate)**
    - Di `services/billing-api/internal/domain/report_test.go`, gunakan `rapid.Check` untuk verifikasi: `net_growth == new - churned`, `arpu == revenue / avg_active` (atau 0), `clv == arpu * avg_lifetime`, `churn_rate == churned / total_start * 100` (atau 0).
    - **Validates: Requirements 7.2, 7.4, 7.5, 7.6**

  - [x] 4.7 Tulis property test: Linear regression prediksi (Property 8)
    - **Property 8: Linear regression — prediksi pada titik data**
    - Di `services/billing-api/internal/domain/forecast_test.go`, gunakan `rapid.Check` untuk verifikasi: `Predict(result, x) == Slope*x + Intercept` untuk semua x. Jika semua Y sama → slope == 0. Jika 2 titik → R² == 1.0. Prediksi pada mean(X) mendekati mean(Y).
    - **Validates: Requirements 21.2**

  - [x] 4.8 Tulis property test: KPI progress dan status label (Property 9)
    - **Property 9: KPI progress dan status label**
    - Di `services/billing-api/internal/domain/report_test.go`, gunakan `rapid.Check` untuk verifikasi: `progress == current / target * 100` (atau 0 jika target == 0). Status: "tercapai" jika >= 100%, "hampir" jika >= 80%, "di_bawah_target" jika < 80%.
    - **Validates: Requirements 20.4**

  - [x] 4.9 Tulis property test: Insight generation (Property 10)
    - **Property 10: Insight generation berdasarkan delta terbesar**
    - Di `services/billing-api/internal/domain/forecast_test.go`, gunakan `rapid.Check` untuk verifikasi: `GenerateInsights` mengembalikan 3-5 insight diurutkan berdasarkan |delta_percentage| terbesar. Setiap insight mengandung nama metrik dan arah (naik/turun). Metrik dengan |delta| < 1% tidak menghasilkan insight.
    - **Validates: Requirements 22.6**

  - [x] 4.10 Tulis property test: Ordering top debtors dan peak payment (Property 11)
    - **Property 11: Ordering dan limiting pada top debtors dan peak payment**
    - Di `services/billing-api/internal/domain/report_test.go`, gunakan `rapid.Check` untuk verifikasi: `top_debtors` diurutkan berdasarkan `total_outstanding` descending, maksimal 10 item. `peak_payment_date` adalah tanggal dengan `total_amount` tertinggi.
    - **Validates: Requirements 2.5, 3.4**

- [x] 5. Checkpoint — Domain layer selesai
  - Pastikan semua file domain compile (`go build ./...` di `services/billing-api`). Pastikan property tests pass. Tanyakan ke user jika ada pertanyaan.


- [x] 6. sqlc queries — CRUD untuk tabel baru
  - [x] 6.1 Buat queries/expense_categories.sql
    - Buat `services/billing-api/queries/expense_categories.sql` dengan sqlc queries: `CreateExpenseCategory` (:one), `GetExpenseCategoryByID` (:one), `UpdateExpenseCategory` (:one), `SoftDeleteExpenseCategory` (:exec), `ListExpenseCategories` (:many, WHERE tenant_id AND deleted_at IS NULL), `ExpenseCategoryNameExists` (:one, SELECT EXISTS), `ExpenseCategoryExpenseCount` (:one, COUNT expenses WHERE category_id AND deleted_at IS NULL), `CreateDefaultExpenseCategories` (:exec, INSERT multiple rows)
    - _Requirements: 6.5, 6.6, 6.7, 6.8, 6.9_

  - [x] 6.2 Buat queries/expenses.sql
    - Buat `services/billing-api/queries/expenses.sql` dengan sqlc queries: `CreateExpense` (:one), `GetExpenseByID` (:one, JOIN expense_categories untuk category_name), `UpdateExpense` (:one), `SoftDeleteExpense` (:exec), `ListExpenses` (:many, WHERE tenant_id AND expense_date BETWEEN AND optional category_id, ORDER BY expense_date DESC), `ListRecurringExpenses` (:many, WHERE is_recurring = true AND deleted_at IS NULL), `SumExpensesByCategory` (:many, GROUP BY category untuk profit-loss)
    - _Requirements: 6.1, 6.2, 6.3, 6.4, 6.10_

  - [x] 6.3 Buat queries/kpi_targets.sql
    - Buat `services/billing-api/queries/kpi_targets.sql` dengan sqlc queries: `GetKPITargetByTenant` (:one), `UpsertKPITarget` (:one, INSERT ON CONFLICT DO UPDATE)
    - _Requirements: 20.1, 20.2_

  - [x] 6.4 Buat queries/report_schedules.sql
    - Buat `services/billing-api/queries/report_schedules.sql` dengan sqlc queries: `CreateReportSchedule` (:one), `GetReportScheduleByID` (:one), `UpdateReportSchedule` (:one), `DeactivateReportSchedule` (:exec), `ListReportSchedulesByTenant` (:many), `ListDueSchedules` (:many, WHERE schedule_type AND is_active)
    - _Requirements: 19.1, 19.2, 19.3, 19.4_

  - [x] 6.5 Buat queries/report_jobs.sql
    - Buat `services/billing-api/queries/report_jobs.sql` dengan sqlc queries: `CreateReportJob` (:one), `GetReportJobByID` (:one), `UpdateReportJobStatus` (:exec), `CleanupOldReportJobs` (:exec, DELETE WHERE created_at < $1)
    - _Requirements: 18.1, 18.6, 19.9_

  - [x] 6.6 Buat queries/custom_report_templates.sql
    - Buat `services/billing-api/queries/custom_report_templates.sql` dengan sqlc queries: `CreateCustomReportTemplate` (:one), `GetCustomReportTemplateByID` (:one), `DeleteCustomReportTemplate` (:exec), `ListCustomReportTemplatesByTenant` (:many)
    - _Requirements: 23.2, 23.3, 23.4_

  - [x] 6.7 Jalankan sqlc generate
    - Jalankan `sqlc generate` di `services/billing-api/` untuk regenerate Go code
    - Verifikasi generated code compile
    - _Requirements: 6.1, 20.1, 19.1, 18.1, 23.1_

- [x] 7. sqlc queries — Aggregasi kompleks untuk laporan
  - [x] 7.1 Buat queries/report_aggregation.sql — Financial aggregations
    - Buat `services/billing-api/queries/report_aggregation.sql` dengan sqlc queries: `GetRevenueSummary` (:one, SUM invoice_payments grouped by item_type + voucher sales, filtered by period/area/package), `GetMonthlyRevenueTrend` (:many, GROUP BY month untuk 12 bulan terakhir), `GetAgingBuckets` (:many, CASE WHEN untuk aging buckets 1-7, 8-14, 15-30, 30+), `GetCollectionRate` (:one, paid vs total due), `GetAvgDaysToPay` (:one, AVG days between due_date dan payment_date), `GetTopDebtors` (:many, ORDER BY outstanding DESC LIMIT 10), `GetReceivablesTrend` (:many, GROUP BY month 6 bulan), `GetPaymentDistribution` (:many, GROUP BY payment_method), `GetDailyPayments` (:many, GROUP BY date), `GetVoucherRevenueByPackage` (:many), `GetVoucherRevenueByReseller` (:many), `GetRevenueByArea` (:many, JOIN areas)
    - _Requirements: 1.1, 1.2, 1.4, 2.1, 2.2, 2.3, 2.4, 2.5, 2.6, 3.1, 3.2, 3.3, 3.4, 4.1, 4.2, 4.3, 10.1, 10.2_

  - [x] 7.2 Buat queries/report_aggregation_customer.sql — Customer aggregations
    - Buat `services/billing-api/queries/report_aggregation_customer.sql` dengan sqlc queries: `GetCustomerGrowthData` (:one, total_active, new, churned, net_growth), `GetMonthlyGrowthTrend` (:many, GROUP BY month 12 bulan), `GetCustomerDistributionByPackage` (:many), `GetCustomerDistributionByArea` (:many), `GetCustomerDistributionByStatus` (:many), `GetCustomerDistributionByConnectionMethod` (:many), `GetChurnAnalysis` (:one, churned count + rate), `GetChurnByReason` (:many), `GetChurnByPackage` (:many), `GetChurnByArea` (:many), `GetAvgCustomerLifetime` (:one), `GetARPU` (:one), `GetCLV` (:one)
    - _Requirements: 7.1, 7.2, 7.3, 7.4, 7.5, 7.6, 8.1, 8.2, 8.3, 8.4, 8.5, 9.1, 9.2, 9.3, 9.4, 9.5, 9.6_

  - [x] 7.3 Buat queries/report_aggregation_operational.sql — Operational aggregations
    - Buat `services/billing-api/queries/report_aggregation_operational.sql` dengan sqlc queries: `GetAdminActivity` (:many, GROUP BY user dari audit_logs), `GetTopActions` (:many, GROUP BY action_type), `GetDashboardData` (:one, aggregasi metrik kunci untuk dashboard widget)
    - _Requirements: 15.1, 15.2, 15.3, 24.1, 24.2_

  - [x] 7.4 Buat queries/report_aggregation_forecast.sql — Forecast data
    - Buat `services/billing-api/queries/report_aggregation_forecast.sql` dengan sqlc queries: `GetMonthlyRevenueHistory` (:many, 6 bulan terakhir untuk linear regression), `GetMonthlyCustomerHistory` (:many), `GetMonthlyReceivablesHistory` (:many)
    - _Requirements: 21.1, 21.2_

  - [x] 7.5 Jalankan sqlc generate untuk aggregation queries
    - Jalankan `sqlc generate` di `services/billing-api/`
    - Verifikasi generated code compile
    - _Requirements: 1.1, 7.1, 15.1, 21.1_


- [x] 8. Repository implementations — CRUD repos
  - [x] 8.1 Buat repository/expense_repo.go
    - Buat `services/billing-api/internal/repository/expense_repo.go` implementing `domain.ExpenseRepository` — wraps sqlc-generated queries, pattern sama dengan existing repos (NewExpenseRepo constructor, pool *pgxpool.Pool)
    - _Requirements: 6.1, 6.2, 6.3, 6.4, 6.10_

  - [x] 8.2 Buat repository/expense_category_repo.go
    - Buat `services/billing-api/internal/repository/expense_category_repo.go` implementing `domain.ExpenseCategoryRepository` — termasuk `CreateDefaults` yang insert 7 kategori default
    - _Requirements: 6.5, 6.6, 6.7, 6.8, 6.9_

  - [x] 8.3 Buat repository/kpi_target_repo.go
    - Buat `services/billing-api/internal/repository/kpi_target_repo.go` implementing `domain.KPITargetRepository` — Upsert menggunakan INSERT ON CONFLICT
    - _Requirements: 20.1, 20.2_

  - [x] 8.4 Buat repository/report_schedule_repo.go
    - Buat `services/billing-api/internal/repository/report_schedule_repo.go` implementing `domain.ReportScheduleRepository`
    - _Requirements: 19.1, 19.2, 19.3, 19.4_

  - [x] 8.5 Buat repository/report_job_repo.go
    - Buat `services/billing-api/internal/repository/report_job_repo.go` implementing `domain.ReportJobRepository`
    - _Requirements: 18.1, 18.6_

  - [x] 8.6 Buat repository/custom_report_template_repo.go
    - Buat `services/billing-api/internal/repository/custom_report_template_repo.go` implementing `domain.CustomReportTemplateRepository`
    - _Requirements: 23.2, 23.3, 23.4_

  - [x] 8.7 Buat repository/report_aggregation_repo.go
    - Buat `services/billing-api/internal/repository/report_aggregation_repo.go` implementing `domain.ReportAggregationRepository` — wraps semua aggregation sqlc queries, assembles DTOs dari raw query results. File ini bisa dipecah ke beberapa file jika melebihi 200 baris (report_aggregation_financial.go, report_aggregation_customer.go, dll)
    - _Requirements: 1.1, 2.1, 3.1, 4.1, 7.1, 8.1, 9.1, 10.1, 15.1, 21.1, 24.1_

- [x] 9. Checkpoint — Data layer selesai
  - Pastikan semua repository files compile (`go build ./...` di `services/billing-api`). Tanyakan ke user jika ada pertanyaan.

- [x] 10. NetworkServiceClient — HTTP client untuk cross-service calls
  - [x] 10.1 Buat usecase/network_client.go implementing NetworkServiceClient
    - Buat `services/billing-api/internal/usecase/network_client.go` dengan: `NetworkClient` struct (baseURL string, httpClient *http.Client, redisClient *redis.Client, logger zerolog.Logger), constructor `NewNetworkClient`, implementasi semua methods dari `domain.NetworkServiceClient` interface
    - Setiap method: HTTP GET ke `{baseURL}/internal/v1/reports/{type}?tenant_id=...&period_start=...&period_end=...` → parse JSON response → return DTO
    - Graceful degradation: jika HTTP call gagal → cek Redis cache (key: `report:network:{type}:{tenant_id}:{filter_hash}`, TTL 1 jam) → jika cache ada → return cached data dengan `stale_data=true` dan `last_updated` → jika tidak ada cache → return empty response dengan `module_inactive=true`
    - Timeout: 10 detik per request
    - _Requirements: 11.1, 11.5, 12.1, 12.5, 13.1, 13.6, 14.1, 14.5, 16.1, 16.5, 17.1, 17.5, 25.1, 25.2, 25.3, 25.4_


- [x] 11. Usecase layer — Report usecases
  - [x] 11.1 Buat usecase/report_manager.go — ReportManager struct dan financial reports
    - Buat `services/billing-api/internal/usecase/report_manager.go` dengan: `ReportManager` struct (aggregationRepo, expenseRepo, kpiTargetRepo, networkClient, redisClient, logger), constructor `NewReportManager`
    - Methods: `GetRevenueReport` (query aggregation + optional comparison + KPI target), `GetAgingReport` (query aging buckets + collection rate + top debtors + trend + KPI), `GetPaymentReport` (query payment distribution + daily + peak), `GetVoucherRevenueReport` (query voucher by package + reseller), `GetProfitLossReport` (revenue dari aggregation + expenses dari expenseRepo.SumByCategory + optional comparison), `GetRevenueByAreaReport` (query per area + identify most profitable + attention needed)
    - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5, 1.6, 2.1, 2.2, 2.3, 2.4, 2.5, 2.6, 2.7, 3.1, 3.2, 3.3, 3.4, 4.1, 4.2, 4.3, 4.4, 5.1, 5.2, 5.3, 5.4, 5.5, 10.1, 10.2, 10.3, 10.4, 10.5_

  - [x] 11.2 Buat usecase/report_customer.go — Customer report methods
    - Buat `services/billing-api/internal/usecase/report_customer.go` dengan methods pada ReportManager: `GetCustomerGrowthReport` (query growth + trend + ARPU + CLV + churn rate + optional comparison + KPI), `GetCustomerDistributionReport` (query distribution by package/area/status/connection), `GetChurnAnalysisReport` (query churn by reason/package/area + avg lifetime)
    - _Requirements: 7.1, 7.2, 7.3, 7.4, 7.5, 7.6, 7.7, 8.1, 8.2, 8.3, 8.4, 8.5, 9.1, 9.2, 9.3, 9.4, 9.5, 9.6_

  - [x] 11.3 Buat usecase/report_network.go — Network report methods (delegasi ke NetworkServiceClient)
    - Buat `services/billing-api/internal/usecase/report_network.go` dengan methods pada ReportManager: `GetUptimeReport`, `GetTrafficReport`, `GetSignalQualityReport`, `GetCapacityReport` — semua delegasi ke networkClient dengan graceful degradation (module_inactive, stale_data)
    - Tambahkan SLA target dari KPI targets ke uptime report
    - _Requirements: 11.1, 11.2, 11.3, 11.4, 11.5, 12.1, 12.2, 12.3, 12.4, 12.5, 13.1, 13.2, 13.3, 13.4, 13.5, 13.6, 14.1, 14.2, 14.3, 14.4, 14.5_

  - [x] 11.4 Buat usecase/report_operational.go — Operational report methods
    - Buat `services/billing-api/internal/usecase/report_operational.go` dengan methods pada ReportManager: `GetActivityReport` (query admin activity + top actions), `GetNotificationReport` (delegasi ke networkClient), `GetSyncReport` (delegasi ke networkClient)
    - _Requirements: 15.1, 15.2, 15.3, 16.1, 16.2, 16.3, 16.4, 16.5, 17.1, 17.2, 17.3, 17.4, 17.5_

  - [x] 11.5 Buat usecase/expense_manager.go — ExpenseManager
    - Buat `services/billing-api/internal/usecase/expense_manager.go` dengan: `ExpenseManager` struct (expenseRepo, categoryRepo, logger), constructor `NewExpenseManager`, methods: `Create`, `GetByID`, `Update`, `Delete` (soft delete), `List`, `ListCategories`, `CreateCategory` (cek name duplicate), `UpdateCategory`, `DeleteCategory` (cek has expenses → reject)
    - _Requirements: 6.1, 6.2, 6.3, 6.4, 6.5, 6.6, 6.7, 6.8_

  - [x] 11.6 Buat usecase/schedule_manager.go — ScheduleManager
    - Buat `services/billing-api/internal/usecase/schedule_manager.go` dengan: `ScheduleManager` struct (scheduleRepo, jobRepo, logger), constructor `NewScheduleManager`, methods: `Create`, `Update`, `Delete` (deactivate), `List`
    - _Requirements: 19.1, 19.2, 19.3, 19.4_

  - [x] 11.7 Buat usecase/forecast_engine.go — ForecastEngine
    - Buat `services/billing-api/internal/usecase/forecast_engine.go` dengan: `ForecastEngine` struct (aggregationRepo, kpiTargetRepo, logger), constructor `NewForecastEngine`, method: `GetForecastReport` — ambil 6 bulan data historis → jika < 3 bulan → return insufficient_data → jalankan LinearRegression untuk revenue, customers, receivables → generate 3 bulan proyeksi → jika KPI targets ada → hitung estimated_target_date → tambahkan disclaimer
    - _Requirements: 21.1, 21.2, 21.3, 21.4, 21.5, 21.6_

  - [x] 11.8 Buat usecase/comparison_engine.go — ComparisonEngine
    - Buat `services/billing-api/internal/usecase/comparison_engine.go` dengan: `ComparisonEngine` struct (aggregationRepo, kpiTargetRepo, logger), constructor `NewComparisonEngine`, method: `GetComparisonReport` — tentukan comparison period berdasarkan type (MoM: bulan sebelumnya, YoY: tahun sebelumnya, QoQ: kuartal sebelumnya, Custom: dari parameter) → query metrik untuk kedua periode → hitung delta menggunakan CalculateComparisonDelta → generate insights menggunakan GenerateInsights
    - _Requirements: 22.1, 22.2, 22.3, 22.4, 22.5, 22.6_

  - [x] 11.9 Buat usecase/custom_report_builder.go — CustomReportBuilder
    - Buat `services/billing-api/internal/usecase/custom_report_builder.go` dengan: `CustomReportBuilder` struct (aggregationRepo, templateRepo, logger), constructor `NewCustomReportBuilder`, methods: `PreviewCustomReport` (validasi max 3 metrik → query aggregation dengan dynamic grouping), `CreateTemplate`, `DeleteTemplate`, `ListTemplates`
    - _Requirements: 23.1, 23.2, 23.3, 23.4, 23.5_

  - [x] 11.10 Buat usecase/dashboard_cache.go — DashboardCache
    - Buat `services/billing-api/internal/usecase/dashboard_cache.go` dengan: `DashboardCache` struct (aggregationRepo, networkClient, kpiTargetRepo, redisClient, logger), constructor `NewDashboardCache`, method: `GetDashboardData` — cek Redis cache (key: `report:dashboard:{tenant_id}`, TTL 5 menit) → jika hit → return cached → jika miss → query aggregation + network data → assemble DashboardData → store cache → return
    - Method: `InvalidateCache(tenantID string)` — delete cache key, dipanggil saat data berubah
    - _Requirements: 24.1, 24.2, 24.3, 24.4, 24.5_

  - [x] 11.11 Buat usecase/report_export.go — Export methods pada ReportManager
    - Buat `services/billing-api/internal/usecase/report_export.go` dengan methods pada ReportManager: `RequestExport` (validasi report_type + format → jika CSV: generate synchronous → jika PDF/XLSX: create report_job + enqueue asynq task → return job_id), `GetExportStatus` (query report_job by ID)
    - _Requirements: 18.1, 18.2, 18.3, 18.6_

- [x] 12. Checkpoint — Usecase layer selesai
  - Pastikan semua usecase files compile (`go build ./...` di `services/billing-api`). Tanyakan ke user jika ada pertanyaan.


- [x] 13. HTTP handlers
  - [x] 13.1 Buat handler/report_handler.go — ReportHandler
    - Buat `services/billing-api/internal/handler/report_handler.go` dengan: `ReportHandler` struct (reportManager, logger), constructor `NewReportHandler`, methods: `Revenue` (parse filter query params → call GetRevenueReport), `Aging`, `Payments`, `Vouchers`, `ProfitLoss`, `RevenueByArea`, `RevenueByAreaCustomers`, `CustomerGrowth`, `CustomerDistribution`, `ChurnAnalysis`, `Uptime`, `Traffic`, `SignalQuality`, `Capacity`, `Activity`, `Notifications`, `Sync`
    - Setiap method: parse query params (period_start, period_end, compare_start, compare_end, area_id, package_id, router_id) → validasi → call usecase → return JSON response
    - Include `mapReportError` helper function
    - Pecah ke beberapa file jika melebihi 200 baris: `report_handler_financial.go`, `report_handler_customer.go`, `report_handler_network.go`, `report_handler_operational.go`
    - _Requirements: 1.1, 2.1, 3.1, 4.1, 5.1, 7.1, 8.1, 9.1, 10.1, 11.1, 12.1, 13.1, 14.1, 15.1, 16.1, 17.1_

  - [x] 13.2 Buat handler/expense_handler.go — ExpenseHandler
    - Buat `services/billing-api/internal/handler/expense_handler.go` dengan: `ExpenseHandler` struct (expenseManager, validate, logger), constructor `NewExpenseHandler`, methods: `List`, `Create` (parse body → validate → call usecase → return 201), `Update`, `Delete` (return 204), `ListCategories`, `CreateCategory`, `UpdateCategory`, `DeleteCategory`
    - _Requirements: 6.1, 6.2, 6.3, 6.4, 6.5, 6.6, 6.7, 6.8_

  - [x] 13.3 Buat handler/export_handler.go — ExportHandler
    - Buat `services/billing-api/internal/handler/export_handler.go` dengan: `ExportHandler` struct (reportManager, logger), constructor `NewExportHandler`, methods: `RequestExport` (parse body → validate report_type + format → call usecase → jika CSV: return file langsung → jika PDF/XLSX: return 202 dengan job_id), `Status` (parse job_id → call usecase → return job status + download_url)
    - _Requirements: 18.1, 18.2, 18.3, 18.6, 18.7_

  - [x] 13.4 Buat handler/schedule_handler.go — ScheduleHandler
    - Buat `services/billing-api/internal/handler/schedule_handler.go` dengan: `ScheduleHandler` struct (scheduleManager, validate, logger), constructor `NewScheduleHandler`, methods: `List`, `Create` (return 201), `Update`, `Delete` (return 204)
    - _Requirements: 19.1, 19.2, 19.3, 19.4_

  - [x] 13.5 Buat handler/kpi_handler.go — KPIHandler
    - Buat `services/billing-api/internal/handler/kpi_handler.go` dengan: `KPIHandler` struct (kpiTargetRepo, validate, logger), constructor `NewKPIHandler`, methods: `Get` (return current KPI targets), `Update` (parse body → validate → upsert → return updated)
    - _Requirements: 20.1, 20.2_

  - [x] 13.6 Buat handler/forecast_handler.go — ForecastHandler
    - Buat `services/billing-api/internal/handler/forecast_handler.go` dengan: `ForecastHandler` struct (forecastEngine, logger), constructor `NewForecastHandler`, method: `Forecast` (call usecase → return forecast report)
    - _Requirements: 21.1_

  - [x] 13.7 Buat handler/comparison_handler.go — ComparisonHandler
    - Buat `services/billing-api/internal/handler/comparison_handler.go` dengan: `ComparisonHandler` struct (comparisonEngine, logger), constructor `NewComparisonHandler`, method: `Compare` (parse comparison_type + periods → call usecase → return comparison report)
    - _Requirements: 22.1_

  - [x] 13.8 Buat handler/custom_report_handler.go — CustomReportHandler
    - Buat `services/billing-api/internal/handler/custom_report_handler.go` dengan: `CustomReportHandler` struct (customReportBuilder, validate, logger), constructor `NewCustomReportHandler`, methods: `Preview` (parse body → validate max 3 metrik → call usecase → return data), `ListTemplates`, `CreateTemplate` (return 201), `DeleteTemplate` (return 204)
    - _Requirements: 23.1, 23.2, 23.3, 23.4, 23.5_

  - [x] 13.9 Buat handler/dashboard_handler.go — DashboardHandler
    - Buat `services/billing-api/internal/handler/dashboard_handler.go` dengan: `DashboardHandler` struct (dashboardCache, logger), constructor `NewDashboardHandler`, method: `Dashboard` (call usecase → return dashboard data, target < 500ms)
    - _Requirements: 24.1, 24.2, 24.3_

- [x] 14. Route registration dan wiring
  - [x] 14.1 Update handler/router.go dengan report, expense, dan semua route baru
    - Modifikasi `services/billing-api/internal/handler/router.go`: tambahkan field baru ke `RouterConfig` struct: `ReportHandler`, `ExpenseHandler`, `ExportHandler`, `ScheduleHandler`, `KPIHandler`, `ForecastHandler`, `ComparisonHandler`, `CustomReportHandler`, `DashboardHandler`
    - Daftarkan semua route sesuai desain dokumen:
      - Reports read (admin + operator + kasir): GET /v1/reports/financial/*, /v1/reports/customers/*, /v1/reports/network/*, /v1/reports/operational/*, /v1/reports/comparison, /v1/reports/forecast, /v1/reports/dashboard, GET /v1/reports/export/:job_id
      - Reports admin (tenant_admin only): POST /v1/reports/export, CRUD /v1/reports/schedules, GET/PUT /v1/reports/kpi-targets, POST /v1/reports/custom/*
      - Expenses admin (tenant_admin only): CRUD /v1/expenses, CRUD /v1/expenses/categories
    - _Requirements: 1.1, 6.1, 18.1, 19.1, 20.1, 21.1, 22.1, 23.1, 24.1_

  - [x] 14.2 Update cmd/main.go dengan dependency injection untuk semua reporting components
    - Modifikasi `services/billing-api/cmd/main.go`:
      - Instantiate repos: `expenseRepo`, `expenseCategoryRepo`, `kpiTargetRepo`, `reportScheduleRepo`, `reportJobRepo`, `customReportTemplateRepo`, `reportAggregationRepo`
      - Instantiate network client: `networkClient := usecase.NewNetworkClient(cfg.NetworkServiceURL, redisClient, appLogger)`
      - Instantiate usecases: `reportManager`, `expenseManager`, `scheduleManager`, `forecastEngine`, `comparisonEngine`, `customReportBuilder`, `dashboardCache`
      - Instantiate handlers: `reportHandler`, `expenseHandler`, `exportHandler`, `scheduleHandler`, `kpiHandler`, `forecastHandler`, `comparisonHandler`, `customReportHandler`, `dashboardHandler`
      - Tambahkan semua handler ke `RouterConfig`
    - _Requirements: 1.1, 6.1, 18.1, 19.1, 20.1, 21.1, 22.1, 23.1, 24.1_

- [x] 15. Checkpoint — HTTP layer selesai
  - Pastikan full service compile (`go build ./...` di `services/billing-api`). Pastikan semua route terdaftar. Tanyakan ke user jika ada pertanyaan.


- [x] 16. Background workers
  - [x] 16.1 Buat worker/export_worker.go — ExportWorker
    - Buat `services/billing-api/internal/worker/export_worker.go` dengan: `ExportWorker` struct (reportManager, jobRepo, logger), constructor `NewExportWorker`, method `RegisterHandlers(mux *asynq.ServeMux)`, handler `HandleExportTask` — dequeue task → update job status: processing → generate report data via reportManager → generate PDF (menggunakan gofpdf dengan tenant branding: logo, ISP name, tagline di header, timestamp + ISPBoss attribution di footer) atau XLSX → simpan file → update job status: completed + download_url → jika gagal: retry 1x → jika retry gagal: mark failed + notify admin
    - Task type constant: `TaskReportExport = "report.export"`
    - _Requirements: 18.3, 18.4, 18.5, 18.7, 18.8_

  - [x] 16.2 Buat worker/schedule_worker.go — ScheduleWorker
    - Buat `services/billing-api/internal/worker/schedule_worker.go` dengan: `ScheduleWorker` struct (scheduleManager, reportManager, jobRepo, queueClient, logger), constructor `NewScheduleWorker`, method `RegisterHandlers(mux *asynq.ServeMux)`, handler `HandleScheduledReport` — query due schedules → untuk setiap schedule: generate report → kirim via configured channels (email attachment / WhatsApp download link via asynq notification task)
    - Task type constants: `TaskScheduledReport = "report.scheduled"`, `TaskCleanupReportJobs = "report.cleanup_jobs"`, `TaskCleanupScheduledFiles = "report.cleanup_files"`
    - _Requirements: 19.5, 19.6, 19.7, 19.8, 19.9_

  - [x] 16.3 Buat worker/recurring_expense_worker.go — RecurringExpenseWorker
    - Buat `services/billing-api/internal/worker/recurring_expense_worker.go` dengan: `RecurringExpenseWorker` struct (expenseRepo, logger), constructor `NewRecurringExpenseWorker`, method `RegisterHandlers(mux *asynq.ServeMux)`, handler `HandleRecurringExpense` — query expenses WHERE is_recurring = true AND recurring_day = today → untuk setiap expense: create new expense record dengan expense_date = today
    - Task type constant: `TaskRecurringExpense = "expense.recurring"`
    - _Requirements: 6.10_

  - [x] 16.4 Update cmd/main.go — Daftarkan workers dan cron jobs
    - Modifikasi `services/billing-api/cmd/main.go`:
      - Instantiate workers: `exportWorker`, `scheduleWorker`, `recurringExpenseWorker`
      - Daftarkan handlers ke asynq mux: `exportWorker.RegisterHandlers(mux)`, `scheduleWorker.RegisterHandlers(mux)`, `recurringExpenseWorker.RegisterHandlers(mux)`
      - Daftarkan cron jobs ke scheduler: daily schedule check (07:00), weekly schedule check (Senin 07:00), monthly schedule check (tanggal 1 07:00), recurring expense (00:10 setiap hari), cleanup old report jobs (03:00 setiap hari)
    - _Requirements: 18.3, 19.5, 19.6, 19.7, 6.10_

- [x] 17. Checkpoint — Backend selesai
  - Pastikan full service compile (`go build ./...` di `services/billing-api`). Pastikan semua workers terdaftar. Pastikan semua cron jobs terdaftar. Tanyakan ke user jika ada pertanyaan.


- [x] 18. Frontend — Setup dan shared components
  - [x] 18.1 Install dependencies dan buat TypeScript types
    - Install Recharts: `npm install recharts` di `apps/web/`
    - Buat `apps/web/app/reports/lib/types.ts` — TypeScript types matching semua backend DTOs: `ReportFilter`, `RevenueReport`, `AgingReport`, `PaymentReport`, `VoucherRevenueReport`, `ProfitLossReport`, `CustomerGrowthReport`, `CustomerDistributionReport`, `ChurnAnalysisReport`, `RevenueByAreaReport`, `UptimeReport`, `TrafficReport`, `SignalQualityReport`, `CapacityReport`, `ActivityReport`, `NotificationReport`, `SyncReport`, `ComparisonReport`, `ForecastReport`, `DashboardData`, `Expense`, `ExpenseCategory`, `KPITarget`, `ReportSchedule`, `ReportJob`, `CustomReportTemplate`
    - _Requirements: 26.1, 27.1, 28.1, 29.1, 30.1_

  - [x] 18.2 Buat API client functions
    - Buat `apps/web/app/reports/lib/api.ts` — fungsi fetch untuk semua report endpoints: `fetchRevenueReport`, `fetchAgingReport`, `fetchPaymentReport`, `fetchVoucherReport`, `fetchProfitLossReport`, `fetchRevenueByAreaReport`, `fetchCustomerGrowthReport`, `fetchDistributionReport`, `fetchChurnReport`, `fetchUptimeReport`, `fetchTrafficReport`, `fetchSignalReport`, `fetchCapacityReport`, `fetchActivityReport`, `fetchNotificationReport`, `fetchSyncReport`, `fetchComparisonReport`, `fetchForecastReport`, `fetchDashboardData`, `fetchExpenses`, `createExpense`, `updateExpense`, `deleteExpense`, `fetchCategories`, `createCategory`, `updateCategory`, `deleteCategory`, `requestExport`, `getExportStatus`, `fetchSchedules`, `createSchedule`, `updateSchedule`, `deleteSchedule`, `fetchKPITargets`, `updateKPITargets`, `previewCustomReport`, `fetchTemplates`, `createTemplate`, `deleteTemplate`
    - _Requirements: 26.1, 27.1, 34.1_

  - [x] 18.3 Buat formatters dan utility functions
    - Buat `apps/web/app/reports/lib/formatters.ts` — helper functions: `formatCurrency(amount: number)` (format Rupiah: Rp 1.234.567), `formatPercentage(value: number)`, `formatNumber(value: number)`, `formatDate(date: string)`, `formatMonth(month: string)`, `formatBytes(bytes: number)` (KB/MB/GB), `formatDelta(delta: number)` (dengan +/- prefix), `getDeltaColor(delta: number)` (green/red/gray)
    - _Requirements: 27.2, 28.2, 31.4_

  - [x] 18.4 Buat shared UI components
    - Buat `apps/web/app/reports/components/shared/MetricCard.tsx` — kartu ringkasan dengan: label, value, delta percentage, delta color (green/red/gray), optional KPI progress bar
    - Buat `apps/web/app/reports/components/shared/EmptyState.tsx` — pesan "Belum ada data untuk periode ini"
    - Buat `apps/web/app/reports/components/shared/StaleDataBanner.tsx` — banner kuning "Data terakhir diperbarui pada {timestamp}"
    - Buat `apps/web/app/reports/components/shared/ModuleInactive.tsx` — pesan "Modul {nama} belum aktif. Aktifkan modul untuk melihat laporan ini."
    - Buat `apps/web/app/reports/components/shared/ProgressBar.tsx` — KPI progress bar dengan color coding: hijau (≥100%), kuning (≥80%), merah (<80%)
    - _Requirements: 25.5, 25.4, 25.1, 25.2, 25.3, 27.6_

  - [x] 18.5 Buat reusable chart components
    - Buat `apps/web/app/reports/components/charts/BarChart.tsx` — wrapper Recharts BarChart dengan responsive container, tooltip, legend
    - Buat `apps/web/app/reports/components/charts/LineChart.tsx` — wrapper Recharts LineChart
    - Buat `apps/web/app/reports/components/charts/PieChart.tsx` — wrapper Recharts PieChart/donut
    - Buat `apps/web/app/reports/components/charts/AreaChart.tsx` — wrapper Recharts AreaChart
    - Semua chart: responsive, horizontally scrollable on mobile, Tailwind styling
    - _Requirements: 27.2, 27.3, 28.2, 28.3, 29.2, 29.3_

- [x] 19. Frontend — Custom hooks
  - [x] 19.1 Buat custom hooks untuk report data fetching dan filter state
    - Buat `apps/web/app/reports/hooks/useReportData.ts` — generic hook untuk fetch report data: loading state, error handling, refetch, caching
    - Buat `apps/web/app/reports/hooks/useFilters.ts` — hook untuk filter state management: sync dengan URL query params (periode, area, paket, router, comparison type), apply/reset functions, persist tab selection
    - Buat `apps/web/app/reports/hooks/useDashboard.ts` — hook untuk dashboard widget data: auto-refresh setiap 5 menit, loading state
    - _Requirements: 26.4, 26.5, 35.3_


- [x] 20. Frontend — Halaman laporan utama (ReportPage)
  - [x] 20.1 Buat ReportPage, FilterBar, dan TabNavigation
    - Buat `apps/web/app/reports/page.tsx` — halaman utama laporan di route `/reports`
    - Buat `apps/web/app/reports/components/FilterBar.tsx` — filter global: Periode (dropdown: Hari ini, Minggu ini, Bulan ini, Kuartal, Tahun, Custom range), Bandingkan (dropdown: MoM, YoY, QoQ, Custom), Area (dropdown), Paket (dropdown), Terapkan/Reset buttons. Collapsible on mobile.
    - Buat `apps/web/app/reports/components/TabNavigation.tsx` — 4 tab kategori: Keuangan (💰), Pelanggan (👥), Jaringan (📡), Operasional (⚙️). Swipeable horizontal on mobile. Router filter muncul saat tab Jaringan aktif.
    - Tab dan filter state di-persist ke URL query params
    - _Requirements: 26.1, 26.2, 26.3, 26.4, 26.5, 26.7_

  - [x] 20.2 Buat section components — Keuangan tab
    - Buat `apps/web/app/reports/components/financial/RevenueSection.tsx` — 4 summary cards (Total, Bulanan, Voucher, Lainnya) + delta + 12-month stacked bar chart
    - Buat `apps/web/app/reports/components/financial/AgingSection.tsx` — 4 aging bucket cards + collection rate progress bar + avg days to pay + top 10 debtors table + 6-month trend line chart
    - Buat `apps/web/app/reports/components/financial/PaymentSection.tsx` — payment method distribution cards + daily payment bar chart
    - Buat `apps/web/app/reports/components/financial/VoucherSection.tsx` — voucher revenue by package + by reseller tables
    - Buat `apps/web/app/reports/components/financial/ProfitLossSection.tsx` — revenue items + expense items + net profit + margin
    - Buat `apps/web/app/reports/components/financial/RevenueByAreaSection.tsx` — area revenue table + most profitable + attention needed
    - _Requirements: 27.1, 27.2, 27.3, 27.4, 27.5, 27.6_

  - [x] 20.3 Buat section components — Pelanggan tab
    - Buat `apps/web/app/reports/components/customer/GrowthSection.tsx` — 4 summary cards (Total Aktif, Baru, Churn, Net Growth) + delta + 12-month line chart + ARPU, CLV, Churn Rate metrics
    - Buat `apps/web/app/reports/components/customer/DistributionSection.tsx` — pie/donut charts: by package, area, status, connection method
    - Buat `apps/web/app/reports/components/customer/ChurnSection.tsx` — churned count + rate + by reason bar chart + by package/area tables + avg lifetime
    - _Requirements: 28.1, 28.2, 28.3, 28.4_

  - [x] 20.4 Buat section components — Jaringan tab
    - Buat `apps/web/app/reports/components/network/UptimeSection.tsx` — router uptime table dengan color coding + SLA target line + routers below SLA highlighted
    - Buat `apps/web/app/reports/components/network/TrafficSection.tsx` — total traffic cards + peak info + traffic per router horizontal bar chart + top 10 customers table
    - Buat `apps/web/app/reports/components/network/SignalSection.tsx` — signal distribution cards + average signal + degrading ONTs list + alarm summary table
    - Buat `apps/web/app/reports/components/network/CapacitySection.tsx` — router capacity table + ODP capacity table + recommendations list
    - Handle module_inactive: tampilkan ModuleInactive component
    - _Requirements: 29.1, 29.2, 29.3, 29.4, 29.5, 29.6_

  - [x] 20.5 Buat section components — Operasional tab
    - Buat `apps/web/app/reports/components/operational/ActivitySection.tsx` — per user activity table + top actions list
    - Buat `apps/web/app/reports/components/operational/NotificationSection.tsx` — summary cards + per channel breakdown + per template usage
    - Buat `apps/web/app/reports/components/operational/SyncSection.tsx` — MikroTik sync table + OLT sync table + overall sync success rate
    - _Requirements: 30.1, 30.2, 30.3, 30.4_

- [x] 21. Frontend — Comparison, Forecast, Custom Report Builder
  - [x] 21.1 Buat ComparisonView dan InsightCard
    - Buat `apps/web/app/reports/components/comparison/ComparisonView.tsx` — side-by-side comparison table: metric name, base value, compare value, delta absolute, delta percentage, trend indicator (arrow up green / arrow down red). Color-coded delta values.
    - Buat `apps/web/app/reports/components/comparison/InsightCard.tsx` — auto-generated insight card dengan text description
    - _Requirements: 31.1, 31.2, 31.3, 31.4_

  - [x] 21.2 Buat ForecastChart
    - Buat `apps/web/app/reports/components/forecast/ForecastChart.tsx` — line chart: actual data (solid line) 6 bulan + projected data (dashed line) 3 bulan. KPI target horizontal line. Estimated target month indicator. Disclaimer text. Insufficient data message.
    - _Requirements: 32.1, 32.2, 32.3, 32.4, 32.5_

  - [x] 21.3 Buat CustomReportBuilder dan TemplateList
    - Buat `apps/web/app/reports/components/custom/CustomReportBuilder.tsx` — form: Nama Laporan, Metrik (checkbox max 3), Group By (radio), Sub-Group (dropdown), Periode (date range), Tampilan (radio: Tabel/Bar/Line/Pie). Preview button, Simpan sebagai Template button, Export button.
    - Buat `apps/web/app/reports/components/custom/TemplateList.tsx` — saved templates list dengan load, edit, delete, schedule actions
    - _Requirements: 33.1, 33.2, 33.3, 33.4, 33.5, 33.6_

  - [x] 21.4 Buat ExportDialog dan ScheduleDialog
    - Buat `apps/web/app/reports/components/export/ExportDialog.tsx` — dialog export: pilih format (PDF, Excel, CSV), trigger export, tampilkan progress/status
    - Buat `apps/web/app/reports/components/export/ScheduleDialog.tsx` — dialog jadwal: Laporan (dropdown), Jadwal (Harian/Mingguan/Bulanan), Format (PDF/Excel), Kirim ke (Email/WhatsApp), Penerima tambahan. List active schedules.
    - Tambahkan export buttons di ReportPage: Export PDF, Export Excel, Jadwalkan Laporan
    - _Requirements: 26.6, 37.1, 37.2, 37.3, 37.4_


- [x] 22. Frontend — Halaman Pengeluaran
  - [x] 22.1 Buat ExpensePage dan components
    - Buat `apps/web/app/expenses/page.tsx` — halaman pengeluaran di route `/expenses`
    - Buat `apps/web/app/expenses/components/ExpenseTable.tsx` — tabel pengeluaran: Kategori, Jumlah, Tanggal, Keterangan, Recurring indicator, action menu (edit, delete). Total expenses di bawah tabel.
    - Buat `apps/web/app/expenses/components/ExpenseForm.tsx` — form tambah/edit: Kategori (dropdown), Jumlah (currency input), Keterangan (text), Tanggal (date picker), Recurring toggle (dengan day-of-month selector)
    - Buat `apps/web/app/expenses/components/CategoryManager.tsx` — manage kategori: add, edit, delete
    - _Requirements: 34.1, 34.2, 34.3, 34.4, 34.5_

- [x] 23. Frontend — Dashboard Widget
  - [x] 23.1 Buat DashboardWidget components
    - Buat/update `apps/web/app/page.tsx` atau buat `apps/web/app/components/DashboardWidget.tsx` — grid metric cards: Total Pelanggan Aktif (with trend), Pendapatan Bulan Ini (with target progress), Tunggakan (with customer count), Router Online/Offline, Collection Rate (with target), Churn Rate (with target), ARPU
    - KPI progress bars dengan color coding: hijau (tercapai), kuning (hampir), merah (di bawah)
    - Auto-refresh setiap 5 menit tanpa full page reload (menggunakan useDashboard hook)
    - Click pada widget → navigate ke section detail di /reports
    - Mobile: single-column stacked layout, min 44x44px touch targets
    - _Requirements: 35.1, 35.2, 35.3, 35.4, 35.5_

- [x] 24. Frontend — Mobile responsive
  - [x] 24.1 Implementasi mobile responsive layout
    - Update semua report components untuk mobile responsive:
      - Card-based layout pada viewport < 768px (stacked single-column)
      - Horizontal swipe untuk category tabs
      - Charts horizontally scrollable saat width melebihi viewport
      - Filter bar collapsible, hanya tampilkan active period by default
      - Minimum touch target 44x44px, text minimal 16px
    - _Requirements: 36.1, 36.2, 36.3, 36.4, 36.5_

- [x] 25. Checkpoint — Frontend selesai
  - Pastikan frontend build berhasil (`npm run build` di `apps/web/`). Tanyakan ke user jika ada pertanyaan.

- [x] 26. Integration tests
  - [x] 26.1 Tulis integration tests untuk report endpoints
    - Di `services/billing-api/internal/handler/report_handler_test.go`, test: HTTP status codes, response shape, RBAC (admin/operator/kasir bisa read, hanya admin bisa write), filter validation (period_start > period_end → 400), graceful degradation (module_inactive flag)
    - _Requirements: 1.1, 2.1, 7.1, 11.1, 25.1_

  - [x] 26.2 Tulis integration tests untuk expense endpoints
    - Di `services/billing-api/internal/handler/expense_handler_test.go`, test: CRUD operations, soft delete, category constraint (delete category with expenses → 409), category name duplicate → 409
    - _Requirements: 6.1, 6.5, 6.8_

  - [x] 26.3 Tulis integration tests untuk export dan schedule
    - Di `services/billing-api/internal/handler/export_handler_test.go`, test: CSV synchronous export, PDF/XLSX async (return 202 + job_id), job status query
    - Di `services/billing-api/internal/handler/schedule_handler_test.go`, test: CRUD schedule operations
    - _Requirements: 18.1, 18.2, 18.3, 19.1_

  - [x] 26.4 Tulis integration tests untuk network service client
    - Di `services/billing-api/internal/usecase/network_client_test.go`, test: successful HTTP call, timeout handling, fallback to cache, module_inactive response
    - _Requirements: 25.4, 11.5, 12.5_

  - [x] 26.5 Tulis integration tests untuk workers
    - Di `services/billing-api/internal/worker/export_worker_test.go`, test: async processing, retry on failure, file generation
    - Di `services/billing-api/internal/worker/schedule_worker_test.go`, test: cron execution, report delivery
    - Di `services/billing-api/internal/worker/recurring_expense_worker_test.go`, test: auto-create recurring expenses
    - _Requirements: 18.3, 18.8, 19.5, 6.10_

- [x] 27. Final checkpoint — Semua tests pass
  - Pastikan semua tests pass (`go test ./...` di `services/billing-api`). Pastikan frontend build berhasil. Tanyakan ke user jika ada pertanyaan.

## Notes

- Task yang ditandai `*` bersifat opsional dan bisa dilewati untuk MVP lebih cepat
- Setiap task mereferensikan requirements spesifik untuk traceability
- Checkpoints memastikan validasi inkremental di setiap layer
- Property tests memvalidasi correctness properties universal dari desain dokumen
- Unit tests memvalidasi contoh spesifik dan edge cases
- Maksimal 200 baris per file Go — pecah ke beberapa file jika perlu
- Semua komentar dalam Bahasa Indonesia
- Gunakan `pgregory.net/rapid` untuk property-based testing
- Gunakan `sqlc` untuk query generation
- Gunakan Fiber v2 untuk HTTP handlers
- Gunakan `asynq` untuk background jobs
- Gunakan Recharts untuk frontend charts
- Monetary values menggunakan BIGINT (Rupiah)
