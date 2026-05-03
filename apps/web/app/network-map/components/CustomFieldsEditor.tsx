'use client';

import { useCallback, useState } from 'react';
import { updateNode } from '../lib/api';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface CustomFieldsEditorProps {
  nodeId: string;
  customFields: Record<string, unknown>;
  onSaved?: () => void;
}

/** Predefined custom field definitions. */
const FIELD_DEFS = [
  { key: 'ip_pool', label: 'IP Pool', placeholder: '192.168.1.0/24' },
  { key: 'vlan', label: 'VLAN', placeholder: '100' },
  { key: 'gateway', label: 'Gateway', placeholder: '192.168.1.1' },
  { key: 'tipe_kabel', label: 'Tipe Kabel', placeholder: 'G.652D' },
  { key: 'lokasi_detail', label: 'Lokasi Detail', placeholder: 'Tiang ke-3 dari kiri' },
  { key: 'catatan', label: 'Catatan', placeholder: 'Catatan bebas…' },
] as const;

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export default function CustomFieldsEditor({
  nodeId,
  customFields,
  onSaved,
}: CustomFieldsEditorProps) {
  const [fields, setFields] = useState<Record<string, string>>(() => {
    const initial: Record<string, string> = {};
    for (const def of FIELD_DEFS) {
      initial[def.key] = String(customFields[def.key] ?? '');
    }
    return initial;
  });
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);

  const handleChange = useCallback((key: string, value: string) => {
    setFields((prev) => ({ ...prev, [key]: value }));
    setSuccess(false);
  }, []);

  const handleSave = useCallback(async () => {
    setSaving(true);
    setError(null);
    setSuccess(false);
    try {
      // Build custom_fields payload — only include non-empty values
      const payload: Record<string, unknown> = {};
      for (const [key, value] of Object.entries(fields)) {
        if (value.trim()) {
          payload[key] = value.trim();
        }
      }
      await updateNode(nodeId, { custom_fields: payload });
      setSuccess(true);
      onSaved?.();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Gagal menyimpan');
    } finally {
      setSaving(false);
    }
  }, [nodeId, fields, onSaved]);

  return (
    <div className="space-y-4">
      <p className="text-sm text-gray-500">
        Isi keterangan tambahan untuk node ini.
      </p>

      {FIELD_DEFS.map((def) => (
        <div key={def.key}>
          <label
            htmlFor={`cf-${def.key}`}
            className="mb-1 block text-sm font-medium text-gray-700"
          >
            {def.label}
          </label>
          {def.key === 'catatan' ? (
            <textarea
              id={`cf-${def.key}`}
              rows={3}
              value={fields[def.key]}
              onChange={(e) => handleChange(def.key, e.target.value)}
              placeholder={def.placeholder}
              className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
            />
          ) : (
            <input
              id={`cf-${def.key}`}
              type="text"
              value={fields[def.key]}
              onChange={(e) => handleChange(def.key, e.target.value)}
              placeholder={def.placeholder}
              className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
            />
          )}
        </div>
      ))}

      {error && <p className="text-sm text-red-500">{error}</p>}
      {success && (
        <p className="text-sm text-green-600">Berhasil disimpan ✓</p>
      )}

      <button
        onClick={handleSave}
        disabled={saving}
        className="w-full rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white shadow-sm hover:bg-blue-700 disabled:opacity-50"
      >
        {saving ? 'Menyimpan…' : 'Simpan Keterangan'}
      </button>
    </div>
  );
}
