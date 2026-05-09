// Karena RecordMultiPayment dan PayAll membutuhkan pool (transaksi DB),
// serta test PayAll yang mendelegasikan ke RecordMultiPayment.
package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// =============================================================================
// Tes: RecordMultiPayment - FIFO allocation (validasi sebelum transaksi)
// =============================================================================

// TestPaymentUsecase_RecordMultiPayment_NoPool menguji bahwa RecordMultiPayment
// membutuhkan pool untuk transaksi. Karena pool.Begin pada nil pool panic,
// kita test format tanggal yang gagal sebelum pool.Begin.
func TestPaymentUsecase_RecordMultiPayment_NoPool(t *testing.T) {
	s := setupPaymentUsecase()
	ctx := context.Background()

	req := domain.MultiPaymentRequest{
		CustomerID:    "cust-1",
		Amount:        100000,
		PaymentMethod: "tunai",
		PaymentDate:   "15-06-2024", // format salah
	}
	actor := domain.ActorInfo{ActorID: "user-1", ActorName: "Test User"}

	_, err := s.uc.RecordMultiPayment(ctx, req, actor)
	if err == nil {
		t.Fatal("expected error untuk format tanggal tidak valid, got nil")
	}
}

func TestPaymentUsecase_RecordMultiPayment_InvalidDateFormat(t *testing.T) {
	s := setupPaymentUsecase()
	ctx := context.Background()

	req := domain.MultiPaymentRequest{
		CustomerID:    "cust-1",
		Amount:        100000,
		PaymentMethod: "tunai",
		PaymentDate:   "2024/06/15", // format salah (slash bukan dash)
	}
	actor := domain.ActorInfo{ActorID: "user-1", ActorName: "Test User"}

	_, err := s.uc.RecordMultiPayment(ctx, req, actor)
	if err == nil {
		t.Fatal("expected error untuk format tanggal tidak valid, got nil")
	}
}

// TestPaymentUsecase_RecordMultiPayment_FIFOAllocation menguji alokasi FIFO
// melalui domain function secara langsung (karena usecase butuh pool).
func TestPaymentUsecase_RecordMultiPayment_FIFOAllocation(t *testing.T) {
	// Tes FIFO allocation secara langsung karena RecordMultiPayment butuh pool
	invoices := []domain.FIFOInput{
		{InvoiceID: "inv-1", InvoiceNumber: "INV-001", TotalAmount: 100000, PaidAmount: 0},
		{InvoiceID: "inv-2", InvoiceNumber: "INV-002", TotalAmount: 150000, PaidAmount: 50000},
	}

	// Bayar 120000 -> inv-1 lunas (100000), inv-2 bayar_sebagian (20000)
	result := domain.AllocatePaymentFIFO(invoices, 120000)

	if len(result.Allocations) != 2 {
		t.Fatalf("expected 2 allocations, got %d", len(result.Allocations))
	}

	// Verifikasi alokasi pertama: inv-1 lunas
	if result.Allocations[0].InvoiceID != "inv-1" {
		t.Fatalf("expected first allocation to inv-1, got %s", result.Allocations[0].InvoiceID)
	}
	if result.Allocations[0].AllocatedAmt != 100000 {
		t.Fatalf("expected allocated 100000, got %d", result.Allocations[0].AllocatedAmt)
	}
	if result.Allocations[0].NewStatus != domain.InvoiceStatusLunas {
		t.Fatalf("expected status lunas, got %s", result.Allocations[0].NewStatus)
	}

	// Verifikasi alokasi kedua: inv-2 bayar_sebagian
	if result.Allocations[1].AllocatedAmt != 20000 {
		t.Fatalf("expected allocated 20000, got %d", result.Allocations[1].AllocatedAmt)
	}
	if result.Allocations[1].NewStatus != domain.InvoiceStatusBayarSebagian {
		t.Fatalf("expected status bayar_sebagian, got %s", result.Allocations[1].NewStatus)
	}

	if result.ExcessToCredit != 0 {
		t.Fatalf("expected excess 0, got %d", result.ExcessToCredit)
	}
}

// TestPaymentUsecase_RecordMultiPayment_InvoiceIDsOverride menguji alokasi
// dengan invoice_ids yang ditentukan secara eksplisit.
func TestPaymentUsecase_RecordMultiPayment_InvoiceIDsOverride(t *testing.T) {
	// Tes FIFO allocation dengan subset invoice
	invoices := []domain.FIFOInput{
		{InvoiceID: "inv-2", InvoiceNumber: "INV-002", TotalAmount: 150000, PaidAmount: 50000},
	}

	// Bayar 100000 ke inv-2 saja -> lunas (remaining = 100000)
	result := domain.AllocatePaymentFIFO(invoices, 100000)

	if len(result.Allocations) != 1 {
		t.Fatalf("expected 1 allocation, got %d", len(result.Allocations))
	}

	if result.Allocations[0].InvoiceID != "inv-2" {
		t.Fatalf("expected allocation to inv-2, got %s", result.Allocations[0].InvoiceID)
	}
	if result.Allocations[0].AllocatedAmt != 100000 {
		t.Fatalf("expected allocated 100000, got %d", result.Allocations[0].AllocatedAmt)
	}
	if result.Allocations[0].NewStatus != domain.InvoiceStatusLunas {
		t.Fatalf("expected status lunas, got %s", result.Allocations[0].NewStatus)
	}
}

