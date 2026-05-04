"use client";

import { useEffect, useState, type FormEvent } from "react";
import { useRouter } from "next/navigation";
import { CheckCircle, WarningCircle } from "@phosphor-icons/react";
import AppShell from "../../../components/app-shell";
import { EmptyState, PageHeader, StatGrid } from "../../../components/ui";
import { MikrotikModuleNav } from "../MikrotikModuleNav";
import { extractMessage, routerToEditForm } from "../../lib/format";
import type {
  MikrotikDetailSection,
  PPPoESession,
  PPPoEUser,
  RouterEditForm,
  RouterRecord,
  SyncStatus,
  SystemResource,
} from "../../lib/types";
import { DHCPPanel } from "./DHCPPanel";
import { FirewallPanel } from "./FirewallPanel";
import { InterfacesPanel } from "./InterfacesPanel";
import { IPPoolsPanel } from "./IPPoolsPanel";
import { LogsPanel } from "./LogsPanel";
import { OverviewPanel } from "./OverviewPanel";
import { PppoeUsersPanel } from "./PppoeUsersPanel";
import { RouterEditPanel } from "./RouterEditPanel";
import { SessionsPanel } from "./SessionsPanel";
import { SyncPanel } from "./SyncPanel";
import { TrafficPanel } from "./TrafficPanel";

