package repository

import (
	"context"
	"fmt"
	"math"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// ResellerTxRepo mengimplementasikan domain.ResellerTransactionRepository dengan membungkus
// sqlc-generated Queries untuk operasi transaksi reseller.
type ResellerTxRepo struct {
	// queries adalah sqlc-generated Queries untuk operasi transaksi reseller.
	queries *Queries
}

// NewResellerTxRepo membuat instance baru ResellerTxRepo.
func NewResellerTxRepo(queries *Queries) *ResellerTxRepo {
	return &ResellerTxRepo{
		queries: queries,
	}
}

// --- Helper function untuk mapping sqlc ResellerTransaction → domain.ResellerTransaction ---

// mapResellerTxRow memetakan ResellerTransaction (sqlc model) ke domain.ResellerTransaction.
func mapResellerTxRow(row ResellerTransaction) *domain.ResellerTransaction {
	return &domain.ResellerTransaction{
		ID:            uuidToString(row.ID),
		TenantID:      uuidToString(row.TenantID),
		ResellerID:    uuidToString(row.ResellerID),
		Type:          domain.TransactionType(row.Type),
		Amount:        row.Amount,
		BalanceBefore: row.BalanceBefore,
		BalanceAfter:  row.BalanceAfter,
		ReferenceID:   uuidToString(row.ReferenceID),
		Notes:         textToString(row.Notes),
		CreatedAt:     timestamptzToTime(row.CreatedAt),
	}
}

// --- Implementasi domain.ResellerTransactionRepository ---

// Create membuat satu transaksi reseller dan mengembalikan transaksi yang dibuat.
func (r *ResellerTxRepo) Create(ctx context.Context, tx *domain.ResellerTransaction) (*domain.ResellerTransaction, error) {
	row, err := r.queries.CreateResellerTransaction(ctx, CreateResellerTransactionParams{
		TenantID:      stringToUUID(tx.TenantID),
		ResellerID:    stringToUUID(tx.ResellerID),
		Type:          string(tx.Type),
		Amount:        tx.Amount,
		BalanceBefore: tx.BalanceBefore,
		BalanceAfter:  tx.BalanceAfter,
		ReferenceID:   stringToUUID(tx.ReferenceID),
		Notes:         stringToText(tx.Notes),
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal membuat transaksi reseller: %w", err)
	}
	return mapResellerTxRow(row), nil
}

// ListByReseller mengambil daftar transaksi reseller dengan paginasi.
// Mengembalikan hasil beserta metadata paginasi (total, page, page_size, total_pages).
func (r *ResellerTxRepo) ListByReseller(ctx context.Context, params domain.ResellerTxListParams) (*domain.ResellerTxListResult, error) {
	// Default values
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 {
		params.PageSize = 25
	}

	// Hitung total transaksi untuk pagination metadata
	total, err := r.queries.CountResellerTransactions(ctx, CountResellerTransactionsParams{
		TenantID:   stringToUUID(params.TenantID),
		ResellerID: stringToUUID(params.ResellerID),
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal menghitung total transaksi reseller: %w", err)
	}

	// Hitung offset untuk pagination
	offset := (params.Page - 1) * params.PageSize

	// Ambil data transaksi
	rows, err := r.queries.ListResellerTransactions(ctx, ListResellerTransactionsParams{
		TenantID:   stringToUUID(params.TenantID),
		ResellerID: stringToUUID(params.ResellerID),
		Limit:      int32(params.PageSize),
		Offset:     int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil daftar transaksi reseller: %w", err)
	}

	// Mapping sqlc rows ke domain model
	transactions := make([]*domain.ResellerTransaction, 0, len(rows))
	for _, row := range rows {
		transactions = append(transactions, mapResellerTxRow(row))
	}

	// Hitung total halaman
	totalPages := int(math.Ceil(float64(total) / float64(params.PageSize)))
	if totalPages < 1 {
		totalPages = 1
	}

	return &domain.ResellerTxListResult{
		Data: transactions,
		Pagination: domain.PaginationMeta{
			Total:      total,
			Page:       params.Page,
			PageSize:   params.PageSize,
			TotalPages: totalPages,
		},
	}, nil
}

// ListDepositsByReseller mengambil daftar deposit reseller dengan paginasi.
// Hanya mengembalikan transaksi dengan type='deposit'.
func (r *ResellerTxRepo) ListDepositsByReseller(ctx context.Context, params domain.ResellerTxListParams) (*domain.ResellerTxListResult, error) {
	// Default values
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 {
		params.PageSize = 25
	}

	// Hitung total deposit untuk pagination metadata
	total, err := r.queries.CountResellerDeposits(ctx, CountResellerDepositsParams{
		TenantID:   stringToUUID(params.TenantID),
		ResellerID: stringToUUID(params.ResellerID),
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal menghitung total deposit reseller: %w", err)
	}

	// Hitung offset untuk pagination
	offset := (params.Page - 1) * params.PageSize

	// Ambil data deposit
	rows, err := r.queries.ListResellerDeposits(ctx, ListResellerDepositsParams{
		TenantID:   stringToUUID(params.TenantID),
		ResellerID: stringToUUID(params.ResellerID),
		Limit:      int32(params.PageSize),
		Offset:     int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil daftar deposit reseller: %w", err)
	}

	// Mapping sqlc rows ke domain model
	transactions := make([]*domain.ResellerTransaction, 0, len(rows))
	for _, row := range rows {
		transactions = append(transactions, mapResellerTxRow(row))
	}

	// Hitung total halaman
	totalPages := int(math.Ceil(float64(total) / float64(params.PageSize)))
	if totalPages < 1 {
		totalPages = 1
	}

	return &domain.ResellerTxListResult{
		Data: transactions,
		Pagination: domain.PaginationMeta{
			Total:      total,
			Page:       params.Page,
			PageSize:   params.PageSize,
			TotalPages: totalPages,
		},
	}, nil
}
