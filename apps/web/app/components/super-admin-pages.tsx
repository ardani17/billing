"use client";

import { useEffect, useMemo, useState } from "react";
import {
  ArrowClockwise,
  ArrowSquareOut,
  CheckCircle,
  CreditCard,
  PauseCircle,
  ShieldCheck,
  UserSwitch,
  XCircle,
} from "@phosphor-icons/react";
import SuperAdminShell from "./super-admin-shell";
import { DataTable, EmptyState, PageHeader, Section, StatGrid, StatusBadge } from "./ui";

type ApiEnvelope<T> = { success: boolean; data?: T; error?: { message?: string } };
type AnyRecord = Record<string, any>;

type PlatformTenant = {
  id: string;
  name: string;
  owner_name: string;
  owner_email: string;
  domain: string;
  domain_status: string;
  plan: string;
  status: string;
  health: string;
  monthly_revenue: number;
  customer_count: number;
  open_invoice_count: number;
  router_count: number;
  olt_count: number;
  reseller_count: number;
  last_activity_at: string;
  created_at: string;
  modules?: string[];
};

type PlatformAudit = {
  id: string;
  tenant_id: string;
  tenant_name: string;
  actor_name: string;
  action: string;
  entity_type: string;
  entity_id: string;
  status: string;
  created_at: string;
};

type PlatformHealth = {
  service: string;
  region: string;
  latency_ms: number;
  uptime: string;
  status: string;
  last_checked?: string;
  last_error?: string;
};

type Subscription = {
  tenant_id: string;
  tenant: string;
  plan: string;
  amount: number;
  currency: string;
  status: string;
  due_date: string;
  trial_ends_at?: string;
  current_period_end: string;
  modules: string[];
  customer_count: number;
  open_invoice_count: number;
};

type UpgradeRequest = {
  id: string;
  tenant_id: string;
  tenant_name: string;
  requested_plan: string;
  requested_modules: string[];
  message: string;
  status: string;
  processed_reason: string;
  created_at: string;
};

type SupportTicket = {
  id: string;
  tenant_id: string;
  tenant_name: string;
  subject: string;
  description: string;
  priority: string;
  status: string;
  assignee_id: string;
  comment_count: number;
  created_at: string;
  updated_at: string;
};

type OverviewData = {
  stats: {
    tenant_total: number;
    tenant_active: number;
    tenant_trial: number;
    tenant_suspended: number;
    customer_total: number;
    monthly_recurring: number;
    upgrade_pending: number;
    support_open: number;
    subscription_overdue: number;
  };
  tenants: PlatformTenant[];
  audit: PlatformAudit[];
  health: PlatformHealth[];
};

const emptyOverview: OverviewData = {
  stats: {
    tenant_total: 0,
    tenant_active: 0,
    tenant_trial: 0,
    tenant_suspended: 0,
    customer_total: 0,
    monthly_recurring: 0,
    upgrade_pending: 0,
    support_open: 0,
    subscription_overdue: 0,
  },
  tenants: [],
  audit: [],
  health: [],
};

const fieldClass =
  "h-10 w-full min-w-0 rounded-md border border-slate-300 bg-white px-3 text-sm outline-none transition focus:border-blue-500 focus:ring-2 focus:ring-blue-100";
const textAreaClass =
  "w-full min-w-0 rounded-md border border-slate-300 bg-white px-3 py-2 text-sm outline-none transition focus:border-blue-500 focus:ring-2 focus:ring-blue-100";
const actionButton =
  "inline-flex h-10 items-center justify-center gap-2 rounded-md border border-slate-300 bg-white px-3 text-sm font-semibold text-slate-700 transition hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-60";

function formatRupiah(value: number) {
  return new Intl.NumberFormat("id-ID", { style: "currency", currency: "IDR", maximumFractionDigits: 0 }).format(value || 0);
}

function formatNumber(value: number) {
  return new Intl.NumberFormat("id-ID").format(value || 0);
}

function formatDate(value?: string) {
  if (!value) return "-";
  return new Intl.DateTimeFormat("id-ID", { day: "2-digit", month: "short", year: "numeric", hour: "2-digit", minute: "2-digit" }).format(new Date(value));
}

function moduleLabel(module: string) {
  if (module === "billing_core") return "Billing Core";
  if (module === "mikrotik") return "MikroTik";
  if (module === "fiber_network") return "OLT + Peta";
  return module;
}

async function platformApi<T>(path: string, init?: RequestInit) {
  const response = await fetch(`/api/super-admin/platform${path}`, {
    cache: "no-store",
    headers: { "Content-Type": "application/json", ...(init?.headers || {}) },
    ...init,
  });
  const contentType = response.headers.get("content-type") || "";
  if (!contentType.includes("application/json")) {
    return (await response.text()) as T;
  }
  const envelope = (await response.json()) as ApiEnvelope<T>;
  if (!response.ok || !envelope.success) throw new Error(envelope.error?.message || "Gagal mengambil data super admin");
  return envelope.data as T;
}

