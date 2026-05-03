package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ExpenseCategoryRepo mengimplementasikan domain.ExpenseCategoryRepository
// dengan membungkus sqlc-generated Queries dan pgxpool.Pool.
type ExpenseCategoryRepo struct {
	// queries adalah sqlc-generated Queries untuk operasi expense_categories.
	queries *Queries

	// pool digunakan untuk koneksi database langsung jika diperlukan.
	pool *pgxpool.Pool
}

// NewExpenseCategoryRepo membuat instance baru ExpenseCategoryRepo.
func NewExpenseCategoryRepo(queries *Queries, pool *pgxpool.Pool) *ExpenseCategoryRepo {
	return &ExpenseCategoryRepo{
		queries: queries,
		pool:    pool,
	}
}

// --- Helper mapping sqlc ↔ domain ---

// mapExpenseCategoryRow memetakan sqlc ExpenseCategory ke domain.ExpenseCategory.
func mapExpenseCategoryRow(row ExpenseCategory) *domain.ExpenseCategory {
	return &domain.ExpenseCategory{
		ID:        uuidToString(row.ID),
		TenantID:  uuidToString(row.TenantID),
		Name:      row.Name,
		IsDefault: row.IsDefault,
		DeletedAt: timestamptzToTimePtr(row.DeletedAt),
		CreatedAt: timestamptzToTime(row.CreatedAt),
		UpdatedAt: timestamptzToTime(row.UpdatedAt),
	}
}

// --- Implementasi domain.ExpenseCategoryRepository ---

// Create membuat kategori pengeluaran baru dan mengembalikan kategori yang dibuat.
func (r *ExpenseCategoryRepo) Create(ctx context.Context, category *domain.ExpenseCategory) (*domain.ExpenseCategory, error) {
	row, err := r.queries.CreateExpenseCategory(ctx, CreateExpenseCategoryParams{
		TenantID:  stringToUUID(category.TenantID),
		Name:      category.Name,
		IsDefault: category.IsDefault,
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal membuat expense category: %w", err)
	}
	return mapExpenseCategoryRow(row), nil
}

// GetByID mengambil kategori pengeluaran berdasarkan ID.
func (r *ExpenseCategoryRepo) GetByID(ctx context.Context, id string) (*domain.ExpenseCategory, error) {
	row, err := r.queries.GetExpenseCategoryByID(ctx, stringToUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrExpenseCategoryNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil expense category by ID: %w", err)
	}
	return mapExpenseCategoryRow(row), nil
}

// Update memperbarui nama kategori dan mengembalikan kategori yang diperbarui.
func (r *ExpenseCategoryRepo) Update(ctx context.Context, category *domain.ExpenseCategory) (*domain.ExpenseCategory, error) {
	row, err := r.queries.UpdateExpenseCategory(ctx, UpdateExpenseCategoryParams{
		ID:   stringToUUID(category.ID),
		Name: category.Name,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrExpenseCategoryNotFound
		}
		return nil, fmt.Errorf("repository: gagal memperbarui expense category: %w", err)
	}
	return mapExpenseCategoryRow(row), nil
}

// SoftDelete menghapus kategori secara soft delete (set deleted_at).
func (r *ExpenseCategoryRepo) SoftDelete(ctx context.Context, id string) error {
	err := r.queries.SoftDeleteExpenseCategory(ctx, stringToUUID(id))
	if err != nil {
		return fmt.Errorf("repository: gagal soft-delete expense category: %w", err)
	}
	return nil
}

// List mengambil semua kategori pengeluaran aktif untuk tenant.
func (r *ExpenseCategoryRepo) List(ctx context.Context, tenantID string) ([]*domain.ExpenseCategory, error) {
	rows, err := r.queries.ListExpenseCategories(ctx, stringToUUID(tenantID))
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil daftar expense categories: %w", err)
	}
	categories := make([]*domain.ExpenseCategory, 0, len(rows))
	for _, row := range rows {
		categories = append(categories, mapExpenseCategoryRow(row))
	}
	return categories, nil
}

// NameExists mengecek apakah nama kategori sudah ada di tenant (exclude ID tertentu).
func (r *ExpenseCategoryRepo) NameExists(ctx context.Context, tenantID, name, excludeID string) (bool, error) {
	exists, err := r.queries.ExpenseCategoryNameExists(ctx, ExpenseCategoryNameExistsParams{
		TenantID: stringToUUID(tenantID),
		Name:     name,
		ID:       stringToUUID(excludeID),
	})
	if err != nil {
		return false, fmt.Errorf("repository: gagal mengecek nama kategori: %w", err)
	}
	return exists, nil
}

// ExpenseCount menghitung jumlah pengeluaran aktif dalam kategori.
func (r *ExpenseCategoryRepo) ExpenseCount(ctx context.Context, id string) (int, error) {
	count, err := r.queries.ExpenseCategoryExpenseCount(ctx, stringToUUID(id))
	if err != nil {
		return 0, fmt.Errorf("repository: gagal menghitung expense count: %w", err)
	}
	return int(count), nil
}

// CreateDefaults membuat 7 kategori default untuk tenant baru.
func (r *ExpenseCategoryRepo) CreateDefaults(ctx context.Context, tenantID string) error {
	err := r.queries.CreateDefaultExpenseCategories(ctx, stringToUUID(tenantID))
	if err != nil {
		return fmt.Errorf("repository: gagal membuat default expense categories: %w", err)
	}
	return nil
}
