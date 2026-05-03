import {
  Bell,
  Broadcast,
  ChartLineUp,
  CheckCircle,
  ClockCounterClockwise,
  CreditCard,
  FileCsv,
  FileText,
  Globe,
  HardDrives,
  MapPin,
  MapTrifold,
  Money,
  Package,
  Plus,
  Receipt,
  ShieldCheck,
  Storefront,
  Ticket,
  UploadSimple,
  Users,
  Warning,
  WifiHigh,
} from "@phosphor-icons/react/dist/ssr";
import AppShell from "./app-shell";
import {
  auditLogs,
  customers,
  invoices,
  notifications,
  olts,
  packages,
  payments,
  resellers,
  routers,
  vouchers,
} from "./mock-data";
import {
  Button,
  DataTable,
  EmptyState,
  FilterBar,
  FormField,
  PageHeader,
  Section,
  SelectInput,
  StatGrid,
  StatusBadge,
  TextInput,
} from "./ui";

const textAreaClass =
  "w-full min-w-0 rounded-md border border-slate-300 px-3 py-2 text-sm outline-none transition focus:border-blue-500 focus:ring-2 focus:ring-blue-100";

const revenueTrend = [
  { month: "Nov", revenue: 36.2, target: 34, gateway: 21.4, manual: 14.8, outstanding: 5.9 },
  { month: "Des", revenue: 42.8, target: 38, gateway: 24.9, manual: 17.9, outstanding: 6.4 },
  { month: "Jan", revenue: 39.7, target: 40, gateway: 22.5, manual: 17.2, outstanding: 7.1 },
  { month: "Feb", revenue: 46.9, target: 42, gateway: 28.1, manual: 18.8, outstanding: 6.8 },
  { month: "Mar", revenue: 44.3, target: 45, gateway: 25.6, manual: 18.7, outstanding: 8.2 },
  { month: "Apr", revenue: 48.7, target: 46, gateway: 31.2, manual: 17.5, outstanding: 8.2 },
];

function RevenueChartMock() {
  const maxValue = Math.max(...revenueTrend.flatMap((item) => [item.revenue, item.target, item.outstanding])) + 8;
  const chartHeight = 210;
  const chartWidth = 640;
  const leftPadding = 44;
  const rightPadding = 18;
  const topPadding = 16;
  const bottomPadding = 36;
  const usableWidth = chartWidth - leftPadding - rightPadding;
  const usableHeight = chartHeight - topPadding - bottomPadding;
  const step = usableWidth / (revenueTrend.length - 1);
  const yFor = (value: number) => topPadding + usableHeight - (value / maxValue) * usableHeight;
  const targetPoints = revenueTrend
    .map((item, index) => `${leftPadding + index * step},${yFor(item.target)}`)
    .join(" ");
  const revenuePoints = revenueTrend
    .map((item, index) => `${leftPadding + index * step},${yFor(item.revenue)}`)
    .join(" ");

  return (
    <div className="grid gap-5">
      <div className="grid gap-3 sm:grid-cols-3">
        {[
          { label: "Total diterima", value: "Rp258,6jt", detail: "+11,8% vs periode lalu" },
          { label: "Via gateway", value: "Rp153,7jt", detail: "59,4% dari pembayaran" },
          { label: "Piutang aktif", value: "Rp8,2jt", detail: "18 invoice belum lunas" },
        ].map((item) => (
          <div key={item.label} className="rounded-lg border border-slate-200 bg-slate-50 px-4 py-3">
            <p className="text-xs font-medium uppercase tracking-[0.14em] text-slate-500">{item.label}</p>
            <p className="mt-2 font-mono text-xl font-semibold text-slate-950">{item.value}</p>
            <p className="mt-1 text-xs text-slate-500">{item.detail}</p>
          </div>
        ))}
      </div>

      <div className="min-w-0">
        <svg
          viewBox={`0 0 ${chartWidth} ${chartHeight}`}
          role="img"
          aria-label="Grafik pendapatan 6 bulan terakhir"
          className="h-auto w-full rounded-lg border border-slate-200 bg-white"
        >
          {[0, 15, 30, 45, 60].map((tick) => {
            const y = yFor(tick);
            return (
              <g key={tick}>
                <line x1={leftPadding} x2={chartWidth - rightPadding} y1={y} y2={y} stroke="#e2e8f0" strokeWidth="1" />
                <text x="12" y={y + 4} className="fill-slate-400 text-[11px]">
                  {tick}jt
                </text>
              </g>
            );
          })}

          {revenueTrend.map((item, index) => {
            const x = leftPadding + index * step;
            const gatewayHeight = (item.gateway / maxValue) * usableHeight;
            const manualHeight = (item.manual / maxValue) * usableHeight;
            const baseY = topPadding + usableHeight;
            const barWidth = 28;
            return (
              <g key={item.month}>
                <rect
                  x={x - barWidth / 2}
                  y={baseY - gatewayHeight}
                  width={barWidth}
                  height={gatewayHeight}
                  rx="5"
                  className="fill-blue-600"
                />
                <rect
                  x={x - barWidth / 2}
                  y={baseY - gatewayHeight - manualHeight}
                  width={barWidth}
                  height={manualHeight}
                  rx="5"
                  className="fill-sky-300"
                />
                <text x={x} y={bottomPadding + usableHeight + 8} textAnchor="middle" className="fill-slate-500 text-[12px] font-medium">
                  {item.month}
                </text>
              </g>
            );
          })}

          <polyline points={targetPoints} fill="none" stroke="#94a3b8" strokeWidth="2" strokeDasharray="5 5" />
          <polyline points={revenuePoints} fill="none" stroke="#0f172a" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round" />
          {revenueTrend.map((item, index) => {
            const x = leftPadding + index * step;
            const y = yFor(item.revenue);
            return (
              <g key={`${item.month}-point`}>
                <circle cx={x} cy={y} r="4.5" className="fill-white stroke-slate-950" strokeWidth="2" />
                {index === revenueTrend.length - 1 && (
                  <g>
                    <rect x={x - 52} y={y - 38} width="104" height="26" rx="6" className="fill-slate-950" />
                    <text x={x} y={y - 21} textAnchor="middle" className="fill-white text-[11px] font-semibold">
                      Rp{item.revenue.toFixed(1)}jt
                    </text>
                  </g>
                )}
              </g>
            );
          })}
        </svg>
      </div>

      <div className="flex flex-wrap gap-x-5 gap-y-2 text-xs text-slate-500">
        <span className="inline-flex items-center gap-2"><span className="h-2.5 w-2.5 rounded-sm bg-blue-600" />Gateway</span>
        <span className="inline-flex items-center gap-2"><span className="h-2.5 w-2.5 rounded-sm bg-sky-300" />Manual</span>
        <span className="inline-flex items-center gap-2"><span className="h-0.5 w-5 bg-slate-950" />Total diterima</span>
        <span className="inline-flex items-center gap-2"><span className="h-0.5 w-5 border-t border-dashed border-slate-400" />Target</span>
      </div>
    </div>
  );
}

