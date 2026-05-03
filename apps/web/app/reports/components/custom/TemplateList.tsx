"use client";

import { useState, useEffect } from "react";
import type { CustomReportTemplate } from "../../lib/types";
import { fetchTemplates, deleteTemplate } from "../../lib/api";
import { formatDate } from "../../lib/formatters";

interface TemplateListProps {
  onLoad?: (template: CustomReportTemplate) => void;
}

export function TemplateList({ onLoad }: TemplateListProps) {
  const [templates, setTemplates] = useState<CustomReportTemplate[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = async () => {
    setLoading(true);
    try {
      const data = await fetchTemplates();
      setTemplates(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Gagal memuat template");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, []);

  const handleDelete = async (id: string) => {
    try {
      await deleteTemplate(id);
      setTemplates((prev) => prev.filter((t) => t.id !== id));
    } catch (err) {
      setError(err instanceof Error ? err.message : "Gagal menghapus template");
    }
  };

  const displayTypeLabel: Record<string, string> = {
    table: "Tabel",
    bar_chart: "Bar Chart",
    line_chart: "Line Chart",
    pie_chart: "Pie Chart",
  };

  if (loading) {
    return (
      <div className="rounded-xl border border-slate-200 bg-white p-5">
        <div className="h-24 animate-pulse rounded-lg bg-slate-50" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="rounded-xl border border-red-200 bg-red-50 px-5 py-4 text-sm text-red-700">
        {error}
      </div>
    );
  }

  if (templates.length === 0) {
    return (
      <div className="rounded-xl border border-dashed border-slate-300 bg-slate-50 px-5 py-8 text-center text-sm text-slate-500">
        Belum ada template tersimpan. Buat laporan kustom dan simpan sebagai template.
      </div>
    );
  }

  return (
    <div className="rounded-xl border border-slate-200 bg-white p-5">
      <h3 className="mb-4 text-sm font-medium text-slate-700">Template Tersimpan</h3>
      <div className="space-y-3">
        {templates.map((t) => (
          <div
            key={t.id}
            className="flex flex-col gap-3 rounded-lg border border-slate-100 p-4 sm:flex-row sm:items-center sm:justify-between"
          >
            <div className="min-w-0">
              <p className="font-medium text-slate-900">{t.name}</p>
              <p className="mt-0.5 text-xs text-slate-500">
                {t.metrics.join(", ")} · {displayTypeLabel[t.display_type] ?? t.display_type} · Dibuat {formatDate(t.created_at)}
              </p>
            </div>
            <div className="flex flex-shrink-0 gap-2">
              <button
                type="button"
                onClick={() => onLoad?.(t)}
                className="rounded-lg bg-blue-50 px-3 py-1.5 text-xs font-medium text-blue-700 hover:bg-blue-100"
                style={{ minHeight: 44 }}
              >
                Muat
              </button>
              <button
                type="button"
                onClick={() => handleDelete(t.id)}
                className="rounded-lg bg-red-50 px-3 py-1.5 text-xs font-medium text-red-700 hover:bg-red-100"
                style={{ minHeight: 44 }}
              >
                Hapus
              </button>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
