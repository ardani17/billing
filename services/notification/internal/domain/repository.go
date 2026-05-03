package domain

import (
	"context"
	"time"
)

// =============================================================================
// ConfigRepository — operasi data untuk tabel notification_configs
// =============================================================================

// ConfigRepository mendefinisikan operasi data untuk tabel notification_configs.
// Diimplementasikan oleh repository.ConfigRepo.
type ConfigRepository interface {
	// GetByTenant mengambil semua konfigurasi notifikasi untuk tenant tertentu.
	GetByTenant(ctx context.Context, tenantID string) ([]*NotificationConfig, error)

	// GetByTenantAndChannel mengambil konfigurasi notifikasi berdasarkan tenant dan channel.
	GetByTenantAndChannel(ctx context.Context, tenantID string, ch Channel) (*NotificationConfig, error)

	// Upsert membuat atau memperbarui konfigurasi notifikasi (insert jika belum ada, update jika sudah).
	Upsert(ctx context.Context, cfg *NotificationConfig) (*NotificationConfig, error)

	// GetSettings mengambil pengaturan umum notifikasi untuk tenant tertentu.
	GetSettings(ctx context.Context, tenantID string) (*ConfigSettings, error)

	// UpdateSettings memperbarui pengaturan umum notifikasi untuk tenant tertentu.
	UpdateSettings(ctx context.Context, tenantID string, s ConfigSettings) error
}

// =============================================================================
// TemplateRepository — operasi data untuk tabel notification_templates
// =============================================================================

// TemplateRepository mendefinisikan operasi data untuk tabel notification_templates.
// Diimplementasikan oleh repository.TemplateRepo.
type TemplateRepository interface {
	// Create membuat template notifikasi baru dan mengembalikan template yang dibuat.
	Create(ctx context.Context, t *NotificationTemplate) (*NotificationTemplate, error)

	// GetByID mengambil template notifikasi berdasarkan ID.
	GetByID(ctx context.Context, id string) (*NotificationTemplate, error)

	// GetBySlug mengambil template notifikasi berdasarkan tenant_id dan slug.
	GetBySlug(ctx context.Context, tenantID, slug string) (*NotificationTemplate, error)

	// GetByEventType mengambil template notifikasi berdasarkan tenant_id dan event_type.
	// Digunakan oleh delivery pipeline untuk resolusi template dari event.
	GetByEventType(ctx context.Context, tenantID, eventType string) (*NotificationTemplate, error)

	// Update memperbarui template notifikasi dan mengembalikan template yang diperbarui.
	Update(ctx context.Context, t *NotificationTemplate) (*NotificationTemplate, error)

	// SoftDelete menonaktifkan template dengan mengatur is_active menjadi false.
	// Hanya template custom (is_default=false) yang boleh dihapus.
	SoftDelete(ctx context.Context, id string) error

	// ListByTenant mengambil semua template notifikasi untuk tenant tertentu.
	ListByTenant(ctx context.Context, tenantID string) ([]*NotificationTemplate, error)

	// BulkCreate membuat beberapa template sekaligus (digunakan untuk seeding default templates).
	BulkCreate(ctx context.Context, templates []*NotificationTemplate) error

	// SlugExists mengecek apakah slug sudah ada di tenant (exclude ID tertentu untuk update).
	SlugExists(ctx context.Context, tenantID, slug, excludeID string) (bool, error)
}

// =============================================================================
// LogRepository — operasi data untuk tabel notification_logs
// =============================================================================

// LogRepository mendefinisikan operasi data untuk tabel notification_logs.
// Diimplementasikan oleh repository.LogRepo.
type LogRepository interface {
	// Create membuat catatan log notifikasi baru dan mengembalikan log yang dibuat.
	Create(ctx context.Context, log *NotificationLog) (*NotificationLog, error)

	// GetByID mengambil catatan log notifikasi berdasarkan ID.
	GetByID(ctx context.Context, id string) (*NotificationLog, error)

	// Update memperbarui catatan log notifikasi (status, retry_count, error_message, sent_at).
	Update(ctx context.Context, log *NotificationLog) error

	// List mengambil daftar log notifikasi dengan filter dan pagination.
	List(ctx context.Context, params LogListParams) (*LogListResult, error)

	// FindByDedupKey mencari log notifikasi berdasarkan dedup_key dalam jendela waktu tertentu.
	// Digunakan untuk pengecekan duplikasi sebelum pengiriman.
	FindByDedupKey(ctx context.Context, dedupKey string, withinHours int) (*NotificationLog, error)

	// CountTodayByCustomer menghitung jumlah notifikasi yang dikirim ke pelanggan hari ini.
	// Menggunakan timezone tenant untuk menentukan batas hari.
	CountTodayByCustomer(ctx context.Context, tenantID, customerID string, tz string) (int, error)

	// LastSentToCustomer mengambil waktu pengiriman terakhir ke pelanggan tertentu.
	// Digunakan untuk pengecekan cooldown antar pesan.
	LastSentToCustomer(ctx context.Context, tenantID, customerID string) (*time.Time, error)
}
