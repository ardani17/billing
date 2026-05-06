"use client";

import { useCallback, useEffect, useState } from "react";
import AppShell from "./app-shell";
import {
  Button,
  DataTable,
  EmptyState,
  FormField,
  PageHeader,
  Section,
  StatGrid,
  StatusBadge,
  TextInput,
} from "./ui";

type AnyRecord = Record<string, any>;
type AddonModule = "mikrotik" | "fiber_network";
type ModuleCapabilities = { billing_core: boolean; mikrotik: boolean; fiber_network: boolean };

const defaultModules: ModuleCapabilities = { billing_core: true, mikrotik: false, fiber_network: false };

const selectClass =
  "h-10 w-full rounded-md border border-slate-300 bg-white px-3 text-sm text-slate-700 outline-none transition focus:border-blue-500 focus:ring-2 focus:ring-blue-100";

const textAreaClass =
  "w-full min-w-0 rounded-md border border-slate-300 px-3 py-2 text-sm outline-none transition focus:border-blue-500 focus:ring-2 focus:ring-blue-100";

const settingsSections: Array<{ href: string; title: string; body: string; module?: AddonModule }> = [
  { href: "/settings/profile", title: "Profil ISP", body: "Identitas tenant, alamat, kontak, dan legal invoice." },
  { href: "/settings/branding", title: "White Label", body: "Logo, warna, domain, invoice, dan walled garden." },
  { href: "/settings/users", title: "User & Role", body: "Manajemen operator, teknisi, kasir, dan akses tenant." },
  { href: "/settings/payment", title: "Payment Gateway", body: "Konfigurasi Xendit atau Midtrans untuk payment link." },
  { href: "/settings/notifications", title: "Notifikasi", body: "Provider, template, dan pengiriman WhatsApp/email/SMS." },
  { href: "/settings/security", title: "Keamanan", body: "Ganti password dan pengaturan akses akun." },
  { href: "/settings/billing", title: "Billing", body: "Aturan jatuh tempo, pajak, denda, isolir, dan billing cycle." },
  { href: "/settings/invoice", title: "Invoice", body: "Format nomor, footer, dan aturan dokumen tagihan." },
  { href: "/settings/mikrotik", title: "MikroTik", body: "Default koneksi dan profil sinkronisasi router.", module: "mikrotik" },
  { href: "/settings/olt", title: "OLT", body: "Default SNMP/CLI dan provisioning OLT.", module: "fiber_network" },
  { href: "/settings/map", title: "Peta", body: "Label dan preferensi FTTH visual mapping.", module: "fiber_network" },
  { href: "/settings/voucher", title: "Voucher", body: "Format kode dan aturan penjualan voucher." },
  { href: "/settings/localization", title: "Lokalisasi", body: "Timezone, mata uang, dan format tanggal." },
  { href: "/settings/reports", title: "Laporan", body: "Target KPI, jadwal laporan, dan template custom report." },
  { href: "/settings/subscription", title: "Subscription", body: "Status paket SaaS tenant." },
  { href: "/settings/audit-log", title: "Audit Log", body: "Jejak perubahan penting pada tenant." },
];

function unwrap(body: any) {
  if (body && typeof body === "object" && "success" in body && "data" in body) return body.data;
  return body;
}

function listOf(payload: any): AnyRecord[] {
  if (Array.isArray(payload)) return payload;
  if (Array.isArray(payload?.data)) return payload.data;
  if (Array.isArray(payload?.items)) return payload.items;
  return [];
}

function apiError(body: any, fallback: string) {
  if (body?.error?.message) return body.error.message;
  if (typeof body?.error === "string") return body.error;
  return fallback;
}

async function apiGet(url: string) {
  const res = await fetch(url, { cache: "no-store" });
  const body = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(apiError(body, `Request gagal (${res.status})`));
  return unwrap(body);
}

