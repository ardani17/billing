"use client";

import type { ReportFilter } from "../../lib/types";
import { fetchRevenueByAreaReport } from "../../lib/api";
import { formatCurrency, formatNumber } from "../../lib/formatters";
import { useReportData } from "../../hooks/useReportData";
import { EmptyState } from "../shared/EmptyState";

interface Props {
  filter: Partial<ReportFilter>;
}

export function RevenueByAreaSection({ filter }: Props) {
  const { data, loading, error } = useReportData({
    fetcher: () => fetchRevenueByAreaReport(filter),
  });

  if (loading) return <Skeleton />;
  if (error) return <ErrorMsg message={error} />;
  if (!data || data.areas.length === 0) return <EmptyState />;

  const { areas, most_profitable_area, attention_needed_area } = data;

  return (
    <section className="space-y-4">
      <h2 className="text-lg font-semibold text-slate-900">Pendapatan per Area</h2>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        {most_profitable_area && (
          <div className="rounded-xl border border-emerald-200 bg-emerald-50 px-5 py-4">
            <p className="text-xs font-medium text-emerald-600">Area Paling Menguntungkan</p>
            <p className="mt-1 text-lg font-semibold text-emerald-900">{most_profitable_area}</p>
          </div>
        )}
        {attention_needed_area && (
          <div className="rounded-xl border border-amber-200 bg-amber-50 px-5 py-4">
            <p className="text-xs font-medium text-amber-600">Perlu Perhatian</p>
            <p className="mt-1 text-lg font-semibold text-amber-900">{attention_needed_area}</p>
          </div>
        )}
      </div>

      <div className="rounded-xl border border-slate-200 bg-white p-5">
        <div className="overflow-x-auto">
          <table className="w-full text-left text-sm">
            <thead>
              <tr className="border-b border-slate-200 text-xs text-slate-500">
                <th className="pb-2 pr-4">Area</th>
                <th className="pb-2 pr-4 text-right">Pelanggan</th>
                <th className="pb-2 pr-4 text-right">Pendapatan</th>
                <th className="pb-2 pr-4 text-right">Piutang</th>
                <th className="pb-2 text-right">ARPU</th>
              </tr>
            </thead>
            <tbody>
              {areas.map((a) => (
                <tr key={a.area_id} className="border-b border-slate-100">
                  <td className="py-2 pr-4 text-slate-700">{a.area_name}</td>
                  <td className="py-2 pr-4 text-right text-slate-600">{formatNumber(a.customer_count)}</td>
                  <td className="py-2 pr-4 text-right font-mono text-slate-900">{formatCurrency(a.total_revenue)}</td>
                  <td className="py-2 pr-4 text-right font-mono text-slate-600">{formatCurrency(a.total_outstanding)}</td>
                  <td className="py-2 text-right font-mono text-slate-600">{formatCurrency(a.arpu)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </section>
  );
}

function Skeleton() {
  return (
    <section className="space-y-4">
      <h2 className="text-lg font-semibold text-slate-900">Pendapatan per Area</h2>
      <div className="h-48 animate-pulse rounded-xl border border-slate-200 bg-slate-50" />
    </section>
  );
}

function ErrorMsg({ message }: { message: string }) {
  return (
    <section className="space-y-4">
      <h2 className="text-lg font-semibold text-slate-900">Pendapatan per Area</h2>
      <div className="rounded-xl border border-red-200 bg-red-50 px-5 py-4 text-sm text-red-700">
        Gagal memuat data: {message}
      </div>
    </section>
  );
}