export function MikrotikRouterWorkspace({
  routerId,
  section,
}: {
  routerId: string;
  section: MikrotikDetailSection;
}) {
  const navigation = useRouter();
  const [router, setRouter] = useState<RouterRecord | null>(null);
  const [system, setSystem] = useState<SystemResource | null>(null);
  const [pppoeUsers, setPppoeUsers] = useState<PPPoEUser[]>([]);
  const [sessions, setSessions] = useState<PPPoESession[]>([]);
  const [syncStatus, setSyncStatus] = useState<SyncStatus | null>(null);
  const [editForm, setEditForm] = useState<RouterEditForm | null>(null);
  const [editMode, setEditMode] = useState(false);
  const [loading, setLoading] = useState(true);
  const [loadingSessions, setLoadingSessions] = useState(false);
  const [sessionsLoaded, setSessionsLoaded] = useState(false);
  const [syncing, setSyncing] = useState(false);
  const [savingRouter, setSavingRouter] = useState(false);
  const [deletingRouter, setDeletingRouter] = useState(false);
  const [actionBusy, setActionBusy] = useState<string | null>(null);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");

  async function loadRouter() {
    setLoading(true);
    setError("");
    try {
      const [routerResponse, usersResponse, syncResponse] = await Promise.all([
        fetch(`/api/network/mikrotik/routers/${routerId}`, { cache: "no-store" }),
        fetch(`/api/network/mikrotik/routers/${routerId}/pppoe/users?page_size=50`, { cache: "no-store" }),
        fetch(`/api/network/mikrotik/routers/${routerId}/pppoe/sync-status`, { cache: "no-store" }),
      ]);

      const routerJson = await routerResponse.json();
      if (!routerResponse.ok || !routerJson.success) throw new Error(routerJson.error?.message || "Gagal mengambil router");

      const routerData = routerJson.data.router as RouterRecord;
      setRouter(routerData);
      setEditForm(routerToEditForm(routerData));

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
    setSuccess("");
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

  async function submitRouterUpdate(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!editForm || !router) return;
    setSavingRouter(true);
    setError("");
    setSuccess("");
    try {
      const payload: Record<string, unknown> = {
        name: editForm.name.trim(),
        host: editForm.host.trim(),
        port: Number(editForm.port || 8728),
        username: editForm.username.trim(),
        use_ssl: editForm.useSsl,
        health_check_interval_sec: Number(editForm.healthCheckIntervalSec || 300),
        notes: editForm.notes.trim(),
      };
      if (editForm.status !== router.status) payload.status = editForm.status;
      if (editForm.password.trim()) payload.password = editForm.password;

      const response = await fetch(`/api/network/mikrotik/routers/${routerId}`, {
        method: "PUT",
        body: JSON.stringify(payload),
      });
      const json = await response.json();
      if (!response.ok || !json.success) throw new Error(json.error?.message || "Gagal mengubah router");
      setSuccess("Router berhasil diperbarui.");
      setEditMode(false);
      await loadRouter();
    } catch (updateError) {
      setError(extractMessage(updateError));
    } finally {
      setSavingRouter(false);
    }
  }

  async function deleteRouter() {
    if (!router) return;
    const ok = window.confirm(`Hapus router ${router.name}? Data router akan dihapus dari aplikasi, tanpa mengubah konfigurasi di MikroTik.`);
    if (!ok) return;
    setDeletingRouter(true);
    setError("");
    try {
      const response = await fetch(`/api/network/mikrotik/routers/${routerId}`, { method: "DELETE" });
      const json = await response.json();
      if (!response.ok || !json.success) throw new Error(json.error?.message || "Gagal menghapus router");
      navigation.push("/mikrotik");
      navigation.refresh();
    } catch (deleteError) {
      setError(extractMessage(deleteError));
    } finally {
      setDeletingRouter(false);
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

  async function updatePppoeUserStatus(user: PPPoEUser, disabled: boolean) {
    setActionBusy(`user-toggle:${user.id}`);
    setError("");
    try {
      const response = await fetch(`/api/network/mikrotik/routers/${routerId}/pppoe/users/${user.id}`, {
        method: "PUT",
        body: JSON.stringify({ disabled }),
      });
      const json = await response.json();
      if (!response.ok || !json.success) throw new Error(json.error?.message || "Update PPPoE user gagal");
      await loadRouter();
    } catch (updateError) {
      setError(extractMessage(updateError));
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

  const title = router?.name || "Router detail";
  const description = router ? `${router.host}:${router.port} - ${router.router_os_version || "RouterOS"}` : "Memuat data router live";

  return (
    <AppShell>
      <div className="space-y-6">
        <PageHeader
          eyebrow="MikroTik"
          title={title}
          description={description}
          actions={
            <>
              <button type="button" onClick={() => setEditMode((value) => !value)} className="rounded-md border border-blue-200 bg-blue-50 px-4 py-2 text-sm font-semibold text-blue-700 hover:bg-blue-100">
                {editMode ? "Tutup Edit" : "Edit Router"}
              </button>
              <button type="button" onClick={() => void testConnection()} className="rounded-md border border-slate-300 bg-white px-4 py-2 text-sm font-semibold text-slate-700 hover:bg-slate-50">
                Test Connection
              </button>
              <button type="button" onClick={() => void deleteRouter()} disabled={deletingRouter} className="rounded-md border border-red-200 bg-white px-4 py-2 text-sm font-semibold text-red-600 hover:bg-red-50 disabled:cursor-wait disabled:opacity-60">
                {deletingRouter ? "Menghapus..." : "Hapus"}
              </button>
            </>
          }
        />

        <MikrotikModuleNav />

        <main className="min-w-0 space-y-6">
            {error && (
              <div className="flex gap-3 rounded-xl border border-red-200 bg-red-50 p-4 text-sm text-red-700">
                <WarningCircle size={20} className="shrink-0" />
                <span className="min-w-0 [overflow-wrap:anywhere]">{error}</span>
              </div>
            )}
            {success && (
              <div className="flex gap-3 rounded-xl border border-emerald-200 bg-emerald-50 p-4 text-sm text-emerald-800">
                <CheckCircle size={20} className="shrink-0" />
                <span className="min-w-0 [overflow-wrap:anywhere]">{success}</span>
              </div>
            )}
            {loading || !router || !editForm ? (
              <EmptyState title="Memuat detail router" description="Mengambil detail dari network-service..." />
            ) : (
              <>
                <StatGrid
                  stats={[
                    { label: "Status", value: router.status },
                    { label: "RouterOS", value: router.router_os_version || "-" },
                    { label: "Board", value: router.board_name || "-" },
                    { label: "RAM", value: router.total_ram_mb ? `${router.total_ram_mb} MB` : "-" },
                  ]}
                />
                {editMode && (
                  <RouterEditPanel
                    router={router}
                    editForm={editForm}
                    saving={savingRouter}
                    onSubmit={(event) => void submitRouterUpdate(event)}
                    onUpdate={(field, value) => setEditForm((current) => (current ? { ...current, [field]: value } : current))}
                    onCancel={() => {
                      setEditForm(routerToEditForm(router));
                      setEditMode(false);
                      setError("");
                    }}
                  />
                )}
                {section === "overview" && <OverviewPanel router={router} system={system} />}
                {section === "pppoe" && (
                  <PppoeUsersPanel
                    users={pppoeUsers}
                    actionBusy={actionBusy}
                    onToggle={(user, disabled) => void updatePppoeUserStatus(user, disabled)}
                    onDisconnect={(userId) => void disconnectPppoeUser(userId)}
                    onDelete={(user) => void deletePppoeUser(user)}
                  />
                )}
                {section === "sessions" && (
                  <SessionsPanel
                    sessions={sessions}
                    loaded={sessionsLoaded}
                    loading={loadingSessions}
                    actionBusy={actionBusy}
                    onLoad={() => void loadLiveSessions()}
                    onDisconnect={(sessionId) => void disconnectSession(sessionId)}
                  />
                )}
                {section === "sync" && <SyncPanel syncStatus={syncStatus} syncing={syncing} onSync={() => void syncPppoe()} />}
                {section === "traffic" && <TrafficPanel routerId={routerId} onError={setError} />}
                {section === "interfaces" && <InterfacesPanel routerId={routerId} onError={setError} />}
                {section === "ip-pool" && <IPPoolsPanel routerId={routerId} onError={setError} />}
                {section === "firewall" && <FirewallPanel routerId={routerId} onError={setError} />}
                {section === "logs" && <LogsPanel routerId={routerId} onError={setError} />}
                {section === "dhcp" && <DHCPPanel routerId={routerId} onError={setError} />}
              </>
            )}
        </main>
      </div>
    </AppShell>
  );
}
