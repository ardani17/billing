// payment_proof_test.go berisi unit test untuk PaymentUsecase — upload dan pengambilan bukti transfer.
package usecase

import (
	"errors"
	"testing"
	"time"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// =============================================================================
// Test: UploadProof — file terlalu besar
// =============================================================================

// TestPaymentUsecase_UploadProof_FileTooLarge menguji error saat file > 5MB.
func TestPaymentUsecase_UploadProof_FileTooLarge(t *testing.T) {
	s := setupPaymentUsecase()
	ctx := ctxWithTenant("tenant-1")

	// Tambahkan payment yang valid
	s.paymentRepo.payments["pay-1"] = &domain.InvoicePayment{
		ID: "pay-1", TenantID: "tenant-1", InvoiceID: "inv-1",
		Amount: 100000, CreatedAt: time.Now(),
	}

	// Buat file data > 5MB (5*1024*1024 + 1 bytes)
	largeFile := make([]byte, 5*1024*1024+1)
	// Set magic bytes JPEG agar lolos validasi format
	largeFile[0] = 0xFF
	largeFile[1] = 0xD8
	largeFile[2] = 0xFF

	_, err := s.uc.UploadProof(ctx, "pay-1", largeFile, "proof.jpg")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, domain.ErrFileTooLarge) {
		t.Fatalf("expected ErrFileTooLarge, got %v", err)
	}
}

// =============================================================================
// Test: UploadProof — format file tidak valid
// =============================================================================

// TestPaymentUsecase_UploadProof_InvalidFormat menguji error saat format file bukan JPEG/PNG/WebP.
func TestPaymentUsecase_UploadProof_InvalidFormat(t *testing.T) {
	s := setupPaymentUsecase()
	ctx := ctxWithTenant("tenant-1")

	// Tambahkan payment yang valid
	s.paymentRepo.payments["pay-1"] = &domain.InvoicePayment{
		ID: "pay-1", TenantID: "tenant-1", InvoiceID: "inv-1",
		Amount: 100000, CreatedAt: time.Now(),
	}

	// PDF magic bytes: %PDF (0x25 0x50 0x44 0x46)
	pdfData := []byte{0x25, 0x50, 0x44, 0x46, 0x2D, 0x31, 0x2E, 0x34}

	_, err := s.uc.UploadProof(ctx, "pay-1", pdfData, "proof.pdf")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, domain.ErrInvalidFileFormat) {
		t.Fatalf("expected ErrInvalidFileFormat, got %v", err)
	}
}

// TestPaymentUsecase_UploadProof_EmptyFile menguji error saat file kosong.
func TestPaymentUsecase_UploadProof_EmptyFile(t *testing.T) {
	s := setupPaymentUsecase()
	ctx := ctxWithTenant("tenant-1")

	s.paymentRepo.payments["pay-1"] = &domain.InvoicePayment{
		ID: "pay-1", TenantID: "tenant-1", InvoiceID: "inv-1",
		Amount: 100000, CreatedAt: time.Now(),
	}

	// File kosong → magic bytes tidak valid
	_, err := s.uc.UploadProof(ctx, "pay-1", []byte{}, "proof.jpg")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, domain.ErrInvalidFileFormat) {
		t.Fatalf("expected ErrInvalidFileFormat, got %v", err)
	}
}

// =============================================================================
// Test: UploadProof — payment tidak ditemukan
// =============================================================================

// TestPaymentUsecase_UploadProof_PaymentNotFound menguji error saat payment tidak ada.
func TestPaymentUsecase_UploadProof_PaymentNotFound(t *testing.T) {
	s := setupPaymentUsecase()
	ctx := ctxWithTenant("tenant-1")

	// JPEG magic bytes
	jpegData := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10}

	_, err := s.uc.UploadProof(ctx, "nonexistent", jpegData, "proof.jpg")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, domain.ErrPaymentNotFound) {
		t.Fatalf("expected ErrPaymentNotFound, got %v", err)
	}
}

// =============================================================================
// Test: GetProof — payment tidak ditemukan
// =============================================================================

// TestPaymentUsecase_GetProof_PaymentNotFound menguji error saat payment tidak ada.
func TestPaymentUsecase_GetProof_PaymentNotFound(t *testing.T) {
	s := setupPaymentUsecase()
	ctx := ctxWithTenant("tenant-1")

	_, _, err := s.uc.GetProof(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, domain.ErrPaymentNotFound) {
		t.Fatalf("expected ErrPaymentNotFound, got %v", err)
	}
}

// TestPaymentUsecase_GetProof_NoProofURL menguji error saat proof_image_url kosong.
func TestPaymentUsecase_GetProof_NoProofURL(t *testing.T) {
	s := setupPaymentUsecase()
	ctx := ctxWithTenant("tenant-1")

	// Payment ada tapi tanpa proof_image_url
	s.paymentRepo.payments["pay-1"] = &domain.InvoicePayment{
		ID: "pay-1", TenantID: "tenant-1", InvoiceID: "inv-1",
		ProofImageURL: "", // kosong
	}

	_, _, err := s.uc.GetProof(ctx, "pay-1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, domain.ErrProofNotFound) {
		t.Fatalf("expected ErrProofNotFound, got %v", err)
	}
}
