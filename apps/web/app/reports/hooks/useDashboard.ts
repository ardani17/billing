"use client";

import { useState, useEffect, useCallback, useRef } from "react";
import type { DashboardData } from "../lib/types";
import { fetchDashboardData } from "../lib/api";

/** Interval auto-refresh: 5 menit */
const REFRESH_INTERVAL_MS = 5 * 60 * 1000;

interface UseDashboardResult {
  data: DashboardData | null;
  loading: boolean;
  error: string | null;
  refetch: () => Promise<void>;
}

/**
 * Hook untuk dashboard widget data dengan auto-refresh setiap 5 menit.
 * Tidak melakukan full page reload — hanya re-fetch data.
 */
export function useDashboard(): UseDashboardResult {
  const [data, setData] = useState<DashboardData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const mountedRef = useRef(true);

  const refetch = useCallback(async () => {
    if (!mountedRef.current) return;
    setLoading(true);
    setError(null);
    try {
      const result = await fetchDashboardData();
      if (mountedRef.current) {
        setData(result);
      }
    } catch (err) {
      if (mountedRef.current) {
        setError(err instanceof Error ? err.message : "Gagal memuat data dashboard");
      }
    } finally {
      if (mountedRef.current) {
        setLoading(false);
      }
    }
  }, []);

  useEffect(() => {
    mountedRef.current = true;
    refetch();

    const interval = setInterval(refetch, REFRESH_INTERVAL_MS);
    return () => {
      mountedRef.current = false;
      clearInterval(interval);
    };
  }, [refetch]);

  return { data, loading, error, refetch };
}