export function DashboardPage() {
  return (
    <AppShell>
      <div className="space-y-6">
        <PageHeader
          eyebrow="Dashboard"
          title="Pusat kontrol operasional"
          description="Ringkasan pelanggan, pendapatan, tunggakan, dan status jaringan dari modul yang aktif."
          actions={<Button href="/customers/new">Tambah Pelanggan</Button>}
        />
        <StatGrid
          stats={[
            { label: "Total pelanggan aktif", value: "847", delta: "+32 bulan ini" },
            { label: "Pendapatan bulan ini", value: "Rp48,7jt", delta: "+12,4%" },
            { label: "Tunggakan", value: "Rp8,2jt", delta: "18 invoice", tone: "amber" },
            { label: "Router online", value: "31/32", delta: "1 offline", tone: "red" },
          ]}
        />
        <div className="grid gap-6 xl:grid-cols-[1.2fr_0.8fr]">
          <Section title="Grafik pendapatan" description="6 bulan terakhir, dari pembayaran manual dan gateway.">
            <RevenueChartMock />
          </Section>
          <Section title="Aktivitas terkini" description="Event penting dari billing dan network.">
            <div className="divide-y divide-slate-100">
              {auditLogs.map((log) => (
                <div key={`${log.time}-${log.action}`} className="flex gap-3 py-3">
                  <span className="font-mono text-xs text-slate-400">{log.time}</span>
                  <span>
                    <span className="block text-sm font-medium text-slate-800">{log.action}</span>
                    <span className="text-xs text-slate-500">{log.user} - {log.module}</span>
                  </span>
                </div>
              ))}
            </div>
          </Section>
        </div>
        <Section title="Status MikroTik" description="Auto-refresh dari Network Service.">
          <DataTable
            columns={["Router", "IP", "Versi", "Uptime", "User Aktif", "CPU", "Status"]}
            rows={routers.map((router) => [
              router.name,
              router.ip,
              router.version,
              router.uptime,
              router.active,
              router.cpu,
              <StatusBadge key={router.name} status={router.status} />,
            ])}
          />
        </Section>
        <Section title="Status OLT" description="Kesehatan OLT, kapasitas PON, jumlah ONT, dan alarm aktif.">
          <DataTable
            columns={["OLT", "Brand", "IP", "PON", "ONT", "Alarm", "Status"]}
            rows={olts.map((olt) => [
              <a key={olt.name} href="/olt/OLT-01" className="font-semibold text-blue-700">{olt.name}</a>,
              olt.brand,
              olt.ip,
              olt.pon,
              olt.ont,
              olt.alarms,
              <StatusBadge key={olt.name} status={olt.status} />,
            ])}
          />
        </Section>
      </div>
    </AppShell>
  );
}

export function CustomersPage() {
  return (
    <AppShell>
      <div className="space-y-6">
        <PageHeader
          eyebrow="Pelanggan"
          title="Daftar pelanggan"
          description="Kelola data pelanggan, paket, status billing, koordinat GPS, dan aksi isolir massal."
          actions={
            <>
              <Button variant="secondary" href="/customers/areas">Area</Button>
              <Button href="/customers/new">Tambah Pelanggan</Button>
            </>
          }
        />
        <StatGrid
          stats={[
            { label: "Aktif", value: "782", delta: "92,3%" },
            { label: "Isolir", value: "38", delta: "H+7", tone: "amber" },
            { label: "Suspend", value: "17", delta: "30+ hari", tone: "red" },
            { label: "Pending", value: "10", delta: "Butuh aktivasi", tone: "blue" },
          ]}
        />
        <FilterBar search="Cari nama, ID pelanggan, telepon..." filters={["Status", "Paket", "Area", "Jatuh Tempo"]} />
        <DataTable
          columns={["ID", "Nama", "Telepon", "Paket", "Area", "Jatuh Tempo", "Tagihan", "Status", "Aksi"]}
          rows={customers.map((customer) => [
            customer.id,
            <a key={customer.id} href={`/customers/${customer.id}`} className="font-semibold text-blue-700">{customer.name}</a>,
            customer.phone,
            customer.package,
            customer.area,
            customer.dueDate,
            customer.balance,
            <StatusBadge key={customer.name} status={customer.status} />,
            <Button key={`edit-${customer.id}`} variant="ghost" href={`/customers/${customer.id}`}>Detail</Button>,
          ])}
        />
      </div>
    </AppShell>
  );
}

export function CustomerFormPage() {
  return (
    <AppShell>
      <div className="space-y-6">
        <PageHeader
          eyebrow="Pelanggan"
          title="Tambah pelanggan"
          description="Data pelanggan baru dipakai untuk billing, notifikasi, PPPoE, dan FTTH mapping."
          actions={<Button>Simpan Pelanggan</Button>}
        />
        <div className="grid gap-6 xl:grid-cols-[1fr_360px]">
          <Section title="Informasi pelanggan">
            <div className="grid gap-5 md:grid-cols-2">
              <FormField label="Nama lengkap"><TextInput placeholder="Ahmad Rizki" /></FormField>
              <FormField label="No. Telepon / WA"><TextInput placeholder="+62 812..." /></FormField>
              <FormField label="Email"><TextInput placeholder="pelanggan@email.com" /></FormField>
              <FormField label="Area"><SelectInput options={["Depok Timur", "Cibinong", "Bojonggede", "Sawangan"]} /></FormField>
              <FormField label="Alamat"><textarea rows={4} className={`${textAreaClass} md:col-span-2`} /></FormField>
              <FormField label="Paket internet"><SelectInput options={["Basic 20M", "Pro 50M", "Ultra 100M", "Office 75M"]} /></FormField>
              <FormField label="Tanggal aktif"><TextInput placeholder="03/05/2026" /></FormField>
            </div>
          </Section>
          <Section title="Network assignment" description="Wajib jika modul MikroTik atau OLT aktif.">
            <div className="grid gap-4">
              <FormField label="Mode koneksi"><SelectInput options={["PPPoE", "DHCP Binding", "Static IP"]} /></FormField>
              <FormField label="Username PPPoE"><TextInput placeholder="ahmad-plg001" /></FormField>
              <FormField label="Router MikroTik"><SelectInput options={["MK-Depok-01", "MK-Cibinong-02", "MK-Sawangan-03"]} /></FormField>
              <FormField label="ODP / Port OLT"><TextInput placeholder="ODP-05-A / Port 3" /></FormField>
              <Button variant="secondary" href="/network-map">Pilih Koordinat di Peta</Button>
            </div>
          </Section>
        </div>
      </div>
    </AppShell>
  );
}

