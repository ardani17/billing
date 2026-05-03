'use client';

import { useCallback, useState } from 'react';
import { createCable } from '../lib/api';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface DrawLineModeProps {
  /** Collected polyline coordinates from map clicks */
  points: [number, number][];
  onCreated?: () => void;
  onCancel: () => void;
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/** Calculate straight-line distance between consecutive points using Haversine. */
function haversineDistance(
  lat1: number,
  lng1: number,
  lat2: number,
  lng2: number,
): number {
  const R = 6371000; // Earth radius in meters
  const toRad = (deg: number) => (deg * Math.PI) / 180;
  const dLat = toRad(lat2 - lat1);
  const dLng = toRad(lng2 - lng1);
  const a =
    Math.sin(dLat / 2) ** 2 +
    Math.cos(toRad(lat1)) * Math.cos(toRad(lat2)) * Math.sin(dLng / 2) ** 2;
  return R * 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a));
}

function totalDistance(points: [number, number][]): number {
  let dist = 0;
  for (let i = 1; i < points.length; i++) {
    const previous = points[i - 1];
    const current = points[i];
    if (!previous || !current) continue;
    dist += haversineDistance(
      previous[0],
      previous[1],
      current[0],
      current[1],
    );
  }
  return dist;
}

function formatDistance(meters: number): string {
  if (meters >= 1000) return `${(meters / 1000).toFixed(2)} km`;
  return `${Math.round(meters)} m`;
}

// ---------------------------------------------------------------------------
// Component — Form to save a cable route after drawing polyline
// ---------------------------------------------------------------------------

export default function DrawLineMode({
  points,
  onCreated,
  onCancel,
}: DrawLineModeProps) {
  const [fromNodeId, setFromNodeId] = useState('');
  const [toNodeId, setToNodeId] = useState('');
  const [routeType, setRouteType] = useState<'backbone' | 'drop'>('drop');
  const [coreCount, setCoreCount] = useState('');
  const [description, setDescription] = useState('');
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const distance = totalDistance(points);

  const handleSubmit = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();
      if (!fromNodeId.trim() || !toNodeId.trim()) {
        setError('From Node dan To Node wajib diisi');
        return;
      }
      if (points.length < 2) {
        setError('Minimal 2 titik diperlukan');
        return;
      }

      setSaving(true);
      setError(null);
      try {
        await createCable({
          from_node_id: fromNodeId.trim(),
          to_node_id: toNodeId.trim(),
          route_type: routeType,
          coordinates: points,
          core_count: coreCount ? parseInt(coreCount, 10) : undefined,
          description: description || undefined,
        });
        onCreated?.();
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Gagal menyimpan jalur');
      } finally {
        setSaving(false);
      }
    },
    [fromNodeId, toNodeId, routeType, points, coreCount, description, onCreated],
  );

  return (
    <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-lg">
      <h3 className="mb-3 text-sm font-semibold text-gray-900">
        Simpan Jalur Kabel
      </h3>

      {/* Distance info */}
      <div className="mb-3 rounded bg-blue-50 px-3 py-2 text-sm text-blue-700">
        📐 Jarak: {formatDistance(distance)} · {points.length} titik
      </div>

      <form onSubmit={handleSubmit} className="space-y-3">
        <div>
          <label className="mb-1 block text-xs font-medium text-gray-700">
            From Node ID
          </label>
          <input
            type="text"
            value={fromNodeId}
            onChange={(e) => setFromNodeId(e.target.value)}
            placeholder="UUID node asal"
            className="w-full rounded-md border border-gray-300 px-2 py-1.5 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
          />
        </div>

        <div>
          <label className="mb-1 block text-xs font-medium text-gray-700">
            To Node ID
          </label>
          <input
            type="text"
            value={toNodeId}
            onChange={(e) => setToNodeId(e.target.value)}
            placeholder="UUID node tujuan"
            className="w-full rounded-md border border-gray-300 px-2 py-1.5 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
          />
        </div>

        <div>
          <label className="mb-1 block text-xs font-medium text-gray-700">
            Tipe Route
          </label>
          <select
            value={routeType}
            onChange={(e) => setRouteType(e.target.value as 'backbone' | 'drop')}
            className="w-full rounded-md border border-gray-300 px-2 py-1.5 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
          >
            <option value="drop">Drop (ODP → ONT)</option>
            <option value="backbone">Backbone (OLT → ODP)</option>
          </select>
        </div>

        <div>
          <label className="mb-1 block text-xs font-medium text-gray-700">
            Jumlah Core
          </label>
          <input
            type="number"
            value={coreCount}
            onChange={(e) => setCoreCount(e.target.value)}
            placeholder="Opsional"
            min={1}
            className="w-full rounded-md border border-gray-300 px-2 py-1.5 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
          />
        </div>

        <div>
          <label className="mb-1 block text-xs font-medium text-gray-700">
            Deskripsi
          </label>
          <input
            type="text"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="Opsional"
            className="w-full rounded-md border border-gray-300 px-2 py-1.5 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
          />
        </div>

        {error && <p className="text-xs text-red-500">{error}</p>}

        <div className="flex gap-2">
          <button
            type="submit"
            disabled={saving}
            className="flex-1 rounded-md bg-blue-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
          >
            {saving ? 'Menyimpan…' : 'Simpan Jalur'}
          </button>
          <button
            type="button"
            onClick={onCancel}
            className="rounded-md border border-gray-300 px-3 py-1.5 text-sm text-gray-700 hover:bg-gray-50"
          >
            Batal
          </button>
        </div>
      </form>
    </div>
  );
}
