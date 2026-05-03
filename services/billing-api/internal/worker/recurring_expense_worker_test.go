// recurring_expense_worker_test.go berisi integration tests untuk RecurringExpenseWorker.
// Test: auto-create recurring expenses, skip non-matching days, error handling.
package worker

import (
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// --- Mock ExpenseRepository untuk recurring expense worker tests ---

// mockExpenseRepo mengimplementasikan domain.ExpenseRepository untuk testing.
type mockExpenseRepo struct {
	expenses map[string]*domain.Expense
	seqID    int
}

func newMockExpenseRepo() *mockExpenseRepo {
	return &mockExpenseRepo{
		expenses: make(map[string]*domain.Expense),
	}
}

func (m *mockExpenseRepo) Create(_ context.Context, expense *domain.Expense) (*domain.Expense, error) {
	if expense.ID == "" {
		m.seqID++
		expense.ID = fmt.Sprintf("exp-%d", m.seqID)
	}
	copy := *expense
	copy.CreatedAt = time.Now()
	copy.UpdatedAt = time.Now()
	m.expenses[copy.ID] = &copy
	return &copy, nil
}

func (m *mockExpenseRepo) GetByID(_ context.Context, id string) (*domain.Expense, error) {
	e, ok := m.expenses[id]
	if !ok {
		return nil, domain.ErrExpenseNotFound
	}
	copy := *e
	return &copy, nil
}

func (m *mockExpenseRepo) Update(_ context.Context, expense *domain.Expense) (*domain.Expense, error) {
	if _, ok := m.expenses[expense.ID]; !ok {
		return nil, domain.ErrExpenseNotFound
	}
	copy := *expense
	m.expenses[copy.ID] = &copy
	return &copy, nil
}

func (m *mockExpenseRepo) SoftDelete(_ context.Context, id string) error {
	e, ok := m.expenses[id]
	if !ok {
		return domain.ErrExpenseNotFound
	}
	now := time.Now()
	e.DeletedAt = &now
	return nil
}

func (m *mockExpenseRepo) List(_ context.Context, _ string, _, _ time.Time, _ string) ([]*domain.Expense, error) {
	var result []*domain.Expense
	for _, e := range m.expenses {
		if e.DeletedAt == nil {
			copy := *e
			result = append(result, &copy)
		}
	}
	return result, nil
}

func (m *mockExpenseRepo) ListRecurring(_ context.Context) ([]*domain.Expense, error) {
	var result []*domain.Expense
	for _, e := range m.expenses {
		if e.IsRecurring && e.DeletedAt == nil {
			copy := *e
			result = append(result, &copy)
		}
	}
	return result, nil
}

func (m *mockExpenseRepo) SumByCategory(_ context.Context, _ string, _, _ time.Time) ([]domain.ProfitLossLineItem, error) {
	return nil, nil
}

// --- Helper ---

// newWorkerTestLogger membuat zerolog.Logger yang discard output (untuk tests).
func newWorkerTestLogger() zerolog.Logger {
	return zerolog.New(io.Discard)
}

// --- Test: Auto-create recurring expenses ---

func TestRecurringExpenseWorker_HandleRecurringExpense_CreatesExpenses(t *testing.T) {
	expenseRepo := newMockExpenseRepo()
	logger := newWorkerTestLogger()

	today := time.Now()
	currentDay := today.Day()

	// Tambahkan expense berulang yang jatuh hari ini
	expenseRepo.expenses["recurring-1"] = &domain.Expense{
		ID:           "recurring-1",
		TenantID:     "tenant-1",
		CategoryID:   "cat-1",
		Amount:       1000000,
		Description:  "Biaya bandwidth bulanan",
		IsRecurring:  true,
		RecurringDay: &currentDay,
		CreatedByID:  "user-1",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	worker := NewRecurringExpenseWorker(expenseRepo, logger)

	task := asynq.NewTask(TaskRecurringExpense, nil)
	err := worker.handleRecurringExpense(context.Background(), task)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verifikasi bahwa expense baru dibuat (recurring-1 + 1 baru)
	totalExpenses := 0
	for _, e := range expenseRepo.expenses {
		if e.DeletedAt == nil {
			totalExpenses++
		}
	}
	if totalExpenses != 2 {
		t.Fatalf("expected 2 expenses (original + created), got %d", totalExpenses)
	}
}

func TestRecurringExpenseWorker_HandleRecurringExpense_SkipsNonMatchingDay(t *testing.T) {
	expenseRepo := newMockExpenseRepo()
	logger := newWorkerTestLogger()

	// Recurring day yang tidak sesuai hari ini
	differentDay := 28
	today := time.Now()
	if today.Day() == 28 {
		differentDay = 1
	}

	expenseRepo.expenses["recurring-1"] = &domain.Expense{
		ID:           "recurring-1",
		TenantID:     "tenant-1",
		CategoryID:   "cat-1",
		Amount:       500000,
		Description:  "Biaya listrik",
		IsRecurring:  true,
		RecurringDay: &differentDay,
		CreatedByID:  "user-1",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	worker := NewRecurringExpenseWorker(expenseRepo, logger)

	task := asynq.NewTask(TaskRecurringExpense, nil)
	err := worker.handleRecurringExpense(context.Background(), task)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Tidak ada expense baru yang dibuat
	if len(expenseRepo.expenses) != 1 {
		t.Fatalf("expected 1 expense (no new created), got %d", len(expenseRepo.expenses))
	}
}

func TestRecurringExpenseWorker_HandleRecurringExpense_NoRecurring(t *testing.T) {
	expenseRepo := newMockExpenseRepo()
	logger := newWorkerTestLogger()

	// Tidak ada expense berulang
	expenseRepo.expenses["normal-1"] = &domain.Expense{
		ID:          "normal-1",
		TenantID:    "tenant-1",
		Amount:      300000,
		IsRecurring: false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	worker := NewRecurringExpenseWorker(expenseRepo, logger)

	task := asynq.NewTask(TaskRecurringExpense, nil)
	err := worker.handleRecurringExpense(context.Background(), task)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Tidak ada expense baru
	if len(expenseRepo.expenses) != 1 {
		t.Fatalf("expected 1 expense (no new created), got %d", len(expenseRepo.expenses))
	}
}

func TestRecurringExpenseWorker_HandleRecurringExpense_NilRecurringDay(t *testing.T) {
	expenseRepo := newMockExpenseRepo()
	logger := newWorkerTestLogger()

	// Expense berulang tapi recurring_day nil
	expenseRepo.expenses["recurring-nil"] = &domain.Expense{
		ID:           "recurring-nil",
		TenantID:     "tenant-1",
		CategoryID:   "cat-1",
		Amount:       200000,
		IsRecurring:  true,
		RecurringDay: nil, // nil recurring_day
		CreatedByID:  "user-1",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	worker := NewRecurringExpenseWorker(expenseRepo, logger)

	task := asynq.NewTask(TaskRecurringExpense, nil)
	err := worker.handleRecurringExpense(context.Background(), task)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Tidak ada expense baru (nil recurring_day di-skip)
	if len(expenseRepo.expenses) != 1 {
		t.Fatalf("expected 1 expense (nil day skipped), got %d", len(expenseRepo.expenses))
	}
}

func TestRecurringExpenseWorker_HandleRecurringExpense_MultipleRecurring(t *testing.T) {
	expenseRepo := newMockExpenseRepo()
	logger := newWorkerTestLogger()

	today := time.Now()
	currentDay := today.Day()

	// Dua expense berulang yang jatuh hari ini
	expenseRepo.expenses["recurring-1"] = &domain.Expense{
		ID: "recurring-1", TenantID: "tenant-1", CategoryID: "cat-1",
		Amount: 1000000, Description: "Bandwidth", IsRecurring: true,
		RecurringDay: &currentDay, CreatedByID: "user-1",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	expenseRepo.expenses["recurring-2"] = &domain.Expense{
		ID: "recurring-2", TenantID: "tenant-1", CategoryID: "cat-2",
		Amount: 500000, Description: "Listrik", IsRecurring: true,
		RecurringDay: &currentDay, CreatedByID: "user-1",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}

	worker := NewRecurringExpenseWorker(expenseRepo, logger)

	task := asynq.NewTask(TaskRecurringExpense, nil)
	err := worker.handleRecurringExpense(context.Background(), task)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// 2 original + 2 baru = 4
	if len(expenseRepo.expenses) != 4 {
		t.Fatalf("expected 4 expenses (2 original + 2 created), got %d", len(expenseRepo.expenses))
	}
}

func TestRecurringExpenseWorker_CreatedExpenseIsNotRecurring(t *testing.T) {
	expenseRepo := newMockExpenseRepo()
	logger := newWorkerTestLogger()

	today := time.Now()
	currentDay := today.Day()

	expenseRepo.expenses["recurring-1"] = &domain.Expense{
		ID: "recurring-1", TenantID: "tenant-1", CategoryID: "cat-1",
		Amount: 1000000, IsRecurring: true, RecurringDay: &currentDay,
		CreatedByID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}

	worker := NewRecurringExpenseWorker(expenseRepo, logger)

	task := asynq.NewTask(TaskRecurringExpense, nil)
	_ = worker.handleRecurringExpense(context.Background(), task)

	// Verifikasi expense baru bukan recurring
	for id, e := range expenseRepo.expenses {
		if id != "recurring-1" && e.IsRecurring {
			t.Fatalf("created expense %s should not be recurring", id)
		}
	}
}

// --- Test: Task type constant ---

func TestRecurringExpenseWorker_TaskTypeConstant(t *testing.T) {
	if TaskRecurringExpense != "expense.recurring" {
		t.Fatalf("expected 'expense.recurring', got %s", TaskRecurringExpense)
	}
}