export function CustomerDetailPage() {
  const customer = customers[0] ?? {
    id: "PLG-001",
    name: "Ahmad Rizki",
    phone: "+62 812-7841-2096",
    package: "Pro 50M",
    area: "Depok Timur",
    dueDate: "05 Mei 2026",
    status: "aktif",
    balance: "Rp0",
  };
  return (
    <AppShell>
      <div className="space-y-6">
        <PageHeader
          eyebrow="Pelanggan"
          title={customer.name}
          description={`${customer.id} - ${customer.package} - ${customer.area}`}
          actions={
            <>
              <Button variant="secondary">Ganti Paket</Button>
              <Button variant="secondary">Kirim Notifikasi</Button>
              <Button>Catat Pembayaran</Button>
            </>
          }
        />
        <StatGrid
          stats={[
            { label: "Status pelanggan", value: "Aktif" },
            { label: "Saldo kredit", value: "Rp0" },
            { label: "Invoice terbuka", value: "1" },
            { label: "Status PPPoE", value: "Online" },
          ]}
        />
        <div className="grid gap-6 xl:grid-cols-[0.9fr_1.1fr]">
          <Section title="Ringkasan">
            <DataTable
              columns={["Field", "Nilai"]}
              rows={[
                ["Telepon", customer.phone],
                ["Alamat", "Jl. Margonda Raya No. 118, Depok"],
                ["Router", "MK-Depok-01"],
                ["ODP", "ODP-05-A Port 3"],
                ["Koordinat", "-6.402341, 106.794201"],
              ]}
            />
          </Section>
          <Section title="Riwayat invoice">
            <DataTable
              columns={["Invoice", "Periode", "Jumlah", "Status"]}
              rows={invoices.slice(0, 3).map((invoice) => [
                invoice.number,
                invoice.period,
                invoice.amount,
                <StatusBadge key={invoice.number} status={invoice.status} />,
              ])}
            />
          </Section>
        </div>
      </div>
    </AppShell>
  );
}

export function CustomerAreasPage() {
  return (
    <AppShell>
      <div className="space-y-6">
        <PageHeader
          eyebrow="Pelanggan"
          title="Area / wilayah"
          description="Kelompokkan pelanggan untuk filter, jadwal teknisi, dan laporan pendapatan per area."
          actions={<Button>Tambah Area</Button>}
        />
        <DataTable
          columns={["Area", "Pelanggan", "Tunggakan", "Teknisi", "Status"]}
          rows={[
            ["Depok Timur", "318", "Rp2,1jt", "Andi Pratama", <StatusBadge key="a" status="aktif" />],
            ["Cibinong", "214", "Rp3,4jt", "Dimas Wicaksono", <StatusBadge key="b" status="aktif" />],
            ["Sawangan", "129", "Rp1,8jt", "Andi Pratama", <StatusBadge key="c" status="aktif" />],
          ]}
        />
      </div>
    </AppShell>
  );
}

export function PackagesPage() {
  return (
    <AppShell>
      <div className="space-y-6">
        <PageHeader
          eyebrow="Paket"
          title="Paket internet"
          description="Kelola paket PPPoE/Static dan Hotspot/Voucher dengan harga, bandwidth, quota, dan profile jaringan."
          actions={<Button href="/packages/new">Tambah Paket</Button>}
        />
        <FilterBar search="Cari nama paket..." filters={["Jenis", "Status"]} />
        <DataTable
          columns={["Nama", "Jenis", "Harga", "Bandwidth", "Pelanggan/Voucher", "Status", "Aksi"]}
          rows={packages.map((item) => [
            item.name,
            item.type,
            item.price,
            item.speed,
            item.customers,
            <StatusBadge key={item.name} status={item.status} />,
            <Button key={`duplicate-${item.name}`} variant="ghost">Duplikat</Button>,
          ])}
        />
      </div>
    </AppShell>
  );
}

export function PackageFormPage() {
  return (
    <AppShell>
      <div className="space-y-6">
        <PageHeader
          eyebrow="Paket"
          title="Tambah paket"
          description="Pilih tipe paket. Field MikroTik dapat disembunyikan jika modul jaringan belum aktif."
          actions={<Button>Simpan Paket</Button>}
        />
        <div className="grid gap-6 xl:grid-cols-2">
          <Section title="Informasi paket">
            <div className="grid gap-5 md:grid-cols-2">
              <FormField label="Nama paket"><TextInput placeholder="Pro 50M" /></FormField>
              <FormField label="Jenis paket"><SelectInput options={["PPPoE/Static", "Hotspot/Voucher"]} /></FormField>
              <FormField label="Harga bulanan"><TextInput placeholder="350000" /></FormField>
              <FormField label="Biaya pasang"><TextInput placeholder="0" /></FormField>
              <FormField label="Download Mbps"><TextInput placeholder="50" /></FormField>
              <FormField label="Upload Mbps"><TextInput placeholder="20" /></FormField>
            </div>
          </Section>
          <Section title="Bandwidth dan quota">
            <div className="grid gap-5 md:grid-cols-2">
              <FormField label="Tipe bandwidth"><SelectInput options={["Dedicated", "Shared up-to"]} /></FormField>
              <FormField label="Quota"><SelectInput options={["Unlimited", "Quota Bulanan", "FUP"]} /></FormField>
              <FormField label="Profile MikroTik"><TextInput placeholder="profile-pro-50m" /></FormField>
              <FormField label="Address pool"><TextInput placeholder="pool-depok" /></FormField>
            </div>
          </Section>
        </div>
      </div>
    </AppShell>
  );
}

