package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// --- Tipe Paket ---

// PackageType mendefinisikan jenis paket internet.
type PackageType string

const (
	PackageTypeMonthly PackageType = "monthly"
	PackageTypePPPoE   PackageType = "pppoe"
	PackageTypeVoucher PackageType = "voucher"
)

func (t PackageType) IsMonthlyBilling() bool {
	return t == PackageTypeMonthly || t == PackageTypePPPoE
}

// BandwidthType mendefinisikan tipe bandwidth paket.
type BandwidthType string

const (
	BandwidthDedicated BandwidthType = "dedicated"
	BandwidthShared    BandwidthType = "shared"
)

// QuotaType mendefinisikan tipe kuota paket.
type QuotaType string

const (
	QuotaUnlimited    QuotaType = "unlimited"
	QuotaMonthlyQuota QuotaType = "monthly_quota"
	QuotaFUP          QuotaType = "fup"
	QuotaQuota        QuotaType = "quota" // khusus voucher
)

// QuotaAction mendefinisikan aksi saat kuota habis.
type QuotaAction string

const (
	QuotaActionThrottle   QuotaAction = "throttle"
	QuotaActionDisconnect QuotaAction = "disconnect"
)

// DurationUnit mendefinisikan satuan durasi paket voucher.
type DurationUnit string

const (
	DurationHours  DurationUnit = "hours"
	DurationDays   DurationUnit = "days"
	DurationWeeks  DurationUnit = "weeks"
	DurationMonths DurationUnit = "months"
)

// --- Entitas Paket ---

// Package merepresentasikan paket internet yang ditawarkan oleh tenant.
// Mendukung dua jenis: PPPoE/Static (bulanan) dan Hotspot/Voucher (durasi).
type Package struct {
	ID                  string      `json:"id"`
	TenantID            string      `json:"tenant_id"`
	Type                PackageType `json:"type"`
	Name                string      `json:"name"`
	Description         string      `json:"description,omitempty"`
	IsActive            bool        `json:"is_active"`
	DownloadMbps        int         `json:"download_mbps"`
	UploadMbps          int         `json:"upload_mbps"`
	BandwidthType       string      `json:"bandwidth_type,omitempty"`
	BurstDownloadMbps   *int        `json:"burst_download_mbps,omitempty"`
	BurstUploadMbps     *int        `json:"burst_upload_mbps,omitempty"`
	BurstThresholdMbps  *int        `json:"burst_threshold_mbps,omitempty"`
	BurstTimeSeconds    *int        `json:"burst_time_seconds,omitempty"`
	QuotaType           QuotaType   `json:"quota_type"`
	QuotaMB             *int        `json:"quota_mb,omitempty"`
	QuotaAction         string      `json:"quota_action,omitempty"`
	ThrottleMbps        *int        `json:"throttle_mbps,omitempty"`
	MonthlyPrice        *int64      `json:"monthly_price,omitempty"`
	InstallationFee     int64       `json:"installation_fee"`
	SellPrice           *int64      `json:"sell_price,omitempty"`
	ResellerPrice       *int64      `json:"reseller_price,omitempty"`
	DurationValue       *int        `json:"duration_value,omitempty"`
	DurationUnit        string      `json:"duration_unit,omitempty"`
	SharedUsers         int         `json:"shared_users"`
	MikrotikProfileName string      `json:"mikrotik_profile_name,omitempty"`
	AddressPool         string      `json:"address_pool,omitempty"`
	ParentQueue         string      `json:"parent_queue,omitempty"`
	HotspotProfileName  string      `json:"hotspot_profile_name,omitempty"`
	CustomerCount       int         `json:"customer_count,omitempty"` // field komputasi, tidak disimpan
	CreatedAt           time.Time   `json:"created_at"`
	UpdatedAt           time.Time   `json:"updated_at"`
}

// --- Error Domain Paket ---
// Catatan: ErrPackageNotFound sudah didefinisikan di domain/customer.go
// Catatan: ErrConfirmationMismatch sudah didefinisikan di domain/customer.go

