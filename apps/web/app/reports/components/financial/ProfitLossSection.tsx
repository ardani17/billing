"use client";

import type { ReportFilter } from "../../lib/types";
import { fetchProfitLossReport } from "../../lib/api";
import { formatCurrency, formatPercentage } from "../../lib/formatters";
import { useReportData } from "../../hooks/useReportData";
import { MetricCard } from "../shared/MetricCard";
import { EmptyState } from "../shared/EmptyState";

interface Props {
  filter: Partial<ReportFilter>;
}

export function ProfitLossSection({ filter }: Props) {
  const { data, loading, error } = useReportData({
    fetcher: () => fetchProfitLossReport(filter),
  });

  if (loading) return <Skeleton />;
  if (error) return <ErrorMsg message={error} />;
  if (!data) return <EmptyState />;

  const { revenue_items, total_revenue, expense_items, total_expenses, net_profit, profit_margin } = data;

  return (
    <section className="space-y-4">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <h2 className="text-lg font-semibold text-slate-900">Laba Rugi</h2>
        <div className="flex flex-wrap gap-2">
          <a
            href="/expenses"
            className="inline-flex items-center justify-center rounded-md border border-slate-300 bg-white px-3 py-2 text-sm font-semibold text-slate-700 transition hover:bg-slate-50"
          >
            Kelola pengeluaran
          </a>
          <a
            href="/cashflow"
            className="inline-flex items-center justify-center rounded-md bg-blue-600 px-3 py-2 text-sm font-semibold text-white transition hover:bg-blue-700"
          >
            Buka arus kas
          </a>
        </div>
      </div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <MetricCard label="Total Pendapatan" value={formatCurrency(total_revenue)} />
        <MetricCard label="Total Pengeluaran" value={formatCurrency(total_expenses)} />
        <MetricCard
          label="Laba Bersih"
          value={formatCurrency(net_profit)}
          delta={net_profit >= 0 ? undefined : undefined}
        />
        <MetricCard label="Margin Laba" value={formatPercentage(profit_margin)} />
      </div>

      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        {/* Pendapatan */}
        <div className="rounded-xl border border-slate-200 bg-white p-5">
          <h3 className="mb-3 text-sm font-medium text-emerald-700">Pendapatan</h3>
          <div className="space-y-2">
            {revenue_items.map((item) => (
              <div key={item.label} className="flex items-center justify-between text-sm">
                <span className="text-slate-600">{item.label}</span>
                <span className="font-mono text-slate-900">{formatCurrency(item.amount)}</span>
              </div>
            ))}
            <div className="border-t border-slate-200 pt-2">
              <div className="flex items-center justify-between text-sm font-semibold">
                <span className="text-slate-900">Total Pendapatan</span>
                <span className="font-mono text-emerald-700">{formatCurrency(total_revenue)}</span>
              </div>
            </div>
          </div>
        </div>

        {/* Pengeluaran */}
        <div className="rounded-xl border border-slate-200 bg-white p-5">
          <h3 className="mb-3 text-sm font-medium text-red-700">Pengeluaran</h3>
          <div className="space-y-2">
            {expense_items.map((item) => (
              <div key={item.label} className="flex items-center justify-between text-sm">
                <span className="text-slate-600">{item.label}</span>
                <span className="font-mono text-slate-900">{formatCurrency(item.amount)}</span>
              </div>
            ))}
            <div className="border-t border-slate-200 pt-2">
              <div className="flex items-center justify-between text-sm font-semibold">
                <span className="text-slate-900">Total Pengeluaran</span>
                <span className="font-mono text-red-700">{formatCurrency(total_expenses)}</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}

function Skeleton() {
  return (
    <section className="space-y-4">
      <h2 className="text-lg font-semibold text-slate-900">Laba Rugi</h2>
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
      <h2 className="text-lg font-semibold text-slate-900">Laba Rugi</h2>
      <div className="rounded-xl border border-red-200 bg-red-50 px-5 py-4 text-sm text-red-700">
        Gagal memuat data: {message}
      </div>
    </section>
  );
}
