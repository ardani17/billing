package domain

import (
	"errors"

	"github.com/gofiber/fiber/v2"
)

// --- DTO permintaan/respons ---

// RegisterRequest adalah payload untuk POST /v1/auth/register.
type RegisterRequest struct {
	// Name adalah nama lengkap pengguna (minimal 3 karakter)
	Name string `json:"name" validate:"required,min=3"`

	// Email adalah alamat email pengguna
	Email string `json:"email" validate:"required,email"`

	// Phone adalah nomor telepon pengguna (harus diawali +62)
	Phone string `json:"phone" validate:"required,startswith=+62"`

	// CompanyName adalah nama perusahaan/ISP yang didaftarkan
	CompanyName string `json:"company_name" validate:"required"`

	// Password adalah kata sandi (minimal 8 karakter)
	Password string `json:"password" validate:"required,min=8"`

	// PasswordConfirmation harus sama dengan Password
	PasswordConfirmation string `json:"password_confirmation" validate:"required,eqfield=Password"`

	// AgreeTerms harus bernilai true untuk menyetujui syarat dan ketentuan
	AgreeTerms bool `json:"agree_terms" validate:"required,eq=true"`
}

// RegisterResponse adalah respons untuk registrasi sukses.
type RegisterResponse struct {
	// UserID adalah UUID user yang baru dibuat
	UserID string `json:"user_id"`

	// TenantID adalah UUID tenant yang baru dibuat
	TenantID string `json:"tenant_id"`
}

// LoginRequest adalah payload untuk POST /v1/auth/login.
type LoginRequest struct {
	// Email adalah alamat email pengguna
	Email string `json:"email" validate:"required,email"`

	// Password adalah kata sandi pengguna
	Password string `json:"password" validate:"required"`

	// RememberMe jika true, token berlaku lebih lama (7 hari)
	RememberMe bool `json:"remember_me"`
}

// LoginResponse adalah respons untuk login sukses.
type LoginResponse struct {
	// AccessToken adalah JWT token untuk autentikasi
	AccessToken string `json:"access_token"`

	// RefreshToken adalah token untuk memperpanjang sesi
	RefreshToken string `json:"refresh_token"`

	// ExpiresIn adalah durasi berlaku access token dalam detik
	ExpiresIn int64 `json:"expires_in"`

	// User adalah data pengguna yang sedang login
	User *User `json:"user"`

	// RedirectPath adalah path redirect berdasarkan role pengguna
	RedirectPath string `json:"redirect_path"`
}

// GoogleLoginRequest adalah payload untuk POST /v1/auth/google.
type GoogleLoginRequest struct {
	// IDToken adalah token dari Google OAuth
	IDToken string `json:"id_token" validate:"required"`
}

// ResetPasswordRequest adalah payload untuk POST /v1/auth/reset-password.
type ResetPasswordRequest struct {
	// Token adalah token reset password yang dikirim via email
	Token string `json:"token" validate:"required"`

	// Password adalah kata sandi baru (minimal 8 karakter)
	Password string `json:"password" validate:"required,min=8"`
}

// ChangePasswordRequest adalah payload untuk POST /v1/settings/security/change-password.
type ChangePasswordRequest struct {
	// CurrentPassword adalah kata sandi saat ini
	CurrentPassword string `json:"current_password" validate:"required"`

	// NewPassword adalah kata sandi baru (minimal 8 karakter)
	NewPassword string `json:"new_password" validate:"required,min=8"`
}

// TokenPair berisi access token dan refresh token.
type TokenPair struct {
	// AccessToken adalah JWT token untuk autentikasi
	AccessToken string `json:"access_token"`

	// RefreshToken adalah token untuk memperpanjang sesi
	RefreshToken string `json:"refresh_token"`

	// ExpiresIn adalah durasi berlaku access token dalam detik
	ExpiresIn int64 `json:"expires_in"`
}

// CreateUserRequest adalah payload untuk POST /v1/settings/users.
type CreateUserRequest struct {
	// Name adalah nama lengkap user baru (minimal 3 karakter)
	Name string `json:"name" validate:"required,min=3"`

	// Email adalah alamat email user baru
	Email string `json:"email" validate:"required,email"`

	// Phone adalah nomor telepon user baru (opsional, harus diawali +62)
	Phone string `json:"phone" validate:"omitempty,startswith=+62"`

	// Password adalah kata sandi user baru (minimal 8 karakter)
	Password string `json:"password" validate:"required,min=8"`

	// Role adalah role user baru (operator, teknisi, kasir, reseller)
	Role UserRole `json:"role" validate:"required,oneof=operator teknisi kasir reseller"`
}

