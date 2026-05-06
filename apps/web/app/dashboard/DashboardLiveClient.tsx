"use client";

import { useEffect, useMemo, useState } from "react";
import { ArrowClockwise, ChartLineUp, WarningCircle } from "@phosphor-icons/react";
import AppShell from "../components/app-shell";
import { Button, DataTable, EmptyState, PageHeader, Section, StatGrid, StatusBadge } from "../components/ui";

type DashboardMetrics = {
  total_active_customers: number;
  customers_trend: number;
  monthly_revenue: number;
  total_receivables: number;
  receivables_count: number;
  routers_online: number;
  routers_offline: number;
  collection_rate: number;
  churn_rate: number;
  arpu: number;
};

type RouterSummary = {
  total_routers: number;
  online_count: number;
  offline_count: number;
  maintenance_count: number;
};

type OltSummary = {
  total?: number;
  total_olts?: number;
  online?: number;
  online_count?: number;
  offline?: number;
  offline_count?: number;
  maintenance?: number;
  maintenance_count?: number;
  active_alarms?: number;
  active_alarm_count?: number;
};

type CashflowSummary = {
  total_cash_in: number;
  total_cash_out: number;
  net_cashflow: number;
  opening_balance: number;
  closing_balance_estimate: number;
  breakdown: { direction: string; source: string; category: string; amount: number }[];
};

type ApiResponse<T> = {
  success: boolean;
  data?: T;
  error?: {
    code: string;
    message: string;
  };
};

type ModuleCapabilities = {
  billing_core: boolean;
  mikrotik: boolean;
  fiber_network: boolean;
};

type StatItem = { label: string; value: string; delta?: string; tone?: "slate" | "blue" | "green" | "amber" | "red" | "violet" };

const defaultModules: ModuleCapabilities = {
  billing_core: true,
  mikrotik: false,
  fiber_network: false,
};

const emptyMetrics: DashboardMetrics = {
  total_active_customers: 0,
  customers_trend: 0,
  monthly_revenue: 0,
  total_receivables: 0,
  receivables_count: 0,
  routers_online: 0,
  routers_offline: 0,
  collection_rate: 0,
  churn_rate: 0,
  arpu: 0,
};

const emptyCashflow: CashflowSummary = {
  total_cash_in: 0,
  total_cash_out: 0,
  net_cashflow: 0,
  opening_balance: 0,
  closing_balance_estimate: 0,
  breakdown: [],
};

function formatCurrency(value: number) {
  return new Intl.NumberFormat("id-ID", {
    style: "currency",
    currency: "IDR",
    maximumFractionDigits: 0,
  }).format(value);
}

function formatPercent(value: number) {
  return `${Math.round(value * 100) / 100}%`;
}

function extractMessage(error: unknown) {
  return error instanceof Error ? error.message : "Terjadi kesalahan";
}

function currentMonthPeriod() {
  const now = new Date();
  const start = new Date(now.getFullYear(), now.getMonth(), 1);
  const end = new Date(now.getFullYear(), now.getMonth() + 1, 0);
  return {
    start: start.toISOString().slice(0, 10),
    end: end.toISOString().slice(0, 10),
  };
}

