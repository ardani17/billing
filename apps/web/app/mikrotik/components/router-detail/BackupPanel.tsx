"use client";

import { useEffect, useState } from "react";
import { ArrowClockwise, DownloadSimple, FloppyDisk, Trash } from "@phosphor-icons/react";
import { DataTable, EmptyState, Section, StatusBadge } from "../../../components/ui";
import { extractMessage, formatBytes } from "../../lib/format";
import type { RouterBackup, RouterBackupList, RouterFirmwareInfo } from "../../lib/types";

export function BackupPanel({ routerId, onError }: { routerId: string; onError: (message: string) => void }) {
  const [backups, setBackups] = useState<RouterBackup[]>([]);
  const [firmware, setFirmware] = useState<RouterFirmwareInfo | null>(null);
  const [loading, setLoading] = useState(false);
  const [creating, setCreating] = useState(false);
  const [deleting, setDeleting] = useState<string | null>(null);
  const [message, setMessage] = useState("");

  async function load() {
    setLoading(true);
    onError("");
    try {
      const [backupResponse, firmwareResponse] = await Promise.all([
        fetch(`/api/network/mikrotik/routers/${routerId}/backups?page_size=20`, { cache: "no-store" }),
        fetch(`/api/network/mikrotik/routers/${routerId}/firmware`, { cache: "no-store" }),
      ]);
      const backupJson = await backupResponse.json();
      const firmwareJson = await firmwareResponse.json();
      if (!backupResponse.ok || !backupJson.success) throw new Error(backupJson.error?.message || "Gagal mengambil backup");
      if (!firmwareResponse.ok || !firmwareJson.success) throw new Error(firmwareJson.error?.message || "Gagal membaca firmware");
      setBackups(((backupJson.data || {}) as RouterBackupList).data || []);
      setFirmware(firmwareJson.data as RouterFirmwareInfo);
    } catch (error) {
      onError(extractMessage(error));
    } finally {
      setLoading(false);
    }
  }

  async function createBackup() {
    setCreating(true);
    setMessage("");
    onError("");
    try {
      const response = await fetch(`/api/network/mikrotik/routers/${routerId}/backups`, { method: "POST" });
      const json = await response.json();
      if (!response.ok || !json.success) throw new Error(json.error?.message || "Gagal membuat backup");
      setMessage("Backup export berhasil dibuat.");
      await load();
    } catch (error) {
      onError(extractMessage(error));
    } finally {
      setCreating(false);
    }
  }

  async function deleteBackup(item: RouterBackup) {
    const ok = window.confirm(`Hapus backup ${item.file_name}? File export akan dihapus dari database aplikasi.`);
    if (!ok) return;
    setDeleting(item.id);
    onError("");
    try {
      const response = await fetch(`/api/network/mikrotik/routers/${routerId}/backups/${item.id}`, { method: "DELETE" });
      const json = await response.json();
      if (!response.ok || !json.success) throw new Error(json.error?.message || "Gagal menghapus backup");
      await load();
    } catch (error) {
      onError(extractMessage(error));
    } finally {
      setDeleting(null);
    }
  }

  useEffect(() => {
    void load();
  }, [routerId]);

  return (
    <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_minmax(22rem,0.55fr)]">
      <Section
        title="Backup export"
        description="Export konfigurasi RouterOS dibuat manual/on-demand dan disimpan sebagai .rsc."
        action={
          <div className="flex flex-wrap gap-2">
            <button type="button" onClick={() => void load()} disabled={loading} className="inline-flex items-center gap-2 rounded-md border border-slate-300 bg-white px-3 py-2 text-sm font-semibold text-slate-700 hover:bg-slate-50 disabled:cursor-wait disabled:opacity-60">
              <ArrowClockwise size={16} />
              {loading ? "Memuat..." : "Refresh"}
            </button>
            <button type="button" onClick={() => void createBackup()} disabled={creating} className="inline-flex items-center gap-2 rounded-md bg-blue-600 px-3 py-2 text-sm font-semibold text-white hover:bg-blue-700 disabled:cursor-wait disabled:opacity-60">
              <FloppyDisk size={16} />
              {creating ? "Membuat..." : "Buat backup"}
            </button>
          </div>
        }
      >
        {message && <p className="mb-4 rounded-lg border border-emerald-200 bg-emerald-50 px-4 py-3 text-sm font-medium text-emerald-800">{message}</p>}
        {backups.length === 0 ? (
          <EmptyState title="Belum ada backup" description="Klik Buat backup untuk mengambil export konfigurasi dari router." />
        ) : (
          <DataTable
            columns={["File", "Ukuran", "Checksum", "Dibuat", "Aksi"]}
            rows={backups.map((item) => [
              item.file_name,
              formatBytes(item.size_bytes),
              <span key={`${item.id}-sum`} className="font-mono text-xs">{item.checksum?.slice(0, 12) || "-"}</span>,
              formatDate(item.created_at),
              <div key={`${item.id}-actions`} className="flex flex-wrap gap-2">
                <a href={`/api/network/mikrotik/routers/${routerId}/backups/${item.id}/download`} className="inline-flex items-center gap-1 rounded-md border border-slate-300 px-2.5 py-1.5 text-xs font-semibold text-slate-700 hover:bg-slate-50">
                  <DownloadSimple size={14} />
                  Download
                </a>
                <button type="button" onClick={() => void deleteBackup(item)} disabled={deleting === item.id} className="inline-flex items-center gap-1 rounded-md border border-red-200 px-2.5 py-1.5 text-xs font-semibold text-red-600 hover:bg-red-50 disabled:cursor-wait disabled:opacity-60">
                  <Trash size={14} />
                  Hapus
                </button>
              </div>,
            ])}
          />
        )}
      </Section>

      <Section title="Firmware dan package" description="Snapshot package RouterOS dibaca langsung saat halaman dibuka.">
        {!firmware ? (
          <EmptyState title="Memuat firmware" description="Membaca resource, routerboard, dan package dari RouterOS." />
        ) : (
          <div className="space-y-5">
            <div className="grid gap-3">
              {[
                ["RouterOS", firmware.routeros_version || "-"],
                ["Architecture", firmware.architecture || "-"],
                ["Board", firmware.board_name || "-"],
                ["Current firmware", firmware.current_firmware || "-"],
                ["Upgrade firmware", firmware.upgrade_firmware || "-"],
              ].map(([label, value]) => (
                <div key={label} className="flex min-w-0 justify-between gap-3 rounded-lg border border-slate-200 bg-slate-50 px-3 py-2 text-sm">
                  <span className="text-slate-500">{label}</span>
                  <span className="min-w-0 text-right font-mono font-semibold text-slate-900 [overflow-wrap:anywhere]">{value}</span>
                </div>
              ))}
              <StatusBadge status={firmware.outdated ? "warning" : "normal"} />
              {firmware.warning && <p className="text-sm leading-6 text-amber-700">{firmware.warning}</p>}
            </div>
            {firmware.packages.length === 0 ? (
              <EmptyState title="Package tidak terbaca" description="Router tidak mengembalikan daftar package." />
            ) : (
              <DataTable
                columns={["Package", "Version", "Status"]}
                rows={firmware.packages.map((pkg) => [
                  pkg.name,
                  pkg.version || "-",
                  <StatusBadge key={pkg.name} status={pkg.disabled ? "disabled" : "aktif"} />,
                ])}
              />
            )}
          </div>
        )}
      </Section>
    </div>
  );
}

function formatDate(value?: string) {
  if (!value) return "-";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString("id-ID", { dateStyle: "medium", timeStyle: "short" });
}
