// =============================================================================
// API client functions — fetch untuk semua report endpoints
// =============================================================================

import type {
  ReportFilter,
  RevenueReport,
  AgingReport,
  PaymentReport,
  VoucherRevenueReport,
  ProfitLossReport,
  RevenueByAreaReport,
  CustomerGrowthReport,
  CustomerDistributionReport,
  ChurnAnalysisReport,
  UptimeReport,
  TrafficReport,
  SignalQualityReport,
  CapacityReport,
  ActivityReport,
  NotificationReport,
  SyncReport,
  ComparisonReport,
  ComparisonType,
  ForecastReport,
  DashboardData,
  Expense,
  ExpenseCategory,
  KPITarget,
  ReportSchedule,
  ReportJob,
  CustomReportTemplate,
  CreateExpenseRequest,
  UpdateExpenseRequest,
  CreateCategoryRequest,
  UpdateCategoryRequest,
  CreateScheduleRequest,
  UpdateScheduleRequest,
  UpdateKPITargetRequest,
  CreateTemplateRequest,
  ExportRequest,
} from "./types";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "/api/billing";
const REPORTS_BASE = `${API_BASE}/reports`;
const EXPENSES_BASE = `${API_BASE}/expenses`;

// --- Helper ---

function buildFilterParams(filter?: Partial<ReportFilter>): URLSearchParams {
  const params = new URLSearchParams();
  if (!filter) return params;
  if (filter.period_start) params.set("period_start", filter.period_start);
  if (filter.period_end) params.set("period_end", filter.period_end);
  if (filter.compare_start) params.set("compare_start", filter.compare_start);
  if (filter.compare_end) params.set("compare_end", filter.compare_end);
  if (filter.area_id) params.set("area_id", filter.area_id);
  if (filter.package_id) params.set("package_id", filter.package_id);
  if (filter.router_id) params.set("router_id", filter.router_id);
  return params;
}

async function fetchJSON<T>(url: string, init?: RequestInit): Promise<T> {
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json", ...init?.headers },
    ...init,
  });
  if (!res.ok) {
    const body = await res.text();
    throw new Error(`API error ${res.status}: ${body}`);
  }
  const body = await res.json();
  if (body && typeof body === "object" && "success" in body && "data" in body) {
    return body.data as T;
  }
  return body as T;
}

async function postJSON<T>(url: string, body: unknown): Promise<T> {
  return fetchJSON<T>(url, {
    method: "POST",
    body: JSON.stringify(body),
  });
}

async function putJSON<T>(url: string, body: unknown): Promise<T> {
  return fetchJSON<T>(url, {
    method: "PUT",
    body: JSON.stringify(body),
  });
}

async function deleteRequest(url: string): Promise<void> {
  const res = await fetch(url, {
    method: "DELETE",
    headers: { "Content-Type": "application/json" },
  });
  if (!res.ok) {
    const body = await res.text();
    throw new Error(`API error ${res.status}: ${body}`);
  }
}

// =============================================================================
// Financial Reports
// =============================================================================

export function fetchRevenueReport(filter?: Partial<ReportFilter>): Promise<RevenueReport> {
  const params = buildFilterParams(filter);
  return fetchJSON(`${REPORTS_BASE}/financial/revenue?${params}`);
}

export function fetchAgingReport(filter?: Partial<ReportFilter>): Promise<AgingReport> {
  const params = buildFilterParams(filter);
  return fetchJSON(`${REPORTS_BASE}/financial/aging?${params}`);
}

export function fetchPaymentReport(filter?: Partial<ReportFilter>): Promise<PaymentReport> {
  const params = buildFilterParams(filter);
  return fetchJSON(`${REPORTS_BASE}/financial/payments?${params}`);
}

export function fetchVoucherReport(filter?: Partial<ReportFilter>): Promise<VoucherRevenueReport> {
  const params = buildFilterParams(filter);
  return fetchJSON(`${REPORTS_BASE}/financial/vouchers?${params}`);
}

