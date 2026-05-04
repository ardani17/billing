"use client";

import { useEffect, useMemo, useState, type FormEvent } from "react";
import { ArrowClockwise, Code, Plus } from "@phosphor-icons/react";
import { DataTable, EmptyState, Section, StatusBadge } from "../../../components/ui";
import { extractMessage } from "../../lib/format";
import type { HotspotActiveSession, HotspotLoginTemplate, HotspotProfile, HotspotUser } from "../../lib/types";

type HotspotMode = "users" | "profiles" | "active" | "template";

const tabs: { id: HotspotMode; label: string }[] = [
  { id: "users", label: "Users" },
  { id: "profiles", label: "Profiles" },
  { id: "active", label: "Active" },
  { id: "template", label: "Login template" },
];

const emptyUserForm = { name: "", password: "", profile: "default", limit_uptime: "", comment: "" };
const emptyTemplateForm = {
  brand_name: "ISPBoss Hotspot",
  primary_color: "#2563eb",
  support_phone: "Hubungi admin ISP",
  message: "Masukkan voucher Anda untuk mulai menggunakan internet.",
};

function formatBytes(value: number) {
  if (!value) return "0 B";
  const units = ["B", "KB", "MB", "GB", "TB"];
  const index = Math.min(Math.floor(Math.log(value) / Math.log(1024)), units.length - 1);
  return `${(value / 1024 ** index).toFixed(index === 0 ? 0 : 1)} ${units[index]}`;
}

