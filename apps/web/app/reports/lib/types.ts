// =============================================================================
// TypeScript types — matching semua backend DTOs untuk Reporting & Analytics
// =============================================================================

// --- Report Filter ---

export interface ReportFilter {
  period_start: string;
  period_end: string;
  compare_start?: string;
  compare_end?: string;
  area_id?: string;
  package_id?: string;
  router_id?: string;
}

// --- Revenue Report ---

export interface RevenueSource {
  monthly_subscription: number;
  voucher_sales: number;
  installation_fees: number;
  late_fees: number;
  other: number;
  total: number;
}

export interface RevenueDelta {
  absolute: number;
  percentage: number;
}

export interface MonthlyRevenueTrend {
  month: string;
  total_revenue: number;
  monthly_subscription: number;
  voucher_sales: number;
  other_revenue: number;
}

export interface RevenueReport {
  current: RevenueSource;
  comparison?: RevenueSource;
  delta?: Record<string, RevenueDelta>;
  trend: MonthlyRevenueTrend[];
  kpi_target?: number;
  kpi_progress?: number;
}

// --- Aging Report ---

export interface AgingBucket {
  label: string;
  total_amount: number;
  customer_count: number;
}

export interface TopDebtor {
  customer_id: string;
  customer_name: string;
  total_outstanding: number;
  months_overdue: number;
}

export interface ReceivablesTrend {
  month: string;
  total_outstanding: number;
}

export interface AgingReport {
  buckets: AgingBucket[];
  total_outstanding: number;
  collection_rate: number;
  avg_days_to_pay: number;
  top_debtors: TopDebtor[];
  trend: ReceivablesTrend[];
  kpi_target?: number;
}

// --- Payment Report ---

export interface PaymentMethodBreakdown {
  method_name: string;
  total_amount: number;
  transaction_count: number;
  percentage: number;
}

export interface DailyPayment {
  date: string;
  total_amount: number;
  transaction_count: number;
}

export interface PaymentReport {
  methods: PaymentMethodBreakdown[];
  daily_payments: DailyPayment[];
  peak_payment_date: string;
  peak_amount: number;
}

// --- Voucher Revenue Report ---

export interface VoucherByPackage {
  package_name: string;
  total_revenue: number;
  voucher_count: number;
  percentage: number;
}

export interface VoucherByReseller {
  reseller_name: string;
  total_revenue: number;
  voucher_count: number;
  reseller_margin: number;
}

export interface VoucherRevenueReport {
  total_revenue: number;
  total_voucher_count: number;
  by_package: VoucherByPackage[];
  by_reseller: VoucherByReseller[];
  total_reseller_margin: number;
}

// --- Profit Loss Report ---

export interface ProfitLossLineItem {
  label: string;
  amount: number;
}

export interface ProfitLossReport {
  revenue_items: ProfitLossLineItem[];
  total_revenue: number;
  expense_items: ProfitLossLineItem[];
  total_expenses: number;
  net_profit: number;
  profit_margin: number;
  comparison?: ProfitLossReport;
}

// --- Customer Growth Report ---

export interface MonthlyGrowthTrend {
  month: string;
  total_active: number;
  new_customers: number;
  churned_customers: number;
}

export interface CustomerGrowthReport {
  total_active: number;
  new_customers: number;
  churned_customers: number;
  net_growth: number;
  arpu: number;
  clv: number;
  churn_rate: number;
  trend: MonthlyGrowthTrend[];
  comparison?: CustomerGrowthReport;
  delta?: Record<string, RevenueDelta>;
}

// --- Customer Distribution Report ---

export interface DistributionItem {
  id?: string;
  name: string;
  count: number;
  percentage: number;
}

export interface CustomerDistributionReport {
  by_package: DistributionItem[];
  by_area: DistributionItem[];
  by_status: Record<string, number>;
  by_connection_method: DistributionItem[];
}

// --- Churn Analysis Report ---

export interface ChurnByReason {
  reason: string;
  count: number;
  percentage: number;
}

export interface ChurnAnalysisReport {
  churned_count: number;
  churn_rate: number;
  by_reason: ChurnByReason[];
  by_package: DistributionItem[];
  by_area: DistributionItem[];
  average_lifetime_months: number;
}

// --- Revenue by Area Report ---

export interface AreaRevenue {
  area_id: string;
  area_name: string;
  customer_count: number;
  total_revenue: number;
  total_outstanding: number;
  arpu: number;
}

export interface RevenueByAreaReport {
  areas: AreaRevenue[];
  total: AreaRevenue;
  most_profitable_area: string;
  attention_needed_area: string;
}

// --- Comparison Report ---

export type ComparisonType = "mom" | "yoy" | "qoq" | "custom";

