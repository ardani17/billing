"use client";

import type { ReactNode } from "react";
import { usePathname } from "next/navigation";
import {
  Buildings,
  ClockCounterClockwise,
  CreditCard,
  GearSix,
  House,
  List,
  Pulse,
  ShieldCheck,
  UserSwitch,
} from "@phosphor-icons/react";

const navItems = [
  { href: "/super-admin", label: "Overview", icon: House },
  { href: "/super-admin/tenants", label: "Tenants", icon: Buildings },
  { href: "/super-admin/subscriptions", label: "Subscriptions", icon: CreditCard },
  { href: "/super-admin/support", label: "Support", icon: UserSwitch },
  { href: "/super-admin/health", label: "Service Health", icon: Pulse },
  { href: "/super-admin/audit", label: "Audit Global", icon: ClockCounterClockwise },
  { href: "/super-admin/settings", label: "Platform Settings", icon: GearSix },
];

export default function SuperAdminShell({ children }: { children: ReactNode }) {
  const pathname = usePathname();

  return (
    <div className="min-h-[100dvh] bg-zinc-50 text-zinc-950">
      <aside className="fixed inset-y-0 left-0 z-40 hidden w-72 flex-col border-r border-zinc-200 bg-zinc-950 text-white lg:flex">
        <div className="flex h-16 items-center gap-3 border-b border-white/10 px-5">
          <span className="grid h-9 w-9 place-items-center rounded-lg bg-white text-sm font-black text-zinc-950">IB</span>
          <div className="min-w-0">
            <p className="truncate font-semibold">ISPBoss Console</p>
            <p className="text-xs text-zinc-400">Super Admin</p>
          </div>
        </div>
        <nav className="grid gap-1 px-3 py-4">
          {navItems.map((item) => {
            const Icon = item.icon;
            const active = pathname === item.href || (item.href !== "/super-admin" && pathname.startsWith(item.href));
            return (
              <a
                key={item.href}
                href={item.href}
                className={`flex h-10 items-center gap-3 rounded-md px-3 text-sm font-medium transition ${
                  active ? "bg-white text-zinc-950" : "text-zinc-300 hover:bg-white/10 hover:text-white"
                }`}
              >
                <Icon size={18} />
                <span className="truncate">{item.label}</span>
              </a>
            );
          })}
        </nav>
        <div className="mt-auto border-t border-white/10 p-4 text-xs leading-5 text-zinc-400">
          Role ini lintas tenant. Aksi data tenant dilakukan via impersonate agar audit trail tetap jelas.
        </div>
      </aside>

      <div className="lg:pl-72">
        <header className="sticky top-0 z-30 border-b border-zinc-200 bg-white/90 backdrop-blur-xl">
          <div className="flex h-16 items-center justify-between gap-3 px-4 sm:px-6 lg:px-8">
            <div className="flex min-w-0 items-center gap-3">
              <span className="grid h-9 w-9 shrink-0 place-items-center rounded-lg bg-zinc-950 text-xs font-black text-white lg:hidden">
                IB
              </span>
              <div className="min-w-0">
                <p className="truncate text-sm font-semibold">Super Admin Console</p>
                <p className="truncate text-xs text-zinc-500">Owner platform ISPBoss</p>
              </div>
            </div>
            <a href="/dashboard" className="inline-flex items-center gap-2 rounded-md border border-zinc-300 px-3 py-2 text-sm font-semibold text-zinc-700">
              <ShieldCheck size={16} />
              Tenant View
            </a>
          </div>
        </header>

        <main className="px-4 pb-24 pt-6 sm:px-6 lg:px-8 lg:pb-8">
          <div className="mx-auto max-w-[1400px]">{children}</div>
        </main>

        <nav className="fixed inset-x-0 bottom-0 z-30 grid grid-cols-5 border-t border-zinc-200 bg-white lg:hidden">
          {navItems.slice(0, 5).map((item) => {
            const Icon = item.icon;
            const active = pathname === item.href || (item.href !== "/super-admin" && pathname.startsWith(item.href));
            return (
              <a key={item.href} href={item.href} className={`flex flex-col items-center gap-1 px-1 py-2 text-[10px] font-medium ${active ? "text-zinc-950" : "text-zinc-500"}`}>
                <Icon size={20} weight={active ? "fill" : "regular"} />
                <span className="max-w-full truncate">{item.label}</span>
              </a>
            );
          })}
        </nav>
      </div>
    </div>
  );
}
