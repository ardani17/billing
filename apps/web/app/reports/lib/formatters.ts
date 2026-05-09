// =============================================================================
// Formatters - helper functions untuk format angka, mata uang, dan tanggal
// =============================================================================

/**
 * Format angka ke Rupiah (Rp 1.234.567).
 * Menggunakan titik sebagai pemisah ribuan sesuai format Indonesia.
 */
export function formatCurrency(amount: number): string {
  const abs = Math.abs(amount);
  const formatted = abs.toLocaleString("id-ID", {
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  });
  return amount < 0 ? `-Rp ${formatted}` : `Rp ${formatted}`;
}

/**
 * Format angka ke persentase (contoh: 85.5%).
 * Menghilangkan desimal jika bilangan bulat.
 */
export function formatPercentage(value: number): string {
  if (Number.isInteger(value)) {
    return `${value}%`;
  }
  return `${value.toFixed(1)}%`;
}

/**
 * Format angka dengan pemisah ribuan (contoh: 1.234).
 */
export function formatNumber(value: number): string {
  return value.toLocaleString("id-ID");
}

/**
 * Format tanggal ISO ke format Indonesia (contoh: 15 Jan 2025).
 */
export function formatDate(date: string): string {
  const d = new Date(date);
  return d.toLocaleDateString("id-ID", {
    day: "numeric",
    month: "short",
    year: "numeric",
  });
}

/**
 * Format bulan dari "2006-01" ke "Jan 2006".
 */
export function formatMonth(month: string): string {
  const [year, m] = month.split("-");
  const d = new Date(Number(year), Number(m) - 1, 1);
  return d.toLocaleDateString("id-ID", {
    month: "short",
    year: "numeric",
  });
}

/**
 * Format bytes ke satuan yang sesuai (KB, MB, GB, TB).
 */
export function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";
  const units = ["B", "KB", "MB", "GB", "TB"];
  const k = 1024;
  const i = Math.floor(Math.log(Math.abs(bytes)) / Math.log(k));
  const idx = Math.min(i, units.length - 1);
  const value = bytes / Math.pow(k, idx);
  return `${value.toFixed(value >= 100 ? 0 : 1)} ${units[idx]}`;
}

/**
 * Format delta dengan prefix +/- (contoh: +12.5%, -3.2%).
 */
export function formatDelta(delta: number): string {
  const prefix = delta > 0 ? "+" : "";
  if (Number.isInteger(delta)) {
    return `${prefix}${delta}%`;
  }
  return `${prefix}${delta.toFixed(1)}%`;
}

/**
 * Warna untuk delta: hijau (positif), merah (negatif), abu-abu (stabil).
 * Mengembalikan Tailwind CSS class name.
 */
export function getDeltaColor(delta: number): string {
  if (delta > 0) return "text-emerald-600";
  if (delta < 0) return "text-red-600";
  return "text-slate-500";
}
