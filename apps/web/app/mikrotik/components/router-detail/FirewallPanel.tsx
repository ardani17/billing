"use client";

import { useEffect, useState } from "react";
import { ArrowClockwise } from "@phosphor-icons/react";
import { DataTable, EmptyState, Section, StatusBadge } from "../../../components/ui";
import { extractMessage } from "../../lib/format";
import type { RouterFirewallRule } from "../../lib/types";

export function FirewallPanel({ routerId, onError }: { routerId: string; onError: (message: string) => void }) {
  const [rules, setRules] = useState<RouterFirewallRule[]>([]);
  const [loading, setLoading] = useState(false);
  const [loaded, setLoaded] = useState(false);

  async function load() {
    setLoading(true);
    onError("");
    try {
      const response = await fetch(`/api/network/mikrotik/routers/${routerId}/firewall/managed`, { cache: "no-store" });
      const json = await response.json();
      if (!response.ok || !json.success) throw new Error(json.error?.message || "Gagal mengambil firewall");
      setRules(json.data || []);
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
    <Section title="Firewall terkelola" description="Menampilkan rule dan address-list yang dibuat atau dipakai ISPBoss." action={loaded ? action : undefined}>
      {!loaded ? (
        <EmptyState title="Membaca firewall" description="Data dibatasi ke rule terkelola supaya konfigurasi router lain tidak bercampur." action={action} />
      ) : rules.length === 0 ? (
        <EmptyState title="Belum ada rule terkelola" description="Tidak ada rule/comment ISPBoss atau address-list isolir/walled garden yang ditemukan." action={action} />
      ) : (
        <DataTable
          columns={["Jenis", "Chain/List", "Action/Address", "Status", "Comment"]}
          rows={rules.map((rule) => [
            rule.kind,
            rule.chain || rule.list || "-",
            rule.action || rule.address || "-",
            <StatusBadge key={`${rule.id}-status`} status={rule.disabled ? "disabled" : "aktif"} />,
            rule.comment || "-",
          ])}
        />
      )}
    </Section>
  );
}
