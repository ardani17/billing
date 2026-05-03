'use client';

import { useCallback, useState } from 'react';
import AddMarkerMode from './AddMarkerMode';
import DrawLineMode from './DrawLineMode';
import MeasureDistanceMode from './MeasureDistanceMode';
import DeleteMode from './DeleteMode';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export type DrawingMode = 'marker' | 'line' | 'measure' | 'delete' | null;

interface DrawingToolbarProps {
  activeMode: DrawingMode;
  onModeChange: (mode: DrawingMode) => void;
}

// ---------------------------------------------------------------------------
// Tool definitions
// ---------------------------------------------------------------------------

const TOOLS: { mode: DrawingMode; icon: string; label: string }[] = [
  { mode: 'marker', icon: '📍', label: 'Tambah Marker' },
  { mode: 'line', icon: '✏️', label: 'Gambar Jalur' },
  { mode: 'measure', icon: '📐', label: 'Ukur Jarak' },
  { mode: 'delete', icon: '🗑️', label: 'Hapus' },
];

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export default function DrawingToolbar({
  activeMode,
  onModeChange,
}: DrawingToolbarProps) {
  const handleToggle = useCallback(
    (mode: DrawingMode) => {
      onModeChange(activeMode === mode ? null : mode);
    },
    [activeMode, onModeChange],
  );

  return (
    <div className="flex items-center gap-1 rounded-lg bg-white p-1 shadow-lg">
      {TOOLS.map((tool) => (
        <button
          key={tool.mode}
          onClick={() => handleToggle(tool.mode)}
          title={tool.label}
          className={`flex h-10 w-10 items-center justify-center rounded-md text-lg transition-colors ${
            activeMode === tool.mode
              ? 'bg-blue-600 text-white shadow-inner'
              : 'text-gray-600 hover:bg-gray-100'
          }`}
          aria-label={tool.label}
          aria-pressed={activeMode === tool.mode}
        >
          {tool.icon}
        </button>
      ))}

      {/* Active mode indicator */}
      {activeMode && (
        <span className="ml-2 hidden text-xs text-gray-500 md:inline">
          {TOOLS.find((t) => t.mode === activeMode)?.label}
        </span>
      )}
    </div>
  );
}
