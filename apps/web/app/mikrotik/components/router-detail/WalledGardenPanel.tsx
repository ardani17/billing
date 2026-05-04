"use client";

import { useEffect, useMemo, useState, type FormEvent } from "react";
import { ArrowClockwise, ShieldCheck, Trash } from "@phosphor-icons/react";
import { DataTable, EmptyState, FormField, Section, StatGrid, StatusBadge, TextInput } from "../../../components/ui";
import { extractMessage } from "../../lib/format";
import type { WalledGardenConfig, WalledGardenStatus } from "../../lib/types";

type WalledGardenForm = {
  method: string;
  walledGardenIP: string;
  dnsServerIP: string;
  isolatedAddressList: string;
  allowedAddressList: string;
  allowedDestinations: string;
};

const defaultForm: WalledGardenForm = {
  method: "dns_redirect",
  walledGardenIP: "10.255.255.1",
  dnsServerIP: "10.255.255.1",
  isolatedAddressList: "ISPBoss:walled-garden-isolated",
  allowedAddressList: "ISPBoss:walled-garden-allowed",
  allowedDestinations: "payment.ispboss.local",
};

export function WalledGardenPanel({ routerId, onError }: { routerId: string; onError: (message: string) => void }) {
  const [status, setStatus] = useState<WalledGardenStatus | null>(null);
  const [form, setForm] = useState<WalledGardenForm>(defaultForm);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);

  async function load() {
    setLoading(true);
    onError("");
    try {
      const response = await fetch(`/api/network/mikrotik/routers/${routerId}/walled-garden`, { cache: "no-store" });
      const json = await response.json();
      if (!response.ok || !json.success) throw new Error(json.error?.message || "Gagal mengambil walled garden");
      const data = json.data as WalledGardenStatus;
      setStatus(data);
      setForm(configToForm(data.config));
    } catch (error) {
      onError(extractMessage(error));
    } finally {
      setLoading(false);
    }
  }

  async function apply(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSaving(true);
    onError("");
    try {
      const response = await fetch(`/api/network/mikrotik/routers/${routerId}/walled-garden/apply`, {
        method: "POST",
        body: JSON.stringify(formToPayload(form)),
      });
      const json = await response.json();
      if (!response.ok || !json.success) throw new Error(json.error?.message || "Gagal menerapkan walled garden");
      const data = json.data as WalledGardenStatus;
      setStatus(data);
      setForm(configToForm(data.config));
    } catch (error) {
      onError(extractMessage(error));
    } finally {
      setSaving(false);
    }
  }

  async function remove() {
    const ok = window.confirm("Hapus rule global walled garden ISPBoss dari router ini?");
    if (!ok) return;
    setSaving(true);
    onError("");
    try {
      const response = await fetch(`/api/network/mikrotik/routers/${routerId}/walled-garden/remove`, { method: "POST" });
      const json = await response.json();
      if (!response.ok || !json.success) throw new Error(json.error?.message || "Gagal menghapus walled garden");
      setStatus(json.data as WalledGardenStatus);
    } catch (error) {
      onError(extractMessage(error));
    } finally {
      setSaving(false);
    }
  }

  useEffect(() => {
    void load();
  }, [routerId]);

  const rules = status?.rules || [];
  const stats = useMemo(
    () => [
      { label: "Status", value: status?.applied ? "applied" : "not applied" },
      { label: "Isolated list", value: String(status?.isolated_count ?? 0) },
      { label: "Whitelist", value: String(status?.allowed_count ?? 0) },
      { label: "Rule managed", value: String(rules.filter((rule) => rule.kind !== "address_list").length) },
    ],
    [rules, status],
  );

  return (
    <div className="space-y-6">
      <StatGrid stats={stats} />

      <Section
        title="Konfigurasi walled garden"
        description="Rule dibuat manual saat tombol Apply ditekan dan dicari ulang dari comment ISPBoss agar tidak dobel."
        action={
          <button type="button" onClick={() => void load()} disabled={loading || saving} className="inline-flex items-center gap-2 rounded-md border border-slate-300 bg-white px-4 py-2 text-sm font-semibold text-slate-700 hover:bg-slate-50 disabled:cursor-wait disabled:opacity-60">
            <ArrowClockwise size={16} />
            {loading ? "Memuat..." : "Refresh"}
          </button>
        }
      >
        <form onSubmit={(event) => void apply(event)} className="grid gap-5">
          <div className="grid gap-4 lg:grid-cols-3">
            <FormField label="Metode isolir">
              <select value={form.method} onChange={(event) => setFormField(setForm, "method", event.target.value)} className="h-10 w-full rounded-md border border-slate-300 bg-white px-3 text-sm outline-none transition focus:border-blue-500 focus:ring-2 focus:ring-blue-100">
                <option value="dns_redirect">DNS redirect</option>
                <option value="http_redirect">HTTP redirect</option>
                <option value="block_all_whitelist">Block all + whitelist</option>
              </select>
            </FormField>
            <FormField label="Walled garden IP">
              <TextInput value={form.walledGardenIP} onChange={(event) => setFormField(setForm, "walledGardenIP", event.target.value)} placeholder="10.255.255.1" />
            </FormField>
            <FormField label="DNS redirect IP">
              <TextInput value={form.dnsServerIP} onChange={(event) => setFormField(setForm, "dnsServerIP", event.target.value)} placeholder="10.255.255.1" />
            </FormField>
          </div>

          <div className="grid gap-4 lg:grid-cols-2">
            <FormField label="Address-list isolir">
              <TextInput value={form.isolatedAddressList} onChange={(event) => setFormField(setForm, "isolatedAddressList", event.target.value)} />
            </FormField>
            <FormField label="Address-list whitelist">
              <TextInput value={form.allowedAddressList} onChange={(event) => setFormField(setForm, "allowedAddressList", event.target.value)} />
            </FormField>
          </div>

          <FormField label="Whitelist tujuan" helper="Satu domain atau IP per baris. Dipakai untuk payment portal dan halaman bantuan pelanggan isolir.">
            <textarea value={form.allowedDestinations} onChange={(event) => setFormField(setForm, "allowedDestinations", event.target.value)} rows={4} className="w-full min-w-0 rounded-md border border-slate-300 px-3 py-2 text-sm outline-none transition focus:border-blue-500 focus:ring-2 focus:ring-blue-100" />
          </FormField>

          <div className="flex flex-wrap gap-2">
            <button type="submit" disabled={saving} className="inline-flex items-center gap-2 rounded-md bg-blue-600 px-4 py-2 text-sm font-semibold text-white hover:bg-blue-700 disabled:cursor-wait disabled:opacity-60">
              <ShieldCheck size={16} />
              {saving ? "Memproses..." : "Apply ke Router"}
            </button>
            <button type="button" onClick={() => void remove()} disabled={saving} className="inline-flex items-center gap-2 rounded-md border border-red-200 bg-white px-4 py-2 text-sm font-semibold text-red-600 hover:bg-red-50 disabled:cursor-wait disabled:opacity-60">
              <Trash size={16} />
              Hapus Rule
            </button>
          </div>
        </form>
      </Section>

      <Section title="Rule terpasang" description="Status live dari NAT, filter, dan address-list yang dikelola walled garden.">
        {rules.length === 0 ? (
          <EmptyState title="Belum ada rule walled garden" description="Apply konfigurasi untuk membuat rule DNS, HTTP, atau block whitelist di router." />
        ) : (
          <DataTable
            columns={["Jenis", "Chain/List", "Action/Address", "Status", "Comment"]}
            rows={rules.map((rule) => [
              rule.kind,
              rule.chain || rule.list || "-",
              rule.action || rule.address || "-",
              <StatusBadge key={`${rule.id}-status`} status={rule.disabled ? "disabled" : "aktif"} />,
              rule.comment || "-",
            ])}
          />
        )}
      </Section>
    </div>
  );
}