async function apiSend(url: string, method: string, payload: AnyRecord) {
  const res = await fetch(url, {
    method,
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
  const body = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(apiError(body, `Request gagal (${res.status})`));
  return unwrap(body);
}

function useModuleCapabilities() {
  const [modules, setModules] = useState<ModuleCapabilities>(defaultModules);

  useEffect(() => {
    let alive = true;
    apiGet("/api/billing/tenant/modules")
      .then((payload) => {
        if (!alive) return;
        const nextModules = payload?.modules;
        if (nextModules) {
          setModules({
            billing_core: nextModules.billing_core !== false,
            mikrotik: nextModules.mikrotik === true,
            fiber_network: nextModules.fiber_network === true,
          });
        }
      })
      .catch(() => {
        if (alive) setModules(defaultModules);
      });
    return () => {
      alive = false;
    };
  }, []);

  return modules;
}

function useApi(url: string, fallback: any) {
  const [version, setVersion] = useState(0);
  const [data, setData] = useState(fallback);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    let alive = true;
    setLoading(true);
    setError("");
    apiGet(url)
      .then((payload) => {
        if (!alive) return;
        setData(payload);
        setLoading(false);
      })
      .catch((err) => {
        if (!alive) return;
        setData(fallback);
        setError(err instanceof Error ? err.message : "Gagal memuat data");
        setLoading(false);
      });
    return () => {
      alive = false;
    };
  }, [url, version]);

  return { data, loading, error, reload: () => setVersion((value) => value + 1) };
}

function Notice({ loading, error }: { loading: boolean; error: string }) {
  if (loading) return <p className="text-sm text-slate-500">Memuat data...</p>;
  if (error) return <p className="text-sm text-red-600">{error}</p>;
  return null;
}

function SubmitBar({
  saving,
  error,
  success,
  label,
}: {
  saving: boolean;
  error: string;
  success: string;
  label: string;
}) {
  return (
    <div className="flex flex-col gap-3 sm:flex-row sm:items-center">
      <button
        type="submit"
        disabled={saving}
        className="inline-flex min-w-0 items-center justify-center rounded-md bg-blue-600 px-4 py-2 text-center text-sm font-semibold leading-5 text-white transition hover:bg-blue-700 disabled:opacity-60"
      >
        {saving ? "Menyimpan..." : label}
      </button>
      {error && <span className="text-sm text-red-600">{error}</span>}
      {success && <span className="text-sm text-emerald-700">{success}</span>}
    </div>
  );
}

function SettingsShell({ children }: { children: React.ReactNode }) {
  return (
    <AppShell>
      <div className="space-y-6">{children}</div>
    </AppShell>
  );
}

export function SettingsIndexLivePage() {
  const modules = useModuleCapabilities();
  const visibleSections = settingsSections.filter((section) => !section.module || modules[section.module]);

  return (
    <SettingsShell>
      <PageHeader
        eyebrow="Pengaturan"
        title="Pengaturan tenant"
        description="Pusat konfigurasi aplikasi tenant. Bagian yang sudah punya backend disambungkan langsung ke API lokal."
      />
      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
        {visibleSections.map((section) => (
          <a key={section.href} href={section.href} className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm transition hover:border-blue-200 hover:bg-blue-50/30">
            <h2 className="font-semibold tracking-tight text-slate-950">{section.title}</h2>
            <p className="mt-2 text-sm leading-6 text-slate-500">{section.body}</p>
          </a>
        ))}
      </div>
    </SettingsShell>
  );
}

export function SettingsUsersLivePage() {
  const users = useApi("/api/billing/settings/users", []);
  const rows = listOf(users.data);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");

  const onSubmit = useCallback(
    async (event: React.FormEvent<HTMLFormElement>) => {
      event.preventDefault();
      const form = new FormData(event.currentTarget);
      setSaving(true);
      setError("");
      setSuccess("");
      try {
        await apiSend("/api/billing/settings/users", "POST", {
          name: String(form.get("name") || ""),
          email: String(form.get("email") || ""),
          phone: String(form.get("phone") || ""),
          password: String(form.get("password") || ""),
          role: String(form.get("role") || "operator"),
        });
        event.currentTarget.reset();
        users.reload();
        setSuccess("User berhasil dibuat.");
      } catch (err) {
        setError(err instanceof Error ? err.message : "Gagal membuat user");
      } finally {
        setSaving(false);
      }
    },
    [users],
  );

  return (
    <SettingsShell>
      <PageHeader eyebrow="Pengaturan" title="User & role" description="Data user tenant dari Billing API." />
      <StatGrid
        stats={[
          { label: "Total user", value: String(rows.length) },
          { label: "Aktif", value: String(rows.filter((user) => user.is_active !== false).length), tone: "green" },
          { label: "Admin tenant", value: String(rows.filter((user) => user.role === "tenant_admin").length) },
          { label: "Operator/kasir", value: String(rows.filter((user) => ["operator", "kasir"].includes(user.role)).length) },
        ]}
      />
      <Section title="Daftar user">
        <Notice loading={users.loading} error={users.error} />
        {rows.length === 0 && !users.loading ? (
          <EmptyState title="Belum ada user" description="User tenant akan muncul setelah dibuat." />
        ) : (
          <DataTable
            columns={["Nama", "Email", "Telepon", "Role", "Status"]}
            rows={rows.map((user) => [
              user.name,
              user.email,
              user.phone ?? "-",
              user.role,
              <StatusBadge key={user.id} status={user.is_active === false ? "nonaktif" : "aktif"} />,
            ])}
          />
        )}
      </Section>
      <Section title="Tambah user">
        <form onSubmit={onSubmit} className="grid gap-4 lg:grid-cols-4">
          <FormField label="Nama"><TextInput name="name" required placeholder="Operator NOC" /></FormField>
          <FormField label="Email"><TextInput name="email" type="email" required placeholder="operator@isp.local" /></FormField>
          <FormField label="Telepon"><TextInput name="phone" placeholder="+628123000111" /></FormField>
          <FormField label="Role">
            <select name="role" className={selectClass} defaultValue="operator">
              <option value="operator">Operator</option>
              <option value="teknisi">Teknisi</option>
              <option value="kasir">Kasir</option>
              <option value="reseller">Reseller</option>
            </select>
          </FormField>
          <FormField label="Password"><TextInput name="password" type="password" required minLength={8} /></FormField>
          <div className="lg:col-span-3">
            <SubmitBar saving={saving} error={error} success={success} label="Tambah user" />
          </div>
        </form>
      </Section>
    </SettingsShell>
  );
}

