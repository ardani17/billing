"use client";

import { useState } from "react";
import {
  ArrowRight,
  CheckCircle,
  EnvelopeSimple,
  GoogleLogo,
  LockKey,
  User,
} from "@phosphor-icons/react";
import { Button, FormField, TextInput } from "./ui";

type AuthEnvelope<T> = {
  success: boolean;
  data?: T;
  error?: {
    message?: string;
    details?: { field: string; message: string }[];
  };
};

type LoginResponse = {
  redirect_path?: string;
  user?: {
    role: string;
  };
};

function authErrorMessage(error: unknown) {
  if (error instanceof Error) return error.message;
  return "Terjadi kesalahan autentikasi";
}

async function authPost<T>(action: string, payload: unknown) {
  const response = await fetch(`/api/auth/${action}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
  const envelope = (await response.json()) as AuthEnvelope<T>;

  if (!response.ok || !envelope.success) {
    const detail = envelope.error?.details?.[0]?.message;
    throw new Error(detail || envelope.error?.message || "Request auth gagal");
  }

  return envelope.data as T;
}

function AuthFrame({
  title,
  description,
  children,
}: {
  title: string;
  description: string;
  children: React.ReactNode;
}) {
  return (
    <main className="grid min-h-[100dvh] bg-slate-950 text-white lg:grid-cols-[0.95fr_1.05fr]">
      <section className="relative hidden overflow-hidden p-10 lg:flex lg:flex-col lg:justify-between">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_25%_20%,rgba(37,99,235,0.32),transparent_32%),linear-gradient(135deg,#0f172a,#020617)]" />
        <div className="relative">
          <a href="/landing" className="inline-flex items-center gap-3">
            <span className="grid h-10 w-10 place-items-center rounded-lg bg-white text-sm font-black text-slate-950">
              IB
            </span>
            <span className="font-semibold tracking-tight">ISPBoss</span>
          </a>
          <h1 className="mt-20 max-w-xl text-6xl font-semibold leading-none tracking-tight">
            Kelola ISP dari satu dashboard.
          </h1>
          <p className="mt-6 max-w-md text-slate-300">
            Billing, pelanggan, jaringan, notifikasi, dan operasional tenant berjalan dalam satu platform.
          </p>
        </div>
        <div className="relative grid gap-3 text-sm text-slate-300">
          {["Trial 3 hari", "Tanpa kartu kredit", "Langsung ke dashboard"].map((item) => (
            <span key={item} className="inline-flex items-center gap-2">
              <CheckCircle className="text-blue-300" size={18} weight="fill" />
              {item}
            </span>
          ))}
        </div>
      </section>
      <section className="flex items-center justify-center bg-white px-4 py-10 text-slate-950 sm:px-6 lg:px-8">
        <div className="w-full max-w-md">
          <a href="/landing" className="mb-10 inline-flex items-center gap-3 lg:hidden">
            <span className="grid h-10 w-10 place-items-center rounded-lg bg-slate-950 text-sm font-black text-white">
              IB
            </span>
            <span className="font-semibold tracking-tight">ISPBoss</span>
          </a>
          <h2 className="text-3xl font-semibold tracking-tight">{title}</h2>
          <p className="mt-2 text-sm leading-6 text-slate-500">{description}</p>
          {children}
        </div>
      </section>
    </main>
  );
}

export function LoginPage() {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [rememberMe, setRememberMe] = useState(true);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setLoading(true);
    setError("");

    try {
      const result = await authPost<LoginResponse>("login", {
        email,
        password,
        remember_me: rememberMe,
      });
      window.location.href = result.redirect_path || (result.user?.role === "super_admin" ? "/super-admin" : "/dashboard");
    } catch (err) {
      setError(authErrorMessage(err));
    } finally {
      setLoading(false);
    }
  }

  return (
    <AuthFrame title="Masuk ke Dashboard" description="Gunakan akun tenant admin, operator, teknisi, atau kasir.">
      <form onSubmit={handleSubmit} className="mt-8 grid gap-5">
        <FormField label="Email">
          <div className="relative">
            <EnvelopeSimple className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" size={18} />
            <input
              value={email}
              onChange={(event) => setEmail(event.target.value)}
              type="email"
              className="h-11 w-full rounded-md border border-slate-300 pl-10 pr-3 text-sm outline-none transition focus:border-blue-500 focus:ring-2 focus:ring-blue-100"
              placeholder="admin@isp.net"
              required
            />
          </div>
        </FormField>
        <FormField label="Password">
          <div className="relative">
            <LockKey className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" size={18} />
            <input
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              type="password"
              className="h-11 w-full rounded-md border border-slate-300 pl-10 pr-3 text-sm outline-none transition focus:border-blue-500 focus:ring-2 focus:ring-blue-100"
              placeholder="Minimal 8 karakter"
              required
            />
          </div>
        </FormField>
        <div className="flex items-center justify-between text-sm">
          <label className="inline-flex items-center gap-2 text-slate-600">
            <input checked={rememberMe} onChange={(event) => setRememberMe(event.target.checked)} type="checkbox" className="h-4 w-4 rounded border-slate-300 text-blue-600" />
            Remember me
          </label>
          <a href="/forgot-password" className="font-semibold text-blue-700">
            Lupa password
          </a>
        </div>
        {error && <div className="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">{error}</div>}
        <button
          type="submit"
          disabled={loading}
          className="inline-flex h-11 items-center justify-center rounded-md bg-blue-600 px-4 text-sm font-semibold text-white transition hover:bg-blue-700 disabled:cursor-not-allowed disabled:opacity-60"
        >
          {loading ? "Memproses..." : "Masuk"}
        </button>
        <button type="button" className="inline-flex h-11 items-center justify-center gap-2 rounded-md border border-slate-300 text-sm font-semibold text-slate-700 transition hover:bg-slate-50">
          <GoogleLogo size={18} />
          Masuk dengan Google
        </button>
      </form>
    </AuthFrame>
  );
}

export function RegisterPage() {
  const [form, setForm] = useState({
    name: "",
    phone: "",
    email: "",
    company_name: "",
    password: "",
    password_confirmation: "",
    agree_terms: true,
  });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  function updateForm(field: keyof typeof form, value: string | boolean) {
    setForm((current) => ({ ...current, [field]: value }));
  }

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setLoading(true);
    setError("");

    try {
      await authPost("register", form);
      window.location.href = "/verify-email";
    } catch (err) {
      setError(authErrorMessage(err));
    } finally {
      setLoading(false);
    }
  }

  return (
    <AuthFrame title="Buat akun trial" description="Trial Starter 3 hari dibuat setelah email diverifikasi.">
      <form onSubmit={handleSubmit} className="mt-8 grid gap-5">
        <div className="grid gap-5 sm:grid-cols-2">
          <FormField label="Nama lengkap"><TextInput value={form.name} onChange={(event) => updateForm("name", event.target.value)} placeholder="Budi Santoso" required /></FormField>
          <FormField label="No. WhatsApp"><TextInput value={form.phone} onChange={(event) => updateForm("phone", event.target.value)} placeholder="+62812..." required /></FormField>
        </div>
        <FormField label="Email"><TextInput value={form.email} onChange={(event) => updateForm("email", event.target.value)} type="email" placeholder="owner@isp.net" required /></FormField>
        <FormField label="Nama ISP / Perusahaan"><TextInput value={form.company_name} onChange={(event) => updateForm("company_name", event.target.value)} placeholder="NusaFiber Depok" required /></FormField>
        <div className="grid gap-5 sm:grid-cols-2">
          <FormField label="Password"><TextInput value={form.password} onChange={(event) => updateForm("password", event.target.value)} type="password" placeholder="Minimal 8 karakter" required /></FormField>
          <FormField label="Konfirmasi"><TextInput value={form.password_confirmation} onChange={(event) => updateForm("password_confirmation", event.target.value)} type="password" placeholder="Ulangi password" required /></FormField>
        </div>
        <label className="inline-flex items-center gap-2 text-sm text-slate-600">
          <input checked={form.agree_terms} onChange={(event) => updateForm("agree_terms", event.target.checked)} type="checkbox" className="h-4 w-4 rounded border-slate-300 text-blue-600" />
          Saya menyetujui syarat penggunaan ISPBoss
        </label>
        {error && <div className="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">{error}</div>}
        <button
          type="submit"
          disabled={loading}
          className="inline-flex h-11 items-center justify-center rounded-md bg-blue-600 px-4 text-sm font-semibold text-white transition hover:bg-blue-700 disabled:cursor-not-allowed disabled:opacity-60"
        >
          {loading ? "Memproses..." : "Coba Gratis 3 Hari"}
        </button>
        <button type="button" className="inline-flex h-11 items-center justify-center gap-2 rounded-md border border-slate-300 text-sm font-semibold text-slate-700 transition hover:bg-slate-50">
          <GoogleLogo size={18} />
          Daftar dengan Google
        </button>
        <p className="text-center text-sm text-slate-500">
          Sudah punya akun? <a href="/login" className="font-semibold text-blue-700">Masuk</a>
        </p>
      </form>
    </AuthFrame>
  );
}

export function ForgotPasswordPage() {
  return (
    <AuthFrame title="Reset password" description="Link reset dikirim ke email dan berlaku 1 jam.">
      <form className="mt-8 grid gap-5">
        <FormField label="Email akun"><TextInput placeholder="admin@isp.net" /></FormField>
        <Button>Kirim Link Reset</Button>
        <a href="/login" className="text-center text-sm font-semibold text-blue-700">
          Kembali ke login
        </a>
      </form>
    </AuthFrame>
  );
}

export function VerifyEmailPage() {
  return (
    <AuthFrame title="Cek email kamu" description="Kami mengirim link verifikasi. Setelah klik link, akun langsung masuk ke dashboard.">
      <div className="mt-8 rounded-xl border border-slate-200 bg-slate-50 p-5">
        <div className="flex items-start gap-4">
          <span className="grid h-11 w-11 shrink-0 place-items-center rounded-lg bg-blue-100 text-blue-700">
            <User size={20} />
          </span>
          <div>
            <p className="font-semibold">Email verifikasi sudah dikirim</p>
            <p className="mt-1 text-sm leading-6 text-slate-500">
              Jika belum menerima email, cek folder spam atau kirim ulang setelah 60 detik.
            </p>
          </div>
        </div>
      </div>
      <div className="mt-6 flex flex-col gap-3 sm:flex-row">
        <Button variant="secondary">Kirim Ulang</Button>
        <Button href="/dashboard">Saya Sudah Verifikasi <ArrowRight size={16} /></Button>
      </div>
    </AuthFrame>
  );
}
