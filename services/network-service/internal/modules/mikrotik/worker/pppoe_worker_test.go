package worker

import (
	"testing"
	"time"

	"pgregory.net/rapid"
)

// =============================================================================
// =============================================================================

// **Memvalidasi: Kebutuhan 10.3**
func TestProperty_RetryBackoffSchedule(t *testing.T) {
	expectedDelays := map[int]time.Duration{
		0: 30 * time.Second,
		1: 60 * time.Second,
		2: 120 * time.Second,
		3: 300 * time.Second,
		4: 600 * time.Second,
	}

	t.Run("within_range", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			attempt := rapid.IntRange(0, 4).Draw(t, "attempt")

			delay := PPPoERetryDelay(attempt, nil, nil)
			expected := expectedDelays[attempt]

			if delay != expected {
				t.Errorf("PPPoERetryDelay(%d) = %v, want %v", attempt, delay, expected)
			}
		})
	})

	t.Run("beyond_range", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			// Buat attempt numbers >= 5 (up to a reasonable upper bound)
			attempt := rapid.IntRange(5, 100).Draw(t, "attempt")

			delay := PPPoERetryDelay(attempt, nil, nil)
			expected := 600 * time.Second

			if delay != expected {
				t.Errorf("PPPoERetryDelay(%d) = %v, want %v (last delay)", attempt, delay, expected)
			}
		})
	})
}

// =============================================================================
// Unit Tests: PPPoERetryDelays variable
// =============================================================================

// TestPPPoERetryDelays_HasExactlyFiveEntries verifies that PPPoERetryDelays
func TestPPPoERetryDelays_HasExactlyFiveEntries(t *testing.T) {
	if len(PPPoERetryDelays) != 5 {
		t.Fatalf("PPPoERetryDelays has %d entries, want 5", len(PPPoERetryDelays))
	}

	expected := []time.Duration{
		30 * time.Second,
		60 * time.Second,
		120 * time.Second,
		300 * time.Second,
		600 * time.Second,
	}

	for i, want := range expected {
		if PPPoERetryDelays[i] != want {
			t.Errorf("PPPoERetryDelays[%d] = %v, want %v", i, PPPoERetryDelays[i], want)
		}
	}
}

// =============================================================================
// Unit Tests: PPPoERetryDelay function
// =============================================================================

func TestPPPoERetryDelay_EachAttempt(t *testing.T) {
	tests := []struct {
		attempt int
		want    time.Duration
	}{
		{0, 30 * time.Second},
		{1, 60 * time.Second},
		{2, 120 * time.Second},
		{3, 300 * time.Second},
		{4, 600 * time.Second},
	}

	for _, tc := range tests {
		delay := PPPoERetryDelay(tc.attempt, nil, nil)
		if delay != tc.want {
			t.Errorf("PPPoERetryDelay(%d) = %v, want %v", tc.attempt, delay, tc.want)
		}
	}
}

// TestPPPoERetryDelay_BeyondMaxRetries verifies that untuk attempt >= 5,
func TestPPPoERetryDelay_BeyondMaxRetries(t *testing.T) {
	beyondAttempts := []int{5, 6, 10, 50, 100}
	want := 600 * time.Second

	for _, attempt := range beyondAttempts {
		delay := PPPoERetryDelay(attempt, nil, nil)
		if delay != want {
			t.Errorf("PPPoERetryDelay(%d) = %v, want %v", attempt, delay, want)
		}
	}
}

// =============================================================================
// Unit Tests: Event type constants
// =============================================================================

func TestEventTypeConstants(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{"EventCustomerActivated", EventCustomerActivated, "customer.activated"},
		{"EventCustomerIsolir", EventCustomerIsolir, "customer.isolir"},
		{"EventCustomerUnIsolir", EventCustomerUnIsolir, "customer.un_isolir"},
		{"EventCustomerSuspend", EventCustomerSuspend, "customer.suspend"},
		{"EventCustomerTerminated", EventCustomerTerminated, "customer.terminated"},
		{"EventPackageChanged", EventPackageChanged, "package.changed"},
	}

	for _, tc := range tests {
		if tc.got != tc.want {
			t.Errorf("%s = %q, want %q", tc.name, tc.got, tc.want)
		}
	}
}

func TestMaxRetries(t *testing.T) {
	if maxRetries != 5 {
		t.Errorf("maxRetries = %d, want 5", maxRetries)
	}
}
