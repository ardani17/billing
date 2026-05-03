export const platformTenants = [
  {
    id: "tenant-nusafiber",
    name: "NusaFiber Depok",
    owner: "Budi Santoso",
    domain: "billing.nusafiber.id",
    plan: "Growth",
    mrr: "Rp799.000",
    customers: "847",
    modules: "Billing, MikroTik, OLT, FTTH",
    status: "aktif",
    health: "normal",
    lastSeen: "3 menit lalu",
  },
  {
    id: "tenant-meganet",
    name: "MegaNet Cibinong",
    owner: "Rina Maheswari",
    domain: "app.meganet.net",
    plan: "Scale",
    mrr: "Rp1.499.000",
    customers: "1.284",
    modules: "Billing, MikroTik, Notifikasi",
    status: "aktif",
    health: "degraded",
    lastSeen: "11 menit lalu",
  },
  {
    id: "tenant-wiralink",
    name: "WiraLink Timur",
    owner: "Dimas Pradana",
    domain: "wiralink.ispboss.id",
    plan: "Starter",
    mrr: "Rp299.000",
    customers: "214",
    modules: "Billing",
    status: "trial",
    health: "normal",
    lastSeen: "1 jam lalu",
  },
  {
    id: "tenant-baratnet",
    name: "BaratNet Access",
    owner: "Maya Lestari",
    domain: "billing.baratnet.id",
    plan: "Growth",
    mrr: "Rp799.000",
    customers: "692",
    modules: "Billing, MikroTik, Voucher",
    status: "suspended",
    health: "blocked",
    lastSeen: "2 hari lalu",
  },
];

export const platformServiceHealth = [
  { service: "Billing API", region: "Jakarta", latency: "84 ms", uptime: "99.98%", status: "online" },
  { service: "Network Service", region: "Jakarta", latency: "112 ms", uptime: "99.91%", status: "online" },
  { service: "Notification Service", region: "Jakarta", latency: "146 ms", uptime: "99.84%", status: "degraded" },
  { service: "VPN Gateway", region: "Jakarta", latency: "31 ms", uptime: "99.95%", status: "online" },
];

export const platformTickets = [
  { code: "SUP-1042", tenant: "MegaNet Cibinong", subject: "Webhook payment lambat", priority: "tinggi", status: "pending" },
  { code: "SUP-1038", tenant: "NusaFiber Depok", subject: "Butuh cek router offline", priority: "sedang", status: "aktif" },
  { code: "SUP-1031", tenant: "WiraLink Timur", subject: "Bantuan setup domain", priority: "rendah", status: "aktif" },
];

export const platformAudit = [
  { time: "15:42", actor: "Super Admin", action: "Impersonate Tenant Admin", target: "MegaNet Cibinong", status: "success" },
  { time: "14:18", actor: "System", action: "Subscription renewed", target: "NusaFiber Depok", status: "success" },
  { time: "13:50", actor: "Super Admin", action: "Suspend tenant", target: "BaratNet Access", status: "success" },
  { time: "11:27", actor: "System", action: "Domain verification failed", target: "WiraLink Timur", status: "warning" },
];

export const platformInvoices = [
  { number: "PLAT-2026-05-001", tenant: "NusaFiber Depok", amount: "Rp799.000", dueDate: "05 Mei 2026", status: "lunas" },
  { number: "PLAT-2026-05-002", tenant: "MegaNet Cibinong", amount: "Rp1.499.000", dueDate: "05 Mei 2026", status: "belum_bayar" },
  { number: "PLAT-2026-05-003", tenant: "WiraLink Timur", amount: "Rp0", dueDate: "Trial", status: "trial" },
  { number: "PLAT-2026-04-014", tenant: "BaratNet Access", amount: "Rp799.000", dueDate: "20 Apr 2026", status: "terlambat" },
];