export function HotspotPanel({ routerId, onError }: { routerId: string; onError: (message: string) => void }) {
  const [mode, setMode] = useState<HotspotMode>("users");
  const [users, setUsers] = useState<HotspotUser[]>([]);
  const [profiles, setProfiles] = useState<HotspotProfile[]>([]);
  const [active, setActive] = useState<HotspotActiveSession[]>([]);
  const [userForm, setUserForm] = useState(emptyUserForm);
  const [templateForm, setTemplateForm] = useState(emptyTemplateForm);
  const [template, setTemplate] = useState<HotspotLoginTemplate | null>(null);
  const [editingId, setEditingId] = useState("");
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [loaded, setLoaded] = useState<Record<HotspotMode, boolean>>({
    users: false,
    profiles: false,
    active: false,
    template: true,
  });

  const profileOptions = useMemo(() => {
    const names = profiles.map((profile) => profile.name).filter(Boolean);
    return names.length > 0 ? names : ["default"];
  }, [profiles]);

  async function load(target = mode) {
    if (target === "template") return;
    setLoading(true);
    onError("");
    try {
      const response = await fetch(`/api/network/mikrotik/routers/${routerId}/hotspot/${target}`, { cache: "no-store" });
      const json = await response.json();
      if (!response.ok || !json.success) throw new Error(json.error?.message || "Gagal mengambil data hotspot");
      if (target === "users") setUsers(json.data || []);
      if (target === "profiles") setProfiles(json.data || []);
      if (target === "active") setActive(json.data || []);
      setLoaded((current) => ({ ...current, [target]: true }));
    } catch (error) {
      onError(extractMessage(error));
    } finally {
      setLoading(false);
    }
  }

  async function submitUser(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSaving(true);
    onError("");
    try {
      const response = await fetch(
        editingId
          ? `/api/network/mikrotik/routers/${routerId}/hotspot/users/${encodeURIComponent(editingId)}`
          : `/api/network/mikrotik/routers/${routerId}/hotspot/users`,
        {
          method: editingId ? "PUT" : "POST",
          body: JSON.stringify(userForm),
        },
      );
      const json = await response.json();
      if (!response.ok || !json.success) throw new Error(json.error?.message || "Gagal menyimpan user hotspot");
      setUserForm(emptyUserForm);
      setEditingId("");
      await load("users");
      setMode("users");
    } catch (error) {
      onError(extractMessage(error));
    } finally {
      setSaving(false);
    }
  }

  async function toggleUser(user: HotspotUser) {
    setSaving(true);
    onError("");
    try {
      const response = await fetch(`/api/network/mikrotik/routers/${routerId}/hotspot/users/${encodeURIComponent(user.id || user.name)}`, {
        method: "PUT",
        body: JSON.stringify({ disabled: !user.disabled }),
      });
      const json = await response.json();
      if (!response.ok || !json.success) throw new Error(json.error?.message || "Gagal mengubah status hotspot");
      await load("users");
    } catch (error) {
      onError(extractMessage(error));
    } finally {
      setSaving(false);
    }
  }

  async function deleteUser(user: HotspotUser) {
    const confirmName = window.prompt(`Ketik username ${user.name} untuk hapus user ini dari RouterOS.`);
    if (confirmName !== user.name) return;
    setSaving(true);
    onError("");
    try {
      const response = await fetch(`/api/network/mikrotik/routers/${routerId}/hotspot/users/${encodeURIComponent(user.id || user.name)}`, { method: "DELETE" });
      const json = await response.json();
      if (!response.ok || !json.success) throw new Error(json.error?.message || "Gagal menghapus user hotspot");
      await load("users");
    } catch (error) {
      onError(extractMessage(error));
    } finally {
      setSaving(false);
    }
  }

  async function generateTemplate(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSaving(true);
    onError("");
    try {
      const response = await fetch(`/api/network/mikrotik/routers/${routerId}/hotspot/login-template/generate`, {
        method: "POST",
        body: JSON.stringify(templateForm),
      });
      const json = await response.json();
      if (!response.ok || !json.success) throw new Error(json.error?.message || "Gagal membuat template login");
      setTemplate(json.data);
    } catch (error) {
      onError(extractMessage(error));
    } finally {
      setSaving(false);
    }
  }

  function editUser(user: HotspotUser) {
    setEditingId(user.id || user.name);
    setUserForm({
      name: user.name,
      password: user.password || "",
      profile: user.profile || "default",
      limit_uptime: user.limit_uptime || "",
      comment: user.comment || "",
    });
    setMode("users");
  }

  useEffect(() => {
    void load("users");
    void load("profiles");
  }, [routerId]);

  useEffect(() => {
    void load(mode);
  }, [mode]);

  const refresh = (
    <button type="button" onClick={() => void load()} disabled={loading || mode === "template"} className="inline-flex items-center gap-2 rounded-md border border-slate-300 bg-white px-4 py-2 text-sm font-semibold text-slate-700 hover:bg-slate-50 disabled:cursor-wait disabled:opacity-60">
      <ArrowClockwise size={16} />
      {loading ? "Memuat..." : "Refresh"}
    </button>
  );

  return (
    <div className="grid gap-6">
      <Section title="Hotspot" description="User voucher, profile, active session, dan template login dibaca atau ditulis manual ke RouterOS." action={refresh}>
        <div className="flex flex-wrap gap-2">
          {tabs.map((tab) => (
            <button key={tab.id} type="button" onClick={() => setMode(tab.id)} className={`rounded-md px-3 py-2 text-sm font-semibold ${mode === tab.id ? "bg-blue-600 text-white" : "bg-slate-100 text-slate-600 hover:bg-slate-200"}`}>
              {tab.label}
            </button>
          ))}
        </div>
      </Section>

      {mode === "users" && (
        <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_22rem]">
          <Section title="Hotspot users" action={refresh}>
            {!loaded.users ? <EmptyState title="Membaca user hotspot" description="Data diambil live dari RouterOS." /> : users.length === 0 ? <EmptyState title="Belum ada user hotspot" description="Tambahkan user voucher untuk test RouterOS." /> : (
              <DataTable
                columns={["User", "Profile", "Uptime", "Traffic", "Status", "Aksi"]}
                rows={users.map((user) => [
                  <span key={`${user.id}-name`} className="font-semibold text-slate-900">{user.name}</span>,
                  user.profile || "-",
                  user.limit_uptime ? `${user.uptime || "0s"} / ${user.limit_uptime}` : user.uptime || "-",
                  `${formatBytes(user.bytes_in)} / ${formatBytes(user.bytes_out)}`,
                  <StatusBadge key={`${user.id}-status`} status={user.disabled ? "disabled" : user.managed ? "ISPBoss" : "aktif"} />,
                  <div key={`${user.id}-actions`} className="flex flex-wrap gap-2">
                    <button type="button" onClick={() => editUser(user)} className="rounded-md px-3 py-2 text-sm font-semibold text-blue-700 hover:bg-blue-50">Edit</button>
                    <button type="button" onClick={() => void toggleUser(user)} disabled={saving} className="rounded-md px-3 py-2 text-sm font-semibold text-amber-700 hover:bg-amber-50 disabled:cursor-wait disabled:opacity-60">{user.disabled ? "Enable" : "Disable"}</button>
                    <button type="button" onClick={() => void deleteUser(user)} disabled={saving} className="rounded-md px-3 py-2 text-sm font-semibold text-red-600 hover:bg-red-50 disabled:cursor-wait disabled:opacity-60">Hapus</button>
                  </div>,
                ])}
              />
            )}
          </Section>

          <Section title={editingId ? "Edit user" : "Tambah user"} description="Aksi ini langsung menulis `/ip hotspot user` ke RouterOS.">
            <form onSubmit={(event) => void submitUser(event)} className="grid gap-3">
              <input required disabled={Boolean(editingId)} value={userForm.name} onChange={(event) => setUserForm((current) => ({ ...current, name: event.target.value }))} placeholder="Username voucher" className="h-10 rounded-md border border-slate-300 px-3 text-sm outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-100 disabled:bg-slate-100" />
              <input required={!editingId} value={userForm.password} onChange={(event) => setUserForm((current) => ({ ...current, password: event.target.value }))} placeholder="Password" className="h-10 rounded-md border border-slate-300 px-3 text-sm outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-100" />
              <select value={userForm.profile} onChange={(event) => setUserForm((current) => ({ ...current, profile: event.target.value }))} className="h-10 rounded-md border border-slate-300 px-3 text-sm outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-100">
                {profileOptions.map((profile) => <option key={profile} value={profile}>{profile}</option>)}
              </select>
              <input value={userForm.limit_uptime} onChange={(event) => setUserForm((current) => ({ ...current, limit_uptime: event.target.value }))} placeholder="Limit uptime, mis. 1d" className="h-10 rounded-md border border-slate-300 px-3 text-sm outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-100" />
              <input value={userForm.comment} onChange={(event) => setUserForm((current) => ({ ...current, comment: event.target.value }))} placeholder="Comment" className="h-10 rounded-md border border-slate-300 px-3 text-sm outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-100" />
              <div className="flex flex-wrap gap-2">
                <button type="submit" disabled={saving} className="inline-flex items-center justify-center gap-2 rounded-md bg-blue-600 px-4 py-2 text-sm font-semibold text-white hover:bg-blue-700 disabled:cursor-wait disabled:opacity-60">
                  <Plus size={16} />
                  {saving ? "Menyimpan..." : editingId ? "Update" : "Tambah"}
                </button>
                {editingId && <button type="button" onClick={() => { setEditingId(""); setUserForm(emptyUserForm); }} className="rounded-md border border-slate-300 bg-white px-4 py-2 text-sm font-semibold text-slate-700 hover:bg-slate-50">Batal</button>}
              </div>
            </form>
          </Section>
        </div>
      )}

      {mode === "profiles" && (
        <Section title="Hotspot profiles" action={refresh}>
          {!loaded.profiles ? <EmptyState title="Membaca profile hotspot" description="Profile dibaca live dari RouterOS." /> : profiles.length === 0 ? <EmptyState title="Profile kosong" description="Router belum memiliki Hotspot user profile." /> : (
            <DataTable columns={["Nama", "Rate limit", "Shared", "Pool", "Proxy", "Comment"]} rows={profiles.map((profile) => [profile.name, profile.rate_limit || "-", profile.shared_users || "-", profile.address_pool || "-", profile.transparent_proxy ? "yes" : "no", profile.comment || "-"])} />
          )}
        </Section>
      )}

      {mode === "active" && (
        <Section title="Active hotspot sessions" action={refresh}>
          {!loaded.active ? <EmptyState title="Membaca active session" description="Session dibaca saat tab dibuka." /> : active.length === 0 ? <EmptyState title="Tidak ada session aktif" description="Belum ada user Hotspot yang login." /> : (
            <DataTable columns={["User", "IP", "MAC", "Uptime", "Traffic", "Server"]} rows={active.map((session) => [session.user, session.address || "-", session.mac_address || "-", session.uptime || "-", `${formatBytes(session.bytes_in)} / ${formatBytes(session.bytes_out)}`, session.server || "-"])} />
          )}
        </Section>
      )}

      {mode === "template" && (
        <div className="grid gap-6 xl:grid-cols-[22rem_minmax(0,1fr)]">
          <Section title="Generate login template" description="Template HTML siap dipakai untuk file `login.html` Hotspot MikroTik.">
            <form onSubmit={(event) => void generateTemplate(event)} className="grid gap-3">
              <input value={templateForm.brand_name} onChange={(event) => setTemplateForm((current) => ({ ...current, brand_name: event.target.value }))} placeholder="Brand name" className="h-10 rounded-md border border-slate-300 px-3 text-sm outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-100" />
              <input value={templateForm.primary_color} onChange={(event) => setTemplateForm((current) => ({ ...current, primary_color: event.target.value }))} placeholder="#2563eb" className="h-10 rounded-md border border-slate-300 px-3 text-sm outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-100" />
              <input value={templateForm.support_phone} onChange={(event) => setTemplateForm((current) => ({ ...current, support_phone: event.target.value }))} placeholder="Kontak admin" className="h-10 rounded-md border border-slate-300 px-3 text-sm outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-100" />
              <textarea value={templateForm.message} onChange={(event) => setTemplateForm((current) => ({ ...current, message: event.target.value }))} placeholder="Pesan login" className="min-h-24 rounded-md border border-slate-300 px-3 py-2 text-sm outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-100" />
              <button type="submit" disabled={saving} className="inline-flex items-center justify-center gap-2 rounded-md bg-blue-600 px-4 py-2 text-sm font-semibold text-white hover:bg-blue-700 disabled:cursor-wait disabled:opacity-60">
                <Code size={16} />
                {saving ? "Membuat..." : "Generate HTML"}
              </button>
            </form>
          </Section>

          <Section title={template?.file_name || "Preview HTML"}>
            {!template ? (
              <EmptyState title="Template belum dibuat" description="Isi branding lalu generate untuk melihat HTML." />
            ) : (
              <textarea readOnly value={template.html} className="min-h-[28rem] w-full resize-y rounded-md border border-slate-300 bg-slate-950 p-4 font-mono text-xs leading-5 text-slate-100 outline-none" />
            )}
          </Section>
        </div>
      )}
    </div>
  );
}
