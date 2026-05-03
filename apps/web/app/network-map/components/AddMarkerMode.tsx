'use client';

import { useCallback, useEffect, useState } from 'react';
import { createNode, reverseGeocode } from '../lib/api';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface AddMarkerModeProps {
  lat: number;
  lng: number;
  onCreated?: () => void;
  onCancel: () => void;
}

// ---------------------------------------------------------------------------
// Component — Form to create a new ODP at clicked coordinates
// ---------------------------------------------------------------------------

export default function AddMarkerMode({
  lat,
  lng,
  onCreated,
  onCancel,
}: AddMarkerModeProps) {
  const [nodeType, setNodeType] = useState<'odp' | 'ont'>('odp');
  const [referenceId, setReferenceId] = useState('');
  const [address, setAddress] = useState('');
  const [geocoding, setGeocoding] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Trigger reverse geocoding on mount
  useEffect(() => {
    let cancelled = false;
    setGeocoding(true);
    reverseGeocode(lat, lng)
      .then((result) => {
        if (!cancelled) setAddress(result.address);
      })
      .catch(() => {
        if (!cancelled) setAddress('');
      })
      .finally(() => {
        if (!cancelled) setGeocoding(false);
      });
    return () => {
      cancelled = true;
    };
  }, [lat, lng]);

  const handleSubmit = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();
      if (!referenceId.trim()) {
        setError('Reference ID wajib diisi');
        return;
      }

      setSaving(true);
      setError(null);
      try {
        await createNode({
          node_type: nodeType,
          reference_id: referenceId.trim(),
          latitude: lat,
          longitude: lng,
          custom_fields: address ? { lokasi_detail: address } : undefined,
        });
        onCreated?.();
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Gagal membuat node');
      } finally {
        setSaving(false);
      }
    },
    [nodeType, referenceId, lat, lng, address, onCreated],
  );

  return (
    <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-lg">
      <h3 className="mb-3 text-sm font-semibold text-gray-900">
        Tambah Marker Baru
      </h3>

      <form onSubmit={handleSubmit} className="space-y-3">
        {/* Coordinates (read-only) */}
        <div className="grid grid-cols-2 gap-2">
          <div>
            <label className="mb-1 block text-xs text-gray-500">Latitude</label>
            <input
              type="text"
              value={lat.toFixed(6)}
              readOnly
              className="w-full rounded border border-gray-200 bg-gray-50 px-2 py-1.5 text-xs text-gray-600"
            />
          </div>
          <div>
            <label className="mb-1 block text-xs text-gray-500">Longitude</label>
            <input
              type="text"
              value={lng.toFixed(6)}
              readOnly
              className="w-full rounded border border-gray-200 bg-gray-50 px-2 py-1.5 text-xs text-gray-600"
            />
          </div>
        </div>

        {/* Address from geocoding */}
        <div>
          <label className="mb-1 block text-xs text-gray-500">Alamat</label>
          <p className="text-xs text-gray-700">
            {geocoding ? 'Mencari alamat…' : address || 'Alamat tidak ditemukan'}
          </p>
        </div>

        {/* Node type */}
        <div>
          <label className="mb-1 block text-xs font-medium text-gray-700">
            Tipe Node
          </label>
          <select
            value={nodeType}
            onChange={(e) => setNodeType(e.target.value as 'odp' | 'ont')}
            className="w-full rounded-md border border-gray-300 px-2 py-1.5 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
          >
            <option value="odp">ODP</option>
            <option value="ont">ONT</option>
          </select>
        </div>

        {/* Reference ID */}
        <div>
          <label className="mb-1 block text-xs font-medium text-gray-700">
            Reference ID
          </label>
          <input
            type="text"
            value={referenceId}
            onChange={(e) => setReferenceId(e.target.value)}
            placeholder="UUID dari ODP/ONT yang sudah ada"
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
            {saving ? 'Menyimpan…' : 'Simpan'}
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
