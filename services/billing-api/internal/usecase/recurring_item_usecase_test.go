// recurring_item_usecase_test.go berisi unit test untuk RecurringItemUsecase.
package usecase

import (
	"context"
	"io"
	"testing"

	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// =============================================================================
// =============================================================================

type recurringUsecaseSetup struct {
	uc            *RecurringItemUsecase
	recurringRepo *mockRecurringItemRepo
	customerRepo  *invMockCustomerRepo
}

func setupRecurringUsecase() *recurringUsecaseSetup {
	recurringRepo := newMockRecurringItemRepo()
	customerRepo := newInvMockCustomerRepo()
	logger := zerolog.New(io.Discard)

	uc := NewRecurringItemUsecase(recurringRepo, customerRepo, logger)

	return &recurringUsecaseSetup{
		uc:            uc,
		recurringRepo: recurringRepo,
		customerRepo:  customerRepo,
	}
}

// =============================================================================
// Unit Tests - RecurringItemUsecase
// =============================================================================

// TestRecurringItem_Create_Success menguji pembuatan item berulang berhasil.
func TestRecurringItem_Create_Success(t *testing.T) {
	s := setupRecurringUsecase()
	ctx := context.Background()

	s.customerRepo.customers["cust-1"] = &domain.Customer{
		ID: "cust-1", TenantID: "tenant-1", Status: domain.CustomerStatusAktif,
	}

	req := domain.CreateRecurringItemRequest{
		Description: "Sewa router WiFi",
		Amount:      25000,
		StartDate:   "2024-01-01",
	}

	item, err := s.uc.Create(ctx, "cust-1", req, domain.ActorInfo{})
	if err != nil {
		t.Fatalf("Create gagal: %v", err)
	}

	if item.Description != "Sewa router WiFi" {
		t.Fatalf("expected description 'Sewa router WiFi', got '%s'", item.Description)
	}
	if item.Amount != 25000 {
		t.Fatalf("expected amount 25000, got %d", item.Amount)
	}
	if !item.IsActive {
		t.Fatal("expected is_active true")
	}
}

func TestRecurringItem_Create_CustomerNotFound(t *testing.T) {
	s := setupRecurringUsecase()
	ctx := context.Background()

	req := domain.CreateRecurringItemRequest{
		Description: "Test",
		Amount:      10000,
		StartDate:   "2024-01-01",
	}

	_, err := s.uc.Create(ctx, "nonexistent", req, domain.ActorInfo{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRecurringItem_Create_InvalidStartDate(t *testing.T) {
	s := setupRecurringUsecase()
	ctx := context.Background()

	s.customerRepo.customers["cust-1"] = &domain.Customer{
		ID: "cust-1", TenantID: "tenant-1",
	}

	req := domain.CreateRecurringItemRequest{
		Description: "Test",
		Amount:      10000,
		StartDate:   "invalid-date",
	}

	_, err := s.uc.Create(ctx, "cust-1", req, domain.ActorInfo{})
	if err == nil {
		t.Fatal("expected error for invalid date, got nil")
	}
}

func TestRecurringItem_List_Success(t *testing.T) {
	s := setupRecurringUsecase()
	ctx := context.Background()

	s.recurringRepo.items["cust-1"] = []*domain.CustomerRecurringItem{
		{ID: "ri-1", CustomerID: "cust-1", Description: "Item 1", Amount: 10000, IsActive: true},
		{ID: "ri-2", CustomerID: "cust-1", Description: "Item 2", Amount: 20000, IsActive: true},
	}

	items, err := s.uc.List(ctx, "cust-1")
	if err != nil {
		t.Fatalf("List gagal: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
}

func TestRecurringItem_Update_Success(t *testing.T) {
	s := setupRecurringUsecase()
	ctx := context.Background()

	s.recurringRepo.items["cust-1"] = []*domain.CustomerRecurringItem{
		{ID: "ri-1", CustomerID: "cust-1", Description: "Item lama", Amount: 10000, IsActive: true},
	}

	newAmount := int64(15000)
	req := domain.UpdateRecurringItemRequest{
		Description: "Item baru",
		Amount:      &newAmount,
	}

	updated, err := s.uc.Update(ctx, "cust-1", "ri-1", req, domain.ActorInfo{})
	if err != nil {
		t.Fatalf("Update gagal: %v", err)
	}
	if updated.Description != "Item baru" {
		t.Fatalf("expected description 'Item baru', got '%s'", updated.Description)
	}
	if updated.Amount != 15000 {
		t.Fatalf("expected amount 15000, got %d", updated.Amount)
	}
}

func TestRecurringItem_Update_NotFound(t *testing.T) {
	s := setupRecurringUsecase()
	ctx := context.Background()

	req := domain.UpdateRecurringItemRequest{Description: "Test"}
	_, err := s.uc.Update(ctx, "cust-1", "nonexistent", req, domain.ActorInfo{})
	if err != domain.ErrRecurringItemNotFound {
		t.Fatalf("expected ErrRecurringItemNotFound, got %v", err)
	}
}

func TestRecurringItem_Update_WrongCustomer(t *testing.T) {
	s := setupRecurringUsecase()
	ctx := context.Background()

	s.recurringRepo.items["cust-2"] = []*domain.CustomerRecurringItem{
		{ID: "ri-1", CustomerID: "cust-2", Description: "Item", Amount: 10000},
	}

	req := domain.UpdateRecurringItemRequest{Description: "Test"}
	_, err := s.uc.Update(ctx, "cust-1", "ri-1", req, domain.ActorInfo{})
	if err != domain.ErrRecurringItemNotFound {
		t.Fatalf("expected ErrRecurringItemNotFound, got %v", err)
	}
}

func TestRecurringItem_Delete_Success(t *testing.T) {
	s := setupRecurringUsecase()
	ctx := context.Background()

	s.recurringRepo.items["cust-1"] = []*domain.CustomerRecurringItem{
		{ID: "ri-1", CustomerID: "cust-1", Description: "Item", Amount: 10000, IsActive: true},
	}

	err := s.uc.Delete(ctx, "cust-1", "ri-1", domain.ActorInfo{})
	if err != nil {
		t.Fatalf("Delete gagal: %v", err)
	}
}

func TestRecurringItem_Delete_NotFound(t *testing.T) {
	s := setupRecurringUsecase()
	ctx := context.Background()

	err := s.uc.Delete(ctx, "cust-1", "nonexistent", domain.ActorInfo{})
	if err != domain.ErrRecurringItemNotFound {
		t.Fatalf("expected ErrRecurringItemNotFound, got %v", err)
	}
}
