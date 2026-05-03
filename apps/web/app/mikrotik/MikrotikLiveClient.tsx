"use client";

import { useEffect, useMemo, useState, type FormEvent } from "react";
import { ArrowClockwise, CheckCircle, WarningCircle } from "@phosphor-icons/react";
import { Button, DataTable, EmptyState, FormField, PageHeader, Section, StatGrid, StatusBadge, TextInput } from "../components/ui";
import AppShell from "../components/app-shell";

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
  cpu_count: number;
  cpu_load: number;
  total_ram: number;
  free_ram: number;
  uptime: number;
  architecture: string;
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
  if (!seconds) return "-";
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

export function MikrotikLivePage() {
  const [routers, setRouters] = useState<RouterRecord[]>([]);
  const [summary, setSummary] = useState<SummaryResponse["data"]>();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [testingId, setTestingId] = useState<string | null>(null);
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
              <Button variant="secondary" href="/mikrotik/vpn">VPN</Button>
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
                <button
                  key={`${router.id}-test`}
                  type="button"
                  disabled={testingId === router.id}
                  onClick={() => void testConnection(router)}
                  className="rounded-md px-3 py-2 text-sm font-semibold text-slate-600 hover:bg-slate-100 disabled:cursor-wait disabled:opacity-60"
                >
                  {testingId === router.id ? "Testing..." : "Test"}
                </button>,
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

export function MikrotikLiveDetailPage({ routerId }: { routerId: string }) {
  const [router, setRouter] = useState<RouterRecord | null>(null);
  const [system, setSystem] = useState<SystemResource | null>(null);
  const [pppoeUsers, setPppoeUsers] = useState<PPPoEUser[]>([]);
  const [sessions, setSessions] = useState<PPPoESession[]>([]);
  const [sessionsLoaded, setSessionsLoaded] = useState(false);
  const [syncStatus, setSyncStatus] = useState<SyncStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [loadingSessions, setLoadingSessions] = useState(false);
  const [syncing, setSyncing] = useState(false);
  const [actionBusy, setActionBusy] = useState<string | null>(null);
  const [error, setError] = useState("");

  async function loadRouter() {
    setLoading(true);
    setError("");
    try {
      const [response, usersResponse, syncResponse] = await Promise.all([
        fetch(`/api/network/mikrotik/routers/${routerId}`, { cache: "no-store" }),
        fetch(`/api/network/mikrotik/routers/${routerId}/pppoe/users?page_size=50`, { cache: "no-store" }),
        fetch(`/api/network/mikrotik/routers/${routerId}/pppoe/sync-status`, { cache: "no-store" }),
      ]);
      const json = await response.json();
      if (!response.ok || !json.success) throw new Error(json.error?.message || "Gagal mengambil router");
      setRouter(json.data.router as RouterRecord);
      const usersJson = await usersResponse.json();
      const syncJson = await syncResponse.json();
      if (usersResponse.ok && usersJson.success) setPppoeUsers(usersJson.data?.items || []);
      if (syncResponse.ok && syncJson.success) setSyncStatus(syncJson.data || null);
    } catch (loadError) {
      setError(extractMessage(loadError));
    } finally {
      setLoading(false);
    }
  }

  async function testConnection() {
    setSystem(null);
    setError("");
    try {
      const response = await fetch(`/api/network/mikrotik/routers/${routerId}/test`, { method: "POST" });
      const json = await response.json();
      if (!response.ok || !json.success) throw new Error(json.error?.message || "Test koneksi gagal");
      setSystem(json.data as SystemResource);
      await loadRouter();
    } catch (testError) {
      setError(extractMessage(testError));
    }
  }

  async function loadLiveSessions() {
    setLoadingSessions(true);
    setError("");
    try {
      const response = await fetch(`/api/network/mikrotik/routers/${routerId}/pppoe/sessions`, { cache: "no-store" });
      const json = await response.json();
      if (!response.ok || !json.success) throw new Error(json.error?.message || "Gagal mengambil session live");
      setSessions(json.data || []);
      setSessionsLoaded(true);
    } catch (sessionError) {
      setError(extractMessage(sessionError));
    } finally {
      setLoadingSessions(false);
    }
  }

  async function syncPppoe() {
    setSyncing(true);
    setError("");
    try {
      const response = await fetch(`/api/network/mikrotik/routers/${routerId}/pppoe/sync`, { method: "POST" });
      const json = await response.json();
      if (!response.ok || !json.success) throw new Error(json.error?.message || "Sync PPPoE gagal");
      await loadRouter();
    } catch (syncError) {
      setError(extractMessage(syncError));
    } finally {
      setSyncing(false);
    }
  }

  async function disconnectSession(sessionId: string) {
    setActionBusy(`session:${sessionId}`);
    setError("");
    try {
      const response = await fetch(`/api/network/mikrotik/routers/${routerId}/pppoe/sessions/${sessionId}/disconnect`, { method: "POST" });
      const json = await response.json();
      if (!response.ok || !json.success) throw new Error(json.error?.message || "Disconnect session gagal");
      await loadLiveSessions();
    } catch (disconnectError) {
      setError(extractMessage(disconnectError));
    } finally {
      setActionBusy(null);
    }
  }

  async function disconnectPppoeUser(userId: string) {
    setActionBusy(`user-disconnect:${userId}`);
    setError("");
    try {
      const response = await fetch(`/api/network/mikrotik/routers/${routerId}/pppoe/users/${userId}/disconnect`, { method: "POST" });
      const json = await response.json();
      if (!response.ok || !json.success) throw new Error(json.error?.message || "Disconnect PPPoE user gagal");
      if (sessionsLoaded) await loadLiveSessions();
    } catch (disconnectError) {
      setError(extractMessage(disconnectError));
    } finally {
      setActionBusy(null);
    }
  }

  async function deletePppoeUser(user: PPPoEUser) {
    const ok = window.confirm(`Hapus PPPoE user ${user.username} dari router dan database?`);
    if (!ok) return;

    setActionBusy(`user-delete:${user.id}`);
    setError("");
    try {
      const response = await fetch(`/api/network/mikrotik/routers/${routerId}/pppoe/users/${user.id}`, { method: "DELETE" });
      const json = await response.json();
      if (!response.ok || !json.success) throw new Error(json.error?.message || "Hapus PPPoE user gagal");
      await loadRouter();
      if (sessionsLoaded) await loadLiveSessions();
    } catch (deleteError) {
      setError(extractMessage(deleteError));
    } finally {
      setActionBusy(null);
    }
  }

  useEffect(() => {
    void loadRouter();
  }, [routerId]);

  return (
    <AppShell>
      <div className="space-y-6">
        <PageHeader
          eyebrow="MikroTik"
          title={router?.name || "Router detail"}
          description={router ? `${router.host}:${router.port} - ${router.router_os_version || "RouterOS"}` : "Memuat data router live"}
          actions={
            <>
              <button type="button" onClick={() => void syncPppoe()} disabled={syncing} className="rounded-md border border-slate-300 bg-white px-4 py-2 text-sm font-semibold text-slate-700 hover:bg-slate-50 disabled:cursor-wait disabled:opacity-60">
                {syncing ? "Syncing..." : "Sync PPPoE"}
              </button>
              <button type="button" onClick={() => void testConnection()} className="rounded-md border border-slate-300 bg-white px-4 py-2 text-sm font-semibold text-slate-700 hover:bg-slate-50">
                Test Connection
              </button>
            </>
          }
        />

        {error && (
          <div className="flex gap-3 rounded-xl border border-red-200 bg-red-50 p-4 text-sm text-red-700">
            <WarningCircle size={20} className="shrink-0" />
            <span className="min-w-0 [overflow-wrap:anywhere]">{error}</span>
          </div>
        )}

        {loading || !router ? (
          <EmptyState title="Memuat detail router" description="Mengambil detail dari network-service..." />
        ) : (
          <>
            <StatGrid
              stats={[
                { label: "Status", value: router.status },
                { label: "RouterOS", value: router.router_os_version || "-" },
                { label: "Board", value: router.board_name || "-" },
                { label: "RAM", value: router.total_ram_mb ? `${router.total_ram_mb} MB` : "-" },
                { label: "PPPoE aktif", value: sessionsLoaded ? String(sessions.length) : "-" },
              ]}
            />
            <div className="grid gap-6 xl:grid-cols-[1fr_1fr]">
              <Section title="Konfigurasi router">
                <DataTable
                  columns={["Field", "Value"]}
                  rows={[
                    ["ID", router.id],
                    ["Host", router.host],
                    ["Port", String(router.port)],
                    ["Username", router.username],
                    ["Use SSL", router.use_ssl ? "Ya" : "Tidak"],
                    ["Service", router.service_types.join(", ")],
                  ]}
                />
              </Section>
              <Section title="System resource live">
                {system ? (
                  <DataTable
                    columns={["Metric", "Value"]}
                    rows={[
                      ["Identity", system.identity],
                      ["Version", system.version],
                      ["Architecture", system.architecture],
                      ["CPU Load", `${system.cpu_load}%`],
                      ["Free RAM", formatMemory(system.free_ram)],
                      ["Uptime", formatUptime(system.uptime)],
                    ]}
                  />
                ) : (
                  <EmptyState title="Belum ada snapshot live" description="Klik Test Connection untuk membaca system resource dari RouterOS." />
                )}
              </Section>
            </div>
            <div className="grid gap-6 xl:grid-cols-[1fr_1fr]">
              <Section
                title="PPPoE sessions live"
                description="Dibaca dari RouterOS API hanya saat diminta."
              >
                {!sessionsLoaded ? (
                  <EmptyState
                    title="Session live belum dimuat"
                    description="Gunakan aksi manual saat perlu melihat pelanggan yang sedang tersambung."
                    action={
                      <button
                        type="button"
                        onClick={() => void loadLiveSessions()}
                        disabled={loadingSessions}
                        className="inline-flex items-center justify-center gap-2 rounded-md border border-slate-300 bg-white px-4 py-2 text-sm font-semibold text-slate-700 hover:bg-slate-50 disabled:cursor-wait disabled:opacity-60"
                      >
                        <ArrowClockwise size={16} />
                        {loadingSessions ? "Memuat..." : "Muat Session Live"}
                      </button>
                    }
                  />
                ) : sessions.length === 0 ? (
                  <EmptyState
                    title="Belum ada session aktif"
                    description="Router online, tetapi belum ada pelanggan PPPoE yang sedang tersambung."
                    action={
                      <button
                        type="button"
                        onClick={() => void loadLiveSessions()}
                        disabled={loadingSessions}
                        className="inline-flex items-center justify-center gap-2 rounded-md border border-slate-300 bg-white px-4 py-2 text-sm font-semibold text-slate-700 hover:bg-slate-50 disabled:cursor-wait disabled:opacity-60"
                      >
                        <ArrowClockwise size={16} />
                        {loadingSessions ? "Memuat..." : "Refresh Live"}
                      </button>
                    }
                  />
                ) : (
                  <div className="space-y-3">
                    <div className="flex justify-end">
                      <button
                        type="button"
                        onClick={() => void loadLiveSessions()}
                        disabled={loadingSessions}
                        className="inline-flex items-center justify-center gap-2 rounded-md border border-slate-300 bg-white px-3 py-2 text-sm font-semibold text-slate-700 hover:bg-slate-50 disabled:cursor-wait disabled:opacity-60"
                      >
                        <ArrowClockwise size={16} />
                        {loadingSessions ? "Memuat..." : "Refresh Live"}
                      </button>
                    </div>
                    <DataTable
                      columns={["User", "IP", "Caller ID", "Uptime", "Traffic", "Aksi"]}
                      rows={sessions.map((session) => [
                        session.username,
                        session.address || "-",
                        session.caller_id || "-",
                        session.uptime || "-",
                        `${formatMemory(session.bytes_in + session.bytes_out)}`,
                        <button
                          key={`${session.id}-disconnect`}
                          type="button"
                          disabled={actionBusy === `session:${session.id}`}
                          onClick={() => void disconnectSession(session.id)}
                          className="rounded-md px-3 py-2 text-sm font-semibold text-red-600 hover:bg-red-50 disabled:cursor-wait disabled:opacity-60"
                        >
                          {actionBusy === `session:${session.id}` ? "Memutus..." : "Disconnect"}
                        </button>,
                      ])}
                    />
                  </div>
                )}
              </Section>
              <Section
                title="PPPoE users terkelola"
                description="User yang dibuat atau disinkronkan ISPBoss ke router."
              >
                {pppoeUsers.length === 0 ? (
                  <EmptyState
                    title="Belum ada user terkelola"
                    description="Saat pelanggan diaktivasi, ISPBoss akan membuat PPPoE secret dan mencatat status sync di sini."
                  />
                ) : (
                  <DataTable
                    columns={["Username", "Profile", "Remote IP", "Sync", "Status", "Aksi"]}
                    rows={pppoeUsers.map((user) => [
                      user.username,
                      user.profile_name,
                      user.remote_address || "-",
                      user.sync_status,
                      <StatusBadge key={user.id} status={user.disabled ? "disabled" : user.status} />,
                      <div key={`${user.id}-actions`} className="flex flex-wrap gap-2">
                        <button
                          type="button"
                          disabled={actionBusy === `user-disconnect:${user.id}`}
                          onClick={() => void disconnectPppoeUser(user.id)}
                          className="rounded-md px-3 py-2 text-sm font-semibold text-slate-600 hover:bg-slate-100 disabled:cursor-wait disabled:opacity-60"
                        >
                          {actionBusy === `user-disconnect:${user.id}` ? "Memutus..." : "Disconnect"}
                        </button>
                        <button
                          type="button"
                          disabled={actionBusy === `user-delete:${user.id}`}
                          onClick={() => void deletePppoeUser(user)}
                          className="rounded-md px-3 py-2 text-sm font-semibold text-red-600 hover:bg-red-50 disabled:cursor-wait disabled:opacity-60"
                        >
                          {actionBusy === `user-delete:${user.id}` ? "Menghapus..." : "Hapus"}
                        </button>
                      </div>,
                    ])}
                  />
                )}
              </Section>
            </div>
            <Section title="Status sinkronisasi PPPoE">
              <DataTable
                columns={["Metric", "Nilai"]}
                rows={[
                  ["Synced", String(syncStatus?.synced_count ?? 0)],
                  ["Missing di router", String(syncStatus?.missing_count ?? 0)],
                  ["Orphan di router", String(syncStatus?.orphan_count ?? 0)],
                  ["Out of sync", String(syncStatus?.out_of_sync_count ?? 0)],
                  ["Terakhir sync", syncStatus?.last_sync_at ? new Date(syncStatus.last_sync_at).toLocaleString("id-ID") : "-"],
                ]}
              />
            </Section>
          </>
        )}
      </div>
    </AppShell>
  );
}
