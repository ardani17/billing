'use client';

import { useCallback, useRef, useState } from 'react';
import { uploadPhoto, deletePhoto, type NodePhoto } from '../lib/api';

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const MAX_PHOTOS = 5;

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface PhotoGalleryProps {
  nodeId: string;
  photos: NodePhoto[];
  onChanged?: () => void;
}

// ---------------------------------------------------------------------------
// Komponen
// ---------------------------------------------------------------------------

export default function PhotoGallery({
  nodeId,
  photos,
  onChanged,
}: PhotoGalleryProps) {
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [uploading, setUploading] = useState(false);
  const [deleting, setDeleting] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [caption, setCaption] = useState('');

  const canUpload = photos.length < MAX_PHOTOS;

  const handleUpload = useCallback(
    async (e: React.ChangeEvent<HTMLInputElement>) => {
      const file = e.target.files?.[0];
      if (!file) return;

      setUploading(true);
      setError(null);
      try {
        await uploadPhoto(nodeId, file, caption || undefined);
        setCaption('');
        onChanged?.();
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Gagal upload foto');
      } finally {
        setUploading(false);
        // Reset file input
        if (fileInputRef.current) fileInputRef.current.value = '';
      }
    },
    [nodeId, caption, onChanged],
  );

  const handleDelete = useCallback(
    async (photoId: string) => {
      if (!confirm('Hapus foto ini?')) return;
      setDeleting(photoId);
      setError(null);
      try {
        await deletePhoto(nodeId, photoId);
        onChanged?.();
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Gagal menghapus foto');
      } finally {
        setDeleting(null);
      }
    },
    [nodeId, onChanged],
  );

  return (
    <div className="space-y-4">
      {/* Photo count indicator*/}
      <div className="flex items-center justify-between">
        <p className="text-sm text-gray-500">
          Foto ({photos.length}/{MAX_PHOTOS})
        </p>
        {!canUpload && (
          <span className="text-xs text-amber-600">Batas foto tercapai</span>
        )}
      </div>

      {/* Photo grid*/}
      {photos.length > 0 ? (
        <div className="grid grid-cols-2 gap-2">
          {photos.map((photo) => (
            <div
              key={photo.id}
              className="group relative overflow-hidden rounded-lg border border-gray-200"
            >
              <img
                src={photo.file_path}
                alt={photo.caption ?? 'Foto node'}
                className="h-32 w-full object-cover"
              />
              {photo.caption && (
                <p className="truncate px-2 py-1 text-xs text-gray-600">
                  {photo.caption}
                </p>
              )}
              <button
                onClick={() => handleDelete(photo.id)}
                disabled={deleting === photo.id}
                className="absolute right-1 top-1 rounded-full bg-black/50 p-1 text-xs text-white opacity-0 transition-opacity hover:bg-black/70 group-hover:opacity-100"
                aria-label="Hapus foto"
              >
                {deleting === photo.id ? '…' : '✕'}
              </button>
            </div>
          ))}
        </div>
      ) : (
        <p className="py-4 text-center text-sm text-gray-400">
          Belum ada foto
        </p>
      )}

      {/* Upload section*/}
      {canUpload && (
        <div className="space-y-2">
          <input
            type="text"
            value={caption}
            onChange={(e) => setCaption(e.target.value)}
            placeholder="Caption (opsional)"
            className="w-full rounded-md border border-gray-300 px-3 py-1.5 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
          />
          <input
            ref={fileInputRef}
            type="file"
            accept="image/jpeg,image/png,image/webp"
            onChange={handleUpload}
            className="hidden"
          />
          <button
            onClick={() => fileInputRef.current?.click()}
            disabled={uploading}
            className="w-full rounded-md border border-dashed border-gray-300 bg-gray-50 px-4 py-3 text-sm text-gray-600 hover:border-blue-400 hover:bg-blue-50 disabled:opacity-50"
          >
            {uploading ? 'Mengupload…' : '📷 Upload Foto'}
          </button>
        </div>
      )}

      {error && <p className="text-sm text-red-500">{error}</p>}
    </div>
  );
}
