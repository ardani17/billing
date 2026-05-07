// Package usecase - implementasi RebootONT.
// Validasi status ONT, eksekusi CLI reboot command, audit log.
package usecase

import (
	"context"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// RebootONT mengirim perintah reboot ke ONT via OLT CLI.
// Hanya ONT dengan status "provisioned" yang boleh di-reboot.
func (pm *provisioningManager) RebootONT(ctx context.Context, ontID string, performedBy string) (*domain.ProvisioningResult, error) {
	if err := pm.ensureWriteEnabled(); err != nil {
		return nil, err
	}

	ont, err := pm.ontRepo.GetByID(ctx, ontID)
	if err != nil {
		return nil, err
	}

	// Guard: hanya ONT provisioned yang boleh di-reboot
	if ont.Status != domain.ONTStatusProvisioned {
		return nil, domain.ErrONTNotProvisioned
	}

	// Ambil OLT untuk membuat adapter
	olt, err := pm.oltRepo.GetByID(ctx, ont.OLTID)
	if err != nil {
		return nil, err
	}

	adapter, err := pm.createAdapter(olt)
	if err != nil {
		return nil, domain.ErrRebootFailed
	}

	// Eksekusi reboot command
	result, err := adapter.RebootONT(ctx, domain.RebootONTParams{
		PONPortIndex: ont.PONPortIndex,
		ONTIndex:     ont.ONTIndex,
	})
	if err != nil {
		failResult := &domain.ProvisioningResult{
			Success:      false,
			ErrorMessage: err.Error(),
		}
		pm.createAuditLog(ctx, ont.TenantID, olt.ID, &ont.ID, domain.AuditActionONTReboot, failResult, performedBy)
		return nil, domain.ErrRebootFailed
	}

	// Audit log
	pm.createAuditLog(ctx, ont.TenantID, olt.ID, &ont.ID, domain.AuditActionONTReboot, result, performedBy)

	return result, nil
}