function RevenueSnapshot({ metrics }: { metrics: DashboardMetrics }) {
  const collected = Math.max(metrics.monthly_revenue, 0);
  const receivables = Math.max(metrics.total_receivables, 0);
  const total = Math.max(collected + receivables, 1);
  const collectedWidth = (collected / total) * 100;
  const receivableWidth = (receivables / total) * 100;

  return (
    <div className="space-y-5">
      <div className="grid gap-3 sm:grid-cols-3">
        {[
          { label: "Diterima bulan ini", value: formatCurrency(collected), detail: "Dari pembayaran tercatat" },
          { label: "Piutang aktif", value: formatCurrency(receivables), detail: `${metrics.receivables_count} invoice belum lunas` },
          { label: "Collection rate", value: formatPercent(metrics.collection_rate), detail: "Rasio pembayaran berjalan" },
        ].map((item) => (
          <div key={item.label} className="rounded-lg border border-slate-200 bg-slate-50 px-4 py-3">
            <p className="text-xs font-medium uppercase tracking-[0.14em] text-slate-500">{item.label}</p>
            <p className="mt-2 font-mono text-xl font-semibold text-slate-950">{item.value}</p>
            <p className="mt-1 text-xs text-slate-500">{item.detail}</p>
          </div>
        ))}
      </div>

      <div className="rounded-lg border border-slate-200 bg-white p-4">
        <div className="mb-3 flex items-center justify-between gap-3">
          <span className="inline-flex items-center gap-2 text-sm font-semibold text-slate-800">
            <ChartLineUp size={18} />
            Komposisi tagihan bulan ini
          </span>
          <span className="font-mono text-xs text-slate-500">{formatCurrency(total === 1 ? 0 : total)}</span>
        </div>
        <div className="flex h-8 overflow-hidden rounded-md bg-slate-100">
          <div className="bg-blue-600" style={{ width: `${collectedWidth}%` }} />
          <div className="bg-amber-400" style={{ width: `${receivableWidth}%` }} />
        </div>
        <div className="mt-3 flex flex-wrap gap-x-5 gap-y-2 text-xs text-slate-500">
          <span className="inline-flex items-center gap-2"><span className="h-2.5 w-2.5 rounded-sm bg-blue-600" />Terbayar</span>
          <span className="inline-flex items-center gap-2"><span className="h-2.5 w-2.5 rounded-sm bg-amber-400" />Piutang</span>
        </div>
      </div>
    </div>
  );
}

function CashflowSnapshot({ summary }: { summary: CashflowSummary }) {
  const total = Math.max(summary.total_cash_in + summary.total_cash_out, 1);
  const incomeWidth = (Math.max(summary.total_cash_in, 0) / total) * 100;
  const expenseWidth = (Math.max(summary.total_cash_out, 0) / total) * 100;
  const incomeBreakdown = summary.breakdown.filter((item) => item.direction === "in");
  const expenseBreakdown = summary.breakdown.filter((item) => item.direction === "out");

  return (
    <div className="space-y-4">
      <div className="grid gap-3 sm:grid-cols-3">
        <div className="rounded-lg border border-emerald-200 bg-emerald-50 px-4 py-3">
          <p className="text-xs font-medium uppercase tracking-[0.14em] text-emerald-700">Kas masuk</p>
          <p className="mt-2 font-mono text-lg font-semibold text-emerald-900">{formatCurrency(summary.total_cash_in)}</p>
        </div>
        <div className="rounded-lg border border-red-200 bg-red-50 px-4 py-3">
          <p className="text-xs font-medium uppercase tracking-[0.14em] text-red-700">Kas keluar</p>
          <p className="mt-2 font-mono text-lg font-semibold text-red-900">{formatCurrency(summary.total_cash_out)}</p>
        </div>
        <div className="rounded-lg border border-slate-200 bg-slate-50 px-4 py-3">
          <p className="text-xs font-medium uppercase tracking-[0.14em] text-slate-500">Saldo akhir</p>
          <p className="mt-2 font-mono text-lg font-semibold text-slate-950">{formatCurrency(summary.closing_balance_estimate)}</p>
        </div>
      </div>
      <div className="flex h-7 overflow-hidden rounded-md bg-slate-100">
        <div className="bg-emerald-500" style={{ width: `${incomeWidth}%` }} />
        <div className="bg-red-400" style={{ width: `${expenseWidth}%` }} />
      </div>
      <div className="grid gap-3 text-sm sm:grid-cols-2">
        <BreakdownList title="Sumber masuk" items={incomeBreakdown} empty="Belum ada kas masuk periode ini." />
        <BreakdownList title="Sumber keluar" items={expenseBreakdown} empty="Belum ada pengeluaran periode ini." />
      </div>
    </div>
  );
}

function BreakdownList({ title, items, empty }: { title: string; items: CashflowSummary["breakdown"]; empty: string }) {
  return (
    <div className="rounded-lg border border-slate-200 p-4">
      <p className="font-semibold text-slate-900">{title}</p>
      <div className="mt-3 space-y-2">
        {items.length === 0 && <p className="text-sm text-slate-500">{empty}</p>}
        {items.slice(0, 4).map((item) => (
          <div key={`${item.direction}-${item.source}-${item.category}`} className="flex min-w-0 items-center justify-between gap-3 text-sm">
            <span className="min-w-0 truncate text-slate-600">{item.category}</span>
            <span className="shrink-0 font-mono font-semibold text-slate-950">{formatCurrency(item.amount)}</span>
          </div>
        ))}
      </div>
    </div>
  );
}

