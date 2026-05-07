"use client";

import { useEffect, useMemo, useState, type FormEvent } from "react";
import { ArrowClockwise, CheckCircle, WarningCircle } from "@phosphor-icons/react";
import { Button, DataTable, EmptyState, FormField, PageHeader, Section, StatGrid, StatusBadge, TextInput } from "../components/ui";
import AppShell from "../components/app-shell";
import { MikrotikModuleNav } from "./components/MikrotikModuleNav";

type RouterRecord = {
  id: string;
  name: string;
  host: string;
  port: number;
  username: string;
  use_ssl: boolean;
  service_types: string[];
  router_os_version?: string;
  board_name?: string;
  cpu_count?: number;
  total_ram_mb?: number;
  identity?: string;
  status: string;
  health_check_interval_sec: number;
  last_online_at?: string;
  last_checked_at?: string;
  last_uptime_sec?: number;
  failure_count: number;
  notes?: string;
};

type RouterEditForm = {
  name: string;
  host: string;
  port: string;
  username: string;
  password: string;
  useSsl: boolean;
  status: string;
  healthCheckIntervalSec: string;
  notes: string;
};

type RouterListResponse = {
  success: boolean;
  data?: {
    items: RouterRecord[];
    total: number;
  };
  error?: {
    code: string;
    message: string;
  };
};

type SummaryResponse = {
  success: boolean;
  data?: {
    total_routers: number;
    online_count: number;
    offline_count: number;
    maintenance_count: number;
  };
};

type CreateRouterResponse = {
  success: boolean;
  data?: {
    router: RouterRecord;
    warning?: string;
  };
  error?: {
    code: string;
    message: string;
  };
};

type SystemResource = {
  version: string;
  board_name: string;
  cpu?: string;
  cpu_count: number;
  cpu_frequency_mhz?: number;
  cpu_load: number;
  total_ram: number;
  free_ram: number;
  total_hdd_space?: number;
  free_hdd_space?: number;
  write_sect_since_reboot?: number;
  write_sect_total?: number;
  uptime: number;
  architecture: string;
  build_time?: string;
  identity: string;
};

type PPPoEUser = {
  id: string;
  username: string;
  profile_name: string;
  remote_address?: string;
  disabled: boolean;
  status: string;
  sync_status: string;
  last_sync_at?: string;
};

type PPPoESession = {
  id: string;
  username: string;
  caller_id: string;
  address: string;
  uptime: string;
  bytes_in: number;
  bytes_out: number;
  service: string;
};

type SyncStatus = {
  synced_count: number;
  orphan_count: number;
  missing_count: number;
  out_of_sync_count: number;
  last_sync_at?: string;
};

function formatUptime(seconds?: number) {
  if (seconds == null) return "-";
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  if (days > 0) return `${days}d ${hours}h`;
  if (hours > 0) return `${hours}h ${minutes}m`;
  return `${minutes}m`;
}

function formatMemory(bytes?: number) {
  if (!bytes) return "-";
  return `${Math.round(bytes / 1024 / 1024)} MB`;
}

function extractMessage(error: unknown) {
  return error instanceof Error ? error.message : "Terjadi kesalahan";
}

function routerToEditForm(router: RouterRecord): RouterEditForm {
  return {
    name: router.name,
    host: router.host,
    port: String(router.port || 8728),
    username: router.username,
    password: "",
    useSsl: router.use_ssl,
    status: router.status,
    healthCheckIntervalSec: String(router.health_check_interval_sec || 300),
    notes: router.notes || "",
  };
}

