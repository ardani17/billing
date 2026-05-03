import {
  ArrowSquareOut,
  Buildings,
  CreditCard,
  Pulse,
  ShieldCheck,
  UserSwitch,
} from "@phosphor-icons/react/dist/ssr";
import SuperAdminShell from "./super-admin-shell";
import {
  platformAudit,
  platformInvoices,
  platformServiceHealth,
  platformTenants,
  platformTickets,
} from "./super-admin-data";
import { Button, DataTable, PageHeader, Section, StatGrid, StatusBadge } from "./ui";

export function SuperAdminOverviewPage() {
  return (
    <SuperAdminShell>
      <div className="space-y-6">
        <PageHeader
          eyebrow="Super Admin"
          title="Platform control center"
          description="Pantau semua tenant, subscription, health service, support, dan audit global ISPBoss."
          actions={<Button href="/super-admin/tenants">Kelola Tenant</Button>}
        />
        <StatGrid
          stats={[
            { label: "Tenant aktif", value: "126", delta: "+8 bulan ini" },
            { label: "MRR platform", value: "Rp94,8jt", delta: "+14,2%" },
            { label: "Trial berjalan", value: "19", delta: "7 konversi" },
            { label: "Insiden terbuka", value: "3", delta: "1 tinggi", tone: "amber" },
          ]}
        />
        <div className="grid gap-6 xl:grid-cols-[1.15fr_0.85fr]">
          <Section title="Tenant terbaru" description="Ringkasan tenant lintas platform.">
            <TenantTable compact />
          </Section>
          <Section title="Service health" description="Status service utama yang dipakai semua tenant.">
            <DataTable
              columns={["Service", "Region", "Latency", "Uptime", "Status"]}
              rows={platformServiceHealth.map((service) => [
                service.service,
                service.region,
                service.latency,
                service.uptime,
                <StatusBadge key={service.service} status={service.status} />,
              ])}
            />
          </Section>
        </div>
        <Section title="Audit global terbaru" description="Aksi lintas tenant dan event sistem.">
          <AuditTable />
        </Section>
      </div>
    </SuperAdminShell>
  );
}

function TenantTable({ compact = false }: { compact?: boolean }) {
  return (
    <DataTable
      columns={compact ? ["Tenant", "Plan", "MRR", "Health", "Aksi"] : ["Tenant", "Owner", "Domain", "Plan", "MRR", "Pelanggan", "Status", "Health", "Aksi"]}
      rows={platformTenants.map((tenant) =>
        compact
          ? [
              tenant.name,
              tenant.plan,
              tenant.mrr,
              <StatusBadge key={`${tenant.id}-health`} status={tenant.health} />,
              <Button key={tenant.id} variant="ghost" href={`/super-admin/tenants/${tenant.id}`}>Detail</Button>,
            ]
          : [
              <a key={tenant.id} href={`/super-admin/tenants/${tenant.id}`} className="font-semibold text-blue-700">{tenant.name}</a>,
              tenant.owner,
              tenant.domain,
              tenant.plan,
              tenant.mrr,
              tenant.customers,
              <StatusBadge key={`${tenant.id}-status`} status={tenant.status} />,
              <StatusBadge key={`${tenant.id}-health`} status={tenant.health} />,
              <Button key={tenant.id} variant="ghost" href={`/super-admin/tenants/${tenant.id}`}>Detail</Button>,
            ],
      )}
    />
  );
}

export function SuperAdminTenantsPage() {
  return (
    <SuperAdminShell>
      <div className="space-y-6">
        <PageHeader
          eyebrow="Tenant Management"
          title="Semua tenant"
          description="Daftar client ISP, subscription, domain, modul aktif, dan status operasional."
          actions={<Button>Tambah Tenant Manual</Button>}
        />
        <TenantTable />
      </div>
    </SuperAdminShell>
  );
}

