// payment_bulk_test.go berisi unit test untuk PaymentUsecase - bulk import CSV.
package usecase

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// =============================================================================
// =============================================================================

// generateCSVHeader mengembalikan header CSV standar.
func generateCSVHeader() string {
	return "customer_id_seq,amount,payment_method,payment_date,reference_number,notes"
}

func generateCSVRows(n int) string {
	var sb strings.Builder
	sb.WriteString(generateCSVHeader() + "\n")
	for i := 0; i < n; i++ {
		sb.WriteString(fmt.Sprintf("PLG-%03d,100000,tunai,2024-06-15,REF-%d,Catatan %d\n", i+1, i+1, i+1))
	}
	return sb.String()
}

// =============================================================================
// =============================================================================

func TestPaymentUsecase_BulkImport_ValidCSV(t *testing.T) {
	s := setupPaymentUsecase()
	ctx := ctxWithTenant("tenant-1")

	csvContent := generateCSVHeader() + "\n" +
		"PLG-001,100000,tunai,2024-06-15,REF-001,Catatan\n" +
		"PLG-002,200000,transfer,2024-06-16,REF-002,\n"

	result, err := s.uc.BulkImport(ctx, []byte(csvContent), domain.ActorInfo{ActorID: "user-1", ActorName: "Admin"})
	if err != nil {
		t.Fatalf("BulkImport gagal: %v", err)
	}

	if result.TotalRows != 2 {
		t.Fatalf("expected 2 total rows, got %d", result.TotalRows)
	}

	if result.FailureCount != 2 {
		t.Fatalf("expected 2 failures (customer not found), got %d", result.FailureCount)
	}
}

// TestPaymentUsecase_BulkImport_ValidCSVWithCustomer menguji import CSV dengan customer yang ada.
// Karena pool nil, RecordMultiPayment akan panic, jadi test ini di-skip.
func TestPaymentUsecase_BulkImport_ValidCSVWithCustomer(t *testing.T) {
	// RecordMultiPayment membutuhkan pool yang tidak nil untuk transaksi DB.
	// processBulkRow memanggil RecordMultiPayment yang akan panic pada pool.Begin(nil).
	t.Skip("Skipped: RecordMultiPayment membutuhkan pool yang tidak nil untuk transaksi DB")
}

// =============================================================================
// Tes: BulkImport - melebihi 500 baris
// =============================================================================

