"use client";

import { useEffect, useMemo, useState } from "react";
import { ArrowClockwise, CheckCircle, Play, WarningCircle } from "@phosphor-icons/react";
import AppShell from "../../components/app-shell";
import { DataTable, EmptyState, PageHeader, Section, StatGrid, StatusBadge } from "../../components/ui";
import { MikrotikModuleNav } from "../components/MikrotikModuleNav";

type RouterRecord = {
  id: string;
  name: string;
  host: string;
  status: string;
  router_os_version?: string;
};

type BulkResult = {
  router_id: string;
  router_name: string;
  action: string;
  status: string;
  message?: string;
  details?: Record<string, unknown>;
  started_at: string;
  finished_at: string;
};

type BulkJob = {
  id: string;
  action: string;
  status: string;
  router_ids: string[];
  total_count: number;
  success_count: number;
  failed_count: number;
  results: BulkResult[];
  error_message?: string;
  created_at: string;
  finished_at?: string;
};

type Envelope<T> = {
  success: boolean;
  data?: T;
  error?: { message?: string };
};

const actionOptions = [
  { value: "firmware_check", label: "Firmware check", description: "Baca versi RouterOS dan package aktif." },
  { value: "backup", label: "Backup router", description: "Buat backup on-demand per router." },
  { value: "pppoe_sync", label: "Sync PPPoE", description: "Samakan PPPoE DB dan router secara manual." },
];

function formatDate(value?: string) {
  if (!value) return "-";
  return new Intl.DateTimeFormat("id-ID", {
    day: "2-digit",
    month: "short",
    hour: "2-digit",
    minute: "2-digit",
  }).format(new Date(value));
}

function formatAction(value: string) {
  return actionOptions.find((item) => item.value === value)?.label || value.replaceAll("_", " ");
}

function extractMessage(error: unknown) {
  return error instanceof Error ? error.message : "Terjadi kesalahan";
}

async function readJson<T>(response: Response) {
  const json = (await response.json()) as Envelope<T>;
  if (!response.ok || !json.success) {
    throw new Error(json.error?.message || `Request gagal (${response.status})`);
  }
  return json.data as T;
}

