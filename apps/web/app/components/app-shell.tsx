"use client";

import type { ReactNode } from "react";
import { useMemo, useState } from "react";
import { usePathname } from "next/navigation";
import {
  Bell,
  Broadcast,
  ArrowsClockwise,
  CaretLeft,
  CaretRight,
  ChartLineUp,
  ChatCircleDots,
  CreditCard,
  GearSix,
  Gauge,
  House,
  List,
  ListChecks,
  MagnifyingGlass,
  MapTrifold,
  Package,
  Pulse,
  Question,
  Receipt,
  SquaresFour,
  Storefront,
  Ticket,
  Users,
  WifiHigh,
  X,
} from "@phosphor-icons/react";

const navGroups = [
  {
    label: "Utama",
    items: [
      { href: "/dashboard", label: "Dashboard", icon: SquaresFour },
      { href: "/customers", label: "Pelanggan", icon: Users },
      { href: "/packages", label: "Paket Internet", icon: Package },
      { href: "/resellers", label: "Reseller", icon: Storefront },
    ],
  },
  {
    label: "Billing",
    items: [
      { href: "/invoices", label: "Invoice", icon: Receipt, badge: "18" },
      { href: "/payments", label: "Pembayaran", icon: CreditCard, badge: "Hari ini" },
      { href: "/vouchers", label: "Voucher", icon: Ticket },
    ],
  },
  {
    label: "Network",
    items: [
      { href: "/mikrotik", label: "MikroTik", icon: WifiHigh, badge: "1" },
      { href: "/olt", label: "OLT", icon: Broadcast },
      { href: "/network-map", label: "Peta Jaringan", icon: MapTrifold },
    ],
  },
  {
    label: "Komunikasi",
    items: [{ href: "/notifications", label: "Notifikasi", icon: ChatCircleDots, badge: "7" }],
  },
  {
    label: "Laporan",
    items: [{ href: "/reports", label: "Laporan", icon: ChartLineUp }],
  },
  {
    label: "Sistem",
    items: [
      { href: "/settings", label: "Pengaturan", icon: GearSix },
      { href: "/help", label: "Bantuan", icon: Question },
    ],
  },
];

const bottomNav = [
  { href: "/dashboard", label: "Home", icon: House },
  { href: "/customers", label: "Pelanggan", icon: Users },
  { href: "/invoices", label: "Billing", icon: Receipt },
  { href: "/mikrotik", label: "MikroTik", icon: WifiHigh },
  { href: "/settings", label: "More", icon: List },
];

function getMikrotikDetailId(pathname: string) {
  const match = pathname.match(/^\/mikrotik\/([^/]+)/);
  const id = match?.[1];
  if (!id || id === "new" || id === "vpn") return null;
  return id;
}

