// credit_note_usecase.go berisi business logic untuk manajemen credit notes.
// CreditNoteUsecase menangani pembuatan credit note dan penyesuaian saldo kredit pelanggan.
package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/pkg/queue"
	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// CreditNoteUsecase mengimplementasikan business logic untuk credit notes.
type CreditNoteUsecase struct {
	creditNoteRepo domain.CreditNoteRepository
	invoiceRepo    domain.InvoiceRepository
	auditRepo      domain.InvoiceAuditLogRepository
	sequenceRepo   domain.InvoiceSequenceRepository
	customerRepo   domain.CustomerRepository
	queueClient    *asynq.Client
	logger         zerolog.Logger
}

// NewCreditNoteUsecase membuat instance baru CreditNoteUsecase.
func NewCreditNoteUsecase(
	creditNoteRepo domain.CreditNoteRepository,
	invoiceRepo domain.InvoiceRepository,
	auditRepo domain.InvoiceAuditLogRepository,
	sequenceRepo domain.InvoiceSequenceRepository,
	customerRepo domain.CustomerRepository,
	queueClient *asynq.Client,
	logger zerolog.Logger,
) *CreditNoteUsecase {
	return &CreditNoteUsecase{
		creditNoteRepo: creditNoteRepo,
		invoiceRepo:    invoiceRepo,
		auditRepo:      auditRepo,
		sequenceRepo:   sequenceRepo,
		customerRepo:   customerRepo,
		queueClient:    queueClient,
		logger:         logger,
	}
}

// Buat membuat credit note baru.
// Alur: validasi invoice ada -> buat nomor credit note -> buat credit note ->
// jika apply_to_credit: tambah saldo kredit pelanggan -> tulis audit log.
func (uc *CreditNoteUsecase) Create(
	ctx context.Context,
	tenantID string,
	req domain.CreateCreditNoteRequest,
	actor domain.ActorInfo,
) (*domain.CreditNote, error) {
	// Validasi invoice ada
	invoice, err := uc.invoiceRepo.GetByID(ctx, req.InvoiceID)
	if err != nil {
		return nil, domain.ErrInvoiceNotFound
	}

	// Buat nomor credit note via sequence atomik
	now := time.Now()
	seq, err := uc.sequenceRepo.NextSequence(ctx, tenantID, now.Year(), int(now.Month()))
	if err != nil {
		return nil, fmt.Errorf("gagal generate nomor credit note: %w", err)
	}
	creditNoteNumber := domain.FormatCreditNoteNumber(now.Year(), int(now.Month()), seq)

	// Tentukan apakah apply_to_credit (bawaan: true)
	applyToCredit := true
	if req.ApplyToCredit != nil {
		applyToCredit = *req.ApplyToCredit
	}

	// Buat credit note
	cn := &domain.CreditNote{
		TenantID:         tenantID,
		CreditNoteNumber: creditNoteNumber,
		InvoiceID:        req.InvoiceID,
		Amount:           req.Amount,
		Reason:           req.Reason,
		ApplyToCredit:    applyToCredit,
		CreatedByID:      actor.ActorID,
		CreatedByName:    actor.ActorName,
	}

	created, err := uc.creditNoteRepo.Create(ctx, cn)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat credit note: %w", err)
	}

	// Jika apply_to_credit: tambah saldo kredit pelanggan secara atomik
	if applyToCredit {
		customer, err := uc.customerRepo.GetByID(ctx, invoice.CustomerID)
		if err != nil {
			uc.logger.Error().Err(err).
				Str("customer_id", invoice.CustomerID).
				Msg("gagal mengambil pelanggan untuk credit note")
		} else {
			customer.CreditBalance += req.Amount
			if _, err := uc.customerRepo.Update(ctx, customer); err != nil {
				uc.logger.Error().Err(err).
					Str("customer_id", invoice.CustomerID).
					Msg("gagal menambah saldo kredit pelanggan")
			}
		}
	}

	// Tulis audit log pada invoice yang direferensikan
	metadata := map[string]interface{}{
		"credit_note_id":     created.ID,
		"credit_note_number": creditNoteNumber,
		"amount":             req.Amount,
		"reason":             req.Reason,
		"apply_to_credit":    applyToCredit,
	}
	uc.writeCreditNoteAuditLog(ctx, tenantID, req.InvoiceID, "credit_note.created", actor, metadata)

	return created, nil
}

// writeCreditNoteAuditLog menulis audit log untuk operasi credit note.
func (uc *CreditNoteUsecase) writeCreditNoteAuditLog(
	ctx context.Context,
	tenantID, invoiceID, action string,
	actor domain.ActorInfo,
	metadata map[string]interface{},
) {
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
			Msg("gagal menulis credit note audit log")
	}
}

// publishCreditNoteEvent mempublikasikan event credit note ke Redis queue.
func (uc *CreditNoteUsecase) publishCreditNoteEvent(tenantID, eventType string, payload interface{}) {
	if uc.queueClient == nil {
		return
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		uc.logger.Error().Err(err).Str("event_type", eventType).Msg("gagal marshal credit note event")
		return
	}

	envelope := queue.TaskEnvelope{
		EventType: eventType,
		TenantID:  tenantID,
		Payload:   payloadJSON,
	}

	if err := queue.EnqueueTask(uc.queueClient, envelope); err != nil {
		uc.logger.Error().Err(err).Str("event_type", eventType).Msg("gagal publish credit note event")
	}
}
