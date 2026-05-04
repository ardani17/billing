"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import {
  Broadcast,
  CreditCard,
  Package,
  Plus,
  Receipt,
  Storefront,
  Ticket,
  Users,
} from "@phosphor-icons/react";
import AppShell from "./app-shell";
import { MikrotikModuleNav } from "../mikrotik/components/MikrotikModuleNav";
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
type ApiState<T> = { data: T; loading: boolean; error: string | null };

const textAreaClass =
  "w-full min-w-0 rounded-md border border-slate-300 px-3 py-2 text-sm outline-none transition focus:border-blue-500 focus:ring-2 focus:ring-blue-100";
const selectClass =
  "h-10 w-full rounded-md border border-slate-300 bg-white px-3 text-sm text-slate-700 outline-none transition focus:border-blue-500 focus:ring-2 focus:ring-blue-100";

function unwrap<T>(body: any): T {
  if (body && typeof body === "object" && "success" in body && "data" in body) {
    return body.data as T;
  }
  return body as T;
}

function apiError(body: any, fallback: string) {
  if (body?.error?.message) return body.error.message;
  if (typeof body?.error === "string") return body.error;
  return fallback;
}

async function apiGet<T>(url: string): Promise<T> {
  const res = await fetch(url, { cache: "no-store" });
  const body = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(apiError(body, `Request gagal (${res.status})`));
  return unwrap<T>(body);
}

