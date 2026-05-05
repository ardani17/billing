"use client";

import type { ReactNode } from "react";
import { useEffect, useState } from "react";
import { LockSimple } from "@phosphor-icons/react";

type AddonModule = "mikrotik" | "fiber_network";

type ModuleCapabilities = {
  billing_core: boolean;
  mikrotik: boolean;
  fiber_network: boolean;
};

const defaultModules: ModuleCapabilities = {
  billing_core: true,
  mikrotik: false,
  fiber_network: false,
};

export function ModuleGate({
  moduleCode,
  moduleName,
  children,
}: {
  moduleCode: AddonModule;
  moduleName: string;
  children: ReactNode;
}) {
  const [modules, setModules] = useState<ModuleCapabilities>(defaultModules);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let mounted = true;

    async function loadModules() {
      try {
        const response = await fetch("/api/billing/tenant/modules", { cache: "no-store" });
        const payload = await response.json();
        const nextModules = payload?.data?.modules;
        if (mounted && nextModules) {
          setModules({
            billing_core: nextModules.billing_core !== false,
            mikrotik: nextModules.mikrotik === true,
            fiber_network: nextModules.fiber_network === true,
          });
        }
      } finally {
        if (mounted) setLoading(false);
      }
    }

    loadModules();
    return () => {
      mounted = false;
    };
  }, []);

  if (loading) {
    return <div className="h-40 animate-pulse rounded-md border border-slate-200 bg-white" />;
  }

  if (!modules[moduleCode]) {
    return (
      <section className="rounded-md border border-slate-200 bg-white p-8">
        <div className="flex max-w-2xl items-start gap-4">
          <span className="grid h-11 w-11 shrink-0 place-items-center rounded-md bg-slate-100 text-slate-600">
            <LockSimple size={22} />
          </span>
          <div className="min-w-0">
            <p className="text-xs font-semibold uppercase tracking-[0.18em] text-slate-400">Modul nonaktif</p>
            <h1 className="mt-2 text-2xl font-semibold tracking-tight text-slate-950">{moduleName} belum aktif</h1>
            <p className="mt-2 text-sm leading-6 text-slate-600">
              Billing Core tetap berjalan normal. Aktifkan add-on ini dari pengaturan subscription sebelum memakai fitur operasionalnya.
            </p>
            <a
              href="/settings/subscription"
              className="mt-5 inline-flex h-10 items-center rounded-md bg-blue-600 px-4 text-sm font-semibold text-white hover:bg-blue-700"
            >
              Buka subscription
            </a>
          </div>
        </div>
      </section>
    );
  }

  return children;
}
