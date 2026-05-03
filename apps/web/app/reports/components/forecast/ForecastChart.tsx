"use client";

import type { ForecastReport } from "../../lib/types";
import { formatCurrency, formatMonth } from "../../lib/formatters";
import { LineChart } from "../charts/LineChart";

interface ForecastChartProps {
  data: ForecastReport;
  /** Data historis aktual (6 bulan) untuk ditampilkan bersama proyeksi */
  historicalRevenue?: { month: string; value: number }[];
  /** Target KPI pendapatan bulanan */
  revenueTarget?: number;
}

export function ForecastChart({ data, historicalRevenue, revenueTarget }: ForecastChartProps) {
  if (data.insufficient_data) {
    return (
      <div className="rounded-xl border border-amber-200 bg-amber-50 px-5 py-6 text-center">
        <p className="text-sm font-medium text-amber-800">Data Tidak Cukup</p>
        <p className="mt-1 text-sm text-amber-700">
          Diperlukan minimal 3 bulan data historis untuk menghasilkan proyeksi.
        </p>
      </div>
    );
  }

  // Gabungkan data historis dan proyeksi
  const chartData: Record<string, unknown>[] = [];

  if (historicalRevenue) {
    for (const h of historicalRevenue) {
      chartData.push({
        month: h.month,
        actual: h.value,
        projected: null,
      });
    }
  }

  for (const p of data.projections) {
    chartData.push({
      month: p.month,
      actual: null,
      projected: p.projected_revenue,
    });
  }

  // Titik penghubung: set projected pada bulan terakhir historis
  if (historicalRevenue && historicalRevenue.length > 0 && chartData.length > 0) {
    const lastHistIdx = historicalRevenue.length - 1;
    if (chartData[lastHistIdx]) {
      chartData[lastHistIdx].projected = chartData[lastHistIdx].actual;
    }
  }

  return (
    <section className="space-y-4">
      <h2 className="text-lg font-semibold text-slate-900">Proyeksi & Forecast</h2>

      <div className="rounded-xl border border-slate-200 bg-white p-5">
        <h3 className="mb-4 text-sm font-medium text-slate-700">Proyeksi Pendapatan</h3>
        <LineChart
          data={chartData}
          xKey="month"
          xFormatter={formatMonth}
          valueFormatter={(v) => formatCurrency(v)}
          lines={[
            { dataKey: "actual", name: "Aktual", color: "#3b82f6" },
            { dataKey: "projected", name: "Proyeksi", color: "#8b5cf6", dashed: true },
          ]}
          referenceLine={
            revenueTarget
              ? { value: revenueTarget, label: `Target: ${formatCurrency(revenueTarget)}`, color: "#ef4444" }
              : undefined
          }
        />
      </div>

      {/* Tabel proyeksi */}
      {data.projections.length > 0 && (
        <div className="rounded-xl border border-slate-200 bg-white p-5">
          <h3 className="mb-3 text-sm font-medium text-slate-700">Detail Proyeksi 3 Bulan</h3>
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm">
              <thead>
                <tr className="border-b border-slate-200 text-xs text-slate-500">
                  <th className="pb-2 pr-4">Bulan</th>
                  <th className="pb-2 pr-4 text-right">Pendapatan</th>
                  <th className="pb-2 pr-4 text-right">Pelanggan</th>
                  <th className="pb-2 text-right">Piutang</th>
                </tr>
              </thead>
              <tbody>
                {data.projections.map((p) => (
                  <tr key={p.month} className="border-b border-slate-100">
                    <td className="py-2 pr-4 text-slate-700">{formatMonth(p.month)}</td>
                    <td className="py-2 pr-4 text-right font-mono text-slate-900">{formatCurrency(p.projected_revenue)}</td>
                    <td className="py-2 pr-4 text-right text-slate-600">{p.projected_customers.toLocaleString("id-ID")}</td>
                    <td className="py-2 text-right font-mono text-slate-600">{formatCurrency(p.projected_receivables)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Estimated target date */}
      {data.estimated_target_date && Object.keys(data.estimated_target_date).length > 0 && (
        <div className="rounded-xl border border-emerald-200 bg-emerald-50 p-5">
          <h3 className="mb-2 text-sm font-medium text-emerald-800">Estimasi Pencapaian Target</h3>
          <div className="space-y-1">
            {Object.entries(data.estimated_target_date).map(([metric, date]) => (
              <p key={metric} className="text-sm text-emerald-700">
                <span className="font-medium">{metric}:</span> {formatMonth(date)}
              </p>
            ))}
          </div>
        </div>
      )}

      {/* Disclaimer */}
      {data.disclaimer && (
        <p className="text-xs italic text-slate-400">{data.disclaimer}</p>
      )}
    </section>
  );
}
