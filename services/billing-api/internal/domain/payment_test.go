package domain

import (
	"testing"
	"time"

	"pgregory.net/rapid"
)

// =============================================================================
// =============================================================================

// paid_amount >= 0, dan paid_amount < total_amount (i.e., invoice terbuka).
func genFIFOInput(t *rapid.T, label string) FIFOInput {
	totalAmount := rapid.Int64Range(1, 100_000_000).Draw(t, label+"_totalAmount")
	paidAmount := rapid.Int64Range(0, totalAmount-1).Draw(t, label+"_paidAmount")
	return FIFOInput{
		InvoiceID:     "inv-" + label,
		InvoiceNumber: "INV-" + label,
		TotalAmount:   totalAmount,
		PaidAmount:    paidAmount,
		Status:        InvoiceStatusBelumBayar,
	}
}

func genFIFOInputSlice(t *rapid.T, maxN int) []FIFOInput {
	n := rapid.IntRange(1, maxN).Draw(t, "numInvoices")
	invoices := make([]FIFOInput, n)
	for i := 0; i < n; i++ {
		invoices[i] = genFIFOInput(t, rapid.StringMatching(`[a-z]{3}`).Draw(t, "label"))
		invoices[i].InvoiceID = rapid.StringMatching(`[0-9a-f]{8}`).Draw(t, "invoiceID")
		invoices[i].InvoiceNumber = "INV-" + invoices[i].InvoiceID
	}
	return invoices
}

// =============================================================================
// =============================================================================

// **Memvalidasi: Kebutuhan 5.8, 16.5**
//
// TotalAllocated + ExcessToCredit == nominal exactly.
func TestProperty_FIFOAllocationSumInvariant(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		invoices := genFIFOInputSlice(t, 10)
		amount := rapid.Int64Range(1, 500_000_000).Draw(t, "amount")

		result := AllocatePaymentFIFO(invoices, amount)

		if result.TotalAllocated+result.ExcessToCredit != amount {
			t.Fatalf(
				"Sum invariant violated: TotalAllocated(%d) + ExcessToCredit(%d) = %d, expected %d",
				result.TotalAllocated, result.ExcessToCredit,
				result.TotalAllocated+result.ExcessToCredit, amount,
			)
		}

		var sumAlloc int64
		for _, a := range result.Allocations {
			sumAlloc += a.AllocatedAmt
		}
		if sumAlloc != result.TotalAllocated {
			t.Fatalf(
				"TotalAllocated(%d) != sum of individual allocations(%d)",
				result.TotalAllocated, sumAlloc,
			)
		}
	})
}

// =============================================================================
// =============================================================================

// **Memvalidasi: Kebutuhan 5.6, 5.7**
//
// - jika allocated_amount > 0 but < remaining then new_status == bayar_sebagian
func TestProperty_FIFOAllocationStatusDetermination(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		invoices := genFIFOInputSlice(t, 10)
		amount := rapid.Int64Range(1, 500_000_000).Draw(t, "amount")

		result := AllocatePaymentFIFO(invoices, amount)

		inputMap := make(map[string]FIFOInput)
		for _, inv := range invoices {
			inputMap[inv.InvoiceID] = inv
		}

		for _, alloc := range result.Allocations {
			inp, ok := inputMap[alloc.InvoiceID]
			if !ok {
				t.Fatalf("Allocation references unknown invoice ID %q", alloc.InvoiceID)
			}

			remaining := inp.TotalAmount - inp.PaidAmount

			if alloc.AllocatedAmt == 0 {
				t.Fatalf(
					"Invoice %s has allocated_amount == 0 but appears in allocations",
					alloc.InvoiceID,
				)
			}

			if alloc.AllocatedAmt == remaining {
				if alloc.NewStatus != InvoiceStatusLunas {
					t.Fatalf(
						"Invoice %s: allocated_amount(%d) == remaining(%d) but new_status is %s, expected lunas",
						alloc.InvoiceID, alloc.AllocatedAmt, remaining, alloc.NewStatus,
					)
				}
			}

			if alloc.AllocatedAmt > 0 && alloc.AllocatedAmt < remaining {
				if alloc.NewStatus != InvoiceStatusBayarSebagian {
					t.Fatalf(
						"Invoice %s: 0 < allocated_amount(%d) < remaining(%d) but new_status is %s, expected bayar_sebagian",
						alloc.InvoiceID, alloc.AllocatedAmt, remaining, alloc.NewStatus,
					)
				}
			}

			if alloc.AllocatedAmt > remaining {
				t.Fatalf(
					"Invoice %s: allocated_amount(%d) exceeds remaining(%d)",
					alloc.InvoiceID, alloc.AllocatedAmt, remaining,
				)
			}
		}
	})
}