export function SettingsPaymentLivePage() {
  const gateways = useApi("/api/billing/settings/payment-gateways", []);
  const rows = listOf(gateways.data);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");

  const onSubmit = useCallback(
    async (event: React.FormEvent<HTMLFormElement>) => {
      event.preventDefault();
      const form = new FormData(event.currentTarget);
      setSaving(true);
      setError("");
      setSuccess("");
      try {
        await apiSend("/api/billing/settings/payment-gateways", "POST", {
          gateway_provider: String(form.get("gateway_provider") || "xendit"),
          api_key: String(form.get("api_key") || ""),
          webhook_secret: String(form.get("webhook_secret") || ""),
          enabled_methods: String(form.get("enabled_methods") || "qris,va_bca")
            .split(",")
            .map((item) => item.trim())
            .filter(Boolean),
          payment_link_expiry_days: Number(form.get("payment_link_expiry_days") || 7),
        });
        event.currentTarget.reset();
        gateways.reload();
        setSuccess("Payment gateway berhasil disimpan.");
      } catch (err) {
        setError(err instanceof Error ? err.message : "Gagal menyimpan gateway");
      } finally {
        setSaving(false);
      }
    },
    [gateways],
  );

  return (
    <SettingsShell>
      <PageHeader eyebrow="Pengaturan" title="Payment gateway" description="Konfigurasi gateway dari Billing API." />
      <Section title="Gateway aktif">
        <Notice loading={gateways.loading} error={gateways.error} />
        {rows.length === 0 && !gateways.loading ? (
          <EmptyState title="Belum ada gateway" description="Tambahkan Xendit atau Midtrans agar payment link bisa dibuat." />
        ) : (
          <DataTable
            columns={["Provider", "API key", "Metode", "Expired link", "Status"]}
            rows={rows.map((gateway) => [
              gateway.gateway_provider,
              gateway.api_key_masked ?? "-",
              Array.isArray(gateway.enabled_methods) ? gateway.enabled_methods.join(", ") : "-",
              `${gateway.payment_link_expiry_days ?? 7} hari`,
              <StatusBadge key={gateway.id} status={gateway.is_active ? "aktif" : "nonaktif"} />,
            ])}
          />
        )}
      </Section>
      <Section title="Tambah gateway">
        <form onSubmit={onSubmit} className="grid gap-4 lg:grid-cols-4">
          <FormField label="Provider">
            <select name="gateway_provider" className={selectClass} defaultValue="xendit">
              <option value="xendit">Xendit</option>
              <option value="midtrans">Midtrans</option>
            </select>
          </FormField>
          <FormField label="API key"><TextInput name="api_key" required minLength={10} placeholder="xnd_development_..." /></FormField>
          <FormField label="Webhook secret"><TextInput name="webhook_secret" required minLength={10} /></FormField>
          <FormField label="Expired link"><TextInput name="payment_link_expiry_days" type="number" min="1" max="30" defaultValue="7" /></FormField>
          <div className="lg:col-span-4">
            <FormField label="Metode aktif"><TextInput name="enabled_methods" defaultValue="qris,va_bca,va_bri,ewallet_dana" /></FormField>
          </div>
          <div className="lg:col-span-4">
            <SubmitBar saving={saving} error={error} success={success} label="Simpan gateway" />
          </div>
        </form>
      </Section>
    </SettingsShell>
  );
}

