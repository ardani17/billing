"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import AppShell from "../components/app-shell";
import { DataTable, EmptyState, FormField, PageHeader, Section, StatGrid, StatusBadge, TextInput } from "../components/ui";

type ApiEnvelope<T> = { success: boolean; data?: T; error?: { message?: string } };
type Item = {
  id: string;
  name: string;
  category: string;
  unit: string;
  track_serial: boolean;
  min_stock: number;
  default_cost: number;
  is_active: boolean;
  stock?: number;
};
type Asset = {
  id: string;
  item_id: string;
  item_name?: string;
  serial_number: string;
  mac_address?: string;
  status: string;
  location_type: string;
  assigned_customer_name?: string;
  purchase_cost: number;
};
type Customer = {
  id: string;
  name: string;
  phone?: string;
  status?: string;
};
type ExpenseCategory = {
  id: string;
  name: string;
};
type Movement = {
  id: string;
  item_id: string;
  item_name?: string;
  movement_type: string;
  quantity: number;
  customer_name?: string;
  unit_cost: number;
  notes?: string;
  created_at: string;
};

const tabs = ["Ringkasan", "Barang", "Aset Serial", "Mutasi Stok"] as const;

function money(value: number | null | undefined) {
  return new Intl.NumberFormat("id-ID", { style: "currency", currency: "IDR", maximumFractionDigits: 0 }).format(value ?? 0);
}

function dateID(value?: string) {
  if (!value) return "-";
  return new Intl.DateTimeFormat("id-ID", { dateStyle: "medium", timeStyle: "short" }).format(new Date(value));
}

async function api<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`/api/billing${path}`, {
    ...init,
    headers: { "Content-Type": "application/json", ...(init?.headers || {}) },
    cache: "no-store",
  });
  const body = (await response.json().catch(() => ({}))) as ApiEnvelope<T>;
  if (!response.ok || body.success === false) throw new Error(body.error?.message || `Request gagal (${response.status})`);
  return (body.data ?? body) as T;
}

function listOf<T>(value: unknown): T[] {
  if (Array.isArray(value)) return value as T[];
  if (value && typeof value === "object") {
    const record = value as Record<string, unknown>;
    if (Array.isArray(record.data)) return record.data as T[];
    if (Array.isArray(record.items)) return record.items as T[];
    if (Array.isArray(record.customers)) return record.customers as T[];
  }
  return [];
}

