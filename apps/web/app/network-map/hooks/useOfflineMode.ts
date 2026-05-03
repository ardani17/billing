'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import {
  cacheNodes,
  cacheCables,
  getCachedNodes,
  getCachedCables,
  addPendingChange,
  getPendingChanges,
  clearPendingChanges,
  cleanExpiredCache,
  registerServiceWorker,
  type PendingChange,
} from '../lib/offline-manager';
import {
  createNode,
  updateNode,
  deleteNode,
  createCable,
  deleteCable,
  type MapNodeWithRef,
  type CableRoute,
} from '../lib/api';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export type OfflineStatus = 'online' | 'offline' | 'syncing' | 'synced';

interface UseOfflineModeReturn {
  /** Current connectivity status */
  status: OfflineStatus;
  /** Whether the app is currently offline */
  isOffline: boolean;
  /** Number of pending changes waiting to sync */
  pendingCount: number;
  /** Cache nodes for offline use */
  cacheNodeData: (nodes: MapNodeWithRef[]) => Promise<void>;
  /** Cache cables for offline use */
  cacheCableData: (cables: CableRoute[]) => Promise<void>;
  /** Get cached nodes when offline */
  getOfflineNodes: () => Promise<MapNodeWithRef[]>;
  /** Get cached cables when offline */
  getOfflineCables: () => Promise<CableRoute[]>;
  /** Queue a change for later sync */
  queueChange: (change: Omit<PendingChange, 'id' | 'created_at'>) => Promise<void>;
  /** Manually trigger sync */
  syncNow: () => Promise<void>;
}

// ---------------------------------------------------------------------------
// Hook
// ---------------------------------------------------------------------------

export function useOfflineMode(): UseOfflineModeReturn {
  const [status, setStatus] = useState<OfflineStatus>('online');
  const [pendingCount, setPendingCount] = useState(0);
  const syncingRef = useRef(false);

  const isOffline = status === 'offline';

  // Register service worker on mount
  useEffect(() => {
    registerServiceWorker();
    cleanExpiredCache().catch(() => {});
  }, []);

  // Listen for online/offline events
  useEffect(() => {
    function handleOnline() {
      setStatus('online');
      // Auto-sync when coming back online
      syncPendingChanges();
    }

    function handleOffline() {
      setStatus('offline');
    }

    window.addEventListener('online', handleOnline);
    window.addEventListener('offline', handleOffline);

    // Set initial status
    if (!navigator.onLine) {
      setStatus('offline');
    }

    return () => {
      window.removeEventListener('online', handleOnline);
      window.removeEventListener('offline', handleOffline);
    };
  }, []);

  // Load pending count on mount
  useEffect(() => {
    getPendingChanges()
      .then((changes) => setPendingCount(changes.length))
      .catch(() => {});
  }, []);

  // Sync pending changes to the server
  const syncPendingChanges = useCallback(async () => {
    if (syncingRef.current || !navigator.onLine) return;
    syncingRef.current = true;
    setStatus('syncing');

    try {
      const changes = await getPendingChanges();
      if (changes.length === 0) {
        setStatus('online');
        syncingRef.current = false;
        return;
      }

      for (const change of changes) {
        try {
          await applyChange(change);
        } catch (err) {
          // Last-write-wins: log conflict but continue
          console.warn('Sync conflict for change:', change, err);
        }
      }

      await clearPendingChanges();
      setPendingCount(0);
      setStatus('synced');

      // Reset to online after showing synced indicator
      setTimeout(() => setStatus('online'), 2000);
    } catch {
      setStatus('online');
    } finally {
      syncingRef.current = false;
    }
  }, []);

  const cacheNodeData = useCallback(async (nodes: MapNodeWithRef[]) => {
    await cacheNodes(nodes as unknown as Record<string, unknown>[]);
  }, []);

  const cacheCableData = useCallback(async (cables: CableRoute[]) => {
    await cacheCables(cables as unknown as Record<string, unknown>[]);
  }, []);

  const getOfflineNodes = useCallback(async (): Promise<MapNodeWithRef[]> => {
    const data = await getCachedNodes();
    return data as unknown as MapNodeWithRef[];
  }, []);

  const getOfflineCables = useCallback(async (): Promise<CableRoute[]> => {
    const data = await getCachedCables();
    return data as unknown as CableRoute[];
  }, []);

  const queueChange = useCallback(
    async (change: Omit<PendingChange, 'id' | 'created_at'>) => {
      await addPendingChange({
        ...change,
        created_at: new Date().toISOString(),
      });
      setPendingCount((prev) => prev + 1);
    },
    [],
  );

  const syncNow = useCallback(async () => {
    await syncPendingChanges();
  }, [syncPendingChanges]);

  return {
    status,
    isOffline,
    pendingCount,
    cacheNodeData,
    cacheCableData,
    getOfflineNodes,
    getOfflineCables,
    queueChange,
    syncNow,
  };
}

// ---------------------------------------------------------------------------
// Apply a single pending change to the API
// ---------------------------------------------------------------------------

async function applyChange(change: PendingChange): Promise<void> {
  const p = change.payload;

  switch (change.type) {
    case 'create_node':
      await createNode({
        node_type: p.node_type as string,
        reference_id: p.reference_id as string,
        latitude: p.latitude as number,
        longitude: p.longitude as number,
        custom_fields: p.custom_fields as Record<string, unknown> | undefined,
      });
      break;

    case 'update_node':
      await updateNode(p.id as string, {
        latitude: p.latitude as number | undefined,
        longitude: p.longitude as number | undefined,
        custom_fields: p.custom_fields as Record<string, unknown> | undefined,
      });
      break;

    case 'delete_node':
      await deleteNode(p.id as string);
      break;

    case 'create_cable':
      await createCable({
        from_node_id: p.from_node_id as string,
        to_node_id: p.to_node_id as string,
        route_type: p.route_type as string,
        coordinates: p.coordinates as [number, number][],
        core_count: p.core_count as number | undefined,
        description: p.description as string | undefined,
      });
      break;

    case 'delete_cable':
      await deleteCable(p.id as string);
      break;
  }
}
