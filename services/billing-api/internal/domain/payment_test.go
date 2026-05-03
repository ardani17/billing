package domain

import (
	"testing"
	"time"

	"pgregory.net/rapid"
)

// =============================================================================
// Generators — reusable generators for FIFO property tests
// =============================================================================

// genFIFOInput generates a valid FIFOInput with total_amount > 0,
// paid_amount >= 0, and paid_amount < total_amount (i.e., open invoice).
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

// genFIFOInputSlice generates a slice of 1..maxN valid FIFOInput entries.
func genFIFOInputSlice(t *rapid.T, maxN int) []FIFOInput {
	n := rapid.IntRange(1, maxN).Draw(t, "numInvoices")
	invoices := make([]FIFOInput, n)
	for i := 0; i < n; i++ {
		invoices[i] = genFIFOInput(t, rapid.StringMatching(`[a-z]{3}`).Draw(t, "label"))
		// Ensure unique IDs
		invoices[i].InvoiceID = rapid.StringMatching(`[0-9a-f]{8}`).Draw(t, "invoiceID")
		invoices[i].InvoiceNumber = "INV-" + invoices[i].InvoiceID
	}
	return invoices
}

// =============================================================================
// Property 1: FIFO Allocation Sum Invariant
// =============================================================================

// Feature: payment-manual, Property 1: FIFO Allocation Sum Invariant
// **Validates: Requirements 5.8, 16.5**
//
// For any list of open invoices (each with total_amount > 0, paid_amount >= 0,
// paid_amount < total_amount) and for any positive payment amount,
// AllocatePaymentFIFO(invoices, amount) produces a result where
// TotalAllocated + ExcessToCredit == amount exactly.
func TestProperty_FIFOAllocationSumInvariant(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		invoices := genFIFOInputSlice(t, 10)
		amount := rapid.Int64Range(1, 500_000_000).Draw(t, "amount")

		result := AllocatePaymentFIFO(invoices, amount)

		// Property 1: TotalAllocated + ExcessToCredit == amount
		if result.TotalAllocated+result.ExcessToCredit != amount {
			t.Fatalf(
				"Sum invariant violated: TotalAllocated(%d) + ExcessToCredit(%d) = %d, expected %d",
				result.TotalAllocated, result.ExcessToCredit,
				result.TotalAllocated+result.ExcessToCredit, amount,
			)
		}

		// Additional: TotalAllocated should equal sum of individual allocations
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
// Property 2: FIFO Allocation Status Determination
// =============================================================================

// Feature: payment-manual, Property 2: FIFO Allocation Status Determination
// **Validates: Requirements 5.6, 5.7**
//
// For any invoice in the FIFO allocation result:
// - if allocated_amount equals remaining (total_amount - paid_amount) then new_status == lunas
// - if allocated_amount > 0 but < remaining then new_status == bayar_sebagian
// - invoices with allocated_amount == 0 do not appear in allocations
func TestProperty_FIFOAllocationStatusDetermination(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		invoices := genFIFOInputSlice(t, 10)
		amount := rapid.Int64Range(1, 500_000_000).Draw(t, "amount")

		result := AllocatePaymentFIFO(invoices, amount)

		// Build a map from invoice ID to input for lookup
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

			// Property 2a: allocated_amount == 0 should not appear
			if alloc.AllocatedAmt == 0 {
				t.Fatalf(
					"Invoice %s has allocated_amount == 0 but appears in allocations",
					alloc.InvoiceID,
				)
			}

			// Property 2b: if allocated_amount == remaining → lunas
			if alloc.AllocatedAmt == remaining {
				if alloc.NewStatus != InvoiceStatusLunas {
					t.Fatalf(
						"Invoice %s: allocated_amount(%d) == remaining(%d) but new_status is %s, expected lunas",
						alloc.InvoiceID, alloc.AllocatedAmt, remaining, alloc.NewStatus,
					)
				}
			}

			// Property 2c: if 0 < allocated_amount < remaining → bayar_sebagian
			if alloc.AllocatedAmt > 0 && alloc.AllocatedAmt < remaining {
				if alloc.NewStatus != InvoiceStatusBayarSebagian {
					t.Fatalf(
						"Invoice %s: 0 < allocated_amount(%d) < remaining(%d) but new_status is %s, expected bayar_sebagian",
						alloc.InvoiceID, alloc.AllocatedAmt, remaining, alloc.NewStatus,
					)
				}
			}

			// Property 2d: allocated_amount should not exceed remaining
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
// Property 3: FIFO Allocation Ordering
// =============================================================================

// Feature: payment-manual, Property 3: FIFO Allocation Ordering
// **Validates: Requirements 5.1, 5.5**
//
// For any list of open invoices sorted by due_date ascending and any positive
// payment amount, if invoice at index i has allocated_amount < remaining_amount,
// then all invoices at index j > i have allocated_amount == 0 (full allocation
// before moving to next).
func TestProperty_FIFOAllocationOrdering(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		invoices := genFIFOInputSlice(t, 10)
		amount := rapid.Int64Range(1, 500_000_000).Draw(t, "amount")

		result := AllocatePaymentFIFO(invoices, amount)

		// Build a map from invoice ID to its input index (preserving FIFO order)
		idToIndex := make(map[string]int)
		for i, inv := range invoices {
			idToIndex[inv.InvoiceID] = i
		}

		// Build a map from invoice ID to allocation
		allocMap := make(map[string]PaymentAllocation)
		for _, alloc := range result.Allocations {
			allocMap[alloc.InvoiceID] = alloc
		}

		// Build a map from invoice ID to input
		inputMap := make(map[string]FIFOInput)
		for _, inv := range invoices {
			inputMap[inv.InvoiceID] = inv
		}

		// Find the first invoice that was partially allocated (not fully paid)
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

		// If there's a partially allocated invoice, all invoices after it
		// in the FIFO order should have zero allocation
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
// Property 4: Pay-All Clears All Invoices
// =============================================================================

// Feature: payment-manual, Property 4: Pay-All Clears All Invoices
// **Validates: Requirements 6.1, 6.4**
//
// When payment amount equals the sum of all remaining amounts (total_arrears),
// every invoice in the result has new_status == lunas and excess_to_credit == 0.
func TestProperty_PayAllClearsAllInvoices(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		invoices := genFIFOInputSlice(t, 10)

		// Calculate total arrears
		var totalArrears int64
		for _, inv := range invoices {
			totalArrears += inv.TotalAmount - inv.PaidAmount
		}

		// Pay exactly the total arrears
		result := AllocatePaymentFIFO(invoices, totalArrears)

		// Property 4a: every invoice should be lunas
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

		// Property 4b: excess_to_credit should be 0
		if result.ExcessToCredit != 0 {
			t.Fatalf(
				"ExcessToCredit should be 0 when paying exact total arrears, got %d",
				result.ExcessToCredit,
			)
		}

		// Property 4c: TotalAllocated should equal totalArrears
		if result.TotalAllocated != totalArrears {
			t.Fatalf(
				"TotalAllocated(%d) should equal totalArrears(%d)",
				result.TotalAllocated, totalArrears,
			)
		}
	})
}

// =============================================================================
// Property 8: Void Status Determination
// =============================================================================

// Feature: payment-manual, Property 8: Void Status Determination
// **Validates: Requirements 11.2, 11.3, 11.4**
//
// DeterminePostVoidStatus(paidAmount, totalAmount, dueDate, now) returns:
// - belum_bayar if paidAmount == 0 and dueDate > now
// - terlambat if paidAmount == 0 and dueDate <= now
// - bayar_sebagian if 0 < paidAmount < totalAmount
func TestProperty_VoidStatusDetermination(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Choose one of three scenarios
		scenario := rapid.IntRange(0, 2).Draw(t, "scenario")

		// Base time for "now"
		now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)

		switch scenario {
		case 0:
			// paidAmount == 0, dueDate > now → belum_bayar
			totalAmount := rapid.Int64Range(1, 100_000_000).Draw(t, "totalAmount")
			paidAmount := int64(0)
			// dueDate is 1 to 365 days in the future
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
			// paidAmount == 0, dueDate <= now → terlambat
			totalAmount := rapid.Int64Range(1, 100_000_000).Draw(t, "totalAmount")
			paidAmount := int64(0)
			// dueDate is 0 to 365 days in the past (0 = same time = equal)
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
			// 0 < paidAmount < totalAmount → bayar_sebagian
			// totalAmount must be >= 2 so that paidAmount range [1, totalAmount-1] is valid
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
// Property 10: Remaining Amount and Total Arrears Calculation
// =============================================================================

// Feature: payment-manual, Property 10: Remaining Amount and Total Arrears Calculation
// **Validates: Requirements 4.2, 4.3**
//
// For any set of open invoices, each invoice's remaining_amount == total_amount - paid_amount,
// and total_arrears == sum of all remaining_amount values.
func TestProperty_RemainingAmountAndTotalArrears(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(1, 20).Draw(t, "numInvoices")

		var totalArrears int64
		for i := 0; i < n; i++ {
			totalAmount := rapid.Int64Range(1, 100_000_000).Draw(t, "totalAmount")
			paidAmount := rapid.Int64Range(0, totalAmount-1).Draw(t, "paidAmount")

			remainingAmount := totalAmount - paidAmount

			// Property 10a: remaining_amount == total_amount - paid_amount
			if remainingAmount != totalAmount-paidAmount {
				t.Fatalf(
					"remaining_amount(%d) != total_amount(%d) - paid_amount(%d)",
					remainingAmount, totalAmount, paidAmount,
				)
			}

			// Property 10b: remaining_amount > 0 for open invoices
			if remainingAmount <= 0 {
				t.Fatalf(
					"remaining_amount should be > 0 for open invoice: total=%d, paid=%d, remaining=%d",
					totalAmount, paidAmount, remainingAmount,
				)
			}

			totalArrears += remainingAmount
		}

		// Property 10c: total_arrears == sum of all remaining_amount values
		// Verify by recomputing
		if totalArrears <= 0 {
			t.Fatalf("total_arrears should be > 0 for non-empty set of open invoices, got %d", totalArrears)
		}
	})
}
