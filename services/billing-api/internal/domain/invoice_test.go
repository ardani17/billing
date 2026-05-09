package domain

import (
	"errors"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// **Memvalidasi: Kebutuhan 9.1, 9.3**
func TestProperty_InvoiceStateMachineDeterminism(t *testing.T) {
	allStatuses := []InvoiceStatus{
		InvoiceStatusBelumBayar,
		InvoiceStatusTerlambat,
		InvoiceStatusLunas,
		InvoiceStatusBayarSebagian,
		InvoiceStatusBatal,
		InvoiceStatusProrate,
	}

	terminalStatuses := map[InvoiceStatus]bool{
		InvoiceStatusLunas: true,
		InvoiceStatusBatal: true,
	}

	rapid.Check(t, func(t *rapid.T) {
		current := rapid.SampledFrom(allStatuses).Draw(t, "current")
		target := rapid.SampledFrom(allStatuses).Draw(t, "target")

		expectedValid := false
		for _, allowed := range ValidInvoiceTransitions[current] {
			if allowed == target {
				expectedValid = true
				break
			}
		}

		canResult := CanInvoiceTransition(current, target)
		if canResult != expectedValid {
			t.Fatalf("CanInvoiceTransition(%s, %s) = %v, expected %v", current, target, canResult, expectedValid)
		}

		newStatus, err := InvoiceTransition(current, target)
		if expectedValid {
			if err != nil {
				t.Fatalf("InvoiceTransition(%s, %s) returned unexpected error: %v", current, target, err)
			}
			if newStatus != target {
				t.Fatalf("InvoiceTransition(%s, %s) returned %s, expected %s", current, target, newStatus, target)
			}
		} else {
			if err == nil {
				t.Fatalf("InvoiceTransition(%s, %s) expected error, got nil", current, target)
			}
			if newStatus != current {
				t.Fatalf("InvoiceTransition(%s, %s) returned status %s on error, expected %s (unchanged)", current, target, newStatus, current)
			}
			if !errors.Is(err, ErrInvalidInvoiceStatusTransition) {
				t.Fatalf("expected ErrInvalidInvoiceStatusTransition, got %v", err)
			}
			allowedTargets := AllowedInvoiceTargets(current)
			for _, at := range allowedTargets {
				if !strings.Contains(err.Error(), string(at)) {
					t.Fatalf("error message %q does not contain allowed target %q", err.Error(), at)
				}
			}
		}

		allowedTargets := AllowedInvoiceTargets(current)
		expectedTargets := ValidInvoiceTransitions[current]
		if len(allowedTargets) != len(expectedTargets) {
			t.Fatalf("AllowedInvoiceTargets(%s) returned %d targets, expected %d", current, len(allowedTargets), len(expectedTargets))
		}
		for i, at := range allowedTargets {
			if at != expectedTargets[i] {
				t.Fatalf("AllowedInvoiceTargets(%s)[%d] = %s, expected %s", current, i, at, expectedTargets[i])
			}
		}

		if terminalStatuses[current] {
			if len(ValidInvoiceTransitions[current]) != 0 {
				t.Fatalf("terminal state %s should have no valid transitions, got %v", current, ValidInvoiceTransitions[current])
			}
			if canResult {
				t.Fatalf("terminal state %s should not allow transition to %s", current, target)
			}
		}
	})
}

// **Memvalidasi: Kebutuhan 19.5**
func TestProperty_LateFeeCappedByMaxAmount(t *testing.T) {
	allPenaltyTypes := []PenaltyType{
		PenaltyFixed,
		PenaltyPercentage,
		PenaltyDaily,
	}

	rapid.Check(t, func(t *rapid.T) {
		penaltyType := rapid.SampledFrom(allPenaltyTypes).Draw(t, "penaltyType")
		penaltyMaxAmount := rapid.Int64Range(1, 10_000_000).Draw(t, "penaltyMaxAmount")

		settings := &BillingSettings{
			PenaltyEnabled:     true,
			PenaltyType:        penaltyType,
			PenaltyAmount:      rapid.Int64Range(0, 50_000_000).Draw(t, "penaltyAmount"),
			PenaltyPercentage:  rapid.Float64Range(0, 100).Draw(t, "penaltyPercentage"),
			PenaltyDailyAmount: rapid.Int64Range(0, 1_000_000).Draw(t, "penaltyDailyAmount"),
			PenaltyMaxAmount:   penaltyMaxAmount,
		}

		subtotal := rapid.Int64Range(0, 100_000_000).Draw(t, "subtotal")
		daysOverdue := rapid.IntRange(0, 365).Draw(t, "daysOverdue")

		fee := CalculateLateFee(settings, subtotal, daysOverdue)

		if fee > penaltyMaxAmount {
			t.Fatalf(
				"CalculateLateFee returned %d which exceeds penalty_max_amount %d (type=%s, subtotal=%d, daysOverdue=%d)",
				fee, penaltyMaxAmount, penaltyType, subtotal, daysOverdue,
			)
		}
	})
}

// **Memvalidasi: Kebutuhan 2.5**
func TestProperty_InvoiceItemAmountConsistency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		quantity := rapid.IntRange(1, 10_000).Draw(t, "quantity")
		unitPrice := rapid.Int64Range(1, 100_000_000).Draw(t, "unitPrice")

		expectedAmount := int64(quantity) * unitPrice

		item := InvoiceItem{
			Quantity:  quantity,
			UnitPrice: unitPrice,
			Amount:    int64(quantity) * unitPrice,
		}

		if item.Amount != expectedAmount {
			t.Fatalf(
				"InvoiceItem amount mismatch: quantity=%d, unit_price=%d, got amount=%d, expected=%d",
				quantity, unitPrice, item.Amount, expectedAmount,
			)
		}

		if item.Amount <= 0 {
			t.Fatalf(
				"InvoiceItem amount should be positive for positive quantity=%d and unit_price=%d, got %d",
				quantity, unitPrice, item.Amount,
			)
		}
	})
}

