"use client";

import { useState } from "react";
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
    <div className="rounded-xl border border-slate-200 bg-white">
      {/* Header — selalu terlihat */}
      <button
        type="button"
        onClick={() => setExpanded(!expanded)}
        className="flex w-full items-center justify-between px-4 py-3 text-left md:hidden"
        style={{ minHeight: 44 }}
      >
        <span className="text-sm font-medium text-slate-700">
          Filter: {PERIOD_OPTIONS.find((o) => o.value === periodPreset)?.label ?? "Bulan Ini"}
        </span>
        <svg
          className={`h-5 w-5 text-slate-400 transition-transform ${expanded ? "rotate-180" : ""}`}
          fill="none"
          viewBox="0 0 24 24"
          strokeWidth={1.5}
          stroke="currentColor"
        >
          <path strokeLinecap="round" strokeLinejoin="round" d="m19.5 8.25-7.5 7.5-7.5-7.5" />
        </svg>
      </button>

      {/* Filter controls */}
      <div className={`${expanded ? "block" : "hidden"} md:block`}>
        <div className="flex flex-col gap-3 px-4 pb-4 pt-2 md:flex-row md:flex-wrap md:items-end md:pt-4">
          {/* Periode */}
          <div className="min-w-[140px]">
            <label className="mb-1 block text-xs font-medium text-slate-500">Periode</label>
            <select
              value={periodPreset}
              onChange={(e) => onPeriodPresetChange(e.target.value as PeriodPreset)}
              className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-700 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
              style={{ minHeight: 44 }}
            >
              {PERIOD_OPTIONS.map((o) => (
                <option key={o.value} value={o.value}>{o.label}</option>
              ))}
            </select>
          </div>

          {/* Custom date range */}
          {periodPreset === "custom" && (
            <>
              <div>
                <label className="mb-1 block text-xs font-medium text-slate-500">Dari</label>
                <input
                  type="date"
                  value={periodStart}
                  onChange={(e) => onCustomPeriodChange(e.target.value, periodEnd)}
                  className="rounded-lg border border-slate-300 px-3 py-2 text-sm text-slate-700 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
                  style={{ minHeight: 44 }}
                />
              </div>
              <div>
                <label className="mb-1 block text-xs font-medium text-slate-500">Sampai</label>
                <input
                  type="date"
                  value={periodEnd}
                  onChange={(e) => onCustomPeriodChange(periodStart, e.target.value)}
                  className="rounded-lg border border-slate-300 px-3 py-2 text-sm text-slate-700 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
                  style={{ minHeight: 44 }}
                />
              </div>
            </>
          )}

          {/* Perbandingan */}
          <div className="min-w-[180px]">
            <label className="mb-1 block text-xs font-medium text-slate-500">Bandingkan</label>
            <select
              value={comparisonType ?? ""}
              onChange={(e) => onComparisonChange(e.target.value ? (e.target.value as ComparisonType) : undefined)}
              className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-700 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
              style={{ minHeight: 44 }}
            >
              {COMPARISON_OPTIONS.map((o) => (
                <option key={o.value} value={o.value}>{o.label}</option>
              ))}
            </select>
          </div>

          {/* Reset */}
          <button
            type="button"
            onClick={onReset}
            className="rounded-lg border border-slate-300 px-4 py-2 text-sm font-medium text-slate-600 hover:bg-slate-50"
            style={{ minHeight: 44 }}
          >
            Reset
          </button>
        </div>
      </div>
    </div>
  );
}
