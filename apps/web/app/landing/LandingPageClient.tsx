"use client";

import { useState } from "react";
import {
  ArrowDown,
  ArrowRight,
  Broadcast,
  CaretDown,
  ChartLineUp,
  ChatCircleDots,
  Check,
  CreditCard,
  FileText,
  GlobeHemisphereEast,
  List,
  MapTrifold,
  Moon,
  Network,
  ShieldCheck,
  Sun,
  WifiHigh,
  X,
} from "@phosphor-icons/react";
import { landingFaqs, publicPricingPlans } from "./content";

type Theme = "light" | "dark";

const features = [
  {
    title: "Billing otomatis",
    body: "Invoice, denda, reminder, dan pembayaran tersusun tanpa spreadsheet manual.",
    icon: FileText,
    className: "md:col-span-2",
  },
  {
    title: "MikroTik terkendali",
    body: "PPPoE, isolir, buka isolir, dan monitoring router dikerjakan dari satu tempat.",
    icon: WifiHigh,
    className: "md:col-span-2",
  },
  {
    title: "OLT multi-brand",
    body: "Kelola OLT, ODP, ONT, alarm, dan provisioning tanpa pindah alat.",
    icon: Broadcast,
    className: "md:row-span-2",
  },
  {
    title: "Notifikasi lengkap",
    body: "WhatsApp, SMS, dan email memakai template yang bisa disesuaikan per tenant.",
    icon: ChatCircleDots,
    className: "",
  },
  {
    title: "FTTH visual mapping",
    body: "Lihat node, kabel, loss budget, foto lapangan, dan riwayat perubahan.",
    icon: MapTrifold,
    className: "",
  },
  {
    title: "White label",
    body: "Logo, warna, domain, invoice, dan walled garden mengikuti identitas ISP.",
    icon: GlobeHemisphereEast,
    className: "md:col-span-2",
  },
];

