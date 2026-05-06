"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { ArrowClockwise, DownloadSimple, WarningCircle } from "@phosphor-icons/react";
import AppShell from "../../components/app-shell";
import {
  fetchAgingReport,
  fetchExpenses,
  fetchPaymentReport,
  fetchProfitLossReport,
  fetchRevenueReport,
  fetchVoucherReport,
  requestExport,
} from "../lib/api";
import type { AgingReport, Expense, PaymentReport, ProfitLossReport, ReportFilter, RevenueReport, VoucherRevenueReport } from "../lib/types";
import { formatCurrency, formatNumber, formatPercentage } from "../lib/formatters";

type AreaOption = { id: string; name: string };
type InvoiceRow = { id: string; customer_id?: string; total_amount?: number };
type CreditNoteRow = { amount?: number; created_at?: string };
type DebitNoteRow = { total_amount?: number; created_at?: string };

type ReconciliationData = {
  revenue: RevenueReport;
  aging: AgingReport;
  payments: PaymentReport;
  vouchers: VoucherRevenueReport;
  profitLoss: ProfitLossReport;
  expenses: Expense[];
  creditNoteImpact: number;
  debitNoteImpact: number;
  noteWarning?: string;
};

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "/api/billing";

function monthStart() {
  const now = new Date();
  return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, "0")}-01`;
}

function today() {
  return new Date().toISOString().slice(0, 10);
}

async function fetchBillingJSON<T>(path: string): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, { cache: "no-store" });
  if (!res.ok) {
    const body = await res.text();
    throw new Error(`API error ${res.status}: ${body}`);
  }
  const payload = await res.json();
  if (payload && typeof payload === "object" && "success" in payload && "data" in payload) {
    return payload.data as T;
  }
  return payload as T;
}

function listOf<T>(payload: unknown): T[] {
  if (Array.isArray(payload)) return payload as T[];
  if (payload && typeof payload === "object") {
    const obj = payload as { data?: unknown; items?: unknown };
    if (Array.isArray(obj.data)) return obj.data as T[];
    if (Array.isArray(obj.items)) return obj.items as T[];
  }
  return [];
}

function dateInPeriod(date: string | undefined, start: string, end: string) {
  if (!date) return true;
  const value = date.slice(0, 10);
  return value >= start && value <= end;
}

async function fetchAreas(): Promise<AreaOption[]> {
  const payload = await fetchBillingJSON<unknown>("/areas?page_size=100");
  return listOf<AreaOption>(payload);
}

async function fetchAdjustmentImpact(periodStart: string, periodEnd: string) {
  try {
    const invoicePayload = await fetchBillingJSON<unknown>("/invoices?page_size=50");
    const invoices = listOf<InvoiceRow>(invoicePayload);
    const invoiceIds = invoices.map((invoice) => invoice.id).filter(Boolean);
    const customerIds = Array.from(new Set(invoices.map((invoice) => invoice.customer_id).filter(Boolean))) as string[];

    const creditResults = await Promise.allSettled(
      invoiceIds.map((invoiceID) => fetchBillingJSON<unknown>(`/credit-notes?invoice_id=${encodeURIComponent(invoiceID)}`)),
    );
    const debitResults = await Promise.allSettled(
      customerIds.map((customerID) => fetchBillingJSON<unknown>(`/debit-notes?customer_id=${encodeURIComponent(customerID)}`)),
    );

    const creditNoteImpact = creditResults
      .flatMap((result) => (result.status === "fulfilled" ? listOf<CreditNoteRow>(result.value) : []))
      .filter((note) => dateInPeriod(note.created_at, periodStart, periodEnd))
      .reduce((sum, note) => sum + Number(note.amount ?? 0), 0);

    const debitNoteImpact = debitResults
      .flatMap((result) => (result.status === "fulfilled" ? listOf<DebitNoteRow>(result.value) : []))
      .filter((note) => dateInPeriod(note.created_at, periodStart, periodEnd))
      .reduce((sum, note) => sum + Number(note.total_amount ?? 0), 0);

    return { creditNoteImpact, debitNoteImpact };
  } catch (error) {
    return {
      creditNoteImpact: 0,
      debitNoteImpact: 0,
      noteWarning: error instanceof Error ? error.message : "Gagal membaca credit/debit note",
    };
  }
}

export default function ReconciliationPage() {
  const [periodStart, setPeriodStart] = useState(monthStart);
  const [periodEnd, setPeriodEnd] = useState(today);
  const [areaId, setAreaId] = useState("");
  const [areas, setAreas] = useState<AreaOption[]>([]);
  const [data, setData] = useState<ReconciliationData | null>(null);
  const [loading, setLoading] = useState(true);
  const [exporting, setExporting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [exportMessage, setExportMessage] = useState<string | null>(null);

  const filter = useMemo<ReportFilter>(
    () => ({
      period_start: periodStart,
      period_end: periodEnd,
      ...(areaId ? { area_id: areaId } : {}),
    }),
    [areaId, periodEnd, periodStart],
  );

  const loadData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [areaRows, revenue, aging, payments, vouchers, profitLoss, expenses, adjustments] = await Promise.all([
        fetchAreas(),
        fetchRevenueReport(filter),
        fetchAgingReport(filter),
        fetchPaymentReport(filter),
        fetchVoucherReport(filter),
        fetchProfitLossReport(filter),
        fetchExpenses(periodStart, periodEnd),
        fetchAdjustmentImpact(periodStart, periodEnd),
      ]);
      setAreas(areaRows);
      setData({
        revenue,
        aging,
        payments,
        vouchers,
        profitLoss,
        expenses,
        creditNoteImpact: adjustments.creditNoteImpact,
        debitNoteImpact: adjustments.debitNoteImpact,
        noteWarning: adjustments.noteWarning,
      });
    } catch (err) {
      setError(err instanceof Error ? err.message : "Gagal memuat rekonsiliasi");
    } finally {
      setLoading(false);
    }
  }, [filter, periodEnd, periodStart]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  const metrics = useMemo(() => {
    if (!data) return null;
    const collected = data.payments.methods.reduce((sum, method) => sum + method.total_amount, 0);
    const invoiceMeasured = data.revenue.current.total + data.aging.total_outstanding + data.creditNoteImpact - data.debitNoteImpact;
    const expenseTotal = data.profitLoss.total_expenses;
    const resellerMargin = data.vouchers.total_reseller_margin;
    const netCollection = collected - expenseTotal - resellerMargin - data.creditNoteImpact + data.debitNoteImpact;
    const anomaly: string[] = [];

    if (Math.abs(data.revenue.current.total - data.profitLoss.total_revenue) > 1) {
      anomaly.push("Total revenue report berbeda dengan total revenue laba rugi.");
    }
    if (data.aging.total_outstanding > 0) {
      anomaly.push("Masih ada piutang terbuka pada akhir periode.");
    }
    if (collected > invoiceMeasured && invoiceMeasured > 0) {
      anomaly.push("Pembayaran tercatat lebih besar dari tagihan terukur periode ini.");
    }
    if (areaId) {
      anomaly.push("Expense masih bersifat tenant-wide karena skema database belum menyimpan area_id pengeluaran.");
    }
    if (data.noteWarning) {
      anomaly.push("Credit/debit note impact terbatas: " + data.noteWarning);
    }

    return { collected, invoiceMeasured, expenseTotal, resellerMargin, netCollection, anomaly };
  }, [areaId, data]);

  const handleExport = async () => {
    setExporting(true);
    setExportMessage(null);
    try {
      const result = await requestExport({ report_type: "profit_loss", format: "xlsx", filters: filter });
      setExportMessage(`Export laba rugi dibuat dengan job ${result.job_id}.`);
    } catch (err) {
      setExportMessage(err instanceof Error ? err.message : "Gagal membuat export");
    } finally {
      setExporting(false);
    }
  };

  return (
    <AppShell>
      <div className="space-y-6">
        <section className="rounded-lg border border-slate-200 bg-white p-5 shadow-sm shadow-slate-200/70">
          <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
            <div>
              <p className="text-xs font-semibold uppercase tracking-[0.16em] text-blue-700">Keuangan</p>
              <h1 className="mt-2 text-2xl font-semibold tracking-tight text-slate-950">Rekonsiliasi keuangan</h1>
              <p className="mt-2 max-w-2xl text-sm leading-6 text-slate-500">
                Cocokkan pendapatan, pembayaran, piutang, expense, voucher, dan penyesuaian invoice dalam satu periode.
              </p>
            </div>
            <div className="grid gap-3 md:grid-cols-[1fr_1fr_1fr_auto_auto]">
              <FilterDate label="Dari" value={periodStart} onChange={setPeriodStart} />
              <FilterDate label="Sampai" value={periodEnd} onChange={setPeriodEnd} />
              <label className="text-sm font-medium text-slate-700">
                Area
                <select
                  value={areaId}
                  onChange={(event) => setAreaId(event.target.value)}
                  className="mt-1 h-11 w-full rounded-md border border-slate-300 px-3 text-sm outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-100"
                >
                  <option value="">Semua area</option>
                  {areas.map((area) => (
                    <option key={area.id} value={area.id}>
                      {area.name}
                    </option>
                  ))}
                </select>
              </label>
              <button
                type="button"
                onClick={loadData}
                className="mt-auto inline-flex h-11 items-center justify-center gap-2 rounded-md border border-slate-300 px-4 text-sm font-semibold text-slate-700 hover:bg-slate-50"
              >
                <ArrowClockwise size={17} />
                Muat
              </button>
              <button
                type="button"
                onClick={handleExport}
                disabled={exporting}
                className="mt-auto inline-flex h-11 items-center justify-center gap-2 rounded-md bg-slate-950 px-4 text-sm font-semibold text-white hover:bg-slate-800 disabled:cursor-wait disabled:opacity-60"
              >
                <DownloadSimple size={17} />
                Export
              </button>
            </div>
          </div>
        </section>

        {error && <div className="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">{error}</div>}
        {exportMessage && <div className="rounded-lg border border-blue-200 bg-blue-50 px-4 py-3 text-sm text-blue-700">{exportMessage}</div>}

        {loading || !data || !metrics ? (
          <div className="h-56 animate-pulse rounded-lg border border-slate-200 bg-slate-100" />
        ) : (
          <>
            <section className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
              <MetricCard label="Tagihan terukur" value={formatCurrency(metrics.invoiceMeasured)} sub="Revenue + piutang + note net" />
              <MetricCard label="Pembayaran diterima" value={formatCurrency(metrics.collected)} sub={`${formatNumber(data.payments.methods.reduce((sum, method) => sum + method.transaction_count, 0))} transaksi`} />
              <MetricCard label="Piutang akhir" value={formatCurrency(data.aging.total_outstanding)} sub={`Collection ${formatPercentage(data.aging.collection_rate)}`} />
              <MetricCard label="Net collection" value={formatCurrency(metrics.netCollection)} sub="Pembayaran - expense - margin - credit + debit" />
            </section>

            <section className="grid gap-4 xl:grid-cols-[1.2fr_0.8fr]">
              <div className="rounded-lg border border-slate-200 bg-white p-5">
                <h2 className="text-base font-semibold text-slate-950">Breakdown rekonsiliasi</h2>
                <div className="mt-4 overflow-x-auto">
                  <table className="min-w-full text-left text-sm">
                    <thead className="text-xs uppercase tracking-[0.12em] text-slate-400">
                      <tr>
                        <th className="border-b border-slate-200 py-2 pr-4">Komponen</th>
                        <th className="border-b border-slate-200 py-2 text-right">Nominal</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-100">
                      <BreakdownRow label="Revenue tertagih" value={data.revenue.current.total} />
                      <BreakdownRow label="Piutang akhir" value={data.aging.total_outstanding} />
                      <BreakdownRow label="Credit note" value={-data.creditNoteImpact} />
                      <BreakdownRow label="Debit note" value={data.debitNoteImpact} />
                      <BreakdownRow label="Pembayaran diterima" value={metrics.collected} />
                      <BreakdownRow label="Expense periode" value={-metrics.expenseTotal} />
                      <BreakdownRow label="Margin reseller voucher" value={-metrics.resellerMargin} />
                      <BreakdownRow label="Voucher revenue" value={data.vouchers.total_revenue} />
                    </tbody>
                  </table>
                </div>
              </div>

              <div className="rounded-lg border border-slate-200 bg-white p-5">
                <h2 className="text-base font-semibold text-slate-950">Anomali dan catatan</h2>
                <div className="mt-4 grid gap-3">
                  {metrics.anomaly.length === 0 ? (
                    <p className="rounded-md border border-emerald-200 bg-emerald-50 px-3 py-2 text-sm text-emerald-700">
                      Tidak ada anomali dasar pada periode ini.
                    </p>
                  ) : (
                    metrics.anomaly.map((item) => (
                      <div key={item} className="flex gap-2 rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-800">
                        <WarningCircle className="mt-0.5 h-4 w-4 shrink-0" />
                        <span>{item}</span>
                      </div>
                    ))
                  )}
                </div>
              </div>
            </section>
          </>
        )}
      </div>
    </AppShell>
  );
}

function FilterDate({ label, value, onChange }: { label: string; value: string; onChange: (value: string) => void }) {
  return (
    <label className="text-sm font-medium text-slate-700">
      {label}
      <input
        type="date"
        value={value}
        onChange={(event) => onChange(event.target.value)}
        className="mt-1 h-11 w-full rounded-md border border-slate-300 px-3 text-sm outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-100"
      />
    </label>
  );
}

function MetricCard({ label, value, sub }: { label: string; value: string; sub: string }) {
  return (
    <div className="rounded-lg border border-slate-200 bg-white p-4">
      <p className="text-xs font-semibold uppercase tracking-[0.14em] text-slate-400">{label}</p>
      <p className="mt-3 text-2xl font-semibold tracking-tight text-slate-950">{value}</p>
      <p className="mt-2 text-sm text-slate-500">{sub}</p>
    </div>
  );
}

function BreakdownRow({ label, value }: { label: string; value: number }) {
  return (
    <tr>
      <td className="py-3 pr-4 text-slate-600">{label}</td>
      <td className={`py-3 text-right font-semibold ${value < 0 ? "text-red-600" : "text-slate-950"}`}>{formatCurrency(value)}</td>
    </tr>
  );
}