function useResource<T>(path: string, fallback: T) {
  const [data, setData] = useState<T>(fallback);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  async function load() {
    setLoading(true);
    setError("");
    try {
      setData(await platformApi<T>(path));
    } catch (err) {
      setError(err instanceof Error ? err.message : "Gagal mengambil data");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void load();
  }, [path]);

  return { data, loading, error, reload: load };
}

function useList<T>(path: string, fallback: T) {
  const resource = useResource<{ data: T }>(path, { data: fallback });
  return { ...resource, data: resource.data.data };
}

function ReloadButton({ onClick, loading }: { onClick: () => void; loading: boolean }) {
  return (
    <button type="button" onClick={onClick} disabled={loading} className={actionButton}>
      <ArrowClockwise size={18} weight="bold" />
      Refresh
    </button>
  );
}

function ErrorBanner({ message }: { message: string }) {
  if (!message) return null;
  return <div className="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">{message}</div>;
}

function InlineMessage({ error, success }: { error?: string; success?: string }) {
  if (error) return <p className="text-sm font-medium text-red-700">{error}</p>;
  if (success) return <p className="text-sm font-medium text-emerald-700">{success}</p>;
  return null;
}

function FormField({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <label className="grid gap-1.5 text-sm">
      <span className="font-medium text-slate-700">{label}</span>
      {children}
    </label>
  );
}

function ModuleBadges({ modules = [] }: { modules?: string[] }) {
  return (
    <div className="flex flex-wrap gap-1.5">
      {modules.length ? modules.map((module) => <StatusBadge key={module} status={moduleLabel(module)} />) : <StatusBadge status="billing core" />}
    </div>
  );
}

function TenantTable({ tenants, compact = false }: { tenants: PlatformTenant[]; compact?: boolean }) {
  if (!tenants.length) {
    return <EmptyState title="Belum ada tenant" description="Tenant akan muncul setelah registrasi atau dibuat oleh owner aplikasi." />;
  }
  return (
    <DataTable
      columns={compact ? ["Tenant", "Plan", "MRR", "Modules", "Health"] : ["Tenant", "Owner", "Domain", "Plan", "MRR", "Pelanggan", "Status", "Modules", "Aksi"]}
      rows={tenants.map((tenant) =>
        compact
          ? [
              <a key={tenant.id} href={`/super-admin/tenants/${tenant.id}`} className="font-semibold text-blue-700">{tenant.name}</a>,
              tenant.plan,
              formatRupiah(tenant.monthly_revenue),
              <ModuleBadges key={`${tenant.id}-modules`} modules={tenant.modules} />,
              <StatusBadge key={`${tenant.id}-health`} status={tenant.health} />,
            ]
          : [
              <a key={tenant.id} href={`/super-admin/tenants/${tenant.id}`} className="font-semibold text-blue-700">{tenant.name}</a>,
              tenant.owner_name || "-",
              tenant.domain || "-",
              tenant.plan,
              formatRupiah(tenant.monthly_revenue),
              formatNumber(tenant.customer_count),
              <StatusBadge key={`${tenant.id}-status`} status={tenant.status} />,
              <ModuleBadges key={`${tenant.id}-modules`} modules={tenant.modules} />,
              <a key={`${tenant.id}-detail`} href={`/super-admin/tenants/${tenant.id}`} className="text-sm font-semibold text-blue-700">Detail</a>,
            ],
      )}
    />
  );
}

function AuditTable({ audit }: { audit: PlatformAudit[] }) {
  if (!audit.length) return <EmptyState title="Belum ada audit global" description="Event audit lintas tenant akan muncul setelah ada aksi admin atau perubahan data." />;
  return (
    <DataTable
      columns={["Waktu", "Tenant", "Actor", "Action", "Target", "Detail"]}
      rows={audit.map((item) => [
        formatDate(item.created_at),
        item.tenant_name,
        item.actor_name || "-",
        item.action,
        `${item.entity_type} ${item.entity_id.slice(0, 8)}`,
        <a key={item.id} href={`/api/super-admin/platform/audit/${item.id}`} className="font-semibold text-blue-700">JSON</a>,
      ])}
    />
  );
}

function HealthTable({ health }: { health: PlatformHealth[] }) {
  if (!health.length) return <EmptyState title="Health belum tersedia" description="Status service akan tampil setelah endpoint platform health aktif." />;
  return (
    <DataTable
      columns={["Service", "Region", "Latency", "Uptime", "Last check", "Status", "Last error"]}
      rows={health.map((service) => [
        service.service,
        service.region,
        service.latency_ms ? `${service.latency_ms} ms` : "-",
        service.uptime,
        formatDate(service.last_checked),
        <StatusBadge key={service.service} status={service.status} />,
        service.last_error || "-",
      ])}
    />
  );
}

