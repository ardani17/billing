import type { Metadata } from "next";
import { createPageMetadata } from "../lib/seo";

export const metadata: Metadata = createPageMetadata({
  title: "Kebijakan Privasi",
  description:
    "Kebijakan privasi ISPBoss untuk data tenant, pelanggan ISP, billing, pembayaran, notifikasi, dan integrasi jaringan.",
  path: "/privacy",
});

export default function PrivacyPage() {
  return (
    <main className="min-h-[100dvh] bg-white text-slate-950">
      <header className="border-b border-slate-200">
        <div className="mx-auto flex h-16 max-w-4xl items-center justify-between px-4 sm:px-6 lg:px-8">
          <a href="/" className="flex items-center gap-3">
            <span className="grid h-9 w-9 place-items-center rounded-lg bg-slate-950 text-sm font-black text-white">
              IB
            </span>
            <span className="text-lg font-semibold tracking-tight">ISPBoss</span>
          </a>
          <a className="text-sm font-semibold text-blue-700" href="/contact">
            Kontak
          </a>
        </div>
      </header>
      <article className="mx-auto max-w-4xl px-4 py-16 sm:px-6 lg:px-8">
        <p className="text-sm font-semibold uppercase tracking-[0.18em] text-blue-700">
          Legal
        </p>
        <h1 className="mt-4 text-4xl font-semibold tracking-tight sm:text-5xl">
          Kebijakan Privasi ISPBoss
        </h1>
        <p className="mt-5 text-sm leading-6 text-slate-600">
          Halaman ini menjelaskan garis besar pengelolaan data pada platform ISPBoss. Detail final dapat disesuaikan dengan kontrak, kebijakan tenant, dan kebutuhan regulasi yang berlaku.
        </p>
        <div className="mt-12 space-y-10">
          {[
            [
              "Data yang dikelola",
              "ISPBoss dapat memproses data akun tenant, user operasional, pelanggan ISP, paket, invoice, pembayaran, notifikasi, audit log, serta data teknis jaringan yang diperlukan untuk menjalankan layanan.",
            ],
            [
              "Penggunaan data",
              "Data dipakai untuk menyediakan billing ISP, portal pelanggan, reminder pembayaran, laporan operasional, integrasi MikroTik, OLT, dan fitur pendukung lain yang diaktifkan tenant.",
            ],
            [
              "Keamanan dan akses",
              "Akses data dibatasi berdasarkan role pengguna, tenant, dan kebutuhan operasional. Aktivitas penting dapat dicatat untuk kebutuhan audit dan keamanan.",
            ],
            [
              "Integrasi pihak ketiga",
              "Jika tenant mengaktifkan payment gateway, WhatsApp, SMS, email, atau integrasi jaringan tertentu, data yang relevan dapat dikirim ke penyedia layanan tersebut sesuai konfigurasi tenant.",
            ],
            [
              "Retensi dan penghapusan",
              "Retensi data mengikuti kebutuhan operasional, kepatuhan, dan perjanjian layanan. Tenant dapat menghubungi tim ISPBoss untuk permintaan ekspor atau penghapusan sesuai kebijakan yang berlaku.",
            ],
          ].map(([title, body]) => (
            <section key={title} className="border-t border-slate-200 pt-6">
              <h2 className="text-xl font-semibold">{title}</h2>
              <p className="mt-3 text-sm leading-6 text-slate-600">{body}</p>
            </section>
          ))}
        </div>
      </article>
    </main>
  );
}
