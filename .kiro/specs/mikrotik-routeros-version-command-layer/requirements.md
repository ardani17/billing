# Requirements - MikroTik RouterOS Version Command Layer

## Overview

Spec ini mengatur pemisahan command RouterOS v6 dan v7 di dalam modul MikroTik tanpa memecah business module menjadi dua aplikasi. Router testing saat ini adalah RouterOS v6, sehingga semua perubahan harus menjaga v6 sebagai baseline aman.

## Requirements

### 1. Version Detection

1.1 Sistem harus bisa membaca major version RouterOS dari string seperti `6.49.18 (long-term)` dan `7.14.3`.

1.2 Sistem harus memiliki status versi minimal: `v6`, `v7`, dan `unknown`.

1.3 Versi kosong, malformed, atau unknown harus fallback ke command v6 untuk operasi eksekusi router.

1.4 Helper lama `IsRouterOSv7` harus tetap tersedia agar kode existing tidak rusak, tetapi implementasinya harus memakai parser versi baru.

### 2. Capability Layer

2.1 Sistem harus memiliki capability object/fungsi untuk menjawab fitur RouterOS berdasarkan versi.

2.2 Capability minimal harus mencakup `SupportsWireGuard`.

2.3 Capability harus default konservatif: unknown dianggap tidak mendukung fitur v7-only.

2.4 Usecase tidak boleh menyebar banyak `strings.HasPrefix(version, "7")`; keputusan versi harus lewat helper/capability.

### 3. PPPoE Command Builder

3.1 PPPoE harus tetap memakai `CommandBuilder` berdasarkan versi router.

3.2 RouterOS v6 testing harus tetap memakai command dan parameter yang kompatibel dengan v6.

3.3 RouterOS v7 harus memiliki tempat override eksplisit untuk command/parameter yang berbeda.

3.4 Test harus mencakup versi v6 real testing: `6.49.18 (long-term)`.

### 4. VPN Command Builder

4.1 VPN command builder harus menjadi version-aware.

4.2 WireGuard harus tetap ditolak untuk router v6 dan unknown.

4.3 Command VPN non-WireGuard harus tetap kompatibel dengan v6 kecuali ada bukti command berbeda.

4.4 Script generator VPN harus memakai capability/version context saat menghasilkan script yang bergantung pada versi.

### 5. Raw Command Migration

5.1 Command raw untuk operasi write harus diprioritaskan masuk ke command layer.

5.2 Prioritas write command: DHCP lease, static IP address-list/queue, hotspot user, walled garden firewall, backup/export lifecycle.

5.3 Read-only operational command boleh tetap raw sementara, tetapi harus dicatat sebagai backlog command catalog.

5.4 Route publik `/api/v1/mikrotik/...` tidak boleh berubah.

### 6. Router Metadata Freshness

6.1 `Create` dengan `TestOnCreate` dan `TestConnection` harus tetap menyimpan `RouterOSVersion`.

6.2 Health checker sukses harus menyegarkan metadata versi/router secara best-effort agar upgrade v6 ke v7 terdeteksi tanpa test manual.

6.3 Kegagalan update metadata saat health check tidak boleh membuat health check dianggap gagal jika resource berhasil dibaca.

### 7. Verification

7.1 `go test ./...` di `services/network-service` harus lulus.

7.2 Test parser versi harus mencakup v6, v7, string dengan suffix, string dengan prefix text, whitespace, dan unknown.

7.3 Test command builder harus membuktikan v6 fallback aman.

7.4 Jika command v7 belum berbeda untuk sebuah fitur, test harus menyatakan v7 intentionally inherits v6 behavior.
