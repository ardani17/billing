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
type ModuleCapabilities = { billing_core: boolean; mikrotik: boolean; fiber_network: boolean };

const defaultModules: ModuleCapabilities = {
  billing_core: true,
  mikrotik: false,
  fiber_network: false,
};

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
  const details = Array.isArray(body?.error?.details)
    ? body.error.details
        .map((detail: AnyRecord) => `${detail.field}: ${detail.message}`)
        .join(", ")
    : "";
  if (body?.error?.message) return details ? `${body.error.message} (${details})` : body.error.message;
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

function useTenantModules() {
  const state = useApi<any>("/api/billing/tenant/modules", { modules: defaultModules });
  const modules = state.data?.modules ?? state.data ?? defaultModules;
  return {
    ...state,
    data: {
      billing_core: modules.billing_core !== false,
      mikrotik: modules.mikrotik === true,
      fiber_network: modules.fiber_network === true,
    } satisfies ModuleCapabilities,
  };
}

function packageTypeLabel(type?: string) {
  if (type === "voucher") return "Voucher";
  if (type === "pppoe") return "Bulanan / PPPoE";
  return "Bulanan";
}

function connectionLabel(method?: string) {
  const labels: Record<string, string> = {
    manual: "Manual",
    pppoe: "PPPoE",
    hotspot: "Hotspot",
    dhcp_binding: "DHCP Binding",
    static: "Static IP",
  };
  return labels[method || ""] || method || "-";
}

function Notice({ loading, error }: { loading: boolean; error: string | null }) {
  if (loading) return <p className="text-sm text-slate-500">Memuat data...</p>;
  if (error) return <p className="text-sm text-red-600">{error}</p>;
  return null;
}

function todayDate() {
  return new Date().toISOString().slice(0, 10);
}

function normalizeIndonesianPhone(value: FormDataEntryValue | null) {
  const raw = String(value || "").trim().replace(/[\s().-]/g, "");
  if (raw.startsWith("+62")) return raw;
  if (raw.startsWith("62")) return `+${raw}`;
  if (raw.startsWith("0")) return `+62${raw.slice(1)}`;
  return raw;
}

