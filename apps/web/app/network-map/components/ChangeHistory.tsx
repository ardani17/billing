'use client';

import type { MapChangeHistory } from '../lib/api';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface ChangeHistoryProps {
  nodeId: string;
  history: MapChangeHistory[];
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

const ACTION_LABELS: Record<string, string> = {
  created: 'Dibuat',
  location_moved: 'Lokasi dipindahkan',
  custom_fields_updated: 'Keterangan diperbarui',
  photo_added: 'Foto ditambahkan',
  photo_removed: 'Foto dihapus',
  deleted: 'Dihapus',
  restored: 'Dipulihkan',
};

const ACTION_ICONS: Record<string, string> = {
  created: '🆕',
  location_moved: '📍',
  custom_fields_updated: '📝',
  photo_added: '📷',
  photo_removed: '🗑️',
  deleted: '❌',
  restored: '♻️',
};

function formatDate(dateStr: string): string {
  const d = new Date(dateStr);
  return d.toLocaleDateString('id-ID', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

function formatCoordinateValue(value: unknown): string {
  if (typeof value === 'number') {
    return value.toFixed(6);
  }
  if (typeof value === 'string') {
    return value;
  }
  return '-';
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export default function ChangeHistory({ history }: ChangeHistoryProps) {
  if (history.length === 0) {
    return (
      <p className="py-4 text-center text-sm text-gray-400">
        Belum ada riwayat perubahan
      </p>
    );
  }

  return (
    <div className="space-y-0">
      <p className="mb-3 text-sm text-gray-500">
        Riwayat perubahan ({history.length})
      </p>

      <div className="relative">
        {/* Timeline line */}
        <div className="absolute left-4 top-0 h-full w-px bg-gray-200" />

        {history.map((entry, idx) => (
          <div key={entry.id} className="relative flex gap-3 pb-4">
            {/* Timeline dot */}
            <div className="relative z-10 flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-full bg-white text-sm shadow-sm ring-1 ring-gray-200">
              {ACTION_ICONS[entry.action] ?? '•'}
            </div>

            {/* Content */}
            <div className="flex-1 pt-0.5">
              <p className="text-sm font-medium text-gray-900">
                {ACTION_LABELS[entry.action] ?? entry.action}
              </p>
              <p className="text-xs text-gray-500">
                {entry.performed_by} · {formatDate(entry.created_at)}
              </p>

              {/* Show changed values for location_moved */}
              {entry.action === 'location_moved' && entry.new_value && (
                <div className="mt-1 rounded bg-gray-50 px-2 py-1 text-xs text-gray-600">
                  Koordinat baru:{' '}
                  {formatCoordinateValue(
                    (entry.new_value as Record<string, unknown>).latitude,
                  )},{' '}
                  {formatCoordinateValue(
                    (entry.new_value as Record<string, unknown>).longitude,
                  )}
                </div>
              )}

              {/* Show changed fields for custom_fields_updated */}
              {entry.action === 'custom_fields_updated' && entry.new_value && (
                <div className="mt-1 rounded bg-gray-50 px-2 py-1 text-xs text-gray-600">
                  {Object.keys(entry.new_value).join(', ')} diperbarui
                </div>
              )}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
