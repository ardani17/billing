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
    <div className="min-w-0 rounded-lg border border-slate-200 bg-white p-4 shadow-sm shadow-slate-200/60 transition duration-200 hover:-translate-y-0.5 hover:border-slate-300 hover:shadow-md hover:shadow-slate-200/70 sm:p-5">
      <p className="text-sm font-medium text-slate-500">{label}</p>
      <div className="mt-2 flex min-w-0 items-start justify-between gap-3">
        <p className="min-w-0 break-words font-mono text-xl font-semibold leading-tight tracking-tight text-slate-950 sm:text-2xl">
          {value}
        </p>
        {delta !== undefined && (
          <span
            className={`inline-flex flex-shrink-0 items-center rounded-md px-2 py-1 text-xs font-semibold ${getDeltaColor(delta)}`}
          >
            {formatDelta(delta)}
          </span>
        )}
      </div>
      {kpiProgress !== undefined && (
        <div className="mt-3">
          <div className="flex items-center justify-between gap-3 text-xs">
            <span className="min-w-0 truncate text-slate-500">{kpiLabel ?? "Target KPI"}</span>
            <span className="font-medium text-slate-700">
              {kpiProgress.toFixed(0)}%
            </span>
          </div>
          <div className="mt-2 h-2 w-full overflow-hidden rounded-full bg-slate-100">
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
