# 09 — Integrasi OLT (Multi-Brand)

---

## Konsep Integrasi

Network Service (Golang) berkomunikasi dengan perangkat OLT via **SNMP** (library `gosnmp`) dan **SSH/Telnet** (library `golang.org/x/crypto/ssh`). OLT mengelola layer fisik jaringan fiber (FTTH), sementara MikroTik mengelola layer bandwidth/billing.

```
Pelanggan FTTH:
  Internet ← MikroTik (bandwidth) ← OLT (fiber) ← ODP (splitter) ← ONT (rumah pelanggan)

ISPBoss mengelola:
  - OLT: provisioning ONT, monitoring signal, alarm
  - MikroTik: PPPoE user, bandwidth, isolir
  - Keduanya terhubung via pelanggan yang sama
```

### Brand OLT yang Didukung

| Brand | Protokol | Keterangan |
|---|---|---|
| **ZTE** (C300, C320, C600) | SNMP + Telnet/SSH | Paling populer di Indonesia |
| **Huawei** (MA56xx) | SNMP + SSH | Enterprise grade |
| **FiberHome** (AN5516) | SNMP + Telnet | Banyak dipakai ISP menengah |
| **VSOL** (V1600G, V1600D) | SNMP + Telnet/SSH | Budget-friendly, populer di RT/RW Net |
| **HSGQ** | SNMP + Telnet | Budget, mirip VSOL |

- Kode **terpisah per brand** (adapter pattern, sama seperti MikroTik v6/v7)
- Setiap brand punya OID SNMP dan command CLI yang berbeda
- Auto-detect brand saat tambah OLT (dari SNMP sysDescr)

### Adapter Pattern per Brand

```
┌─────────────────────────────────────────────────────┐
│ OLTService                                           │
│                                                     │
│  interface: OLTProvider                              │
│  ├── ZTEAdapter      (SNMP + Telnet)                │
│  ├── HuaweiAdapter   (SNMP + SSH)                   │
│  ├── FiberHomeAdapter(SNMP + Telnet)                │
│  ├── VSOLAdapter     (SNMP + Telnet/SSH)            │
│  └── HSGQAdapter     (SNMP + Telnet)                │
│                                                     │
│  Setiap adapter implement:                           │
│  - GetONTList()                                      │
│  - ProvisionONT()                                    │
│  - DecommissionONT()                                 │
│  - GetONTSignal()                                    │
│  - GetONTStatus()                                    │
│  - GetAlarms()                                       │
│  - GetPortStatus()                                   │
│  - RebootONT()                                       │
└─────────────────────────────────────────────────────┘
```

