package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
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
		ID:            uuidToString(row.ID),
		TenantID:      uuidToString(row.TenantID),
		CategoryID:    uuidToString(row.CategoryID),
		Amount:        row.Amount,
		Description:   row.Description,
		ExpenseDate:   dateToTime(row.ExpenseDate),
		PaymentMethod: "",
		VendorName:    "",
		ReferenceNo:   "",
		AttachmentURL: "",
		IsRecurring:   row.IsRecurring,
		RecurringDay:  int4ToIntPtr(row.RecurringDay),
		CreatedByID:   uuidToString(row.CreatedByID),
		DeletedAt:     timestamptzToTimePtr(row.DeletedAt),
		CreatedAt:     timestamptzToTime(row.CreatedAt),
		UpdatedAt:     timestamptzToTime(row.UpdatedAt),
	}
}

// mapGetExpenseByIDRow memetakan GetExpenseByIDRow (JOIN dengan category) ke domain.Expense.
func mapGetExpenseByIDRow(row GetExpenseByIDRow) *domain.Expense {
	return &domain.Expense{
		ID:            uuidToString(row.ID),
		TenantID:      uuidToString(row.TenantID),
		CategoryID:    uuidToString(row.CategoryID),
		CategoryName:  row.CategoryName,
		Amount:        row.Amount,
		Description:   row.Description,
		ExpenseDate:   dateToTime(row.ExpenseDate),
		PaymentMethod: "",
		VendorName:    "",
		ReferenceNo:   "",
		AttachmentURL: "",
		IsRecurring:   row.IsRecurring,
		RecurringDay:  int4ToIntPtr(row.RecurringDay),
		CreatedByID:   uuidToString(row.CreatedByID),
		DeletedAt:     timestamptzToTimePtr(row.DeletedAt),
		CreatedAt:     timestamptzToTime(row.CreatedAt),
		UpdatedAt:     timestamptzToTime(row.UpdatedAt),
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

// Buat membuat pengeluaran baru dan mengembalikan pengeluaran yang dibuat.
func (r *ExpenseRepo) Create(ctx context.Context, expense *domain.Expense) (*domain.Expense, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO expenses (
			tenant_id, category_id, amount, description, expense_date,
			payment_method, vendor_name, reference_number, attachment_url,
			is_recurring, recurring_day, created_by_id
		) VALUES ($1, $2, $3, $4, $5, NULLIF($6,''), NULLIF($7,''), NULLIF($8,''), NULLIF($9,''), $10, $11, $12)
		RETURNING id::text, tenant_id::text, category_id::text, amount, description,
			expense_date::timestamptz, COALESCE(payment_method,''), COALESCE(vendor_name,''),
			COALESCE(reference_number,''), COALESCE(attachment_url,''), is_recurring, recurring_day,
			created_by_id::text, deleted_at, created_at, updated_at, ''::text`,
		expense.TenantID, expense.CategoryID, expense.Amount, expense.Description, expense.ExpenseDate,
		expense.PaymentMethod, expense.VendorName, expense.ReferenceNo, expense.AttachmentURL,
		expense.IsRecurring, intPtrToInt4(expense.RecurringDay), expense.CreatedByID,
	)
	created, err := scanExpense(row)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal membuat expense: %w", err)
	}
	return created, nil
}

// GetByID mengambil pengeluaran berdasarkan ID (tenant-scoped via RLS).
func (r *ExpenseRepo) GetByID(ctx context.Context, id string) (*domain.Expense, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT e.id::text, e.tenant_id::text, e.category_id::text, e.amount, e.description,
			e.expense_date::timestamptz, COALESCE(e.payment_method,''), COALESCE(e.vendor_name,''),
			COALESCE(e.reference_number,''), COALESCE(e.attachment_url,''), e.is_recurring, e.recurring_day,
			e.created_by_id::text, e.deleted_at, e.created_at, e.updated_at, ec.name
		FROM expenses e
		JOIN expense_categories ec ON ec.id = e.category_id
		WHERE e.id = $1::uuid AND e.deleted_at IS NULL`, id)
	expense, err := scanExpense(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrExpenseNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil expense by ID: %w", err)
	}
	return expense, nil
}

// Perbarui memperbarui data pengeluaran dan mengembalikan pengeluaran yang diperbarui.
func (r *ExpenseRepo) Update(ctx context.Context, expense *domain.Expense) (*domain.Expense, error) {
	row := r.pool.QueryRow(ctx, `
		UPDATE expenses SET
			category_id = $2::uuid,
			amount = $3,
			description = $4,
			expense_date = $5,
			payment_method = NULLIF($6,''),
			vendor_name = NULLIF($7,''),
			reference_number = NULLIF($8,''),
			attachment_url = NULLIF($9,''),
			is_recurring = $10,
			recurring_day = $11,
			updated_at = NOW()
		WHERE id = $1::uuid AND deleted_at IS NULL
		RETURNING id::text, tenant_id::text, category_id::text, amount, description,
			expense_date::timestamptz, COALESCE(payment_method,''), COALESCE(vendor_name,''),
			COALESCE(reference_number,''), COALESCE(attachment_url,''), is_recurring, recurring_day,
			created_by_id::text, deleted_at, created_at, updated_at, ''::text`,
		expense.ID, expense.CategoryID, expense.Amount, expense.Description, expense.ExpenseDate,
		expense.PaymentMethod, expense.VendorName, expense.ReferenceNo, expense.AttachmentURL,
		expense.IsRecurring, intPtrToInt4(expense.RecurringDay),
	)
	updated, err := scanExpense(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrExpenseNotFound
		}
		return nil, fmt.Errorf("repository: gagal memperbarui expense: %w", err)
	}
	return updated, nil
}

