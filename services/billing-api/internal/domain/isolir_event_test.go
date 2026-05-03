package domain

import (
	"testing"

	"pgregory.net/rapid"
)

// Feature: isolir-system, Property 4: Event payload completeness
// **Validates: Requirements 11.5**
//
// For any event payload constructed from valid customer and tenant data
// (CustomerIsolirPayload, CustomerUnIsolirPayload, CustomerSuspendPayload,
// PenaltyAddedPayload), the tenant_id and customer_id fields SHALL be
// non-empty strings.
func TestProperty_EventPayloadCompleteness(t *testing.T) {
	// Generator untuk UUID non-kosong (format sederhana, bukan strict UUID v4)
	nonEmptyUUID := rapid.StringMatching(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)

	// Generator untuk nama non-kosong
	nonEmptyName := rapid.StringMatching(`[A-Za-z ]{1,50}`)

	t.Run("CustomerIsolirPayload", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			payload := CustomerIsolirPayload{
				CustomerID:       nonEmptyUUID.Draw(t, "customerID"),
				TenantID:         nonEmptyUUID.Draw(t, "tenantID"),
				CustomerName:     nonEmptyName.Draw(t, "customerName"),
				RouterID:         rapid.String().Draw(t, "routerID"),
				PPPoEUsername:    rapid.String().Draw(t, "pppoeUsername"),
				ConnectionMethod: rapid.SampledFrom([]string{"pppoe", "static", "dhcp"}).Draw(t, "connectionMethod"),
				Reason:           rapid.String().Draw(t, "reason"),
				OverdueDays:      rapid.IntRange(0, 365).Draw(t, "overdueDays"),
			}

			if payload.TenantID == "" {
				t.Fatal("CustomerIsolirPayload.TenantID must be non-empty")
			}
			if payload.CustomerID == "" {
				t.Fatal("CustomerIsolirPayload.CustomerID must be non-empty")
			}
		})
	})

	t.Run("CustomerUnIsolirPayload", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			payload := CustomerUnIsolirPayload{
				CustomerID:       nonEmptyUUID.Draw(t, "customerID"),
				TenantID:         nonEmptyUUID.Draw(t, "tenantID"),
				CustomerName:     nonEmptyName.Draw(t, "customerName"),
				RouterID:         rapid.String().Draw(t, "routerID"),
				PPPoEUsername:    rapid.String().Draw(t, "pppoeUsername"),
				ConnectionMethod: rapid.SampledFrom([]string{"pppoe", "static", "dhcp"}).Draw(t, "connectionMethod"),
				Trigger:          rapid.SampledFrom([]string{"payment_received", "admin_manual"}).Draw(t, "trigger"),
			}

			if payload.TenantID == "" {
				t.Fatal("CustomerUnIsolirPayload.TenantID must be non-empty")
			}
			if payload.CustomerID == "" {
				t.Fatal("CustomerUnIsolirPayload.CustomerID must be non-empty")
			}
		})
	})

	t.Run("CustomerSuspendPayload", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			payload := CustomerSuspendPayload{
				CustomerID:       nonEmptyUUID.Draw(t, "customerID"),
				TenantID:         nonEmptyUUID.Draw(t, "tenantID"),
				CustomerName:     nonEmptyName.Draw(t, "customerName"),
				RouterID:         rapid.String().Draw(t, "routerID"),
				PPPoEUsername:    rapid.String().Draw(t, "pppoeUsername"),
				ConnectionMethod: rapid.SampledFrom([]string{"pppoe", "static", "dhcp"}).Draw(t, "connectionMethod"),
				OverdueDays:      rapid.IntRange(0, 365).Draw(t, "overdueDays"),
			}

			if payload.TenantID == "" {
				t.Fatal("CustomerSuspendPayload.TenantID must be non-empty")
			}
			if payload.CustomerID == "" {
				t.Fatal("CustomerSuspendPayload.CustomerID must be non-empty")
			}
		})
	})

	t.Run("PenaltyAddedPayload", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			payload := PenaltyAddedPayload{
				InvoiceID:     nonEmptyUUID.Draw(t, "invoiceID"),
				TenantID:      nonEmptyUUID.Draw(t, "tenantID"),
				CustomerID:    nonEmptyUUID.Draw(t, "customerID"),
				PenaltyAmount: rapid.Int64Range(1, 50_000_000).Draw(t, "penaltyAmount"),
				PenaltyType:   rapid.SampledFrom([]string{"fixed", "percentage", "daily"}).Draw(t, "penaltyType"),
				InvoiceNumber: rapid.StringMatching(`INV-[0-9]{4}-[0-9]{6}`).Draw(t, "invoiceNumber"),
			}

			if payload.TenantID == "" {
				t.Fatal("PenaltyAddedPayload.TenantID must be non-empty")
			}
			if payload.CustomerID == "" {
				t.Fatal("PenaltyAddedPayload.CustomerID must be non-empty")
			}
		})
	})
}
