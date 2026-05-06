package usecase

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

type mockExpenseRepo struct {
	expenses map[string]*domain.Expense
}

func newMockExpenseRepo() *mockExpenseRepo {
	return &mockExpenseRepo{expenses: make(map[string]*domain.Expense)}
}

func (m *mockExpenseRepo) Create(_ context.Context, expense *domain.Expense) (*domain.Expense, error) {
	if expense.ID == "" {
		expense.ID = fmt.Sprintf("expense-%d", len(m.expenses)+1)
	}
	copy := *expense
	m.expenses[copy.ID] = &copy
	return &copy, nil
}

func (m *mockExpenseRepo) GetByID(_ context.Context, id string) (*domain.Expense, error) {
	expense, ok := m.expenses[id]
	if !ok || expense.DeletedAt != nil {
		return nil, domain.ErrExpenseNotFound
	}
	copy := *expense
	return &copy, nil
}

func (m *mockExpenseRepo) Update(_ context.Context, expense *domain.Expense) (*domain.Expense, error) {
	if _, ok := m.expenses[expense.ID]; !ok {
		return nil, domain.ErrExpenseNotFound
	}
	copy := *expense
	copy.UpdatedAt = time.Now()
	m.expenses[copy.ID] = &copy
	return &copy, nil
}

func (m *mockExpenseRepo) SoftDelete(_ context.Context, id string) error {
	expense, ok := m.expenses[id]
	if !ok {
		return domain.ErrExpenseNotFound
	}
	now := time.Now()
	expense.DeletedAt = &now
	return nil
}

func (m *mockExpenseRepo) List(_ context.Context, tenantID string, periodStart, periodEnd time.Time, categoryID string) ([]*domain.Expense, error) {
	var result []*domain.Expense
	for _, expense := range m.expenses {
		if expense.TenantID != tenantID || expense.DeletedAt != nil {
			continue
		}
		if expense.ExpenseDate.Before(periodStart) || expense.ExpenseDate.After(periodEnd) {
			continue
		}
		if categoryID != "" && expense.CategoryID != categoryID {
			continue
		}
		copy := *expense
		result = append(result, &copy)
	}
	return result, nil
}

func (m *mockExpenseRepo) ListRecurring(_ context.Context) ([]*domain.Expense, error) {
	return nil, nil
}

func (m *mockExpenseRepo) SumByCategory(_ context.Context, _ string, _, _ time.Time) ([]domain.ProfitLossLineItem, error) {
	return nil, nil
}

type mockExpenseCategoryRepo struct {
	categories map[string]*domain.ExpenseCategory
}

func newMockExpenseCategoryRepo() *mockExpenseCategoryRepo {
	return &mockExpenseCategoryRepo{categories: make(map[string]*domain.ExpenseCategory)}
}

func (m *mockExpenseCategoryRepo) Create(_ context.Context, category *domain.ExpenseCategory) (*domain.ExpenseCategory, error) {
	if category.ID == "" {
		category.ID = fmt.Sprintf("category-%d", len(m.categories)+1)
	}
	copy := *category
	m.categories[copy.ID] = &copy
	return &copy, nil
}

func (m *mockExpenseCategoryRepo) GetByID(_ context.Context, id string) (*domain.ExpenseCategory, error) {
	category, ok := m.categories[id]
	if !ok {
		return nil, domain.ErrExpenseCategoryNotFound
	}
	copy := *category
	return &copy, nil
}

func (m *mockExpenseCategoryRepo) Update(_ context.Context, category *domain.ExpenseCategory) (*domain.ExpenseCategory, error) {
	if _, ok := m.categories[category.ID]; !ok {
		return nil, domain.ErrExpenseCategoryNotFound
	}
	copy := *category
	m.categories[copy.ID] = &copy
	return &copy, nil
}

func (m *mockExpenseCategoryRepo) SoftDelete(_ context.Context, id string) error {
	delete(m.categories, id)
	return nil
}