export function SettingsNotificationsLivePage() {
  const config = useApi("/api/notification/notifications/config", {});
  const templates = useApi("/api/notification/notifications/templates?page_size=50", []);
  const rows = listOf(templates.data);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");

  const onSubmit = useCallback(
    async (event: React.FormEvent<HTMLFormElement>) => {
      event.preventDefault();
      const form = new FormData(event.currentTarget);
      setSaving(true);
      setError("");
      setSuccess("");
      try {
        await apiSend("/api/notification/notifications/templates", "POST", {
          slug: String(form.get("slug") || ""),
          name: String(form.get("name") || ""),
          category: String(form.get("category") || "transactional"),
          event_type: String(form.get("event_type") || ""),
          channels: String(form.get("channels") || "whatsapp").split(",").map((item) => item.trim()).filter(Boolean),
          body_whatsapp: String(form.get("body_whatsapp") || ""),
          body_email_subject: String(form.get("body_email_subject") || ""),
          body_email_html: String(form.get("body_email_html") || ""),
        });
        event.currentTarget.reset();
        templates.reload();
        setSuccess("Template berhasil dibuat.");
      } catch (err) {
        setError(err instanceof Error ? err.message : "Gagal membuat template");
      } finally {
        setSaving(false);
      }
    },
    [templates],
  );

  return (
    <SettingsShell>
      <PageHeader eyebrow="Pengaturan" title="Notifikasi" description="Template dan konfigurasi notification-service." />
      <Section title="Konfigurasi provider">
        <Notice loading={config.loading} error={config.error} />
        {!config.loading && !config.error && (
          <DataTable
            columns={["Channel", "Provider", "Status"]}
            rows={listOf(config.data?.configs ?? config.data).map((item) => [
              item.channel,
              item.provider,
              <StatusBadge key={item.id ?? item.channel} status={item.is_enabled ? "aktif" : "nonaktif"} />,
            ])}
          />
        )}
      </Section>
      <Section title="Template">
        <Notice loading={templates.loading} error={templates.error} />
        {rows.length === 0 && !templates.loading ? (
          <EmptyState title="Belum ada template" description="Template akan dipakai oleh event billing dan broadcast." />
        ) : (
          <DataTable
            columns={["Nama", "Slug", "Event", "Channel", "Status"]}
            rows={rows.map((template) => [
              template.name,
              template.slug,
              template.event_type ?? "-",
              Array.isArray(template.channels) ? template.channels.join(", ") : "-",
              <StatusBadge key={template.id} status={template.is_active === false ? "nonaktif" : "aktif"} />,
            ])}
          />
        )}
      </Section>
      <Section title="Tambah template">
        <form onSubmit={onSubmit} className="grid gap-4 lg:grid-cols-4">
          <FormField label="Slug"><TextInput name="slug" required placeholder="payment-reminder" /></FormField>
          <FormField label="Nama"><TextInput name="name" required placeholder="Pengingat pembayaran" /></FormField>
          <FormField label="Kategori">
            <select name="category" className={selectClass} defaultValue="transactional">
              <option value="transactional">Transactional</option>
              <option value="reminder">Reminder</option>
              <option value="promotion">Promotion</option>
              <option value="information">Information</option>
            </select>
          </FormField>
          <FormField label="Event"><TextInput name="event_type" placeholder="invoice.created" /></FormField>
          <div className="lg:col-span-2">
            <FormField label="Channel"><TextInput name="channels" defaultValue="whatsapp,email" /></FormField>
          </div>
          <div className="lg:col-span-2">
            <FormField label="Subject email"><TextInput name="body_email_subject" placeholder="opsional" /></FormField>
          </div>
          <div className="lg:col-span-2">
            <FormField label="Body WhatsApp"><textarea name="body_whatsapp" required className={textAreaClass} rows={4} /></FormField>
          </div>
          <div className="lg:col-span-2">
            <FormField label="Body Email HTML"><textarea name="body_email_html" className={textAreaClass} rows={4} /></FormField>
          </div>
          <div className="lg:col-span-4">
            <SubmitBar saving={saving} error={error} success={success} label="Tambah template" />
          </div>
        </form>
      </Section>
    </SettingsShell>
  );
}