export function SuperAdminOverviewPage() {
  const { data, loading, error, reload } = useResource<OverviewData>("/overview", emptyOverview);
  const riskyTenants = data.tenants.filter((tenant) => tenant.health !== "normal" || tenant.status !== "active");

  return (
    <SuperAdminShell>
      <div className="space-y-6">
        <PageHeader eyebrow="Super Admin" title="Platform control center" description="Owner console ISPBoss untuk tenant, subscription, add-on, support, health, dan audit global." actions={<ReloadButton onClick={reload} loading={loading} />} />
        <ErrorBanner message={error} />
        <StatGrid
          stats={[
            { label: "Tenant aktif", value: formatNumber(data.stats.tenant_active), delta: `${formatNumber(data.stats.tenant_total)} total` },
            { label: "MRR platform", value: formatRupiah(data.stats.monthly_recurring), delta: "plan aktif" },
            { label: "Upgrade pending", value: formatNumber(data.stats.upgrade_pending), delta: `${formatNumber(data.stats.subscription_overdue)} overdue`, tone: data.stats.upgrade_pending ? "amber" : "green" },
            { label: "Support open", value: formatNumber(data.stats.support_open), delta: `${formatNumber(data.stats.tenant_trial)} trial`, tone: data.stats.support_open ? "amber" : "green" },
          ]}
        />
        <div className="grid gap-6 xl:grid-cols-[1.15fr_0.85fr]">
          <Section title="Tenant perlu perhatian" description="Tenant warning, trial, suspended, atau subscription bermasalah.">
            <TenantTable tenants={(riskyTenants.length ? riskyTenants : data.tenants).slice(0, 6)} compact />
          </Section>
          <Section title="Service health" description="Status service utama yang dipakai semua tenant.">
            <HealthTable health={data.health} />
          </Section>
        </div>
        <Section title="Audit global terbaru" description="Aksi lintas tenant dan event sistem.">
          <AuditTable audit={data.audit} />
        </Section>
      </div>
    </SuperAdminShell>
  );
}

function TenantCreateForm({ onDone }: { onDone: () => void }) {
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  async function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    setSaving(true);
    setError("");
    setSuccess("");
    try {
      await platformApi("/tenants", {
        method: "POST",
        body: JSON.stringify(Object.fromEntries(form.entries())),
      });
      event.currentTarget.reset();
      setSuccess("Tenant berhasil dibuat.");
      await onDone();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Gagal membuat tenant");
    } finally {
      setSaving(false);
    }
  }
  return (
    <form onSubmit={submit} className="grid gap-4 lg:grid-cols-4">
      <FormField label="Nama tenant"><input name="name" required className={fieldClass} /></FormField>
      <FormField label="Domain"><input name="domain" className={fieldClass} placeholder="isp.example.com" /></FormField>
      <FormField label="Plan">
        <select name="plan" className={fieldClass} defaultValue="starter">
          <option value="starter">Billing Core</option>
          <option value="growth">Billing + MikroTik</option>
          <option value="scale">Billing + MikroTik + Fiber</option>
        </select>
      </FormField>
      <FormField label="Status">
        <select name="status" className={fieldClass} defaultValue="trial">
          <option value="trial">Trial</option>
          <option value="active">Active</option>
        </select>
      </FormField>
      <FormField label="Owner name"><input name="owner_name" required className={fieldClass} /></FormField>
      <FormField label="Owner email"><input name="owner_email" required type="email" className={fieldClass} /></FormField>
      <div className="lg:col-span-2"><FormField label="Alasan"><input name="reason" required className={fieldClass} placeholder="Onboarding tenant baru" /></FormField></div>
      <div className="lg:col-span-4 flex flex-wrap items-center gap-3">
        <button disabled={saving} className="rounded-md bg-blue-600 px-4 py-2 text-sm font-semibold text-white disabled:opacity-60">Buat tenant</button>
        <InlineMessage error={error} success={success} />
      </div>
    </form>
  );
}

export function SuperAdminTenantsPage() {
  const { data, loading, error, reload } = useList<PlatformTenant[]>("/tenants", []);
  const [search, setSearch] = useState("");
  const [status, setStatus] = useState("");
  const filtered = data.filter((tenant) => {
    const haystack = `${tenant.name} ${tenant.owner_name} ${tenant.domain} ${tenant.plan}`.toLowerCase();
    return (!search || haystack.includes(search.toLowerCase())) && (!status || tenant.status === status);
  });

  return (
    <SuperAdminShell>
      <div className="space-y-6">
        <PageHeader eyebrow="Tenant Management" title="Semua tenant" description="Daftar client ISP, subscription, domain, modul aktif, dan status operasional." actions={<ReloadButton onClick={reload} loading={loading} />} />
        <ErrorBanner message={error} />
        <div className="grid gap-3 rounded-xl border border-slate-200 bg-white p-3 md:grid-cols-[1fr_180px]">
          <input value={search} onChange={(event) => setSearch(event.target.value)} className={fieldClass} placeholder="Cari tenant, owner, domain, plan..." />
          <select value={status} onChange={(event) => setStatus(event.target.value)} className={fieldClass}>
            <option value="">Semua status</option>
            <option value="trial">Trial</option>
            <option value="active">Active</option>
            <option value="suspended">Suspended</option>
            <option value="cancelled">Cancelled</option>
          </select>
        </div>
        <TenantTable tenants={filtered} />
        <Section title="Buat tenant manual" description="Untuk onboarding atau migrasi oleh tim ISPBoss.">
          <TenantCreateForm onDone={reload} />
        </Section>
      </div>
    </SuperAdminShell>
  );
}

