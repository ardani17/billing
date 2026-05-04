"use client";

import { useEffect, useState } from "react";
import { ArrowClockwise } from "@phosphor-icons/react";
import { DataTable, EmptyState, Section } from "../../../components/ui";
import { extractMessage } from "../../lib/format";
import type { RouterLogEntry } from "../../lib/types";

export function LogsPanel({ routerId, onError }: { routerId: string; onError: (message: string) => void }) {
  const [items, setItems] = useState<RouterLogEntry[]>([]);
  const [topic, setTopic] = useState("");
  const [search, setSearch] = useState("ISPBoss");
  const [loading, setLoading] = useState(false);
  const [loaded, setLoaded] = useState(false);

  async function load() {
    setLoading(true);
    onError("");
    try {
      const query = new URLSearchParams({ limit: "100" });
      if (topic.trim()) query.set("topic", topic.trim());
      if (search.trim()) query.set("search", search.trim());
      const response = await fetch(`/api/network/mikrotik/routers/${routerId}/logs?${query.toString()}`, { cache: "no-store" });
      const json = await response.json();
      if (!response.ok || !json.success) throw new Error(json.error?.message || "Gagal mengambil log");
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
    <Section title="Log RouterOS" description="Audit operasional router dengan filter topic/search manual." action={loaded ? action : undefined}>
      <div className="mb-4 grid gap-3 md:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_auto]">
        <input value={topic} onChange={(event) => setTopic(event.target.value)} placeholder="Topic, mis. pppoe" className="h-10 min-w-0 rounded-md border border-slate-300 px-3 text-sm outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-100" />
        <input value={search} onChange={(event) => setSearch(event.target.value)} placeholder="Search log" className="h-10 min-w-0 rounded-md border border-slate-300 px-3 text-sm outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-100" />
        {action}
      </div>

      {!loaded ? (
        <EmptyState title="Membaca log" description="Log diambil saat halaman atau refresh manual dibuka." />
      ) : items.length === 0 ? (
        <EmptyState title="Log tidak ditemukan" description="Coba ubah filter topic atau search." />
      ) : (
        <DataTable
          columns={["Waktu", "Topics", "Message"]}
          rows={items.map((item) => [item.time || "-", item.topics || "-", item.message || "-"])}
        />
      )}
    </Section>
  );
}
