'use client';

import { useEffect, useMemo } from 'react';
import {
  MapContainer,
  Marker,
  Polyline,
  TileLayer,
  Tooltip,
  useMap,
  useMapEvents,
} from 'react-leaflet';
import * as L from 'leaflet';
import type { BoundingBox, CableRoute, MapNodeWithRef } from '../lib/api';
import type { LayerVisibility } from './LayerControl';
import MarkerCluster from './MarkerCluster';

import 'leaflet/dist/leaflet.css';

const BACKBONE_STYLE: L.PolylineOptions = {
  color: '#0f3d5e',
  weight: 4,
  opacity: 0.9,
};

const DROP_ONLINE_STYLE: L.PolylineOptions = {
  color: '#16a34a',
  weight: 2,
  opacity: 0.8,
};

const DROP_OFFLINE_STYLE: L.PolylineOptions = {
  color: '#dc2626',
  weight: 2,
  opacity: 0.82,
  dashArray: '8 4',
};

function oltIcon(selected = false): L.DivIcon {
  const size = selected ? 32 : 26;
  return L.divIcon({
    html: `<div style="
      background:#0f3d5e;
      width:${size}px;
      height:${size}px;
      display:flex;
      align-items:center;
      justify-content:center;
      border-radius:6px;
      border:2px solid #fff;
      box-shadow:0 8px 18px rgba(15,61,94,.28);
      color:#fff;
      font:700 10px/1 system-ui, sans-serif;
      letter-spacing:.02em;
    ">OLT</div>`,
    className: 'olt-marker',
    iconSize: L.point(size, size),
    iconAnchor: L.point(size / 2, size / 2),
  });
}

function odpIcon(selected = false): L.DivIcon {
  const size = selected ? 24 : 18;
  return L.divIcon({
    html: `<div style="
      background:#2563eb;
      width:${size}px;
      height:${size}px;
      border-radius:4px;
      border:2px solid #fff;
      box-shadow:0 6px 14px rgba(37,99,235,.24);
    "></div>`,
    className: 'odp-marker',
    iconSize: L.point(size, size),
    iconAnchor: L.point(size / 2, size / 2),
  });
}

function BoundsWatcher({
  onBoundsChange,
}: {
  onBoundsChange: (bounds: BoundingBox) => void;
}) {
  useMapEvents({
    moveend(e) {
      const map = e.target as L.Map;
      const b = map.getBounds();
      onBoundsChange({
        minLat: b.getSouth(),
        minLng: b.getWest(),
        maxLat: b.getNorth(),
        maxLng: b.getEast(),
      });
    },
    load(e) {
      const map = e.target as L.Map;
      const b = map.getBounds();
      onBoundsChange({
        minLat: b.getSouth(),
        minLng: b.getWest(),
        maxLat: b.getNorth(),
        maxLng: b.getEast(),
      });
    },
  });

  return null;
}

function MapClickHandler({
  onClick,
}: {
  onClick: (lat: number, lng: number) => void;
}) {
  useMapEvents({
    click(e) {
      onClick(e.latlng.lat, e.latlng.lng);
    },
  });

  return null;
}

function FocusMap({
  target,
}: {
  target?: { lat: number; lng: number; zoom?: number };
}) {
  const map = useMap();

  useEffect(() => {
    if (!target) return;

    map.flyTo(
      [target.lat, target.lng],
      target.zoom ?? Math.max(map.getZoom(), 16),
      { duration: 0.65 },
    );
  }, [map, target]);

  return null;
}

interface MapCanvasProps {
  nodes: MapNodeWithRef[];
  cables: CableRoute[];
  selectedNodeId?: string;
  onNodeClick?: (nodeId: string) => void;
  onMapClick?: (lat: number, lng: number) => void;
  onBoundsChange: (bounds: BoundingBox) => void;
  layers?: LayerVisibility;
  focusTarget?: { lat: number; lng: number; zoom?: number };
  center?: [number, number];
  zoom?: number;
}