export default function LandingPageClient() {
  const [theme, setTheme] = useState<Theme>("light");
  const [mobileOpen, setMobileOpen] = useState(false);
  const [openFaq, setOpenFaq] = useState(0);

  const dark = theme === "dark";

  return (
    <main
      className={`min-h-[100dvh] overflow-hidden transition-colors duration-300 ${
        dark ? "dark bg-slate-950 text-slate-50" : "bg-white text-slate-950"
      }`}
    >
      <header className="fixed inset-x-0 top-0 z-30 border-b border-white/10 bg-white/80 backdrop-blur-xl dark:bg-slate-950/80">
        <div className="mx-auto flex h-16 max-w-7xl items-center justify-between px-4 sm:px-6 lg:px-8">
          <a href="#top" className="flex items-center gap-3">
            <span className="grid h-9 w-9 place-items-center rounded-lg bg-slate-950 text-sm font-black text-white dark:bg-white dark:text-slate-950">
              IB
            </span>
            <span className="text-lg font-semibold tracking-tight">ISPBoss</span>
          </a>

          <nav className="hidden items-center gap-8 text-sm font-medium text-slate-600 dark:text-slate-300 md:flex">
            <a className="hover:text-slate-950 dark:hover:text-white" href="#fitur">
              Fitur
            </a>
            <a className="hover:text-slate-950 dark:hover:text-white" href="#harga">
              Harga
            </a>
            <a className="hover:text-slate-950 dark:hover:text-white" href="#white-label">
              White label
            </a>
            <a className="hover:text-slate-950 dark:hover:text-white" href="#faq">
              FAQ
            </a>
          </nav>

          <div className="hidden items-center gap-2 md:flex">
            <button
              type="button"
              onClick={() => setTheme(dark ? "light" : "dark")}
              className="grid h-10 w-10 place-items-center rounded-md border border-slate-200 text-slate-600 transition hover:border-slate-300 hover:bg-slate-50 active:scale-[0.98] dark:border-slate-800 dark:text-slate-300 dark:hover:bg-slate-900"
              aria-label="Ganti tema"
            >
              {dark ? <Sun size={18} /> : <Moon size={18} />}
            </button>
            <a
              href="/login"
              className="rounded-md px-4 py-2 text-sm font-medium text-slate-700 transition hover:bg-slate-100 active:scale-[0.98] dark:text-slate-200 dark:hover:bg-slate-900"
            >
              Masuk
            </a>
            <a
              href="/register"
              className="rounded-md bg-blue-600 px-4 py-2 text-sm font-semibold text-white transition hover:bg-blue-700 active:scale-[0.98]"
            >
              Coba Gratis
            </a>
          </div>

          <button
            type="button"
            onClick={() => setMobileOpen((value) => !value)}
            className="grid h-10 w-10 place-items-center rounded-md border border-slate-200 md:hidden dark:border-slate-800"
            aria-label="Buka menu"
            aria-expanded={mobileOpen}
          >
            {mobileOpen ? <X size={20} /> : <List size={20} />}
          </button>
        </div>
        {mobileOpen && (
          <div className="border-t border-slate-200 bg-white px-4 py-4 dark:border-slate-800 dark:bg-slate-950 md:hidden">
            <div className="flex flex-col gap-3 text-sm">
              <a href="#fitur" onClick={() => setMobileOpen(false)}>
                Fitur
              </a>
              <a href="#harga" onClick={() => setMobileOpen(false)}>
                Harga
              </a>
              <a href="#white-label" onClick={() => setMobileOpen(false)}>
                White label
              </a>
              <a href="#faq" onClick={() => setMobileOpen(false)}>
                FAQ
              </a>
              <a className="rounded-md bg-blue-600 px-4 py-2 text-center font-semibold text-white" href="/register">
                Coba Gratis
              </a>
            </div>
          </div>
        )}
      </header>

      <section id="top" className="relative min-h-[100dvh] pt-16">
        <div className="absolute inset-0 -z-10 bg-[radial-gradient(circle_at_75%_30%,rgba(37,99,235,0.20),transparent_34%),linear-gradient(120deg,#eff6ff_0%,#ffffff_55%,#f8fafc_100%)] dark:bg-[radial-gradient(circle_at_75%_30%,rgba(37,99,235,0.24),transparent_34%),linear-gradient(120deg,#0f172a_0%,#111827_56%,#020617_100%)]" />
        <div className="mx-auto grid min-h-[calc(100dvh-4rem)] max-w-7xl items-center gap-12 px-4 py-14 sm:px-6 lg:grid-cols-[0.9fr_1.1fr] lg:px-8">
          <div className="max-w-xl animate-[ispboss-rise_700ms_ease-out_both]">
            <p className="mb-5 inline-flex items-center gap-2 rounded-full border border-blue-200 bg-white/70 px-3 py-1 text-xs font-semibold uppercase tracking-[0.18em] text-blue-700 dark:border-blue-500/30 dark:bg-blue-500/10 dark:text-blue-200">
              Platform ISP Indonesia
            </p>
            <h1 className="text-5xl font-semibold leading-none tracking-tight sm:text-6xl lg:text-7xl">
              ISPBoss
              <span className="mt-3 block text-3xl text-slate-600 dark:text-slate-300 sm:text-5xl">
                Kelola ISP dari satu dashboard.
              </span>
            </h1>
            <p className="mt-6 max-w-lg text-base leading-7 text-slate-600 dark:text-slate-300">
              Billing, pelanggan, MikroTik, OLT, notifikasi, dan peta FTTH bekerja dalam satu alur operasional.
            </p>
            <div className="mt-8 flex flex-col gap-3 sm:flex-row">
              <a
                href="/register"
                className="inline-flex items-center justify-center gap-2 rounded-md bg-blue-600 px-5 py-3 text-sm font-semibold text-white transition hover:bg-blue-700 active:scale-[0.98]"
              >
                Coba Gratis 3 Hari <ArrowRight size={17} />
              </a>
              <a
                href="#fitur"
                className="inline-flex items-center justify-center gap-2 rounded-md border border-slate-300 px-5 py-3 text-sm font-semibold text-slate-700 transition hover:bg-white active:scale-[0.98] dark:border-slate-700 dark:text-slate-200 dark:hover:bg-slate-900"
              >
                Lihat Fitur <ArrowDown size={17} />
              </a>
            </div>
            <div className="mt-7 flex flex-wrap gap-4 text-sm text-slate-600 dark:text-slate-300">
              {["Tanpa kartu kredit", "Setup cepat", "Bisa cancel kapan saja"].map((item) => (
                <span key={item} className="inline-flex items-center gap-2">
                  <Check className="text-blue-600" size={16} weight="bold" />
                  {item}
                </span>
              ))}
            </div>
          </div>

          <ProductPreview />
        </div>
      </section>

      <section id="fitur" className="border-y border-slate-200 bg-slate-50 py-24 dark:border-slate-800 dark:bg-slate-900/40">
        <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <div className="max-w-2xl">
            <p className="text-sm font-semibold uppercase tracking-[0.18em] text-blue-700 dark:text-blue-300">
              Modul utama
            </p>
            <h2 className="mt-3 text-4xl font-semibold tracking-tight sm:text-5xl">
              Dibuat untuk ritme kerja operator ISP.
            </h2>
          </div>
          <div className="mt-12 grid grid-cols-1 gap-px overflow-hidden rounded-xl border border-slate-200 bg-slate-200 dark:border-slate-800 dark:bg-slate-800 md:grid-cols-5">
            {features.map((feature) => {
              const Icon = feature.icon;
              return (
                <article
                  key={feature.title}
                  className={`min-h-56 bg-white p-6 transition hover:-translate-y-1 hover:bg-blue-50 dark:bg-slate-950 dark:hover:bg-slate-900 ${feature.className}`}
                >
                  <Icon size={28} className="text-blue-600" />
                  <h3 className="mt-7 text-xl font-semibold">{feature.title}</h3>
                  <p className="mt-3 text-sm leading-6 text-slate-600 dark:text-slate-300">
                    {feature.body}
                  </p>
                </article>
              );
            })}
          </div>
        </div>
      </section>

      <section className="py-24">
        <div className="mx-auto grid max-w-7xl gap-12 px-4 sm:px-6 lg:grid-cols-[0.8fr_1.2fr] lg:px-8">
          <div>
            <p className="text-sm font-semibold uppercase tracking-[0.18em] text-blue-700 dark:text-blue-300">
              Cara kerja
            </p>
            <h2 className="mt-3 text-4xl font-semibold tracking-tight">
              Mulai dari data pelanggan, lalu hubungkan perangkat.
            </h2>
          </div>
          <div className="grid gap-8 md:grid-cols-3">
            {[
              ["01", "Daftar dan setup tenant", "Lengkapi profil ISP, user, role, dan branding dasar."],
              ["02", "Hubungkan jaringan", "Tambahkan router, OLT, ODP, ONT, dan parameter isolir."],
              ["03", "Operasikan harian", "Generate invoice, terima pembayaran, kirim reminder, pantau jaringan."],
            ].map(([step, title, body]) => (
              <div key={step} className="border-t border-slate-200 pt-5 dark:border-slate-800">
                <span className="font-mono text-sm text-blue-600">{step}</span>
                <h3 className="mt-4 text-lg font-semibold">{title}</h3>
                <p className="mt-2 text-sm leading-6 text-slate-600 dark:text-slate-300">{body}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section id="harga" className="bg-slate-950 py-24 text-white">
        <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <div className="flex flex-col justify-between gap-8 md:flex-row md:items-end">
            <div>
              <p className="text-sm font-semibold uppercase tracking-[0.18em] text-blue-300">
                Harga
              </p>
              <h2 className="mt-3 max-w-xl text-4xl font-semibold tracking-tight">
                Paket mengikuti skala pelanggan.
              </h2>
            </div>
            <p className="max-w-md text-sm leading-6 text-slate-300">
              Semua paket membawa modul inti: pelanggan, paket, billing, pembayaran, dan akses support.
            </p>
          </div>
          <div className="mt-12 grid gap-px overflow-hidden rounded-xl border border-white/10 bg-white/10 md:grid-cols-4">
            {publicPricingPlans.map((plan) => (
              <article
                key={plan.name}
                className={`bg-slate-950 p-6 ${plan.featured ? "relative ring-1 ring-inset ring-blue-400" : ""}`}
              >
                {plan.featured && (
                  <span className="mb-5 inline-flex rounded-full bg-blue-500 px-3 py-1 text-xs font-semibold">
                    Paling sering dipilih
                  </span>
                )}
                <h3 className="text-xl font-semibold">{plan.name}</h3>
                <p className="mt-2 text-sm text-slate-400">{plan.range}</p>
                <p className="mt-6 font-mono text-3xl font-semibold">{plan.price}</p>
                <p className="mt-3 min-h-12 text-sm leading-6 text-slate-300">{plan.detail}</p>
                <a
                  href={plan.name === "Enterprise" ? "/contact" : "/register"}
                  className={`mt-8 inline-flex w-full items-center justify-center rounded-md px-4 py-2.5 text-sm font-semibold transition active:scale-[0.98] ${
                    plan.featured
                      ? "bg-blue-500 text-white hover:bg-blue-400"
                      : "border border-white/15 text-white hover:bg-white/10"
                  }`}
                >
                  {plan.name === "Enterprise" ? "Hubungi Tim" : "Coba Paket"}
                </a>
              </article>
            ))}
          </div>
        </div>
      </section>

      <section id="white-label" className="border-b border-slate-200 py-24 dark:border-slate-800">
        <div className="mx-auto grid max-w-7xl gap-12 px-4 sm:px-6 lg:grid-cols-[0.9fr_1.1fr] lg:px-8">
          <div>
            <p className="text-sm font-semibold uppercase tracking-[0.18em] text-blue-700 dark:text-blue-300">
              White label
            </p>
            <h2 className="mt-3 text-4xl font-semibold tracking-tight">
              Brand ISP tetap menjadi wajah pelanggan.
            </h2>
            <p className="mt-5 max-w-xl text-sm leading-6 text-slate-600 dark:text-slate-300">
              Logo, warna, domain, invoice, email, dan walled garden bisa mengikuti identitas ISP tanpa memisahkan alur billing dan jaringan.
            </p>
          </div>
          <div className="grid gap-4 sm:grid-cols-2">
            {[
              ["Domain sendiri", "Custom domain untuk portal pelanggan dan halaman publik tenant."],
              ["Invoice branded", "Logo, alamat, footer, rekening, dan pesan pembayaran tampil konsisten."],
              ["Portal pelanggan", "Pengalaman pelanggan tetap membawa nama ISP lokal."],
              ["Template komunikasi", "WhatsApp, SMS, dan email memakai gaya pesan tenant."],
            ].map(([title, body]) => (
              <article key={title} className="border-t border-slate-200 pt-5 dark:border-slate-800">
                <h3 className="text-lg font-semibold">{title}</h3>
                <p className="mt-2 text-sm leading-6 text-slate-600 dark:text-slate-300">{body}</p>
              </article>
            ))}
          </div>
        </div>
      </section>

      <section id="faq" className="py-24">
        <div className="mx-auto grid max-w-7xl gap-12 px-4 sm:px-6 lg:grid-cols-[0.7fr_1.3fr] lg:px-8">
          <div>
            <p className="text-sm font-semibold uppercase tracking-[0.18em] text-blue-700 dark:text-blue-300">
              FAQ
            </p>
            <h2 className="mt-3 text-4xl font-semibold tracking-tight">
              Pertanyaan sebelum mulai.
            </h2>
          </div>
          <div className="divide-y divide-slate-200 border-y border-slate-200 dark:divide-slate-800 dark:border-slate-800">
            {landingFaqs.map((faq, index) => {
              const isOpen = openFaq === index;
              const panelId = `faq-panel-${index}`;
              return (
              <button
                key={faq.q}
                type="button"
                onClick={() => setOpenFaq(openFaq === index ? -1 : index)}
                className="w-full py-6 text-left"
                aria-expanded={isOpen}
                aria-controls={panelId}
              >
                <span className="flex items-center justify-between gap-4">
                  <span className="text-lg font-semibold">{faq.q}</span>
                  <CaretDown
                    size={18}
                    className={`shrink-0 transition ${isOpen ? "rotate-180" : ""}`}
                  />
                </span>
                <span
                  id={panelId}
                  className={`block max-w-2xl overflow-hidden text-sm leading-6 text-slate-600 transition-all duration-200 dark:text-slate-300 ${
                    isOpen ? "mt-3 max-h-40 opacity-100" : "mt-0 max-h-0 opacity-0"
                  }`}
                >
                  {faq.a}
                </span>
              </button>
              );
            })}
          </div>
        </div>
      </section>

      <section className="px-4 pb-6 sm:px-6 lg:px-8">
        <div className="mx-auto max-w-7xl rounded-xl bg-blue-700 px-6 py-14 text-white sm:px-10 lg:px-14">
          <div className="flex flex-col justify-between gap-8 md:flex-row md:items-center">
            <div>
              <h2 className="max-w-2xl text-4xl font-semibold tracking-tight">
                Siap kelola ISP lebih rapi?
              </h2>
              <p className="mt-3 text-blue-100">
                Coba dengan data operasional kecil dulu, lalu scale saat tim siap.
              </p>
            </div>
            <a
              href="/register"
              className="inline-flex items-center justify-center gap-2 rounded-md bg-white px-5 py-3 text-sm font-semibold text-blue-700 transition hover:bg-blue-50 active:scale-[0.98]"
            >
              Coba Gratis <ArrowRight size={17} />
            </a>
          </div>
        </div>
      </section>

      <footer className="border-t border-slate-200 py-12 dark:border-slate-800">
        <div className="mx-auto grid max-w-7xl gap-8 px-4 text-sm text-slate-500 sm:px-6 md:grid-cols-4 lg:px-8">
          <div className="md:col-span-2">
            <div className="text-lg font-semibold text-slate-950 dark:text-white">ISPBoss</div>
            <p className="mt-3 max-w-sm leading-6">
              Platform billing dan manajemen jaringan untuk ISP Indonesia.
            </p>
          </div>
          <div>
            <h3 className="font-semibold text-slate-900 dark:text-white">Produk</h3>
            <div className="mt-3 grid gap-2">
              <a href="#fitur">Fitur</a>
              <a href="#harga">Harga</a>
              <a href="#white-label">White label</a>
            </div>
          </div>
          <div>
            <h3 className="font-semibold text-slate-900 dark:text-white">Perusahaan</h3>
            <div className="mt-3 grid gap-2">
              <a href="/contact">Kontak</a>
              <a href="/privacy">Kebijakan Privasi</a>
              <span>2026 ISPBoss</span>
            </div>
          </div>
        </div>
      </footer>
    </main>
  );
}

function ProductPreview() {
  return (
    <div className="relative animate-[ispboss-rise_800ms_120ms_ease-out_both]">
      <div className="absolute -inset-8 -z-10 rounded-[2rem] bg-blue-500/10 blur-2xl" />
      <div className="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-[0_40px_90px_-55px_rgba(15,23,42,0.85)] dark:border-slate-800 dark:bg-slate-950">
        <div className="flex h-12 items-center justify-between border-b border-slate-200 px-4 dark:border-slate-800">
          <div className="flex items-center gap-2">
            <span className="h-2.5 w-2.5 rounded-full bg-red-400" />
            <span className="h-2.5 w-2.5 rounded-full bg-amber-400" />
            <span className="h-2.5 w-2.5 rounded-full bg-emerald-400" />
          </div>
          <span className="font-mono text-xs text-slate-400">app.ispboss.id</span>
        </div>
        <div className="grid min-h-[460px] grid-cols-[56px_minmax(0,1fr)] sm:grid-cols-[72px_minmax(0,1fr)]">
          <aside className="border-r border-slate-200 bg-slate-50 p-2 sm:p-3 dark:border-slate-800 dark:bg-slate-900">
            <div className="mb-6 grid h-9 w-9 place-items-center rounded-lg bg-blue-600 text-xs font-bold text-white sm:h-10 sm:w-10">
              IB
            </div>
            {[ChartLineUp, Network, CreditCard, MapTrifold, ShieldCheck].map((Icon, index) => (
              <div
                key={index}
                className={`mb-3 grid h-9 w-9 place-items-center rounded-md sm:h-10 sm:w-10 ${
                  index === 0
                    ? "bg-blue-100 text-blue-700 dark:bg-blue-500/20 dark:text-blue-200"
                    : "text-slate-400"
                }`}
              >
                <Icon size={18} />
              </div>
            ))}
          </aside>
          <div className="min-w-0 p-4 sm:p-5">
            <div className="mb-6 flex items-start justify-between gap-4">
              <div className="min-w-0">
                <div className="h-3 w-28 rounded bg-slate-200 dark:bg-slate-800" />
                <div className="mt-3 h-8 w-36 rounded bg-slate-900 sm:w-52 dark:bg-white" />
              </div>
              <div className="hidden h-10 w-32 shrink-0 rounded-md bg-blue-600 sm:block" />
            </div>
            <div className="grid gap-3 md:grid-cols-3">
              {[
                ["847", "Pelanggan aktif"],
                ["Rp48,7jt", "Pembayaran bulan ini"],
                ["31", "Router online"],
              ].map(([value, label]) => (
                <div key={label} className="border-t border-slate-200 pt-4 dark:border-slate-800">
                  <div className="font-mono text-2xl font-semibold">{value}</div>
                  <div className="mt-1 text-xs text-slate-500">{label}</div>
                </div>
              ))}
            </div>
            <div className="mt-8 grid gap-5 lg:grid-cols-[1.1fr_0.9fr]">
              <div className="min-h-48 rounded-xl border border-slate-200 p-4 dark:border-slate-800">
                <div className="mb-8 flex items-center justify-between">
                  <div className="h-3 w-32 rounded bg-slate-200 dark:bg-slate-800" />
                  <div className="h-3 w-16 rounded bg-blue-200 dark:bg-blue-900" />
                </div>
                <div className="flex h-28 items-end gap-3">
                  {[42, 64, 55, 79, 71, 92, 84].map((height, index) => (
                    <div
                      key={index}
                      className="flex-1 rounded-t bg-blue-600/80"
                      style={{ height: `${height}%` }}
                    />
                  ))}
                </div>
              </div>
              <div className="min-h-48 rounded-xl border border-slate-200 p-4 dark:border-slate-800">
                <div className="mb-5 h-3 w-32 rounded bg-slate-200 dark:bg-slate-800" />
                <div className="space-y-3">
                  {["INV-2605-014", "MK-JKT-03", "ODP-BGR-18", "WA Reminder"].map((item, index) => (
                    <div key={item} className="flex items-center justify-between border-t border-slate-100 pt-3 dark:border-slate-800">
                      <span className="text-sm">{item}</span>
                      <span className={`h-2 w-2 rounded-full ${index === 1 ? "bg-amber-400" : "bg-emerald-400"}`} />
                    </div>
                  ))}
                </div>
              </div>
            </div>
            <div className="mt-5 h-24 rounded-xl border border-slate-200 bg-[linear-gradient(135deg,rgba(37,99,235,0.14),transparent),repeating-linear-gradient(90deg,transparent,transparent_22px,rgba(148,163,184,0.16)_23px),repeating-linear-gradient(0deg,transparent,transparent_22px,rgba(148,163,184,0.16)_23px)] dark:border-slate-800" />
          </div>
        </div>
      </div>
    </div>
  );
}
