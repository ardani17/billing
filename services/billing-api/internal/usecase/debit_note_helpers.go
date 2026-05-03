// debit_note_helpers.go berisi helper methods untuk DebitNoteUsecase.
package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ispboss/ispboss/pkg/queue"
	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// createInvoiceFromDebitNote membuat invoice dari items debit note.
func (uc *DebitNoteUsecase) createInvoiceFromDebitNote(
	ctx context.Context,
	tenantID string,
	req domain.CreateDebitNoteRequest,
	dn *domain.DebitNote,
	dueDate time.Time,
	actor domain.ActorInfo,
) (string, error) {
	// Generate nomor invoice
	periodMonth := int(dueDate.Month())
	periodYear := dueDate.Year()

	settings, _ := uc.settingsRepo.GetByTenantID(ctx, tenantID)
	prefix := "INV"
	if settings != nil && settings.InvoicePrefix != "" {
		prefix = settings.InvoicePrefix
	}

	seq, err := uc.sequenceRepo.NextSequence(ctx, tenantID, periodYear, periodMonth)
	if err != nil {
		return "", fmt.Errorf("gagal generate nomor invoice: %w", err)
	}
	invoiceNumber := domain.FormatInvoiceNumber(prefix, periodYear, periodMonth, seq)

	// Buat invoice
	invoice := &domain.Invoice{
		TenantID:      tenantID,
		CustomerID:    req.CustomerID,
		InvoiceNumber: invoiceNumber,
		PeriodMonth:   periodMonth,
		PeriodYear:    periodYear,
		DueDate:       dueDate,
		Subtotal:      dn.TotalAmount,
		TotalAmount:   dn.TotalAmount,
		Status:        domain.InvoiceStatusBelumBayar,
		Notes:         fmt.Sprintf("Invoice dari debit note %s", dn.DebitNoteNumber),
		Version:       1,
	}

	created, err := uc.invoiceRepo.Create(ctx, invoice)
	if err != nil {
		return "", fmt.Errorf("gagal membuat invoice: %w", err)
	}

	// Buat invoice items dari debit note items
	invoiceItems := make([]*domain.InvoiceItem, 0, len(req.Items))
	for i, item := range req.Items {
		invoiceItems = append(invoiceItems, &domain.InvoiceItem{
			TenantID:    tenantID,
			InvoiceID:   created.ID,
			ItemType:    domain.ItemTypeCustom,
			Description: item.Description,
			Quantity:    1,
			UnitPrice:   item.Amount,
			Amount:      item.Amount,
			SortOrder:   i + 1,
		})
	}

	if _, err := uc.itemRepo.BulkCreate(ctx, invoiceItems); err != nil {
		return "", fmt.Errorf("gagal membuat invoice items: %w", err)
	}

	return created.ID, nil
}

// writeDebitNoteAuditLog menulis audit log untuk operasi debit note.
func (uc *DebitNoteUsecase) writeDebitNoteAuditLog(
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
			Msg("gagal menulis debit note audit log")
	}
}

// publishDebitNoteEvent mempublikasikan event debit note ke Redis queue.
func (uc *DebitNoteUsecase) publishDebitNoteEvent(tenantID, eventType string, payload interface{}) {
	if uc.queueClient == nil {
		return
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		uc.logger.Error().Err(err).Str("event_type", eventType).Msg("gagal marshal debit note event")
		return
	}

	envelope := queue.TaskEnvelope{
		EventType: eventType,
		TenantID:  tenantID,
		Payload:   payloadJSON,
	}

	if err := queue.EnqueueTask(uc.queueClient, envelope); err != nil {
		uc.logger.Error().Err(err).Str("event_type", eventType).Msg("gagal publish debit note event")
	}
}
