package domain

import (
	"errors"
	"time"
)

// =============================================================================
// Tipe Domain - Alokasi Pembayaran FIFO
// =============================================================================

// PaymentAllocation merepresentasikan satu alokasi pembayaran ke invoice.
type PaymentAllocation struct {
	InvoiceID     string        `json:"invoice_id"`
	InvoiceNumber string        `json:"invoice_number"`
	AllocatedAmt  int64         `json:"allocated_amount"`
	NewPaidAmount int64         `json:"new_paid_amount"`
	NewStatus     InvoiceStatus `json:"new_status"`
}

// FIFOInput merepresentasikan invoice yang memenuhi syarat untuk alokasi FIFO.
// Invoice harus sudah diurutkan berdasarkan due_date ASC (terlama dulu).
type FIFOInput struct {
	InvoiceID     string
	InvoiceNumber string
	TotalAmount   int64
	PaidAmount    int64
	Status        InvoiceStatus
}

// FIFOResult berisi hasil alokasi FIFO.
type FIFOResult struct {
	Allocations    []PaymentAllocation
	TotalAllocated int64
	ExcessToCredit int64
}

// =============================================================================
// AllocatePaymentFIFO - fungsi murni alokasi pembayaran FIFO
// =============================================================================

// AllocatePaymentFIFO mendistribusikan jumlah pembayaran ke invoice secara FIFO.
// Invoice harus sudah diurutkan berdasarkan due_date ascending (terlama dulu).
// Mengembalikan alokasi per invoice dan sisa yang masuk ke saldo kredit.
//
// Invarian: TotalAllocated + ExcessToCredit == nominal
// Invarian: Untuk setiap alokasi, AllocatedAmt <= (TotalAmount - PaidAmount)
// Invarian: Jika AllocatedAmt == (TotalAmount - PaidAmount), NewStatus == lunas
// Invarian: Jika 0 < AllocatedAmt < (TotalAmount - PaidAmount), NewStatus == bayar_sebagian
func AllocatePaymentFIFO(invoices []FIFOInput, amount int64) FIFOResult {
	remaining := amount
	var allocations []PaymentAllocation
	var totalAllocated int64

	for _, inv := range invoices {
		if remaining <= 0 {
			break
		}

		outstanding := inv.TotalAmount - inv.PaidAmount
		if outstanding <= 0 {
			continue
		}

		// Alokasikan minimum dari sisa pembayaran dan sisa tagihan
		alloc := remaining
		if alloc > outstanding {
			alloc = outstanding
		}

		newPaidAmount := inv.PaidAmount + alloc

		var newStatus InvoiceStatus
		if newPaidAmount >= inv.TotalAmount {
			newStatus = InvoiceStatusLunas
		} else {
			newStatus = InvoiceStatusBayarSebagian
		}

		allocations = append(allocations, PaymentAllocation{
			InvoiceID:     inv.InvoiceID,
			InvoiceNumber: inv.InvoiceNumber,
			AllocatedAmt:  alloc,
			NewPaidAmount: newPaidAmount,
			NewStatus:     newStatus,
		})

		totalAllocated += alloc
		remaining -= alloc
	}

	return FIFOResult{
		Allocations:    allocations,
		TotalAllocated: totalAllocated,
		ExcessToCredit: amount - totalAllocated,
	}
}

// =============================================================================
// DeterminePostVoidStatus - menentukan status invoice setelah void pembayaran
// =============================================================================

// DeterminePostVoidStatus menentukan status invoice setelah pembayaran di-void.
//   - Jika paidAmount == 0 dan dueDate setelah now -> belum_bayar
//   - Jika paidAmount == 0 dan dueDate sebelum/sama dengan now -> terlambat
//   - Jika 0 < paidAmount < totalAmount -> bayar_sebagian
func DeterminePostVoidStatus(paidAmount, totalAmount int64, dueDate time.Time, now time.Time) InvoiceStatus {
	if paidAmount == 0 {
		if dueDate.After(now) {
			return InvoiceStatusBelumBayar
		}
		return InvoiceStatusTerlambat
	}

	if paidAmount > 0 && paidAmount < totalAmount {
		return InvoiceStatusBayarSebagian
	}

	// Jika paidAmount >= totalAmount, seharusnya tidak terjadi setelah void,
	// tapi kembalikan lunas sebagai cadangan aman.
	return InvoiceStatusLunas
}

// =============================================================================
// Error Domain - error khusus domain pembayaran
// =============================================================================

var (
	// ErrPaymentNotFound dikembalikan saat pembayaran tidak ditemukan
	ErrPaymentNotFound = errors.New("pembayaran tidak ditemukan")

	// ErrPaymentAlreadyVoided dikembalikan saat pembayaran sudah di-void
	ErrPaymentAlreadyVoided = errors.New("pembayaran sudah di-void")

	// ErrVoidTimeLimitExceeded dikembalikan saat batas waktu void 24 jam terlampaui
	ErrVoidTimeLimitExceeded = errors.New("batas waktu void 24 jam terlampaui")

	// ErrNoOpenInvoices dikembalikan saat tidak ada invoice terbuka
	ErrNoOpenInvoices = errors.New("tidak ada invoice terbuka")

	// ErrInvalidInvoiceSelection dikembalikan saat pilihan invoice tidak valid
	ErrInvalidInvoiceSelection = errors.New("pilihan invoice tidak valid")

	// ErrPencarianTermTooShort dikembalikan saat kata pencarian kurang dari 2 karakter
	ErrSearchTermTooShort = errors.New("kata pencarian minimal 2 karakter")

	// ErrCSVTooLarge dikembalikan saat file CSV melebihi batas 500 baris
	ErrCSVTooLarge = errors.New("file CSV melebihi batas 500 baris")

	// ErrConcurrentModification dikembalikan saat terjadi konflik modifikasi bersamaan
	ErrConcurrentModification = errors.New("konflik modifikasi bersamaan")

	// ErrFileTooLarge dikembalikan saat file melebihi batas ukuran
	ErrFileTooLarge = errors.New("file melebihi batas 5 MB")

	// ErrInvalidFileFormat dikembalikan saat format file tidak valid
	ErrInvalidFileFormat = errors.New("format file tidak valid")

	// ErrProofNotFound dikembalikan saat bukti transfer tidak ditemukan
	ErrProofNotFound = errors.New("bukti transfer tidak ditemukan")
)
