// isolir_usecase.go berisi business logic untuk auto-isolir dan suspend pelanggan.
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

// IsolirUsecase mengimplementasikan business logic untuk modul isolir.
type IsolirUsecase struct {
	customerRepo    domain.CustomerRepository
	invoiceRepo     domain.InvoiceRepository
	invoiceItemRepo domain.InvoiceItemRepository
	pendingSyncRepo domain.PendingSyncRepository
	settingsRepo    domain.BillingSettingsRepository
	moduleRepo      TenantModuleRepository
	auditRepo       domain.InvoiceAuditLogRepository
	pool            *pgxpool.Pool
	queueClient     *asynq.Client
	logger          zerolog.Logger
}

// SetTenantModuleRepository memasang repositori entitlement modul tenant.
func (uc *IsolirUsecase) SetTenantModuleRepository(moduleRepo TenantModuleRepository) {
	uc.moduleRepo = moduleRepo
}

// NewIsolirUsecase membuat instance baru IsolirUsecase.
func NewIsolirUsecase(custRepo domain.CustomerRepository, invRepo domain.InvoiceRepository,
	itemRepo domain.InvoiceItemRepository, syncRepo domain.PendingSyncRepository,
	settRepo domain.BillingSettingsRepository, auditRepo domain.InvoiceAuditLogRepository,
	pool *pgxpool.Pool, qClient *asynq.Client, logger zerolog.Logger) *IsolirUsecase {
	return &IsolirUsecase{customerRepo: custRepo, invoiceRepo: invRepo,
		invoiceItemRepo: itemRepo, pendingSyncRepo: syncRepo, settingsRepo: settRepo,
		auditRepo: auditRepo, pool: pool, queueClient: qClient, logger: logger}
}

// ProcessAutoIsolir memproses auto-isolir untuk semua tenant dengan auto_isolir aktif.
func (uc *IsolirUsecase) ProcessAutoIsolir(ctx context.Context) error {
	allSettings, err := uc.settingsRepo.ListAll(ctx)
	if err != nil {
		return fmt.Errorf("gagal mengambil billing settings: %w", err)
	}
	for _, s := range allSettings {
		if !s.AutoIsolir {
			continue
		}
		if err := uc.processIsolirTenant(ctx, s); err != nil {
			uc.logger.Error().Err(err).Str("tenant_id", s.TenantID).Msg("gagal auto-isolir tenant")
		}
	}
	return nil
}
func (uc *IsolirUsecase) processIsolirTenant(ctx context.Context, s *domain.BillingSettings) error {
	now := domain.CurrentDateInTimezone(s.Timezone)
	invoices, err := uc.invoiceRepo.FindOverdueForIsolir(ctx, s.TenantID, s.GracePeriodDays, now.Time)
	if err != nil {
		return fmt.Errorf("gagal mencari invoice overdue: %w", err)
	}
	seen := make(map[string]bool)
	for _, inv := range invoices {
		if seen[inv.CustomerID] {
			continue
		}
		seen[inv.CustomerID] = true
		if err := uc.isolirCustomer(ctx, inv, now); err != nil {
			uc.logger.Error().Err(err).Str("customer_id", inv.CustomerID).Msg("gagal isolir pelanggan")
		}
	}
	return nil
}
func (uc *IsolirUsecase) isolirCustomer(ctx context.Context, inv *domain.Invoice, now domain.LocalDate) error {
	cust, err := uc.customerRepo.GetByID(ctx, inv.CustomerID)
	if err != nil {
		return fmt.Errorf("gagal mengambil customer: %w", err)
	}
	if cust.Status != domain.CustomerStatusAktif {
		return nil // idempotency: skip jika bukan aktif
	}
	newStatus, err := domain.Transition(cust.Status, domain.CustomerStatusIsolir)
	if err != nil {
		return err
	}
	if _, err := uc.customerRepo.UpdateStatus(ctx, cust.ID, newStatus); err != nil {
		return fmt.Errorf("gagal update status ke isolir: %w", err)
	}
	overdue := domain.DaysOverdue(inv.DueDate, now.Time)
	p := domain.CustomerIsolirPayload{
		CustomerID: cust.ID, TenantID: cust.TenantID, CustomerName: cust.Name,
		RouterID: cust.RouterID, PPPoEUsername: cust.PPPoEUsername,
		ConnectionMethod: string(cust.ConnectionMethod),
		Reason:           "auto_isolir: invoice terlambat melewati grace period", OverdueDays: overdue,
	}
	if uc.mikrotikEnabled(ctx, cust.TenantID) {
		uc.createPendingSync(ctx, cust.TenantID, cust.ID, domain.SyncOpIsolir)
		uc.publishEvent(cust.TenantID, domain.TaskCustomerIsolir, p)
	}
	uc.publishEvent(cust.TenantID, domain.TaskNotifIsolir, p)
	uc.writeAuditLog(ctx, cust.TenantID, inv.ID, "customer.isolir",
		map[string]interface{}{"overdue_days": overdue, "customer_id": cust.ID})
	return nil
}