export function VouchersPage() {
  return (
    <AppShell>
      <div className="space-y-6">
        <PageHeader
          eyebrow="Voucher"
          title="Manajemen voucher"
          description="Generate, assign ke reseller, print PDF, void, dan pantau lifecycle voucher."
          actions={<Button>Generate Voucher</Button>}
        />
        <StatGrid
          stats={[
            { label: "Tersedia", value: "1.428" },
            { label: "Terjual", value: "842", tone: "blue" },
            { label: "Aktif", value: "319", tone: "amber" },
            { label: "Expired", value: "27", tone: "red" },
          ]}
        />
        <DataTable
          columns={["Kode", "Paket", "Reseller", "Harga", "Berlaku Sampai", "Status"]}
          rows={vouchers.map((voucher) => [
            <span key={voucher.code} className="font-mono font-semibold">{voucher.code}</span>,
            voucher.package,
            voucher.reseller,
            voucher.price,
            voucher.expires,
            <StatusBadge key={voucher.code} status={voucher.status} />,
          ])}
        />
      </div>
    </AppShell>
  );
}

export function ResellersPage() {
  return (
    <AppShell>
      <div className="space-y-6">
        <PageHeader
          eyebrow="Reseller"
          title="Mitra reseller voucher"
          description="Kelola akun reseller, saldo, limit pembelian harian, dan deposit."
          actions={<Button>Tambah Reseller</Button>}
        />
        <DataTable
          columns={["Nama", "Telepon", "Saldo", "Terjual Hari Ini", "Limit", "Status", "Aksi"]}
          rows={resellers.map((reseller) => [
            reseller.name,
            reseller.phone,
            reseller.balance,
            reseller.soldToday,
            reseller.limit,
            <StatusBadge key={reseller.name} status={reseller.status} />,
            <Button key={`deposit-${reseller.name}`} variant="ghost">Deposit</Button>,
          ])}
        />
      </div>
    </AppShell>
  );
}

export function ResellerDashboardPage() {
  return (
    <main className="min-h-[100dvh] bg-slate-950 px-4 py-6 text-white sm:px-6 lg:px-8">
      <div className="mx-auto max-w-6xl space-y-6">
        <PageHeader
          eyebrow="Dashboard Reseller"
          title="Kios Net Barokah"
          description="Beli voucher, cetak voucher, dan pantau deposit dari dashboard terpisah."
          actions={<Button>Beli Voucher</Button>}
        />
        <StatGrid
          stats={[
            { label: "Saldo", value: "Rp1,42jt" },
            { label: "Terjual hari ini", value: "47" },
            { label: "Voucher tersedia", value: "318" },
            { label: "Komisi bulan ini", value: "Rp684rb" },
          ]}
        />
        <DataTable
          columns={["Kode", "Paket", "Harga", "Status"]}
          rows={vouchers.map((voucher) => [
            <span key={voucher.code} className="font-mono font-semibold">{voucher.code}</span>,
            voucher.package,
            voucher.price,
            <StatusBadge key={voucher.code} status={voucher.status} />,
          ])}
        />
      </div>
    </main>
  );
}

export function InvoicesPage() {
  return (
    <AppShell>
      <div className="space-y-6">
        <PageHeader
          eyebrow="Invoice"
          title="Daftar invoice"
          description="Pantau invoice bulanan, payment link, denda, prorate, dan status pembayaran."
          actions={
            <>
              <Button variant="secondary">Bulk Reminder</Button>
              <Button>Buat Invoice</Button>
            </>
          }
        />
        <FilterBar search="Cari no invoice, nama pelanggan, ID pelanggan..." filters={["Status", "Periode", "Paket", "Area"]} />
        <DataTable
          columns={["No Invoice", "Pelanggan", "Periode", "Jumlah", "Jatuh Tempo", "Status", "Aksi"]}
          rows={invoices.map((invoice) => [
            <a key={invoice.number} href={`/invoices/${invoice.number}`} className="font-mono font-semibold text-blue-700">{invoice.number}</a>,
            invoice.customer,
            invoice.period,
            invoice.amount,
            invoice.dueDate,
            <StatusBadge key={invoice.number} status={invoice.status} />,
            <Button key={`pay-${invoice.number}`} variant="ghost" href={`/invoices/${invoice.number}`}>Detail</Button>,
          ])}
        />
      </div>
    </AppShell>
  );
}

export function InvoiceDetailPage() {
  return (
    <AppShell>
      <div className="space-y-6">
        <PageHeader
          eyebrow="Invoice"
          title="INV-2026-05-001"
          description="Ahmad Rizki - Periode Mei 2026 - Jatuh tempo 05 Mei 2026"
          actions={
            <>
              <Button variant="secondary">Download PDF</Button>
              <Button>Catat Pembayaran</Button>
            </>
          }
        />
        <div className="grid gap-6 xl:grid-cols-[1fr_360px]">
          <Section title="Item invoice">
            <DataTable
              columns={["Item", "Qty", "Harga", "Subtotal"]}
              rows={[
                ["Paket Pro 50M", "1", "Rp350.000", "Rp350.000"],
                ["PPN 11%", "1", "Rp38.500", "Rp38.500"],
                ["Diskon loyalitas", "1", "-Rp20.000", "-Rp20.000"],
              ]}
            />
          </Section>
          <Section title="Ringkasan pembayaran">
            <div className="space-y-3 text-sm">
              <Row label="Subtotal" value="Rp350.000" />
              <Row label="PPN" value="Rp38.500" />
              <Row label="Diskon" value="-Rp20.000" />
              <Row label="Total" value="Rp368.500" strong />
              <Row label="Dibayar" value="Rp0" />
              <Row label="Sisa" value="Rp368.500" strong />
            </div>
          </Section>
        </div>
      </div>
    </AppShell>
  );
}

