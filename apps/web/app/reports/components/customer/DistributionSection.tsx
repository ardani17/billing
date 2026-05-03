"use client";

import type { ReportFilter } from "../../lib/types";
import { fetchDistributionReport } from "../../lib/api";
import { formatNumber } from "../../lib/formatters";
import { useReportData } from "../../hooks/useReportData";
import { EmptyState } from "../shared/EmptyState";
import { PieChart } from "../charts/PieChart";

interface Props {
  filter: Partial<ReportFilter>;
}

export function DistributionSection({ filter }: Props) {
  const { data, loading, error } = useReportData({
    fetcher: () => fetchDistributionReport(filter),
  });

  if (loading) return <Skeleton />;
  if (error) return <ErrorMsg message={error} />;
  if (!data) return <EmptyState />;

  const { by_package, by_area, by_status, by_connection_method } = data;

  const statusItems = Object.entries(by_status).map(([name, count]) => ({
    name,
    value: count,
  }));

  return (
    <section className="space-y-4">
      <h2 className="text-lg font-semibold text-slate-900">Distribusi Pelanggan</h2>

      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        {by_package.length > 0 && (
          <div className="rounded-xl border border-slate-200 bg-white p-5">
            <h3 className="mb-4 text-sm font-medium text-slate-700">Per Paket</h3>
            <PieChart
              data={by_package.map((d) => ({ name: d.name, value: d.count }))}
              donut
              valueFormatter={(v) => formatNumber(v)}
              height={280}
            />
          </div>
        )}

        {by_area.length > 0 && (
          <div className="rounded-xl border border-slate-200 bg-white p-5">
            <h3 className="mb-4 text-sm font-medium text-slate-700">Per Area</h3>
            <PieChart
              data={by_area.map((d) => ({ name: d.name, value: d.count }))}
              donut
              valueFormatter={(v) => formatNumber(v)}
              height={280}
            />
          </div>
        )}

        {statusItems.length > 0 && (
          <div className="rounded-xl border border-slate-200 bg-white p-5">
            <h3 className="mb-4 text-sm font-medium text-slate-700">Per Status</h3>
            <PieChart
              data={statusItems}
              donut
              valueFormatter={(v) => formatNumber(v)}
              height={280}
            />
          </div>
        )}

        {by_connection_method.length > 0 && (
          <div className="rounded-xl border border-slate-200 bg-white p-5">
            <h3 className="mb-4 text-sm font-medium text-slate-700">Per Metode Koneksi</h3>
            <PieChart
              data={by_connection_method.map((d) => ({ name: d.name, value: d.count }))}
              donut
              valueFormatter={(v) => formatNumber(v)}
              height={280}
            />
          </div>
        )}
      </div>
    </section>
  );
}

function Skeleton() {
  return (
    <section className="space-y-4">
      <h2 className="text-lg font-semibold text-slate-900">Distribusi Pelanggan</h2>
      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        {[1, 2].map((i) => (
          <div key={i} className="h-72 animate-pulse rounded-xl border border-slate-200 bg-slate-50" />
        ))}
      </div>
    </section>
  );
}

function ErrorMsg({ message }: { message: string }) {
  return (
    <section className="space-y-4">
      <h2 className="text-lg font-semibold text-slate-900">Distribusi Pelanggan</h2>
      <div className="rounded-xl border border-red-200 bg-red-50 px-5 py-4 text-sm text-red-700">
        Gagal memuat data: {message}
      </div>
    </section>
  );
}
