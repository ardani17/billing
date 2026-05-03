// webhook_usecase.go berisi struct WebhookUsecase, constructor, dan method ProcessWebhook.
// Method pemrosesan pembayaran (processPaymentPaid, processPaymentExpired, processPaymentFailed)
// ada di file terpisah (webhook_payment.go).
package usecase

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/ispboss/ispboss/services/billing-api/internal/gateway"
)

// WebhookUsecase mengimplementasikan business logic untuk pemrosesan webhook.
type WebhookUsecase struct {
	webhookRepo    domain.WebhookLogRepository
	linkRepo       domain.PaymentLinkRepository
	invoiceRepo    domain.InvoiceRepository
	paymentRepo    domain.InvoicePaymentRepository
	auditRepo      domain.InvoiceAuditLogRepository
	receiptSeqRepo domain.ReceiptSequenceRepository
	customerRepo   domain.CustomerRepository
	configRepo     domain.GatewayConfigRepository
	pool           *pgxpool.Pool
	queueClient    *asynq.Client
	masterKey      []byte // AES-256 master key untuk dekripsi webhook secret
	logger         zerolog.Logger
}

// NewWebhookUsecase membuat instance baru WebhookUsecase.
func NewWebhookUsecase(
	webhookRepo domain.WebhookLogRepository,
	linkRepo domain.PaymentLinkRepository,
	invoiceRepo domain.InvoiceRepository,
	paymentRepo domain.InvoicePaymentRepository,
	auditRepo domain.InvoiceAuditLogRepository,
	receiptSeqRepo domain.ReceiptSequenceRepository,
	customerRepo domain.CustomerRepository,
	configRepo domain.GatewayConfigRepository,
	pool *pgxpool.Pool,
	queueClient *asynq.Client,
	masterKey []byte,
	logger zerolog.Logger,
) *WebhookUsecase {
	return &WebhookUsecase{
		webhookRepo:    webhookRepo,
		linkRepo:       linkRepo,
		invoiceRepo:    invoiceRepo,
		paymentRepo:    paymentRepo,
		auditRepo:      auditRepo,
		receiptSeqRepo: receiptSeqRepo,
		customerRepo:   customerRepo,
		configRepo:     configRepo,
		pool:           pool,
		queueClient:    queueClient,
		masterKey:      masterKey,
		logger:         logger,
	}
}

