"use client";

import { useEffect, useMemo, useState } from "react";
import { CreditCard, MagnifyingGlass, Phone, ShieldWarning } from "@phosphor-icons/react";
import { DataTable, EmptyState, StatusBadge, TextInput } from "../../components/ui";

type ApiEnvelope<T> = {
  success: boolean;
  data?: T;
  error?: {
    code?: string;
    message?: string;
  };
};

type OpenInvoice = {
  id: string;
  invoice_number: string;
  period_month: number;
  period_year: number;
  total_amount: number;
  paid_amount: number;
  remaining_amount: number;
  status: string;
  due_date: string;
};

type WalledGardenInfo = {
  payment_url?: string;
  total_arrears: number;
  invoices: OpenInvoice[];
  customer_name: string;
};

const demoCustomerId = "39260476-d6a6-47f9-90a6-25a7e37fbfaf";

function formatRupiah(value: number) {
  return new Intl.NumberFormat("id-ID", {
    style: "currency",
    currency: "IDR",
    maximumFractionDigits: 0,
  }).format(value);
}

function formatDate(value: string) {
  return new Intl.DateTimeFormat("id-ID", {
    day: "2-digit",
    month: "short",
    year: "numeric",
  }).format(new Date(value));
}

function monthName(month: number) {
  return new Intl.DateTimeFormat("id-ID", { month: "long" }).format(new Date(2026, month - 1, 1));
}

async function fetchPaymentInfo(customerId: string) {
  const response = await fetch(`/api/public/walled-garden/${customerId}/payment-info`, { cache: "no-store" });
  const envelope = (await response.json()) as ApiEnvelope<WalledGardenInfo>;

  if (!response.ok || !envelope.success) {
    throw new Error(envelope.error?.message || "Gagal mengambil info pembayaran");
  }

  return envelope.data as WalledGardenInfo;
}

