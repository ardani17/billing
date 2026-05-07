package domain

import "context"

// =============================================================================
// ONTRepository - operasi data untuk tabel onts
// =============================================================================

// ONTRepository mendefinisikan operasi data untuk tabel onts.
// Diimplementasikan oleh repositori.ONTRepo menggunakan sqlc.
type ONTRepository interface {
	// Buat membuat record ONT baru.
	Create(ctx context.Context, ont *ONT) (*ONT, error)

	// GetByID mengambil ONT berdasarkan ID (tenant-scoped via RLS).
	GetByID(ctx context.Context, id string) (*ONT, error)

	// GetBySerialNumber mengambil ONT berdasarkan tenant_id dan serial_number.
	GetBySerialNumber(ctx context.Context, tenantID, serialNumber string) (*ONT, error)

	// Perbarui memperbarui record ONT.
	Update(ctx context.Context, ont *ONT) (*ONT, error)

	// SoftDelete melakukan hapus lunak ONT (atur deleted_at).
	SoftDelete(ctx context.Context, id string) error

	// List mengambil daftar ONT dengan paginasi dan filter.
	List(ctx context.Context, params ONTListParams) (*ONTListResult, error)

	// ListByOLTAndStatus mengambil ONT berdasarkan olt_id dan status.
	ListByOLTAndStatus(ctx context.Context, oltID, status string) ([]*ONT, error)

	// GetByCustomerID mengambil ONT berdasarkan customer_id.
	GetByCustomerID(ctx context.Context, customerID string) (*ONT, error)

	// SerialNumberExists mengecek apakah serial number sudah ada di tenant.
	SerialNumberExists(ctx context.Context, tenantID, serialNumber, excludeID string) (bool, error)

	// PositionExists mengecek apakah posisi (olt_id, pon_port, ont_index) sudah terisi.
	PositionExists(ctx context.Context, oltID string, ponPort, ontIndex int, excludeID string) (bool, error)

	// UpdateStatus memperbarui status dan provisioning_state ONT.
	UpdateStatus(ctx context.Context, id string, status, provisioningState string) error

	// UpdatePortMigration memperbarui pon_port_index dan ont_index setelah migrasi.
	UpdatePortMigration(ctx context.Context, id string, newPort, newONTIndex int) error

	// DeleteUnregisteredByOLT menghapus ONT unregistered yang tidak lagi terdeteksi.
	DeleteUnregisteredByOLT(ctx context.Context, oltID string, keepSerialNumbers []string) (int64, error)
}

// =============================================================================
// VLANRepository - operasi data untuk tabel vlans
// =============================================================================

// VLANRepository mendefinisikan operasi data untuk tabel vlans.
// Diimplementasikan oleh repositori.VLANRepo menggunakan sqlc.
type VLANRepository interface {
	// Buat membuat VLAN baru.
	Create(ctx context.Context, vlan *VLAN) (*VLAN, error)

	// GetByID mengambil VLAN berdasarkan ID (tenant-scoped via RLS).
	GetByID(ctx context.Context, id string) (*VLAN, error)

	// Perbarui memperbarui data VLAN.
	Update(ctx context.Context, vlan *VLAN) (*VLAN, error)

	// SoftDelete melakukan hapus lunak VLAN (atur deleted_at).
	SoftDelete(ctx context.Context, id string) error

	// List mengambil daftar VLAN per OLT dengan paginasi.
	List(ctx context.Context, oltID string, params VLANListParams) (*VLANListResult, error)

	// GetByOLTAndVLANID mengambil VLAN berdasarkan olt_id dan vlan_id.
	GetByOLTAndVLANID(ctx context.Context, oltID string, vlanID int) (*VLAN, error)

	// GetDefaultVLAN mengambil VLAN bawaan untuk OLT (VLAN pertama tipe data).
	GetDefaultVLAN(ctx context.Context, oltID string) (*VLAN, error)

	// VLANIDExists mengecek apakah vlan_id sudah ada pada OLT yang sama.
	VLANIDExists(ctx context.Context, oltID string, vlanID int, excludeID string) (bool, error)

	// CountActiveONTs menghitung jumlah ONT aktif yang menggunakan VLAN ini.
	CountActiveONTs(ctx context.Context, vlanID string) (int64, error)
}

// =============================================================================
// ServiceProfileRepository - operasi data untuk tabel service_profiles
// =============================================================================

