// gateway_worker_test.go berisi unit test untuk GatewayWorker.
// - handleExpirePaymentLinks: mencari dan expire batch tautan pembayaran
// - handleCleanupWebhookLogs: menghapus webhook logs lama sesuai retensi
package worker

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// =============================================================================
// =============================================================================

// mockLinkRepo mengimplementasikan subset PaymentLinkRepository untuk worker test.
type mockLinkRepo struct {
	links      []*domain.PaymentLink
	expiredIDs []string // ID yang sudah di-expire
}

func (m *mockLinkRepo) FindExpired(_ context.Context, batchSize int) ([]*domain.PaymentLink, error) {
	if batchSize > len(m.links) {
		return m.links, nil
	}
	return m.links[:batchSize], nil
}

func (m *mockLinkRepo) ExpireByID(_ context.Context, id string) error {
	m.expiredIDs = append(m.expiredIDs, id)
	return nil
}

// Method lain yang tidak dipakai oleh handler yang ditest.
func (m *mockLinkRepo) Create(_ context.Context, _ *domain.PaymentLink, _ []string) (*domain.PaymentLink, error) {
	return nil, nil
}
func (m *mockLinkRepo) GetByID(_ context.Context, _ string) (*domain.PaymentLink, error) {
	return nil, nil
}
func (m *mockLinkRepo) GetByExternalID(_ context.Context, _ string) (*domain.PaymentLink, error) {
	return nil, nil
}
func (m *mockLinkRepo) GetActiveByCustomer(_ context.Context, _ string) (*domain.PaymentLink, error) {
	return nil, nil
}
func (m *mockLinkRepo) GetInvoiceIDsByLinkID(_ context.Context, _ string) ([]string, error) {
	return nil, nil
}
func (m *mockLinkRepo) UpdateStatus(_ context.Context, _ string, _ domain.PaymentLinkStatus) error {
	return nil
}
func (m *mockLinkRepo) UpdateStatusPaid(_ context.Context, _ string, _ string, _ time.Time) error {
	return nil
}
func (m *mockLinkRepo) ListByInvoice(_ context.Context, _ string) ([]*domain.PaymentLink, error) {
	return nil, nil
}

// =============================================================================
// =============================================================================

// mockWebhookRepo mengimplementasikan subset WebhookLogRepository untuk worker test.
type mockWebhookRepo struct {
	deletedBefore time.Time // waktu cutoff yang diterima DeleteOlderThan
	deletedCount  int64     // jumlah baris yang "dihapus"
}

func (m *mockWebhookRepo) DeleteOlderThan(_ context.Context, olderThan time.Time) (int64, error) {
	m.deletedBefore = olderThan
	return m.deletedCount, nil
}

// Method lain yang tidak dipakai oleh handler yang ditest.
func (m *mockWebhookRepo) Create(_ context.Context, _ *domain.WebhookLog) (*domain.WebhookLog, error) {
	return nil, nil
}
func (m *mockWebhookRepo) GetByID(_ context.Context, _ string) (*domain.WebhookLog, error) {
	return nil, nil
}
func (m *mockWebhookRepo) UpdateStatus(_ context.Context, _ string, _ domain.WebhookProcessingStatus, _ string) error {
	return nil
}
func (m *mockWebhookRepo) UpdateSignatureValid(_ context.Context, _ string, _ bool) error {
	return nil
}
func (m *mockWebhookRepo) IsAlreadyProcessed(_ context.Context, _, _ string) (bool, error) {
	return false, nil
}
func (m *mockWebhookRepo) ListByPaymentLink(_ context.Context, _ string) ([]*domain.WebhookLog, error) {
	return nil, nil
}

// =============================================================================
// Tes: handleExpirePaymentLinks - mencari dan expire batch
// =============================================================================

func TestHandleExpirePaymentLinks_FindsAndExpiresBatch(t *testing.T) {
	linkRepo := &mockLinkRepo{
		links: []*domain.PaymentLink{
			{ID: "link-1", Status: domain.PaymentLinkActive},
			{ID: "link-2", Status: domain.PaymentLinkActive},
			{ID: "link-3", Status: domain.PaymentLinkActive},
		},
	}
	w := NewGatewayWorker(nil, nil, linkRepo, &mockWebhookRepo{}, 90, zerolog.New(io.Discard))

	task := asynq.NewTask(TaskExpirePaymentLinks, nil)
	err := w.handleExpirePaymentLinks(context.Background(), task)
	if err != nil {
		t.Fatalf("handleExpirePaymentLinks gagal: %v", err)
	}

	// Verifikasi semua link di-expire
	if len(linkRepo.expiredIDs) != 3 {
		t.Fatalf("expected 3 expired, got %d", len(linkRepo.expiredIDs))
	}
	expected := map[string]bool{"link-1": true, "link-2": true, "link-3": true}
	for _, id := range linkRepo.expiredIDs {
		if !expected[id] {
			t.Fatalf("unexpected expired ID: %s", id)
		}
	}
}

