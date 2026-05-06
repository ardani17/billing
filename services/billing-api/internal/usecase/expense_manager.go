// expense_manager.go berisi ExpenseManager yang mengimplementasikan
// domain.ExpenseUsecase untuk CRUD pengeluaran dan kategori pengeluaran.
package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// ExpenseManager mengimplementasikan business logic untuk pengeluaran.
type ExpenseManager struct {
	expenseRepo  domain.ExpenseRepository
	categoryRepo domain.ExpenseCategoryRepository
	auditRepo    domain.AuditLogRepository
	logger       zerolog.Logger
}

// NewExpenseManager membuat instance baru ExpenseManager.
func NewExpenseManager(
	expenseRepo domain.ExpenseRepository,
	categoryRepo domain.ExpenseCategoryRepository,
	auditRepo domain.AuditLogRepository,
	logger zerolog.Logger,
) *ExpenseManager {
	return &ExpenseManager{
		expenseRepo:  expenseRepo,
		categoryRepo: categoryRepo,
		auditRepo:    auditRepo,
		logger:       logger.With().Str("component", "expense_manager").Logger(),
	}
}

// Create membuat pengeluaran baru.
func (em *ExpenseManager) Create(ctx context.Context, tenantID string, req domain.CreateExpenseRequest, actor domain.ActorInfo) (*domain.Expense, error) {
	expenseDate, err := time.Parse("2006-01-02", req.ExpenseDate)
	if err != nil {
		return nil, err
	}

	expense := &domain.Expense{
		ID:            uuid.New().String(),
		TenantID:      tenantID,
		CategoryID:    req.CategoryID,
		Amount:        req.Amount,
		Description:   req.Description,
		ExpenseDate:   expenseDate,
		PaymentMethod: req.PaymentMethod,
		VendorName:    req.VendorName,
		ReferenceNo:   req.ReferenceNo,
		AttachmentURL: req.AttachmentURL,
		IsRecurring:   req.IsRecurring,
		RecurringDay:  req.RecurringDay,
		CreatedByID:   actor.ActorID,
	}

	created, err := em.expenseRepo.Create(ctx, expense)
	if err != nil {
		em.logger.Error().Err(err).Str("tenant_id", tenantID).Msg("gagal membuat pengeluaran")
		return nil, err
	}
	em.writeAuditLog(ctx, tenantID, created.ID, "expense.created", actor, map[string]interface{}{
		"amount":      created.Amount,
		"category_id": created.CategoryID,
	})
	return created, nil
}

// GetByID mengambil pengeluaran berdasarkan ID.
func (em *ExpenseManager) GetByID(ctx context.Context, id string) (*domain.Expense, error) {
	expense, err := em.expenseRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if expense == nil {
		return nil, domain.ErrExpenseNotFound
	}
	return expense, nil
}

// Update memperbarui data pengeluaran.
func (em *ExpenseManager) Update(ctx context.Context, id string, req domain.UpdateExpenseRequest, actor domain.ActorInfo) (*domain.Expense, error) {
	existing, err := em.expenseRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, domain.ErrExpenseNotFound
	}

	// Terapkan perubahan dari request
	if req.CategoryID != "" {
		existing.CategoryID = req.CategoryID
	}
	if req.Amount != nil {
		existing.Amount = *req.Amount
	}
	if req.Description != nil {
		existing.Description = *req.Description
	}
	if req.ExpenseDate != nil {
		parsed, err := time.Parse("2006-01-02", *req.ExpenseDate)
		if err != nil {
			return nil, err
		}
		existing.ExpenseDate = parsed
	}
	if req.PaymentMethod != nil {
		existing.PaymentMethod = *req.PaymentMethod
	}
	if req.VendorName != nil {
		existing.VendorName = *req.VendorName
	}
	if req.ReferenceNo != nil {
		existing.ReferenceNo = *req.ReferenceNo
	}
	if req.AttachmentURL != nil {
		existing.AttachmentURL = *req.AttachmentURL
	}
	if req.IsRecurring != nil {
		existing.IsRecurring = *req.IsRecurring
	}
	if req.RecurringDay != nil {
		existing.RecurringDay = req.RecurringDay
	}

	updated, err := em.expenseRepo.Update(ctx, existing)
	if err != nil {
		em.logger.Error().Err(err).Str("id", id).Msg("gagal memperbarui pengeluaran")
		return nil, err
	}
	em.writeAuditLog(ctx, updated.TenantID, updated.ID, "expense.updated", actor, map[string]interface{}{
		"amount":      updated.Amount,
		"category_id": updated.CategoryID,
	})
	return updated, nil
}

