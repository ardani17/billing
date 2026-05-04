"use client";

import { useEffect, useMemo, useState, type FormEvent } from "react";
import { ArrowClockwise, Play, ShieldWarning } from "@phosphor-icons/react";
import { DataTable, EmptyState, Section, StatusBadge } from "../../../components/ui";
import { extractMessage } from "../../lib/format";
import type { MikroTikCommandAuditList, MikroTikCommandAuditLog, TerminalExecuteResult } from "../../lib/types";

const commandPresets = [
  { label: "System resource", command: "/system/resource/print", params: "{}" },
  { label: "Identity", command: "/system/identity/print", params: "{}" },
  { label: "Interface", command: "/interface/print", params: "{\"=.proplist\":\".id,name,type,running,disabled,rx-byte,tx-byte\"}" },
  { label: "Traffic once", command: "/interface/monitor-traffic", params: "{\"=interface\":\"ether1\",\"=once\":\"\"}" },
  { label: "PPP users", command: "/ppp/secret/print", params: "{\"=.proplist\":\".id,name,profile,service,disabled,comment\"}" },
  { label: "PPP active", command: "/ppp/active/print", params: "{}" },
  { label: "Hotspot users", command: "/ip/hotspot/user/print", params: "{\"=.proplist\":\".id,name,profile,limit-uptime,uptime,disabled,comment\"}" },
  { label: "Firewall NAT", command: "/ip/firewall/nat/print", params: "{\"=.proplist\":\".id,chain,action,disabled,comment\"}" },
  { label: "Log", command: "/log/print", params: "{\"=.proplist\":\".id,time,topics,message\"}" },
];

const initialPreset = commandPresets[0] ?? { command: "/system/resource/print", params: "{}" };