> **Referensi Implementasi:** Repo riset `snmp-zte` (https://github.com/ardani17/snmp-zte) sudah memiliki implementasi ZTE C320 yang bisa dijadikan dasar adapter ZTE di ISPBoss. Repo ini memiliki 71 endpoint (51 READ + 20 WRITE) yang sudah ditest terhadap OLT real.

### Hybrid System: SNMP + CLI

Berdasarkan riset di repo `snmp-zte`, ZTE OLT menggunakan **hybrid system**:

| Operasi | Via SNMP | Via CLI (Telnet/SSH) | Keterangan |
|---|---|---|---|
| **Monitoring ONT** (status, signal, traffic) | ✅ | ✅ | SNMP lebih efisien untuk polling |
| **Create/Delete ONT** | ✅ (via RowStatus) | ✅ | SNMP bisa, tapi CLI lebih reliable |
| **Rename ONT** | ✅ (SNMP SET) | ✅ | Keduanya works |
| **Distance ONT** | ✅ | ✅ | SNMP works |
| **VLAN Config per ONT** | ❌ | ✅ | Hanya bisa via CLI |
| **Create/Delete VLAN** | ❌ | ✅ | Hanya bisa via CLI |
| **Service Port Config** | ❌ | ✅ | Hanya bisa via CLI |
| **T-CONT & GEM Port** | ❌ | ✅ | Hanya bisa via CLI |
| **Profile Management** | ❌ | ✅ | Hanya bisa via CLI |
| **Bandwidth Assignment** | ❌ | ✅ | Hanya bisa via CLI |
| **Hardware Info** (card, fan, temp) | ✅ | ✅ | Keduanya works |

**Strategi:** SNMP untuk monitoring (polling berkala), CLI untuk provisioning dan konfigurasi.

### OID yang Sudah Diverifikasi (ZTE C320)

Dari riset `snmp-zte`, berikut OID yang sudah terbukti works:

```
ONU Management:
  Base: 1.3.6.1.4.1.3902.1012.3.28.1.1.{field}.{oltId}.{onuId}
  .1 = TypeName        .2 = Name (WRITEABLE)
  .3 = Description     .5 = SerialNumber
  .8 = TargetState     .9 = RowStatus (create/delete)

Distance:
  Base: 1.3.6.1.4.1.3902.1012.3.11.4.1.{field}.{oltId}.{onuId}
  .2 = Distance (meters)

Bandwidth Profiles:
  Base: 1.3.6.1.4.1.3902.1012.3.26.2.1.{field}.{profileIndex}
  .2 = ProfileName  .3 = FixedBW  .4 = AssuredBW

PON Port Stats:
  Base: 1.3.6.1.4.1.3902.1015.1010.5.4.1.{field}.{oltId}
  .2 = RxOctets  .3 = RxPkts  .17 = TxOctets  .18 = TxPkts

VLAN List:
  Base: 1.3.6.1.2.1.17.7.1.4.3.1.1.{vlanId}

Index Calculation (PON Port → OLT ID):
  oltId = (1 << 28) | (0 << 24) | (board << 16) | (pon << 8)
  Board 1, PON 1 = 268501248
  Board 1, PON 2 = 268501504
```

### SNMP Community Strings (ZTE)
| Community | Akses | Fungsi |
|---|---|---|
| `public` | Read-Only | Monitoring |
| `globalrw` | Read-Write | Provisioning (create/delete/rename ONT) |

> ⚠️ Community string default harus diganti di production. ISPBoss menyimpan community string terenkripsi per OLT.

### Koneksi ke OLT

Sama seperti MikroTik, OLT diakses via **VPN tunnel** ISPBoss:

| Protokol | Port | Kegunaan |
|---|---|---|
| SNMP v2c/v3 | 161 | Monitoring: signal, status, alarm, traffic |
| SSH | 22 | Provisioning: add/remove ONT, konfigurasi |
| Telnet | 23 | Provisioning (brand yang tidak support SSH) |

- Koneksi via VPN IP (tidak perlu IP publik OLT)
- Credential terenkripsi AES-256 di database
- **SNMP v3 direkomendasikan** (enkripsi + auth), fallback ke v2c jika OLT tidak support


---

## Halaman Daftar OLT (`/olt`)

### Layout Desktop

```
╔══════════════════════════════════════════════════════════════════════════╗
║  Dashboard > OLT                                                         ║
║                                                                          ║
║  OLT                                                [+ Tambah OLT]       ║
║                                                                          ║
║  ┌────────────┬────────────┬────────────┬────────────┐                   ║
║  │ 📡 Total   │ 🟢 Online  │ 🔴 Offline │ ⚠️ Alarm   │                   ║
║  │ 3 OLT      │ 2          │ 0          │ 1 alarm    │                   ║
║  └────────────┴────────────┴────────────┴────────────┘                   ║
║                                                                          ║
║  ┌──────────┬──────────┬──────────────┬────────┬──────┬──────┬────────┐  ║
║  │ Nama     │ Brand    │ IP Address   │ PON    │ ONT  │Status│ Aksi   │  ║
║  │          │          │              │ Port   │ Total│      │        │  ║
║  ├──────────┼──────────┼──────────────┼────────┼──────┼──────┼────────┤  ║
║  │ OLT-01   │ ZTE C320 │ 10.99.0.10   │ 8 port │ 245  │🟢 On │ ⋯      │  ║
║  │ Pusat    │          │ (via VPN)    │        │      │      │        │  ║
║  ├──────────┼──────────┼──────────────┼────────┼──────┼──────┼────────┤  ║
║  │ OLT-02   │ VSOL     │ 10.99.0.11   │ 4 port │ 120  │🟢 On │ ⋯      │  ║
║  │ Cabang A │ V1600G   │ (via VPN)    │        │      │      │        │  ║
║  ├──────────┼──────────┼──────────────┼────────┼──────┼──────┼────────┤  ║
║  │ OLT-03   │ Huawei   │ 10.99.0.12   │ 16 port│ 580  │🟢 On │ ⋯      │  ║
║  │ Cabang B │ MA5608T  │ (via VPN)    │ ⚠️1 alm│      │      │        │  ║
║  └──────────┴──────────┴──────────────┴────────┴──────┴──────┴────────┘  ║
╚══════════════════════════════════════════════════════════════════════════╝
```

### Layout Mobile (Card List)

```
┌──────────────────────────────┐
│ 📡 OLT-01 Pusat       🟢 On │
│ ZTE C320 • 10.99.0.10       │
│ PON: 8 port • ONT: 245      │
│ Alarm: 0               [⋯]  │
├──────────────────────────────┤
│ 📡 OLT-03 Cabang B    🟢 On │
│ Huawei MA5608T • 10.99.0.12 │
│ PON: 16 port • ONT: 580     │
│ ⚠️ 1 alarm aktif       [⋯]  │
└──────────────────────────────┘
```

### Status OLT
| Status | Warna | Arti |
|---|---|---|
| 🟢 Online | Green | SNMP reachable, OLT beroperasi normal |
| 🔴 Offline | Red | SNMP tidak bisa dihubungi |
| 🟡 Maintenance | Amber | Sedang maintenance (diset manual) |
| ⚠️ Alarm | Orange | Online tapi ada alarm aktif (LOS, power, dll) |

---

## Form Tambah OLT (`/olt/new`)

```
╔══════════════════════════════════════════════════════════════╗
║  Dashboard > OLT > Tambah OLT                                ║
║                                                              ║
║  ┌─── Informasi OLT ─────────────────────────────────────┐   ║
║  │  Nama OLT *              Lokasi / Keterangan          │   ║
║  │  [OLT-01 Pusat_______]  [Gedung utama lt.1_______]   │   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║  ┌─── Koneksi SNMP ──────────────────────────────────────┐   ║
║  │  IP Address *              SNMP Version *             │   ║
║  │  [10.99.0.10_________]    ● v2c  ○ v3                │   ║
║  │                                                       │   ║
║  │  (jika v2c):                                          │   ║
║  │  Community String *                                   │   ║
║  │  [public_____________]                                │   ║
║  │                                                       │   ║
║  │  (jika v3):                                           │   ║
║  │  Username *          Auth Protocol *                   │   ║
║  │  [admin_________]   [SHA ▼]                           │   ║
║  │  Auth Password *     Privacy Protocol *               │   ║
║  │  [••••••••••]       [AES ▼]                           │   ║
║  │  Privacy Password *                                   │   ║
║  │  [••••••••••]                                         │   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║  ┌─── Koneksi CLI (SSH/Telnet) ──────────────────────────┐   ║
║  │  Protokol *            Port *                         │   ║
║  │  ● SSH  ○ Telnet      [22____]                        │   ║
║  │                                                       │   ║
║  │  Username *            Password *                     │   ║
║  │  [admin_________]     [••••••••••      👁️]            │   ║
║  │                                                       │   ║
║  │  Enable Password (jika ada)                           │   ║
║  │  [••••••••••]                                         │   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║  ┌─── Test & Auto-Detect ────────────────────────────────┐   ║
║  │  [🔍 Test Koneksi SNMP]  [🔍 Test Koneksi CLI]       │   ║
║  │                                                       │   ║
║  │  Hasil Auto-Detect:                                   │   ║
║  │  Brand: ZTE                                           │   ║
║  │  Model: C320                                          │   ║
║  │  Firmware: V2.1.0                                     │   ║
║  │  PON Ports: 8 (GPON)                                  │   ║
║  │  Uptime: 120 hari 5 jam                               │   ║
║  │  Total ONT: 245                                       │   ║
║  └───────────────────────────────────────────────────────┘   ║
║                                                              ║
║  Health Check Interval: [300] detik (5 menit)                ║
║                                                              ║
║                              [Batal]  [Simpan OLT]           ║
╚══════════════════════════════════════════════════════════════╝
```

### Field OLT
| Field | Wajib | Keterangan |
|---|---|---|
| Nama OLT | ✅ | Nama identifikasi, unik per tenant |
| Lokasi / Keterangan | ❌ | Deskripsi lokasi fisik |
| IP Address | ✅ | IP OLT (via VPN atau langsung) |
| SNMP Version | ✅ | v2c (community string) atau v3 (user/auth/priv) |
| Community String | Conditional | Wajib jika SNMP v2c |
| SNMP v3 Credentials | Conditional | Wajib jika SNMP v3 (username, auth, privacy) |
| CLI Protocol | ✅ | SSH (rekomendasi) atau Telnet |
| CLI Port | ✅ | Default: 22 (SSH) atau 23 (Telnet) |
| CLI Username | ✅ | Username login OLT |
| CLI Password | ✅ | Password login OLT (terenkripsi) |
| Enable Password | ❌ | Beberapa OLT butuh enable password untuk privileged mode |
| Health Check Interval | ❌ | Default 300 detik (5 menit). OLT lebih jarang dicek dari MikroTik |

### Auto-Detect
Saat test koneksi berhasil:
1. **SNMP**: baca `sysDescr`, `sysName`, `sysUpTime` → detect brand, model, firmware, uptime
2. **CLI**: login → jalankan command info → detect jumlah PON port, total ONT
3. Hasil ditampilkan di section "Hasil Auto-Detect"
4. Brand otomatis menentukan adapter mana yang dipakai


---

## Detail OLT (`/olt/:id`)

```
╔══════════════════════════════════════════════════════════════════╗
║  Dashboard > OLT > OLT-01 Pusat                                 ║
║                                                                  ║
║  ┌──────────────────────────────────────────────────────────┐    ║
║  │  📡 OLT-01 Pusat                         🟢 Online       │    ║
║  │  ZTE C320 • 10.99.0.10 • Firmware V2.1.0                │    ║
║  │  Uptime: 120 hari • PON: 8 port • ONT: 245              │    ║
║  │                                                          │    ║
║  │  [Edit]  [Scan ONT]  [Refresh]  [⋯ Lainnya]             │    ║
║  └──────────────────────────────────────────────────────────┘    ║
║                                                                  ║
║  ┌─ Tab ────────────────────────────────────────────────────┐    ║
║  │ [PON Ports] [ONT List] [Alarm] [Traffic] [Unregistered]  │    ║
║  │ [Log]                                                    │    ║
║  └──────────────────────────────────────────────────────────┘    ║
╚══════════════════════════════════════════════════════════════════╝
```

### Tab PON Ports

```
┌──────────────────────────────────────────────────────────────────┐
│ PON Ports — OLT-01 Pusat (ZTE C320)                              │
│                                                                  │
│ ┌──────┬──────────┬──────────┬──────────┬──────────┬───────────┐ │
│ │ Port │ Status   │ ONT      │ ONT      │ Traffic  │ Aksi      │ │
│ │      │          │ Total    │ Online   │ (↓/↑)    │           │ │
│ ├──────┼──────────┼──────────┼──────────┼──────────┼───────────┤ │
│ │ 0/1  │ 🟢 Up    │ 64       │ 60       │ 1.2G/300M│ [Detail]  │ │
│ │ 0/2  │ 🟢 Up    │ 58       │ 55       │ 980M/250M│ [Detail]  │ │
│ │ 0/3  │ 🟢 Up    │ 45       │ 42       │ 750M/180M│ [Detail]  │ │
│ │ 0/4  │ 🟢 Up    │ 38       │ 35       │ 620M/150M│ [Detail]  │ │
│ │ 0/5  │ 🟢 Up    │ 25       │ 24       │ 400M/100M│ [Detail]  │ │
│ │ 0/6  │ 🟢 Up    │ 15       │ 14       │ 200M/50M │ [Detail]  │ │
│ │ 0/7  │ 🟢 Up    │ 0        │ 0        │ -        │ [Detail]  │ │
│ │ 0/8  │ 🔴 Down  │ 0        │ 0        │ -        │ [Detail]  │ │
│ └──────┴──────────┴──────────┴──────────┴──────────┴───────────┘ │
│                                                                  │
│ Total ONT: 245 • Online: 230 • Offline: 15                      │
│ ⚠️ Port 0/1 mendekati kapasitas (64/64 ONT per PON)            │
└──────────────────────────────────────────────────────────────────┘
```

### Tab ONT List

```
┌──────────────────────────────────────────────────────────────────┐
│ ONT List — OLT-01 Pusat                                          │
│                                                                  │
│ 🔍 Cari SN, nama...  Filter: [Port ▼] [Status ▼] [Reset]      │
│                                                                  │
│ ┌──────┬──────────────┬──────────┬────────┬────────┬──────┬────┐ │
│ │ Port │ SN           │ Pelanggan│ Signal │ Status │ Uptime│Aksi│ │
│ ├──────┼──────────────┼──────────┼────────┼────────┼──────┼────┤ │
│ │ 0/1:1│ ZTEG12345678 │ Ahmad R. │ -18.5  │ 🟢 On  │ 45d  │ ⋯  │ │
│ │      │              │ PLG-001  │ dBm    │        │      │    │ │
│ ├──────┼──────────────┼──────────┼────────┼────────┼──────┼────┤ │
│ │ 0/1:2│ ZTEG87654321 │ Budi S.  │ -22.3  │ 🟢 On  │ 30d  │ ⋯  │ │
│ │      │              │ PLG-002  │ dBm    │        │      │    │ │
│ ├──────┼──────────────┼──────────┼────────┼────────┼──────┼────┤ │
│ │ 0/1:3│ ZTEG11223344 │ Citra D. │ -28.1  │ ⚠️ Weak│ 15d  │ ⋯  │ │
│ │      │              │ PLG-003  │ dBm    │        │      │    │ │
│ ├──────┼──────────────┼──────────┼────────┼────────┼──────┼────┤ │
│ │ 0/2:1│ ZTEG55667788 │ Dewi A.  │ -      │ 🔴 Off │ -    │ ⋯  │ │
│ │      │              │ PLG-004  │        │ (LOS)  │      │    │ │
│ └──────┴──────────────┴──────────┴────────┴────────┴──────┴────┘ │
│                                                                  │
│ Total: 245 • Online: 230 • Offline: 10 • Weak Signal: 5         │
└──────────────────────────────────────────────────────────────────┘
```

### Signal Level Indicator
| Range (dBm) | Status | Warna | Keterangan |
|---|---|---|---|
| -8 s/d -25 | 🟢 Normal | Green | Signal bagus |
| -25 s/d -27 | 🟡 Warning | Amber | Signal mulai lemah, perlu perhatian |
| -27 s/d -30 | ⚠️ Weak | Orange | Signal lemah, rawan putus |
| < -30 atau LOS | 🔴 Critical / LOS | Red | Signal sangat lemah atau Loss of Signal |

- Threshold configurable per tenant (default di atas)
- Alert otomatis ke teknisi jika ada ONT dengan signal weak/critical

### ONT Status vs Customer Status

ONT status dan customer status adalah **2 hal independen**:

| ONT Status | Customer Status | Situasi | Aksi |
|---|---|---|---|
| 🟢 Online | 🟢 Aktif | Normal, semua baik | Tidak perlu aksi |
| 🟢 Online | 🔴 Isolir | Internet di-redirect tapi ONT masih nyala | Normal saat isolir |
| 🔴 Offline | 🟢 Aktif | ONT mati/kabel putus, tapi billing aktif | Alert teknisi, troubleshoot |
| 🔴 Offline | 🔴 Isolir | ONT mati + tunggakan | Prioritas rendah (bayar dulu) |
| 🔴 Offline | ⚫ Berhenti | Pelanggan sudah berhenti | Decommission ONT |
| ⚪ Pending | 🟡 Pending | ONT belum dipasang | Tunggu teknisi pasang |

**Aturan penting:**
- Isolir dilakukan di **MikroTik** (disable PPPoE), bukan di OLT
- ONT tetap online meskipun pelanggan diisolir (ONT hanya layer fisik)
- Jika ONT offline > 30 hari dan customer Aktif → alert ke teknisi (kemungkinan churn diam-diam)
- Jika ONT offline dan customer Berhenti → jadwalkan decommission

### Aksi per ONT
| Aksi | Deskripsi | Konfirmasi |
|---|---|---|
| Detail | Lihat info lengkap ONT (signal history, traffic, config) | Tidak |
| Reboot | Reboot ONT dari OLT | Ya |
| Decommission | Hapus ONT dari OLT (saat pelanggan berhenti) | Ya |
| Lihat Pelanggan | Navigasi ke detail pelanggan di billing | Tidak |
| Lihat di Peta | Buka FTTH map centered di lokasi ONT | Tidak |

### Tab Alarm

```
┌──────────────────────────────────────────────────────────────────┐
│ Alarm — OLT-01 Pusat                          [Refresh] [Export] │
│                                                                  │
│ Filter: [Severity ▼] [Port ▼] [Status ▼] [Reset]               │
│                                                                  │
│ ┌──────────┬──────────┬──────────────────────┬────────┬────────┐ │
│ │ Waktu    │ Severity │ Alarm                │ Port   │ Status │ │
│ ├──────────┼──────────┼──────────────────────┼────────┼────────┤ │
│ │ 14:30    │ 🔴 Major │ ONT LOS (Loss of     │ 0/2:1  │ Active │ │
│ │          │          │ Signal)               │ PLG-004│        │ │
│ ├──────────┼──────────┼──────────────────────┼────────┼────────┤ │
│ │ 12:15    │ 🟡 Minor │ ONT Signal Degraded  │ 0/1:3  │ Active │ │
│ │          │          │ (-28.1 dBm)           │ PLG-003│        │ │
│ ├──────────┼──────────┼──────────────────────┼────────┼────────┤ │
│ │ 08:00    │ 🟢 Clear │ ONT LOS Cleared      │ 0/3:5  │ Cleared│ │
│ │          │          │ (kembali online)      │ PLG-015│        │ │
│ └──────────┴──────────┴──────────────────────┴────────┴────────┘ │
│                                                                  │
│ Active Alarms: 2 (1 Major, 1 Minor)                             │
└──────────────────────────────────────────────────────────────────┘
```

### Tipe Alarm
| Alarm | Severity | Penyebab Umum |
|---|---|---|
| ONT LOS (Loss of Signal) | 🔴 Major | Kabel fiber putus, ONT mati, konektor lepas |
| ONT Signal Degraded | 🟡 Minor | Kabel kotor, bending, jarak terlalu jauh |
| PON Port Down | 🔴 Critical | SFP module rusak, port OLT bermasalah |
| Power Failure | 🔴 Major | OLT kehilangan power (jika ada UPS monitoring) |
| High Temperature | 🟡 Minor | Suhu OLT terlalu tinggi |
| ONT Dying Gasp | 🟡 Minor | ONT kehilangan power (PLN mati di rumah pelanggan) |

- Alarm diambil via **SNMP trap** (push dari OLT) atau **SNMP polling** (pull berkala)
- Alarm aktif → notifikasi ke Teknisi via WA (lihat dokumen 07)
- Alarm history disimpan 90 hari

### Tab Traffic

```
┌──────────────────────────────────────────────────────────────────┐
│ Traffic — OLT-01 Pusat                                           │
│                                                                  │
│ PON Port: [0/1 ▼]    Refresh: [Auto 30s ▼]                     │
│                                                                  │
│ Downstream: 1.2 Gbps  ████████████████░░░░░  (48% of 2.5G)     │
│ Upstream:   300 Mbps   ██████░░░░░░░░░░░░░░  (12% of 2.5G)     │
│                                                                  │
│ Traffic 24 Jam:                                                  │
│ ↓ ▁▂▃▅▆█▇▅▆▇█▆▅▃▂▁▂▃▅▆█▇▅                                    │
│ ↑ ▁▁▂▂▃▃▄▃▃▄▅▄▃▂▁▁▂▂▃▃▄▃▃                                    │
│   00:00    06:00    12:00    18:00    sekarang                   │
│                                                                  │
│ Total Hari Ini: ↓ 2.8 TB  ↑ 680 GB                             │
└──────────────────────────────────────────────────────────────────┘
```

### Tab Unregistered ONT

```
┌──────────────────────────────────────────────────────────────────┐
│ Unregistered ONT — OLT-01 Pusat              [Refresh] [Scan]   │
│                                                                  │
│ ONT yang terdeteksi tapi belum di-provisioning:                  │
│                                                                  │
│ ┌──────┬──────────────┬──────────┬──────────────────────────────┐ │
│ │ Port │ SN           │ Signal   │ Aksi                         │ │
│ ├──────┼──────────────┼──────────┼──────────────────────────────┤ │
│ │ 0/3  │ ZTEG99887766 │ -19.2 dBm│ [Provisioning] [Abaikan]    │ │
│ │ 0/5  │ ZTEG11224455 │ -21.0 dBm│ [Provisioning] [Abaikan]    │ │
│ └──────┴──────────────┴──────────┴──────────────────────────────┘ │
│                                                                  │
│ 2 ONT belum terdaftar                                            │
└──────────────────────────────────────────────────────────────────┘
```

- OLT secara otomatis mendeteksi ONT baru yang terhubung ke port
- Admin bisa langsung provisioning dari sini
- Atau abaikan (misal ONT test/sementara)


---

## ODP / Splitter Management

ODP (Optical Distribution Point) adalah splitter yang membagi sinyal fiber dari OLT ke beberapa ONT. Posisi di jaringan: **OLT → ODP → ONT**.

### Halaman Daftar ODP (`/olt/odp`)

```
╔══════════════════════════════════════════════════════════════════════════╗
║  Dashboard > OLT > ODP / Splitter                                        ║
║                                                                          ║
║  ODP / Splitter                                     [+ Tambah ODP]       ║
║                                                                          ║
║  ┌──────────┬──────────┬──────────┬──────────┬──────────┬──────┬──────┐  ║
║  │ Nama     │ OLT      │ PON Port │ Tipe     │ Terpakai │Lokasi│ Aksi │  ║
║  ├──────────┼──────────┼──────────┼──────────┼──────────┼──────┼──────┤  ║
║  │ ODP-01-A │ OLT-01   │ 0/1      │ 1:8      │ 7/8      │ Jl.  │ ⋯   │  ║
║  │          │ Pusat    │          │          │ ⚠️ 87%   │Merdk │      │  ║
║  ├──────────┼──────────┼──────────┼──────────┼──────────┼──────┼──────┤  ║
║  │ ODP-01-B │ OLT-01   │ 0/1      │ 1:16     │ 10/16    │ Jl.  │ ⋯   │  ║
║  │          │ Pusat    │          │          │ 63%      │Sudmn │      │  ║
║  ├──────────┼──────────┼──────────┼──────────┼──────────┼──────┼──────┤  ║
║  │ ODP-02-A │ OLT-01   │ 0/2      │ 1:8      │ 3/8      │ Jl.  │ ⋯   │  ║
║  │          │ Pusat    │          │          │ 38%      │Ahmad │      │  ║
║  └──────────┴──────────┴──────────┴──────────┴──────────┴──────┴──────┘  ║
╚══════════════════════════════════════════════════════════════════════════╝
```

### Form Tambah ODP

```
╔══════════════════════════════════════════════════════════════╗
║  Tambah ODP / Splitter                                       ║
║                                                              ║
║  Nama ODP *             Tipe Splitter *                      ║
║  [ODP-01-A_________]   ○ 1:4  ● 1:8  ○ 1:16  ○ 1:32       ║
║                                                              ║
║  OLT *                  PON Port *                           ║
║  [OLT-01 Pusat ▼]     [0/1 ▼]                              ║
║                                                              ║
║  Lokasi / Alamat                                             ║
║  [Jl. Merdeka No. 5, tiang listrik depan masjid]           ║
║                                                              ║
║  Koordinat GPS                                               ║
║  [Lat: _________]  [Lng: _________]  [📍 Pilih Map]        ║
║                                                              ║
║                              [Batal]  [Simpan ODP]           ║
╚══════════════════════════════════════════════════════════════╝
```

### Field ODP
| Field | Wajib | Keterangan |
|---|---|---|
| Nama ODP | ✅ | Unik per tenant. Contoh: ODP-01-A |
| Tipe Splitter | ✅ | 1:4, 1:8, 1:16, 1:32 (menentukan kapasitas port) |
| OLT | ✅ | OLT parent yang terhubung |
| PON Port | ✅ | Port di OLT yang terhubung ke ODP ini |
| Lokasi / Alamat | ❌ | Deskripsi lokasi fisik (tiang, gedung, dll) |
| Koordinat GPS | ❌ | Untuk FTTH mapping (dokumen 10) |

### Relasi ODP ke ONT
- Saat provisioning ONT, admin bisa pilih ODP mana yang dipakai
- ODP menentukan port fisik di splitter (port 1-8 untuk splitter 1:8)
- Warning jika ODP sudah penuh (semua port terpakai)
- Tampilkan daftar ONT per ODP di detail ODP

---

## Provisioning ONT

### Alur Provisioning ONT Baru (Pelanggan Baru FTTH)

```
Pelanggan baru diaktivasi (metode: PPPoE via FTTH)
  │
  ▼
Teknisi pasang ONT di rumah pelanggan
  → ONT muncul di tab "Unregistered ONT"
  │
  ▼
Admin/Teknisi klik [Provisioning] atau dari form pelanggan:
  │
  ▼
Form Provisioning ONT:
  ├── Pilih pelanggan (atau auto-fill jika dari detail pelanggan)
  ├── SN ONT (auto-detect dari unregistered list)
  ├── PON Port (auto-detect)
  ├── Service Profile (dari paket pelanggan)
  ├── VLAN (opsional, configurable)
  ├── Deskripsi (auto: nama pelanggan + ID)
  │
  ▼
Network Service kirim command ke OLT via CLI:
  │
  ├── ZTE: 
  │     onu add sn {SN} ont-lineprofile-id {profile} ont-srvprofile-id {srv}
  │     service-port add vlan {vlan} gpon {port} ont {id} gemport {gem}
  │
  ├── Huawei:
  │     ont add {port} sn-auth {SN} omci ont-lineprofile-id {profile}
  │     service-port vlan {vlan} gpon {port} ont {id} gemport {gem}
  │
  ├── VSOL:
  │     onu add gpon-olt {port} sn {SN} profile {profile}
  │
  └── (setiap brand punya command berbeda, ditangani adapter)
  │
  ▼
Hasil:
  ├── Sukses → ONT terdaftar, link ke pelanggan, log
  └── Gagal → Error message, notifikasi admin
```

### Form Provisioning

```
╔══════════════════════════════════════════════════════════════╗
║  Provisioning ONT                                            ║
║                                                              ║
║  Pelanggan *                                                 ║
║  [Ahmad Rizki — PLG-001 — Pro 50M]  ← autocomplete          ║
║                                                              ║
║  OLT *                                                       ║
║  [OLT-01 Pusat (ZTE C320) ▼]                               ║
║                                                              ║
║  Serial Number ONT *                                         ║
║  [ZTEG99887766______]  atau [Pilih dari Unregistered ▼]     ║
║                                                              ║
║  PON Port *                                                  ║
║  [0/3 ▼]  (auto-detect dari SN jika sudah terdeteksi)      ║
║                                                              ║
║  Service Profile                                             ║
║  [Pro-50M ▼]  (auto dari paket pelanggan)                   ║
║                                                              ║
║  VLAN (opsional)                                             ║
║  [100___]                                                    ║
║                                                              ║
║  Deskripsi                                                   ║
║  [Ahmad Rizki PLG-001]  (auto-generate)                     ║
║                                                              ║
║                    [Batal]  [Provisioning ONT]                ║
╚══════════════════════════════════════════════════════════════╝
```

### Bulk Provisioning (CSV Upload)

Untuk migrasi dari sistem lama atau pasang banyak pelanggan sekaligus:

```
╔══════════════════════════════════════════════════════════════╗
║  Bulk Provisioning ONT                                       ║
║                                                              ║
║  OLT *: [OLT-01 Pusat (ZTE C320) ▼]                        ║
║                                                              ║
║  Upload CSV *                                                ║
║  [📎 Upload file CSV]  [📥 Download Template]               ║
║                                                              ║
║  Preview (5 baris pertama):                                  ║
║  ┌──────────────┬──────────┬──────┬──────┬──────────────────┐║
║  │ SN ONT       │ Pelanggan│ Port │ VLAN │ Status           │║
║  ├──────────────┼──────────┼──────┼──────┼──────────────────┤║
║  │ ZTEG99887766 │ PLG-001  │ 0/3  │ 100  │ ✅ Valid         │║
║  │ ZTEG11224455 │ PLG-002  │ 0/3  │ 100  │ ✅ Valid         │║
║  │ ZTEG00000000 │ PLG-999  │ 0/5  │ 100  │ ❌ PLG not found │║
║  └──────────────┴──────────┴──────┴──────┴──────────────────┘║
║                                                              ║
║  Valid: 2 • Error: 1 • Total: 3                              ║
║                                                              ║
║                    [Batal]  [Provisioning 2 ONT]             ║
╚══════════════════════════════════════════════════════════════╝
```

Template CSV:
```csv
sn_ont,pelanggan_id,pon_port,vlan,odp,deskripsi
ZTEG99887766,PLG-001,0/3,100,ODP-01-A,Ahmad Rizki
ZTEG11224455,PLG-002,0/3,100,ODP-01-A,Budi Santoso
```

### Auto-Provisioning (Opsional)

Jika SN ONT sudah diinput di form pelanggan sebelum ONT dipasang:

```
ONT baru terdeteksi di OLT (unregistered)
  │
  ▼
Sistem cek: apakah SN ini sudah terdaftar di database pelanggan?
  ├── Ya → Auto-provisioning:
  │     ├── Buat ONT di OLT dengan profile dari paket pelanggan
  │     ├── Link ONT ke pelanggan
  │     ├── Notifikasi ke Teknisi: "ONT {SN} auto-provisioned untuk {pelanggan}"
  │     └── Log
  │
  └── Tidak → Tampilkan di tab Unregistered seperti biasa
```

- Auto-provisioning **default nonaktif** (harus diaktifkan di Settings)
- Mencegah provisioning ONT yang salah
- Cocok untuk ISP yang sudah input SN ONT saat registrasi pelanggan

### Decommission ONT (Pelanggan Berhenti)

```
Event: customer.terminated (dari Billing API)
  │
  ▼
Network Service:
  ├── Hapus ONT dari OLT via CLI:
  │     ZTE: onu delete {port} {ont_id}
  │     Huawei: ont delete {port} {ont_id}
  │
  ├── Hapus service-port terkait
  ├── Update database: ONT → unlinked
  └── Log: "ONT {SN} decommissioned — pelanggan {id} berhenti"
```

---

## Detail ONT (`/olt/:olt_id/ont/:ont_id`)

```
╔══════════════════════════════════════════════════════════════════╗
║  Dashboard > OLT > OLT-01 > ONT ZTEG12345678                    ║
║                                                                  ║
║  ┌──────────────────────────────────────────────────────────┐    ║
║  │  ONT ZTEG12345678                         🟢 Online       │    ║
║  │  Pelanggan: Ahmad Rizki (PLG-001)                        │    ║
║  │  OLT: OLT-01 Pusat • Port: 0/1:1 • VLAN: 100           │    ║
║  │  Signal: -18.5 dBm (🟢 Normal)                          │    ║
║  │                                                          │    ║
║  │  [Reboot ONT]  [Lihat Pelanggan]  [Lihat di Peta]       │    ║
║  └──────────────────────────────────────────────────────────┘    ║
║                                                                  ║
║  ┌─── Info ONT ──────────────────────────────────────────────┐   ║
║  │  Serial Number: ZTEG12345678                              │   ║
║  │  Model: F660 v6.0                                         │   ║
║  │  Firmware: V6.0.10P2T1                                    │   ║
║  │  MAC Address: AA:BB:CC:DD:EE:FF                           │   ║
║  │  Uptime: 45 hari 3 jam                                    │   ║
║  │  Distance: 2.3 km (estimasi dari signal)                  │   ║
║  │  Last Down: 15 Apr 2026 08:30 (PLN mati)                 │   ║
║  └───────────────────────────────────────────────────────────┘   ║
║                                                                  ║
║  ┌─── Signal History (7 hari) ───────────────────────────────┐   ║
║  │  -18 ─────────────────────────────────────── Normal       │   ║
║  │  -20 ─────────────────────────────────────────            │   ║
║  │  -22 ─────────────────────────────────────────            │   ║
║  │  -25 ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ Warning      │   ║
║  │  -27 ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ Weak         │   ║
║  │  -30 ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ Critical     │   ║
║  │       Mon  Tue  Wed  Thu  Fri  Sat  Sun                   │   ║
║  └───────────────────────────────────────────────────────────┘   ║
║                                                                  ║
║  ┌─── Traffic ONT ───────────────────────────────────────────┐   ║
║  │  Download: 45.2 Mbps    Upload: 12.8 Mbps                │   ║
║  │  Total Hari Ini: ↓ 15 GB  ↑ 3.2 GB                      │   ║
║  └───────────────────────────────────────────────────────────┘   ║
╚══════════════════════════════════════════════════════════════════╝
```

---

## Monitoring & Health Check

### SNMP Polling (Berkala)
| Data | OID (contoh ZTE) | Interval |
|---|---|---|
| ONT Status (online/offline) | `.1.3.6.1.4.1.3902.1082.500.10.2.3.3.1.2` | 5 menit |
| ONT RX Signal (dBm) | `.1.3.6.1.4.1.3902.1082.500.10.2.3.3.1.4` | 5 menit |
| PON Port Traffic | `.1.3.6.1.2.1.2.2.1.10` / `.1.3.6.1.2.1.2.2.1.16` | 1 menit |
| OLT CPU/Memory | `.1.3.6.1.4.1.3902.1082.500.1.2.1` | 5 menit |
| OLT Temperature | Brand-specific | 5 menit |
| ONT Uptime | Brand-specific | 15 menit |

- OID berbeda per brand → ditangani oleh adapter masing-masing
- Data disimpan di Redis (time-series, retention 30 hari untuk signal, 7 hari untuk traffic)

### SNMP Trap (Push dari OLT)
Beberapa alarm dikirim langsung oleh OLT ke ISPBoss (tidak perlu polling):
- ONT LOS (Loss of Signal)
- ONT Dying Gasp (power failure)
- PON Port Down
- OLT Power Failure

ISPBoss menjalankan **SNMP trap receiver** yang listen di port 162:
```
Network Service
  └── SNMP Trap Receiver (port 162)
        ├── Terima trap dari OLT
        ├── Parse: brand-specific trap format
        ├── Simpan alarm ke database
        ├── Kirim notifikasi ke Teknisi (via event queue)
        └── Update status ONT di dashboard (real-time via WebSocket)
```

### Health Check OLT
- SNMP ping setiap **5 menit** (configurable)
- Jika gagal 3x berturut-turut → status OLT → 🔴 Offline
- Notifikasi ke Teknisi
- Jika kembali online → notifikasi "OLT kembali online"

---

## VLAN Management

### Halaman VLAN per OLT (`/olt/:id/vlan`)

```
┌──────────────────────────────────────────────────────────────────┐
│ VLAN — OLT-01 Pusat                            [+ Tambah VLAN]  │
│                                                                  │
│ ┌──────┬──────────────────┬──────────┬──────────┬──────────────┐ │
│ │ VLAN │ Nama             │ Tipe     │ Pelanggan│ Keterangan   │ │
│ ├──────┼──────────────────┼──────────┼──────────┼──────────────┤ │
│ │ 100  │ VLAN-Internet    │ Per Paket│ 200      │ Semua paket  │ │
│ │ 200  │ VLAN-IPTV        │ Service  │ 50       │ Layanan IPTV │ │
│ │ 300  │ VLAN-Management  │ Mgmt     │ -        │ OLT & ONT    │ │
│ └──────┴──────────────────┴──────────┴──────────┴──────────────┘ │
└──────────────────────────────────────────────────────────────────┘
```

### VLAN Assignment Strategy
| Strategy | Keterangan | Kapan Dipakai |
|---|---|---|
| **Single VLAN** | Semua pelanggan 1 VLAN (default) | ISP kecil, sederhana |
| **Per Paket** | VLAN berbeda per paket internet | Memisahkan traffic per tier |
| **Per ODP** | VLAN berbeda per ODP/splitter | Isolasi per area |
| **Per Pelanggan** | VLAN unik per pelanggan | Enterprise, isolasi penuh |

- Admin pilih strategy di **Settings > OLT > VLAN Strategy**
- Default: Single VLAN (paling sederhana)
- VLAN translation: jika OLT dan MikroTik pakai VLAN ID berbeda, mapping dikonfigurasi di OLT

---

## SFP Module Monitoring

Monitoring SFP (Small Form-factor Pluggable) module di setiap PON port:

```
┌──────────────────────────────────────────────────────────────────┐
│ SFP Status — OLT-01 Pusat                                        │
│                                                                  │
│ ┌──────┬──────────┬──────────┬──────────┬──────────┬───────────┐ │
│ │ Port │ SFP Type │ TX Power │ RX Power │ Temp     │ Status    │ │
│ ├──────┼──────────┼──────────┼──────────┼──────────┼───────────┤ │
│ │ 0/1  │ GPON C+  │ +3.2 dBm │ -18.5 dBm│ 42°C     │ 🟢 Normal │ │
│ │ 0/2  │ GPON C+  │ +3.1 dBm │ -19.0 dBm│ 43°C     │ 🟢 Normal │ │
│ │ 0/3  │ GPON C+  │ +2.8 dBm │ -20.2 dBm│ 45°C     │ 🟢 Normal │ │
│ │ 0/4  │ GPON B+  │ +1.5 dBm │ -22.0 dBm│ 48°C     │ 🟡 Warm   │ │
│ │ 0/5  │ GPON C+  │ +3.0 dBm │ -18.8 dBm│ 41°C     │ 🟢 Normal │ │
│ │ 0/6  │ GPON C+  │ +0.5 dBm │ -25.1 dBm│ 52°C     │ ⚠️ Degrad │ │
│ │ 0/7  │ -        │ -        │ -        │ -        │ ⚫ Empty  │ │
│ │ 0/8  │ -        │ -        │ -        │ -        │ ⚫ Empty  │ │
│ └──────┴──────────┴──────────┴──────────┴──────────┴───────────┘ │
│                                                                  │
│ ⚠️ Port 0/6: TX Power degraded, pertimbangkan ganti SFP module │
└──────────────────────────────────────────────────────────────────┘
```

| Status SFP | Kondisi | Aksi |
|---|---|---|
| 🟢 Normal | TX/RX dalam range, suhu normal | Tidak perlu aksi |
| 🟡 Warm | Suhu > 45°C tapi < 60°C | Monitor, cek ventilasi |
| ⚠️ Degraded | TX power turun signifikan | Pertimbangkan ganti SFP |
| 🔴 Failed | TX/RX di luar range, atau tidak terdeteksi | Ganti SFP segera |
| ⚫ Empty | Tidak ada SFP terpasang | - |

- Data via SNMP (OID brand-specific untuk SFP diagnostics)
- Alert ke Teknisi jika SFP degraded atau failed
- Membantu diagnosa masalah **sebelum** pelanggan komplain

---

## ONT Pindah Port (Port Migration Detection)

Jika teknisi memindah ONT dari satu port ke port lain (misal karena port penuh atau maintenance):

```
Sync mendeteksi:
  ONT ZTEG12345678 (PLG-001)
    Database: port 0/1:1
    OLT aktual: port 0/3:5
  │
  ▼
Sistem mendeteksi "Port Migration":
  ├── Notifikasi ke admin: "ONT {SN} pindah dari port 0/1:1 ke 0/3:5"
  ├── Opsi:
  │     ○ Auto-update database (terima port baru)
  │     ○ Tanya admin dulu (default)
  │
  ▼
Jika auto-update aktif:
  ├── Update port di database
  ├── Update ODP assignment (jika port baru = ODP berbeda)
  └── Log: "ONT {SN} port migration: 0/1:1 → 0/3:5"
```

- Default: **tanya admin dulu** (notifikasi, admin konfirmasi)
- Bisa diset auto-update di Settings > OLT > Auto Port Migration
- Penting karena port migration bisa juga berarti ONT dicuri/dipindah tanpa izin

---

## Kapasitas Planning

Dashboard untuk membantu ISP merencanakan pertumbuhan jaringan:

```
┌────────────────────────────────────────────────────────────────┐
│ Kapasitas Jaringan — OLT-01 Pusat                              │
│                                                                │
│ ┌──────────────┬──────────────┬──────────────┬───────────────┐ │
│ │ PON Ports    │ ONT Total    │ ONT Sisa     │ Estimasi      │ │
│ │ 6/8 aktif    │ 245/512      │ 267 slot     │ ~8 bulan lagi │ │
│ │ 2 kosong     │ (48% terisi) │              │ (33 plgn/bln) │ │
│ └──────────────┴──────────────┴──────────────┴───────────────┘ │
│                                                                │
│ Kapasitas per PON Port:                                        │
│ Port 0/1: ████████████████████████████████ 64/64 (100%) ⛔ FULL│
│ Port 0/2: ██████████████████████████████░░ 58/64 (91%)  ⚠️    │
│ Port 0/3: ██████████████████████░░░░░░░░░ 45/64 (70%)         │
│ Port 0/4: ████████████████░░░░░░░░░░░░░░░ 38/64 (59%)         │
│ Port 0/5: ██████████░░░░░░░░░░░░░░░░░░░░░ 25/64 (39%)         │
│ Port 0/6: ██████░░░░░░░░░░░░░░░░░░░░░░░░░ 15/64 (23%)         │
│ Port 0/7: ░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░  0/64 (0%)          │
│ Port 0/8: 🔴 Down                                              │
│                                                                │
│ ODP Capacity:                                                  │
│ ODP-01-A (1:8):  ███████░ 7/8  (87%) ⚠️                      │
│ ODP-01-B (1:16): ██████████░░░░░░ 10/16 (63%)                 │
│ ODP-02-A (1:8):  ███░░░░░ 3/8  (38%)                          │
│                                                                │
│ Rekomendasi:                                                   │
│ ⚠️ Port 0/1 sudah penuh. Pelanggan baru harus ke port lain.  │
│ ⚠️ ODP-01-A hampir penuh. Pertimbangkan pasang ODP baru.     │
│ ℹ️ Dengan growth rate 33 pelanggan/bulan, OLT ini cukup      │
│    untuk ~8 bulan ke depan.                                    │
│                                                                │
│ [Export PDF]  [Export Excel]                                    │
└────────────────────────────────────────────────────────────────┘
```

- **Growth rate** dihitung dari rata-rata pelanggan baru per bulan (3 bulan terakhir)
- **Estimasi kapasitas** = sisa slot / growth rate per bulan
- Alert ke admin jika:
  - PON port > 90% terisi
  - ODP > 80% terisi
  - Estimasi kapasitas < 3 bulan
- Detail lengkap di dokumen **11 — Reporting & Analytics**

---

## Troubleshooting Guide (In-Dashboard Help)

Panel bantuan untuk teknisi di setiap halaman OLT:

| Masalah | Kemungkinan Penyebab | Solusi |
|---|---|---|
| ONT tidak terdeteksi (unregistered) | Kabel fiber putus, konektor lepas, ONT mati | Cek kabel & konektor, cek power ONT, cek SFP module |
| ONT terdeteksi tapi signal lemah | Jarak terlalu jauh, bending kabel, konektor kotor | Cek jarak (max ~20km GPON), bersihkan konektor, cek jalur kabel |
| Provisioning gagal | Command salah, profile tidak ada, VLAN salah | Cek adapter brand, pastikan profile sudah dibuat, cek VLAN config |
| ONT sering offline (flapping) | Power tidak stabil (PLN), kabel longgar | Cek power supply ONT, kencangkan konektor, cek UPS pelanggan |
| Signal tiba-tiba turun drastis | Kabel tertekuk/tergigit, konektor rusak | Inspeksi jalur kabel, ganti konektor, OTDR test |
| OLT tidak bisa dihubungi | VPN putus, OLT mati, IP berubah | Cek VPN status, cek power OLT, cek IP config |
| SNMP timeout | Community string salah, firewall block, SNMP disabled | Cek SNMP config di OLT, cek firewall, test dari server |
| Alarm LOS massal (banyak ONT sekaligus) | SFP module rusak, PON port down, kabel backbone putus | Cek SFP, cek port status, cek kabel backbone ke ODP |

- Tampilkan sebagai **collapsible help panel** di sidebar kanan
- Kontekstual per halaman (alarm page → troubleshooting alarm, ONT page → troubleshooting ONT)

---

## Sinkronisasi OLT ↔ Database

### Periodic Sync

```
Cron job setiap 30 menit:
  │
  ▼
Untuk setiap OLT online:
  ├── Ambil semua ONT dari OLT (via SNMP/CLI)
  ├── Bandingkan dengan data di database ISPBoss
  │
  ├── ONT ada di OLT tapi tidak di DB:
  │     → Tandai sebagai "Unmanaged"
  │     → Tampilkan di tab Unregistered
  │
  ├── ONT ada di DB tapi tidak di OLT:
  │     → Tandai sebagai "Missing"
  │     → Mungkin ONT dicabut atau OLT di-reset
  │
  ├── ONT ada di keduanya tapi data berbeda:
  │     → Update DB sesuai data OLT (OLT = source of truth untuk data fisik)
  │
  └── Semua cocok → status "Synced" ✅
```

> **Catatan:** Untuk OLT, **OLT = source of truth** untuk data fisik (SN, port, signal). Berbeda dengan MikroTik di mana database = source of truth. Alasannya: ONT adalah perangkat fisik yang bisa dipindah/dicabut tanpa melalui ISPBoss.

---

## Koneksi ke OLT (Connection Strategy)

Berbeda dengan MikroTik yang pakai connection pool persistent, OLT menggunakan **Connect on Demand**:

| Aspek | MikroTik | OLT |
|---|---|---|
| Frekuensi perintah | Tinggi (isolir, buka isolir, sync tiap 15 menit) | Rendah (provisioning jarang, monitoring via SNMP) |
| Protokol perintah | RouterOS API (persistent-friendly) | SSH/Telnet (session-based, tidak cocok persistent) |
| Monitoring | Via API (butuh koneksi) | Via SNMP (protokol terpisah, stateless) |
| Strategy | Connection Pool + Lazy Connect | Connect on Demand + SNMP Polling |

```
Perintah CLI ke OLT (provisioning, decommission, reboot):
  → Buka SSH/Telnet session
  → Login
  → Kirim command
  → Tunggu response
  → Tutup session

Monitoring OLT:
  → SNMP polling (stateless, tidak perlu session)
  → SNMP trap receiver (push, listen terus)
```

- SSH/Telnet session **tidak di-pool** (session-based, bukan connection-based)
- Setiap perintah buka session baru → kirim command → tutup
- SNMP polling berjalan terpisah (goroutine per OLT, interval configurable)
- SNMP trap receiver berjalan terus (1 goroutine, listen port 162)

---

## Keamanan

### Keamanan Koneksi
- Credential OLT disimpan **terenkripsi AES-256** di database
- Koneksi via **VPN tunnel** (sama seperti MikroTik)
- **SSH direkomendasikan** daripada Telnet (Telnet tidak terenkripsi)
- SNMP v3 direkomendasikan (auth + encryption)
- Jika harus pakai Telnet/SNMP v2c → pastikan via VPN (traffic terenkripsi di level VPN)

### Keamanan Development
- Default `NETWORK_MODE=mock` — tidak pernah konek ke OLT production saat development
- Mock adapter mengembalikan response simulasi per brand
- Tidak ada OLT virtual seperti MikroTik CHR → mock saja untuk testing

### Audit Trail
Semua perintah ke OLT dicatat:
```
┌──────────────────┬──────────────────────────────────┬──────────────┐
│ Waktu            │ Perintah                         │ Oleh         │
├──────────────────┼──────────────────────────────────┼──────────────┤
│ 28/04/26 14:30   │ onu add sn ZTEG99887766          │ Teknisi Andi │
│ 28/04/26 14:25   │ show gpon onu state 0/1          │ System (sync)│
│ 28/04/26 10:00   │ onu delete 0/2:1                 │ System (term)│
└──────────────────┴──────────────────────────────────┴──────────────┘
```

---

## Integrasi dengan Modul Lain

| Modul | Integrasi |
|---|---|
| **Pelanggan (04)** | Assign ONT ke pelanggan, lihat signal di tab Network |
| **Paket (05)** | Service profile OLT dari paket (opsional) |
| **Billing (06)** | Decommission ONT saat pelanggan berhenti |
| **Notifikasi (07)** | Alarm OLT → notifikasi ke Teknisi |
| **MikroTik (08)** | Pelanggan FTTH: OLT untuk fiber, MikroTik untuk bandwidth. VPN tunnel shared |
| **FTTH Mapping (10)** | Visualisasi OLT, ODP, ONT di peta |
| **Laporan (11)** | Signal quality report, alarm history, ONT uptime |
| **Settings (12)** | Signal threshold, health check interval, SNMP config |

---

## Keputusan

| Keputusan | Detail |
|---|---|
| Brand support | ZTE, Huawei, FiberHome, VSOL, HSGQ — adapter pattern per brand |
| Auto-detect | ✅ Brand, model, firmware, PON ports, total ONT via SNMP sysDescr |
| Protokol monitoring | SNMP v2c/v3 (polling) + SNMP trap (push) |
| Protokol provisioning | SSH (rekomendasi) atau Telnet via CLI command |
| Connection strategy | Connect on Demand untuk CLI, SNMP polling untuk monitoring |
| Koneksi | Via VPN tunnel ISPBoss (sama seperti MikroTik) |
| Credential | Terenkripsi AES-256, SNMP v3 direkomendasikan |
| ODP management | ✅ CRUD ODP/splitter, tipe 1:4/1:8/1:16/1:32, kapasitas tracking, GPS untuk peta |
| Provisioning | ✅ Add/remove ONT dari dashboard, auto-detect unregistered ONT |
| Bulk provisioning | ✅ CSV upload untuk migrasi/pasang massal, validasi sebelum eksekusi |
| Auto-provisioning | ✅ Opsional — ONT baru auto-provision jika SN sudah terdaftar di pelanggan |
| VLAN management | ✅ 4 strategy: single (default), per paket, per ODP, per pelanggan. VLAN translation |
| SFP monitoring | ✅ TX/RX power, suhu per PON port. Alert jika degraded/failed |
| ONT port migration | ✅ Deteksi ONT pindah port, notifikasi admin, auto-update opsional |
| Signal monitoring | ✅ RX power (dBm), threshold configurable, alert jika weak/critical |
| Signal history | ✅ Grafik 7 hari per ONT, data di Redis 30 hari |
| Alarm | ✅ SNMP trap + polling, 6 tipe alarm, notifikasi ke Teknisi, history 90 hari |
| PON port monitoring | ✅ Status, ONT count, traffic per port, kapasitas warning |
| ONT detail | ✅ Info, signal history, traffic, model, firmware, distance, last down |
| Unregistered ONT | ✅ Auto-detect ONT baru, provisioning langsung dari dashboard |
| Decommission | ✅ Otomatis saat pelanggan berhenti (via event), atau manual |
| Sinkronisasi | ✅ Periodic sync 30 menit, OLT = source of truth untuk data fisik |
| Health check | ✅ SNMP ping setiap 5 menit, 3x gagal → offline, notifikasi |
| Kapasitas planning | ✅ Sisa slot per port & ODP, growth rate, estimasi bulan, rekomendasi |
| Troubleshooting guide | ✅ In-dashboard help panel, kontekstual per halaman |
| ONT firmware | ❌ Fase lanjutan — tracking versi firmware ONT, bulk update remote |
| Reboot ONT | ✅ Via OLT command, dari dashboard |
| Mobile layout | ✅ Card list untuk daftar OLT dan ONT |
| Development | Mock adapter per brand, tidak ada OLT virtual |
| Referensi implementasi | Repo `snmp-zte` (github.com/ardani17/snmp-zte) — 71 endpoint ZTE C320, SNMP + CLI sudah ditest |
| Hybrid system | ✅ SNMP untuk monitoring, CLI untuk provisioning/VLAN/service-port. Berdasarkan riset OID |
| OID verified | ✅ 598 OID ZTE C320 sudah ditemukan dan didokumentasikan di repo riset |
| CLI status | SNMP sudah ditest works, CLI perlu testing lanjutan per command |
| Audit trail | ✅ Semua perintah ke OLT dicatat (append-only) |