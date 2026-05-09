"use client";

import { ChartBarHorizontal } from "@phosphor-icons/react";

interface EmptyStateProps {
  /** Pesan utama*/
  message?: string;
  /** Deskripsi tambahan*/
  description?: string;
}

export function EmptyState({
  message = "Belum ada data",
  description = "Belum ada data untuk periode ini. Coba ubah filter atau pilih periode lain.",
}: EmptyStateProps) {
  return (
    <div className="rounded-lg border border-dashed border-slate-300 bg-white px-5 py-10 text-center shadow-sm shadow-slate-200/60">
      <div className="mx-auto mb-3 flex h-12 w-12 items-center justify-center rounded-lg bg-slate-100 text-slate-500">
        <ChartBarHorizontal className="h-6 w-6" weight="duotone" aria-hidden="true" />
      </div>
      <h3 className="text-sm font-semibold text-slate-900">{message}</h3>
      <p className="mx-auto mt-1 max-w-md text-sm leading-6 text-slate-500">{description}</p>
    </div>
  );
}