// TestPaymentUsecase_RecordMultiPayment_ExcessToCredit menguji kelebihan bayar masuk ke kredit.
func TestPaymentUsecase_RecordMultiPayment_ExcessToCredit(t *testing.T) {
	invoices := []domain.FIFOInput{
		{InvoiceID: "inv-1", InvoiceNumber: "INV-001", TotalAmount: 100000, PaidAmount: 0},
	}

	// Bayar 150000 untuk invoice 100000 -> excess 50000
	result := domain.AllocatePaymentFIFO(invoices, 150000)

	if len(result.Allocations) != 1 {
		t.Fatalf("expected 1 allocation, got %d", len(result.Allocations))
	}

	if result.Allocations[0].AllocatedAmt != 100000 {
		t.Fatalf("expected allocated 100000, got %d", result.Allocations[0].AllocatedAmt)
	}

	if result.ExcessToCredit != 50000 {
		t.Fatalf("expected excess 50000, got %d", result.ExcessToCredit)
	}

	// Verifikasi invariant: TotalAllocated + ExcessToCredit == nominal
	if result.TotalAllocated+result.ExcessToCredit != 150000 {
		t.Fatalf("invariant violated: %d + %d != 150000", result.TotalAllocated, result.ExcessToCredit)
	}
}

// =============================================================================
// Tes: PayAll - multiple invoices dan no open invoices
// =============================================================================

// TestPaymentUsecase_PayAll_MultipleInvoices menguji PayAll menghitung total tunggakan.
// PayAll mendelegasikan ke RecordMultiPayment yang butuh pool, jadi kita verifikasi
// bahwa total tunggakan dihitung dengan benar melalui GetOpenInvoices.
func TestPaymentUsecase_PayAll_MultipleInvoices(t *testing.T) {
	s := setupPaymentUsecase()
	ctx := context.Background()

	// Setup 2 invoice terbuka
	s.invoiceRepo.invoices["inv-1"] = &domain.Invoice{
		ID: "inv-1", CustomerID: "cust-1", TenantID: "tenant-1",
		TotalAmount: 100000, PaidAmount: 0,
		Status:  domain.InvoiceStatusBelumBayar,
		DueDate: time.Now().Add(24 * time.Hour),
	}
	s.invoiceRepo.invoices["inv-2"] = &domain.Invoice{
		ID: "inv-2", CustomerID: "cust-1", TenantID: "tenant-1",
		TotalAmount: 150000, PaidAmount: 50000,
		Status:  domain.InvoiceStatusBayarSebagian,
		DueDate: time.Now().Add(48 * time.Hour),
	}

	// Verifikasi total tunggakan melalui GetOpenInvoices
	openResult, err := s.uc.GetOpenInvoices(ctx, "cust-1")
	if err != nil {
		t.Fatalf("GetOpenInvoices gagal: %v", err)
	}

	// Total tunggakan = (100000-0) + (150000-50000) = 200000
	expectedArrears := int64(200000)
	if openResult.TotalArrears != expectedArrears {
		t.Fatalf("expected total_arrears %d, got %d", expectedArrears, openResult.TotalArrears)
	}

	// Verifikasi FIFO allocation dengan total tunggakan
	fifoInputs := []domain.FIFOInput{
		{InvoiceID: "inv-1", InvoiceNumber: "INV-001", TotalAmount: 100000, PaidAmount: 0},
		{InvoiceID: "inv-2", InvoiceNumber: "INV-002", TotalAmount: 150000, PaidAmount: 50000},
	}
	result := domain.AllocatePaymentFIFO(fifoInputs, expectedArrears)

	// Semua invoice harus lunas
	for _, alloc := range result.Allocations {
		if alloc.NewStatus != domain.InvoiceStatusLunas {
			t.Fatalf("invoice %s: expected lunas, got %s", alloc.InvoiceID, alloc.NewStatus)
		}
	}
	if result.ExcessToCredit != 0 {
		t.Fatalf("expected excess 0, got %d", result.ExcessToCredit)
	}
}

func TestPaymentUsecase_PayAll_NoOpenInvoices(t *testing.T) {
	s := setupPaymentUsecase()
	ctx := context.Background()

	// Tidak ada invoice terbuka untuk cust-1
	req := domain.PayAllRequest{
		CustomerID:    "cust-1",
		PaymentMethod: "tunai",
		PaymentDate:   "2024-06-15",
	}
	actor := domain.ActorInfo{ActorID: "user-1", ActorName: "Test User"}

	_, err := s.uc.PayAll(ctx, req, actor)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, domain.ErrNoOpenInvoices) {
		t.Fatalf("expected ErrNoOpenInvoices, got %v", err)
	}
}

func TestPaymentUsecase_PayAll_OnlyLunasInvoices(t *testing.T) {
	s := setupPaymentUsecase()
	ctx := context.Background()

	// Invoice lunas tidak termasuk dalam FindOpenByCustomer
	s.invoiceRepo.invoices["inv-1"] = &domain.Invoice{
		ID: "inv-1", CustomerID: "cust-1",
		TotalAmount: 100000, PaidAmount: 100000,
		Status: domain.InvoiceStatusLunas,
	}

	req := domain.PayAllRequest{
		CustomerID:    "cust-1",
		PaymentMethod: "tunai",
		PaymentDate:   "2024-06-15",
	}
	actor := domain.ActorInfo{ActorID: "user-1", ActorName: "Test User"}

	_, err := s.uc.PayAll(ctx, req, actor)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, domain.ErrNoOpenInvoices) {
		t.Fatalf("expected ErrNoOpenInvoices, got %v", err)
	}
}
