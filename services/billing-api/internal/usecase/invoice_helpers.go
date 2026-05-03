// invoice_helpers.go berisi helper methods untuk InvoiceUsecase.
package usecase

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ispboss/ispboss/pkg/queue"
	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// writeInvoiceAuditLog menulis audit log entry untuk invoice.
// Tidak mengembalikan error agar operasi utama tidak gagal karena audit log.
func (uc *InvoiceUsecase) writeInvoiceAuditLog(ctx context.Context, tenantID, invoiceID, action string, actor domain.ActorInfo, metadata map[string]interface{}) {
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
func (uc *InvoiceUsecase) publishEvent(tenantID, eventType string, payload interface{}) {
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


// adjustCreditBalance mengubah credit_balance pelanggan secara atomik menggunakan SQL langsung.
// delta positif = tambah kredit, delta negatif = kurangi kredit.
// Menggunakan UPDATE ... SET credit_balance = credit_balance + $1 untuk atomicity.
func (uc *InvoiceUsecase) adjustCreditBalance(ctx context.Context, customerID string, delta int64) error {
	if uc.pool == nil || delta == 0 {
		return nil
	}
	_, err := uc.pool.Exec(ctx,
		`UPDATE customers SET credit_balance = credit_balance + $1, updated_at = NOW() WHERE id = $2`,
		delta, customerID,
	)
	if err != nil {
		return fmt.Errorf("gagal update credit_balance: %w", err)
	}
	return nil
}

// adjustCreditBalance mengubah credit_balance pelanggan secara atomik menggunakan SQL langsung.
// delta positif = tambah kredit, delta negatif = kurangi kredit.
// Menggunakan UPDATE ... SET credit_balance = credit_balance + $1 untuk atomicity.
func (uc *InvoiceActionUsecase) adjustCreditBalance(ctx context.Context, customerID string, delta int64) error {
	if uc.pool == nil || delta == 0 {
		return nil
	}
	_, err := uc.pool.Exec(ctx,
		`UPDATE customers SET credit_balance = credit_balance + $1, updated_at = NOW() WHERE id = $2`,
		delta, customerID,
	)
	if err != nil {
		return fmt.Errorf("gagal update credit_balance: %w", err)
	}
	return nil
}
