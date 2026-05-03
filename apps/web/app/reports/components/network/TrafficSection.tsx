"use client";

import type { ReportFilter } from "../../lib/types";
import { fetchTrafficReport } from "../../lib/api";
import { formatBytes, formatNumber } from "../../lib/formatters";
import { useReportData } from "../../hooks/useReportData";
import { MetricCard } from "../shared/MetricCard";
import { EmptyState } from "../shared/EmptyState";
import { ModuleInactive } from "../shared/ModuleInactive";
import { BarChart } from "../charts/BarChart";

interface Props {
  filter: Partial<ReportFilter>;
}

export function TrafficSection({ filter }: Props) {
  const { data, loading, error } = useReportData({
    fetcher: () => fetchTrafficReport(filter),
  });

  if (loading) return <Skeleton />;
  if (error) return <ErrorMsg message={error} />;
  if (!data) return <EmptyState />;
  if (data.module_inactive) return <ModuleInactive moduleName="MikroTik" />;

  const { total_download_bytes, total_upload_bytes, total_traffic_bytes, peak_traffic_bps, by_router, top_customers } = data;

  return (
    <section className="space-y-4">
      <h2 className="text-lg font-semibold text-slate-900">Traffic Jaringan</h2>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <MetricCard label="Total Traffic" value={formatBytes(total_traffic_bytes)} />
        <MetricCard label="Download" value={formatBytes(total_download_bytes)} />
        <MetricCard label="Upload" value={formatBytes(total_upload_bytes)} />
        <MetricCard label="Peak Traffic" value={formatBytes(peak_traffic_bps) + "/s"} />
      </div>

      {by_router.length > 0 && (
        <div className="rounded-xl border border-slate-200 bg-white p-5">
          <h3 className="mb-4 text-sm font-medium text-slate-700">Traffic per Router</h3>
          <BarChart
            data={by_router}
            xKey="router_name"
            bars={[
              { dataKey: "download_bytes", name: "Download", color: "#3b82f6" },
              { dataKey: "upload_bytes", name: "Upload", color: "#10b981" },
            ]}
            valueFormatter={(v) => formatBytes(v)}
          />
        </div>
      )}

      {top_customers.length > 0 && (
        <div className="rounded-xl border border-slate-200 bg-white p-5">
          <h3 className="mb-3 text-sm font-medium text-slate-700">Top 10 Pelanggan</h3>
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm">
              <thead>
                <tr className="border-b border-slate-200 text-xs text-slate-500">
                  <th className="pb-2 pr-4">Pelanggan</th>
                  <th className="pb-2 pr-4">Paket</th>
                  <th className="pb-2 pr-4 text-right">Download</th>
                  <th className="pb-2 pr-4 text-right">Upload</th>
                  <th className="pb-2">Flag</th>
                </tr>
              </thead>
              <tbody>
                {top_customers.map((c) => (
                  <tr key={c.customer_id} className="border-b border-slate-100">
                    <td className="py-2 pr-4 text-slate-700">{c.customer_name}</td>
                    <td className="py-2 pr-4 text-slate-600">{c.package_name}</td>
                    <td className="py-2 pr-4 text-right font-mono text-slate-900">{formatBytes(c.download_bytes)}</td>
                    <td className="py-2 pr-4 text-right font-mono text-slate-600">{formatBytes(c.upload_bytes)}</td>
                    <td className="py-2">
                      {c.over_use_flag && (
                        <span className="inline-flex rounded-full bg-amber-100 px-2 py-0.5 text-xs font-medium text-amber-700">
                          Over-use
                        </span>
                      )}
                    </td>
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
      <h2 className="text-lg font-semibold text-slate-900">Traffic Jaringan</h2>
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
      <h2 className="text-lg font-semibold text-slate-900">Traffic Jaringan</h2>
      <div className="rounded-xl border border-red-200 bg-red-50 px-5 py-4 text-sm text-red-700">
        Gagal memuat data: {message}
      </div>
    </section>
  );
}