export function SettingsSecurityLivePage() {
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");

  const onSubmit = useCallback(async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    setSaving(true);
    setError("");
    setSuccess("");
    try {
      await apiSend("/api/billing/settings/security/change-password", "POST", {
        current_password: String(form.get("current_password") || ""),
        new_password: String(form.get("new_password") || ""),
      });
      event.currentTarget.reset();
      setSuccess("Password berhasil diganti.");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Gagal mengganti password");
    } finally {
      setSaving(false);
    }
  }, []);

  return (
    <SettingsShell>
      <PageHeader eyebrow="Pengaturan" title="Keamanan" description="Aksi keamanan yang sudah tersedia di Billing API." />
      <Section title="Ganti password">
        <form onSubmit={onSubmit} className="grid gap-4 lg:grid-cols-2">
          <FormField label="Password saat ini"><TextInput name="current_password" type="password" required /></FormField>
          <FormField label="Password baru"><TextInput name="new_password" type="password" required minLength={8} /></FormField>
          <div className="lg:col-span-2">
            <SubmitBar saving={saving} error={error} success={success} label="Ganti password" />
          </div>
        </form>
      </Section>
    </SettingsShell>
  );
}

export function SettingsBillingLivePage() {
  const settings = useApi("/api/billing/settings/billing", {});
  const data = settings.data ?? {};
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");

  const onSubmit = useCallback(
    async (event: React.FormEvent<HTMLFormElement>) => {
      event.preventDefault();
      const form = new FormData(event.currentTarget);
      setSaving(true);
      setError("");
      setSuccess("");
      try {
        await apiSend("/api/billing/settings/billing", "PUT", {
          generate_days: Number(form.get("generate_days") || 1),
          grace_period_days: Number(form.get("grace_period_days") || 0),
          suspend_days: Number(form.get("suspend_days") || 0),
          tax_enabled: form.get("tax_enabled") === "on",
          tax_rate: Number(form.get("tax_rate") || 0),
          penalty_enabled: form.get("penalty_enabled") === "on",
          penalty_type: String(form.get("penalty_type") || "fixed"),
          penalty_amount: Number(form.get("penalty_amount") || 0),
          penalty_percentage: Number(form.get("penalty_percentage") || 0),
          penalty_daily_amount: Number(form.get("penalty_daily_amount") || 0),
          penalty_max_amount: Number(form.get("penalty_max_amount") || 0),
          invoice_prefix: String(form.get("invoice_prefix") || "INV"),
          new_customer_billing: String(form.get("new_customer_billing") || "prorate"),
          timezone: String(form.get("timezone") || "Asia/Jakarta"),
          auto_isolir: form.get("auto_isolir") === "on",
          auto_open_isolir: form.get("auto_open_isolir") === "on",
        });
        settings.reload();
        setSuccess("Billing settings berhasil disimpan.");
      } catch (err) {
        setError(err instanceof Error ? err.message : "Gagal menyimpan billing settings");
      } finally {
        setSaving(false);
      }
    },
    [settings],
  );

  return (
    <SettingsShell>
      <PageHeader eyebrow="Pengaturan" title="Billing" description="Konfigurasi billing tenant yang tersambung langsung ke Billing API." />
      <Notice loading={settings.loading} error={settings.error} />
      <form key={data.updated_at ?? data.id ?? "billing-defaults"} onSubmit={onSubmit} className="space-y-6">
        <Section title="Invoice dan siklus tagihan" description="Atur nomor invoice, tanggal generate, dan perlakuan pelanggan baru.">
          <div className="grid gap-4 lg:grid-cols-4">
            <FormField label="Prefix invoice"><TextInput name="invoice_prefix" required minLength={2} maxLength={12} defaultValue={data.invoice_prefix ?? "INV"} /></FormField>
            <FormField label="Generate H-"><TextInput name="generate_days" type="number" min="0" max="31" defaultValue={String(data.generate_days ?? 1)} /></FormField>
            <FormField label="Grace period"><TextInput name="grace_period_days" type="number" min="0" max="90" defaultValue={String(data.grace_period_days ?? 3)} /></FormField>
            <FormField label="Suspend setelah"><TextInput name="suspend_days" type="number" min="0" max="180" defaultValue={String(data.suspend_days ?? 30)} /></FormField>
            <FormField label="Billing pelanggan baru">
              <select name="new_customer_billing" className={selectClass} defaultValue={data.new_customer_billing ?? "prorate"}>
                <option value="prorate">Prorate</option>
                <option value="full_next_cycle">Full next cycle</option>
                <option value="immediate">Immediate</option>
              </select>
            </FormField>
            <FormField label="Timezone">
              <select name="timezone" className={selectClass} defaultValue={data.timezone ?? "Asia/Jakarta"}>
                <option value="Asia/Jakarta">Asia/Jakarta</option>
                <option value="Asia/Makassar">Asia/Makassar</option>
                <option value="Asia/Jayapura">Asia/Jayapura</option>
              </select>
            </FormField>
          </div>
        </Section>

        <Section title="Pajak dan denda" description="Nilai ini dipakai saat invoice baru dibuat atau saat denda keterlambatan dihitung.">
          <div className="grid gap-4 lg:grid-cols-4">
            <label className="flex items-center gap-3 rounded-md border border-slate-200 px-3 py-2 text-sm font-medium text-slate-800">
              <input name="tax_enabled" type="checkbox" defaultChecked={data.tax_enabled === true} className="h-4 w-4 rounded border-slate-300" />
              Aktifkan pajak
            </label>
            <FormField label="Tax rate (%)"><TextInput name="tax_rate" type="number" min="0" max="100" step="0.01" defaultValue={String(data.tax_rate ?? 0)} /></FormField>
            <label className="flex items-center gap-3 rounded-md border border-slate-200 px-3 py-2 text-sm font-medium text-slate-800">
              <input name="penalty_enabled" type="checkbox" defaultChecked={data.penalty_enabled === true} className="h-4 w-4 rounded border-slate-300" />
              Aktifkan denda
            </label>
            <FormField label="Tipe denda">
              <select name="penalty_type" className={selectClass} defaultValue={data.penalty_type ?? "fixed"}>
                <option value="fixed">Fixed</option>
                <option value="percentage">Percentage</option>
                <option value="daily">Daily</option>
              </select>
            </FormField>
            <FormField label="Denda fixed"><TextInput name="penalty_amount" type="number" min="0" defaultValue={String(data.penalty_amount ?? 0)} /></FormField>
            <FormField label="Denda persen"><TextInput name="penalty_percentage" type="number" min="0" max="100" step="0.01" defaultValue={String(data.penalty_percentage ?? 0)} /></FormField>
            <FormField label="Denda harian"><TextInput name="penalty_daily_amount" type="number" min="0" defaultValue={String(data.penalty_daily_amount ?? 0)} /></FormField>
            <FormField label="Maksimum denda"><TextInput name="penalty_max_amount" type="number" min="0" defaultValue={String(data.penalty_max_amount ?? 0)} /></FormField>
          </div>
        </Section>

        <Section title="Isolir otomatis" description="Kontrol otomatisasi isolir dan buka isolir setelah pembayaran.">
          <div className="grid gap-4 lg:grid-cols-2">
            <label className="flex items-center gap-3 rounded-md border border-slate-200 px-3 py-2 text-sm font-medium text-slate-800">
              <input name="auto_isolir" type="checkbox" defaultChecked={data.auto_isolir === true} className="h-4 w-4 rounded border-slate-300" />
              Auto isolir pelanggan overdue
            </label>
            <label className="flex items-center gap-3 rounded-md border border-slate-200 px-3 py-2 text-sm font-medium text-slate-800">
              <input name="auto_open_isolir" type="checkbox" defaultChecked={data.auto_open_isolir !== false} className="h-4 w-4 rounded border-slate-300" />
              Auto buka isolir setelah lunas
            </label>
          </div>
        </Section>

        <SubmitBar saving={saving} error={error} success={success} label="Simpan billing settings" />
      </form>
    </SettingsShell>
  );
}