export function TerminalPanel({ routerId, onError }: { routerId: string; onError: (message: string) => void }) {
  const [command, setCommand] = useState(initialPreset.command);
  const [paramsText, setParamsText] = useState(initialPreset.params);
  const [result, setResult] = useState<TerminalExecuteResult | null>(null);
  const [audit, setAudit] = useState<MikroTikCommandAuditLog[]>([]);
  const [running, setRunning] = useState(false);
  const [loadingAudit, setLoadingAudit] = useState(false);
  const [message, setMessage] = useState("");

  const columns = useMemo(() => {
    if (!result?.rows?.length) return [];
    return Array.from(new Set(result.rows.flatMap((row) => Object.keys(row)))).slice(0, 8);
  }, [result]);

  async function loadAudit() {
    setLoadingAudit(true);
    onError("");
    try {
      const response = await fetch(`/api/network/mikrotik/routers/${routerId}/terminal/audit?page_size=20`, { cache: "no-store" });
      const json = await response.json();
      if (!response.ok || !json.success) throw new Error(json.error?.message || "Gagal mengambil audit terminal");
      const data = json.data as MikroTikCommandAuditList;
      setAudit(data.data || []);
    } catch (error) {
      onError(extractMessage(error));
    } finally {
      setLoadingAudit(false);
    }
  }

  async function execute(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setRunning(true);
    setMessage("");
    onError("");
    try {
      let params: Record<string, string> = {};
      if (paramsText.trim()) {
        params = JSON.parse(paramsText);
      }
      const response = await fetch(`/api/network/mikrotik/routers/${routerId}/terminal/execute`, {
        method: "POST",
        body: JSON.stringify({ command: command.trim(), params }),
      });
      const json = await response.json();
      if (!response.ok || !json.success) throw new Error(json.error?.message || "Command ditolak atau gagal dijalankan");
      setResult(json.data as TerminalExecuteResult);
      setMessage("Command berhasil dijalankan dan dicatat di audit.");
      await loadAudit();
    } catch (error) {
      onError(extractMessage(error));
      await loadAudit();
    } finally {
      setRunning(false);
    }
  }

  useEffect(() => {
    void loadAudit();
  }, [routerId]);

  return (
    <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_minmax(22rem,0.55fr)]">
      <Section
        title="Terminal read-only"
        description="Command RouterOS dijalankan manual dan hanya allowlist diagnostik yang diterima."
        action={
          <span className="inline-flex items-center gap-2 rounded-full bg-amber-50 px-3 py-1 text-xs font-semibold text-amber-800 ring-1 ring-amber-200">
            <ShieldWarning size={15} />
            Write command ditolak
          </span>
        }
      >
        <form onSubmit={(event) => void execute(event)} className="grid gap-4">
          <div className="grid gap-3 md:grid-cols-[minmax(0,0.9fr)_minmax(0,1.1fr)]">
            <label className="grid gap-2 text-sm font-medium text-slate-800">
              Preset
              <select
                value={command}
                onChange={(event) => {
                  const preset = commandPresets.find((item) => item.command === event.target.value);
                  setCommand(event.target.value);
                  if (preset) setParamsText(preset.params);
                }}
                className="h-10 min-w-0 rounded-md border border-slate-300 bg-white px-3 text-sm outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-100"
              >
                {commandPresets.map((item) => (
                  <option key={item.label} value={item.command}>
                    {item.label}
                  </option>
                ))}
              </select>
            </label>
            <label className="grid gap-2 text-sm font-medium text-slate-800">
              Command
              <input
                value={command}
                onChange={(event) => setCommand(event.target.value)}
                className="h-10 min-w-0 rounded-md border border-slate-300 px-3 font-mono text-sm outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-100"
              />
            </label>
          </div>

          <label className="grid gap-2 text-sm font-medium text-slate-800">
            Params JSON
            <textarea
              value={paramsText}
              onChange={(event) => setParamsText(event.target.value)}
              rows={5}
              spellCheck={false}
              className="min-w-0 rounded-md border border-slate-300 px-3 py-2 font-mono text-sm outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-100"
            />
          </label>

          <div className="flex flex-wrap items-center gap-3">
            <button
              type="submit"
              disabled={running}
              className="inline-flex items-center gap-2 rounded-md bg-blue-600 px-4 py-2 text-sm font-semibold text-white hover:bg-blue-700 disabled:cursor-wait disabled:opacity-60"
            >
              <Play size={16} weight="fill" />
              {running ? "Menjalankan..." : "Run command"}
            </button>
            {message && <span className="text-sm font-medium text-emerald-700">{message}</span>}
          </div>
        </form>

        <div className="mt-6">
          {!result ? (
            <EmptyState title="Belum ada hasil command" description="Pilih preset, cek params, lalu jalankan manual." />
          ) : result.rows.length === 0 ? (
            <EmptyState title="Command berhasil tanpa row" description="RouterOS tidak mengembalikan data untuk command ini." />
          ) : columns.length > 0 ? (
            <DataTable columns={columns} rows={result.rows.map((row) => columns.map((column) => row[column] || "-"))} />
          ) : (
            <pre className="max-h-96 overflow-auto rounded-xl bg-slate-950 p-4 text-xs text-slate-100">
              {JSON.stringify(result.rows, null, 2)}
            </pre>
          )}
        </div>
      </Section>

      <Section
        title="Audit command"
        description="Semua attempt terminal dicatat, termasuk command yang ditolak."
        action={
          <button
            type="button"
            onClick={() => void loadAudit()}
            disabled={loadingAudit}
            className="inline-flex items-center gap-2 rounded-md border border-slate-300 bg-white px-3 py-2 text-sm font-semibold text-slate-700 hover:bg-slate-50 disabled:cursor-wait disabled:opacity-60"
          >
            <ArrowClockwise size={16} />
            {loadingAudit ? "Memuat..." : "Refresh"}
          </button>
        }
      >
        {audit.length === 0 ? (
          <EmptyState title="Belum ada audit" description="Audit akan muncul setelah command dijalankan." />
        ) : (
          <div className="grid gap-3">
            {audit.map((item) => (
              <div key={item.id} className="min-w-0 rounded-lg border border-slate-200 bg-white p-3">
                <div className="flex min-w-0 items-start justify-between gap-3">
                  <div className="min-w-0">
                    <p className="truncate font-mono text-sm font-semibold text-slate-900">{item.command}</p>
                    <p className="mt-1 text-xs text-slate-500">{formatAuditTime(item.created_at)}</p>
                  </div>
                  <StatusBadge status={item.status} />
                </div>
                {item.error_message && (
                  <p className="mt-2 text-xs leading-5 text-red-600 [overflow-wrap:anywhere]">{item.error_message}</p>
                )}
              </div>
            ))}
          </div>
        )}
      </Section>
    </div>
  );
}

function formatAuditTime(value?: string) {
  if (!value) return "-";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString("id-ID", { dateStyle: "medium", timeStyle: "short" });
}
