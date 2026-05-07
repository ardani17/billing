# Requirements - MikroTik Module Restructure

## Overview

Spec ini mengatur pemindahan kode MikroTik backend ke folder modul khusus tanpa mengubah behavior aplikasi. Tujuannya memperbaiki debug path dan ownership kode sebelum fitur MikroTik bertambah jauh.

## Requirements

### 1. Folder Boundary

1.1 Sistem harus memiliki folder khusus untuk kode MikroTik di `services/network-service/internal/modules/mikrotik`.

1.2 Folder MikroTik harus memuat package untuk adapter RouterOS, usecase MikroTik, handler MikroTik, dan worker PPPoE.

1.3 Kode OLT, provisioning, dan FTTH mapping tidak boleh dipindahkan dalam fase MikroTik pertama.

### 2. Runtime Compatibility

2.1 Public API route `/api/v1/mikrotik/...` harus tetap sama.

2.2 Next.js proxy di `apps/web/app/api/network/mikrotik` tidak boleh perlu perubahan route.

2.3 `cmd/main.go` harus tetap bisa bootstrap MikroTik, OLT, provisioning, map, VPN, dan workers.

2.4 Module guard `domain.ModuleMikroTik` harus tetap melindungi route MikroTik.

### 3. RouterOS Behavior

3.1 RouterOS live/mock adapter harus mempertahankan interface `domain.RouterOSAdapter`.

3.2 Command builder RouterOS v6/v7 harus tetap memakai behavior existing.

3.3 Pool manager harus tetap menerima factory adapter tanpa perubahan kontrak domain.

### 4. PPPoE and Worker Safety

4.1 Semua file PPPoE manager harus tetap satu package agar receiver dan helper unexported tidak pecah.

4.2 PPPoE event worker harus tetap register event yang sama.

4.3 Retry behavior worker tidak boleh berubah.

4.4 PPPoE sync, CRUD, sessions, isolir, unisolir, suspend, package change, dan profile sync tidak boleh berubah behavior.

### 5. Repository and SQLC Safety

5.1 `internal/repository`, `queries`, `migrations`, dan generated `*.sql.go` tidak boleh dipindahkan pada fase pertama.

5.2 `sqlc.yaml` tidak boleh diubah pada fase pertama.

5.3 Domain interfaces boleh tetap berada di `internal/domain` pada fase pertama.

### 6. Test and Verification

6.1 Test package yang terkait file pindahan harus ikut pindah.

6.2 `go list ./...` harus lulus setelah setiap fase besar.

6.3 `go test ./...` di `services/network-service` harus lulus sebelum pekerjaan dianggap selesai.

6.4 Jika frontend diverifikasi, build web boleh dijalankan, tetapi kegagalan unrelated harus dicatat terpisah.
