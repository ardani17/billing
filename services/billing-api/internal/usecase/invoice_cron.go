// invoice_cron.go berisi business logic untuk cron job invoice.
// InvoiceCronUsecase menangani auto-generate invoice dan update status overdue.
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

// InvoiceCronUsecase mengimplementasikan business logic untuk cron job invoice.
type InvoiceCronUsecase struct {
	invoiceRepo      domain.InvoiceRepository
	itemRepo         domain.InvoiceItemRepository
	auditRepo        domain.InvoiceAuditLogRepository
	sequenceRepo     domain.InvoiceSequenceRepository
	settingsRepo     domain.BillingSettingsRepository
	customerRepo     domain.CustomerRepository
	packageRepo      domain.PackageRepository
	recurringItemRepo domain.CustomerRecurringItemRepository
	pool             *pgxpool.Pool
	queueClient      *asynq.Client
	logger           zerolog.Logger
}

// NewInvoiceCronUsecase membuat instance baru InvoiceCronUsecase.
func NewInvoiceCronUsecase(
	invoiceRepo domain.InvoiceRepository,
	itemRepo domain.InvoiceItemRepository,
	auditRepo domain.InvoiceAuditLogRepository,
	sequenceRepo domain.InvoiceSequenceRepository,
	settingsRepo domain.BillingSettingsRepository,
	customerRepo domain.CustomerRepository,
	packageRepo domain.PackageRepository,
	recurringItemRepo domain.CustomerRecurringItemRepository,
	pool *pgxpool.Pool,
	queueClient *asynq.Client,
	logger zerolog.Logger,
) *InvoiceCronUsecase {
	return &InvoiceCronUsecase{
		invoiceRepo:       invoiceRepo,
		itemRepo:          itemRepo,
		auditRepo:         auditRepo,
		sequenceRepo:      sequenceRepo,
		settingsRepo:      settingsRepo,
		customerRepo:      customerRepo,
		packageRepo:       packageRepo,
		recurringItemRepo: recurringItemRepo,
		pool:              pool,
		queueClient:       queueClient,
		logger:            logger,
	}
}

// ProcessAutoGenerate memproses auto-generate invoice untuk semua tenant.
// Mengambil semua billing settings → untuk setiap tenant: cari pelanggan eligible →
// untuk setiap pelanggan: cek idempotency → generate invoice.
// Kegagalan satu tenant/pelanggan tidak memblokir yang lain.
func (uc *InvoiceCronUsecase) ProcessAutoGenerate(ctx context.Context) error {
	// Ambil semua billing settings (satu per tenant)
	allSettings, err := uc.settingsRepo.ListAll(ctx)
	if err != nil {
		return fmt.Errorf("gagal mengambil billing settings: %w", err)
	}

	now := time.Now()

	for _, settings := range allSettings {
		if err := uc.processAutoGenerateForTenant(ctx, settings, now); err != nil {
			uc.logger.Error().Err(err).
				Str("tenant_id", settings.TenantID).
				Msg("gagal auto-generate invoice untuk tenant")
			// Lanjutkan ke tenant berikutnya
		}
	}

	return nil
}

// processAutoGenerateForTenant memproses auto-generate invoice untuk satu tenant.
func (uc *InvoiceCronUsecase) processAutoGenerateForTenant(ctx context.Context, settings *domain.BillingSettings, now time.Time) error {
	// Cari pelanggan aktif yang due_date-nya cocok dengan generate_days
	// Pelanggan eligible: status aktif, tanggal saat ini == due_date - generate_days
	customers, err := uc.customerRepo.List(ctx, domain.CustomerListParams{
		TenantID: settings.TenantID,
		Status:   string(domain.CustomerStatusAktif),
		PageSize: 50,
		Page:     1,
	})
	if err != nil {
		return fmt.Errorf("gagal mengambil daftar pelanggan: %w", err)
	}

	// Proses semua halaman pelanggan
	allCustomers := customers.Data
	totalPages := customers.Pagination.TotalPages
	for page := 2; page <= totalPages; page++ {
		next, err := uc.customerRepo.List(ctx, domain.CustomerListParams{
			TenantID: settings.TenantID,
			Status:   string(domain.CustomerStatusAktif),
			PageSize: 50,
			Page:     page,
		})
		if err != nil {
			uc.logger.Error().Err(err).
				Str("tenant_id", settings.TenantID).
				Int("page", page).
				Msg("gagal mengambil halaman pelanggan berikutnya")
			break
		}
		allCustomers = append(allCustomers, next.Data...)
	}

	// Filter pelanggan yang eligible berdasarkan due_date dan generate_days
	for _, customer := range allCustomers {
		if !uc.isEligibleForInvoice(customer, settings, now) {
			continue
		}
		if err := uc.generateInvoiceForCustomer(ctx, settings, customer, now); err != nil {
			uc.logger.Error().Err(err).
				Str("tenant_id", settings.TenantID).
				Str("customer_id", customer.ID).
				Msg("gagal generate invoice untuk pelanggan")
			// Lanjutkan ke pelanggan berikutnya
		}
	}

	return nil
}

// isEligibleForInvoice memeriksa apakah pelanggan eligible untuk auto-generate invoice.
// Pelanggan eligible jika hari ini == tanggal jatuh tempo - generate_days.
func (uc *InvoiceCronUsecase) isEligibleForInvoice(customer *domain.Customer, settings *domain.BillingSettings, now time.Time) bool {
	// Hitung tanggal generate: due_date - generate_days
	dueDay := customer.DueDate
	currentDay := now.Day()

	// Hitung hari target generate (due_date - generate_days)
	targetDay := dueDay - settings.GenerateDays
	if targetDay <= 0 {
		targetDay += 30 // wrap ke bulan sebelumnya (pendekatan sederhana)
	}

	return currentDay == targetDay
}
