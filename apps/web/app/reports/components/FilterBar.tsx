"use client";

import { useState } from "react";
import { ArrowCounterClockwise, CaretDown, SlidersHorizontal } from "@phosphor-icons/react";
import type { ComparisonType } from "../lib/types";
import type { PeriodPreset } from "../hooks/useFilters";

const PERIOD_OPTIONS: { value: PeriodPreset; label: string }[] = [
  { value: "hari_ini", label: "Hari Ini" },
  { value: "minggu_ini", label: "Minggu Ini" },
  { value: "bulan_ini", label: "Bulan Ini" },
  { value: "kuartal", label: "Kuartal Ini" },
  { value: "tahun", label: "Tahun Ini" },
  { value: "custom", label: "Kustom" },
];

const COMPARISON_OPTIONS: { value: ComparisonType | ""; label: string }[] = [
  { value: "", label: "Tanpa Perbandingan" },
  { value: "mom", label: "Bulan ke Bulan (MoM)" },
  { value: "yoy", label: "Tahun ke Tahun (YoY)" },
  { value: "qoq", label: "Kuartal ke Kuartal (QoQ)" },
];

interface FilterBarProps {
  periodPreset: PeriodPreset;
  periodStart: string;
  periodEnd: string;
  comparisonType?: ComparisonType;
  areaId?: string;
  packageId?: string;
  onPeriodPresetChange: (preset: PeriodPreset) => void;
  onCustomPeriodChange: (start: string, end: string) => void;
  onComparisonChange: (type?: ComparisonType) => void;
  onAreaChange: (id?: string) => void;
  onPackageChange: (id?: string) => void;
  onReset: () => void;
}

export function FilterBar({
  periodPreset,
  periodStart,
  periodEnd,
  comparisonType,
  onPeriodPresetChange,
  onCustomPeriodChange,
  onComparisonChange,
  onReset,
}: FilterBarProps) {
  const [expanded, setExpanded] = useState(false);

  return (
    <div className="rounded-lg border border-slate-200 bg-slate-50/70">
      <button
        type="button"
        onClick={() => setExpanded(!expanded)}
        className="flex min-h-12 w-full items-center justify-between gap-3 px-3 py-3 text-left md:hidden"
      >
        <span className="flex min-w-0 items-center gap-2 text-sm font-semibold text-slate-800">
          <SlidersHorizontal className="h-5 w-5 flex-shrink-0 text-slate-500" />
          <span className="truncate">
            {PERIOD_OPTIONS.find((o) => o.value === periodPreset)?.label ?? "Bulan Ini"}
          </span>
        </span>
        <CaretDown className={`h-5 w-5 text-slate-400 transition-transform ${expanded ? "rotate-180" : ""}`} />
      </button>

      <div className={`${expanded ? "block" : "hidden"} md:block`}>
        <div className="grid grid-cols-1 gap-3 px-3 pb-3 pt-1 md:grid-cols-[minmax(150px,1fr)_minmax(210px,1.2fr)_auto] md:items-end md:pt-3">
          <div className="min-w-0">
            <label className="mb-1.5 block text-xs font-semibold text-slate-500">Periode</label>
            <select
              value={periodPreset}
              onChange={(e) => onPeriodPresetChange(e.target.value as PeriodPreset)}
              className="min-h-11 w-full rounded-md border border-slate-200 bg-white px-3 py-2 text-sm font-medium text-slate-800 shadow-sm outline-none transition focus:border-blue-500 focus:ring-2 focus:ring-blue-100"
            >
              {PERIOD_OPTIONS.map((o) => (
                <option key={o.value} value={o.value}>{o.label}</option>
              ))}
            </select>
          </div>

          {periodPreset === "custom" && (
            <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
              <div className="min-w-0">
                <label className="mb-1.5 block text-xs font-semibold text-slate-500">Dari</label>
                <input
                  type="date"
                  value={periodStart}
                  onChange={(e) => onCustomPeriodChange(e.target.value, periodEnd)}
                  className="min-h-11 w-full rounded-md border border-slate-200 bg-white px-3 py-2 text-sm font-medium text-slate-800 shadow-sm outline-none transition focus:border-blue-500 focus:ring-2 focus:ring-blue-100"
                />
              </div>
              <div className="min-w-0">
                <label className="mb-1.5 block text-xs font-semibold text-slate-500">Sampai</label>
                <input
                  type="date"
                  value={periodEnd}
                  onChange={(e) => onCustomPeriodChange(periodStart, e.target.value)}
                  className="min-h-11 w-full rounded-md border border-slate-200 bg-white px-3 py-2 text-sm font-medium text-slate-800 shadow-sm outline-none transition focus:border-blue-500 focus:ring-2 focus:ring-blue-100"
                />
              </div>
            </div>
          )}

          <div className="min-w-0">
            <label className="mb-1.5 block text-xs font-semibold text-slate-500">Bandingkan</label>
            <select
              value={comparisonType ?? ""}
              onChange={(e) => onComparisonChange(e.target.value ? (e.target.value as ComparisonType) : undefined)}
              className="min-h-11 w-full rounded-md border border-slate-200 bg-white px-3 py-2 text-sm font-medium text-slate-800 shadow-sm outline-none transition focus:border-blue-500 focus:ring-2 focus:ring-blue-100"
            >
              {COMPARISON_OPTIONS.map((o) => (
                <option key={o.value} value={o.value}>{o.label}</option>
              ))}
            </select>
          </div>

          <button
            type="button"
            onClick={onReset}
            className="inline-flex min-h-11 items-center justify-center gap-2 rounded-md border border-slate-200 bg-white px-4 py-2 text-sm font-semibold text-slate-700 shadow-sm transition hover:border-blue-200 hover:bg-blue-50/60 hover:text-blue-700 active:scale-[0.98] focus:outline-none focus:ring-2 focus:ring-blue-100"
          >
            <ArrowCounterClockwise className="h-4 w-4" />
            Reset
          </button>
        </div>
      </div>
    </div>
  );
}
