'use client';

import { useCallback, useMemo, useState } from 'react';
import dynamic from 'next/dynamic';
import {
  Broadcast,
  CaretDown,
  CellSignalHigh,
  DotsThreeVertical,
  Funnel,
  MapPinArea,
  Path,
  SquaresFour,
  X,
} from '@phosphor-icons/react';
import { useMapNodes } from './hooks/useMapNodes';
import { useCableRoutes } from './hooks/useCableRoutes';
import type { BoundingBox, MapNodeWithRef, NodeFilters, SearchResult } from './lib/api';
import DetailPanel from './components/DetailPanel';
import LayerControl, {
  DEFAULT_LAYERS,
  type LayerVisibility,
} from './components/LayerControl';
import SearchBar from './components/SearchBar';
import TopologyView from './components/TopologyView';

const MapCanvas = dynamic(() => import('./components/MapCanvas'), {
  ssr: false,
  loading: () => <MapSkeleton />,
});

function MapSkeleton() {
  return (
    <div className="flex h-full w-full items-center justify-center bg-slate-100 text-sm text-slate-500">
      Memuat peta jaringan...
    </div>
  );
}

function countSummary(nodes: MapNodeWithRef[]) {
  const summary = {
    olt: 0,
    odp: 0,
    ont: 0,
    online: 0,
    weak: 0,
    offline: 0,
    pending: 0,
  };

  for (const node of nodes) {
    if (node.node_type === 'olt') summary.olt += 1;
    if (node.node_type === 'odp') summary.odp += 1;
    if (node.node_type === 'ont') {
      summary.ont += 1;
      if (node.status === 'online' || node.status === 'active') summary.online += 1;
      else if (node.status === 'weak') summary.weak += 1;
      else if (node.status === 'offline' || node.status === 'los') summary.offline += 1;
      else summary.pending += 1;
    }
  }

  return summary;
}

function compactNumber(value: number) {
  return new Intl.NumberFormat('id-ID').format(value);
}

function StatTile({
  label,
  value,
  tone,
}: {
  label: string;
  value: number | string;
  tone?: 'green' | 'amber' | 'red' | 'blue';
}) {
  const tones = {
    green: 'text-emerald-700',
    amber: 'text-amber-700',
    red: 'text-red-700',
    blue: 'text-sky-800',
  };

  return (
    <div className="min-w-0 border-r border-slate-200 px-3 py-2 last:border-r-0">
      <div className={`font-mono text-lg font-semibold ${tone ? tones[tone] : 'text-slate-950'}`}>
        {value}
      </div>
      <div className="truncate text-[11px] font-medium uppercase tracking-wide text-slate-500">
        {label}
      </div>
    </div>
  );
}

function MapLegend() {
  return (
    <div className="grid grid-cols-2 gap-x-4 gap-y-2 text-xs text-slate-600 sm:grid-cols-4">
      <span className="flex items-center gap-2">
        <span className="h-1 w-8 rounded bg-[#0f3d5e]" />
        Backbone
      </span>
      <span className="flex items-center gap-2">
        <span className="h-1 w-8 rounded bg-emerald-600" />
        Drop aktif
      </span>
      <span className="flex items-center gap-2">
        <span className="h-1 w-8 rounded border-t border-dashed border-red-600" />
        Drop gangguan
      </span>
      <span className="flex items-center gap-2">
        <span className="h-3 w-3 rounded-full bg-amber-500" />
        Signal lemah
      </span>
    </div>
  );
}

