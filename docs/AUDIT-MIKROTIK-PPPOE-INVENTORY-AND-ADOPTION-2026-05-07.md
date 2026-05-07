# Audit dan Spec: MikroTik PPPoE Inventory dan Adoption

Tanggal: 2026-05-07
Repo: `C:\laragon\www\billing`
Scope: modul MikroTik, khusus PPPoE user list, sync, dan pemisahan user ISPBoss vs user manual RouterOS.

## Ringkasan Keputusan

Web saat ini hanya menampilkan PPPoE user dari tabel `pppoe_users`, sehingga user yang dibuat langsung di MikroTik tidak muncul di menu PPPoE. Ini bukan bug rendering, melainkan batas kontrak API saat ini: endpoint list bersifat database-only.

Solusi yang aman bukan memasukkan semua secret router ke tabel `pppoe_users` secara otomatis. Tabel saat ini mewajibkan `customer_id` dan `password_encrypted`, sedangkan user manual di MikroTik tidak membawa data pelanggan dan password tidak bisa dibaca ulang dengan aman. Jika dipaksa masuk ke DB, modul billing bisa keliru mengisolir, membuka isolir, mengubah profile, atau menghapus user yang sebenarnya dibuat manual.

Implementasi yang direkomendasikan adalah menambah mode `PPPoE Inventory` yang menggabungkan data DB dan live RouterOS secara read-only, memberi label asal/ownership, lalu menyediakan aksi `Adopt` yang eksplisit dan aman untuk user manual yang ingin dijadikan managed by ISPBoss.

## Audit Kondisi Saat Ini

### 1. Jalur UI list PPPoE

- UI detail router memuat PPPoE user dari `GET /api/network/mikrotik/routers/:routerId/pppoe/users?page_size=50`.
- Komponen yang menampilkan list adalah `PppoeUsersPanel`, dengan judul `PPPoE users terkelola`.
- Data type frontend `PPPoEUser` hanya punya field DB: `id`, `username`, `profile_name`, `remote_address`, `disabled`, `status`, `sync_status`, `last_sync_at`.
- Tidak ada field untuk membedakan `managed`, `manual_router`, `missing_router`, atau `orphan_ispboss`.
- Tombol aksi `Disable`, `Disconnect`, dan `Hapus` diasumsikan aman karena semua row berasal dari DB.

Kesimpulan: UI saat ini benar untuk "managed users", tetapi belum cocok untuk "semua user yang ada di MikroTik".

### 2. Jalur API list PPPoE

- Next.js API route meneruskan request ke network-service endpoint `/api/v1/mikrotik/routers/:id/pppoe/users`.
- Handler `ListUsers` di network-service memanggil `manager.ListUsers`.
- Usecase `ListUsers` hanya mengisi `params.RouterID` lalu memanggil `userRepo.List`.
- Query `ListPPPoEUsers` hanya membaca tabel `pppoe_users WHERE router_id = $1 AND deleted_at IS NULL`.

Kesimpulan: jumlah user di web mengikuti jumlah row DB, bukan jumlah secret di RouterOS.

### 3. Jalur sync PPPoE

- `SyncRouter` membaca live `/ppp/secret/print`, lalu membaca DB melalui `GetByRouterID`.
- `GetByRouterID` hanya mengambil row `deleted_at IS NULL AND status = 'active'`.
- Secret router tanpa comment `ISPBoss:` dihitung sebagai orphan dan dilewati.
- Secret router dengan comment `ISPBoss:` tetapi tidak ada di DB juga dihitung sebagai orphan dan dilewati.
- DB user aktif yang hilang di router dihitung missing dan dibuat ulang di router.
- DB user aktif yang berbeda profile/disabled akan diperbaiki ke versi DB.

Kesimpulan: sync adalah operasi mutating. Ia cocok untuk menjaga user managed by ISPBoss, tetapi tidak boleh dijadikan mekanisme list semua user manual.

### 4. Status sync di UI