function TenantActionPanel({ tenant, admins, reload }: { tenant: PlatformTenant; admins: AnyRecord[]; reload: () => void }) {
  const [saving, setSaving] = useState("");
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [modules, setModules] = useState<string[]>(tenant.modules || ["billing_core"]);

  async function post(path: string, payload: AnyRecord, label: string) {
    if (!window.confirm(`${label} tenant ini? Aksi akan masuk audit.`)) return;
    setSaving(label);
    setError("");
    setSuccess("");
    try {
      await platformApi(path, { method: "POST", body: JSON.stringify(payload) });
      setSuccess(`${label} berhasil.`);
      await reload();
    } catch (err) {
      setError(err instanceof Error ? err.message : `Gagal ${label}`);
    } finally {
      setSaving("");
    }
  }

  async function saveModules(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!window.confirm("Simpan perubahan entitlement modul tenant ini?")) return;
    const form = new FormData(event.currentTarget);
    setSaving("modules");
    setError("");
    setSuccess("");
    try {
      await platformApi(`/tenants/${tenant.id}/modules`, {
        method: "PUT",
        body: JSON.stringify({ modules, reason: String(form.get("reason") || "") }),
      });
      setSuccess("Entitlement modul tersimpan.");
      await reload();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Gagal menyimpan modul");
    } finally {
      setSaving("");
    }
  }

  async function impersonate(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    const userId = String(form.get("user_id") || "");
    const reason = String(form.get("reason") || "");
    if (!userId || !reason) {
      setError("Pilih admin tenant dan isi alasan impersonate.");
      return;
    }
    setSaving("impersonate");
    setError("");
    try {
      const response = await fetch("/api/super-admin/impersonate", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ tenant_id: tenant.id, user_id: userId, reason }),
      });
      const payload = await response.json().catch(() => null);
      if (!response.ok || payload?.success === false) {
        throw new Error(payload?.error?.message || "Gagal impersonate");
      }
      window.localStorage.setItem("ispboss_impersonating", JSON.stringify({ tenant_id: tenant.id, tenant_name: tenant.name, reason }));
      window.location.href = "/dashboard";
    } catch (err) {
      setError(err instanceof Error ? err.message : "Gagal impersonate");
    } finally {
      setSaving("");
    }
  }

  function toggleModule(module: string, checked: boolean) {
    setModules((current) => {
      const next = new Set(current);
      if (checked) next.add(module);
      else next.delete(module);
      next.add("billing_core");
      return Array.from(next);
    });
  }

  return (
    <div className="grid gap-6 xl:grid-cols-2">
      <Section title="Tenant lifecycle" description="Aksi akses tenant selalu membutuhkan alasan dan audit.">
        <div className="grid gap-3">
          <div className="grid gap-3 sm:grid-cols-2">
            <button className={actionButton} disabled={!!saving} onClick={() => void post(`/tenants/${tenant.id}/activate`, { reason: "Aktivasi oleh Super Admin" }, "Activate")}><CheckCircle size={18} /> Activate</button>
            <button className={actionButton} disabled={!!saving} onClick={() => void post(`/tenants/${tenant.id}/suspend`, { reason: "Suspend oleh Super Admin" }, "Suspend")}><PauseCircle size={18} /> Suspend</button>
            <button className={actionButton} disabled={!!saving} onClick={() => void post(`/tenants/${tenant.id}/resume`, { reason: "Resume oleh Super Admin" }, "Resume")}><ArrowClockwise size={18} /> Resume</button>
            <button className={actionButton} disabled={!!saving} onClick={() => void post(`/tenants/${tenant.id}/cancel`, { reason: "Cancel oleh Super Admin" }, "Cancel")}><XCircle size={18} /> Cancel</button>
          </div>
          <InlineMessage error={error} success={success} />
        </div>
      </Section>
      <Section title="Module entitlement" description="Billing Core selalu aktif. MikroTik dan Fiber Network adalah add-on berbayar.">
        <form onSubmit={saveModules} className="grid gap-3">
          {["billing_core", "mikrotik", "fiber_network"].map((module) => (
            <label key={module} className="flex items-center justify-between rounded-lg border border-slate-200 p-3 text-sm">
              <span className="font-semibold">{moduleLabel(module)}</span>
              <input type="checkbox" checked={modules.includes(module)} disabled={module === "billing_core"} onChange={(event) => toggleModule(module, event.target.checked)} />
            </label>
          ))}
          <FormField label="Alasan perubahan"><input name="reason" required className={fieldClass} placeholder="Upgrade/downgrade paket tenant" /></FormField>
          <button disabled={saving === "modules"} className="rounded-md bg-blue-600 px-4 py-2 text-sm font-semibold text-white disabled:opacity-60">Simpan entitlement</button>
        </form>
      </Section>
      <Section title="Impersonate tenant admin" description="Gunakan hanya untuk support. Mode impersonate akan masuk audit.">
        <form onSubmit={impersonate} className="grid gap-3">
          <FormField label="Tenant admin">
            <select name="user_id" required className={fieldClass} defaultValue="">
              <option value="">Pilih admin</option>
              {admins.map((admin) => <option key={admin.id} value={admin.id}>{admin.name} - {admin.email}</option>)}
            </select>
          </FormField>
          <FormField label="Alasan impersonate"><input name="reason" required className={fieldClass} placeholder="Contoh: bantu cek invoice pelanggan" /></FormField>
          <button disabled={saving === "impersonate"} className="rounded-md bg-zinc-950 px-4 py-2 text-sm font-semibold text-white disabled:opacity-60">Mulai impersonate</button>
        </form>
      </Section>
      <Section title="Buka tenant app" description="Gunakan domain tenant jika sudah siap.">
        <a className={actionButton} href={tenant.domain ? `https://${tenant.domain}` : "/dashboard"}>
          <ArrowSquareOut size={18} />
          {tenant.domain || "Tenant view lokal"}
        </a>
      </Section>
    </div>
  );
}