function configToForm(config?: WalledGardenConfig): WalledGardenForm {
  if (!config) return defaultForm;
  return {
    method: config.method || defaultForm.method,
    walledGardenIP: config.walled_garden_ip || defaultForm.walledGardenIP,
    dnsServerIP: config.dns_server_ip || defaultForm.dnsServerIP,
    isolatedAddressList: config.isolated_address_list || defaultForm.isolatedAddressList,
    allowedAddressList: config.allowed_address_list || defaultForm.allowedAddressList,
    allowedDestinations: (config.allowed_destinations || []).join("\n") || defaultForm.allowedDestinations,
  };
}

function formToPayload(form: WalledGardenForm) {
  return {
    method: form.method,
    walled_garden_ip: form.walledGardenIP.trim(),
    dns_server_ip: form.dnsServerIP.trim(),
    isolated_address_list: form.isolatedAddressList.trim(),
    allowed_address_list: form.allowedAddressList.trim(),
    allowed_destinations: form.allowedDestinations.split(/\r?\n/).map((item) => item.trim()).filter(Boolean),
  };
}

function setFormField(
  setForm: (updater: (current: WalledGardenForm) => WalledGardenForm) => void,
  field: keyof WalledGardenForm,
  value: string,
) {
  setForm((current) => ({ ...current, [field]: value }));
}
