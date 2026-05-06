package domain

import (
	"context"
	"time"
)

// UserRepository mendefinisikan operasi data untuk tabel users.
// Diimplementasikan oleh repository.UserRepo.
type UserRepository interface {
	// CreateUser membuat user baru dan mengembalikan user yang dibuat.
	CreateUser(ctx context.Context, user *User) (*User, error)
	// GetByID mengambil user berdasarkan ID (tenant-scoped via RLS).
	GetByID(ctx context.Context, id string) (*User, error)
	// GetByEmail mengambil user berdasarkan email (lintas tenant, bypass RLS).
	GetByEmail(ctx context.Context, email string) (*User, error)
	// GetByTenantAndEmail mengambil user berdasarkan tenant_id dan email.
	GetByTenantAndEmail(ctx context.Context, tenantID, email string) (*User, error)
	// GetByGoogleID mengambil user berdasarkan google_id (lintas tenant, bypass RLS).
	GetByGoogleID(ctx context.Context, googleID string) (*User, error)
	// UpdateUser memperbarui data user (name, phone, role).
	UpdateUser(ctx context.Context, user *User) (*User, error)
	// UpdateLastLogin memperbarui timestamp last_login.
	UpdateLastLogin(ctx context.Context, userID string) error
	// UpdatePasswordHash memperbarui password_hash user.
	UpdatePasswordHash(ctx context.Context, userID, hash string) error
	// UpdateStatus memperbarui status user (active/inactive).
	UpdateStatus(ctx context.Context, userID string, status UserStatus) error
	// LinkGoogleID menambahkan google_id ke user yang sudah ada.
	LinkGoogleID(ctx context.Context, userID, googleID string) error
	// SetEmailVerified mengatur email_verified menjadi true.
	SetEmailVerified(ctx context.Context, userID string) error
	// DeleteUser menghapus user secara permanen.
	DeleteUser(ctx context.Context, userID string) error
	// ListByTenant mengambil semua user dalam satu tenant.
	ListByTenant(ctx context.Context, tenantID string) ([]*User, error)
	// EmailExistsGlobal mengecek apakah email sudah terdaftar di tenant manapun (bypass RLS).
	EmailExistsGlobal(ctx context.Context, email string) (bool, error)
}

// SessionRepository mendefinisikan operasi data untuk tabel sessions.
// Diimplementasikan oleh repository.SessionRepo.
type SessionRepository interface {
	// CreateSession membuat session baru dan mengembalikan session yang dibuat.
	CreateSession(ctx context.Context, session *Session) (*Session, error)
	// GetByTokenHash mengambil session berdasarkan hash refresh token.
	GetByTokenHash(ctx context.Context, tokenHash string) (*Session, error)
	// ListByUserID mengambil semua session aktif (belum expired) untuk user.
	ListByUserID(ctx context.Context, userID string) ([]*Session, error)
	// DeleteByID menghapus session berdasarkan ID.
	DeleteByID(ctx context.Context, sessionID string) error
	// DeleteByTokenHash menghapus session berdasarkan token hash.
	DeleteByTokenHash(ctx context.Context, tokenHash string) error
	// DeleteByUserID menghapus semua session untuk user.
	DeleteByUserID(ctx context.Context, userID string) error
	// DeleteOtherSessions menghapus semua session kecuali yang diberikan.
	DeleteOtherSessions(ctx context.Context, userID, currentSessionID string) error
	// DeleteExpired menghapus session yang sudah expired.
	DeleteExpired(ctx context.Context) error
}

// ResellerSessionRepository mendefinisikan operasi data untuk sesi reseller.
// Sesi reseller dipisahkan dari sessions user admin karena tabel sessions
// memiliki foreign key ke users.
type ResellerSessionRepository interface {
	CreateSession(ctx context.Context, session *Session) (*Session, error)
	GetByTokenHash(ctx context.Context, tokenHash string) (*Session, error)
	DeleteByTokenHash(ctx context.Context, tokenHash string) error
	DeleteByUserID(ctx context.Context, resellerID string) error
	DeleteExpired(ctx context.Context) error
}

// TokenRepository mendefinisikan operasi data untuk password_resets dan email_verifications.
// Diimplementasikan oleh repository.TokenRepo.
type TokenRepository interface {
	// CreatePasswordReset membuat token reset password baru.
	CreatePasswordReset(ctx context.Context, pr *PasswordReset) error
	// GetPasswordResetByHash mengambil password reset berdasarkan token hash.
	GetPasswordResetByHash(ctx context.Context, tokenHash string) (*PasswordReset, error)
	// MarkPasswordResetUsed menandai token sebagai sudah digunakan.
	MarkPasswordResetUsed(ctx context.Context, id string) error
	// InvalidatePasswordResets menandai semua token reset yang belum dipakai untuk user.
	InvalidatePasswordResets(ctx context.Context, userID string) error
	// CreateEmailVerification membuat token verifikasi email baru.
	CreateEmailVerification(ctx context.Context, ev *EmailVerification) error
	// GetEmailVerificationByHash mengambil verifikasi email berdasarkan token hash.
	GetEmailVerificationByHash(ctx context.Context, tokenHash string) (*EmailVerification, error)
	// MarkEmailVerificationUsed menandai token sebagai sudah digunakan.
	MarkEmailVerificationUsed(ctx context.Context, id string) error
	// InvalidateEmailVerifications menandai semua token verifikasi yang belum dipakai untuk user.
	InvalidateEmailVerifications(ctx context.Context, userID string) error
}

// --- Customer Repository ---

// CustomerRepository mendefinisikan operasi data untuk tabel customers.
type CustomerRepository interface {
	Create(ctx context.Context, customer *Customer) (*Customer, error)
	GetByID(ctx context.Context, id string) (*Customer, error)
	Update(ctx context.Context, customer *Customer) (*Customer, error)
	SoftDelete(ctx context.Context, id string) error
	List(ctx context.Context, params CustomerListParams) (*CustomerListResult, error)
	UpdateStatus(ctx context.Context, id string, status CustomerStatus) (*Customer, error)
	UpdatePackage(ctx context.Context, id string, packageID string) (*Customer, error)
	CountByStatus(ctx context.Context) (map[CustomerStatus]int64, error)
	GetMaxSeq(ctx context.Context, tenantID string) (int, error)
	PhoneExists(ctx context.Context, tenantID, phone, excludeID string) (bool, error)
	BulkUpdateStatus(ctx context.Context, ids []string, status CustomerStatus) ([]BulkResult, error)
	BulkUpdateFields(ctx context.Context, ids []string, fields map[string]interface{}) ([]BulkResult, error)
	BulkSoftDelete(ctx context.Context, ids []string) ([]BulkResult, error)
	GetByIDs(ctx context.Context, ids []string) ([]*Customer, error)
	// SearchForPayment mencari pelanggan berdasarkan nama, customer_id_seq, atau telepon.
	// Mengembalikan maksimal 10 hasil, hanya status aktif/isolir.
	SearchForPayment(ctx context.Context, tenantID, searchTerm string) ([]*Customer, error)
}

// AreaRepository mendefinisikan operasi data untuk tabel areas.
type AreaRepository interface {
	Create(ctx context.Context, area *Area) (*Area, error)
	GetByID(ctx context.Context, id string) (*Area, error)
	Update(ctx context.Context, area *Area) (*Area, error)
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, tenantID string) ([]*Area, error)
	NameExists(ctx context.Context, tenantID, name, excludeID string) (bool, error)
	CustomerCount(ctx context.Context, id string) (int, error)
}

// AuditLogRepository mendefinisikan operasi data untuk tabel audit_logs.
type AuditLogRepository interface {
	Create(ctx context.Context, log *AuditLog) error
	ListByEntity(ctx context.Context, entityType, entityID string) ([]*AuditLog, error)
}

// --- Request/Response DTOs ---

// CustomerListParams berisi parameter untuk list/filter pelanggan.
type CustomerListParams struct {
	TenantID  string `query:"tenant_id"`
	Page      int    `query:"page" validate:"omitempty,min=1"`
	PageSize  int    `query:"page_size" validate:"omitempty,oneof=10 25 50"`
	Search    string `query:"search"`
	Status    string `query:"status" validate:"omitempty,oneof=pending aktif isolir suspend berhenti"`
	PackageID string `query:"package_id" validate:"omitempty,uuid"`
	AreaID    string `query:"area_id" validate:"omitempty,uuid"`
	DueDate   *int   `query:"due_date" validate:"omitempty,min=1,max=28"`
	SortBy    string `query:"sort_by" validate:"omitempty,oneof=name customer_id_seq status created_at due_date"`
	SortOrder string `query:"sort_order" validate:"omitempty,oneof=asc desc"`
}

// CustomerListResult berisi hasil list pelanggan dengan metadata paginasi.
type CustomerListResult struct {
	Data       []*Customer    `json:"data"`
	Pagination PaginationMeta `json:"pagination"`
}

// PaginationMeta berisi metadata paginasi.
type PaginationMeta struct {
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalPages int   `json:"total_pages"`
}

// CustomerDetail berisi detail pelanggan lengkap termasuk audit log.
type CustomerDetail struct {
	Customer  *Customer   `json:"customer"`
	AuditLogs []*AuditLog `json:"audit_logs,omitempty"`
}

// BulkActionResult berisi hasil bulk action.
type BulkActionResult struct {
	Total        int           `json:"total"`
	SuccessCount int           `json:"success_count"`
	FailureCount int           `json:"failure_count"`
	Failures     []BulkFailure `json:"failures,omitempty"`
}

// BulkFailure berisi detail kegagalan per item dalam bulk action.
type BulkFailure struct {
	CustomerID string `json:"customer_id"`
	Reason     string `json:"reason"`
}

// BulkEditFields berisi field yang bisa di-edit secara massal.
type BulkEditFields struct {
	AreaID  string `json:"area_id" validate:"omitempty,uuid"`
	DueDate *int   `json:"due_date" validate:"omitempty,min=1,max=28"`
	Notes   string `json:"notes" validate:"omitempty"`
}

// BulkResult berisi hasil operasi per item dalam bulk action.
type BulkResult struct {
	ID      string
	Success bool
	Error   error
}

// CreateCustomerRequest adalah payload untuk POST /v1/customers.
type CreateCustomerRequest struct {
	Name             string  `json:"name" validate:"required,min=3,max=255"`
	Phone            string  `json:"phone" validate:"required,phone_id"`
	Email            string  `json:"email" validate:"omitempty,email"`
	Address          string  `json:"address" validate:"required,max=1000"`
	AreaID           string  `json:"area_id" validate:"omitempty,uuid"`
	Latitude         float64 `json:"latitude" validate:"omitempty,min=-90,max=90"`
	Longitude        float64 `json:"longitude" validate:"omitempty,min=-180,max=180"`
	PackageID        string  `json:"package_id" validate:"required,uuid"`
	ActivationDate   string  `json:"activation_date" validate:"required,datetime=2006-01-02"`
	DueDate          int     `json:"due_date" validate:"required,min=1,max=28"`
	ConnectionMethod string  `json:"connection_method" validate:"required,oneof=manual pppoe hotspot dhcp_binding static"`
	PPPoEUsername    string  `json:"pppoe_username" validate:"omitempty"`
	PPPoEPassword    string  `json:"pppoe_password" validate:"omitempty"`
	MACAddress       string  `json:"mac_address" validate:"required_if=ConnectionMethod dhcp_binding,omitempty,mac_addr"`
	RouterID         string  `json:"router_id" validate:"omitempty,uuid"`
	ODPPort          string  `json:"odp_port" validate:"omitempty"`
	Notes            string  `json:"notes" validate:"omitempty"`
}