export function SettingsReportsLivePage() {
  const kpi = useApi("/api/billing/reports/kpi-targets", {});
  const schedules = useApi("/api/billing/reports/schedules", []);
  const templates = useApi("/api/billing/reports/custom/templates", []);
  const [savingKpi, setSavingKpi] = useState(false);
  const [savingSchedule, setSavingSchedule] = useState(false);
  const [savingTemplate, setSavingTemplate] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");

  const onKpiSubmit = useCallback(
    async (event: React.FormEvent<HTMLFormElement>) => {
      event.preventDefault();
      const form = new FormData(event.currentTarget);
      setSavingKpi(true);
      setError("");
      setSuccess("");
      try {
        await apiSend("/api/billing/reports/kpi-targets", "PUT", {
          monthly_revenue_target: Number(form.get("monthly_revenue_target") || 0) || undefined,
          collection_rate_target: Number(form.get("collection_rate_target") || 0) || undefined,
          max_receivables: Number(form.get("max_receivables") || 0) || undefined,
          new_customers_monthly_target: Number(form.get("new_customers_monthly_target") || 0) || undefined,
          max_churn_rate: Number(form.get("max_churn_rate") || 0) || undefined,
        });
        kpi.reload();
        setSuccess("Target KPI berhasil disimpan.");
      } catch (err) {
        setError(err instanceof Error ? err.message : "Gagal menyimpan KPI");
      } finally {
        setSavingKpi(false);
      }
    },
    [kpi],
  );

  const onScheduleSubmit = useCallback(
    async (event: React.FormEvent<HTMLFormElement>) => {
      event.preventDefault();
      const form = new FormData(event.currentTarget);
      setSavingSchedule(true);
      setError("");
      setSuccess("");
      try {
        await apiSend("/api/billing/reports/schedules", "POST", {
          report_type: String(form.get("report_type") || "profit_loss"),
          schedule_type: String(form.get("schedule_type") || "monthly"),
          format: String(form.get("format") || "xlsx"),
          recipients: [{ type: String(form.get("recipient_type") || "email"), address: String(form.get("recipient_address") || "") }],
        });
        event.currentTarget.reset();
        schedules.reload();
        setSuccess("Jadwal laporan berhasil dibuat.");
      } catch (err) {
        setError(err instanceof Error ? err.message : "Gagal membuat jadwal");
      } finally {
        setSavingSchedule(false);
      }
    },
    [schedules],
  );

  const onTemplateSubmit = useCallback(
    async (event: React.FormEvent<HTMLFormElement>) => {
      event.preventDefault();
      const form = new FormData(event.currentTarget);
      setSavingTemplate(true);
      setError("");
      setSuccess("");
      try {
        await apiSend("/api/billing/reports/custom/templates", "POST", {
          name: String(form.get("name") || ""),
          metrics: String(form.get("metrics") || "revenue")
            .split(",")
            .map((item) => item.trim())
            .filter(Boolean)
            .slice(0, 3),
          group_by: String(form.get("group_by") || "month"),
          sub_group_by: String(form.get("sub_group_by") || ""),
          display_type: String(form.get("display_type") || "table"),
          default_period_range: String(form.get("default_period_range") || "this_month"),
        });
        event.currentTarget.reset();
        templates.reload();
        setSuccess("Template custom report berhasil dibuat.");
      } catch (err) {
        setError(err instanceof Error ? err.message : "Gagal membuat template");
      } finally {
        setSavingTemplate(false);
      }
    },
    [templates],
  );

  const kpiData = kpi.data ?? {};
  const scheduleRows = listOf(schedules.data);
  const templateRows = listOf(templates.data);

  return (
    <SettingsShell>
      <PageHeader eyebrow="Pengaturan" title="Laporan" description="Target KPI, jadwal laporan otomatis, dan template custom report." />
      {(error || success) && <SubmitBar saving={false} error={error} success={success} label="Status" />}
      <Section title="Target KPI">
        <Notice loading={kpi.loading} error={kpi.error} />
        <form key={kpiData.updated_at ?? kpiData.id ?? "kpi-defaults"} onSubmit={onKpiSubmit} className="grid gap-4 lg:grid-cols-5">
          <FormField label="Revenue bulanan"><TextInput name="monthly_revenue_target" type="number" min="1" defaultValue={String(kpiData.monthly_revenue_target ?? "")} /></FormField>
          <FormField label="Collection rate"><TextInput name="collection_rate_target" type="number" min="0" max="100" step="0.01" defaultValue={String(kpiData.collection_rate_target ?? "")} /></FormField>
          <FormField label="Maks piutang"><TextInput name="max_receivables" type="number" min="1" defaultValue={String(kpiData.max_receivables ?? "")} /></FormField>
          <FormField label="Pelanggan baru"><TextInput name="new_customers_monthly_target" type="number" min="1" defaultValue={String(kpiData.new_customers_monthly_target ?? "")} /></FormField>
          <FormField label="Maks churn"><TextInput name="max_churn_rate" type="number" min="0" max="100" step="0.01" defaultValue={String(kpiData.max_churn_rate ?? "")} /></FormField>
          <div className="lg:col-span-5">
            <SubmitBar saving={savingKpi} error="" success="" label="Simpan KPI" />
          </div>
        </form>
      </Section>

      <Section title="Jadwal laporan">
        <Notice loading={schedules.loading} error={schedules.error} />
        {scheduleRows.length > 0 && (
          <DataTable
            columns={["Report", "Frekuensi", "Format", "Penerima", "Status"]}
            rows={scheduleRows.map((schedule) => [
              schedule.report_type,
              schedule.schedule_type,
              schedule.format,
              Array.isArray(schedule.recipients) ? schedule.recipients.map((item: AnyRecord) => item.address).join(", ") : "-",
              <StatusBadge key={schedule.id} status={schedule.is_active === false ? "nonaktif" : "aktif"} />,
            ])}
          />
        )}
        <form onSubmit={onScheduleSubmit} className="mt-5 grid gap-4 lg:grid-cols-5">
          <FormField label="Report type"><TextInput name="report_type" defaultValue="profit_loss" /></FormField>
          <FormField label="Frekuensi">
            <select name="schedule_type" className={selectClass} defaultValue="monthly">
              <option value="daily">Daily</option>
              <option value="weekly">Weekly</option>
              <option value="monthly">Monthly</option>
            </select>
          </FormField>
          <FormField label="Format">
            <select name="format" className={selectClass} defaultValue="xlsx">
              <option value="xlsx">XLSX</option>
              <option value="pdf">PDF</option>
            </select>
          </FormField>
          <FormField label="Tipe penerima">
            <select name="recipient_type" className={selectClass} defaultValue="email">
              <option value="email">Email</option>
              <option value="whatsapp">WhatsApp</option>
            </select>
          </FormField>
          <FormField label="Alamat penerima"><TextInput name="recipient_address" required /></FormField>
          <div className="lg:col-span-5">
            <SubmitBar saving={savingSchedule} error="" success="" label="Tambah jadwal" />
          </div>
        </form>
      </Section>

      <Section title="Template custom report">
        <Notice loading={templates.loading} error={templates.error} />
        {templateRows.length > 0 && (
          <DataTable
            columns={["Nama", "Metric", "Group", "Tampilan"]}
            rows={templateRows.map((template) => [
              template.name,
              Array.isArray(template.metrics) ? template.metrics.join(", ") : "-",
              [template.group_by, template.sub_group_by].filter(Boolean).join(" / "),
              template.display_type,
            ])}
          />
        )}
        <form onSubmit={onTemplateSubmit} className="mt-5 grid gap-4 lg:grid-cols-5">
          <FormField label="Nama"><TextInput name="name" required placeholder="Laporan revenue area" /></FormField>
          <FormField label="Metrics"><TextInput name="metrics" defaultValue="revenue,expenses,net_profit" /></FormField>
          <FormField label="Group by">
            <select name="group_by" className={selectClass} defaultValue="month">
              <option value="area">Area</option>
              <option value="package">Package</option>
              <option value="month">Month</option>
              <option value="status">Status</option>
              <option value="connection_method">Connection method</option>
              <option value="router">Router</option>
            </select>
          </FormField>
          <FormField label="Display">
            <select name="display_type" className={selectClass} defaultValue="table">
              <option value="table">Table</option>
              <option value="bar_chart">Bar chart</option>
              <option value="line_chart">Line chart</option>
              <option value="pie_chart">Pie chart</option>
            </select>
          </FormField>
          <FormField label="Periode default"><TextInput name="default_period_range" defaultValue="this_month" /></FormField>
          <div className="lg:col-span-5">
            <SubmitBar saving={savingTemplate} error="" success="" label="Tambah template" />
          </div>
        </form>
      </Section>
    </SettingsShell>
  );
}

