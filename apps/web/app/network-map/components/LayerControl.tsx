'use client';

import { useCallback } from 'react';
import {
  BoundingBox,
  Broadcast,
  Circle,
  GlobeHemisphereEast,
  Path,
  Pulse,
  Square,
  ThermometerHot,
} from '@phosphor-icons/react';
import type { NodeFilters } from '../lib/api';

export interface LayerVisibility {
  olt: boolean;
  odp: boolean;
  ontOnline: boolean;
  ontOffline: boolean;
  cableBackbone: boolean;
  cableDrop: boolean;
  area: boolean;
  satellite: boolean;
  heatmap: boolean;
}

export const DEFAULT_LAYERS: LayerVisibility = {
  olt: true,
  odp: true,
  ontOnline: true,
  ontOffline: true,
  cableBackbone: true,
  cableDrop: false,
  area: false,
  satellite: false,
  heatmap: false,
};

interface LayerControlProps {
  layers: LayerVisibility;
  onLayerChange: (layer: keyof LayerVisibility, visible: boolean) => void;
  filters: NodeFilters;
  onFilterChange: (filters: NodeFilters) => void;
  visibleCount: number;
  totalCount: number;
  onResetFilters: () => void;
}

const LAYER_DEFS = [
  { key: 'olt', label: 'OLT', icon: Broadcast },
  { key: 'odp', label: 'ODP', icon: Square },
  { key: 'ontOnline', label: 'ONT Online', icon: Circle },
  { key: 'ontOffline', label: 'ONT Offline', icon: Pulse },
  { key: 'cableBackbone', label: 'Kabel Backbone', icon: Path },
  { key: 'cableDrop', label: 'Kabel Drop', icon: Broadcast },
  { key: 'area', label: 'Area / Wilayah', icon: BoundingBox },
  { key: 'satellite', label: 'Satellite', icon: GlobeHemisphereEast },
  { key: 'heatmap', label: 'Heatmap Signal', icon: ThermometerHot },
] as const;

export default function LayerControl({
  layers,
  onLayerChange,
  filters,
  onFilterChange,
  visibleCount,
  totalCount,
  onResetFilters,
}: LayerControlProps) {
  const handleFilterChange = useCallback(
    (key: keyof NodeFilters, value: string) => {
      onFilterChange({ ...filters, [key]: value || undefined });
    },
    [filters, onFilterChange],
  );

  const hasActiveFilters = Object.values(filters).some((value) => value);

  return (
    <div className="max-h-[70dvh] w-full overflow-y-auto rounded-lg border border-slate-200 bg-white p-4 shadow-sm">
      <h3 className="mb-2 text-xs font-semibold uppercase tracking-wide text-slate-400">
        Layer
      </h3>
      <div className="space-y-1">
        {LAYER_DEFS.map((def) => (
          <label
            key={def.key}
            className="flex cursor-pointer items-center gap-2 rounded px-2 py-1.5 hover:bg-slate-50"
          >
            <input
              type="checkbox"
              checked={layers[def.key]}
              onChange={(event) => onLayerChange(def.key, event.target.checked)}
              className="h-4 w-4 rounded border-slate-300 text-sky-700 focus:ring-sky-600"
            />
            <def.icon size={16} className="text-slate-500" />
            <span className="text-sm text-slate-700">{def.label}</span>
          </label>
        ))}
      </div>

      <hr className="my-3 border-slate-200" />

      <h3 className="mb-2 text-xs font-semibold uppercase tracking-wide text-slate-400">
        Filter
      </h3>
      <div className="space-y-2">
        <FilterSelect
          label="Status ONT"
          value={filters.status ?? ''}
          onChange={(value) => handleFilterChange('status', value)}
          options={[
            { value: '', label: 'Semua' },
            { value: 'online', label: 'Online' },
            { value: 'offline', label: 'Offline' },
            { value: 'weak', label: 'Weak Signal' },
          ]}
        />
        <FilterSelect
          label="Billing"
          value={filters.billing_status ?? ''}
          onChange={(value) => handleFilterChange('billing_status', value)}
          options={[
            { value: '', label: 'Semua' },
            { value: 'aktif', label: 'Aktif' },
            { value: 'isolir', label: 'Isolir' },
            { value: 'pending', label: 'Pending' },
          ]}
        />
      </div>

      <hr className="my-3 border-slate-200" />

      <p className="text-xs text-slate-500">
        Menampilkan: {visibleCount} dari {totalCount} pelanggan
      </p>

      {hasActiveFilters && (
        <button
          type="button"
          onClick={onResetFilters}
          className="mt-2 w-full rounded-md border border-slate-300 px-3 py-1.5 text-xs font-medium text-slate-600 hover:bg-slate-50"
        >
          Reset filter
        </button>
      )}
    </div>
  );
}

function FilterSelect({
  label,
  value,
  onChange,
  options,
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  options: { value: string; label: string }[];
}) {
  return (
    <div>
      <label className="mb-0.5 block text-xs text-slate-500">{label}</label>
      <select
        value={value}
        onChange={(event) => onChange(event.target.value)}
        className="w-full rounded border border-slate-300 px-2 py-1 text-sm focus:border-sky-600 focus:outline-none focus:ring-1 focus:ring-sky-600"
      >
        {options.map((option) => (
          <option key={option.value} value={option.value}>
            {option.label}
          </option>
        ))}
      </select>
    </div>
  );
}
