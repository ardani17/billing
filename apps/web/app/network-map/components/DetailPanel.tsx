'use client';

import { useCallback, useEffect, useState } from 'react';
import { X } from '@phosphor-icons/react';
import type { MapNodeDetail } from '../lib/api';
import { fetchNodeDetail } from '../lib/api';
import CustomFieldsEditor from './CustomFieldsEditor';
import PhotoGallery from './PhotoGallery';
import ChangeHistory from './ChangeHistory';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

type TabKey = 'info' | 'keterangan' | 'foto' | 'riwayat';

interface DetailPanelProps {
  nodeId: string;
  onClose: () => void;
  onEditLocation?: (nodeId: string) => void;
  onNavigate?: (lat: number, lng: number) => void;
}

// ---------------------------------------------------------------------------
// Tab button
// ---------------------------------------------------------------------------

function TabButton({
  label,
  active,
  onClick,
}: {
  label: string;
  active: boolean;
  onClick: () => void;
}) {
  return (
    <button
      onClick={onClick}
      className={`px-3 py-2 text-sm font-medium transition-colors ${
        active
          ? 'border-b-2 border-blue-600 text-blue-600'
          : 'text-gray-500 hover:text-gray-700'
      }`}
    >
      {label}
    </button>
  );
}

// ---------------------------------------------------------------------------
// OLT Info View
// ---------------------------------------------------------------------------