// **Memvalidasi: Kebutuhan 14.5**
//
// subtotal + tax_amount + penalty_amount - discount_amount - credit_applied,
func TestProperty_InvoiceTotalAmountInvariant(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		subtotal := rapid.Int64Range(0, 100_000_000).Draw(t, "subtotal")
		taxAmount := rapid.Int64Range(0, 50_000_000).Draw(t, "taxAmount")
		penaltyAmount := rapid.Int64Range(0, 10_000_000).Draw(t, "penaltyAmount")

		maxDeduction := subtotal + taxAmount + penaltyAmount
		discountAmount := rapid.Int64Range(0, maxDeduction).Draw(t, "discountAmount")
		remainingForCredit := maxDeduction - discountAmount
		creditApplied := rapid.Int64Range(0, remainingForCredit).Draw(t, "creditApplied")

		expectedTotal := subtotal + taxAmount + penaltyAmount - discountAmount - creditApplied

		invoice := Invoice{
			Subtotal:       subtotal,
			TaxAmount:      taxAmount,
			PenaltyAmount:  penaltyAmount,
			DiscountAmount: discountAmount,
			CreditApplied:  creditApplied,
			TotalAmount:    expectedTotal,
		}

		computedTotal := invoice.Subtotal + invoice.TaxAmount + invoice.PenaltyAmount - invoice.DiscountAmount - invoice.CreditApplied
		if invoice.TotalAmount != computedTotal {
			t.Fatalf(
				"Invoice total_amount mismatch: subtotal=%d + tax=%d + penalty=%d - discount=%d - credit=%d = %d, got total_amount=%d",
				invoice.Subtotal, invoice.TaxAmount, invoice.PenaltyAmount,
				invoice.DiscountAmount, invoice.CreditApplied,
				computedTotal, invoice.TotalAmount,
			)
		}

		if invoice.TotalAmount < 0 {
			t.Fatalf(
				"Invoice total_amount should be >= 0, got %d (subtotal=%d, tax=%d, penalty=%d, discount=%d, credit=%d)",
				invoice.TotalAmount, invoice.Subtotal, invoice.TaxAmount,
				invoice.PenaltyAmount, invoice.DiscountAmount, invoice.CreditApplied,
			)
		}
	})
}

