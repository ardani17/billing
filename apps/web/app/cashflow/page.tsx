"use client";

import { useCallback, useEffect, useState } from "react";
import AppShell from "../components/app-shell";
import { DataTable, EmptyState, FormField, PageHeader, Section, StatGrid, StatusBadge, TextInput } from "../components/ui";

type ApiEnvelope<T> = { success: boolean; data?: T; error?: { message?: string } };
type CashflowTransaction = {
  id: string;
  date: string;
  direction: string;
  source: string;
  category: string;
  description: string;
  amount: number;
};
type CashflowSummary = {
  opening_balance: number;
  total_cash_in: number;
  total_cash_out: number;
  net_cashflow: number;
  closing_balance_estimate: number;
  breakdown: { direction: string; source: string; category: string; amount: number }[];
  latest_transactions: CashflowTransaction[];
};
type TrendPoint = { date: string; cash_in: number; cash_out: number; net: number };

function monthStart() {
  const now = new Date();
  return new Date(now.getFullYear(), now.getMonth(), 1).toISOString().slice(0, 10);
}

function today() {
  return new Date().toISOString().slice(0, 10);
}

function money(value: number | null | undefined) {
  return new Intl.NumberFormat("id-ID", { style: "currency", currency: "IDR", maximumFractionDigits: 0 }).format(value ?? 0);
}

