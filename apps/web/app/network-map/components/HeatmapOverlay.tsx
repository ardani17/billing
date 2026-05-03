'use client';

import { useEffect } from 'react';
import { useMap } from 'react-leaflet';
import * as L from 'leaflet';
import type { MapNodeWithRef } from '../lib/api';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface HeatmapOverlayProps {
  /** ONT nodes with signal data */
  nodes: MapNodeWithRef[];
  /** Whether the heatmap layer is visible */
  visible: boolean;
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/**
 * Convert signal dBm to a 0–1 intensity value.
 * Good signal (-8 to -20 dBm) → high intensity (green)
 * Weak signal (-20 to -25 dBm) → medium intensity (yellow)
 * Poor signal (-25 to -27 dBm) → lower intensity (orange)
 * Critical (below -27 dBm) → lowest intensity (red)
 */
function signalToIntensity(dbm: number): number {
  // Clamp to reasonable range
  const clamped = Math.max(-35, Math.min(-5, dbm));
  // Map -5..-35 to 1..0
  return (clamped + 35) / 30;
}

// ---------------------------------------------------------------------------
// Legend Component
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
// Component
// ---------------------------------------------------------------------------

/**
 * HeatmapOverlay renders a heatmap layer using leaflet-heat.
 * Falls back to colored circle markers if leaflet-heat is not available.
 */
export default function HeatmapOverlay({
  nodes,
  visible,
}: HeatmapOverlayProps) {
  const map = useMap();

  useEffect(() => {
    if (!visible) return;

    // Filter ONT nodes with signal data
    const heatData = nodes
      .filter((n) => n.node_type === 'ont' && n.signal_dbm != null)
      .map((n) => ({
        lat: n.latitude,
        lng: n.longitude,
        intensity: signalToIntensity(n.signal_dbm!),
        dbm: n.signal_dbm!,
      }));

    if (heatData.length === 0) return;

    // Try to use leaflet-heat if available, otherwise use circle markers
    const layerGroup = L.layerGroup();

    try {
      // Attempt to use L.heatLayer (from leaflet-heat plugin)
      const heat = (L as unknown as Record<string, unknown>).heatLayer;
      if (typeof heat === 'function') {
        const heatPoints = heatData.map((d) => [d.lat, d.lng, d.intensity]);
        const heatLayer = (heat as Function)(heatPoints, {
          radius: 25,
          blur: 15,
          maxZoom: 17,
          gradient: {
            0.0: '#ef4444', // red — critical
            0.3: '#f97316', // orange — poor
            0.5: '#eab308', // yellow — weak
            0.8: '#22c55e', // green — good
            1.0: '#16a34a', // dark green — excellent
          },
        });
        layerGroup.addLayer(heatLayer);
      } else {
        throw new Error('leaflet-heat not available');
      }
    } catch {
      // Fallback: colored circle markers
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
