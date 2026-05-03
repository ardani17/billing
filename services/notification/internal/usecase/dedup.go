package usecase

import (
	"context"
	"fmt"

	"github.com/ispboss/ispboss/services/notification/internal/domain"
)

// =============================================================================
// DedupChecker — pengecekan duplikasi notifikasi
// =============================================================================

// DedupChecker bertanggung jawab untuk mengecek apakah notifikasi sudah pernah
// dikirim dalam jendela waktu tertentu, mencegah pengiriman duplikat akibat
// event retry atau cron overlap.
type DedupChecker struct {
	logRepo domain.LogRepository
}

// NewDedupChecker membuat instance baru DedupChecker dengan dependency LogRepository.
func NewDedupChecker(logRepo domain.LogRepository) *DedupChecker {
	return &DedupChecker{logRepo: logRepo}
}

// GenerateDedupKey menghasilkan kunci deduplikasi dengan format:
// "{tenantID}:{customerID}:{templateSlug}:{periode}"
// Kunci ini digunakan untuk mengidentifikasi notifikasi yang sama
// dalam jendela waktu 1 jam.
func GenerateDedupKey(tenantID, customerID, templateSlug, periode string) string {
	return fmt.Sprintf("%s:%s:%s:%s", tenantID, customerID, templateSlug, periode)
}

// CheckDuplicate mengecek apakah notifikasi dengan dedup_key yang sama
// sudah ada dalam jendela waktu 1 jam terakhir.
// Mengembalikan true jika duplikat ditemukan, false jika tidak.
func (d *DedupChecker) CheckDuplicate(ctx context.Context, dedupKey string) (bool, error) {
	// Cari log dengan dedup_key yang sama dalam 1 jam terakhir
	existing, err := d.logRepo.FindByDedupKey(ctx, dedupKey, 1)
	if err != nil {
		return false, fmt.Errorf("gagal mengecek duplikasi: %w", err)
	}

	// Jika ditemukan log yang cocok, berarti duplikat
	return existing != nil, nil
}