export function MikrotikBulkActionsPage() {
  const [routers, setRouters] = useState<RouterRecord[]>([]);
  const [jobs, setJobs] = useState<BulkJob[]>([]);
  const [selected, setSelected] = useState<string[]>([]);
  const [action, setAction] = useState("firmware_check");
  const [scope, setScope] = useState<"selected" | "all_active">("selected");
  const [loading, setLoading] = useState(true);
  const [running, setRunning] = useState(false);
  const [error, setError] = useState("");
  const [lastJob, setLastJob] = useState<BulkJob | null>(null);

  async function load() {
    setLoading(true);
    setError("");
    try {
      const [routerData, jobData] = await Promise.all([
        fetch("/api/network/mikrotik/routers", { cache: "no-store" }).then((res) => readJson<{ items: RouterRecord[] }>(res)),
        fetch("/api/network/mikrotik/bulk-jobs?page_size=10", { cache: "no-store" }).then((res) => readJson<{ data: BulkJob[] }>(res)),
      ]);
      setRouters(routerData.items || []);
      setJobs(jobData.data || []);
    } catch (err) {
      setError(extractMessage(err));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void load();
  }, []);

  const targetCount = scope === "all_active" ? routers.filter((router) => router.status !== "maintenance").length : selected.length;
  const selectedSet = useMemo(() => new Set(selected), [selected]);

  function toggleRouter(id: string) {
    setSelected((current) => (current.includes(id) ? current.filter((item) => item !== id) : [...current, id]));
  }

  async function runBulkAction() {
    setRunning(true);
    setError("");
    setLastJob(null);
    try {
      const response = await fetch("/api/network/mikrotik/bulk-jobs", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ action, scope, router_ids: scope === "selected" ? selected : [] }),
      });
      const job = await readJson<BulkJob>(response);
      setLastJob(job);
      await load();
    } catch (err) {
      setError(extractMessage(err));
    } finally {
      setRunning(false);
    }
  }

  return (
    <AppShell>
      <div className="space-y-6">
        <PageHeader
          eyebrow="MikroTik"
          title="Bulk actions"
          description="Jalankan aksi operasional ke beberapa router secara manual tanpa scheduler atau polling otomatis."
          actions={
            <button
              type="button"
              onClick={() => void load()}
              className="inline-flex h-10 items-center justify-center gap-2 rounded-md border border-slate-300 bg-white px-4 text-sm font-semibold text-slate-700 transition hover:border-blue-200 hover:text-blue-700"
            >
              <ArrowClockwise size={18} />
              Refresh
            </button>
          }
        />
        <MikrotikModuleNav />

        <StatGrid
          stats={[
            { label: "Router tersedia", value: String(routers.length) },
            { label: "Target aksi", value: String(targetCount), tone: targetCount > 0 ? "blue" : "amber" },
            { label: "Job terakhir", value: lastJob ? formatAction(lastJob.action) : "-" },
            { label: "Status terakhir", value: lastJob?.status?.replaceAll("_", " ") || "-" },
          ]}
        />

        {error && (
          <div className="flex items-start gap-2 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
            <WarningCircle size={18} className="mt-0.5 shrink-0" />
            <span>{error}</span>
          </div>
        )}

        <Section title="Jalankan bulk action" description="Pilih action dan router target. Semua koneksi RouterOS terjadi hanya setelah tombol dijalankan.">
          <div className="grid gap-5 xl:grid-cols-[1fr_1.4fr]">
            <div className="space-y-4">
              <div>
                <label className="text-xs font-semibold uppercase tracking-[0.18em] text-slate-500">Action</label>
                <div className="mt-2 grid gap-2">
                  {actionOptions.map((item) => (
                    <label key={item.value} className={`flex cursor-pointer gap-3 rounded-lg border p-3 transition ${action === item.value ? "border-blue-300 bg-blue-50" : "border-slate-200 bg-white hover:border-blue-200"}`}>
                      <input type="radio" name="action" value={item.value} checked={action === item.value} onChange={() => setAction(item.value)} className="mt-1" />
                      <span>
                        <span className="block text-sm font-semibold text-slate-950">{item.label}</span>
                        <span className="mt-1 block text-sm leading-5 text-slate-500">{item.description}</span>
                      </span>
                    </label>
                  ))}
                </div>
              </div>

              <div>
                <label className="text-xs font-semibold uppercase tracking-[0.18em] text-slate-500">Scope</label>
                <div className="mt-2 grid grid-cols-2 overflow-hidden rounded-lg border border-slate-200 bg-white">
                  {[
                    { value: "selected", label: "Dipilih" },
                    { value: "all_active", label: "Semua aktif" },
                  ].map((item) => (
                    <button
                      key={item.value}
                      type="button"
                      onClick={() => setScope(item.value as "selected" | "all_active")}
                      className={`h-10 text-sm font-semibold transition ${scope === item.value ? "bg-blue-600 text-white" : "text-slate-600 hover:bg-blue-50"}`}
                    >
                      {item.label}
                    </button>
                  ))}
                </div>
              </div>

              <button
                type="button"
                onClick={() => void runBulkAction()}
                disabled={running || targetCount === 0}
                className="inline-flex h-11 w-full items-center justify-center gap-2 rounded-md bg-blue-600 px-4 text-sm font-semibold text-white transition hover:bg-blue-700 disabled:cursor-not-allowed disabled:opacity-60"
              >
                {running ? <ArrowClockwise size={18} className="animate-spin" /> : <Play size={18} weight="fill" />}
                {running ? "Memproses..." : `Jalankan ke ${targetCount} router`}
              </button>
            </div>

            <div>
              <div className="mb-3 flex items-center justify-between">
                <h3 className="text-sm font-semibold text-slate-950">Router target</h3>
                <button type="button" onClick={() => setSelected(routers.map((router) => router.id))} className="text-sm font-semibold text-blue-700 hover:text-blue-800">
                  Pilih semua
                </button>
              </div>
              {loading ? (
                <p className="text-sm text-slate-500">Memuat router...</p>
              ) : routers.length === 0 ? (
                <EmptyState title="Belum ada router" description="Tambahkan router MikroTik dulu sebelum menjalankan bulk action." />
              ) : (
                <div className="grid max-h-[420px] gap-2 overflow-auto pr-1">
                  {routers.map((router) => {
                    const checked = selectedSet.has(router.id);
                    return (
                      <label key={router.id} className={`flex cursor-pointer items-center gap-3 rounded-lg border p-3 transition ${checked ? "border-blue-300 bg-blue-50" : "border-slate-200 bg-white hover:border-blue-200"}`}>
                        <input type="checkbox" checked={checked} onChange={() => toggleRouter(router.id)} disabled={scope === "all_active"} />
                        <span className="min-w-0 flex-1">
                          <span className="block truncate text-sm font-semibold text-slate-950">{router.name}</span>
                          <span className="mt-0.5 block truncate text-xs text-slate-500">{router.host} {router.router_os_version ? `- ${router.router_os_version}` : ""}</span>
                        </span>
                        <StatusBadge status={router.status} />
                      </label>
                    );
                  })}
                </div>
              )}
            </div>
          </div>
        </Section>

        {lastJob && (
          <Section title="Hasil terakhir" description={lastJob.error_message || "Ringkasan hasil bulk action terakhir."}>
            <DataTable
              columns={["Router", "Action", "Status", "Pesan"]}
              rows={lastJob.results.map((result) => [
                result.router_name || result.router_id,
                formatAction(result.action),
                <StatusBadge key={`${result.router_id}-status`} status={result.status} />,
                result.message || "-",
              ])}
            />
          </Section>
        )}

        <Section title="Riwayat bulk job" description="Job terbaru tersimpan untuk audit tenant.">
          {jobs.length === 0 && !loading ? (
            <EmptyState title="Belum ada bulk job" description="Riwayat akan muncul setelah bulk action pertama dijalankan." />
          ) : (
            <DataTable
              columns={["Waktu", "Action", "Status", "Router", "Berhasil", "Gagal"]}
              rows={jobs.map((job) => [
                formatDate(job.created_at),
                formatAction(job.action),
                <StatusBadge key={`${job.id}-status`} status={job.status} />,
                String(job.total_count),
                <span key={`${job.id}-ok`} className="inline-flex items-center gap-1 text-emerald-700"><CheckCircle size={16} />{job.success_count}</span>,
                String(job.failed_count),
              ])}
            />
          )}
        </Section>
      </div>
    </AppShell>
  );
}

