package domain

import (
	"testing"

	"pgregory.net/rapid"
)

// Feature: invoice-generation, Property 9: Prorate Calculation Correctness
// **Validates: Requirements 17.1, 18.1, 18.2, 18.7**
//
// For any monthly_price (positive), remaining_days (1-30):
// - CalculateProrate(monthly_price, remaining_days) returns RoundUpTo500(monthly_price * remaining_days / 30) and is non-negative.
// - For upgrade (new > old): charge = RoundUpTo500((new - old) * remaining_days / 30) is non-negative.
// - For downgrade (old > new): credit = RoundDownTo500((old - new) * remaining_days / 30) is non-negative.
func TestProperty_ProrateCalculationCorrectness(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		monthlyPrice := rapid.Int64Range(1, 100_000_000).Draw(t, "monthlyPrice")
		remainingDays := rapid.IntRange(1, 30).Draw(t, "remainingDays")

		// Property 9a: CalculateProrate matches RoundUpTo500(monthlyPrice * remainingDays / 30)
		result := CalculateProrate(monthlyPrice, remainingDays)
		expected := RoundUpTo500(monthlyPrice * int64(remainingDays) / 30)

		if result != expected {
			t.Fatalf(
				"CalculateProrate(%d, %d) = %d, expected RoundUpTo500(%d * %d / 30) = %d",
				monthlyPrice, remainingDays, result, monthlyPrice, remainingDays, expected,
			)
		}

		// Property 9b: Prorate result is non-negative
		if result < 0 {
			t.Fatalf(
				"CalculateProrate(%d, %d) = %d, expected non-negative",
				monthlyPrice, remainingDays, result,
			)
		}

		// Property 9c: Upgrade prorate (new > old) — charge is non-negative
		newPrice := rapid.Int64Range(1, 100_000_000).Draw(t, "newPrice")
		oldPrice := rapid.Int64Range(0, newPrice-1).Draw(t, "oldPrice")

		diff := newPrice - oldPrice
		upgradeCharge := RoundUpTo500(diff * int64(remainingDays) / 30)

		if upgradeCharge < 0 {
			t.Fatalf(
				"Upgrade prorate charge for new=%d, old=%d, days=%d is %d, expected non-negative",
				newPrice, oldPrice, remainingDays, upgradeCharge,
			)
		}

		// Verify CalculateProrate with the price difference matches the expected upgrade charge
		calcUpgrade := CalculateProrate(diff, remainingDays)
		if calcUpgrade != upgradeCharge {
			t.Fatalf(
				"CalculateProrate(%d, %d) = %d, expected upgrade charge %d",
				diff, remainingDays, calcUpgrade, upgradeCharge,
			)
		}

		// Property 9d: Downgrade prorate (old > new) — credit is non-negative
		downOld := rapid.Int64Range(2, 100_000_000).Draw(t, "downOld")
		downNew := rapid.Int64Range(1, downOld-1).Draw(t, "downNew")

		downDiff := downOld - downNew
		downgradeCredit := RoundDownTo500(downDiff * int64(remainingDays) / 30)

		if downgradeCredit < 0 {
			t.Fatalf(
				"Downgrade prorate credit for old=%d, new=%d, days=%d is %d, expected non-negative",
				downOld, downNew, remainingDays, downgradeCredit,
			)
		}

		// Verify CalculateProrateCredit with the price difference matches the expected downgrade credit
		calcDowngrade := CalculateProrateCredit(downDiff, remainingDays)
		if calcDowngrade != downgradeCredit {
			t.Fatalf(
				"CalculateProrateCredit(%d, %d) = %d, expected downgrade credit %d",
				downDiff, remainingDays, calcDowngrade, downgradeCredit,
			)
		}
	})
}

// Feature: invoice-generation, Property 10: Rounding Functions Correctness
// **Validates: Requirements 17.3, 18.5**
//
// For any non-negative integer amount:
// - RoundUpTo500(amount) >= amount, is a multiple of 500, and RoundUpTo500(amount) - amount < 500.
// - RoundDownTo500(amount) <= amount, is a multiple of 500, and amount - RoundDownTo500(amount) < 500.
// - Both are idempotent.
func TestProperty_RoundingFunctionsCorrectness(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		amount := rapid.Int64Range(0, 1_000_000_000).Draw(t, "amount")

		// --- RoundUpTo500 properties ---
		up := RoundUpTo500(amount)

		// Property 10a: RoundUpTo500(amount) >= amount
		if up < amount {
			t.Fatalf("RoundUpTo500(%d) = %d, expected >= %d", amount, up, amount)
		}

		// Property 10b: RoundUpTo500(amount) is a multiple of 500
		if amount > 0 && up%500 != 0 {
			t.Fatalf("RoundUpTo500(%d) = %d, expected multiple of 500", amount, up)
		}

		// Property 10c: RoundUpTo500(amount) - amount < 500
		if up-amount >= 500 {
			t.Fatalf("RoundUpTo500(%d) = %d, difference %d >= 500", amount, up, up-amount)
		}

		// Property 10d: RoundUpTo500 is idempotent
		upUp := RoundUpTo500(up)
		if upUp != up {
			t.Fatalf("RoundUpTo500 not idempotent: RoundUpTo500(%d) = %d, RoundUpTo500(%d) = %d", amount, up, up, upUp)
		}

		// --- RoundDownTo500 properties ---
		down := RoundDownTo500(amount)

		// Property 10e: RoundDownTo500(amount) <= amount
		if down > amount {
			t.Fatalf("RoundDownTo500(%d) = %d, expected <= %d", amount, down, amount)
		}

		// Property 10f: RoundDownTo500(amount) is a multiple of 500
		if amount > 0 && down%500 != 0 {
			t.Fatalf("RoundDownTo500(%d) = %d, expected multiple of 500", amount, down)
		}

		// Property 10g: amount - RoundDownTo500(amount) < 500
		if amount-down >= 500 {
			t.Fatalf("RoundDownTo500(%d) = %d, difference %d >= 500", amount, down, amount-down)
		}

		// Property 10h: RoundDownTo500 is idempotent
		downDown := RoundDownTo500(down)
		if downDown != down {
			t.Fatalf("RoundDownTo500 not idempotent: RoundDownTo500(%d) = %d, RoundDownTo500(%d) = %d", amount, down, down, downDown)
		}
	})
}