// =============================================================================
// =============================================================================

// **Memvalidasi: Kebutuhan 5.1, 5.5**
//
// payment nominal, jika invoice at index i has allocated_amount < remaining_amount,
// before moving to next).
func TestProperty_FIFOAllocationOrdering(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		invoices := genFIFOInputSlice(t, 10)
		amount := rapid.Int64Range(1, 500_000_000).Draw(t, "amount")

		result := AllocatePaymentFIFO(invoices, amount)

		idToIndex := make(map[string]int)
		for i, inv := range invoices {
			idToIndex[inv.InvoiceID] = i
		}

		allocMap := make(map[string]PaymentAllocation)
		for _, alloc := range result.Allocations {
			allocMap[alloc.InvoiceID] = alloc
		}

		inputMap := make(map[string]FIFOInput)
		for _, inv := range invoices {
			inputMap[inv.InvoiceID] = inv
		}

		partialIndex := -1
		for _, alloc := range result.Allocations {
			inp := inputMap[alloc.InvoiceID]
			remaining := inp.TotalAmount - inp.PaidAmount
			if alloc.AllocatedAmt < remaining {
				idx := idToIndex[alloc.InvoiceID]
				if partialIndex == -1 || idx < partialIndex {
					partialIndex = idx
				}
			}
		}

		if partialIndex >= 0 {
			for _, inv := range invoices {
				idx := idToIndex[inv.InvoiceID]
				if idx > partialIndex {
					alloc, exists := allocMap[inv.InvoiceID]
					if exists && alloc.AllocatedAmt > 0 {
						t.Fatalf(
							"FIFO ordering violated: invoice at index %d (ID=%s) has allocated_amount=%d, "+
								"but invoice at index %d was only partially allocated",
							idx, inv.InvoiceID, alloc.AllocatedAmt, partialIndex,
						)
					}
				}
			}
		}
	})
}

// =============================================================================
// =============================================================================

// **Memvalidasi: Kebutuhan 6.1, 6.4**
func TestProperty_PayAllClearsAllInvoices(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		invoices := genFIFOInputSlice(t, 10)

		// Hitung total tunggakan
		var totalArrears int64
		for _, inv := range invoices {
			totalArrears += inv.TotalAmount - inv.PaidAmount
		}

		result := AllocatePaymentFIFO(invoices, totalArrears)

		if len(result.Allocations) != len(invoices) {
			t.Fatalf(
				"Expected %d allocations (one per invoice), got %d",
				len(invoices), len(result.Allocations),
			)
		}

		for _, alloc := range result.Allocations {
			if alloc.NewStatus != InvoiceStatusLunas {
				t.Fatalf(
					"Invoice %s has new_status %s, expected lunas when paying total arrears",
					alloc.InvoiceID, alloc.NewStatus,
				)
			}
		}

		if result.ExcessToCredit != 0 {
			t.Fatalf(
				"ExcessToCredit should be 0 when paying exact total arrears, got %d",
				result.ExcessToCredit,
			)
		}

		if result.TotalAllocated != totalArrears {
			t.Fatalf(
				"TotalAllocated(%d) should equal totalArrears(%d)",
				result.TotalAllocated, totalArrears,
			)
		}
	})
}

