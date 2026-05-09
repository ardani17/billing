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
// Alur: ambil customer -> validasi transition (aktif -> isolir) via domain.CanTransition ->
// perbarui status -> tulis audit log -> terbitkan customer.isolated event.
func (uc *CustomerUsecase) Isolir(ctx context.Context, id string, actor ActorInfo) (*domain.Customer, error) {
	// Ambil existing customer
	customer, err := uc.customerRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if customer.DeletedAt != nil {
		return nil, domain.ErrCustomerNotFound
	}

	// Validasi transition via domain state machine
	newStatus, err := domain.Transition(customer.Status, domain.CustomerStatusIsolir)
	if err != nil {
		return nil, err
	}

	// Perbarui status in database
	updated, err := uc.customerRepo.UpdateStatus(ctx, id, newStatus)
	if err != nil {
		return nil, fmt.Errorf("usecase: gagal update status ke isolir: %w", err)
	}

	// Tulis audit log
	changes := map[string]interface{}{
		"status": map[string]interface{}{
			"old": string(customer.Status),
			"new": string(newStatus),
		},
	}
	uc.writeAuditLog(ctx, customer.TenantID, id, "customer.status_changed", actor, changes)

	if uc.mikrotikEnabled(ctx, customer.TenantID) {
		uc.publishEvent(customer.TenantID, domain.TaskCustomerIsolir, domain.CustomerIsolirPayload{
			CustomerID:       customer.ID,
			TenantID:         customer.TenantID,
			CustomerName:     customer.Name,
			RouterID:         customer.RouterID,
			PPPoEUsername:    customer.PPPoEUsername,
			ConnectionMethod: string(customer.ConnectionMethod),
			Reason:           "admin_manual",
		})
	}

	return updated, nil
}

// Activate mentransisikan status pelanggan dari pending/isolir/suspend ke aktif.
// Alur: ambil customer -> validasi transition -> perbarui status -> tulis audit log ->
func (uc *CustomerUsecase) Activate(ctx context.Context, id string, actor ActorInfo) (*domain.Customer, error) {
	// Ambil existing customer
	customer, err := uc.customerRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if customer.DeletedAt != nil {
		return nil, domain.ErrCustomerNotFound
	}

	previousStatus := customer.Status

	// Validasi transition via domain state machine
	newStatus, err := domain.Transition(customer.Status, domain.CustomerStatusAktif)
	if err != nil {
		return nil, err
	}

	// Perbarui status in database
	updated, err := uc.customerRepo.UpdateStatus(ctx, id, newStatus)
	if err != nil {
		return nil, fmt.Errorf("usecase: gagal update status ke aktif: %w", err)
	}

	// Tulis audit log
	changes := map[string]interface{}{
		"status": map[string]interface{}{
			"old": string(previousStatus),
			"new": string(newStatus),
		},
	}
	uc.writeAuditLog(ctx, customer.TenantID, id, "customer.status_changed", actor, changes)

	// Terbitkan event: unblocked jika from isolir, activated otherwise
	if previousStatus == domain.CustomerStatusIsolir && uc.mikrotikEnabled(ctx, customer.TenantID) {
		uc.publishEvent(customer.TenantID, domain.TaskCustomerUnIsolir, domain.CustomerUnIsolirPayload{
			CustomerID:       customer.ID,
			TenantID:         customer.TenantID,
			CustomerName:     customer.Name,
			RouterID:         customer.RouterID,
			PPPoEUsername:    customer.PPPoEUsername,
			ConnectionMethod: string(customer.ConnectionMethod),
			Trigger:          "admin_manual",
		})
	} else if uc.mikrotikEnabled(ctx, customer.TenantID) {
		profileName, downloadMbps, uploadMbps, addressPool := uc.packageNetworkFields(ctx, customer.PackageID)
		uc.publishEvent(customer.TenantID, "customer.activated", domain.CustomerActivatedPayload{
			CustomerID:          customer.ID,
			TenantID:            customer.TenantID,
			Name:                customer.Name,
			PackageID:           customer.PackageID,
			ConnectionMethod:    string(customer.ConnectionMethod),
			PPPoEUsername:       customer.PPPoEUsername,
			PPPoEPassword:       customer.PPPoEPassword,
			RouterID:            customer.RouterID,
			MikrotikProfileName: profileName,
			DownloadMbps:        downloadMbps,
			UploadMbps:          uploadMbps,
			AddressPool:         addressPool,
		})
	}

	return updated, nil
}

// ChangePackage mengubah paket pelanggan.
// tulis audit log -> terbitkan package.changed event.
func (uc *CustomerUsecase) ChangePackage(ctx context.Context, id string, packageID string, actor ActorInfo) (*domain.Customer, error) {
	// Ambil existing customer
	customer, err := uc.customerRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if customer.DeletedAt != nil {
		return nil, domain.ErrCustomerNotFound
	}

	// Validasi package_id differs from current
	if customer.PackageID == packageID {
		return nil, domain.ErrSamePackage
	}

	oldPackageID := customer.PackageID

	// Perbarui package in database
	updated, err := uc.customerRepo.UpdatePackage(ctx, id, packageID)
	if err != nil {
		return nil, fmt.Errorf("usecase: gagal update package: %w", err)
	}

	// Tulis audit log
	changes := map[string]interface{}{
		"package_id": map[string]interface{}{
			"old": oldPackageID,
			"new": packageID,
		},
	}
	uc.writeAuditLog(ctx, customer.TenantID, id, "customer.package_changed", actor, changes)

	profileName, downloadMbps, uploadMbps, addressPool := uc.packageNetworkFields(ctx, packageID)

	if uc.mikrotikEnabled(ctx, customer.TenantID) {
		uc.publishEvent(customer.TenantID, "package.changed", domain.PackageChangedPayload{
			CustomerID:          customer.ID,
			TenantID:            customer.TenantID,
			OldPackageID:        oldPackageID,
			NewPackageID:        packageID,
			ConnectionMethod:    string(customer.ConnectionMethod),
			RouterID:            customer.RouterID,
			MikrotikProfileName: profileName,
			DownloadMbps:        downloadMbps,
			UploadMbps:          uploadMbps,
			AddressPool:         addressPool,
		})
	}

	return updated, nil
}
