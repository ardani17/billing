package repository

import (
	"context"
	"fmt"
)

// ReceiptSequenceRepo mengimplementasikan domain.ReceiptSequenceRepository dengan membungkus
// sqlc-generated Queries untuk operasi receipt sequences.
type ReceiptSequenceRepo struct {
	// queries adalah sqlc-generated Queries untuk operasi receipt sequences.
	queries *Queries
}

// NewReceiptSequenceRepo membuat instance baru ReceiptSequenceRepo.
func NewReceiptSequenceRepo(queries *Queries) *ReceiptSequenceRepo {
	return &ReceiptSequenceRepo{
		queries: queries,
	}
}

// --- Implementasi domain.ReceiptSequenceRepository ---

// NextSequence mengambil dan increment sequence kwitansi secara atomik menggunakan INSERT ON CONFLICT.
// Jika row belum ada untuk tenant/year/month, buat baru dengan last_seq = 1.
// Jika sudah ada, increment last_seq dan kembalikan nilai baru.
// Operasi ini atomik dan aman untuk concurrent access.
func (r *ReceiptSequenceRepo) NextSequence(ctx context.Context, tenantID string, year, month int) (int, error) {
	seq, err := r.queries.NextReceiptSequence(ctx, NextReceiptSequenceParams{
		TenantID: stringToUUID(tenantID),
		Year:     int32(year),
		Month:    int32(month),
	})
	if err != nil {
		return 0, fmt.Errorf("repository: gagal mengambil next receipt sequence: %w", err)
	}
	return int(seq), nil
}
