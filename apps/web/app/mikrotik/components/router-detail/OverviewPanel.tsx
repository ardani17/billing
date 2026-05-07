"use client";

import { DataTable, EmptyState, Section } from "../../../components/ui";
import { formatUptime } from "../../lib/format";
import type { RouterRecord, SystemResource } from "../../lib/types";

function formatResourceBytes(bytes?: number) {
  if (bytes == null || bytes <= 0) return "-";
  const units = ["B", "KiB", "MiB", "GiB", "TiB"];
  let value = bytes;
  let unit = 0;
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024;
    unit += 1;
  }
  const formatted = unit === 0 ? String(value) : value.toFixed(1);
  return `${formatted} ${units[unit]}`;
}

function formatNumber(value?: number) {
  if (value == null || value <= 0) return "-";
  return new Intl.NumberFormat("id-ID").format(value);
}

export function OverviewPanel({
  router,
  system,
  loadingSystem = false,
}: {
  router: RouterRecord;
  system: SystemResource | null;
  loadingSystem?: boolean;
}) {
  return (
    <div className="grid gap-6 xl:grid-cols-2">
      <Section title="Konfigurasi router">
        <DataTable
          columns={["Field", "Value"]}
          rows={[
            ["ID", router.id],
            ["Host", router.host],
            ["Port", String(router.port)],
            ["Username", router.username],
            ["Use SSL", router.use_ssl ? "Ya" : "Tidak"],
            ["Service", router.service_types.join(", ")],
          ]}
        />
      </Section>

      <Section title="System resource live">
        {system ? (
          <DataTable
            columns={["Metric", "Value"]}
            rows={[
              ["Uptime", formatUptime(system.uptime)],
              ["Free Memory", formatResourceBytes(system.free_ram)],
              ["Total Memory", formatResourceBytes(system.total_ram)],
              ["CPU", system.cpu || "-"],
              ["CPU Count", String(system.cpu_count || "-")],
              ["CPU Frequency", system.cpu_frequency_mhz ? `${system.cpu_frequency_mhz} MHz` : "-"],
              ["CPU Load", `${system.cpu_load}%`],
              ["Free HDD Space", formatResourceBytes(system.free_hdd_space)],
              ["Total HDD Size", formatResourceBytes(system.total_hdd_space)],
              ["Sector Writes Since Reboot", formatNumber(system.write_sect_since_reboot)],
              ["Total Sector Writes", formatNumber(system.write_sect_total)],
              ["Architecture Name", system.architecture || "-"],
              ["Board Name", system.board_name || "-"],
              ["Version", system.version || "-"],
              ["Build Time", system.build_time || "-"],
              ["Identity", system.identity || "-"],
            ]}
          />
        ) : loadingSystem ? (
          <EmptyState title="Membaca resource live" description="Mengambil snapshot dari RouterOS API..." />
        ) : (
          <EmptyState title="Belum ada snapshot live" description="Klik Test Connection untuk membaca resource RouterOS." />
        )}
      </Section>
    </div>
  );
}
