"use client";

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

  return (
    <div className="mx-auto max-w-7xl space-y-6 px-4 py-6 md:px-6">
      <div>
        <h1 className="text-2xl font-bold text-slate-900">Laporan & Analitik</h1>
        <p className="mt-1 text-sm text-slate-500">
          Pantau performa bisnis dan operasional ISP Anda
        </p>
      </div>

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

      <div className="space-y-6">
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
  );
}
