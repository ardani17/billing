package domain

import (
	"crypto/rand"
	"errors"
	"fmt"
	"strings"
	"time"
)

// --- Customer Status State Machine ---

// CustomerStatus mendefinisikan status pelanggan dalam sistem.
type CustomerStatus string

const (
	CustomerStatusPending  CustomerStatus = "pending"
	CustomerStatusAktif    CustomerStatus = "aktif"
	CustomerStatusIsolir   CustomerStatus = "isolir"
	CustomerStatusSuspend  CustomerStatus = "suspend"
	CustomerStatusBerhenti CustomerStatus = "berhenti"
)

// ValidTransitions mendefinisikan transisi status yang valid.
// Key: status asal, Value: daftar status tujuan yang diizinkan.
var ValidTransitions = map[CustomerStatus][]CustomerStatus{
	CustomerStatusPending:  {CustomerStatusAktif},
	CustomerStatusAktif:    {CustomerStatusIsolir, CustomerStatusBerhenti},
	CustomerStatusIsolir:   {CustomerStatusAktif, CustomerStatusSuspend, CustomerStatusBerhenti},
	CustomerStatusSuspend:  {CustomerStatusAktif, CustomerStatusBerhenti},
	CustomerStatusBerhenti: {}, // terminal state
}

// CanTransition memeriksa apakah transisi dari current ke target valid.
func CanTransition(current, target CustomerStatus) bool {
	targets, ok := ValidTransitions[current]
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

// Transition melakukan transisi status dan mengembalikan status baru.
// Mengembalikan error jika transisi tidak valid.
func Transition(current, target CustomerStatus) (CustomerStatus, error) {
	if CanTransition(current, target) {
		return target, nil
	}
	allowed := AllowedTargets(current)
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
	return current, fmt.Errorf("%w: %s", ErrInvalidStatusTransition, msg)
}

// AllowedTargets mengembalikan daftar status tujuan yang valid dari status saat ini.
func AllowedTargets(current CustomerStatus) []CustomerStatus {
	targets, ok := ValidTransitions[current]
	if !ok {
		return nil
	}
	return targets
}

// --- Connection Method ---

// ConnectionMethod mendefinisikan metode koneksi internet pelanggan.
type ConnectionMethod string

const (
	ConnectionPPPoE       ConnectionMethod = "pppoe"
	ConnectionHotspot     ConnectionMethod = "hotspot"
	ConnectionDHCPBinding ConnectionMethod = "dhcp_binding"
	ConnectionStatic      ConnectionMethod = "static"
)

// --- Customer Entity ---

// Customer merepresentasikan pelanggan ISP yang dikelola oleh tenant.
type Customer struct {
	ID               string           `json:"id"`
	TenantID         string           `json:"tenant_id"`
	CustomerIDSeq    string           `json:"customer_id_seq"`
	Name             string           `json:"name"`
	Phone            string           `json:"phone"`
	Email            string           `json:"email,omitempty"`
	Address          string           `json:"address"`
	AreaID           string           `json:"area_id,omitempty"`
	AreaName         string           `json:"area_name,omitempty"`
	Latitude         float64          `json:"latitude"`
	Longitude        float64          `json:"longitude"`
	PackageID        string           `json:"package_id"`
	PackageName      string           `json:"package_name,omitempty"`
	ActivationDate   time.Time        `json:"activation_date"`
	DueDate          int              `json:"due_date"`
	ConnectionMethod ConnectionMethod `json:"connection_method"`
	PPPoEUsername    string           `json:"pppoe_username,omitempty"`
	PPPoEPassword    string           `json:"pppoe_password,omitempty"`
	MACAddress       string           `json:"mac_address,omitempty"`
	RouterID         string           `json:"router_id,omitempty"`
	ODPPort          string           `json:"odp_port,omitempty"`
	CreditBalance    int64            `json:"credit_balance"`
	Notes            string           `json:"notes,omitempty"`
	Status           CustomerStatus   `json:"status"`
	DeletedAt        *time.Time       `json:"deleted_at,omitempty"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
}

// --- Domain Error Variables ---

var (
	// ErrCustomerNotFound dikembalikan saat pelanggan tidak ditemukan atau milik tenant lain
	ErrCustomerNotFound = errors.New("pelanggan tidak ditemukan")

	// ErrPhoneDuplicate dikembalikan saat nomor telepon sudah terdaftar di tenant yang sama
	ErrPhoneDuplicate = errors.New("nomor telepon sudah terdaftar")

	// ErrInvalidStatusTransition dikembalikan saat transisi status tidak valid
	ErrInvalidStatusTransition = errors.New("transisi status tidak valid")

	// ErrConfirmationMismatch dikembalikan saat nama konfirmasi tidak cocok
	ErrConfirmationMismatch = errors.New("nama konfirmasi tidak cocok")

	// ErrSamePackage dikembalikan saat paket yang diminta sama dengan paket saat ini
	ErrSamePackage = errors.New("paket sama dengan paket saat ini")

	// ErrPackageNotFound dikembalikan saat package_id tidak ditemukan
	ErrPackageNotFound = errors.New("paket tidak ditemukan")

	// ErrCustomerDeleted dikembalikan saat pelanggan sudah di-soft-delete
	ErrCustomerDeleted = errors.New("pelanggan sudah dihapus")
)

// --- Helper Functions ---

// GenerateCustomerID menghasilkan customer_id_seq berdasarkan sequence terakhir.
// Format: PLG-001, PLG-002, ..., PLG-999, PLG-1000, ...
// Zero-padded minimal 3 digit, expand otomatis jika > 999.
func GenerateCustomerID(lastSeq int) string {
	next := lastSeq + 1
	if next < 1000 {
		return fmt.Sprintf("PLG-%03d", next)
	}
	return fmt.Sprintf("PLG-%d", next)
}

// GeneratePPPoEUsername menghasilkan username PPPoE dari nama dan customer ID.
// Format: {first-name-lowercase}-{customer-id-lowercase-no-dash}
// Contoh: "Ahmad Rizki" + "PLG-001" → "ahmad-plg001"
func GeneratePPPoEUsername(name, customerIDSeq string) string {
	fields := strings.Fields(name)
	firstName := ""
	if len(fields) > 0 {
		firstName = strings.ToLower(fields[0])
	}
	idPart := strings.ToLower(strings.ReplaceAll(customerIDSeq, "-", ""))
	return firstName + "-" + idPart
}

// GeneratePPPoEPassword menghasilkan password PPPoE acak 8 karakter alfanumerik.
func GeneratePPPoEPassword() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 8)
	randomBytes := make([]byte, 8)
	_, err := rand.Read(randomBytes)
	if err != nil {
		// Fallback: should never happen in practice
		panic("crypto/rand failed: " + err.Error())
	}
	for i := range b {
		b[i] = charset[int(randomBytes[i])%len(charset)]
	}
	return string(b)
}