// UpdateCustomerRequest adalah payload untuk PUT /v1/customers/:id.
type UpdateCustomerRequest struct {
	Name             string   `json:"name" validate:"omitempty,min=3,max=255"`
	Phone            string   `json:"phone" validate:"omitempty,phone_id"`
	Email            string   `json:"email" validate:"omitempty,email"`
	Address          string   `json:"address" validate:"omitempty,max=1000"`
	AreaID           string   `json:"area_id" validate:"omitempty,uuid"`
	Latitude         *float64 `json:"latitude" validate:"omitempty,min=-90,max=90"`
	Longitude        *float64 `json:"longitude" validate:"omitempty,min=-180,max=180"`
	PackageID        string   `json:"package_id" validate:"omitempty,uuid"`
	ActivationDate   string   `json:"activation_date" validate:"omitempty,datetime=2006-01-02"`
	DueDate          *int     `json:"due_date" validate:"omitempty,min=1,max=28"`
	ConnectionMethod string   `json:"connection_method" validate:"omitempty,oneof=manual pppoe hotspot dhcp_binding static"`
	PPPoEUsername    string   `json:"pppoe_username" validate:"omitempty"`
	PPPoEPassword    string   `json:"pppoe_password" validate:"omitempty"`
	MACAddress       string   `json:"mac_address" validate:"omitempty,mac_addr"`
	RouterID         string   `json:"router_id" validate:"omitempty,uuid"`
	ODPPort          string   `json:"odp_port" validate:"omitempty"`
	Notes            string   `json:"notes" validate:"omitempty"`
}

// DeleteCustomerRequest adalah payload untuk DELETE /v1/customers/:id.
type DeleteCustomerRequest struct {
	ConfirmationName string `json:"confirmation_name" validate:"required"`
}

// ChangePackageRequest adalah payload untuk POST /v1/customers/:id/change-package.
type ChangePackageRequest struct {
	PackageID string `json:"package_id" validate:"required,uuid"`
}

// BulkIDsRequest berisi daftar customer IDs untuk bulk action.
type BulkIDsRequest struct {
	CustomerIDs []string `json:"customer_ids" validate:"required,min=1,dive,uuid"`
}

// BulkNotifyRequest berisi daftar customer IDs dan template untuk notifikasi massal.
type BulkNotifyRequest struct {
	CustomerIDs []string `json:"customer_ids" validate:"required,min=1,dive,uuid"`
	TemplateID  string   `json:"template_id" validate:"required"`
}

// BulkChangePackageRequest berisi daftar customer IDs dan package_id baru.
type BulkChangePackageRequest struct {
	CustomerIDs []string `json:"customer_ids" validate:"required,min=1,dive,uuid"`
	PackageID   string   `json:"package_id" validate:"required,uuid"`
}

// BulkEditRequest berisi daftar customer IDs dan field yang akan diupdate.
type BulkEditRequest struct {
	CustomerIDs []string       `json:"customer_ids" validate:"required,min=1,dive,uuid"`
	Fields      BulkEditFields `json:"fields" validate:"required"`
}

// --- Package Repository ---

// PackageRepository mendefinisikan operasi data untuk tabel packages.
type PackageRepository interface {
	// Create membuat paket baru dan mengembalikan paket yang dibuat.
	Create(ctx context.Context, pkg *Package) (*Package, error)
	// GetByID mengambil paket berdasarkan ID (tenant-scoped via RLS).
	GetByID(ctx context.Context, id string) (*Package, error)
	// Update memperbarui data paket dan mengembalikan paket yang diperbarui.
	Update(ctx context.Context, pkg *Package) (*Package, error)
	// Delete menghapus paket secara permanen (hard delete).
	Delete(ctx context.Context, id string) error
	// List mengambil daftar paket dengan filter, search, sorting, dan paginasi.
	List(ctx context.Context, params PackageListParams) (*PackageListResult, error)
	// UpdateIsActive memperbarui status aktif paket.
	UpdateIsActive(ctx context.Context, id string, isActive bool) (*Package, error)
	// NameExists mengecek apakah nama paket sudah ada di tenant (exclude ID tertentu).
	NameExists(ctx context.Context, tenantID, name, excludeID string) (bool, error)
	// CustomerCount menghitung jumlah pelanggan aktif (deleted_at IS NULL) yang menggunakan paket.
	CustomerCount(ctx context.Context, id string) (int, error)
	// ListNamesByPrefix mengambil daftar nama paket yang dimulai dengan prefix tertentu.
	// Digunakan untuk generate nama duplikat yang unik.
	ListNamesByPrefix(ctx context.Context, tenantID, prefix string) ([]string, error)
}

// --- Package DTOs ---

// ActorInfo berisi informasi aktor yang melakukan operasi.
// Diambil dari JWT claims dan user data oleh handler.
type ActorInfo struct {
	ActorID   string
	ActorName string
}

// CreatePackageRequest adalah payload untuk POST /v1/packages.
// Validasi bersifat type-conditional: field yang wajib bergantung pada nilai type.
type CreatePackageRequest struct {
	Type                string `json:"type" validate:"required,oneof=monthly pppoe voucher"`
	Name                string `json:"name" validate:"required,min=2,max=255"`
	Description         string `json:"description" validate:"omitempty"`
	DownloadMbps        int    `json:"download_mbps" validate:"required,gt=0"`
	UploadMbps          int    `json:"upload_mbps" validate:"required,gt=0"`
	BandwidthType       string `json:"bandwidth_type" validate:"omitempty,oneof=dedicated shared"`
	BurstDownloadMbps   *int   `json:"burst_download_mbps" validate:"omitempty,gt=0"`
	BurstUploadMbps     *int   `json:"burst_upload_mbps" validate:"omitempty,gt=0"`
	BurstThresholdMbps  *int   `json:"burst_threshold_mbps" validate:"omitempty,gt=0"`
	BurstTimeSeconds    *int   `json:"burst_time_seconds" validate:"omitempty,gt=0"`
	QuotaType           string `json:"quota_type" validate:"required"`
	QuotaMB             *int   `json:"quota_mb" validate:"omitempty,gt=0"`
	QuotaAction         string `json:"quota_action" validate:"omitempty,oneof=throttle disconnect"`
	ThrottleMbps        *int   `json:"throttle_mbps" validate:"omitempty,gt=0"`
	MonthlyPrice        *int64 `json:"monthly_price" validate:"omitempty,gt=0"`
	InstallationFee     *int64 `json:"installation_fee" validate:"omitempty,gte=0"`
	SellPrice           *int64 `json:"sell_price" validate:"omitempty,gt=0"`
	ResellerPrice       *int64 `json:"reseller_price" validate:"omitempty,gt=0"`
	DurationValue       *int   `json:"duration_value" validate:"omitempty,gt=0"`
	DurationUnit        string `json:"duration_unit" validate:"omitempty,oneof=hours days weeks months"`
	SharedUsers         *int   `json:"shared_users" validate:"omitempty,gt=0"`
	MikrotikProfileName string `json:"mikrotik_profile_name" validate:"omitempty"`
	AddressPool         string `json:"address_pool" validate:"omitempty"`
	ParentQueue         string `json:"parent_queue" validate:"omitempty"`
	HotspotProfileName  string `json:"hotspot_profile_name" validate:"omitempty"`
}

// UpdatePackageRequest adalah payload untuk PUT /v1/packages/:id.
// Field type TIDAK boleh diubah setelah pembuatan.
type UpdatePackageRequest struct {
	Name                string `json:"name" validate:"omitempty,min=2,max=255"`
	Description         string `json:"description" validate:"omitempty"`
	DownloadMbps        *int   `json:"download_mbps" validate:"omitempty,gt=0"`
	UploadMbps          *int   `json:"upload_mbps" validate:"omitempty,gt=0"`
	BandwidthType       string `json:"bandwidth_type" validate:"omitempty,oneof=dedicated shared"`
	BurstDownloadMbps   *int   `json:"burst_download_mbps" validate:"omitempty,gt=0"`
	BurstUploadMbps     *int   `json:"burst_upload_mbps" validate:"omitempty,gt=0"`
	BurstThresholdMbps  *int   `json:"burst_threshold_mbps" validate:"omitempty,gt=0"`
	BurstTimeSeconds    *int   `json:"burst_time_seconds" validate:"omitempty,gt=0"`
	QuotaType           string `json:"quota_type" validate:"omitempty"`
	QuotaMB             *int   `json:"quota_mb" validate:"omitempty,gt=0"`
	QuotaAction         string `json:"quota_action" validate:"omitempty,oneof=throttle disconnect"`
	ThrottleMbps        *int   `json:"throttle_mbps" validate:"omitempty,gt=0"`
	MonthlyPrice        *int64 `json:"monthly_price" validate:"omitempty,gt=0"`
	InstallationFee     *int64 `json:"installation_fee" validate:"omitempty,gte=0"`
	SellPrice           *int64 `json:"sell_price" validate:"omitempty,gt=0"`
	ResellerPrice       *int64 `json:"reseller_price" validate:"omitempty,gt=0"`
	DurationValue       *int   `json:"duration_value" validate:"omitempty,gt=0"`
	DurationUnit        string `json:"duration_unit" validate:"omitempty,oneof=hours days weeks months"`
	SharedUsers         *int   `json:"shared_users" validate:"omitempty,gt=0"`
	MikrotikProfileName string `json:"mikrotik_profile_name" validate:"omitempty"`
	AddressPool         string `json:"address_pool" validate:"omitempty"`
	ParentQueue         string `json:"parent_queue" validate:"omitempty"`
	HotspotProfileName  string `json:"hotspot_profile_name" validate:"omitempty"`
}

// DeletePackageRequest adalah payload untuk DELETE /v1/packages/:id.
type DeletePackageRequest struct {
	ConfirmationName string `json:"confirmation_name" validate:"required"`
}

// PackageListParams berisi parameter untuk list/filter paket.
type PackageListParams struct {
	TenantID  string `query:"tenant_id"`
	Page      int    `query:"page" validate:"omitempty,min=1"`
	PageSize  int    `query:"page_size" validate:"omitempty,oneof=10 25 50"`
	Search    string `query:"search"`
	Type      string `query:"type" validate:"omitempty,oneof=monthly pppoe voucher"`
	IsActive  *bool  `query:"is_active"`
	SortBy    string `query:"sort_by" validate:"omitempty,oneof=name monthly_price sell_price download_mbps created_at"`
	SortOrder string `query:"sort_order" validate:"omitempty,oneof=asc desc"`
}

// PackageListResult berisi hasil list paket dengan metadata paginasi.
type PackageListResult struct {
	Data       []*Package     `json:"data"`
	Pagination PaginationMeta `json:"pagination"`
}

// PackageDetail berisi detail paket lengkap termasuk audit log.
type PackageDetail struct {
	Package   *Package    `json:"package"`
	AuditLogs []*AuditLog `json:"audit_logs,omitempty"`
}

