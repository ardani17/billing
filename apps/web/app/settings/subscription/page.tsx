"use client";

import { useEffect, useState } from "react";
import AppShell from "../../components/app-shell";
import { DataTable, EmptyState, FormField, PageHeader, Section, StatusBadge } from "../../components/ui";

type AnyRecord = Record<string, any>;

const inputClass =
  "h-10 w-full min-w-0 rounded-md border border-slate-300 bg-white px-3 text-sm outline-none transition focus:border-blue-500 focus:ring-2 focus:ring-blue-100";
const textAreaClass =
  "w-full min-w-0 rounded-md border border-slate-300 bg-white px-3 py-2 text-sm outline-none transition focus:border-blue-500 focus:ring-2 focus:ring-blue-100";

function unwrap(body: any) {
  if (body && typeof body === "object" && "success" in body && "data" in body) return body.data;
  return body;
}

async function apiGet(path: string) {
  const res = await fetch(`/api/billing/${path}`, { cache: "no-store" });
  const body = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(body?.error?.message || "Gagal mengambil data");
  return unwrap(body);
}

async function apiPost(path: string, payload: AnyRecord) {
  const res = await fetch(`/api/billing/${path}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
  const body = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(body?.error?.message || "Gagal menyimpan request");
  return unwrap(body);
}

function moduleLabel(module: string) {
  if (module === "billing_core") return "Billing Core";
  if (module === "mikrotik") return "MikroTik";
  if (module === "fiber_network") return "OLT + Peta Jaringan";
  return module;
}

export default function Page() {
  const [modules, setModules] = useState<AnyRecord>({ billing_core: true, mikrotik: false, fiber_network: false });
  const [requests, setRequests] = useState<AnyRecord[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");

  async function load() {
    setLoading(true);
    setError("");
    try {
      const [moduleData, requestData] = await Promise.all([
        apiGet("tenant/modules"),
        apiGet("tenant/upgrade-requests"),
      ]);
      setModules(moduleData || {});
      setRequests(Array.isArray(requestData?.data) ? requestData.data : []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Gagal mengambil subscription");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void load();
  }, []);

  async function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    const requested_modules = ["billing_core"];
    if (form.get("mikrotik") === "on") requested_modules.push("mikrotik");
    if (form.get("fiber_network") === "on") requested_modules.push("fiber_network");
    setError("");
    setSuccess("");
    try {
      await apiPost("tenant/upgrade-requests", {
        requested_plan: String(form.get("requested_plan") || ""),
        requested_modules,
        message: String(form.get("message") || ""),
      });
      event.currentTarget.reset();
      setSuccess("Request upgrade terkirim ke Super Admin.");
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Gagal membuat request upgrade");
    }
  }

  return (
    <AppShell>
      <div className="space-y-6">
        <PageHeader eyebrow="Subscription" title="Paket SaaS tenant" description="Lihat add-on aktif dan ajukan upgrade. Aktivasi add-on berbayar diproses oleh Super Admin." />
        {error && <p className="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">{error}</p>}
        {success && <p className="rounded-lg border border-emerald-200 bg-emerald-50 px-4 py-3 text-sm text-emerald-700">{success}</p>}
        <Section title="Modul aktif">
          <DataTable
            columns={["Module", "Status"]}
            rows={["billing_core", "mikrotik", "fiber_network"].map((module) => [
              moduleLabel(module),
              <StatusBadge key={module} status={modules[module] ? "active" : "inactive"} />,
            ])}
          />
        </Section>
        <Section title="Ajukan upgrade" description="Request ini masuk ke console Super Admin untuk approval.">
          <form onSubmit={submit} className="grid gap-4 lg:grid-cols-3">
            <FormField label="Plan tujuan">
              <select name="requested_plan" className={inputClass} defaultValue="growth">
                <option value="starter">Billing Core</option>
                <option value="growth">Billing + MikroTik</option>
                <option value="scale">Billing + MikroTik + Fiber</option>
              </select>
            </FormField>
            <label className="flex items-center gap-3 rounded-md border border-slate-200 px-3 text-sm font-medium text-slate-700">
              <input type="checkbox" name="mikrotik" defaultChecked={Boolean(modules.mikrotik)} /> MikroTik
            </label>
            <label className="flex items-center gap-3 rounded-md border border-slate-200 px-3 text-sm font-medium text-slate-700">
              <input type="checkbox" name="fiber_network" defaultChecked={Boolean(modules.fiber_network)} /> OLT + Peta
            </label>
            <div className="lg:col-span-3">
              <FormField label="Pesan">
                <textarea name="message" rows={3} className={textAreaClass} placeholder="Jelaskan kebutuhan upgrade..." />
              </FormField>
            </div>
            <div className="lg:col-span-3">
              <button className="rounded-md bg-blue-600 px-4 py-2 text-sm font-semibold text-white">Kirim request upgrade</button>
            </div>
          </form>
        </Section>
        <Section title="Riwayat request">
          {requests.length ? (
            <DataTable
              columns={["Waktu", "Plan", "Modules", "Status", "Catatan"]}
              rows={requests.map((request) => [
                new Date(request.created_at).toLocaleString("id-ID"),
                request.requested_plan || "-",
                (request.requested_modules || []).map(moduleLabel).join(", "),
                <StatusBadge key={request.id} status={request.status} />,
                request.processed_reason || request.message || "-",
              ])}
            />
          ) : (
            <EmptyState title={loading ? "Memuat request" : "Belum ada request"} description="Request upgrade tenant akan tampil di sini." />
          )}
        </Section>
      </div>
    </AppShell>
  );
}
