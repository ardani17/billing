export default function Home() {
  return (
    <main className="flex min-h-[100dvh] flex-col items-center justify-center bg-slate-950 px-4 text-center text-white">
      <p className="text-sm font-semibold uppercase tracking-[0.18em] text-blue-300">
        ISPBoss
      </p>
      <h1 className="mt-4 max-w-3xl text-5xl font-semibold tracking-tight">
        Platform billing dan manajemen jaringan untuk ISP.
      </h1>
      <p className="mt-5 max-w-xl text-slate-300">
        Pilih landing page publik, dashboard tenant, atau console owner platform.
      </p>
      <div className="mt-8 flex flex-col gap-3 sm:flex-row">
        <a className="rounded-md bg-blue-600 px-5 py-3 text-sm font-semibold text-white" href="/landing">
          Landing Page
        </a>
        <a className="rounded-md border border-white/15 px-5 py-3 text-sm font-semibold text-white" href="/dashboard">
          Tenant Dashboard
        </a>
        <a className="rounded-md border border-white/15 px-5 py-3 text-sm font-semibold text-white" href="/super-admin">
          Super Admin
        </a>
      </div>
    </main>
  );
}
