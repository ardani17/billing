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

// PaymentLinkRepo mengimplementasikan domain.PaymentLinkRepository
// dengan membungkus sqlc-generated Queries.
type PaymentLinkRepo struct {
	// queries adalah sqlc-generated Queries untuk operasi payment_links.
	queries *Queries

	// pool digunakan untuk transaksi pada method Buat.
	pool *pgxpool.Pool
}

// NewPaymentLinkRepo membuat instance baru PaymentLinkRepo.
func NewPaymentLinkRepo(queries *Queries, pool *pgxpool.Pool) *PaymentLinkRepo {
	return &PaymentLinkRepo{
		queries: queries,
		pool:    pool,
	}
}

// --- Helper: mapping sqlc PaymentLink -> domain.PaymentLink ---

// mapPaymentLinkRow memetakan PaymentLink (sqlc model) ke domain.PaymentLink.
// Konversi: pgtype.UUID -> string, pgtype.Timestamptz -> time.Time,
// pgtype.Text -> string.
func mapPaymentLinkRow(row PaymentLink) *domain.PaymentLink {
	return &domain.PaymentLink{
		ID:              uuidToString(row.ID),
		TenantID:        uuidToString(row.TenantID),
		CustomerID:      uuidToString(row.CustomerID),
		GatewayProvider: domain.GatewayProvider(row.GatewayProvider),
		GatewayConfigID: uuidToString(row.GatewayConfigID),
		ExternalID:      row.ExternalID,
		PaymentURL:      row.PaymentUrl,
		Amount:          row.Amount,
		Status:          domain.PaymentLinkStatus(row.Status),
		ExpiresAt:       timestamptzToTime(row.ExpiresAt),
		PaidAt:          timestamptzToTimePtr(row.PaidAt),
		PaidMethod:      textToString(row.PaidMethod),
		CreatedAt:       timestamptzToTime(row.CreatedAt),
		UpdatedAt:       timestamptzToTime(row.UpdatedAt),
	}
}

// --- Implementasi domain.PaymentLinkRepository ---