// PackageUsecase mendefinisikan business logic untuk manajemen paket.
type PackageUsecase interface {
	Create(ctx context.Context, tenantID string, req CreatePackageRequest, actor ActorInfo) (*Package, error)
	GetByID(ctx context.Context, id string, includeAudit bool) (*PackageDetail, error)
	Update(ctx context.Context, id string, req UpdatePackageRequest, actor ActorInfo) (*Package, error)
	Delete(ctx context.Context, id string, confirmName string, actor ActorInfo) error
	List(ctx context.Context, params PackageListParams) (*PackageListResult, error)
	Activate(ctx context.Context, id string, actor ActorInfo) (*Package, error)
	Deactivate(ctx context.Context, id string, actor ActorInfo) (*Package, error)
	Duplicate(ctx context.Context, id string, actor ActorInfo) (*Package, error)
}

// =============================================================================
// Reseller Repository — operasi data untuk tabel resellers
// =============================================================================

// ResellerRepository mendefinisikan operasi data untuk tabel resellers.
type ResellerRepository interface {
	// Create membuat reseller baru dan mengembalikan reseller yang dibuat.
	Create(ctx context.Context, reseller *Reseller) (*Reseller, error)
	// GetByID mengambil reseller berdasarkan ID (tenant-scoped via RLS).
	GetByID(ctx context.Context, id string) (*Reseller, error)
	// GetByPhone mengambil reseller berdasarkan tenant_id dan nomor telepon (untuk login).
	GetByPhone(ctx context.Context, tenantID, phone string) (*Reseller, error)
	// GetByPhoneGlobal mengambil reseller berdasarkan phone saja (lintas tenant, bypass RLS).
	// Digunakan untuk login reseller yang tidak memiliki konteks tenant.
	GetByPhoneGlobal(ctx context.Context, phone string) (*Reseller, error)
	// Update memperbarui data reseller dan mengembalikan reseller yang diperbarui.
	Update(ctx context.Context, reseller *Reseller) (*Reseller, error)
	// UpdateStatus memperbarui status reseller.
	UpdateStatus(ctx context.Context, id string, status ResellerStatus) (*Reseller, error)
	// UpdatePasswordHash memperbarui password_hash reseller.
	UpdatePasswordHash(ctx context.Context, id, hash string) error
	// UpdateLastLogin memperbarui timestamp last_login reseller.
	UpdateLastLogin(ctx context.Context, id string) error
	// List mengambil daftar reseller dengan filter, search, sorting, dan paginasi.
	List(ctx context.Context, params ResellerListParams) (*ResellerListResult, error)
	// PhoneExists mengecek apakah nomor telepon sudah ada di tenant (exclude ID tertentu).
	PhoneExists(ctx context.Context, tenantID, phone, excludeID string) (bool, error)
	// GetForUpdate mengambil reseller dengan row lock (SELECT ... FOR UPDATE).
	// Digunakan dalam transaksi untuk operasi balance atomik.
	GetForUpdate(ctx context.Context, id string) (*Reseller, error)
	// UpdateBalance memperbarui saldo reseller.
	UpdateBalance(ctx context.Context, id string, newBalance int64) error
	// CountTodayPurchases menghitung jumlah voucher yang dibeli reseller hari ini.
	CountTodayPurchases(ctx context.Context, resellerID string) (int, error)
}

// =============================================================================
// Voucher Repository — operasi data untuk tabel vouchers
// =============================================================================

// VoucherRepository mendefinisikan operasi data untuk tabel vouchers.
type VoucherRepository interface {
	// BulkCreate membuat beberapa voucher sekaligus dan mengembalikan voucher yang dibuat.
	BulkCreate(ctx context.Context, vouchers []*Voucher) ([]*Voucher, error)
	// GetByID mengambil voucher berdasarkan ID (dengan joined package_name dan reseller_name).
	GetByID(ctx context.Context, id string) (*Voucher, error)
	// GetByCode mengambil voucher berdasarkan kode dalam tenant tertentu.
	GetByCode(ctx context.Context, tenantID, code string) (*Voucher, error)
	// UpdateStatus memperbarui status voucher.
	UpdateStatus(ctx context.Context, id string, status VoucherStatus) (*Voucher, error)
	// Activate memperbarui voucher menjadi aktif, set activated_at dan expires_at penggunaan.
	Activate(ctx context.Context, id string, expiresAt time.Time) (*Voucher, error)
	// List mengambil daftar voucher dengan filter, search, sorting, dan paginasi (admin).
	List(ctx context.Context, params VoucherListParams) (*VoucherListResult, error)
	// ListByReseller mengambil daftar voucher milik reseller tertentu.
	ListByReseller(ctx context.Context, params ResellerVoucherListParams) (*VoucherListResult, error)
	// GetAvailableByPackage mengambil voucher tersedia untuk paket tertentu (untuk assign).
	GetAvailableByPackage(ctx context.Context, packageID string, limit int) ([]*Voucher, error)
	// BulkUpdateStatus memperbarui status beberapa voucher sekaligus.
	BulkUpdateStatus(ctx context.Context, ids []string, status VoucherStatus) ([]BulkResult, error)
	// BulkAssign meng-assign voucher ke reseller (admin assignment, tanpa potong saldo).
	BulkAssign(ctx context.Context, ids []string, resellerID string) ([]BulkResult, error)
	// AssignToReseller meng-assign voucher ke reseller saat pembelian (set snapshot, purchased_at, expires_at).
	AssignToReseller(ctx context.Context, id string, resellerID string, sellSnapshot, resellerSnapshot int64, expiresAt time.Time) (*Voucher, error)
	// GetExpiredVouchers mengambil voucher terjual yang sudah melewati expires_at.
	GetExpiredVouchers(ctx context.Context, batchSize int) ([]*Voucher, error)
	// CodeExists mengecek apakah kode voucher sudah ada di tenant.
	CodeExists(ctx context.Context, tenantID, code string) (bool, error)
	// GetByIDs mengambil beberapa voucher berdasarkan ID.
	GetByIDs(ctx context.Context, ids []string) ([]*Voucher, error)
	// CountByResellerAndStatus menghitung voucher per reseller dan status.
	CountByResellerAndStatus(ctx context.Context, resellerID string, statuses []VoucherStatus) (int, error)
	// CountSoldToday menghitung voucher yang dibeli reseller hari ini.
	CountSoldToday(ctx context.Context, resellerID string) (int, error)
}

// =============================================================================
// Voucher Audit Log Repository — operasi data untuk tabel voucher_audit_logs
// =============================================================================

// VoucherAuditLogRepository mendefinisikan operasi data untuk tabel voucher_audit_logs.
type VoucherAuditLogRepository interface {
	// Create membuat satu entri audit log voucher.
	Create(ctx context.Context, log *VoucherAuditLog) error
	// BulkCreate membuat beberapa entri audit log voucher sekaligus.
	BulkCreate(ctx context.Context, logs []*VoucherAuditLog) error
	// ListByVoucher mengambil semua audit log untuk voucher tertentu.
	ListByVoucher(ctx context.Context, voucherID string) ([]*VoucherAuditLog, error)
}

// =============================================================================
// Reseller Transaction Repository — operasi data untuk tabel reseller_transactions
// =============================================================================

// ResellerTransactionRepository mendefinisikan operasi data untuk tabel reseller_transactions.
type ResellerTransactionRepository interface {
	// Create membuat satu transaksi reseller dan mengembalikan transaksi yang dibuat.
	Create(ctx context.Context, tx *ResellerTransaction) (*ResellerTransaction, error)
	// ListByReseller mengambil daftar transaksi reseller dengan paginasi.
	ListByReseller(ctx context.Context, params ResellerTxListParams) (*ResellerTxListResult, error)
	// ListDepositsByReseller mengambil daftar deposit reseller dengan paginasi.
	ListDepositsByReseller(ctx context.Context, params ResellerTxListParams) (*ResellerTxListResult, error)
}

// =============================================================================
// Reseller DTOs — request/response untuk manajemen reseller
// =============================================================================

// CreateResellerRequest adalah payload untuk POST /v1/resellers.
type CreateResellerRequest struct {
	Name               string `json:"name" validate:"required,min=3,max=255"`
	Phone              string `json:"phone" validate:"required,phone_id"`
	Email              string `json:"email" validate:"omitempty,email"`
	Address            string `json:"address" validate:"omitempty,max=1000"`
	Password           string `json:"password" validate:"required,min=8"`
	Balance            *int64 `json:"balance" validate:"omitempty,gte=0"`
	DailyPurchaseLimit *int   `json:"daily_purchase_limit" validate:"omitempty,gte=0"`
}

// UpdateResellerRequest adalah payload untuk PUT /v1/resellers/:id.
type UpdateResellerRequest struct {
	Name               string `json:"name" validate:"omitempty,min=3,max=255"`
	Phone              string `json:"phone" validate:"omitempty,phone_id"`
	Email              string `json:"email" validate:"omitempty,email"`
	Address            string `json:"address" validate:"omitempty,max=1000"`
	DailyPurchaseLimit *int   `json:"daily_purchase_limit" validate:"omitempty,gte=0"`
}

// DepositRequest adalah payload untuk POST /v1/resellers/:id/deposit.
type DepositRequest struct {
	Amount int64  `json:"amount" validate:"required,gt=0"`
	Notes  string `json:"notes" validate:"omitempty,max=500"`
}

// WithdrawRequest adalah payload untuk POST /v1/resellers/:id/withdraw.
type WithdrawRequest struct {
	Amount int64  `json:"amount" validate:"required,gt=0"`
	Notes  string `json:"notes" validate:"omitempty,max=500"`
}

// DeactivateResellerRequest adalah payload untuk POST /v1/resellers/:id/deactivate.
type DeactivateResellerRequest struct {
	ConfirmationName string `json:"confirmation_name" validate:"required"`
}

// ResellerListParams berisi parameter untuk list/filter reseller.
type ResellerListParams struct {
	TenantID  string `query:"tenant_id"`
	Page      int    `query:"page" validate:"omitempty,min=1"`
	PageSize  int    `query:"page_size" validate:"omitempty,oneof=10 25 50"`
	Search    string `query:"search"`
	Status    string `query:"status" validate:"omitempty,oneof=aktif suspended nonaktif"`
	SortBy    string `query:"sort_by" validate:"omitempty,oneof=name balance created_at"`
	SortOrder string `query:"sort_order" validate:"omitempty,oneof=asc desc"`
}

// ResellerListResult berisi hasil list reseller dengan metadata paginasi.
type ResellerListResult struct {
	Data       []*Reseller    `json:"data"`
	Pagination PaginationMeta `json:"pagination"`
}

// ResellerDetail berisi detail reseller lengkap termasuk audit log.
type ResellerDetail struct {
	Reseller  *Reseller   `json:"reseller"`
	AuditLogs []*AuditLog `json:"audit_logs,omitempty"`
}

// =============================================================================
// Reseller Auth DTOs — request/response untuk autentikasi reseller
// =============================================================================

// ResellerLoginRequest adalah payload untuk POST /v1/reseller/auth/login.
type ResellerLoginRequest struct {
	Phone    string `json:"phone" validate:"required,phone_id"`
	Password string `json:"password" validate:"required"`
}

// ResellerLoginResponse adalah respons untuk login reseller sukses.
type ResellerLoginResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresIn    int64     `json:"expires_in"`
	Reseller     *Reseller `json:"reseller"`
}

// =============================================================================
// Voucher DTOs — request/response untuk manajemen voucher
// =============================================================================

