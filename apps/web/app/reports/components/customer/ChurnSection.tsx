"use client";

import type { ReportFilter } from "../../lib/types";
import { fetchChurnReport } from "../../lib/api";
import { formatNumber, formatPercentage } from "../../lib/formatters";
import { useReportData } from "../../hooks/useReportData";
import { MetricCard } from "../shared/MetricCard";
import { EmptyState } from "../shared/EmptyState";
import { BarChart } from "../charts/BarChart";

interface Props {
  filter: Partial<ReportFilter>;
}

export function ChurnSection({ filter }: Props) {
  const { data, loading, error } = useReportData({
    fetcher: () => fetchChurnReport(filter),
  });

  if (loading) return <Skeleton />;
  if (error) return <ErrorMsg message={error} />;
  if (!data) return <EmptyState />;

  const { churned_count, churn_rate, by_reason, by_package, by_area, average_lifetime_months } = data;

  return (
    <section className="space-y-4">
      <h2 className="text-lg font-semibold text-slate-900">Analisis Churn</h2>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
        <MetricCard label="Pelanggan Churn" value={formatNumber(churned_count)} />
        <MetricCard label="Churn Rate" value={formatPercentage(churn_rate)} />
        <MetricCard label="Rata-rata Masa Berlangganan" value={`${average_lifetime_months.toFixed(1)} bulan`} />
      </div>

      {by_reason.length > 0 && (
        <div className="rounded-xl border border-slate-200 bg-white p-5">
          <h3 className="mb-4 text-sm font-medium text-slate-700">Alasan Churn</h3>
          <BarChart
            data={by_reason}
            xKey="reason"
            bars={[{ dataKey: "count", name: "Jumlah", color: "#ef4444" }]}
            valueFormatter={(v) => formatNumber(v)}
          />
        </div>
      )}

      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        {by_package.length > 0 && (
          <div className="rounded-xl border border-slate-200 bg-white p-5">
            <h3 className="mb-3 text-sm font-medium text-slate-700">Per Paket</h3>
            <div className="overflow-x-auto">
              <table className="w-full text-left text-sm">
                <thead>
                  <tr className="border-b border-slate-200 text-xs text-slate-500">
                    <th className="pb-2 pr-4">Paket</th>
                    <th className="pb-2 pr-4 text-right">Jumlah</th>
                    <th className="pb-2 text-right">%</th>
                  </tr>
                </thead>
                <tbody>
                  {by_package.map((p) => (
                    <tr key={p.name} className="border-b border-slate-100">
                      <td className="py-2 pr-4 text-slate-700">{p.name}</td>
                      <td className="py-2 pr-4 text-right text-slate-600">{formatNumber(p.count)}</td>
                      <td className="py-2 text-right text-slate-600">{formatPercentage(p.percentage)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}

        {by_area.length > 0 && (
          <div className="rounded-xl border border-slate-200 bg-white p-5">
            <h3 className="mb-3 text-sm font-medium text-slate-700">Per Area</h3>
            <div className="overflow-x-auto">
              <table className="w-full text-left text-sm">
                <thead>
                  <tr className="border-b border-slate-200 text-xs text-slate-500">
                    <th className="pb-2 pr-4">Area</th>
                    <th className="pb-2 pr-4 text-right">Jumlah</th>
                    <th className="pb-2 text-right">%</th>
                  </tr>
                </thead>
                <tbody>
                  {by_area.map((a) => (
                    <tr key={a.name} className="border-b border-slate-100">
                      <td className="py-2 pr-4 text-slate-700">{a.name}</td>
                      <td className="py-2 pr-4 text-right text-slate-600">{formatNumber(a.count)}</td>
                      <td className="py-2 text-right text-slate-600">{formatPercentage(a.percentage)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}
      </div>
    </section>
  );
}

function Skeleton() {
  return (
    <section className="space-y-4">
      <h2 className="text-lg font-semibold text-slate-900">Analisis Churn</h2>
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
        {[1, 2, 3].map((i) => (
          <div key={i} className="h-28 animate-pulse rounded-xl border border-slate-200 bg-slate-50" />
        ))}
      </div>
    </section>
  );
}

function ErrorMsg({ message }: { message: string }) {
  return (
    <section className="space-y-4">
      <h2 className="text-lg font-semibold text-slate-900">Analisis Churn</h2>
      <div className="rounded-xl border border-red-200 bg-red-50 px-5 py-4 text-sm text-red-700">
        Gagal memuat data: {message}
      </div>
    </section>
  );
}
