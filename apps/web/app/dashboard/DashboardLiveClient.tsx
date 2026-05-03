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

type ApiResponse<T> = {
  success: boolean;
  data?: T;
  error?: {
    code: string;
    message: string;
  };
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

export default function DashboardLiveClient() {
  const [metrics, setMetrics] = useState<DashboardMetrics>(emptyMetrics);
  const [routerSummary, setRouterSummary] = useState<RouterSummary | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  async function loadDashboard() {
    setLoading(true);
    setError("");
    try {
      const [dashboardResponse, routerResponse] = await Promise.all([
        fetch("/api/billing/reports/dashboard", { cache: "no-store" }),
        fetch("/api/network/mikrotik/status/summary", { cache: "no-store" }),
      ]);
      const dashboardJson = (await dashboardResponse.json()) as ApiResponse<DashboardMetrics>;
      const routerJson = (await routerResponse.json()) as ApiResponse<RouterSummary>;

      if (!dashboardResponse.ok || !dashboardJson.success) {
        throw new Error(dashboardJson.error?.message || "Gagal mengambil dashboard");
      }
      setMetrics(dashboardJson.data || emptyMetrics);
      if (routerResponse.ok && routerJson.success) setRouterSummary(routerJson.data || null);
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
    const online = routerSummary?.online_count ?? metrics.routers_online;
    const offline = routerSummary?.offline_count ?? metrics.routers_offline;
    const totalRouters = routerSummary?.total_routers ?? online + offline;
    return [
      { label: "Pelanggan aktif", value: String(metrics.total_active_customers), delta: `${metrics.customers_trend >= 0 ? "+" : ""}${metrics.customers_trend} bulan ini` },
      { label: "Pendapatan bulan ini", value: formatCurrency(metrics.monthly_revenue), delta: `ARPU ${formatCurrency(metrics.arpu)}` },
      { label: "Piutang", value: formatCurrency(metrics.total_receivables), delta: `${metrics.receivables_count} invoice`, tone: metrics.receivables_count > 0 ? ("amber" as const) : undefined },
      { label: "Router online", value: `${online}/${totalRouters}`, delta: `${offline} offline`, tone: offline > 0 ? ("red" as const) : undefined },
    ];
  }, [metrics, routerSummary]);

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
          <Section title="Status jaringan" description="Ringkasan router dari network-service tanpa dial RouterOS.">
            <DataTable
              columns={["Metric", "Nilai", "Status"]}
              rows={[
                ["Router terdaftar", String(routerSummary?.total_routers ?? 0), <StatusBadge key="total" status="info" />],
                ["Online", String(routerSummary?.online_count ?? metrics.routers_online), <StatusBadge key="online" status="online" />],
                ["Offline", String(routerSummary?.offline_count ?? metrics.routers_offline), <StatusBadge key="offline" status={(routerSummary?.offline_count ?? metrics.routers_offline) > 0 ? "offline" : "online"} />],
                ["Maintenance", String(routerSummary?.maintenance_count ?? 0), <StatusBadge key="maintenance" status="maintenance" />],
              ]}
            />
          </Section>
        </div>
      </div>
    </AppShell>
  );
}
