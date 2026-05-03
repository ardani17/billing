package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ExpenseRepo mengimplementasikan domain.ExpenseRepository dengan membungkus
// sqlc-generated Queries dan pgxpool.Pool.
type ExpenseRepo struct {
	// queries adalah sqlc-generated Queries untuk operasi expense.
	queries *Queries

	// pool digunakan untuk koneksi database langsung jika diperlukan.
	pool *pgxpool.Pool
}

// NewExpenseRepo membuat instance baru ExpenseRepo.
func NewExpenseRepo(queries *Queries, pool *pgxpool.Pool) *ExpenseRepo {
	return &ExpenseRepo{
		queries: queries,
		pool:    pool,
	}
}

// --- Helper functions untuk mapping sqlc ↔ domain ---

// mapExpenseRow memetakan sqlc Expense model ke domain.Expense.
func mapExpenseRow(row Expense) *domain.Expense {
	return &domain.Expense{
		ID:           uuidToString(row.ID),
		TenantID:     uuidToString(row.TenantID),
		CategoryID:   uuidToString(row.CategoryID),
		Amount:       row.Amount,
		Description:  row.Description,
		ExpenseDate:  dateToTime(row.ExpenseDate),
		IsRecurring:  row.IsRecurring,
		RecurringDay: int4ToIntPtr(row.RecurringDay),
		CreatedByID:  uuidToString(row.CreatedByID),
		DeletedAt:    timestamptzToTimePtr(row.DeletedAt),
		CreatedAt:    timestamptzToTime(row.CreatedAt),
		UpdatedAt:    timestamptzToTime(row.UpdatedAt),
	}
}

// mapGetExpenseByIDRow memetakan GetExpenseByIDRow (JOIN dengan category) ke domain.Expense.
func mapGetExpenseByIDRow(row GetExpenseByIDRow) *domain.Expense {
	return &domain.Expense{
		ID:           uuidToString(row.ID),
		TenantID:     uuidToString(row.TenantID),
		CategoryID:   uuidToString(row.CategoryID),
		CategoryName: row.CategoryName,
		Amount:       row.Amount,
		Description:  row.Description,
		ExpenseDate:  dateToTime(row.ExpenseDate),
		IsRecurring:  row.IsRecurring,
		RecurringDay: int4ToIntPtr(row.RecurringDay),
		CreatedByID:  uuidToString(row.CreatedByID),
		DeletedAt:    timestamptzToTimePtr(row.DeletedAt),
		CreatedAt:    timestamptzToTime(row.CreatedAt),
		UpdatedAt:    timestamptzToTime(row.UpdatedAt),
	}
}

// mapListExpensesRow memetakan ListExpensesRow (JOIN dengan category) ke domain.Expense.
func mapListExpensesRow(row ListExpensesRow) *domain.Expense {
	return &domain.Expense{
		ID:           uuidToString(row.ID),
		TenantID:     uuidToString(row.TenantID),
		CategoryID:   uuidToString(row.CategoryID),
		CategoryName: row.CategoryName,
		Amount:       row.Amount,
		Description:  row.Description,
		ExpenseDate:  dateToTime(row.ExpenseDate),
		IsRecurring:  row.IsRecurring,
		RecurringDay: int4ToIntPtr(row.RecurringDay),
		CreatedByID:  uuidToString(row.CreatedByID),
		DeletedAt:    timestamptzToTimePtr(row.DeletedAt),
		CreatedAt:    timestamptzToTime(row.CreatedAt),
		UpdatedAt:    timestamptzToTime(row.UpdatedAt),
	}
}

// --- Implementasi domain.ExpenseRepository ---

