'use client';

import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { Marker, Popup } from 'react-leaflet';
import * as L from 'leaflet';
import { updateNode, reverseGeocode } from '../lib/api';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface DraggableMarkerProps {
  nodeId: string;
  initialLat: number;
  initialLng: number;
  onConfirm?: () => void;
  onCancel: () => void;
}

// ---------------------------------------------------------------------------
// Draggable marker icon
// ---------------------------------------------------------------------------

function draggableIcon(): L.DivIcon {
  return L.divIcon({
    html: `<div style="
      background:#3b82f6;
      width:24px;
      height:24px;
      border-radius:50%;
      border:3px solid #fff;
      box-shadow:0 2px 8px rgba(59,130,246,.5);
      cursor:grab;
    "></div>`,
    className: 'draggable-marker',
    iconSize: L.point(24, 24),
    iconAnchor: L.point(12, 12),
  });
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export default function DraggableMarker({
  nodeId,
  initialLat,
  initialLng,
  onConfirm,
  onCancel,
}: DraggableMarkerProps) {
  const [position, setPosition] = useState<[number, number]>([
    initialLat,
    initialLng,
  ]);
  const [address, setAddress] = useState('');
  const [geocoding, setGeocoding] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const markerRef = useRef<L.Marker | null>(null);

  const icon = useMemo(() => draggableIcon(), []);

  // Reverse geocode on position change
  useEffect(() => {
    let cancelled = false;
    setGeocoding(true);
    reverseGeocode(position[0], position[1])
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
  }, [position]);

  const eventHandlers = useMemo(
    () => ({
      dragend() {
        const marker = markerRef.current;
        if (marker) {
          const latlng = marker.getLatLng();
          setPosition([latlng.lat, latlng.lng]);
        }
      },
    }),
    [],
  );

  const handleConfirm = useCallback(async () => {
    setSaving(true);
    setError(null);
    try {
      await updateNode(nodeId, {
        latitude: position[0],
        longitude: position[1],
      });
      onConfirm?.();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Gagal menyimpan lokasi');
    } finally {
      setSaving(false);
    }
  }, [nodeId, position, onConfirm]);

  const hasMoved =
    position[0] !== initialLat || position[1] !== initialLng;

  return (
    <Marker
      ref={markerRef}
      position={position}
      icon={icon}
      draggable
      eventHandlers={eventHandlers}
    >
      <Popup minWidth={220} closeButton={false}>
        <div className="space-y-2">
          <p className="text-xs font-semibold text-gray-900">Edit Lokasi</p>
          <p className="text-xs text-gray-600">
            {position[0].toFixed(6)}, {position[1].toFixed(6)}
          </p>
          <p className="text-xs text-gray-500">
            {geocoding ? 'Mencari alamat…' : address || 'Alamat tidak ditemukan'}
          </p>

          {hasMoved && (
            <p className="text-xs text-blue-600">
              Lokasi berubah — seret marker atau konfirmasi
            </p>
          )}

          {error && <p className="text-xs text-red-500">{error}</p>}

          <div className="flex gap-1">
            <button
              onClick={handleConfirm}
              disabled={saving || !hasMoved}
              className="flex-1 rounded bg-blue-600 px-2 py-1 text-xs font-medium text-white hover:bg-blue-700 disabled:opacity-50"
            >
              {saving ? '…' : 'Konfirmasi'}
            </button>
            <button
              onClick={onCancel}
              className="rounded border border-gray-300 px-2 py-1 text-xs text-gray-600 hover:bg-gray-50"
            >
              Batal
            </button>
          </div>
        </div>
      </Popup>
    </Marker>
  );
}
