'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import { useMap } from 'react-leaflet';
import * as L from 'leaflet';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface MyLocationProps {
  /** Target node coordinates for distance calculation */
  targetLat?: number;
  targetLng?: number;
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function haversineDistance(
  lat1: number,
  lng1: number,
  lat2: number,
  lng2: number,
): number {
  const R = 6371000;
  const toRad = (deg: number) => (deg * Math.PI) / 180;
  const dLat = toRad(lat2 - lat1);
  const dLng = toRad(lng2 - lng1);
  const a =
    Math.sin(dLat / 2) ** 2 +
    Math.cos(toRad(lat1)) * Math.cos(toRad(lat2)) * Math.sin(dLng / 2) ** 2;
  return R * 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a));
}

function formatDistance(meters: number): string {
  if (meters >= 1000) return `${(meters / 1000).toFixed(1)} km`;
  return `${Math.round(meters)} m`;
}

/** Create a pulsing blue marker icon for user location. */
function userLocationIcon(): L.DivIcon {
  return L.divIcon({
    html: `<div style="position:relative">
      <div style="
        background:rgba(59,130,246,0.2);
        width:32px;
        height:32px;
        border-radius:50%;
        position:absolute;
        top:-10px;
        left:-10px;
        animation:pulse 2s infinite;
      "></div>
      <div style="
        background:#3b82f6;
        width:12px;
        height:12px;
        border-radius:50%;
        border:2px solid #fff;
        box-shadow:0 1px 4px rgba(0,0,0,.3);
      "></div>
    </div>
    <style>
      @keyframes pulse {
        0% { transform:scale(1); opacity:1; }
        100% { transform:scale(2.5); opacity:0; }
      }
    </style>`,
    className: 'user-location-marker',
    iconSize: L.point(12, 12),
    iconAnchor: L.point(6, 6),
  });
}

// ---------------------------------------------------------------------------
// Navigation deep links
// ---------------------------------------------------------------------------

function openGoogleMaps(lat: number, lng: number) {
  window.open(
    `https://www.google.com/maps/dir/?api=1&destination=${lat},${lng}`,
    '_blank',
  );
}

function openWaze(lat: number, lng: number) {
  window.open(
    `https://waze.com/ul?ll=${lat},${lng}&navigate=yes`,
    '_blank',
  );
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export default function MyLocation({ targetLat, targetLng }: MyLocationProps) {
  const map = useMap();
  const [userPos, setUserPos] = useState<[number, number] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [locating, setLocating] = useState(false);
  const markerRef = useRef<L.Marker | null>(null);

  // Clean up marker on unmount
  useEffect(() => {
    return () => {
      if (markerRef.current) {
        map.removeLayer(markerRef.current);
      }
    };
  }, [map]);

  // Update marker when position changes
  useEffect(() => {
    if (!userPos) return;

    if (markerRef.current) {
      markerRef.current.setLatLng(userPos);
    } else {
      markerRef.current = L.marker(userPos, {
        icon: userLocationIcon(),
        zIndexOffset: 1000,
      }).addTo(map);
    }
  }, [map, userPos]);

  const handleLocate = useCallback(() => {
    if (!navigator.geolocation) {
      setError('Geolocation tidak didukung browser ini');
      return;
    }

    setLocating(true);
    setError(null);

    navigator.geolocation.getCurrentPosition(
      (pos) => {
        const coords: [number, number] = [
          pos.coords.latitude,
          pos.coords.longitude,
        ];
        setUserPos(coords);
        map.flyTo(coords, 16);
        setLocating(false);
      },
      (err) => {
        setError(
          err.code === err.PERMISSION_DENIED
            ? 'Izin lokasi ditolak. Aktifkan GPS untuk fitur navigasi.'
            : 'Gagal mendapatkan lokasi',
        );
        setLocating(false);
      },
      { enableHighAccuracy: true, timeout: 10000 },
    );
  }, [map]);

  const distance =
    userPos && targetLat != null && targetLng != null
      ? haversineDistance(userPos[0], userPos[1], targetLat, targetLng)
      : null;

  return (
    <div className="flex flex-col gap-2">
      {/* Locate button */}
      <button
        onClick={handleLocate}
        disabled={locating}
        className="flex items-center gap-1.5 rounded-lg bg-white px-3 py-2 text-sm shadow-lg hover:bg-gray-50 disabled:opacity-50"
        title="Lokasi Saya"
      >
        <span>{locating ? '⏳' : '📍'}</span>
        <span className="hidden md:inline">Lokasi Saya</span>
      </button>

      {/* Distance indicator */}
      {distance != null && (
        <div className="rounded-lg bg-white px-3 py-1.5 text-xs text-gray-600 shadow">
          Jarak: {formatDistance(distance)}
        </div>
      )}

      {/* Navigation buttons */}
      {userPos && targetLat != null && targetLng != null && (
        <div className="flex gap-1">
          <button
            onClick={() => openGoogleMaps(targetLat, targetLng)}
            className="flex-1 rounded bg-blue-600 px-2 py-1 text-xs text-white hover:bg-blue-700"
          >
            Google Maps
          </button>
          <button
            onClick={() => openWaze(targetLat, targetLng)}
            className="flex-1 rounded bg-purple-600 px-2 py-1 text-xs text-white hover:bg-purple-700"
          >
            Waze
          </button>
        </div>
      )}

      {/* Error message */}
      {error && (
        <p className="rounded bg-red-50 px-2 py-1 text-xs text-red-600">
          {error}
        </p>
      )}
    </div>
  );
}