export default function DashboardLiveClient() {
  const [metrics, setMetrics] = useState<DashboardMetrics>(emptyMetrics);
  const [cashflowSummary, setCashflowSummary] = useState<CashflowSummary>(emptyCashflow);
  const [routerSummary, setRouterSummary] = useState<RouterSummary | null>(null);
  const [oltSummary, setOltSummary] = useState<OltSummary | null>(null);
  const [modules, setModules] = useState<ModuleCapabilities>(defaultModules);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  async function loadDashboard() {
    setLoading(true);
    setError("");
    try {
      const moduleResponse = await fetch("/api/billing/tenant/modules", { cache: "no-store" });
      const moduleJson = await moduleResponse.json().catch(() => ({}));
      const nextModuleData = moduleJson?.data?.modules ?? moduleJson?.modules ?? defaultModules;
      const nextModules = {
        billing_core: nextModuleData.billing_core !== false,
        mikrotik: nextModuleData.mikrotik === true,
        fiber_network: nextModuleData.fiber_network === true,
      };
      setModules(nextModules);
      const period = currentMonthPeriod();

      const [dashboardResponse, cashflowResponse, routerResponse, oltResponse] = await Promise.all([
        fetch("/api/billing/reports/dashboard", { cache: "no-store" }),
        fetch(`/api/billing/cashflow/summary?period_start=${period.start}&period_end=${period.end}`, { cache: "no-store" }),
        nextModules.mikrotik
          ? fetch("/api/network/mikrotik/status/summary", { cache: "no-store" })
          : Promise.resolve(null),
        nextModules.fiber_network
          ? fetch("/api/network-service/olt/summary", { cache: "no-store" })
          : Promise.resolve(null),
      ]);
      const dashboardJson = (await dashboardResponse.json()) as ApiResponse<DashboardMetrics>;
      const cashflowJson = (await cashflowResponse.json()) as ApiResponse<CashflowSummary>;
      const routerJson = routerResponse ? ((await routerResponse.json()) as ApiResponse<RouterSummary>) : null;
      const oltJson = oltResponse ? ((await oltResponse.json()) as ApiResponse<OltSummary>) : null;

      if (!dashboardResponse.ok || !dashboardJson.success) {
        throw new Error(dashboardJson.error?.message || "Gagal mengambil dashboard");
      }
      setMetrics(dashboardJson.data || emptyMetrics);
      setCashflowSummary(cashflowResponse.ok && cashflowJson.success ? cashflowJson.data || emptyCashflow : emptyCashflow);
      setRouterSummary(routerResponse?.ok && routerJson?.success ? routerJson.data || null : null);
      setOltSummary(oltResponse?.ok && oltJson?.success ? oltJson.data || null : null);
    } catch (loadError) {
      setError(extractMessage(loadError));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void loadDashboard();
  }, []);

  const stats = useMemo(() => {
    const items: StatItem[] = [
      { label: "Pelanggan aktif", value: String(metrics.total_active_customers), delta: `${metrics.customers_trend >= 0 ? "+" : ""}${metrics.customers_trend} bulan ini` },
      { label: "Pendapatan bulan ini", value: formatCurrency(metrics.monthly_revenue), delta: `ARPU ${formatCurrency(metrics.arpu)}` },
      { label: "Piutang", value: formatCurrency(metrics.total_receivables), delta: `${metrics.receivables_count} invoice`, tone: metrics.receivables_count > 0 ? ("amber" as const) : undefined },
      { label: "Arus kas bersih", value: formatCurrency(cashflowSummary.net_cashflow), delta: "Bulan ini", tone: cashflowSummary.net_cashflow >= 0 ? ("green" as const) : ("red" as const) },
    ];
    if (modules.mikrotik) {
      const online = routerSummary?.online_count ?? metrics.routers_online;
      const offline = routerSummary?.offline_count ?? metrics.routers_offline;
      const totalRouters = routerSummary?.total_routers ?? online + offline;
      items.push({ label: "Router online", value: `${online}/${totalRouters}`, delta: `${offline} offline`, tone: offline > 0 ? ("red" as const) : undefined });
    }
    return items;
  }, [cashflowSummary.net_cashflow, metrics, modules.mikrotik, routerSummary]);

  const oltTotal = oltSummary?.total ?? oltSummary?.total_olts ?? 0;
  const oltOnline = oltSummary?.online ?? oltSummary?.online_count ?? 0;
  const oltOffline = oltSummary?.offline ?? oltSummary?.offline_count ?? 0;
  const oltAlarms = oltSummary?.active_alarms ?? oltSummary?.active_alarm_count ?? 0;

  return (
    <AppShell>
      <div className="space-y-6">
        <PageHeader
          eyebrow="Dashboard"
          title="Pusat kontrol operasional"
          description="Ringkasan real dari billing-api dan network-service lokal."
          actions={
            <>
              <Button href="/customers/new">Tambah Pelanggan</Button>
              <button
                type="button"
                onClick={() => void loadDashboard()}
                disabled={loading}
                className="inline-flex min-w-0 items-center justify-center gap-2 rounded-md border border-slate-300 bg-white px-4 py-2 text-center text-sm font-semibold leading-5 text-slate-700 transition hover:bg-slate-50 disabled:cursor-wait disabled:opacity-60"
              >
                <ArrowClockwise size={16} />
                {loading ? "Memuat..." : "Refresh"}
              </button>
            </>
          }
        />

        {error && (
          <div className="flex gap-3 rounded-xl border border-red-200 bg-red-50 p-4 text-sm text-red-700">
            <WarningCircle size={20} className="shrink-0" />
            <span className="min-w-0 [overflow-wrap:anywhere]">{error}</span>
          </div>
        )}

        <StatGrid stats={stats} />

        <div className="grid gap-6 xl:grid-cols-[1.2fr_0.8fr]">
          <Section title="Pendapatan real" description="Data dari laporan dashboard billing-api.">
            {loading ? <EmptyState title="Memuat dashboard" description="Mengambil data dari billing-api..." /> : <RevenueSnapshot metrics={metrics} />}
          </Section>
          <Section title="Cashflow operasional" description="Kas masuk, kas keluar, dan saldo akhir bulan berjalan." action={<Button href="/cashflow" variant="secondary">Buka arus kas</Button>}>
            {loading ? <EmptyState title="Memuat arus kas" description="Mengambil ringkasan cashflow..." /> : <CashflowSnapshot summary={cashflowSummary} />}
          </Section>
        </div>

        {(modules.mikrotik || modules.fiber_network) && (
          <div className="grid gap-6 xl:grid-cols-[1.2fr_0.8fr]">
            <Section title="Status jaringan" description="Ringkasan add-on jaringan aktif.">
              <DataTable
                columns={["Metric", "Nilai", "Status"]}
                rows={[
                  ...(modules.mikrotik
                    ? [
                        ["Router terdaftar", String(routerSummary?.total_routers ?? 0), <StatusBadge key="router-total" status="info" />],
                        ["Router online", String(routerSummary?.online_count ?? metrics.routers_online), <StatusBadge key="router-online" status="online" />],
                        ["Router offline", String(routerSummary?.offline_count ?? metrics.routers_offline), <StatusBadge key="router-offline" status={(routerSummary?.offline_count ?? metrics.routers_offline) > 0 ? "offline" : "online"} />],
                      ]
                    : []),
                  ...(modules.fiber_network
                    ? [
                        ["OLT terdaftar", String(oltTotal), <StatusBadge key="olt-total" status="info" />],
                        ["OLT online", String(oltOnline), <StatusBadge key="olt-online" status="online" />],
                        ["OLT offline", String(oltOffline), <StatusBadge key="olt-offline" status={oltOffline > 0 ? "offline" : "online"} />],
                        ["Alarm OLT", String(oltAlarms), <StatusBadge key="olt-alarm" status={oltAlarms > 0 ? "warning" : "normal"} />],
                      ]
                    : []),
                ]}
              />
            </Section>
          </div>
        )}
      </div>
    </AppShell>
  );
}
