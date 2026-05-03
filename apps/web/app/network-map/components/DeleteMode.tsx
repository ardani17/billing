'use client';

import { useCallback, useState } from 'react';
import { deleteNode, deleteCable } from '../lib/api';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface DeleteModeProps {
  /** The type of entity to delete */
  targetType: 'node' | 'cable';
  /** The ID of the entity to delete */
  targetId: string;
  /** Display name for the confirmation dialog */
  targetName?: string;
  onDeleted?: () => void;
  onCancel: () => void;
}

// ---------------------------------------------------------------------------
// Component — Confirmation dialog for soft delete
// ---------------------------------------------------------------------------

export default function DeleteMode({
  targetType,
  targetId,
  targetName,
  onDeleted,
  onCancel,
}: DeleteModeProps) {
  const [deleting, setDeleting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleDelete = useCallback(async () => {
    setDeleting(true);
    setError(null);
    try {
      if (targetType === 'node') {
        await deleteNode(targetId);
      } else {
        await deleteCable(targetId);
      }
      onDeleted?.();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Gagal menghapus');
    } finally {
      setDeleting(false);
    }
  }, [targetType, targetId, onDeleted]);

  const typeLabel = targetType === 'node' ? 'node' : 'jalur kabel';

  return (
    <div className="rounded-lg border border-red-200 bg-white p-4 shadow-lg">
      <h3 className="mb-2 text-sm font-semibold text-gray-900">
        Hapus {typeLabel}?
      </h3>
      <p className="mb-3 text-sm text-gray-600">
        {targetName ? (
          <>
            Apakah Anda yakin ingin menghapus{' '}
            <span className="font-medium">{targetName}</span>?
          </>
        ) : (
          <>Apakah Anda yakin ingin menghapus {typeLabel} ini?</>
        )}
      </p>
      <p className="mb-4 text-xs text-gray-400">
        Data akan dipindahkan ke Trash dan dapat dipulihkan dalam 30 hari.
      </p>

      {error && <p className="mb-3 text-xs text-red-500">{error}</p>}

      <div className="flex gap-2">
        <button
          onClick={handleDelete}
          disabled={deleting}
          className="flex-1 rounded-md bg-red-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-red-700 disabled:opacity-50"
        >
          {deleting ? 'Menghapus…' : 'Ya, Hapus'}
        </button>
        <button
          onClick={onCancel}
          className="rounded-md border border-gray-300 px-3 py-1.5 text-sm text-gray-700 hover:bg-gray-50"
        >
          Batal
        </button>
      </div>
    </div>
  );
}
