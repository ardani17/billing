'use client';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface MeasureDistanceModeProps {
  /** Dua titik yang dipilih pada peta: [[lat1, lng1], [lat2, lng2]]*/
  points: [number, number][];
  onClear: () => void;
}

// ---------------------------------------------------------------------------
// Fungsi bantus
// ---------------------------------------------------------------------------

function haversineDistance(
  lat1: number,
  lng1: number,
  lat2: number,
  lng2: number,
): number {
  const R = 6371000; // Earth radius in meters
  const toRad = (deg: number) => (deg * Math.PI) / 180;
  const dLat = toRad(lat2 - lat1);
  const dLng = toRad(lng2 - lng1);
  const a =
    Math.sin(dLat / 2) ** 2 +
    Math.cos(toRad(lat1)) * Math.cos(toRad(lat2)) * Math.sin(dLng / 2) ** 2;
  return R * 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a));
}

function formatDistance(meters: number): string {
  if (meters >= 1000) return `${(meters / 1000).toFixed(2)} km`;
  return `${Math.round(meters)} m`;
}

// ---------------------------------------------------------------------------
// Komponen - Menampilkan jarak garis lurus antara 2 titik
// ---------------------------------------------------------------------------

export default function MeasureDistanceMode({
  points,
  onClear,
}: MeasureDistanceModeProps) {
  const firstPoint = points[0];
  const secondPoint = points[1];

  if (points.length < 2) {
    return (
      <div className="rounded-lg bg-white px-4 py-3 shadow-lg">
        <p className="text-sm text-gray-600">
          Klik {2 - points.length} titik lagi di peta untuk mengukur jarak
        </p>
        {firstPoint && (
          <p className="mt-1 text-xs text-gray-400">
            Titik 1: {firstPoint[0].toFixed(6)}, {firstPoint[1].toFixed(6)}
          </p>
        )}
      </div>
    );
  }

  if (!firstPoint || !secondPoint) return null;

  const distance = haversineDistance(
    firstPoint[0],
    firstPoint[1],
    secondPoint[0],
    secondPoint[1],
  );

  return (
    <div className="rounded-lg bg-white px-4 py-3 shadow-lg">
      <div className="flex items-center justify-between">
        <div>
          <p className="text-sm font-medium text-gray-900">
            Jarak: {formatDistance(distance)}
          </p>
          <p className="mt-1 text-xs text-gray-400">
            ({firstPoint[0].toFixed(6)}, {firstPoint[1].toFixed(6)}) ke{' '}
            ({secondPoint[0].toFixed(6)}, {secondPoint[1].toFixed(6)})
          </p>
        </div>
        <button
          onClick={onClear}
          className="rounded-md border border-gray-300 px-2 py-1 text-xs text-gray-600 hover:bg-gray-50"
        >
          Ukur Ulang
        </button>
      </div>
    </div>
  );
}
