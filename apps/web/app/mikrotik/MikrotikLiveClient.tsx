"use client";

import { useEffect, useMemo, useState } from "react";
import { ArrowClockwise, CheckCircle, WarningCircle } from "@phosphor-icons/react";
import { Button, DataTable, EmptyState, PageHeader, Section, StatGrid, StatusBadge } from "../components/ui";
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
            <EmptyState title="Belum ada router" description="Seed CHR MikroTik akan membuat router pertama untuk tenant demo." />
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
                      columns={["User", "IP", "Caller ID", "Uptime", "Traffic"]}
                      rows={sessions.map((session) => [
                        session.username,
                        session.address || "-",
                        session.caller_id || "-",
                        session.uptime || "-",
                        `${formatMemory(session.bytes_in + session.bytes_out)}`,
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
                    columns={["Username", "Profile", "Remote IP", "Sync", "Status"]}
                    rows={pppoeUsers.map((user) => [
                      user.username,
                      user.profile_name,
                      user.remote_address || "-",
                      user.sync_status,
                      <StatusBadge key={user.id} status={user.disabled ? "disabled" : user.status} />,
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
