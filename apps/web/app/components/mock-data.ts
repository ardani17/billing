export const customers = [
  {
    id: "PLG-001",
    name: "Ahmad Rizki",
    phone: "+62 812-7841-2096",
    package: "Pro 50M",
    area: "Depok Timur",
    dueDate: "05 Mei 2026",
    status: "aktif",
    balance: "Rp0",
  },
  {
    id: "PLG-014",
    name: "Rina Maheswari",
    phone: "+62 813-5629-8840",
    package: "Basic 20M",
    area: "Cibinong",
    dueDate: "03 Mei 2026",
    status: "isolir",
    balance: "Rp388.500",
  },
  {
    id: "PLG-027",
    name: "Surya Kencana",
    phone: "+62 857-4308-2219",
    package: "Ultra 100M",
    area: "Bojonggede",
    dueDate: "12 Mei 2026",
    status: "pending",
    balance: "Rp0",
  },
  {
    id: "PLG-063",
    name: "Maya Lestari",
    phone: "+62 811-2930-4751",
    package: "Office 75M",
    area: "Sawangan",
    dueDate: "01 Mei 2026",
    status: "suspend",
    balance: "Rp742.000",
  },
];

export const packages = [
  {
    name: "Basic 20M",
    type: "PPPoE",
    price: "Rp185.000",
    speed: "20/8 Mbps",
    customers: "184",
    status: "aktif",
  },
  {
    name: "Pro 50M",
    type: "PPPoE",
    price: "Rp350.000",
    speed: "50/20 Mbps",
    customers: "431",
    status: "aktif",
  },
  {
    name: "Ultra 100M",
    type: "PPPoE",
    price: "Rp620.000",
    speed: "100/40 Mbps",
    customers: "76",
    status: "aktif",
  },
  {
    name: "Voucher 1 Hari 5M",
    type: "Hotspot",
    price: "Rp3.000",
    speed: "5/2 Mbps",
    customers: "2.184 voucher",
    status: "aktif",
  },
];

export const invoices = [
  {
    number: "INV-2026-05-001",
    customer: "Ahmad Rizki",
    period: "Mei 2026",
    amount: "Rp350.000",
    dueDate: "05 Mei 2026",
    status: "belum_bayar",
  },
  {
    number: "INV-2026-05-014",
    customer: "Rina Maheswari",
    period: "Mei 2026",
    amount: "Rp388.500",
    dueDate: "03 Mei 2026",
    status: "terlambat",
  },
  {
    number: "INV-2026-04-287",
    customer: "Maya Lestari",
    period: "April 2026",
    amount: "Rp742.000",
    dueDate: "01 Mei 2026",
    status: "bayar_sebagian",
  },
  {
    number: "INV-2026-05-032",
    customer: "Surya Kencana",
    period: "Mei 2026",
    amount: "Rp620.000",
    dueDate: "12 Mei 2026",
    status: "lunas",
  },
];

export const payments = [
  {
    receipt: "RCPT-2026-05-118",
    customer: "Surya Kencana",
    amount: "Rp620.000",
    method: "QRIS",
    date: "03 Mei 2026 14:32",
    status: "lunas",
  },
  {
    receipt: "RCPT-2026-05-112",
    customer: "Bima Prasetya",
    amount: "Rp185.000",
    method: "Transfer BCA",
    date: "03 Mei 2026 11:08",
    status: "lunas",
  },
  {
    receipt: "RCPT-2026-05-101",
    customer: "Maya Lestari",
    amount: "Rp280.000",
    method: "Tunai",
    date: "02 Mei 2026 17:41",
    status: "bayar_sebagian",
  },
];

export const routers = [
  {
    name: "MK-Depok-01",
    ip: "10.10.8.1",
    version: "RouterOS 7.15",
    uptime: "18h 42m",
    active: "318",
    cpu: "27%",
    status: "online",
  },
  {
    name: "MK-Cibinong-02",
    ip: "10.10.12.1",
    version: "RouterOS 6.49",
    uptime: "7d 3h",
    active: "214",
    cpu: "41%",
    status: "online",
  },
  {
    name: "MK-Sawangan-03",
    ip: "10.10.18.1",
    version: "RouterOS 7.14",
    uptime: "0m",
    active: "0",
    cpu: "-",
    status: "offline",
  },
];

export const olts = [
  {
    name: "OLT-01 Pusat",
    brand: "ZTE C320",
    ip: "10.20.1.2",
    pon: "16 port",
    ont: "624",
    alarms: "3",
    status: "online",
  },
  {
    name: "OLT-02 Barat",
    brand: "Huawei MA5608T",
    ip: "10.20.4.2",
    pon: "8 port",
    ont: "291",
    alarms: "0",
    status: "online",
  },
  {
    name: "OLT-03 Timur",
    brand: "FiberHome AN5516",
    ip: "10.20.8.2",
    pon: "8 port",
    ont: "184",
    alarms: "9",
    status: "degraded",
  },
];

export const notifications = [
  {
    template: "Invoice Baru",
    channel: "WhatsApp",
    sent: "842",
    failed: "11",
    lastRun: "03 Mei 2026 09:15",
    status: "aktif",
  },
  {
    template: "Reminder H+3",
    channel: "WhatsApp, SMS",
    sent: "126",
    failed: "4",
    lastRun: "03 Mei 2026 10:05",
    status: "aktif",
  },
  {
    template: "Router Offline",
    channel: "WhatsApp",
    sent: "7",
    failed: "0",
    lastRun: "03 Mei 2026 13:29",
    status: "aktif",
  },
];

export const vouchers = [
  {
    code: "NF-8K2P7Q",
    package: "1 Hari 5M",
    reseller: "Kios Net Barokah",
    price: "Rp3.000",
    expires: "02 Agu 2026",
    status: "tersedia",
  },
  {
    code: "NF-Q4M9TA",
    package: "7 Hari 10M",
    reseller: "Raja Voucher Cibinong",
    price: "Rp18.000",
    expires: "01 Agu 2026",
    status: "terjual",
  },
  {
    code: "NF-X1P8BD",
    package: "1 Hari 5M",
    reseller: "Kios Net Barokah",
    price: "Rp3.000",
    expires: "03 Mei 2026",
    status: "aktif",
  },
];

export const resellers = [
  {
    name: "Kios Net Barokah",
    phone: "+62 812-8840-2193",
    balance: "Rp1.420.000",
    soldToday: "47",
    limit: "200/hari",
    status: "aktif",
  },
  {
    name: "Raja Voucher Cibinong",
    phone: "+62 857-1029-6631",
    balance: "Rp380.000",
    soldToday: "19",
    limit: "120/hari",
    status: "aktif",
  },
  {
    name: "Warung Online Sawangan",
    phone: "+62 813-7750-8802",
    balance: "Rp0",
    soldToday: "0",
    limit: "80/hari",
    status: "suspended",
  },
];

export const auditLogs = [
  {
    time: "14:30",
    user: "Admin Budi",
    action: "Edit pelanggan PLG-001",
    module: "Pelanggan",
  },
  {
    time: "14:25",
    user: "System",
    action: "Isolir otomatis PLG-014",
    module: "Billing",
  },
  {
    time: "14:20",
    user: "Kasir Ani",
    action: "Catat pembayaran Rp620.000",
    module: "Pembayaran",
  },
  {
    time: "14:10",
    user: "Teknisi Andi",
    action: "Disconnect session PPPoE",
    module: "MikroTik",
  },
];