export function PaymentsPage() {
  return (
    <AppShell>
      <div className="space-y-6">
        <PageHeader
          eyebrow="Pembayaran"
          title="Pembayaran manual dan gateway"
          description="Catat pembayaran cepat, multi-invoice, bukti bayar, receipt, void, dan rekonsiliasi."
          actions={<Button>Bayar Cepat</Button>}
        />
        <StatGrid
          stats={[
            { label: "Diterima hari ini", value: "Rp7,84jt" },
            { label: "Transaksi", value: "38" },
            { label: "Gateway pending", value: "6", tone: "amber" },
            { label: "Void bulan ini", value: "2", tone: "red" },
          ]}
        />
        <DataTable
          columns={["Receipt", "Pelanggan", "Jumlah", "Metode", "Waktu", "Status"]}
          rows={payments.map((payment) => [
            <span key={payment.receipt} className="font-mono font-semibold">{payment.receipt}</span>,
            payment.customer,
            payment.amount,
            payment.method,
            payment.date,
            <StatusBadge key={payment.receipt} status={payment.status} />,
          ])}
        />
      </div>
    </AppShell>
  );
}

export function NotificationsPage() {
  return (
    <AppShell>
      <div className="space-y-6">
        <PageHeader
          eyebrow="Notifikasi"
          title="Template dan broadcast"
          description="Kelola provider WhatsApp, SMS, email, template, quiet hours, throttle, dan resend."
          actions={
            <>
              <Button variant="secondary">Test Send</Button>
              <Button>Broadcast</Button>
            </>
          }
        />
        <DataTable
          columns={["Template", "Channel", "Terkirim", "Gagal", "Terakhir Jalan", "Status"]}
          rows={notifications.map((item) => [
            item.template,
            item.channel,
            item.sent,
            item.failed,
            item.lastRun,
            <StatusBadge key={item.template} status={item.status} />,
          ])}
        />
      </div>
    </AppShell>
  );
}

export function MikrotikPage() {
  return (
    <AppShell>
      <div className="space-y-6">
        <PageHeader
          eyebrow="MikroTik"
          title="Router dan PPPoE"
          description="Kelola RouterOS v6/v7, PPPoE users, sessions, sync status, backup, firmware, dan isolir."
          actions={
            <>
              <Button variant="secondary" href="/mikrotik/vpn">VPN</Button>
              <Button>Tambah Router</Button>
            </>
          }
        />
        <StatGrid
          stats={[
            { label: "Router online", value: "31/32" },
            { label: "Active session", value: "7.218" },
            { label: "Sync pending", value: "14", tone: "amber" },
            { label: "Command failed", value: "3", tone: "red" },
          ]}
        />
        <DataTable
          columns={["Router", "IP", "Versi", "Uptime", "User Aktif", "CPU", "Status", "Aksi"]}
          rows={routers.map((router) => [
            <a key={router.name} href="/mikrotik/MK-Depok-01" className="font-semibold text-blue-700">{router.name}</a>,
            router.ip,
            router.version,
            router.uptime,
            router.active,
            router.cpu,
            <StatusBadge key={router.name} status={router.status} />,
            <Button key={`sync-${router.name}`} variant="ghost">Sync</Button>,
          ])}
        />
      </div>
    </AppShell>
  );
}

export function MikrotikDetailPage() {
  return (
    <AppShell>
      <div className="space-y-6">
        <PageHeader
          eyebrow="MikroTik"
          title="MK-Depok-01"
          description="10.10.8.1 - RouterOS 7.15 - 318 active sessions"
          actions={
            <>
              <Button variant="secondary">Backup Config</Button>
              <Button>Open Terminal</Button>
            </>
          }
        />
        <div className="grid gap-6 xl:grid-cols-3">
          <Section title="PPPoE users">
            <DataTable columns={["User", "Profile", "Status"]} rows={[["ahmad-plg001", "Pro 50M", <StatusBadge key="u1" status="online" />], ["rina-plg014", "Basic 20M", <StatusBadge key="u2" status="isolir" />]]} />
          </Section>
          <Section title="DHCP leases">
            <DataTable columns={["MAC", "IP", "Type"]} rows={[["C8:7F:54:19:2A:01", "10.8.1.22", "Static"], ["B0:4E:26:91:AB:10", "10.8.1.24", "Dynamic"]]} />
          </Section>
          <Section title="Firewall review">
            <EmptyState title="Tidak ada rule berisiko" description="Perintah berbahaya diblokir dari terminal dan dicatat di audit log." />
          </Section>
        </div>
      </div>
    </AppShell>
  );
}

export function MikrotikVpnPage() {
  return (
    <AppShell>
      <div className="space-y-6">
        <PageHeader
          eyebrow="MikroTik"
          title="VPN tunnel"
          description="Kelola WireGuard/SSTP/L2TP tunnel, failover endpoint, bandwidth cap, dan script konfigurasi."
          actions={<Button>Setup VPN Wizard</Button>}
        />
        <DataTable
          columns={["Tunnel", "Protocol", "Endpoint", "Bandwidth", "Last Seen", "Status"]}
          rows={[
            ["WG-Depok-01", "WireGuard", "vpn1.ispboss.id", "84 Mbps", "12 detik lalu", <StatusBadge key="vpn1" status="online" />],
            ["SSTP-Cibinong-02", "SSTP", "vpn2.ispboss.id", "27 Mbps", "1 menit lalu", <StatusBadge key="vpn2" status="online" />],
          ]}
        />
      </div>
    </AppShell>
  );
}

export function OltPage() {
  return (
    <AppShell>
      <div className="space-y-6">
        <PageHeader
          eyebrow="OLT"
          title="OLT multi-brand"
          description="Pantau ZTE, Huawei, FiberHome, VSOL, HSGQ, alarm, PON, ONT, SFP, dan kapasitas."
          actions={
            <>
              <Button variant="secondary" href="/olt/odp">ODP</Button>
              <Button variant="secondary" href="/olt/provisioning">Provisioning</Button>
              <Button href="/olt/new">Tambah OLT</Button>
            </>
          }
        />
        <DataTable
          columns={["OLT", "Brand", "IP", "PON", "ONT", "Alarm", "Status", "Aksi"]}
          rows={olts.map((olt) => [
            <a key={olt.name} href="/olt/OLT-01" className="font-semibold text-blue-700">{olt.name}</a>,
            olt.brand,
            olt.ip,
            olt.pon,
            olt.ont,
            olt.alarms,
            <StatusBadge key={olt.name} status={olt.status} />,
            <Button key={`test-${olt.name}`} variant="ghost">Test</Button>,
          ])}
        />
      </div>
    </AppShell>
  );
}

