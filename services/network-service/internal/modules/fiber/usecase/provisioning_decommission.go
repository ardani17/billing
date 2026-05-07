// Package usecase - implementasi DecommissionONT dan HandleCustomerTerminated.
// Menghapus ONT dari OLT via CLI, perbarui DB, audit log, terbitkan event.
package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// DecommissionONT menghapus ONT dari OLT dan perbarui DB.
// Alur: get ONT -> atur in_progress -> remove service-port -> remove ONT -> perbarui status -> audit -> event.
func (pm *provisioningManager) DecommissionONT(ctx context.Context, ontID string, performedBy string) error {
	if err := pm.ensureWriteEnabled(); err != nil {
		return err
	}

	ont, err := pm.ontRepo.GetByID(ctx, ontID)
	if err != nil {
		return err
	}

	// Set provisioning state ke in_progress
	if err := pm.ontRepo.UpdateStatus(ctx, ont.ID, string(ont.Status), string(domain.ProvisioningStateInProgress)); err != nil {
		return err
	}

	// Ambil OLT untuk membuat adapter
	olt, err := pm.oltRepo.GetByID(ctx, ont.OLTID)
	if err != nil {
		return err
	}

	adapter, err := pm.createAdapter(olt)
	if err != nil {
		_ = pm.ontRepo.UpdateStatus(ctx, ont.ID, string(ont.Status), string(domain.ProvisioningStateFailed))
		return domain.ErrDecommissionFailed
	}

	// Remove service-port terlebih dahulu
	vlanID := 0
	if ont.VLANID != nil {
		vlan, vErr := pm.vlanRepo.GetByID(ctx, *ont.VLANID)
		if vErr == nil {
			vlanID = vlan.VLANID
		}
	}

	spResult, err := adapter.RemoveServicePort(ctx, domain.RemoveServicePortParams{
		PONPortIndex: ont.PONPortIndex,
		ONTIndex:     ont.ONTIndex,
		VLANID:       vlanID,
	})
	if err != nil || (spResult != nil && !spResult.Success) {
		_ = pm.ontRepo.UpdateStatus(ctx, ont.ID, string(ont.Status), string(domain.ProvisioningStateFailed))
		if spResult != nil {
			pm.createAuditLog(ctx, ont.TenantID, olt.ID, &ont.ID, domain.AuditActionServicePortRemove, spResult, performedBy)
		}
		return domain.ErrDecommissionFailed
	}

	// Remove ONT dari OLT
	removeResult, err := adapter.RemoveONT(ctx, domain.RemoveONTParams{
		PONPortIndex: ont.PONPortIndex,
		ONTIndex:     ont.ONTIndex,
	})
	if err != nil || (removeResult != nil && !removeResult.Success) {
		_ = pm.ontRepo.UpdateStatus(ctx, ont.ID, string(ont.Status), string(domain.ProvisioningStateFailed))
		if removeResult != nil {
			pm.createAuditLog(ctx, ont.TenantID, olt.ID, &ont.ID, domain.AuditActionONTDecommission, removeResult, performedBy)
		}
		return domain.ErrDecommissionFailed
	}

	customerID := ""
	if ont.CustomerID != nil {
		customerID = *ont.CustomerID
	}

	// Perbarui status ke decommissioned, clear customer_id, atur last_decommissioned_at
	decommAt := time.Now()
	ont.Status = domain.ONTStatusDecommissioned
	ont.ProvisioningState = domain.ProvisioningStateCompleted
	ont.CustomerID = nil
	ont.LastDecommissionedAt = &decommAt
	ont.UpdatedAt = decommAt
	if _, err := pm.ontRepo.Update(ctx, ont); err != nil {
		log.Error().Err(err).Str("ont_id", ont.ID).Msg("gagal update status ONT setelah decommission")
	}

	// Gabungkan hasil untuk audit log
	combinedResult := &domain.ProvisioningResult{
		Success:      true,
		CommandsSent: append(spResult.CommandsSent, removeResult.CommandsSent...),
		Responses:    append(spResult.Responses, removeResult.Responses...),
	}
	pm.createAuditLog(ctx, ont.TenantID, olt.ID, &ont.ID, domain.AuditActionONTDecommission, combinedResult, performedBy)

	// Terbitkan event
	_ = pm.eventPub.PublishONTDecommissioned(ctx, domain.ONTDecommissionedPayload{
		CorrelationID: uuid.New().String(),
		ONTID:         ont.ID,
		SerialNumber:  ont.SerialNumber,
		CustomerID:    customerID,
		OLTID:         olt.ID,
		OLTName:       olt.Name,
		PONPortIndex:  ont.PONPortIndex,
		TenantID:      ont.TenantID,
	})

	return nil
}

// HandleCustomerTerminated memproses event customer.terminated untuk decommission ONT.
// Lookup ONT by customer_id, lalu jalankan decommission.
func (pm *provisioningManager) HandleCustomerTerminated(ctx context.Context, customerID, tenantID string) error {
	if err := pm.ensureWriteEnabled(); err != nil {
		return err
	}

	ont, err := pm.ontRepo.GetByCustomerID(ctx, customerID)
	if err != nil {
		if err == domain.ErrONTNotFound {
			// Customer tidak punya ONT, skip
			log.Info().Str("customer_id", customerID).Msg("customer terminated: tidak ada ONT terkait")
			return nil
		}
		return err
	}

	return pm.DecommissionONT(ctx, ont.ID, "system:customer_terminated")
}
