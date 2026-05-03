"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { ArrowClockwise, LockKey, SignOut, Storefront, Ticket, Wallet } from "@phosphor-icons/react";
import { DataTable, EmptyState, PageHeader, Section, StatGrid, StatusBadge, TextInput } from "../components/ui";

type ApiEnvelope<T> = {
  success: boolean;
  data?: T;
  error?: {
    code?: string;
    message?: string;
  };
};

type Reseller = {
  id: string;
  name: string;
  phone: string;
  email?: string;
  balance: number;
  daily_purchase_limit: number;
  status: string;
  last_login?: string;
};

type LoginResponse = {
  access_token: string;
  refresh_token: string;
  expires_in: number;
  reseller: Reseller;
};

type DashboardSummary = {
  balance: number;
  sold_today: number;
  available_vouchers: number;
};

type Pagination<T> = {
  data: T[];
  pagination?: {
    page: number;
    page_size: number;
    total: number;
    total_pages: number;
  };
};

type Voucher = {
  id: string;
  code: string;
  package_name?: string;
  status: string;
  sell_price_snapshot?: number | null;
  reseller_price_snapshot?: number | null;
  purchased_at?: string | null;
  expires_at?: string | null;
};

type Transaction = {
  id: string;
  type: string;
  amount: number;
  balance_before: number;
  balance_after: number;
  notes?: string;
  created_at: string;
};

type PortalData = {
  summary: DashboardSummary;
  vouchers: Pagination<Voucher>;
  transactions: Pagination<Transaction>;
  deposits: Pagination<Transaction>;
};

const ACCESS_TOKEN_KEY = "ispboss_reseller_access_token";
const REFRESH_TOKEN_KEY = "ispboss_reseller_refresh_token";

function formatRupiah(value: number | null | undefined) {
  return new Intl.NumberFormat("id-ID", {
    style: "currency",
    currency: "IDR",
    maximumFractionDigits: 0,
  }).format(value ?? 0);
}

function formatDate(value?: string | null) {
  if (!value) return "-";
  return new Intl.DateTimeFormat("id-ID", {
    day: "2-digit",
    month: "short",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  }).format(new Date(value));
}

function apiMessage(error: unknown) {
  return error instanceof Error ? error.message : "Terjadi kesalahan tidak dikenal";
}

async function resellerApi<T>(path: string, init?: RequestInit, token?: string) {
  const response = await fetch(`/api/reseller${path}`, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...(init?.headers || {}),
    },
    cache: "no-store",
  });

  const envelope = (await response.json()) as ApiEnvelope<T>;

  if (!response.ok || !envelope.success) {
    const message = envelope.error?.message || `Request reseller gagal (${response.status})`;
    const error = new Error(message);
    error.name = response.status === 401 ? "Unauthorized" : envelope.error?.code || "ResellerApiError";
    throw error;
  }

  return envelope.data as T;
}