async function apiSend<T>(url: string, method: string, payload: AnyRecord): Promise<T> {
  const res = await fetch(url, {
    method,
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
  const body = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(apiError(body, `Request gagal (${res.status})`));
  return unwrap<T>(body);
}

function listOf(payload: any): AnyRecord[] {
  if (Array.isArray(payload)) return payload;
  if (Array.isArray(payload?.data)) return payload.data;
  if (Array.isArray(payload?.items)) return payload.items;
  return [];
}

function firstOf(payload: any, keys: string[]): AnyRecord {
  for (const key of keys) {
    if (payload?.[key] && typeof payload[key] === "object") return payload[key];
  }
  return payload && typeof payload === "object" ? payload : {};
}

function money(value: unknown) {
  const number = Number(value ?? 0);
  return new Intl.NumberFormat("id-ID", {
    style: "currency",
    currency: "IDR",
    maximumFractionDigits: 0,
  }).format(number);
}

function dateID(value?: string) {
  if (!value) return "-";
  return new Intl.DateTimeFormat("id-ID", { dateStyle: "medium" }).format(
    new Date(value),
  );
}

function useApi<T>(url: string, fallback: T): ApiState<T> & { reload: () => void } {
  const [version, setVersion] = useState(0);
  const [state, setState] = useState<ApiState<T>>({
    data: fallback,
    loading: true,
    error: null,
  });

  useEffect(() => {
    let alive = true;
    setState((current) => ({ ...current, loading: true, error: null }));
    apiGet<T>(url)
      .then((data) => alive && setState({ data, loading: false, error: null }))
      .catch((error) =>
        alive &&
        setState({
          data: fallback,
          loading: false,
          error: error instanceof Error ? error.message : "Gagal memuat data",
        }),
      );
    return () => {
      alive = false;
    };
  }, [url, version]);

  return { ...state, reload: () => setVersion((value) => value + 1) };
}

function Notice({ loading, error }: { loading: boolean; error: string | null }) {
  if (loading) return <p className="text-sm text-slate-500">Memuat data...</p>;
  if (error) return <p className="text-sm text-red-600">{error}</p>;
  return null;
}

function todayDate() {
  return new Date().toISOString().slice(0, 10);
}

function DetailGrid({ items }: { items: { label: string; value: React.ReactNode }[] }) {
  return (
    <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
      {items.map((item) => (
        <div key={item.label} className="min-w-0 rounded-lg border border-slate-200 bg-slate-50 p-4">
          <p className="text-xs font-semibold uppercase tracking-[0.14em] text-slate-400">{item.label}</p>
          <div className="mt-2 min-w-0 text-sm font-semibold text-slate-800 [overflow-wrap:anywhere]">{item.value}</div>
        </div>
      ))}
    </div>
  );
}

function SubmitBar({
  saving,
  error,
  success,
  label = "Simpan",
}: {
  saving: boolean;
  error: string | null;
  success: string | null;
  label?: string;
}) {
  return (
    <div className="flex flex-col gap-3 sm:flex-row sm:items-center">
      <button
        type="submit"
        disabled={saving}
        className="inline-flex min-w-0 items-center justify-center rounded-md bg-blue-600 px-4 py-2 text-center text-sm font-semibold leading-5 text-white transition hover:bg-blue-700 active:scale-[0.98] disabled:opacity-60"
      >
        {saving ? "Menyimpan..." : label}
      </button>
      {error && <span className="text-sm text-red-600">{error}</span>}
      {success && <span className="text-sm text-emerald-700">{success}</span>}
    </div>
  );
}

function RealShell({ children }: { children: React.ReactNode }) {
  return (
    <AppShell>
      <div className="space-y-6">{children}</div>
    </AppShell>
  );
}

export function CustomersLivePage() {
  const customers = useApi<any>("/api/billing/customers", { data: [] });
  const stats = useApi<any>("/api/billing/customers/stats", {});
  const rows = listOf(customers.data);

  return (
    <RealShell>
      <PageHeader
        eyebrow="Pelanggan"
        title="Daftar pelanggan"
        description="Data langsung dari Billing API: paket, status billing, koneksi, dan koordinat pelanggan."
        actions={<Button href="/customers/new">Tambah Pelanggan</Button>}
      />
      <StatGrid
        stats={[
          { label: "Total pelanggan", value: String(stats.data.total ?? rows.length) },
          { label: "Aktif", value: String(stats.data.aktif ?? stats.data.active ?? 0) },
          { label: "Isolir", value: String(stats.data.isolir ?? 0), tone: "amber" },
          { label: "Suspend", value: String(stats.data.suspend ?? 0), tone: "red" },
        ]}
      />
      <Section title="Pelanggan" description="Daftar ini memakai database tenant, bukan mock.">
        <Notice loading={customers.loading} error={customers.error} />
        {!customers.loading && rows.length === 0 ? (
          <EmptyState
            title="Belum ada pelanggan"
            description="Tambahkan paket dulu, lalu buat pelanggan pertama dari form pelanggan."
            action={<Button href="/customers/new">Tambah Pelanggan</Button>}
          />
        ) : (
          <DataTable
            columns={["ID", "Nama", "Telepon", "Paket", "Jatuh Tempo", "Koneksi", "Status"]}
            rows={rows.map((customer) => [
              customer.customer_id_seq ?? customer.id?.slice(0, 8),
              <a key={customer.id} href={`/customers/${customer.id}`} className="font-semibold text-blue-700">
                {customer.name}
              </a>,
              customer.phone ?? "-",
              customer.package_name ?? "-",
              customer.due_date ? `Tanggal ${customer.due_date}` : "-",
              customer.connection_method ?? "-",
              <StatusBadge key={`${customer.id}-status`} status={customer.status ?? "pending"} />,
            ])}
          />
        )}
      </Section>
    </RealShell>
  );
}

export function CustomerFormLivePage() {
  const packages = useApi<any>("/api/billing/packages?page_size=50", { data: [] });
  const areas = useApi<any>("/api/billing/areas?page_size=50", { data: [] });
  const packageRows = listOf(packages.data);
  const areaRows = listOf(areas.data);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  const onSubmit = useCallback(async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    setSaving(true);
    setError(null);
    setSuccess(null);
    try {
      await apiSend("/api/billing/customers", "POST", {
        name: String(form.get("name") || ""),
        phone: String(form.get("phone") || ""),
        email: String(form.get("email") || ""),
        address: String(form.get("address") || ""),
        area_id: String(form.get("area_id") || "") || undefined,
        latitude: Number(form.get("latitude") || -6.2),
        longitude: Number(form.get("longitude") || 106.816),
        package_id: String(form.get("package_id") || ""),
        activation_date: String(form.get("activation_date") || todayDate()),
        due_date: Number(form.get("due_date") || 10),
        connection_method: String(form.get("connection_method") || "pppoe"),
        pppoe_username: String(form.get("pppoe_username") || ""),
        pppoe_password: String(form.get("pppoe_password") || ""),
        odp_port: String(form.get("odp_port") || ""),
        notes: String(form.get("notes") || ""),
      });
      event.currentTarget.reset();
      setSuccess("Pelanggan berhasil dibuat.");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Gagal membuat pelanggan");
    } finally {
      setSaving(false);
    }
  }, []);

  return (
    <RealShell>
      <PageHeader
        eyebrow="Pelanggan"
        title="Tambah pelanggan"
        description="Form ini langsung menyimpan ke Billing API tenant aktif."
      />
      <Section title="Data pelanggan">
        <form onSubmit={onSubmit} className="grid gap-4 lg:grid-cols-2">
          <FormField label="Nama"><TextInput name="name" required placeholder="Nama pelanggan" /></FormField>
          <FormField label="Telepon"><TextInput name="phone" required placeholder="+6281234567890" /></FormField>
          <FormField label="Email"><TextInput name="email" type="email" placeholder="opsional" /></FormField>
          <FormField label="Paket">
            <select name="package_id" required className={selectClass} defaultValue="">
              <option value="">Pilih paket</option>
              {packageRows.map((pkg) => (
                <option key={pkg.id} value={pkg.id}>{pkg.name}</option>
              ))}
            </select>
          </FormField>
          <FormField label="Area">
            <select name="area_id" className={selectClass} defaultValue="">
              <option value="">Tanpa area</option>
              {areaRows.map((area) => (
                <option key={area.id} value={area.id}>{area.name}</option>
              ))}
            </select>
          </FormField>
          <FormField label="Metode koneksi">
            <select name="connection_method" className={selectClass} defaultValue="pppoe">
              <option value="pppoe">PPPoE</option>
              <option value="static">Static</option>
              <option value="dhcp_binding">DHCP Binding</option>
              <option value="hotspot">Hotspot</option>
            </select>
          </FormField>
          <FormField label="Tanggal aktif"><TextInput name="activation_date" type="date" defaultValue={todayDate()} /></FormField>
          <FormField label="Tanggal jatuh tempo"><TextInput name="due_date" type="number" min="1" max="28" defaultValue="10" /></FormField>
          <FormField label="Latitude"><TextInput name="latitude" type="number" step="0.000001" defaultValue="-6.2" /></FormField>
          <FormField label="Longitude"><TextInput name="longitude" type="number" step="0.000001" defaultValue="106.816" /></FormField>
          <FormField label="Username PPPoE"><TextInput name="pppoe_username" placeholder="opsional, auto jika kosong" /></FormField>
          <FormField label="Password PPPoE"><TextInput name="pppoe_password" placeholder="opsional, auto jika kosong" /></FormField>
          <div className="lg:col-span-2">
            <FormField label="Alamat"><textarea name="address" required className={textAreaClass} rows={3} /></FormField>
          </div>
          <FormField label="ODP / Port"><TextInput name="odp_port" placeholder="ODP-01 / Port 3" /></FormField>
          <FormField label="Catatan"><TextInput name="notes" placeholder="opsional" /></FormField>
          <div className="lg:col-span-2">
            <SubmitBar saving={saving} error={error} success={success} />
          </div>
        </form>
      </Section>
    </RealShell>
  );
}

export function CustomerDetailLivePage({ id }: { id: string }) {
  const customerState = useApi<any>(`/api/billing/customers/${id}`, {});
  const invoices = useApi<any>(`/api/billing/invoices?customer_id=${id}&page_size=50`, { data: [] });
  const customer = firstOf(customerState.data, ["customer"]);
  const invoiceRows = listOf(invoices.data);
  const [actionState, setActionState] = useState<{ loading: boolean; message: string; error: string }>({
    loading: false,
    message: "",
    error: "",
  });

  const runCustomerAction = useCallback(
    async (action: "activate" | "isolir") => {
      setActionState({ loading: true, message: "", error: "" });
      try {
        await apiSend(`/api/billing/customers/${id}/${action}`, "POST", {});
        customerState.reload();
        setActionState({
          loading: false,
          message: action === "activate" ? "Pelanggan berhasil diaktifkan." : "Pelanggan berhasil diisolir.",
          error: "",
        });
      } catch (err) {
        setActionState({
          loading: false,
          message: "",
          error: err instanceof Error ? err.message : "Aksi pelanggan gagal",
        });
      }
    },
    [customerState, id],
  );

  return (
    <RealShell>
      <PageHeader
        eyebrow="Pelanggan"
        title={customer.name ?? "Detail pelanggan"}
        description="Detail ini dibaca langsung dari Billing API, termasuk status layanan dan invoice pelanggan."
        actions={<Button href="/customers">Kembali</Button>}
      />
      <Notice loading={customerState.loading} error={customerState.error} />
      {!customerState.loading && !customer.id ? (
        <EmptyState title="Pelanggan tidak ditemukan" description="Data tidak tersedia di tenant aktif." />
      ) : (
        <>
          <Section
            title="Profil pelanggan"
            action={
              <div className="flex flex-wrap gap-2">
                <button
                  type="button"
                  disabled={actionState.loading}
                  onClick={() => void runCustomerAction("activate")}
                  className="rounded-md bg-emerald-600 px-3 py-2 text-sm font-semibold text-white disabled:opacity-60"
                >
                  Aktifkan
                </button>
                <button
                  type="button"
                  disabled={actionState.loading}
                  onClick={() => void runCustomerAction("isolir")}
                  className="rounded-md border border-amber-300 bg-amber-50 px-3 py-2 text-sm font-semibold text-amber-800 disabled:opacity-60"
                >
                  Isolir
                </button>
              </div>
            }
          >
            {actionState.error && <p className="mb-4 text-sm text-red-600">{actionState.error}</p>}
            {actionState.message && <p className="mb-4 text-sm text-emerald-700">{actionState.message}</p>}
            <DetailGrid
              items={[
                { label: "ID pelanggan", value: customer.customer_id_seq ?? customer.id },
                { label: "Status", value: <StatusBadge status={customer.status ?? "pending"} /> },
                { label: "Telepon", value: customer.phone ?? "-" },
                { label: "Email", value: customer.email ?? "-" },
                { label: "Paket", value: customer.package_name ?? customer.package_id ?? "-" },
                { label: "Jatuh tempo", value: customer.due_date ? `Tanggal ${customer.due_date}` : "-" },
                { label: "Koneksi", value: customer.connection_method ?? "-" },
                { label: "Username PPPoE", value: customer.pppoe_username ?? "-" },
                { label: "Koordinat", value: customer.latitude != null ? `${customer.latitude}, ${customer.longitude}` : "-" },
              ]}
            />
            <div className="mt-4 rounded-lg border border-slate-200 p-4">
              <p className="text-xs font-semibold uppercase tracking-[0.14em] text-slate-400">Alamat</p>
              <p className="mt-2 text-sm leading-6 text-slate-700">{customer.address ?? "-"}</p>
            </div>
          </Section>

          <Section title="Invoice pelanggan">
            <Notice loading={invoices.loading} error={invoices.error} />
            {invoiceRows.length === 0 && !invoices.loading ? (
              <EmptyState title="Belum ada invoice" description="Buat invoice dari halaman invoice untuk pelanggan ini." />
            ) : (
              <DataTable
                columns={["Nomor", "Periode", "Jatuh tempo", "Total", "Dibayar", "Status"]}
                rows={invoiceRows.map((invoice) => [
                  <a key={invoice.id} href={`/invoices/${invoice.id}`} className="font-semibold text-blue-700">
                    {invoice.invoice_number}
                  </a>,
                  `${invoice.period_month}/${invoice.period_year}`,
                  dateID(invoice.due_date),
                  money(invoice.total_amount),
                  money(invoice.paid_amount),
                  <StatusBadge key={`${invoice.id}-status`} status={invoice.status ?? "belum_bayar"} />,
                ])}
              />
            )}
          </Section>
        </>
      )}
    </RealShell>
  );
}

export function CustomerAreasLivePage() {
  const areas = useApi<any>("/api/billing/areas?page_size=50", { data: [] });
  const rows = listOf(areas.data);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  const onSubmit = useCallback(
    async (event: React.FormEvent<HTMLFormElement>) => {
      event.preventDefault();
      const form = new FormData(event.currentTarget);
      setSaving(true);
      setError(null);
      setSuccess(null);
      try {
        await apiSend("/api/billing/areas", "POST", {
          name: String(form.get("name") || ""),
          description: String(form.get("description") || ""),
        });
        event.currentTarget.reset();
        areas.reload();
        setSuccess("Area berhasil dibuat.");
      } catch (err) {
        setError(err instanceof Error ? err.message : "Gagal membuat area");
      } finally {
        setSaving(false);
      }
    },
    [areas],
  );

  return (
    <RealShell>
      <PageHeader eyebrow="Pelanggan" title="Area pelanggan" description="Master area dari Billing API tenant aktif." />
      <div className="grid gap-6 xl:grid-cols-[0.8fr_1.2fr]">
        <Section title="Tambah area">
          <form onSubmit={onSubmit} className="grid gap-4">
            <FormField label="Nama area"><TextInput name="name" required placeholder="Depok Timur" /></FormField>
            <FormField label="Deskripsi"><textarea name="description" className={textAreaClass} rows={3} /></FormField>
            <SubmitBar saving={saving} error={error} success={success} />
          </form>
        </Section>
        <Section title="Daftar area">
          <Notice loading={areas.loading} error={areas.error} />
          {rows.length === 0 && !areas.loading ? (
            <EmptyState title="Belum ada area" description="Area membantu filter pelanggan, invoice, dan laporan." />
          ) : (
            <DataTable
              columns={["Nama", "Deskripsi", "Pelanggan"]}
              rows={rows.map((area) => [
                area.name,
                area.description ?? "-",
                String(area.customer_count ?? 0),
              ])}
            />
          )}
        </Section>
      </div>
    </RealShell>
  );
}

export function PackagesLivePage() {
  const packages = useApi<any>("/api/billing/packages?page_size=50", { data: [] });
  const rows = listOf(packages.data);

  return (
    <RealShell>
      <PageHeader
        eyebrow="Paket"
        title="Paket internet"
        description="Paket PPPoE dan voucher dari database Billing API."
        actions={<Button href="/packages/new">Tambah Paket</Button>}
      />
      <Section title="Daftar paket">
        <Notice loading={packages.loading} error={packages.error} />
        {!packages.loading && rows.length === 0 ? (
          <EmptyState title="Belum ada paket" description="Buat paket PPPoE pertama agar pelanggan bisa ditambahkan." action={<Button href="/packages/new">Tambah Paket</Button>} />
        ) : (
          <DataTable
            columns={["Nama", "Tipe", "Bandwidth", "Harga", "Pelanggan", "Status"]}
            rows={rows.map((pkg) => [
              pkg.name,
              pkg.type,
              `${pkg.download_mbps}/${pkg.upload_mbps} Mbps`,
              pkg.type === "voucher" ? money(pkg.sell_price) : money(pkg.monthly_price),
              String(pkg.customer_count ?? 0),
              <StatusBadge key={pkg.id} status={pkg.is_active ? "aktif" : "nonaktif"} />,
            ])}
          />
        )}
      </Section>
    </RealShell>
  );
}

export function PackageFormLivePage() {
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [type, setType] = useState("pppoe");

  const onSubmit = useCallback(async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    const packageType = String(form.get("type") || "pppoe");
    const monthlyPrice = Number(form.get("monthly_price") || 0);
    const sellPrice = Number(form.get("sell_price") || 0);
    setSaving(true);
    setError(null);
    setSuccess(null);
    try {
      await apiSend("/api/billing/packages", "POST", {
        type: packageType,
        name: String(form.get("name") || ""),
        description: String(form.get("description") || ""),
        download_mbps: Number(form.get("download_mbps") || 1),
        upload_mbps: Number(form.get("upload_mbps") || 1),
        bandwidth_type: "shared",
        quota_type: packageType === "voucher" ? "quota" : "unlimited",
        monthly_price: packageType === "pppoe" ? monthlyPrice : undefined,
        installation_fee: Number(form.get("installation_fee") || 0),
        sell_price: packageType === "voucher" ? sellPrice : undefined,
        reseller_price: packageType === "voucher" ? Number(form.get("reseller_price") || 0) : undefined,
        duration_value: packageType === "voucher" ? Number(form.get("duration_value") || 1) : undefined,
        duration_unit: packageType === "voucher" ? String(form.get("duration_unit") || "days") : undefined,
        shared_users: Number(form.get("shared_users") || 1),
        mikrotik_profile_name: String(form.get("mikrotik_profile_name") || ""),
      });
      event.currentTarget.reset();
      setType("pppoe");
      setSuccess("Paket berhasil dibuat.");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Gagal membuat paket");
    } finally {
      setSaving(false);
    }
  }, []);

  return (
    <RealShell>
      <PageHeader eyebrow="Paket" title="Tambah paket internet" description="Menyimpan langsung ke Billing API." />
      <Section title="Konfigurasi paket">
        <form onSubmit={onSubmit} className="grid gap-4 lg:grid-cols-2">
          <FormField label="Tipe paket">
            <select name="type" value={type} onChange={(event) => setType(event.target.value)} className={selectClass}>
              <option value="pppoe">PPPoE bulanan</option>
              <option value="voucher">Voucher hotspot</option>
            </select>
          </FormField>
          <FormField label="Nama paket"><TextInput name="name" required placeholder="Home 30 Mbps" /></FormField>
          <FormField label="Download Mbps"><TextInput name="download_mbps" type="number" min="1" defaultValue="30" /></FormField>
          <FormField label="Upload Mbps"><TextInput name="upload_mbps" type="number" min="1" defaultValue="10" /></FormField>
          <FormField label="Harga bulanan"><TextInput name="monthly_price" type="number" min="0" defaultValue="150000" disabled={type === "voucher"} /></FormField>
          <FormField label="Biaya instalasi"><TextInput name="installation_fee" type="number" min="0" defaultValue="0" /></FormField>
          <FormField label="Harga jual voucher"><TextInput name="sell_price" type="number" min="0" defaultValue="5000" disabled={type !== "voucher"} /></FormField>
          <FormField label="Harga reseller"><TextInput name="reseller_price" type="number" min="0" defaultValue="4000" disabled={type !== "voucher"} /></FormField>
          <FormField label="Durasi voucher"><TextInput name="duration_value" type="number" min="1" defaultValue="1" disabled={type !== "voucher"} /></FormField>
          <FormField label="Satuan durasi">
            <select name="duration_unit" className={selectClass} disabled={type !== "voucher"} defaultValue="days">
              <option value="hours">Jam</option>
              <option value="days">Hari</option>
              <option value="weeks">Minggu</option>
              <option value="months">Bulan</option>
            </select>
          </FormField>
          <FormField label="Shared users"><TextInput name="shared_users" type="number" min="1" defaultValue="1" /></FormField>
          <FormField label="MikroTik profile"><TextInput name="mikrotik_profile_name" placeholder="home-30m" /></FormField>
          <div className="lg:col-span-2">
            <FormField label="Deskripsi"><textarea name="description" className={textAreaClass} rows={3} /></FormField>
          </div>
          <div className="lg:col-span-2">
            <SubmitBar saving={saving} error={error} success={success} />
          </div>
        </form>
      </Section>
    </RealShell>
  );
}

