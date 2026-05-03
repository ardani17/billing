"use client";

import type { ReportFilter } from "../../lib/types";
import { fetchNotificationReport } from "../../lib/api";
import { formatNumber, formatCurrency, formatPercentage } from "../../lib/formatters";
import { useReportData } from "../../hooks/useReportData";
import { MetricCard } from "../shared/MetricCard";
import { EmptyState } from "../shared/EmptyState";
import { ModuleInactive } from "../shared/ModuleInactive";

interface Props {
  filter: Partial<ReportFilter>;
}

export function NotificationSection({ filter }: Props) {
  const { data, loading, error } = useReportData({
    fetcher: () => fetchNotificationReport(filter),
  });

  if (loading) return <Skeleton />;
  if (error) return <ErrorMsg message={error} />;
  if (!data) return <EmptyState />;
  if (data.module_inactive) return <ModuleInactive moduleName="Notifikasi" />;

  const { total_sent, total_delivered, total_failed, success_rate, total_cost, per_channel, per_template } = data;

  return (
    <section className="space-y-4">
      <h2 className="text-lg font-semibold text-slate-900">Laporan Notifikasi</h2>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-5">
        <MetricCard label="Total Terkirim" value={formatNumber(total_sent)} />
        <MetricCard label="Delivered" value={formatNumber(total_delivered)} />
        <MetricCard label="Gagal" value={formatNumber(total_failed)} />
        <MetricCard label="Success Rate" value={formatPercentage(success_rate)} />
        <MetricCard label="Total Biaya" value={formatCurrency(total_cost)} />
      </div>

      {per_channel.length > 0 && (
        <div className="rounded-xl border border-slate-200 bg-white p-5">
          <h3 className="mb-3 text-sm font-medium text-slate-700">Per Channel</h3>
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm">
              <thead>
                <tr className="border-b border-slate-200 text-xs text-slate-500">
                  <th className="pb-2 pr-4">Channel</th>
                  <th className="pb-2 pr-4 text-right">Terkirim</th>
                  <th className="pb-2 pr-4 text-right">Delivered</th>
                  <th className="pb-2 pr-4 text-right">Gagal</th>
                  <th className="pb-2 pr-4 text-right">Success Rate</th>
                  <th className="pb-2 text-right">Biaya</th>
                </tr>
              </thead>
              <tbody>
                {per_channel.map((c) => (
                  <tr key={c.channel} className="border-b border-slate-100">
                    <td className="py-2 pr-4 text-slate-700">{c.channel}</td>
                    <td className="py-2 pr-4 text-right text-slate-600">{formatNumber(c.sent_count)}</td>
                    <td className="py-2 pr-4 text-right text-slate-600">{formatNumber(c.delivered_count)}</td>
                    <td className="py-2 pr-4 text-right text-slate-600">{formatNumber(c.failed_count)}</td>
                    <td className="py-2 pr-4 text-right font-mono text-slate-900">{formatPercentage(c.success_rate)}</td>
                    <td className="py-2 text-right font-mono text-slate-600">{formatCurrency(c.cost)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {per_template.length > 0 && (
        <div className="rounded-xl border border-slate-200 bg-white p-5">
          <h3 className="mb-3 text-sm font-medium text-slate-700">Per Template</h3>
          <div className="space-y-2">
            {per_template.map((t) => (
              <div key={t.template_name} className="flex items-center justify-between text-sm">
                <span className="text-slate-700">{t.template_name}</span>
                <span className="font-mono text-slate-900">{formatNumber(t.sent_count)}</span>
              </div>
            ))}
          </div>
        </div>
      )}
    </section>
  );
}

function Skeleton() {
  return (
    <section className="space-y-4">
      <h2 className="text-lg font-semibold text-slate-900">Laporan Notifikasi</h2>
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-5">
        {[1, 2, 3, 4, 5].map((i) => (
          <div key={i} className="h-28 animate-pulse rounded-xl border border-slate-200 bg-slate-50" />
        ))}
      </div>
    </section>
  );
}

function ErrorMsg({ message }: { message: string }) {
  return (
    <section className="space-y-4">
      <h2 className="text-lg font-semibold text-slate-900">Laporan Notifikasi</h2>
      <div className="rounded-xl border border-red-200 bg-red-50 px-5 py-4 text-sm text-red-700">
        Gagal memuat data: {message}
      </div>
    </section>
  );
}
