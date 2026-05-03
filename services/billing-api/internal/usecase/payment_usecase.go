// payment_usecase.go berisi struct PaymentUsecase, constructor, dan method list/summary/search.
// PaymentUsecase adalah struct terpisah dari InvoiceActionUsecase untuk single-responsibility.
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

// PaymentUsecase mengimplementasikan business logic untuk modul pembayaran manual.
type PaymentUsecase struct {
	invoiceRepo    domain.InvoiceRepository
	itemRepo       domain.InvoiceItemRepository
	paymentRepo    domain.InvoicePaymentRepository
	auditRepo      domain.InvoiceAuditLogRepository
	receiptSeqRepo domain.ReceiptSequenceRepository
	settingsRepo   domain.BillingSettingsRepository
	customerRepo   domain.CustomerRepository
	pool           *pgxpool.Pool
	queueClient    *asynq.Client
	logger         zerolog.Logger
}

// NewPaymentUsecase membuat instance baru PaymentUsecase.
func NewPaymentUsecase(
	invoiceRepo domain.InvoiceRepository,
	itemRepo domain.InvoiceItemRepository,
	paymentRepo domain.InvoicePaymentRepository,
	auditRepo domain.InvoiceAuditLogRepository,
	receiptSeqRepo domain.ReceiptSequenceRepository,
	settingsRepo domain.BillingSettingsRepository,
	customerRepo domain.CustomerRepository,
	pool *pgxpool.Pool,
	queueClient *asynq.Client,
	logger zerolog.Logger,
) *PaymentUsecase {
	return &PaymentUsecase{
		invoiceRepo:    invoiceRepo,
		itemRepo:       itemRepo,
		paymentRepo:    paymentRepo,
		auditRepo:      auditRepo,
		receiptSeqRepo: receiptSeqRepo,
		settingsRepo:   settingsRepo,
		customerRepo:   customerRepo,
		pool:           pool,
		queueClient:    queueClient,
		logger:         logger,
	}
}

// List mengambil daftar pembayaran dengan filter dan paginasi.
// Default: page=1, page_size=25.
func (uc *PaymentUsecase) List(ctx context.Context, params domain.PaymentListParams) (*domain.PaymentListResult, error) {
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 25
	}
	return uc.paymentRepo.ListWithFilters(ctx, params)
}

// Summary mengambil ringkasan statistik pembayaran untuk dashboard.
func (uc *PaymentUsecase) Summary(ctx context.Context, tenantID string, periodMonth, periodYear *int) (*domain.PaymentSummary, error) {
	// Ambil timezone dari billing settings, default "Asia/Jakarta"
	timezone := "Asia/Jakarta"
	settings, _ := uc.settingsRepo.GetByTenantID(ctx, tenantID)
	if settings != nil && settings.Timezone != "" {
		timezone = settings.Timezone
	}
	return uc.paymentRepo.GetSummary(ctx, tenantID, timezone, periodMonth, periodYear)
}

// SearchCustomers mencari pelanggan untuk pembayaran cepat.
// Validasi: searchTerm minimal 2 karakter.
func (uc *PaymentUsecase) SearchCustomers(ctx context.Context, tenantID, searchTerm string) ([]*domain.Customer, error) {
	if len(searchTerm) < 2 {
		return nil, domain.ErrSearchTermTooShort
	}
	return uc.customerRepo.SearchForPayment(ctx, tenantID, searchTerm)
}

// GetOpenInvoices mengambil daftar invoice terbuka untuk customer.
// Menghitung remaining_amount per invoice dan total_arrears.
func (uc *PaymentUsecase) GetOpenInvoices(ctx context.Context, customerID string) (*domain.OpenInvoicesResponse, error) {
	invoices, err := uc.invoiceRepo.FindOpenByCustomer(ctx, customerID)
	if err != nil {
		return nil, err
	}

	var items []domain.OpenInvoiceItem
	var totalArrears int64

	for _, inv := range invoices {
		remaining := inv.TotalAmount - inv.PaidAmount
		totalArrears += remaining
		items = append(items, domain.OpenInvoiceItem{
			ID:              inv.ID,
			InvoiceNumber:   inv.InvoiceNumber,
			PeriodMonth:     inv.PeriodMonth,
			PeriodYear:      inv.PeriodYear,
			TotalAmount:     inv.TotalAmount,
			PaidAmount:      inv.PaidAmount,
			RemainingAmount: remaining,
			Status:          inv.Status,
			DueDate:         inv.DueDate,
		})
	}

	if items == nil {
		items = []domain.OpenInvoiceItem{}
	}

	return &domain.OpenInvoicesResponse{
		Invoices:     items,
		TotalArrears: totalArrears,
	}, nil
}

// writePaymentAuditLog menulis audit log entry untuk invoice.
// Tidak mengembalikan error agar operasi utama tidak gagal.
func (uc *PaymentUsecase) writePaymentAuditLog(ctx context.Context, tenantID, invoiceID, action string, actor domain.ActorInfo, metadata map[string]interface{}) {
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

// publishPaymentEvent mempublikasikan event ke Redis queue.
// Tidak mengembalikan error agar operasi utama tidak gagal.
func (uc *PaymentUsecase) publishPaymentEvent(tenantID, eventType string, payload interface{}) {
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

// publishSyncPaymentLinkEvent mempublikasikan event ke asynq queue untuk sinkronisasi payment link.
// Dipanggil setelah pembayaran manual dicatat atau di-void, agar payment link yang aktif
// di-expire dan di-regenerate dengan jumlah terbaru.
// Non-blocking: error hanya di-log, tidak menggagalkan operasi utama.
func (uc *PaymentUsecase) publishSyncPaymentLinkEvent(tenantID, invoiceID, customerID string) {
	if uc.queueClient == nil {
		return
	}
	payload, err := json.Marshal(struct {
		TenantID   string `json:"tenant_id"`
		InvoiceID  string `json:"invoice_id"`
		CustomerID string `json:"customer_id"`
	}{
		TenantID:   tenantID,
		InvoiceID:  invoiceID,
		CustomerID: customerID,
	})
	if err != nil {
		uc.logger.Error().Err(err).
			Str("invoice_id", invoiceID).
			Msg("gagal marshal payload sync payment link")
		return
	}
	task := asynq.NewTask("gateway.sync_payment_link_amount", payload)
	if _, err := uc.queueClient.Enqueue(task); err != nil {
		uc.logger.Error().Err(err).
			Str("invoice_id", invoiceID).
			Msg("gagal enqueue task sync payment link amount")
	}
}

// adjustPaymentCreditBalance mengubah credit_balance pelanggan secara atomik.
// delta positif = tambah kredit, delta negatif = kurangi kredit.
func (uc *PaymentUsecase) adjustPaymentCreditBalance(ctx context.Context, customerID string, delta int64) error {
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