export function InvoicesLivePage() {
  const invoices = useApi<any>("/api/billing/invoices?page_size=50", { data: [] });
  const summary = useApi<any>("/api/billing/invoices/summary", {});
  const customers = useApi<any>("/api/billing/customers?page_size=50", { data: [] });
  const rows = listOf(invoices.data);
  const customerRows = listOf(customers.data);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  const onSubmit = useCallback(
    async (event: React.FormEvent<HTMLFormElement>) => {
      event.preventDefault();
      const form = new FormData(event.currentTarget);
      setSaving(true);
      setError(null);
      setSuccess(null);
      try {
        await apiSend("/api/billing/invoices", "POST", {
          customer_id: String(form.get("customer_id") || ""),
          due_date: String(form.get("due_date") || todayDate()),
          items: [
            {
              description: String(form.get("description") || "Tagihan internet bulanan"),
              quantity: Number(form.get("quantity") || 1),
              unit_price: Number(form.get("unit_price") || 0),
            },
          ],
          notes: String(form.get("notes") || ""),
          apply_tax: false,
          apply_credit: true,
        });
        event.currentTarget.reset();
        invoices.reload();
        summary.reload();
        setSuccess("Invoice berhasil dibuat.");
      } catch (err) {
        setError(err instanceof Error ? err.message : "Gagal membuat invoice");
      } finally {
        setSaving(false);
      }
    },
    [invoices, summary],
  );

  return (
    <RealShell>
      <PageHeader eyebrow="Invoice" title="Invoice pelanggan" description="Invoice real dari Billing API." />
      <StatGrid
        stats={[
          { label: "Total invoice", value: String(summary.data.total_invoices ?? rows.length) },
          { label: "Belum lunas", value: String(summary.data.unpaid_count ?? summary.data.belum_bayar ?? 0), tone: "amber" },
          { label: "Terlambat", value: String(summary.data.overdue_count ?? 0), tone: "red" },
          { label: "Outstanding", value: money(summary.data.outstanding_amount ?? 0) },
        ]}
      />
      <Section title="Daftar invoice">
        <Notice loading={invoices.loading} error={invoices.error} />
        {rows.length === 0 && !invoices.loading ? (
          <EmptyState title="Belum ada invoice" description="Invoice akan muncul setelah dibuat manual atau generated oleh billing cycle." />
        ) : (
          <DataTable
            columns={["Nomor", "Pelanggan", "Periode", "Jatuh tempo", "Total", "Dibayar", "Status"]}
            rows={rows.map((invoice) => [
              invoice.invoice_number,
              invoice.customer_name ?? invoice.customer_id_seq ?? "-",
              `${invoice.period_month}/${invoice.period_year}`,
              dateID(invoice.due_date),
              money(invoice.total_amount),
              money(invoice.paid_amount),
              <StatusBadge key={invoice.id} status={invoice.status ?? "belum_bayar"} />,
            ])}
          />
        )}
      </Section>
      <Section title="Buat invoice manual" description="Pelanggan harus berstatus aktif sebelum invoice bisa dibuat.">
        <form onSubmit={onSubmit} className="grid gap-4 lg:grid-cols-4">
          <FormField label="Pelanggan">
            <select name="customer_id" required className={selectClass} defaultValue="">
              <option value="">Pilih pelanggan</option>
              {customerRows.map((customer) => (
                <option key={customer.id} value={customer.id}>
                  {customer.customer_id_seq ?? customer.id.slice(0, 8)} - {customer.name} ({customer.status})
                </option>
              ))}
            </select>
          </FormField>
          <FormField label="Jatuh tempo"><TextInput name="due_date" type="date" defaultValue={todayDate()} /></FormField>
          <FormField label="Deskripsi"><TextInput name="description" defaultValue="Tagihan internet bulanan" /></FormField>
          <FormField label="Nominal"><TextInput name="unit_price" type="number" min="1" defaultValue="150000" /></FormField>
          <FormField label="Qty"><TextInput name="quantity" type="number" min="1" defaultValue="1" /></FormField>
          <div className="lg:col-span-3">
            <FormField label="Catatan"><TextInput name="notes" placeholder="opsional" /></FormField>
          </div>
          <div className="lg:col-span-4">
            <SubmitBar saving={saving} error={error} success={success} label="Buat invoice" />
          </div>
        </form>
      </Section>
    </RealShell>
  );
}

