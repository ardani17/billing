"use client";

import { useState } from "react";
import { previewCustomReport, createTemplate } from "../../lib/api";

const AVAILABLE_METRICS = [
  { value: "revenue", label: "Pendapatan" },
  { value: "customers", label: "Jumlah Pelanggan" },
  { value: "receivables", label: "Piutang" },
  { value: "payments", label: "Pembayaran" },
  { value: "churn", label: "Churn" },
  { value: "arpu", label: "ARPU" },
  { value: "expenses", label: "Pengeluaran" },
];

const GROUP_BY_OPTIONS = [
  { value: "month", label: "Bulan" },
  { value: "area", label: "Area" },
  { value: "package", label: "Paket" },
  { value: "payment_method", label: "Metode Pembayaran" },
];

const DISPLAY_TYPES = [
  { value: "table", label: "Tabel" },
  { value: "bar_chart", label: "Bar Chart" },
  { value: "line_chart", label: "Line Chart" },
  { value: "pie_chart", label: "Pie Chart" },
] as const;

interface CustomReportBuilderProps {
  onPreviewData?: (data: unknown) => void;
}

export function CustomReportBuilder({ onPreviewData }: CustomReportBuilderProps) {
  const [name, setName] = useState("");
  const [selectedMetrics, setSelectedMetrics] = useState<string[]>([]);
  const [groupBy, setGroupBy] = useState("month");
  const [subGroupBy, setSubGroupBy] = useState<string | undefined>();
  const [displayType, setDisplayType] = useState<string>("table");
  const [periodStart, setPeriodStart] = useState("");
  const [periodEnd, setPeriodEnd] = useState("");
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const toggleMetric = (metric: string) => {
    setSelectedMetrics((prev) => {
      if (prev.includes(metric)) return prev.filter((m) => m !== metric);
      if (prev.length >= 3) return prev;
      return [...prev, metric];
    });
  };

  const handlePreview = async () => {
    if (selectedMetrics.length === 0 || !periodStart || !periodEnd) return;
    setLoading(true);
    setError(null);
    try {
      const data = await previewCustomReport(selectedMetrics, groupBy, subGroupBy, periodStart, periodEnd, displayType);
      onPreviewData?.(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Gagal memuat preview");
    } finally {
      setLoading(false);
    }
  };

  const handleSave = async () => {
    if (!name || selectedMetrics.length === 0) return;
    setSaving(true);
    setError(null);
    try {
      await createTemplate({
        name,
        metrics: selectedMetrics,
        group_by: groupBy,
        sub_group_by: subGroupBy,
        display_type: displayType as "table" | "bar_chart" | "line_chart" | "pie_chart",
        default_period_range: undefined,
      });
      setName("");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Gagal menyimpan template");
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="rounded-xl border border-slate-200 bg-white p-5">
      <h3 className="mb-4 text-lg font-semibold text-slate-900">Laporan Kustom</h3>

      <div className="space-y-4">
        {/* Nama */}
        <div>
          <label className="mb-1 block text-sm font-medium text-slate-700">Nama Laporan</label>
          <input
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="Contoh: Laporan Pendapatan per Area"
            className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
            style={{ minHeight: 44 }}
          />
        </div>

        {/* Metrik (max 3) */}
        <div>
          <label className="mb-2 block text-sm font-medium text-slate-700">
            Metrik (maks. 3)
          </label>
          <div className="flex flex-wrap gap-2">
            {AVAILABLE_METRICS.map((m) => {
              const selected = selectedMetrics.includes(m.value);
              const disabled = !selected && selectedMetrics.length >= 3;
              return (
                <button
                  key={m.value}
                  type="button"
                  onClick={() => toggleMetric(m.value)}
                  disabled={disabled}
                  className={`rounded-full px-3 py-1.5 text-sm font-medium transition-colors ${
                    selected
                      ? "bg-blue-600 text-white"
                      : disabled
                        ? "bg-slate-100 text-slate-400 cursor-not-allowed"
                        : "bg-slate-100 text-slate-700 hover:bg-slate-200"
                  }`}
                  style={{ minHeight: 44 }}
                >
                  {m.label}
                </button>
              );
            })}
          </div>
        </div>

        {/* Group By */}
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div>
            <label className="mb-1 block text-sm font-medium text-slate-700">Group By</label>
            <select
              value={groupBy}
              onChange={(e) => setGroupBy(e.target.value)}
              className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
              style={{ minHeight: 44 }}
            >
              {GROUP_BY_OPTIONS.map((o) => (
                <option key={o.value} value={o.value}>{o.label}</option>
              ))}
            </select>
          </div>
          <div>
            <label className="mb-1 block text-sm font-medium text-slate-700">Sub-Group (opsional)</label>
            <select
              value={subGroupBy ?? ""}
              onChange={(e) => setSubGroupBy(e.target.value || undefined)}
              className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
              style={{ minHeight: 44 }}
            >
              <option value="">Tidak ada</option>
              {GROUP_BY_OPTIONS.filter((o) => o.value !== groupBy).map((o) => (
                <option key={o.value} value={o.value}>{o.label}</option>
              ))}
            </select>
          </div>
        </div>

        {/* Periode */}
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div>
            <label className="mb-1 block text-sm font-medium text-slate-700">Dari</label>
            <input
              type="date"
              value={periodStart}
              onChange={(e) => setPeriodStart(e.target.value)}
              className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
              style={{ minHeight: 44 }}
            />
          </div>
          <div>
            <label className="mb-1 block text-sm font-medium text-slate-700">Sampai</label>
            <input
              type="date"
              value={periodEnd}
              onChange={(e) => setPeriodEnd(e.target.value)}
              className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
              style={{ minHeight: 44 }}
            />
          </div>
        </div>

        {/* Display type */}
        <div>
          <label className="mb-2 block text-sm font-medium text-slate-700">Tampilan</label>
          <div className="flex flex-wrap gap-2">
            {DISPLAY_TYPES.map((d) => (
              <button
                key={d.value}
                type="button"
                onClick={() => setDisplayType(d.value)}
                className={`rounded-lg px-3 py-1.5 text-sm font-medium transition-colors ${
                  displayType === d.value
                    ? "bg-blue-600 text-white"
                    : "bg-slate-100 text-slate-700 hover:bg-slate-200"
                }`}
                style={{ minHeight: 44 }}
              >
                {d.label}
              </button>
            ))}
          </div>
        </div>

        {error && (
          <div className="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">{error}</div>
        )}

        {/* Actions */}
        <div className="flex flex-wrap gap-3 pt-2">
          <button
            type="button"
            onClick={handlePreview}
            disabled={loading || selectedMetrics.length === 0}
            className="rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
            style={{ minHeight: 44 }}
          >
            {loading ? "Memuat..." : "Preview"}
          </button>
          <button
            type="button"
            onClick={handleSave}
            disabled={saving || !name || selectedMetrics.length === 0}
            className="rounded-lg border border-slate-300 px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 disabled:opacity-50"
            style={{ minHeight: 44 }}
          >
            {saving ? "Menyimpan..." : "Simpan sebagai Template"}
          </button>
        </div>
      </div>
    </div>
  );
}