export function SuperAdminTenantDetailPage() {
  const tenant = platformTenants[0] ?? {
    id: "tenant-demo",
    name: "Demo Tenant",
    owner: "Tenant Admin",
    domain: "demo.ispboss.id",
    plan: "Starter",
    mrr: "Rp0",
    customers: "0",
    modules: "Billing",
    status: "trial",
    health: "normal",
    lastSeen: "Baru saja",
  };
  return (
    <SuperAdminShell>
      <div className="space-y-6">
        <PageHeader
          eyebrow="Tenant Detail"
          title={tenant.name}
          description={`${tenant.owner} - ${tenant.domain} - ${tenant.modules}`}
          actions={
            <>
              <Button variant="secondary">Suspend</Button>
              <Button>Impersonate Admin</Button>
            </>
          }
        />
        <StatGrid
          stats={[
            { label: "Subscription", value: tenant.plan, delta: tenant.status },
            { label: "MRR", value: tenant.mrr, delta: "aktif" },
            { label: "Pelanggan", value: tenant.customers, delta: "tenant data" },
            { label: "Health", value: tenant.health, delta: tenant.lastSeen, tone: "amber" },
          ]}
        />
        <div className="grid gap-6 xl:grid-cols-[1fr_360px]">
          <Section title="Kontrol support" description="Aksi lintas tenant harus meninggalkan audit trail.">
            <div className="grid gap-3 sm:grid-cols-2">
              {[
                { title: "Impersonate Tenant Admin", icon: UserSwitch, description: "Masuk sebagai admin tenant untuk troubleshooting." },
                { title: "Buka tenant app", icon: ArrowSquareOut, description: "Buka dashboard tenant di tab yang sama." },
                { title: "Cek subscription", icon: CreditCard, description: "Lihat invoice platform dan renewal." },
                { title: "Review security", icon: ShieldCheck, description: "Audit login, domain, dan akses user." },
              ].map(({ title, icon: TypedIcon, description }) => {
                return (
                  <button key={title} type="button" className="rounded-lg border border-zinc-200 p-4 text-left hover:bg-zinc-50">
                    <TypedIcon size={22} />
                    <span className="mt-3 block font-semibold text-zinc-950">{title}</span>
                    <span className="mt-1 block text-sm leading-6 text-zinc-500">{description}</span>
                  </button>
                );
              })}
            </div>
          </Section>
          <Section title="Tenant profile">
            <DataTable
              columns={["Field", "Value"]}
              rows={[
                ["Tenant ID", tenant.id],
                ["Owner", tenant.owner],
                ["Domain", tenant.domain],
                ["Modules", tenant.modules],
                ["Last seen", tenant.lastSeen],
              ]}
            />
          </Section>
        </div>
      </div>
    </SuperAdminShell>
  );
}

export function SuperAdminSubscriptionsPage() {
  return (
    <SuperAdminShell>
      <div className="space-y-6">
        <PageHeader eyebrow="Platform Billing" title="Subscription tenant" description="Invoice platform, renewal, trial, dan status pembayaran tenant." />
        <DataTable
          columns={["Invoice", "Tenant", "Amount", "Jatuh Tempo", "Status"]}
          rows={platformInvoices.map((invoice) => [
            invoice.number,
            invoice.tenant,
            invoice.amount,
            invoice.dueDate,
            <StatusBadge key={invoice.number} status={invoice.status} />,
          ])}
        />
      </div>
    </SuperAdminShell>
  );
}

export function SuperAdminSupportPage() {
  return (
    <SuperAdminShell>
      <div className="space-y-6">
        <PageHeader eyebrow="Support Console" title="Tiket support lintas tenant" description="Tempat tim ISPBoss memprioritaskan bantuan, escalation, dan impersonate tenant." />
        <DataTable
          columns={["Kode", "Tenant", "Subject", "Priority", "Status"]}
          rows={platformTickets.map((ticket) => [
            ticket.code,
            ticket.tenant,
            ticket.subject,
            ticket.priority,
            <StatusBadge key={ticket.code} status={ticket.status} />,
          ])}
        />
      </div>
    </SuperAdminShell>
  );
}

export function SuperAdminHealthPage() {
  return (
    <SuperAdminShell>
      <div className="space-y-6">
        <PageHeader eyebrow="Service Health" title="Kesehatan platform" description="Pantau API, worker, notification, dan VPN gateway yang dipakai semua tenant." />
        <DataTable
          columns={["Service", "Region", "Latency", "Uptime", "Status"]}
          rows={platformServiceHealth.map((service) => [
            service.service,
            service.region,
            service.latency,
            service.uptime,
            <StatusBadge key={service.service} status={service.status} />,
          ])}
        />
      </div>
    </SuperAdminShell>
  );
}

function AuditTable() {
  return (
    <DataTable
      columns={["Waktu", "Actor", "Action", "Target", "Status"]}
      rows={platformAudit.map((audit) => [
        audit.time,
        audit.actor,
        audit.action,
        audit.target,
        <StatusBadge key={`${audit.time}-${audit.action}`} status={audit.status} />,
      ])}
    />
  );
}

export function SuperAdminAuditPage() {
  return (
    <SuperAdminShell>
      <div className="space-y-6">
        <PageHeader eyebrow="Audit Global" title="Aktivitas Super Admin" description="Semua aksi lintas tenant, impersonate, suspend, dan perubahan subscription." />
        <AuditTable />
      </div>
    </SuperAdminShell>
  );
}

export function SuperAdminSettingsPage() {
  return (
    <SuperAdminShell>
      <div className="space-y-6">
        <PageHeader eyebrow="Platform Settings" title="Konfigurasi platform" description="Pengaturan global untuk plan, limit, default module, dan support channel ISPBoss." />
        <div className="grid gap-6 xl:grid-cols-2">
          <Section title="Plan defaults">
            <DataTable columns={["Plan", "Tenant Limit", "Included Modules"]} rows={[["Starter", "300 pelanggan", "Billing"], ["Growth", "1.500 pelanggan", "Billing, MikroTik, OLT"], ["Scale", "5.000 pelanggan", "All modules"]]} />
          </Section>
          <Section title="Security policy">
            <DataTable columns={["Policy", "Value"]} rows={[["Impersonate reason", "Required"], ["Global audit retention", "24 bulan"], ["Super Admin MFA", "Required"], ["RLS bypass", "Support only"]]} />
          </Section>
        </div>
      </div>
    </SuperAdminShell>
  );
}
