"use client";

import type { ReportFilter } from "../../lib/types";
import { fetchUptimeReport } from "../../lib/api";
import { formatPercentage, formatNumber } from "../../lib/formatters";
import { useReportData } from "../../hooks/useReportData";
import { EmptyState } from "../shared/EmptyState";
import { ModuleInactive } from "../shared/ModuleInactive";
import { StaleDataBanner } from "../shared/StaleDataBanner";

interface Props {
  filter: Partial<ReportFilter>;
}

export function UptimeSection({ filter }: Props) {
  const { data, loading, error } = useReportData({
    fetcher: () => fetchUptimeReport(filter),
  });

  if (loading) return <Skeleton />;
  if (error) return <ErrorMsg message={error} />;
  if (!data) return <EmptyState />;
  if (data.module_inactive) return <ModuleInactive moduleName="MikroTik" />;

  const { routers, sla_target, routers_below_sla, stale_data, last_updated } = data;

  return (
    <section className="space-y-4">
      <h2 className="text-lg font-semibold text-slate-900">Uptime Router</h2>

      {stale_data && <StaleDataBanner lastUpdated={last_updated} />}

      {sla_target && (
        <div className="rounded-xl border border-blue-200 bg-blue-50 px-5 py-3 text-sm text-blue-700">
          Target SLA: {formatPercentage(sla_target)}
          {routers_below_sla && routers_below_sla.length > 0 && (
            <span className="ml-2 font-medium text-red-600">
              — {routers_below_sla.length} router di bawah SLA
            </span>
          )}
        </div>
      )}

      {routers.length > 0 ? (
        <div className="rounded-xl border border-slate-200 bg-white p-5">
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm">
              <thead>
                <tr className="border-b border-slate-200 text-xs text-slate-500">
                  <th className="pb-2 pr-4">Router</th>
                  <th className="pb-2 pr-4 text-right">Uptime</th>
                  <th className="pb-2 pr-4 text-right">Downtime (menit)</th>
                  <th className="pb-2 pr-4 text-right">Reboot</th>
                  <th className="pb-2">Status</th>
                </tr>
              </thead>
              <tbody>
                {routers.map((r) => {
                  const belowSla = sla_target ? r.uptime_percentage < sla_target : false;
                  return (
                    <tr
                      key={r.router_id}
                      className={`border-b border-slate-100 ${belowSla ? "bg-red-50" : ""}`}
                    >
                      <td className="py-2 pr-4 text-slate-700">{r.router_name}</td>
                      <td className={`py-2 pr-4 text-right font-mono ${belowSla ? "text-red-600 font-semibold" : "text-slate-900"}`}>
                        {formatPercentage(r.uptime_percentage)}
                      </td>
                      <td className="py-2 pr-4 text-right text-slate-600">{formatNumber(r.total_downtime_minutes)}</td>
                      <td className="py-2 pr-4 text-right text-slate-600">{r.reboot_count}</td>
                      <td className="py-2">
                        <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${
                          r.status_label === "baik" ? "bg-emerald-100 text-emerald-700" :
                          r.status_label === "perhatian" ? "bg-amber-100 text-amber-700" :
                          "bg-red-100 text-red-700"
                        }`}>
                          {r.status_label}
                        </span>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </div>
      ) : (
        <EmptyState />
      )}
    </section>
  );
}

function Skeleton() {
  return (
    <section className="space-y-4">
      <h2 className="text-lg font-semibold text-slate-900">Uptime Router</h2>
      <div className="h-48 animate-pulse rounded-xl border border-slate-200 bg-slate-50" />
    </section>
  );
}

function ErrorMsg({ message }: { message: string }) {
  return (
    <section className="space-y-4">
      <h2 className="text-lg font-semibold text-slate-900">Uptime Router</h2>
      <div className="rounded-xl border border-red-200 bg-red-50 px-5 py-4 text-sm text-red-700">
        Gagal memuat data: {message}
      </div>
    </section>
  );
}