export interface ComparisonMetric {
  metric_name: string;
  base_value: number;
  compare_value: number;
  delta_absolute: number;
  delta_percentage: number;
  trend: "improving" | "declining" | "stable";
}

export interface ComparisonReport {
  comparison_type: ComparisonType;
  base_period: string;
  compare_period: string;
  metrics: ComparisonMetric[];
  insights: string[];
}

// --- Forecast Report ---

export interface ForecastMonth {
  month: string;
  projected_revenue: number;
  projected_customers: number;
  projected_receivables: number;
}

export interface ForecastReport {
  projections: ForecastMonth[];
  estimated_target_date?: Record<string, string>;
  insufficient_data: boolean;
  disclaimer?: string;
}

// --- Dashboard Data ---

export interface DashboardData {
  total_active_customers: number;
  customers_trend: number;
  monthly_revenue: number;
  revenue_target?: number;
  revenue_progress?: number;
  total_receivables: number;
  receivables_count: number;
  routers_online: number;
  routers_offline: number;
  collection_rate: number;
  collection_target?: number;
  churn_rate: number;
  churn_target?: number;
  arpu: number;
  module_inactive?: Record<string, boolean>;
}

// --- Network Reports ---

export interface RouterUptimeItem {
  router_id: string;
  router_name: string;
  uptime_percentage: number;
  total_downtime_minutes: number;
  reboot_count: number;
  status_label: string;
}

export interface DowntimeEvent {
  start_time: string;
  end_time: string;
  duration_minutes: number;
  cause?: string;
}

export interface UptimeReport {
  routers: RouterUptimeItem[];
  sla_target?: number;
  routers_below_sla?: RouterUptimeItem[];
  downtime_timeline?: DowntimeEvent[];
  module_inactive: boolean;
  stale_data: boolean;
  last_updated?: string;
}

export interface RouterTraffic {
  router_id: string;
  router_name: string;
  download_bytes: number;
  upload_bytes: number;
  percentage: number;
}

export interface CustomerTraffic {
  customer_id: string;
  customer_name: string;
  package_name: string;
  download_bytes: number;
  upload_bytes: number;
  over_use_flag: boolean;
}

export interface TrafficReport {
  total_download_bytes: number;
  total_upload_bytes: number;
  total_traffic_bytes: number;
  peak_traffic_bps: number;
  peak_traffic_time?: string;
  average_traffic_bps: number;
  by_router: RouterTraffic[];
  top_customers: CustomerTraffic[];
  module_inactive: boolean;
}

export interface DegradingONT {
  customer_name: string;
  customer_id: string;
  current_signal_dbm: number;
  signal_change_db: number;
}

export interface AlarmTypeSummary {
  alarm_type: string;
  count: number;
  avg_duration_minutes: number;
  resolved_percentage: number;
}

export interface SignalQualityReport {
  normal_count: number;
  warning_count: number;
  weak_count: number;
  critical_count: number;
  total_ont_count: number;
  average_signal_dbm: number;
  degrading_onts: DegradingONT[];
  alarm_summary: AlarmTypeSummary[];
  module_inactive: boolean;
}

export interface RouterCapacity {
  router_id: string;
  router_name: string;
  current_customers: number;
  max_capacity: number;
  usage_percentage: number;
  estimated_full_date?: string;
}

export interface ODPCapacity {
  odp_id: string;
  odp_name: string;
  used_ports: number;
  total_ports: number;
  usage_percentage: number;
  status_label: string;
}

export interface CapacityReport {
  router_capacity?: RouterCapacity[];
  odp_capacity?: ODPCapacity[];
  recommendations: string[];
  module_inactive?: Record<string, boolean>;
}

// --- Operational Reports ---

export interface UserActivity {
  user_id: string;
  user_name: string;
  role: string;
  login_days: number;
  action_count: number;
  last_active_at: string;
}

export interface ActionSummary {
  action_type: string;
  count: number;
  percentage: number;
}

export interface ActivityReport {
  per_user: UserActivity[];
  top_actions: ActionSummary[];
}

export interface ChannelStats {
  channel: string;
  sent_count: number;
  delivered_count: number;
  failed_count: number;
  success_rate: number;
  cost: number;
}

export interface TemplateStats {
  template_name: string;
  sent_count: number;
}

export interface NotificationReport {
  total_sent: number;
  total_delivered: number;
  total_failed: number;
  success_rate: number;
  total_cost: number;
  per_channel: ChannelStats[];
  per_template: TemplateStats[];
  module_inactive: boolean;
}

export interface RouterSyncStatus {
  router_id: string;
  router_name: string;
  sync_ok_count: number;
  sync_failed_count: number;
  orphan_user_count: number;
  pending_sync_count: number;
}

