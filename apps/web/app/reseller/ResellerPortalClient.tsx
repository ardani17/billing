"use client";

import { useCallback, useEffect, useMemo, useState, type FormEvent } from "react";
import {
  ArrowClockwise,
  CaretLeft,
  CaretRight,
  Copy,
  DownloadSimple,
  LockKey,
  MagnifyingGlass,
  Printer,
  SignOut,
  Storefront,
  Ticket,
  UserCircle,
  Wallet,
} from "@phosphor-icons/react";
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

type VoucherPackage = {
  id: string;
  name: string;
  is_active: boolean;
  download_mbps: number;
  upload_mbps: number;
  sell_price?: number | null;
  reseller_price?: number | null;
  duration_value?: number | null;
  duration_unit?: string | null;
};

type Voucher = {
  id: string;
  code: string;
  package_id?: string;
  package_name?: string;
  username?: string;
  password?: string;
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

type BuyVoucherResult = {
  vouchers: Voucher[];
  total_cost: number;
  balance_after: number;
};

type PortalData = {
  summary: DashboardSummary;
  vouchers: Pagination<Voucher>;
  transactions: Pagination<Transaction>;
  deposits: Pagination<Transaction>;
  packages: Pagination<VoucherPackage>;
};

type TabKey = "summary" | "buy" | "vouchers" | "transactions";

const ACCESS_TOKEN_KEY = "ispboss_reseller_access_token";
const REFRESH_TOKEN_KEY = "ispboss_reseller_refresh_token";
const RESELLER_PROFILE_KEY = "ispboss_reseller_profile";

const tabs: { id: TabKey; label: string }[] = [
  { id: "summary", label: "Ringkasan" },
  { id: "buy", label: "Beli voucher" },
  { id: "vouchers", label: "Voucher" },
  { id: "transactions", label: "Transaksi" },
];

const voucherStatuses = [
  { value: "", label: "Semua status" },
  { value: "terjual", label: "Terjual" },
  { value: "aktif", label: "Aktif" },
  { value: "selesai", label: "Selesai" },
  { value: "expired", label: "Expired" },
];

const transactionTypes = [
  { value: "", label: "Semua tipe" },
  { value: "deposit", label: "Deposit" },
  { value: "purchase", label: "Pembelian" },
  { value: "refund", label: "Refund" },
  { value: "withdraw", label: "Withdraw" },
];

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

function normalizeIndonesianPhone(value: string) {
  const raw = String(value || "").trim().replace(/[\s().-]/g, "");
  if (!raw) return "";
  if (raw.startsWith("+62")) return raw;
  if (raw.startsWith("62")) return `+${raw}`;
  if (raw.startsWith("0")) return `+62${raw.slice(1)}`;
  return raw;
}

function apiMessage(error: unknown) {
  return error instanceof Error ? error.message : "Terjadi kesalahan tidak dikenal";
}

function readStoredProfile() {
  try {
    const raw = localStorage.getItem(RESELLER_PROFILE_KEY);
    return raw ? (JSON.parse(raw) as Reseller) : null;
  } catch {
    return null;
  }
}

function saveStoredProfile(reseller: Reseller) {
  localStorage.setItem(RESELLER_PROFILE_KEY, JSON.stringify(reseller));
}

function paginationMeta<T>(page: Pagination<T>) {
  return page.pagination || { page: 1, page_size: page.data.length || 10, total: page.data.length, total_pages: 1 };
}

async function parseApiError(response: Response) {
  const text = await response.text();
  try {
    const envelope = JSON.parse(text) as ApiEnvelope<unknown>;
    return envelope.error?.message || `Request gagal (${response.status})`;
  } catch {
    return text || `Request gagal (${response.status})`;
  }
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

  const text = await response.text();
  let envelope: ApiEnvelope<T>;
  try {
    envelope = JSON.parse(text) as ApiEnvelope<T>;
  } catch {
    const message = text || `Response reseller tidak valid (${response.status})`;
    const error = new Error(message);
    error.name = response.status === 401 ? "Unauthorized" : "InvalidResellerApiResponse";
    throw error;
  }

  if (!response.ok || !envelope.success) {
    const message = envelope.error?.message || `Request reseller gagal (${response.status})`;
    const error = new Error(message);
    error.name = response.status === 401 ? "Unauthorized" : envelope.error?.code || "ResellerApiError";
    throw error;
  }

  return envelope.data as T;
}

function QuerySelect({
  value,
  onChange,
  children,
}: {
  value: string;
  onChange: (value: string) => void;
  children: React.ReactNode;
}) {
  return (
    <select
      value={value}
      onChange={(event) => onChange(event.target.value)}
      className="h-10 min-w-0 rounded-md border border-slate-300 bg-white px-3 text-sm text-slate-700 outline-none transition focus:border-blue-500 focus:ring-2 focus:ring-blue-100"
    >
      {children}
    </select>
  );
}

function Pager({
  page,
  totalPages,
  total,
  onPageChange,
}: {
  page: number;
  totalPages: number;
  total: number;
  onPageChange: (page: number) => void;
}) {
  const safeTotalPages = Math.max(totalPages || 1, 1);
  return (
    <div className="flex flex-col gap-3 border-t border-slate-100 px-1 pt-4 text-sm text-slate-500 sm:flex-row sm:items-center sm:justify-between">
      <span>{total} data</span>
      <div className="inline-flex items-center gap-2">
        <button
          type="button"
          onClick={() => onPageChange(Math.max(page - 1, 1))}
          disabled={page <= 1}
          className="inline-flex h-9 w-9 items-center justify-center rounded-md border border-slate-300 bg-white text-slate-700 transition hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-50"
          aria-label="Halaman sebelumnya"
        >
          <CaretLeft size={16} weight="bold" />
        </button>
        <span className="min-w-20 text-center font-medium text-slate-700">
          {page} / {safeTotalPages}
        </span>
        <button
          type="button"
          onClick={() => onPageChange(Math.min(page + 1, safeTotalPages))}
          disabled={page >= safeTotalPages}
          className="inline-flex h-9 w-9 items-center justify-center rounded-md border border-slate-300 bg-white text-slate-700 transition hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-50"
          aria-label="Halaman berikutnya"
        >
          <CaretRight size={16} weight="bold" />
        </button>
      </div>
    </div>
  );
}

export function ResellerPortalClient() {
  const [accessToken, setAccessToken] = useState("");
  const [refreshToken, setRefreshToken] = useState("");
  const [reseller, setReseller] = useState<Reseller | null>(null);
  const [data, setData] = useState<PortalData | null>(null);
  const [phone, setPhone] = useState("");
  const [password, setPassword] = useState("");
  const [activeTab, setActiveTab] = useState<TabKey>("summary");
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [buying, setBuying] = useState(false);
  const [printing, setPrinting] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [selectedPackageId, setSelectedPackageId] = useState("");
  const [quantity, setQuantity] = useState(1);
  const [voucherStatus, setVoucherStatus] = useState("");
  const [voucherPackageId, setVoucherPackageId] = useState("");
  const [voucherSearch, setVoucherSearch] = useState("");
  const [voucherPage, setVoucherPage] = useState(1);
  const [transactionType, setTransactionType] = useState("");
  const [transactionSearch, setTransactionSearch] = useState("");
  const [transactionPage, setTransactionPage] = useState(1);
  const [selectedVoucherIds, setSelectedVoucherIds] = useState<string[]>([]);

  const voucherPackages = data?.packages.data || [];
  const selectedPackage = voucherPackages.find((pkg) => pkg.id === selectedPackageId);
  const resellerPrice = selectedPackage?.reseller_price ?? 0;
  const sellPrice = selectedPackage?.sell_price ?? resellerPrice;
  const totalCost = resellerPrice * quantity;
  const profitEstimate = Math.max((sellPrice - resellerPrice) * quantity, 0);
  const dailyLimit = reseller?.daily_purchase_limit ?? 0;
  const soldToday = data?.summary.sold_today ?? 0;
  const remainingLimit = dailyLimit > 0 ? Math.max(dailyLimit - soldToday, 0) : null;
  const balance = data?.summary.balance ?? reseller?.balance ?? 0;

  const stats = useMemo(() => {
    const summary = data?.summary;
    return [
      { label: "Saldo", value: formatRupiah(summary?.balance ?? reseller?.balance ?? 0), tone: "green" as const },
      { label: "Terjual hari ini", value: String(summary?.sold_today ?? 0), tone: "blue" as const },
      { label: "Voucher tersedia", value: String(summary?.available_vouchers ?? 0), tone: "amber" as const },
      { label: "Limit harian", value: dailyLimit > 0 ? String(dailyLimit) : "Unlimited", tone: "slate" as const },
    ];
  }, [dailyLimit, data?.summary, reseller]);

  const filteredVouchers = useMemo(() => {
    const query = voucherSearch.trim().toLowerCase();
    const vouchers = data?.vouchers.data || [];
    if (!query) return vouchers;
    return vouchers.filter((voucher) =>
      [voucher.code, voucher.package_name, voucher.username, voucher.password, voucher.status]
        .filter(Boolean)
        .some((value) => String(value).toLowerCase().includes(query)),
    );
  }, [data?.vouchers.data, voucherSearch]);

  const filteredTransactions = useMemo(() => {
    const query = transactionSearch.trim().toLowerCase();
    const transactions = data?.transactions.data || [];
    if (!query) return transactions;
    return transactions.filter((tx) =>
      [tx.type, tx.notes, String(tx.amount), String(tx.balance_after)]
        .filter(Boolean)
        .some((value) => String(value).toLowerCase().includes(query)),
    );
  }, [data?.transactions.data, transactionSearch]);

  const selectedVisibleVoucherIds = filteredVouchers
    .filter((voucher) => selectedVoucherIds.includes(voucher.id))
    .map((voucher) => voucher.id);

  const clearSession = useCallback(() => {
    localStorage.removeItem(ACCESS_TOKEN_KEY);
    localStorage.removeItem(REFRESH_TOKEN_KEY);
    localStorage.removeItem(RESELLER_PROFILE_KEY);
    setAccessToken("");
    setRefreshToken("");
    setReseller(null);
    setData(null);
    setSelectedVoucherIds([]);
  }, []);

  const updateStoredReseller = useCallback((updater: (current: Reseller | null) => Reseller | null) => {
    setReseller((current) => {
      const next = updater(current);
      if (next) saveStoredProfile(next);
      return next;
    });
  }, []);

  const loadPortalData = useCallback(
    async (token: string) => {
      setError("");

      const voucherParams = new URLSearchParams({
        page: String(voucherPage),
        page_size: "10",
        sort_order: "desc",
      });
      if (voucherStatus) voucherParams.set("status", voucherStatus);
      if (voucherPackageId) voucherParams.set("package_id", voucherPackageId);

      const transactionParams = new URLSearchParams({
        page: String(transactionPage),
        page_size: "10",
        sort_order: "desc",
      });
      if (transactionType) transactionParams.set("type", transactionType);

      const [summary, vouchers, transactions, deposits, packages] = await Promise.all([
        resellerApi<DashboardSummary>("/dashboard", undefined, token),
        resellerApi<Pagination<Voucher>>(`/vouchers?${voucherParams.toString()}`, undefined, token),
        resellerApi<Pagination<Transaction>>(`/history?${transactionParams.toString()}`, undefined, token),
        resellerApi<Pagination<Transaction>>("/deposit?page_size=10&sort_order=desc", undefined, token),
        resellerApi<Pagination<VoucherPackage>>("/packages?page_size=50&sort_by=name&sort_order=asc", undefined, token),
      ]);

      setData({ summary, vouchers, transactions, deposits, packages });
      updateStoredReseller((current) => (current ? { ...current, balance: summary.balance } : current));
      setSelectedVoucherIds((current) => current.filter((id) => vouchers.data.some((voucher) => voucher.id === id)));
    },
    [transactionPage, transactionType, updateStoredReseller, voucherPackageId, voucherPage, voucherStatus],
  );

  useEffect(() => {
    async function restoreSession() {
      const storedAccess = localStorage.getItem(ACCESS_TOKEN_KEY) || "";
      const storedRefresh = localStorage.getItem(REFRESH_TOKEN_KEY) || "";
      const storedProfile = readStoredProfile();

      if (storedProfile) setReseller(storedProfile);

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
            saveStoredProfile(refreshed.reseller);
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
    }

    void restoreSession();
  }, []);

  useEffect(() => {
    if (!accessToken || loading) return;
    void loadPortalData(accessToken).catch((err) => {
      if ((err as Error).name === "Unauthorized") clearSession();
      setError(apiMessage(err));
    });
  }, [accessToken, clearSession, loadPortalData, loading]);

  async function handleLogin(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSubmitting(true);
    setError("");
    setSuccess("");

    try {
      const normalizedPhone = normalizeIndonesianPhone(phone);
      const response = await resellerApi<LoginResponse>("/auth/login", {
        method: "POST",
        body: JSON.stringify({ phone: normalizedPhone, password }),
      });

      localStorage.setItem(ACCESS_TOKEN_KEY, response.access_token);
      localStorage.setItem(REFRESH_TOKEN_KEY, response.refresh_token);
      saveStoredProfile(response.reseller);
      setAccessToken(response.access_token);
      setRefreshToken(response.refresh_token);
      setReseller(response.reseller);
      setPhone(normalizedPhone);
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
        await resellerApi(
          "/auth/logout",
          {
            method: "POST",
            body: JSON.stringify({ refresh_token: refresh }),
          },
          token,
        );
      } catch {
        // Session already cleared locally; backend cleanup can fail if token expired.
      }
    }
  }

  async function handleRefresh() {
    if (!accessToken) return;
    setLoading(true);
    setSuccess("");
    try {
      await loadPortalData(accessToken);
    } catch (err) {
      if ((err as Error).name === "Unauthorized") clearSession();
      setError(apiMessage(err));
    } finally {
      setLoading(false);
    }
  }

  async function handleBuyVoucher(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!accessToken || !selectedPackage) return;

    setBuying(true);
    setError("");
    setSuccess("");

    try {
      const result = await resellerApi<BuyVoucherResult>(
        "/vouchers/buy",
        {
          method: "POST",
          body: JSON.stringify({ package_id: selectedPackage.id, quantity }),
        },
        accessToken,
      );

      updateStoredReseller((current) => (current ? { ...current, balance: result.balance_after } : current));
      setSuccess(`${result.vouchers.length} voucher berhasil dibeli. Saldo tersisa ${formatRupiah(result.balance_after)}.`);
      setActiveTab("vouchers");
      setVoucherPage(1);
      setSelectedVoucherIds(result.vouchers.map((voucher) => voucher.id));
      await loadPortalData(accessToken);
    } catch (err) {
      setError(apiMessage(err));
    } finally {
      setBuying(false);
    }
  }

  async function handlePrintSelected() {
    if (!accessToken || selectedVoucherIds.length === 0) return;

    setPrinting(true);
    setError("");
    setSuccess("");

    try {
      const response = await fetch("/api/reseller/vouchers/print", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${accessToken}`,
        },
        body: JSON.stringify({ voucher_ids: selectedVoucherIds }),
      });

      if (!response.ok) throw new Error(await parseApiError(response));

      const blob = await response.blob();
      const url = URL.createObjectURL(blob);
      const link = document.createElement("a");
      link.href = url;
      link.download = "voucher-reseller.pdf";
      document.body.appendChild(link);
      link.click();
      link.remove();
      URL.revokeObjectURL(url);
      setSuccess(`${selectedVoucherIds.length} voucher disiapkan untuk dicetak.`);
    } catch (err) {
      setError(apiMessage(err));
    } finally {
      setPrinting(false);
    }
  }

  async function copyVoucherCode(voucher: Voucher) {
    try {
      await navigator.clipboard.writeText(voucher.code);
      setSuccess(`Kode voucher ${voucher.code} disalin.`);
      setError("");
    } catch {
      setError("Browser tidak mengizinkan salin otomatis. Silakan salin kode manual.");
    }
  }

  function toggleVoucherSelection(voucherId: string) {
    setSelectedVoucherIds((current) =>
      current.includes(voucherId) ? current.filter((id) => id !== voucherId) : [...current, voucherId],
    );
  }

  function toggleVisibleVoucherSelection() {
    const visibleIds = filteredVouchers.map((voucher) => voucher.id);
    const allSelected = visibleIds.length > 0 && visibleIds.every((id) => selectedVoucherIds.includes(id));
    setSelectedVoucherIds((current) =>
      allSelected
        ? current.filter((id) => !visibleIds.includes(id))
        : Array.from(new Set([...current, ...visibleIds])),
    );
  }

  const buyDisabled =
    buying ||
    !selectedPackage ||
    quantity < 1 ||
    quantity > 100 ||
    totalCost <= 0 ||
    totalCost > balance ||
    (remainingLimit !== null && quantity > remainingLimit);

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
                Login memakai akun reseller yang dibuat oleh admin ISP. Nomor 08 akan otomatis diubah ke format +62.
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
                  placeholder="0812xxxxxxxx atau +62812xxxxxxxx"
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

  const voucherMeta = paginationMeta(data?.vouchers || { data: [] });
  const transactionMeta = paginationMeta(data?.transactions || { data: [] });

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

        {error && <div className="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">{error}</div>}
        {success && (
          <div className="rounded-lg border border-emerald-200 bg-emerald-50 px-4 py-3 text-sm text-emerald-700">
            {success}
          </div>
        )}

        <nav className="flex gap-2 overflow-x-auto rounded-xl border border-slate-200 bg-white p-2 shadow-sm" aria-label="Tab reseller">
          {tabs.map((tab) => (
            <button
              key={tab.id}
              type="button"
              onClick={() => setActiveTab(tab.id)}
              className={`h-10 shrink-0 rounded-lg px-4 text-sm font-semibold transition ${
                activeTab === tab.id ? "bg-blue-600 text-white shadow-sm" : "text-slate-600 hover:bg-slate-50 hover:text-slate-950"
              }`}
            >
              {tab.label}
            </button>
          ))}
        </nav>

        {activeTab === "summary" && (
          <div className="space-y-6">
            <StatGrid stats={stats} />
            <div className="grid gap-6 xl:grid-cols-[minmax(0,1.2fr)_minmax(320px,0.8fr)]">
              <Section title="Profil reseller" description="Status akun dan batas pembelian harian.">
                <div className="grid gap-3 sm:grid-cols-2">
                  {[
                    ["Nama", reseller?.name || "-"],
                    ["Nomor HP", reseller?.phone || "-"],
                    ["Email", reseller?.email || "-"],
                    ["Status", <StatusBadge key="status" status={reseller?.status || "-"} />],
                    ["Last login", formatDate(reseller?.last_login)],
                    ["Sisa limit hari ini", remainingLimit === null ? "Unlimited" : String(remainingLimit)],
                  ].map(([label, value]) => (
                    <div key={String(label)} className="min-w-0 rounded-lg border border-slate-200 bg-slate-50 p-4">
                      <p className="text-xs font-semibold uppercase tracking-[0.12em] text-slate-400">{label}</p>
                      <div className="mt-2 min-w-0 text-sm font-semibold text-slate-800 [overflow-wrap:anywhere]">{value}</div>
                    </div>
                  ))}
                </div>
              </Section>

              <Section
                title="Top up saldo"
                description="Deposit saat ini tetap diproses oleh admin tenant."
                action={
                  <span className="inline-flex items-center gap-2 text-sm font-semibold text-slate-500">
                    <Wallet size={18} weight="duotone" />
                    Manual admin
                  </span>
                }
              >
                <div className="rounded-lg border border-slate-200 bg-slate-50 p-4">
                  <p className="text-sm font-semibold text-slate-800">Ajukan deposit ke admin ISP.</p>
                  <p className="mt-2 text-sm leading-6 text-slate-500">
                    Setelah pembayaran diverifikasi, admin menambah saldo dan riwayat deposit akan tampil di portal ini.
                  </p>
                </div>
                <div className="mt-4 space-y-3">
                  {(data?.deposits.data || []).slice(0, 3).map((tx) => (
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
                  {!data?.deposits.data.length && (
                    <EmptyState title="Belum ada deposit" description="Riwayat top up reseller akan muncul ketika admin menambah saldo." />
                  )}
                </div>
              </Section>
            </div>
          </div>
        )}

        {activeTab === "buy" && (
          <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_360px]">
            <Section title="Beli voucher" description="Pilih paket voucher, masukkan jumlah, lalu konfirmasi pembelian dari saldo reseller.">
              <form onSubmit={handleBuyVoucher} className="grid gap-4 lg:grid-cols-4">
                <label className="grid gap-2 lg:col-span-2">
                  <span className="text-sm font-medium text-slate-800">Paket voucher</span>
                  <select
                    value={selectedPackageId}
                    onChange={(event) => setSelectedPackageId(event.target.value)}
                    className="h-10 min-w-0 rounded-md border border-slate-300 bg-white px-3 text-sm text-slate-700 outline-none transition focus:border-blue-500 focus:ring-2 focus:ring-blue-100"
                    required
                  >
                    <option value="">Pilih paket voucher</option>
                    {voucherPackages.map((pkg) => (
                      <option key={pkg.id} value={pkg.id}>
                        {pkg.name} - {formatRupiah(pkg.reseller_price)}
                      </option>
                    ))}
                  </select>
                </label>
                <label className="grid gap-2">
                  <span className="text-sm font-medium text-slate-800">Jumlah</span>
                  <TextInput
                    type="number"
                    min={1}
                    max={100}
                    value={quantity}
                    onChange={(event) => setQuantity(Number(event.target.value || 1))}
                    required
                  />
                </label>
                <div className="grid content-end">
                  <button
                    type="submit"
                    disabled={buyDisabled}
                    className="inline-flex h-10 items-center justify-center rounded-md bg-blue-600 px-4 text-sm font-semibold text-white transition hover:bg-blue-700 disabled:cursor-not-allowed disabled:opacity-60"
                  >
                    {buying ? "Membeli..." : "Beli voucher"}
                  </button>
                </div>
              </form>

              {selectedPackage ? (
                <div className="mt-5 grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
                  <div className="rounded-lg border border-slate-200 bg-slate-50 p-4">
                    <p className="text-xs font-semibold uppercase tracking-[0.12em] text-slate-400">Harga reseller</p>
                    <p className="mt-2 font-semibold text-slate-950">{formatRupiah(resellerPrice)}</p>
                  </div>
                  <div className="rounded-lg border border-slate-200 bg-slate-50 p-4">
                    <p className="text-xs font-semibold uppercase tracking-[0.12em] text-slate-400">Harga jual</p>
                    <p className="mt-2 font-semibold text-slate-950">{formatRupiah(sellPrice)}</p>
                  </div>
                  <div className="rounded-lg border border-slate-200 bg-slate-50 p-4">
                    <p className="text-xs font-semibold uppercase tracking-[0.12em] text-slate-400">Total biaya</p>
                    <p className="mt-2 font-semibold text-slate-950">{formatRupiah(totalCost)}</p>
                  </div>
                  <div className="rounded-lg border border-slate-200 bg-slate-50 p-4">
                    <p className="text-xs font-semibold uppercase tracking-[0.12em] text-slate-400">Estimasi margin</p>
                    <p className="mt-2 font-semibold text-slate-950">{formatRupiah(profitEstimate)}</p>
                  </div>
                </div>
              ) : (
                <EmptyState title="Pilih paket voucher" description="Paket voucher aktif dari menu Paket Internet akan tersedia di sini." />
              )}

              {totalCost > balance && (
                <p className="mt-4 rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-700">
                  Saldo tidak cukup. Saldo saat ini {formatRupiah(balance)}.
                </p>
              )}
              {remainingLimit !== null && quantity > remainingLimit && (
                <p className="mt-4 rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-700">
                  Jumlah melebihi sisa limit harian reseller.
                </p>
              )}
            </Section>

            <Section title="Batas pembelian" description="Ringkasan saldo dan kuota harian sebelum transaksi.">
              <div className="space-y-3">
                <div className="rounded-lg border border-slate-200 bg-slate-50 p-4">
                  <p className="text-xs font-semibold uppercase tracking-[0.12em] text-slate-400">Saldo tersedia</p>
                  <p className="mt-2 text-xl font-semibold text-slate-950">{formatRupiah(balance)}</p>
                </div>
                <div className="rounded-lg border border-slate-200 bg-slate-50 p-4">
                  <p className="text-xs font-semibold uppercase tracking-[0.12em] text-slate-400">Pembelian hari ini</p>
                  <p className="mt-2 text-xl font-semibold text-slate-950">{soldToday}</p>
                </div>
                <div className="rounded-lg border border-slate-200 bg-slate-50 p-4">
                  <p className="text-xs font-semibold uppercase tracking-[0.12em] text-slate-400">Sisa limit</p>
                  <p className="mt-2 text-xl font-semibold text-slate-950">
                    {remainingLimit === null ? "Unlimited" : remainingLimit}
                  </p>
                </div>
              </div>
            </Section>
          </div>
        )}

        {activeTab === "vouchers" && (
          <Section
            title="Voucher saya"
            description="Daftar voucher yang sudah dibeli atau dialokasikan untuk reseller ini."
            action={
              <button
                type="button"
                onClick={handlePrintSelected}
                disabled={printing || selectedVoucherIds.length === 0}
                className="inline-flex h-10 items-center gap-2 rounded-md bg-slate-900 px-4 text-sm font-semibold text-white transition hover:bg-slate-800 disabled:cursor-not-allowed disabled:opacity-50"
              >
                <Printer size={18} weight="bold" />
                {printing ? "Menyiapkan..." : `Cetak (${selectedVoucherIds.length})`}
              </button>
            }
          >
            <div className="mb-4 grid gap-3 lg:grid-cols-[minmax(0,1fr)_180px_220px]">
              <label className="relative">
                <MagnifyingGlass className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" size={18} />
                <input
                  value={voucherSearch}
                  onChange={(event) => setVoucherSearch(event.target.value)}
                  placeholder="Cari kode, paket, username"
                  className="h-10 w-full min-w-0 rounded-md border border-slate-300 bg-white pl-10 pr-3 text-sm outline-none transition focus:border-blue-500 focus:ring-2 focus:ring-blue-100"
                />
              </label>
              <QuerySelect
                value={voucherStatus}
                onChange={(value) => {
                  setVoucherStatus(value);
                  setVoucherPage(1);
                }}
              >
                {voucherStatuses.map((status) => (
                  <option key={status.value} value={status.value}>
                    {status.label}
                  </option>
                ))}
              </QuerySelect>
              <QuerySelect
                value={voucherPackageId}
                onChange={(value) => {
                  setVoucherPackageId(value);
                  setVoucherPage(1);
                }}
              >
                <option value="">Semua paket</option>
                {voucherPackages.map((pkg) => (
                  <option key={pkg.id} value={pkg.id}>
                    {pkg.name}
                  </option>
                ))}
              </QuerySelect>
            </div>

            {filteredVouchers.length ? (
              <div className="space-y-4">
                <DataTable
                  columns={["Pilih", "Kode", "Paket", "Harga", "Status", "Dibeli", "Aksi"]}
                  rows={filteredVouchers.map((voucher) => [
                    <input
                      key={`${voucher.id}-select`}
                      type="checkbox"
                      checked={selectedVoucherIds.includes(voucher.id)}
                      onChange={() => toggleVoucherSelection(voucher.id)}
                      className="h-4 w-4 rounded border-slate-300 text-blue-600"
                      aria-label={`Pilih voucher ${voucher.code}`}
                    />,
                    <span key={voucher.id} className="font-mono font-semibold text-slate-950">
                      {voucher.code}
                    </span>,
                    voucher.package_name || "-",
                    formatRupiah(voucher.sell_price_snapshot ?? voucher.reseller_price_snapshot),
                    <StatusBadge key={`${voucher.id}-status`} status={voucher.status} />,
                    formatDate(voucher.purchased_at),
                    <button
                      key={`${voucher.id}-copy`}
                      type="button"
                      onClick={() => copyVoucherCode(voucher)}
                      className="inline-flex h-8 items-center gap-2 rounded-md border border-slate-300 bg-white px-3 text-xs font-semibold text-slate-700 transition hover:bg-slate-50"
                    >
                      <Copy size={14} weight="bold" />
                      Salin
                    </button>,
                  ])}
                />
                <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                  <button
                    type="button"
                    onClick={toggleVisibleVoucherSelection}
                    className="inline-flex h-9 items-center justify-center rounded-md border border-slate-300 bg-white px-3 text-sm font-semibold text-slate-700 transition hover:bg-slate-50"
                  >
                    {selectedVisibleVoucherIds.length === filteredVouchers.length ? "Batalkan pilih halaman" : "Pilih semua di halaman"}
                  </button>
                  <Pager page={voucherMeta.page} totalPages={voucherMeta.total_pages} total={voucherMeta.total} onPageChange={setVoucherPage} />
                </div>
              </div>
            ) : (
              <EmptyState
                title="Voucher tidak ditemukan"
                description="Ubah filter atau beli voucher baru bila stok tersedia untuk reseller."
              />
            )}
          </Section>
        )}

        {activeTab === "transactions" && (
          <Section
            title="Riwayat transaksi"
            description="Pembelian, refund, deposit, atau penarikan saldo reseller."
            action={
              <span className="inline-flex items-center gap-2 text-sm font-semibold text-slate-500">
                <DownloadSimple size={18} weight="duotone" />
                Live API
              </span>
            }
          >
            <div className="mb-4 grid gap-3 lg:grid-cols-[minmax(0,1fr)_220px]">
              <label className="relative">
                <MagnifyingGlass className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" size={18} />
                <input
                  value={transactionSearch}
                  onChange={(event) => setTransactionSearch(event.target.value)}
                  placeholder="Cari catatan, tipe, nominal"
                  className="h-10 w-full min-w-0 rounded-md border border-slate-300 bg-white pl-10 pr-3 text-sm outline-none transition focus:border-blue-500 focus:ring-2 focus:ring-blue-100"
                />
              </label>
              <QuerySelect
                value={transactionType}
                onChange={(value) => {
                  setTransactionType(value);
                  setTransactionPage(1);
                }}
              >
                {transactionTypes.map((type) => (
                  <option key={type.value} value={type.value}>
                    {type.label}
                  </option>
                ))}
              </QuerySelect>
            </div>

            {filteredTransactions.length ? (
              <div className="space-y-4">
                <DataTable
                  columns={["Waktu", "Tipe", "Nominal", "Saldo Awal", "Saldo Akhir", "Catatan"]}
                  rows={filteredTransactions.map((tx) => [
                    formatDate(tx.created_at),
                    <StatusBadge key={`${tx.id}-type`} status={tx.type} />,
                    formatRupiah(tx.amount),
                    formatRupiah(tx.balance_before),
                    formatRupiah(tx.balance_after),
                    tx.notes || "-",
                  ])}
                />
                <Pager
                  page={transactionMeta.page}
                  totalPages={transactionMeta.total_pages}
                  total={transactionMeta.total}
                  onPageChange={setTransactionPage}
                />
              </div>
            ) : (
              <EmptyState
                title="Belum ada transaksi"
                description="Aktivitas reseller akan muncul otomatis setelah ada deposit atau pembelian voucher."
              />
            )}
          </Section>
        )}

        <div className="rounded-xl border border-slate-200 bg-white p-4">
          <div className="flex flex-col gap-3 text-sm text-slate-500 sm:flex-row sm:items-center sm:justify-between">
            <span className="inline-flex items-center gap-2">
              <UserCircle size={18} weight="duotone" />
              Bantuan akun, deposit, dan akun suspended ditangani admin ISP.
            </span>
            <StatusBadge status={reseller?.status || "unknown"} />
          </div>
        </div>
      </div>
    </main>
  );
}
