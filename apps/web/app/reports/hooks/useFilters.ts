"use client";

import { useState, useCallback, useEffect } from "react";
import type { ReportFilter, ComparisonType } from "../lib/types";

export type ReportTab = "keuangan" | "pelanggan" | "jaringan" | "operasional";

export type PeriodPreset =
  | "hari_ini"
  | "minggu_ini"
  | "bulan_ini"
  | "kuartal"
  | "tahun"
  | "custom";

interface FilterState {
  tab: ReportTab;
  periodPreset: PeriodPreset;
  periodStart: string;
  periodEnd: string;
  comparisonType?: ComparisonType;
  areaId?: string;
  packageId?: string;
  routerId?: string;
}

function getToday(): string {
  return new Date().toISOString().slice(0, 10);
}

function getStartOfWeek(): string {
  const d = new Date();
  d.setDate(d.getDate() - d.getDay() + 1);
  return d.toISOString().slice(0, 10);
}

function getStartOfMonth(): string {
  const d = new Date();
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-01`;
}

function getStartOfQuarter(): string {
  const d = new Date();
  const q = Math.floor(d.getMonth() / 3) * 3;
  return `${d.getFullYear()}-${String(q + 1).padStart(2, "0")}-01`;
}

function getStartOfYear(): string {
  return `${new Date().getFullYear()}-01-01`;
}

function periodToRange(preset: PeriodPreset): { start: string; end: string } {
  const end = getToday();
  switch (preset) {
    case "hari_ini":
      return { start: end, end };
    case "minggu_ini":
      return { start: getStartOfWeek(), end };
    case "bulan_ini":
      return { start: getStartOfMonth(), end };
    case "kuartal":
      return { start: getStartOfQuarter(), end };
    case "tahun":
      return { start: getStartOfYear(), end };
    default:
      return { start: getStartOfMonth(), end };
  }
}

function readFromURL(): Partial<FilterState> {
  if (typeof window === "undefined") return {};
  const params = new URLSearchParams(window.location.search);
  const state: Partial<FilterState> = {};
  if (params.get("tab")) state.tab = params.get("tab") as ReportTab;
  if (params.get("preset")) state.periodPreset = params.get("preset") as PeriodPreset;
  if (params.get("start")) state.periodStart = params.get("start")!;
  if (params.get("end")) state.periodEnd = params.get("end")!;
  if (params.get("compare")) state.comparisonType = params.get("compare") as ComparisonType;
  if (params.get("area")) state.areaId = params.get("area")!;
  if (params.get("package")) state.packageId = params.get("package")!;
  if (params.get("router")) state.routerId = params.get("router")!;
  return state;
}

function writeToURL(state: FilterState) {
  if (typeof window === "undefined") return;
  const params = new URLSearchParams();
  params.set("tab", state.tab);
  params.set("preset", state.periodPreset);
  if (state.periodPreset === "custom") {
    params.set("start", state.periodStart);
    params.set("end", state.periodEnd);
  }
  if (state.comparisonType) params.set("compare", state.comparisonType);
  if (state.areaId) params.set("area", state.areaId);
  if (state.packageId) params.set("package", state.packageId);
  if (state.routerId) params.set("router", state.routerId);
  window.history.replaceState(null, "", `?${params.toString()}`);
}

const defaultRange = periodToRange("bulan_ini");

const DEFAULT_STATE: FilterState = {
  tab: "keuangan",
  periodPreset: "bulan_ini",
  periodStart: defaultRange.start,
  periodEnd: defaultRange.end,
};

/**
 * Hook untuk filter manajemen state yang sync dengan URL kueri params.
 */
export function useFilters() {
  const [filters, setFilters] = useState<FilterState>(() => {
    const fromURL = readFromURL();
    const preset = fromURL.periodPreset ?? DEFAULT_STATE.periodPreset;
    const range = preset === "custom"
      ? { start: fromURL.periodStart ?? defaultRange.start, end: fromURL.periodEnd ?? defaultRange.end }
      : periodToRange(preset);
    return {
      ...DEFAULT_STATE,
      ...fromURL,
      periodStart: range.start,
      periodEnd: range.end,
    };
  });

  useEffect(() => {
    writeToURL(filters);
  }, [filters]);

  const setTab = useCallback((tab: ReportTab) => {
    setFilters((prev) => ({ ...prev, tab }));
  }, []);

  const setPeriodPreset = useCallback((preset: PeriodPreset) => {
    const range = periodToRange(preset);
    setFilters((prev) => ({
      ...prev,
      periodPreset: preset,
      periodStart: preset === "custom" ? prev.periodStart : range.start,
      periodEnd: preset === "custom" ? prev.periodEnd : range.end,
    }));
  }, []);

  const setCustomPeriod = useCallback((start: string, end: string) => {
    setFilters((prev) => ({
      ...prev,
      periodPreset: "custom" as PeriodPreset,
      periodStart: start,
      periodEnd: end,
    }));
  }, []);

  const setComparisonType = useCallback((type?: ComparisonType) => {
    setFilters((prev) => ({ ...prev, comparisonType: type }));
  }, []);

  const setAreaId = useCallback((id?: string) => {
    setFilters((prev) => ({ ...prev, areaId: id }));
  }, []);

  const setPackageId = useCallback((id?: string) => {
    setFilters((prev) => ({ ...prev, packageId: id }));
  }, []);

  const setRouterId = useCallback((id?: string) => {
    setFilters((prev) => ({ ...prev, routerId: id }));
  }, []);

  const reset = useCallback(() => {
    const range = periodToRange("bulan_ini");
    setFilters({
      ...DEFAULT_STATE,
      periodStart: range.start,
      periodEnd: range.end,
    });
  }, []);

  const toReportFilter = useCallback((): Partial<ReportFilter> => ({
    period_start: filters.periodStart,
    period_end: filters.periodEnd,
    area_id: filters.areaId,
    package_id: filters.packageId,
    router_id: filters.routerId,
  }), [filters]);

  return {
    filters,
    setTab,
    setPeriodPreset,
    setCustomPeriod,
    setComparisonType,
    setAreaId,
    setPackageId,
    setRouterId,
    reset,
    toReportFilter,
  };
}