export default function MapCanvas({
  nodes,
  cables,
  selectedNodeId,
  onNodeClick,
  onMapClick,
  onBoundsChange,
  layers,
  focusTarget,
  center = [-6.2, 106.816],
  zoom = 12,
}: MapCanvasProps) {
  const { oltNodes, odpNodes, ontNodes } = useMemo(() => {
    const olt: MapNodeWithRef[] = [];
    const odp: MapNodeWithRef[] = [];
    const ont: MapNodeWithRef[] = [];

    for (const node of nodes) {
      if (node.node_type === 'olt') olt.push(node);
      else if (node.node_type === 'odp') odp.push(node);
      else ont.push(node);
    }

    return { oltNodes: olt, odpNodes: odp, ontNodes: ont };
  }, [nodes]);

  const visibleCables = useMemo(
    () =>
      cables.filter((cable) =>
        cable.route_type === 'backbone'
          ? layers?.cableBackbone ?? true
          : layers?.cableDrop ?? true,
      ),
    [cables, layers],
  );

  const visibleOntNodes = useMemo(
    () =>
      ontNodes.filter((node) => {
        const isOffline = node.status === 'offline' || node.status === 'los';
        return isOffline
          ? layers?.ontOffline ?? true
          : layers?.ontOnline ?? true;
      }),
    [ontNodes, layers],
  );

  const cableStyle = (cable: CableRoute): L.PolylineOptions => {
    if (cable.route_type === 'backbone') return BACKBONE_STYLE;
    const isOffline =
      cable.to_node_status === 'offline' || cable.to_node_status === 'los';
    return isOffline ? DROP_OFFLINE_STYLE : DROP_ONLINE_STYLE;
  };

  const tileUrl = layers?.satellite
    ? 'https://server.arcgisonline.com/ArcGIS/rest/services/World_Imagery/MapServer/tile/{z}/{y}/{x}'
    : 'https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png';

  const attribution = layers?.satellite
    ? 'Tiles &copy; Esri, Maxar, Earthstar Geographics'
    : '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors';

  return (
    <MapContainer
      center={center}
      zoom={zoom}
      className="h-full w-full"
      zoomControl
      attributionControl
    >
      <TileLayer attribution={attribution} url={tileUrl} />
      <BoundsWatcher onBoundsChange={onBoundsChange} />
      <FocusMap target={focusTarget} />

      {onMapClick && <MapClickHandler onClick={onMapClick} />}

      {visibleCables.map((cable) => (
        <Polyline
          key={cable.id}
          positions={cable.coordinates.map(([lat, lng]) => [lat, lng])}
          pathOptions={cableStyle(cable)}
        />
      ))}

      {(layers?.olt ?? true) &&
        oltNodes.map((node) => (
          <Marker
            key={node.id}
            position={[node.latitude, node.longitude]}
            icon={oltIcon(node.id === selectedNodeId)}
            eventHandlers={
              onNodeClick ? { click: () => onNodeClick(node.id) } : undefined
            }
          >
            <Tooltip permanent direction="top" offset={[0, -12]} opacity={0.92}>
              <span className="text-[11px] font-semibold">
                {node.name ?? 'OLT'}
              </span>
            </Tooltip>
          </Marker>
        ))}

      {(layers?.odp ?? true) &&
        odpNodes.map((node) => (
          <Marker
            key={node.id}
            position={[node.latitude, node.longitude]}
            icon={odpIcon(node.id === selectedNodeId)}
            eventHandlers={
              onNodeClick ? { click: () => onNodeClick(node.id) } : undefined
            }
          >
            <Tooltip permanent direction="right" offset={[12, 0]} opacity={0.88}>
              <span className="text-[11px] font-medium">
                {node.name ?? 'ODP'}
              </span>
            </Tooltip>
          </Marker>
        ))}

      <MarkerCluster nodes={visibleOntNodes} onNodeClick={onNodeClick} />
    </MapContainer>
  );
}
