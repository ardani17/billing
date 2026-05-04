"use client";

import { useEffect, useState, type FormEvent } from "react";
import { ArrowClockwise, Plus } from "@phosphor-icons/react";
import { DataTable, EmptyState, Section, StatusBadge } from "../../../components/ui";
import { extractMessage } from "../../lib/format";
import type { DHCPBinding, DHCPLease, DHCPNetwork, DHCPServer } from "../../lib/types";

type DHCPMode = "servers" | "leases" | "bindings" | "networks";

const tabs: { id: DHCPMode; label: string }[] = [
  { id: "servers", label: "Servers" },
  { id: "leases", label: "Leases" },
  { id: "bindings", label: "Static Bindings" },
  { id: "networks", label: "Networks" },
];

const emptyForm = {
  server: "all",
  mac_address: "",
  ip_address: "",
  host_name: "",
  comment: "",
  disabled: false,
};

export function DHCPPanel({ routerId, onError }: { routerId: string; onError: (message: string) => void }) {
  const [mode, setMode] = useState<DHCPMode>("servers");
  const [servers, setServers] = useState<DHCPServer[]>([]);
  const [leases, setLeases] = useState<DHCPLease[]>([]);
  const [bindings, setBindings] = useState<DHCPBinding[]>([]);
  const [networks, setNetworks] = useState<DHCPNetwork[]>([]);
  const [form, setForm] = useState(emptyForm);
  const [editingId, setEditingId] = useState("");
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [loaded, setLoaded] = useState<Record<DHCPMode, boolean>>({
    servers: false,
    leases: false,
    bindings: false,
    networks: false,
  });

  async function load(target = mode) {
    setLoading(true);
    onError("");
    try {
      const endpoint = `/api/network/mikrotik/routers/${routerId}/dhcp/${target === "bindings" ? "bindings?page_size=50" : target}`;
      const response = await fetch(endpoint, { cache: "no-store" });
      const json = await response.json();
      if (!response.ok || !json.success) throw new Error(json.error?.message || "Gagal mengambil data DHCP");
      if (target === "servers") setServers(json.data || []);
      if (target === "leases") setLeases(json.data || []);
      if (target === "bindings") setBindings(json.data?.items || json.data || []);
      if (target === "networks") setNetworks(json.data || []);
      setLoaded((current) => ({ ...current, [target]: true }));
    } catch (error) {
      onError(extractMessage(error));
    } finally {
      setLoading(false);
    }
  }

  async function submitBinding(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSaving(true);
    onError("");
    try {
      const response = await fetch(
        editingId
          ? `/api/network/mikrotik/routers/${routerId}/dhcp/bindings/${editingId}`
          : `/api/network/mikrotik/routers/${routerId}/dhcp/bindings`,
        {
          method: editingId ? "PUT" : "POST",
          body: JSON.stringify(form),
        },
      );
      const json = await response.json();
      if (!response.ok || !json.success) throw new Error(json.error?.message || "Gagal menyimpan DHCP binding");
      setForm(emptyForm);
      setEditingId("");
      await load("bindings");
      await load("leases");
      setMode("bindings");
    } catch (error) {
      onError(extractMessage(error));
    } finally {
      setSaving(false);
    }
  }

  async function deleteBinding(binding: DHCPBinding) {
    const confirmMac = window.prompt(`Ketik MAC ${binding.mac_address} untuk hapus static binding ini.`);
    if (!confirmMac) return;
    setSaving(true);
    onError("");
    try {
      const response = await fetch(`/api/network/mikrotik/routers/${routerId}/dhcp/bindings/${binding.id}`, {
        method: "DELETE",
        body: JSON.stringify({ confirm_mac: confirmMac }),
      });
      const json = await response.json();
      if (!response.ok || !json.success) throw new Error(json.error?.message || "Gagal menghapus DHCP binding");
      await load("bindings");
      await load("leases");
    } catch (error) {
      onError(extractMessage(error));
    } finally {
      setSaving(false);
    }
  }

  function editBinding(binding: DHCPBinding) {
    setEditingId(binding.id);
    setForm({
      server: binding.server || "all",
      mac_address: binding.mac_address,
      ip_address: binding.ip_address,
      host_name: binding.host_name || "",
      comment: binding.comment || "",
      disabled: binding.disabled,
    });
    setMode("bindings");
  }

  useEffect(() => {
    void load(mode);
  }, [routerId, mode]);

  const refresh = (
    <button type="button" onClick={() => void load()} disabled={loading} className="inline-flex items-center gap-2 rounded-md border border-slate-300 bg-white px-4 py-2 text-sm font-semibold text-slate-700 hover:bg-slate-50 disabled:cursor-wait disabled:opacity-60">
      <ArrowClockwise size={16} />
      {loading ? "Memuat..." : "Refresh"}
    </button>
  );

  return (
    <div className="grid gap-6">
      <Section title="DHCP" description="Server, lease, static binding, dan network dibaca live/on-demand dari RouterOS." action={refresh}>
        <div className="flex flex-wrap gap-2">
          {tabs.map((tab) => (
            <button key={tab.id} type="button" onClick={() => setMode(tab.id)} className={`rounded-md px-3 py-2 text-sm font-semibold ${mode === tab.id ? "bg-blue-600 text-white" : "bg-slate-100 text-slate-600 hover:bg-slate-200"}`}>
              {tab.label}
            </button>
          ))}
        </div>
      </Section>

      {mode === "servers" && (
        <Section title="DHCP servers" action={refresh}>
          {!loaded.servers ? <EmptyState title="Membaca DHCP server" description="Data diambil saat tab dibuka." /> : servers.length === 0 ? <EmptyState title="DHCP server kosong" description="Router belum memiliki DHCP server." /> : (
            <DataTable columns={["Nama", "Interface", "Pool", "Lease time", "Status"]} rows={servers.map((item) => [item.name, item.interface || "-", item.address_pool || "-", item.lease_time || "-", <StatusBadge key={item.id} status={item.disabled ? "disabled" : "aktif"} />])} />
          )}
        </Section>
      )}

      {mode === "leases" && (
        <Section title="DHCP leases" action={refresh}>
          {!loaded.leases ? <EmptyState title="Membaca DHCP lease" description="Data diambil saat tab dibuka." /> : leases.length === 0 ? <EmptyState title="Lease kosong" description="Belum ada DHCP lease di router." /> : (
            <DataTable columns={["IP", "MAC", "Host", "Server", "Status", "Managed"]} rows={leases.map((item) => [item.address || "-", item.mac_address, item.host_name || "-", item.server || "-", <StatusBadge key={`${item.id}-status`} status={item.disabled ? "disabled" : item.status || "unknown"} />, item.managed ? "ISPBoss" : item.dynamic ? "Dynamic" : "Static"])} />
          )}
        </Section>
      )}

      {mode === "bindings" && (
        <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_22rem]">
          <Section title="Static bindings terkelola" action={refresh}>
            {!loaded.bindings ? <EmptyState title="Membaca static binding" description="Binding terkelola dibaca dari database dan dicocokkan ke RouterOS." /> : bindings.length === 0 ? <EmptyState title="Belum ada binding" description="Tambahkan binding static untuk pelanggan DHCP." /> : (
              <DataTable
                columns={["IP", "MAC", "Host", "Server", "Sync", "Aksi"]}
                rows={bindings.map((binding) => [
                  binding.ip_address,
                  binding.mac_address,
                  binding.host_name || "-",
                  binding.server || "all",
                  <StatusBadge key={`${binding.id}-sync`} status={binding.sync_status} />,
                  <div key={`${binding.id}-actions`} className="flex flex-wrap gap-2">
                    <button type="button" onClick={() => editBinding(binding)} className="rounded-md px-3 py-2 text-sm font-semibold text-blue-700 hover:bg-blue-50">Edit</button>
                    <button type="button" onClick={() => void deleteBinding(binding)} disabled={saving} className="rounded-md px-3 py-2 text-sm font-semibold text-red-600 hover:bg-red-50 disabled:cursor-wait disabled:opacity-60">Hapus</button>
                  </div>,
                ])}
              />
            )}
          </Section>

          <Section title={editingId ? "Edit binding" : "Tambah binding"} description="Aksi ini langsung menulis static lease ke RouterOS.">
            <form onSubmit={(event) => void submitBinding(event)} className="grid gap-3">
              <input value={form.server} onChange={(event) => setForm((current) => ({ ...current, server: event.target.value }))} placeholder="Server, mis. all" className="h-10 rounded-md border border-slate-300 px-3 text-sm outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-100" />
              <input required value={form.mac_address} onChange={(event) => setForm((current) => ({ ...current, mac_address: event.target.value }))} placeholder="MAC address" className="h-10 rounded-md border border-slate-300 px-3 text-sm outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-100" />
              <input required value={form.ip_address} onChange={(event) => setForm((current) => ({ ...current, ip_address: event.target.value }))} placeholder="IP address" className="h-10 rounded-md border border-slate-300 px-3 text-sm outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-100" />
              <input value={form.host_name} onChange={(event) => setForm((current) => ({ ...current, host_name: event.target.value }))} placeholder="Host name" className="h-10 rounded-md border border-slate-300 px-3 text-sm outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-100" />
              <input value={form.comment} onChange={(event) => setForm((current) => ({ ...current, comment: event.target.value }))} placeholder="Comment" className="h-10 rounded-md border border-slate-300 px-3 text-sm outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-100" />
              <label className="flex items-center gap-2 text-sm font-medium text-slate-700">
                <input type="checkbox" checked={form.disabled} onChange={(event) => setForm((current) => ({ ...current, disabled: event.target.checked }))} />
                Disabled
              </label>
              <div className="flex flex-wrap gap-2">
                <button type="submit" disabled={saving} className="inline-flex items-center justify-center gap-2 rounded-md bg-blue-600 px-4 py-2 text-sm font-semibold text-white hover:bg-blue-700 disabled:cursor-wait disabled:opacity-60">
                  <Plus size={16} />
                  {saving ? "Menyimpan..." : editingId ? "Update" : "Tambah"}
                </button>
                {editingId && (
                  <button type="button" onClick={() => { setEditingId(""); setForm(emptyForm); }} className="rounded-md border border-slate-300 bg-white px-4 py-2 text-sm font-semibold text-slate-700 hover:bg-slate-50">
                    Batal
                  </button>
                )}
              </div>
            </form>
          </Section>
        </div>
      )}

      {mode === "networks" && (
        <Section title="DHCP networks" action={refresh}>
          {!loaded.networks ? <EmptyState title="Membaca DHCP network" description="Network bersifat read-only di dashboard." /> : networks.length === 0 ? <EmptyState title="Network kosong" description="Router belum memiliki DHCP network." /> : (
            <DataTable columns={["Address", "Gateway", "DNS", "Domain", "Comment"]} rows={networks.map((item) => [item.address, item.gateway || "-", item.dns_server?.join(", ") || "-", item.domain || "-", item.comment || "-"])} />
          )}
        </Section>
      )}
    </div>
  );
}