function dateID(value: string) {
  return new Intl.DateTimeFormat("id-ID", { dateStyle: "medium" }).format(new Date(value));
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

export default function CashflowPage() {
  const [periodStart, setPeriodStart] = useState(monthStart());
  const [periodEnd, setPeriodEnd] = useState(today());
  const [direction, setDirection] = useState("");
  const [source, setSource] = useState("");
  const [category, setCategory] = useState("");
  const [search, setSearch] = useState("");
  const [summary, setSummary] = useState<CashflowSummary | null>(null);
  const [transactions, setTransactions] = useState<CashflowTransaction[]>([]);
  const [trend, setTrend] = useState<TrendPoint[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");

  const params = new URLSearchParams({
    period_start: periodStart,
    period_end: periodEnd,
    direction,
    source,
    category,
    search,
  }).toString();
  const loadData = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const [nextSummary, nextTransactions, nextTrend] = await Promise.all([
        api<CashflowSummary>(`/cashflow/summary?${params}`),
        api<CashflowTransaction[]>(`/cashflow/transactions?${params}`),
        api<TrendPoint[]>(`/cashflow/trend?${params}`),
      ]);
      setSummary(nextSummary);
      setTransactions(nextTransactions);
      setTrend(nextTrend);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Gagal memuat arus kas");
    } finally {
      setLoading(false);
    }
  }, [params]);

  useEffect(() => {
    void loadData();
  }, [loadData]);

  const maxTrend = Math.max(...trend.map((point) => Math.max(point.cash_in, point.cash_out)), 1);
  const categories = Array.from(new Set([...(summary?.breakdown.map((item) => item.category) || []), ...transactions.map((tx) => tx.category)])).filter(Boolean);

  async function submitManual(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    setSaving(true);
    setError("");
    setSuccess("");
    try {
      await api<CashflowTransaction>("/cashflow/manual", {
        method: "POST",
        body: JSON.stringify({
          direction: String(form.get("direction") || "out"),
          category: String(form.get("manual_category") || ""),
          description: String(form.get("description") || ""),
          amount: Number(form.get("amount") || 0),
          transaction_date: String(form.get("transaction_date") || today()),
        }),
      });
      event.currentTarget.reset();
      setSuccess("Transaksi kas manual berhasil dicatat.");
      await loadData();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Gagal mencatat kas manual");
    } finally {
      setSaving(false);
    }
  }

  return (
    <AppShell>
      <div className="space-y-6">
        <PageHeader
          eyebrow="Keuangan"
          title="Arus Kas"
          description="Pantau uang masuk dan keluar berdasarkan transaksi kas, berbeda dari laba rugi."
          actions={<a href={`/api/billing/cashflow/export?${params}`} className="inline-flex h-10 items-center rounded-md bg-slate-900 px-4 text-sm font-semibold text-white">Export CSV</a>}
        />

        {error && <div className="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">{error}</div>}
        {success && <div className="rounded-lg border border-emerald-200 bg-emerald-50 px-4 py-3 text-sm text-emerald-700">{success}</div>}

        <Section title="Filter periode">
          <div className="grid gap-3 lg:grid-cols-6">
            <label className="grid gap-2 text-sm font-medium text-slate-700">Dari tanggal<input type="date" value={periodStart} onChange={(event) => setPeriodStart(event.target.value)} className="h-10 rounded-md border border-slate-300 px-3 text-sm" /></label>
            <label className="grid gap-2 text-sm font-medium text-slate-700">Sampai tanggal<input type="date" value={periodEnd} onChange={(event) => setPeriodEnd(event.target.value)} className="h-10 rounded-md border border-slate-300 px-3 text-sm" /></label>
            <label className="grid gap-2 text-sm font-medium text-slate-700">Arah
              <select value={direction} onChange={(event) => setDirection(event.target.value)} className="h-10 rounded-md border border-slate-300 bg-white px-3 text-sm">
                <option value="">Semua</option>
                <option value="in">Masuk</option>
                <option value="out">Keluar</option>
              </select>
            </label>
            <label className="grid gap-2 text-sm font-medium text-slate-700">Sumber
              <select value={source} onChange={(event) => setSource(event.target.value)} className="h-10 rounded-md border border-slate-300 bg-white px-3 text-sm">
                <option value="">Semua</option>
                <option value="pembayaran">Pembayaran</option>
                <option value="voucher">Voucher</option>
                <option value="pengeluaran">Pengeluaran</option>
                <option value="inventaris">Inventaris</option>
                <option value="reseller">Reseller</option>
                <option value="manual">Manual</option>
              </select>
            </label>
            <label className="grid gap-2 text-sm font-medium text-slate-700">Kategori
              <select value={category} onChange={(event) => setCategory(event.target.value)} className="h-10 rounded-md border border-slate-300 bg-white px-3 text-sm">
                <option value="">Semua</option>
                {categories.map((item) => <option key={item} value={item}>{item}</option>)}
              </select>
            </label>
            <label className="grid gap-2 text-sm font-medium text-slate-700">Cari<input value={search} onChange={(event) => setSearch(event.target.value)} className="h-10 rounded-md border border-slate-300 px-3 text-sm" placeholder="Deskripsi/sumber" /></label>
            <div className="grid content-end"><button type="button" onClick={() => void loadData()} className="h-10 rounded-md bg-blue-600 px-4 text-sm font-semibold text-white">Refresh</button></div>
          </div>
        </Section>

        <Section title="Catat kas manual" description="Untuk pemasukan atau pengeluaran yang tidak berasal dari invoice, reseller, voucher, atau inventory.">
          <form onSubmit={submitManual} className="grid gap-3 lg:grid-cols-[150px_180px_minmax(0,1fr)_170px_160px_120px]">
            <FormField label="Arah">
              <select name="direction" className="h-10 rounded-md border border-slate-300 bg-white px-3 text-sm">
                <option value="out">Keluar</option>
                <option value="in">Masuk</option>
              </select>
            </FormField>
            <FormField label="Kategori"><TextInput name="manual_category" required placeholder="Operasional" /></FormField>
            <FormField label="Deskripsi"><TextInput name="description" required placeholder="Keterangan transaksi" /></FormField>
            <FormField label="Nominal"><TextInput name="amount" type="number" min={1} required /></FormField>
            <FormField label="Tanggal"><TextInput name="transaction_date" type="date" defaultValue={today()} required /></FormField>
            <div className="grid content-end"><button disabled={saving} className="h-10 rounded-md bg-slate-900 px-4 text-sm font-semibold text-white disabled:opacity-60">Simpan</button></div>
          </form>
        </Section>

        {loading ? <div className="h-48 animate-pulse rounded-xl border border-slate-200 bg-white" /> : null}

        {!loading && summary && (
          <>
            <StatGrid
              stats={[
                { label: "Saldo awal estimasi", value: money(summary.opening_balance), tone: "slate" },
                { label: "Kas masuk", value: money(summary.total_cash_in), tone: "green" },
                { label: "Kas keluar", value: money(summary.total_cash_out), tone: "red" },
                { label: "Saldo akhir estimasi", value: money(summary.closing_balance_estimate), tone: summary.net_cashflow >= 0 ? "blue" : "amber" },
              ]}
            />

            <Section title="Trend harian" description="Bar hijau kas masuk, bar merah kas keluar.">
              {trend.length ? (
                <div className="grid gap-3">
                  {trend.map((point) => (
                    <div key={point.date} className="grid gap-2 md:grid-cols-[120px_minmax(0,1fr)_160px] md:items-center">
                      <span className="text-sm font-medium text-slate-600">{dateID(point.date)}</span>
                      <div className="grid gap-1">
                        <div className="h-2 rounded-full bg-slate-100"><div className="h-2 rounded-full bg-emerald-500" style={{ width: `${(point.cash_in / maxTrend) * 100}%` }} /></div>
                        <div className="h-2 rounded-full bg-slate-100"><div className="h-2 rounded-full bg-red-500" style={{ width: `${(point.cash_out / maxTrend) * 100}%` }} /></div>
                      </div>
                      <span className="text-right font-mono text-sm text-slate-700">{money(point.net)}</span>
                    </div>
                  ))}
                </div>
              ) : (
                <EmptyState title="Belum ada trend" description="Trend akan muncul setelah ada pembayaran atau pengeluaran." />
              )}
            </Section>

            <div className="grid gap-6 xl:grid-cols-[380px_minmax(0,1fr)]">
              <Section title="Breakdown">
                {summary.breakdown.length ? (
                  <div className="space-y-3">
                    {summary.breakdown.map((item) => (
                      <div key={`${item.direction}-${item.source}-${item.category}`} className="flex items-center justify-between gap-3 rounded-lg border border-slate-200 p-3">
                        <div>
                          <p className="text-sm font-semibold text-slate-900">{item.category}</p>
                          <p className="text-xs text-slate-500">{item.source}</p>
                        </div>
                        <div className="text-right">
                          <StatusBadge status={item.direction === "in" ? "masuk" : "keluar"} />
                          <p className="mt-1 font-mono text-sm font-semibold">{money(item.amount)}</p>
                        </div>
                      </div>
                    ))}
                  </div>
                ) : (
                  <EmptyState title="Belum ada breakdown" description="Data akan muncul setelah ada transaksi kas." />
                )}
              </Section>

              <Section title="Transaksi kas">
                {transactions.length ? (
                  <DataTable
                    columns={["Tanggal", "Arah", "Sumber", "Kategori", "Deskripsi", "Nominal"]}
                    rows={transactions.map((tx) => [
                      dateID(tx.date),
                      <StatusBadge key={`${tx.id}-dir`} status={tx.direction === "in" ? "masuk" : "keluar"} />,
                      tx.source,
                      tx.category,
                      tx.description || "-",
                      money(tx.amount),
                    ])}
                  />
                ) : (
                  <EmptyState title="Belum ada transaksi" description="Pembayaran dan pengeluaran pada periode ini akan tampil di sini." />
                )}
              </Section>
            </div>
          </>
        )}
      </div>
    </AppShell>
  );
}
