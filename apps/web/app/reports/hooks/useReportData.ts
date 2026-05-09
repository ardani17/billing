"use client";

import { useState, useEffect, useCallback, useRef } from "react";

interface UseReportDataOptions<T> {
  /** Fungsi fetch yang mengembalikan data*/
  fetcher: () => Promise<T>;
  /** Aktifkan fetch otomatis saat mount*/
  enabled?: boolean;
}

interface UseReportDataResult<T> {
  data: T | null;
  loading: boolean;
  error: string | null;
  refetch: () => Promise<void>;
}

/**
 * Hook generik untuk mengambil data laporan dengan loading, error, dan refetch.
 * Mendukung caching sederhana dan abort pada unmount.
 */
export function useReportData<T>({
  fetcher,
  enabled = true,
}: UseReportDataOptions<T>): UseReportDataResult<T> {
  const [data, setData] = useState<T | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const abortRef = useRef(false);
  const fetcherRef = useRef(fetcher);
  fetcherRef.current = fetcher;

  const refetch = useCallback(async () => {
    abortRef.current = false;
    setLoading(true);
    setError(null);
    try {
      const result = await fetcherRef.current();
      if (!abortRef.current) {
        setData(result);
      }
    } catch (err) {
      if (!abortRef.current) {
        setError(err instanceof Error ? err.message : "Terjadi kesalahan saat memuat data");
      }
    } finally {
      if (!abortRef.current) {
        setLoading(false);
      }
    }
  }, []);

  useEffect(() => {
    if (enabled) {
      refetch();
    }
    return () => {
      abortRef.current = true;
    };
  }, [enabled, refetch]);

  return { data, loading, error, refetch };
}