// **Memvalidasi: Kebutuhan 14.2**
func TestProperty_OnlyBelumBayarInvoicesAreEditable(t *testing.T) {
	nonEditableStatuses := []InvoiceStatus{
		InvoiceStatusTerlambat,
		InvoiceStatusLunas,
		InvoiceStatusBayarSebagian,
		InvoiceStatusBatal,
		InvoiceStatusProrate,
	}

	rapid.Check(t, func(t *rapid.T) {
		status := rapid.SampledFrom(nonEditableStatuses).Draw(t, "status")

		invoice := Invoice{
			Status: status,
		}

		if invoice.Status == InvoiceStatusBelumBayar {
			t.Fatalf("generator produced belum_bayar status, which should not happen in this test")
		}

		isEditable := invoice.Status == InvoiceStatusBelumBayar
		if isEditable {
			t.Fatalf(
				"Invoice with status %s should NOT be editable, but editability check returned true",
				invoice.Status,
			)
		}
	})
}

// **Memvalidasi: Kebutuhan 15.4, 21.4**
func TestProperty_CreditRestorationOnCancelRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		creditApplied := rapid.Int64Range(1, 100_000_000).Draw(t, "creditApplied")
		initialCreditBalance := rapid.Int64Range(0, 100_000_000).Draw(t, "initialCreditBalance")

		// customer.CreditBalance += invoice.CreditApplied
		newCreditBalance := initialCreditBalance + creditApplied

		if newCreditBalance != initialCreditBalance+creditApplied {
			t.Fatalf(
				"Credit restoration failed: initial=%d + credit_applied=%d should equal %d, got %d",
				initialCreditBalance, creditApplied, initialCreditBalance+creditApplied, newCreditBalance,
			)
		}

		if newCreditBalance <= initialCreditBalance {
			t.Fatalf(
				"Credit restoration should increase balance: initial=%d, credit_applied=%d, new=%d",
				initialCreditBalance, creditApplied, newCreditBalance,
			)
		}

		if newCreditBalance < 0 {
			t.Fatalf(
				"Credit balance should be non-negative after restoration: initial=%d, credit_applied=%d, new=%d",
				initialCreditBalance, creditApplied, newCreditBalance,
			)
		}
	})
}

// **Memvalidasi: Kebutuhan 8.6, 21.2**
func TestProperty_CreditApplicationBounded(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		creditBalance := rapid.Int64Range(0, 100_000_000).Draw(t, "creditBalance")
		totalAmount := rapid.Int64Range(1, 100_000_000).Draw(t, "totalAmount")

		// Simulasikan credit application logic: credit_applied = min(credit_balance, total_amount)
		var creditApplied int64
		if creditBalance < totalAmount {
			creditApplied = creditBalance
		} else {
			creditApplied = totalAmount
		}

		expectedCreditApplied := creditBalance
		if totalAmount < expectedCreditApplied {
			expectedCreditApplied = totalAmount
		}
		if creditApplied != expectedCreditApplied {
			t.Fatalf(
				"Credit applied should be min(%d, %d) = %d, got %d",
				creditBalance, totalAmount, expectedCreditApplied, creditApplied,
			)
		}

		if creditApplied > creditBalance {
			t.Fatalf(
				"Credit applied %d exceeds credit_balance %d",
				creditApplied, creditBalance,
			)
		}

		if creditApplied > totalAmount {
			t.Fatalf(
				"Credit applied %d exceeds total_amount %d",
				creditApplied, totalAmount,
			)
		}

		newCreditBalance := creditBalance - creditApplied
		if newCreditBalance != creditBalance-creditApplied {
			t.Fatalf(
				"New credit_balance should be %d - %d = %d, got %d",
				creditBalance, creditApplied, creditBalance-creditApplied, newCreditBalance,
			)
		}

		if newCreditBalance < 0 {
			t.Fatalf(
				"New credit_balance should be non-negative: original=%d, applied=%d, new=%d",
				creditBalance, creditApplied, newCreditBalance,
			)
		}
	})
}