- UI memuat `GET /pppoe/sync-status`.
- Backend `GetSyncStatusSummary` hanya agregasi `sync_status` dari DB.
- `orphan_count` tidak dihitung dari live router pada endpoint ini.
- Karena itu panel bisa menampilkan orphan 0 meski router memiliki user manual.

Kesimpulan: sync-status saat ini adalah status DB, bukan audit live router. Untuk kebutuhan inventory, perlu summary live/read-only atau endpoint inventory yang mengembalikan summary.

### 5. Constraint tabel `pppoe_users`

Tabel `pppoe_users` memiliki constraint penting:

- `customer_id UUID NOT NULL`
- `password_encrypted TEXT NOT NULL`
- unique index `(router_id, username)` untuk row yang belum soft-delete
- `comment TEXT NOT NULL`
- RLS tenant aktif

Kesimpulan: user manual tidak bisa diadopsi tanpa keputusan admin tentang customer dan password. Password RouterOS existing tidak boleh diasumsikan tersedia.

### 6. Jalur billing lifecycle

Operasi billing yang menyentuh MikroTik bergantung pada data pelanggan dan PPPoE:

- Aktivasi pelanggan membuat secret di router, membuat comment `ISPBoss:{customer_id}:{tenant_id}`, lalu menyimpan row DB.
- Isolir melakukan disable secret, disconnect session, dan menambah firewall redirect.
- Unisolir melakukan enable secret dan menghapus rule isolir.
- Suspend/terminate melakukan disconnect, remove secret, remove queue/firewall, lalu soft-delete row DB.
- Ubah paket mengambil PPPoE user dari DB berdasarkan `customer_id`, mengubah profile di router, lalu update DB.

Kesimpulan: hanya user yang sudah punya ownership DB/customer yang boleh masuk ke lifecycle billing otomatis. User manual router harus read-only sampai diadopsi secara eksplisit.

### 7. Bukti live terakhir

Pada audit read-only melalui terminal endpoint `/ppp/secret/print`, DB memiliki 2 row PPPoE, sedangkan router mengembalikan lebih banyak secret live. Contoh yang terlihat:

- `ispboss-sync-test-20260504164443` dengan comment `ISPBoss:...`
- `ispboss-missing-test-20260504164650` dengan comment `ISPBoss:...`
- `onu1` tanpa comment ISPBoss
- `tes` tanpa comment ISPBoss

Catatan: user menyebut WinBox berisi 5 user; pembacaan API saat audit melihat 4 secret pada saat itu. Perbedaan ini perlu diverifikasi ulang saat implementasi dengan filter/proplist yang sama, tetapi akar masalah tetap sama: UI membaca DB-only.

## Risiko Konflik Jika Salah Implementasi

1. Auto-import manual user ke DB akan gagal atau menghasilkan data palsu karena `customer_id` dan password tidak tersedia.
2. Jika manual user diperlakukan sebagai managed, tombol `Hapus` dapat menghapus secret RouterOS yang tidak pernah dibuat aplikasi.
3. Event isolir/suspend dapat mengubah user manual bila username/customer mapping dipaksa tanpa proses adopsi.
4. Sync mutating dapat mengubah profile/disabled router ke nilai DB, sehingga inventory live harus dipisah dari sync repair.
5. Username sama antara DB dan router tetapi comment berbeda adalah kasus berisiko tinggi: perlu ditandai conflict, bukan langsung di-fix.
6. `GetByRouterID` hanya mengambil active users; disabled DB user bisa luput dari perbandingan sync. Spec baru harus jelas apakah inventory membaca semua non-deleted DB users atau hanya active.
7. Sync-status DB-only bisa memberi rasa aman palsu karena orphan live tidak terlihat.

## Ownership dan Status yang Disarankan

Inventory harus mengklasifikasikan setiap username per router ke satu status utama:

| Status | Kondisi | Aksi Aman |
| --- | --- | --- |
| `managed_synced` | Ada di DB dan router, comment ISPBoss cocok, profile/disabled cocok | Update, disable/enable, disconnect, delete |
| `managed_out_of_sync` | Ada di DB dan router, ownership cocok, tetapi profile/disabled/remote berbeda | Tampilkan diff, allow sync repair |
| `missing_on_router` | Ada di DB aktif, tidak ada di router | Recreate/sync dari DB |
| `manual_router` | Ada di router, tidak ada di DB, comment bukan ISPBoss | Read-only, allow adopt |
| `orphan_ispboss` | Ada di router, comment ISPBoss, tetapi tidak ada di DB | Read-only, allow recover/adopt after validation |
| `ownership_conflict` | Username sama, DB ada, router ada, tetapi comment ISPBoss mengarah ke customer/tenant berbeda atau comment bukan ISPBoss | Read-only sampai admin resolve |
| `db_disabled_present` | Ada di DB status disabled dan masih ada di router | Tampilkan disabled managed; jangan hilang dari inventory |

Aturan utama:

- Inventory tidak boleh mengubah router.
- Manual router user tidak boleh mendapat tombol delete/disable dari flow managed.
- Adopt harus action eksplisit, terkonfirmasi, dan diaudit.
- Suspend/isolir/package-change hanya boleh berjalan untuk user managed by ISPBoss.

## Spec Implementasi

### 1. Backend domain DTO

Tambahkan DTO baru di network-service untuk inventory:

```go
type PPPoEInventoryItem struct {
    ID              string            `json:"id"`
    RouterID        string            `json:"router_id"`
    Username        string            `json:"username"`
    ProfileName     string            `json:"profile_name"`
    Service         string            `json:"service"`
    RemoteAddress   string            `json:"remote_address,omitempty"`
    Disabled        bool              `json:"disabled"`
    Comment         string            `json:"comment,omitempty"`
    Source          string            `json:"source"`       // db, router, both
    Ownership       string            `json:"ownership"`    // ispboss, manual, conflict
    InventoryStatus string            `json:"inventory_status"`
    DBUserID        string            `json:"db_user_id,omitempty"`
    CustomerID      string            `json:"customer_id,omitempty"`
    RouterSecretID  string            `json:"router_secret_id,omitempty"`
    SyncStatus      SyncStatus        `json:"sync_status,omitempty"`
    MismatchFields  []string          `json:"mismatch_fields,omitempty"`
    Capabilities    map[string]bool   `json:"capabilities"`
}
```

Tambahkan summary:

```go
type PPPoEInventorySummary struct {
    Total             int `json:"total"`
    Managed          int `json:"managed"`
    ManualRouter     int `json:"manual_router"`
    MissingOnRouter  int `json:"missing_on_router"`
    OutOfSync        int `json:"out_of_sync"`
    OrphanISPBoss    int `json:"orphan_ispboss"`
    Conflict         int `json:"conflict"`
}
```

### 2. Backend usecase

Tambah method di `PPPoEManager`:

```go
ListInventory(ctx context.Context, routerID string, params PPPoEInventoryListParams) (*PPPoEInventoryResult, error)
AdoptRouterSecret(ctx context.Context, routerID string, req AdoptPPPoESecretRequest) (*PPPoEUser, error)
```

Implementasi `ListInventory`:

- Ambil router dan koneksi pool dengan `PriorityLow`.
- Read-only command ke RouterOS: `/ppp/secret/print`.
- Gunakan proplist minimal: `.id,name,profile,service,disabled,comment,remote-address`.
- Ambil semua DB user non-deleted untuk router, bukan hanya active.
- Merge by `username`.
- Parse comment dengan `ParseComment` jika prefix `ISPBoss:`.
- Buat classification menggunakan pure function, misalnya `BuildPPPoEInventory(routerSecrets, dbUsers)`.
- Return paginated/filterable result.

Implementasi `AdoptRouterSecret`:

