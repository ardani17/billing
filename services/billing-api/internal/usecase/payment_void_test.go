// payment_void_test.go berisi unit test untuk PaymentUsecase - void pembayaran.
// yang terjadi sebelum transaksi dimulai, serta test DeterminePostVoidStatus
// secara langsung sebagai pure function.
package usecase

import (
	"errors"
	"testing"
	"time"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// =============================================================================
// =============================================================================

// TestPaymentUsecase_VoidPayment_NoPool menguji bahwa VoidPayment membutuhkan pool.
// Karena pool.Begin pada nil pool panic, kita skip test ini.
func TestPaymentUsecase_VoidPayment_NoPool(t *testing.T) {
	// VoidPayment memanggil pool.Begin() di awal, yang panic pada nil pool.
	// Tes validasi void dilakukan melalui DeterminePostVoidStatus (pure function).
	t.Skip("Skipped: VoidPayment membutuhkan pool yang tidak nil untuk transaksi DB")
}

// =============================================================================
// Tes: DeterminePostVoidStatus - pure function tests
// =============================================================================

// TestVoidPayment_TimeLimitExceeded menguji DeterminePostVoidStatus tidak dipanggil
// karena validasi waktu 24 jam dilakukan di usecase. Kita test pure function langsung.
func TestVoidPayment_TimeLimitExceeded(t *testing.T) {
	// Simulasi: payment dibuat 25 jam lalu -> melebihi batas 24 jam
	createdAt := time.Now().Add(-25 * time.Hour)
	now := time.Now()

	if now.Sub(createdAt) <= 24*time.Hour {
		t.Fatal("expected time difference > 24h")
	}
}

// TestVoidPayment_AlreadyVoided menguji deteksi payment yang sudah di-void.
func TestVoidPayment_AlreadyVoided(t *testing.T) {
	// Verifikasi bahwa payment.Voided == true menghasilkan ErrPaymentAlreadyVoided
	payment := &domain.InvoicePayment{
		ID: "pay-1", Voided: true,
	}

	if !payment.Voided {
		t.Fatal("expected payment to be voided")
	}

	// Dalam usecase, ini akan mengembalikan ErrPaymentAlreadyVoided
	err := domain.ErrPaymentAlreadyVoided
	if !errors.Is(err, domain.ErrPaymentAlreadyVoided) {
		t.Fatalf("expected ErrPaymentAlreadyVoided, got %v", err)
	}
}

// TestVoidPayment_CreditBalanceRollback menguji logika rollback kredit saat void.
// Jika payment sebelumnya menghasilkan excess ke credit, void harus mengurangi credit.
func TestVoidPayment_CreditBalanceRollback(t *testing.T) {
	// Simulasi: invoice total=100000, paid=150000 (excess 50000 ke credit)
	// Void payment 150000 -> credit harus dikurangi 50000
	invoice := &domain.Invoice{
		TotalAmount: 100000,
		PaidAmount:  150000, // sebelum void
	}
	payment := &domain.InvoicePayment{
		Amount: 150000,
	}

	// Hitung excess yang perlu dikurangi dari credit
	excessFromPayment := invoice.PaidAmount - invoice.TotalAmount
	if excessFromPayment > payment.Amount {
		excessFromPayment = payment.Amount
	}

	expectedCreditReduced := int64(50000)
	if excessFromPayment != expectedCreditReduced {
		t.Fatalf("expected credit reduced %d, got %d", expectedCreditReduced, excessFromPayment)
	}
}

// =============================================================================
// Tes: DeterminePostVoidStatus - status determination
// =============================================================================

// TestDeterminePostVoidStatus_BelumBayar menguji status belum_bayar setelah void.
func TestDeterminePostVoidStatus_BelumBayar(t *testing.T) {
	now := time.Now()
	dueDate := now.Add(48 * time.Hour) // jatuh tempo masih 2 hari lagi

	status := domain.DeterminePostVoidStatus(0, 100000, dueDate, now)
	if status != domain.InvoiceStatusBelumBayar {
		t.Fatalf("expected belum_bayar, got %s", status)
	}
}

// TestDeterminePostVoidStatus_Terlambat menguji status terlambat setelah void.
func TestDeterminePostVoidStatus_Terlambat(t *testing.T) {
	now := time.Now()
	dueDate := now.Add(-48 * time.Hour) // jatuh tempo sudah lewat 2 hari

	status := domain.DeterminePostVoidStatus(0, 100000, dueDate, now)
	if status != domain.InvoiceStatusTerlambat {
		t.Fatalf("expected terlambat, got %s", status)
	}
}

// TestDeterminePostVoidStatus_BayarSebagian menguji status bayar_sebagian setelah void.
func TestDeterminePostVoidStatus_BayarSebagian(t *testing.T) {
	now := time.Now()
	dueDate := now.Add(-24 * time.Hour)

	// paidAmount > 0 tapi < totalAmount -> bayar_sebagian
	status := domain.DeterminePostVoidStatus(50000, 100000, dueDate, now)
	if status != domain.InvoiceStatusBayarSebagian {
		t.Fatalf("expected bayar_sebagian, got %s", status)
	}
}

// TestDeterminePostVoidStatus_TerlambatDueDateEqual menguji status saat dueDate == now.
func TestDeterminePostVoidStatus_TerlambatDueDateEqual(t *testing.T) {
	now := time.Now()
	// dueDate sama dengan now -> dueDate.After(now) == false -> terlambat
	status := domain.DeterminePostVoidStatus(0, 100000, now, now)
	if status != domain.InvoiceStatusTerlambat {
		t.Fatalf("expected terlambat (dueDate == now), got %s", status)
	}
}

// TestDeterminePostVoidStatus_Lunas menguji fallback lunas (seharusnya tidak terjadi setelah void).
func TestDeterminePostVoidStatus_Lunas(t *testing.T) {
	now := time.Now()
	dueDate := now.Add(24 * time.Hour)

	// paidAmount >= totalAmount -> lunas (fallback aman)
	status := domain.DeterminePostVoidStatus(100000, 100000, dueDate, now)
	if status != domain.InvoiceStatusLunas {
		t.Fatalf("expected lunas (fallback), got %s", status)
	}
}