// Delete menghapus pengeluaran secara soft delete.
func (em *ExpenseManager) Delete(ctx context.Context, id string, actor domain.ActorInfo) error {
	existing, err := em.expenseRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if err := em.expenseRepo.SoftDelete(ctx, id); err != nil {
		return err
	}
	em.writeAuditLog(ctx, existing.TenantID, id, "expense.deleted", actor, map[string]interface{}{
		"amount":      existing.Amount,
		"category_id": existing.CategoryID,
	})
	return nil
}

// List mengambil daftar pengeluaran dengan filter periode dan kategori.
func (em *ExpenseManager) List(ctx context.Context, tenantID string, periodStart, periodEnd time.Time, categoryID string) ([]*domain.Expense, error) {
	return em.expenseRepo.List(ctx, tenantID, periodStart, periodEnd, categoryID)
}

// ListCategories mengambil semua kategori pengeluaran aktif untuk tenant.
func (em *ExpenseManager) ListCategories(ctx context.Context, tenantID string) ([]*domain.ExpenseCategory, error) {
	return em.categoryRepo.List(ctx, tenantID)
}

// CreateCategory membuat kategori pengeluaran baru.
// Mengembalikan error jika nama kategori sudah ada di tenant.
func (em *ExpenseManager) CreateCategory(ctx context.Context, tenantID, name string) (*domain.ExpenseCategory, error) {
	exists, err := em.categoryRepo.NameExists(ctx, tenantID, name, "")
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, domain.ErrCategoryNameDuplicate
	}

	category := &domain.ExpenseCategory{
		ID:       uuid.New().String(),
		TenantID: tenantID,
		Name:     name,
	}

	created, err := em.categoryRepo.Create(ctx, category)
	if err != nil {
		em.logger.Error().Err(err).Str("tenant_id", tenantID).Msg("gagal membuat kategori")
		return nil, err
	}
	return created, nil
}

// UpdateCategory memperbarui nama kategori pengeluaran.
func (em *ExpenseManager) UpdateCategory(ctx context.Context, id, name string) (*domain.ExpenseCategory, error) {
	existing, err := em.categoryRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, domain.ErrExpenseCategoryNotFound
	}

	// Cek duplikasi nama (exclude ID saat ini)
	exists, err := em.categoryRepo.NameExists(ctx, existing.TenantID, name, id)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, domain.ErrCategoryNameDuplicate
	}

	existing.Name = name
	updated, err := em.categoryRepo.Update(ctx, existing)
	if err != nil {
		em.logger.Error().Err(err).Str("id", id).Msg("gagal memperbarui kategori")
		return nil, err
	}
	return updated, nil
}

// DeleteCategory menghapus kategori pengeluaran.
// Ditolak jika masih ada pengeluaran terkait.
func (em *ExpenseManager) DeleteCategory(ctx context.Context, id string) error {
	count, err := em.categoryRepo.ExpenseCount(ctx, id)
	if err != nil {
		return err
	}
	if count > 0 {
		return domain.ErrCategoryHasExpenses
	}
	return em.categoryRepo.SoftDelete(ctx, id)
}

func (em *ExpenseManager) writeAuditLog(ctx context.Context, tenantID, entityID, action string, actor domain.ActorInfo, changes map[string]interface{}) {
	if em.auditRepo == nil {
		return
	}
	log := &domain.AuditLog{
		TenantID:   tenantID,
		EntityType: "expense",
		EntityID:   entityID,
		Action:     action,
		ActorID:    actor.ActorID,
		ActorName:  actor.ActorName,
		Changes:    changes,
	}
	if err := em.auditRepo.Create(ctx, log); err != nil {
		em.logger.Error().Err(err).Str("entity_id", entityID).Str("action", action).Msg("gagal menulis audit log expense")
	}
}