- Input minimal: `router_secret_id` atau `username`, `customer_id`, `password`, optional `profile_name`, `remote_address`, `use_simple_queue`.
- Validasi username masih ada di router.
- Validasi belum ada row DB aktif dengan `(router_id, username)`.
- Karena password existing RouterOS tidak bisa dipercaya tersedia, adopsi awal harus mensyaratkan password baru.
- Mutasi RouterOS eksplisit:
  - set `password`
  - set `comment = ISPBoss:{customer_id}:{tenant_id}`
  - set `service = pppoe` bila diperlukan
  - set `profile` bila admin memilih profile
- Setelah router update sukses, create row `pppoe_users` dengan `sync_status = synced`.
- Publish/audit command result dengan operation `adopt`.

Adopt tanpa reset password tidak masuk fase pertama karena tabel membutuhkan `password_encrypted` yang benar untuk future recreate/sync.

### 3. API route network-service

Tambah route:

- `GET /api/v1/mikrotik/routers/:id/pppoe/inventory`
- `POST /api/v1/mikrotik/routers/:id/pppoe/adoptions`

Query inventory:

- `page`
- `page_size`
- `search`
- `status`
- `source`
- `ownership`

Response:

```json
{
  "success": true,
  "data": {
    "items": [],
    "summary": {},
    "page": 1,
    "page_size": 50,
    "total": 0,
    "total_pages": 0
  }
}
```

### 4. Next.js proxy route

Tambah:

- `apps/web/app/api/network/mikrotik/routers/[id]/pppoe/inventory/route.ts`
- `apps/web/app/api/network/mikrotik/routers/[id]/pppoe/adoptions/route.ts`

Proxy ini meneruskan auth/session seperti route PPPoE existing.

### 5. UI detail MikroTik

Ubah section PPPoE menjadi inventory:

- Judul: `PPPoE Inventory`
- Summary chips: Total, Managed, Manual, Missing, Out of sync, Conflict.
- Filter: All, Managed, Manual Router, Missing, Out of Sync, Conflict.
- Kolom: Username, Source, Profile, Remote IP, Router status, Sync, Aksi.

Badge source:

- `ISPBoss`
- `Manual Router`
- `Missing`
- `Out of sync`
- `Conflict`

Capability rules:

- `managed_synced`: show Disable/Enable, Disconnect, Hapus.
- `managed_out_of_sync`: show Sync Repair, optionally Disable/Disconnect/Hapus if ownership valid.
- `missing_on_router`: show Recreate from DB.
- `manual_router`: show Adopt only; no Disable/Hapus from managed flow.
- `orphan_ispboss`: show Recover/Adopt; no destructive action until resolved.
- `ownership_conflict`: show detail; no mutating action.

Adopt modal:

- Show router username, router profile, comment, disabled state.
- Require customer selection.
- Require password/new password.
- Optional profile override.
- Explain that after adoption, billing automation may manage this PPPoE user.

### 6. Sync status panel

Do not reuse DB-only `sync-status` as live truth.

Option recommended:

- Inventory endpoint returns `summary` live/read-only.
- Sync panel can show both:
  - DB sync status from existing endpoint
  - Live inventory summary from new endpoint

If UI needs one source of truth, prefer inventory summary for counts visible to operator.

### 7. Database strategy

Fase pertama tidak membutuhkan migration baru.

Alasan:

- Manual router user tetap ephemeral/read-only.
- Adoption membuat row valid di tabel existing setelah admin memberi `customer_id` dan password.
- Tidak mengubah nullability `customer_id` atau `password_encrypted`, sehingga risiko migration produksi lebih rendah.

Migration baru hanya diperlukan jika nanti ingin fitur `ignore manual user`, `link-only tanpa password`, atau riwayat inventory snapshot.

### 8. Test plan

Backend unit tests:

- `BuildPPPoEInventory` mengembalikan semua username unik tanpa duplikasi.
- Manual no-comment menjadi `manual_router`.
- Router-only comment ISPBoss menjadi `orphan_ispboss`.
- DB-only active menjadi `missing_on_router`.
- DB+router ownership cocok dan data cocok menjadi `managed_synced`.
- DB+router ownership cocok tapi profile/disabled berbeda menjadi `managed_out_of_sync`.
- Username sama tetapi comment customer/tenant berbeda menjadi `ownership_conflict`.
- Disabled DB user tetap muncul di inventory.

Handler tests:

- `GET /inventory` tidak memanggil command mutating.
- Filter/search/status berjalan.
- `POST /adoptions` menolak password kosong.
- `POST /adoptions` menolak username yang sudah ada di DB.
- `POST /adoptions` menolak secret yang sudah hilang di router.

Frontend tests/manual QA:

- Router dengan 2 DB user dan 2 manual secret menampilkan 4 item.
- Manual user hanya memiliki tombol Adopt.
- Managed user tetap memiliki Disable/Disconnect/Hapus.
- Sync summary tidak lagi menampilkan orphan 0 bila router punya manual user.
- Adopt mengubah user menjadi managed dan list refresh tanpa duplikasi.

Regression:

- `go test ./...` di `services/network-service`
- `npm.cmd --workspace @ispboss/web run build`

Catatan build saat audit terakhir: build web pernah gagal pada page unrelated `/resellers/page` dan `/super-admin/upgrade-requests/page`. Kegagalan itu perlu dipisahkan dari verifikasi MikroTik.

## Rencana Pengerjaan Bertahap

### Fase 1 - Read-only inventory

1. Tambah DTO inventory dan pure classification function.
2. Tambah repository query untuk semua PPPoE user non-deleted by router.
3. Tambah usecase `ListInventory`.
4. Tambah handler dan route `/pppoe/inventory`.
5. Tambah Next.js proxy route.
6. Ubah UI PPPoE panel agar membaca inventory dan menampilkan badge source/status.
7. Tambah tests backend untuk classification.

Output fase 1: semua user RouterOS muncul di web, dengan pemisah aman, tanpa mutasi router.

### Fase 2 - Sync/status cleanup

1. Tampilkan live inventory summary di panel sync.
2. Jaga endpoint DB-only tetap ada untuk kompatibilitas.
3. Pertimbangkan rename copy UI agar operator tahu mana DB sync dan mana live inventory.
4. Tambah test agar orphan live tidak lagi hilang dari summary.

Output fase 2: operator melihat angka manual/orphan yang benar.

### Fase 3 - Adoption

1. Tambah request/response DTO adoption.
2. Tambah usecase `AdoptRouterSecret`.
3. Tambah handler route `/pppoe/adoptions`.
4. Tambah modal UI Adopt dengan customer dan password.
5. Pastikan adoption menulis comment ISPBoss dan row DB hanya setelah router update sukses.
6. Tambah tests validation dan happy path.

Output fase 3: user manual bisa dijadikan managed secara sadar dan aman.

### Fase 4 - Conflict resolution lanjutan

1. Tambah flow recover untuk `orphan_ispboss`.
2. Tambah flow resolve untuk `ownership_conflict`.
3. Pertimbangkan fitur ignore/hide manual user dengan tabel baru jika operator membutuhkan.

Output fase 4: kasus migrasi router lama bisa dibereskan tanpa operasi berbahaya.

## Acceptance Criteria

- Web PPPoE inventory menampilkan gabungan DB + live RouterOS.
- Setiap item memiliki label source/ownership yang jelas.
- User manual RouterOS tidak bisa dihapus/disable dari action managed.
- Endpoint inventory bersifat read-only dan tidak memanggil sync repair.
- Existing endpoint `/pppoe/users` tetap dapat dipakai sebagai list managed users agar kontrak lama tidak pecah.
- Sync mutating tetap hanya memperbaiki user managed by ISPBoss.
- Adoption membutuhkan customer dan password baru.
- Setelah adoption sukses, user muncul sebagai managed dan eligible untuk lifecycle billing.
