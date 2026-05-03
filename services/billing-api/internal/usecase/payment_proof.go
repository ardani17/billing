// payment_proof.go berisi business logic untuk upload dan pengambilan bukti transfer.
package usecase

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/ispboss/ispboss/pkg/tenant"
	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// maxProofFileSize adalah batas ukuran file bukti transfer (5 MB).
const maxProofFileSize = 5 * 1024 * 1024

// UploadProof mengunggah bukti transfer untuk pembayaran.
// Validasi: payment ada, ukuran file <= 5MB, format JPEG/PNG/WebP.
// Menyimpan file ke filesystem lokal dan update proof_image_url.
func (uc *PaymentUsecase) UploadProof(ctx context.Context, paymentID string, fileData []byte, filename string) (string, error) {
	tenantID := tenant.FromContext(ctx)

	// Validasi payment ada
	payment, err := uc.paymentRepo.GetByID(ctx, paymentID)
	if err != nil {
		return "", domain.ErrPaymentNotFound
	}

	// Validasi ukuran file
	if len(fileData) > maxProofFileSize {
		return "", domain.ErrFileTooLarge
	}

	// Validasi format file berdasarkan magic bytes
	if !isValidImageFormat(fileData) {
		return "", domain.ErrInvalidFileFormat
	}

	// Generate path penyimpanan: uploads/payments/{tenant_id}/{payment_id}/{filename}
	storagePath := filepath.Join("uploads", "payments", tenantID, paymentID, filename)

	// Buat direktori jika belum ada
	dir := filepath.Dir(storagePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("gagal membuat direktori: %w", err)
	}

	// Tulis file
	if err := os.WriteFile(storagePath, fileData, 0o644); err != nil {
		return "", fmt.Errorf("gagal menyimpan file: %w", err)
	}

	// Update proof_image_url di database
	// Gunakan pool langsung karena VoidPayment sudah menggunakan paymentRepo.VoidPayment
	_, err = uc.pool.Exec(ctx,
		`UPDATE invoice_payments SET proof_image_url = $1 WHERE id = $2`,
		storagePath, payment.ID,
	)
	if err != nil {
		return "", fmt.Errorf("gagal update proof_image_url: %w", err)
	}

	return storagePath, nil
}

// GetProof mengambil file bukti transfer untuk pembayaran.
// Mengembalikan data file dan content type.
func (uc *PaymentUsecase) GetProof(ctx context.Context, paymentID string) ([]byte, string, error) {
	// Ambil payment
	payment, err := uc.paymentRepo.GetByID(ctx, paymentID)
	if err != nil {
		return nil, "", domain.ErrPaymentNotFound
	}

	// Cek proof_image_url ada
	if payment.ProofImageURL == "" {
		return nil, "", domain.ErrProofNotFound
	}

	// Baca file dari storage
	data, err := os.ReadFile(payment.ProofImageURL)
	if err != nil {
		return nil, "", domain.ErrProofNotFound
	}

	// Deteksi content type
	contentType := http.DetectContentType(data)

	return data, contentType, nil
}

// isValidImageFormat memeriksa apakah data file adalah JPEG, PNG, atau WebP
// berdasarkan magic bytes.
func isValidImageFormat(data []byte) bool {
	if len(data) < 4 {
		return false
	}

	// JPEG: FF D8 FF
	if data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
		return true
	}

	// PNG: 89 50 4E 47
	if data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 {
		return true
	}

	// WebP: RIFF....WEBP (bytes 0-3 = "RIFF", bytes 8-11 = "WEBP")
	if len(data) >= 12 &&
		data[0] == 'R' && data[1] == 'I' && data[2] == 'F' && data[3] == 'F' &&
		data[8] == 'W' && data[9] == 'E' && data[10] == 'B' && data[11] == 'P' {
		return true
	}

	return false
}