export function SuperAdminTenantDetailPage({ tenantId }: { tenantId?: string }) {
  const path = tenantId ? `/tenants/${tenantId}` : "/tenants";
  const { data, loading, error, reload } = useResource<{ tenant?: PlatformTenant; audit?: PlatformAudit[]; admins?: AnyRecord[]; subscription?: Subscription; tickets?: SupportTicket[] }>(path, {});
  const tenant = data.tenant;

  return (
    <SuperAdminShell>
      <div className="space-y-6">
        <PageHeader eyebrow="Tenant Detail" title={tenant?.name || "Tenant"} description={tenant ? `${tenant.owner_name || "Owner belum diisi"} - ${tenant.domain || "domain belum diisi"}` : "Memuat data tenant live."} actions={<ReloadButton onClick={reload} loading={loading} />} />
        <ErrorBanner message={error} />
        {tenant ? (
          <>
            <StatGrid
              stats={[
                { label: "Subscription", value: tenant.plan, delta: tenant.status },
                { label: "MRR", value: formatRupiah(tenant.monthly_revenue), delta: "plan" },
                { label: "Pelanggan", value: formatNumber(tenant.customer_count), delta: `${formatNumber(tenant.open_invoice_count)} invoice terbuka` },
                { label: "Health", value: tenant.health, delta: formatDate(tenant.last_activity_at), tone: tenant.health === "blocked" ? "red" : tenant.health === "warning" ? "amber" : "green" },
              ]}
            />
            <TenantActionPanel tenant={tenant} admins={data.admins || []} reload={reload} />
            <div className="grid gap-6 xl:grid-cols-[1fr_380px]">
              <Section title="Tenant profile">
                <DataTable columns={["Field", "Value"]} rows={[["Tenant ID", tenant.id], ["Owner", tenant.owner_name || "-"], ["Email owner", tenant.owner_email || "-"], ["Domain", tenant.domain || "-"], ["Domain status", <StatusBadge key="domain-status" status={tenant.domain_status || "unverified"} />], ["Modules", <ModuleBadges key="modules" modules={tenant.modules} />], ["Router", formatNumber(tenant.router_count)], ["OLT", formatNumber(tenant.olt_count)], ["Reseller", formatNumber(tenant.reseller_count)]]} />
              </Section>
              <Section title="Subscription">
                <DataTable columns={["Field", "Value"]} rows={[["Plan", data.subscription?.plan || tenant.plan], ["Status", <StatusBadge key="substatus" status={data.subscription?.status || tenant.status} />], ["Amount", formatRupiah(data.subscription?.amount || tenant.monthly_revenue)], ["Renewal", formatDate(data.subscription?.current_period_end || data.subscription?.due_date)]]} />
              </Section>
            </div>
            <Section title="Support tenant terbaru">
              {data.tickets?.length ? <SupportTable tickets={data.tickets} /> : <EmptyState title="Belum ada tiket" description="Tiket support tenant ini akan tampil di sini." />}
            </Section>
            <Section title="Audit tenant terbaru">
              <AuditTable audit={data.audit || []} />
            </Section>
          </>
        ) : (
          <EmptyState title="Tenant tidak ditemukan" description="Data tenant belum tersedia atau endpoint mengembalikan error." />
        )}
      </div>
    </SuperAdminShell>
  );
}

