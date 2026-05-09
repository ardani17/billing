// invoice_cron.go berisi business logic untuk job cron invoice.
// InvoiceCronUsecase menangani auto-buat invoice dan perbarui status terlambat.
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

// InvoiceCronUsecase mengimplementasikan business logic untuk job cron invoice.
type InvoiceCronUsecase struct {
	invoiceRepo       domain.InvoiceRepository
	itemRepo          domain.InvoiceItemRepository
	auditRepo         domain.InvoiceAuditLogRepository
	sequenceRepo      domain.InvoiceSequenceRepository
	settingsRepo      domain.BillingSettingsRepository
	customerRepo      domain.CustomerRepository
	packageRepo       domain.PackageRepository
	recurringItemRepo domain.CustomerRecurringItemRepository
	pool              *pgxpool.Pool
	queueClient       *asynq.Client
	logger            zerolog.Logger
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

// ProcessAutoGenerate memproses auto-buat invoice untuk semua tenant.
// Mengambil semua billing settings -> untuk setiap tenant: cari pelanggan eligible ->
// untuk setiap pelanggan: cek idempotency -> buat invoice.
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

// GenerateDueForTenant menjalankan auto-buat invoice untuk satu tenant secara on-demand.
func (uc *InvoiceCronUsecase) GenerateDueForTenant(ctx context.Context, tenantID string) error {
	settings, err := uc.settingsRepo.GetByTenantID(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("gagal mengambil billing settings tenant: %w", err)
	}
	return uc.processAutoGenerateForTenant(ctx, settings, time.Now())
}

// processAutoGenerateForTenant memproses auto-buat invoice untuk satu tenant.
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

// isEligibleForInvoice memeriksa apakah pelanggan eligible untuk auto-buat invoice.
// Pelanggan eligible sejak tanggal buat sampai tanggal jatuh tempo periode berjalan.
func (uc *InvoiceCronUsecase) isEligibleForInvoice(customer *domain.Customer, settings *domain.BillingSettings, now time.Time) bool {
	periodMonth, periodYear := uc.calculatePeriod(customer.DueDate, now)
	dueDate := time.Date(periodYear, time.Month(periodMonth), customer.DueDate, 0, 0, 0, 0, now.Location())
	generateDate := dueDate.AddDate(0, 0, -settings.GenerateDays)
	currentDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	if currentDate.Before(generateDate) {
		return false
	}

	return !currentDate.After(dueDate)
}
