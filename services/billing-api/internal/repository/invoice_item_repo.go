package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/jackc/pgx/v5/pgtype"
)

// InvoiceItemRepo mengimplementasikan domain.InvoiceItemRepository dengan membungkus
// sqlc-generated Queries untuk operasi invoice items.
type InvoiceItemRepo struct {
	// queries adalah sqlc-generated Queries untuk operasi invoice items.
	queries *Queries
}

// NewInvoiceItemRepo membuat instance baru InvoiceItemRepo.
func NewInvoiceItemRepo(queries *Queries) *InvoiceItemRepo {
	return &InvoiceItemRepo{
		queries: queries,
	}
}

// --- Helper function untuk mapping sqlc InvoiceItem -> domain.InvoiceItem ---

// mapInvoiceItemRow memetakan InvoiceItem (sqlc model) ke domain.InvoiceItem.
func mapInvoiceItemRow(row InvoiceItem) *domain.InvoiceItem {
	item := &domain.InvoiceItem{
		ID:          uuidToString(row.ID),
		TenantID:    uuidToString(row.TenantID),
		InvoiceID:   uuidToString(row.InvoiceID),
		ItemType:    domain.InvoiceItemType(row.ItemType),
		Description: row.Description,
		Quantity:    int(row.Quantity),
		UnitPrice:   row.UnitPrice,
		Amount:      row.Amount,
		SortOrder:   int(row.SortOrder),
		CreatedAt:   timestamptzToTime(row.CreatedAt),
	}

	// Konversi metadata JSON jika ada
	if len(row.Metadata) > 0 {
		var meta map[string]interface{}
		if err := json.Unmarshal(row.Metadata, &meta); err == nil {
			item.Metadata = meta
		}
	}

	return item
}

// --- Implementasi domain.InvoiceItemRepository ---

// BulkCreate membuat beberapa item invoice sekaligus menggunakan PostgreSQL COPY protocol.
// Mengembalikan daftar item yang dibuat (diambil ulang dari database setelah insert).
func (r *InvoiceItemRepo) BulkCreate(ctx context.Context, items []*domain.InvoiceItem) ([]*domain.InvoiceItem, error) {
	if len(items) == 0 {
		return []*domain.InvoiceItem{}, nil
	}

	// Konversi domain items ke sqlc params untuk copyfrom
	params := make([]BulkCreateInvoiceItemsParams, 0, len(items))
	for _, item := range items {
		// Serialisasi metadata ke JSON
		var metadata []byte
		if item.Metadata != nil {
			var err error
			metadata, err = json.Marshal(item.Metadata)
			if err != nil {
				return nil, fmt.Errorf("repository: gagal serialisasi metadata item: %w", err)
			}
		}

		params = append(params, BulkCreateInvoiceItemsParams{
			TenantID:    stringToUUID(item.TenantID),
			InvoiceID:   stringToUUID(item.InvoiceID),
			ItemType:    string(item.ItemType),
			Description: item.Description,
			Quantity:    int32(item.Quantity),
			UnitPrice:   item.UnitPrice,
			Amount:      item.Amount,
			SortOrder:   int32(item.SortOrder),
			Metadata:    metadata,
		})
	}

	// Bulk insert menggunakan COPY protocol
	_, err := r.queries.BulkCreateInvoiceItems(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal bulk create invoice items: %w", err)
	}

	// Ambil ulang items dari database untuk mendapatkan ID dan created_at
	// Gunakan invoice_id dari item pertama (semua item punya invoice_id yang sama)
	invoiceID := stringToUUID(items[0].InvoiceID)
	return r.listByInvoiceUUID(ctx, invoiceID)
}

// ListByInvoice mengambil semua item untuk invoice tertentu (urut berdasarkan urut_order).
func (r *InvoiceItemRepo) ListByInvoice(ctx context.Context, invoiceID string) ([]*domain.InvoiceItem, error) {
	return r.listByInvoiceUUID(ctx, stringToUUID(invoiceID))
}

// listByInvoiceUUID adalah helper internal untuk mengambil items berdasarkan UUID invoice.
func (r *InvoiceItemRepo) listByInvoiceUUID(ctx context.Context, invoiceID pgtype.UUID) ([]*domain.InvoiceItem, error) {
	rows, err := r.queries.ListInvoiceItemsByInvoice(ctx, invoiceID)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil invoice items: %w", err)
	}

	result := make([]*domain.InvoiceItem, 0, len(rows))
	for _, row := range rows {
		result = append(result, mapInvoiceItemRow(row))
	}
	return result, nil
}

// DeleteByInvoice menghapus semua item untuk invoice tertentu (digunakan saat edit invoice).
func (r *InvoiceItemRepo) DeleteByInvoice(ctx context.Context, invoiceID string) error {
	err := r.queries.DeleteInvoiceItemsByInvoice(ctx, stringToUUID(invoiceID))
	if err != nil {
		return fmt.Errorf("repository: gagal menghapus invoice items: %w", err)
	}
	return nil
}
