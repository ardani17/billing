"use client";

import { useEffect, useState } from "react";
import { ArrowClockwise } from "@phosphor-icons/react";
import { DataTable, EmptyState, Section, StatusBadge } from "../../../components/ui";
import { extractMessage, formatBytes } from "../../lib/format";
import type { RouterInterface } from "../../lib/types";

export function InterfacesPanel({ routerId, onError }: { routerId: string; onError: (message: string) => void }) {
  const [items, setItems] = useState<RouterInterface[]>([]);
  const [loading, setLoading] = useState(false);
  const [loaded, setLoaded] = useState(false);

  async function load() {
    setLoading(true);
    onError("");
    try {
      const response = await fetch(`/api/network/mikrotik/routers/${routerId}/interfaces`, { cache: "no-store" });
      const json = await response.json();
      if (!response.ok || !json.success) throw new Error(json.error?.message || "Gagal mengambil interface");
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
    <Section title="Interface RouterOS" description="Snapshot port, bridge, dan interface virtual dari router." action={loaded ? action : undefined}>
      {!loaded ? (
        <EmptyState title="Membaca interface" description="Data diambil live dari RouterOS API." action={action} />
      ) : items.length === 0 ? (
        <EmptyState title="Belum ada interface" description="Router tidak mengembalikan data interface." action={action} />
      ) : (
        <DataTable
          columns={["Nama", "Tipe", "Status", "MTU", "RX", "TX", "MAC"]}
          rows={items.map((item) => [
            item.name,
            item.type || "-",
            <StatusBadge key={`${item.id}-status`} status={item.disabled ? "disabled" : item.running ? "online" : "offline"} />,
            String(item.mtu || "-"),
            formatBytes(item.rx_byte),
            formatBytes(item.tx_byte),
            item.mac_address || "-",
          ])}
        />
      )}
    </Section>
  );
}
