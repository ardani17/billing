// webhook_payment.go berisi method WebhookUsecase untuk pemrosesan event
// payment.expired dan payment.failed, serta helper terbitkan event.
package usecase

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ispboss/ispboss/pkg/queue"
	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// processPaymentExpired memproses event payment.expired dari webhook.
// Perbarui status link pembayaran menjadi expired tanpa mengubah status invoice.
func (uc *WebhookUsecase) processPaymentExpired(ctx context.Context, event *domain.WebhookEvent, link *domain.PaymentLink) error {
	if err := uc.linkRepo.UpdateStatus(ctx, link.ID, domain.PaymentLinkExpired); err != nil {
		return err
	}
	uc.logger.Info().
		Str("payment_link_id", link.ID).
		Str("external_id", link.ExternalID).
		Msg("payment link expired via webhook")
	return nil
}

// processPaymentFailed memproses event payment.failed dari webhook.
// Log kegagalan dan terbitkan event notifikasi ke customer.
func (uc *WebhookUsecase) processPaymentFailed(ctx context.Context, event *domain.WebhookEvent, link *domain.PaymentLink) error {
	uc.logger.Warn().
		Str("payment_link_id", link.ID).
		Str("external_id", link.ExternalID).
		Str("paid_method", event.PaidMethod).
		Msg("pembayaran online gagal")

	// Terbitkan event notifikasi kegagalan ke customer
	uc.publishWebhookEvent(link.TenantID, "payment.online.failed", map[string]interface{}{
		"tenant_id":        link.TenantID,
		"customer_id":      link.CustomerID,
		"payment_link_id":  link.ID,
		"gateway_provider": string(link.GatewayProvider),
		"amount":           link.Amount,
	})
	return nil
}

// publishWebhookEvent mempublikasikan event ke Redis queue.
// Tidak mengembalikan error agar operasi utama tidak gagal.
func (uc *WebhookUsecase) publishWebhookEvent(tenantID, eventType string, payload interface{}) {
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

// adjustCreditBalance mengubah credit_balance pelanggan secara atomik.
func (uc *WebhookUsecase) adjustCreditBalance(ctx context.Context, customerID string, delta int64) error {
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

// writeWebhookAuditLog menulis audit log entry untuk invoice dari webhook.
// Tidak mengembalikan error agar operasi utama tidak gagal.
func (uc *WebhookUsecase) writeWebhookAuditLog(ctx context.Context, tenantID, invoiceID, action string, metadata map[string]interface{}) {
	log := &domain.InvoiceAuditLog{
		TenantID:  tenantID,
		InvoiceID: invoiceID,
		Action:    action,
		ActorID:   "system",
		ActorName: "Payment Gateway",
		Metadata:  metadata,
	}
	if err := uc.auditRepo.Create(ctx, log); err != nil {
		uc.logger.Error().Err(err).
			Str("invoice_id", invoiceID).
			Str("action", action).
			Msg("gagal menulis invoice audit log")
	}
}
