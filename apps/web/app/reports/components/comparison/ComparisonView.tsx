"use client";

import type { ComparisonMetric } from "../../lib/types";
import { formatCurrency, formatDelta, getDeltaColor } from "../../lib/formatters";

interface ComparisonViewProps {
  basePeriod: string;
  comparePeriod: string;
  metrics: ComparisonMetric[];
}

function formatMetricValue(name: string, value: number): string {
  const lower = name.toLowerCase();
  if (lower.includes("revenue") || lower.includes("pendapatan") || lower.includes("piutang") || lower.includes("arpu") || lower.includes("clv")) {
    return formatCurrency(value);
  }
  if (lower.includes("rate") || lower.includes("margin") || lower.includes("persentase")) {
    return `${value.toFixed(1)}%`;
  }
  return value.toLocaleString("id-ID");
}

export function ComparisonView({ basePeriod, comparePeriod, metrics }: ComparisonViewProps) {
  if (metrics.length === 0) return null;

  return (
    <div className="rounded-xl border border-slate-200 bg-white p-5">
      <h3 className="mb-4 text-sm font-medium text-slate-700">Perbandingan Periode</h3>

      <div className="overflow-x-auto">
        <table className="w-full text-left text-sm">
          <thead>
            <tr className="border-b border-slate-200 text-xs text-slate-500">
              <th className="pb-2 pr-4">Metrik</th>
              <th className="pb-2 pr-4 text-right">{basePeriod}</th>
              <th className="pb-2 pr-4 text-right">{comparePeriod}</th>
              <th className="pb-2 pr-4 text-right">Delta</th>
              <th className="pb-2 text-right">Perubahan</th>
              <th className="pb-2 text-center">Tren</th>
            </tr>
          </thead>
          <tbody>
            {metrics.map((m) => (
              <tr key={m.metric_name} className="border-b border-slate-100">
                <td className="py-2 pr-4 font-medium text-slate-700">{m.metric_name}</td>
                <td className="py-2 pr-4 text-right font-mono text-slate-900">
                  {formatMetricValue(m.metric_name, m.base_value)}
                </td>
                <td className="py-2 pr-4 text-right font-mono text-slate-600">
                  {formatMetricValue(m.metric_name, m.compare_value)}
                </td>
                <td className="py-2 pr-4 text-right font-mono text-slate-600">
                  {formatMetricValue(m.metric_name, m.delta_absolute)}
                </td>
                <td className={`py-2 pr-4 text-right font-mono font-semibold ${getDeltaColor(m.delta_percentage)}`}>
                  {formatDelta(m.delta_percentage)}
                </td>
                <td className="py-2 text-center">
                  <TrendIndicator trend={m.trend} />
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

function TrendIndicator({ trend }: { trend: "improving" | "declining" | "stable" }) {
  if (trend === "improving") {
    return (
      <span className="inline-flex items-center gap-1 text-xs font-medium text-emerald-600">
        <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" d="M4.5 19.5l15-15m0 0H8.25m11.25 0v11.25" />
        </svg>
        Naik
      </span>
    );
  }
  if (trend === "declining") {
    return (
      <span className="inline-flex items-center gap-1 text-xs font-medium text-red-600">
        <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" d="M4.5 4.5l15 15m0 0V8.25m0 11.25H8.25" />
        </svg>
        Turun
      </span>
    );
  }
  return (
    <span className="inline-flex items-center gap-1 text-xs font-medium text-slate-500">
      <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" d="M5 12h14" />
      </svg>
      Stabil
    </span>
  );
}
