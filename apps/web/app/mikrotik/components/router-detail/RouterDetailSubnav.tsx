"use client";

import { ArrowsClockwise, Gauge, ListChecks, Pulse } from "@phosphor-icons/react";
import type { Icon } from "@phosphor-icons/react";
import type { MikrotikDetailSection } from "../../lib/types";

type Item = {
  id: MikrotikDetailSection;
  href: string;
  label: string;
  description: string;
  icon: Icon;
};

export function RouterDetailSubnav({
  routerId,
  active,
}: {
  routerId: string;
  active: MikrotikDetailSection;
}) {
  const items: Item[] = [
    {
      id: "overview",
      href: `/mikrotik/${routerId}`,
      label: "Overview",
      description: "Resource dan edit router",
      icon: Gauge,
    },
    {
      id: "pppoe",
      href: `/mikrotik/${routerId}/pppoe`,
      label: "PPPoE users",
      description: "Secret terkelola",
      icon: ListChecks,
    },
    {
      id: "sessions",
      href: `/mikrotik/${routerId}/sessions`,
      label: "Session live",
      description: "Active PPPoE manual",
      icon: Pulse,
    },
    {
      id: "sync",
      href: `/mikrotik/${routerId}/sync`,
      label: "Sinkronisasi",
      description: "Status data router",
      icon: ArrowsClockwise,
    },
  ];

  return (
    <aside className="min-w-0">
      <nav
        aria-label="Subnavigasi detail MikroTik"
        className="flex gap-2 overflow-x-auto rounded-xl border border-slate-200 bg-white p-2 shadow-sm lg:sticky lg:top-4 lg:block lg:space-y-1 lg:overflow-visible"
      >
        {items.map((item) => {
          const Icon = item.icon;
          const isActive = active === item.id;
          return (
            <a
              key={item.id}
              href={item.href}
              aria-current={isActive ? "page" : undefined}
              className={`group flex min-w-[13rem] items-center gap-3 rounded-lg px-3 py-3 text-left transition focus:outline-none focus:ring-2 focus:ring-blue-100 lg:min-w-0 ${
                isActive
                  ? "bg-blue-50 text-blue-700 ring-1 ring-inset ring-blue-100"
                  : "text-slate-600 hover:bg-slate-50 hover:text-slate-900"
              }`}
            >
              <span
                className={`grid h-9 w-9 shrink-0 place-items-center rounded-md ${
                  isActive ? "bg-blue-600 text-white" : "bg-slate-100 text-slate-500 group-hover:text-slate-700"
                }`}
              >
                <Icon size={18} weight={isActive ? "duotone" : "regular"} />
              </span>
              <span className="min-w-0">
                <span className="block truncate text-sm font-semibold">{item.label}</span>
                <span className="mt-0.5 block truncate text-xs text-slate-500">{item.description}</span>
              </span>
            </a>
          );
        })}
      </nav>
    </aside>
  );
}