function OLTInfoView({
  node,
  onNavigate,
}: {
  node: MapNodeDetail;
  onNavigate?: (lat: number, lng: number) => void;
}) {
  return (
    <div className="space-y-3">
      <InfoRow label="Nama" value={node.name ?? '-'} />
      <InfoRow label="Brand / Model" value={node.brand_model ?? '-'} />
      <InfoRow label="Status" value={<StatusBadge status={node.status} />} />
      <InfoRow label="Jumlah ONT" value={String(node.ont_count ?? 0)} />
      <InfoRow label="Latitude" value={String(node.latitude)} />
      <InfoRow label="Longitude" value={String(node.longitude)} />

      <div className="flex flex-wrap gap-2 pt-3">
        <ActionButton label="Lihat Detail OLT" />
        <ActionButton label="Lihat ONT di OLT ini" />
        {onNavigate && (
          <ActionButton
            label="Navigasi"
            onClick={() => onNavigate(node.latitude, node.longitude)}
          />
        )}
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// ODP Info View
// ---------------------------------------------------------------------------

function ODPInfoView({
  node,
  onEditLocation,
  onNavigate,
}: {
  node: MapNodeDetail;
  onEditLocation?: (nodeId: string) => void;
  onNavigate?: (lat: number, lng: number) => void;
}) {
  return (
    <div className="space-y-3">
      <InfoRow label="Nama" value={node.name ?? '-'} />
      <InfoRow label="Splitter" value={node.splitter_type ?? '-'} />
      <InfoRow label="Port Usage" value={node.port_usage ?? '-'} />
      <InfoRow label="Kapasitas" value={node.capacity ?? '-'} />
      <InfoRow label="Latitude" value={String(node.latitude)} />
      <InfoRow label="Longitude" value={String(node.longitude)} />

      <div className="flex flex-wrap gap-2 pt-3">
        <ActionButton label="Lihat Detail ODP" />
        <ActionButton label="Tambah ONT" />
        {onEditLocation && (
          <ActionButton
            label="Edit Lokasi"
            onClick={() => onEditLocation(node.id)}
          />
        )}
        {onNavigate && (
          <ActionButton
            label="Navigasi"
            onClick={() => onNavigate(node.latitude, node.longitude)}
          />
        )}
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// ONT Info View
// ---------------------------------------------------------------------------

function ONTInfoView({
  node,
  onEditLocation,
  onNavigate,
}: {
  node: MapNodeDetail;
  onEditLocation?: (nodeId: string) => void;
  onNavigate?: (lat: number, lng: number) => void;
}) {
  return (
    <div className="space-y-3">
      <InfoRow label="Pelanggan" value={node.customer_name ?? '-'} />
      <InfoRow label="ID Pelanggan" value={node.customer_id ?? '-'} />
      <InfoRow label="Paket" value={node.package_name ?? '-'} />
      <InfoRow
        label="Signal"
        value={
          node.signal_dbm != null ? `${node.signal_dbm} dBm` : '-'
        }
      />
      <InfoRow label="Serial Number" value={node.serial_number ?? '-'} />
      <InfoRow label="ODP" value={node.odp_name ?? '-'} />
      <InfoRow label="Status" value={<StatusBadge status={node.status} />} />
      <InfoRow label="Billing" value={node.billing_status ?? '-'} />

      <div className="flex flex-wrap gap-2 pt-3">
        <ActionButton label="Lihat Pelanggan" />
        <ActionButton label="Lihat Detail ONT" />
        {onEditLocation && (
          <ActionButton
            label="Edit Lokasi"
            onClick={() => onEditLocation(node.id)}
          />
        )}
        {onNavigate && (
          <ActionButton
            label="Navigasi"
            onClick={() => onNavigate(node.latitude, node.longitude)}
          />
        )}
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Shared helpers
// ---------------------------------------------------------------------------

function InfoRow({
  label,
  value,
}: {
  label: string;
  value: React.ReactNode;
}) {
  return (
    <div className="flex items-start justify-between gap-2">
      <span className="text-sm text-gray-500">{label}</span>
      <span className="text-right text-sm font-medium text-gray-900">
        {value}
      </span>
    </div>
  );
}

function StatusBadge({ status }: { status?: string }) {
  const colors: Record<string, string> = {
    online: 'bg-green-100 text-green-700',
    offline: 'bg-red-100 text-red-700',
    los: 'bg-red-100 text-red-700',
    weak: 'bg-yellow-100 text-yellow-700',
    pending: 'bg-gray-100 text-gray-600',
    active: 'bg-green-100 text-green-700',
  };
  const cls = colors[status ?? ''] ?? 'bg-gray-100 text-gray-600';
  return (
    <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${cls}`}>
      {status ?? 'unknown'}
    </span>
  );
}

function ActionButton({
  label,
  onClick,
}: {
  label: string;
  onClick?: () => void;
}) {
  return (
    <button
      onClick={onClick}
      className="rounded-md border border-gray-300 bg-white px-3 py-1.5 text-xs font-medium text-gray-700 shadow-sm hover:bg-gray-50"
    >
      {label}
    </button>
  );
}

function nodeTypeLabel(type: string): string {
  if (type === 'olt') return 'OLT';
  if (type === 'odp') return 'ODP';
  return 'ONT';
}

// ---------------------------------------------------------------------------
// DetailPanel
// ---------------------------------------------------------------------------

export default function DetailPanel({
  nodeId,
  onClose,
  onEditLocation,
  onNavigate,
}: DetailPanelProps) {
  const [node, setNode] = useState<MapNodeDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<TabKey>('info');

  const loadNode = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await fetchNodeDetail(nodeId);
      setNode(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Gagal memuat data');
    } finally {
      setLoading(false);
    }
  }, [nodeId]);

  useEffect(() => {
    loadNode();
  }, [loadNode]);

  if (loading) {
    return (
      <div className="flex h-full items-center justify-center p-4 text-gray-400">
        Memuat detail...
      </div>
    );
  }

  if (error || !node) {
    return (
      <div className="p-4">
        <p className="text-sm text-red-500">{error ?? 'Node tidak ditemukan'}</p>
        <button
          onClick={onClose}
          className="mt-2 text-sm text-blue-600 hover:underline"
        >
          Tutup
        </button>
      </div>
    );
  }

  return (
    <div className="flex h-full flex-col">
      {/* Header*/}
      <div className="flex items-center justify-between border-b border-gray-200 px-4 py-3">
        <div>
          <span className="text-xs font-medium uppercase text-gray-400">
            {nodeTypeLabel(node.node_type)}
          </span>
          <h2 className="text-lg font-semibold text-gray-900">
            {node.name ?? node.id.slice(0, 8)}
          </h2>
        </div>
        <button
          onClick={onClose}
          className="rounded p-1 text-gray-400 hover:bg-gray-100 hover:text-gray-600"
          aria-label="Tutup panel"
        >
          <X size={18} />
        </button>
      </div>

      {/* Tabs*/}
      <div className="flex border-b border-gray-200 px-4">
        <TabButton label="Info" active={activeTab === 'info'} onClick={() => setActiveTab('info')} />
        <TabButton label="Keterangan" active={activeTab === 'keterangan'} onClick={() => setActiveTab('keterangan')} />
        <TabButton label="Foto" active={activeTab === 'foto'} onClick={() => setActiveTab('foto')} />
        <TabButton label="Riwayat" active={activeTab === 'riwayat'} onClick={() => setActiveTab('riwayat')} />
      </div>

      {/* Tab content*/}
      <div className="flex-1 overflow-y-auto p-4">
        {activeTab === 'info' && (
          <>
            {node.node_type === 'olt' && (
              <OLTInfoView node={node} onNavigate={onNavigate} />
            )}
            {node.node_type === 'odp' && (
              <ODPInfoView node={node} onEditLocation={onEditLocation} onNavigate={onNavigate} />
            )}
            {node.node_type === 'ont' && (
              <ONTInfoView node={node} onEditLocation={onEditLocation} onNavigate={onNavigate} />
            )}
          </>
        )}

        {activeTab === 'keterangan' && (
          <CustomFieldsEditor
            nodeId={node.id}
            customFields={node.custom_fields ?? {}}
            onSaved={loadNode}
          />
        )}

        {activeTab === 'foto' && (
          <PhotoGallery
            nodeId={node.id}
            photos={node.photos ?? []}
            onChanged={loadNode}
          />
        )}

        {activeTab === 'riwayat' && (
          <ChangeHistory
            nodeId={node.id}
            history={node.history ?? []}
          />
        )}
      </div>
    </div>
  );
}