function dateInputValue(value?: string) {
  if (!value) return todayDate();
  const datePart = String(value).slice(0, 10);
  return /^\d{4}-\d{2}-\d{2}$/.test(datePart) ? datePart : todayDate();
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
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [deleteError, setDeleteError] = useState<string | null>(null);
  const [deleteSuccess, setDeleteSuccess] = useState<string | null>(null);

  const deleteCustomer = useCallback(
    async (customer: AnyRecord) => {
      const confirmName = window.prompt(`Ketik nama pelanggan "${customer.name}" untuk menghapus pelanggan ini.`);
      if (confirmName == null) return;
      if (confirmName !== customer.name) {
        setDeleteError("Nama konfirmasi tidak sama, pelanggan tidak dihapus.");
        setDeleteSuccess(null);
        return;
      }

      setDeletingId(customer.id);
      setDeleteError(null);
      setDeleteSuccess(null);
      try {
        await apiSend(`/api/billing/customers/${customer.id}`, "DELETE", { confirmation_name: confirmName });
        customers.reload();
        stats.reload();
        setDeleteSuccess(`Pelanggan ${customer.name} berhasil dihapus.`);
      } catch (err) {
        setDeleteError(err instanceof Error ? err.message : "Gagal menghapus pelanggan");
      } finally {
        setDeletingId(null);
      }
    },
    [customers, stats],
  );

  return (
    <RealShell>
      <PageHeader
        eyebrow="Pelanggan"
        title="Daftar pelanggan"
        description="Data langsung dari Billing API: paket, status billing, dan data layanan pelanggan."
        actions={
          <>
            <Button variant="secondary" href="/customers/areas">Area Pelanggan</Button>
            <Button href="/customers/new">Tambah Pelanggan</Button>
          </>
        }
      />
      <StatGrid
        stats={[
          { label: "Total pelanggan", value: String(stats.data.total ?? rows.length) },
          { label: "Aktif", value: String(stats.data.aktif ?? stats.data.active ?? 0) },
          { label: "Isolir", value: String(stats.data.isolir ?? 0), tone: "amber" },
          { label: "Suspend", value: String(stats.data.suspend ?? 0), tone: "red" },
        ]}
      />
      <Section
        title="Area pelanggan"
        description="Buat dan kelola master area sebelum memilih area pada form tambah pelanggan."
        action={<Button href="/customers/areas">Buka Area Pelanggan</Button>}
      >
        <div className="rounded-lg border border-blue-100 bg-blue-50 px-4 py-3 text-sm leading-6 text-blue-800">
          Area yang dibuat di sini akan muncul sebagai pilihan pada field Area di halaman tambah pelanggan.
        </div>
      </Section>
      <Section title="Pelanggan" description="Daftar ini memakai database tenant, bukan mock.">
        <Notice loading={customers.loading} error={customers.error} />
        {deleteError && <p className="mb-3 text-sm text-red-600">{deleteError}</p>}
        {deleteSuccess && <p className="mb-3 text-sm text-emerald-700">{deleteSuccess}</p>}
        {!customers.loading && rows.length === 0 ? (
          <EmptyState
            title="Belum ada pelanggan"
            description="Tambahkan paket dulu, lalu buat pelanggan pertama dari form pelanggan."
            action={
              <div className="flex flex-wrap justify-center gap-2">
                <Button variant="secondary" href="/customers/areas">Area Pelanggan</Button>
                <Button href="/customers/new">Tambah Pelanggan</Button>
              </div>
            }
          />
        ) : (
          <DataTable
            columns={["ID", "Nama", "Telepon", "Paket", "Jatuh Tempo", "Koneksi", "Status", "Aksi"]}
            rows={rows.map((customer) => [
              customer.customer_id_seq ?? customer.id?.slice(0, 8),
              <a key={customer.id} href={`/customers/${customer.id}`} className="font-semibold text-blue-700">
                {customer.name}
              </a>,
              customer.phone ?? "-",
              customer.package_name ?? "-",
              customer.due_date ? `Tanggal ${customer.due_date}` : "-",
              connectionLabel(customer.connection_method),
              <StatusBadge key={`${customer.id}-status`} status={customer.status ?? "pending"} />,
              <div key={`${customer.id}-actions`} className="flex flex-wrap justify-end gap-2 lg:justify-start">
                <a href={`/customers/${customer.id}/edit`} className="rounded-md px-3 py-1.5 text-sm font-semibold text-blue-700 hover:bg-blue-50">
                  Edit
                </a>
                <button
                  type="button"
                  onClick={() => void deleteCustomer(customer)}
                  disabled={deletingId === customer.id}
                  className="rounded-md px-3 py-1.5 text-sm font-semibold text-red-600 hover:bg-red-50 disabled:cursor-wait disabled:opacity-60"
                >
                  {deletingId === customer.id ? "Menghapus..." : "Hapus"}
                </button>
              </div>,
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
  const modules = useTenantModules();
  const packageRows = listOf(packages.data);
  const areaRows = listOf(areas.data);
  const canUseMikrotik = modules.data.mikrotik === true;
  const canUseFiber = modules.data.fiber_network === true;
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  const onSubmit = useCallback(async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const formEl = event.currentTarget;
    const form = new FormData(formEl);
    setSaving(true);
    setError(null);
    setSuccess(null);
      try {
      const payload: AnyRecord = {
        name: String(form.get("name") || ""),
        phone: normalizeIndonesianPhone(form.get("phone")),
        email: String(form.get("email") || ""),
        address: String(form.get("address") || ""),
        area_id: String(form.get("area_id") || "") || undefined,
        package_id: String(form.get("package_id") || ""),
        activation_date: String(form.get("activation_date") || todayDate()),
        due_date: Number(form.get("due_date") || 10),
        connection_method: String(form.get("connection_method") || "manual"),
        notes: String(form.get("notes") || ""),
      };
      if (canUseMikrotik) {
        payload.pppoe_username = String(form.get("pppoe_username") || "");
        payload.pppoe_password = String(form.get("pppoe_password") || "");
        payload.mac_address = String(form.get("mac_address") || "");
      }
      if (canUseFiber) {
        const latitude = String(form.get("latitude") || "");
        const longitude = String(form.get("longitude") || "");
        if (latitude) payload.latitude = Number(latitude);
        if (longitude) payload.longitude = Number(longitude);
        payload.odp_port = String(form.get("odp_port") || "");
      }
      const created = await apiSend<AnyRecord>("/api/billing/customers", "POST", payload);
      const createdCustomer = firstOf(created, ["customer"]);
      const createdId = String(createdCustomer.id ?? created.id ?? "");
      const activateNow = form.get("activate_now") === "on";
      if (activateNow && createdId) {
        await apiSend(`/api/billing/customers/${createdId}/activate`, "POST", {});
        await apiSend("/api/billing/invoices/generate-due", "POST", {});
      }
      formEl.reset();
      setSuccess(activateNow ? "Pelanggan berhasil dibuat, diaktifkan, dan tagihan jatuh tempo diproses." : "Pelanggan berhasil dibuat.");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Gagal membuat pelanggan");
    } finally {
      setSaving(false);
    }
  }, [canUseFiber, canUseMikrotik]);

  return (
    <RealShell>
      <PageHeader
        eyebrow="Pelanggan"
        title="Tambah pelanggan"
        description="Form ini menyimpan data pelanggan Billing Core. Field jaringan hanya tampil jika add-on aktif."
      />
      <Section title="Data pelanggan">
        <form onSubmit={onSubmit} className="grid gap-4 lg:grid-cols-2">
          <FormField label="Nama"><TextInput name="name" required placeholder="Nama pelanggan" /></FormField>
          <FormField label="Telepon"><TextInput name="phone" required placeholder="081234567890 atau +6281234567890" /></FormField>
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
            <select name="connection_method" className={selectClass} defaultValue="manual">
              <option value="manual">Manual / Billing saja</option>
              {canUseMikrotik && (
                <>
                  <option value="pppoe">PPPoE</option>
                  <option value="static">Static IP</option>
                  <option value="dhcp_binding">DHCP Binding</option>
                  <option value="hotspot">Hotspot</option>
                </>
              )}
            </select>
          </FormField>
          <FormField label="Tanggal aktif"><TextInput name="activation_date" type="date" defaultValue={todayDate()} /></FormField>
          <FormField label="Tanggal jatuh tempo"><TextInput name="due_date" type="number" min="1" max="28" defaultValue="10" /></FormField>
          <label className="flex items-center gap-3 rounded-lg border border-slate-200 bg-white px-4 py-3 text-sm text-slate-700">
            <input name="activate_now" type="checkbox" defaultChecked className="h-4 w-4 rounded border-slate-300 text-blue-600 focus:ring-blue-500" />
            <span>
              <span className="block font-semibold text-slate-900">Aktifkan setelah simpan</span>
              <span className="block text-xs text-slate-500">Jika jatuh tempo masuk periode billing, invoice langsung diproses.</span>
            </span>
          </label>
          {canUseFiber && (
            <>
              <FormField label="Latitude"><TextInput name="latitude" type="number" step="0.000001" placeholder="-6.2" /></FormField>
              <FormField label="Longitude"><TextInput name="longitude" type="number" step="0.000001" placeholder="106.816" /></FormField>
            </>
          )}
          {canUseMikrotik && (
            <>
              <FormField label="Username PPPoE"><TextInput name="pppoe_username" placeholder="opsional, auto jika kosong" /></FormField>
              <FormField label="Password PPPoE"><TextInput name="pppoe_password" placeholder="opsional, auto jika kosong" /></FormField>
              <FormField label="MAC address"><TextInput name="mac_address" placeholder="AA:BB:CC:DD:EE:FF" /></FormField>
            </>
          )}
          <div className="lg:col-span-2">
            <FormField label="Alamat"><textarea name="address" required className={textAreaClass} rows={3} /></FormField>
          </div>
          {canUseFiber && <FormField label="ODP / Port"><TextInput name="odp_port" placeholder="ODP-01 / Port 3" /></FormField>}
          <FormField label="Catatan"><TextInput name="notes" placeholder="opsional" /></FormField>
          <div className="lg:col-span-2">
            <SubmitBar saving={saving} error={error} success={success} />
          </div>
        </form>
      </Section>
    </RealShell>
  );
}

export function CustomerEditLivePage({ id }: { id: string }) {
  const customerState = useApi<any>(`/api/billing/customers/${id}`, {});
  const packages = useApi<any>("/api/billing/packages?page_size=50", { data: [] });
  const areas = useApi<any>("/api/billing/areas?page_size=50", { data: [] });
  const modules = useTenantModules();
  const customer = firstOf(customerState.data, ["customer"]);
  const packageRows = listOf(packages.data);
  const areaRows = listOf(areas.data);
  const canUseMikrotik = modules.data.mikrotik === true;
  const canUseFiber = modules.data.fiber_network === true;
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  const onSubmit = useCallback(
    async (event: React.FormEvent<HTMLFormElement>) => {
      event.preventDefault();
      const form = new FormData(event.currentTarget);
      const dueDate = Number(form.get("due_date") || customer.due_date || 10);
      const payload: AnyRecord = {
        name: String(form.get("name") || ""),
        phone: normalizeIndonesianPhone(form.get("phone")),
        email: String(form.get("email") || ""),
        address: String(form.get("address") || ""),
        area_id: String(form.get("area_id") || "") || undefined,
        package_id: String(form.get("package_id") || ""),
        activation_date: String(form.get("activation_date") || todayDate()),
        due_date: dueDate,
        connection_method: String(form.get("connection_method") || "manual"),
        notes: String(form.get("notes") || ""),
      };

      if (canUseMikrotik) {
        const pppoePassword = String(form.get("pppoe_password") || "");
        payload.pppoe_username = String(form.get("pppoe_username") || "");
        payload.mac_address = String(form.get("mac_address") || "");
        if (pppoePassword) payload.pppoe_password = pppoePassword;
      }
      if (canUseFiber) {
        const latitude = String(form.get("latitude") || "");
        const longitude = String(form.get("longitude") || "");
        if (latitude) payload.latitude = Number(latitude);
        if (longitude) payload.longitude = Number(longitude);
        payload.odp_port = String(form.get("odp_port") || "");
      }

      setSaving(true);
      setError(null);
      setSuccess(null);
      try {
        await apiSend(`/api/billing/customers/${id}`, "PUT", payload);
        customerState.reload();
        setSuccess("Pelanggan berhasil diperbarui.");
      } catch (err) {
        setError(err instanceof Error ? err.message : "Gagal memperbarui pelanggan");
      } finally {
        setSaving(false);
      }
    },
    [canUseFiber, canUseMikrotik, customer.due_date, customerState, id],
  );

  return (
    <RealShell>
      <PageHeader
        eyebrow="Pelanggan"
        title="Edit pelanggan"
        description="Perbarui data billing pelanggan. Field jaringan hanya aktif jika modul add-on tersedia."
        actions={
          <>
            <Button href={`/customers/${id}`} variant="secondary">Detail</Button>
            <Button href="/customers" variant="secondary">Daftar Pelanggan</Button>
          </>
        }
      />
      <Notice loading={customerState.loading || packages.loading || areas.loading} error={customerState.error || packages.error || areas.error} />
      {!customerState.loading && !customer.id ? (
        <EmptyState title="Pelanggan tidak ditemukan" description="Data pelanggan tidak tersedia di tenant aktif." />
      ) : (
        <Section title="Data pelanggan">
          <form key={customer.id || "customer-edit"} onSubmit={onSubmit} className="grid gap-4 lg:grid-cols-2">
            <FormField label="Nama"><TextInput name="name" required defaultValue={customer.name ?? ""} /></FormField>
            <FormField label="Telepon"><TextInput name="phone" required defaultValue={customer.phone ?? ""} placeholder="081234567890 atau +6281234567890" /></FormField>
            <FormField label="Email"><TextInput name="email" type="email" defaultValue={customer.email ?? ""} placeholder="opsional" /></FormField>
            <FormField label="Paket">
              <select name="package_id" required className={selectClass} defaultValue={customer.package_id ?? ""}>
                <option value="">Pilih paket</option>
                {packageRows.map((pkg) => (
                  <option key={pkg.id} value={pkg.id}>{pkg.name}</option>
                ))}
              </select>
            </FormField>
            <FormField label="Area">
              <select name="area_id" className={selectClass} defaultValue={customer.area_id ?? ""}>
                <option value="">Tanpa area</option>
                {areaRows.map((area) => (
                  <option key={area.id} value={area.id}>{area.name}</option>
                ))}
              </select>
            </FormField>
            <FormField label="Metode koneksi">
              <select name="connection_method" className={selectClass} defaultValue={customer.connection_method ?? "manual"}>
                <option value="manual">Manual / Billing saja</option>
                {canUseMikrotik && (
                  <>
                    <option value="pppoe">PPPoE</option>
                    <option value="static">Static IP</option>
                    <option value="dhcp_binding">DHCP Binding</option>
                    <option value="hotspot">Hotspot</option>
                  </>
                )}
              </select>
            </FormField>
            <FormField label="Tanggal aktif"><TextInput name="activation_date" type="date" defaultValue={dateInputValue(customer.activation_date)} /></FormField>
            <FormField label="Tanggal jatuh tempo"><TextInput name="due_date" type="number" min="1" max="28" defaultValue={customer.due_date ?? 10} /></FormField>
            {canUseFiber && (
              <>
                <FormField label="Latitude"><TextInput name="latitude" type="number" step="0.000001" defaultValue={customer.latitude ?? ""} placeholder="-6.2" /></FormField>
                <FormField label="Longitude"><TextInput name="longitude" type="number" step="0.000001" defaultValue={customer.longitude ?? ""} placeholder="106.816" /></FormField>
              </>
            )}
            {canUseMikrotik && (
              <>
                <FormField label="Username PPPoE"><TextInput name="pppoe_username" defaultValue={customer.pppoe_username ?? ""} placeholder="opsional" /></FormField>
                <FormField label="Password PPPoE"><TextInput name="pppoe_password" placeholder="Kosongkan jika tidak diubah" /></FormField>
                <FormField label="MAC address"><TextInput name="mac_address" defaultValue={customer.mac_address ?? ""} placeholder="AA:BB:CC:DD:EE:FF" /></FormField>
              </>
            )}
            <div className="lg:col-span-2">
              <FormField label="Alamat"><textarea name="address" required className={textAreaClass} rows={3} defaultValue={customer.address ?? ""} /></FormField>
            </div>
            {canUseFiber && <FormField label="ODP / Port"><TextInput name="odp_port" defaultValue={customer.odp_port ?? ""} placeholder="ODP-01 / Port 3" /></FormField>}
            <FormField label="Catatan"><TextInput name="notes" defaultValue={customer.notes ?? ""} placeholder="opsional" /></FormField>
            <div className="lg:col-span-2">
              <SubmitBar saving={saving} error={error} success={success} label="Update Pelanggan" />
            </div>
          </form>
        </Section>
      )}
    </RealShell>
  );
}

export function CustomerDetailLivePage({ id }: { id: string }) {
  const customerState = useApi<any>(`/api/billing/customers/${id}`, {});
  const invoices = useApi<any>(`/api/billing/invoices?customer_id=${id}&page_size=50`, { data: [] });
  const modules = useTenantModules();
  const customer = firstOf(customerState.data, ["customer"]);
  const invoiceRows = listOf(invoices.data);
  const customerStatus = String(customer.status ?? "pending");
  const hasCustomer = Boolean(customer.id);
  const canActivateCustomer = hasCustomer && !["aktif", "berhenti"].includes(customerStatus);
  const canIsolirCustomer = hasCustomer && customerStatus === "aktif";
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
        if (action === "activate") {
          await apiSend("/api/billing/invoices/generate-due", "POST", {});
          invoices.reload();
        }
        customerState.reload();
        setActionState({
          loading: false,
          message: action === "activate" ? "Pelanggan berhasil diaktifkan dan tagihan jatuh tempo diproses." : "Pelanggan berhasil diisolir.",
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
    [customerState, id, invoices],
  );

  const renderCustomerActions = () => (
    <div className="flex w-full flex-col gap-2 sm:w-auto sm:flex-row sm:flex-wrap sm:items-center sm:justify-end">
      <Button href="/customers" variant="secondary">Kembali</Button>
      {hasCustomer && <Button href={`/customers/${id}/edit`} variant="secondary">Edit</Button>}
      <button
        type="button"
        disabled={!canActivateCustomer || actionState.loading}
        onClick={() => void runCustomerAction("activate")}
        className="inline-flex h-10 items-center justify-center rounded-md bg-emerald-600 px-4 text-sm font-semibold text-white transition hover:bg-emerald-700 disabled:cursor-not-allowed disabled:opacity-50"
      >
        {actionState.loading && canActivateCustomer ? "Memproses..." : customerStatus === "aktif" ? "Sudah aktif" : "Aktifkan"}
      </button>
      <button
        type="button"
        disabled={!canIsolirCustomer || actionState.loading}
        onClick={() => void runCustomerAction("isolir")}
        className="inline-flex h-10 items-center justify-center rounded-md border border-amber-300 bg-amber-50 px-4 text-sm font-semibold text-amber-800 transition hover:bg-amber-100 disabled:cursor-not-allowed disabled:opacity-50"
      >
        {actionState.loading && canIsolirCustomer ? "Memproses..." : customerStatus === "isolir" ? "Sudah isolir" : "Isolir"}
      </button>
    </div>
  );

  return (
    <RealShell>
      <PageHeader
        eyebrow="Pelanggan"
        title={customer.name ?? "Detail pelanggan"}
        description="Detail ini dibaca langsung dari Billing API, termasuk status layanan dan invoice pelanggan."
        actions={renderCustomerActions()}
      />
      <Notice loading={customerState.loading} error={customerState.error} />
      {!customerState.loading && !customer.id ? (
        <EmptyState title="Pelanggan tidak ditemukan" description="Data tidak tersedia di tenant aktif." />
      ) : (
        <>
          <Section
            title="Profil pelanggan"
            action={renderCustomerActions()}
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
                { label: "Koneksi", value: connectionLabel(customer.connection_method) },
                ...(modules.data.mikrotik ? [{ label: "Username PPPoE", value: customer.pppoe_username ?? "-" }] : []),
                ...(modules.data.fiber_network
                  ? [{ label: "Koordinat", value: customer.latitude != null ? `${customer.latitude}, ${customer.longitude}` : "-" }]
                  : []),
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
      const formEl = event.currentTarget;
    const form = new FormData(formEl);
      setSaving(true);
      setError(null);
      setSuccess(null);
      try {
        await apiSend("/api/billing/areas", "POST", {
          name: String(form.get("name") || ""),
          description: String(form.get("description") || ""),
        });
        formEl.reset();
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
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [deleteError, setDeleteError] = useState<string | null>(null);
  const [deleteSuccess, setDeleteSuccess] = useState<string | null>(null);

  const deletePackage = useCallback(
    async (pkg: AnyRecord) => {
      const confirmName = window.prompt(`Ketik nama paket "${pkg.name}" untuk menghapus paket ini.`);
      if (confirmName == null) return;
      if (confirmName !== pkg.name) {
        setDeleteError("Nama konfirmasi tidak sama, paket tidak dihapus.");
        setDeleteSuccess(null);
        return;
      }

      setDeletingId(pkg.id);
      setDeleteError(null);
      setDeleteSuccess(null);
      try {
        await apiSend(`/api/billing/packages/${pkg.id}`, "DELETE", { confirmation_name: confirmName });
        packages.reload();
        setDeleteSuccess(`Paket ${pkg.name} berhasil dihapus.`);
      } catch (err) {
        setDeleteError(err instanceof Error ? err.message : "Gagal menghapus paket");
      } finally {
        setDeletingId(null);
      }
    },
    [packages],
  );

  return (
    <RealShell>
      <PageHeader
        eyebrow="Paket"
        title="Paket internet"
        description="Paket bulanan dan voucher dari database Billing API."
        actions={<Button href="/packages/new">Tambah Paket</Button>}
      />
      <Section title="Daftar paket">
        <Notice loading={packages.loading} error={packages.error} />
        {deleteError && <p className="mb-3 text-sm text-red-600">{deleteError}</p>}
        {deleteSuccess && <p className="mb-3 text-sm text-emerald-700">{deleteSuccess}</p>}
        {!packages.loading && rows.length === 0 ? (
          <EmptyState title="Belum ada paket" description="Buat paket bulanan pertama agar pelanggan bisa ditambahkan." action={<Button href="/packages/new">Tambah Paket</Button>} />
        ) : (
          <DataTable
            columns={["Nama", "Tipe", "Bandwidth", "Harga", "Pelanggan", "Status", "Aksi"]}
            rows={rows.map((pkg) => [
              pkg.name,
              packageTypeLabel(pkg.type),
              `${pkg.download_mbps}/${pkg.upload_mbps} Mbps`,
              pkg.type === "voucher" ? money(pkg.sell_price) : money(pkg.monthly_price),
              String(pkg.customer_count ?? 0),
              <StatusBadge key={pkg.id} status={pkg.is_active ? "aktif" : "nonaktif"} />,
              <div key={`${pkg.id}-actions`} className="flex flex-wrap justify-end gap-2 lg:justify-start">
                <a href={`/packages/${pkg.id}`} className="rounded-md px-3 py-1.5 text-sm font-semibold text-blue-700 hover:bg-blue-50">
                  Edit
                </a>
                <button
                  type="button"
                  onClick={() => void deletePackage(pkg)}
                  disabled={deletingId === pkg.id}
                  className="rounded-md px-3 py-1.5 text-sm font-semibold text-red-600 hover:bg-red-50 disabled:cursor-wait disabled:opacity-60"
                >
                  {deletingId === pkg.id ? "Menghapus..." : "Hapus"}
                </button>
              </div>,
            ])}
          />
        )}
      </Section>
    </RealShell>
  );
}

export function PackageFormLivePage() {
  const modules = useTenantModules();
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [type, setType] = useState("monthly");
  const canUseMikrotik = modules.data.mikrotik === true;
  const isVoucher = type === "voucher";
  const isMonthlyPackage = !isVoucher;

  const onSubmit = useCallback(async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const formEl = event.currentTarget;
    const form = new FormData(formEl);
    const packageType = String(form.get("type") || "monthly");
    const submittingVoucher = packageType === "voucher";
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
        quota_mb: packageType === "voucher" ? Number(form.get("quota_mb") || 1024) : undefined,
        monthly_price: !submittingVoucher ? monthlyPrice : undefined,
        installation_fee: !submittingVoucher ? Number(form.get("installation_fee") || 0) : undefined,
        sell_price: packageType === "voucher" ? sellPrice : undefined,
        reseller_price: packageType === "voucher" ? Number(form.get("reseller_price") || 0) : undefined,
        duration_value: packageType === "voucher" ? Number(form.get("duration_value") || 1) : undefined,
        duration_unit: packageType === "voucher" ? String(form.get("duration_unit") || "days") : undefined,
        shared_users: packageType === "voucher" ? Number(form.get("shared_users") || 1) : undefined,
        mikrotik_profile_name: canUseMikrotik && !submittingVoucher ? String(form.get("mikrotik_profile_name") || "") : undefined,
        hotspot_profile_name: canUseMikrotik && submittingVoucher ? String(form.get("hotspot_profile_name") || "") : undefined,
      });
      formEl.reset();
      setType("monthly");
      setSuccess("Paket berhasil dibuat.");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Gagal membuat paket");
    } finally {
      setSaving(false);
    }
  }, [canUseMikrotik]);

  return (
    <RealShell>
      <PageHeader eyebrow="Paket" title="Tambah paket internet" description="Menyimpan paket Billing Core. Field MikroTik hanya tampil jika add-on aktif." />
      <Section title="Konfigurasi paket">
        <form onSubmit={onSubmit} className="grid gap-4 lg:grid-cols-2">
          <FormField label="Tipe paket">
            <select name="type" value={type} onChange={(event) => setType(event.target.value)} className={selectClass}>
              <option value="monthly">Paket bulanan</option>
              {canUseMikrotik && <option value="pppoe">PPPoE bulanan</option>}
              <option value="voucher">Voucher hotspot</option>
            </select>
          </FormField>
          <FormField label="Nama paket"><TextInput name="name" required placeholder="Home 30 Mbps" /></FormField>
          <FormField label="Download Mbps"><TextInput name="download_mbps" type="number" min="1" defaultValue="30" /></FormField>
          <FormField label="Upload Mbps"><TextInput name="upload_mbps" type="number" min="1" defaultValue="10" /></FormField>
          {isMonthlyPackage ? (
            <>
              <FormField label="Harga bulanan"><TextInput name="monthly_price" type="number" min="0" defaultValue="150000" /></FormField>
              <FormField label="Biaya instalasi"><TextInput name="installation_fee" type="number" min="0" defaultValue="0" /></FormField>
              {canUseMikrotik && <FormField label="MikroTik profile"><TextInput name="mikrotik_profile_name" placeholder="home-30m" /></FormField>}
            </>
          ) : (
            <>
              <FormField label="Harga jual voucher"><TextInput name="sell_price" type="number" min="0" defaultValue="5000" /></FormField>
              <FormField label="Harga reseller"><TextInput name="reseller_price" type="number" min="0" defaultValue="4000" /></FormField>
              <FormField label="Kuota voucher (MB)"><TextInput name="quota_mb" type="number" min="1" defaultValue="1024" /></FormField>
              <FormField label="Durasi voucher"><TextInput name="duration_value" type="number" min="1" defaultValue="1" /></FormField>
              <FormField label="Satuan durasi">
                <select name="duration_unit" className={selectClass} defaultValue="days">
                  <option value="hours">Jam</option>
                  <option value="days">Hari</option>
                  <option value="weeks">Minggu</option>
                  <option value="months">Bulan</option>
                </select>
              </FormField>
              <FormField label="Shared users"><TextInput name="shared_users" type="number" min="1" defaultValue="1" /></FormField>
              {canUseMikrotik && <FormField label="Hotspot profile"><TextInput name="hotspot_profile_name" placeholder="voucher-1hari" /></FormField>}
            </>
          )}
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

export function PackageEditLivePage({ id }: { id: string }) {
  const modules = useTenantModules();
  const packageState = useApi<any>(`/api/billing/packages/${id}`, {});
  const pkg = firstOf(packageState.data, ["package"]);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [type, setType] = useState("monthly");
  const canUseMikrotik = modules.data.mikrotik === true;
  const isVoucher = type === "voucher";
  const isMonthlyPackage = !isVoucher;

  useEffect(() => {
    if (pkg.type) setType(String(pkg.type));
  }, [pkg.type]);

  const onSubmit = useCallback(
    async (event: React.FormEvent<HTMLFormElement>) => {
      event.preventDefault();
      const formEl = event.currentTarget;
    const form = new FormData(formEl);
      const packageType = type || String(pkg.type || "monthly");
      const submittingVoucher = packageType === "voucher";
      const monthlyPrice = Number(form.get("monthly_price") || 0);
      const sellPrice = Number(form.get("sell_price") || 0);
      setSaving(true);
      setError(null);
      setSuccess(null);
      try {
        await apiSend(`/api/billing/packages/${id}`, "PUT", {
          name: String(form.get("name") || ""),
          description: String(form.get("description") || ""),
          download_mbps: Number(form.get("download_mbps") || 1),
          upload_mbps: Number(form.get("upload_mbps") || 1),
          bandwidth_type: !submittingVoucher ? "shared" : undefined,
          quota_type: submittingVoucher ? "quota" : "unlimited",
          quota_mb: submittingVoucher ? Number(form.get("quota_mb") || 1024) : undefined,
          monthly_price: !submittingVoucher ? monthlyPrice : undefined,
          installation_fee: !submittingVoucher ? Number(form.get("installation_fee") || 0) : undefined,
          sell_price: submittingVoucher ? sellPrice : undefined,
          reseller_price: submittingVoucher ? Number(form.get("reseller_price") || 0) : undefined,
          duration_value: submittingVoucher ? Number(form.get("duration_value") || 1) : undefined,
          duration_unit: submittingVoucher ? String(form.get("duration_unit") || "days") : undefined,
          shared_users: submittingVoucher ? Number(form.get("shared_users") || 1) : undefined,
          mikrotik_profile_name: canUseMikrotik && !submittingVoucher ? String(form.get("mikrotik_profile_name") || "") : undefined,
          hotspot_profile_name: canUseMikrotik && submittingVoucher ? String(form.get("hotspot_profile_name") || "") : undefined,
        });
        packageState.reload();
        setSuccess("Paket berhasil diperbarui.");
      } catch (err) {
        setError(err instanceof Error ? err.message : "Gagal memperbarui paket");
      } finally {
        setSaving(false);
      }
    },
    [canUseMikrotik, id, packageState, pkg.type, type],
  );

  return (
    <RealShell>
      <PageHeader
        eyebrow="Paket"
        title="Edit paket internet"
        description="Tipe paket tidak diubah setelah dibuat; field edit mengikuti tipe paket saat ini."
        actions={<Button href="/packages">Kembali</Button>}
      />
      <Notice loading={packageState.loading} error={packageState.error} />
      {!packageState.loading && !pkg.id ? (
        <EmptyState title="Paket tidak ditemukan" description="Data paket tidak tersedia di tenant aktif." />
      ) : (
        <Section title="Konfigurasi paket">
          <form key={pkg.id || "package-edit"} onSubmit={onSubmit} className="grid gap-4 lg:grid-cols-2">
            <FormField label="Tipe paket">
              <TextInput value={packageTypeLabel(type)} disabled readOnly />
            </FormField>
            <FormField label="Nama paket"><TextInput name="name" required defaultValue={pkg.name ?? ""} /></FormField>
            <FormField label="Download Mbps"><TextInput name="download_mbps" type="number" min="1" defaultValue={pkg.download_mbps ?? 30} /></FormField>
            <FormField label="Upload Mbps"><TextInput name="upload_mbps" type="number" min="1" defaultValue={pkg.upload_mbps ?? 10} /></FormField>
            {isMonthlyPackage ? (
              <>
                <FormField label="Harga bulanan"><TextInput name="monthly_price" type="number" min="0" defaultValue={pkg.monthly_price ?? 150000} /></FormField>
                <FormField label="Biaya instalasi"><TextInput name="installation_fee" type="number" min="0" defaultValue={pkg.installation_fee ?? 0} /></FormField>
                {canUseMikrotik && <FormField label="MikroTik profile"><TextInput name="mikrotik_profile_name" defaultValue={pkg.mikrotik_profile_name ?? ""} placeholder="home-30m" /></FormField>}
              </>
            ) : (
              <>
                <FormField label="Harga jual voucher"><TextInput name="sell_price" type="number" min="0" defaultValue={pkg.sell_price ?? 5000} /></FormField>
                <FormField label="Harga reseller"><TextInput name="reseller_price" type="number" min="0" defaultValue={pkg.reseller_price ?? 4000} /></FormField>
                <FormField label="Kuota voucher (MB)"><TextInput name="quota_mb" type="number" min="1" defaultValue={pkg.quota_mb ?? 1024} /></FormField>
                <FormField label="Durasi voucher"><TextInput name="duration_value" type="number" min="1" defaultValue={pkg.duration_value ?? 1} /></FormField>
                <FormField label="Satuan durasi">
                  <select name="duration_unit" className={selectClass} defaultValue={pkg.duration_unit ?? "days"}>
                    <option value="hours">Jam</option>
                    <option value="days">Hari</option>
                    <option value="weeks">Minggu</option>
                    <option value="months">Bulan</option>
                  </select>
                </FormField>
                <FormField label="Shared users"><TextInput name="shared_users" type="number" min="1" defaultValue={pkg.shared_users ?? 1} /></FormField>
                {canUseMikrotik && <FormField label="Hotspot profile"><TextInput name="hotspot_profile_name" defaultValue={pkg.hotspot_profile_name ?? ""} placeholder="voucher-1hari" /></FormField>}
              </>
            )}
            <div className="lg:col-span-2">
              <FormField label="Deskripsi"><textarea name="description" className={textAreaClass} rows={3} defaultValue={pkg.description ?? ""} /></FormField>
            </div>
            <div className="lg:col-span-2">
              <SubmitBar saving={saving} error={error} success={success} label="Update Paket" />
            </div>
          </form>
        </Section>
      )}
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
  const [generating, setGenerating] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  const onSubmit = useCallback(
    async (event: React.FormEvent<HTMLFormElement>) => {
      event.preventDefault();
      const formEl = event.currentTarget;
    const form = new FormData(formEl);
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
        formEl.reset();
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

  const generateDueInvoices = useCallback(async () => {
    setGenerating(true);
    setError(null);
    setSuccess(null);
    try {
      await apiSend("/api/billing/invoices/generate-due", "POST", {});
      invoices.reload();
      summary.reload();
      setSuccess("Tagihan jatuh tempo berhasil diproses.");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Gagal memproses tagihan jatuh tempo");
    } finally {
      setGenerating(false);
    }
  }, [invoices, summary]);

  return (
    <RealShell>
      <PageHeader
        eyebrow="Invoice"
        title="Invoice pelanggan"
        description="Invoice real dari Billing API."
        actions={
          <button
            type="button"
            onClick={() => void generateDueInvoices()}
            disabled={generating}
            className="rounded-md bg-blue-600 px-4 py-2 text-sm font-semibold text-white disabled:opacity-60"
          >
            {generating ? "Memproses..." : "Generate jatuh tempo"}
          </button>
        }
      />
      <StatGrid
        stats={[
          { label: "Total invoice", value: String(summary.data.total_invoices ?? rows.length) },
          { label: "Belum lunas", value: String(summary.data.unpaid_count ?? summary.data.belum_bayar ?? 0), tone: "amber" },
          { label: "Terlambat", value: String(summary.data.overdue_count ?? 0), tone: "red" },
          { label: "Outstanding", value: money(summary.data.outstanding_amount ?? 0) },
        ]}
      />
      <Section title="Daftar invoice">
        {error && <p className="mb-4 text-sm text-red-600">{error}</p>}
        {success && <p className="mb-4 text-sm text-emerald-700">{success}</p>}
        <Notice loading={invoices.loading} error={invoices.error} />
        {rows.length === 0 && !invoices.loading ? (
          <EmptyState title="Belum ada invoice" description="Aktifkan pelanggan lalu tekan Generate jatuh tempo, atau buat invoice manual." />
        ) : (
          <DataTable
            columns={["Nomor", "Pelanggan", "Periode", "Jatuh tempo", "Total", "Dibayar", "Status", "Aksi"]}
            rows={rows.map((invoice) => [
              <a key={invoice.id} href={`/invoices/${invoice.id}`} className="font-mono font-semibold text-blue-700 hover:text-blue-900">
                {invoice.invoice_number}
              </a>,
              invoice.customer_name ?? invoice.customer_id_seq ?? "-",
              `${invoice.period_month}/${invoice.period_year}`,
              dateID(invoice.due_date),
              money(invoice.total_amount),
              money(invoice.paid_amount),
              <StatusBadge key={invoice.id} status={invoice.status ?? "belum_bayar"} />,
              <Button key={`detail-${invoice.id}`} variant="ghost" href={`/invoices/${invoice.id}`}>
                Detail
              </Button>,
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
  const remainingAmount = Math.max(Number(invoice.total_amount ?? 0) - Number(invoice.paid_amount ?? 0), 0);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  const onSubmit = useCallback(
    async (event: React.FormEvent<HTMLFormElement>) => {
      event.preventDefault();
      const formEl = event.currentTarget;
    const form = new FormData(formEl);
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
        formEl.reset();
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
                { label: "Sisa", value: money(remainingAmount) },
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

          <Section title="Catat pembayaran" description="Nominal boleh sebagian. Jika nominal melebihi sisa tagihan atau invoice sudah lunas, kelebihannya masuk saldo kredit pelanggan.">
            <form onSubmit={onSubmit} className="grid gap-4 lg:grid-cols-4">
              <FormField label="Nominal"><TextInput name="amount" type="number" min="1" defaultValue={remainingAmount > 0 ? String(remainingAmount) : ""} placeholder={remainingAmount > 0 ? "" : "Nominal pembayaran dobel"} /></FormField>
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
  const customerOptions = useMemo(() => {
    const seen = new Set<string>();
    return invoiceRows
      .filter((invoice) => {
        const customerID = String(invoice.customer_id || "");
        if (!customerID || seen.has(customerID)) return false;
        seen.add(customerID);
        return true;
      })
      .map((invoice) => ({
        id: String(invoice.customer_id),
        label: `${invoice.customer_name ?? invoice.customer_id_seq ?? "Pelanggan"}${invoice.customer_id_seq ? ` (${invoice.customer_id_seq})` : ""}`,
      }));
  }, [invoiceRows]);
  const [multiSaving, setMultiSaving] = useState(false);
  const [multiError, setMultiError] = useState<string | null>(null);
  const [multiSuccess, setMultiSuccess] = useState<string | null>(null);
  const [multiCustomerId, setMultiCustomerId] = useState("");
  const [selectedInvoiceIds, setSelectedInvoiceIds] = useState<string[]>([]);

  const invoicesForCustomer = invoiceRows.filter((invoice) => String(invoice.customer_id || "") === multiCustomerId);
  const selectedInvoices = invoicesForCustomer.filter((invoice) => selectedInvoiceIds.includes(String(invoice.id)));
  const selectedTotal = selectedInvoices.reduce(
    (sum, invoice) => sum + Math.max(Number(invoice.total_amount ?? 0) - Number(invoice.paid_amount ?? 0), 0),
    0,
  );
  const customerTotal = invoicesForCustomer.reduce(
    (sum, invoice) => sum + Math.max(Number(invoice.total_amount ?? 0) - Number(invoice.paid_amount ?? 0), 0),
    0,
  );
  const allocationLabel =
    selectedInvoiceIds.length === 0
      ? "Semua invoice terbuka"
      : selectedInvoiceIds.length === 1
        ? "Satu invoice"
        : `${selectedInvoiceIds.length} invoice`;

  const onMultiSubmit = useCallback(
    async (event: React.FormEvent<HTMLFormElement>) => {
      event.preventDefault();
      const formEl = event.currentTarget;
      const form = new FormData(formEl);
      const customerID = String(form.get("customer_id") || "");
      const amount = Number(form.get("amount") || 0);
      setMultiSaving(true);
      setMultiError(null);
      setMultiSuccess(null);
      try {
        const response = await apiSend("/api/billing/payments/multi", "POST", {
          customer_id: customerID,
          amount,
          payment_method: String(form.get("payment_method") || "tunai"),
          payment_date: String(form.get("payment_date") || todayDate()),
          reference_number: String(form.get("reference_number") || ""),
          notes: String(form.get("notes") || ""),
          invoice_ids: selectedInvoiceIds,
        });
        formEl.reset();
        setMultiCustomerId("");
        setSelectedInvoiceIds([]);
        payments.reload();
        summary.reload();
        invoices.reload();
        const result = (response || {}) as AnyRecord;
        const excess = Number(result.excess_to_credit || 0);
        setMultiSuccess(
          excess > 0
            ? `Pembayaran dialokasikan. Kelebihan ${money(excess)} masuk saldo kredit pelanggan.`
            : "Pembayaran multi-invoice berhasil dialokasikan.",
        );
      } catch (err) {
        setMultiError(err instanceof Error ? err.message : "Gagal mencatat pembayaran multi-invoice");
      } finally {
        setMultiSaving(false);
      }
    },
    [invoices, payments, selectedInvoiceIds, summary],
  );

  const toggleInvoice = useCallback((invoiceID: string, checked: boolean) => {
    setSelectedInvoiceIds((current) => {
      if (checked) return Array.from(new Set([...current, invoiceID]));
      return current.filter((id) => id !== invoiceID);
    });
  }, []);

  return (
    <RealShell>
      <PageHeader
        eyebrow="Pembayaran"
        title="Pembayaran"
        description="Catat pembayaran sebagian, pembayaran beberapa invoice sekaligus, dan kelebihan bayar sebagai kredit pelanggan."
      />
      <StatGrid
        stats={[
          { label: "Transaksi terakhir", value: String(rows.length) },
          { label: "Bulan ini", value: money(summary.data.this_month?.total_amount ?? 0) },
          { label: "Hari ini", value: money(summary.data.today?.total_amount ?? 0) },
          { label: "Void", value: String(rows.filter((payment) => payment.voided).length), tone: "red" },
        ]}
      />

      <Section title="Catat pembayaran" description="Satu form untuk bayar sebagian, satu invoice, beberapa bulan, atau kelebihan bayar sebagai saldo kredit pelanggan.">
        <form onSubmit={onMultiSubmit} className="grid gap-4 lg:grid-cols-4">
          <FormField label="Pelanggan">
            <select
              name="customer_id"
              required
              className={selectClass}
              value={multiCustomerId}
              onChange={(event) => {
                setMultiCustomerId(event.target.value);
                setSelectedInvoiceIds([]);
              }}
            >
              <option value="">Pilih pelanggan</option>
              {customerOptions.map((customer) => (
                <option key={customer.id} value={customer.id}>{customer.label}</option>
              ))}
            </select>
          </FormField>
          <FormField label="Nominal dibayar">
            <TextInput key={`${multiCustomerId}-${selectedTotal || customerTotal}`} name="amount" type="number" min="1" defaultValue={String(selectedTotal || customerTotal || 0)} />
          </FormField>
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
            <FormField label="Catatan"><TextInput name="notes" placeholder="contoh: bayar Jan-Feb sebagian" /></FormField>
          </div>
          {multiCustomerId ? (
            <div className="lg:col-span-4 rounded-lg border border-slate-200 bg-slate-50 p-3">
              <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
                <div>
                  <p className="text-sm font-semibold text-slate-800">Invoice terbuka: {invoicesForCustomer.length}</p>
                  <p className="text-xs text-slate-500">{allocationLabel}</p>
                </div>
                <p className="font-mono text-sm text-slate-700">Total dialokasikan {money(selectedTotal || customerTotal)}</p>
              </div>
              <div className="grid gap-2 md:grid-cols-2 xl:grid-cols-3">
                {invoicesForCustomer.map((invoice) => {
                  const remaining = Math.max(Number(invoice.total_amount ?? 0) - Number(invoice.paid_amount ?? 0), 0);
                  const checked = selectedInvoiceIds.includes(String(invoice.id));
                  return (
                    <label key={invoice.id} className="flex min-h-[76px] gap-3 rounded-md border border-slate-200 bg-white p-3 text-sm">
                      <input
                        type="checkbox"
                        checked={checked}
                        onChange={(event) => toggleInvoice(String(invoice.id), event.target.checked)}
                        className="mt-1"
                      />
                      <span className="min-w-0">
                        <span className="block truncate font-semibold text-slate-900">{invoice.invoice_number}</span>
                        <span className="block text-xs text-slate-500">{invoice.period_month}/{invoice.period_year} · {invoice.status}</span>
                        <span className="block font-mono text-xs text-slate-700">{money(remaining)}</span>
                      </span>
                    </label>
                  );
                })}
              </div>
              {selectedInvoiceIds.length === 0 ? (
                <p className="mt-3 text-xs text-slate-500">Jika tidak memilih invoice, sistem akan mengalokasikan ke semua invoice terbuka pelanggan ini mulai dari yang paling lama.</p>
              ) : null}
            </div>
          ) : null}
          <div className="lg:col-span-4">
            <SubmitBar saving={multiSaving} error={multiError} success={multiSuccess} label="Catat pembayaran" />
          </div>
        </form>
      </Section>

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
      const formEl = event.currentTarget;
    const form = new FormData(formEl);
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
        formEl.reset();
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
  const activeRows = rows.filter((reseller) => reseller.status !== "nonaktif");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [balanceSaving, setBalanceSaving] = useState(false);
  const [balanceError, setBalanceError] = useState<string | null>(null);
  const [balanceSuccess, setBalanceSuccess] = useState<string | null>(null);
  const [selectedResellerId, setSelectedResellerId] = useState("");
  const [balanceAction, setBalanceAction] = useState<"deposit" | "withdraw">("deposit");
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [deleteError, setDeleteError] = useState<string | null>(null);
  const [deleteSuccess, setDeleteSuccess] = useState<string | null>(null);
  const selectedReseller = rows.find((reseller) => reseller.id === selectedResellerId);

  const onSubmit = useCallback(
    async (event: React.FormEvent<HTMLFormElement>) => {
      event.preventDefault();
      const formEl = event.currentTarget;
    const form = new FormData(formEl);
      setSaving(true);
      setError(null);
      setSuccess(null);
      try {
        await apiSend("/api/billing/resellers", "POST", {
          name: String(form.get("name") || ""),
          phone: normalizeIndonesianPhone(form.get("phone")),
          email: String(form.get("email") || ""),
          address: String(form.get("address") || ""),
          password: String(form.get("password") || ""),
          balance: Number(form.get("balance") || 0),
          daily_purchase_limit: Number(form.get("daily_purchase_limit") || 0),
        });
        formEl.reset();
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

  const onBalanceSubmit = useCallback(
    async (event: React.FormEvent<HTMLFormElement>) => {
      event.preventDefault();
      const formEl = event.currentTarget;
      const form = new FormData(formEl);
      const resellerId = String(form.get("reseller_id") || "");
      const action = String(form.get("action") || "deposit") as "deposit" | "withdraw";
      const amount = Number(form.get("amount") || 0);
      const notes = String(form.get("notes") || "");

      if (!resellerId) {
        setBalanceError("Pilih reseller terlebih dahulu.");
        setBalanceSuccess(null);
        return;
      }
      if (amount <= 0) {
        setBalanceError("Nominal harus lebih dari 0.");
        setBalanceSuccess(null);
        return;
      }

      const target = rows.find((reseller) => reseller.id === resellerId);
      if (action === "withdraw" && target && amount > Number(target.balance || 0)) {
        setBalanceError("Saldo reseller tidak cukup untuk withdraw.");
        setBalanceSuccess(null);
        return;
      }

      setBalanceSaving(true);
      setBalanceError(null);
      setBalanceSuccess(null);
      try {
        await apiSend(`/api/billing/resellers/${resellerId}/${action}`, "POST", { amount, notes });
        resellers.reload();
        setSelectedResellerId(resellerId);
        formEl.reset();
        setBalanceAction(action);
        setBalanceSuccess(
          `${action === "deposit" ? "Deposit" : "Withdraw"} ${money(amount)} untuk ${target?.name ?? "reseller"} berhasil diproses.`,
        );
      } catch (err) {
        setBalanceError(err instanceof Error ? err.message : "Gagal mengelola saldo reseller");
      } finally {
        setBalanceSaving(false);
      }
    },
    [resellers, rows],
  );

  const deactivateReseller = useCallback(
    async (reseller: AnyRecord) => {
      const confirmName = window.prompt(`Ketik nama reseller "${reseller.name}" untuk menonaktifkan reseller ini.`);
      if (confirmName == null) return;
      if (confirmName !== reseller.name) {
        setDeleteError("Nama konfirmasi tidak sama, reseller tidak dihapus.");
        setDeleteSuccess(null);
        return;
      }

      setDeletingId(reseller.id);
      setDeleteError(null);
      setDeleteSuccess(null);
      try {
        await apiSend(`/api/billing/resellers/${reseller.id}/deactivate`, "POST", { confirmation_name: confirmName });
        resellers.reload();
        setDeleteSuccess(`Reseller ${reseller.name} berhasil dinonaktifkan.`);
      } catch (err) {
        setDeleteError(err instanceof Error ? err.message : "Gagal menghapus reseller");
      } finally {
        setDeletingId(null);
      }
    },
    [resellers],
  );

  return (
    <RealShell>
      <PageHeader eyebrow="Reseller" title="Reseller voucher" description="Akun reseller real dari Billing API." />
      <Section
        title="Kelola saldo reseller"
        description="Tambah saldo deposit atau kurangi saldo withdraw. Semua perubahan tercatat sebagai transaksi reseller."
      >
        <form onSubmit={onBalanceSubmit} className="grid gap-4 xl:grid-cols-[minmax(0,1.2fr)_160px_180px_minmax(0,1fr)_auto]">
          <FormField label="Reseller">
            <select
              name="reseller_id"
              required
              className={selectClass}
              value={selectedResellerId}
              onChange={(event) => setSelectedResellerId(event.target.value)}
            >
              <option value="">Pilih reseller</option>
              {activeRows.map((reseller) => (
                <option key={reseller.id} value={reseller.id}>
                  {reseller.name} - {money(reseller.balance)}
                </option>
              ))}
            </select>
          </FormField>
          <FormField label="Aksi">
            <select
              name="action"
              className={selectClass}
              value={balanceAction}
              onChange={(event) => setBalanceAction(event.target.value as "deposit" | "withdraw")}
            >
              <option value="deposit">Deposit</option>
              <option value="withdraw">Withdraw</option>
            </select>
          </FormField>
          <FormField label="Nominal">
            <TextInput name="amount" type="number" min="1" step="1" required placeholder="50000" />
          </FormField>
          <FormField label="Catatan">
            <TextInput name="notes" placeholder={balanceAction === "deposit" ? "Top up saldo reseller" : "Koreksi saldo reseller"} />
          </FormField>
          <div className="grid content-end">
            <button
              type="submit"
              disabled={balanceSaving || activeRows.length === 0}
              className="inline-flex h-10 items-center justify-center rounded-md bg-slate-900 px-4 text-sm font-semibold text-white transition hover:bg-slate-800 disabled:cursor-not-allowed disabled:opacity-60"
            >
              {balanceSaving ? "Memproses..." : balanceAction === "deposit" ? "Tambah saldo" : "Kurangi saldo"}
            </button>
          </div>
        </form>
        <div className="mt-4 grid gap-3 md:grid-cols-3">
          <div className="rounded-lg border border-slate-200 bg-slate-50 p-4">
            <p className="text-xs font-semibold uppercase tracking-[0.14em] text-slate-400">Saldo reseller dipilih</p>
            <p className="mt-2 text-lg font-semibold text-slate-950">{selectedReseller ? money(selectedReseller.balance) : "-"}</p>
          </div>
          <div className="rounded-lg border border-slate-200 bg-slate-50 p-4">
            <p className="text-xs font-semibold uppercase tracking-[0.14em] text-slate-400">Limit harian</p>
            <p className="mt-2 text-lg font-semibold text-slate-950">
              {selectedReseller ? String(selectedReseller.daily_purchase_limit ?? 0) : "-"}
            </p>
          </div>
          <div className="rounded-lg border border-slate-200 bg-slate-50 p-4">
            <p className="text-xs font-semibold uppercase tracking-[0.14em] text-slate-400">Status</p>
            <div className="mt-2">{selectedReseller ? <StatusBadge status={selectedReseller.status ?? "aktif"} /> : "-"}</div>
          </div>
        </div>
        {balanceError && <p className="mt-3 text-sm text-red-600">{balanceError}</p>}
        {balanceSuccess && <p className="mt-3 text-sm text-emerald-700">{balanceSuccess}</p>}
      </Section>
      <Section title="Daftar reseller">
        <Notice loading={resellers.loading} error={resellers.error} />
        {deleteError && <p className="mb-3 text-sm text-red-600">{deleteError}</p>}
        {deleteSuccess && <p className="mb-3 text-sm text-emerald-700">{deleteSuccess}</p>}
        {rows.length === 0 && !resellers.loading ? (
          <EmptyState title="Belum ada reseller" description="Reseller akan muncul setelah akun reseller dibuat." />
        ) : (
          <DataTable
            columns={["Nama", "Telepon", "Email", "Saldo", "Limit harian", "Voucher terjual", "Status", "Aksi"]}
            rows={rows.map((reseller) => [
              reseller.name,
              reseller.phone,
              reseller.email ?? "-",
              money(reseller.balance),
              String(reseller.daily_purchase_limit ?? 0),
              String(reseller.total_vouchers_sold ?? 0),
              <StatusBadge key={reseller.id} status={reseller.status ?? "aktif"} />,
              <div key={`${reseller.id}-actions`} className="flex flex-wrap justify-end gap-2 lg:justify-start">
                <button
                  type="button"
                  onClick={() => {
                    setSelectedResellerId(reseller.id);
                    setBalanceError(null);
                    setBalanceSuccess(null);
                  }}
                  disabled={reseller.status === "nonaktif"}
                  className="rounded-md px-3 py-1.5 text-sm font-semibold text-emerald-700 hover:bg-emerald-50 disabled:cursor-not-allowed disabled:opacity-50"
                >
                  Saldo
                </button>
                <a href={`/resellers/${reseller.id}/edit`} className="rounded-md px-3 py-1.5 text-sm font-semibold text-blue-700 hover:bg-blue-50">
                  Edit
                </a>
                <button
                  type="button"
                  onClick={() => void deactivateReseller(reseller)}
                  disabled={deletingId === reseller.id || reseller.status === "nonaktif"}
                  className="rounded-md px-3 py-1.5 text-sm font-semibold text-red-600 hover:bg-red-50 disabled:cursor-not-allowed disabled:opacity-50"
                >
                  {deletingId === reseller.id ? "Menghapus..." : reseller.status === "nonaktif" ? "Nonaktif" : "Hapus"}
                </button>
              </div>,
            ])}
          />
        )}
      </Section>
      <Section title="Tambah reseller">
        <form onSubmit={onSubmit} className="grid gap-4 lg:grid-cols-4">
          <FormField label="Nama"><TextInput name="name" required placeholder="Reseller Depok" /></FormField>
          <FormField label="Telepon"><TextInput name="phone" required placeholder="081299990000 atau +6281299990000" /></FormField>
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

export function ResellerEditLivePage({ id }: { id: string }) {
  const resellerState = useApi<any>(`/api/billing/resellers/${id}`, {});
  const reseller = firstOf(resellerState.data, ["reseller"]);
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
        await apiSend(`/api/billing/resellers/${id}`, "PUT", {
          name: String(form.get("name") || ""),
          phone: normalizeIndonesianPhone(form.get("phone")),
          email: String(form.get("email") || ""),
          address: String(form.get("address") || ""),
          daily_purchase_limit: Number(form.get("daily_purchase_limit") || 0),
        });
        resellerState.reload();
        setSuccess("Reseller berhasil diperbarui.");
      } catch (err) {
        setError(err instanceof Error ? err.message : "Gagal memperbarui reseller");
      } finally {
        setSaving(false);
      }
    },
    [id, resellerState],
  );

  return (
    <RealShell>
      <PageHeader
        eyebrow="Reseller"
        title="Edit reseller voucher"
        description="Perbarui profil reseller. Saldo tetap dikelola lewat transaksi deposit/withdraw."
        actions={<Button href="/resellers" variant="secondary">Daftar Reseller</Button>}
      />
      <Notice loading={resellerState.loading} error={resellerState.error} />
      {!resellerState.loading && !reseller.id ? (
        <EmptyState title="Reseller tidak ditemukan" description="Data reseller tidak tersedia di tenant aktif." />
      ) : (
        <Section title="Data reseller">
          <form key={reseller.id || "reseller-edit"} onSubmit={onSubmit} className="grid gap-4 lg:grid-cols-4">
            <FormField label="Nama"><TextInput name="name" required defaultValue={reseller.name ?? ""} /></FormField>
            <FormField label="Telepon"><TextInput name="phone" required defaultValue={reseller.phone ?? ""} placeholder="081299990000 atau +6281299990000" /></FormField>
            <FormField label="Email"><TextInput name="email" type="email" defaultValue={reseller.email ?? ""} placeholder="opsional" /></FormField>
            <FormField label="Limit harian"><TextInput name="daily_purchase_limit" type="number" min="0" defaultValue={reseller.daily_purchase_limit ?? 0} /></FormField>
            <div className="lg:col-span-4">
              <FormField label="Alamat"><TextInput name="address" defaultValue={reseller.address ?? ""} placeholder="opsional" /></FormField>
            </div>
            <div className="lg:col-span-4">
              <SubmitBar saving={saving} error={error} success={success} label="Update reseller" />
            </div>
          </form>
        </Section>
      )}
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
    const formEl = event.currentTarget;
    const form = new FormData(formEl);
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
      formEl.reset();
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
      const formEl = event.currentTarget;
    const form = new FormData(formEl);
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
        formEl.reset();
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