// ProcessWebhook memproses webhook yang sudah di-log secara asinkron.
// Langkah: fetch log → lookup payment link → verify signature → check duplicate →
// acquire advisory lock → dispatch ke handler event → update status log.
func (uc *WebhookUsecase) ProcessWebhook(ctx context.Context, webhookLogID string) error {
	// 1. Ambil webhook log berdasarkan ID
	wlog, err := uc.webhookRepo.GetByID(ctx, webhookLogID)
	if err != nil {
		return fmt.Errorf("gagal mengambil webhook log %s: %w", webhookLogID, err)
	}

	// 2. Cari payment link berdasarkan external_id
	link, err := uc.linkRepo.GetByExternalID(ctx, wlog.ExternalID)
	if err != nil || link == nil {
		_ = uc.webhookRepo.UpdateStatus(ctx, webhookLogID, domain.WebhookFailed, "payment_link_not_found")
		uc.logger.Warn().Str("webhook_id", webhookLogID).Str("external_id", wlog.ExternalID).
			Msg("payment link tidak ditemukan untuk webhook")
		return nil
	}

	// 3. Identifikasi tenant dari payment link
	tenantID := link.TenantID

	// 4. Ambil konfigurasi gateway untuk tenant dan provider
	config, err := uc.configRepo.GetActiveByProvider(ctx, tenantID, wlog.GatewayProvider)
	if err != nil {
		_ = uc.webhookRepo.UpdateStatus(ctx, webhookLogID, domain.WebhookFailed, "gateway_config_not_found")
		return fmt.Errorf("gagal mengambil config gateway: %w", err)
	}

	// 5. Dekripsi webhook secret
	webhookSecret, err := gateway.DecryptAESGCM(config.WebhookSecretEncrypted, uc.masterKey)
	if err != nil {
		_ = uc.webhookRepo.UpdateStatus(ctx, webhookLogID, domain.WebhookFailed, "decryption_failed")
		return fmt.Errorf("gagal dekripsi webhook secret: %w", err)
	}

	// 6. Buat adapter berdasarkan provider
	plainKey, err := gateway.DecryptAESGCM(config.APIKeyEncrypted, uc.masterKey)
	if err != nil {
		_ = uc.webhookRepo.UpdateStatus(ctx, webhookLogID, domain.WebhookFailed, "decryption_failed")
		return fmt.Errorf("gagal dekripsi api key: %w", err)
	}
	adapter, err := gateway.NewAdapter(config.GatewayProvider, plainKey)
	if err != nil {
		_ = uc.webhookRepo.UpdateStatus(ctx, webhookLogID, domain.WebhookFailed, "invalid_provider")
		return fmt.Errorf("gagal membuat adapter: %w", err)
	}

	// 7. Verifikasi signature webhook
	headers := extractWebhookHeaders(wlog)
	valid, err := adapter.VerifyWebhookSignature(ctx, headers, wlog.RequestBody, webhookSecret)
	if err != nil || !valid {
		_ = uc.webhookRepo.UpdateSignatureValid(ctx, webhookLogID, false)
		_ = uc.webhookRepo.UpdateStatus(ctx, webhookLogID, domain.WebhookFailed, "signature_invalid")
		uc.logger.Warn().Str("webhook_id", webhookLogID).Msg("signature webhook tidak valid")
		return nil
	}
	_ = uc.webhookRepo.UpdateSignatureValid(ctx, webhookLogID, true)

	// 8. Parse payload webhook menjadi event
	event, err := adapter.ParseWebhookPayload(wlog.RequestBody)
	if err != nil {
		_ = uc.webhookRepo.UpdateStatus(ctx, webhookLogID, domain.WebhookFailed, "parse_failed: "+err.Error())
		return fmt.Errorf("gagal parse webhook payload: %w", err)
	}

	// 9. Cek duplikat — apakah webhook dengan external_id + event_type sudah diproses
	duplicate, err := uc.webhookRepo.IsAlreadyProcessed(ctx, wlog.ExternalID, event.EventType)
	if err != nil {
		return fmt.Errorf("gagal cek duplikat webhook: %w", err)
	}
	if duplicate {
		_ = uc.webhookRepo.UpdateStatus(ctx, webhookLogID, domain.WebhookDuplicate, "")
		uc.logger.Info().Str("webhook_id", webhookLogID).Msg("webhook duplikat, dilewati")
		return nil
	}

	// 10. Acquire pg_advisory_xact_lock pada payment link ID untuk mencegah race condition
	if err := uc.acquireAdvisoryLock(ctx, link.ID); err != nil {
		return fmt.Errorf("gagal acquire advisory lock: %w", err)
	}

	// 11. Dispatch ke handler berdasarkan event type
	switch event.EventType {
	case "payment.paid":
		if err := uc.processPaymentPaid(ctx, event, link); err != nil {
			_ = uc.webhookRepo.UpdateStatus(ctx, webhookLogID, domain.WebhookFailed, err.Error())
			return err
		}
	case "payment.expired":
		if err := uc.processPaymentExpired(ctx, event, link); err != nil {
			_ = uc.webhookRepo.UpdateStatus(ctx, webhookLogID, domain.WebhookFailed, err.Error())
			return err
		}
	case "payment.failed":
		if err := uc.processPaymentFailed(ctx, event, link); err != nil {
			_ = uc.webhookRepo.UpdateStatus(ctx, webhookLogID, domain.WebhookFailed, err.Error())
			return err
		}
	default:
		_ = uc.webhookRepo.UpdateStatus(ctx, webhookLogID, domain.WebhookFailed, "unknown_event: "+event.EventType)
		return nil
	}

	// 12. Update webhook log status menjadi processed
	_ = uc.webhookRepo.UpdateStatus(ctx, webhookLogID, domain.WebhookProcessed, "")
	uc.logger.Info().Str("webhook_id", webhookLogID).Str("event", event.EventType).
		Msg("webhook berhasil diproses")

	return nil
}

// acquireAdvisoryLock mengambil pg_advisory_xact_lock berdasarkan payment link ID.
// Menggunakan hash dari UUID string sebagai lock key.
// Jika pool nil (misalnya dalam unit test), lock dilewati.
func (uc *WebhookUsecase) acquireAdvisoryLock(ctx context.Context, linkID string) error {
	if uc.pool == nil {
		return nil
	}
	// Gunakan hashcode sederhana dari link ID sebagai advisory lock key
	var lockKey int64
	for _, c := range linkID {
		lockKey = lockKey*31 + int64(c)
	}
	_, err := uc.pool.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", lockKey)
	return err
}

// extractWebhookHeaders mengekstrak headers dari webhook log untuk verifikasi signature.
// Webhook handler menyimpan headers relevan (x-callback-token) di field _headers
// dalam request_body JSON saat logging. Midtrans tidak memerlukan headers (signature di body).
func extractWebhookHeaders(wlog *domain.WebhookLog) map[string]string {
	headers := make(map[string]string)
	// Parse request_body untuk mengekstrak _headers jika ada
	var body map[string]interface{}
	if err := json.Unmarshal(wlog.RequestBody, &body); err != nil {
		return headers
	}
	if h, ok := body["_headers"].(map[string]interface{}); ok {
		for k, v := range h {
			if s, ok := v.(string); ok {
				headers[k] = s
			}
		}
	}
	return headers
}
