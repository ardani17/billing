package domain

import (
	"crypto/rand"
	"errors"
	"fmt"
	"strings"
	"time"
)

// =============================================================================
// Voucher Status State Machine
// =============================================================================

// VoucherStatus mendefinisikan status voucher dalam sistem.
type VoucherStatus string

const (
	VoucherStatusTersedia VoucherStatus = "tersedia"
	VoucherStatusTerjual  VoucherStatus = "terjual"
	VoucherStatusAktif    VoucherStatus = "aktif"
	VoucherStatusSelesai  VoucherStatus = "selesai"
	VoucherStatusExpired  VoucherStatus = "expired"
	VoucherStatusVoid     VoucherStatus = "void"
)

// ValidVoucherTransitions mendefinisikan transisi status voucher yang valid.
// Key: status asal, Value: daftar status tujuan yang diizinkan.
var ValidVoucherTransitions = map[VoucherStatus][]VoucherStatus{
	VoucherStatusTersedia: {VoucherStatusTerjual, VoucherStatusVoid},
	VoucherStatusTerjual:  {VoucherStatusAktif, VoucherStatusExpired, VoucherStatusVoid},
	VoucherStatusAktif:    {VoucherStatusSelesai},
	VoucherStatusSelesai:  {}, // terminal state
	VoucherStatusExpired:  {}, // terminal state
	VoucherStatusVoid:     {}, // terminal state
}

