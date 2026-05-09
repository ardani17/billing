"use client";

import type { ReportFilter } from "../../lib/types";
import { fetchPaymentReport } from "../../lib/api";
import { formatCurrency, formatDate, formatNumber, formatPercentage } from "../../lib/formatters";
import { useReportData } from "../../hooks/useReportData";
import { MetricCard } from "../shared/MetricCard";
import { EmptyState } from "../shared/EmptyState";
import { BarChart } from "../charts/BarChart";

interface Props {
  filter: Partial<ReportFilter>;
}

export function PaymentSection({ filter }: Props) {
  const { data, loading, error } = useReportData({
    fetcher: () => fetchPaymentReport(filter),
  });

  if (loading) return <Skeleton />;
  if (error) return <ErrorMsg message={error} />;
  if (!data) return <EmptyState />;

  const { methods, daily_payments, peak_payment_date, peak_amount } = data;

  return (
    <section className="space-y-4">
      <h2 className="text-lg font-semibold text-slate-900">Distribusi Pembayaran</h2>

      {/* Metode pembayaran*/}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {methods.map((m) => (
          <MetricCard
            key={m.method_name}
            label={m.method_name}
            value={formatCurrency(m.total_amount)}
          />
        ))}
        <MetricCard
          label="Puncak Pembayaran"
          value={formatCurrency(peak_amount)}
          kpiLabel={peak_payment_date ? formatDate(peak_payment_date) : undefined}
        />
      </div>

      {/* Tabel metode*/}
      {methods.length > 0 && (
        <div className="rounded-xl border border-slate-200 bg-white p-5">
          <h3 className="mb-3 text-sm font-medium text-slate-700">Rincian Metode Pembayaran</h3>
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm">
              <thead>
                <tr className="border-b border-slate-200 text-xs text-slate-500">
                  <th className="pb-2 pr-4">Metode</th>
                  <th className="pb-2 pr-4 text-right">Total</th>
                  <th className="pb-2 pr-4 text-right">Transaksi</th>
                  <th className="pb-2 text-right">Persentase</th>
                </tr>
              </thead>
              <tbody>
                {methods.map((m) => (
                  <tr key={m.method_name} className="border-b border-slate-100">
                    <td className="py-2 pr-4 text-slate-700">{m.method_name}</td>
                    <td className="py-2 pr-4 text-right font-mono text-slate-900">{formatCurrency(m.total_amount)}</td>
                    <td className="py-2 pr-4 text-right text-slate-600">{formatNumber(m.transaction_count)}</td>
                    <td className="py-2 text-right text-slate-600">{formatPercentage(m.percentage)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Chart pembayaran harian*/}
      {daily_payments.length > 0 && (
        <div className="rounded-xl border border-slate-200 bg-white p-5">
          <h3 className="mb-4 text-sm font-medium text-slate-700">Pembayaran Harian</h3>
          <BarChart
            data={daily_payments}
            xKey="date"
            xFormatter={formatDate}
            valueFormatter={(v) => formatCurrency(v)}
            bars={[{ dataKey: "total_amount", name: "Total Pembayaran", color: "#3b82f6" }]}
          />
        </div>
      )}
    </section>
  );
}

function Skeleton() {
  return (
    <section className="space-y-4">
      <h2 className="text-lg font-semibold text-slate-900">Distribusi Pembayaran</h2>
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
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
      <h2 className="text-lg font-semibold text-slate-900">Distribusi Pembayaran</h2>
      <div className="rounded-xl border border-red-200 bg-red-50 px-5 py-4 text-sm text-red-700">
        Gagal memuat data: {message}
      </div>
    </section>
  );
}