export function SettingsGenericLivePage({
  title,
  description,
  moduleCode,
  moduleName,
}: {
  title: string;
  description: string;
  moduleCode?: AddonModule;
  moduleName?: string;
}) {
  const modules = useModuleCapabilities();

  if (moduleCode && !modules[moduleCode]) {
    return (
      <SettingsShell>
        <PageHeader eyebrow="Pengaturan" title={title} description={description} />
        <Section title="Modul nonaktif">
          <EmptyState
            title={`${moduleName || title} belum aktif`}
            description="Billing Core tetap berjalan normal. Aktifkan add-on ini dari pengaturan subscription sebelum memakai konfigurasi modulnya."
            action={<Button href="/settings/subscription">Buka subscription</Button>}
          />
        </Section>
      </SettingsShell>
    );
  }

  return (
    <SettingsShell>
      <PageHeader eyebrow="Pengaturan" title={title} description={description} />
      <Section title="Status implementasi">
        <EmptyState
          title="Belum ada endpoint persistensi"
          description="UI ini sengaja tidak menampilkan data palsu. Begitu endpoint backend modul ini tersedia, halaman ini bisa langsung disambungkan seperti users, payment, dan notifikasi."
          action={<Button href="/settings">Kembali ke pengaturan</Button>}
        />
      </Section>
    </SettingsShell>
  );
}