export function InvoiceDetailLivePage({ id }: { id: string }) {
  const invoiceState = useApi<any>(`/api/billing/invoices/${id}?include=audit_logs`, {});
  const invoice = firstOf(invoiceState.data, ["invoice"]);
  const items = listOf(invoiceState.data?.items ?? invoice.items ?? []);
  const payments = listOf(invoiceState.data?.payments ?? invoice.payments ?? []);
  const auditLogs = listOf(invoiceState.data?.audit_logs ?? invoice.audit_logs ?? []);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  const onSubmit = useCallback(
    async (event: React.FormEvent<HTMLFormElement>) => {
      event.preventDefault();
      const form = new FormData(event.currentTarget);
      setSaving(true);
      setError(null);
      setSuccess(null);
      try {
        await apiSend(`/api/billing/invoices/${id}/payment`, "POST", {
          amount: Number(form.get("amount") || 0),
          payment_method: String(form.get("payment_method") || "tunai"),
          payment_date: String(form.get("payment_date") || todayDate()),
          reference_number: String(form.get("reference_number") || ""),
          notes: String(form.get("notes") || ""),
        });
        event.currentTarget.reset();
        invoiceState.reload();
        setSuccess("Pembayaran invoice berhasil dicatat.");
      } catch (err) {
        setError(err instanceof Error ? err.message : "Gagal mencatat pembayaran");
      } finally {
        setSaving(false);
      }
    },
    [id, invoiceState],
  );

  return (
    <RealShell>
      <PageHeader
        eyebrow="Invoice"
        title={invoice.invoice_number ?? "Detail invoice"}
        description="Detail invoice, item, pembayaran, dan audit log dari Billing API."
        actions={<Button href="/invoices">Kembali</Button>}
      />
      <Notice loading={invoiceState.loading} error={invoiceState.error} />
      {!invoiceState.loading && !invoice.id ? (
        <EmptyState title="Invoice tidak ditemukan" description="Data tidak tersedia di tenant aktif." />
      ) : (
        <>
          <Section title="Ringkasan invoice">
            <DetailGrid
              items={[
                { label: "Pelanggan", value: invoice.customer_name ?? invoice.customer_id_seq ?? "-" },
                { label: "Periode", value: `${invoice.period_month ?? "-"}/${invoice.period_year ?? "-"}` },
                { label: "Jatuh tempo", value: dateID(invoice.due_date) },
                { label: "Subtotal", value: money(invoice.subtotal) },
                { label: "Total", value: money(invoice.total_amount) },
                { label: "Dibayar", value: money(invoice.paid_amount) },
                { label: "Status", value: <StatusBadge status={invoice.status ?? "belum_bayar"} /> },
                { label: "Denda", value: money(invoice.penalty_amount) },
                { label: "Kredit dipakai", value: money(invoice.credit_applied) },
              ]}
            />
          </Section>

          <Section title="Item invoice">
            <DataTable
              columns={["Deskripsi", "Qty", "Harga", "Jumlah"]}
              rows={items.map((item) => [
                item.description,
                String(item.quantity ?? 1),
                money(item.unit_price),
                money(item.amount ?? Number(item.quantity ?? 1) * Number(item.unit_price ?? 0)),
              ])}
            />
          </Section>

          <Section title="Catat pembayaran">
            <form onSubmit={onSubmit} className="grid gap-4 lg:grid-cols-4">
              <FormField label="Nominal"><TextInput name="amount" type="number" min="1" defaultValue={String(Math.max(Number(invoice.total_amount ?? 0) - Number(invoice.paid_amount ?? 0), 0))} /></FormField>
              <FormField label="Metode">
                <select name="payment_method" className={selectClass} defaultValue="tunai">
                  <option value="tunai">Tunai</option>
                  <option value="transfer">Transfer</option>
                  <option value="xendit">Xendit</option>
                  <option value="midtrans">Midtrans</option>
                  <option value="lainnya">Lainnya</option>
                </select>
              </FormField>
              <FormField label="Tanggal"><TextInput name="payment_date" type="date" defaultValue={todayDate()} /></FormField>
              <FormField label="Referensi"><TextInput name="reference_number" placeholder="opsional" /></FormField>
              <div className="lg:col-span-4">
                <FormField label="Catatan"><TextInput name="notes" placeholder="opsional" /></FormField>
              </div>
              <div className="lg:col-span-4">
                <SubmitBar saving={saving} error={error} success={success} label="Catat pembayaran" />
              </div>
            </form>
          </Section>

          <Section title="Riwayat pembayaran">
            {payments.length === 0 ? (
              <EmptyState title="Belum ada pembayaran" description="Transaksi pembayaran invoice ini akan muncul di sini." />
            ) : (
              <DataTable
                columns={["Tanggal", "Metode", "Nominal", "Referensi", "Receipt"]}
                rows={payments.map((payment) => [
                  dateID(payment.payment_date),
                  payment.payment_method ?? "-",
                  money(payment.amount),
                  payment.reference_number ?? "-",
                  payment.receipt_number ?? "-",
                ])}
              />
            )}
          </Section>

          <Section title="Audit log">
            {auditLogs.length === 0 ? (
              <EmptyState title="Belum ada audit log" description="Aktivitas invoice akan tercatat otomatis." />
            ) : (
              <DataTable
                columns={["Waktu", "Aksi", "Aktor"]}
                rows={auditLogs.map((log) => [
                  dateID(log.created_at),
                  log.action ?? "-",
                  log.actor_name ?? log.actor_id ?? "-",
                ])}
              />
            )}
          </Section>
        </>
      )}
    </RealShell>
  );
}