// GenerateVoucherRequest adalah payload untuk POST /v1/vouchers/generate.
type GenerateVoucherRequest struct {
	PackageID  string `json:"package_id" validate:"required,uuid"`
	Quantity   int    `json:"quantity" validate:"required,gt=0"`
	CodeFormat string `json:"code_format" validate:"required,oneof=digits letters mixed"`
	CodeLength int    `json:"code_length" validate:"required,min=6,max=16"`
	Prefix     string `json:"prefix" validate:"omitempty,max=10,alphanum_hyphen"`
}

// GenerateVoucherResult berisi hasil generate voucher.
type GenerateVoucherResult struct {
	TotalRequested int        `json:"total_requested"`
	TotalGenerated int        `json:"total_generated"`
	TotalFailed    int        `json:"total_failed"`
	Vouchers       []*Voucher `json:"vouchers,omitempty"` // hanya untuk sync generate
	JobID          string     `json:"job_id,omitempty"`   // hanya untuk async generate
}

// VoucherListParams berisi parameter untuk list/filter voucher (admin).
type VoucherListParams struct {
	TenantID   string `query:"tenant_id"`
	Page       int    `query:"page" validate:"omitempty,min=1"`
	PageSize   int    `query:"page_size" validate:"omitempty,oneof=10 25 50"`
	Search     string `query:"search"`
	PackageID  string `query:"package_id" validate:"omitempty,uuid"`
	Status     string `query:"status" validate:"omitempty,oneof=tersedia terjual aktif selesai expired void"`
	ResellerID string `query:"reseller_id" validate:"omitempty,uuid"`
	SortBy     string `query:"sort_by" validate:"omitempty,oneof=code status created_at purchased_at"`
	SortOrder  string `query:"sort_order" validate:"omitempty,oneof=asc desc"`
}

// VoucherListResult berisi hasil list voucher dengan metadata paginasi.
type VoucherListResult struct {
	Data       []*Voucher     `json:"data"`
	Pagination PaginationMeta `json:"pagination"`
}

// VoucherDetail berisi detail voucher lengkap termasuk audit log.
type VoucherDetail struct {
	Voucher   *Voucher           `json:"voucher"`
	AuditLogs []*VoucherAuditLog `json:"audit_logs,omitempty"`
}

// BulkVoucherIDsRequest berisi daftar voucher IDs untuk bulk action.
type BulkVoucherIDsRequest struct {
	VoucherIDs []string `json:"voucher_ids" validate:"required,min=1,dive,uuid"`
}

// BulkAssignRequest berisi daftar voucher IDs dan reseller target.
type BulkAssignRequest struct {
	VoucherIDs []string `json:"voucher_ids" validate:"required,min=1,dive,uuid"`
	ResellerID string   `json:"reseller_id" validate:"required,uuid"`
}

// ActivateVoucherRequest adalah payload untuk mengaktifkan voucher Hotspot.
type ActivateVoucherRequest struct {
	Code       string `json:"code" validate:"required"`
	RouterID   string `json:"router_id" validate:"omitempty,uuid"`
	MACAddress string `json:"mac_address" validate:"omitempty"`
}

// =============================================================================
// Dashboard DTOs — request/response untuk dashboard reseller
// =============================================================================

// DashboardSummary berisi ringkasan dashboard reseller.
type DashboardSummary struct {
	Balance           int64 `json:"balance"`
	SoldToday         int   `json:"sold_today"`
	AvailableVouchers int   `json:"available_vouchers"`
}

// BuyVoucherRequest adalah payload untuk POST /v1/reseller/vouchers/buy.
type BuyVoucherRequest struct {
	PackageID string `json:"package_id" validate:"required,uuid"`
	Quantity  int    `json:"quantity" validate:"required,min=1,max=100"`
}

// BuyVoucherResult berisi hasil pembelian voucher.
type BuyVoucherResult struct {
	Vouchers     []*Voucher `json:"vouchers"`
	TotalCost    int64      `json:"total_cost"`
	BalanceAfter int64      `json:"balance_after"`
}

// ResellerVoucherListParams berisi parameter untuk list voucher reseller.
type ResellerVoucherListParams struct {
	ResellerID string `query:"reseller_id"`
	TenantID   string `query:"tenant_id"`
	Page       int    `query:"page" validate:"omitempty,min=1"`
	PageSize   int    `query:"page_size" validate:"omitempty,oneof=10 25 50"`
	Status     string `query:"status" validate:"omitempty,oneof=terjual aktif selesai expired"`
	PackageID  string `query:"package_id" validate:"omitempty,uuid"`
	SortBy     string `query:"sort_by" validate:"omitempty,oneof=code status purchased_at"`
	SortOrder  string `query:"sort_order" validate:"omitempty,oneof=asc desc"`
}

// ResellerTxListParams berisi parameter untuk list transaksi reseller.
type ResellerTxListParams struct {
	ResellerID string `query:"reseller_id"`
	TenantID   string `query:"tenant_id"`
	Page       int    `query:"page" validate:"omitempty,min=1"`
	PageSize   int    `query:"page_size" validate:"omitempty,oneof=10 25 50"`
	Type       string `query:"type" validate:"omitempty,oneof=deposit purchase refund withdraw"`
	SortOrder  string `query:"sort_order" validate:"omitempty,oneof=asc desc"`
}

// ResellerTxListResult berisi hasil list transaksi reseller.
type ResellerTxListResult struct {
	Data       []*ResellerTransaction `json:"data"`
	Pagination PaginationMeta         `json:"pagination"`
}

// =============================================================================
// Invoice Repository — operasi data untuk tabel invoices
// =============================================================================

// InvoiceRepository mendefinisikan operasi data untuk tabel invoices.
type InvoiceRepository interface {
	// Create membuat invoice baru dan mengembalikan invoice yang dibuat.
	Create(ctx context.Context, invoice *Invoice) (*Invoice, error)
	// GetByID mengambil invoice berdasarkan ID (dengan JOIN ke customers dan packages).
	GetByID(ctx context.Context, id string) (*Invoice, error)
	// Update memperbarui data invoice dan mengembalikan invoice yang diperbarui.
	Update(ctx context.Context, invoice *Invoice) (*Invoice, error)
	// UpdateStatus memperbarui status invoice dengan optimistic locking via version.
	UpdateStatus(ctx context.Context, id string, status InvoiceStatus, version int) (*Invoice, error)
	// UpdatePaidAmount memperbarui jumlah yang sudah dibayar dengan optimistic locking via version.
	UpdatePaidAmount(ctx context.Context, id string, paidAmount int64, version int) (*Invoice, error)
	// List mengambil daftar invoice dengan filter, search, sorting, dan paginasi.
	List(ctx context.Context, params InvoiceListParams) (*InvoiceListResult, error)
	// ExistsForPeriod mengecek apakah invoice sudah ada untuk customer dan periode tertentu.
	ExistsForPeriod(ctx context.Context, customerID string, month, year int) (bool, error)
	// ExistsForPeriodPrepaid mengecek apakah invoice prepaid sudah mencakup periode tertentu.
	ExistsForPeriodPrepaid(ctx context.Context, customerID string, month, year int) (bool, error)
	// FindOverdue mengambil semua invoice yang sudah melewati jatuh tempo (status belum_bayar).
	FindOverdue(ctx context.Context, currentDate time.Time) ([]*Invoice, error)
	// GetSummary mengambil ringkasan invoice per status untuk dashboard.
	GetSummary(ctx context.Context, tenantID string, periodMonth, periodYear *int) (*InvoiceSummary, error)
	// GetByIDs mengambil beberapa invoice berdasarkan daftar ID.
	GetByIDs(ctx context.Context, ids []string) ([]*Invoice, error)
	// FindOpenByCustomer mengambil semua invoice terbuka untuk customer, urut berdasarkan due_date ASC.
	// Terbuka = status in (belum_bayar, terlambat, bayar_sebagian).
	FindOpenByCustomer(ctx context.Context, customerID string) ([]*Invoice, error)
	// FindOpenByCustomerForUpdate sama seperti FindOpenByCustomer tapi dengan SELECT FOR UPDATE.
	// Harus dipanggil dalam transaksi.
	FindOpenByCustomerForUpdate(ctx context.Context, customerID string) ([]*Invoice, error)
	// GetByIDsForUpdate mengambil invoice berdasarkan ID dengan SELECT FOR UPDATE.
	// Harus dipanggil dalam transaksi.
	GetByIDsForUpdate(ctx context.Context, ids []string) ([]*Invoice, error)
	// FindOverdueForIsolir mengambil invoice terlambat yang sudah melewati grace period.
	// Mengembalikan invoice beserta customer_id yang eligible untuk isolir.
	FindOverdueForIsolir(ctx context.Context, tenantID string, gracePeriodDays int, currentDate time.Time) ([]*Invoice, error)
	// FindOverdueForSuspend mengambil invoice terlambat yang sudah melewati suspend_days.
	FindOverdueForSuspend(ctx context.Context, tenantID string, suspendDays int, currentDate time.Time) ([]*Invoice, error)
	// HasOutstandingInvoices mengecek apakah customer masih punya invoice belum lunas.
	HasOutstandingInvoices(ctx context.Context, customerID string) (bool, error)
	// SumOutstandingAmount menghitung total tagihan outstanding untuk customer.
	SumOutstandingAmount(ctx context.Context, customerID string) (int64, error)
	// CountOutstandingInvoices menghitung jumlah invoice outstanding untuk customer.
	CountOutstandingInvoices(ctx context.Context, customerID string) (int, error)
}

// =============================================================================
// Invoice Item Repository — operasi data untuk tabel invoice_items
// =============================================================================

// InvoiceItemRepository mendefinisikan operasi data untuk tabel invoice_items.
type InvoiceItemRepository interface {
	// BulkCreate membuat beberapa item invoice sekaligus.
	BulkCreate(ctx context.Context, items []*InvoiceItem) ([]*InvoiceItem, error)
	// ListByInvoice mengambil semua item untuk invoice tertentu (urut berdasarkan sort_order).
	ListByInvoice(ctx context.Context, invoiceID string) ([]*InvoiceItem, error)
	// DeleteByInvoice menghapus semua item untuk invoice tertentu (digunakan saat edit).
	DeleteByInvoice(ctx context.Context, invoiceID string) error
}

// =============================================================================
// Invoice Payment Repository — operasi data untuk tabel invoice_payments
// =============================================================================

// InvoicePaymentRepository mendefinisikan operasi data untuk tabel invoice_payments.
type InvoicePaymentRepository interface {
	// Create membuat catatan pembayaran baru dan mengembalikan pembayaran yang dibuat.
	Create(ctx context.Context, payment *InvoicePayment) (*InvoicePayment, error)
	// ListByInvoice mengambil semua pembayaran non-void untuk invoice tertentu.
	ListByInvoice(ctx context.Context, invoiceID string) ([]*InvoicePayment, error)
	// VoidPayment menandai pembayaran sebagai void dengan alasan.
	VoidPayment(ctx context.Context, id string, voidedBy string, reason string) error
	// GetByID mengambil satu pembayaran berdasarkan ID.
	GetByID(ctx context.Context, id string) (*InvoicePayment, error)
	// ListWithFilters mengambil daftar pembayaran dengan filter, pencarian, dan paginasi.
	// Join dengan customers dan invoices untuk field pencarian dan tampilan.
	ListWithFilters(ctx context.Context, params PaymentListParams) (*PaymentListResult, error)
	// GetSummary mengambil statistik pembayaran agregat untuk tenant.
	GetSummary(ctx context.Context, tenantID string, timezone string, periodMonth, periodYear *int) (*PaymentSummary, error)
	// FindDuplicate mengecek potensi duplikasi pembayaran dalam 24 jam terakhir.
	FindDuplicate(ctx context.Context, customerID string, amount int64, method string, paymentDate time.Time) (bool, error)
}