// UpdateUserRequest adalah payload untuk PUT /v1/settings/users/:id.
type UpdateUserRequest struct {
	// Name adalah nama lengkap user (opsional, minimal 3 karakter)
	Name string `json:"name" validate:"omitempty,min=3"`

	// Phone adalah nomor telepon user (opsional, harus diawali +62)
	Phone string `json:"phone" validate:"omitempty,startswith=+62"`

	// Role adalah role user (opsional, operator/teknisi/kasir/reseller)
	Role UserRole `json:"role" validate:"omitempty,oneof=operator teknisi kasir reseller"`
}

// ImpersonateRequest adalah payload untuk POST /v1/admin/impersonate.
type ImpersonateRequest struct {
	// TenantID adalah UUID tenant target impersonasi
	TenantID string `json:"tenant_id" validate:"required,uuid"`

	// UserID adalah UUID user target impersonasi
	UserID string `json:"user_id" validate:"required,uuid"`

	// Reason adalah alasan support yang wajib dicatat sebelum impersonasi
	Reason string `json:"reason" validate:"required,min=5"`
}

// --- Tipe respons API ---

// APIResponse adalah format standar respons API.
type APIResponse struct {
	// Success menunjukkan apakah permintaan berhasil
	Success bool `json:"success"`

	// Data berisi data respons jika sukses
	Data interface{} `json:"data,omitempty"`

	// Error berisi detail error jika gagal
	Error *APIError `json:"error,omitempty"`
}

// APIError adalah format standar error API.
type APIError struct {
	// Code adalah kode error (contoh: VALIDATION_ERROR, UNAUTHORIZED)
	Code string `json:"code"`

	// Message adalah pesan error yang bisa ditampilkan ke pengguna
	Message string `json:"message"`

	// Details berisi detail error per field untuk validation error
	Details []FieldError `json:"details,omitempty"`
}

// FieldError adalah detail error per field untuk validation error.
type FieldError struct {
	// Field adalah nama field yang tidak valid
	Field string `json:"field"`

	// Message adalah pesan error untuk field tersebut
	Message string `json:"message"`
}

// --- Variabel error domain ---

// Daftar error domain untuk auth.
var (
	// ErrEmailAlreadyExists dikembalikan saat email sudah terdaftar
	ErrEmailAlreadyExists = errors.New("email sudah terdaftar")

	// ErrInvalidCredentials dikembalikan saat email atau password salah
	ErrInvalidCredentials = errors.New("email atau password salah")

	// ErrEmailNotVerified dikembalikan saat email belum diverifikasi
	ErrEmailNotVerified = errors.New("email belum diverifikasi")

	// ErrAccountDisabled dikembalikan saat akun dinonaktifkan
	ErrAccountDisabled = errors.New("akun dinonaktifkan")

	// ErrAccountLocked dikembalikan saat akun terkunci karena terlalu banyak percobaan login
	ErrAccountLocked = errors.New("akun terkunci sementara")

	// ErrTokenExpired dikembalikan saat token sudah kedaluwarsa
	ErrTokenExpired = errors.New("token sudah kedaluwarsa")

	// ErrTokenAlreadyUsed dikembalikan saat token sudah pernah digunakan
	ErrTokenAlreadyUsed = errors.New("token sudah digunakan")

	// ErrTokenNotFound dikembalikan saat token tidak ditemukan di database
	ErrTokenNotFound = errors.New("token tidak ditemukan")

	// ErrUserNotFound dikembalikan saat user tidak ditemukan
	ErrUserNotFound = errors.New("user tidak ditemukan")

	// ErrForbidden dikembalikan saat user tidak memiliki akses
	ErrForbidden = errors.New("tidak memiliki akses")

	// ErrCannotDeleteSelf dikembalikan saat user mencoba menghapus akun sendiri
	ErrCannotDeleteSelf = errors.New("tidak bisa menghapus akun sendiri")

	// ErrCannotDeactivateSelf dikembalikan saat user mencoba menonaktifkan akun sendiri
	ErrCannotDeactivateSelf = errors.New("tidak bisa menonaktifkan akun sendiri")

	// ErrInvalidRole dikembalikan saat role yang diberikan tidak valid
	ErrInvalidRole = errors.New("role tidak valid")

	// ErrResendCooldown dikembalikan saat pengiriman ulang verifikasi terlalu cepat
	ErrResendCooldown = errors.New("tunggu sebelum kirim ulang")
)

// --- Fungsi bantu Functions ---

// ErrorResponse mengembalikan respons error JSON dengan format standar.
func ErrorResponse(c *fiber.Ctx, status int, code, message string, details ...FieldError) error {
	resp := APIResponse{
		Success: false,
		Error: &APIError{
			Code:    code,
			Message: message,
		},
	}
	if len(details) > 0 {
		resp.Error.Details = details
	}
	return c.Status(status).JSON(resp)
}

// SuccessResponse mengembalikan respons sukses JSON dengan format standar.
func SuccessResponse(c *fiber.Ctx, status int, data interface{}) error {
	return c.Status(status).JSON(APIResponse{
		Success: true,
		Data:    data,
	})
}
