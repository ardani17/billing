// recurring_item_usecase.go berisi business logic untuk manajemen recurring items pelanggan.
// RecurringItemUsecase menangani CRUD recurring items (Create, List, Update, Delete/Deactivate).
package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// RecurringItemUsecase mengimplementasikan business logic untuk recurring items.
type RecurringItemUsecase struct {
	recurringItemRepo domain.CustomerRecurringItemRepository
	customerRepo      domain.CustomerRepository
	logger            zerolog.Logger
}

// NewRecurringItemUsecase membuat instance baru RecurringItemUsecase.
func NewRecurringItemUsecase(
	recurringItemRepo domain.CustomerRecurringItemRepository,
	customerRepo domain.CustomerRepository,
	logger zerolog.Logger,
) *RecurringItemUsecase {
	return &RecurringItemUsecase{
		recurringItemRepo: recurringItemRepo,
		customerRepo:      customerRepo,
		logger:            logger,
	}
}

// Create membuat recurring item baru untuk pelanggan.
// Flow: validasi pelanggan ada → parse tanggal → buat item dengan is_active=true.
func (uc *RecurringItemUsecase) Create(
	ctx context.Context,
	customerID string,
	req domain.CreateRecurringItemRequest,
	actor domain.ActorInfo,
) (*domain.CustomerRecurringItem, error) {
	// Validasi pelanggan ada
	customer, err := uc.customerRepo.GetByID(ctx, customerID)
	if err != nil {
		return nil, domain.ErrCustomerNotFound
	}

	// Parse start_date
	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		return nil, fmt.Errorf("format start_date tidak valid: %w", err)
	}

	// Parse end_date (opsional)
	var endDate *time.Time
	if req.EndDate != "" {
		parsed, err := time.Parse("2006-01-02", req.EndDate)
		if err != nil {
			return nil, fmt.Errorf("format end_date tidak valid: %w", err)
		}
		endDate = &parsed
	}

	item := &domain.CustomerRecurringItem{
		TenantID:    customer.TenantID,
		CustomerID:  customerID,
		Description: req.Description,
		Amount:      req.Amount,
		IsActive:    true,
		StartDate:   startDate,
		EndDate:     endDate,
	}

	created, err := uc.recurringItemRepo.Create(ctx, item)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat recurring item: %w", err)
	}

	return created, nil
}

// List mengambil semua recurring items untuk pelanggan tertentu.
func (uc *RecurringItemUsecase) List(ctx context.Context, customerID string) ([]*domain.CustomerRecurringItem, error) {
	return uc.recurringItemRepo.ListByCustomer(ctx, customerID)
}

// Update memperbarui recurring item yang ada.
// Flow: ambil item → verifikasi milik pelanggan → update field yang diberikan.
func (uc *RecurringItemUsecase) Update(
	ctx context.Context,
	customerID, itemID string,
	req domain.UpdateRecurringItemRequest,
	actor domain.ActorInfo,
) (*domain.CustomerRecurringItem, error) {
	// Ambil item yang ada
	item, err := uc.recurringItemRepo.GetByID(ctx, itemID)
	if err != nil {
		return nil, domain.ErrRecurringItemNotFound
	}

	// Verifikasi item milik pelanggan yang dimaksud
	if item.CustomerID != customerID {
		return nil, domain.ErrRecurringItemNotFound
	}

	// Update field yang diberikan
	if req.Description != "" {
		item.Description = req.Description
	}
	if req.Amount != nil {
		item.Amount = *req.Amount
	}
	if req.EndDate != "" {
		parsed, err := time.Parse("2006-01-02", req.EndDate)
		if err != nil {
			return nil, fmt.Errorf("format end_date tidak valid: %w", err)
		}
		item.EndDate = &parsed
	}

	updated, err := uc.recurringItemRepo.Update(ctx, item)
	if err != nil {
		return nil, fmt.Errorf("gagal update recurring item: %w", err)
	}

	return updated, nil
}

// Delete menonaktifkan recurring item (soft delete via deactivate).
// Flow: ambil item → verifikasi milik pelanggan → set is_active=false.
func (uc *RecurringItemUsecase) Delete(
	ctx context.Context,
	customerID, itemID string,
	actor domain.ActorInfo,
) error {
	// Ambil item yang ada
	item, err := uc.recurringItemRepo.GetByID(ctx, itemID)
	if err != nil {
		return domain.ErrRecurringItemNotFound
	}

	// Verifikasi item milik pelanggan yang dimaksud
	if item.CustomerID != customerID {
		return domain.ErrRecurringItemNotFound
	}

	// Deactivate (soft delete)
	if err := uc.recurringItemRepo.Deactivate(ctx, itemID); err != nil {
		return fmt.Errorf("gagal menonaktifkan recurring item: %w", err)
	}

	return nil
}