// =============================================================================
// Invoice Audit Log Repository — operasi data untuk tabel invoice_audit_logs
// =============================================================================

// InvoiceAuditLogRepository mendefinisikan operasi data untuk tabel invoice_audit_logs (append-only).
type InvoiceAuditLogRepository interface {
	// Create membuat satu entri audit log invoice.
	Create(ctx context.Context, log *InvoiceAuditLog) error
	// ListByInvoice mengambil semua audit log untuk invoice tertentu (urut berdasarkan created_at).
	ListByInvoice(ctx context.Context, invoiceID string) ([]*InvoiceAuditLog, error)
}

// =============================================================================
// Invoice Sequence Repository — operasi data untuk tabel invoice_sequences
// =============================================================================

// InvoiceSequenceRepository mendefinisikan operasi data untuk tabel invoice_sequences.
type InvoiceSequenceRepository interface {
	// NextSequence mengambil dan increment sequence secara atomik (SELECT FOR UPDATE).
	// Membuat row baru jika belum ada untuk tenant/year/month.
	NextSequence(ctx context.Context, tenantID string, year, month int) (int, error)
}

// =============================================================================
// Receipt Sequence Repository — operasi data untuk tabel receipt_sequences
// =============================================================================

// ReceiptSequenceRepository mendefinisikan operasi data untuk tabel receipt_sequences.
type ReceiptSequenceRepository interface {
	// NextSequence mengambil dan increment sequence kwitansi secara atomik.
	// Membuat row baru jika belum ada untuk tenant/year/month.
	// Menggunakan SELECT FOR UPDATE untuk keamanan konkurensi.
	NextSequence(ctx context.Context, tenantID string, year, month int) (int, error)
}

// =============================================================================
// Billing Settings Repository — operasi data untuk tabel billing_settings
// =============================================================================

// BillingSettingsRepository mendefinisikan operasi data untuk tabel billing_settings.
type BillingSettingsRepository interface {
	// GetByTenantID mengambil billing settings berdasarkan tenant ID.
	GetByTenantID(ctx context.Context, tenantID string) (*BillingSettings, error)
	// Upsert membuat atau memperbarui billing settings untuk tenant.
	Upsert(ctx context.Context, settings *BillingSettings) (*BillingSettings, error)
	// ListAll mengambil semua billing settings (untuk cron job lintas tenant).
	ListAll(ctx context.Context) ([]*BillingSettings, error)
}

// =============================================================================
// Customer Recurring Item Repository — operasi data untuk tabel customer_recurring_items
// =============================================================================

// CustomerRecurringItemRepository mendefinisikan operasi data untuk tabel customer_recurring_items.
type CustomerRecurringItemRepository interface {
	// Create membuat recurring item baru dan mengembalikan item yang dibuat.
	Create(ctx context.Context, item *CustomerRecurringItem) (*CustomerRecurringItem, error)
	// GetByID mengambil recurring item berdasarkan ID.
	GetByID(ctx context.Context, id string) (*CustomerRecurringItem, error)
	// Update memperbarui recurring item dan mengembalikan item yang diperbarui.
	Update(ctx context.Context, item *CustomerRecurringItem) (*CustomerRecurringItem, error)
	// Deactivate menonaktifkan recurring item (set is_active = false).
	Deactivate(ctx context.Context, id string) error
	// ListByCustomer mengambil semua recurring item untuk customer tertentu.
	ListByCustomer(ctx context.Context, customerID string) ([]*CustomerRecurringItem, error)
	// ListActiveByCustomer mengambil recurring item aktif untuk customer pada tanggal periode tertentu.
	ListActiveByCustomer(ctx context.Context, customerID string, periodDate time.Time) ([]*CustomerRecurringItem, error)
}

// =============================================================================
// Credit Note Repository — operasi data untuk credit notes
// =============================================================================

// CreditNoteRepository mendefinisikan operasi data untuk credit notes.
type CreditNoteRepository interface {
	// Create membuat credit note baru dan mengembalikan credit note yang dibuat.
	Create(ctx context.Context, cn *CreditNote) (*CreditNote, error)
	// GetByID mengambil credit note berdasarkan ID.
	GetByID(ctx context.Context, id string) (*CreditNote, error)
	// ListByInvoice mengambil semua credit note untuk invoice tertentu.
	ListByInvoice(ctx context.Context, invoiceID string) ([]*CreditNote, error)
}

// =============================================================================
// Debit Note Repository — operasi data untuk debit notes
// =============================================================================

// DebitNoteRepository mendefinisikan operasi data untuk debit notes.
type DebitNoteRepository interface {
	// Create membuat debit note baru dan mengembalikan debit note yang dibuat.
	Create(ctx context.Context, dn *DebitNote) (*DebitNote, error)
	// GetByID mengambil debit note berdasarkan ID.
	GetByID(ctx context.Context, id string) (*DebitNote, error)
	// ListByCustomer mengambil semua debit note untuk customer tertentu.
	ListByCustomer(ctx context.Context, customerID string) ([]*DebitNote, error)
}

// =============================================================================
// Invoice DTOs — request/response untuk manajemen invoice
// =============================================================================

// CreateInvoiceRequest adalah payload untuk POST /v1/invoices (pembuatan invoice manual).
type CreateInvoiceRequest struct {
	CustomerID  string                     `json:"customer_id" validate:"required,uuid"`
	DueDate     string                     `json:"due_date" validate:"required,datetime=2006-01-02"`
	Items       []CreateInvoiceItemRequest `json:"items" validate:"required,min=1,dive"`
	Notes       string                     `json:"notes" validate:"omitempty,max=1000"`
	ApplyTax    *bool                      `json:"apply_tax"`
	ApplyCredit *bool                      `json:"apply_credit"`
}

// CreateInvoiceItemRequest adalah payload untuk satu item dalam pembuatan invoice.
type CreateInvoiceItemRequest struct {
	Description string `json:"description" validate:"required,max=500"`
	Quantity    int    `json:"quantity" validate:"required,gt=0"`
	UnitPrice   int64  `json:"unit_price" validate:"required,gt=0"`
}

// EditInvoiceRequest adalah payload untuk PUT /v1/invoices/:id.
type EditInvoiceRequest struct {
	DueDate string                     `json:"due_date" validate:"omitempty,datetime=2006-01-02"`
	Items   []CreateInvoiceItemRequest `json:"items" validate:"omitempty,min=1,dive"`
	Notes   string                     `json:"notes" validate:"omitempty,max=1000"`
}

// CancelInvoiceRequest adalah payload untuk POST /v1/invoices/:id/cancel.
type CancelInvoiceRequest struct {
	ConfirmationNumber string `json:"confirmation_number" validate:"required"`
	Reason             string `json:"reason" validate:"required,min=5,max=500"`
}

// RecordPaymentRequest adalah payload untuk POST /v1/invoices/:id/payment.
type RecordPaymentRequest struct {
	Amount          int64  `json:"amount" validate:"required,gt=0"`
	PaymentMethod   string `json:"payment_method" validate:"required,oneof=tunai transfer xendit midtrans lainnya"`
	PaymentDate     string `json:"payment_date" validate:"required,datetime=2006-01-02"`
	ReferenceNumber string `json:"reference_number" validate:"omitempty"`
	Notes           string `json:"notes" validate:"omitempty,max=500"`
}

// CreatePrepaidInvoiceRequest adalah payload untuk POST /v1/invoices/prepaid.
type CreatePrepaidInvoiceRequest struct {
	CustomerID       string `json:"customer_id" validate:"required,uuid"`
	Months           int    `json:"months" validate:"required,oneof=3 6 12"`
	StartPeriodMonth int    `json:"start_period_month" validate:"required,min=1,max=12"`
	StartPeriodYear  int    `json:"start_period_year" validate:"required"`
	DiscountMonths   int    `json:"discount_months" validate:"omitempty,gte=0"`
}

// BulkInvoiceIDsRequest berisi daftar invoice IDs untuk bulk action.
type BulkInvoiceIDsRequest struct {
	InvoiceIDs []string `json:"invoice_ids" validate:"required,min=1,dive,uuid"`
}

// BulkCancelRequest berisi daftar invoice IDs dan alasan untuk bulk cancel.
type BulkCancelRequest struct {
	InvoiceIDs []string `json:"invoice_ids" validate:"required,min=1,dive,uuid"`
	Reason     string   `json:"reason" validate:"required,min=5,max=500"`
}

// =============================================================================
// Invoice List/Detail DTOs — response untuk list dan detail invoice
// =============================================================================

// InvoiceListParams berisi parameter untuk list/filter invoice.
type InvoiceListParams struct {
	TenantID    string `query:"tenant_id"`
	CustomerID  string `query:"customer_id" validate:"omitempty,uuid"`
	Page        int    `query:"page" validate:"omitempty,min=1"`
	PageSize    int    `query:"page_size" validate:"omitempty,oneof=10 25 50"`
	Search      string `query:"search"`
	Status      string `query:"status" validate:"omitempty,oneof=belum_bayar terlambat lunas bayar_sebagian batal prorate"`
	PeriodMonth *int   `query:"period_month" validate:"omitempty,min=1,max=12"`
	PeriodYear  *int   `query:"period_year"`
	PackageID   string `query:"package_id" validate:"omitempty,uuid"`
	AreaID      string `query:"area_id" validate:"omitempty,uuid"`
	SortBy      string `query:"sort_by" validate:"omitempty,oneof=invoice_number due_date total_amount status created_at"`
	SortOrder   string `query:"sort_order" validate:"omitempty,oneof=asc desc"`
}

// InvoiceListResult berisi hasil list invoice dengan metadata paginasi.
type InvoiceListResult struct {
	Data       []*Invoice     `json:"data"`
	Pagination PaginationMeta `json:"pagination"`
}

// InvoiceDetail berisi detail invoice lengkap termasuk items, payments, dan audit logs.
type InvoiceDetail struct {
	Invoice   *Invoice           `json:"invoice"`
	Items     []*InvoiceItem     `json:"items"`
	Payments  []*InvoicePayment  `json:"payments"`
	AuditLogs []*InvoiceAuditLog `json:"audit_logs,omitempty"`
}

// InvoiceSummary berisi ringkasan invoice per status untuk dashboard.
type InvoiceSummary struct {
	Total    InvoiceSummaryStat                   `json:"total"`
	ByStatus map[InvoiceStatus]InvoiceSummaryStat `json:"by_status"`
}

// InvoiceSummaryStat berisi statistik ringkasan invoice (jumlah dan total nominal).
type InvoiceSummaryStat struct {
	Count       int64 `json:"count"`
	TotalAmount int64 `json:"total_amount"`
}

// =============================================================================
// Recurring Item DTOs — request/response untuk recurring items pelanggan
// =============================================================================

