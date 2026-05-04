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

export type RouterInterface = {
  id: string;
  name: string;
  type: string;
  mtu: number;
  mac_address?: string;
  running: boolean;
  disabled: boolean;
  rx_byte: number;
  tx_byte: number;
  rx_packet: number;
  tx_packet: number;
  comment?: string;
};

export type RouterTrafficSample = {
  interface: string;
  rx_bps: number;
  tx_bps: number;
  rx_packets_per_second: number;
  tx_packets_per_second: number;
};

export type RouterIPPoolUsage = {
  name: string;
  ranges: string[];
  used: number;
  total: number;
  available: number;
  usage_percent: number;
  warning_level: string;
};

export type RouterFirewallRule = {
  id: string;
  kind: string;
  chain?: string;
  action?: string;
  list?: string;
  address?: string;
  disabled: boolean;
  comment?: string;
};

export type RouterLogEntry = {
  id: string;
  time: string;
  topics: string;
  message: string;
};

export type DHCPServer = {
  id: string;
  name: string;
  interface: string;
  address_pool: string;
  lease_time: string;
  authoritative?: string;
  disabled: boolean;
  comment?: string;
};

export type DHCPLease = {
  id: string;
  server?: string;
  address?: string;
  mac_address: string;
  host_name?: string;
  client_id?: string;
  status?: string;
  dynamic: boolean;
  disabled: boolean;
  expires_after?: string;
  last_seen?: string;
  comment?: string;
  managed: boolean;
};

export type DHCPNetwork = {
  id: string;
  address: string;
  gateway?: string;
  dns_server?: string[];
  domain?: string;
  comment?: string;
};

export type DHCPBinding = {
  id: string;
  router_id: string;
  customer_id?: string;
  router_lease_id?: string;
  server: string;
  mac_address: string;
  ip_address: string;
  host_name?: string;
  comment: string;
  disabled: boolean;
  status: string;
  last_sync_at?: string;
  sync_status: string;
  created_at: string;
  updated_at: string;
};

export type StaticIPAssignment = {
  id: string;
  router_id: string;
  customer_id?: string;
  ip_address: string;
  address_list: string;
  queue_name?: string;
  rate_limit?: string;
  comment: string;
  status: string;
  last_sync_at?: string;
  sync_status: string;
  created_at: string;
  updated_at: string;
};

export type MikrotikDetailSection =
  | "overview"
  | "pppoe"
  | "sessions"
  | "sync"
  | "traffic"
  | "interfaces"
  | "ip-pool"
  | "firewall"
  | "logs"
  | "dhcp"
  | "static-ip";
