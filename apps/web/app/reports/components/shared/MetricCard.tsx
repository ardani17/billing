"use client";

import { formatDelta, getDeltaColor } from "../../lib/formatters";

interface MetricCardProps {
  /** Label metrik (contoh: "Total Pendapatan") */
  label: string;
  /** Nilai yang sudah diformat (contoh: "Rp 12.345.678") */
  value: string;
  /** Delta persentase dibanding periode sebelumnya */
  delta?: number;
  /** Progress KPI (0-100) */
  kpiProgress?: number;
  /** Label target KPI */
  kpiLabel?: string;
}

export function MetricCard({ label, value, delta, kpiProgress, kpiLabel }: MetricCardProps) {
  return (
    <div className="rounded-xl border border-slate-200 bg-white p-5">
      <p className="text-sm text-slate-500">{label}</p>
      <div className="mt-2 flex items-end justify-between gap-3">
        <p className="font-mono text-2xl font-semibold tracking-tight text-slate-950">
          {value}
        </p>
        {delta !== undefined && (
          <span
            className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-semibold ${getDeltaColor(delta)}`}
          >
            {formatDelta(delta)}
          </span>
        )}
      </div>
      {kpiProgress !== undefined && (
        <div className="mt-3">
          <div className="flex items-center justify-between text-xs">
            <span className="text-slate-500">{kpiLabel ?? "Target KPI"}</span>
            <span className="font-medium text-slate-700">
              {kpiProgress.toFixed(0)}%
            </span>
          </div>
          <div className="mt-1 h-2 w-full overflow-hidden rounded-full bg-slate-100">
            <div
              className={`h-full rounded-full transition-all ${
                kpiProgress >= 100
                  ? "bg-emerald-500"
                  : kpiProgress >= 80
                    ? "bg-amber-500"
                    : "bg-red-500"
              }`}
              style={{ width: `${Math.min(kpiProgress, 100)}%` }}
            />
          </div>
        </div>
      )}
    </div>
  );
}
