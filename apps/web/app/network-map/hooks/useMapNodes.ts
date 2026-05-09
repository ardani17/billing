'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import {
  fetchNodes,
  type BoundingBox,
  type MapNodeWithRef,
  type NodeFilters,
} from '../lib/api';

const DEBOUNCE_MS = 300;

interface UseMapNodesOptions {
  filters?: NodeFilters;
  enabled?: boolean;
}

interface UseMapNodesReturn {
  nodes: MapNodeWithRef[];
  loading: boolean;
  error: string | null;
  /** Dipanggil saat viewport peta berubah. Debounce dilakukan internal.*/
  onBoundsChange: (bounds: BoundingBox) => void;
  refetch: () => void;
}

export function useMapNodes(options: UseMapNodesOptions = {}): UseMapNodesReturn {
  const { filters, enabled = true } = options;

  const [nodes, setNodes] = useState<MapNodeWithRef[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const boundsRef = useRef<BoundingBox | null>(null);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const abortRef = useRef<AbortController | null>(null);

  const load = useCallback(
    async (bounds: BoundingBox) => {
      if (!enabled) return;

      // Batalkan permintaan yang masih berjalan
      abortRef.current?.abort();
      const controller = new AbortController();
      abortRef.current = controller;

      setLoading(true);
      setError(null);

      try {
        const data = await fetchNodes(bounds, filters);
        if (!controller.signal.aborted) {
          setNodes(data);
        }
      } catch (err) {
        if (!controller.signal.aborted) {
          setError(err instanceof Error ? err.message : 'Failed to fetch nodes');
        }
      } finally {
        if (!controller.signal.aborted) {
          setLoading(false);
        }
      }
    },
    [filters, enabled],
  );

  const onBoundsChange = useCallback(
    (bounds: BoundingBox) => {
      boundsRef.current = bounds;
      if (timerRef.current) clearTimeout(timerRef.current);
      timerRef.current = setTimeout(() => {
        load(bounds);
      }, DEBOUNCE_MS);
    },
    [load],
  );

  const refetch = useCallback(() => {
    if (boundsRef.current) {
      load(boundsRef.current);
    }
  }, [load]);

  // Refetch saat filter berubah jika batas sudah tersedia
  useEffect(() => {
    if (boundsRef.current) {
      load(boundsRef.current);
    }
  }, [load]);

  // Bersihkan saat komponen dilepas
  useEffect(() => {
    return () => {
      if (timerRef.current) clearTimeout(timerRef.current);
      abortRef.current?.abort();
    };
  }, []);

  return { nodes, loading, error, onBoundsChange, refetch };
}
