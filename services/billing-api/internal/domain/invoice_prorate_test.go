package domain

import (
	"testing"

	"pgregory.net/rapid"
)

// **Memvalidasi: Kebutuhan 17.1, 18.1, 18.2, 18.7**
//
// - For upgrade (new > old): charge = RoundUpTo500((new - old) * remaining_days / 30) is non-negative.
// - For downgrade (old > new): credit = RoundDownTo500((old - new) * remaining_days / 30) is non-negative.
func TestProperty_ProrateCalculationCorrectness(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		monthlyPrice := rapid.Int64Range(1, 100_000_000).Draw(t, "monthlyPrice")
		remainingDays := rapid.IntRange(1, 30).Draw(t, "remainingDays")

		result := CalculateProrate(monthlyPrice, remainingDays)
		expected := RoundUpTo500(monthlyPrice * int64(remainingDays) / 30)

		if result != expected {
			t.Fatalf(
				"CalculateProrate(%d, %d) = %d, expected RoundUpTo500(%d * %d / 30) = %d",
				monthlyPrice, remainingDays, result, monthlyPrice, remainingDays, expected,
			)
		}

		if result < 0 {
			t.Fatalf(
				"CalculateProrate(%d, %d) = %d, expected non-negative",
				monthlyPrice, remainingDays, result,
			)
		}

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

		calcUpgrade := CalculateProrate(diff, remainingDays)
		if calcUpgrade != upgradeCharge {
			t.Fatalf(
				"CalculateProrate(%d, %d) = %d, expected upgrade charge %d",
				diff, remainingDays, calcUpgrade, upgradeCharge,
			)
		}

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

		calcDowngrade := CalculateProrateCredit(downDiff, remainingDays)
		if calcDowngrade != downgradeCredit {
			t.Fatalf(
				"CalculateProrateCredit(%d, %d) = %d, expected downgrade credit %d",
				downDiff, remainingDays, calcDowngrade, downgradeCredit,
			)
		}
	})
}

// **Memvalidasi: Kebutuhan 17.3, 18.5**
//
// - Both are idempotent.
func TestProperty_RoundingFunctionsCorrectness(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		amount := rapid.Int64Range(0, 1_000_000_000).Draw(t, "amount")

		// --- RoundUpTo500 properties ---
		up := RoundUpTo500(amount)

		if up < amount {
			t.Fatalf("RoundUpTo500(%d) = %d, expected >= %d", amount, up, amount)
		}

		if amount > 0 && up%500 != 0 {
			t.Fatalf("RoundUpTo500(%d) = %d, expected multiple of 500", amount, up)
		}

		if up-amount >= 500 {
			t.Fatalf("RoundUpTo500(%d) = %d, difference %d >= 500", amount, up, up-amount)
		}

		upUp := RoundUpTo500(up)
		if upUp != up {
			t.Fatalf("RoundUpTo500 not idempotent: RoundUpTo500(%d) = %d, RoundUpTo500(%d) = %d", amount, up, up, upUp)
		}

		// --- RoundDownTo500 properties ---
		down := RoundDownTo500(amount)

		if down > amount {
			t.Fatalf("RoundDownTo500(%d) = %d, expected <= %d", amount, down, amount)
		}

		if amount > 0 && down%500 != 0 {
			t.Fatalf("RoundDownTo500(%d) = %d, expected multiple of 500", amount, down)
		}

		if amount-down >= 500 {
			t.Fatalf("RoundDownTo500(%d) = %d, difference %d >= 500", amount, down, amount-down)
		}

		downDown := RoundDownTo500(down)
		if downDown != down {
			t.Fatalf("RoundDownTo500 not idempotent: RoundDownTo500(%d) = %d, RoundDownTo500(%d) = %d", amount, down, down, downDown)
		}
	})
}