// CreateRecurringItemRequest adalah payload untuk POST /v1/customers/:id/recurring-items.
type CreateRecurringItemRequest struct {
	Description string `json:"description" validate:"required,min=3,max=255"`
	Amount      int64  `json:"amount" validate:"required,gt=0"`
	StartDate   string `json:"start_date" validate:"required,datetime=2006-01-02"`
	EndDate     string `json:"end_date" validate:"omitempty,datetime=2006-01-02"`
}

// UpdateRecurringItemRequest adalah payload untuk PUT /v1/customers/:id/recurring-items/:item_id.
type UpdateRecurringItemRequest struct {
	Description string `json:"description" validate:"omitempty,min=3,max=255"`
	Amount      *int64 `json:"amount" validate:"omitempty,gt=0"`
	EndDate     string `json:"end_date" validate:"omitempty,datetime=2006-01-02"`
}

// =============================================================================
// Credit/Debit Note DTOs — request/response untuk credit dan debit notes
// =============================================================================

// CreateCreditNoteRequest adalah payload untuk POST /v1/credit-notes.
type CreateCreditNoteRequest struct {
	InvoiceID     string `json:"invoice_id" validate:"required,uuid"`
	Amount        int64  `json:"amount" validate:"required,gt=0"`
	Reason        string `json:"reason" validate:"required,min=5,max=500"`
	ApplyToCredit *bool  `json:"apply_to_credit"`
}

// CreateDebitNoteRequest adalah payload untuk POST /v1/debit-notes.
type CreateDebitNoteRequest struct {
	CustomerID    string                 `json:"customer_id" validate:"required,uuid"`
	Items         []DebitNoteItemRequest `json:"items" validate:"required,min=1,dive"`
	DueDate       string                 `json:"due_date" validate:"required,datetime=2006-01-02"`
	CreateInvoice bool                   `json:"create_invoice"`
}

// DebitNoteItemRequest adalah payload untuk satu item dalam pembuatan debit note.
type DebitNoteItemRequest struct {
	Description string `json:"description" validate:"required,max=500"`
	Amount      int64  `json:"amount" validate:"required,gt=0"`
}

// =============================================================================
// Invoice Bulk Action Result — hasil bulk action khusus invoice
// =============================================================================

// InvoiceBulkActionResult berisi hasil bulk action untuk invoice.
type InvoiceBulkActionResult struct {
	Total        int                  `json:"total"`
	SuccessCount int                  `json:"success_count"`
	FailureCount int                  `json:"failure_count"`
	Failures     []InvoiceBulkFailure `json:"failures,omitempty"`
}

// InvoiceBulkFailure berisi detail kegagalan per item dalam bulk action invoice.
type InvoiceBulkFailure struct {
	InvoiceID string `json:"invoice_id"`
	Reason    string `json:"reason"`
}

// =============================================================================
// Payment DTOs — request/response untuk modul pembayaran manual
// =============================================================================

// PaymentListParams berisi parameter untuk list/filter pembayaran.
type PaymentListParams struct {
	TenantID      string `query:"tenant_id"`
	Page          int    `query:"page" validate:"omitempty,min=1"`
	PageSize      int    `query:"page_size" validate:"omitempty,oneof=10 25 50"`
	PaymentMethod string `query:"payment_method" validate:"omitempty,oneof=tunai transfer lainnya"`
	DateFrom      string `query:"date_from" validate:"omitempty,datetime=2006-01-02"`
	DateTo        string `query:"date_to" validate:"omitempty,datetime=2006-01-02"`
	RecordedBy    string `query:"recorded_by" validate:"omitempty,uuid"`
	Search        string `query:"search"`
	IncludeVoided bool   `query:"include_voided"`
}