// Buat membuat link pembayaran baru beserta junction ke invoices (payment_link_invoices).
// Menggunakan transaksi untuk atomicity antara insert payment_link dan payment_link_invoices.
func (r *PaymentLinkRepo) Create(ctx context.Context, link *domain.PaymentLink, invoiceIDs []string) (*domain.PaymentLink, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal memulai transaksi: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := r.queries.WithTx(tx)

	// Insert payment_link
	row, err := qtx.CreatePaymentLink(ctx, CreatePaymentLinkParams{
		TenantID:        stringToUUID(link.TenantID),
		CustomerID:      stringToUUID(link.CustomerID),
		GatewayProvider: string(link.GatewayProvider),
		GatewayConfigID: stringToUUID(link.GatewayConfigID),
		ExternalID:      link.ExternalID,
		PaymentUrl:      link.PaymentURL,
		Amount:          link.Amount,
		Status:          string(link.Status),
		ExpiresAt:       timeToTimestamptz(link.ExpiresAt),
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal membuat payment link: %w", err)
	}

	// Insert junction rows payment_link_invoices
	for _, invoiceID := range invoiceIDs {
		err := qtx.CreatePaymentLinkInvoice(ctx, CreatePaymentLinkInvoiceParams{
			PaymentLinkID: row.ID,
			InvoiceID:     stringToUUID(invoiceID),
		})
		if err != nil {
			return nil, fmt.Errorf("repository: gagal membuat payment_link_invoice: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("repository: gagal commit transaksi: %w", err)
	}

	return mapPaymentLinkRow(row), nil
}

// GetByID mengambil link pembayaran berdasarkan ID (tenant-scoped via RLS).
func (r *PaymentLinkRepo) GetByID(ctx context.Context, id string) (*domain.PaymentLink, error) {
	row, err := r.queries.GetPaymentLinkByID(ctx, stringToUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrPaymentLinkNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil payment link by ID: %w", err)
	}
	return mapPaymentLinkRow(row), nil
}

// GetByExternalID mengambil link pembayaran berdasarkan external_id dari gateway.
func (r *PaymentLinkRepo) GetByExternalID(ctx context.Context, externalID string) (*domain.PaymentLink, error) {
	row, err := r.queries.GetPaymentLinkByExternalID(ctx, externalID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrPaymentLinkNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil payment link by external ID: %w", err)
	}
	return mapPaymentLinkRow(row), nil
}

// GetActiveByCustomer mengambil link pembayaran aktif (status='active') untuk customer.
func (r *PaymentLinkRepo) GetActiveByCustomer(ctx context.Context, customerID string) (*domain.PaymentLink, error) {
	row, err := r.queries.GetActivePaymentLinkByCustomer(ctx, stringToUUID(customerID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrPaymentLinkNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil payment link aktif by customer: %w", err)
	}
	return mapPaymentLinkRow(row), nil
}

// GetInvoiceIDsByLinkID mengambil daftar invoice ID yang terkait dengan link pembayaran.
func (r *PaymentLinkRepo) GetInvoiceIDsByLinkID(ctx context.Context, linkID string) ([]string, error) {
	uuids, err := r.queries.GetInvoiceIDsByPaymentLinkID(ctx, stringToUUID(linkID))
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil invoice IDs by link ID: %w", err)
	}
	ids := make([]string, 0, len(uuids))
	for _, u := range uuids {
		ids = append(ids, uuidToString(u))
	}
	return ids, nil
}

// UpdateStatus memperbarui status link pembayaran.
func (r *PaymentLinkRepo) UpdateStatus(ctx context.Context, id string, status domain.PaymentLinkStatus) error {
	err := r.queries.UpdatePaymentLinkStatus(ctx, UpdatePaymentLinkStatusParams{
		ID:     stringToUUID(id),
		Status: string(status),
	})
	if err != nil {
		return fmt.Errorf("repository: gagal memperbarui status payment link: %w", err)
	}
	return nil
}

// UpdateStatusPaid memperbarui status ke paid beserta metode pembayaran dan waktu bayar.
func (r *PaymentLinkRepo) UpdateStatusPaid(ctx context.Context, id string, paidMethod string, paidAt time.Time) error {
	err := r.queries.UpdatePaymentLinkPaid(ctx, UpdatePaymentLinkPaidParams{
		ID:         stringToUUID(id),
		PaidMethod: pgtype.Text{String: paidMethod, Valid: paidMethod != ""},
		PaidAt:     timeToTimestamptz(paidAt),
	})
	if err != nil {
		return fmt.Errorf("repository: gagal memperbarui payment link ke paid: %w", err)
	}
	return nil
}

// ListByInvoice mengambil semua link pembayarans untuk invoice tertentu (via junction table).
func (r *PaymentLinkRepo) ListByInvoice(ctx context.Context, invoiceID string) ([]*domain.PaymentLink, error) {
	rows, err := r.queries.ListPaymentLinksByInvoice(ctx, stringToUUID(invoiceID))
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil payment link by invoice: %w", err)
	}
	links := make([]*domain.PaymentLink, 0, len(rows))
	for _, row := range rows {
		links = append(links, mapPaymentLinkRow(row))
	}
	return links, nil
}

// FindExpired mengambil link pembayarans yang sudah melewati expires_at tapi masih active.
func (r *PaymentLinkRepo) FindExpired(ctx context.Context, batchSize int) ([]*domain.PaymentLink, error) {
	rows, err := r.queries.FindExpiredPaymentLinks(ctx, int32(batchSize))
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil payment link expired: %w", err)
	}
	links := make([]*domain.PaymentLink, 0, len(rows))
	for _, row := range rows {
		links = append(links, mapPaymentLinkRow(row))
	}
	return links, nil
}

// ExpireByID mengubah status link pembayaran menjadi expired berdasarkan ID.
func (r *PaymentLinkRepo) ExpireByID(ctx context.Context, id string) error {
	err := r.queries.ExpirePaymentLinkByID(ctx, stringToUUID(id))
	if err != nil {
		return fmt.Errorf("repository: gagal meng-expire payment link: %w", err)
	}
	return nil
}