// ServiceProfileRepository mendefinisikan operasi data untuk tabel service_profiles.
// Diimplementasikan oleh repositori.ServiceProfileRepo menggunakan sqlc.
type ServiceProfileRepository interface {
	// Buat membuat service profile baru.
	Create(ctx context.Context, profile *ServiceProfile) (*ServiceProfile, error)

	// GetByID mengambil service profile berdasarkan ID (tenant-scoped via RLS).
	GetByID(ctx context.Context, id string) (*ServiceProfile, error)

	// Perbarui memperbarui data service profile.
	Update(ctx context.Context, profile *ServiceProfile) (*ServiceProfile, error)

	// SoftDelete melakukan hapus lunak service profile (atur deleted_at).
	SoftDelete(ctx context.Context, id string) error

	// List mengambil daftar service profile per OLT dengan paginasi.
	List(ctx context.Context, oltID string, params ServiceProfileListParams) (*ServiceProfileListResult, error)

	// GetByPackageAndOLT mengambil service profile berdasarkan package_id dan olt_id.
	GetByPackageAndOLT(ctx context.Context, oltID, packageID string) (*ServiceProfile, error)

	// ProfileExists mengecek apakah kombinasi profile sudah ada pada OLT.
	ProfileExists(ctx context.Context, oltID string, lineProfileID, serviceProfileID int, excludeID string) (bool, error)

	// CountActiveONTs menghitung jumlah ONT aktif yang menggunakan profile ini.
	CountActiveONTs(ctx context.Context, profileID string) (int64, error)
}

// =============================================================================
// AuditLogRepository - operasi data untuk tabel provisioning_audit_logs
// =============================================================================

// AuditLogRepository mendefinisikan operasi data untuk tabel provisioning_audit_logs.
// Append-only: tidak ada operasi Perbarui atau Hapus.
type AuditLogRepository interface {
	// Buat menyimpan record audit log baru.
	Create(ctx context.Context, log *ProvisioningAuditLog) (*ProvisioningAuditLog, error)

	// List mengambil daftar audit log dengan paginasi dan filter.
	List(ctx context.Context, params AuditLogListParams) (*AuditLogListResult, error)
}

// =============================================================================
// ProvisioningSettingsRepository - operasi data untuk tabel provisioning_settings
// =============================================================================

// ProvisioningSettingsRepository mendefinisikan operasi data untuk tabel provisioning_settings.
// Satu record per tenant, upsert untuk buat/perbarui.
type ProvisioningSettingsRepository interface {
	// GetByTenantID mengambil settings berdasarkan tenant_id.
	GetByTenantID(ctx context.Context, tenantID string) (*ProvisioningSettings, error)

	// Upsert membuat atau memperbarui settings untuk tenant.
	Upsert(ctx context.Context, settings *ProvisioningSettings) (*ProvisioningSettings, error)
}

// =============================================================================
// ProvisioningManager - business logic untuk provisioning ONT
// =============================================================================

