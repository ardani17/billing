export const landingFaqs = [
  {
    q: "Apa itu software billing ISP?",
    a: "Software billing ISP membantu operator internet mengelola pelanggan, paket, invoice, pembayaran, reminder, dan status layanan dari satu sistem operasional.",
  },
  {
    q: "Apakah ISPBoss cocok untuk RT/RW Net?",
    a: "Cocok. ISPBoss bisa dipakai dari skala RT/RW Net sampai ISP lokal yang sudah memiliki banyak area, router, dan tim operasional.",
  },
  {
    q: "Apakah bisa integrasi MikroTik PPPoE?",
    a: "Bisa. ISPBoss disiapkan untuk mengelola PPPoE, isolir, buka isolir, session, dan sinkronisasi data pelanggan dengan MikroTik RouterOS.",
  },
  {
    q: "Apakah mendukung OLT dan jaringan FTTH?",
    a: "Ya. Modul jaringan dirancang untuk membantu pengelolaan OLT, ODP, ONT, alarm, provisioning, dan peta jaringan FTTH.",
  },
  {
    q: "Bagaimana migrasi dari billing lama?",
    a: "Data pelanggan, paket, invoice, pembayaran, dan area bisa disiapkan melalui import terstruktur agar migrasi berjalan bertahap.",
  },
  {
    q: "Apakah trial perlu kartu kredit?",
    a: "Tidak. Trial dibuat agar operator bisa memvalidasi alur billing ISP dan operasional jaringan terlebih dahulu.",
  },
];

export const publicPricingPlans = [
  {
    name: "Starter",
    range: "0-100 pelanggan",
    price: "Rp150rb",
    detail: "Untuk RT/RW Net yang mulai rapi.",
  },
  {
    name: "Growth",
    range: "101-500 pelanggan",
    price: "Rp350rb",
    detail: "Untuk ISP lokal dengan tim operasional.",
  },
  {
    name: "Pro",
    range: "501-2000 pelanggan",
    price: "Rp750rb",
    detail: "Untuk jaringan multi-router dan multi-area.",
    featured: true,
  },
  {
    name: "Enterprise",
    range: "2000+ pelanggan",
    price: "Custom",
    detail: "Untuk kebutuhan khusus, SLA, dan migrasi besar.",
  },
];
