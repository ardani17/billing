# Requirements Document

## Introduction

Spec ini menutup gap audit pertama untuk area umum project, dengan pengecualian MikroTik, OLT, dan Map. Fokusnya adalah production readiness, verifikasi frontend, settings non-finance, permission matrix, smoke test UI, dan sinkronisasi status dokumen.

Bagian keuangan detail ditangani pada spec terpisah: `.kiro/specs/financial-completion`.

## Requirements

### Requirement 1: Frontend build readiness

**User Story:** Sebagai maintainer, saya ingin frontend bisa dibuild secara konsisten agar semua halaman dapat diverifikasi sebelum production.

#### Acceptance Criteria

1. WHEN dependency frontend belum tersedia THEN dokumentasi SHALL menjelaskan command install yang benar.
2. WHEN dependency sudah tersedia THEN `npm.cmd --workspace @ispboss/web run build` SHALL berhasil.
3. WHEN build gagal THEN error SHALL diklasifikasikan sebagai dependency, type error, lint issue, atau runtime import issue.
4. WHEN build berhasil THEN hasilnya SHALL dicatat pada dokumen audit atau release note.

### Requirement 2: Settings page completion audit

**User Story:** Sebagai tenant admin, saya ingin seluruh halaman settings yang terlihat di navigasi benar-benar live atau diberi status yang jelas.

#### Acceptance Criteria

1. WHEN admin membuka settings users THEN sistem SHALL memakai data live dan action live.
2. WHEN admin membuka settings payment THEN sistem SHALL memakai gateway settings live.
3. WHEN admin membuka settings notifications THEN sistem SHALL memakai konfigurasi notification live.
4. WHEN admin membuka settings security THEN sistem SHALL memakai konfigurasi security live.
5. WHEN admin membuka settings branding THEN sistem SHALL memakai konfigurasi branding live.
6. WHEN halaman settings belum punya endpoint persistence THEN UI SHALL menampilkan status yang jelas atau spec implementasi SHALL dibuat.

### Requirement 3: Permission matrix for sensitive actions

**User Story:** Sebagai owner, saya ingin action sensitif hanya bisa dilakukan role yang benar agar data operasional aman.

#### Acceptance Criteria

1. WHEN user tanpa permission mengakses user management THEN sistem SHALL menolak request.
2. WHEN user tanpa permission mengubah settings THEN sistem SHALL menolak request.
3. WHEN user tanpa permission menjalankan action destruktif THEN sistem SHALL menolak request.
4. WHEN permission ditolak THEN response API SHALL konsisten dan UI SHALL menampilkan pesan yang jelas.
5. WHEN permission matrix selesai THEN dokumen role/action SHALL tersedia.

### Requirement 4: Core UI smoke test

**User Story:** Sebagai maintainer, saya ingin workflow utama bisa diuji dari browser agar implementasi tidak hanya lolos secara backend.

#### Acceptance Criteria

1. WHEN smoke test dijalankan THEN login dan session SHALL berhasil.
2. WHEN smoke test dijalankan THEN dashboard SHALL render tanpa error.
3. WHEN smoke test dijalankan THEN customer list/detail/create/edit SHALL bisa diakses.
4. WHEN smoke test dijalankan THEN package list/create/edit SHALL bisa diakses.
5. WHEN smoke test dijalankan THEN notification page SHALL bisa diakses.
6. WHEN smoke test dijalankan THEN report page SHALL bisa diakses.

### Requirement 5: Notification integration verification

**User Story:** Sebagai admin operasional, saya ingin notification reminder benar-benar terhubung dengan billing dan template agar pesan otomatis dapat dipercaya.

#### Acceptance Criteria

1. WHEN invoice reminder dijalankan THEN sistem SHALL membuat notification job/event sesuai channel.
2. WHEN template notification dipilih THEN sistem SHALL memakai template yang benar.
3. WHEN channel tidak aktif THEN sistem SHALL tidak mengirim pesan dan SHALL menampilkan status yang jelas.
4. WHEN pengiriman gagal THEN sistem SHALL mencatat failure reason.
5. WHEN pengiriman berhasil THEN sistem SHALL mencatat delivery status.

### Requirement 6: Diskusi status synchronization

**User Story:** Sebagai tim pengembang, saya ingin status dokumen diskusi sesuai implementasi nyata agar pekerjaan berikutnya tidak salah prioritas.

#### Acceptance Criteria

1. WHEN fitur hanya ada backend THEN dokumen SHALL menandainya sebagai backend done, UI pending.
2. WHEN fitur sudah usable dari UI THEN dokumen SHALL menandainya sebagai usable.
3. WHEN fitur berada di luar scope saat ini THEN dokumen SHALL menandainya sebagai deferred.
4. WHEN audit selesai THEN dokumen audit SHALL menautkan spec lanjutan yang relevan.