// ProcessSuspend memproses suspend untuk semua tenant (isolir -> suspend).
func (uc *IsolirUsecase) ProcessSuspend(ctx context.Context) error {
	allSettings, err := uc.settingsRepo.ListAll(ctx)
	if err != nil {
		return fmt.Errorf("gagal mengambil billing settings: %w", err)
	}
	for _, s := range allSettings {
		if err := uc.processSuspendTenant(ctx, s); err != nil {
			uc.logger.Error().Err(err).Str("tenant_id", s.TenantID).Msg("gagal suspend tenant")
		}
	}
	return nil
}

func (uc *IsolirUsecase) processSuspendTenant(ctx context.Context, s *domain.BillingSettings) error {
	now := domain.CurrentDateInTimezone(s.Timezone)
	invoices, err := uc.invoiceRepo.FindOverdueForSuspend(ctx, s.TenantID, s.SuspendDays, now.Time)
	if err != nil {
		return fmt.Errorf("gagal mencari invoice overdue untuk suspend: %w", err)
	}
	seen := make(map[string]bool)
	for _, inv := range invoices {
		if seen[inv.CustomerID] {
			continue
		}
		seen[inv.CustomerID] = true
		if err := uc.suspendCustomer(ctx, inv, now); err != nil {
			uc.logger.Error().Err(err).Str("customer_id", inv.CustomerID).Msg("gagal suspend pelanggan")
		}
	}
	return nil
}

func (uc *IsolirUsecase) suspendCustomer(ctx context.Context, inv *domain.Invoice, now domain.LocalDate) error {
	cust, err := uc.customerRepo.GetByID(ctx, inv.CustomerID)
	if err != nil {
		return fmt.Errorf("gagal mengambil customer: %w", err)
	}
	if cust.Status != domain.CustomerStatusIsolir {
		return nil // idempotency: skip jika bukan isolir
	}
	newStatus, err := domain.Transition(cust.Status, domain.CustomerStatusSuspend)
	if err != nil {
		return err
	}
	if _, err := uc.customerRepo.UpdateStatus(ctx, cust.ID, newStatus); err != nil {
		return fmt.Errorf("gagal update status ke suspend: %w", err)
	}
	overdue := domain.DaysOverdue(inv.DueDate, now.Time)
	p := domain.CustomerSuspendPayload{
		CustomerID: cust.ID, TenantID: cust.TenantID, CustomerName: cust.Name,
		RouterID: cust.RouterID, PPPoEUsername: cust.PPPoEUsername,
		ConnectionMethod: string(cust.ConnectionMethod), OverdueDays: overdue,
	}
	if uc.mikrotikEnabled(ctx, cust.TenantID) {
		uc.createPendingSync(ctx, cust.TenantID, cust.ID, domain.SyncOpSuspend)
		uc.publishEvent(cust.TenantID, domain.TaskCustomerSuspend, p)
	}
	uc.publishEvent(cust.TenantID, domain.TaskNotifSuspend, p)
	uc.writeAuditLog(ctx, cust.TenantID, inv.ID, "customer.suspend",
		map[string]interface{}{"overdue_days": overdue, "customer_id": cust.ID})
	return nil
}

// createPendingSync membuat record pending_sync baru.
func (uc *IsolirUsecase) createPendingSync(ctx context.Context, tenantID, customerID string, op domain.SyncOperationType) {
	if !uc.mikrotikEnabled(ctx, tenantID) {
		return
	}
	ps := &domain.PendingSync{TenantID: tenantID, CustomerID: customerID,
		OperationType: op, Status: domain.SyncStatusPending, MaxRetries: 5}
	if _, err := uc.pendingSyncRepo.Create(ctx, ps); err != nil {
		uc.logger.Error().Err(err).Str("customer_id", customerID).Msg("gagal membuat pending_sync")
	}
}

func (uc *IsolirUsecase) mikrotikEnabled(ctx context.Context, tenantID string) bool {
	if uc.moduleRepo == nil {
		return true
	}
	caps, err := uc.moduleRepo.Capabilities(ctx, tenantID)
	if err != nil {
		uc.logger.Warn().Err(err).Str("tenant_id", tenantID).Msg("gagal cek entitlement MikroTik")
		return false
	}
	return caps.MikroTik
}

// publishEvent mempublikasikan event ke Redis queue.
func (uc *IsolirUsecase) publishEvent(tenantID, eventType string, payload interface{}) {
	if uc.queueClient == nil {
		return
	}
	data, err := json.Marshal(payload)
	if err != nil {
		uc.logger.Error().Err(err).Str("event_type", eventType).Msg("gagal marshal payload")
		return
	}
	if err := queue.EnqueueTask(uc.queueClient, queue.TaskEnvelope{
		EventType: eventType, TenantID: tenantID, Payload: data,
	}); err != nil {
		uc.logger.Error().Err(err).Str("event_type", eventType).Msg("gagal publish event")
	}
}

// writeAuditLog menulis audit log ke invoice_audit_logs dengan actor System.
func (uc *IsolirUsecase) writeAuditLog(ctx context.Context, tenantID, invoiceID, action string, meta map[string]interface{}) {
	if err := uc.auditRepo.Create(ctx, &domain.InvoiceAuditLog{
		TenantID: tenantID, InvoiceID: invoiceID, Action: action,
		ActorID: "system", ActorName: "System", Metadata: meta,
	}); err != nil {
		uc.logger.Error().Err(err).Str("invoice_id", invoiceID).Msg("gagal menulis audit log")
	}
}
