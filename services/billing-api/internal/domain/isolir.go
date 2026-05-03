package domain

import (
	"errors"
	"time"
)

// =============================================================================
// Sync Operation Type — tipe operasi sinkronisasi router
// =============================================================================

// SyncOperationType mendefinisikan tipe operasi sinkronisasi router.
type SyncOperationType string

const (
	SyncOpIsolir   SyncOperationType = "isolir"
	SyncOpUnIsolir SyncOperationType = "un_isolir"
	SyncOpSuspend  SyncOperationType = "suspend"
)

// =============================================================================
// Sync Status — status operasi sinkronisasi
// =============================================================================

// SyncStatus mendefinisikan status operasi sinkronisasi.
type SyncStatus string

const (
	SyncStatusPending    SyncStatus = "pending"
	SyncStatusInProgress SyncStatus = "in_progress"
	SyncStatusCompleted  SyncStatus = "completed"
	SyncStatusFailed     SyncStatus = "failed"
)

// =============================================================================
// PendingSync Entity — operasi sinkronisasi router yang tertunda
// =============================================================================

// PendingSync merepresentasikan operasi sinkronisasi router yang tertunda.
type PendingSync struct {
	ID            string            `json:"id"`
	TenantID      string            `json:"tenant_id"`
	CustomerID    string            `json:"customer_id"`
	OperationType SyncOperationType `json:"operation_type"`
	Status        SyncStatus        `json:"status"`
	RetryCount    int               `json:"retry_count"`
	MaxRetries    int               `json:"max_retries"`
	LastRetryAt   *time.Time        `json:"last_retry_at,omitempty"`
	NextRetryAt   *time.Time        `json:"next_retry_at,omitempty"`
	ErrorMessage  string            `json:"error_message,omitempty"`
	Metadata      map[string]any    `json:"metadata,omitempty"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
	// Field gabungan (dari query JOIN)
	CustomerName  string `json:"customer_name,omitempty"`
	CustomerIDSeq string `json:"customer_id_seq,omitempty"`
}

// =============================================================================
// PendingSyncListResult — hasil paginasi daftar pending sync
// =============================================================================

// PendingSyncListResult merepresentasikan hasil paginasi daftar pending sync.
type PendingSyncListResult struct {
	Items      []*PendingSync `json:"items"`
	Total      int64          `json:"total"`
	Page       int            `json:"page"`
	PageSize   int            `json:"page_size"`
	TotalPages int            `json:"total_pages"`
}

// =============================================================================
// IsolirSummary — ringkasan statistik isolir untuk dashboard
// =============================================================================

// IsolirSummary merepresentasikan ringkasan statistik isolir untuk dashboard.
type IsolirSummary struct {
	TotalIsolir      int64 `json:"total_isolir"`
	TotalSuspend     int64 `json:"total_suspend"`
	TotalPendingSync int64 `json:"total_pending_sync"`
	RevenueAtRisk    int64 `json:"revenue_at_risk"`
}

// =============================================================================
// CalculateNextRetryAt — menghitung waktu retry berikutnya
// =============================================================================

// backoffDelays mendefinisikan delay untuk setiap retry.
// retry 0 = immediate, retry 1 = +5m, retry 2 = +30m, retry 3 = +2h, retry 4 = +6h.
var backoffDelays = []time.Duration{
	0,               // retry 0: langsung
	5 * time.Minute, // retry 1: 5 menit
	30 * time.Minute, // retry 2: 30 menit
	2 * time.Hour,   // retry 3: 2 jam
	6 * time.Hour,   // retry 4: 6 jam
}

// CalculateNextRetryAt menghitung waktu retry berikutnya berdasarkan retry_count.
// retryCount adalah jumlah retry yang sudah dilakukan (0-indexed).
// Mengembalikan now jika retryCount di luar range.
func CalculateNextRetryAt(retryCount int, now time.Time) time.Time {
	if retryCount < 0 || retryCount >= len(backoffDelays) {
		return now
	}
	return now.Add(backoffDelays[retryCount])
}

// =============================================================================
// Helper — fungsi bantu untuk perhitungan tanggal dan timezone
// =============================================================================

// LocalDate membungkus time.Time untuk representasi tanggal lokal tenant.
type LocalDate struct {
	Time time.Time
}

// currentDateInTimezone mengembalikan tanggal saat ini di timezone tenant.
// Fallback ke Asia/Jakarta jika timezone tidak valid.
func currentDateInTimezone(tz string) time.Time {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc, _ = time.LoadLocation("Asia/Jakarta")
	}
	return time.Now().In(loc)
}

// CurrentDateInTimezone mengembalikan tanggal saat ini di timezone tenant (exported).
// Fallback ke Asia/Jakarta jika timezone tidak valid.
func CurrentDateInTimezone(tz string) LocalDate {
	return LocalDate{Time: currentDateInTimezone(tz)}
}

// daysOverdue menghitung jumlah hari keterlambatan dari due_date.
// Mengembalikan 0 jika currentDate belum melewati dueDate.
func daysOverdue(dueDate time.Time, currentDate time.Time) int {
	diff := currentDate.Sub(dueDate)
	days := int(diff.Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
}

// DaysOverdue menghitung jumlah hari keterlambatan dari due_date (exported).
// Mengembalikan 0 jika currentDate belum melewati dueDate.
func DaysOverdue(dueDate time.Time, currentDate time.Time) int {
	return daysOverdue(dueDate, currentDate)
}

// =============================================================================
// Domain Error Variables — error khusus domain isolir
// =============================================================================

var (
	// ErrNoPendingSync dikembalikan saat customer tidak memiliki pending_sync record
	ErrNoPendingSync = errors.New("tidak ada pending sync untuk pelanggan ini")

	// ErrNoPenaltyToWaive dikembalikan saat invoice tidak memiliki item denda
	ErrNoPenaltyToWaive = errors.New("invoice tidak memiliki denda untuk dihapus")

	// ErrOutstandingInvoicesExist dikembalikan saat customer masih memiliki invoice belum lunas
	ErrOutstandingInvoicesExist = errors.New("pelanggan masih memiliki invoice yang belum lunas")
)
