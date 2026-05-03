/**
 * Offline Manager — Service Worker registration and IndexedDB schema
 * for caching map tiles, node data, cable routes, and photo thumbnails.
 *
 * Max 100MB per area, expire after 7 days.
 */

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const DB_NAME = 'network-map-offline';
const DB_VERSION = 1;
const MAX_CACHE_SIZE_BYTES = 100 * 1024 * 1024; // 100 MB
const CACHE_EXPIRY_DAYS = 7;

// Store names
const STORES = {
  NODES: 'nodes',
  CABLES: 'cables',
  TILES: 'tiles',
  PHOTOS: 'photos',
  PENDING_CHANGES: 'pending_changes',
  META: 'meta',
} as const;

// ---------------------------------------------------------------------------
// IndexedDB helpers
// ---------------------------------------------------------------------------

function openDB(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const request = indexedDB.open(DB_NAME, DB_VERSION);

    request.onupgradeneeded = (event) => {
      const db = (event.target as IDBOpenDBRequest).result;

      // Nodes store — keyed by node ID
      if (!db.objectStoreNames.contains(STORES.NODES)) {
        const nodeStore = db.createObjectStore(STORES.NODES, { keyPath: 'id' });
        nodeStore.createIndex('node_type', 'node_type', { unique: false });
        nodeStore.createIndex('cached_at', 'cached_at', { unique: false });
      }

      // Cables store — keyed by cable ID
      if (!db.objectStoreNames.contains(STORES.CABLES)) {
        const cableStore = db.createObjectStore(STORES.CABLES, { keyPath: 'id' });
        cableStore.createIndex('cached_at', 'cached_at', { unique: false });
      }

      // Tiles store — keyed by tile URL
      if (!db.objectStoreNames.contains(STORES.TILES)) {
        const tileStore = db.createObjectStore(STORES.TILES, { keyPath: 'url' });
        tileStore.createIndex('cached_at', 'cached_at', { unique: false });
      }

      // Photos store — keyed by photo ID
      if (!db.objectStoreNames.contains(STORES.PHOTOS)) {
        const photoStore = db.createObjectStore(STORES.PHOTOS, { keyPath: 'id' });
        photoStore.createIndex('map_node_id', 'map_node_id', { unique: false });
        photoStore.createIndex('cached_at', 'cached_at', { unique: false });
      }

      // Pending changes store — keyed by auto-increment
      if (!db.objectStoreNames.contains(STORES.PENDING_CHANGES)) {
        const changeStore = db.createObjectStore(STORES.PENDING_CHANGES, {
          keyPath: 'id',
          autoIncrement: true,
        });
        changeStore.createIndex('created_at', 'created_at', { unique: false });
      }

      // Meta store — for cache metadata
      if (!db.objectStoreNames.contains(STORES.META)) {
        db.createObjectStore(STORES.META, { keyPath: 'key' });
      }
    };

    request.onsuccess = () => resolve(request.result);
    request.onerror = () => reject(request.error);
  });
}

// ---------------------------------------------------------------------------
// Generic CRUD operations
// ---------------------------------------------------------------------------

async function putItems<T>(storeName: string, items: T[]): Promise<void> {
  const db = await openDB();
  const tx = db.transaction(storeName, 'readwrite');
  const store = tx.objectStore(storeName);
  const now = new Date().toISOString();

  for (const item of items) {
    store.put({ ...item, cached_at: now });
  }

  return new Promise((resolve, reject) => {
    tx.oncomplete = () => resolve();
    tx.onerror = () => reject(tx.error);
  });
}

async function getAllItems<T>(storeName: string): Promise<T[]> {
  const db = await openDB();
  const tx = db.transaction(storeName, 'readonly');
  const store = tx.objectStore(storeName);
  const request = store.getAll();

  return new Promise((resolve, reject) => {
    request.onsuccess = () => resolve(request.result as T[]);
    request.onerror = () => reject(request.error);
  });
}

async function clearStore(storeName: string): Promise<void> {
  const db = await openDB();
  const tx = db.transaction(storeName, 'readwrite');
  const store = tx.objectStore(storeName);
  store.clear();

  return new Promise((resolve, reject) => {
    tx.oncomplete = () => resolve();
    tx.onerror = () => reject(tx.error);
  });
}

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

export interface PendingChange {
  id?: number;
  type: 'create_node' | 'update_node' | 'delete_node' | 'create_cable' | 'delete_cable';
  payload: Record<string, unknown>;
  created_at: string;
}

/** Cache nodes to IndexedDB. */
export async function cacheNodes(nodes: Record<string, unknown>[]): Promise<void> {
  await putItems(STORES.NODES, nodes);
}

/** Get cached nodes from IndexedDB. */
export async function getCachedNodes(): Promise<Record<string, unknown>[]> {
  return getAllItems(STORES.NODES);
}

/** Cache cables to IndexedDB. */
export async function cacheCables(cables: Record<string, unknown>[]): Promise<void> {
  await putItems(STORES.CABLES, cables);
}

/** Get cached cables from IndexedDB. */
export async function getCachedCables(): Promise<Record<string, unknown>[]> {
  return getAllItems(STORES.CABLES);
}

/** Store a pending change for later sync. */
export async function addPendingChange(change: Omit<PendingChange, 'id'>): Promise<void> {
  await putItems(STORES.PENDING_CHANGES, [change]);
}

/** Get all pending changes. */
export async function getPendingChanges(): Promise<PendingChange[]> {
  return getAllItems(STORES.PENDING_CHANGES);
}

/** Clear all pending changes after successful sync. */
export async function clearPendingChanges(): Promise<void> {
  await clearStore(STORES.PENDING_CHANGES);
}

/** Remove expired cache entries (older than 7 days). */
export async function cleanExpiredCache(): Promise<void> {
  const expiryDate = new Date();
  expiryDate.setDate(expiryDate.getDate() - CACHE_EXPIRY_DAYS);
  const expiryStr = expiryDate.toISOString();

  const db = await openDB();

  for (const storeName of [STORES.NODES, STORES.CABLES, STORES.TILES, STORES.PHOTOS]) {
    const tx = db.transaction(storeName, 'readwrite');
    const store = tx.objectStore(storeName);
    const index = store.index('cached_at');
    const range = IDBKeyRange.upperBound(expiryStr);
    const request = index.openCursor(range);

    await new Promise<void>((resolve, reject) => {
      request.onsuccess = () => {
        const cursor = request.result;
        if (cursor) {
          cursor.delete();
          cursor.continue();
        } else {
          resolve();
        }
      };
      request.onerror = () => reject(request.error);
    });
  }
}

/** Register the service worker for offline tile caching. */
export async function registerServiceWorker(): Promise<void> {
  if (typeof window === 'undefined' || !('serviceWorker' in navigator)) {
    return;
  }

  try {
    await navigator.serviceWorker.register('/network-map-sw.js', {
      scope: '/network-map/',
    });
  } catch (err) {
    console.warn('Service Worker registration failed:', err);
  }
}

/** Get estimated cache size in bytes. */
export async function getCacheSize(): Promise<number> {
  if (navigator.storage && navigator.storage.estimate) {
    const estimate = await navigator.storage.estimate();
    return estimate.usage ?? 0;
  }
  return 0;
}

/** Check if cache size exceeds the limit. */
export async function isCacheFull(): Promise<boolean> {
  const size = await getCacheSize();
  return size >= MAX_CACHE_SIZE_BYTES;
}

export { STORES, DB_NAME, MAX_CACHE_SIZE_BYTES, CACHE_EXPIRY_DAYS };