export function PaymentsLivePage() {
  const payments = useApi<any>("/api/billing/payments?page_size=50", { data: [] });
  const summary = useApi<any>("/api/billing/payments/summary", {});
  const invoices = useApi<any>("/api/billing/invoices?page_size=50", { data: [] });
  const rows = listOf(payments.data);
  const invoiceRows = listOf(invoices.data).filter((invoice) => !["lunas", "batal"].includes(String(invoice.status)));
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  const onSubmit = useCallback(
    async (event: React.FormEvent<HTMLFormElement>) => {
      event.preventDefault();
      const form = new FormData(event.currentTarget);
      const invoiceID = String(form.get("invoice_id") || "");
      setSaving(true);
      setError(null);
      setSuccess(null);
      try {
        await apiSend(`/api/billing/invoices/${invoiceID}/payment`, "POST", {
          amount: Number(form.get("amount") || 0),
          payment_method: String(form.get("payment_method") || "tunai"),
          payment_date: String(form.get("payment_date") || todayDate()),
          reference_number: String(form.get("reference_number") || ""),
          notes: String(form.get("notes") || ""),
        });
        event.currentTarget.reset();
        payments.reload();
        summary.reload();
        invoices.reload();
        setSuccess("Pembayaran berhasil dicatat.");
      } catch (err) {
        setError(err instanceof Error ? err.message : "Gagal mencatat pembayaran");
      } finally {
        setSaving(false);
      }
    },
    [invoices, payments, summary],
  );

  return (
    <RealShell>
      <PageHeader eyebrow="Pembayaran" title="Pembayaran" description="Catatan pembayaran real dari Billing API." />
      <StatGrid
        stats={[
          { label: "Transaksi", value: String(summary.data.total_payments ?? rows.length) },
          { label: "Total diterima", value: money(summary.data.total_amount ?? 0) },
          { label: "Hari ini", value: money(summary.data.today_amount ?? 0) },
          { label: "Void", value: String(summary.data.void_count ?? 0), tone: "red" },
        ]}
      />
      <Section title="Riwayat pembayaran">
        <Notice loading={payments.loading} error={payments.error} />
        {rows.length === 0 && !payments.loading ? (
          <EmptyState title="Belum ada pembayaran" description="Pembayaran manual dan gateway akan muncul di sini." />
        ) : (
          <DataTable
            columns={["Tanggal", "Invoice", "Metode", "Nominal", "Referensi", "Petugas"]}
            rows={rows.map((payment) => [
              dateID(payment.payment_date),
              payment.invoice_number ?? payment.invoice_id?.slice(0, 8),
              payment.payment_method ?? "-",
              money(payment.amount),
              payment.reference_number ?? "-",
              payment.recorded_by_name ?? "-",
            ])}
          />
        )}
      </Section>
      <Section title="Catat pembayaran invoice">
        <form onSubmit={onSubmit} className="grid gap-4 lg:grid-cols-4">
          <FormField label="Invoice">
            <select name="invoice_id" required className={selectClass} defaultValue="">
              <option value="">Pilih invoice</option>
              {invoiceRows.map((invoice) => (
                <option key={invoice.id} value={invoice.id}>
                  {invoice.invoice_number} - {invoice.customer_name ?? invoice.customer_id_seq ?? "-"} - {money(Number(invoice.total_amount ?? 0) - Number(invoice.paid_amount ?? 0))}
                </option>
              ))}
            </select>
          </FormField>
          <FormField label="Nominal"><TextInput name="amount" type="number" min="1" defaultValue="150000" /></FormField>
          <FormField label="Metode">
            <select name="payment_method" className={selectClass} defaultValue="tunai">
              <option value="tunai">Tunai</option>
              <option value="transfer">Transfer</option>
              <option value="xendit">Xendit</option>
              <option value="midtrans">Midtrans</option>
              <option value="lainnya">Lainnya</option>
            </select>
          </FormField>
          <FormField label="Tanggal"><TextInput name="payment_date" type="date" defaultValue={todayDate()} /></FormField>
          <FormField label="Referensi"><TextInput name="reference_number" placeholder="opsional" /></FormField>
          <div className="lg:col-span-3">
            <FormField label="Catatan"><TextInput name="notes" placeholder="opsional" /></FormField>
          </div>
          <div className="lg:col-span-4">
            <SubmitBar saving={saving} error={error} success={success} label="Catat pembayaran" />
          </div>
        </form>
      </Section>
    </RealShell>
  );
}

