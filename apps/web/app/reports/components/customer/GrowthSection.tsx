"use client";

import type { ReportFilter } from "../../lib/types";
import { fetchCustomerGrowthReport } from "../../lib/api";
import { formatNumber, formatCurrency, formatPercentage, formatMonth } from "../../lib/formatters";
import { useReportData } from "../../hooks/useReportData";
import { MetricCard } from "../shared/MetricCard";
import { EmptyState } from "../shared/EmptyState";
import { LineChart } from "../charts/LineChart";

interface Props {
  filter: Partial<ReportFilter>;
}

export function GrowthSection({ filter }: Props) {
  const { data, loading, error } = useReportData({
    fetcher: () => fetchCustomerGrowthReport(filter),
  });

  if (loading) return <Skeleton />;
  if (error) return <ErrorMsg message={error} />;
  if (!data) return <EmptyState />;

  const { total_active, new_customers, churned_customers, net_growth, arpu, clv, churn_rate, trend, delta } = data;

  return (
    <section className="space-y-4">
      <h2 className="text-lg font-semibold text-slate-900">Pertumbuhan Pelanggan</h2>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <MetricCard
          label="Total Aktif"
          value={formatNumber(total_active)}
          delta={delta?.total_active?.percentage}
        />
        <MetricCard
          label="Pelanggan Baru"
          value={formatNumber(new_customers)}
          delta={delta?.new_customers?.percentage}
        />
        <MetricCard
          label="Churn"
          value={formatNumber(churned_customers)}
          delta={delta?.churned_customers?.percentage}
        />
        <MetricCard
          label="Net Growth"
          value={formatNumber(net_growth)}
          delta={delta?.net_growth?.percentage}
        />
      </div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
        <MetricCard label="ARPU" value={formatCurrency(arpu)} />
        <MetricCard label="CLV" value={formatCurrency(clv)} />
        <MetricCard label="Churn Rate" value={formatPercentage(churn_rate)} />
      </div>

      {trend.length > 0 && (
        <div className="rounded-xl border border-slate-200 bg-white p-5">
          <h3 className="mb-4 text-sm font-medium text-slate-700">Tren Pertumbuhan 12 Bulan</h3>
          <LineChart
            data={trend}
            xKey="month"
            xFormatter={formatMonth}
            valueFormatter={(v) => formatNumber(v)}
            lines={[
              { dataKey: "total_active", name: "Total Aktif", color: "#3b82f6" },
              { dataKey: "new_customers", name: "Baru", color: "#10b981" },
              { dataKey: "churned_customers", name: "Churn", color: "#ef4444" },
            ]}
          />
        </div>
      )}
    </section>
  );
}

function Skeleton() {
  return (
    <section className="space-y-4">
      <h2 className="text-lg font-semibold text-slate-900">Pertumbuhan Pelanggan</h2>
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {[1, 2, 3, 4].map((i) => (
          <div key={i} className="h-28 animate-pulse rounded-xl border border-slate-200 bg-slate-50" />
        ))}
      </div>
    </section>
  );
}

function ErrorMsg({ message }: { message: string }) {
  return (
    <section className="space-y-4">
      <h2 className="text-lg font-semibold text-slate-900">Pertumbuhan Pelanggan</h2>
      <div className="rounded-xl border border-red-200 bg-red-50 px-5 py-4 text-sm text-red-700">
        Gagal memuat data: {message}
      </div>
    </section>
  );
}
