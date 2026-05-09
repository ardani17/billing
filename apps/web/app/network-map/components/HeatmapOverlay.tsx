'use client';

import { useEffect } from 'react';
import { useMap } from 'react-leaflet';
import * as L from 'leaflet';
import type { MapNodeWithRef } from '../lib/api';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface HeatmapOverlayProps {
  /** Node ONT dengan data sinyal*/
  nodes: MapNodeWithRef[];
  /** Menentukan apakah layer heatmap terlihat*/
  visible: boolean;
}

// ---------------------------------------------------------------------------
// Fungsi bantus
// ---------------------------------------------------------------------------

/**
 * Konversi sinyal dBm ke nilai intensitas 0-1.
 * Sinyal bagus (-8 to -20 dBm) → intensitas tinggi (hijau)
 * Sinyal lemah (-20 to -25 dBm) → intensitas sedang (kuning)
 * Sinyal buruk (-25 to -27 dBm) → intensitas lebih rendah (oranye)
 * Kritis (di bawah -27 dBm) → intensitas paling rendah (merah)
 */
function signalToIntensity(dbm: number): number {
  // Clamp to reasonable range
  const clamped = Math.max(-35, Math.min(-5, dbm));
  // Petakan -5..-35 ke 1..0
  return (clamped + 35) / 30;
}

// ---------------------------------------------------------------------------
// Legend Komponen
// ---------------------------------------------------------------------------

export function HeatmapLegend() {
  return (
    <div className="rounded-lg bg-white p-3 shadow-lg">
      <h4 className="mb-2 text-xs font-semibold uppercase tracking-wide text-gray-500">
        Kualitas Signal
      </h4>
      <div className="space-y-1">
        <LegendItem color="bg-green-500" label="Baik" range="-8 s/d -20 dBm" />
        <LegendItem color="bg-yellow-500" label="Lemah" range="-20 s/d -25 dBm" />
        <LegendItem color="bg-orange-500" label="Buruk" range="-25 s/d -27 dBm" />
        <LegendItem color="bg-red-500" label="Kritis" range="< -27 dBm" />
      </div>
    </div>
  );
}

function LegendItem({
  color,
  label,
  range,
}: {
  color: string;
  label: string;
  range: string;
}) {
  return (
    <div className="flex items-center gap-2">
      <span className={`inline-block h-3 w-3 rounded-sm ${color}`} />
      <span className="text-xs text-gray-700">{label}</span>
      <span className="text-xs text-gray-400">({range})</span>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Komponen
// ---------------------------------------------------------------------------

/**
 * HeatmapOverlay renders a heatmap layer using leaflet-heat.
 */
export default function HeatmapOverlay({
  nodes,
  visible,
}: HeatmapOverlayProps) {
  const map = useMap();

  useEffect(() => {
    if (!visible) return;

    // Filter Node ONT dengan data sinyal
    const heatData = nodes
      .filter((n) => n.node_type === 'ont' && n.signal_dbm != null)
      .map((n) => ({
        lat: n.latitude,
        lng: n.longitude,
        intensity: signalToIntensity(n.signal_dbm!),
        dbm: n.signal_dbm!,
      }));

    if (heatData.length === 0) return;

    const layerGroup = L.layerGroup();

    try {
      const heat = (L as unknown as Record<string, unknown>).heatLayer;
      if (typeof heat === 'function') {
        const heatPoints = heatData.map((d) => [d.lat, d.lng, d.intensity]);
        const heatLayer = (heat as Function)(heatPoints, {
          radius: 25,
          blur: 15,
          maxZoom: 17,
          gradient: {
            0.0: '#ef4444', // merah - kritis
            0.3: '#f97316', // oranye - buruk
            0.5: '#eab308', // kuning - lemah
            0.8: '#22c55e', // hijau - bagus
            1.0: '#16a34a', // dark green - excellent
          },
        });
        layerGroup.addLayer(heatLayer);
      } else {
        throw new Error('leaflet-heat not available');
      }
    } catch {
      // Cadangan: marker lingkaran berwarna
      for (const d of heatData) {
        let color = '#ef4444'; // red
        if (d.dbm >= -20) color = '#22c55e'; // green
        else if (d.dbm >= -25) color = '#eab308'; // yellow
        else if (d.dbm >= -27) color = '#f97316'; // orange

        const circle = L.circleMarker([d.lat, d.lng], {
          radius: 8,
          fillColor: color,
          fillOpacity: 0.5,
          stroke: false,
        });
        layerGroup.addLayer(circle);
      }
    }

    map.addLayer(layerGroup);

    return () => {
      map.removeLayer(layerGroup);
    };
  }, [map, nodes, visible]);

  return null;
}
