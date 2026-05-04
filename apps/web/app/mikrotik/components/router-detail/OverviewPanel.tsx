"use client";

import { DataTable, EmptyState, Section } from "../../../components/ui";
import { formatMemory, formatUptime } from "../../lib/format";
import type { RouterRecord, SystemResource } from "../../lib/types";

export function OverviewPanel({
  router,
  system,
}: {
  router: RouterRecord;
  system: SystemResource | null;
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
              ["Identity", system.identity],
              ["Version", system.version],
              ["Architecture", system.architecture],
              ["CPU Load", `${system.cpu_load}%`],
              ["Free RAM", formatMemory(system.free_ram)],
              ["Uptime", formatUptime(system.uptime)],
            ]}
          />
        ) : (
          <EmptyState title="Belum ada snapshot live" description="Klik Test Connection untuk membaca resource RouterOS." />
        )}
      </Section>
    </div>
  );
}
