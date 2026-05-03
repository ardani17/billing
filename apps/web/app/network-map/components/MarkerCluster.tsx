'use client';

import { useEffect } from 'react';
import { useMap } from 'react-leaflet';
import * as L from 'leaflet';
import 'leaflet.markercluster';
import type { MapNodeWithRef } from '../lib/api';

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/** Determine cluster icon color based on the statuses of child ONT markers. */
function clusterColor(
  statuses: string[],
): 'green' | 'yellow' | 'red' {
  const hasOffline = statuses.some(
    (s) => s === 'offline' || s === 'los',
  );
  const hasWeak = statuses.some((s) => s === 'weak');

  if (hasOffline) return 'red';
  if (hasWeak) return 'yellow';
  return 'green';
}

const CLUSTER_COLORS: Record<string, string> = {
  green: '#22c55e',
  yellow: '#eab308',
  red: '#ef4444',
};

function createClusterIcon(cluster: L.MarkerCluster): L.DivIcon {
  const markers = cluster.getAllChildMarkers();
  const statuses = markers.map(
    (m) => ((m.options as { status?: string }).status ?? 'online'),
  );
  const color = clusterColor(statuses);
  const count = cluster.getChildCount();

  return L.divIcon({
    html: `<div style="
      background:${CLUSTER_COLORS[color]};
      color:#fff;
      border-radius:50%;
      width:36px;
      height:36px;
      display:flex;
      align-items:center;
      justify-content:center;
      font-weight:600;
      font-size:13px;
      border:2px solid #fff;
      box-shadow:0 2px 6px rgba(0,0,0,.3);
    ">${count}</div>`,
    className: 'marker-cluster-custom',
    iconSize: L.point(36, 36),
  });
}

/** Create a small circle marker icon for an ONT node. */
function ontMarkerIcon(status?: string): L.DivIcon {
  let bg = '#9ca3af'; // gray — pending
  if (status === 'online') bg = '#22c55e';
  else if (status === 'weak') bg = '#eab308';
  else if (status === 'offline' || status === 'los') bg = '#ef4444';

  return L.divIcon({
    html: `<div style="
      background:${bg};
      width:12px;
      height:12px;
      border-radius:50%;
      border:2px solid #fff;
      box-shadow:0 1px 3px rgba(0,0,0,.3);
    "></div>`,
    className: 'ont-marker',
    iconSize: L.point(12, 12),
    iconAnchor: L.point(6, 6),
  });
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface MarkerClusterProps {
  nodes: MapNodeWithRef[];
  onNodeClick?: (nodeId: string) => void;
}

/**
 * MarkerCluster renders ONT nodes using Leaflet.markercluster.
 * Clusters are colored based on the aggregate status of contained markers:
 *   green  — all online
 *   yellow — some weak signal
 *   red    — some offline / LOS
 */
export default function MarkerCluster({
  nodes,
  onNodeClick,
}: MarkerClusterProps) {
  const map = useMap();

  useEffect(() => {
    const clusterGroup = L.markerClusterGroup({
      iconCreateFunction: createClusterIcon,
      maxClusterRadius: 50,
      spiderfyOnMaxZoom: true,
      showCoverageOnHover: false,
      zoomToBoundsOnClick: true,
    });

    for (const node of nodes) {
      const marker = L.marker([node.latitude, node.longitude], {
        icon: ontMarkerIcon(node.status),
        // Attach status so the cluster icon function can read it
        status: node.status,
      } as L.MarkerOptions & { status?: string });

      if (onNodeClick) {
        marker.on('click', () => onNodeClick(node.id));
      }

      clusterGroup.addLayer(marker);
    }

    map.addLayer(clusterGroup);

    return () => {
      map.removeLayer(clusterGroup);
    };
  }, [map, nodes, onNodeClick]);

  return null;
}