export default function MapPage() {
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null);
  const [layers, setLayers] = useState<LayerVisibility>(DEFAULT_LAYERS);
  const [filters, setFilters] = useState<NodeFilters>({});
  const [showLayers, setShowLayers] = useState(false);
  const [showTopology, setShowTopology] = useState(true);
  const [focusTarget, setFocusTarget] = useState<{ lat: number; lng: number; zoom?: number }>();

  const {
    nodes,
    loading: nodesLoading,
    error: nodesError,
    onBoundsChange: onNodesBoundsChange,
  } = useMapNodes({ filters });

  const {
    cables,
    loading: cablesLoading,
    error: cablesError,
    onBoundsChange: onCablesBoundsChange,
  } = useCableRoutes();

  const summary = useMemo(() => countSummary(nodes), [nodes]);

  const visibleOntCount = useMemo(
    () =>
      nodes.filter((node) => {
        if (node.node_type !== 'ont') return false;
        const isOffline = node.status === 'offline' || node.status === 'los';
        return isOffline ? layers.ontOffline : layers.ontOnline;
      }).length,
    [nodes, layers],
  );

  const handleBoundsChange = useCallback(
    (bounds: BoundingBox) => {
      onNodesBoundsChange(bounds);
      onCablesBoundsChange(bounds);
    },
    [onNodesBoundsChange, onCablesBoundsChange],
  );

  const handleLayerChange = useCallback(
    (layer: keyof LayerVisibility, visible: boolean) => {
      setLayers((current) => ({ ...current, [layer]: visible }));
    },
    [],
  );

  const handleSearchSelect = useCallback((result: SearchResult) => {
    setFocusTarget({ lat: result.latitude, lng: result.longitude, zoom: 17 });

    const match = nodes.find(
      (node) =>
        node.node_type === result.type &&
        Math.abs(node.latitude - result.latitude) < 0.000001 &&
        Math.abs(node.longitude - result.longitude) < 0.000001,
    );
    if (match) setSelectedNodeId(match.id);
  }, [nodes]);

  const handleTopologyClick = useCallback((_nodeType: string, nodeId: string) => {
    const node = nodes.find((item) => item.id === nodeId);
    if (node) {
      setFocusTarget({ lat: node.latitude, lng: node.longitude, zoom: 17 });
    }
    setSelectedNodeId(nodeId);
  }, [nodes]);

  const activeError = nodesError || cablesError;

  return (
    <div className="flex h-[calc(100dvh-8rem)] min-h-[38rem] min-w-0 flex-col overflow-hidden bg-slate-100 text-slate-950 md:flex-row">
      <section className="relative min-h-0 min-w-0 flex-1">
        <MapCanvas
          nodes={nodes}
          cables={cables}
          selectedNodeId={selectedNodeId ?? undefined}
          layers={layers}
          focusTarget={focusTarget}
          onNodeClick={setSelectedNodeId}
          onBoundsChange={handleBoundsChange}
        />

        <div className="pointer-events-none absolute inset-x-0 top-0 z-[1000] p-3 sm:p-4">
          <div className="pointer-events-auto mx-auto flex max-w-6xl flex-col gap-3">
            <div className="flex min-w-0 flex-col gap-3 rounded-lg border border-white/70 bg-white/95 p-3 shadow-sm backdrop-blur md:flex-row md:items-center">
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2 text-xs font-semibold uppercase tracking-wide text-slate-500">
                  <MapPinArea size={15} weight="bold" />
                  Peta Jaringan FTTH
                </div>
                <div className="mt-1 max-w-full text-sm text-slate-600">
                  OLT, ODP, ONT, jalur backbone, dan drop cable dalam satu view operasional.
                </div>
              </div>
              <SearchBar onSelect={handleSearchSelect} />
              <div className="flex shrink-0 gap-2">
                <button
                  type="button"
                  onClick={() => setShowTopology((value) => !value)}
                  className="inline-flex h-10 items-center gap-2 rounded-md border border-slate-200 bg-white px-3 text-sm font-medium text-slate-700 shadow-sm transition hover:bg-slate-50 active:translate-y-px"
                >
                  <SquaresFour size={17} />
                  <span className="hidden sm:inline">Topology</span>
                </button>
                <button
                  type="button"
                  onClick={() => setShowLayers((value) => !value)}
                  className="inline-flex h-10 items-center gap-2 rounded-md border border-slate-200 bg-white px-3 text-sm font-medium text-slate-700 shadow-sm transition hover:bg-slate-50 active:translate-y-px"
                >
                  <Funnel size={17} />
                  <span className="hidden sm:inline">Layer</span>
                  <CaretDown size={14} />
                </button>
              </div>
            </div>

            <div className="grid overflow-hidden rounded-lg border border-white/70 bg-white/95 shadow-sm backdrop-blur sm:grid-cols-4 md:max-w-2xl">
              <StatTile label="OLT" value={compactNumber(summary.olt)} tone="blue" />
              <StatTile label="ODP" value={compactNumber(summary.odp)} />
              <StatTile label="ONT aktif" value={compactNumber(summary.online)} tone="green" />
              <StatTile label="Gangguan" value={compactNumber(summary.offline + summary.weak)} tone={summary.offline ? 'red' : 'amber'} />
            </div>
          </div>
        </div>

        {showLayers && (
          <div className="absolute right-3 top-[176px] z-[1000] w-[min(18rem,calc(100vw-1.5rem))] sm:right-4 md:top-[116px]">
            <LayerControl
              layers={layers}
              onLayerChange={handleLayerChange}
              filters={filters}
              onFilterChange={setFilters}
              visibleCount={visibleOntCount}
              totalCount={summary.ont}
              onResetFilters={() => setFilters({})}
            />
          </div>
        )}

        {showTopology && (
          <div className="absolute bottom-4 left-3 z-[1000] hidden max-h-[42vh] w-80 overflow-hidden rounded-lg border border-slate-200 bg-white shadow-sm md:block">
            <div className="flex items-center justify-between border-b border-slate-200 px-4 py-3">
              <div className="flex items-center gap-2 text-sm font-semibold">
                <Path size={17} />
                Topology
              </div>
              <button
                type="button"
                onClick={() => setShowTopology(false)}
                className="rounded p-1 text-slate-400 hover:bg-slate-100 hover:text-slate-700"
                aria-label="Tutup topology"
              >
                <X size={15} />
              </button>
            </div>
            <TopologyView nodes={nodes} onNodeClick={handleTopologyClick} />
          </div>
        )}

        <div className="absolute bottom-4 right-3 z-[1000] hidden rounded-lg border border-slate-200 bg-white px-4 py-3 shadow-sm sm:block">
          <MapLegend />
        </div>

        {(nodesLoading || cablesLoading) && (
          <div className="absolute left-1/2 top-[178px] z-[1000] -translate-x-1/2 rounded-full border border-slate-200 bg-white px-3 py-1 text-xs font-medium text-slate-500 shadow-sm md:top-[120px]">
            Memuat data peta...
          </div>
        )}

        {activeError && (
          <div className="absolute left-3 right-3 top-[178px] z-[1000] rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 shadow-sm md:left-auto md:right-4 md:top-[116px] md:w-96">
            {activeError}
          </div>
        )}

        {nodes.length === 0 && !nodesLoading && !activeError && (
          <div className="absolute left-1/2 top-1/2 z-[999] w-[min(24rem,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 rounded-lg border border-slate-200 bg-white p-5 text-center shadow-sm">
            <Broadcast className="mx-auto text-slate-400" size={28} />
            <h2 className="mt-3 text-base font-semibold text-slate-950">
              Belum ada node di area ini
            </h2>
            <p className="mt-1 text-sm text-slate-600">
              Tambahkan OLT, ODP, dan ONT dari modul jaringan, lalu pasang koordinatnya ke peta.
            </p>
          </div>
        )}
      </section>

      <aside
        className={`
          fixed inset-x-0 bottom-0 z-[1001] max-h-[72dvh] min-h-[18rem] overflow-y-auto
          rounded-t-xl border-t border-slate-200 bg-white shadow-2xl transition-transform duration-300
          ${selectedNodeId ? 'translate-y-0' : 'translate-y-[calc(100%-4rem)]'}
          md:static md:h-full md:max-h-none md:min-h-0 md:w-[22rem] md:translate-y-0 md:rounded-none md:border-l md:border-t-0 md:shadow-none
        `}
      >
        {selectedNodeId ? (
          <DetailPanel
            nodeId={selectedNodeId}
            onClose={() => setSelectedNodeId(null)}
            onNavigate={(lat, lng) => setFocusTarget({ lat, lng, zoom: 17 })}
          />
        ) : (
          <div className="flex h-full min-h-[14rem] flex-col justify-center p-5 text-slate-500">
            <div className="flex items-center gap-2 text-sm font-semibold text-slate-800">
              <CellSignalHigh size={18} />
              Detail node
            </div>
            <p className="mt-2 text-sm">
              Pilih marker OLT, ODP, atau ONT untuk melihat kapasitas, signal, foto lokasi, dan riwayat perubahan.
            </p>
            <div className="mt-4 inline-flex w-fit items-center gap-2 rounded-md border border-slate-200 px-3 py-2 text-xs font-medium text-slate-600 md:hidden">
              <DotsThreeVertical size={16} />
              Geser panel untuk membuka
            </div>
          </div>
        )}
      </aside>
    </div>
  );
}
