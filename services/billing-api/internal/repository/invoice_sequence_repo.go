package repository

import (
	"context"
	"fmt"
)

// InvoiceSequenceRepo mengimplementasikan domain.InvoiceSequenceRepository dengan membungkus
// sqlc-generated Queries untuk operasi invoice sequences.
type InvoiceSequenceRepo struct {
	// queries adalah sqlc-generated Queries untuk operasi invoice sequences.
	queries *Queries
}

// NewInvoiceSequenceRepo membuat instance baru InvoiceSequenceRepo.
func NewInvoiceSequenceRepo(queries *Queries) *InvoiceSequenceRepo {
	return &InvoiceSequenceRepo{
		queries: queries,
	}
}

// --- Implementasi domain.InvoiceSequenceRepository ---

// NextSequence mengambil dan increment sequence secara atomik menggunakan INSERT ON CONFLICT.
// Jika row belum ada untuk tenant/year/month, buat baru dengan last_seq = 1.
// Jika sudah ada, increment last_seq dan kembalikan nilai baru.
// Operasi ini atomik dan aman untuk concurrent access.
func (r *InvoiceSequenceRepo) NextSequence(ctx context.Context, tenantID string, year, month int) (int, error) {
	seq, err := r.queries.NextInvoiceSequence(ctx, NextInvoiceSequenceParams{
		TenantID: stringToUUID(tenantID),
		Year:     int32(year),
		Month:    int32(month),
	})
	if err != nil {
		return 0, fmt.Errorf("repository: gagal mengambil next invoice sequence: %w", err)
	}
	return int(seq), nil
}
