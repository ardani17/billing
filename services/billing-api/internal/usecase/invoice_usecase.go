// invoice_usecase.go berisi business logic untuk manajemen invoice (CRUD).
// Mengimplementasikan Buat dan CreatePrepaid pada InvoiceUsecase.
package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// InvoiceUsecase mengimplementasikan business logic untuk manajemen invoice.
type InvoiceUsecase struct {
	invoiceRepo  domain.InvoiceRepository
	itemRepo     domain.InvoiceItemRepository
	paymentRepo  domain.InvoicePaymentRepository
	auditRepo    domain.InvoiceAuditLogRepository
	sequenceRepo domain.InvoiceSequenceRepository
	settingsRepo domain.BillingSettingsRepository
	customerRepo domain.CustomerRepository
	packageRepo  domain.PackageRepository
	pool         *pgxpool.Pool
	queueClient  *asynq.Client
	logger       zerolog.Logger
}

// NewInvoiceUsecase membuat instance baru InvoiceUsecase.
func NewInvoiceUsecase(
	invoiceRepo domain.InvoiceRepository,
	itemRepo domain.InvoiceItemRepository,
	paymentRepo domain.InvoicePaymentRepository,
	auditRepo domain.InvoiceAuditLogRepository,
	sequenceRepo domain.InvoiceSequenceRepository,
	settingsRepo domain.BillingSettingsRepository,
	customerRepo domain.CustomerRepository,
	packageRepo domain.PackageRepository,
	pool *pgxpool.Pool,
	queueClient *asynq.Client,
	logger zerolog.Logger,
) *InvoiceUsecase {
	return &InvoiceUsecase{
		invoiceRepo:  invoiceRepo,
		itemRepo:     itemRepo,
		paymentRepo:  paymentRepo,
		auditRepo:    auditRepo,
		sequenceRepo: sequenceRepo,
		settingsRepo: settingsRepo,
		customerRepo: customerRepo,
		packageRepo:  packageRepo,
		pool:         pool,
		queueClient:  queueClient,
		logger:       logger,
	}
}

// Buat membuat invoice manual dengan item-item yang ditentukan.
// Alur: validasi customer -> buat nomor invoice -> hitung subtotal ->
// hitung pajak (opsional) -> terapkan kredit (opsional) -> buat invoice ->
// buat item -> tulis audit log -> terbitkan event.
func (uc *InvoiceUsecase) Create(ctx context.Context, tenantID string, req domain.CreateInvoiceRequest, actor domain.ActorInfo) (*domain.Invoice, error) {
	// Validasi customer ada dan aktif
	customer, err := uc.customerRepo.GetByID(ctx, req.CustomerID)
	if err != nil {
		return nil, domain.ErrCustomerNotFound
	}
	if customer.Status != domain.CustomerStatusAktif {
		return nil, fmt.Errorf("pelanggan tidak aktif (status: %s)", customer.Status)
	}

	// Parsing due date
	dueDate, err := time.Parse("2006-01-02", req.DueDate)
	if err != nil {
		return nil, fmt.Errorf("format due_date tidak valid: %w", err)
	}

	// Buat nomor invoice via sequence atomik
	periodMonth := int(dueDate.Month())
	periodYear := dueDate.Year()

	settings, _ := uc.settingsRepo.GetByTenantID(ctx, tenantID)
	prefix := "INV"
	if settings != nil && settings.InvoicePrefix != "" {
		prefix = settings.InvoicePrefix
	}

	seq, err := uc.sequenceRepo.NextSequence(ctx, tenantID, periodYear, periodMonth)
	if err != nil {
		return nil, fmt.Errorf("gagal generate nomor invoice: %w", err)
	}
	invoiceNumber := domain.FormatInvoiceNumber(prefix, periodYear, periodMonth, seq)

	// Hitung subtotal dari items
	var subtotal int64
	items := make([]*domain.InvoiceItem, 0, len(req.Items))
	for i, item := range req.Items {
		amount := int64(item.Quantity) * item.UnitPrice
		subtotal += amount
		items = append(items, &domain.InvoiceItem{
			TenantID:    tenantID,
			ItemType:    domain.ItemTypeCustom,
			Description: item.Description,
			Quantity:    item.Quantity,
			UnitPrice:   item.UnitPrice,
			Amount:      amount,
			SortOrder:   i + 1,
		})
	}

	// Hitung pajak jika apply_tax aktif
	var taxAmount int64
	applyTax := settings != nil && settings.TaxEnabled
	if req.ApplyTax != nil {
		applyTax = *req.ApplyTax
	}
	if applyTax && settings != nil && settings.TaxEnabled {
		taxAmount = subtotal * int64(settings.TaxRate) / 100
		items = append(items, &domain.InvoiceItem{
			TenantID:    tenantID,
			ItemType:    domain.ItemTypeTax,
			Description: fmt.Sprintf("PPN %v%%", settings.TaxRate),
			Quantity:    1,
			UnitPrice:   taxAmount,
			Amount:      taxAmount,
			SortOrder:   len(items) + 1,
		})
	}

	totalAmount := subtotal + taxAmount

	// Terapkan kredit jika apply_credit aktif dan customer punya saldo
	var creditApplied int64
	applyCredit := true
	if req.ApplyCredit != nil {
		applyCredit = *req.ApplyCredit
	}
	if applyCredit && customer.CreditBalance > 0 && totalAmount > 0 {
		creditApplied = customer.CreditBalance
		if creditApplied > totalAmount {
			creditApplied = totalAmount
		}
		// Kurangi credit_balance customer secara atomik
		if err := uc.adjustCreditBalance(ctx, req.CustomerID, -creditApplied); err != nil {
			return nil, fmt.Errorf("gagal update credit balance: %w", err)
		}
		items = append(items, &domain.InvoiceItem{
			TenantID:    tenantID,
			ItemType:    domain.ItemTypeCreditApplied,
			Description: "Kredit diterapkan",
			Quantity:    1,
			UnitPrice:   creditApplied,
			Amount:      creditApplied,
			SortOrder:   len(items) + 1,
		})
		totalAmount -= creditApplied
	}

	// Buat invoice
	invoice := &domain.Invoice{
		TenantID:      tenantID,
		CustomerID:    req.CustomerID,
		InvoiceNumber: invoiceNumber,
		PeriodMonth:   periodMonth,
		PeriodYear:    periodYear,
		DueDate:       dueDate,
		Subtotal:      subtotal,
		TaxAmount:     taxAmount,
		CreditApplied: creditApplied,
		TotalAmount:   totalAmount,
		Status:        domain.InvoiceStatusBelumBayar,
		Notes:         req.Notes,
		Version:       1,
	}

	created, err := uc.invoiceRepo.Create(ctx, invoice)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat invoice: %w", err)
	}

	// Set invoice_id pada semua items dan bulk buat
	for _, item := range items {
		item.InvoiceID = created.ID
	}
	if _, err := uc.itemRepo.BulkCreate(ctx, items); err != nil {
		return nil, fmt.Errorf("gagal membuat invoice items: %w", err)
	}

	// Tulis audit log
	uc.writeInvoiceAuditLog(ctx, tenantID, created.ID, "invoice.created_manual", actor, nil)

	// Terbitkan event invoice.created
	uc.publishEvent(tenantID, "invoice.created", domain.InvoiceCreatedPayload{
		InvoiceID:     created.ID,
		TenantID:      tenantID,
		CustomerID:    req.CustomerID,
		InvoiceNumber: invoiceNumber,
		TotalAmount:   totalAmount,
		DueDate:       req.DueDate,
	})

	return created, nil
}
