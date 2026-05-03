package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// RecurringItemRepo mengimplementasikan domain.CustomerRecurringItemRepository dengan membungkus
// sqlc-generated Queries untuk operasi customer recurring items.
type RecurringItemRepo struct {
	// queries adalah sqlc-generated Queries untuk operasi customer recurring items.
	queries *Queries
}

// NewRecurringItemRepo membuat instance baru RecurringItemRepo.
func NewRecurringItemRepo(queries *Queries) *RecurringItemRepo {
	return &RecurringItemRepo{
		queries: queries,
	}
}

// --- Helper function untuk konversi nullable date ---

// dateToTimePtr mengkonversi pgtype.Date ke *time.Time.
// Mengembalikan nil jika Date tidak valid (NULL).
func dateToTimePtr(d pgtype.Date) *time.Time {
	if !d.Valid {
		return nil
	}
	t := d.Time
	return &t
}

// timePtrToDate mengkonversi *time.Time ke pgtype.Date.
// Mengembalikan Date tidak valid jika pointer nil.
func timePtrToDate(t *time.Time) pgtype.Date {
	if t == nil {
		return pgtype.Date{Valid: false}
	}
	return pgtype.Date{Time: *t, Valid: true}
}

// --- Helper function untuk mapping sqlc CustomerRecurringItem → domain.CustomerRecurringItem ---

// mapRecurringItemRow memetakan CustomerRecurringItem (sqlc model) ke domain.CustomerRecurringItem.
func mapRecurringItemRow(row CustomerRecurringItem) *domain.CustomerRecurringItem {
	return &domain.CustomerRecurringItem{
		ID:          uuidToString(row.ID),
		TenantID:    uuidToString(row.TenantID),
		CustomerID:  uuidToString(row.CustomerID),
		Description: row.Description,
		Amount:      row.Amount,
		IsActive:    row.IsActive,
		StartDate:   dateToTime(row.StartDate),
		EndDate:     dateToTimePtr(row.EndDate),
		CreatedAt:   timestamptzToTime(row.CreatedAt),
		UpdatedAt:   timestamptzToTime(row.UpdatedAt),
	}
}

// --- Implementasi domain.CustomerRecurringItemRepository ---

// Create membuat recurring item baru dan mengembalikan item yang dibuat.
func (r *RecurringItemRepo) Create(ctx context.Context, item *domain.CustomerRecurringItem) (*domain.CustomerRecurringItem, error) {
	row, err := r.queries.CreateRecurringItem(ctx, CreateRecurringItemParams{
		TenantID:    stringToUUID(item.TenantID),
		CustomerID:  stringToUUID(item.CustomerID),
		Description: item.Description,
		Amount:      item.Amount,
		IsActive:    item.IsActive,
		StartDate:   timeToDate(item.StartDate),
		EndDate:     timePtrToDate(item.EndDate),
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal membuat recurring item: %w", err)
	}
	return mapRecurringItemRow(row), nil
}

// GetByID mengambil recurring item berdasarkan ID.
// Mengembalikan ErrRecurringItemNotFound jika tidak ditemukan.
func (r *RecurringItemRepo) GetByID(ctx context.Context, id string) (*domain.CustomerRecurringItem, error) {
	row, err := r.queries.GetRecurringItemByID(ctx, stringToUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrRecurringItemNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil recurring item by ID: %w", err)
	}
	return mapRecurringItemRow(row), nil
}

// Update memperbarui recurring item dan mengembalikan item yang diperbarui.
// Mengembalikan ErrRecurringItemNotFound jika tidak ditemukan.
func (r *RecurringItemRepo) Update(ctx context.Context, item *domain.CustomerRecurringItem) (*domain.CustomerRecurringItem, error) {
	row, err := r.queries.UpdateRecurringItem(ctx, UpdateRecurringItemParams{
		ID:          stringToUUID(item.ID),
		Description: item.Description,
		Amount:      item.Amount,
		EndDate:     timePtrToDate(item.EndDate),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrRecurringItemNotFound
		}
		return nil, fmt.Errorf("repository: gagal memperbarui recurring item: %w", err)
	}
	return mapRecurringItemRow(row), nil
}

// Deactivate menonaktifkan recurring item (set is_active = false).
func (r *RecurringItemRepo) Deactivate(ctx context.Context, id string) error {
	err := r.queries.DeactivateRecurringItem(ctx, stringToUUID(id))
	if err != nil {
		return fmt.Errorf("repository: gagal menonaktifkan recurring item: %w", err)
	}
	return nil
}

// ListByCustomer mengambil semua recurring item untuk customer tertentu.
func (r *RecurringItemRepo) ListByCustomer(ctx context.Context, customerID string) ([]*domain.CustomerRecurringItem, error) {
	rows, err := r.queries.ListRecurringItemsByCustomer(ctx, stringToUUID(customerID))
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil recurring items by customer: %w", err)
	}

	result := make([]*domain.CustomerRecurringItem, 0, len(rows))
	for _, row := range rows {
		result = append(result, mapRecurringItemRow(row))
	}
	return result, nil
}

// ListActiveByCustomer mengambil recurring item aktif untuk customer pada tanggal periode tertentu.
// Item aktif: is_active = true, start_date <= periodDate, dan (end_date IS NULL atau end_date > periodDate).
func (r *RecurringItemRepo) ListActiveByCustomer(ctx context.Context, customerID string, periodDate time.Time) ([]*domain.CustomerRecurringItem, error) {
	rows, err := r.queries.ListActiveRecurringItemsByCustomer(ctx, ListActiveRecurringItemsByCustomerParams{
		CustomerID: stringToUUID(customerID),
		StartDate:  timeToDate(periodDate),
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil active recurring items: %w", err)
	}

	result := make([]*domain.CustomerRecurringItem, 0, len(rows))
	for _, row := range rows {
		result = append(result, mapRecurringItemRow(row))
	}
	return result, nil
}
