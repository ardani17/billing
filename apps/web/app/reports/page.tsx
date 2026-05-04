"use client";

import { ChartLineUp, ClockCounterClockwise, FileText, Pulse } from "@phosphor-icons/react";
import AppShell from "../components/app-shell";
import { FilterBar } from "./components/FilterBar";
import { TabNavigation } from "./components/TabNavigation";
import { useFilters } from "./hooks/useFilters";
import { RevenueSection } from "./components/financial/RevenueSection";
import { AgingSection } from "./components/financial/AgingSection";
import { PaymentSection } from "./components/financial/PaymentSection";
import { VoucherSection } from "./components/financial/VoucherSection";
import { ProfitLossSection } from "./components/financial/ProfitLossSection";
import { RevenueByAreaSection } from "./components/financial/RevenueByAreaSection";
import { GrowthSection } from "./components/customer/GrowthSection";
import { DistributionSection } from "./components/customer/DistributionSection";
import { ChurnSection } from "./components/customer/ChurnSection";
import { UptimeSection } from "./components/network/UptimeSection";
import { TrafficSection } from "./components/network/TrafficSection";
import { SignalSection } from "./components/network/SignalSection";
import { CapacitySection } from "./components/network/CapacitySection";
import { ActivitySection } from "./components/operational/ActivitySection";
import { NotificationSection } from "./components/operational/NotificationSection";
import { SyncSection } from "./components/operational/SyncSection";

export default function ReportPage() {
  const {
    filters,
    setTab,
    setPeriodPreset,
    setCustomPeriod,
    setComparisonType,
    setAreaId,
    setPackageId,
    reset,
    toReportFilter,
  } = useFilters();

  const reportFilter = toReportFilter();
  const periodLabel = formatReportPeriod(filters.periodPreset, filters.periodStart, filters.periodEnd);

  return (
    <AppShell>
      <div className="flex w-full flex-col gap-6">
        <section className="overflow-hidden rounded-lg border border-slate-200 bg-white shadow-sm shadow-slate-200/70">
          <div className="flex flex-col gap-5 border-b border-slate-100 px-4 py-5 sm:px-5 lg:flex-row lg:items-end lg:justify-between lg:px-6">
            <div className="min-w-0">
              <div className="mb-3 inline-flex items-center gap-2 rounded-md bg-blue-50 px-2.5 py-1 text-xs font-semibold uppercase tracking-[0.16em] text-blue-700">
                <ChartLineUp className="h-4 w-4" weight="duotone" />
                Laporan
              </div>
              <h1 className="text-balance text-2xl font-semibold tracking-tight text-slate-950 sm:text-3xl">
                Laporan dan analitik
              </h1>
              <p className="mt-2 max-w-2xl text-sm leading-6 text-slate-500">
                Pantau pendapatan, pelanggan, jaringan, dan aktivitas operasional dari data terbaru tenant.
              </p>
            </div>

            <div className="grid min-w-0 grid-cols-1 gap-2 text-sm sm:grid-cols-3 lg:min-w-[520px]">
              <ReportStatus icon={<ClockCounterClockwise className="h-4 w-4" />} label="Periode" value={periodLabel} />
              <ReportStatus icon={<Pulse className="h-4 w-4" />} label="Banding" value={filters.comparisonType?.toUpperCase() ?? "Tidak aktif"} />
              <ReportStatus icon={<FileText className="h-4 w-4" />} label="Tab" value={tabLabel(filters.tab)} />
            </div>
          </div>

          <div className="space-y-4 p-3 sm:p-4 lg:p-5">
            <FilterBar
              periodPreset={filters.periodPreset}
              periodStart={filters.periodStart}
              periodEnd={filters.periodEnd}
              comparisonType={filters.comparisonType}
              areaId={filters.areaId}
              packageId={filters.packageId}
              onPeriodPresetChange={setPeriodPreset}
              onCustomPeriodChange={setCustomPeriod}
              onComparisonChange={setComparisonType}
              onAreaChange={setAreaId}
              onPackageChange={setPackageId}
              onReset={reset}
            />

            <TabNavigation activeTab={filters.tab} onTabChange={setTab} />
          </div>
        </section>

        <div className="space-y-5">
          {filters.tab === "keuangan" && (
            <>
              <RevenueSection filter={reportFilter} />
              <AgingSection filter={reportFilter} />
              <PaymentSection filter={reportFilter} />
              <VoucherSection filter={reportFilter} />
              <ProfitLossSection filter={reportFilter} />
              <RevenueByAreaSection filter={reportFilter} />
            </>
          )}

          {filters.tab === "pelanggan" && (
            <>
              <GrowthSection filter={reportFilter} />
              <DistributionSection filter={reportFilter} />
              <ChurnSection filter={reportFilter} />
            </>
          )}

          {filters.tab === "jaringan" && (
            <>
              <UptimeSection filter={reportFilter} />
              <TrafficSection filter={reportFilter} />
              <SignalSection filter={reportFilter} />
              <CapacitySection />
            </>
          )}

          {filters.tab === "operasional" && (
            <>
              <ActivitySection filter={reportFilter} />
              <NotificationSection filter={reportFilter} />
              <SyncSection filter={reportFilter} />
            </>
          )}
        </div>
      </div>
    </AppShell>
  );
}

function ReportStatus({ icon, label, value }: { icon: React.ReactNode; label: string; value: string }) {
  return (
    <div className="min-w-0 rounded-lg border border-slate-200 bg-slate-50 px-3 py-2.5">
      <div className="flex items-center gap-2 text-xs font-medium text-slate-500">
        {icon}
        <span>{label}</span>
      </div>
      <p className="mt-1 truncate text-sm font-semibold text-slate-900">{value}</p>
    </div>
  );
}

function tabLabel(tab: string) {
  const labels: Record<string, string> = {
    keuangan: "Keuangan",
    pelanggan: "Pelanggan",
    jaringan: "Jaringan",
    operasional: "Operasional",
  };
  return labels[tab] ?? tab;
}

function formatReportPeriod(preset: string, start: string, end: string) {
  const labels: Record<string, string> = {
    hari_ini: "Hari ini",
    minggu_ini: "Minggu ini",
    bulan_ini: "Bulan ini",
    kuartal: "Kuartal ini",
    tahun: "Tahun ini",
  };
  if (preset !== "custom") return labels[preset] ?? "Bulan ini";
  if (!start || !end) return "Kustom";
  return `${start} sampai ${end}`;
}