// SoftDelete menghapus pengeluaran secara hapus lunak (atur deleted_at).
func (r *ExpenseRepo) SoftDelete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `UPDATE expenses SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1::uuid AND deleted_at IS NULL`, id)
	if err != nil {
		return fmt.Errorf("repository: gagal soft-delete expense: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrExpenseNotFound
	}
	return nil
}

// List mengambil daftar pengeluaran dengan filter periode dan kategori.
func (r *ExpenseRepo) List(ctx context.Context, tenantID string, periodStart, periodEnd time.Time, categoryID string) ([]*domain.Expense, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT e.id::text, e.tenant_id::text, e.category_id::text, e.amount, e.description,
			e.expense_date::timestamptz, COALESCE(e.payment_method,''), COALESCE(e.vendor_name,''),
			COALESCE(e.reference_number,''), COALESCE(e.attachment_url,''), e.is_recurring, e.recurring_day,
			e.created_by_id::text, e.deleted_at, e.created_at, e.updated_at, ec.name
		FROM expenses e
		JOIN expense_categories ec ON ec.id = e.category_id
		WHERE e.tenant_id = $1::uuid
			AND e.expense_date >= $2
			AND e.expense_date <= $3
			AND (NULLIF($4,'') IS NULL OR e.category_id = $4::uuid)
			AND e.deleted_at IS NULL
		ORDER BY e.expense_date DESC, e.created_at DESC`, tenantID, periodStart, periodEnd, categoryID)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil daftar expense: %w", err)
	}
	defer rows.Close()
	expenses := make([]*domain.Expense, 0)
	for rows.Next() {
		expense, err := scanExpense(rows)
		if err != nil {
			return nil, err
		}
		expenses = append(expenses, expense)
	}
	return expenses, rows.Err()
}

type expenseScanner interface {
	Scan(dest ...interface{}) error
}

func scanExpense(row expenseScanner) (*domain.Expense, error) {
	var expense domain.Expense
	var recurringDay pgtype.Int4
	var deletedAt pgtype.Timestamptz
	var categoryName pgtype.Text
	err := row.Scan(
		&expense.ID,
		&expense.TenantID,
		&expense.CategoryID,
		&expense.Amount,
		&expense.Description,
		&expense.ExpenseDate,
		&expense.PaymentMethod,
		&expense.VendorName,
		&expense.ReferenceNo,
		&expense.AttachmentURL,
		&expense.IsRecurring,
		&recurringDay,
		&expense.CreatedByID,
		&deletedAt,
		&expense.CreatedAt,
		&expense.UpdatedAt,
		&categoryName,
	)
	if err != nil {
		return nil, err
	}
	expense.RecurringDay = int4ToIntPtr(recurringDay)
	expense.DeletedAt = timestamptzToTimePtr(deletedAt)
	expense.CategoryName = textToString(categoryName)
	return &expense, nil
}

// ListRecurring mengambil semua pengeluaran berulang yang aktif (untuk auto-buat bulanan).
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