export function OltNewPage() {
  return (
    <AppShell>
      <div className="space-y-6">
        <PageHeader eyebrow="OLT" title="Tambah OLT" description="Auto-detect brand, firmware, board, PON ports, dan kemampuan SNMP/CLI." actions={<Button>Simpan OLT</Button>} />
        <Section title="Informasi OLT">
          <div className="grid gap-5 md:grid-cols-2">
            <FormField label="Nama OLT"><TextInput placeholder="OLT-01 Pusat" /></FormField>
            <FormField label="Brand"><SelectInput options={["ZTE", "Huawei", "FiberHome", "VSOL", "HSGQ"]} /></FormField>
            <FormField label="IP Management"><TextInput placeholder="10.20.1.2" /></FormField>
            <FormField label="SNMP Community"><TextInput placeholder="public" /></FormField>
            <FormField label="CLI Username"><TextInput placeholder="admin" /></FormField>
            <FormField label="CLI Password"><TextInput placeholder="Tersimpan terenkripsi" /></FormField>
          </div>
        </Section>
      </div>
    </AppShell>
  );
}

export function OltDetailPage() {
  return (
    <AppShell>
      <div className="space-y-6">
        <PageHeader eyebrow="OLT" title="OLT-01 Pusat" description="ZTE C320 - 624 ONT - 3 alarm aktif" actions={<Button>Provision ONT</Button>} />
        <div className="grid gap-6 xl:grid-cols-[1fr_1fr]">
          <Section title="PON ports">
            <DataTable columns={["Port", "ONT", "Power Avg", "Traffic", "Status"]} rows={[["1/1/1", "58/64", "-22.8 dBm", "840 Mbps", <StatusBadge key="p1" status="online" />], ["1/1/2", "61/64", "-24.1 dBm", "732 Mbps", <StatusBadge key="p2" status="degraded" />]]} />
          </Section>
          <Section title="Alarm">
            <DataTable columns={["Severity", "Port", "Pesan", "Waktu"]} rows={[["Major", "1/1/2", "LOS ONT ZTEG12345678", "14:02"], ["Minor", "1/1/4", "High attenuation", "13:48"]]} />
          </Section>
        </div>
      </div>
    </AppShell>
  );
}

export function OdpPage() {
  return (
    <AppShell>
      <div className="space-y-6">
        <PageHeader eyebrow="OLT" title="ODP / Splitter" description="Kelola ODP, splitter ratio, kapasitas, koordinat, dan relasi ONT." actions={<Button>Tambah ODP</Button>} />
        <DataTable columns={["ODP", "Area", "OLT/PON", "Splitter", "Terpakai", "Koordinat"]} rows={[["ODP-05-A", "Depok Timur", "OLT-01 / 1/1/1", "1:8", "7/8", "-6.402, 106.794"], ["ODP-09-C", "Cibinong", "OLT-02 / 1/1/3", "1:16", "11/16", "-6.474, 106.856"]]} />
      </div>
    </AppShell>
  );
}

export function ProvisioningPage() {
  return (
    <AppShell>
      <div className="space-y-6">
        <PageHeader eyebrow="OLT" title="Provisioning ONT" description="Provision ONT single/bulk, unregistered ONT, service profile, VLAN, dan audit provisioning." actions={<Button>Provision ONT</Button>} />
        <div className="grid gap-6 xl:grid-cols-[1fr_360px]">
          <Section title="Unregistered ONT">
            <DataTable columns={["Serial Number", "OLT", "PON", "Signal", "Action"]} rows={[["ZTEG12345678", "OLT-01 Pusat", "1/1/1", "-21.7 dBm", <Button key="prov1" variant="ghost">Provision</Button>], ["HWTC98110421", "OLT-02 Barat", "1/1/3", "-23.2 dBm", <Button key="prov2" variant="ghost">Provision</Button>]]} />
          </Section>
          <Section title="Bulk upload">
            <EmptyState title="Siapkan CSV provisioning" description="Upload data pelanggan, SN ONT, VLAN, service profile, dan ODP untuk provisioning massal." action={<Button variant="secondary">Upload CSV</Button>} />
          </Section>
        </div>
      </div>
    </AppShell>
  );
}

export function HelpPage() {
  return (
    <AppShell>
      <div className="space-y-6">
        <PageHeader eyebrow="Bantuan" title="Pusat bantuan" description="Panduan pengguna, troubleshooting, changelog, video tutorial, dan tiket support." actions={<Button>Buat Tiket Support</Button>} />
        <div className="grid gap-6 md:grid-cols-2 xl:grid-cols-3">
          {[
            ["Panduan pengguna", FileText, "Dokumentasi fitur per modul."],
            ["Troubleshooting", Warning, "MikroTik, OLT, billing, dan notifikasi."],
            ["Video tutorial", ChartLineUp, "Materi onboarding untuk operator."],
            ["Changelog", ClockCounterClockwise, "Riwayat update fitur ISPBoss."],
            ["API Docs", Globe, "Referensi endpoint dan webhook."],
            ["Support", ShieldCheck, "Hubungi tim ISPBoss."],
          ].map(([title, Icon, body]) => (
            <Section key={String(title)} title={String(title)} description={String(body)}>
              <Button variant="secondary">Buka</Button>
            </Section>
          ))}
        </div>
      </div>
    </AppShell>
  );
}

export function SettingsIndexPage() {
  const items = [
    ["Profil ISP", "/settings/profile", "Nama, alamat, telepon, email, NPWP, timezone."],
    ["White Label", "/settings/branding", "Logo, favicon, warna primer, custom domain."],
    ["User & Role", "/settings/users", "CRUD user, role, session, preferensi notifikasi."],
    ["Billing", "/settings/billing", "Invoice, denda, pajak, reminder, isolir."],
    ["Payment Gateway", "/settings/payment", "Xendit, Midtrans, webhook, channel."],
    ["Notifikasi", "/settings/notifications", "Provider, quiet hours, throttle, template."],
    ["MikroTik", "/settings/mikrotik", "Isolir method, PPPoE format, sync interval."],
    ["OLT", "/settings/olt", "Signal threshold, VLAN strategy, auto-provisioning."],
    ["Voucher", "/settings/voucher", "Format kode, masa berlaku, limit generate."],
    ["Lokalisasi", "/settings/localization", "Tanggal, mata uang, bahasa interface."],
    ["Invoice", "/settings/invoice", "Footer, email signature, rekening bank."],
    ["Peta", "/settings/map", "Geocoding, label node, default map center."],
    ["Keamanan", "/settings/security", "Password, 2FA, session, API key."],
    ["Subscription", "/settings/subscription", "Paket saat ini, module registry, billing SaaS."],
    ["Audit Log", "/settings/audit-log", "Append-only log, filter, export CSV."],
  ];

  return (
    <AppShell>
      <div className="space-y-6">
        <PageHeader eyebrow="Pengaturan" title="Pusat konfigurasi tenant" description="Semua pengaturan modul ISPBoss dikelompokkan supaya mudah ditemukan." />
        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
          {items.map(([title, href, body]) => (
            <a key={href} href={href} className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm transition hover:-translate-y-1 hover:border-blue-200">
              <h2 className="font-semibold text-slate-950">{title}</h2>
              <p className="mt-2 text-sm leading-6 text-slate-500">{body}</p>
            </a>
          ))}
        </div>
      </div>
    </AppShell>
  );
}