export function VouchersLivePage() {
  const vouchers = useApi<any>("/api/billing/vouchers?page_size=50", { data: [] });
  const packages = useApi<any>("/api/billing/packages?page_size=50", { data: [] });
  const rows = listOf(vouchers.data);
  const voucherPackages = listOf(packages.data).filter((pkg) => pkg.type === "voucher");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  const onSubmit = useCallback(
    async (event: React.FormEvent<HTMLFormElement>) => {
      event.preventDefault();
      const form = new FormData(event.currentTarget);
      setSaving(true);
      setError(null);
      setSuccess(null);
      try {
        await apiSend("/api/billing/vouchers/generate", "POST", {
          package_id: String(form.get("package_id") || ""),
          quantity: Number(form.get("quantity") || 1),
          code_format: String(form.get("code_format") || "mixed"),
          code_length: Number(form.get("code_length") || 8),
          prefix: String(form.get("prefix") || ""),
        });
        event.currentTarget.reset();
        vouchers.reload();
        setSuccess("Voucher berhasil digenerate.");
      } catch (err) {
        setError(err instanceof Error ? err.message : "Gagal generate voucher");
      } finally {
        setSaving(false);
      }
    },
    [vouchers],
  );

  return (
    <RealShell>
      <PageHeader eyebrow="Voucher" title="Voucher hotspot" description="Kode voucher dari Billing API." />
      <Section title="Daftar voucher">
        <Notice loading={vouchers.loading} error={vouchers.error} />
        {rows.length === 0 && !vouchers.loading ? (
          <EmptyState title="Belum ada voucher" description="Buat paket tipe voucher, lalu generate kode voucher." />
        ) : (
          <DataTable
            columns={["Kode", "Paket", "Reseller", "Harga", "Status", "Dibuat"]}
            rows={rows.map((voucher) => [
              <span key={voucher.id} className="font-mono font-semibold">{voucher.code}</span>,
              voucher.package_name ?? "-",
              voucher.reseller_name ?? "-",
              money(voucher.sell_price_snapshot ?? 0),
              <StatusBadge key={`${voucher.id}-status`} status={voucher.status ?? "tersedia"} />,
              dateID(voucher.created_at),
            ])}
          />
        )}
      </Section>
      <Section title="Generate voucher">
        <form onSubmit={onSubmit} className="grid gap-4 lg:grid-cols-5">
          <FormField label="Paket voucher">
            <select name="package_id" required className={selectClass} defaultValue="">
              <option value="">Pilih paket</option>
              {voucherPackages.map((pkg) => (
                <option key={pkg.id} value={pkg.id}>{pkg.name}</option>
              ))}
            </select>
          </FormField>
          <FormField label="Jumlah"><TextInput name="quantity" type="number" min="1" defaultValue="10" /></FormField>
          <FormField label="Format">
            <select name="code_format" className={selectClass} defaultValue="mixed">
              <option value="mixed">Mixed</option>
              <option value="digits">Digits</option>
              <option value="letters">Letters</option>
            </select>
          </FormField>
          <FormField label="Panjang kode"><TextInput name="code_length" type="number" min="6" max="16" defaultValue="8" /></FormField>
          <FormField label="Prefix"><TextInput name="prefix" placeholder="IB" /></FormField>
          <div className="lg:col-span-5">
            <SubmitBar saving={saving} error={error} success={success} label="Generate" />
          </div>
        </form>
      </Section>
    </RealShell>
  );
}

export function ResellersLivePage() {
  const resellers = useApi<any>("/api/billing/resellers?page_size=50", { data: [] });
  const rows = listOf(resellers.data);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  const onSubmit = useCallback(
    async (event: React.FormEvent<HTMLFormElement>) => {
      event.preventDefault();
      const form = new FormData(event.currentTarget);
      setSaving(true);
      setError(null);
      setSuccess(null);
      try {
        await apiSend("/api/billing/resellers", "POST", {
          name: String(form.get("name") || ""),
          phone: String(form.get("phone") || ""),
          email: String(form.get("email") || ""),
          address: String(form.get("address") || ""),
          password: String(form.get("password") || ""),
          balance: Number(form.get("balance") || 0),
          daily_purchase_limit: Number(form.get("daily_purchase_limit") || 0),
        });
        event.currentTarget.reset();
        resellers.reload();
        setSuccess("Reseller berhasil dibuat.");
      } catch (err) {
        setError(err instanceof Error ? err.message : "Gagal membuat reseller");
      } finally {
        setSaving(false);
      }
    },
    [resellers],
  );

  return (
    <RealShell>
      <PageHeader eyebrow="Reseller" title="Reseller voucher" description="Akun reseller real dari Billing API." />
      <Section title="Daftar reseller">
        <Notice loading={resellers.loading} error={resellers.error} />
        {rows.length === 0 && !resellers.loading ? (
          <EmptyState title="Belum ada reseller" description="Reseller akan muncul setelah akun reseller dibuat." />
        ) : (
          <DataTable
            columns={["Nama", "Telepon", "Email", "Saldo", "Limit harian", "Voucher terjual", "Status"]}
            rows={rows.map((reseller) => [
              reseller.name,
              reseller.phone,
              reseller.email ?? "-",
              money(reseller.balance),
              String(reseller.daily_purchase_limit ?? 0),
              String(reseller.total_vouchers_sold ?? 0),
              <StatusBadge key={reseller.id} status={reseller.status ?? "aktif"} />,
            ])}
          />
        )}
      </Section>
      <Section title="Tambah reseller">
        <form onSubmit={onSubmit} className="grid gap-4 lg:grid-cols-4">
          <FormField label="Nama"><TextInput name="name" required placeholder="Reseller Depok" /></FormField>
          <FormField label="Telepon"><TextInput name="phone" required placeholder="+6281299990000" /></FormField>
          <FormField label="Email"><TextInput name="email" type="email" placeholder="opsional" /></FormField>
          <FormField label="Password"><TextInput name="password" type="password" required minLength={8} placeholder="minimal 8 karakter" /></FormField>
          <FormField label="Saldo awal"><TextInput name="balance" type="number" min="0" defaultValue="0" /></FormField>
          <FormField label="Limit harian"><TextInput name="daily_purchase_limit" type="number" min="0" defaultValue="0" /></FormField>
          <div className="lg:col-span-2">
            <FormField label="Alamat"><TextInput name="address" placeholder="opsional" /></FormField>
          </div>
          <div className="lg:col-span-4">
            <SubmitBar saving={saving} error={error} success={success} label="Tambah reseller" />
          </div>
        </form>
      </Section>
    </RealShell>
  );
}

export function NotificationsLivePage() {
  const logs = useApi<any>("/api/notification/notifications/logs?page_size=50", { data: [] });
  const templates = useApi<any>("/api/notification/notifications/templates?page_size=50", { data: [] });
  const logRows = listOf(logs.data);
  const templateRows = listOf(templates.data);

  return (
    <RealShell>
      <PageHeader eyebrow="Notifikasi" title="Notifikasi" description="Log dan template dari notification-service." />
      <div className="grid gap-6 xl:grid-cols-[1fr_0.8fr]">
        <Section title="Log pengiriman">
          <Notice loading={logs.loading} error={logs.error} />
          {logRows.length === 0 && !logs.loading ? (
            <EmptyState title="Belum ada log" description="Log muncul setelah billing atau admin mengirim notifikasi." />
          ) : (
            <DataTable
              columns={["Waktu", "Channel", "Tujuan", "Template", "Status"]}
              rows={logRows.map((log) => [
                dateID(log.created_at),
                log.channel ?? "-",
                log.recipient ?? log.phone ?? "-",
                log.template_name ?? log.template_id ?? "-",
                <StatusBadge key={log.id} status={log.status ?? "pending"} />,
              ])}
            />
          )}
        </Section>
        <Section title="Template">
          <Notice loading={templates.loading} error={templates.error} />
          {templateRows.length === 0 && !templates.loading ? (
            <EmptyState title="Belum ada template" description="Template default bisa dibuat dari notification-service seed atau form template." />
          ) : (
            <div className="divide-y divide-slate-100">
              {templateRows.map((template) => (
                <div key={template.id} className="py-3">
                  <div className="flex items-center justify-between gap-3">
                    <p className="font-semibold text-slate-900">{template.name}</p>
                    <StatusBadge status={template.is_active === false ? "nonaktif" : "aktif"} />
                  </div>
                  <p className="mt-1 text-sm text-slate-500">{template.category ?? template.event_type ?? "-"}</p>
                </div>
              ))}
            </div>
          )}
        </Section>
      </div>
    </RealShell>
  );
}