export function MikrotikLivePage() {
  const [routers, setRouters] = useState<RouterRecord[]>([]);
  const [summary, setSummary] = useState<SummaryResponse["data"]>();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [testingId, setTestingId] = useState<string | null>(null);
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [testResult, setTestResult] = useState<{ router: string; resource: SystemResource } | null>(null);

  async function loadRouters() {
    setLoading(true);
    setError("");
    try {
      const [routerResponse, summaryResponse] = await Promise.all([
        fetch("/api/network/mikrotik/routers", { cache: "no-store" }),
        fetch("/api/network/mikrotik/status/summary", { cache: "no-store" }),
      ]);
      const routerJson = (await routerResponse.json()) as RouterListResponse;
      const summaryJson = (await summaryResponse.json()) as SummaryResponse;

      if (!routerResponse.ok || !routerJson.success) {
        throw new Error(routerJson.error?.message || "Gagal mengambil router");
      }
      setRouters(routerJson.data?.items || []);
      setSummary(summaryJson.data);
    } catch (loadError) {
      setError(extractMessage(loadError));
    } finally {
      setLoading(false);
    }
  }

  async function testConnection(router: RouterRecord) {
    setTestingId(router.id);
    setTestResult(null);
    try {
      const response = await fetch(`/api/network/mikrotik/routers/${router.id}/test`, { method: "POST" });
      const json = await response.json();
      if (!response.ok || !json.success) throw new Error(json.error?.message || "Test koneksi gagal");
      setTestResult({ router: router.name, resource: json.data as SystemResource });
      await loadRouters();
    } catch (testError) {
      setError(extractMessage(testError));
    } finally {
      setTestingId(null);
    }
  }

  async function deleteRouterFromList(router: RouterRecord) {
    const ok = window.confirm(`Hapus router ${router.name}? Data router akan dihapus dari aplikasi, tanpa mengubah konfigurasi di MikroTik.`);
    if (!ok) return;

    setDeletingId(router.id);
    setError("");
    try {
      const response = await fetch(`/api/network/mikrotik/routers/${router.id}`, { method: "DELETE" });
      const json = await response.json();
      if (!response.ok || !json.success) throw new Error(json.error?.message || "Gagal menghapus router");
      await loadRouters();
    } catch (deleteError) {
      setError(extractMessage(deleteError));
    } finally {
      setDeletingId(null);
    }
  }

  useEffect(() => {
    void loadRouters();
  }, []);

  const stats = useMemo(() => {
    const online = summary?.online_count ?? routers.filter((router) => router.status === "online").length;
    const offline = summary?.offline_count ?? routers.filter((router) => router.status === "offline").length;
    const maintenance = summary?.maintenance_count ?? routers.filter((router) => router.status === "maintenance").length;
    return [
      { label: "Router online", value: `${online}/${summary?.total_routers ?? routers.length}` },
      { label: "Offline", value: String(offline), tone: offline > 0 ? ("red" as const) : undefined },
      { label: "Maintenance", value: String(maintenance), tone: maintenance > 0 ? ("amber" as const) : undefined },
      { label: "Mode", value: "Live CHR" },
    ];
  }, [routers, summary]);

  return (
    <AppShell>
      <div className="space-y-6">
        <PageHeader
          eyebrow="MikroTik"
          title="Router dan PPPoE"
          description="Data router diambil dari network-service dan diuji langsung ke RouterOS API."
          actions={
            <>
              <Button href="/mikrotik/new">Tambah Router</Button>
              <button
                type="button"
                onClick={() => void loadRouters()}
                className="inline-flex min-w-0 items-center justify-center gap-2 rounded-md border border-slate-300 bg-white px-4 py-2 text-center text-sm font-semibold leading-5 text-slate-700 transition hover:bg-slate-50 active:scale-[0.98]"
              >
                <ArrowClockwise size={16} />
                Refresh
              </button>
            </>
          }
        />

        <MikrotikModuleNav />

        <StatGrid stats={stats} />

        {error && (
          <div className="flex gap-3 rounded-xl border border-red-200 bg-red-50 p-4 text-sm text-red-700">
            <WarningCircle size={20} className="shrink-0" />
            <span className="min-w-0 [overflow-wrap:anywhere]">{error}</span>
          </div>
        )}

        {testResult && (
          <div className="flex flex-col gap-3 rounded-xl border border-emerald-200 bg-emerald-50 p-4 text-sm text-emerald-800 sm:flex-row sm:items-center sm:justify-between">
            <span className="inline-flex items-center gap-2">
              <CheckCircle size={20} />
              Test koneksi {testResult.router} berhasil.
            </span>
            <span className="font-mono text-xs">
              {testResult.resource.identity} - RouterOS {testResult.resource.version} - CPU {testResult.resource.cpu_load}%
            </span>
          </div>
        )}

        <Section title="Router live" description="Daftar router tenant dari database, dengan aksi test langsung ke CHR.">
          {loading ? (
            <EmptyState title="Memuat router" description="Mengambil data dari network-service..." />
          ) : routers.length === 0 ? (
            <EmptyState
              title="Belum ada router"
              description="Tambahkan CHR atau router kantor pusat. Data disimpan dulu, test koneksi dijalankan manual agar tidak ada login API otomatis."
              action={<Button href="/mikrotik/new">Tambah Router</Button>}
            />
          ) : (
            <DataTable
              columns={["Router", "IP", "Port", "Versi", "Board", "Uptime", "Status", "Aksi"]}
              rows={routers.map((router) => [
                <a key={router.id} href={`/mikrotik/${router.id}`} className="font-semibold text-blue-700">{router.name}</a>,
                router.host,
                String(router.port),
                router.router_os_version || "-",
                router.board_name || "-",
                formatUptime(router.last_uptime_sec),
                <StatusBadge key={`${router.id}-status`} status={router.status} />,
                <div key={`${router.id}-actions`} className="flex flex-wrap gap-2">
                  <a href={`/mikrotik/${router.id}`} className="rounded-md px-3 py-2 text-sm font-semibold text-blue-700 hover:bg-blue-50">
                    Edit
                  </a>
                  <button
                    type="button"
                    disabled={testingId === router.id}
                    onClick={() => void testConnection(router)}
                    className="rounded-md px-3 py-2 text-sm font-semibold text-slate-600 hover:bg-slate-100 disabled:cursor-wait disabled:opacity-60"
                  >
                    {testingId === router.id ? "Testing..." : "Test"}
                  </button>
                  <button
                    type="button"
                    disabled={deletingId === router.id}
                    onClick={() => void deleteRouterFromList(router)}
                    className="rounded-md px-3 py-2 text-sm font-semibold text-red-600 hover:bg-red-50 disabled:cursor-wait disabled:opacity-60"
                  >
                    {deletingId === router.id ? "Menghapus..." : "Hapus"}
                  </button>
                </div>,
              ])}
            />
          )}
        </Section>
      </div>
    </AppShell>
  );
}

