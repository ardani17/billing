package domain

import "fmt"

// =============================================================================
// Invoice Number Formatting - fungsi pemformatan nomor invoice, credit note, debit note
// =============================================================================

// FormatInvoiceNumber memformat nomor invoice dari komponen.
// Format: {prefix}-{YYYY}-{MM}-{SEQ} dengan SEQ zero-padded minimal 3 digit.
// Contoh: INV-2026-04-001, INV-2026-04-1000
func FormatInvoiceNumber(prefix string, year, month, seq int) string {
	return fmt.Sprintf("%s-%04d-%02d-%s", prefix, year, month, zeroPadSeq(seq))
}

// FormatCreditNoteNumber memformat nomor credit note.
// Format: CN-{YYYY}-{MM}-{SEQ} dengan SEQ zero-padded minimal 3 digit.
func FormatCreditNoteNumber(year, month, seq int) string {
	return FormatInvoiceNumber("CN", year, month, seq)
}

// FormatDebitNoteNumber memformat nomor debit note.
// Format: DN-{YYYY}-{MM}-{SEQ} dengan SEQ zero-padded minimal 3 digit.
func FormatDebitNoteNumber(year, month, seq int) string {
	return FormatInvoiceNumber("DN", year, month, seq)
}

// zeroPadSeq memformat sequence number dengan zero-padding minimal 3 digit.
// Jika seq >= 1000, digit asli dipertahankan tanpa padding tambahan.
func zeroPadSeq(seq int) string {
	if seq < 1000 {
		return fmt.Sprintf("%03d", seq)
	}
	return fmt.Sprintf("%d", seq)
}
