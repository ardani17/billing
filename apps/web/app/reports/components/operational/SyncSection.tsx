"use client";

import type { ReportFilter } from "../../lib/types";
import { fetchSyncReport } from "../../lib/api";
import { formatNumber, formatPercentage } from "../../lib/formatters";
import { useReportData } from "../../hooks/useReportData";
import { MetricCard } from "../shared/MetricCard";
import { EmptyState } from "../shared/EmptyState";
import { ModuleInactive } from "../shared/ModuleInactive";

interface Props {
  filter: Partial<ReportFilter>;
}

export function SyncSection({ filter }: Props) {
  const { data, loading, error } = useReportData({
    fetcher: () => fetchSyncReport(filter),
  });

  if (loading) return <Skeleton />;
  if (error) return <ErrorMsg message={error} />;
  if (!data) return <EmptyState />;

  const mikrotikInactive = data.module_inactive?.mikrotik;
  const oltInactive = data.module_inactive?.fiber_network ?? data.module_inactive?.olt;

  if (mikrotikInactive && oltInactive) {
    return <ModuleInactive moduleName="MikroTik & OLT" />;
  }

  return (
    <section className="space-y-4">
      <h2 className="text-lg font-semibold text-slate-900">Sinkronisasi</h2>

      <MetricCard label="Sync Success Rate" value={formatPercentage(data.sync_success_rate)} />

      {/* MikroTik sync*/}
      {mikrotikInactive ? (
        <ModuleInactive moduleName="MikroTik" />
      ) : data.mikrotik_sync && data.mikrotik_sync.length > 0 ? (
        <div className="rounded-xl border border-slate-200 bg-white p-5">
          <h3 className="mb-3 text-sm font-medium text-slate-700">MikroTik Sync</h3>
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm">
              <thead>
                <tr className="border-b border-slate-200 text-xs text-slate-500">
                  <th className="pb-2 pr-4">Router</th>
                  <th className="pb-2 pr-4 text-right">OK</th>
                  <th className="pb-2 pr-4 text-right">Gagal</th>
                  <th className="pb-2 pr-4 text-right">Orphan</th>
                  <th className="pb-2 text-right">Pending</th>
                </tr>
              </thead>
              <tbody>
                {data.mikrotik_sync.map((r) => (
                  <tr key={r.router_id} className="border-b border-slate-100">
                    <td className="py-2 pr-4 text-slate-700">{r.router_name}</td>
                    <td className="py-2 pr-4 text-right text-emerald-600">{formatNumber(r.sync_ok_count)}</td>
                    <td className="py-2 pr-4 text-right text-red-600">{formatNumber(r.sync_failed_count)}</td>
                    <td className="py-2 pr-4 text-right text-amber-600">{formatNumber(r.orphan_user_count)}</td>
                    <td className="py-2 text-right text-slate-600">{formatNumber(r.pending_sync_count)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      ) : null}

      {/* OLT sync*/}
      {oltInactive ? (
        <ModuleInactive moduleName="OLT" />
      ) : data.olt_sync && data.olt_sync.length > 0 ? (
        <div className="rounded-xl border border-slate-200 bg-white p-5">
          <h3 className="mb-3 text-sm font-medium text-slate-700">OLT Sync</h3>
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm">
              <thead>
                <tr className="border-b border-slate-200 text-xs text-slate-500">
                  <th className="pb-2 pr-4">OLT</th>
                  <th className="pb-2 pr-4 text-right">OK</th>
                  <th className="pb-2 pr-4 text-right">Gagal</th>
                  <th className="pb-2 text-right">Unmanaged</th>
                </tr>
              </thead>
              <tbody>
                {data.olt_sync.map((o) => (
                  <tr key={o.olt_id} className="border-b border-slate-100">
                    <td className="py-2 pr-4 text-slate-700">{o.olt_name}</td>
                    <td className="py-2 pr-4 text-right text-emerald-600">{formatNumber(o.sync_ok_count)}</td>
                    <td className="py-2 pr-4 text-right text-red-600">{formatNumber(o.sync_failed_count)}</td>
                    <td className="py-2 text-right text-amber-600">{formatNumber(o.unmanaged_ont_count)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      ) : null}
    </section>
  );
}

function Skeleton() {
  return (
    <section className="space-y-4">
      <h2 className="text-lg font-semibold text-slate-900">Sinkronisasi</h2>
      <div className="h-48 animate-pulse rounded-xl border border-slate-200 bg-slate-50" />
    </section>
  );
}

function ErrorMsg({ message }: { message: string }) {
  return (
    <section className="space-y-4">
      <h2 className="text-lg font-semibold text-slate-900">Sinkronisasi</h2>
      <div className="rounded-xl border border-red-200 bg-red-50 px-5 py-4 text-sm text-red-700">
        Gagal memuat data: {message}
      </div>
    </section>
  );
}