export function MikrotikCreatePage() {
  const [form, setForm] = useState({
    name: "SG MIKROTIK CHR BRONZE",
    host: "",
    port: "8728",
    username: "",
    password: "",
    useSsl: false,
    healthCheckIntervalSec: "300",
    serviceTypes: ["pppoe"],
    notes: "",
    testOnCreate: false,
  });
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState<CreateRouterResponse["data"] | null>(null);

  function updateField(field: keyof typeof form, value: string | boolean | string[]) {
    setForm((current) => ({ ...current, [field]: value }));
  }

  function toggleService(service: string) {
    setForm((current) => {
      const exists = current.serviceTypes.includes(service);
      const serviceTypes = exists
        ? current.serviceTypes.filter((item) => item !== service)
        : [...current.serviceTypes, service];
      return { ...current, serviceTypes };
    });
  }

  async function submitRouter(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSaving(true);
    setError("");
    setSuccess(null);
    try {
      const payload = {
        name: form.name.trim(),
        host: form.host.trim(),
        port: Number(form.port || 8728),
        username: form.username.trim(),
        password: form.password,
        use_ssl: form.useSsl,
        service_types: form.serviceTypes.length > 0 ? form.serviceTypes : ["pppoe"],
        health_check_interval_sec: Number(form.healthCheckIntervalSec || 300),
        notes: form.notes.trim(),
        test_on_create: form.testOnCreate,
      };
      const response = await fetch("/api/network/mikrotik/routers", {
        method: "POST",
        body: JSON.stringify(payload),
      });
      const json = (await response.json()) as CreateRouterResponse;
      if (!response.ok || !json.success || !json.data?.router) {
        throw new Error(json.error?.message || "Gagal menyimpan router");
      }
      setSuccess(json.data);
      setForm((current) => ({ ...current, password: "" }));
    } catch (submitError) {
      setError(extractMessage(submitError));
    } finally {
      setSaving(false);
    }
  }

  const createdRouter = success?.router;

  return (
    <AppShell>
      <div className="space-y-6">
        <PageHeader
          eyebrow="MikroTik"
          title="Tambah Router"
          description="Simpan koneksi MikroTik ke database tenant. Test RouterOS API hanya berjalan bila dipilih atau melalui tombol test manual di halaman detail."
          actions={<Button variant="secondary" href="/mikrotik">Kembali</Button>}
        />

        <MikrotikModuleNav />

        {error && (
          <div className="flex gap-3 rounded-xl border border-red-200 bg-red-50 p-4 text-sm text-red-700">
            <WarningCircle size={20} className="shrink-0" />
            <span className="min-w-0 [overflow-wrap:anywhere]">{error}</span>
          </div>
        )}

        {createdRouter && (
          <div className="flex flex-col gap-3 rounded-xl border border-emerald-200 bg-emerald-50 p-4 text-sm text-emerald-800 sm:flex-row sm:items-center sm:justify-between">
            <span className="inline-flex items-center gap-2">
              <CheckCircle size={20} />
              Router {createdRouter.name} berhasil disimpan.
            </span>
            <a className="font-semibold text-emerald-900 underline-offset-4 hover:underline" href={`/mikrotik/${createdRouter.id}`}>
              Buka detail
            </a>
          </div>
        )}

        <Section
          title="Koneksi RouterOS"
          description="Gunakan port 8728 untuk API biasa atau 8729 untuk API-SSL. Untuk CHR lokal testing, pastikan service api/api-ssl di MikroTik sudah di-enable dan firewall mengizinkan IP aplikasi."
        >
          <form onSubmit={(event) => void submitRouter(event)} className="grid gap-5">
            <div className="grid gap-4 lg:grid-cols-2">
              <FormField label="Nama router">
                <TextInput value={form.name} onChange={(event) => updateField("name", event.target.value)} required />
              </FormField>
              <FormField label="Host / IP publik">
                <TextInput value={form.host} onChange={(event) => updateField("host", event.target.value)} placeholder="82.41.42.51" required />
              </FormField>
              <FormField label="Username">
                <TextInput value={form.username} onChange={(event) => updateField("username", event.target.value)} autoComplete="username" required />
              </FormField>
              <FormField label="Password">
                <TextInput
                  type="password"
                  value={form.password}
                  onChange={(event) => updateField("password", event.target.value)}
                  autoComplete="current-password"
                  required={!createdRouter}
                />
              </FormField>
              <FormField label="Port API">
                <TextInput type="number" min={1} max={65535} value={form.port} onChange={(event) => updateField("port", event.target.value)} required />
              </FormField>
              <FormField label="Interval health check" helper="Scheduler periodik tetap nonaktif secara default; nilai ini dipakai saat fitur monitoring diaktifkan.">
                <TextInput
                  type="number"
                  min={10}
                  max={3600}
                  value={form.healthCheckIntervalSec}
                  onChange={(event) => updateField("healthCheckIntervalSec", event.target.value)}
                />
              </FormField>
            </div>

            <div className="grid gap-3 rounded-lg border border-slate-200 bg-slate-50 p-4">
              <p className="text-sm font-semibold text-slate-900">Service yang dikelola</p>
              <div className="flex flex-wrap gap-3">
                {["pppoe", "hotspot", "dhcp_binding", "static"].map((service) => (
                  <label key={service} className="inline-flex items-center gap-2 rounded-md border border-slate-200 bg-white px-3 py-2 text-sm text-slate-700">
                    <input
                      type="checkbox"
                      checked={form.serviceTypes.includes(service)}
                      onChange={() => toggleService(service)}
                      className="h-4 w-4 rounded border-slate-300 text-blue-600 focus:ring-blue-500"
                    />
                    <span className="capitalize">{service.replace("_", " ")}</span>
                  </label>
                ))}
              </div>
            </div>

            <FormField label="Catatan">
              <TextInput value={form.notes} onChange={(event) => updateField("notes", event.target.value)} placeholder="Lokasi, upstream, atau akses VPN" />
            </FormField>

            <div className="grid gap-3 rounded-lg border border-amber-200 bg-amber-50 p-4 text-sm text-amber-900">
              <label className="inline-flex items-start gap-3">
                <input
                  type="checkbox"
                  checked={form.useSsl}
                  onChange={(event) => updateField("useSsl", event.target.checked)}
                  className="mt-0.5 h-4 w-4 rounded border-amber-300 text-blue-600 focus:ring-blue-500"
                />
                <span>Gunakan API-SSL. Aktifkan ini hanya jika port 8729 dan sertifikat API-SSL sudah siap.</span>
              </label>
              <label className="inline-flex items-start gap-3">
                <input
                  type="checkbox"
                  checked={form.testOnCreate}
                  onChange={(event) => updateField("testOnCreate", event.target.checked)}
                  className="mt-0.5 h-4 w-4 rounded border-amber-300 text-blue-600 focus:ring-blue-500"
                />
                <span>Test koneksi saat simpan. Jika tidak dicentang, tidak ada login API ke MikroTik sampai tombol Test Connection ditekan manual.</span>
              </label>
            </div>

            <div className="flex flex-col-reverse gap-3 sm:flex-row sm:justify-end">
              <Button variant="secondary" href="/mikrotik">Batal</Button>
              <button
                type="submit"
                disabled={saving}
                className="inline-flex min-w-0 items-center justify-center rounded-md bg-blue-600 px-4 py-2 text-center text-sm font-semibold leading-5 text-white transition hover:bg-blue-700 active:scale-[0.98] disabled:cursor-wait disabled:opacity-60"
              >
                {saving ? "Menyimpan..." : "Simpan Router"}
              </button>
            </div>
          </form>
        </Section>
      </div>
    </AppShell>
  );
}