export function SuperAdminSubscriptionsPage() {
  const { data, loading, error, reload } = useList<Subscription[]>("/subscriptions", []);
  const [search, setSearch] = useState("");
  const [status, setStatus] = useState("");
  const filtered = data.filter((subscription) => {
    const haystack = `${subscription.tenant} ${subscription.plan} ${subscription.modules.join(" ")}`.toLowerCase();
    return (!search || haystack.includes(search.toLowerCase())) && (!status || subscription.status === status);
  });
  return (
    <SuperAdminShell>
      <div className="space-y-6">
        <PageHeader eyebrow="Platform Billing" title="Subscription tenant" description="Plan, add-on, renewal, trial, dan status subscription tenant." actions={<ReloadButton onClick={reload} loading={loading} />} />
        <ErrorBanner message={error} />
        <div className="grid gap-3 rounded-xl border border-slate-200 bg-white p-3 md:grid-cols-[1fr_180px]">
          <input value={search} onChange={(event) => setSearch(event.target.value)} className={fieldClass} placeholder="Cari tenant, plan, modul..." />
          <select value={status} onChange={(event) => setStatus(event.target.value)} className={fieldClass}>
            <option value="">Semua status</option>
            <option value="trial">Trial</option>
            <option value="active">Active</option>
            <option value="overdue">Overdue</option>
            <option value="suspended">Suspended</option>
            <option value="cancelled">Cancelled</option>
          </select>
        </div>
        {filtered.length ? (
          <DataTable
            columns={["Tenant", "Plan", "Modules", "Amount", "Renewal", "Status", "Aksi"]}
            rows={filtered.map((invoice) => [
              invoice.tenant,
              invoice.plan,
              <ModuleBadges key={`${invoice.tenant_id}-modules`} modules={invoice.modules} />,
              formatRupiah(invoice.amount),
              formatDate(invoice.current_period_end || invoice.due_date),
              <StatusBadge key={invoice.tenant_id} status={invoice.status} />,
              <a key={`${invoice.tenant_id}-detail`} href={`/super-admin/tenants/${invoice.tenant_id}`} className="font-semibold text-blue-700">Kelola</a>,
            ])}
          />
        ) : (
          <EmptyState title="Belum ada subscription" description="Subscription platform akan mengikuti tenant yang sudah terdaftar." />
        )}
      </div>
    </SuperAdminShell>
  );
}

export function SuperAdminUpgradeRequestsPage() {
  const { data, loading, error, reload } = useList<UpgradeRequest[]>("/upgrade-requests", []);
  const [actionError, setActionError] = useState("");
  async function decide(id: string, action: "approve" | "reject" | "cancel") {
    const reason = window.prompt("Alasan keputusan upgrade request:");
    if (!reason) return;
    setActionError("");
    try {
      await platformApi(`/upgrade-requests/${id}/${action}`, { method: "POST", body: JSON.stringify({ reason }) });
      await reload();
    } catch (err) {
      setActionError(err instanceof Error ? err.message : "Gagal memproses upgrade request");
    }
  }
  return (
    <SuperAdminShell>
      <div className="space-y-6">
        <PageHeader eyebrow="Upgrade Requests" title="Permintaan upgrade tenant" description="Tenant Admin hanya bisa meminta upgrade. Super Admin yang memutuskan add-on aktif." actions={<ReloadButton onClick={reload} loading={loading} />} />
        <ErrorBanner message={error || actionError} />
        {data.length ? (
          <DataTable
            columns={["Tenant", "Plan", "Modules", "Message", "Status", "Waktu", "Aksi"]}
            rows={data.map((request) => [
              request.tenant_name,
              request.requested_plan || "-",
              <ModuleBadges key={request.id} modules={request.requested_modules} />,
              request.message || "-",
              <StatusBadge key={`${request.id}-status`} status={request.status} />,
              formatDate(request.created_at),
              request.status === "pending" ? (
                <span key={`${request.id}-actions`} className="flex flex-wrap gap-2">
                  <button className="font-semibold text-emerald-700" onClick={() => void decide(request.id, "approve")}>Approve</button>
                  <button className="font-semibold text-red-700" onClick={() => void decide(request.id, "reject")}>Reject</button>
                </span>
              ) : request.processed_reason || "-",
            ])}
          />
        ) : (
          <EmptyState title="Belum ada upgrade request" description="Request upgrade dari tenant akan tampil di sini." />
        )}
      </div>
    </SuperAdminShell>
  );
}

function SupportTable({ tickets }: { tickets: SupportTicket[] }) {
  return (
    <DataTable
      columns={["Tenant", "Subject", "Priority", "Status", "Komentar", "Update"]}
      rows={tickets.map((ticket) => [
        ticket.tenant_name || "-",
        ticket.subject,
        <StatusBadge key={`${ticket.id}-priority`} status={ticket.priority} />,
        <StatusBadge key={`${ticket.id}-status`} status={ticket.status} />,
        formatNumber(ticket.comment_count),
        formatDate(ticket.updated_at),
      ])}
    />
  );
}

