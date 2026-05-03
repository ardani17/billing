package domain

import (
	"fmt"
	"strconv"
	"strings"
)

// =============================================================================
// Receipt Number Formatting — fungsi pemformatan nomor kwitansi pembayaran
// =============================================================================

// FormatReceiptNumber memformat nomor kwitansi dari komponen.
// Format: PAY-{YYYY}-{MM}-{SEQ} dengan SEQ zero-padded minimal 4 digit.
// Contoh: PAY-2026-04-0001, PAY-2026-04-10000
func FormatReceiptNumber(year, month, seq int) string {
	return fmt.Sprintf("PAY-%04d-%02d-%s", year, month, zeroPadReceiptSeq(seq))
}

// zeroPadReceiptSeq memformat sequence number dengan zero-padding minimal 4 digit.
// Jika seq >= 10000, digit asli dipertahankan tanpa padding tambahan.
func zeroPadReceiptSeq(seq int) string {
	if seq < 10000 {
		return fmt.Sprintf("%04d", seq)
	}
	return fmt.Sprintf("%d", seq)
}

// ParseReceiptNumber mem-parse string nomor kwitansi kembali ke komponen.
// Mengembalikan year, month, seq, error.
// Format yang diharapkan: PAY-{YYYY}-{MM}-{SEQ}
func ParseReceiptNumber(receiptNumber string) (int, int, int, error) {
	parts := strings.SplitN(receiptNumber, "-", 4)
	if len(parts) != 4 {
		return 0, 0, 0, fmt.Errorf("format nomor kwitansi tidak valid: %q, diharapkan 4 bagian dipisahkan '-'", receiptNumber)
	}

	if parts[0] != "PAY" {
		return 0, 0, 0, fmt.Errorf("prefix nomor kwitansi tidak valid: %q, diharapkan 'PAY'", parts[0])
	}

	year, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("tahun tidak valid dalam nomor kwitansi %q: %w", receiptNumber, err)
	}

	month, err := strconv.Atoi(parts[2])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("bulan tidak valid dalam nomor kwitansi %q: %w", receiptNumber, err)
	}

	seq, err := strconv.Atoi(parts[3])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("sequence tidak valid dalam nomor kwitansi %q: %w", receiptNumber, err)
	}

	return year, month, seq, nil
}

// =============================================================================
// Payment Event Payloads — payload event pembayaran
// =============================================================================

// PaymentRecordedPayload adalah payload event payment.recorded.
// Dikirim saat pembayaran berhasil dicatat (single, multi, atau pay-all).
type PaymentRecordedPayload struct {
	TenantID      string `json:"tenant_id"`
	CustomerID    string `json:"customer_id"`
	ReceiptNumber string `json:"receipt_number"`
	TotalAmount   int64  `json:"total_amount"`
	PaymentMethod string `json:"payment_method"`
	InvoiceCount  int    `json:"invoice_count"`
}

// PaymentVoidedReIsolirPayload adalah payload event payment.voided.re_isolir.
// Dikirim saat void pembayaran menyebabkan invoice kembali ke status terlambat
// dan pelanggan perlu di-isolir ulang oleh modul isolir.
type PaymentVoidedReIsolirPayload struct {
	TenantID   string `json:"tenant_id"`
	CustomerID string `json:"customer_id"`
	InvoiceID  string `json:"invoice_id"`
	Reason     string `json:"reason"`
}