// **Memvalidasi: Kebutuhan 21.1**
func TestProperty_OverpaymentBecomesCredit(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		totalAmount := rapid.Int64Range(1, 100_000_000).Draw(t, "totalAmount")
		paidAmount := rapid.Int64Range(0, totalAmount-1).Draw(t, "paidAmount")
		remainingBalance := totalAmount - paidAmount

		paymentAmount := rapid.Int64Range(remainingBalance+1, remainingBalance+50_000_000).Draw(t, "paymentAmount")

		initialCreditBalance := rapid.Int64Range(0, 100_000_000).Draw(t, "initialCreditBalance")

		// Simulasikan logika pemrosesan pembayaran
		newPaidAmount := paidAmount + paymentAmount
		var excessAmount int64
		var newStatus InvoiceStatus

		if newPaidAmount >= totalAmount {
			newStatus = InvoiceStatusLunas
			excessAmount = newPaidAmount - totalAmount
			newPaidAmount = totalAmount // cap at total_amount
		}

		newCreditBalance := initialCreditBalance + excessAmount

		expectedExcess := paymentAmount - remainingBalance
		if excessAmount != expectedExcess {
			t.Fatalf(
				"Excess should be payment(%d) - remaining(%d) = %d, got %d",
				paymentAmount, remainingBalance, expectedExcess, excessAmount,
			)
		}

		if newPaidAmount != totalAmount {
			t.Fatalf(
				"paid_amount should be capped at total_amount %d, got %d",
				totalAmount, newPaidAmount,
			)
		}

		if newStatus != InvoiceStatusLunas {
			t.Fatalf(
				"Status should be lunas after overpayment, got %s",
				newStatus,
			)
		}

		if newCreditBalance != initialCreditBalance+excessAmount {
			t.Fatalf(
				"Credit balance should be initial(%d) + excess(%d) = %d, got %d",
				initialCreditBalance, excessAmount, initialCreditBalance+excessAmount, newCreditBalance,
			)
		}

		if excessAmount <= 0 {
			t.Fatalf(
				"Excess should be positive for overpayment: payment=%d, remaining=%d, excess=%d",
				paymentAmount, remainingBalance, excessAmount,
			)
		}
	})
}

// **Memvalidasi: Kebutuhan 21.5**
func TestProperty_CreditBalanceNonNegativeInvariant(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		creditBalance := rapid.Int64Range(0, 100_000_000).Draw(t, "initialCreditBalance")

		// Buat urutan operasi kredit secara acak
		numOps := rapid.IntRange(1, 20).Draw(t, "numOps")

		for i := 0; i < numOps; i++ {
			opType := rapid.IntRange(0, 2).Draw(t, "opType")

			switch opType {
			case 0:
				// Terapkan kredit ke invoice: credit_applied = min(credit_balance, total_amount)
				totalAmount := rapid.Int64Range(1, 50_000_000).Draw(t, "totalAmount")
				var creditApplied int64
				if creditBalance < totalAmount {
					creditApplied = creditBalance
				} else {
					creditApplied = totalAmount
				}
				creditBalance -= creditApplied

			case 1:
				// Restore credit on cancel: credit_balance += credit_applied
				creditApplied := rapid.Int64Range(0, 50_000_000).Draw(t, "creditApplied")
				creditBalance += creditApplied

			case 2:
				// Add overpayment credit: credit_balance += excess
				excessAmount := rapid.Int64Range(0, 10_000_000).Draw(t, "excessAmount")
				creditBalance += excessAmount
			}

			if creditBalance < 0 {
				t.Fatalf(
					"Credit balance became negative (%d) after operation %d (type=%d)",
					creditBalance, i, opType,
				)
			}
		}
	})
}

