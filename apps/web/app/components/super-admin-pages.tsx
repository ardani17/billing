"use client";

import { useEffect, useMemo, useState } from "react";
import {
  ArrowClockwise,
  ArrowSquareOut,
  CreditCard,
  ShieldCheck,
  UserSwitch,
} from "@phosphor-icons/react";
import SuperAdminShell from "./super-admin-shell";
import { DataTable, EmptyState, PageHeader, Section, StatGrid, StatusBadge } from "./ui";

type ApiEnvelope<T> = {
  success: boolean;
  data?: T;
  error?: {
    message?: string;
  };
};

type PlatformTenant = {
  id: string;
  name: string;
  owner_name: string;
  owner_email: string;
  domain: string;
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
};

type Subscription = {
  tenant_id: string;
  tenant: string;
  plan: string;
  amount: number;
  status: string;
  due_date: string;
};

type OverviewData = {
  stats: {
    tenant_total: number;
    tenant_active: number;
    tenant_trial: number;
    tenant_suspended: number;
    customer_total: number;
    monthly_recurring: number;
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
  },
  tenants: [],
  audit: [],
  health: [],
};

function formatRupiah(value: number) {
  return new Intl.NumberFormat("id-ID", {
    style: "currency",
    currency: "IDR",
    maximumFractionDigits: 0,
  }).format(value || 0);
}

function formatNumber(value: number) {
  return new Intl.NumberFormat("id-ID").format(value || 0);
}

function formatDate(value?: string) {
  if (!value) return "-";
  return new Intl.DateTimeFormat("id-ID", {
    day: "2-digit",
    month: "short",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  }).format(new Date(value));
}

async function platformApi<T>(path: string) {
  const response = await fetch(`/api/super-admin/platform${path}`, { cache: "no-store" });
  const envelope = (await response.json()) as ApiEnvelope<T>;

  if (!response.ok || !envelope.success) {
    throw new Error(envelope.error?.message || "Gagal mengambil data super admin");
  }

  return envelope.data as T;
}

function useOverview() {
  const [data, setData] = useState<OverviewData>(emptyOverview);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  async function load() {
    setLoading(true);
    setError("");
    try {
      setData(await platformApi<OverviewData>("/overview"));
    } catch (err) {
      setError(err instanceof Error ? err.message : "Gagal mengambil data super admin");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void load();
  }, []);

  return { data, loading, error, reload: load };
}