func (m *mockExpenseCategoryRepo) List(_ context.Context, tenantID string) ([]*domain.ExpenseCategory, error) {
	var result []*domain.ExpenseCategory
	for _, category := range m.categories {
		if category.TenantID == tenantID {
			copy := *category
			result = append(result, &copy)
		}
	}
	return result, nil
}

func (m *mockExpenseCategoryRepo) NameExists(_ context.Context, tenantID, name, excludeID string) (bool, error) {
	for _, category := range m.categories {
		if category.TenantID == tenantID && category.Name == name && category.ID != excludeID {
			return true, nil
		}
	}
	return false, nil
}

func (m *mockExpenseCategoryRepo) ExpenseCount(_ context.Context, _ string) (int, error) {
	return 0, nil
}

func (m *mockExpenseCategoryRepo) CreateDefaults(_ context.Context, _ string) error {
	return nil
}

func TestExpenseManager_CreateWritesAuditLog(t *testing.T) {
	expenseRepo := newMockExpenseRepo()
	auditLogRepo := newMockAuditLogRepo()
	uc := NewExpenseManager(expenseRepo, newMockExpenseCategoryRepo(), auditLogRepo, newTestLogger())

	created, err := uc.Create(context.Background(), "tenant-1", domain.CreateExpenseRequest{
		CategoryID:  "category-1",
		Amount:      125000,
		Description: "Upstream",
		ExpenseDate: "2026-05-06",
	}, domain.ActorInfo{ActorID: "user-1", ActorName: "Admin"})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	logs := auditLogRepo.logsForEntity("expense", created.ID)
	if len(logs) != 1 || logs[0].Action != "expense.created" {
		t.Fatalf("expected expense.created audit log, got %#v", logs)
	}
}

func TestExpenseManager_UpdateWritesAuditLogChanges(t *testing.T) {
	expenseRepo := newMockExpenseRepo()
	auditLogRepo := newMockAuditLogRepo()
	uc := NewExpenseManager(expenseRepo, newMockExpenseCategoryRepo(), auditLogRepo, newTestLogger())

	created, err := uc.Create(context.Background(), "tenant-1", domain.CreateExpenseRequest{
		CategoryID:  "category-1",
		Amount:      125000,
		Description: "Upstream",
		ExpenseDate: "2026-05-06",
	}, domain.ActorInfo{ActorID: "user-1", ActorName: "Admin"})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	auditLogRepo.reset()

	nextAmount := int64(175000)
	_, err = uc.Update(context.Background(), created.ID, domain.UpdateExpenseRequest{Amount: &nextAmount}, domain.ActorInfo{ActorID: "user-1", ActorName: "Admin"})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}

	logs := auditLogRepo.logsForEntity("expense", created.ID)
	if len(logs) != 1 || logs[0].Action != "expense.updated" {
		t.Fatalf("expected expense.updated audit log, got %#v", logs)
	}
	if _, ok := logs[0].Changes["amount"]; !ok {
		t.Fatalf("expected amount change, got %#v", logs[0].Changes)
	}
}

func TestExpenseManager_DeleteWritesAuditLog(t *testing.T) {
	expenseRepo := newMockExpenseRepo()
	auditLogRepo := newMockAuditLogRepo()
	uc := NewExpenseManager(expenseRepo, newMockExpenseCategoryRepo(), auditLogRepo, newTestLogger())

	created, err := uc.Create(context.Background(), "tenant-1", domain.CreateExpenseRequest{
		CategoryID:  "category-1",
		Amount:      125000,
		Description: "Upstream",
		ExpenseDate: "2026-05-06",
	}, domain.ActorInfo{ActorID: "user-1", ActorName: "Admin"})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	auditLogRepo.reset()

	if err := uc.Delete(context.Background(), created.ID, domain.ActorInfo{ActorID: "user-1", ActorName: "Admin"}); err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	logs := auditLogRepo.logsForEntity("expense", created.ID)
	if len(logs) != 1 || logs[0].Action != "expense.deleted" {
		t.Fatalf("expected expense.deleted audit log, got %#v", logs)
	}
}