// PaymentListItem merepresentasikan satu item dalam daftar pembayaran.
type PaymentListItem struct {
	ID              string    `json:"id"`
	InvoiceID       string    `json:"invoice_id"`
	InvoiceNumber   string    `json:"invoice_number"`
	CustomerName    string    `json:"customer_name"`
	CustomerIDSeq   string    `json:"customer_id_seq"`
	Amount          int64     `json:"amount"`
	PaymentMethod   string    `json:"payment_method"`
	PaymentDate     time.Time `json:"payment_date"`
	ReferenceNumber string    `json:"reference_number,omitempty"`
	ReceiptNumber   string    `json:"receipt_number,omitempty"`
	RecordedByName  string    `json:"recorded_by_name"`
	Voided          bool      `json:"voided"`
	VoidReason      string    `json:"void_reason,omitempty"`
	ProofImageURL   string    `json:"proof_image_url,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

// PaymentListResult berisi hasil list pembayaran dengan metadata paginasi.
type PaymentListResult struct {
	Data       []PaymentListItem `json:"data"`
	Pagination PaginationMeta    `json:"pagination"`
}

// PaymentSummaryStat berisi statistik ringkasan pembayaran (jumlah dan total nominal).
type PaymentSummaryStat struct {
	Count       int64 `json:"count"`
	TotalAmount int64 `json:"total_amount"`
}

// PaymentSummary berisi ringkasan pembayaran untuk dashboard.
type PaymentSummary struct {
	Today     PaymentSummaryStat            `json:"today"`
	ThisMonth PaymentSummaryStat            `json:"this_month"`
	ByMethod  map[string]PaymentSummaryStat `json:"by_method"`
}

// OpenInvoiceItem merepresentasikan satu invoice terbuka untuk pembayaran cepat.
type OpenInvoiceItem struct {
	ID              string        `json:"id"`
	InvoiceNumber   string        `json:"invoice_number"`
	PeriodMonth     int           `json:"period_month"`
	PeriodYear      int           `json:"period_year"`
	TotalAmount     int64         `json:"total_amount"`
	PaidAmount      int64         `json:"paid_amount"`
	RemainingAmount int64         `json:"remaining_amount"`
	Status          InvoiceStatus `json:"status"`
	DueDate         time.Time     `json:"due_date"`
}

// OpenInvoicesResponse berisi daftar invoice terbuka dan total tunggakan.
type OpenInvoicesResponse struct {
	Invoices     []OpenInvoiceItem `json:"invoices"`
	TotalArrears int64             `json:"total_arrears"`
}

// MultiPaymentRequest adalah payload untuk POST /v1/payments/multi.
type MultiPaymentRequest struct {
	CustomerID      string   `json:"customer_id" validate:"required,uuid"`
	Amount          int64    `json:"amount" validate:"required,gt=0"`
	PaymentMethod   string   `json:"payment_method" validate:"required,oneof=tunai transfer lainnya"`
	PaymentDate     string   `json:"payment_date" validate:"required,datetime=2006-01-02"`
	ReferenceNumber string   `json:"reference_number" validate:"omitempty"`
	Notes           string   `json:"notes" validate:"omitempty,max=500"`
	InvoiceIDs      []string `json:"invoice_ids" validate:"omitempty,dive,uuid"`
}

// MultiPaymentResponse berisi hasil pembayaran multi-invoice.
type MultiPaymentResponse struct {
	Allocations    []PaymentAllocation `json:"allocations"`
	TotalAllocated int64               `json:"total_allocated"`
	ExcessToCredit int64               `json:"excess_to_credit"`
	ReceiptNumber  string              `json:"receipt_number"`
	ReceiptID      string              `json:"receipt_id"`
}

// PayAllRequest adalah payload untuk POST /v1/payments/pay-all.
type PayAllRequest struct {
	CustomerID      string `json:"customer_id" validate:"required,uuid"`
	PaymentMethod   string `json:"payment_method" validate:"required,oneof=tunai transfer lainnya"`
	PaymentDate     string `json:"payment_date" validate:"required,datetime=2006-01-02"`
	ReferenceNumber string `json:"reference_number" validate:"omitempty"`
	Notes           string `json:"notes" validate:"omitempty,max=500"`
}

// VoidPaymentRequest adalah payload untuk POST /v1/payments/:payment_id/void.
type VoidPaymentRequest struct {
	Reason string `json:"reason" validate:"required,min=5,max=500"`
}

// VoidPaymentResponse berisi hasil void pembayaran.
type VoidPaymentResponse struct {
	PaymentID        string        `json:"payment_id"`
	InvoiceID        string        `json:"invoice_id"`
	VoidedAmount     int64         `json:"voided_amount"`
	NewPaidAmount    int64         `json:"new_paid_amount"`
	NewInvoiceStatus InvoiceStatus `json:"new_invoice_status"`
	CreditReduced    int64         `json:"credit_reduced"`
}

// ReceiptInvoice merepresentasikan satu invoice dalam kwitansi.
type ReceiptInvoice struct {
	InvoiceNumber string `json:"invoice_number"`
	Amount        int64  `json:"amount"`
}

// ReceiptData berisi data kwitansi pembayaran untuk cetak thermal.
type ReceiptData struct {
	ReceiptNumber  string           `json:"receipt_number"`
	TenantName     string           `json:"tenant_name"`
	PaymentDate    time.Time        `json:"payment_date"`
	CustomerName   string           `json:"customer_name"`
	CustomerIDSeq  string           `json:"customer_id_seq"`
	Invoices       []ReceiptInvoice `json:"invoices"`
	TotalAmount    int64            `json:"total_amount"`
	PaymentMethod  string           `json:"payment_method"`
	RecordedByName string           `json:"recorded_by_name"`
	Voided         bool             `json:"voided"`
	VoidReason     string           `json:"void_reason,omitempty"`
}

// BulkImportResult berisi hasil per baris dalam bulk import pembayaran.
type BulkImportResult struct {
	Row           int    `json:"row"`
	Status        string `json:"status"`
	ReceiptNumber string `json:"receipt_number,omitempty"`
	Reason        string `json:"reason,omitempty"`
}

// BulkImportResponse berisi hasil keseluruhan bulk import pembayaran CSV.
type BulkImportResponse struct {
	TotalRows         int                `json:"total_rows"`
	SuccessCount      int                `json:"success_count"`
	FailureCount      int                `json:"failure_count"`
	DuplicatesSkipped int                `json:"duplicates_skipped"`
	Results           []BulkImportResult `json:"results"`
}

// =============================================================================
// PendingSyncRepository — operasi data untuk tabel pending_syncs
// =============================================================================

// PendingSyncRepository mendefinisikan operasi data untuk tabel pending_syncs.
type PendingSyncRepository interface {
	// Create membuat pending sync baru dan mengembalikan record yang dibuat.
	Create(ctx context.Context, sync *PendingSync) (*PendingSync, error)
	// GetByID mengambil pending sync berdasarkan ID.
	GetByID(ctx context.Context, id string) (*PendingSync, error)
	// UpdateStatus memperbarui status pending sync.
	UpdateStatus(ctx context.Context, id string, status SyncStatus) error
	// UpdateRetry memperbarui retry_count, next_retry_at, dan error_message.
	UpdateRetry(ctx context.Context, id string, retryCount int, nextRetryAt time.Time, errMsg string) error
	// MarkCompleted menandai pending sync sebagai selesai (status completed).
	MarkCompleted(ctx context.Context, id string) error
	// MarkFailed menandai pending sync sebagai gagal (status failed) dengan pesan error.
	MarkFailed(ctx context.Context, id string, errMsg string) error
	// FindPendingForRetry mengambil pending_syncs yang siap di-retry (status pending, next_retry_at <= now).
	FindPendingForRetry(ctx context.Context, batchSize int) ([]*PendingSync, error)
	// FindByCustomer mengambil pending_syncs untuk customer tertentu.
	FindByCustomer(ctx context.Context, customerID string) ([]*PendingSync, error)
	// FindByTenantAndStatus mengambil pending_syncs berdasarkan tenant dan status (paginated).
	FindByTenantAndStatus(ctx context.Context, tenantID string, status *SyncStatus, page, pageSize int) (*PendingSyncListResult, error)
	// ResetRetryForCustomer mereset retry_count ke 0 untuk customer tertentu.
	ResetRetryForCustomer(ctx context.Context, customerID string) error
	// ResetRetryAll mereset retry_count ke 0 untuk semua pending/failed records di tenant.
	ResetRetryAll(ctx context.Context, tenantID string) (int, error)
	// CountByTenantAndStatuses menghitung jumlah pending_syncs berdasarkan tenant dan status.
	CountByTenantAndStatuses(ctx context.Context, tenantID string, statuses []SyncStatus) (int64, error)
}

// =============================================================================
// ExpenseRepository — operasi data untuk tabel expenses
// =============================================================================

// ExpenseRepository mendefinisikan operasi data untuk tabel expenses.
// Diimplementasikan oleh repository.ExpenseRepo.
type ExpenseRepository interface {
	// Create membuat pengeluaran baru dan mengembalikan pengeluaran yang dibuat.
	Create(ctx context.Context, expense *Expense) (*Expense, error)
	// GetByID mengambil pengeluaran berdasarkan ID (tenant-scoped via RLS).
	GetByID(ctx context.Context, id string) (*Expense, error)
	// Update memperbarui data pengeluaran dan mengembalikan pengeluaran yang diperbarui.
	Update(ctx context.Context, expense *Expense) (*Expense, error)
	// SoftDelete menghapus pengeluaran secara soft delete (set deleted_at).
	SoftDelete(ctx context.Context, id string) error
	// List mengambil daftar pengeluaran dengan filter periode dan kategori.
	List(ctx context.Context, tenantID string, periodStart, periodEnd time.Time, categoryID string) ([]*Expense, error)
	// ListRecurring mengambil semua pengeluaran berulang yang aktif (untuk auto-create bulanan).
	ListRecurring(ctx context.Context) ([]*Expense, error)
	// SumByCategory menghitung total pengeluaran per kategori untuk laporan laba rugi.
	SumByCategory(ctx context.Context, tenantID string, periodStart, periodEnd time.Time) ([]ProfitLossLineItem, error)
}

// =============================================================================
// ExpenseCategoryRepository — operasi data untuk tabel expense_categories
// =============================================================================

// ExpenseCategoryRepository mendefinisikan operasi data untuk tabel expense_categories.
// Diimplementasikan oleh repository.ExpenseCategoryRepo.
type ExpenseCategoryRepository interface {
	// Create membuat kategori pengeluaran baru dan mengembalikan kategori yang dibuat.
	Create(ctx context.Context, category *ExpenseCategory) (*ExpenseCategory, error)
	// GetByID mengambil kategori pengeluaran berdasarkan ID (tenant-scoped via RLS).
	GetByID(ctx context.Context, id string) (*ExpenseCategory, error)
	// Update memperbarui data kategori dan mengembalikan kategori yang diperbarui.
	Update(ctx context.Context, category *ExpenseCategory) (*ExpenseCategory, error)
	// SoftDelete menghapus kategori secara soft delete (set deleted_at).
	SoftDelete(ctx context.Context, id string) error
	// List mengambil semua kategori pengeluaran aktif untuk tenant.
	List(ctx context.Context, tenantID string) ([]*ExpenseCategory, error)
	// NameExists mengecek apakah nama kategori sudah ada di tenant (exclude ID tertentu).
	NameExists(ctx context.Context, tenantID, name, excludeID string) (bool, error)
	// ExpenseCount menghitung jumlah pengeluaran aktif dalam kategori.
	ExpenseCount(ctx context.Context, id string) (int, error)
	// CreateDefaults membuat kategori default untuk tenant baru.
	CreateDefaults(ctx context.Context, tenantID string) error
}

// =============================================================================
// KPITargetRepository — operasi data untuk tabel kpi_targets
// =============================================================================

// KPITargetRepository mendefinisikan operasi data untuk tabel kpi_targets.
// Diimplementasikan oleh repository.KPITargetRepo.
type KPITargetRepository interface {
	// GetByTenant mengambil target KPI berdasarkan tenant ID.
	GetByTenant(ctx context.Context, tenantID string) (*KPITarget, error)
	// Upsert membuat atau memperbarui target KPI untuk tenant (INSERT ON CONFLICT DO UPDATE).
	Upsert(ctx context.Context, target *KPITarget) (*KPITarget, error)
}

// =============================================================================
// ReportScheduleRepository — operasi data untuk tabel report_schedules
// =============================================================================

// ReportScheduleRepository mendefinisikan operasi data untuk tabel report_schedules.
// Diimplementasikan oleh repository.ReportScheduleRepo.
type ReportScheduleRepository interface {
	// Create membuat jadwal laporan baru dan mengembalikan jadwal yang dibuat.
	Create(ctx context.Context, schedule *ReportSchedule) (*ReportSchedule, error)
	// GetByID mengambil jadwal laporan berdasarkan ID.
	GetByID(ctx context.Context, id string) (*ReportSchedule, error)
	// Update memperbarui konfigurasi jadwal dan mengembalikan jadwal yang diperbarui.
	Update(ctx context.Context, schedule *ReportSchedule) (*ReportSchedule, error)
	// Deactivate menonaktifkan jadwal laporan (set is_active = false).
	Deactivate(ctx context.Context, id string) error
	// ListByTenant mengambil semua jadwal laporan aktif untuk tenant.
	ListByTenant(ctx context.Context, tenantID string) ([]*ReportSchedule, error)
	// ListDue mengambil jadwal yang perlu dijalankan berdasarkan tipe jadwal.
	ListDue(ctx context.Context, scheduleType ScheduleType) ([]*ReportSchedule, error)
}

// =============================================================================
// ReportJobRepository — operasi data untuk tabel report_jobs
// =============================================================================

// ReportJobRepository mendefinisikan operasi data untuk tabel report_jobs.
// Diimplementasikan oleh repository.ReportJobRepo.
type ReportJobRepository interface {
	// Create membuat job export baru dan mengembalikan job yang dibuat.
	Create(ctx context.Context, job *ReportJob) (*ReportJob, error)
	// GetByID mengambil job export berdasarkan ID.
	GetByID(ctx context.Context, id string) (*ReportJob, error)
	// UpdateStatus memperbarui status job beserta download URL dan pesan error.
	UpdateStatus(ctx context.Context, id string, status ReportJobStatus, downloadURL, errMsg string) error
	// CleanupOld menghapus job yang lebih lama dari waktu yang ditentukan.
	CleanupOld(ctx context.Context, olderThan time.Time) error
}

// =============================================================================
// CustomReportTemplateRepository — operasi data untuk tabel custom_report_templates
// =============================================================================

// CustomReportTemplateRepository mendefinisikan operasi data untuk tabel custom_report_templates.
// Diimplementasikan oleh repository.CustomReportTemplateRepo.
type CustomReportTemplateRepository interface {
	// Create membuat template laporan custom baru dan mengembalikan template yang dibuat.
	Create(ctx context.Context, template *CustomReportTemplate) (*CustomReportTemplate, error)
	// GetByID mengambil template laporan berdasarkan ID.
	GetByID(ctx context.Context, id string) (*CustomReportTemplate, error)
	// Delete menghapus template laporan secara permanen.
	Delete(ctx context.Context, id string) error
	// ListByTenant mengambil semua template laporan untuk tenant.
	ListByTenant(ctx context.Context, tenantID string) ([]*CustomReportTemplate, error)
}

// =============================================================================
// ReportAggregationRepository — query aggregasi kompleks untuk laporan
// =============================================================================

// ReportAggregationRepository mendefinisikan query aggregasi untuk laporan.
// Ini adalah repository khusus yang menjalankan complex SQL queries
// untuk menghasilkan data laporan dari berbagai tabel.
type ReportAggregationRepository interface {
	// --- Financial ---

	// GetRevenueSummary mengambil ringkasan pendapatan per sumber untuk periode tertentu.
	GetRevenueSummary(ctx context.Context, tenantID string, periodStart, periodEnd time.Time, areaID, packageID string) (*RevenueSource, error)
	// GetMonthlyRevenueTrend mengambil trend pendapatan bulanan untuk N bulan terakhir.
	GetMonthlyRevenueTrend(ctx context.Context, tenantID string, months int) ([]MonthlyRevenueTrend, error)
	// GetAgingReport mengambil laporan piutang/aging dengan bucket, collection rate, dan top debtors.
	GetAgingReport(ctx context.Context, tenantID string, periodEnd time.Time, areaID, packageID string) (*AgingReport, error)
	// GetPaymentDistribution mengambil distribusi pembayaran per metode dan harian.
	GetPaymentDistribution(ctx context.Context, tenantID string, periodStart, periodEnd time.Time, areaID, packageID string) (*PaymentReport, error)
	// GetVoucherRevenue mengambil laporan pendapatan voucher per paket dan reseller.
	GetVoucherRevenue(ctx context.Context, tenantID string, periodStart, periodEnd time.Time) (*VoucherRevenueReport, error)
	// GetRevenueByArea mengambil laporan pendapatan per area.
	GetRevenueByArea(ctx context.Context, tenantID string, periodStart, periodEnd time.Time) (*RevenueByAreaReport, error)

	// --- Customer ---

	// GetCustomerGrowth mengambil data pertumbuhan pelanggan untuk periode tertentu.
	GetCustomerGrowth(ctx context.Context, tenantID string, periodStart, periodEnd time.Time) (*CustomerGrowthReport, error)
	// GetMonthlyGrowthTrend mengambil trend pertumbuhan pelanggan bulanan untuk N bulan terakhir.
	GetMonthlyGrowthTrend(ctx context.Context, tenantID string, months int) ([]MonthlyGrowthTrend, error)
	// GetCustomerDistribution mengambil distribusi pelanggan per paket, area, status, dan metode koneksi.
	GetCustomerDistribution(ctx context.Context, tenantID string, periodEnd time.Time) (*CustomerDistributionReport, error)
	// GetChurnAnalysis mengambil analisis churn pelanggan per alasan, paket, dan area.
	GetChurnAnalysis(ctx context.Context, tenantID string, periodStart, periodEnd time.Time) (*ChurnAnalysisReport, error)

	// --- Operational ---

	// GetAdminActivity mengambil laporan aktivitas admin/user dari audit logs.
	GetAdminActivity(ctx context.Context, tenantID string, periodStart, periodEnd time.Time) (*ActivityReport, error)

	// --- Dashboard ---

	// GetDashboardData mengambil data ringkasan untuk dashboard widget.
	GetDashboardData(ctx context.Context, tenantID string) (*DashboardData, error)

	// --- Custom Report ---

	// GetCustomReportData mengambil data laporan custom berdasarkan metrik dan dimensi yang dipilih.
	GetCustomReportData(ctx context.Context, tenantID string, metrics []string, groupBy, subGroupBy string, periodStart, periodEnd time.Time) (interface{}, error)

	// --- Forecast Data ---

	// GetMonthlyRevenueHistory mengambil data historis pendapatan bulanan untuk linear regression.
	GetMonthlyRevenueHistory(ctx context.Context, tenantID string, months int) ([]DataPoint, error)
	// GetMonthlyCustomerHistory mengambil data historis jumlah pelanggan bulanan untuk linear regression.
	GetMonthlyCustomerHistory(ctx context.Context, tenantID string, months int) ([]DataPoint, error)
	// GetMonthlyReceivablesHistory mengambil data historis piutang bulanan untuk linear regression.
	GetMonthlyReceivablesHistory(ctx context.Context, tenantID string, months int) ([]DataPoint, error)
}

// =============================================================================
// NetworkServiceClient — HTTP client untuk komunikasi dengan network-service
// =============================================================================

// NetworkServiceClient mendefinisikan interface untuk komunikasi dengan network-service.
// Diimplementasikan oleh usecase.NetworkClient dengan graceful degradation
// (cache fallback jika service down, module_inactive jika modul belum aktif).
type NetworkServiceClient interface {
	// GetUptimeReport mengambil laporan uptime router dari network-service.
	GetUptimeReport(ctx context.Context, tenantID string, periodStart, periodEnd time.Time, routerID string) (*UptimeReport, error)
	// GetTrafficReport mengambil laporan traffic jaringan dari network-service.
	GetTrafficReport(ctx context.Context, tenantID string, periodStart, periodEnd time.Time, routerID string) (*TrafficReport, error)
	// GetSignalQualityReport mengambil laporan kualitas signal OLT dari network-service.
	GetSignalQualityReport(ctx context.Context, tenantID string, periodStart, periodEnd time.Time, oltID string) (*SignalQualityReport, error)
	// GetCapacityReport mengambil laporan kapasitas jaringan dari network-service.
	GetCapacityReport(ctx context.Context, tenantID string) (*CapacityReport, error)
	// GetSyncReport mengambil laporan status sync MikroTik dan OLT dari network-service.
	GetSyncReport(ctx context.Context, tenantID string, periodStart, periodEnd time.Time) (*SyncReport, error)
	// GetNotificationReport mengambil laporan statistik notifikasi dari network-service.
	GetNotificationReport(ctx context.Context, tenantID string, periodStart, periodEnd time.Time) (*NotificationReport, error)
}

// =============================================================================
// ReportUsecase — business logic untuk semua laporan
// =============================================================================

// ReportUsecase mendefinisikan business logic untuk laporan.
// Diimplementasikan oleh usecase.ReportManager.
type ReportUsecase interface {
	// --- Financial ---

	// GetRevenueReport mengambil laporan ringkasan pendapatan per sumber.
	GetRevenueReport(ctx context.Context, tenantID string, filter ReportFilter) (*RevenueReport, error)
	// GetAgingReport mengambil laporan piutang/aging dengan bucket dan top debtors.
	GetAgingReport(ctx context.Context, tenantID string, periodEnd time.Time, areaID, packageID string) (*AgingReport, error)
	// GetPaymentReport mengambil laporan distribusi pembayaran per metode.
	GetPaymentReport(ctx context.Context, tenantID string, filter ReportFilter) (*PaymentReport, error)
	// GetVoucherRevenueReport mengambil laporan pendapatan voucher per paket dan reseller.
	GetVoucherRevenueReport(ctx context.Context, tenantID string, filter ReportFilter) (*VoucherRevenueReport, error)
	// GetProfitLossReport mengambil laporan laba rugi sederhana.
	GetProfitLossReport(ctx context.Context, tenantID string, filter ReportFilter) (*ProfitLossReport, error)
	// GetRevenueByAreaReport mengambil laporan pendapatan per area.
	GetRevenueByAreaReport(ctx context.Context, tenantID string, filter ReportFilter) (*RevenueByAreaReport, error)

	// --- Customer ---

	// GetCustomerGrowthReport mengambil laporan pertumbuhan pelanggan.
	GetCustomerGrowthReport(ctx context.Context, tenantID string, filter ReportFilter) (*CustomerGrowthReport, error)
	// GetCustomerDistributionReport mengambil laporan distribusi pelanggan per paket/area/status.
	GetCustomerDistributionReport(ctx context.Context, tenantID string, periodEnd time.Time) (*CustomerDistributionReport, error)
	// GetChurnAnalysisReport mengambil laporan analisis churn pelanggan.
	GetChurnAnalysisReport(ctx context.Context, tenantID string, filter ReportFilter) (*ChurnAnalysisReport, error)

	// --- Network (via network-service) ---

	// GetUptimeReport mengambil laporan uptime router dari network-service.
	GetUptimeReport(ctx context.Context, tenantID string, filter ReportFilter) (*UptimeReport, error)
	// GetTrafficReport mengambil laporan traffic jaringan dari network-service.
	GetTrafficReport(ctx context.Context, tenantID string, filter ReportFilter) (*TrafficReport, error)
	// GetSignalQualityReport mengambil laporan kualitas signal OLT dari network-service.
	GetSignalQualityReport(ctx context.Context, tenantID string, filter ReportFilter) (*SignalQualityReport, error)
	// GetCapacityReport mengambil laporan kapasitas jaringan dari network-service.
	GetCapacityReport(ctx context.Context, tenantID string) (*CapacityReport, error)

	// --- Operational ---

	// GetActivityReport mengambil laporan aktivitas admin/user.
	GetActivityReport(ctx context.Context, tenantID string, filter ReportFilter) (*ActivityReport, error)
	// GetNotificationReport mengambil laporan statistik notifikasi.
	GetNotificationReport(ctx context.Context, tenantID string, filter ReportFilter) (*NotificationReport, error)
	// GetSyncReport mengambil laporan status sync MikroTik dan OLT.
	GetSyncReport(ctx context.Context, tenantID string, filter ReportFilter) (*SyncReport, error)

	// --- Dashboard ---

	// GetDashboardData mengambil data ringkasan untuk dashboard widget.
	GetDashboardData(ctx context.Context, tenantID string) (*DashboardData, error)

	// --- Export ---

	// RequestExport membuat job export laporan secara async dan mengembalikan job ID.
	RequestExport(ctx context.Context, tenantID, userID, reportType, format string, filters ReportFilter) (string, error)
	// GetExportStatus mengambil status job export berdasarkan job ID.
	GetExportStatus(ctx context.Context, jobID string) (*ReportJob, error)
}

// =============================================================================
// ExpenseUsecase — business logic untuk pengeluaran
// =============================================================================

// ExpenseUsecase mendefinisikan business logic untuk pengeluaran.
// Diimplementasikan oleh usecase.ExpenseManager.
type ExpenseUsecase interface {
	// Create membuat pengeluaran baru.
	Create(ctx context.Context, tenantID string, req CreateExpenseRequest, actor ActorInfo) (*Expense, error)
	// GetByID mengambil pengeluaran berdasarkan ID.
	GetByID(ctx context.Context, id string) (*Expense, error)
	// Update memperbarui data pengeluaran.
	Update(ctx context.Context, id string, req UpdateExpenseRequest, actor ActorInfo) (*Expense, error)
	// Delete menghapus pengeluaran secara soft delete.
	Delete(ctx context.Context, id string, actor ActorInfo) error
	// List mengambil daftar pengeluaran dengan filter periode dan kategori.
	List(ctx context.Context, tenantID string, periodStart, periodEnd time.Time, categoryID string) ([]*Expense, error)
	// ListCategories mengambil semua kategori pengeluaran aktif untuk tenant.
	ListCategories(ctx context.Context, tenantID string) ([]*ExpenseCategory, error)
	// CreateCategory membuat kategori pengeluaran baru (cek duplikasi nama).
	CreateCategory(ctx context.Context, tenantID, name string) (*ExpenseCategory, error)
	// UpdateCategory memperbarui nama kategori pengeluaran.
	UpdateCategory(ctx context.Context, id, name string) (*ExpenseCategory, error)
	// DeleteCategory menghapus kategori (ditolak jika masih ada pengeluaran terkait).
	DeleteCategory(ctx context.Context, id string) error
}

// =============================================================================
// ScheduleUsecase — business logic untuk jadwal laporan otomatis
// =============================================================================

// ScheduleUsecase mendefinisikan business logic untuk jadwal laporan.
// Diimplementasikan oleh usecase.ScheduleManager.
type ScheduleUsecase interface {
	// Create membuat jadwal laporan baru.
	Create(ctx context.Context, tenantID string, req CreateScheduleRequest, actor ActorInfo) (*ReportSchedule, error)
	// Update memperbarui konfigurasi jadwal laporan.
	Update(ctx context.Context, id string, req UpdateScheduleRequest) (*ReportSchedule, error)
	// Delete menonaktifkan jadwal laporan.
	Delete(ctx context.Context, id string) error
	// List mengambil semua jadwal laporan aktif untuk tenant.
	List(ctx context.Context, tenantID string) ([]*ReportSchedule, error)
}

// =============================================================================
// KPITargetUsecase — business logic untuk target KPI
// =============================================================================

// KPITargetUsecase mendefinisikan business logic untuk target KPI.
// Diimplementasikan oleh usecase layer (bisa langsung wrap KPITargetRepository).
type KPITargetUsecase interface {
	// Get mengambil target KPI untuk tenant.
	Get(ctx context.Context, tenantID string) (*KPITarget, error)
	// Upsert membuat atau memperbarui target KPI untuk tenant.
	Upsert(ctx context.Context, tenantID string, req UpdateKPITargetRequest) (*KPITarget, error)
}

// =============================================================================
// CustomReportTemplateUsecase — business logic untuk template laporan custom
// =============================================================================

// CustomReportTemplateUsecase mendefinisikan business logic untuk template custom.
// Diimplementasikan oleh usecase.CustomReportBuilder.
type CustomReportTemplateUsecase interface {
	// PreviewCustomReport menjalankan laporan custom tanpa menyimpan template.
	PreviewCustomReport(ctx context.Context, tenantID string, metrics []string, groupBy, subGroupBy string, periodStart, periodEnd time.Time, displayType string) (interface{}, error)
	// CreateTemplate menyimpan konfigurasi laporan custom sebagai template.
	CreateTemplate(ctx context.Context, tenantID string, req CreateTemplateRequest, actor ActorInfo) (*CustomReportTemplate, error)
	// DeleteTemplate menghapus template laporan custom.
	DeleteTemplate(ctx context.Context, id string) error
	// ListTemplates mengambil semua template laporan custom untuk tenant.
	ListTemplates(ctx context.Context, tenantID string) ([]*CustomReportTemplate, error)
}

// =============================================================================
// ForecastUsecase — business logic untuk proyeksi/forecasting
// =============================================================================

// ForecastUsecase mendefinisikan business logic untuk proyeksi bisnis.
// Diimplementasikan oleh usecase.ForecastEngine.
type ForecastUsecase interface {
	// GetForecastReport mengambil proyeksi 3 bulan ke depan berdasarkan data historis 6 bulan.
	GetForecastReport(ctx context.Context, tenantID string) (*ForecastReport, error)
}

// =============================================================================
// ComparisonUsecase — business logic untuk perbandingan antar periode
// =============================================================================

// ComparisonUsecase mendefinisikan business logic untuk perbandingan periode.
// Diimplementasikan oleh usecase.ComparisonEngine.
type ComparisonUsecase interface {
	// GetComparisonReport mengambil laporan perbandingan metrik antara dua periode.
	GetComparisonReport(ctx context.Context, tenantID string, compType ComparisonType, basePeriodStart, basePeriodEnd time.Time, comparePeriodStart, comparePeriodEnd *time.Time) (*ComparisonReport, error)
}
