"use client";

import { fetchCapacityReport } from "../../lib/api";
import { formatNumber, formatPercentage } from "../../lib/formatters";
import { useReportData } from "../../hooks/useReportData";
import { EmptyState } from "../shared/EmptyState";
import { ModuleInactive } from "../shared/ModuleInactive";

export function CapacitySection() {
  const { data, loading, error } = useReportData({
    fetcher: () => fetchCapacityReport(),
  });

  if (loading) return <Skeleton />;
  if (error) return <ErrorMsg message={error} />;
  if (!data) return <EmptyState />;

  const mikrotikInactive = data.module_inactive?.mikrotik;
  const oltInactive = data.module_inactive?.fiber_network ?? data.module_inactive?.olt;

  return (
    <section className="space-y-4">
      <h2 className="text-lg font-semibold text-slate-900">Kapasitas</h2>

      {mikrotikInactive && oltInactive ? (
        <ModuleInactive moduleName="MikroTik & OLT" />
      ) : (
        <>
          {/* Router capacity*/}
          {mikrotikInactive ? (
            <ModuleInactive moduleName="MikroTik" />
          ) : data.router_capacity && data.router_capacity.length > 0 ? (
            <div className="rounded-xl border border-slate-200 bg-white p-5">
              <h3 className="mb-3 text-sm font-medium text-slate-700">Kapasitas Router</h3>
              <div className="overflow-x-auto">
                <table className="w-full text-left text-sm">
                  <thead>
                    <tr className="border-b border-slate-200 text-xs text-slate-500">
                      <th className="pb-2 pr-4">Router</th>
                      <th className="pb-2 pr-4 text-right">Pelanggan</th>
                      <th className="pb-2 pr-4 text-right">Maks</th>
                      <th className="pb-2 pr-4 text-right">Penggunaan</th>
                      <th className="pb-2">Est. Penuh</th>
                    </tr>
                  </thead>
                  <tbody>
                    {data.router_capacity.map((r) => (
                      <tr key={r.router_id} className="border-b border-slate-100">
                        <td className="py-2 pr-4 text-slate-700">{r.router_name}</td>
                        <td className="py-2 pr-4 text-right text-slate-600">{formatNumber(r.current_customers)}</td>
                        <td className="py-2 pr-4 text-right text-slate-600">{formatNumber(r.max_capacity)}</td>
                        <td className="py-2 pr-4 text-right">
                          <span className={`font-mono font-medium ${
                            r.usage_percentage >= 90 ? "text-red-600" :
                            r.usage_percentage >= 75 ? "text-amber-600" :
                            "text-slate-900"
                          }`}>
                            {formatPercentage(r.usage_percentage)}
                          </span>
                        </td>
                        <td className="py-2 text-slate-600">{r.estimated_full_date ?? "—"}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          ) : null}

          {/* ODP capacity*/}
          {oltInactive ? (
            <ModuleInactive moduleName="OLT" />
          ) : data.odp_capacity && data.odp_capacity.length > 0 ? (
            <div className="rounded-xl border border-slate-200 bg-white p-5">
              <h3 className="mb-3 text-sm font-medium text-slate-700">Kapasitas ODP</h3>
              <div className="overflow-x-auto">
                <table className="w-full text-left text-sm">
                  <thead>
                    <tr className="border-b border-slate-200 text-xs text-slate-500">
                      <th className="pb-2 pr-4">ODP</th>
                      <th className="pb-2 pr-4 text-right">Terpakai</th>
                      <th className="pb-2 pr-4 text-right">Total Port</th>
                      <th className="pb-2 pr-4 text-right">Penggunaan</th>
                      <th className="pb-2">Status</th>
                    </tr>
                  </thead>
                  <tbody>
                    {data.odp_capacity.map((o) => (
                      <tr key={o.odp_id} className="border-b border-slate-100">
                        <td className="py-2 pr-4 text-slate-700">{o.odp_name}</td>
                        <td className="py-2 pr-4 text-right text-slate-600">{o.used_ports}</td>
                        <td className="py-2 pr-4 text-right text-slate-600">{o.total_ports}</td>
                        <td className="py-2 pr-4 text-right font-mono text-slate-900">{formatPercentage(o.usage_percentage)}</td>
                        <td className="py-2">
                          <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${
                            o.status_label === "penuh" ? "bg-red-100 text-red-700" :
                            o.status_label === "hampir_penuh" ? "bg-amber-100 text-amber-700" :
                            "bg-emerald-100 text-emerald-700"
                          }`}>
                            {o.status_label}
                          </span>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          ) : null}

          {/* Rekomendasi*/}
          {data.recommendations.length > 0 && (
            <div className="rounded-xl border border-blue-200 bg-blue-50 p-5">
              <h3 className="mb-2 text-sm font-medium text-blue-800">Rekomendasi</h3>
              <ul className="list-inside list-disc space-y-1 text-sm text-blue-700">
                {data.recommendations.map((r, i) => (
                  <li key={i}>{r}</li>
                ))}
              </ul>
            </div>
          )}
        </>
      )}
    </section>
  );
}

function Skeleton() {
  return (
    <section className="space-y-4">
      <h2 className="text-lg font-semibold text-slate-900">Kapasitas</h2>
      <div className="h-48 animate-pulse rounded-xl border border-slate-200 bg-slate-50" />
    </section>
  );
}

function ErrorMsg({ message }: { message: string }) {
  return (
    <section className="space-y-4">
      <h2 className="text-lg font-semibold text-slate-900">Kapasitas</h2>
      <div className="rounded-xl border border-red-200 bg-red-50 px-5 py-4 text-sm text-red-700">
        Gagal memuat data: {message}
      </div>
    </section>
  );
}