export default function InventoryPage() {
  const [items, setItems] = useState<Item[]>([]);
  const [assets, setAssets] = useState<Asset[]>([]);
  const [customers, setCustomers] = useState<Customer[]>([]);
  const [expenseCategories, setExpenseCategories] = useState<ExpenseCategory[]>([]);
  const [movements, setMovements] = useState<Movement[]>([]);
  const [activeTab, setActiveTab] = useState<(typeof tabs)[number]>("Ringkasan");
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");

  const loadData = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const [nextItems, nextAssets, nextMovements] = await Promise.all([
        api<Item[]>("/inventory/items"),
        api<Asset[]>("/inventory/assets"),
        api<Movement[]>("/inventory/movements"),
      ]);
      setItems(nextItems);
      setAssets(nextAssets);
      setMovements(nextMovements);
      api<unknown>("/customers?page_size=100")
        .then((value) => setCustomers(listOf<Customer>(value)))
        .catch(() => setCustomers([]));
      api<unknown>("/expenses/categories")
        .then((value) => setExpenseCategories(listOf<ExpenseCategory>(value)))
        .catch(() => setExpenseCategories([]));
    } catch (err) {
      setError(err instanceof Error ? err.message : "Gagal memuat inventaris");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadData();
  }, [loadData]);

  const lowStock = items.filter((item) => Number(item.stock || 0) <= Number(item.min_stock || 0));
  const totalValue = items.reduce((sum, item) => sum + Number(item.stock || 0) * Number(item.default_cost || 0), 0);
  const serialAssets = assets.filter((asset) => asset.status !== "retired");

  async function submitItem(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    setSaving(true);
    setError("");
    setSuccess("");
    try {
      await api("/inventory/items", {
        method: "POST",
        body: JSON.stringify({
          name: String(form.get("name") || ""),
          category: String(form.get("category") || "Perangkat"),
          unit: String(form.get("unit") || "unit"),
          track_serial: form.get("track_serial") === "on",
          min_stock: Number(form.get("min_stock") || 0),
          default_cost: Number(form.get("default_cost") || 0),
        }),
      });
      event.currentTarget.reset();
      setSuccess("Item inventaris berhasil dibuat.");
      await loadData();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Gagal menyimpan item");
    } finally {
      setSaving(false);
    }
  }

  async function submitAsset(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    setSaving(true);
    setError("");
    setSuccess("");
    try {
      await api("/inventory/assets", {
        method: "POST",
        body: JSON.stringify({
          item_id: String(form.get("item_id") || ""),
          serial_number: String(form.get("serial_number") || ""),
          mac_address: String(form.get("mac_address") || ""),
          purchase_cost: Number(form.get("purchase_cost") || 0),
          purchase_date: String(form.get("purchase_date") || ""),
        }),
      });
      event.currentTarget.reset();
      setSuccess("Aset serial berhasil ditambahkan.");
      await loadData();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Gagal menyimpan aset");
    } finally {
      setSaving(false);
    }
  }

  async function submitMovement(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    setSaving(true);
    setError("");
    setSuccess("");
    try {
      await api("/inventory/movements", {
        method: "POST",
        body: JSON.stringify({
          item_id: String(form.get("item_id") || ""),
          asset_id: String(form.get("asset_id") || ""),
          movement_type: String(form.get("movement_type") || "purchase"),
          quantity: Number(form.get("quantity") || 1),
          unit_cost: Number(form.get("unit_cost") || 0),
          notes: String(form.get("notes") || ""),
          create_expense: form.get("create_expense") === "on",
          expense_category_id: String(form.get("expense_category_id") || ""),
        }),
      });
      event.currentTarget.reset();
      setSuccess("Mutasi stok berhasil dicatat.");
      await loadData();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Gagal mencatat mutasi");
    } finally {
      setSaving(false);
    }
  }

  async function submitAssetAction(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    const assetId = String(form.get("asset_id") || "");
    const action = String(form.get("action") || "assign");
    const customerID = String(form.get("customer_id") || "");
    setSaving(true);
    setError("");
    setSuccess("");
    try {
      await api(`/inventory/assets/${assetId}/${action}`, {
        method: "POST",
        body: JSON.stringify({
          customer_id: customerID,
          location_type: action === "assign" ? "customer" : action === "mark-damaged" ? "damaged" : action === "mark-lost" ? "lost" : action === "mark-rma" ? "rma" : "warehouse",
          notes: String(form.get("notes") || ""),
        }),
      });
      event.currentTarget.reset();
      setSuccess("Status aset berhasil diperbarui.");
      await loadData();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Gagal memperbarui aset");
    } finally {
      setSaving(false);
    }
  }

  async function quickAssetAction(assetId: string, action: "return" | "mark-damaged" | "mark-lost" | "mark-rma" | "retire") {
    setSaving(true);
    setError("");
    setSuccess("");
    try {
      await api(`/inventory/assets/${assetId}/${action}`, {
        method: "POST",
        body: JSON.stringify({
          location_type: action === "mark-damaged" ? "damaged" : "warehouse",
          notes: action === "mark-lost" ? "Aset hilang" : action === "mark-rma" ? "Aset RMA" : action === "retire" ? "Aset dipensiunkan" : "",
        }),
      });
      const messages: Record<typeof action, string> = {
        "return": "Aset dikembalikan ke gudang.",
        "mark-damaged": "Aset ditandai rusak.",
        "mark-lost": "Aset ditandai hilang.",
        "mark-rma": "Aset masuk RMA.",
        "retire": "Aset dipensiunkan.",
      };
      setSuccess(messages[action]);
      await loadData();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Gagal memperbarui aset");
    } finally {
      setSaving(false);
    }
  }

  const serialItems = useMemo(() => items.filter((item) => item.track_serial), [items]);

  return (
    <AppShell>
      <div className="space-y-6">
        <PageHeader
          eyebrow="Keuangan"
          title="Inventaris"
          description="Kelola stok perangkat, aset serial, dan mutasi barang operasional ISP."
        />

        {error && <div className="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">{error}</div>}
        {success && <div className="rounded-lg border border-emerald-200 bg-emerald-50 px-4 py-3 text-sm text-emerald-700">{success}</div>}

        <StatGrid
          stats={[
            { label: "Item aktif", value: String(items.length), tone: "blue" },
            { label: "Aset serial", value: String(serialAssets.length), tone: "green" },
            { label: "Stok rendah", value: String(lowStock.length), tone: lowStock.length ? "amber" : "slate" },
            { label: "Estimasi nilai stok", value: money(totalValue), tone: "violet" },
          ]}
        />

        <nav className="flex gap-2 overflow-x-auto rounded-xl border border-slate-200 bg-white p-2">
          {tabs.map((tab) => (
            <button
              key={tab}
              type="button"
              onClick={() => setActiveTab(tab)}
              className={`h-10 shrink-0 rounded-lg px-4 text-sm font-semibold ${activeTab === tab ? "bg-blue-600 text-white" : "text-slate-600 hover:bg-slate-50"}`}
            >
              {tab}
            </button>
          ))}
        </nav>

        {loading ? <div className="h-48 animate-pulse rounded-xl border border-slate-200 bg-white" /> : null}

        {!loading && activeTab === "Ringkasan" && (
          <Section title="Stok menipis" description="Item dengan stok sama atau di bawah minimum.">
            {lowStock.length ? (
              <DataTable
                columns={["Item", "Kategori", "Stok", "Minimum", "Status"]}
                rows={lowStock.map((item) => [item.name, item.category, `${item.stock ?? 0} ${item.unit}`, String(item.min_stock), <StatusBadge key={item.id} status="rendah" />])}
              />
            ) : (
              <EmptyState title="Stok aman" description="Tidak ada item yang berada di bawah batas minimum." />
            )}
          </Section>
        )}

        {!loading && activeTab === "Barang" && (
          <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_380px]">
            <Section title="Master barang">
              <DataTable
                columns={["Nama", "Kategori", "Stok", "Unit", "Serial", "Biaya default", "Status"]}
                rows={items.map((item) => [
                  item.name,
                  item.category,
                  String(item.stock ?? 0),
                  item.unit,
                  item.track_serial ? "Ya" : "Tidak",
                  money(item.default_cost),
                  <StatusBadge key={item.id} status={item.is_active ? "aktif" : "nonaktif"} />,
                ])}
              />
            </Section>
            <Section title="Tambah barang">
              <form onSubmit={submitItem} className="grid gap-4">
                <FormField label="Nama"><TextInput name="name" required placeholder="ONT Huawei HG8245H" /></FormField>
                <FormField label="Kategori"><TextInput name="category" defaultValue="Perangkat" /></FormField>
                <FormField label="Unit"><TextInput name="unit" defaultValue="unit" /></FormField>
                <FormField label="Minimum stok"><TextInput name="min_stock" type="number" min={0} defaultValue={0} /></FormField>
                <FormField label="Biaya default"><TextInput name="default_cost" type="number" min={0} defaultValue={0} /></FormField>
                <label className="flex items-center gap-2 text-sm font-medium text-slate-700"><input name="track_serial" type="checkbox" /> Lacak serial number</label>
                <button disabled={saving} className="h-10 rounded-md bg-blue-600 px-4 text-sm font-semibold text-white disabled:opacity-60">Simpan barang</button>
              </form>
            </Section>
          </div>
        )}

        {!loading && activeTab === "Aset Serial" && (
          <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_380px]">
            <Section title="Aset serial">
              <DataTable
                columns={["Serial", "Item", "MAC", "Lokasi", "Pelanggan", "Biaya", "Status", "Aksi"]}
                rows={assets.map((asset) => [
                  <span key={asset.id} className="font-mono font-semibold">{asset.serial_number}</span>,
                  asset.item_name || "-",
                  asset.mac_address || "-",
                  asset.location_type,
                  asset.assigned_customer_name || "-",
                  money(asset.purchase_cost),
                  <StatusBadge key={`${asset.id}-status`} status={asset.status} />,
                  <div key={`${asset.id}-actions`} className="flex flex-wrap gap-2">
                    <button
                      type="button"
                      disabled={saving || asset.status !== "assigned"}
                      onClick={() => void quickAssetAction(asset.id, "return")}
                      className="rounded-md px-2.5 py-1.5 text-xs font-semibold text-blue-700 hover:bg-blue-50 disabled:cursor-not-allowed disabled:text-slate-400"
                    >
                      Return
                    </button>
                    <button
                      type="button"
                      disabled={saving || asset.status === "damaged"}
                      onClick={() => void quickAssetAction(asset.id, "mark-damaged")}
                      className="rounded-md px-2.5 py-1.5 text-xs font-semibold text-red-700 hover:bg-red-50 disabled:cursor-not-allowed disabled:text-slate-400"
                    >
                      Rusak
                    </button>
                    <button
                      type="button"
                      disabled={saving || asset.status === "lost"}
                      onClick={() => void quickAssetAction(asset.id, "mark-lost")}
                      className="rounded-md px-2.5 py-1.5 text-xs font-semibold text-amber-700 hover:bg-amber-50 disabled:cursor-not-allowed disabled:text-slate-400"
                    >
                      Hilang
                    </button>
                    <button
                      type="button"
                      disabled={saving || asset.status === "rma"}
                      onClick={() => void quickAssetAction(asset.id, "mark-rma")}
                      className="rounded-md px-2.5 py-1.5 text-xs font-semibold text-violet-700 hover:bg-violet-50 disabled:cursor-not-allowed disabled:text-slate-400"
                    >
                      RMA
                    </button>
                    <button
                      type="button"
                      disabled={saving || asset.status === "retired"}
                      onClick={() => void quickAssetAction(asset.id, "retire")}
                      className="rounded-md px-2.5 py-1.5 text-xs font-semibold text-slate-700 hover:bg-slate-100 disabled:cursor-not-allowed disabled:text-slate-400"
                    >
                      Retire
                    </button>
                  </div>,
                ])}
              />
            </Section>
            <div className="space-y-6">
              <Section title="Tambah aset serial">
                <form onSubmit={submitAsset} className="grid gap-4">
                  <FormField label="Item">
                    <select name="item_id" required className="h-10 rounded-md border border-slate-300 bg-white px-3 text-sm">
                      <option value="">Pilih item serial</option>
                      {serialItems.map((item) => <option key={item.id} value={item.id}>{item.name}</option>)}
                    </select>
                  </FormField>
                  <FormField label="Serial number"><TextInput name="serial_number" required /></FormField>
                  <FormField label="MAC address"><TextInput name="mac_address" /></FormField>
                  <FormField label="Biaya beli"><TextInput name="purchase_cost" type="number" min={0} defaultValue={0} /></FormField>
                  <FormField label="Tanggal beli"><TextInput name="purchase_date" type="date" /></FormField>
                  <button disabled={saving} className="h-10 rounded-md bg-blue-600 px-4 text-sm font-semibold text-white disabled:opacity-60">Simpan aset</button>
                </form>
              </Section>
              <Section title="Assign aset">
                <form onSubmit={submitAssetAction} className="grid gap-4">
                  <FormField label="Aset">
                    <select name="asset_id" required className="h-10 rounded-md border border-slate-300 bg-white px-3 text-sm">
                      <option value="">Pilih aset</option>
                      {assets.map((asset) => <option key={asset.id} value={asset.id}>{asset.serial_number} - {asset.item_name || "Item"}</option>)}
                    </select>
                  </FormField>
                  <FormField label="Aksi">
                    <select name="action" className="h-10 rounded-md border border-slate-300 bg-white px-3 text-sm">
                      <option value="assign">Assign ke pelanggan</option>
                      <option value="return">Return ke gudang</option>
                      <option value="mark-damaged">Tandai rusak</option>
                      <option value="mark-lost">Tandai hilang</option>
                      <option value="mark-rma">Masuk RMA</option>
                      <option value="retire">Retire aset</option>
                    </select>
                  </FormField>
                  <FormField label="Pelanggan">
                    <select name="customer_id" className="h-10 rounded-md border border-slate-300 bg-white px-3 text-sm">
                      <option value="">Pilih pelanggan untuk assign</option>
                      {customers.map((customer) => <option key={customer.id} value={customer.id}>{customer.name} {customer.phone ? `- ${customer.phone}` : ""}</option>)}
                    </select>
                  </FormField>
                  <FormField label="Catatan"><TextInput name="notes" placeholder="Dipakai instalasi / perangkat dikembalikan" /></FormField>
                  <button disabled={saving} className="h-10 rounded-md bg-slate-900 px-4 text-sm font-semibold text-white disabled:opacity-60">Proses aset</button>
                </form>
              </Section>
            </div>
          </div>
        )}

        {!loading && activeTab === "Mutasi Stok" && (
          <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_380px]">
            <Section title="Riwayat mutasi">
              <DataTable
                columns={["Waktu", "Item", "Tipe", "Qty", "Biaya", "Catatan"]}
                rows={movements.map((movement) => [
                  dateID(movement.created_at),
                  movement.item_name || "-",
                  <StatusBadge key={`${movement.id}-type`} status={movement.movement_type} />,
                  String(movement.quantity),
                  money(movement.unit_cost),
                  movement.notes || "-",
                ])}
              />
            </Section>
            <Section title="Catat mutasi">
              <form onSubmit={submitMovement} className="grid gap-4">
                <FormField label="Item">
                  <select name="item_id" required className="h-10 rounded-md border border-slate-300 bg-white px-3 text-sm">
                    <option value="">Pilih item</option>
                    {items.map((item) => <option key={item.id} value={item.id}>{item.name}</option>)}
                  </select>
                </FormField>
                <FormField label="Tipe">
                  <select name="movement_type" className="h-10 rounded-md border border-slate-300 bg-white px-3 text-sm">
                    <option value="purchase">Stok masuk / beli</option>
                    <option value="install">Keluar instalasi</option>
                    <option value="return">Kembali</option>
                    <option value="damaged">Rusak</option>
                    <option value="lost">Hilang</option>
                    <option value="rma">RMA</option>
                    <option value="retired">Retired</option>
                    <option value="adjustment">Penyesuaian</option>
                  </select>
                </FormField>
                <FormField label="Aset serial">
                  <select name="asset_id" className="h-10 rounded-md border border-slate-300 bg-white px-3 text-sm">
                    <option value="">Opsional, wajib untuk item serial</option>
                    {assets.map((asset) => <option key={asset.id} value={asset.id}>{asset.serial_number} - {asset.item_name || "Item"}</option>)}
                  </select>
                </FormField>
                <FormField label="Jumlah"><TextInput name="quantity" type="number" defaultValue={1} /></FormField>
                <FormField label="Biaya satuan"><TextInput name="unit_cost" type="number" min={0} defaultValue={0} /></FormField>
                <FormField label="Kategori expense">
                  <select name="expense_category_id" className="h-10 rounded-md border border-slate-300 bg-white px-3 text-sm">
                    <option value="">Pilih bila stok masuk perlu menjadi pengeluaran</option>
                    {expenseCategories.map((category) => <option key={category.id} value={category.id}>{category.name}</option>)}
                  </select>
                </FormField>
                <label className="flex items-start gap-2 text-sm font-medium text-slate-700">
                  <input name="create_expense" type="checkbox" className="mt-1" />
                  Catat pembelian stok sebagai pengeluaran
                </label>
                <FormField label="Catatan"><TextInput name="notes" placeholder="Pembelian ONT / dipakai instalasi" /></FormField>
                <button disabled={saving} className="h-10 rounded-md bg-blue-600 px-4 text-sm font-semibold text-white disabled:opacity-60">Simpan mutasi</button>
              </form>
            </Section>
          </div>
        )}
      </div>
    </AppShell>
  );
}