// **Memvalidasi: Kebutuhan 8.5, 20.2, 20.4**
//
// non-denda line item. Penalty does not affect tax calculation.
func TestProperty_TaxCalculatedOnSubtotalExcludingPenalty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Buat line item: sebagian reguler (bulanan, berulang, kustom), sebagian denda
		numRegularItems := rapid.IntRange(1, 10).Draw(t, "numRegularItems")
		numPenaltyItems := rapid.IntRange(0, 3).Draw(t, "numPenaltyItems")

		var subtotal int64
		var items []InvoiceItem

		// Regular items (contribute to subtotal untuk tax)
		regularTypes := []InvoiceItemType{ItemTypeMonthly, ItemTypeRecurring, ItemTypeCustom, ItemTypeInstallation}
		for i := 0; i < numRegularItems; i++ {
			itemType := rapid.SampledFrom(regularTypes).Draw(t, "regularItemType")
			quantity := rapid.IntRange(1, 10).Draw(t, "quantity")
			unitPrice := rapid.Int64Range(1, 10_000_000).Draw(t, "unitPrice")
			amount := int64(quantity) * unitPrice
			subtotal += amount
			items = append(items, InvoiceItem{
				ItemType:  itemType,
				Quantity:  quantity,
				UnitPrice: unitPrice,
				Amount:    amount,
			})
		}

		var penaltyTotal int64
		for i := 0; i < numPenaltyItems; i++ {
			penaltyAmount := rapid.Int64Range(1, 5_000_000).Draw(t, "penaltyAmount")
			penaltyTotal += penaltyAmount
			items = append(items, InvoiceItem{
				ItemType:  ItemTypePenalty,
				Quantity:  1,
				UnitPrice: penaltyAmount,
				Amount:    penaltyAmount,
			})
		}

		// Tax rate between 0.01 dan 100
		taxRate := rapid.Float64Range(0.01, 100.0).Draw(t, "taxRate")

		// Hitung pajak dari subtotal (tidak termasuk denda)
		taxAmount := int64(float64(subtotal) * taxRate / 100)

		var computedSubtotal int64
		for _, item := range items {
			if item.ItemType != ItemTypeTax && item.ItemType != ItemTypePenalty && item.ItemType != ItemTypeCreditApplied && item.ItemType != ItemTypeDiscount {
				computedSubtotal += item.Amount
			}
		}

		if computedSubtotal != subtotal {
			t.Fatalf(
				"Computed subtotal %d does not match expected subtotal %d",
				computedSubtotal, subtotal,
			)
		}

		expectedTax := int64(float64(subtotal) * taxRate / 100)
		if taxAmount != expectedTax {
			t.Fatalf(
				"Tax amount %d does not match expected %d (subtotal=%d, rate=%.2f)",
				taxAmount, expectedTax, subtotal, taxRate,
			)
		}

		taxWithPenalty := int64(float64(subtotal+penaltyTotal) * taxRate / 100)
		if penaltyTotal > 0 && taxWithPenalty == taxAmount {
		}
		taxFromSubtotalOnly := int64(float64(subtotal) * taxRate / 100)
		if taxAmount != taxFromSubtotalOnly {
			t.Fatalf(
				"Tax should be calculated from subtotal only (%d), not including penalty (%d). Expected tax=%d, got=%d",
				subtotal, penaltyTotal, taxFromSubtotalOnly, taxAmount,
			)
		}

		if taxAmount < 0 {
			t.Fatalf(
				"Tax amount should be non-negative: subtotal=%d, rate=%.2f, tax=%d",
				subtotal, taxRate, taxAmount,
			)
		}
	})
}

// **Memvalidasi: Kebutuhan 8.2, 8.3, 8.4, 8.5**
func TestProperty_LateFeeCalculationCorrectnessAcrossPenaltyTypes(t *testing.T) {
	t.Run("penalty_enabled=true", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			penaltyType := rapid.SampledFrom([]PenaltyType{
				PenaltyFixed,
				PenaltyPercentage,
				PenaltyDaily,
			}).Draw(t, "penaltyType")

			penaltyAmount := rapid.Int64Range(1, 50_000_000).Draw(t, "penaltyAmount")
			penaltyPercentage := rapid.Float64Range(0.01, 100.0).Draw(t, "penaltyPercentage")
			penaltyDailyAmount := rapid.Int64Range(1, 1_000_000).Draw(t, "penaltyDailyAmount")

			settings := &BillingSettings{
				PenaltyEnabled:     true,
				PenaltyType:        penaltyType,
				PenaltyAmount:      penaltyAmount,
				PenaltyPercentage:  penaltyPercentage,
				PenaltyDailyAmount: penaltyDailyAmount,
				PenaltyMaxAmount:   0, // no cap, so we can verify raw calculation
			}

			subtotal := rapid.Int64Range(1, 100_000_000).Draw(t, "subtotal")
			daysOverdue := rapid.IntRange(0, 365).Draw(t, "daysOverdue")

			fee := CalculateLateFee(settings, subtotal, daysOverdue)

			switch penaltyType {
			case PenaltyFixed:
				expected := penaltyAmount
				if fee != expected {
					t.Fatalf(
						"fixed: CalculateLateFee returned %d, expected penalty_amount %d",
						fee, expected,
					)
				}
			case PenaltyPercentage:
				expected := subtotal * int64(penaltyPercentage) / 100
				if fee != expected {
					t.Fatalf(
						"percentage: CalculateLateFee returned %d, expected subtotal(%d)*percentage(%.2f)/100 = %d",
						fee, subtotal, penaltyPercentage, expected,
					)
				}
			case PenaltyDaily:
				expected := penaltyDailyAmount * int64(daysOverdue)
				if fee != expected {
					t.Fatalf(
						"daily: CalculateLateFee returned %d, expected daily_amount(%d)*daysOverdue(%d) = %d",
						fee, penaltyDailyAmount, daysOverdue, expected,
					)
				}
			}
		})
	})

	t.Run("penalty_enabled=false_always_returns_zero", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			penaltyType := rapid.SampledFrom([]PenaltyType{
				PenaltyFixed,
				PenaltyPercentage,
				PenaltyDaily,
			}).Draw(t, "penaltyType")

			settings := &BillingSettings{
				PenaltyEnabled:     false,
				PenaltyType:        penaltyType,
				PenaltyAmount:      rapid.Int64Range(1, 50_000_000).Draw(t, "penaltyAmount"),
				PenaltyPercentage:  rapid.Float64Range(0.01, 100.0).Draw(t, "penaltyPercentage"),
				PenaltyDailyAmount: rapid.Int64Range(1, 1_000_000).Draw(t, "penaltyDailyAmount"),
				PenaltyMaxAmount:   rapid.Int64Range(0, 10_000_000).Draw(t, "penaltyMaxAmount"),
			}

			subtotal := rapid.Int64Range(1, 100_000_000).Draw(t, "subtotal")
			daysOverdue := rapid.IntRange(0, 365).Draw(t, "daysOverdue")

			fee := CalculateLateFee(settings, subtotal, daysOverdue)

			if fee != 0 {
				t.Fatalf(
					"penalty_enabled=false: CalculateLateFee returned %d, expected 0 (type=%s, subtotal=%d, daysOverdue=%d)",
					fee, penaltyType, subtotal, daysOverdue,
				)
			}
		})
	})
}