// TestHandleExpirePaymentLinks_EmptyBatch menguji saat tidak ada link expired.
func TestHandleExpirePaymentLinks_EmptyBatch(t *testing.T) {
	linkRepo := &mockLinkRepo{links: nil}
	w := NewGatewayWorker(nil, nil, linkRepo, &mockWebhookRepo{}, 90, zerolog.New(io.Discard))

	task := asynq.NewTask(TaskExpirePaymentLinks, nil)
	err := w.handleExpirePaymentLinks(context.Background(), task)
	if err != nil {
		t.Fatalf("handleExpirePaymentLinks gagal: %v", err)
	}
	if len(linkRepo.expiredIDs) != 0 {
		t.Fatalf("expected 0 expired, got %d", len(linkRepo.expiredIDs))
	}
}

// =============================================================================
// Tes: handleCleanupWebhookLogs - panggil DeleteOlderThan dengan retensi benar
// =============================================================================

func TestHandleCleanupWebhookLogs_CorrectRetention(t *testing.T) {
	webhookRepo := &mockWebhookRepo{deletedCount: 42}
	w := NewGatewayWorker(nil, nil, &mockLinkRepo{}, webhookRepo, 90, zerolog.New(io.Discard))

	before := time.Now()
	task := asynq.NewTask(TaskCleanupWebhookLogs, nil)
	err := w.handleCleanupWebhookLogs(context.Background(), task)
	if err != nil {
		t.Fatalf("handleCleanupWebhookLogs gagal: %v", err)
	}

	// Verifikasi cutoff sekitar 90 hari yang lalu (toleransi 1 menit)
	expectedCutoff := before.AddDate(0, 0, -90)
	diff := webhookRepo.deletedBefore.Sub(expectedCutoff)
	if diff < -time.Minute || diff > time.Minute {
		t.Fatalf("cutoff tidak sesuai: expected ~%v, got %v", expectedCutoff, webhookRepo.deletedBefore)
	}
}

// TestHandleCleanupWebhookLogs_CustomRetention menguji retensi kustom (30 hari).
func TestHandleCleanupWebhookLogs_CustomRetention(t *testing.T) {
	webhookRepo := &mockWebhookRepo{deletedCount: 10}
	w := NewGatewayWorker(nil, nil, &mockLinkRepo{}, webhookRepo, 30, zerolog.New(io.Discard))

	before := time.Now()
	task := asynq.NewTask(TaskCleanupWebhookLogs, nil)
	err := w.handleCleanupWebhookLogs(context.Background(), task)
	if err != nil {
		t.Fatalf("handleCleanupWebhookLogs gagal: %v", err)
	}

	// Verifikasi cutoff sekitar 30 hari yang lalu
	expectedCutoff := before.AddDate(0, 0, -30)
	diff := webhookRepo.deletedBefore.Sub(expectedCutoff)
	if diff < -time.Minute || diff > time.Minute {
		t.Fatalf("cutoff tidak sesuai: expected ~%v, got %v", expectedCutoff, webhookRepo.deletedBefore)
	}
}

// TestHandleCleanupWebhookLogs_DefaultRetention menguji bawaan retensi saat 0.
func TestHandleCleanupWebhookLogs_DefaultRetention(t *testing.T) {
	webhookRepo := &mockWebhookRepo{deletedCount: 5}
	// retentionDays=0 harus bawaan ke 90 di NewGatewayWorker
	w := NewGatewayWorker(nil, nil, &mockLinkRepo{}, webhookRepo, 0, zerolog.New(io.Discard))

	before := time.Now()
	task := asynq.NewTask(TaskCleanupWebhookLogs, nil)
	err := w.handleCleanupWebhookLogs(context.Background(), task)
	if err != nil {
		t.Fatalf("handleCleanupWebhookLogs gagal: %v", err)
	}

	// Bawaan 90 hari
	expectedCutoff := before.AddDate(0, 0, -90)
	diff := webhookRepo.deletedBefore.Sub(expectedCutoff)
	if diff < -time.Minute || diff > time.Minute {
		t.Fatalf("cutoff tidak sesuai: expected ~%v, got %v", expectedCutoff, webhookRepo.deletedBefore)
	}
}
