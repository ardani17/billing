"use client";

import { useState, useEffect } from "react";
import type { ReportSchedule, ScheduleType, Recipient } from "../../lib/types";
import { fetchSchedules, createSchedule, deleteSchedule } from "../../lib/api";

const REPORT_TYPES = [
  { value: "revenue", label: "Pendapatan" },
  { value: "aging", label: "Aging Piutang" },
  { value: "payments", label: "Pembayaran" },
  { value: "profit_loss", label: "Laba Rugi" },
  { value: "customer_growth", label: "Pertumbuhan Pelanggan" },
  { value: "churn", label: "Analisis Churn" },
];

const SCHEDULE_TYPES: { value: ScheduleType; label: string }[] = [
  { value: "daily", label: "Harian" },
  { value: "weekly", label: "Mingguan" },
  { value: "monthly", label: "Bulanan" },
];

interface ScheduleDialogProps {
  open: boolean;
  onClose: () => void;
}

export function ScheduleDialog({ open, onClose }: ScheduleDialogProps) {
  const [schedules, setSchedules] = useState<ReportSchedule[]>([]);
  const [loading, setLoading] = useState(true);
  const [reportType, setReportType] = useState("revenue");
  const [scheduleType, setScheduleType] = useState<ScheduleType>("monthly");
  const [format, setFormat] = useState<"pdf" | "xlsx">("pdf");
  const [recipientType, setRecipientType] = useState<"email" | "whatsapp">("email");
  const [recipientAddress, setRecipientAddress] = useState("");
  const [recipients, setRecipients] = useState<Recipient[]>([]);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (open) {
      fetchSchedules()
        .then(setSchedules)
        .catch(() => {})
        .finally(() => setLoading(false));
    }
  }, [open]);

  const addRecipient = () => {
    if (!recipientAddress.trim()) return;
    setRecipients((prev) => [...prev, { type: recipientType, address: recipientAddress.trim() }]);
    setRecipientAddress("");
  };

  const removeRecipient = (idx: number) => {
    setRecipients((prev) => prev.filter((_, i) => i !== idx));
  };

  const handleCreate = async () => {
    if (recipients.length === 0) {
      setError("Tambahkan minimal 1 penerima");
      return;
    }
    setSaving(true);
    setError(null);
    try {
      const schedule = await createSchedule({
        report_type: reportType,
        schedule_type: scheduleType,
        format,
        recipients,
      });
      setSchedules((prev) => [...prev, schedule]);
      setRecipients([]);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Gagal membuat jadwal");
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async (id: string) => {
    try {
      await deleteSchedule(id);
      setSchedules((prev) => prev.filter((s) => s.id !== id));
    } catch {
      // Tangani tanpa menampilkan error
    }
  };

  if (!open) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
      <div className="max-h-[90vh] w-full max-w-lg overflow-y-auto rounded-xl bg-white p-6 shadow-xl">
        <div className="mb-4 flex items-center justify-between">
          <h2 className="text-lg font-semibold text-slate-900">Jadwalkan Laporan</h2>
          <button
            type="button"
            onClick={onClose}
            className="rounded-lg p-2 text-slate-400 hover:bg-slate-100"
            style={{ minHeight: 44, minWidth: 44 }}
            aria-label="Tutup"
          >
            <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" d="M6 18 18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <div className="space-y-4">
          {/* Laporan*/}
          <div>
            <label className="mb-1 block text-sm font-medium text-slate-700">Laporan</label>
            <select
              value={reportType}
              onChange={(e) => setReportType(e.target.value)}
              className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
              style={{ minHeight: 44 }}
            >
              {REPORT_TYPES.map((r) => (
                <option key={r.value} value={r.value}>{r.label}</option>
              ))}
            </select>
          </div>

          {/* Jadwal*/}
          <div>
            <label className="mb-1 block text-sm font-medium text-slate-700">Jadwal</label>
            <div className="flex gap-2">
              {SCHEDULE_TYPES.map((s) => (
                <button
                  key={s.value}
                  type="button"
                  onClick={() => setScheduleType(s.value)}
                  className={`rounded-lg px-3 py-2 text-sm font-medium transition-colors ${
                    scheduleType === s.value
                      ? "bg-blue-600 text-white"
                      : "bg-slate-100 text-slate-700 hover:bg-slate-200"
                  }`}
                  style={{ minHeight: 44 }}
                >
                  {s.label}
                </button>
              ))}
            </div>
          </div>

          {/* Format*/}
          <div>
            <label className="mb-1 block text-sm font-medium text-slate-700">Format</label>
            <div className="flex gap-2">
              {(["pdf", "xlsx"] as const).map((f) => (
                <button
                  key={f}
                  type="button"
                  onClick={() => setFormat(f)}
                  className={`rounded-lg px-4 py-2 text-sm font-medium uppercase transition-colors ${
                    format === f
                      ? "bg-blue-600 text-white"
                      : "bg-slate-100 text-slate-700 hover:bg-slate-200"
                  }`}
                  style={{ minHeight: 44 }}
                >
                  {f}
                </button>
              ))}
            </div>
          </div>

          {/* Penerima*/}
          <div>
            <label className="mb-1 block text-sm font-medium text-slate-700">Penerima</label>
            <div className="flex gap-2">
              <select
                value={recipientType}
                onChange={(e) => setRecipientType(e.target.value as "email" | "whatsapp")}
                className="rounded-lg border border-slate-300 px-3 py-2 text-sm"
                style={{ minHeight: 44 }}
              >
                <option value="email">Email</option>
                <option value="whatsapp">WhatsApp</option>
              </select>
              <input
                type="text"
                value={recipientAddress}
                onChange={(e) => setRecipientAddress(e.target.value)}
                placeholder={recipientType === "email" ? "email@contoh.com" : "08123456789"}
                className="flex-1 rounded-lg border border-slate-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
                style={{ minHeight: 44 }}
                onKeyDown={(e) => e.key === "Enter" && addRecipient()}
              />
              <button
                type="button"
                onClick={addRecipient}
                className="rounded-lg bg-slate-100 px-3 py-2 text-sm font-medium text-slate-700 hover:bg-slate-200"
                style={{ minHeight: 44 }}
              >
                Tambah
              </button>
            </div>
            {recipients.length > 0 && (
              <div className="mt-2 flex flex-wrap gap-2">
                {recipients.map((r, i) => (
                  <span
                    key={i}
                    className="inline-flex items-center gap-1 rounded-full bg-blue-50 px-3 py-1 text-xs text-blue-700"
                  >
                    {r.type === "whatsapp" ? "📱" : "📧"} {r.address}
                    <button type="button" onClick={() => removeRecipient(i)} className="ml-1 text-blue-400 hover:text-blue-600">
                      ×
                    </button>
                  </span>
                ))}
              </div>
            )}
          </div>

          {error && (
            <div className="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">{error}</div>
          )}

          <button
            type="button"
            onClick={handleCreate}
            disabled={saving}
            className="w-full rounded-lg bg-blue-600 px-4 py-2.5 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
            style={{ minHeight: 44 }}
          >
            {saving ? "Menyimpan..." : "Buat Jadwal"}
          </button>
        </div>

        {/* Active schedules*/}
        {!loading && schedules.length > 0 && (
          <div className="mt-6 border-t border-slate-200 pt-4">
            <h3 className="mb-3 text-sm font-medium text-slate-700">Jadwal Aktif</h3>
            <div className="space-y-2">
              {schedules.map((s) => (
                <div key={s.id} className="flex items-center justify-between rounded-lg border border-slate-100 p-3">
                  <div>
                    <p className="text-sm font-medium text-slate-900">{s.report_type}</p>
                    <p className="text-xs text-slate-500">
                      {s.schedule_type} · {s.format.toUpperCase()} · {s.recipients.length} penerima
                    </p>
                  </div>
                  <button
                    type="button"
                    onClick={() => handleDelete(s.id)}
                    className="rounded-lg p-2 text-red-400 hover:bg-red-50 hover:text-red-600"
                    style={{ minHeight: 44, minWidth: 44 }}
                    aria-label="Hapus jadwal"
                  >
                    <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" d="m14.74 9-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 0 1-2.244 2.077H8.084a2.25 2.25 0 0 1-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 0 0-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 0 1 3.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 0 0-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 0 0-7.5 0" />
                    </svg>
                  </button>
                </div>
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
