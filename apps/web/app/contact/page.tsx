import type { Metadata } from "next";
import type { ReactNode } from "react";
import { ArrowRight, ChatCircleDots, EnvelopeSimple } from "@phosphor-icons/react/dist/ssr";
import { createPageMetadata } from "../lib/seo";

export const metadata: Metadata = createPageMetadata({
  title: "Kontak",
  description:
    "Hubungi tim ISPBoss untuk trial, demo, migrasi billing ISP, integrasi MikroTik, OLT, dan kebutuhan white label.",
  path: "/contact",
});

export default function ContactPage() {
  return (
    <main className="min-h-[100dvh] bg-white text-slate-950">
      <PublicHeader />
      <section className="border-b border-slate-200 px-4 py-20 sm:px-6 lg:px-8">
        <div className="mx-auto grid max-w-6xl gap-12 lg:grid-cols-[0.9fr_1.1fr]">
          <div>
            <p className="text-sm font-semibold uppercase tracking-[0.18em] text-blue-700">
              Kontak
            </p>
            <h1 className="mt-4 text-5xl font-semibold tracking-tight sm:text-6xl">
              Diskusikan kebutuhan billing ISP kamu.
            </h1>
            <p className="mt-6 max-w-xl text-base leading-7 text-slate-600">
              Tim ISPBoss membantu validasi alur trial, migrasi dari billing lama, integrasi MikroTik PPPoE, OLT, FTTH, dan kebutuhan white label.
            </p>
          </div>
          <div className="grid content-start gap-4">
            <ContactCard
              icon={<ChatCircleDots size={24} />}
              title="Trial dan demo"
              body="Buat akun trial untuk validasi alur billing, lalu lanjutkan diskusi kebutuhan operasional."
              href="/register"
              label="Mulai Trial"
            />
            <ContactCard
              icon={<EnvelopeSimple size={24} />}
              title="Email"
              body="Kirim kebutuhan teknis, skala pelanggan, dan perangkat jaringan yang dipakai."
              href="mailto:hello@ispboss.id"
              label="Kirim Email"
            />
          </div>
        </div>
      </section>
      <PublicFooter />
    </main>
  );
}

function ContactCard({
  icon,
  title,
  body,
  href,
  label,
}: {
  icon: ReactNode;
  title: string;
  body: string;
  href: string;
  label: string;
}) {
  return (
    <article className="border-t border-slate-200 py-6">
      <div className="flex items-start gap-4">
        <div className="grid h-12 w-12 shrink-0 place-items-center rounded-md bg-blue-50 text-blue-700">
          {icon}
        </div>
        <div>
          <h2 className="text-xl font-semibold">{title}</h2>
          <p className="mt-2 text-sm leading-6 text-slate-600">{body}</p>
          <a
            href={href}
            className="mt-4 inline-flex items-center gap-2 text-sm font-semibold text-blue-700"
          >
            {label} <ArrowRight size={16} />
          </a>
        </div>
      </div>
    </article>
  );
}

function PublicHeader() {
  return (
    <header className="border-b border-slate-200">
      <div className="mx-auto flex h-16 max-w-6xl items-center justify-between px-4 sm:px-6 lg:px-8">
        <a href="/" className="flex items-center gap-3">
          <span className="grid h-9 w-9 place-items-center rounded-lg bg-slate-950 text-sm font-black text-white">
            IB
          </span>
          <span className="text-lg font-semibold tracking-tight">ISPBoss</span>
        </a>
        <nav className="flex items-center gap-4 text-sm font-semibold">
          <a className="text-slate-600 hover:text-slate-950" href="/">
            Landing
          </a>
          <a className="rounded-md bg-blue-600 px-4 py-2 text-white" href="/register">
            Coba Gratis
          </a>
        </nav>
      </div>
    </header>
  );
}

function PublicFooter() {
  return (
    <footer className="px-4 py-10 text-sm text-slate-500 sm:px-6 lg:px-8">
      <div className="mx-auto flex max-w-6xl flex-col justify-between gap-4 border-t border-slate-200 pt-6 sm:flex-row">
        <span>2026 ISPBoss</span>
        <div className="flex gap-4">
          <a href="/privacy">Kebijakan Privasi</a>
          <a href="/">Landing</a>
        </div>
      </div>
    </footer>
  );
}
