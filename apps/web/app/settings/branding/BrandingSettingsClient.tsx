"use client";

import { useMemo, useState } from "react";
import {
  ArrowClockwise,
  Bank,
  Bell,
  Check,
  CloudArrowUp,
  Copy,
  CreditCard,
  Desktop,
  FileText,
  Globe,
  HouseLine,
  Image,
  Invoice,
  List,
  LockKey,
  MapTrifold,
  Palette,
  Receipt,
  ShieldCheck,
  Storefront,
  Trash,
  UploadSimple,
  Users,
  WifiHigh,
} from "@phosphor-icons/react";

const navItems = [
  { label: "Profil ISP", icon: HouseLine, href: "/settings/profile" },
  { label: "White Label", icon: Palette, href: "/settings/branding", active: true },
  { label: "User & Role", icon: Users, href: "/settings/users" },
  { label: "Billing", icon: Receipt, href: "/settings/billing" },
  { label: "Payment Gateway", icon: CreditCard, href: "/settings/payment" },
  { label: "Notifikasi", icon: Bell, href: "/settings/notifications" },
  { label: "MikroTik", icon: WifiHigh, href: "/settings/mikrotik" },
  { label: "Peta", icon: MapTrifold, href: "/settings/map" },
  { label: "Keamanan", icon: LockKey, href: "/settings/security" },
  { label: "Subscription", icon: Storefront, href: "/settings/subscription" },
];

const brandUses = [
  "Sidebar dashboard",
  "Invoice PDF",
  "Walled garden",
  "Notifikasi WA dan email",
  "Voucher print",
  "Hotspot login page",
];

