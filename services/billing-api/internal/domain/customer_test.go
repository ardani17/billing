package domain

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// Feature: customer-crud, Property 1: Customer ID Generation Format
// **Validates: Requirements 4.1, 4.2**
//
// For any positive integer sequence number, GenerateCustomerID(seq) SHALL produce
// a string matching the pattern PLG-{zero-padded-seq} where the sequence is
// zero-padded to at least 3 digits (e.g., PLG-001 for seq=0, PLG-999 for seq=998,
// PLG-1000 for seq=999), and parsing the numeric suffix back SHALL yield the
// original sequence number (seq+1).
func TestProperty_CustomerIDGenerationFormat(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a non-negative integer as the "lastSeq" input
		lastSeq := rapid.IntRange(0, 100000).Draw(t, "lastSeq")

		result := GenerateCustomerID(lastSeq)
		next := lastSeq + 1

		// Property 1a: Must start with "PLG-"
		if !strings.HasPrefix(result, "PLG-") {
			t.Fatalf("expected prefix PLG-, got %q", result)
		}

		// Extract the numeric suffix
		suffix := strings.TrimPrefix(result, "PLG-")

		// Property 1b: Suffix must be at least 3 digits
		if len(suffix) < 3 {
			t.Fatalf("expected suffix to be at least 3 digits, got %q (len=%d)", suffix, len(suffix))
		}

		// Property 1c: For next < 1000, suffix must be exactly 3 digits (zero-padded)
		if next < 1000 {
			expected := fmt.Sprintf("%03d", next)
			if suffix != expected {
				t.Fatalf("for next=%d, expected suffix %q, got %q", next, expected, suffix)
			}
		}

		// Property 1d: Parsing the numeric suffix back yields the original next value
		parsed, err := strconv.Atoi(suffix)
		if err != nil {
			t.Fatalf("failed to parse suffix %q as integer: %v", suffix, err)
		}
		if parsed != next {
			t.Fatalf("round-trip failed: lastSeq=%d, next=%d, parsed=%d", lastSeq, next, parsed)
		}
	})
}

// Feature: customer-crud, Property 5: State Machine Determinism and Completeness
// **Validates: Requirements 11.3, 11.4, 23.1, 23.2, 23.3**
//
// For any pair of CustomerStatus values (current, target), CanTransition returns
// true iff target is in ValidTransitions[current]. Transition returns target on
// valid transitions, and returns error with allowed targets on invalid transitions.
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

		// Determine expected result from ValidTransitions
		expectedValid := false
		for _, allowed := range ValidTransitions[current] {
			if allowed == target {
				expectedValid = true
				break
			}
		}

		// Property 5a: CanTransition returns true iff target is in ValidTransitions[current]
		canResult := CanTransition(current, target)
		if canResult != expectedValid {
			t.Fatalf("CanTransition(%s, %s) = %v, expected %v", current, target, canResult, expectedValid)
		}

		// Property 5b: Transition returns target on valid transitions
		newStatus, err := Transition(current, target)
		if expectedValid {
			if err != nil {
				t.Fatalf("Transition(%s, %s) returned unexpected error: %v", current, target, err)
			}
			if newStatus != target {
				t.Fatalf("Transition(%s, %s) returned %s, expected %s", current, target, newStatus, target)
			}
		} else {
			// Property 5c: Transition returns error on invalid transitions
			if err == nil {
				t.Fatalf("Transition(%s, %s) expected error, got nil", current, target)
			}

			// The returned status should be the current status (unchanged)
			if newStatus != current {
				t.Fatalf("Transition(%s, %s) returned status %s on error, expected %s (unchanged)", current, target, newStatus, current)
			}

			// Error message should contain allowed targets
			allowedTargets := AllowedTargets(current)
			for _, at := range allowedTargets {
				if !strings.Contains(err.Error(), string(at)) {
					t.Fatalf("error message %q does not contain allowed target %q", err.Error(), at)
				}
			}
		}

		// Property 5d: AllowedTargets returns the same set as ValidTransitions[current]
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