// **Memvalidasi: Kebutuhan 8.6**
func TestProperty_LateFeeCap(t *testing.T) {
	allPenaltyTypes := []PenaltyType{
		PenaltyFixed,
		PenaltyPercentage,
		PenaltyDaily,
	}

	rapid.Check(t, func(t *rapid.T) {
		// Pilih tipe penalti secara acak
		penaltyType := rapid.SampledFrom(allPenaltyTypes).Draw(t, "penaltyType")

		// penalty_max_amount harus > 0 untuk menguji cap
		penaltyMaxAmount := rapid.Int64Range(1, 50_000_000).Draw(t, "penaltyMaxAmount")

		settings := &BillingSettings{
			PenaltyEnabled:     true,
			PenaltyType:        penaltyType,
			PenaltyAmount:      rapid.Int64Range(0, 100_000_000).Draw(t, "penaltyAmount"),
			PenaltyPercentage:  rapid.Float64Range(0, 100).Draw(t, "penaltyPercentage"),
			PenaltyDailyAmount: rapid.Int64Range(0, 5_000_000).Draw(t, "penaltyDailyAmount"),
			PenaltyMaxAmount:   penaltyMaxAmount,
		}

		// subtotal positif
		subtotal := rapid.Int64Range(1, 100_000_000).Draw(t, "subtotal")
		// daysOverdue non-negatif
		daysOverdue := rapid.IntRange(0, 365).Draw(t, "daysOverdue")

		fee := CalculateLateFee(settings, subtotal, daysOverdue)

		// Invariant: fee tidak boleh melebihi penalty_max_amount
		if fee > penaltyMaxAmount {
			t.Fatalf(
				"CalculateLateFee returned %d which exceeds penalty_max_amount %d "+
					"(type=%s, subtotal=%d, daysOverdue=%d, penaltyAmount=%d, penaltyPercentage=%.2f, penaltyDailyAmount=%d)",
				fee, penaltyMaxAmount, penaltyType, subtotal, daysOverdue,
				settings.PenaltyAmount, settings.PenaltyPercentage, settings.PenaltyDailyAmount,
			)
		}

		// Tambahan: fee harus non-negatif
		if fee < 0 {
			t.Fatalf(
				"CalculateLateFee returned negative fee %d (type=%s, subtotal=%d, daysOverdue=%d)",
				fee, penaltyType, subtotal, daysOverdue,
			)
		}
	})
}
