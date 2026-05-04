"use client";

import { ArrowClockwise } from "@phosphor-icons/react";
import { DataTable, EmptyState, Section } from "../../../components/ui";
import { formatMemory } from "../../lib/format";
import type { PPPoESession } from "../../lib/types";

export function SessionsPanel({
  sessions,
  loaded,
  loading,
  actionBusy,
  onLoad,
  onDisconnect,
}: {
  sessions: PPPoESession[];
  loaded: boolean;
  loading: boolean;
  actionBusy: string | null;
  onLoad: () => void;
  onDisconnect: (sessionId: string) => void;
}) {
  const refreshButton = (
    <button
      type="button"
      onClick={onLoad}
      disabled={loading}
      className="inline-flex items-center justify-center gap-2 rounded-md border border-slate-300 bg-white px-4 py-2 text-sm font-semibold text-slate-700 hover:bg-slate-50 disabled:cursor-wait disabled:opacity-60"
    >
      <ArrowClockwise size={16} />
      {loading ? "Memuat..." : loaded ? "Refresh Live" : "Muat Session Live"}
    </button>
  );

  return (
    <Section title="PPPoE sessions live" description="Dibaca dari RouterOS API hanya saat diminta." action={loaded ? refreshButton : undefined}>
      {!loaded ? (
        <EmptyState
          title="Session live belum dimuat"
          description="Gunakan aksi manual saat perlu melihat pelanggan yang sedang tersambung."
          action={refreshButton}
        />
      ) : sessions.length === 0 ? (
        <EmptyState
          title="Belum ada session aktif"
          description="Router online, tetapi belum ada pelanggan PPPoE yang sedang tersambung."
          action={refreshButton}
        />
      ) : (
        <DataTable
          columns={["User", "IP", "Caller ID", "Uptime", "Traffic", "Aksi"]}
          rows={sessions.map((session) => [
            session.username,
            session.address || "-",
            session.caller_id || "-",
            session.uptime || "-",
            formatMemory(session.bytes_in + session.bytes_out),
            <button
              key={`${session.id}-disconnect`}
              type="button"
              disabled={actionBusy === `session:${session.id}`}
              onClick={() => onDisconnect(session.id)}
              className="rounded-md px-3 py-2 text-sm font-semibold text-red-600 hover:bg-red-50 disabled:cursor-wait disabled:opacity-60"
            >
              {actionBusy === `session:${session.id}` ? "Memutus..." : "Disconnect"}
            </button>,
          ])}
        />
      )}
    </Section>
  );
}