export interface OLTSyncStatus {
  olt_id: string;
  olt_name: string;
  sync_ok_count: number;
  sync_failed_count: number;
  unmanaged_ont_count: number;
}

export interface SyncReport {
  mikrotik_sync?: RouterSyncStatus[];
  olt_sync?: OLTSyncStatus[];
  sync_success_rate: number;
  module_inactive?: Record<string, boolean>;
}

// --- Expense ---

export interface Expense {
  id: string;
  tenant_id: string;
  category_id: string;
  category_name?: string;
  amount: number;
  description: string;
  expense_date: string;
  payment_method?: string;
  vendor_name?: string;
  reference_number?: string;
  attachment_url?: string;
  is_recurring: boolean;
  recurring_day?: number;
  created_by_id: string;
  created_by_name?: string;
  deleted_at?: string;
  created_at: string;
  updated_at: string;
}

export interface ExpenseCategory {
  id: string;
  tenant_id: string;
  name: string;
  is_default: boolean;
  expense_count?: number;
  deleted_at?: string;
  created_at: string;
  updated_at: string;
}

// --- KPI Target ---

export interface KPITarget {
  id: string;
  tenant_id: string;
  monthly_revenue_target?: number;
  collection_rate_target?: number;
  max_receivables?: number;
  new_customers_monthly_target?: number;
  max_churn_rate?: number;
  total_customers_target?: number;
  sla_uptime_target?: number;
  max_active_alarms?: number;
  min_signal_quality_percentage?: number;
  created_at: string;
  updated_at: string;
}

// --- Report Schedule ---

export type ScheduleType = "daily" | "weekly" | "monthly";

export interface Recipient {
  type: "email" | "whatsapp";
  address: string;
}

export interface ReportSchedule {
  id: string;
  tenant_id: string;
  report_type: string;
  schedule_type: ScheduleType;
  format: string;
  recipients: Recipient[];
  filters: ReportFilter;
  is_active: boolean;
  created_by_id: string;
  created_at: string;
  updated_at: string;
}

// --- Report Job ---

export type ReportJobStatus = "pending" | "processing" | "completed" | "failed";

export interface ReportJob {
  id: string;
  tenant_id: string;
  report_type: string;
  format: string;
  filters: ReportFilter;
  status: ReportJobStatus;
  download_url?: string;
  error?: string;
  requested_by: string;
  created_at: string;
  updated_at: string;
}

// --- Custom Report Template ---

export interface CustomReportTemplate {
  id: string;
  tenant_id: string;
  name: string;
  metrics: string[];
  group_by: string;
  sub_group_by?: string;
  display_type: "table" | "bar_chart" | "line_chart" | "pie_chart";
  default_period_range?: string;
  created_by_id: string;
  created_at: string;
  updated_at: string;
}

// --- Request DTOs ---

export interface CreateExpenseRequest {
  category_id: string;
  amount: number;
  description?: string;
  expense_date: string;
  payment_method?: string;
  vendor_name?: string;
  reference_number?: string;
  attachment_url?: string;
  is_recurring: boolean;
  recurring_day?: number;
}

export interface UpdateExpenseRequest {
  category_id?: string;
  amount?: number;
  description?: string;
  expense_date?: string;
  payment_method?: string;
  vendor_name?: string;
  reference_number?: string;
  attachment_url?: string;
  is_recurring?: boolean;
  recurring_day?: number;
}

export interface CreateCategoryRequest {
  name: string;
}

export interface UpdateCategoryRequest {
  name: string;
}

export interface CreateScheduleRequest {
  report_type: string;
  schedule_type: ScheduleType;
  format: "pdf" | "xlsx";
  recipients: Recipient[];
  filters?: ReportFilter;
}

export interface UpdateScheduleRequest {
  report_type?: string;
  schedule_type?: ScheduleType;
  format?: "pdf" | "xlsx";
  recipients?: Recipient[];
  filters?: ReportFilter;
}

export interface UpdateKPITargetRequest {
  monthly_revenue_target?: number;
  collection_rate_target?: number;
  max_receivables?: number;
  new_customers_monthly_target?: number;
  max_churn_rate?: number;
  total_customers_target?: number;
  sla_uptime_target?: number;
  max_active_alarms?: number;
  min_signal_quality_percentage?: number;
}

export interface CreateTemplateRequest {
  name: string;
  metrics: string[];
  group_by: string;
  sub_group_by?: string;
  display_type: "table" | "bar_chart" | "line_chart" | "pie_chart";
  default_period_range?: string;
}

export interface ExportRequest {
  report_type: string;
  format: "pdf" | "xlsx" | "csv";
  filters?: ReportFilter;
}