export function OltNewLivePage() {
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  const onSubmit = useCallback(async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    setSaving(true);
    setError(null);
    setSuccess(null);
    try {
      await apiSend("/api/network-service/olt/devices", "POST", {
        name: String(form.get("name") || ""),
        host: String(form.get("host") || ""),
        snmp_version: String(form.get("snmp_version") || "v2c"),
        snmp_port: Number(form.get("snmp_port") || 161),
        snmp_community: String(form.get("snmp_community") || "public"),
        cli_protocol: String(form.get("cli_protocol") || "telnet"),
        cli_port: Number(form.get("cli_port") || 23),
        cli_username: String(form.get("cli_username") || ""),
        cli_password: String(form.get("cli_password") || ""),
        cli_enable_password: String(form.get("cli_enable_password") || ""),
        health_check_interval_sec: Number(form.get("health_check_interval_sec") || 300),
        notes: String(form.get("notes") || ""),
      });
      event.currentTarget.reset();
      setSuccess("OLT berhasil didaftarkan. Test SNMP/CLI tetap dijalankan manual dari detail OLT.");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Gagal menambahkan OLT");
    } finally {
      setSaving(false);
    }
  }, []);

  return (
    <RealShell>
      <PageHeader eyebrow="OLT" title="Tambah OLT" description="Mendaftarkan perangkat ke database network-service tanpa melakukan test koneksi otomatis." actions={<Button href="/olt">Kembali</Button>} />
      <Section title="Koneksi OLT">
        <form onSubmit={onSubmit} className="grid gap-4 lg:grid-cols-2">
          <FormField label="Nama OLT"><TextInput name="name" required placeholder="OLT POP Depok" /></FormField>
          <FormField label="Host / IP"><TextInput name="host" required placeholder="192.168.10.2" /></FormField>
          <FormField label="SNMP version">
            <select name="snmp_version" className={selectClass} defaultValue="v2c">
              <option value="v2c">v2c</option>
              <option value="v3">v3</option>
            </select>
          </FormField>
          <FormField label="SNMP port"><TextInput name="snmp_port" type="number" min="1" max="65535" defaultValue="161" /></FormField>
          <FormField label="SNMP community"><TextInput name="snmp_community" defaultValue="public" /></FormField>
          <FormField label="CLI protocol">
            <select name="cli_protocol" className={selectClass} defaultValue="telnet">
              <option value="telnet">Telnet</option>
              <option value="ssh">SSH</option>
            </select>
          </FormField>
          <FormField label="CLI port"><TextInput name="cli_port" type="number" min="1" max="65535" defaultValue="23" /></FormField>
          <FormField label="CLI username"><TextInput name="cli_username" required placeholder="admin" /></FormField>
          <FormField label="CLI password"><TextInput name="cli_password" type="password" required /></FormField>
          <FormField label="Enable password"><TextInput name="cli_enable_password" type="password" placeholder="opsional" /></FormField>
          <FormField label="Health interval detik"><TextInput name="health_check_interval_sec" type="number" min="60" max="3600" defaultValue="300" /></FormField>
          <FormField label="Catatan"><TextInput name="notes" placeholder="opsional" /></FormField>
          <div className="lg:col-span-2">
            <SubmitBar saving={saving} error={error} success={success} label="Simpan OLT" />
          </div>
        </form>
      </Section>
    </RealShell>
  );
}

export function OltLivePage() {
  const olts = useApi<any>("/api/network-service/olt/devices?page_size=50", { data: [] });
  const summary = useApi<any>("/api/network-service/olt/summary", {});
  const rows = listOf(olts.data);
  const totalOlts = summary.data.total ?? summary.data.total_olts ?? rows.length;
  const onlineOlts = summary.data.online ?? summary.data.online_count ?? 0;
  const offlineOlts = summary.data.offline ?? summary.data.offline_count ?? 0;
  const activeAlarms = summary.data.active_alarms ?? summary.data.active_alarm_count ?? 0;

  return (
    <RealShell>
      <PageHeader eyebrow="OLT" title="OLT multi-brand" description="Data OLT dari network-service." actions={<Button href="/olt/new">Tambah OLT</Button>} />
      <StatGrid
        stats={[
          { label: "OLT", value: String(totalOlts) },
          { label: "Online", value: String(onlineOlts), tone: "green" },
          { label: "Offline", value: String(offlineOlts), tone: "red" },
          { label: "Alarm aktif", value: String(activeAlarms), tone: "amber" },
        ]}
      />
      <Section title="Daftar OLT">
        <Notice loading={olts.loading} error={olts.error} />
        {rows.length === 0 && !olts.loading ? (
          <EmptyState title="Belum ada OLT" description="Tambahkan perangkat OLT saat OLT fisik sudah siap diintegrasikan." action={<Button href="/olt/new">Tambah OLT</Button>} />
        ) : (
          <DataTable
            columns={["Nama", "Brand", "IP", "Lokasi", "Status", "Terakhir sync"]}
            rows={rows.map((olt) => [
              <a key={olt.id} href={`/olt/${olt.id}`} className="font-semibold text-blue-700">{olt.name}</a>,
              `${olt.brand ?? "-"} ${olt.model ?? ""}`.trim(),
              olt.host ?? olt.ip_address ?? "-",
              olt.location ?? "-",
              <StatusBadge key={`${olt.id}-status`} status={olt.status ?? "unknown"} />,
              dateID(olt.last_sync_at ?? olt.updated_at),
            ])}
          />
        )}
      </Section>
    </RealShell>
  );
}

export function OltDetailLivePage({ id }: { id: string }) {
  const oltState = useApi<any>(`/api/network-service/olt/devices/${id}`, {});
  const alarms = useApi<any>(`/api/network-service/olt/devices/${id}/alarms`, { data: [] });
  const olt = firstOf(oltState.data, ["olt", "device"]);
  const alarmRows = listOf(alarms.data);

  return (
    <RealShell>
      <PageHeader
        eyebrow="OLT"
        title={olt.name ?? "Detail OLT"}
        description="Detail perangkat OLT dari network-service. Endpoint monitoring tetap tampil sebagai aksi manual."
        actions={<Button href="/olt">Kembali</Button>}
      />
      <Notice loading={oltState.loading} error={oltState.error} />
      {!oltState.loading && !olt.id ? (
        <EmptyState title="OLT tidak ditemukan" description="Data tidak tersedia di tenant aktif." />
      ) : (
        <>
          <Section title="Profil perangkat">
            <DetailGrid
              items={[
                { label: "Status", value: <StatusBadge status={olt.status ?? "unknown"} /> },
                { label: "Host", value: olt.host ?? olt.ip_address ?? "-" },
                { label: "Brand", value: `${olt.brand ?? "-"} ${olt.model ?? ""}`.trim() },
                { label: "SNMP", value: `${olt.snmp_version ?? "-"} :${olt.snmp_port ?? 161}` },
                { label: "CLI", value: `${olt.cli_protocol ?? "-"} :${olt.cli_port ?? "-"}` },
                { label: "Interval health", value: `${olt.health_check_interval_sec ?? "-"} detik` },
                { label: "Last sync", value: dateID(olt.last_sync_at ?? olt.updated_at) },
                { label: "Lokasi", value: olt.location ?? "-" },
                { label: "Catatan", value: olt.notes ?? "-" },
              ]}
            />
          </Section>
          <Section title="Alarm OLT">
            <Notice loading={alarms.loading} error={alarms.error} />
            {alarmRows.length === 0 && !alarms.loading ? (
              <EmptyState title="Belum ada alarm" description="Alarm akan muncul setelah monitoring OLT berjalan." />
            ) : (
              <DataTable
                columns={["Waktu", "Severity", "Port", "Pesan", "Status"]}
                rows={alarmRows.map((alarm) => [
                  dateID(alarm.created_at ?? alarm.occurred_at),
                  alarm.severity ?? "-",
                  alarm.port ?? alarm.pon_port ?? "-",
                  alarm.message ?? alarm.description ?? "-",
                  <StatusBadge key={alarm.id} status={alarm.status ?? "aktif"} />,
                ])}
              />
            )}
          </Section>
        </>
      )}
    </RealShell>
  );
}