var (
	// ErrPackageNameDuplicate dikembalikan saat nama paket sudah ada di tenant yang sama
	ErrPackageNameDuplicate = errors.New("nama paket sudah terdaftar")

	// ErrPackageHasCustomers dikembalikan saat paket masih digunakan pelanggan (hard delete)
	ErrPackageHasCustomers = errors.New("paket masih digunakan pelanggan")

	// ErrPackageHasVouchers dikembalikan saat paket masih memiliki voucher terkait.
	ErrPackageHasVouchers = errors.New("paket masih memiliki voucher")

	// ErrPackageAlreadyActive dikembalikan saat mengaktifkan paket yang sudah aktif
	ErrPackageAlreadyActive = errors.New("paket sudah aktif")

	// ErrPackageAlreadyInactive dikembalikan saat menonaktifkan paket yang sudah nonaktif
	ErrPackageAlreadyInactive = errors.New("paket sudah nonaktif")

	// ErrInsufficientMargin dikembalikan saat margin reseller < 500 Rupiah
	ErrInsufficientMargin = errors.New("margin reseller tidak mencukupi")

	// ErrTypeChangeNotAllowed dikembalikan saat mencoba mengubah tipe paket setelah dibuat
	ErrTypeChangeNotAllowed = errors.New("tipe paket tidak dapat diubah setelah dibuat")

	// ErrBurstFieldsIncomplete dikembalikan saat burst fields tidak lengkap (harus semua atau tidak ada)
	ErrBurstFieldsIncomplete = errors.New("field burst harus diisi semua atau tidak sama sekali")
)

// --- Fungsi Helper ---

// ValidateResellerMargin memvalidasi margin reseller pada paket voucher.
// Mengembalikan error jika reseller_price >= sell_price atau margin < 500.
func ValidateResellerMargin(sellPrice, resellerPrice int64) error {
	margin := sellPrice - resellerPrice
	if resellerPrice >= sellPrice || margin < 500 {
		return fmt.Errorf("%w: margin saat ini Rp %d, minimum Rp 500", ErrInsufficientMargin, margin)
	}
	return nil
}

// GenerateProfileName menghasilkan nama profile MikroTik/Hotspot dari nama paket.
// Format: lowercase, spasi diganti dengan tanda hubung.
// Contoh: "Pro 50M" → "pro-50m"
func GenerateProfileName(name string) string {
	return strings.ToLower(strings.ReplaceAll(name, " ", "-"))
}

// GenerateDuplicateName menghasilkan nama paket duplikat yang unik.
// Format: "{original} (Copy)", "{original} (Copy 2)", dst.
// Parameter existingNames digunakan untuk menghindari collision.
func GenerateDuplicateName(originalName string, existingNames []string) string {
	existing := make(map[string]struct{}, len(existingNames))
	for _, n := range existingNames {
		existing[n] = struct{}{}
	}

	// Coba nama pertama: "{original} (Copy)"
	candidate := originalName + " (Copy)"
	if _, found := existing[candidate]; !found {
		return candidate
	}

	// Coba nama berikutnya: "{original} (Copy 2)", "(Copy 3)", dst.
	for i := 2; ; i++ {
		candidate = fmt.Sprintf("%s (Copy %d)", originalName, i)
		if _, found := existing[candidate]; !found {
			return candidate
		}
	}
}

// ValidateBurstFields memvalidasi bahwa burst fields diisi semua atau tidak sama sekali.
// Mengembalikan ErrBurstFieldsIncomplete jika hanya sebagian field yang diisi.
func ValidateBurstFields(burstDown, burstUp, burstThreshold, burstTime *int) error {
	count := 0
	if burstDown != nil {
		count++
	}
	if burstUp != nil {
		count++
	}
	if burstThreshold != nil {
		count++
	}
	if burstTime != nil {
		count++
	}

	// Semua diisi (4) atau tidak ada (0) → valid
	if count == 0 || count == 4 {
		return nil
	}
	return ErrBurstFieldsIncomplete
}
