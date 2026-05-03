"use client";

import type { ReportFilter } from "../../lib/types";
import { fetchSignalReport } from "../../lib/api";
import { formatNumber } from "../../lib/formatters";
import { useReportData } from "../../hooks/useReportData";
import { MetricCard } from "../shared/MetricCard";
import { EmptyState } from "../shared/EmptyState";
import { ModuleInactive } from "../shared/ModuleInactive";

interface Props {
  filter: Partial<ReportFilter>;
}

export function SignalSection({ filter }: Props) {
  const { data, loading, error } = useReportData({
    fetcher: () => fetchSignalReport(filter),
  });

  if (loading) return <Skeleton />;
  if (error) return <ErrorMsg message={error} />;
  if (!data) return <EmptyState />;
  if (data.module_inactive) return <ModuleInactive moduleName="OLT" />;

  const { normal_count, warning_count, weak_count, critical_count, total_ont_count, average_signal_dbm, degrading_onts, alarm_summary } = data;

  return (
    <section className="space-y-4">
      <h2 className="text-lg font-semibold text-slate-900">Kualitas Sinyal</h2>

      <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-6">
        <MetricCard label="Total ONT" value={formatNumber(total_ont_count)} />
        <MetricCard label="Normal" value={formatNumber(normal_count)} />
        <MetricCard label="Warning" value={formatNumber(warning_count)} />
        <MetricCard label="Lemah" value={formatNumber(weak_count)} />
        <MetricCard label="Kritis" value={formatNumber(critical_count)} />
        <MetricCard label="Rata-rata Signal" value={`${average_signal_dbm.toFixed(1)} dBm`} />
      </div>

      {degrading_onts.length > 0 && (
        <div className="rounded-xl border border-slate-200 bg-white p-5">
          <h3 className="mb-3 text-sm font-medium text-slate-700">ONT dengan Sinyal Menurun</h3>
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm">
              <thead>
                <tr className="border-b border-slate-200 text-xs text-slate-500">
                  <th className="pb-2 pr-4">Pelanggan</th>
                  <th className="pb-2 pr-4 text-right">Signal (dBm)</th>
                  <th className="pb-2 text-right">Perubahan (dB)</th>
                </tr>
              </thead>
              <tbody>
                {degrading_onts.map((ont) => (
                  <tr key={ont.customer_id} className="border-b border-slate-100">
                    <td className="py-2 pr-4 text-slate-700">{ont.customer_name}</td>
                    <td className="py-2 pr-4 text-right font-mono text-slate-900">{ont.current_signal_dbm.toFixed(1)}</td>
                    <td className="py-2 text-right font-mono text-red-600">{ont.signal_change_db.toFixed(1)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {alarm_summary.length > 0 && (
        <div className="rounded-xl border border-slate-200 bg-white p-5">
          <h3 className="mb-3 text-sm font-medium text-slate-700">Ringkasan Alarm</h3>
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm">
              <thead>
                <tr className="border-b border-slate-200 text-xs text-slate-500">
                  <th className="pb-2 pr-4">Tipe Alarm</th>
                  <th className="pb-2 pr-4 text-right">Jumlah</th>
                  <th className="pb-2 pr-4 text-right">Rata-rata Durasi</th>
                  <th className="pb-2 text-right">Resolved</th>
                </tr>
              </thead>
              <tbody>
                {alarm_summary.map((a) => (
                  <tr key={a.alarm_type} className="border-b border-slate-100">
                    <td className="py-2 pr-4 text-slate-700">{a.alarm_type}</td>
                    <td className="py-2 pr-4 text-right text-slate-600">{formatNumber(a.count)}</td>
                    <td className="py-2 pr-4 text-right text-slate-600">{a.avg_duration_minutes.toFixed(0)} menit</td>
                    <td className="py-2 text-right text-slate-600">{a.resolved_percentage.toFixed(0)}%</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </section>
  );
}

function Skeleton() {
  return (
    <section className="space-y-4">
      <h2 className="text-lg font-semibold text-slate-900">Kualitas Sinyal</h2>
      <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-6">
        {[1, 2, 3, 4, 5, 6].map((i) => (
          <div key={i} className="h-28 animate-pulse rounded-xl border border-slate-200 bg-slate-50" />
        ))}
      </div>
    </section>
  );
}

function ErrorMsg({ message }: { message: string }) {
  return (
    <section className="space-y-4">
      <h2 className="text-lg font-semibold text-slate-900">Kualitas Sinyal</h2>
      <div className="rounded-xl border border-red-200 bg-red-50 px-5 py-4 text-sm text-red-700">
        Gagal memuat data: {message}
      </div>
    </section>
  );
}