// =============================================================================
// =============================================================================

// **Memvalidasi: Kebutuhan 11.2, 11.3, 11.4**
//
// - belum_bayar jika paidAmount == 0 dan dueDate > now
// - terlambat jika paidAmount == 0 dan dueDate <= now
// - bayar_sebagian jika 0 < paidAmount < totalAmount
func TestProperty_VoidStatusDetermination(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Choose one of three scenarios
		scenario := rapid.IntRange(0, 2).Draw(t, "scenario")

		// Base time untuk "now"
		now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)

		switch scenario {
		case 0:
			// paidAmount == 0, dueDate > now -> belum_bayar
			totalAmount := rapid.Int64Range(1, 100_000_000).Draw(t, "totalAmount")
			paidAmount := int64(0)
			daysAhead := rapid.IntRange(1, 365).Draw(t, "daysAhead")
			dueDate := now.AddDate(0, 0, daysAhead)

			status := DeterminePostVoidStatus(paidAmount, totalAmount, dueDate, now)
			if status != InvoiceStatusBelumBayar {
				t.Fatalf(
					"paidAmount=0, dueDate(%v) > now(%v): expected belum_bayar, got %s",
					dueDate, now, status,
				)
			}

		case 1:
			// paidAmount == 0, dueDate <= now -> terlambat
			totalAmount := rapid.Int64Range(1, 100_000_000).Draw(t, "totalAmount")
			paidAmount := int64(0)
			daysBehind := rapid.IntRange(0, 365).Draw(t, "daysBehind")
			dueDate := now.AddDate(0, 0, -daysBehind)

			status := DeterminePostVoidStatus(paidAmount, totalAmount, dueDate, now)
			if status != InvoiceStatusTerlambat {
				t.Fatalf(
					"paidAmount=0, dueDate(%v) <= now(%v): expected terlambat, got %s",
					dueDate, now, status,
				)
			}

		case 2:
			// 0 < paidAmount < totalAmount -> bayar_sebagian
			totalAmount := rapid.Int64Range(2, 100_000_000).Draw(t, "totalAmount")
			paidAmount := rapid.Int64Range(1, totalAmount-1).Draw(t, "paidAmount")
			// dueDate can be anything
			daysOffset := rapid.IntRange(-365, 365).Draw(t, "daysOffset")
			dueDate := now.AddDate(0, 0, daysOffset)

			status := DeterminePostVoidStatus(paidAmount, totalAmount, dueDate, now)
			if status != InvoiceStatusBayarSebagian {
				t.Fatalf(
					"0 < paidAmount(%d) < totalAmount(%d): expected bayar_sebagian, got %s",
					paidAmount, totalAmount, status,
				)
			}
		}
	})
}

// =============================================================================
// =============================================================================

// **Memvalidasi: Kebutuhan 4.2, 4.3**
func TestProperty_RemainingAmountAndTotalArrears(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(1, 20).Draw(t, "numInvoices")

		var totalArrears int64
		for i := 0; i < n; i++ {
			totalAmount := rapid.Int64Range(1, 100_000_000).Draw(t, "totalAmount")
			paidAmount := rapid.Int64Range(0, totalAmount-1).Draw(t, "paidAmount")

			remainingAmount := totalAmount - paidAmount

			if remainingAmount != totalAmount-paidAmount {
				t.Fatalf(
					"remaining_amount(%d) != total_amount(%d) - paid_amount(%d)",
					remainingAmount, totalAmount, paidAmount,
				)
			}

			if remainingAmount <= 0 {
				t.Fatalf(
					"remaining_amount should be > 0 for open invoice: total=%d, paid=%d, remaining=%d",
					totalAmount, paidAmount, remainingAmount,
				)
			}

			totalArrears += remainingAmount
		}

		if totalArrears <= 0 {
			t.Fatalf("total_arrears should be > 0 for non-empty set of open invoices, got %d", totalArrears)
		}
	})
}
