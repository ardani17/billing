package domain

import (
	"context"
	"time"
)

// =============================================================================
// GatewayConfigRepository - operasi data untuk tabel payment_gateway_configs
// =============================================================================

// GatewayConfigRepository mendefinisikan operasi data untuk tabel payment_gateway_configs.
// Diimplementasikan oleh repositori.GatewayConfigRepo.
type GatewayConfigRepository interface {
	// Buat membuat konfigurasi gateway baru dan mengembalikan konfigurasi yang dibuat.
	Create(ctx context.Context, config *GatewayConfig) (*GatewayConfig, error)
	// GetByID mengambil konfigurasi gateway berdasarkan ID (tenant-scoped via RLS).
	GetByID(ctx context.Context, id string) (*GatewayConfig, error)
	// Perbarui memperbarui konfigurasi gateway dan mengembalikan konfigurasi yang diperbarui.
	Update(ctx context.Context, config *GatewayConfig) (*GatewayConfig, error)
	// Deactivate menonaktifkan konfigurasi gateway (hapus lunak, atur is_active=false).
	Deactivate(ctx context.Context, id string) error
	// ListByTenant mengambil semua konfigurasi gateway untuk tenant tertentu.
	ListByTenant(ctx context.Context, tenantID string) ([]*GatewayConfig, error)
	// GetActiveByTenant mengambil konfigurasi gateway aktif untuk tenant tertentu.
	GetActiveByTenant(ctx context.Context, tenantID string) ([]*GatewayConfig, error)
	// GetActiveByProvider mengambil konfigurasi gateway aktif berdasarkan provider untuk tenant.
	GetActiveByProvider(ctx context.Context, tenantID string, provider GatewayProvider) (*GatewayConfig, error)
	// ExistsByProvider mengecek apakah konfigurasi aktif sudah ada untuk provider di tenant.
	ExistsByProvider(ctx context.Context, tenantID string, provider GatewayProvider) (bool, error)
}

// =============================================================================
// PaymentLinkRepository - operasi data untuk tabel payment_links
// =============================================================================

// PaymentLinkRepository mendefinisikan operasi data untuk tabel payment_links.
// Diimplementasikan oleh repositori.PaymentLinkRepo.
type PaymentLinkRepository interface {
	// Buat membuat link pembayaran baru beserta junction ke invoices (payment_link_invoices).
	Create(ctx context.Context, link *PaymentLink, invoiceIDs []string) (*PaymentLink, error)
	// GetByID mengambil link pembayaran berdasarkan ID (tenant-scoped via RLS).
	GetByID(ctx context.Context, id string) (*PaymentLink, error)
	// GetByExternalID mengambil link pembayaran berdasarkan external_id dari gateway.
	GetByExternalID(ctx context.Context, externalID string) (*PaymentLink, error)
	// GetActiveByCustomer mengambil link pembayaran aktif (status='active') untuk customer.
	GetActiveByCustomer(ctx context.Context, customerID string) (*PaymentLink, error)
	// GetInvoiceIDsByLinkID mengambil daftar invoice ID yang terkait dengan link pembayaran.
	GetInvoiceIDsByLinkID(ctx context.Context, linkID string) ([]string, error)
	// UpdateStatus memperbarui status link pembayaran.
	UpdateStatus(ctx context.Context, id string, status PaymentLinkStatus) error
	// UpdateStatusPaid memperbarui status ke paid beserta metode pembayaran dan waktu bayar.
	UpdateStatusPaid(ctx context.Context, id string, paidMethod string, paidAt time.Time) error
	// ListByInvoice mengambil semua link pembayarans untuk invoice tertentu (via junction table).
	ListByInvoice(ctx context.Context, invoiceID string) ([]*PaymentLink, error)
	// FindExpired mengambil link pembayarans yang sudah melewati expires_at tapi masih active.
	FindExpired(ctx context.Context, batchSize int) ([]*PaymentLink, error)
	// ExpireByID mengubah status link pembayaran menjadi expired berdasarkan ID.
	ExpireByID(ctx context.Context, id string) error
}

// =============================================================================
// WebhookLogRepository - operasi data untuk tabel webhook_logs
// =============================================================================

// WebhookLogRepository mendefinisikan operasi data untuk tabel webhook_logs.
// Diimplementasikan oleh repositori.WebhookLogRepo.
type WebhookLogRepository interface {
	// Buat membuat log webhook baru dan mengembalikan log yang dibuat.
	Create(ctx context.Context, log *WebhookLog) (*WebhookLog, error)
	// GetByID mengambil webhook log berdasarkan ID.
	GetByID(ctx context.Context, id string) (*WebhookLog, error)
	// UpdateStatus memperbarui status pemrosesan dan pesan error webhook log.
	UpdateStatus(ctx context.Context, id string, status WebhookProcessingStatus, errMsg string) error
	// UpdateSignatureValid memperbarui flag signature_valid pada webhook log.
	UpdateSignatureValid(ctx context.Context, id string, valid bool) error
	// IsAlreadyProcessed mengecek apakah webhook dengan external_id dan event_type sudah diproses.
	IsAlreadyProcessed(ctx context.Context, externalID, eventType string) (bool, error)
	// ListByPaymentLink mengambil semua webhook logs berdasarkan external_id link pembayaran.
	ListByPaymentLink(ctx context.Context, externalID string) ([]*WebhookLog, error)
	// DeleteOlderThan menghapus webhook logs yang lebih tua dari waktu yang ditentukan.
	// Tidak menghapus logs dengan processing_status=failed atau signature_valid=false.
	// Mengembalikan jumlah baris yang dihapus.
	DeleteOlderThan(ctx context.Context, olderThan time.Time) (int64, error)
}
