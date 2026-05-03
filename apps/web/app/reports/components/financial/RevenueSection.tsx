"use client";

import type { ReportFilter } from "../../lib/types";
import { fetchRevenueReport } from "../../lib/api";
import { formatCurrency, formatMonth } from "../../lib/formatters";
import { useReportData } from "../../hooks/useReportData";
import { MetricCard } from "../shared/MetricCard";
import { EmptyState } from "../shared/EmptyState";
import { BarChart } from "../charts/BarChart";

interface Props {
  filter: Partial<ReportFilter>;
}

export function RevenueSection({ filter }: Props) {
  const { data, loading, error } = useReportData({
    fetcher: () => fetchRevenueReport(filter),
  });

  if (loading) {
    return <SectionSkeleton title="Pendapatan" />;
  }
  if (error) {
    return <SectionError title="Pendapatan" message={error} />;
  }
  if (!data) {
    return <EmptyState />;
  }

  const { current, delta, trend, kpi_target, kpi_progress } = data;

  return (
    <section className="space-y-4">
      <h2 className="text-lg font-semibold text-slate-900">Pendapatan</h2>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <MetricCard
          label="Total Pendapatan"
          value={formatCurrency(current.total)}
          delta={delta?.total?.percentage}
          kpiProgress={kpi_progress}
          kpiLabel={kpi_target ? `Target: ${formatCurrency(kpi_target)}` : undefined}
        />
        <MetricCard
          label="Langganan Bulanan"
          value={formatCurrency(current.monthly_subscription)}
          delta={delta?.monthly_subscription?.percentage}
        />
        <MetricCard
          label="Penjualan Voucher"
          value={formatCurrency(current.voucher_sales)}
          delta={delta?.voucher_sales?.percentage}
        />
        <MetricCard
          label="Pendapatan Lainnya"
          value={formatCurrency(current.installation_fees + current.late_fees + current.other)}
        />
      </div>

      {trend.length > 0 && (
        <div className="rounded-xl border border-slate-200 bg-white p-5">
          <h3 className="mb-4 text-sm font-medium text-slate-700">Tren Pendapatan 12 Bulan</h3>
          <BarChart
            data={trend}
            xKey="month"
            xFormatter={formatMonth}
            valueFormatter={(v) => formatCurrency(v)}
            bars={[
              { dataKey: "monthly_subscription", name: "Langganan", color: "#3b82f6", stackId: "rev" },
              { dataKey: "voucher_sales", name: "Voucher", color: "#10b981", stackId: "rev" },
              { dataKey: "other_revenue", name: "Lainnya", color: "#f59e0b", stackId: "rev" },
            ]}
          />
        </div>
      )}
    </section>
  );
}

function SectionSkeleton({ title }: { title: string }) {
  return (
    <section className="space-y-4">
      <h2 className="text-lg font-semibold text-slate-900">{title}</h2>
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {[1, 2, 3, 4].map((i) => (
          <div key={i} className="h-28 animate-pulse rounded-xl border border-slate-200 bg-slate-50" />
        ))}
      </div>
    </section>
  );
}

function SectionError({ title, message }: { title: string; message: string }) {
  return (
    <section className="space-y-4">
      <h2 className="text-lg font-semibold text-slate-900">{title}</h2>
      <div className="rounded-xl border border-red-200 bg-red-50 px-5 py-4 text-sm text-red-700">
        Gagal memuat data: {message}
      </div>
    </section>
  );
}
