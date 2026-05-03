// debit_note_usecase.go berisi business logic untuk manajemen debit notes.
// DebitNoteUsecase menangani pembuatan debit note dan opsional pembuatan invoice terkait.
package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// DebitNoteUsecase mengimplementasikan business logic untuk debit notes.
type DebitNoteUsecase struct {
	debitNoteRepo domain.DebitNoteRepository
	invoiceRepo   domain.InvoiceRepository
	itemRepo      domain.InvoiceItemRepository
	auditRepo     domain.InvoiceAuditLogRepository
	sequenceRepo  domain.InvoiceSequenceRepository
	customerRepo  domain.CustomerRepository
	settingsRepo  domain.BillingSettingsRepository
	queueClient   *asynq.Client
	logger        zerolog.Logger
}

// NewDebitNoteUsecase membuat instance baru DebitNoteUsecase.
func NewDebitNoteUsecase(
	debitNoteRepo domain.DebitNoteRepository,
	invoiceRepo domain.InvoiceRepository,
	itemRepo domain.InvoiceItemRepository,
	auditRepo domain.InvoiceAuditLogRepository,
	sequenceRepo domain.InvoiceSequenceRepository,
	customerRepo domain.CustomerRepository,
	settingsRepo domain.BillingSettingsRepository,
	queueClient *asynq.Client,
	logger zerolog.Logger,
) *DebitNoteUsecase {
	return &DebitNoteUsecase{
		debitNoteRepo: debitNoteRepo,
		invoiceRepo:   invoiceRepo,
		itemRepo:      itemRepo,
		auditRepo:     auditRepo,
		sequenceRepo:  sequenceRepo,
		customerRepo:  customerRepo,
		settingsRepo:  settingsRepo,
		queueClient:   queueClient,
		logger:        logger,
	}
}

// Create membuat debit note baru dengan items.
// Flow: validasi pelanggan ada → generate nomor debit note → buat debit note →
// jika create_invoice: buat invoice terkait → tulis audit log.
func (uc *DebitNoteUsecase) Create(
	ctx context.Context,
	tenantID string,
	req domain.CreateDebitNoteRequest,
	actor domain.ActorInfo,
) (*domain.DebitNote, error) {
	// Validasi pelanggan ada
	_, err := uc.customerRepo.GetByID(ctx, req.CustomerID)
	if err != nil {
		return nil, domain.ErrCustomerNotFound
	}

	// Parse due_date
	dueDate, err := time.Parse("2006-01-02", req.DueDate)
	if err != nil {
		return nil, fmt.Errorf("format due_date tidak valid: %w", err)
	}

	// Generate nomor debit note via sequence atomik
	now := time.Now()
	seq, err := uc.sequenceRepo.NextSequence(ctx, tenantID, now.Year(), int(now.Month()))
	if err != nil {
		return nil, fmt.Errorf("gagal generate nomor debit note: %w", err)
	}
	debitNoteNumber := domain.FormatDebitNoteNumber(now.Year(), int(now.Month()), seq)

	// Bangun items dan hitung total
	var totalAmount int64
	items := make([]domain.DebitNoteItem, 0, len(req.Items))
	for _, item := range req.Items {
		items = append(items, domain.DebitNoteItem{
			Description: item.Description,
			Amount:      item.Amount,
		})
		totalAmount += item.Amount
	}

	// Buat debit note
	dn := &domain.DebitNote{
		TenantID:        tenantID,
		DebitNoteNumber: debitNoteNumber,
		CustomerID:      req.CustomerID,
		DueDate:         dueDate,
		Items:           items,
		TotalAmount:     totalAmount,
		CreatedByID:     actor.ActorID,
		CreatedByName:   actor.ActorName,
	}

	created, err := uc.debitNoteRepo.Create(ctx, dn)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat debit note: %w", err)
	}

	// Jika create_invoice: buat invoice terkait dari items debit note
	if req.CreateInvoice {
		invoiceID, err := uc.createInvoiceFromDebitNote(ctx, tenantID, req, created, dueDate, actor)
		if err != nil {
			uc.logger.Error().Err(err).
				Str("debit_note_id", created.ID).
				Msg("gagal membuat invoice dari debit note")
		} else {
			created.InvoiceID = &invoiceID
		}
	}

	// Tulis audit log (menggunakan invoice_id jika ada, atau debit_note_id sebagai referensi)
	metadata := map[string]interface{}{
		"debit_note_id":     created.ID,
		"debit_note_number": debitNoteNumber,
		"total_amount":      totalAmount,
		"create_invoice":    req.CreateInvoice,
	}
	if created.InvoiceID != nil {
		uc.writeDebitNoteAuditLog(ctx, tenantID, *created.InvoiceID, "debit_note.created", actor, metadata)
	}

	return created, nil
}


