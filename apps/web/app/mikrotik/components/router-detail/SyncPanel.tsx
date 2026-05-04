"use client";

import { DataTable, Section } from "../../../components/ui";
import type { SyncStatus } from "../../lib/types";

export function SyncPanel({
  syncStatus,
  syncing,
  onSync,
}: {
  syncStatus: SyncStatus | null;
  syncing: boolean;
  onSync: () => void;
}) {
  return (
    <Section
      title="Status sinkronisasi PPPoE"
      description="Perbandingan data ISPBoss dengan secret PPPoE di router."
      action={
        <button
          type="button"
          onClick={onSync}
          disabled={syncing}
          className="rounded-md border border-slate-300 bg-white px-4 py-2 text-sm font-semibold text-slate-700 hover:bg-slate-50 disabled:cursor-wait disabled:opacity-60"
        >
          {syncing ? "Syncing..." : "Sync PPPoE"}
        </button>
      }
    >
      <DataTable
        columns={["Metric", "Nilai"]}
        rows={[
          ["Synced", String(syncStatus?.synced_count ?? 0)],
          ["Missing di router", String(syncStatus?.missing_count ?? 0)],
          ["Orphan di router", String(syncStatus?.orphan_count ?? 0)],
          ["Out of sync", String(syncStatus?.out_of_sync_count ?? 0)],
          ["Terakhir sync", syncStatus?.last_sync_at ? new Date(syncStatus.last_sync_at).toLocaleString("id-ID") : "-"],
        ]}
      />
    </Section>
  );
}
