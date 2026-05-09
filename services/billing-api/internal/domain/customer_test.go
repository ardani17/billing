package domain

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// **Memvalidasi: Kebutuhan 4.1, 4.2**
//
// zero-padded to at least 3 digits (e.g., PLG-001 untuk seq=0, PLG-999 untuk seq=998,
// original sequence number (seq+1).
func TestProperty_CustomerIDGenerationFormat(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		lastSeq := rapid.IntRange(0, 100000).Draw(t, "lastSeq")

		result := GenerateCustomerID(lastSeq)
		next := lastSeq + 1

		if !strings.HasPrefix(result, "PLG-") {
			t.Fatalf("expected prefix PLG-, got %q", result)
		}

		suffix := strings.TrimPrefix(result, "PLG-")

		if len(suffix) < 3 {
			t.Fatalf("expected suffix to be at least 3 digits, got %q (len=%d)", suffix, len(suffix))
		}

		if next < 1000 {
			expected := fmt.Sprintf("%03d", next)
			if suffix != expected {
				t.Fatalf("for next=%d, expected suffix %q, got %q", next, expected, suffix)
			}
		}

		parsed, err := strconv.Atoi(suffix)
		if err != nil {
			t.Fatalf("failed to parse suffix %q as integer: %v", suffix, err)
		}
		if parsed != next {
			t.Fatalf("round-trip failed: lastSeq=%d, next=%d, parsed=%d", lastSeq, next, parsed)
		}
	})
}

// **Memvalidasi: Kebutuhan 11.3, 11.4, 23.1, 23.2, 23.3**
func TestProperty_StateMachineDeterminism(t *testing.T) {
	allStatuses := []CustomerStatus{
		CustomerStatusPending,
		CustomerStatusAktif,
		CustomerStatusIsolir,
		CustomerStatusSuspend,
		CustomerStatusBerhenti,
	}

	rapid.Check(t, func(t *rapid.T) {
		current := rapid.SampledFrom(allStatuses).Draw(t, "current")
		target := rapid.SampledFrom(allStatuses).Draw(t, "target")

		expectedValid := false
		for _, allowed := range ValidTransitions[current] {
			if allowed == target {
				expectedValid = true
				break
			}
		}

		canResult := CanTransition(current, target)
		if canResult != expectedValid {
			t.Fatalf("CanTransition(%s, %s) = %v, expected %v", current, target, canResult, expectedValid)
		}

		newStatus, err := Transition(current, target)
		if expectedValid {
			if err != nil {
				t.Fatalf("Transition(%s, %s) returned unexpected error: %v", current, target, err)
			}
			if newStatus != target {
				t.Fatalf("Transition(%s, %s) returned %s, expected %s", current, target, newStatus, target)
			}
		} else {
			if err == nil {
				t.Fatalf("Transition(%s, %s) expected error, got nil", current, target)
			}

			if newStatus != current {
				t.Fatalf("Transition(%s, %s) returned status %s on error, expected %s (unchanged)", current, target, newStatus, current)
			}

			allowedTargets := AllowedTargets(current)
			for _, at := range allowedTargets {
				if !strings.Contains(err.Error(), string(at)) {
					t.Fatalf("error message %q does not contain allowed target %q", err.Error(), at)
				}
			}
		}

		allowedTargets := AllowedTargets(current)
		expectedTargets := ValidTransitions[current]
		if len(allowedTargets) != len(expectedTargets) {
			t.Fatalf("AllowedTargets(%s) returned %d targets, expected %d", current, len(allowedTargets), len(expectedTargets))
		}
		for i, at := range allowedTargets {
			if at != expectedTargets[i] {
				t.Fatalf("AllowedTargets(%s)[%d] = %s, expected %s", current, i, at, expectedTargets[i])
			}
		}
	})
}