// CanVoucherTransition memeriksa apakah transisi dari current ke target valid.
func CanVoucherTransition(current, target VoucherStatus) bool {
	targets, ok := ValidVoucherTransitions[current]
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

// VoucherTransition melakukan transisi status dan mengembalikan status baru.
// Mengembalikan error jika transisi tidak valid.
func VoucherTransition(current, target VoucherStatus) (VoucherStatus, error) {
	if CanVoucherTransition(current, target) {
		return target, nil
	}
	allowed := AllowedVoucherTargets(current)
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
	return current, fmt.Errorf("%w: %s", ErrInvalidVoucherTransition, msg)
}

// AllowedVoucherTargets mengembalikan daftar status tujuan yang valid dari status saat ini.
func AllowedVoucherTargets(current VoucherStatus) []VoucherStatus {
	targets, ok := ValidVoucherTransitions[current]
	if !ok {
		return nil
	}
	return targets
}

// =============================================================================
// Code Format — format karakter kode voucher
// =============================================================================

// CodeFormat mendefinisikan format karakter kode voucher.
type CodeFormat string

const (
	CodeFormatDigits  CodeFormat = "digits"
	CodeFormatLetters CodeFormat = "letters"
	CodeFormatMixed   CodeFormat = "mixed"
)

// charsetForFormat mengembalikan charset berdasarkan CodeFormat.
func charsetForFormat(format CodeFormat) string {
	switch format {
	case CodeFormatDigits:
		return "0123456789"
	case CodeFormatLetters:
		return "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	case CodeFormatMixed:
		return "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	default:
		return "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	}
}

// =============================================================================
// Transaction Type — jenis transaksi keuangan reseller
// =============================================================================

// TransactionType mendefinisikan jenis transaksi reseller.
type TransactionType string

const (
	TransactionDeposit  TransactionType = "deposit"
	TransactionPurchase TransactionType = "purchase"
	TransactionRefund   TransactionType = "refund"
	TransactionWithdraw TransactionType = "withdraw"
)

// =============================================================================
// Voucher Entity
// =============================================================================

// Voucher merepresentasikan satu kode voucher internet.
type Voucher struct {
	ID                    string        `json:"id"`
	TenantID              string        `json:"tenant_id"`
	Code                  string        `json:"code"`
	PackageID             string        `json:"package_id"`
	PackageName           string        `json:"package_name,omitempty"` // joined field
	ResellerID            string        `json:"reseller_id,omitempty"`
	ResellerName          string        `json:"reseller_name,omitempty"` // joined field
	Status                VoucherStatus `json:"status"`
	SellPriceSnapshot     *int64        `json:"sell_price_snapshot,omitempty"`
	ResellerPriceSnapshot *int64        `json:"reseller_price_snapshot,omitempty"`
	PurchasedAt           *time.Time    `json:"purchased_at,omitempty"`
	ActivatedAt           *time.Time    `json:"activated_at,omitempty"`
	ExpiresAt             *time.Time    `json:"expires_at,omitempty"`
	VoidedAt              *time.Time    `json:"voided_at,omitempty"`
	CreatedAt             time.Time     `json:"created_at"`
	UpdatedAt             time.Time     `json:"updated_at"`
}

// =============================================================================
// Voucher Audit Log Entity — catatan lifecycle voucher (append-only)
// =============================================================================

// VoucherAuditLog merepresentasikan catatan lifecycle voucher (append-only).
type VoucherAuditLog struct {
	ID        string                 `json:"id"`
	TenantID  string                 `json:"tenant_id"`
	VoucherID string                 `json:"voucher_id"`
	Action    string                 `json:"action"`
	ActorID   string                 `json:"actor_id"`
	ActorName string                 `json:"actor_name"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
}

// =============================================================================
// Reseller Transaction Entity — catatan transaksi keuangan reseller
// =============================================================================

// ResellerTransaction merepresentasikan satu transaksi keuangan reseller.
type ResellerTransaction struct {
	ID            string          `json:"id"`
	TenantID      string          `json:"tenant_id"`
	ResellerID    string          `json:"reseller_id"`
	Type          TransactionType `json:"type"`
	Amount        int64           `json:"amount"`
	BalanceBefore int64           `json:"balance_before"`
	BalanceAfter  int64           `json:"balance_after"`
	ReferenceID   string          `json:"reference_id,omitempty"`
	Notes         string          `json:"notes,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
}

// =============================================================================
// Domain Error Variables — error khusus domain voucher
// =============================================================================

var (
	// ErrVoucherNotFound dikembalikan saat voucher tidak ditemukan atau milik tenant lain
	ErrVoucherNotFound = errors.New("voucher tidak ditemukan")

	// ErrInvalidVoucherTransition dikembalikan saat transisi status voucher tidak valid
	ErrInvalidVoucherTransition = errors.New("transisi status voucher tidak valid")

	// ErrInvalidPackageType dikembalikan saat package_id bukan tipe voucher
	ErrInvalidPackageType = errors.New("paket harus bertipe voucher")

	// ErrVoucherPackagePriceInvalid dikembalikan saat paket voucher belum punya harga reseller/jual valid.
	ErrVoucherPackagePriceInvalid = errors.New("harga paket voucher belum valid")

	// ErrVoucherStockInsufficient dikembalikan saat stok voucher tersedia tidak cukup untuk pembelian.
	ErrVoucherStockInsufficient = errors.New("stok voucher tersedia tidak mencukupi")
)

// =============================================================================
// Voucher Code Generation — menggunakan crypto/rand untuk keamanan kriptografis
// =============================================================================

// GenerateVoucherCode menghasilkan satu kode voucher acak menggunakan crypto/rand.
// Parameter format menentukan charset (digits/letters/mixed).
// Parameter length menentukan panjang bagian acak kode (tanpa prefix).
// Parameter prefix ditambahkan di depan kode.
// Mengembalikan kode lengkap (prefix + bagian acak).
func GenerateVoucherCode(format CodeFormat, length int, prefix string) (string, error) {
	charset := charsetForFormat(format)
	charsetLen := len(charset)

	// Alokasi buffer untuk bagian acak kode
	randomBytes := make([]byte, length)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("gagal membaca crypto/rand: %w", err)
	}

	code := make([]byte, length)
	for i := 0; i < length; i++ {
		code[i] = charset[int(randomBytes[i])%charsetLen]
	}

	return prefix + string(code), nil
}

// GenerateVoucherCodes menghasilkan batch kode voucher unik.
// Untuk setiap kode, jika terjadi collision dengan existingCodes, retry hingga maxRetries kali.
// Mengembalikan daftar kode yang berhasil di-generate dan jumlah yang gagal (collision persisten).
func GenerateVoucherCodes(format CodeFormat, length int, prefix string, quantity int, existingCodes map[string]struct{}, maxRetries int) ([]string, int) {
	codes := make([]string, 0, quantity)
	failed := 0

	// Set untuk melacak kode yang sudah di-generate dalam batch ini
	generated := make(map[string]struct{}, quantity)

	for i := 0; i < quantity; i++ {
		var code string
		success := false

		for attempt := 0; attempt <= maxRetries; attempt++ {
			var err error
			code, err = GenerateVoucherCode(format, length, prefix)
			if err != nil {
				// Jika crypto/rand gagal, skip kode ini
				break
			}

			// Cek collision dengan kode yang sudah ada di database
			if _, exists := existingCodes[code]; exists {
				continue
			}

			// Cek collision dengan kode yang sudah di-generate dalam batch ini
			if _, exists := generated[code]; exists {
				continue
			}

			// Kode unik ditemukan
			success = true
			break
		}

		if success {
			codes = append(codes, code)
			generated[code] = struct{}{}
			// Tambahkan ke existingCodes agar kode berikutnya tidak collision
			existingCodes[code] = struct{}{}
		} else {
			failed++
		}
	}

	return codes, failed
}
