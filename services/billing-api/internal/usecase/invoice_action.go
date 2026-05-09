// invoice_action.go berisi struct InvoiceActionUsecase, constructor, Cancel, dan helper methods.
// InvoiceActionUsecase adalah struct terpisah dari InvoiceUsecase.
package usecase

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/pkg/queue"
	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// InvoiceActionUsecase mengimplementasikan business logic untuk aksi invoice.
type InvoiceActionUsecase struct {
	invoiceRepo  domain.InvoiceRepository
	itemRepo     domain.InvoiceItemRepository
	paymentRepo  domain.InvoicePaymentRepository
	auditRepo    domain.InvoiceAuditLogRepository
	settingsRepo domain.BillingSettingsRepository
	customerRepo domain.CustomerRepository
	pool         *pgxpool.Pool
	queueClient  *asynq.Client
	logger       zerolog.Logger
}

// NewInvoiceActionUsecase membuat instance baru InvoiceActionUsecase.
func NewInvoiceActionUsecase(
	invoiceRepo domain.InvoiceRepository,
	itemRepo domain.InvoiceItemRepository,
	paymentRepo domain.InvoicePaymentRepository,
	auditRepo domain.InvoiceAuditLogRepository,
	settingsRepo domain.BillingSettingsRepository,
	customerRepo domain.CustomerRepository,
	pool *pgxpool.Pool,
	queueClient *asynq.Client,
	logger zerolog.Logger,
) *InvoiceActionUsecase {
	return &InvoiceActionUsecase{
		invoiceRepo:  invoiceRepo,
		itemRepo:     itemRepo,
		paymentRepo:  paymentRepo,
		auditRepo:    auditRepo,
		settingsRepo: settingsRepo,
		customerRepo: customerRepo,
		pool:         pool,
		queueClient:  queueClient,
		logger:       logger,
	}
}

// Cancel membatalkan invoice dengan verifikasi konfirmasi.
// Alur: ambil invoice -> verifikasi status -> verifikasi konfirmasi ->
// kembalikan kredit jika ada -> transisi ke batal -> tulis audit log -> terbitkan event.
func (uc *InvoiceActionUsecase) Cancel(ctx context.Context, id string, req domain.CancelInvoiceRequest, actor domain.ActorInfo) (*domain.Invoice, error) {
	// Ambil invoice
	invoice, err := uc.invoiceRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Verifikasi status bisa dibatalkan (bukan lunas atau batal)
	if invoice.Status == domain.InvoiceStatusLunas || invoice.Status == domain.InvoiceStatusBatal {
		return nil, domain.ErrInvoiceNotCancellable
	}

	// Verifikasi confirmation_number cocok dengan invoice_number
	if req.ConfirmationNumber != invoice.InvoiceNumber {
		return nil, domain.ErrInvoiceConfirmationMismatch
	}

	// Kembalikan kredit ke customer jika credit_applied > 0
	if invoice.CreditApplied > 0 {
		if err := uc.adjustCreditBalance(ctx, invoice.CustomerID, invoice.CreditApplied); err != nil {
			return nil, fmt.Errorf("gagal mengembalikan kredit pelanggan: %w", err)
		}
	}

	// Transisi status ke batal dengan optimistic locking
	updated, err := uc.invoiceRepo.UpdateStatus(ctx, id, domain.InvoiceStatusBatal, invoice.Version)
	if err != nil {
		return nil, fmt.Errorf("gagal membatalkan invoice: %w", err)
	}

	// Tulis audit log dengan alasan pembatalan
	metadata := map[string]interface{}{
		"reason": req.Reason,
	}
	uc.writeAuditLog(ctx, invoice.TenantID, id, "invoice.cancelled", actor, metadata)

	// Terbitkan event invoice.cancelled
	uc.publishEvent(invoice.TenantID, "invoice.cancelled", domain.InvoiceCancelledPayload{
		InvoiceID:     id,
		TenantID:      invoice.TenantID,
		CustomerID:    invoice.CustomerID,
		InvoiceNumber: invoice.InvoiceNumber,
		Reason:        req.Reason,
	})

	return updated, nil
}

// writeAuditLog menulis audit log entry untuk invoice.
// Tidak mengembalikan error agar operasi utama tidak gagal karena audit log.
func (uc *InvoiceActionUsecase) writeAuditLog(ctx context.Context, tenantID, invoiceID, action string, actor domain.ActorInfo, metadata map[string]interface{}) {
	log := &domain.InvoiceAuditLog{
		TenantID:  tenantID,
		InvoiceID: invoiceID,
		Action:    action,
		ActorID:   actor.ActorID,
		ActorName: actor.ActorName,
		Metadata:  metadata,
	}

	if err := uc.auditRepo.Create(ctx, log); err != nil {
		uc.logger.Error().Err(err).
			Str("invoice_id", invoiceID).
			Str("action", action).
			Msg("gagal menulis invoice audit log")
	}
}

// publishEvent mempublikasikan event ke Redis queue.
// Tidak mengembalikan error agar operasi utama tidak gagal.
func (uc *InvoiceActionUsecase) publishEvent(tenantID, eventType string, payload interface{}) {
	if uc.queueClient == nil {
		return
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		uc.logger.Error().Err(err).Str("event_type", eventType).Msg("gagal marshal event payload")
		return
	}

	envelope := queue.TaskEnvelope{
		EventType: eventType,
		TenantID:  tenantID,
		Payload:   payloadJSON,
	}

	if err := queue.EnqueueTask(uc.queueClient, envelope); err != nil {
		uc.logger.Error().Err(err).Str("event_type", eventType).Msg("gagal publish event")
	}
}
