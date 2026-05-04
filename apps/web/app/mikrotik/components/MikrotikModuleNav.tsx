"use client";

import { Plus, ShieldCheck, TreeStructure, WifiHigh } from "@phosphor-icons/react";
import type { Icon } from "@phosphor-icons/react";
import { usePathname } from "next/navigation";

type NavItem = {
  href: string;
  label: string;
  description: string;
  icon: Icon;
  match: (pathname: string) => boolean;
};

const baseItems: NavItem[] = [
  {
    href: "/mikrotik",
    label: "Router",
    description: "Status dan koneksi",
    icon: WifiHigh,
    match: (pathname) => pathname === "/mikrotik",
  },
  {
    href: "/mikrotik/new",
    label: "Tambah router",
    description: "Simpan akses API",
    icon: Plus,
    match: (pathname) => pathname === "/mikrotik/new",
  },
  {
    href: "/mikrotik/vpn",
    label: "VPN tunnel",
    description: "Akses remote aman",
    icon: ShieldCheck,
    match: (pathname) => pathname.startsWith("/mikrotik/vpn"),
  },
];

function isRouterDetail(pathname: string) {
  return pathname.startsWith("/mikrotik/") && pathname !== "/mikrotik/new" && !pathname.startsWith("/mikrotik/vpn");
}

export function MikrotikModuleNav() {
  const pathname = usePathname();
  const items = isRouterDetail(pathname)
    ? [
        ...baseItems,
        {
          href: pathname,
          label: "Detail router",
          description: "PPPoE dan resource",
          icon: TreeStructure,
          match: () => true,
        },
      ]
    : baseItems;

  return (
    <nav
      aria-label="Navigasi MikroTik"
      className="overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm"
    >
      <div className="grid grid-cols-1 gap-px bg-slate-200 sm:grid-cols-2 xl:grid-cols-4">
        {items.map((item) => {
          const active = item.match(pathname);
          const Icon = item.icon;
          return (
            <a
              key={item.label}
              href={item.href}
              aria-current={active ? "page" : undefined}
              className={`group flex min-h-16 min-w-0 items-center gap-3 bg-white px-4 py-3 text-left transition focus:outline-none focus:ring-2 focus:ring-inset focus:ring-blue-100 ${
                active ? "text-blue-700" : "text-slate-600 hover:bg-blue-50/60 hover:text-blue-700"
              }`}
            >
              <span
                className={`grid h-10 w-10 shrink-0 place-items-center rounded-md transition ${
                  active ? "bg-blue-600 text-white shadow-sm shadow-blue-200" : "bg-slate-100 text-slate-500 group-hover:bg-blue-100 group-hover:text-blue-700"
                }`}
              >
                <Icon size={20} weight={active ? "duotone" : "regular"} />
              </span>
              <span className="min-w-0">
                <span className="block truncate text-sm font-semibold">{item.label}</span>
                <span className="mt-0.5 block truncate text-xs text-slate-500">{item.description}</span>
              </span>
            </a>
          );
        })}
      </div>
      <div className="flex flex-col gap-1 border-t border-slate-200 bg-slate-50 px-4 py-3 text-xs text-slate-500 sm:flex-row sm:items-center sm:justify-between">
        <span className="font-medium text-slate-700">RouterOS API berjalan manual/on-demand.</span>
        <span>Test, sync, dan baca session hanya saat tombol aksi ditekan.</span>
      </div>
    </nav>
  );
}