func TestPaymentUsecase_BulkImport_ExceedsMaxRows(t *testing.T) {
	s := setupPaymentUsecase()
	ctx := ctxWithTenant("tenant-1")

	// Buat 501 baris
	csvContent := generateCSVRows(501)

	_, err := s.uc.BulkImport(ctx, []byte(csvContent), domain.ActorInfo{ActorID: "user-1", ActorName: "Admin"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, domain.ErrCSVTooLarge) {
		t.Fatalf("expected ErrCSVTooLarge, got %v", err)
	}
}

// TestPaymentUsecase_BulkImport_Exactly500Rows menguji bahwa 500 baris tepat diterima.
func TestPaymentUsecase_BulkImport_Exactly500Rows(t *testing.T) {
	s := setupPaymentUsecase()
	ctx := ctxWithTenant("tenant-1")

	// Buat tepat 500 baris
	csvContent := generateCSVRows(500)

	result, err := s.uc.BulkImport(ctx, []byte(csvContent), domain.ActorInfo{ActorID: "user-1", ActorName: "Admin"})
	if err != nil {
		t.Fatalf("BulkImport gagal untuk 500 baris: %v", err)
	}

	if result.TotalRows != 500 {
		t.Fatalf("expected 500 total rows, got %d", result.TotalRows)
	}
}

// =============================================================================
// =============================================================================

func TestPaymentUsecase_BulkImport_ValidationErrors(t *testing.T) {
	s := setupPaymentUsecase()
	ctx := ctxWithTenant("tenant-1")

	csvContent := generateCSVHeader() + "\n" +
		"PLG-001,-100,tunai,2024-06-15,,\n" + // nominal negatif
		"PLG-002,100000,bitcoin,2024-06-15,,\n" + // method tidak valid
		"PLG-003,100000,tunai,15-06-2024,,\n" + // format tanggal salah
		"PLG-004,abc,tunai,2024-06-15,,\n" // nominal bukan angka

	result, err := s.uc.BulkImport(ctx, []byte(csvContent), domain.ActorInfo{ActorID: "user-1", ActorName: "Admin"})
	if err != nil {
		t.Fatalf("BulkImport gagal: %v", err)
	}

	if result.TotalRows != 4 {
		t.Fatalf("expected 4 total rows, got %d", result.TotalRows)
	}

	// Semua baris harus gagal validasi
	if result.FailureCount != 4 {
		t.Fatalf("expected 4 failures, got %d", result.FailureCount)
	}

	for _, r := range result.Results {
		if r.Status != "failed" {
			t.Fatalf("row %d: expected status failed, got %s", r.Row, r.Status)
		}
		if r.Reason == "" {
			t.Fatalf("row %d: expected reason, got empty", r.Row)
		}
	}
}

func TestPaymentUsecase_BulkImport_TooFewColumns(t *testing.T) {
	s := setupPaymentUsecase()
	ctx := ctxWithTenant("tenant-1")

	csvContent := generateCSVHeader() + "\n" +
		"PLG-001,100000,tunai\n"

	_, err := s.uc.BulkImport(ctx, []byte(csvContent), domain.ActorInfo{ActorID: "user-1", ActorName: "Admin"})
	if err == nil {
		t.Fatal("expected error dari CSV reader, got nil")
	}

	if !strings.Contains(err.Error(), "wrong number of fields") && !strings.Contains(err.Error(), "gagal membaca baris CSV") {
		t.Fatalf("expected CSV parse error, got: %v", err)
	}
}

// =============================================================================
// Tes: BulkImport - duplikasi terdeteksi (skip)
// =============================================================================

// TestPaymentUsecase_BulkImport_DuplicateDetection menguji deteksi duplikasi.
func TestPaymentUsecase_BulkImport_DuplicateDetection(t *testing.T) {
	s := setupPaymentUsecase()
	ctx := ctxWithTenant("tenant-1")

	// dan FindDuplicate mengembalikan true (duplikat terdeteksi)
	s.customerRepo.searchResult = []*domain.Customer{
		{ID: "cust-1", CustomerIDSeq: "PLG-001", Name: "Ahmad", Status: domain.CustomerStatusAktif},
	}
	s.paymentRepo.findDupResult = true // semua pembayaran dianggap duplikat

	csvContent := generateCSVHeader() + "\n" +
		"PLG-001,100000,tunai,2024-06-15,REF-001,\n"

	result, err := s.uc.BulkImport(ctx, []byte(csvContent), domain.ActorInfo{ActorID: "user-1", ActorName: "Admin"})
	if err != nil {
		t.Fatalf("BulkImport gagal: %v", err)
	}

	if result.DuplicatesSkipped != 1 {
		t.Fatalf("expected 1 duplicate skipped, got %d", result.DuplicatesSkipped)
	}

	if result.Results[0].Status != "skipped" {
		t.Fatalf("expected status skipped, got %s", result.Results[0].Status)
	}
}

// =============================================================================
// Tes: BulkImport - CSV kosong (hanya header)
// =============================================================================

// TestPaymentUsecase_BulkImport_EmptyCSV menguji import CSV kosong (hanya header).
func TestPaymentUsecase_BulkImport_EmptyCSV(t *testing.T) {
	s := setupPaymentUsecase()
	ctx := ctxWithTenant("tenant-1")

	csvContent := generateCSVHeader() + "\n"

	result, err := s.uc.BulkImport(ctx, []byte(csvContent), domain.ActorInfo{ActorID: "user-1", ActorName: "Admin"})
	if err != nil {
		t.Fatalf("BulkImport gagal: %v", err)
	}

	if result.TotalRows != 0 {
		t.Fatalf("expected 0 total rows, got %d", result.TotalRows)
	}
}