export function SuperAdminSupportPage() {
  const { data, loading, error, reload } = useList<SupportTicket[]>("/support", []);
  const [formError, setFormError] = useState("");
  const [success, setSuccess] = useState("");
  const [search, setSearch] = useState("");
  const [status, setStatus] = useState("");
  const [priority, setPriority] = useState("");
  const filtered = data.filter((ticket) => {
    const haystack = `${ticket.tenant_name} ${ticket.subject} ${ticket.description}`.toLowerCase();
    return (
      (!search || haystack.includes(search.toLowerCase())) &&
      (!status || ticket.status === status) &&
      (!priority || ticket.priority === priority)
    );
  });
  async function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    setFormError("");
    setSuccess("");
    try {
      await platformApi("/support", { method: "POST", body: JSON.stringify(Object.fromEntries(form.entries())) });
      event.currentTarget.reset();
      setSuccess("Tiket support berhasil dibuat.");
      await reload();
    } catch (err) {
      setFormError(err instanceof Error ? err.message : "Gagal membuat tiket");
    }
  }
  return (
    <SuperAdminShell>
      <div className="space-y-6">
        <PageHeader eyebrow="Support Console" title="Tiket support lintas tenant" description="Queue support internal ISPBoss lintas tenant." actions={<ReloadButton onClick={reload} loading={loading} />} />
        <ErrorBanner message={error} />
        <div className="grid gap-3 rounded-xl border border-slate-200 bg-white p-3 md:grid-cols-[1fr_170px_170px]">
          <input value={search} onChange={(event) => setSearch(event.target.value)} className={fieldClass} placeholder="Cari tenant, subject, deskripsi..." />
          <select value={status} onChange={(event) => setStatus(event.target.value)} className={fieldClass}>
            <option value="">Semua status</option>
            <option value="open">Open</option>
            <option value="in_progress">In Progress</option>
            <option value="waiting_tenant">Waiting Tenant</option>
            <option value="resolved">Resolved</option>
            <option value="closed">Closed</option>
          </select>
          <select value={priority} onChange={(event) => setPriority(event.target.value)} className={fieldClass}>
            <option value="">Semua priority</option>
            <option value="low">Low</option>
            <option value="normal">Normal</option>
            <option value="high">High</option>
            <option value="urgent">Urgent</option>
          </select>
        </div>
        {filtered.length ? <SupportTable tickets={filtered} /> : <EmptyState title="Belum ada tiket support" description="Buat tiket internal atau tunggu request dari tenant." />}
        <Section title="Buat tiket internal" description="Untuk mencatat pekerjaan support lintas tenant.">
          <form onSubmit={submit} className="grid gap-4 lg:grid-cols-4">
            <FormField label="Tenant ID"><input name="tenant_id" className={fieldClass} placeholder="opsional" /></FormField>
            <FormField label="Priority">
              <select name="priority" className={fieldClass} defaultValue="normal">
                <option value="low">Low</option>
                <option value="normal">Normal</option>
                <option value="high">High</option>
                <option value="urgent">Urgent</option>
              </select>
            </FormField>
            <div className="lg:col-span-2"><FormField label="Subject"><input name="subject" required className={fieldClass} /></FormField></div>
            <div className="lg:col-span-4"><FormField label="Description"><textarea name="description" rows={3} className={textAreaClass} /></FormField></div>
            <div className="lg:col-span-4 flex flex-wrap items-center gap-3">
              <button className="rounded-md bg-blue-600 px-4 py-2 text-sm font-semibold text-white">Buat tiket</button>
              <InlineMessage error={formError} success={success} />
            </div>
          </form>
        </Section>
      </div>
    </SuperAdminShell>
  );
}

export function SuperAdminHealthPage() {
  const { data, loading, error, reload } = useList<PlatformHealth[]>("/health", []);
  return (
    <SuperAdminShell>
      <div className="space-y-6">
        <PageHeader eyebrow="Service Health" title="Kesehatan platform" description="Status service, latency, queue, dan provider utama." actions={<ReloadButton onClick={reload} loading={loading} />} />
        <ErrorBanner message={error} />
        <HealthTable health={data} />
      </div>
    </SuperAdminShell>
  );
}

export function SuperAdminAuditPage() {
  const [query, setQuery] = useState("");
  const { data, loading, error, reload } = useList<PlatformAudit[]>(`/audit${query}`, []);
  function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    const params = new URLSearchParams();
    for (const [key, value] of form.entries()) {
      if (String(value)) params.set(key, String(value));
    }
    setQuery(params.toString() ? `?${params.toString()}` : "");
  }
  return (
    <SuperAdminShell>
      <div className="space-y-6">
        <PageHeader eyebrow="Audit Global" title="Aktivitas Super Admin" description="Filter event lintas tenant untuk investigasi." actions={<ReloadButton onClick={reload} loading={loading} />} />
        <ErrorBanner message={error} />
        <form onSubmit={submit} className="grid gap-3 rounded-xl border border-slate-200 bg-white p-3 md:grid-cols-7">
          <input name="tenant_id" className={fieldClass} placeholder="Tenant ID" />
          <input name="actor" className={fieldClass} placeholder="Actor" />
          <input name="action" className={fieldClass} placeholder="Action" />
          <input name="entity" className={fieldClass} placeholder="Entity" />
          <input name="from" type="date" className={fieldClass} />
          <input name="to" type="date" className={fieldClass} />
          <button className="rounded-md bg-zinc-950 px-4 py-2 text-sm font-semibold text-white">Filter</button>
        </form>
        <div className="flex justify-end">
          <a className={actionButton} href={`/api/super-admin/platform/audit/export${query}`}>Export CSV</a>
        </div>
        <AuditTable audit={data} />
      </div>
    </SuperAdminShell>
  );
}

