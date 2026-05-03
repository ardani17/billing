"use client";

import type { ReportFilter } from "../../lib/types";
import { fetchAgingReport } from "../../lib/api";
import { formatCurrency, formatNumber, formatMonth } from "../../lib/formatters";
import { useReportData } from "../../hooks/useReportData";
import { MetricCard } from "../shared/MetricCard";
import { EmptyState } from "../shared/EmptyState";
import { ProgressBar } from "../shared/ProgressBar";
import { LineChart } from "../charts/LineChart";

interface Props {
  filter: Partial<ReportFilter>;
}

export function AgingSection({ filter }: Props) {
  const { data, loading, error } = useReportData({
    fetcher: () => fetchAgingReport(filter),
  });

  if (loading) return <Skeleton />;
  if (error) return <ErrorMsg message={error} />;
  if (!data) return <EmptyState />;

  const { buckets, total_outstanding, collection_rate, avg_days_to_pay, top_debtors, trend, kpi_target } = data;

  return (
    <section className="space-y-4">
      <h2 className="text-lg font-semibold text-slate-900">Aging Piutang</h2>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <MetricCard label="Total Piutang" value={formatCurrency(total_outstanding)} />
        <MetricCard label="Rata-rata Hari Bayar" value={`${avg_days_to_pay} hari`} />
        {buckets.map((b) => (
          <MetricCard key={b.label} label={b.label} value={formatCurrency(b.total_amount)} />
        ))}
      </div>

      <div className="rounded-xl border border-slate-200 bg-white p-5">
        <h3 className="mb-2 text-sm font-medium text-slate-700">Collection Rate</h3>
        <ProgressBar
          value={collection_rate}
          label={kpi_target ? `Target: ${kpi_target}%` : "Collection Rate"}
          showPercentage
        />
      </div>

      {top_debtors.length > 0 && (
        <div className="rounded-xl border border-slate-200 bg-white p-5">
          <h3 className="mb-3 text-sm font-medium text-slate-700">Top 10 Debitur</h3>
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm">
              <thead>
                <tr className="border-b border-slate-200 text-xs text-slate-500">
                  <th className="pb-2 pr-4">Pelanggan</th>
                  <th className="pb-2 pr-4 text-right">Tunggakan</th>
                  <th className="pb-2 text-right">Bulan</th>
                </tr>
              </thead>
              <tbody>
                {top_debtors.map((d) => (
                  <tr key={d.customer_id} className="border-b border-slate-100">
                    <td className="py-2 pr-4 text-slate-700">{d.customer_name}</td>
                    <td className="py-2 pr-4 text-right font-mono text-slate-900">{formatCurrency(d.total_outstanding)}</td>
                    <td className="py-2 text-right text-slate-600">{d.months_overdue}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {trend.length > 0 && (
        <div className="rounded-xl border border-slate-200 bg-white p-5">
          <h3 className="mb-4 text-sm font-medium text-slate-700">Tren Piutang 6 Bulan</h3>
          <LineChart
            data={trend}
            xKey="month"
            xFormatter={formatMonth}
            valueFormatter={(v) => formatCurrency(v)}
            lines={[{ dataKey: "total_outstanding", name: "Total Piutang", color: "#ef4444" }]}
          />
        </div>
      )}
    </section>
  );
}

function Skeleton() {
  return (
    <section className="space-y-4">
      <h2 className="text-lg font-semibold text-slate-900">Aging Piutang</h2>
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
      <h2 className="text-lg font-semibold text-slate-900">Aging Piutang</h2>
      <div className="rounded-xl border border-red-200 bg-red-50 px-5 py-4 text-sm text-red-700">
        Gagal memuat data: {message}
      </div>
    </section>
  );
}
