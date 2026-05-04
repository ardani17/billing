// customer_bulk.go berisi business logic untuk bulk actions pada pelanggan.
// Mengimplementasikan BulkIsolir, BulkActivate, BulkNotify, BulkChangePackage,
// BulkEdit, BulkDelete pada CustomerUsecase.
package usecase

import (
	"context"
	"fmt"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// BulkIsolir mentransisikan status beberapa pelanggan ke isolir.
// Iterasi per customer ID → apply Isolir → collect successes/failures.
func (uc *CustomerUsecase) BulkIsolir(ctx context.Context, ids []string, actor ActorInfo) (*domain.BulkActionResult, error) {
	result := &domain.BulkActionResult{
		Total: len(ids),
	}

	for _, id := range ids {
		_, err := uc.Isolir(ctx, id, actor)
		if err != nil {
			result.FailureCount++
			result.Failures = append(result.Failures, domain.BulkFailure{
				CustomerID: id,
				Reason:     err.Error(),
			})
		} else {
			result.SuccessCount++
		}
	}

	return result, nil
}

// BulkActivate mentransisikan status beberapa pelanggan ke aktif.
// Iterasi per customer ID → apply Activate → collect successes/failures.
func (uc *CustomerUsecase) BulkActivate(ctx context.Context, ids []string, actor ActorInfo) (*domain.BulkActionResult, error) {
	result := &domain.BulkActionResult{
		Total: len(ids),
	}

	for _, id := range ids {
		_, err := uc.Activate(ctx, id, actor)
		if err != nil {
			result.FailureCount++
			result.Failures = append(result.Failures, domain.BulkFailure{
				CustomerID: id,
				Reason:     err.Error(),
			})
		} else {
			result.SuccessCount++
		}
	}

	return result, nil
}

// BulkNotify mengirim notifikasi ke beberapa pelanggan.
// Iterasi per customer ID → fetch customer → publish notification event → collect results.
func (uc *CustomerUsecase) BulkNotify(ctx context.Context, ids []string, templateID string, actor ActorInfo) (*domain.BulkActionResult, error) {
	result := &domain.BulkActionResult{
		Total: len(ids),
	}

	for _, id := range ids {
		customer, err := uc.customerRepo.GetByID(ctx, id)
		if err != nil {
			result.FailureCount++
			result.Failures = append(result.Failures, domain.BulkFailure{
				CustomerID: id,
				Reason:     err.Error(),
			})
			continue
		}

		if customer.DeletedAt != nil {
			result.FailureCount++
			result.Failures = append(result.Failures, domain.BulkFailure{
				CustomerID: id,
				Reason:     domain.ErrCustomerNotFound.Error(),
			})
			continue
		}

		// Publish notification event
		uc.publishEvent(customer.TenantID, "customer.notification", map[string]interface{}{
			"customer_id": customer.ID,
			"name":        customer.Name,
			"phone":       customer.Phone,
			"email":       customer.Email,
			"template_id": templateID,
		})

		// Write audit log
		uc.writeAuditLog(ctx, customer.TenantID, id, "customer.notified", actor, nil)

		result.SuccessCount++
	}

	return result, nil
}

// BulkChangePackage mengubah paket beberapa pelanggan.
// Iterasi per customer ID → apply ChangePackage → collect successes/failures.
func (uc *CustomerUsecase) BulkChangePackage(ctx context.Context, ids []string, packageID string, actor ActorInfo) (*domain.BulkActionResult, error) {
	result := &domain.BulkActionResult{
		Total: len(ids),
	}

	for _, id := range ids {
		_, err := uc.ChangePackage(ctx, id, packageID, actor)
		if err != nil {
			result.FailureCount++
			result.Failures = append(result.Failures, domain.BulkFailure{
				CustomerID: id,
				Reason:     err.Error(),
			})
		} else {
			result.SuccessCount++
		}
	}

	return result, nil
}

// BulkEdit mengubah field tertentu pada beberapa pelanggan.
// Iterasi per customer ID → update fields → write audit log → collect results.
func (uc *CustomerUsecase) BulkEdit(ctx context.Context, ids []string, fields domain.BulkEditFields, actor ActorInfo) (*domain.BulkActionResult, error) {
	result := &domain.BulkActionResult{
		Total: len(ids),
	}

	// Build fields map for repository
	fieldsMap := make(map[string]interface{})
	if fields.AreaID != "" {
		fieldsMap["area_id"] = fields.AreaID
	}
	if fields.DueDate != nil {
		fieldsMap["due_date"] = *fields.DueDate
	}
	if fields.Notes != "" {
		fieldsMap["notes"] = fields.Notes
	}

	if len(fieldsMap) == 0 {
		// Nothing to update, all succeed trivially
		result.SuccessCount = len(ids)
		return result, nil
	}

	bulkResults, err := uc.customerRepo.BulkUpdateFields(ctx, ids, fieldsMap)
	if err != nil {
		return nil, fmt.Errorf("usecase: gagal bulk edit: %w", err)
	}

	for _, br := range bulkResults {
		if br.Success {
			result.SuccessCount++
			// Write audit log per customer
			uc.writeAuditLog(ctx, "", br.ID, "customer.updated", actor, map[string]interface{}{
				"bulk_edit": fieldsMap,
			})
		} else {
			result.FailureCount++
			reason := "unknown error"
			if br.Error != nil {
				reason = br.Error.Error()
			}
			result.Failures = append(result.Failures, domain.BulkFailure{
				CustomerID: br.ID,
				Reason:     reason,
			})
		}
	}

	return result, nil
}

// BulkDelete menghapus beberapa pelanggan secara soft delete.
// Iterasi per customer ID → soft delete → write audit log → publish event → collect results.
func (uc *CustomerUsecase) BulkDelete(ctx context.Context, ids []string, actor ActorInfo) (*domain.BulkActionResult, error) {
	result := &domain.BulkActionResult{
		Total: len(ids),
	}

	// Fetch customers first for event publishing
	customers, err := uc.customerRepo.GetByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("usecase: gagal fetch customers for bulk delete: %w", err)
	}

	customerMap := make(map[string]*domain.Customer)
	for _, c := range customers {
		customerMap[c.ID] = c
	}

	bulkResults, err := uc.customerRepo.BulkSoftDelete(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("usecase: gagal bulk delete: %w", err)
	}

	for _, br := range bulkResults {
		if br.Success {
			result.SuccessCount++

			customer := customerMap[br.ID]
			tenantID := ""
			if customer != nil {
				tenantID = customer.TenantID
			}

			// Write audit log per customer
			uc.writeAuditLog(ctx, tenantID, br.ID, "customer.deleted", actor, nil)

			// Publish customer.terminated event
			if customer != nil {
				uc.publishEvent(customer.TenantID, "customer.terminated", domain.CustomerTerminatedPayload{
					CustomerID:       customer.ID,
					TenantID:         customer.TenantID,
					Name:             customer.Name,
					RouterID:         customer.RouterID,
					PPPoEUsername:    customer.PPPoEUsername,
					ConnectionMethod: string(customer.ConnectionMethod),
				})
			}
		} else {
			result.FailureCount++
			reason := "unknown error"
			if br.Error != nil {
				reason = br.Error.Error()
			}
			result.Failures = append(result.Failures, domain.BulkFailure{
				CustomerID: br.ID,
				Reason:     reason,
			})
		}
	}

	return result, nil
}
