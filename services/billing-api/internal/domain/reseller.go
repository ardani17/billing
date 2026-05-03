package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// --- Reseller Status State Machine ---

// ResellerStatus mendefinisikan status reseller dalam sistem.
type ResellerStatus string

const (
	ResellerStatusAktif     ResellerStatus = "aktif"
	ResellerStatusSuspended ResellerStatus = "suspended"
	ResellerStatusNonaktif  ResellerStatus = "nonaktif"
)

// ValidResellerTransitions mendefinisikan transisi status reseller yang valid.
// Key: status asal, Value: daftar status tujuan yang diizinkan.
var ValidResellerTransitions = map[ResellerStatus][]ResellerStatus{
	ResellerStatusAktif:     {ResellerStatusSuspended, ResellerStatusNonaktif},
	ResellerStatusSuspended: {ResellerStatusAktif, ResellerStatusNonaktif},
	ResellerStatusNonaktif:  {}, // terminal state
}

// CanResellerTransition memeriksa apakah transisi dari current ke target valid.
func CanResellerTransition(current, target ResellerStatus) bool {
	targets, ok := ValidResellerTransitions[current]
	if !ok {
		return false
	}
	for _, t := range targets {
		if t == target {
			return true
		}
	}
	return false
}

// ResellerTransition melakukan transisi status dan mengembalikan status baru.
// Mengembalikan error jika transisi tidak valid.
func ResellerTransition(current, target ResellerStatus) (ResellerStatus, error) {
	if CanResellerTransition(current, target) {
		return target, nil
	}
	allowed := AllowedResellerTargets(current)
	allowedStrs := make([]string, len(allowed))
	for i, a := range allowed {
		allowedStrs[i] = string(a)
	}
	var msg string
	if len(allowedStrs) == 0 {
		msg = fmt.Sprintf("transisi dari %s ke %s tidak diizinkan, transisi yang diizinkan: (tidak ada)", current, target)
	} else {
		msg = fmt.Sprintf("transisi dari %s ke %s tidak diizinkan, transisi yang diizinkan: %s", current, target, strings.Join(allowedStrs, ", "))
	}
	return current, fmt.Errorf("%w: %s", ErrInvalidResellerTransition, msg)
}

// AllowedResellerTargets mengembalikan daftar status tujuan yang valid dari status saat ini.
func AllowedResellerTargets(current ResellerStatus) []ResellerStatus {
	targets, ok := ValidResellerTransitions[current]
	if !ok {
		return nil
	}
	return targets
}

// --- Reseller Entity ---

// Reseller merepresentasikan reseller voucher yang dikelola oleh tenant.
type Reseller struct {
	ID                 string         `json:"id"`
	TenantID           string         `json:"tenant_id"`
	Name               string         `json:"name"`
	Phone              string         `json:"phone"`
	Email              string         `json:"email,omitempty"`
	Address            string         `json:"address,omitempty"`
	PasswordHash       string         `json:"-"` // tidak di-expose ke JSON
	Balance            int64          `json:"balance"`
	DailyPurchaseLimit int            `json:"daily_purchase_limit"`
	Status             ResellerStatus `json:"status"`
	LastLogin          *time.Time     `json:"last_login,omitempty"`
	TotalVouchersSold  int            `json:"total_vouchers_sold,omitempty"` // field komputasi, tidak disimpan
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
}

// --- Domain Error Variables ---
// Catatan: ErrConfirmationMismatch sudah didefinisikan di domain/customer.go

var (
	// ErrResellerNotFound dikembalikan saat reseller tidak ditemukan atau milik tenant lain
	ErrResellerNotFound = errors.New("reseller tidak ditemukan")

	// ErrResellerPhoneDuplicate dikembalikan saat nomor telepon sudah terdaftar di tenant yang sama
	ErrResellerPhoneDuplicate = errors.New("nomor telepon sudah terdaftar")

	// ErrResellerAccountDisabled dikembalikan saat akun reseller suspended/nonaktif
	ErrResellerAccountDisabled = errors.New("akun reseller dinonaktifkan")

	// ErrInvalidResellerTransition dikembalikan saat transisi status reseller tidak valid
	ErrInvalidResellerTransition = errors.New("transisi status reseller tidak valid")

	// ErrResellerInvalidCredentials dikembalikan saat phone/password salah saat login
	ErrResellerInvalidCredentials = errors.New("nomor telepon atau password salah")

	// ErrResellerAccountLocked dikembalikan saat akun terkunci karena terlalu banyak percobaan login
	ErrResellerAccountLocked = errors.New("akun terkunci sementara")

	// ErrInsufficientBalance dikembalikan saat saldo reseller tidak cukup untuk pembelian
	ErrInsufficientBalance = errors.New("saldo tidak mencukupi")

	// ErrDailyLimitExceeded dikembalikan saat batas pembelian harian reseller terlampaui
	ErrDailyLimitExceeded = errors.New("batas pembelian harian terlampaui")

	// ErrVoucherForbidden dikembalikan saat reseller mengakses voucher milik reseller lain
	ErrVoucherForbidden = errors.New("tidak memiliki akses ke voucher ini")

	// ErrPackageNotActive dikembalikan saat paket voucher tidak aktif
	ErrPackageNotActive = errors.New("paket tidak aktif")
)
