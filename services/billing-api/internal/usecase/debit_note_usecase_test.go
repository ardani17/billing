// debit_note_usecase_test.go berisi unit test untuk DebitNoteUsecase.
// Menguji pembuatan debit note dengan dan tanpa pembuatan invoice terkait.
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
// Mock repositories khusus untuk DebitNoteUsecase tests
// =============================================================================

// mockDebitNoteRepo adalah implementasi in-memory dari domain.DebitNoteRepository.
type mockDebitNoteRepo struct {
	notes   map[string]*domain.DebitNote
	counter int
}

func newMockDebitNoteRepo() *mockDebitNoteRepo {
	return &mockDebitNoteRepo{notes: make(map[string]*domain.DebitNote)}
}

func (m *mockDebitNoteRepo) Create(_ context.Context, dn *domain.DebitNote) (*domain.DebitNote, error) {
	m.counter++
	dn.ID = fmt.Sprintf("dn-%d", m.counter)
	copy := *dn
	m.notes[copy.ID] = &copy
	return &copy, nil
}

func (m *mockDebitNoteRepo) GetByID(_ context.Context, id string) (*domain.DebitNote, error) {
	dn, ok := m.notes[id]
	if !ok {
		return nil, domain.ErrDebitNoteNotFound
	}
	copy := *dn
	return &copy, nil
}

func (m *mockDebitNoteRepo) ListByCustomer(_ context.Context, customerID string) ([]*domain.DebitNote, error) {
	var result []*domain.DebitNote
	for _, dn := range m.notes {
		if dn.CustomerID == customerID {
			copy := *dn
			result = append(result, &copy)
		}
	}
	return result, nil
}

// =============================================================================
// Helper untuk membuat DebitNoteUsecase dengan mock repos
// =============================================================================

type debitNoteUsecaseSetup struct {
	uc            *DebitNoteUsecase
	debitNoteRepo *mockDebitNoteRepo
	invoiceRepo   *mockInvoiceRepo
	customerRepo  *invMockCustomerRepo
}

func setupDebitNoteUsecase() *debitNoteUsecaseSetup {
	debitNoteRepo := newMockDebitNoteRepo()
	invoiceRepo := newMockInvoiceRepo()
	itemRepo := newMockItemRepo()
	auditRepo := &mockAuditRepo{}
	sequenceRepo := &mockSequenceRepo{}
	customerRepo := newInvMockCustomerRepo()
	settingsRepo := newMockSettingsRepo()
	logger := zerolog.New(io.Discard)

	uc := NewDebitNoteUsecase(
		debitNoteRepo, invoiceRepo, itemRepo, auditRepo,
		sequenceRepo, customerRepo, settingsRepo, nil, logger,
	)

	return &debitNoteUsecaseSetup{
		uc:            uc,
		debitNoteRepo: debitNoteRepo,
		invoiceRepo:   invoiceRepo,
		customerRepo:  customerRepo,
	}
}

// =============================================================================
// Unit Tests — DebitNoteUsecase
// =============================================================================

// TestDebitNote_Create_Success menguji pembuatan debit note berhasil tanpa invoice.
func TestDebitNote_Create_Success(t *testing.T) {
	s := setupDebitNoteUsecase()
	ctx := context.Background()

	s.customerRepo.customers["cust-1"] = &domain.Customer{
		ID: "cust-1", TenantID: "tenant-1",
	}

	req := domain.CreateDebitNoteRequest{
		CustomerID: "cust-1",
		Items: []domain.DebitNoteItemRequest{
			{Description: "Biaya perbaikan kabel", Amount: 75000},
			{Description: "Biaya kunjungan teknisi", Amount: 50000},
		},
		DueDate:       "2024-07-15",
		CreateInvoice: false,
	}

	dn, err := s.uc.Create(ctx, "tenant-1", req, domain.ActorInfo{ActorID: "user-1", ActorName: "Test"})
	if err != nil {
		t.Fatalf("Create gagal: %v", err)
	}

	// Total = 75000 + 50000 = 125000
	if dn.TotalAmount != 125000 {
		t.Fatalf("expected total 125000, got %d", dn.TotalAmount)
	}
	if len(dn.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(dn.Items))
	}
	if dn.InvoiceID != nil {
		t.Fatal("expected no invoice_id")
	}
}

// TestDebitNote_Create_WithInvoice menguji pembuatan debit note dengan invoice terkait.
func TestDebitNote_Create_WithInvoice(t *testing.T) {
	s := setupDebitNoteUsecase()
	ctx := context.Background()

	s.customerRepo.customers["cust-1"] = &domain.Customer{
		ID: "cust-1", TenantID: "tenant-1",
	}

	req := domain.CreateDebitNoteRequest{
		CustomerID: "cust-1",
		Items: []domain.DebitNoteItemRequest{
			{Description: "Biaya tambahan", Amount: 100000},
		},
		DueDate:       "2024-07-15",
		CreateInvoice: true,
	}

	dn, err := s.uc.Create(ctx, "tenant-1", req, domain.ActorInfo{ActorID: "user-1", ActorName: "Test"})
	if err != nil {
		t.Fatalf("Create gagal: %v", err)
	}

	// Harus ada invoice_id karena create_invoice = true
	if dn.InvoiceID == nil {
		t.Fatal("expected invoice_id to be set")
	}

	// Verifikasi invoice dibuat di repo
	if len(s.invoiceRepo.invoices) != 1 {
		t.Fatalf("expected 1 invoice, got %d", len(s.invoiceRepo.invoices))
	}
}

// TestDebitNote_Create_CustomerNotFound menguji error saat pelanggan tidak ditemukan.
func TestDebitNote_Create_CustomerNotFound(t *testing.T) {
	s := setupDebitNoteUsecase()
	ctx := context.Background()

	req := domain.CreateDebitNoteRequest{
		CustomerID: "nonexistent",
		Items: []domain.DebitNoteItemRequest{
			{Description: "Test", Amount: 10000},
		},
		DueDate: "2024-07-15",
	}

	_, err := s.uc.Create(ctx, "tenant-1", req, domain.ActorInfo{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// TestDebitNote_Create_InvalidDueDate menguji error saat format due_date tidak valid.
func TestDebitNote_Create_InvalidDueDate(t *testing.T) {
	s := setupDebitNoteUsecase()
	ctx := context.Background()

	s.customerRepo.customers["cust-1"] = &domain.Customer{
		ID: "cust-1", TenantID: "tenant-1",
	}

	req := domain.CreateDebitNoteRequest{
		CustomerID: "cust-1",
		Items: []domain.DebitNoteItemRequest{
			{Description: "Test", Amount: 10000},
		},
		DueDate: "invalid-date",
	}

	_, err := s.uc.Create(ctx, "tenant-1", req, domain.ActorInfo{})
	if err == nil {
		t.Fatal("expected error for invalid date, got nil")
	}
}
