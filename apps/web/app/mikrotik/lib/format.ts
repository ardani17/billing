import type { RouterEditForm, RouterRecord } from "./types";

export function formatUptime(seconds?: number) {
  if (!seconds) return "-";
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  if (days > 0) return `${days}d ${hours}h`;
  if (hours > 0) return `${hours}h ${minutes}m`;
  return `${minutes}m`;
}

export function formatMemory(bytes?: number) {
  if (!bytes) return "-";
  return `${Math.round(bytes / 1024 / 1024)} MB`;
}

export function extractMessage(error: unknown) {
  return error instanceof Error ? error.message : "Terjadi kesalahan";
}

export function routerToEditForm(router: RouterRecord): RouterEditForm {
  return {
    name: router.name,
    host: router.host,
    port: String(router.port || 8728),
    username: router.username,
    password: "",
    useSsl: router.use_ssl,
    status: router.status,
    healthCheckIntervalSec: String(router.health_check_interval_sec || 300),
    notes: router.notes || "",
  };
}
