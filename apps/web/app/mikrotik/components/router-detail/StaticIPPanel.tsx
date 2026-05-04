"use client";

import { useEffect, useState, type FormEvent } from "react";
import { ArrowClockwise, Plus } from "@phosphor-icons/react";
import { DataTable, EmptyState, Section, StatusBadge } from "../../../components/ui";
import { extractMessage } from "../../lib/format";
import type { StaticIPAssignment } from "../../lib/types";

const emptyForm = { ip_address: "", queue_name: "", rate_limit: "", comment: "" };

export function StaticIPPanel({ routerId, onError }: { routerId: string; onError: (message: string) => void }) {
  const [items, setItems] = useState<StaticIPAssignment[]>([]);
  const [form, setForm] = useState(emptyForm);
  const [editingId, setEditingId] = useState("");
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [loaded, setLoaded] = useState(false);

  async function load() {
    setLoading(true);
    onError("");
    try {
      const response = await fetch(`/api/network/mikrotik/routers/${routerId}/static-ip/assignments?page_size=50`, { cache: "no-store" });
      const json = await response.json();
      if (!response.ok || !json.success) throw new Error(json.error?.message || "Gagal mengambil static IP");
      setItems(json.data?.items || []);
      setLoaded(true);
    } catch (error) {
      onError(extractMessage(error));
    } finally {
      setLoading(false);
    }
  }

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSaving(true);
    onError("");
    try {
      const response = await fetch(
        editingId ? `/api/network/mikrotik/routers/${routerId}/static-ip/assignments/${editingId}` : `/api/network/mikrotik/routers/${routerId}/static-ip/assignments`,
        { method: editingId ? "PUT" : "POST", body: JSON.stringify(form) },
      );
      const json = await response.json();
      if (!response.ok || !json.success) throw new Error(json.error?.message || "Gagal menyimpan static IP");
      setForm(emptyForm);
      setEditingId("");
      await load();
    } catch (error) {
      onError(extractMessage(error));
    } finally {
      setSaving(false);
    }
  }

  async function action(item: StaticIPAssignment, actionName: "isolate" | "unisolate") {
    setSaving(true);
    onError("");
    try {
      const response = await fetch(`/api/network/mikrotik/routers/${routerId}/static-ip/assignments/${item.id}/${actionName}`, { method: "POST" });
      const json = await response.json();
      if (!response.ok || !json.success) throw new Error(json.error?.message || "Aksi static IP gagal");
      await load();
    } catch (error) {
      onError(extractMessage(error));
    } finally {
      setSaving(false);
    }
  }

  async function remove(item: StaticIPAssignment) {
    const confirmIP = window.prompt(`Ketik IP ${item.ip_address} untuk hapus assignment ini.`);
    if (!confirmIP) return;
    setSaving(true);
    onError("");
    try {
      const response = await fetch(`/api/network/mikrotik/routers/${routerId}/static-ip/assignments/${item.id}`, {
        method: "DELETE",
        body: JSON.stringify({ confirm_ip: confirmIP }),
      });
      const json = await response.json();
      if (!response.ok || !json.success) throw new Error(json.error?.message || "Gagal menghapus static IP");
      await load();
    } catch (error) {
      onError(extractMessage(error));
    } finally {
      setSaving(false);
    }
  }

  function edit(item: StaticIPAssignment) {
    setEditingId(item.id);
    setForm({
      ip_address: item.ip_address,
      queue_name: item.queue_name || "",
      rate_limit: item.rate_limit || "",
      comment: item.comment || "",
    });
  }

  useEffect(() => {
    void load();
  }, [routerId]);

  const refresh = (
    <button type="button" onClick={() => void load()} disabled={loading} className="inline-flex items-center gap-2 rounded-md border border-slate-300 bg-white px-4 py-2 text-sm font-semibold text-slate-700 hover:bg-slate-50 disabled:cursor-wait disabled:opacity-60">
      <ArrowClockwise size={16} />
      {loading ? "Memuat..." : "Refresh"}
    </button>
  );

  return (
    <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_22rem]">
      <Section title="Static IP assignments" description="Mengelola address-list ISPBoss dan simple queue opsional." action={refresh}>
        {!loaded ? (
          <EmptyState title="Membaca static IP" description="Data assignment diambil dari database tenant." />
        ) : items.length === 0 ? (
          <EmptyState title="Belum ada static IP" description="Tambahkan pelanggan static IP untuk provisioning ke MikroTik." />
        ) : (
          <DataTable
            columns={["IP", "Address list", "Queue", "Rate", "Status", "Aksi"]}
            rows={items.map((item) => [
              item.ip_address,
              item.address_list,
              item.queue_name || "-",
              item.rate_limit || "-",
              <StatusBadge key={`${item.id}-status`} status={item.status} />,
              <div key={`${item.id}-actions`} className="flex flex-wrap gap-2">
                <button type="button" onClick={() => edit(item)} className="rounded-md px-3 py-2 text-sm font-semibold text-blue-700 hover:bg-blue-50">Edit</button>
                <button type="button" onClick={() => void action(item, item.status === "isolated" ? "unisolate" : "isolate")} disabled={saving} className="rounded-md px-3 py-2 text-sm font-semibold text-amber-700 hover:bg-amber-50 disabled:cursor-wait disabled:opacity-60">{item.status === "isolated" ? "Buka" : "Isolir"}</button>
                <button type="button" onClick={() => void remove(item)} disabled={saving} className="rounded-md px-3 py-2 text-sm font-semibold text-red-600 hover:bg-red-50 disabled:cursor-wait disabled:opacity-60">Hapus</button>
              </div>,
            ])}
          />
        )}
      </Section>

      <Section title={editingId ? "Edit static IP" : "Tambah static IP"} description="Rate limit opsional, contoh: 10M/10M.">
        <form onSubmit={(event) => void submit(event)} className="grid gap-3">
          <input required value={form.ip_address} onChange={(event) => setForm((current) => ({ ...current, ip_address: event.target.value }))} placeholder="IP address" className="h-10 rounded-md border border-slate-300 px-3 text-sm outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-100" />
          <input value={form.queue_name} onChange={(event) => setForm((current) => ({ ...current, queue_name: event.target.value }))} placeholder="Queue name" className="h-10 rounded-md border border-slate-300 px-3 text-sm outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-100" />
          <input value={form.rate_limit} onChange={(event) => setForm((current) => ({ ...current, rate_limit: event.target.value }))} placeholder="Rate limit" className="h-10 rounded-md border border-slate-300 px-3 text-sm outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-100" />
          <input value={form.comment} onChange={(event) => setForm((current) => ({ ...current, comment: event.target.value }))} placeholder="Comment" className="h-10 rounded-md border border-slate-300 px-3 text-sm outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-100" />
          <div className="flex flex-wrap gap-2">
            <button type="submit" disabled={saving} className="inline-flex items-center justify-center gap-2 rounded-md bg-blue-600 px-4 py-2 text-sm font-semibold text-white hover:bg-blue-700 disabled:cursor-wait disabled:opacity-60">
              <Plus size={16} />
              {saving ? "Menyimpan..." : editingId ? "Update" : "Tambah"}
            </button>
            {editingId && <button type="button" onClick={() => { setEditingId(""); setForm(emptyForm); }} className="rounded-md border border-slate-300 bg-white px-4 py-2 text-sm font-semibold text-slate-700 hover:bg-slate-50">Batal</button>}
          </div>
        </form>
      </Section>
    </div>
  );
}
