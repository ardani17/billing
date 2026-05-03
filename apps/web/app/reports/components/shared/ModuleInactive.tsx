"use client";

interface ModuleInactiveProps {
  /** Nama modul yang belum aktif (contoh: "MikroTik", "OLT") */
  moduleName: string;
}

export function ModuleInactive({ moduleName }: ModuleInactiveProps) {
  return (
    <div className="flex items-start gap-3 rounded-xl border border-slate-200 bg-slate-50 px-5 py-6">
      <div className="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-full bg-slate-200">
        <svg
          className="h-5 w-5 text-slate-500"
          fill="none"
          viewBox="0 0 24 24"
          strokeWidth={1.5}
          stroke="currentColor"
          aria-hidden="true"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            d="M18.364 18.364A9 9 0 0 0 5.636 5.636m12.728 12.728A9 9 0 0 1 5.636 5.636m12.728 12.728L5.636 5.636"
          />
        </svg>
      </div>
      <div>
        <h3 className="text-sm font-semibold text-slate-900">
          Modul {moduleName} belum aktif
        </h3>
        <p className="mt-1 text-sm text-slate-500">
          Aktifkan modul {moduleName} untuk melihat laporan ini. Kunjungi halaman
          Pengaturan untuk mengaktifkan modul.
        </p>
      </div>
    </div>
  );
}
