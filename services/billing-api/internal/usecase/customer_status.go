// customer_status.go berisi business logic untuk transisi status pelanggan
// dan perubahan paket. Mengimplementasikan Isolir, Activate, ChangePackage
// pada CustomerUsecase.
package usecase

import (
	"context"
	"fmt"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// Isolir mentransisikan status pelanggan dari aktif ke isolir.
// Flow: fetch customer → validate transition (aktif → isolir) via domain.CanTransition →
// update status → write audit log → publish customer.isolated event.
func (uc *CustomerUsecase) Isolir(ctx context.Context, id string, actor ActorInfo) (*domain.Customer, error) {
	// Fetch existing customer
	customer, err := uc.customerRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if customer.DeletedAt != nil {
		return nil, domain.ErrCustomerNotFound
	}

	// Validate transition via domain state machine
	newStatus, err := domain.Transition(customer.Status, domain.CustomerStatusIsolir)
	if err != nil {
		return nil, err
	}

	// Update status in database
	updated, err := uc.customerRepo.UpdateStatus(ctx, id, newStatus)
	if err != nil {
		return nil, fmt.Errorf("usecase: gagal update status ke isolir: %w", err)
	}

	// Write audit log
	changes := map[string]interface{}{
		"status": map[string]interface{}{
			"old": string(customer.Status),
			"new": string(newStatus),
		},
	}
	uc.writeAuditLog(ctx, customer.TenantID, id, "customer.status_changed", actor, changes)

	// Publish customer.isolated event
	uc.publishEvent(customer.TenantID, "customer.isolated", domain.CustomerIsolatedPayload{
		CustomerID:    customer.ID,
		Name:          customer.Name,
		RouterID:      customer.RouterID,
		PPPoEUsername: customer.PPPoEUsername,
	})

	return updated, nil
}

// Activate mentransisikan status pelanggan dari pending/isolir/suspend ke aktif.
// Flow: fetch customer → validate transition → update status → write audit log →
// publish customer.activated or customer.unblocked event (unblocked if from isolir).
func (uc *CustomerUsecase) Activate(ctx context.Context, id string, actor ActorInfo) (*domain.Customer, error) {
	// Fetch existing customer
	customer, err := uc.customerRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if customer.DeletedAt != nil {
		return nil, domain.ErrCustomerNotFound
	}

	previousStatus := customer.Status

	// Validate transition via domain state machine
	newStatus, err := domain.Transition(customer.Status, domain.CustomerStatusAktif)
	if err != nil {
		return nil, err
	}

	// Update status in database
	updated, err := uc.customerRepo.UpdateStatus(ctx, id, newStatus)
	if err != nil {
		return nil, fmt.Errorf("usecase: gagal update status ke aktif: %w", err)
	}

	// Write audit log
	changes := map[string]interface{}{
		"status": map[string]interface{}{
			"old": string(previousStatus),
			"new": string(newStatus),
		},
	}
	uc.writeAuditLog(ctx, customer.TenantID, id, "customer.status_changed", actor, changes)

	// Publish event: unblocked if from isolir, activated otherwise
	if previousStatus == domain.CustomerStatusIsolir {
		uc.publishEvent(customer.TenantID, "customer.unblocked", domain.CustomerUnblockedPayload{
			CustomerID:    customer.ID,
			Name:          customer.Name,
			RouterID:      customer.RouterID,
			PPPoEUsername: customer.PPPoEUsername,
		})
	} else {
		uc.publishEvent(customer.TenantID, "customer.activated", domain.CustomerActivatedPayload{
			CustomerID:       customer.ID,
			Name:             customer.Name,
			PackageID:        customer.PackageID,
			ConnectionMethod: string(customer.ConnectionMethod),
			PPPoEUsername:    customer.PPPoEUsername,
			PPPoEPassword:    customer.PPPoEPassword,
			RouterID:         customer.RouterID,
		})
	}

	return updated, nil
}

// ChangePackage mengubah paket pelanggan.
// Flow: fetch customer → validate package_id differs → update package →
// write audit log → publish package.changed event.
func (uc *CustomerUsecase) ChangePackage(ctx context.Context, id string, packageID string, actor ActorInfo) (*domain.Customer, error) {
	// Fetch existing customer
	customer, err := uc.customerRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if customer.DeletedAt != nil {
		return nil, domain.ErrCustomerNotFound
	}

	// Validate package_id differs from current
	if customer.PackageID == packageID {
		return nil, domain.ErrSamePackage
	}

	oldPackageID := customer.PackageID

	// Update package in database
	updated, err := uc.customerRepo.UpdatePackage(ctx, id, packageID)
	if err != nil {
		return nil, fmt.Errorf("usecase: gagal update package: %w", err)
	}

	// Write audit log
	changes := map[string]interface{}{
		"package_id": map[string]interface{}{
			"old": oldPackageID,
			"new": packageID,
		},
	}
	uc.writeAuditLog(ctx, customer.TenantID, id, "customer.package_changed", actor, changes)

	// Publish package.changed event
	uc.publishEvent(customer.TenantID, "package.changed", domain.PackageChangedPayload{
		CustomerID:       customer.ID,
		OldPackageID:     oldPackageID,
		NewPackageID:     packageID,
		ConnectionMethod: string(customer.ConnectionMethod),
		RouterID:         customer.RouterID,
	})

	return updated, nil
}
