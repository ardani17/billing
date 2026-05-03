"use client";

import type { ReportTab } from "../hooks/useFilters";

const TABS: { value: ReportTab; label: string; icon: string }[] = [
  { value: "keuangan", label: "Keuangan", icon: "💰" },
  { value: "pelanggan", label: "Pelanggan", icon: "👥" },
  { value: "jaringan", label: "Jaringan", icon: "📡" },
  { value: "operasional", label: "Operasional", icon: "⚙️" },
];

interface TabNavigationProps {
  activeTab: ReportTab;
  onTabChange: (tab: ReportTab) => void;
}

export function TabNavigation({ activeTab, onTabChange }: TabNavigationProps) {
  return (
    <div className="overflow-x-auto scrollbar-hide">
      <nav className="flex gap-1 border-b border-slate-200" role="tablist">
        {TABS.map((tab) => {
          const isActive = activeTab === tab.value;
          return (
            <button
              key={tab.value}
              role="tab"
              aria-selected={isActive}
              onClick={() => onTabChange(tab.value)}
              className={`flex flex-shrink-0 items-center gap-2 whitespace-nowrap border-b-2 px-4 py-3 text-sm font-medium transition-colors ${
                isActive
                  ? "border-blue-600 text-blue-600"
                  : "border-transparent text-slate-500 hover:border-slate-300 hover:text-slate-700"
              }`}
              style={{ minHeight: 44, minWidth: 44 }}
            >
              <span aria-hidden="true">{tab.icon}</span>
              {tab.label}
            </button>
          );
        })}
      </nav>
    </div>
  );
}
