package usecase

import (
	"context"
	"fmt"
	"testing"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"pgregory.net/rapid"
)

// Feature: customer-crud, Property 11: Bulk Action Result Invariant
// **Validates: Requirements 14.7**
//
// For any bulk action result, total == success_count + failure_count,
// total equals input IDs count, and failure_count equals length of failures array.
func TestProperty_BulkActionResultInvariant(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		customerRepo := newMockCustomerRepo()
		auditLogRepo := newMockAuditLogRepo()
		logger := newTestLogger()

		uc := NewCustomerUsecase(customerRepo, auditLogRepo, nil, logger)

		tenantID := genTenantID().Draw(t, "tenantID")
		actor := ActorInfo{
			ID:   genUUID().Draw(t, "actorID"),
			Name: "Test Actor",
		}

		ctx := context.Background()

		// Generate a mix of existing and non-existing customer IDs
		existingCount := rapid.IntRange(0, 5).Draw(t, "existingCount")
		nonExistingCount := rapid.IntRange(0, 5).Draw(t, "nonExistingCount")

		// Ensure at least one ID total
		if existingCount+nonExistingCount == 0 {
			existingCount = 1
		}

		var ids []string

		// Create existing customers with status aktif (so isolir can work)
		for i := 0; i < existingCount; i++ {
			req := genValidCreateRequest(t)
			req.Phone = genPhone().Draw(t, fmt.Sprintf("phone_%d", i))
			created, err := uc.Create(ctx, tenantID, req, actor)
			if err != nil {
				t.Fatalf("create failed: %v", err)
			}
			// Activate the customer (pending → aktif)
			_, err = uc.Activate(ctx, created.ID, actor)
			if err != nil {
				t.Fatalf("activate failed: %v", err)
			}
			ids = append(ids, created.ID)
		}

		// Add non-existing IDs
		for i := 0; i < nonExistingCount; i++ {
			ids = append(ids, genUUID().Draw(t, fmt.Sprintf("fakeID_%d", i)))
		}

		// Pick a random bulk action
		bulkAction := rapid.SampledFrom([]string{
			"isolir", "activate", "notify", "change_package", "edit", "delete",
		}).Draw(t, "bulkAction")

		var result *domain.BulkActionResult
		var bulkErr error

		switch bulkAction {
		case "isolir":
			result, bulkErr = uc.BulkIsolir(ctx, ids, actor)
		case "activate":
			// For activate, we need customers in isolir state
			// Re-isolir the existing ones first
			for _, id := range ids[:existingCount] {
				_, _ = uc.Isolir(ctx, id, actor)
			}
			result, bulkErr = uc.BulkActivate(ctx, ids, actor)
		case "notify":
			result, bulkErr = uc.BulkNotify(ctx, ids, "template-1", actor)
		case "change_package":
			newPkgID := genUUID().Draw(t, "bulkPkgID")
			result, bulkErr = uc.BulkChangePackage(ctx, ids, newPkgID, actor)
		case "edit":
			dueDate := rapid.IntRange(1, 28).Draw(t, "bulkDueDate")
			fields := domain.BulkEditFields{
				DueDate: &dueDate,
			}
			result, bulkErr = uc.BulkEdit(ctx, ids, fields, actor)
		case "delete":
			result, bulkErr = uc.BulkDelete(ctx, ids, actor)
		}

		if bulkErr != nil {
			t.Fatalf("bulk action %q failed: %v", bulkAction, bulkErr)
		}

		// Property 11a: total == success_count + failure_count
		if result.Total != result.SuccessCount+result.FailureCount {
			t.Fatalf("total (%d) != success_count (%d) + failure_count (%d)",
				result.Total, result.SuccessCount, result.FailureCount)
		}

		// Property 11b: total equals input IDs count
		if result.Total != len(ids) {
			t.Fatalf("total (%d) != input IDs count (%d)", result.Total, len(ids))
		}

		// Property 11c: failure_count equals length of failures array
		if result.FailureCount != len(result.Failures) {
			t.Fatalf("failure_count (%d) != len(failures) (%d)",
				result.FailureCount, len(result.Failures))
		}
	})
}
