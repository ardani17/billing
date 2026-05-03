import {
  ArrowRight,
  CheckCircle,
  EnvelopeSimple,
  GoogleLogo,
  LockKey,
  User,
} from "@phosphor-icons/react/dist/ssr";
import { Button, FormField, TextInput } from "./ui";

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
  return (
    <AuthFrame title="Masuk ke Dashboard" description="Gunakan akun tenant admin, operator, teknisi, atau kasir.">
      <form className="mt-8 grid gap-5">
        <FormField label="Email">
          <div className="relative">
            <EnvelopeSimple className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" size={18} />
            <input className="h-11 w-full rounded-md border border-slate-300 pl-10 pr-3 text-sm outline-none transition focus:border-blue-500 focus:ring-2 focus:ring-blue-100" placeholder="admin@isp.net" />
          </div>
        </FormField>
        <FormField label="Password">
          <div className="relative">
            <LockKey className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" size={18} />
            <input type="password" className="h-11 w-full rounded-md border border-slate-300 pl-10 pr-3 text-sm outline-none transition focus:border-blue-500 focus:ring-2 focus:ring-blue-100" placeholder="Minimal 8 karakter" />
          </div>
        </FormField>
        <div className="flex items-center justify-between text-sm">
          <label className="inline-flex items-center gap-2 text-slate-600">
            <input type="checkbox" className="h-4 w-4 rounded border-slate-300 text-blue-600" />
            Remember me
          </label>
          <a href="/forgot-password" className="font-semibold text-blue-700">
            Lupa password
          </a>
        </div>
        <Button href="/dashboard">Masuk</Button>
        <button type="button" className="inline-flex h-11 items-center justify-center gap-2 rounded-md border border-slate-300 text-sm font-semibold text-slate-700 transition hover:bg-slate-50">
          <GoogleLogo size={18} />
          Masuk dengan Google
        </button>
      </form>
    </AuthFrame>
  );
}

export function RegisterPage() {
  return (
    <AuthFrame title="Buat akun trial" description="Trial Starter 3 hari dibuat setelah email diverifikasi.">
      <form className="mt-8 grid gap-5">
        <div className="grid gap-5 sm:grid-cols-2">
          <FormField label="Nama lengkap"><TextInput placeholder="Budi Santoso" /></FormField>
          <FormField label="No. WhatsApp"><TextInput placeholder="+62 812..." /></FormField>
        </div>
        <FormField label="Email"><TextInput placeholder="owner@isp.net" /></FormField>
        <FormField label="Nama ISP / Perusahaan"><TextInput placeholder="NusaFiber Depok" /></FormField>
        <div className="grid gap-5 sm:grid-cols-2">
          <FormField label="Password"><TextInput placeholder="Minimal 8 karakter" /></FormField>
          <FormField label="Konfirmasi"><TextInput placeholder="Ulangi password" /></FormField>
        </div>
        <Button href="/verify-email">Coba Gratis 3 Hari</Button>
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
