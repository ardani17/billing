"use client";

import {
  ChartLineUp,
  CurrencyCircleDollar,
  GearSix,
  Network,
  UsersThree,
} from "@phosphor-icons/react";
import type { Icon } from "@phosphor-icons/react";
import type { ReportTab } from "../hooks/useFilters";

const TABS: { value: ReportTab; label: string; description: string; icon: Icon }[] = [
  { value: "keuangan", label: "Keuangan", description: "Pendapatan, pembayaran, aging", icon: CurrencyCircleDollar },
  { value: "pelanggan", label: "Pelanggan", description: "Pertumbuhan, distribusi, churn", icon: UsersThree },
  { value: "jaringan", label: "Jaringan", description: "Uptime, traffic, signal", icon: Network },
  { value: "operasional", label: "Operasional", description: "Aktivitas, notifikasi, sync", icon: GearSix },
];

interface TabNavigationProps {
  activeTab: ReportTab;
  onTabChange: (tab: ReportTab) => void;
}

export function TabNavigation({ activeTab, onTabChange }: TabNavigationProps) {
  return (
    <div className="min-w-0">
      <div className="mb-3 flex items-center gap-2 text-sm font-semibold text-slate-800">
        <ChartLineUp className="h-4 w-4 text-slate-500" weight="duotone" />
        Jenis laporan
      </div>
      <nav className="grid grid-cols-1 gap-2 sm:grid-cols-2 xl:grid-cols-4" role="tablist">
        {TABS.map((tab) => {
          const isActive = activeTab === tab.value;
          const IconComponent = tab.icon;
          return (
            <button
              key={tab.value}
              role="tab"
              aria-selected={isActive}
              onClick={() => onTabChange(tab.value)}
              className={`group flex min-h-16 min-w-0 items-center gap-3 rounded-lg border px-3 py-3 text-left transition duration-200 focus:outline-none focus:ring-2 focus:ring-slate-900/10 ${
                isActive
                  ? "border-slate-900 bg-slate-950 text-white shadow-sm"
                  : "border-slate-200 bg-white text-slate-700 hover:border-slate-300 hover:bg-slate-50"
              }`}
            >
              <span
                className={`flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-md ${
                  isActive ? "bg-white/12 text-white" : "bg-slate-100 text-slate-500 group-hover:text-slate-900"
                }`}
                aria-hidden="true"
              >
                <IconComponent className="h-5 w-5" weight={isActive ? "duotone" : "regular"} />
              </span>
              <span className="min-w-0">
                <span className="block truncate text-sm font-semibold">{tab.label}</span>
                <span className={`mt-0.5 block truncate text-xs ${isActive ? "text-slate-300" : "text-slate-500"}`}>
                  {tab.description}
                </span>
              </span>
            </button>
          );
        })}
      </nav>
    </div>
  );
}