export function GenericSettingsPage({
  title,
  description,
  fields,
}: {
  title: string;
  description: string;
  fields: { label: string; helper?: string; type?: "select" | "textarea" | "toggle"; options?: string[]; value?: string }[];
}) {
  return (
    <AppShell>
      <div className="space-y-6">
        <PageHeader eyebrow="Pengaturan" title={title} description={description} actions={<Button>Simpan</Button>} />
        <Section title={title} description="Perubahan tersimpan di tenant dan dipakai lintas modul.">
          <div className="grid gap-5 md:grid-cols-2">
            {fields.map((field) => (
              <FormField key={field.label} label={field.label} helper={field.helper}>
                {field.type === "select" ? (
                  <SelectInput options={field.options ?? ["Aktif", "Nonaktif"]} />
                ) : field.type === "textarea" ? (
                  <textarea rows={4} className={textAreaClass} defaultValue={field.value} />
                ) : field.type === "toggle" ? (
                  <label className="flex h-10 items-center gap-3 rounded-md border border-slate-300 px-3 text-sm">
                    <input type="checkbox" defaultChecked className="h-4 w-4 rounded border-slate-300 text-blue-600" />
                    Aktif
                  </label>
                ) : (
                  <TextInput defaultValue={field.value} />
                )}
              </FormField>
            ))}
          </div>
        </Section>
      </div>
    </AppShell>
  );
}

export function AuditLogPage() {
  return (
    <AppShell>
      <div className="space-y-6">
        <PageHeader eyebrow="Pengaturan" title="Audit log" description="Log append-only untuk auth, pelanggan, billing, network, notifikasi, dan settings." actions={<Button variant="secondary">Export CSV</Button>} />
        <FilterBar search="Cari aksi, user, modul..." filters={["User", "Aksi", "Modul", "Periode"]} />
        <DataTable columns={["Waktu", "User", "Aksi", "Modul"]} rows={auditLogs.map((log) => [log.time, log.user, log.action, log.module])} />
      </div>
    </AppShell>
  );
}

export function PublicWalledGardenPage() {
  return (
    <main className="min-h-[100dvh] bg-slate-950 px-4 py-10 text-white">
      <div className="mx-auto grid min-h-[calc(100dvh-5rem)] max-w-5xl items-center gap-8 md:grid-cols-[0.9fr_1.1fr]">
        <div>
          <span className="inline-flex rounded-full bg-amber-400/15 px-3 py-1 text-xs font-semibold uppercase tracking-[0.18em] text-amber-200">
            Layanan sementara dibatasi
          </span>
          <h1 className="mt-5 text-4xl font-semibold tracking-tight md:text-6xl">
            Tagihan internet belum dibayar.
          </h1>
          <p className="mt-5 max-w-xl text-slate-300">
            Selesaikan pembayaran untuk mengaktifkan kembali layanan. Setelah pembayaran diterima, sistem akan membuka isolir otomatis.
          </p>
          <div className="mt-8 flex flex-col gap-3 sm:flex-row">
            <Button>Bayar Sekarang</Button>
            <Button variant="secondary">Hubungi Admin</Button>
          </div>
        </div>
        <div className="rounded-2xl border border-white/10 bg-white/5 p-6">
          <div className="mb-6 flex items-center gap-3">
            <span className="grid h-12 w-12 place-items-center rounded-lg bg-blue-600 text-sm font-black">NF</span>
            <div>
              <p className="font-semibold">NusaFiber Depok</p>
              <p className="text-sm text-slate-400">billing.nusafiber.id</p>
            </div>
          </div>
          <DataTable
            columns={["Info", "Nilai"]}
            rows={[
              ["Pelanggan", "Rina Maheswari"],
              ["Invoice", "INV-2026-05-014"],
              ["Periode", "Mei 2026"],
              ["Total", "Rp388.500"],
            ]}
          />
        </div>
      </div>
    </main>
  );
}

function Row({ label, value, strong }: { label: string; value: string; strong?: boolean }) {
  return (
    <div className={`flex justify-between border-b border-slate-100 pb-2 ${strong ? "font-semibold text-slate-950" : "text-slate-600"}`}>
      <span>{label}</span>
      <span>{value}</span>
    </div>
  );
}