// Create membuat pengeluaran baru dan mengembalikan pengeluaran yang dibuat.
func (r *ExpenseRepo) Create(ctx context.Context, expense *domain.Expense) (*domain.Expense, error) {
	row, err := r.queries.CreateExpense(ctx, CreateExpenseParams{
		TenantID:     stringToUUID(expense.TenantID),
		CategoryID:   stringToUUID(expense.CategoryID),
		Amount:       expense.Amount,
		Description:  expense.Description,
		ExpenseDate:  timeToDate(expense.ExpenseDate),
		IsRecurring:  expense.IsRecurring,
		RecurringDay: intPtrToInt4(expense.RecurringDay),
		CreatedByID:  stringToUUID(expense.CreatedByID),
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal membuat expense: %w", err)
	}
	return mapExpenseRow(row), nil
}

// GetByID mengambil pengeluaran berdasarkan ID (tenant-scoped via RLS).
func (r *ExpenseRepo) GetByID(ctx context.Context, id string) (*domain.Expense, error) {
	row, err := r.queries.GetExpenseByID(ctx, stringToUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrExpenseNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil expense by ID: %w", err)
	}
	return mapGetExpenseByIDRow(row), nil
}

// Update memperbarui data pengeluaran dan mengembalikan pengeluaran yang diperbarui.
func (r *ExpenseRepo) Update(ctx context.Context, expense *domain.Expense) (*domain.Expense, error) {
	row, err := r.queries.UpdateExpense(ctx, UpdateExpenseParams{
		ID:           stringToUUID(expense.ID),
		CategoryID:   stringToUUID(expense.CategoryID),
		Amount:       expense.Amount,
		Description:  expense.Description,
		ExpenseDate:  timeToDate(expense.ExpenseDate),
		IsRecurring:  expense.IsRecurring,
		RecurringDay: intPtrToInt4(expense.RecurringDay),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrExpenseNotFound
		}
		return nil, fmt.Errorf("repository: gagal memperbarui expense: %w", err)
	}
	return mapExpenseRow(row), nil
}

// SoftDelete menghapus pengeluaran secara soft delete (set deleted_at).
func (r *ExpenseRepo) SoftDelete(ctx context.Context, id string) error {
	err := r.queries.SoftDeleteExpense(ctx, stringToUUID(id))
	if err != nil {
		return fmt.Errorf("repository: gagal soft-delete expense: %w", err)
	}
	return nil
}

// List mengambil daftar pengeluaran dengan filter periode dan kategori.
func (r *ExpenseRepo) List(ctx context.Context, tenantID string, periodStart, periodEnd time.Time, categoryID string) ([]*domain.Expense, error) {
	rows, err := r.queries.ListExpenses(ctx, ListExpensesParams{
		TenantID:      stringToUUID(tenantID),
		ExpenseDate:   timeToDate(periodStart),
		ExpenseDate_2: timeToDate(periodEnd),
		CategoryID:    stringToUUID(categoryID),
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil daftar expense: %w", err)
	}
	expenses := make([]*domain.Expense, 0, len(rows))
	for _, row := range rows {
		expenses = append(expenses, mapListExpensesRow(row))
	}
	return expenses, nil
}

// ListRecurring mengambil semua pengeluaran berulang yang aktif (untuk auto-create bulanan).
func (r *ExpenseRepo) ListRecurring(ctx context.Context) ([]*domain.Expense, error) {
	rows, err := r.queries.ListRecurringExpenses(ctx)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil recurring expenses: %w", err)
	}
	expenses := make([]*domain.Expense, 0, len(rows))
	for _, row := range rows {
		expenses = append(expenses, mapExpenseRow(row))
	}
	return expenses, nil
}

// SumByCategory menghitung total pengeluaran per kategori untuk laporan laba rugi.
func (r *ExpenseRepo) SumByCategory(ctx context.Context, tenantID string, periodStart, periodEnd time.Time) ([]domain.ProfitLossLineItem, error) {
	rows, err := r.queries.SumExpensesByCategory(ctx, SumExpensesByCategoryParams{
		TenantID:      stringToUUID(tenantID),
		ExpenseDate:   timeToDate(periodStart),
		ExpenseDate_2: timeToDate(periodEnd),
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal menghitung sum expenses by category: %w", err)
	}
	items := make([]domain.ProfitLossLineItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, domain.ProfitLossLineItem{
			Label:  row.Label,
			Amount: row.Amount,
		})
	}
	return items, nil
}
