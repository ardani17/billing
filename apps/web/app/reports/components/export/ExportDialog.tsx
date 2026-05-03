"use client";

import { useState, useEffect, useRef } from "react";
import { requestExport, getExportStatus } from "../../lib/api";
import type { ReportFilter, ReportJobStatus } from "../../lib/types";

interface ExportDialogProps {
  open: boolean;
  onClose: () => void;
  reportType: string;
  filters?: Partial<ReportFilter>;
}

export function ExportDialog({ open, onClose, reportType, filters }: ExportDialogProps) {
  const [format, setFormat] = useState<"pdf" | "xlsx" | "csv">("pdf");
  const [status, setStatus] = useState<"idle" | "exporting" | "done" | "error">("idle");
  const [jobStatus, setJobStatus] = useState<ReportJobStatus | null>(null);
  const [downloadUrl, setDownloadUrl] = useState<string | null>(null);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null);

  useEffect(() => {
    return () => {
      if (pollRef.current) clearInterval(pollRef.current);
    };
  }, []);

  const handleExport = async () => {
    setStatus("exporting");
    setErrorMsg(null);
    try {
      const { job_id } = await requestExport({
        report_type: reportType,
        format,
        filters: filters as ReportFilter | undefined,
      });

      // CSV langsung selesai
      if (format === "csv") {
        setStatus("done");
        return;
      }

      // Poll status untuk PDF/XLSX
      setJobStatus("processing");
      pollRef.current = setInterval(async () => {
        try {
          const job = await getExportStatus(job_id);
          setJobStatus(job.status);
          if (job.status === "completed") {
            setDownloadUrl(job.download_url ?? null);
            setStatus("done");
            if (pollRef.current) clearInterval(pollRef.current);
          } else if (job.status === "failed") {
            setErrorMsg(job.error ?? "Export gagal");
            setStatus("error");
            if (pollRef.current) clearInterval(pollRef.current);
          }
        } catch {
          // Lanjut polling
        }
      }, 2000);
    } catch (err) {
      setErrorMsg(err instanceof Error ? err.message : "Gagal memulai export");
      setStatus("error");
    }
  };

  if (!open) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
      <div className="w-full max-w-md rounded-xl bg-white p-6 shadow-xl">
        <div className="mb-4 flex items-center justify-between">
          <h2 className="text-lg font-semibold text-slate-900">Export Laporan</h2>
          <button
            type="button"
            onClick={onClose}
            className="rounded-lg p-2 text-slate-400 hover:bg-slate-100 hover:text-slate-600"
            style={{ minHeight: 44, minWidth: 44 }}
            aria-label="Tutup"
          >
            <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" d="M6 18 18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        {status === "idle" && (
          <div className="space-y-4">
            <div>
              <label className="mb-2 block text-sm font-medium text-slate-700">Format</label>
              <div className="flex gap-2">
                {(["pdf", "xlsx", "csv"] as const).map((f) => (
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
            <button
              type="button"
              onClick={handleExport}
              className="w-full rounded-lg bg-blue-600 px-4 py-2.5 text-sm font-medium text-white hover:bg-blue-700"
              style={{ minHeight: 44 }}
            >
              Export
            </button>
          </div>
        )}

        {status === "exporting" && (
          <div className="py-8 text-center">
            <div className="mx-auto mb-3 h-8 w-8 animate-spin rounded-full border-4 border-blue-200 border-t-blue-600" />
            <p className="text-sm text-slate-600">
              {jobStatus === "processing" ? "Sedang memproses laporan..." : "Memulai export..."}
            </p>
          </div>
        )}

        {status === "done" && (
          <div className="py-6 text-center">
            <div className="mx-auto mb-3 flex h-12 w-12 items-center justify-center rounded-full bg-emerald-100">
              <svg className="h-6 w-6 text-emerald-600" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" d="m4.5 12.75 6 6 9-13.5" />
              </svg>
            </div>
            <p className="text-sm font-medium text-slate-900">Export berhasil!</p>
            {downloadUrl && (
              <a
                href={downloadUrl}
                className="mt-3 inline-block rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700"
                style={{ minHeight: 44 }}
              >
                Download
              </a>
            )}
          </div>
        )}

        {status === "error" && (
          <div className="py-6 text-center">
            <p className="text-sm text-red-600">{errorMsg}</p>
            <button
              type="button"
              onClick={() => setStatus("idle")}
              className="mt-3 rounded-lg border border-slate-300 px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50"
              style={{ minHeight: 44 }}
            >
              Coba Lagi
            </button>
          </div>
        )}
      </div>
    </div>
  );
}
