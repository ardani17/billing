"use client";

import { useEffect, useState } from "react";
import { ArrowClockwise } from "@phosphor-icons/react";
import { DataTable, EmptyState, Section, StatusBadge } from "../../../components/ui";
import { extractMessage } from "../../lib/format";
import type { RouterIPPoolUsage } from "../../lib/types";

export function IPPoolsPanel({ routerId, onError }: { routerId: string; onError: (message: string) => void }) {
  const [items, setItems] = useState<RouterIPPoolUsage[]>([]);
  const [loading, setLoading] = useState(false);
  const [loaded, setLoaded] = useState(false);

  async function load() {
    setLoading(true);
    onError("");
    try {
      const response = await fetch(`/api/network/mikrotik/routers/${routerId}/ip-pools`, { cache: "no-store" });
      const json = await response.json();
      if (!response.ok || !json.success) throw new Error(json.error?.message || "Gagal mengambil IP pool");
      setItems(json.data || []);
      setLoaded(true);
    } catch (error) {
      onError(extractMessage(error));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void load();
  }, [routerId]);

  const action = (
    <button type="button" onClick={() => void load()} disabled={loading} className="inline-flex items-center gap-2 rounded-md border border-slate-300 bg-white px-4 py-2 text-sm font-semibold text-slate-700 hover:bg-slate-50 disabled:cursor-wait disabled:opacity-60">
      <ArrowClockwise size={16} />
      {loading ? "Memuat..." : "Refresh"}
    </button>
  );

  return (
    <Section title="IP pool" description="Kapasitas pool PPPoE dan pool isolir berdasarkan data RouterOS." action={loaded ? action : undefined}>
      {!loaded ? (
        <EmptyState title="Membaca IP pool" description="Data pool dan alamat terpakai diambil live dari RouterOS." action={action} />
      ) : items.length === 0 ? (
        <EmptyState title="Belum ada IP pool" description="Router belum mengembalikan konfigurasi IP pool." action={action} />
      ) : (
        <DataTable
          columns={["Pool", "Range", "Terpakai", "Sisa", "Usage", "Status"]}
          rows={items.map((item) => [
            item.name,
            item.ranges.join(", ") || "-",
            `${item.used}/${item.total || "-"}`,
            String(item.available),
            `${item.usage_percent}%`,
            <StatusBadge key={`${item.name}-status`} status={item.warning_level} />,
          ])}
        />
      )}
    </Section>
  );
}