export function fetchProfitLossReport(filter?: Partial<ReportFilter>): Promise<ProfitLossReport> {
  const params = buildFilterParams(filter);
  return fetchJSON(`${REPORTS_BASE}/financial/profit-loss?${params}`);
}

export function fetchRevenueByAreaReport(filter?: Partial<ReportFilter>): Promise<RevenueByAreaReport> {
  const params = buildFilterParams(filter);
  return fetchJSON(`${REPORTS_BASE}/financial/revenue-by-area?${params}`);
}

// =============================================================================
// Customer Reports
// =============================================================================

export function fetchCustomerGrowthReport(filter?: Partial<ReportFilter>): Promise<CustomerGrowthReport> {
  const params = buildFilterParams(filter);
  return fetchJSON(`${REPORTS_BASE}/customers/growth?${params}`);
}

export function fetchDistributionReport(filter?: Partial<ReportFilter>): Promise<CustomerDistributionReport> {
  const params = buildFilterParams(filter);
  return fetchJSON(`${REPORTS_BASE}/customers/distribution?${params}`);
}

export function fetchChurnReport(filter?: Partial<ReportFilter>): Promise<ChurnAnalysisReport> {
  const params = buildFilterParams(filter);
  return fetchJSON(`${REPORTS_BASE}/customers/churn?${params}`);
}

// =============================================================================
// Network Reports
// =============================================================================

export function fetchUptimeReport(filter?: Partial<ReportFilter>): Promise<UptimeReport> {
  const params = buildFilterParams(filter);
  return fetchJSON(`${REPORTS_BASE}/network/uptime?${params}`);
}

export function fetchTrafficReport(filter?: Partial<ReportFilter>): Promise<TrafficReport> {
  const params = buildFilterParams(filter);
  return fetchJSON(`${REPORTS_BASE}/network/traffic?${params}`);
}

export function fetchSignalReport(filter?: Partial<ReportFilter>): Promise<SignalQualityReport> {
  const params = buildFilterParams(filter);
  return fetchJSON(`${REPORTS_BASE}/network/signal-quality?${params}`);
}

export function fetchCapacityReport(): Promise<CapacityReport> {
  return fetchJSON(`${REPORTS_BASE}/network/capacity`);
}

// =============================================================================
// Operational Reports
// =============================================================================

export function fetchActivityReport(filter?: Partial<ReportFilter>): Promise<ActivityReport> {
  const params = buildFilterParams(filter);
  return fetchJSON(`${REPORTS_BASE}/operational/activity?${params}`);
}

export function fetchNotificationReport(filter?: Partial<ReportFilter>): Promise<NotificationReport> {
  const params = buildFilterParams(filter);
  return fetchJSON(`${REPORTS_BASE}/operational/notifications?${params}`);
}

export function fetchSyncReport(filter?: Partial<ReportFilter>): Promise<SyncReport> {
  const params = buildFilterParams(filter);
  return fetchJSON(`${REPORTS_BASE}/operational/sync?${params}`);
}

// =============================================================================
// Comparison & Forecast
// =============================================================================

export function fetchComparisonReport(
  comparisonType: ComparisonType,
  basePeriodStart: string,
  basePeriodEnd: string,
  comparePeriodStart?: string,
  comparePeriodEnd?: string,
): Promise<ComparisonReport> {
  const params = new URLSearchParams();
  params.set("comparison_type", comparisonType);
  params.set("base_period_start", basePeriodStart);
  params.set("base_period_end", basePeriodEnd);
  if (comparePeriodStart) params.set("compare_period_start", comparePeriodStart);
  if (comparePeriodEnd) params.set("compare_period_end", comparePeriodEnd);
  return fetchJSON(`${REPORTS_BASE}/comparison?${params}`);
}

export function fetchForecastReport(): Promise<ForecastReport> {
  return fetchJSON(`${REPORTS_BASE}/forecast`);
}

// =============================================================================
// Dashboard
// =============================================================================

export function fetchDashboardData(): Promise<DashboardData> {
  return fetchJSON(`${REPORTS_BASE}/dashboard`);
}