export function WalledGardenLiveClient() {
  const [customerId, setCustomerId] = useState("");
  const [info, setInfo] = useState<WalledGardenInfo | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const hasArrears = (info?.total_arrears ?? 0) > 0;

  const title = useMemo(() => {
    if (!info) return "Cek tagihan pelanggan";
    return hasArrears ? "Layanan sementara dibatasi" : "Tidak ada tunggakan aktif";
  }, [hasArrears, info]);

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const id = params.get("customer_id");
    if (id) {
      setCustomerId(id);
      void load(id);
    }
  }, []);

  async function load(id: string) {
    setLoading(true);
    setError("");
    try {
      const data = await fetchPaymentInfo(id.trim());
      setInfo(data);
    } catch (err) {
      setInfo(null);
      setError(err instanceof Error ? err.message : "Gagal mengambil info pembayaran");
    } finally {
      setLoading(false);
    }
  }

  function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!customerId.trim()) return;
    void load(customerId);
  }

  return (
    <main className="min-h-[100dvh] bg-slate-950 px-4 py-6 text-white sm:px-6 lg:px-8">
      <div className="mx-auto grid min-h-[calc(100dvh-3rem)] max-w-6xl items-center gap-8 lg:grid-cols-[0.9fr_1.1fr]">
        <section className="min-w-0">
          <div className={`inline-flex h-12 w-12 items-center justify-center rounded-xl ${hasArrears ? "bg-amber-400 text-slate-950" : "bg-emerald-400 text-slate-950"}`}>
            <ShieldWarning size={26} weight="duotone" />
          </div>
          <p className="mt-6 text-xs font-semibold uppercase tracking-[0.18em] text-amber-200">
            Walled Garden
          </p>
          <h1 className="mt-3 max-w-2xl text-3xl font-semibold tracking-tight sm:text-5xl">
            {title}
          </h1>
          <p className="mt-5 max-w-xl text-sm leading-6 text-slate-300 sm:text-base">
            Halaman ini membaca data pembayaran publik dari billing-api. Router MikroTik dapat mengarahkan pelanggan
            isolir ke URL ini dengan parameter customer_id.
          </p>

          <form onSubmit={handleSubmit} className="mt-7 grid max-w-xl gap-3 rounded-xl border border-white/10 bg-white/5 p-3 sm:grid-cols-[minmax(0,1fr)_auto]">
            <TextInput
              value={customerId}
              onChange={(event) => setCustomerId(event.target.value)}
              placeholder={demoCustomerId}
              aria-label="Customer ID"
            />
            <button
              type="submit"
              disabled={loading || !customerId.trim()}
              className="inline-flex h-10 items-center justify-center gap-2 rounded-md bg-blue-600 px-4 text-sm font-semibold text-white transition hover:bg-blue-700 disabled:cursor-not-allowed disabled:opacity-60"
            >
              <MagnifyingGlass size={18} weight="bold" />
              Cek
            </button>
          </form>

          {error && (
            <div className="mt-4 max-w-xl rounded-lg border border-red-400/30 bg-red-500/10 px-4 py-3 text-sm text-red-100">
              {error}
            </div>
          )}
        </section>

        <section className="rounded-xl border border-white/10 bg-white p-4 text-slate-950 shadow-2xl sm:p-6">
          <div className="mb-5 flex flex-col gap-4 border-b border-slate-200 pb-5 sm:flex-row sm:items-center sm:justify-between">
            <div className="min-w-0">
              <p className="text-sm text-slate-500">Pelanggan</p>
              <h2 className="mt-1 text-xl font-semibold [overflow-wrap:anywhere]">
                {info?.customer_name || "Belum dipilih"}
              </h2>
            </div>
            <div className="rounded-lg bg-slate-50 px-4 py-3">
              <p className="text-xs font-semibold uppercase tracking-[0.14em] text-slate-400">Total tunggakan</p>
              <p className={`mt-1 font-mono text-2xl font-semibold ${hasArrears ? "text-amber-700" : "text-emerald-700"}`}>
                {formatRupiah(info?.total_arrears ?? 0)}
              </p>
            </div>
          </div>

          {!info ? (
            <EmptyState
              title="Masukkan Customer ID"
              description="Gunakan ID pelanggan dari halaman pelanggan untuk melihat status walled garden live."
            />
          ) : info.invoices.length ? (
            <div className="space-y-5">
              <DataTable
                columns={["Invoice", "Periode", "Sisa", "Jatuh Tempo", "Status"]}
                rows={info.invoices.map((invoice) => [
                  <span key={invoice.id} className="font-mono font-semibold text-slate-950">{invoice.invoice_number}</span>,
                  `${monthName(invoice.period_month)} ${invoice.period_year}`,
                  formatRupiah(invoice.remaining_amount),
                  formatDate(invoice.due_date),
                  <StatusBadge key={`${invoice.id}-status`} status={invoice.status} />,
                ])}
              />

              <div className="flex flex-col gap-3 sm:flex-row">
                {info.payment_url ? (
                  <a
                    href={info.payment_url}
                    className="inline-flex h-11 items-center justify-center gap-2 rounded-md bg-blue-600 px-4 text-sm font-semibold text-white transition hover:bg-blue-700"
                  >
                    <CreditCard size={19} weight="bold" />
                    Bayar sekarang
                  </a>
                ) : (
                  <button
                    type="button"
                    disabled
                    className="inline-flex h-11 items-center justify-center gap-2 rounded-md bg-slate-200 px-4 text-sm font-semibold text-slate-500"
                  >
                    <CreditCard size={19} weight="bold" />
                    Payment link belum aktif
                  </button>
                )}
                <a
                  href="tel:+6281234567890"
                  className="inline-flex h-11 items-center justify-center gap-2 rounded-md border border-slate-300 px-4 text-sm font-semibold text-slate-700 transition hover:bg-slate-50"
                >
                  <Phone size={19} weight="bold" />
                  Hubungi admin
                </a>
              </div>
            </div>
          ) : (
            <EmptyState
              title="Tidak ada invoice terbuka"
              description="Pelanggan ini tidak memiliki tunggakan aktif di billing-api."
            />
          )}
        </section>
      </div>
    </main>
  );
}