export function ResellerPortalClient() {
  const [accessToken, setAccessToken] = useState("");
  const [refreshToken, setRefreshToken] = useState("");
  const [reseller, setReseller] = useState<Reseller | null>(null);
  const [data, setData] = useState<PortalData | null>(null);
  const [phone, setPhone] = useState("");
  const [password, setPassword] = useState("");
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");

  const stats = useMemo(() => {
    const summary = data?.summary;
    return [
      { label: "Saldo", value: formatRupiah(summary?.balance ?? reseller?.balance ?? 0), tone: "green" as const },
      { label: "Terjual hari ini", value: String(summary?.sold_today ?? 0), tone: "blue" as const },
      { label: "Voucher tersedia", value: String(summary?.available_vouchers ?? 0), tone: "amber" as const },
      { label: "Limit harian", value: String(reseller?.daily_purchase_limit ?? 0), tone: "slate" as const },
    ];
  }, [data?.summary, reseller]);

  const clearSession = useCallback(() => {
    localStorage.removeItem(ACCESS_TOKEN_KEY);
    localStorage.removeItem(REFRESH_TOKEN_KEY);
    setAccessToken("");
    setRefreshToken("");
    setReseller(null);
    setData(null);
  }, []);

  const loadPortalData = useCallback(
    async (token: string) => {
      setError("");
      const [summary, vouchers, transactions, deposits] = await Promise.all([
        resellerApi<DashboardSummary>("/dashboard", undefined, token),
        resellerApi<Pagination<Voucher>>("/vouchers?page_size=10&sort_order=desc", undefined, token),
        resellerApi<Pagination<Transaction>>("/history?page_size=10&sort_order=desc", undefined, token),
        resellerApi<Pagination<Transaction>>("/deposit?page_size=10&sort_order=desc", undefined, token),
      ]);

      setData({ summary, vouchers, transactions, deposits });
      setReseller((current) => (current ? { ...current, balance: summary.balance } : current));
    },
    [],
  );

  const restoreSession = useCallback(async () => {
    const storedAccess = localStorage.getItem(ACCESS_TOKEN_KEY) || "";
    const storedRefresh = localStorage.getItem(REFRESH_TOKEN_KEY) || "";

    if (!storedAccess) {
      setLoading(false);
      return;
    }

    setAccessToken(storedAccess);
    setRefreshToken(storedRefresh);

    try {
      await loadPortalData(storedAccess);
    } catch (err) {
      if ((err as Error).name === "Unauthorized" && storedRefresh) {
        try {
          const refreshed = await resellerApi<LoginResponse>(
            "/auth/refresh",
            { method: "POST", body: JSON.stringify({ refresh_token: storedRefresh }) },
          );
          localStorage.setItem(ACCESS_TOKEN_KEY, refreshed.access_token);
          localStorage.setItem(REFRESH_TOKEN_KEY, refreshed.refresh_token);
          setAccessToken(refreshed.access_token);
          setRefreshToken(refreshed.refresh_token);
          setReseller(refreshed.reseller);
          await loadPortalData(refreshed.access_token);
          return;
        } catch {
          clearSession();
        }
      } else {
        setError(apiMessage(err));
      }
    } finally {
      setLoading(false);
    }
  }, [clearSession, loadPortalData]);

  useEffect(() => {
    void restoreSession();
  }, [restoreSession]);

  async function handleLogin(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSubmitting(true);
    setError("");

    try {
      const response = await resellerApi<LoginResponse>("/auth/login", {
        method: "POST",
        body: JSON.stringify({ phone, password }),
      });

      localStorage.setItem(ACCESS_TOKEN_KEY, response.access_token);
      localStorage.setItem(REFRESH_TOKEN_KEY, response.refresh_token);
      setAccessToken(response.access_token);
      setRefreshToken(response.refresh_token);
      setReseller(response.reseller);
      await loadPortalData(response.access_token);
    } catch (err) {
      setError(apiMessage(err));
    } finally {
      setSubmitting(false);
    }
  }

  async function handleLogout() {
    const token = accessToken;
    const refresh = refreshToken;
    clearSession();

    if (token && refresh) {
      try {
        await resellerApi("/auth/logout", {
          method: "POST",
          body: JSON.stringify({ refresh_token: refresh }),
        }, token);
      } catch {
        // Session already cleared locally; backend cleanup can fail if token expired.
      }
    }
  }

  async function handleRefresh() {
    if (!accessToken) return;
    setLoading(true);
    try {
      await loadPortalData(accessToken);
    } catch (err) {
      if ((err as Error).name === "Unauthorized") clearSession();
      setError(apiMessage(err));
    } finally {
      setLoading(false);
    }
  }

  if (loading) {
    return (
      <main className="min-h-[100dvh] bg-slate-50 px-4 py-8 text-slate-950 sm:px-6 lg:px-8">
        <div className="mx-auto max-w-6xl rounded-xl border border-slate-200 bg-white p-8 shadow-sm">
          <p className="text-sm font-semibold text-slate-500">Memuat portal reseller...</p>
        </div>
      </main>
    );
  }

  if (!accessToken) {
    return (
      <main className="min-h-[100dvh] bg-slate-950 px-4 py-8 text-white sm:px-6 lg:px-8">
        <div className="mx-auto grid min-h-[calc(100dvh-4rem)] max-w-5xl items-center gap-8 lg:grid-cols-[1fr_420px]">
          <section className="min-w-0 space-y-5">
            <div className="inline-flex h-12 w-12 items-center justify-center rounded-xl bg-blue-500 text-white">
              <Storefront size={26} weight="duotone" />
            </div>
            <div>
              <p className="text-xs font-semibold uppercase tracking-[0.18em] text-blue-300">Portal Reseller</p>
              <h1 className="mt-3 max-w-2xl text-3xl font-semibold tracking-tight text-white sm:text-5xl">
                Kelola saldo, voucher, dan riwayat transaksi reseller.
              </h1>
              <p className="mt-4 max-w-2xl text-sm leading-6 text-slate-300 sm:text-base">
                Login memakai akun reseller yang dibuat oleh admin ISP. Data yang tampil di sini langsung dari API reseller,
                bukan data contoh.
              </p>
            </div>
          </section>

          <form onSubmit={handleLogin} className="rounded-xl border border-white/10 bg-white p-5 text-slate-950 shadow-2xl">
            <div className="mb-5 flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-blue-50 text-blue-700">
                <LockKey size={22} weight="duotone" />
              </div>
              <div>
                <h2 className="text-lg font-semibold">Masuk reseller</h2>
                <p className="text-sm text-slate-500">Gunakan nomor HP dan password reseller.</p>
              </div>
            </div>

            <div className="space-y-4">
              <label className="grid gap-2">
                <span className="text-sm font-medium">Nomor HP</span>
                <TextInput
                  value={phone}
                  onChange={(event) => setPhone(event.target.value)}
                  placeholder="08xxxxxxxxxx"
                  autoComplete="tel"
                  required
                />
              </label>
              <label className="grid gap-2">
                <span className="text-sm font-medium">Password</span>
                <TextInput
                  value={password}
                  onChange={(event) => setPassword(event.target.value)}
                  type="password"
                  autoComplete="current-password"
                  required
                />
              </label>

              {error && (
                <div className="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
                  {error}
                </div>
              )}

              <button
                type="submit"
                disabled={submitting}
                className="inline-flex h-11 w-full items-center justify-center rounded-md bg-blue-600 px-4 text-sm font-semibold text-white transition hover:bg-blue-700 disabled:cursor-not-allowed disabled:opacity-60"
              >
                {submitting ? "Memproses..." : "Masuk"}
              </button>
            </div>
          </form>
        </div>
      </main>
    );
  }

  return (
    <main className="min-h-[100dvh] bg-slate-50 px-4 py-6 text-slate-950 sm:px-6 lg:px-8">
      <div className="mx-auto max-w-7xl space-y-6">
        <PageHeader
          eyebrow="Portal Reseller"
          title={reseller?.name || "Dashboard reseller"}
          description={`Akun ${reseller?.phone || "-"}${reseller?.email ? ` - ${reseller.email}` : ""}`}
          actions={
            <>
              <button
                type="button"
                onClick={handleRefresh}
                className="inline-flex h-10 items-center gap-2 rounded-md border border-slate-300 bg-white px-4 text-sm font-semibold text-slate-700 transition hover:bg-slate-50"
              >
                <ArrowClockwise size={18} weight="bold" />
                Refresh
              </button>
              <button
                type="button"
                onClick={handleLogout}
                className="inline-flex h-10 items-center gap-2 rounded-md bg-slate-900 px-4 text-sm font-semibold text-white transition hover:bg-slate-800"
              >
                <SignOut size={18} weight="bold" />
                Keluar
              </button>
            </>
          }
        />

        {error && (
          <div className="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
            {error}
          </div>
        )}

        <StatGrid stats={stats} />

        <div className="grid gap-6 xl:grid-cols-[minmax(0,1.3fr)_minmax(320px,0.7fr)]">
          <Section
            title="Voucher saya"
            description="Daftar voucher yang sudah dibeli atau dialokasikan untuk reseller ini."
            action={
              <span className="inline-flex items-center gap-2 text-sm font-semibold text-slate-500">
                <Ticket size={18} weight="duotone" />
                {data?.vouchers.pagination?.total ?? data?.vouchers.data.length ?? 0} voucher
              </span>
            }
          >
            {data?.vouchers.data.length ? (
              <DataTable
                columns={["Kode", "Paket", "Harga", "Status", "Dibeli", "Expired"]}
                rows={data.vouchers.data.map((voucher) => [
                  <span key={voucher.id} className="font-mono font-semibold text-slate-950">{voucher.code}</span>,
                  voucher.package_name || "-",
                  formatRupiah(voucher.sell_price_snapshot ?? voucher.reseller_price_snapshot),
                  <StatusBadge key={`${voucher.id}-status`} status={voucher.status} />,
                  formatDate(voucher.purchased_at),
                  formatDate(voucher.expires_at),
                ])}
              />
            ) : (
              <EmptyState
                title="Belum ada voucher"
                description="Voucher akan tampil setelah admin mengalokasikan voucher atau reseller melakukan pembelian."
              />
            )}
          </Section>

          <Section
            title="Riwayat deposit"
            description="Mutasi saldo masuk reseller."
            action={
              <span className="inline-flex items-center gap-2 text-sm font-semibold text-slate-500">
                <Wallet size={18} weight="duotone" />
                Live
              </span>
            }
          >
            {data?.deposits.data.length ? (
              <div className="space-y-3">
                {data.deposits.data.map((tx) => (
                  <div key={tx.id} className="rounded-lg border border-slate-200 p-3">
                    <div className="flex min-w-0 items-start justify-between gap-3">
                      <div className="min-w-0">
                        <p className="font-semibold text-slate-950">{formatRupiah(tx.amount)}</p>
                        <p className="mt-1 text-xs text-slate-500">{formatDate(tx.created_at)}</p>
                      </div>
                      <StatusBadge status={tx.type} />
                    </div>
                    {tx.notes && <p className="mt-2 text-sm text-slate-500 [overflow-wrap:anywhere]">{tx.notes}</p>}
                  </div>
                ))}
              </div>
            ) : (
              <EmptyState
                title="Belum ada deposit"
                description="Riwayat top up reseller akan muncul ketika admin menambah saldo."
              />
            )}
          </Section>
        </div>

        <Section title="Riwayat transaksi" description="Pembelian, refund, deposit, atau penarikan saldo reseller.">
          {data?.transactions.data.length ? (
            <DataTable
              columns={["Waktu", "Tipe", "Nominal", "Saldo Akhir", "Catatan"]}
              rows={data.transactions.data.map((tx) => [
                formatDate(tx.created_at),
                <StatusBadge key={`${tx.id}-type`} status={tx.type} />,
                formatRupiah(tx.amount),
                formatRupiah(tx.balance_after),
                tx.notes || "-",
              ])}
            />
          ) : (
            <EmptyState
              title="Belum ada transaksi"
              description="Aktivitas reseller akan muncul otomatis setelah ada deposit atau pembelian voucher."
            />
          )}
        </Section>
      </div>
    </main>
  );
}
