// payment_bulk.go berisi business logic untuk bulk import pembayaran dari CSV.
package usecase

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/ispboss/ispboss/pkg/tenant"
	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// BulkImport mengimpor pembayaran dari data CSV.
// Batas: maksimal 500 baris. Setiap baris diproses independen.
func (uc *PaymentUsecase) BulkImport(ctx context.Context, csvData []byte, actor domain.ActorInfo) (*domain.BulkImportResponse, error) {
	tenantID := tenant.FromContext(ctx)

	// Parsing CSV
	reader := csv.NewReader(bytes.NewReader(csvData))
	reader.TrimLeadingSpace = true

	// Baca header (skip)
	if _, err := reader.Read(); err != nil {
		return nil, fmt.Errorf("gagal membaca header CSV: %w", err)
	}

	// Baca semua baris
	var rows [][]string
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("gagal membaca baris CSV: %w", err)
		}
		rows = append(rows, record)
	}

	// Validasi jumlah baris
	if len(rows) > 500 {
		return nil, domain.ErrCSVTooLarge
	}

	var results []domain.BulkImportResult
	var successCount, failureCount, duplicatesSkipped int

	for i, row := range rows {
		rowNum := i + 2 // +2 karena header di baris 1, data mulai baris 2

		result := uc.processBulkRow(ctx, tenantID, row, rowNum, actor)
		results = append(results, result)

		switch result.Status {
		case "success":
			successCount++
		case "skipped":
			duplicatesSkipped++
		case "failed":
			failureCount++
		}
	}

	return &domain.BulkImportResponse{
		TotalRows:         len(rows),
		SuccessCount:      successCount,
		FailureCount:      failureCount,
		DuplicatesSkipped: duplicatesSkipped,
		Results:           results,
	}, nil
}

// processBulkRow memproses satu baris CSV untuk bulk import.
func (uc *PaymentUsecase) processBulkRow(ctx context.Context, tenantID string, row []string, rowNum int, actor domain.ActorInfo) domain.BulkImportResult {
	// Validasi jumlah kolom minimal
	if len(row) < 4 {
		return domain.BulkImportResult{
			Row:    rowNum,
			Status: "failed",
			Reason: "jumlah kolom kurang dari 4 (customer_id_seq, amount, payment_method, payment_date)",
		}
	}

	customerIDSeq := strings.TrimSpace(row[0])
	amountStr := strings.TrimSpace(row[1])
	paymentMethod := strings.TrimSpace(row[2])
	paymentDateStr := strings.TrimSpace(row[3])

	referenceNumber := ""
	if len(row) > 4 {
		referenceNumber = strings.TrimSpace(row[4])
	}
	notes := ""
	if len(row) > 5 {
		notes = strings.TrimSpace(row[5])
	}

	// Validasi nominal
	amount, err := strconv.ParseInt(amountStr, 10, 64)
	if err != nil || amount <= 0 {
		return domain.BulkImportResult{
			Row:    rowNum,
			Status: "failed",
			Reason: "amount tidak valid, harus angka positif",
		}
	}

	// Validasi payment_method
	validMethods := map[string]bool{"tunai": true, "transfer": true, "lainnya": true}
	if !validMethods[paymentMethod] {
		return domain.BulkImportResult{
			Row:    rowNum,
			Status: "failed",
			Reason: "payment_method tidak valid, harus tunai/transfer/lainnya",
		}
	}

	// Validasi payment_date
	paymentDate, err := time.Parse("2006-01-02", paymentDateStr)
	if err != nil {
		return domain.BulkImportResult{
			Row:    rowNum,
			Status: "failed",
			Reason: "format payment_date tidak valid, gunakan YYYY-MM-DD",
		}
	}

	// Cari customer berdasarkan customer_id_seq
	customers, err := uc.customerRepo.SearchForPayment(ctx, tenantID, customerIDSeq)
	if err != nil || len(customers) == 0 {
		return domain.BulkImportResult{
			Row:    rowNum,
			Status: "failed",
			Reason: fmt.Sprintf("pelanggan %s tidak ditemukan", customerIDSeq),
		}
	}

	// Ambil customer pertama yang cocok persis
	var customer *domain.Customer
	for _, c := range customers {
		if c.CustomerIDSeq == customerIDSeq {
			customer = c
			break
		}
	}
	if customer == nil {
		return domain.BulkImportResult{
			Row:    rowNum,
			Status: "failed",
			Reason: fmt.Sprintf("pelanggan %s tidak ditemukan", customerIDSeq),
		}
	}

	// Cek duplikasi dalam 24 jam terakhir
	isDuplicate, err := uc.paymentRepo.FindDuplicate(ctx, customer.ID, amount, paymentMethod, paymentDate)
	if err == nil && isDuplicate {
		return domain.BulkImportResult{
			Row:    rowNum,
			Status: "skipped",
			Reason: "duplikasi terdeteksi dalam 24 jam terakhir",
		}
	}

	// Proses pembayaran menggunakan FIFO allocation
	multiReq := domain.MultiPaymentRequest{
		CustomerID:      customer.ID,
		Amount:          amount,
		PaymentMethod:   paymentMethod,
		PaymentDate:     paymentDateStr,
		ReferenceNumber: referenceNumber,
		Notes:           notes,
	}

	resp, err := uc.RecordMultiPayment(ctx, multiReq, actor)
	if err != nil {
		return domain.BulkImportResult{
			Row:    rowNum,
			Status: "failed",
			Reason: err.Error(),
		}
	}

	return domain.BulkImportResult{
		Row:           rowNum,
		Status:        "success",
		ReceiptNumber: resp.ReceiptNumber,
	}
}