export function OdpLivePage() {
  const odps = useApi<any>("/api/network-service/olt/odp?page_size=50", { data: [] });
  const rows = listOf(odps.data);

  return (
    <RealShell>
      <PageHeader eyebrow="OLT" title="ODP / Splitter" description="Data ODP dari network-service." />
      <Section title="Daftar ODP">
        <Notice loading={odps.loading} error={odps.error} />
        {rows.length === 0 && !odps.loading ? (
          <EmptyState title="Belum ada ODP" description="ODP akan dipakai oleh provisioning dan peta jaringan." />
        ) : (
          <DataTable
            columns={["Nama", "OLT", "PON", "Splitter", "Terpakai", "Koordinat", "Status"]}
            rows={rows.map((odp) => [
              odp.name,
              odp.olt_name ?? odp.olt_id?.slice(0, 8) ?? "-",
              odp.pon_port ?? "-",
              odp.splitter_type ?? "-",
              `${odp.used_ports ?? 0}/${odp.total_ports ?? odp.capacity ?? "-"}`,
              odp.latitude != null ? `${odp.latitude}, ${odp.longitude}` : "-",
              <StatusBadge key={odp.id} status={odp.status ?? "aktif"} />,
            ])}
          />
        )}
      </Section>
    </RealShell>
  );
}

export function ProvisioningLivePage() {
  const onts = useApi<any>("/api/network-service/olt/provisioning/onts?page_size=50", { data: [] });
  const rows = listOf(onts.data);

  return (
    <RealShell>
      <PageHeader eyebrow="OLT" title="Provisioning ONT" description="ONT dan audit provisioning dari network-service." />
      <Section title="ONT terdaftar">
        <Notice loading={onts.loading} error={onts.error} />
        {rows.length === 0 && !onts.loading ? (
          <EmptyState title="Belum ada ONT" description="ONT akan muncul setelah provisioning berjalan." />
        ) : (
          <DataTable
            columns={["Serial", "Pelanggan", "OLT", "PON", "Signal", "Status"]}
            rows={rows.map((ont) => [
              ont.serial_number ?? ont.sn ?? "-",
              ont.customer_name ?? ont.customer_id ?? "-",
              ont.olt_name ?? ont.olt_id?.slice(0, 8) ?? "-",
              ont.pon_port ?? "-",
              ont.signal_dbm != null ? `${ont.signal_dbm} dBm` : "-",
              <StatusBadge key={ont.id} status={ont.status ?? "pending"} />,
            ])}
          />
        )}
      </Section>
    </RealShell>
  );
}

export function MikrotikVpnLivePage() {
  const tunnels = useApi<any>("/api/network-service/mikrotik/vpn/tunnels?page_size=50", { data: [] });
  const summary = useApi<any>("/api/network-service/mikrotik/vpn/summary", {});
  const routers = useApi<any>("/api/network/mikrotik/routers", { data: [] });
  const rows = listOf(tunnels.data);
  const routerRows = listOf(routers.data);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  const onSubmit = useCallback(
    async (event: React.FormEvent<HTMLFormElement>) => {
      event.preventDefault();
      const form = new FormData(event.currentTarget);
      setSaving(true);
      setError(null);
      setSuccess(null);
      try {
        await apiSend("/api/network-service/mikrotik/vpn/tunnels", "POST", {
          tunnel_name: String(form.get("tunnel_name") || ""),
          protocol: String(form.get("protocol") || "wireguard"),
          router_id: String(form.get("router_id") || "") || undefined,
          notes: String(form.get("notes") || ""),
        });
        event.currentTarget.reset();
        tunnels.reload();
        summary.reload();
        setSuccess("Tunnel VPN berhasil dibuat.");
      } catch (err) {
        setError(err instanceof Error ? err.message : "Gagal membuat tunnel VPN");
      } finally {
        setSaving(false);
      }
    },
    [summary, tunnels],
  );

  return (
    <RealShell>
      <PageHeader eyebrow="MikroTik" title="VPN tunnel" description="Manajemen tunnel VPN dari network-service. Test koneksi dan auto-configure tetap aksi manual." />
      <MikrotikModuleNav />
      <StatGrid
        stats={[
          { label: "Tunnel", value: String(summary.data.total_tunnels ?? rows.length) },
          { label: "Connected", value: String(summary.data.connected_count ?? 0), tone: "green" },
          { label: "Disconnected", value: String(summary.data.disconnected_count ?? 0), tone: "red" },
          { label: "Pending", value: String(summary.data.pending_count ?? 0), tone: "amber" },
        ]}
      />
      <Section title="Daftar tunnel">
        <Notice loading={tunnels.loading} error={tunnels.error} />
        {rows.length === 0 && !tunnels.loading ? (
          <EmptyState title="Belum ada tunnel VPN" description="Buat tunnel untuk menghubungkan router remote secara aman." />
        ) : (
          <DataTable
            columns={["Nama", "Protocol", "Router", "VPN IP", "Endpoint", "Status"]}
            rows={rows.map((tunnel) => [
              tunnel.tunnel_name,
              tunnel.protocol,
              tunnel.router_name ?? tunnel.router_id ?? "-",
              tunnel.vpn_ip ?? "-",
              tunnel.server_endpoint ?? "-",
              <StatusBadge key={tunnel.id} status={tunnel.status ?? "pending"} />,
            ])}
          />
        )}
      </Section>
      <Section title="Buat tunnel">
        <form onSubmit={onSubmit} className="grid gap-4 lg:grid-cols-4">
          <FormField label="Nama tunnel"><TextInput name="tunnel_name" required placeholder="vpn-pop-depok" /></FormField>
          <FormField label="Protocol">
            <select name="protocol" className={selectClass} defaultValue="wireguard">
              <option value="wireguard">WireGuard</option>
              <option value="l2tp_ipsec">L2TP/IPsec</option>
              <option value="sstp">SSTP</option>
              <option value="openvpn">OpenVPN</option>
              <option value="pptp">PPTP</option>
            </select>
          </FormField>
          <FormField label="Router">
            <select name="router_id" className={selectClass} defaultValue="">
              <option value="">Tanpa router</option>
              {routerRows.map((router) => (
                <option key={router.id} value={router.id}>{router.name}</option>
              ))}
            </select>
          </FormField>
          <FormField label="Catatan"><TextInput name="notes" placeholder="opsional" /></FormField>
          <div className="lg:col-span-4">
            <SubmitBar saving={saving} error={error} success={success} label="Buat tunnel" />
          </div>
        </form>
      </Section>
    </RealShell>
  );
}
