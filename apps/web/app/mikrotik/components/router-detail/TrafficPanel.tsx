"use client";

import { useEffect, useState } from "react";
import { ArrowClockwise } from "@phosphor-icons/react";
import { DataTable, EmptyState, Section } from "../../../components/ui";
import { extractMessage, formatBps } from "../../lib/format";
import type { RouterTrafficSample } from "../../lib/types";

export function TrafficPanel({ routerId, onError }: { routerId: string; onError: (message: string) => void }) {
  const [samples, setSamples] = useState<RouterTrafficSample[]>([]);
  const [loading, setLoading] = useState(false);
  const [loaded, setLoaded] = useState(false);

  async function load() {
    setLoading(true);
    onError("");
    try {
      const response = await fetch(`/api/network/mikrotik/routers/${routerId}/traffic`, { cache: "no-store" });
      const json = await response.json();
      if (!response.ok || !json.success) throw new Error(json.error?.message || "Gagal mengambil traffic");
      setSamples(json.data || []);
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
      {loading ? "Sampling..." : "Ambil Sample"}
    </button>
  );

  return (
    <Section title="Traffic interface" description="Sample sekali jalan dari /interface/monitor-traffic, tanpa polling otomatis." action={loaded ? action : undefined}>
      {!loaded ? (
        <EmptyState title="Traffic belum disampling" description="Klik untuk membaca traffic live saat ini." action={action} />
      ) : samples.length === 0 ? (
        <EmptyState title="Traffic kosong" description="Tidak ada interface aktif yang mengembalikan sample traffic." action={action} />
      ) : (
        <DataTable
          columns={["Interface", "RX", "TX", "RX packet/s", "TX packet/s"]}
          rows={samples.map((sample) => [
            sample.interface,
            formatBps(sample.rx_bps),
            formatBps(sample.tx_bps),
            String(sample.rx_packets_per_second || 0),
            String(sample.tx_packets_per_second || 0),
          ])}
        />
      )}
    </Section>
  );
}