export const settingsConfigs = {
  profile: {
    title: "Profil ISP",
    description: "Nama, alamat, kontak, NPWP, dan timezone yang muncul di invoice, notifikasi, walled garden, dan laporan.",
    fields: [
      { label: "Nama ISP", value: "NusaFiber Depok" },
      { label: "No. Telepon", value: "+62 812-8472-1091" },
      { label: "Email", value: "admin@nusafiber.id" },
      { label: "Website", value: "www.nusafiber.id" },
      { label: "Alamat", type: "textarea" as const, value: "Jl. Margonda Raya No. 118, Depok, Jawa Barat" },
      { label: "Timezone", type: "select" as const, options: ["WIB (UTC+7)", "WITA (UTC+8)", "WIT (UTC+9)"] },
    ],
  },
  users: {
    title: "User & Role",
    description: "Kelola akses tenant admin, operator, teknisi, kasir, dan preferensi notifikasi per user.",
    fields: [
      { label: "Nama user", value: "Dewi Lestari" },
      { label: "Email", value: "dewi@nusafiber.id" },
      { label: "Role", type: "select" as const, options: ["Tenant Admin", "Operator", "Teknisi", "Kasir"] },
      { label: "Status", type: "select" as const, options: ["Aktif", "Nonaktif"] },
      { label: "Notifikasi invoice gagal", type: "toggle" as const },
      { label: "Router offline", type: "toggle" as const },
    ],
  },
  billing: {
    title: "Billing Settings",
    description: "Konfigurasi generate invoice, isolir, toleransi, denda, pajak, dan reminder bertingkat.",
    fields: [
      { label: "Generate invoice H-", value: "5" },
      { label: "Prefix invoice", value: "INV" },
      { label: "Grace period", value: "7" },
      { label: "Batas toleransi suspend", value: "30" },
      { label: "Auto-isolir", type: "toggle" as const },
      { label: "Auto-buka isolir", type: "toggle" as const },
    ],
  },
  payment: {
    title: "Payment Gateway",
    description: "Konfigurasi Xendit, Midtrans, webhook, sandbox mode, channel payment, dan payment link expiry.",
    fields: [
      { label: "Aktifkan Xendit", type: "toggle" as const },
      { label: "Xendit API Key", value: "***************" },
      { label: "Aktifkan Midtrans", type: "toggle" as const },
      { label: "Payment link expiry", value: "7 hari" },
      { label: "Sandbox mode", type: "toggle" as const },
      { label: "Webhook URL", value: "https://api.ispboss.id/v1/webhooks/xendit" },
    ],
  },
  notifications: {
    title: "Notifikasi Settings",
    description: "Provider WA/SMS/Email, quiet hours, throttle, deduplication, dan channel priority.",
    fields: [
      { label: "Provider WhatsApp", type: "select" as const, options: ["Fonnte", "Zenziva", "Custom"] },
      { label: "Quiet hours mulai", value: "21:00" },
      { label: "Quiet hours selesai", value: "07:00" },
      { label: "Max broadcast per menit", value: "120" },
      { label: "Fallback ke SMS", type: "toggle" as const },
      { label: "Deduplication window", value: "24 jam" },
    ],
  },
  mikrotik: {
    title: "MikroTik Settings",
    description: "Bandwidth method, isolir method, PPPoE username format, sync interval, dan walled garden.",
    fields: [
      { label: "Isolir method", type: "select" as const, options: ["firewall_nat_redirect", "pppoe_disable", "address_list"] },
      { label: "Username PPPoE format", value: "{nama-depan}-{id-pelanggan}" },
      { label: "Sync interval", value: "15 menit" },
      { label: "Health check interval", value: "60 detik" },
      { label: "Walled garden message", type: "textarea" as const, value: "Tagihan Anda belum dibayar." },
      { label: "Auto port migration", type: "toggle" as const },
    ],
  },
  olt: {
    title: "OLT Settings",
    description: "Signal threshold, VLAN assignment strategy, auto-provisioning, trap receiver, dan ONT limit.",
    fields: [
      { label: "Signal warning threshold", value: "-25 dBm" },
      { label: "Signal critical threshold", value: "-28 dBm" },
      { label: "VLAN strategy", type: "select" as const, options: ["Per area", "Per OLT", "Per package"] },
      { label: "Auto-provisioning", type: "toggle" as const },
      { label: "SNMP trap port", value: "162" },
      { label: "Max ONT per PON", value: "64" },
    ],
  },
  voucher: {
    title: "Voucher Settings",
    description: "Format kode, prefix, masa berlaku, max generate, collision retry, dan refund expired.",
    fields: [
      { label: "Format kode", type: "select" as const, options: ["Gabungan", "Angka", "Huruf"] },
      { label: "Panjang kode", value: "8" },
      { label: "Prefix", value: "NF" },
      { label: "Masa berlaku", value: "90 hari" },
      { label: "Max generate sinkron", value: "500" },
      { label: "Refund saat expired", type: "toggle" as const },
    ],
  },
  localization: {
    title: "Lokalisasi",
    description: "Format tanggal, mata uang, pemisah ribuan, dan bahasa interface.",
    fields: [
      { label: "Format tanggal", type: "select" as const, options: ["DD/MM/YYYY", "MM/DD/YYYY"] },
      { label: "Mata uang", value: "Rp" },
      { label: "Pemisah ribuan", type: "select" as const, options: ["Titik", "Koma"] },
      { label: "Bahasa interface", type: "select" as const, options: ["Bahasa Indonesia", "English"] },
    ],
  },
  invoice: {
    title: "Kustomisasi Invoice",
    description: "Footer invoice, email signature, nomor rekening bank, dan pesan pembayaran.",
    fields: [
      { label: "Footer invoice", type: "textarea" as const, value: "Terima kasih atas pembayaran Anda." },
      { label: "Email signature", type: "textarea" as const, value: "Salam,\nTim NusaFiber Depok" },
      { label: "Bank utama", value: "BCA" },
      { label: "No. rekening", value: "123-456-789" },
      { label: "Atas nama", value: "PT NusaFiber Depok" },
    ],
  },
  map: {
    title: "Peta Settings",
    description: "Geocoding provider, label node, default center, offline mode, dan share map.",
    fields: [
      { label: "Geocoding provider", type: "select" as const, options: ["Nominatim", "Google Geocoding"] },
      { label: "Default latitude", value: "-6.402341" },
      { label: "Default longitude", value: "106.794201" },
      { label: "Min zoom label", value: "16" },
      { label: "Share map enabled", type: "toggle" as const },
      { label: "Offline area download", type: "toggle" as const },
    ],
  },
  security: {
    title: "Keamanan",
    description: "Ubah password, 2FA, session management, API key, dan rate limit login.",
    fields: [
      { label: "Password saat ini", value: "" },
      { label: "Password baru", value: "" },
      { label: "2FA Google Authenticator", type: "toggle" as const },
      { label: "Session expiry", value: "24 jam" },
      { label: "Remember me expiry", value: "7 hari" },
      { label: "Login max attempts", value: "5" },
    ],
  },
  subscription: {
    title: "Subscription",
    description: "Paket SaaS saat ini, upgrade, riwayat pembayaran, dan module registry tenant.",
    fields: [
      { label: "Paket saat ini", type: "select" as const, options: ["Starter", "Growth", "Pro", "Enterprise"] },
      { label: "Pelanggan saat ini", value: "847" },
      { label: "Berlaku sampai", value: "05 Mei 2026" },
      { label: "MikroTik module", type: "toggle" as const },
      { label: "OLT module", type: "toggle" as const },
      { label: "FTTH Mapping module", type: "toggle" as const },
    ],
  },
};
