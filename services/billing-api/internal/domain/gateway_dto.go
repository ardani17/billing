package domain

import (
	"encoding/json"
	"time"
)

// =============================================================================
// DTO: Konfigurasi Gateway - Permintaan/respons
// =============================================================================

// CreateGatewayConfigRequest adalah payload untuk POST /v1/settings/payment-gateways.
type CreateGatewayConfigRequest struct {
	GatewayProvider       string   `json:"gateway_provider" validate:"required,oneof=xendit midtrans"`
	APIKey                string   `json:"api_key" validate:"required,min=10"`
	WebhookSecret         string   `json:"webhook_secret" validate:"required,min=10"`
	EnabledMethods        []string `json:"enabled_methods" validate:"required,min=1,dive,required"`
	PaymentLinkExpiryDays *int     `json:"payment_link_expiry_days" validate:"omitempty,min=1,max=30"`
}

// UpdateGatewayConfigRequest adalah payload untuk PUT /v1/settings/payment-gateways/:id.
type UpdateGatewayConfigRequest struct {
	APIKey                string   `json:"api_key" validate:"omitempty,min=10"`
	WebhookSecret         string   `json:"webhook_secret" validate:"omitempty,min=10"`
	EnabledMethods        []string `json:"enabled_methods" validate:"omitempty,min=1,dive,required"`
	PaymentLinkExpiryDays *int     `json:"payment_link_expiry_days" validate:"omitempty,min=1,max=30"`
}

// =============================================================================
// DTO: Link pembayaran - Permintaan/respons
// =============================================================================

// GeneratePaymentLinkRequest adalah payload untuk task buat link pembayaran.
type GeneratePaymentLinkRequest struct {
	TenantID   string   `json:"tenant_id" validate:"required,uuid"`
	CustomerID string   `json:"customer_id" validate:"required,uuid"`
	InvoiceIDs []string `json:"invoice_ids" validate:"required,min=1,dive,uuid"`
}

// RegeneratePaymentLinkRequest adalah payload untuk POST /v1/customers/:customer_id/payment-link/regenerate.
type RegeneratePaymentLinkRequest struct {
	CustomerID string `json:"customer_id" validate:"required,uuid"`
}

// PaymentLinkResponse adalah respons dari gateway adapter setelah buat link pembayaran.
type PaymentLinkResponse struct {
	ExternalID string    `json:"external_id"`
	PaymentURL string    `json:"payment_url"`
	ExpiresAt  time.Time `json:"expires_at"`
}

type CustomerPaymentLinkResponse struct {
	PaymentLink  *PaymentLink      `json:"payment_link"`
	Invoices     []OpenInvoiceItem `json:"invoices"`
	TotalArrears int64             `json:"total_arrears"`
}

// =============================================================================
// DTO: Webhook - Event yang diparsing dari payload gateway
// =============================================================================

// WebhookEvent adalah hasil parsing webhook payload oleh adapter.
type WebhookEvent struct {
	EventType       string          `json:"event_type"`     // "payment.paid", "payment.expired", "payment.failed"
	ExternalID      string          `json:"external_id"`    // ID link pembayaran di gateway
	TransactionID   string          `json:"transaction_id"` // ID transaksi unik dari gateway (idempotency key)
	Amount          int64           `json:"amount"`
	PaidMethod      string          `json:"paid_method"` // e.g., "va_bca", "qris", "ewallet_gopay"
	GatewayProvider GatewayProvider `json:"gateway_provider"`
	RawPayload      json.RawMessage `json:"raw_payload"`
}

// =============================================================================
// DTO: Kueri Status Pembayaran
// =============================================================================

type InvoicePaymentLinksResponse struct {
	PaymentLinks []PaymentLink `json:"payment_links"`
}

// PaymentLinkWebhooksResponse adalah respons GET /v1/payment-links/:id/webhooks.
type PaymentLinkWebhooksResponse struct {
	Webhooks []WebhookLog `json:"webhooks"`
}

// =============================================================================
// DTO: Walled Garden - Info pembayaran untuk halaman captive portal
// =============================================================================

type WalledGardenPaymentInfo struct {
	PaymentURL   string            `json:"payment_url"`
	TotalArrears int64             `json:"total_arrears"`
	Invoices     []OpenInvoiceItem `json:"invoices"`
	CustomerName string            `json:"customer_name"`
}

// =============================================================================
// DTO: Health Periksa Gateway
// =============================================================================

// GatewayTestResult adalah respons POST /v1/settings/payment-gateways/:id/test.
type GatewayTestResult struct {
	Success      bool   `json:"success"`
	ErrorCode    string `json:"error_code,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
	LatencyMs    int64  `json:"latency_ms"`
}
