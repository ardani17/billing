package domain

import "time"

// UserRole mendefinisikan tipe role yang valid di sistem.
type UserRole string

const (
	RoleSuperAdmin  UserRole = "super_admin"
	RoleTenantAdmin UserRole = "tenant_admin"
	RoleOperator    UserRole = "operator"
	RoleTeknisi     UserRole = "teknisi"
	RoleKasir       UserRole = "kasir"
	RoleReseller    UserRole = "reseller"
)

// UserStatus mendefinisikan status user.
type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusInactive UserStatus = "inactive"
)

// User merepresentasikan pengguna sistem ISPBoss.
type User struct {
	// ID adalah UUID unik untuk user
	ID string `json:"id"`

	// TenantID adalah UUID tenant tempat user terdaftar
	TenantID string `json:"tenant_id"`

	// Name adalah nama lengkap user
	Name string `json:"name"`

	// Email adalah alamat email user
	Email string `json:"email"`

	// Phone adalah nomor telepon user (opsional)
	Phone string `json:"phone,omitempty"`

	// PasswordHash adalah hash bcrypt dari password user (tidak di-expose ke JSON)
	PasswordHash string `json:"-"`

	// Role adalah role user dalam sistem RBAC
	Role UserRole `json:"role"`

	// EmailVerified menunjukkan apakah email sudah diverifikasi
	EmailVerified bool `json:"email_verified"`

	// GoogleID adalah ID dari Google OAuth (tidak di-expose ke JSON)
	GoogleID string `json:"-"`

	// Status menunjukkan status user (active, inactive)
	Status UserStatus `json:"status"`

	// LastLogin adalah waktu terakhir user login (opsional)
	LastLogin *time.Time `json:"last_login,omitempty"`

	// CreatedAt adalah waktu pembuatan record
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt adalah waktu terakhir record diperbarui
	UpdatedAt time.Time `json:"updated_at"`
}
