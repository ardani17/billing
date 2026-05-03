// credit_note_usecase_test.go berisi unit test untuk CreditNoteUsecase.
// Menguji pembuatan credit note dengan update saldo kredit pelanggan.
package usecase

import (
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// =============================================================================
// Mock repositories khusus untuk CreditNoteUsecase tests
// =============================================================================

// mockCreditNoteRepo adalah implementasi in-memory dari domain.CreditNoteRepository.
type mockCreditNoteRepo struct {
	notes   map[string]*domain.CreditNote
	counter int
}

func newMockCreditNoteRepo() *mockCreditNoteRepo {
	return &mockCreditNoteRepo{notes: make(map[string]*domain.CreditNote)}
}

func (m *mockCreditNoteRepo) Create(_ context.Context, cn *domain.CreditNote) (*domain.CreditNote, error) {
	m.counter++
	cn.ID = fmt.Sprintf("cn-%d", m.counter)
	copy := *cn
	m.notes[copy.ID] = &copy
	return &copy, nil
}

func (m *mockCreditNoteRepo) GetByID(_ context.Context, id string) (*domain.CreditNote, error) {
	cn, ok := m.notes[id]
	if !ok {
		return nil, domain.ErrCreditNoteNotFound
	}
	copy := *cn
	return &copy, nil
}

func (m *mockCreditNoteRepo) ListByInvoice(_ context.Context, invoiceID string) ([]*domain.CreditNote, error) {
	var result []*domain.CreditNote
	for _, cn := range m.notes {
		if cn.InvoiceID == invoiceID {
			copy := *cn
			result = append(result, &copy)
		}
	}
	return result, nil
}

// =============================================================================
// Helper untuk membuat CreditNoteUsecase dengan mock repos
// =============================================================================

type creditNoteUsecaseSetup struct {
	uc             *CreditNoteUsecase
	creditNoteRepo *mockCreditNoteRepo
	invoiceRepo    *mockInvoiceRepo
	customerRepo   *invMockCustomerRepo
}

func setupCreditNoteUsecase() *creditNoteUsecaseSetup {
	creditNoteRepo := newMockCreditNoteRepo()
	invoiceRepo := newMockInvoiceRepo()
	auditRepo := &mockAuditRepo{}
	sequenceRepo := &mockSequenceRepo{}
	customerRepo := newInvMockCustomerRepo()
	logger := zerolog.New(io.Discard)

	uc := NewCreditNoteUsecase(
		creditNoteRepo, invoiceRepo, auditRepo, sequenceRepo,
		customerRepo, nil, logger,
	)

	return &creditNoteUsecaseSetup{
		uc:             uc,
		creditNoteRepo: creditNoteRepo,
		invoiceRepo:    invoiceRepo,
		customerRepo:   customerRepo,
	}
}

// =============================================================================
// Unit Tests — CreditNoteUsecase
// =============================================================================

// TestCreditNote_Create_WithCreditUpdate menguji pembuatan credit note dengan update saldo kredit.
func TestCreditNote_Create_WithCreditUpdate(t *testing.T) {
	s := setupCreditNoteUsecase()
	ctx := context.Background()

	// Setup invoice dan pelanggan
	s.invoiceRepo.invoices["inv-1"] = &domain.Invoice{
		ID: "inv-1", TenantID: "tenant-1", CustomerID: "cust-1",
	}
	s.customerRepo.customers["cust-1"] = &domain.Customer{
		ID: "cust-1", TenantID: "tenant-1", CreditBalance: 10000,
	}

	applyToCredit := true
	req := domain.CreateCreditNoteRequest{
		InvoiceID:     "inv-1",
		Amount:        50000,
		Reason:        "Koreksi tagihan yang salah",
		ApplyToCredit: &applyToCredit,
	}

	cn, err := s.uc.Create(ctx, "tenant-1", req, domain.ActorInfo{ActorID: "user-1", ActorName: "Test"})
	if err != nil {
		t.Fatalf("Create gagal: %v", err)
	}

	if cn.Amount != 50000 {
		t.Fatalf("expected amount 50000, got %d", cn.Amount)
	}
	if !cn.ApplyToCredit {
		t.Fatal("expected apply_to_credit true")
	}

	// Verifikasi saldo kredit pelanggan bertambah
	cust := s.customerRepo.customers["cust-1"]
	if cust.CreditBalance != 60000 {
		t.Fatalf("expected credit_balance 60000, got %d", cust.CreditBalance)
	}
}

// TestCreditNote_Create_WithoutCreditUpdate menguji pembuatan credit note tanpa update saldo.
func TestCreditNote_Create_WithoutCreditUpdate(t *testing.T) {
	s := setupCreditNoteUsecase()
	ctx := context.Background()

	s.invoiceRepo.invoices["inv-1"] = &domain.Invoice{
		ID: "inv-1", TenantID: "tenant-1", CustomerID: "cust-1",
	}
	s.customerRepo.customers["cust-1"] = &domain.Customer{
		ID: "cust-1", TenantID: "tenant-1", CreditBalance: 10000,
	}

	applyToCredit := false
	req := domain.CreateCreditNoteRequest{
		InvoiceID:     "inv-1",
		Amount:        50000,
		Reason:        "Koreksi tanpa kredit update",
		ApplyToCredit: &applyToCredit,
	}

	_, err := s.uc.Create(ctx, "tenant-1", req, domain.ActorInfo{ActorID: "user-1"})
	if err != nil {
		t.Fatalf("Create gagal: %v", err)
	}

	// Saldo kredit tidak berubah
	cust := s.customerRepo.customers["cust-1"]
	if cust.CreditBalance != 10000 {
		t.Fatalf("expected credit_balance 10000, got %d", cust.CreditBalance)
	}
}

// TestCreditNote_Create_InvoiceNotFound menguji error saat invoice tidak ditemukan.
func TestCreditNote_Create_InvoiceNotFound(t *testing.T) {
	s := setupCreditNoteUsecase()
	ctx := context.Background()

	req := domain.CreateCreditNoteRequest{
		InvoiceID: "nonexistent",
		Amount:    50000,
		Reason:    "Alasan yang valid untuk testing",
	}

	_, err := s.uc.Create(ctx, "tenant-1", req, domain.ActorInfo{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