// =============================================================================
// Export
// =============================================================================

export function requestExport(req: ExportRequest): Promise<{ job_id: string }> {
  return postJSON(`${REPORTS_BASE}/export`, req);
}

export function getExportStatus(jobId: string): Promise<ReportJob> {
  return fetchJSON(`${REPORTS_BASE}/export/${jobId}`);
}

// =============================================================================
// Schedules
// =============================================================================

export function fetchSchedules(): Promise<ReportSchedule[]> {
  return fetchJSON(`${REPORTS_BASE}/schedules`);
}

export function createSchedule(req: CreateScheduleRequest): Promise<ReportSchedule> {
  return postJSON(`${REPORTS_BASE}/schedules`, req);
}

export function updateSchedule(id: string, req: UpdateScheduleRequest): Promise<ReportSchedule> {
  return putJSON(`${REPORTS_BASE}/schedules/${id}`, req);
}

export function deleteSchedule(id: string): Promise<void> {
  return deleteRequest(`${REPORTS_BASE}/schedules/${id}`);
}

// =============================================================================
// KPI Targets
// =============================================================================

export function fetchKPITargets(): Promise<KPITarget> {
  return fetchJSON(`${REPORTS_BASE}/kpi-targets`);
}

export function updateKPITargets(req: UpdateKPITargetRequest): Promise<KPITarget> {
  return putJSON(`${REPORTS_BASE}/kpi-targets`, req);
}

// =============================================================================
// Custom Reports
// =============================================================================

export function previewCustomReport(
  metrics: string[],
  groupBy: string,
  subGroupBy: string | undefined,
  periodStart: string,
  periodEnd: string,
  displayType: string,
): Promise<unknown> {
  return postJSON(`${REPORTS_BASE}/custom/preview`, {
    metrics,
    group_by: groupBy,
    sub_group_by: subGroupBy,
    period_start: periodStart,
    period_end: periodEnd,
    display_type: displayType,
  });
}

export function fetchTemplates(): Promise<CustomReportTemplate[]> {
  return fetchJSON(`${REPORTS_BASE}/custom/templates`);
}

export function createTemplate(req: CreateTemplateRequest): Promise<CustomReportTemplate> {
  return postJSON(`${REPORTS_BASE}/custom/templates`, req);
}

export function deleteTemplate(id: string): Promise<void> {
  return deleteRequest(`${REPORTS_BASE}/custom/templates/${id}`);
}

// =============================================================================
// Expenses
// =============================================================================

export function fetchExpenses(
  periodStart?: string,
  periodEnd?: string,
  categoryId?: string,
): Promise<Expense[]> {
  const params = new URLSearchParams();
  if (periodStart) params.set("period_start", periodStart);
  if (periodEnd) params.set("period_end", periodEnd);
  if (categoryId) params.set("category_id", categoryId);
  return fetchJSON(`${EXPENSES_BASE}?${params}`);
}

export function createExpense(req: CreateExpenseRequest): Promise<Expense> {
  return postJSON(EXPENSES_BASE, req);
}

export function updateExpense(id: string, req: UpdateExpenseRequest): Promise<Expense> {
  return putJSON(`${EXPENSES_BASE}/${id}`, req);
}

export function deleteExpense(id: string): Promise<void> {
  return deleteRequest(`${EXPENSES_BASE}/${id}`);
}

export function fetchCategories(): Promise<ExpenseCategory[]> {
  return fetchJSON(`${EXPENSES_BASE}/categories`);
}

export function createCategory(req: CreateCategoryRequest): Promise<ExpenseCategory> {
  return postJSON(`${EXPENSES_BASE}/categories`, req);
}

export function updateCategory(id: string, req: UpdateCategoryRequest): Promise<ExpenseCategory> {
  return putJSON(`${EXPENSES_BASE}/categories/${id}`, req);
}

export function deleteCategory(id: string): Promise<void> {
  return deleteRequest(`${EXPENSES_BASE}/categories/${id}`);
}
