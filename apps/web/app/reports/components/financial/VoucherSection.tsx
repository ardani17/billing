"use client";

import type { ReportFilter } from "../../lib/types";
import { fetchVoucherReport } from "../../lib/api";
import { formatCurrency, formatNumber, formatPercentage } from "../../lib/formatters";
import { useReportData } from "../../hooks/useReportData";
import { MetricCard } from "../shared/MetricCard";
import { EmptyState } from "../shared/EmptyState";

interface Props {
  filter: Partial<ReportFilter>;
}

export function VoucherSection({ filter }: Props) {
  const { data, loading, error } = useReportData({
    fetcher: () => fetchVoucherReport(filter),
  });

  if (loading) return <Skeleton />;
  if (error) return <ErrorMsg message={error} />;
  if (!data) return <EmptyState />;

  const { total_revenue, total_voucher_count, by_package, by_reseller, total_reseller_margin } = data;

  return (
    <section className="space-y-4">
      <h2 className="text-lg font-semibold text-slate-900">Pendapatan Voucher</h2>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
        <MetricCard label="Total Pendapatan Voucher" value={formatCurrency(total_revenue)} />
        <MetricCard label="Total Voucher Terjual" value={formatNumber(total_voucher_count)} />
        <MetricCard label="Total Margin Reseller" value={formatCurrency(total_reseller_margin)} />
      </div>

      {by_package.length > 0 && (
        <div className="rounded-xl border border-slate-200 bg-white p-5">
          <h3 className="mb-3 text-sm font-medium text-slate-700">Per Paket</h3>
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm">
              <thead>
                <tr className="border-b border-slate-200 text-xs text-slate-500">
                  <th className="pb-2 pr-4">Paket</th>
                  <th className="pb-2 pr-4 text-right">Pendapatan</th>
                  <th className="pb-2 pr-4 text-right">Jumlah</th>
                  <th className="pb-2 text-right">%</th>
                </tr>
              </thead>
              <tbody>
                {by_package.map((p) => (
                  <tr key={p.package_name} className="border-b border-slate-100">
                    <td className="py-2 pr-4 text-slate-700">{p.package_name}</td>
                    <td className="py-2 pr-4 text-right font-mono text-slate-900">{formatCurrency(p.total_revenue)}</td>
                    <td className="py-2 pr-4 text-right text-slate-600">{formatNumber(p.voucher_count)}</td>
                    <td className="py-2 text-right text-slate-600">{formatPercentage(p.percentage)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {by_reseller.length > 0 && (
        <div className="rounded-xl border border-slate-200 bg-white p-5">
          <h3 className="mb-3 text-sm font-medium text-slate-700">Per Reseller</h3>
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm">
              <thead>
                <tr className="border-b border-slate-200 text-xs text-slate-500">
                  <th className="pb-2 pr-4">Reseller</th>
                  <th className="pb-2 pr-4 text-right">Pendapatan</th>
                  <th className="pb-2 pr-4 text-right">Jumlah</th>
                  <th className="pb-2 text-right">Margin</th>
                </tr>
              </thead>
              <tbody>
                {by_reseller.map((r) => (
                  <tr key={r.reseller_name} className="border-b border-slate-100">
                    <td className="py-2 pr-4 text-slate-700">{r.reseller_name}</td>
                    <td className="py-2 pr-4 text-right font-mono text-slate-900">{formatCurrency(r.total_revenue)}</td>
                    <td className="py-2 pr-4 text-right text-slate-600">{formatNumber(r.voucher_count)}</td>
                    <td className="py-2 text-right text-slate-600">{formatCurrency(r.reseller_margin)}</td>
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
      <h2 className="text-lg font-semibold text-slate-900">Pendapatan Voucher</h2>
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
      <h2 className="text-lg font-semibold text-slate-900">Pendapatan Voucher</h2>
      <div className="rounded-xl border border-red-200 bg-red-50 px-5 py-4 text-sm text-red-700">
        Gagal memuat data: {message}
      </div>
    </section>
  );
}
