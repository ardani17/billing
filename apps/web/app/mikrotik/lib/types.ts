export type RouterRecord = {
  id: string;
  name: string;
  host: string;
  port: number;
  username: string;
  use_ssl: boolean;
  service_types: string[];
  router_os_version?: string;
  board_name?: string;
  cpu_count?: number;
  total_ram_mb?: number;
  identity?: string;
  status: string;
  health_check_interval_sec: number;
  last_online_at?: string;
  last_checked_at?: string;
  last_uptime_sec?: number;
  failure_count: number;
  notes?: string;
};

export type RouterEditForm = {
  name: string;
  host: string;
  port: string;
  username: string;
  password: string;
  useSsl: boolean;
  status: string;
  healthCheckIntervalSec: string;
  notes: string;
};

export type SystemResource = {
  version: string;
  board_name: string;
  cpu_count: number;
  cpu_load: number;
  total_ram: number;
  free_ram: number;
  uptime: number;
  architecture: string;
  identity: string;
};

export type PPPoEUser = {
  id: string;
  username: string;
  profile_name: string;
  remote_address?: string;
  disabled: boolean;
  status: string;
  sync_status: string;
  last_sync_at?: string;
};

export type PPPoESession = {
  id: string;
  username: string;
  caller_id: string;
  address: string;
  uptime: string;
  bytes_in: number;
  bytes_out: number;
  service: string;
};

export type SyncStatus = {
  synced_count: number;
  orphan_count: number;
  missing_count: number;
  out_of_sync_count: number;
  last_sync_at?: string;
};

export type MikrotikDetailSection = "overview" | "pppoe" | "sessions" | "sync";
