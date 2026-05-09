"use client";

interface ProgressBarProps {
  /** Nilai progress (0-100+)*/
  value: number;
  /** Label di sebelah kiri*/
  label?: string;
  /** Tampilkan persentase di sebelah kanan*/
  showPercentage?: boolean;
  /** Ukuran bar*/
  size?: "sm" | "md" | "lg";
}

/**
 * KPI progress bar dengan color coding:
 * - Hijau: ≥100% (tercapai)
 * - Kuning: ≥80% (hampir)
 * - Merah: <80% (di bawah target)
 */
export function ProgressBar({
  value,
  label,
  showPercentage = true,
  size = "md",
}: ProgressBarProps) {
  const barColor =
    value >= 100
      ? "bg-emerald-500"
      : value >= 80
        ? "bg-amber-500"
        : "bg-red-500";

  const statusLabel =
    value >= 100
      ? "Tercapai"
      : value >= 80
        ? "Hampir tercapai"
        : "Di bawah target";

  const heightClass = size === "sm" ? "h-1.5" : size === "lg" ? "h-3" : "h-2";

  return (
    <div>
      {(label || showPercentage) && (
        <div className="mb-1 flex items-center justify-between text-xs">
          {label && <span className="text-slate-500">{label}</span>}
          {showPercentage && (
            <span className="font-medium text-slate-700">
              {value.toFixed(0)}% — {statusLabel}
            </span>
          )}
        </div>
      )}
      <div
        className={`w-full overflow-hidden rounded-full bg-slate-100 ${heightClass}`}
        role="progressbar"
        aria-valuenow={Math.min(value, 100)}
        aria-valuemin={0}
        aria-valuemax={100}
        aria-label={label ?? "Progress"}
      >
        <div
          className={`${heightClass} rounded-full transition-all duration-300 ${barColor}`}
          style={{ width: `${Math.min(value, 100)}%` }}
        />
      </div>
    </div>
  );
}