// ProvisioningManager mendefinisikan business logic untuk provisioning ONT.
// Menangani single/bulk provisioning, decommission, reboot, auto-provisioning,
// port migration, dan audit trail.
type ProvisioningManager interface {
	// ProvisionONT melakukan provisioning satu ONT ke OLT.
	ProvisionONT(ctx context.Context, tenantID string, req ProvisionONTRequest) (*ONTResponse, error)

	// PreviewProvisionONT membangun preview command provisioning tanpa eksekusi ke OLT.
	PreviewProvisionONT(ctx context.Context, tenantID string, req ProvisionONTRequest) (*ProvisioningDryRun, error)

	// DecommissionONT menghapus ONT dari OLT dan perbarui DB.
	DecommissionONT(ctx context.Context, ontID string, performedBy string) error

	// RebootONT mengirim perintah reboot ke ONT via OLT CLI.
	RebootONT(ctx context.Context, ontID string, performedBy string) (*ProvisioningResult, error)

	// ValidateBulk memvalidasi CSV upload dan mengembalikan preview.
	ValidateBulk(ctx context.Context, tenantID string, oltID string, csvData []byte) (*BulkPreview, error)

	// ExecuteBulk mengeksekusi bulk provisioning untuk semua row valid.
	ExecuteBulk(ctx context.Context, bulkID string, performedBy string) (*BulkResult, error)

	// GetBulkTemplate mengembalikan CSV template untuk bulk provisioning.
	GetBulkTemplate() []byte

	// HandleUnregisteredONT memproses ONT unregistered yang terdeteksi sync engine.
	HandleUnregisteredONT(ctx context.Context, oltID string, ont UnregisteredONT) error

	// HandlePortMigration memproses deteksi port migration dari sync engine.
	HandlePortMigration(ctx context.Context, ontID string, oldPort, newPort, oldONTIdx, newONTIdx int) error

	// ConfirmMigration mengkonfirmasi port migration dan perbarui DB.
	ConfirmMigration(ctx context.Context, ontID string) error

	// HandleCustomerTerminated memproses event customer.terminated untuk decommission.
	HandleCustomerTerminated(ctx context.Context, customerID, tenantID string) error

	// GetONTByID mengambil detail ONT termasuk relasi.
	GetONTByID(ctx context.Context, id string) (*ONTDetailResponse, error)

	// ListONTs mengambil daftar ONT dengan paginasi dan filter.
	ListONTs(ctx context.Context, params ONTListParams) (*ONTListResult, error)

	// GetUnregisteredONTs mengambil daftar ONT unregistered untuk satu OLT.
	GetUnregisteredONTs(ctx context.Context, oltID string) ([]*ONTResponse, error)

	// GetAuditLogs mengambil daftar audit log dengan paginasi dan filter.
	GetAuditLogs(ctx context.Context, params AuditLogListParams) (*AuditLogListResult, error)

	// GetSettings mengambil provisioning settings untuk tenant.
	GetSettings(ctx context.Context, tenantID string) (*ProvisioningSettings, error)

	// UpdateSettings memperbarui provisioning settings untuk tenant.
	UpdateSettings(ctx context.Context, tenantID string, req UpdateSettingsRequest) (*ProvisioningSettings, error)
}

// =============================================================================
// VLANManager - business logic untuk manajemen VLAN per OLT
// =============================================================================

// VLANManager mendefinisikan business logic untuk manajemen VLAN per OLT.
// Menangani CRUD VLAN dan resolusi VLAN berdasarkan strategy saat provisioning.
type VLANManager interface {
	// Buat membuat VLAN baru untuk OLT.
	Create(ctx context.Context, tenantID string, req CreateVLANRequest) (*VLANResponse, error)

	// GetByID mengambil detail VLAN.
	GetByID(ctx context.Context, id string) (*VLANResponse, error)

	// Perbarui memperbarui data VLAN.
	Update(ctx context.Context, id string, req UpdateVLANRequest) (*VLANResponse, error)

	// Hapus hapus lunak VLAN (cek tidak ada ONT aktif yang menggunakan).
	Delete(ctx context.Context, id string) error

	// List mengambil daftar VLAN per OLT dengan paginasi.
	List(ctx context.Context, oltID string, params VLANListParams) (*VLANListResult, error)

	// ResolveVLAN menentukan VLAN berdasarkan strategy tenant saat provisioning.
	ResolveVLAN(ctx context.Context, oltID string, strategy VLANStrategy, resolveCtx VLANResolveContext) (*VLAN, error)
}

// =============================================================================
// ServiceProfileManager - business logic untuk manajemen service profile
// =============================================================================

// ServiceProfileManager mendefinisikan business logic untuk manajemen service profile.
// Menangani CRUD profile dan mapping ke paket ISPBoss.
type ServiceProfileManager interface {
	// Buat membuat service profile baru untuk OLT.
	Create(ctx context.Context, tenantID string, req CreateServiceProfileRequest) (*ServiceProfileResponse, error)

	// GetByID mengambil detail service profile.
	GetByID(ctx context.Context, id string) (*ServiceProfileResponse, error)

	// Perbarui memperbarui data service profile.
	Update(ctx context.Context, id string, req UpdateServiceProfileRequest) (*ServiceProfileResponse, error)

	// Hapus hapus lunak service profile (cek tidak ada ONT aktif yang menggunakan).
	Delete(ctx context.Context, id string) error

	// List mengambil daftar service profile per OLT dengan paginasi.
	List(ctx context.Context, oltID string, params ServiceProfileListParams) (*ServiceProfileListResult, error)

	// ResolveProfile menentukan service profile berdasarkan package_id dan olt_id.
	ResolveProfile(ctx context.Context, oltID string, packageID string) (*ServiceProfile, error)
}