export default function BrandingSettingsClient() {
  const [primaryColor, setPrimaryColor] = useState("#2563EB");
  const [customColor, setCustomColor] = useState(true);
  const [brandName, setBrandName] = useState("NusaFiber Depok");
  const [domain, setDomain] = useState("billing.nusafiber.id");
  const [domainStatus, setDomainStatus] = useState<"idle" | "checking" | "verified">("idle");
  const [footerText, setFooterText] = useState(
    "Terima kasih atas pembayaran Anda. Simpan invoice ini sebagai bukti transaksi resmi.",
  );
  const [walledMessage, setWalledMessage] = useState(
    "Tagihan internet Anda belum dibayar. Selesaikan pembayaran untuk mengaktifkan kembali layanan.",
  );
  const [saveState, setSaveState] = useState<"idle" | "saving" | "saved">("idle");

  const domainBadge = useMemo(() => {
    if (domainStatus === "verified") return "Terverifikasi";
    if (domainStatus === "checking") return "Memeriksa DNS";
    return "Belum dikonfigurasi";
  }, [domainStatus]);

  function handleVerifyDomain() {
    setDomainStatus("checking");
    window.setTimeout(() => setDomainStatus("verified"), 900);
  }

  function handleSave() {
    setSaveState("saving");
    window.setTimeout(() => setSaveState("saved"), 800);
    window.setTimeout(() => setSaveState("idle"), 2500);
  }

  return (
    <main className="min-h-[100dvh] bg-slate-50 text-slate-950">
      <div className="flex min-h-[100dvh]">
        <aside className="hidden w-72 shrink-0 border-r border-slate-200 bg-white xl:block">
          <div className="flex h-16 items-center gap-3 border-b border-slate-200 px-5">
            <span className="grid h-9 w-9 place-items-center rounded-lg bg-slate-950 text-xs font-black text-white">
              IB
            </span>
            <div>
              <div className="font-semibold tracking-tight">ISPBoss</div>
              <div className="text-xs text-slate-500">Tenant Admin</div>
            </div>
          </div>
          <nav className="px-3 py-4">
            <p className="px-3 pb-2 text-xs font-semibold uppercase tracking-[0.18em] text-slate-400">
              Pengaturan
            </p>
            <div className="grid gap-1">
              {navItems.map((item) => {
                const Icon = item.icon;
                return (
                  <a
                    key={item.label}
                    href={item.href}
                    className={`flex items-center gap-3 rounded-md px-3 py-2.5 text-sm font-medium transition ${
                      item.active
                        ? "bg-blue-50 text-blue-700"
                        : "text-slate-600 hover:bg-slate-50 hover:text-slate-950"
                    }`}
                  >
                    <Icon size={18} />
                    {item.label}
                  </a>
                );
              })}
            </div>
          </nav>
        </aside>

        <section className="flex min-w-0 flex-1 flex-col">
          <header className="sticky top-0 z-20 border-b border-slate-200 bg-white/85 backdrop-blur-xl">
            <div className="flex h-16 items-center justify-between gap-4 px-4 sm:px-6 lg:px-8">
              <div className="flex items-center gap-3">
                <button
                  type="button"
                  className="grid h-10 w-10 place-items-center rounded-md border border-slate-200 xl:hidden"
                  aria-label="Buka menu settings"
                >
                  <List size={20} />
                </button>
                <div>
                  <p className="text-xs text-slate-500">Dashboard / Pengaturan</p>
                  <h1 className="text-lg font-semibold tracking-tight">White Label / Branding</h1>
                </div>
              </div>
              <div className="flex items-center gap-2">
                {saveState === "saved" && (
                  <span className="hidden items-center gap-2 rounded-md bg-emerald-50 px-3 py-2 text-sm font-medium text-emerald-700 sm:inline-flex">
                    <Check size={16} weight="bold" />
                    Tersimpan
                  </span>
                )}
                <button
                  type="button"
                  onClick={handleSave}
                  disabled={saveState === "saving"}
                  className="inline-flex items-center justify-center gap-2 rounded-md bg-blue-600 px-4 py-2 text-sm font-semibold text-white transition hover:bg-blue-700 active:scale-[0.98] disabled:cursor-wait disabled:bg-blue-400"
                >
                  {saveState === "saving" ? <ArrowClockwise className="animate-spin" size={16} /> : <CloudArrowUp size={16} />}
                  Simpan
                </button>
              </div>
            </div>
          </header>

          <div className="grid flex-1 gap-6 p-4 sm:p-6 lg:grid-cols-[minmax(0,1fr)_380px] lg:p-8">
            <div className="space-y-6">
              <section className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
                <div className="flex items-start justify-between gap-4">
                  <div>
                    <h2 className="text-base font-semibold">Logo dan aset brand</h2>
                    <p className="mt-1 text-sm text-slate-500">
                      Digunakan di sidebar, invoice, notifikasi, voucher, dan halaman pelanggan isolir.
                    </p>
                  </div>
                  <Image className="text-blue-600" size={24} />
                </div>
                <div className="mt-6 grid gap-5 md:grid-cols-[220px_1fr]">
                  <div className="rounded-lg border border-dashed border-slate-300 bg-slate-50 p-4">
                    <div className="grid h-28 place-items-center rounded-md border border-slate-200 bg-white">
                      <div className="text-center">
                        <div className="mx-auto grid h-12 w-12 place-items-center rounded-lg" style={{ backgroundColor: primaryColor }}>
                          <span className="text-xs font-black text-white">NF</span>
                        </div>
                        <p className="mt-2 text-sm font-semibold">{brandName}</p>
                      </div>
                    </div>
                    <p className="mt-3 text-xs leading-5 text-slate-500">
                      Rekomendasi 200x60 px, PNG atau SVG, maksimal 500 KB.
                    </p>
                  </div>
                  <div className="grid gap-4">
                    <Field label="Nama brand / ISP" helper="Nama ini muncul di invoice, notifikasi, dan walled garden.">
                      <input
                        value={brandName}
                        onChange={(event) => setBrandName(event.target.value)}
                        className="h-10 rounded-md border border-slate-300 px-3 text-sm outline-none transition focus:border-blue-500 focus:ring-2 focus:ring-blue-100"
                      />
                    </Field>
                    <div className="grid gap-3 sm:grid-cols-2">
                      <button
                        type="button"
                        className="inline-flex items-center justify-center gap-2 rounded-md border border-slate-300 px-4 py-2 text-sm font-semibold text-slate-700 transition hover:bg-slate-50 active:scale-[0.98]"
                      >
                        <UploadSimple size={16} />
                        Upload Logo
                      </button>
                      <button
                        type="button"
                        className="inline-flex items-center justify-center gap-2 rounded-md border border-slate-300 px-4 py-2 text-sm font-semibold text-slate-700 transition hover:bg-slate-50 active:scale-[0.98]"
                      >
                        <Trash size={16} />
                        Hapus Logo
                      </button>
                    </div>
                    <button
                      type="button"
                      className="inline-flex items-center justify-center gap-2 rounded-md border border-slate-300 px-4 py-2 text-sm font-semibold text-slate-700 transition hover:bg-slate-50 active:scale-[0.98] sm:w-fit"
                    >
                      <UploadSimple size={16} />
                      Upload Favicon
                    </button>
                  </div>
                </div>
              </section>

              <section className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
                <div className="flex items-start justify-between gap-4">
                  <div>
                    <h2 className="text-base font-semibold">Warna utama</h2>
                    <p className="mt-1 text-sm text-slate-500">
                      Preview langsung untuk tombol, link, sidebar aktif, dan elemen pelanggan.
                    </p>
                  </div>
                  <Palette className="text-blue-600" size={24} />
                </div>
                <div className="mt-6 grid gap-5 lg:grid-cols-[260px_1fr]">
                  <Field label="Warna primer" helper="Default ISPBoss adalah #2563EB.">
                    <div className="flex items-center gap-3">
                      <input
                        type="color"
                        value={primaryColor}
                        onChange={(event) => setPrimaryColor(event.target.value)}
                        className="h-10 w-14 rounded-md border border-slate-300 bg-white p-1"
                      />
                      <input
                        value={primaryColor}
                        onChange={(event) => setPrimaryColor(event.target.value)}
                        className="h-10 flex-1 rounded-md border border-slate-300 px-3 font-mono text-sm outline-none transition focus:border-blue-500 focus:ring-2 focus:ring-blue-100"
                      />
                    </div>
                  </Field>
                  <div className="rounded-lg border border-slate-200 p-4">
                    <div className="flex flex-wrap items-center gap-3">
                      <button
                        type="button"
                        style={{ backgroundColor: primaryColor }}
                        className="rounded-md px-4 py-2 text-sm font-semibold text-white"
                      >
                        Tombol Primer
                      </button>
                      <a href="#preview" style={{ color: primaryColor }} className="text-sm font-semibold">
                        Link tindakan
                      </a>
                      <span
                        className="rounded-md px-3 py-2 text-sm font-semibold"
                        style={{ backgroundColor: `${primaryColor}14`, color: primaryColor }}
                      >
                        Sidebar Active
                      </span>
                    </div>
                  </div>
                </div>
                <label className="mt-5 flex cursor-pointer items-start gap-3 text-sm">
                  <input
                    type="checkbox"
                    checked={customColor}
                    onChange={(event) => setCustomColor(event.target.checked)}
                    className="mt-1 h-4 w-4 rounded border-slate-300 text-blue-600"
                  />
                  <span>
                    <span className="block font-medium">Gunakan warna custom</span>
                    <span className="block text-slate-500">Jika nonaktif, tenant memakai tema default ISPBoss.</span>
                  </span>
                </label>
              </section>

              <section className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
                <div className="flex items-start justify-between gap-4">
                  <div>
                    <h2 className="text-base font-semibold">Custom domain</h2>
                    <p className="mt-1 text-sm text-slate-500">
                      Hubungkan domain tenant untuk portal admin dan halaman pembayaran pelanggan.
                    </p>
                  </div>
                  <Globe className="text-blue-600" size={24} />
                </div>
                <div className="mt-6 grid gap-5 lg:grid-cols-[1fr_260px]">
                  <Field label="Domain custom" helper="Tambahkan CNAME domain ini ke app.ispboss.id.">
                    <input
                      value={domain}
                      onChange={(event) => {
                        setDomain(event.target.value);
                        setDomainStatus("idle");
                      }}
                      className="h-10 rounded-md border border-slate-300 px-3 text-sm outline-none transition focus:border-blue-500 focus:ring-2 focus:ring-blue-100"
                    />
                  </Field>
                  <div className="rounded-lg border border-slate-200 p-4">
                    <p className="text-xs font-semibold uppercase tracking-[0.16em] text-slate-400">Status</p>
                    <p
                      className={`mt-2 text-sm font-semibold ${
                        domainStatus === "verified"
                          ? "text-emerald-700"
                          : domainStatus === "checking"
                            ? "text-amber-700"
                            : "text-slate-600"
                      }`}
                    >
                      {domainBadge}
                    </p>
                  </div>
                </div>
                <div className="mt-5 rounded-lg bg-slate-50 p-4">
                  <div className="grid gap-3 text-sm text-slate-600 md:grid-cols-3">
                    <Step number="1" title="CNAME" body={`${domain} ke app.ispboss.id`} />
                    <Step number="2" title="Verifikasi DNS" body="Sistem membaca propagasi domain." />
                    <Step number="3" title="SSL otomatis" body="Sertifikat diterbitkan setelah valid." />
                  </div>
                </div>
                <button
                  type="button"
                  onClick={handleVerifyDomain}
                  className="mt-5 inline-flex items-center gap-2 rounded-md border border-slate-300 px-4 py-2 text-sm font-semibold text-slate-700 transition hover:bg-slate-50 active:scale-[0.98]"
                >
                  {domainStatus === "checking" ? <ArrowClockwise className="animate-spin" size={16} /> : <ShieldCheck size={16} />}
                  Verifikasi Domain
                </button>
              </section>

              <section className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
                <div className="flex items-start justify-between gap-4">
                  <div>
                    <h2 className="text-base font-semibold">Invoice dan walled garden</h2>
                    <p className="mt-1 text-sm text-slate-500">
                      Teks ini dipakai di invoice PDF, email invoice, dan halaman pelanggan yang terisolir.
                    </p>
                  </div>
                  <Invoice className="text-blue-600" size={24} />
                </div>
                <div className="mt-6 grid gap-5 md:grid-cols-2">
                  <Field label="Footer invoice" helper="Muncul di bagian bawah PDF invoice.">
                    <textarea
                      value={footerText}
                      onChange={(event) => setFooterText(event.target.value)}
                      rows={4}
                      className="rounded-md border border-slate-300 px-3 py-2 text-sm outline-none transition focus:border-blue-500 focus:ring-2 focus:ring-blue-100"
                    />
                  </Field>
                  <Field label="Pesan walled garden" helper="Tampil saat pelanggan masuk halaman isolir.">
                    <textarea
                      value={walledMessage}
                      onChange={(event) => setWalledMessage(event.target.value)}
                      rows={4}
                      className="rounded-md border border-slate-300 px-3 py-2 text-sm outline-none transition focus:border-blue-500 focus:ring-2 focus:ring-blue-100"
                    />
                  </Field>
                </div>
                <div className="mt-5 grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
                  {["Tombol bayar", "Kontak admin", "Detail tagihan", "Nomor rekening"].map((item, index) => (
                    <label key={item} className="flex items-center gap-2 rounded-md border border-slate-200 px-3 py-2 text-sm">
                      <input defaultChecked={index < 3} type="checkbox" className="h-4 w-4 rounded border-slate-300 text-blue-600" />
                      {item}
                    </label>
                  ))}
                </div>
              </section>
            </div>

            <aside className="space-y-6 lg:sticky lg:top-24 lg:self-start">
              <section id="preview" className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
                <div className="flex items-center justify-between">
                  <h2 className="text-base font-semibold">Live preview</h2>
                  <Desktop className="text-slate-400" size={20} />
                </div>
                <div className="mt-5 overflow-hidden rounded-lg border border-slate-200">
                  <div className="flex h-11 items-center gap-2 border-b border-slate-200 bg-slate-50 px-3">
                    <span className="grid h-7 w-7 place-items-center rounded-md text-xs font-black text-white" style={{ backgroundColor: primaryColor }}>
                      NF
                    </span>
                    <span className="text-sm font-semibold">{brandName}</span>
                  </div>
                  <div className="grid grid-cols-[70px_1fr]">
                    <div className="space-y-2 border-r border-slate-200 bg-white p-2">
                      {[Users, Receipt, MapTrifold, Palette].map((Icon, index) => (
                        <div
                          key={index}
                          className="grid h-9 place-items-center rounded-md"
                          style={index === 3 ? { backgroundColor: `${primaryColor}14`, color: primaryColor } : {}}
                        >
                          <Icon size={17} />
                        </div>
                      ))}
                    </div>
                    <div className="p-4">
                      <div className="h-3 w-28 rounded bg-slate-200" />
                      <div className="mt-3 h-7 w-40 rounded bg-slate-900" />
                      <div className="mt-5 rounded-md border border-slate-200 p-3">
                        <div className="flex items-center justify-between">
                          <span className="text-xs text-slate-500">Invoice</span>
                          <span className="font-mono text-xs">INV-2605-118</span>
                        </div>
                        <p className="mt-3 text-xs leading-5 text-slate-500">{footerText}</p>
                        <button
                          type="button"
                          className="mt-4 w-full rounded-md py-2 text-xs font-semibold text-white"
                          style={{ backgroundColor: primaryColor }}
                        >
                          Bayar Sekarang
                        </button>
                      </div>
                    </div>
                  </div>
                </div>
              </section>

              <section className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
                <h2 className="text-base font-semibold">Walled garden</h2>
                <div className="mt-4 rounded-lg border border-slate-200 bg-slate-50 p-4">
                  <div className="flex items-center gap-3">
                    <span className="grid h-10 w-10 place-items-center rounded-lg text-xs font-black text-white" style={{ backgroundColor: primaryColor }}>
                      NF
                    </span>
                    <div>
                      <p className="font-semibold">{brandName}</p>
                      <p className="text-xs text-slate-500">{domain}</p>
                    </div>
                  </div>
                  <p className="mt-4 text-sm leading-6 text-slate-600">{walledMessage}</p>
                  <button
                    type="button"
                    className="mt-4 inline-flex w-full items-center justify-center gap-2 rounded-md py-2 text-sm font-semibold text-white"
                    style={{ backgroundColor: primaryColor }}
                  >
                    <Bank size={16} />
                    Lihat Cara Bayar
                  </button>
                </div>
              </section>

              <section className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
                <h2 className="text-base font-semibold">Dipakai di modul</h2>
                <div className="mt-4 grid gap-2">
                  {brandUses.map((item) => (
                    <div key={item} className="flex items-center justify-between rounded-md border border-slate-200 px-3 py-2 text-sm">
                      <span>{item}</span>
                      <Check className="text-emerald-600" size={16} weight="bold" />
                    </div>
                  ))}
                </div>
              </section>

              <button
                type="button"
                className="inline-flex w-full items-center justify-center gap-2 rounded-md border border-slate-300 bg-white px-4 py-2.5 text-sm font-semibold text-slate-700 transition hover:bg-slate-50 active:scale-[0.98]"
              >
                <Copy size={16} />
                Salin konfigurasi brand
              </button>
            </aside>
          </div>
        </section>
      </div>
    </main>
  );
}

function Field({
  label,
  helper,
  children,
}: {
  label: string;
  helper?: string;
  children: React.ReactNode;
}) {
  return (
    <label className="grid gap-2">
      <span className="text-sm font-medium text-slate-800">{label}</span>
      {children}
      {helper && <span className="text-xs leading-5 text-slate-500">{helper}</span>}
    </label>
  );
}

function Step({
  number,
  title,
  body,
}: {
  number: string;
  title: string;
  body: string;
}) {
  return (
    <div className="flex gap-3">
      <span className="grid h-7 w-7 shrink-0 place-items-center rounded-full bg-slate-950 font-mono text-xs font-semibold text-white">
        {number}
      </span>
      <span>
        <span className="block font-semibold text-slate-900">{title}</span>
        <span className="block text-xs leading-5 text-slate-500">{body}</span>
      </span>
    </div>
  );
}