export default function AppShell({ children }: { children: ReactNode }) {
  const pathname = usePathname();
  const [collapsed, setCollapsed] = useState(false);
  const [mobileOpen, setMobileOpen] = useState(false);
  const mikrotikDetailId = getMikrotikDetailId(pathname);

  const currentLabel = useMemo(() => {
    for (const group of navGroups) {
      const item = group.items.find((nav) => nav.href !== "#reports-in-progress" && pathname.startsWith(nav.href));
      if (item) return item.label;
    }
    return "Dashboard";
  }, [pathname]);

  return (
    <div className="min-h-[100dvh] bg-slate-50 text-slate-950">
      {mobileOpen && (
        <div className="fixed inset-0 z-40 bg-slate-950/30 lg:hidden" onClick={() => setMobileOpen(false)} />
      )}

      <aside
        className={`fixed inset-y-0 left-0 z-50 flex w-72 flex-col border-r border-slate-200 bg-white transition-transform duration-200 lg:translate-x-0 ${
          mobileOpen ? "translate-x-0" : "-translate-x-full"
        } ${collapsed ? "lg:w-20" : "lg:w-72"}`}
      >
        <div className="flex h-16 items-center justify-between border-b border-slate-200 px-4">
          <a href="/dashboard" className="flex min-w-0 items-center gap-3">
            <span className="grid h-9 w-9 shrink-0 place-items-center rounded-lg bg-slate-950 text-xs font-black text-white">
              IB
            </span>
            {!collapsed && (
              <span className="min-w-0">
                <span className="block truncate font-semibold tracking-tight">ISPBoss</span>
                <span className="block text-xs text-slate-500">NusaFiber Depok</span>
              </span>
            )}
          </a>
          <button className="grid h-9 w-9 place-items-center rounded-md hover:bg-slate-100 lg:hidden" onClick={() => setMobileOpen(false)} type="button">
            <X size={18} />
          </button>
        </div>

        <nav className="flex-1 overflow-y-auto px-3 py-4">
          {navGroups.map((group) => (
            <div key={group.label} className="mb-5">
              {!collapsed && (
                <p className="px-3 pb-2 text-xs font-semibold uppercase tracking-[0.18em] text-slate-400">
                  {group.label}
                </p>
              )}
              <div className="grid gap-1">
                {group.items.map((item) => {
                  const Icon = item.icon;
                  const active = pathname.startsWith(item.href);
                  const showMikrotikChildren = item.href === "/mikrotik" && !collapsed && mikrotikDetailId;
                  const mikrotikChildren = showMikrotikChildren
                    ? [
                        { href: `/mikrotik/${mikrotikDetailId}`, label: "Overview", icon: Gauge },
                        { href: `/mikrotik/${mikrotikDetailId}/pppoe`, label: "PPPoE users", icon: ListChecks },
                        { href: `/mikrotik/${mikrotikDetailId}/sessions`, label: "Session live", icon: Pulse },
                        { href: `/mikrotik/${mikrotikDetailId}/sync`, label: "Sinkronisasi", icon: ArrowsClockwise },
                      ]
                    : [];
                  return (
                    <div key={item.label}>
                      <a
                        href={item.href}
                        onClick={() => {
                          setMobileOpen(false);
                        }}
                        title={collapsed ? item.label : undefined}
                        className={`flex h-10 items-center gap-3 rounded-md px-3 text-sm font-medium transition ${
                          active
                            ? "bg-blue-50 text-blue-700"
                            : "text-slate-600 hover:bg-slate-50 hover:text-slate-950"
                        } ${collapsed ? "justify-center" : ""}`}
                      >
                        <Icon size={19} />
                        {!collapsed && <span className="min-w-0 flex-1 truncate">{item.label}</span>}
                        {!collapsed && item.badge && (
                          <span className="rounded-full bg-slate-100 px-2 py-0.5 text-[11px] font-semibold text-slate-500">
                            {item.badge}
                          </span>
                        )}
                      </a>
                      {mikrotikChildren.length > 0 && (
                        <div className="ml-8 mt-1 grid gap-1 border-l border-blue-100 pl-3">
                          {mikrotikChildren.map((child) => {
                            const ChildIcon = child.icon;
                            const childActive = pathname === child.href;
                            return (
                              <a
                                key={child.href}
                                href={child.href}
                                onClick={() => {
                                  setMobileOpen(false);
                                }}
                                className={`flex h-9 min-w-0 items-center gap-2 rounded-md px-2 text-sm font-medium transition ${
                                  childActive
                                    ? "bg-blue-600 text-white shadow-sm shadow-blue-100"
                                    : "text-slate-500 hover:bg-slate-50 hover:text-slate-900"
                                }`}
                              >
                                <ChildIcon size={16} />
                                <span className="truncate">{child.label}</span>
                              </a>
                            );
                          })}
                        </div>
                      )}
                    </div>
                  );
                })}
              </div>
            </div>
          ))}
        </nav>

        <div className="hidden border-t border-slate-200 p-3 lg:block">
          <button
            type="button"
            onClick={() => setCollapsed((value) => !value)}
            className="flex h-10 w-full items-center justify-center gap-2 rounded-md text-sm font-medium text-slate-600 hover:bg-slate-50"
          >
            {collapsed ? <CaretRight size={16} /> : <CaretLeft size={16} />}
            {!collapsed && "Collapse"}
          </button>
        </div>
      </aside>

      <div className={`min-h-[100dvh] transition-[padding] duration-200 ${collapsed ? "lg:pl-20" : "lg:pl-72"}`}>
        <header className="sticky top-0 z-30 border-b border-slate-200 bg-white/85 backdrop-blur-xl">
          <div className="flex h-16 items-center justify-between gap-4 px-4 sm:px-6 lg:px-8">
            <div className="flex min-w-0 items-center gap-3">
              <button
                type="button"
                className="grid h-10 w-10 place-items-center rounded-md border border-slate-200 lg:hidden"
                onClick={() => setMobileOpen(true)}
                aria-label="Buka sidebar"
              >
                <List size={20} />
              </button>
              <div className="hidden min-w-0 items-center gap-2 rounded-md bg-slate-100 px-3 py-2 text-sm text-slate-500 md:flex">
                <MagnifyingGlass size={17} />
                <span className="w-72 truncate">Cari pelanggan, invoice, router...</span>
              </div>
              <p className="truncate text-sm font-semibold text-slate-700 md:hidden">{currentLabel}</p>
            </div>
            <div className="flex items-center gap-2">
              <button className="relative grid h-10 w-10 place-items-center rounded-md border border-slate-200 text-slate-600 hover:bg-slate-50" type="button">
                <Bell size={18} />
                <span className="absolute right-2 top-2 h-2 w-2 rounded-full bg-red-500" />
              </button>
              <button className="flex h-10 items-center gap-2 rounded-md border border-slate-200 px-2 text-sm font-semibold hover:bg-slate-50" type="button">
                <span className="grid h-7 w-7 place-items-center rounded-md bg-blue-600 text-xs text-white">AB</span>
                <span className="hidden sm:inline">Admin Budi</span>
              </button>
            </div>
          </div>
        </header>

        <main className="px-4 pb-24 pt-6 sm:px-6 lg:px-8 lg:pb-8">
          <div className="mx-auto max-w-[1400px]">{children}</div>
        </main>

        <nav className="fixed inset-x-0 bottom-0 z-30 grid grid-cols-5 border-t border-slate-200 bg-white lg:hidden">
          {bottomNav.map((item) => {
            const Icon = item.icon;
            const active = pathname.startsWith(item.href);
            return (
              <a
                key={item.href}
                href={item.href}
                className={`flex flex-col items-center gap-1 px-2 py-2 text-[11px] font-medium ${active ? "text-blue-700" : "text-slate-500"}`}
              >
                <Icon size={20} weight={active ? "fill" : "regular"} />
                {item.label}
              </a>
            );
          })}
        </nav>
      </div>
    </div>
  );
}
