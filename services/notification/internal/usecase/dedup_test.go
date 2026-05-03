package usecase

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/ispboss/ispboss/services/notification/internal/domain"
	"pgregory.net/rapid"
)

// genNonEmptyNoColon menghasilkan string non-kosong yang tidak mengandung karakter ":".
func genNonEmptyNoColon(t *rapid.T, label string) string {
	return rapid.StringMatching(`[a-zA-Z0-9_\-]{1,30}`).Draw(t, label)
}

// Feature: notification-service, Property 5: Dedup key format consistency
// **Validates: Requirements 9.1**
//
// Untuk setiap tenant_id, customer_id, template_slug, dan periode string
// (non-kosong, tanpa karakter ":"), dedup key yang dihasilkan HARUS sama dengan
// "{tenant_id}:{customer_id}:{template_slug}:{periode}" dan memisahkan key
// berdasarkan ":" HARUS menghasilkan tepat 4 komponen asli.
func TestProperty_DedupKeyFormatConsistency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		tenantID := genNonEmptyNoColon(t, "tenantID")
		customerID := genNonEmptyNoColon(t, "customerID")
		templateSlug := genNonEmptyNoColon(t, "templateSlug")
		periode := genNonEmptyNoColon(t, "periode")

		// Generate dedup key
		key := GenerateDedupKey(tenantID, customerID, templateSlug, periode)

		// Verifikasi format: harus sama dengan "{tenantID}:{customerID}:{templateSlug}:{periode}"
		expected := tenantID + ":" + customerID + ":" + templateSlug + ":" + periode
		if key != expected {
			t.Fatalf(
				"DedupKey format salah:\n  expected: %q\n  got:      %q",
				expected, key,
			)
		}

		// Verifikasi split: memisahkan berdasarkan ":" harus menghasilkan tepat 4 komponen
		parts := strings.Split(key, ":")
		if len(parts) != 4 {
			t.Fatalf(
				"DedupKey split menghasilkan %d komponen, expected 4: %v",
				len(parts), parts,
			)
		}

		// Verifikasi setiap komponen cocok dengan input asli
		if parts[0] != tenantID {
			t.Fatalf("Komponen 0 (tenantID): expected %q, got %q", tenantID, parts[0])
		}
		if parts[1] != customerID {
			t.Fatalf("Komponen 1 (customerID): expected %q, got %q", customerID, parts[1])
		}
		if parts[2] != templateSlug {
			t.Fatalf("Komponen 2 (templateSlug): expected %q, got %q", templateSlug, parts[2])
		}
		if parts[3] != periode {
			t.Fatalf("Komponen 3 (periode): expected %q, got %q", periode, parts[3])
		}
	})
}

// =============================================================================
// Mock LogRepository untuk Property 6
// =============================================================================

// dedupMockLogRepo adalah implementasi in-memory dari domain.LogRepository
// yang melacak dedup key untuk pengujian deduplication invariant.
type dedupMockLogRepo struct {
	seen map[string]bool
}

func newDedupMockLogRepo() *dedupMockLogRepo {
	return &dedupMockLogRepo{seen: make(map[string]bool)}
}

func (m *dedupMockLogRepo) FindByDedupKey(_ context.Context, dedupKey string, _ int) (*domain.NotificationLog, error) {
	if m.seen[dedupKey] {
		return &domain.NotificationLog{DedupKey: dedupKey, Status: domain.StatusSent}, nil
	}
	return nil, nil
}

// MarkSent menandai dedup key sebagai sudah terkirim (dipanggil setelah CheckDuplicate pertama).
func (m *dedupMockLogRepo) MarkSent(dedupKey string) {
	m.seen[dedupKey] = true
}

// Method-method berikut tidak digunakan dalam test ini, hanya untuk memenuhi interface.
func (m *dedupMockLogRepo) Create(_ context.Context, _ *domain.NotificationLog) (*domain.NotificationLog, error) {
	return nil, nil
}
func (m *dedupMockLogRepo) GetByID(_ context.Context, _ string) (*domain.NotificationLog, error) {
	return nil, nil
}
func (m *dedupMockLogRepo) Update(_ context.Context, _ *domain.NotificationLog) error { return nil }
func (m *dedupMockLogRepo) List(_ context.Context, _ domain.LogListParams) (*domain.LogListResult, error) {
	return nil, nil
}
func (m *dedupMockLogRepo) CountTodayByCustomer(_ context.Context, _, _, _ string) (int, error) {
	return 0, nil
}
func (m *dedupMockLogRepo) LastSentToCustomer(_ context.Context, _, _ string) (*time.Time, error) {
	return nil, nil
}

// Feature: notification-service, Property 6: Deduplication invariant
// **Validates: Requirements 9.5**
//
// Untuk setiap urutan N notifikasi (N >= 2) dengan dedup_key yang sama
// yang diproses dalam jendela waktu 1 jam, hanya notifikasi pertama yang
// harus dikirim (CheckDuplicate = false); semua notifikasi berikutnya
// harus di-skip (CheckDuplicate = true).
func TestProperty_DeduplicationInvariant(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate jumlah notifikasi acak (2-10)
		n := rapid.IntRange(2, 10).Draw(t, "n")

		// Generate dedup key acak
		dedupKey := genNonEmptyNoColon(t, "dedupKey")

		// Buat mock repo dan dedup checker
		repo := newDedupMockLogRepo()
		checker := NewDedupChecker(repo)
		ctx := context.Background()

		for i := 0; i < n; i++ {
			isDup, err := checker.CheckDuplicate(ctx, dedupKey)
			if err != nil {
				t.Fatalf("CheckDuplicate gagal pada iterasi %d: %v", i, err)
			}

			if i == 0 {
				// Notifikasi pertama: harus BUKAN duplikat
				if isDup {
					t.Fatalf(
						"Notifikasi pertama (i=0) seharusnya bukan duplikat, tapi CheckDuplicate mengembalikan true untuk key %q",
						dedupKey,
					)
				}
				// Simulasikan pengiriman berhasil: tandai key sebagai sudah terkirim
				repo.MarkSent(dedupKey)
			} else {
				// Notifikasi ke-2 dan seterusnya: harus duplikat
				if !isDup {
					t.Fatalf(
						"Notifikasi ke-%d seharusnya duplikat, tapi CheckDuplicate mengembalikan false untuk key %q",
						i+1, dedupKey,
					)
				}
			}
		}
	})
}