function useList<T>(path: string, fallback: T) {
  const [data, setData] = useState<T>(fallback);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  async function load() {
    setLoading(true);
    setError("");
    try {
      const result = await platformApi<{ data: T }>(path);
      setData(result.data);
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

function ReloadButton({ onClick, loading }: { onClick: () => void; loading: boolean }) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={loading}
      className="inline-flex h-10 items-center gap-2 rounded-md border border-slate-300 bg-white px-4 text-sm font-semibold text-slate-700 transition hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-60"
    >
      <ArrowClockwise size={18} weight="bold" />
      Refresh
    </button>
  );
}

function ErrorBanner({ message }: { message: string }) {
  if (!message) return null;
  return <div className="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">{message}</div>;
}

function TenantTable({ tenants, compact = false }: { tenants: PlatformTenant[]; compact?: boolean }) {
  if (!tenants.length) {
    return <EmptyState title="Belum ada tenant" description="Tenant akan muncul setelah registrasi atau dibuat oleh owner aplikasi." />;
  }

  return (
    <DataTable
      columns={compact ? ["Tenant", "Plan", "MRR", "Health", "Aksi"] : ["Tenant", "Owner", "Domain", "Plan", "MRR", "Pelanggan", "Status", "Health", "Aksi"]}
      rows={tenants.map((tenant) =>
        compact
          ? [
              tenant.name,
              tenant.plan,
              formatRupiah(tenant.monthly_revenue),
              <StatusBadge key={`${tenant.id}-health`} status={tenant.health} />,
              <a key={tenant.id} href={`/super-admin/tenants/${tenant.id}`} className="text-sm font-semibold text-blue-700">Detail</a>,
            ]
          : [
              <a key={tenant.id} href={`/super-admin/tenants/${tenant.id}`} className="font-semibold text-blue-700">{tenant.name}</a>,
              tenant.owner_name || "-",
              tenant.domain || "-",
              tenant.plan,
              formatRupiah(tenant.monthly_revenue),
              formatNumber(tenant.customer_count),
              <StatusBadge key={`${tenant.id}-status`} status={tenant.status} />,
              <StatusBadge key={`${tenant.id}-health`} status={tenant.health} />,
              <a key={`${tenant.id}-detail`} href={`/super-admin/tenants/${tenant.id}`} className="text-sm font-semibold text-blue-700">Detail</a>,
            ],
      )}
    />
  );
}

function AuditTable({ audit }: { audit: PlatformAudit[] }) {
  if (!audit.length) {
    return <EmptyState title="Belum ada audit global" description="Event audit lintas tenant akan muncul setelah ada aksi admin atau perubahan data." />;
  }

  return (
    <DataTable
      columns={["Waktu", "Tenant", "Actor", "Action", "Target", "Status"]}
      rows={audit.map((item) => [
        formatDate(item.created_at),
        item.tenant_name,
        item.actor_name,
        item.action,
        `${item.entity_type} ${item.entity_id.slice(0, 8)}`,
        <StatusBadge key={item.id} status={item.status} />,
      ])}
    />
  );
}

function HealthTable({ health }: { health: PlatformHealth[] }) {
  if (!health.length) {
    return <EmptyState title="Health belum tersedia" description="Status service akan tampil setelah endpoint platform health aktif." />;
  }

  return (
    <DataTable
      columns={["Service", "Region", "Latency", "Uptime", "Status"]}
      rows={health.map((service) => [
        service.service,
        service.region,
        service.latency_ms ? `${service.latency_ms} ms` : "-",
        service.uptime,
        <StatusBadge key={service.service} status={service.status} />,
      ])}
    />
  );
}

export function SuperAdminOverviewPage() {
  const { data, loading, error, reload } = useOverview();

  return (
    <SuperAdminShell>
      <div className="space-y-6">
        <PageHeader
          eyebrow="Super Admin"
          title="Platform control center"
          description="Pantau tenant, subscription, health service, support, dan audit global ISPBoss dari data live."
          actions={<ReloadButton onClick={reload} loading={loading} />}
        />
        <ErrorBanner message={error} />
        <StatGrid
          stats={[
            { label: "Tenant aktif", value: formatNumber(data.stats.tenant_active), delta: `${formatNumber(data.stats.tenant_total)} total` },
            { label: "MRR platform", value: formatRupiah(data.stats.monthly_recurring), delta: "plan aktif" },
            { label: "Pelanggan tenant", value: formatNumber(data.stats.customer_total), delta: "lintas tenant" },
            { label: "Tenant suspend", value: formatNumber(data.stats.tenant_suspended), delta: `${formatNumber(data.stats.tenant_trial)} trial`, tone: data.stats.tenant_suspended ? "amber" : "green" },
          ]}
        />
        <div className="grid gap-6 xl:grid-cols-[1.15fr_0.85fr]">
          <Section title="Tenant terbaru" description="Ringkasan tenant lintas platform.">
            <TenantTable tenants={data.tenants.slice(0, 5)} compact />
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

export function SuperAdminTenantsPage() {
  const { data, loading, error, reload } = useList<PlatformTenant[]>("/tenants", []);

  return (
    <SuperAdminShell>
      <div className="space-y-6">
        <PageHeader
          eyebrow="Tenant Management"
          title="Semua tenant"
          description="Daftar client ISP, subscription, domain, modul aktif, dan status operasional."
          actions={<ReloadButton onClick={reload} loading={loading} />}
        />
        <ErrorBanner message={error} />
        <TenantTable tenants={data} />
      </div>
    </SuperAdminShell>
  );
}

export function SuperAdminTenantDetailPage({ tenantId }: { tenantId?: string }) {
  const path = tenantId ? `/tenants/${tenantId}` : "/tenants";
  const { data, loading, error, reload } = useList<{ tenant?: PlatformTenant; audit?: PlatformAudit[] }>(path, {});
  const tenant = data.tenant;

  return (
    <SuperAdminShell>
      <div className="space-y-6">
        <PageHeader
          eyebrow="Tenant Detail"
          title={tenant?.name || "Tenant"}
          description={tenant ? `${tenant.owner_name || "Owner belum diisi"} - ${tenant.domain || "domain belum diisi"}` : "Memuat data tenant live."}
          actions={<ReloadButton onClick={reload} loading={loading} />}
        />
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
            <div className="grid gap-6 xl:grid-cols-[1fr_360px]">
              <Section title="Kontrol support" description="Aksi lintas tenant harus meninggalkan audit trail.">
                <div className="grid gap-3 sm:grid-cols-2">
                  {[
                    { title: "Impersonate Tenant Admin", icon: UserSwitch, description: "Endpoint tersedia, eksekusi akan diaktifkan setelah selector admin tenant siap." },
                    { title: "Buka tenant app", icon: ArrowSquareOut, description: "Gunakan domain tenant jika sudah terverifikasi." },
                    { title: "Cek subscription", icon: CreditCard, description: "Lihat status plan dan billing platform tenant." },
                    { title: "Review security", icon: ShieldCheck, description: "Audit domain, user, dan akses tenant." },
                  ].map(({ title, icon: TypedIcon, description }) => (
                    <button key={title} type="button" className="rounded-lg border border-zinc-200 p-4 text-left hover:bg-zinc-50">
                      <TypedIcon size={22} />
                      <span className="mt-3 block font-semibold text-zinc-950">{title}</span>
                      <span className="mt-1 block text-sm leading-6 text-zinc-500">{description}</span>
                    </button>
                  ))}
                </div>
              </Section>
              <Section title="Tenant profile">
                <DataTable
                  columns={["Field", "Value"]}
                  rows={[
                    ["Tenant ID", tenant.id],
                    ["Owner", tenant.owner_name || "-"],
                    ["Email owner", tenant.owner_email || "-"],
                    ["Domain", tenant.domain || "-"],
                    ["Router", formatNumber(tenant.router_count)],
                    ["OLT", formatNumber(tenant.olt_count)],
                    ["Reseller", formatNumber(tenant.reseller_count)],
                  ]}
                />
              </Section>
            </div>
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

  return (
    <SuperAdminShell>
      <div className="space-y-6">
        <PageHeader eyebrow="Platform Billing" title="Subscription tenant" description="Plan dan nilai MRR tenant dari data live." actions={<ReloadButton onClick={reload} loading={loading} />} />
        <ErrorBanner message={error} />
        {data.length ? (
          <DataTable
            columns={["Tenant", "Plan", "Amount", "Renewal", "Status"]}
            rows={data.map((invoice) => [
              invoice.tenant,
              invoice.plan,
              formatRupiah(invoice.amount),
              formatDate(invoice.due_date),
              <StatusBadge key={invoice.tenant_id} status={invoice.status} />,
            ])}
          />
        ) : (
          <EmptyState title="Belum ada subscription" description="Subscription platform akan mengikuti tenant yang sudah terdaftar." />
        )}
      </div>
    </SuperAdminShell>
  );
}

export function SuperAdminSupportPage() {
  const { data, loading, error, reload } = useList<unknown[]>("/support", []);

  return (
    <SuperAdminShell>
      <div className="space-y-6">
        <PageHeader eyebrow="Support Console" title="Tiket support lintas tenant" description="Endpoint support sudah siap untuk data live; tabel tiket belum ada di backend." actions={<ReloadButton onClick={reload} loading={loading} />} />
        <ErrorBanner message={error} />
        {data.length ? (
          <DataTable columns={["Tenant", "Subject", "Priority", "Status"]} rows={[]} />
        ) : (
          <EmptyState title="Belum ada tiket support" description="Nanti modul support bisa memakai tabel khusus ticketing agar tidak memakai data contoh." />
        )}
      </div>
    </SuperAdminShell>
  );
}

export function SuperAdminHealthPage() {
  const { data, loading, error, reload } = useList<PlatformHealth[]>("/health", []);

  return (
    <SuperAdminShell>
      <div className="space-y-6">
        <PageHeader eyebrow="Service Health" title="Kesehatan platform" description="Status service dan database yang dipakai semua tenant." actions={<ReloadButton onClick={reload} loading={loading} />} />
        <ErrorBanner message={error} />
        <HealthTable health={data} />
      </div>
    </SuperAdminShell>
  );
}

export function SuperAdminAuditPage() {
  const { data, loading, error, reload } = useList<PlatformAudit[]>("/audit", []);

  return (
    <SuperAdminShell>
      <div className="space-y-6">
        <PageHeader eyebrow="Audit Global" title="Aktivitas Super Admin" description="Semua aksi lintas tenant, impersonate, suspend, dan perubahan subscription." actions={<ReloadButton onClick={reload} loading={loading} />} />
        <ErrorBanner message={error} />
        <AuditTable audit={data} />
      </div>
    </SuperAdminShell>
  );
}

export function SuperAdminSettingsPage() {
  const plans = useMemo(
    () => [
      ["Starter", "Rp299.000", "Billing"],
      ["Growth", "Rp799.000", "Billing, MikroTik, OLT"],
      ["Scale", "Rp1.499.000", "Semua modul operasional"],
    ],
    [],
  );

  return (
    <SuperAdminShell>
      <div className="space-y-6">
        <PageHeader eyebrow="Platform Settings" title="Konfigurasi platform" description="Konfigurasi global untuk plan, limit, default module, dan support channel ISPBoss." />
        <div className="grid gap-6 xl:grid-cols-2">
          <Section title="Plan defaults">
            <DataTable columns={["Plan", "MRR", "Included Modules"]} rows={plans} />
          </Section>
          <Section title="Security policy">
            <DataTable columns={["Policy", "Value"]} rows={[["Impersonate reason", "Required"], ["Global audit retention", "24 bulan"], ["Super Admin MFA", "Required"], ["RLS bypass", "Support only"]]} />
          </Section>
        </div>
      </div>
    </SuperAdminShell>
  );
}
