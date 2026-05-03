"use client";

import { useDashboard } from "../hooks/useDashboard";
import { formatCurrency, formatNumber, formatPercentage } from "../lib/formatters";
import { MetricCard } from "./shared/MetricCard";
import { ProgressBar } from "./shared/ProgressBar";

interface DashboardWidgetProps {
  onNavigate?: (section: string) => void;
}

export function DashboardWidget({ onNavigate }: DashboardWidgetProps) {
  const { data, loading, error } = useDashboard();

  if (loading) {
    return (
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {[1, 2, 3, 4, 5, 6, 7].map((i) => (
          <div key={i} className="h-28 animate-pulse rounded-xl border border-slate-200 bg-slate-50" />
        ))}
      </div>
    );
  }

  if (error) {
    return (
      <div className="rounded-xl border border-red-200 bg-red-50 px-5 py-4 text-sm text-red-700">
        Gagal memuat data dashboard: {error}
      </div>
    );
  }

  if (!data) return null;

  const handleClick = (section: string) => {
    onNavigate?.(section);
  };

  return (
    <div className="space-y-4">
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <button
          type="button"
          onClick={() => handleClick("pelanggan")}
          className="text-left"
          style={{ minHeight: 44 }}
        >
          <MetricCard
            label="Total Pelanggan Aktif"
            value={formatNumber(data.total_active_customers)}
            delta={data.customers_trend}
          />
        </button>

        <button
          type="button"
          onClick={() => handleClick("keuangan")}
          className="text-left"
          style={{ minHeight: 44 }}
        >
          <MetricCard
            label="Pendapatan Bulan Ini"
            value={formatCurrency(data.monthly_revenue)}
            kpiProgress={data.revenue_progress}
            kpiLabel={data.revenue_target ? `Target: ${formatCurrency(data.revenue_target)}` : undefined}
          />
        </button>

        <button
          type="button"
          onClick={() => handleClick("keuangan")}
          className="text-left"
          style={{ minHeight: 44 }}
        >
          <MetricCard
            label="Tunggakan"
            value={formatCurrency(data.total_receivables)}
            kpiLabel={`${formatNumber(data.receivables_count)} pelanggan`}
          />
        </button>

        <button
          type="button"
          onClick={() => handleClick("jaringan")}
          className="text-left"
          style={{ minHeight: 44 }}
        >
          <div className="rounded-xl border border-slate-200 bg-white p-5">
            <p className="text-sm text-slate-500">Router</p>
            <div className="mt-2 flex items-end gap-3">
              <span className="font-mono text-2xl font-semibold text-emerald-600">{data.routers_online}</span>
              <span className="text-sm text-slate-400">online</span>
              <span className="font-mono text-lg text-red-500">{data.routers_offline}</span>
              <span className="text-sm text-slate-400">offline</span>
            </div>
          </div>
        </button>
      </div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
        <button
          type="button"
          onClick={() => handleClick("keuangan")}
          className="text-left"
          style={{ minHeight: 44 }}
        >
          <div className="rounded-xl border border-slate-200 bg-white p-5">
            <p className="mb-2 text-sm text-slate-500">Collection Rate</p>
            <ProgressBar
              value={data.collection_rate}
              label={data.collection_target ? `Target: ${formatPercentage(data.collection_target)}` : undefined}
              showPercentage
            />
          </div>
        </button>

        <button
          type="button"
          onClick={() => handleClick("pelanggan")}
          className="text-left"
          style={{ minHeight: 44 }}
        >
          <div className="rounded-xl border border-slate-200 bg-white p-5">
            <p className="mb-2 text-sm text-slate-500">Churn Rate</p>
            <div className="flex items-end justify-between">
              <span className="font-mono text-2xl font-semibold text-slate-900">
                {formatPercentage(data.churn_rate)}
              </span>
              {data.churn_target && (
                <span className="text-xs text-slate-500">
                  Maks: {formatPercentage(data.churn_target)}
                </span>
              )}
            </div>
          </div>
        </button>

        <button
          type="button"
          onClick={() => handleClick("pelanggan")}
          className="text-left"
          style={{ minHeight: 44 }}
        >
          <MetricCard label="ARPU" value={formatCurrency(data.arpu)} />
        </button>
      </div>
    </div>
  );
}