export function SuperAdminSettingsPage() {
  const { data, loading, error, reload } = useResource<{ settings: AnyRecord }>("/settings", { settings: {} });
  const [formError, setFormError] = useState("");
  const [success, setSuccess] = useState("");
  const settings = data.settings || {};
  const security = settings.security_policy || {};
  const limits = settings.tenant_limits || {};
  const support = settings.support_contact || {};
  const planDefaults = settings.plan_defaults || {};
  const plans = Array.isArray(planDefaults.plans) ? planDefaults.plans : [];
  const planAmount = (code: string, fallback: number) => Number(plans.find((plan: AnyRecord) => plan.code === code)?.amount || fallback);
  async function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    setFormError("");
    setSuccess("");
    try {
      await platformApi("/settings", {
        method: "PUT",
        body: JSON.stringify({
          reason: String(form.get("reason") || ""),
          plan_defaults: {
            trial_days: Number(form.get("trial_days") || planDefaults.trial_days || 14),
            plans: [
              { code: "billing_core", name: "Billing Core", amount: Number(form.get("billing_core_amount") || 299000), modules: ["billing_core"] },
              { code: "growth", name: "Billing Core + MikroTik", amount: Number(form.get("growth_amount") || 799000), modules: ["billing_core", "mikrotik"] },
              { code: "scale", name: "Billing Core + MikroTik + Fiber", amount: Number(form.get("scale_amount") || 1499000), modules: ["billing_core", "mikrotik", "fiber_network"] },
            ],
          },
          security_policy: {
            impersonate_reason_required: true,
            audit_retention_months: Number(form.get("audit_retention_months") || 24),
            super_admin_mfa_required: form.get("super_admin_mfa_required") === "on",
          },
          tenant_limits: {
            customer_limit: Number(form.get("customer_limit") || 1000),
            router_limit: Number(form.get("router_limit") || 5),
            olt_limit: Number(form.get("olt_limit") || 2),
            reseller_limit: Number(form.get("reseller_limit") || 50),
          },
          support_contact: {
            email: String(form.get("email") || ""),
            whatsapp: String(form.get("whatsapp") || ""),
          },
        }),
      });
      setSuccess("Settings platform tersimpan.");
      await reload();
    } catch (err) {
      setFormError(err instanceof Error ? err.message : "Gagal menyimpan settings");
    }
  }
  return (
    <SuperAdminShell>
      <div className="space-y-6">
        <PageHeader eyebrow="Platform Settings" title="Konfigurasi platform" description="Konfigurasi global untuk plan, limit, support, dan security policy ISPBoss." actions={<ReloadButton onClick={reload} loading={loading} />} />
        <ErrorBanner message={error} />
        <form onSubmit={submit} className="grid gap-6 xl:grid-cols-2">
          <Section title="Plan defaults">
            <div className="grid gap-4 sm:grid-cols-2">
              <FormField label="Trial hari"><input name="trial_days" type="number" min="0" className={fieldClass} defaultValue={planDefaults.trial_days || 14} /></FormField>
              <FormField label="Billing Core"><input name="billing_core_amount" type="number" min="0" className={fieldClass} defaultValue={planAmount("billing_core", 299000)} /></FormField>
              <FormField label="Billing + MikroTik"><input name="growth_amount" type="number" min="0" className={fieldClass} defaultValue={planAmount("growth", 799000)} /></FormField>
              <FormField label="Billing + MikroTik + Fiber"><input name="scale_amount" type="number" min="0" className={fieldClass} defaultValue={planAmount("scale", 1499000)} /></FormField>
            </div>
          </Section>
          <Section title="Security policy">
            <div className="grid gap-4">
              <FormField label="Audit retention bulan"><input name="audit_retention_months" type="number" min="1" className={fieldClass} defaultValue={security.audit_retention_months || 24} /></FormField>
              <label className="flex items-center gap-3 text-sm font-medium text-slate-700"><input name="super_admin_mfa_required" type="checkbox" defaultChecked={security.super_admin_mfa_required !== false} /> Super Admin MFA required</label>
            </div>
          </Section>
          <Section title="Tenant limits">
            <div className="grid gap-4 sm:grid-cols-2">
              <FormField label="Customer limit"><input name="customer_limit" type="number" className={fieldClass} defaultValue={limits.customer_limit || 1000} /></FormField>
              <FormField label="Router limit"><input name="router_limit" type="number" className={fieldClass} defaultValue={limits.router_limit || 5} /></FormField>
              <FormField label="OLT limit"><input name="olt_limit" type="number" className={fieldClass} defaultValue={limits.olt_limit || 2} /></FormField>
              <FormField label="Reseller limit"><input name="reseller_limit" type="number" className={fieldClass} defaultValue={limits.reseller_limit || 50} /></FormField>
            </div>
          </Section>
          <Section title="Support contact">
            <div className="grid gap-4">
              <FormField label="Email"><input name="email" type="email" className={fieldClass} defaultValue={support.email || "support@ispboss.id"} /></FormField>
              <FormField label="WhatsApp"><input name="whatsapp" className={fieldClass} defaultValue={support.whatsapp || ""} /></FormField>
            </div>
          </Section>
          <Section title="Simpan perubahan">
            <div className="grid gap-4">
              <FormField label="Alasan"><input name="reason" required className={fieldClass} placeholder="Update policy platform" /></FormField>
              <button className="rounded-md bg-blue-600 px-4 py-2 text-sm font-semibold text-white">Simpan settings</button>
              <InlineMessage error={formError} success={success} />
            </div>
          </Section>
        </form>
      </div>
    </SuperAdminShell>
  );
}
