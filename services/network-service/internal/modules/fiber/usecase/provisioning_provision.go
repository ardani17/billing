// Package usecase - implementasi ProvisionONT untuk provisioning satu ONT ke OLT.
// Validasi input, resolve profile & VLAN, eksekusi CLI command, perbarui DB, audit log, terbitkan event.
package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// ProvisionONT melakukan provisioning satu ONT ke OLT.
// Alur: validasi -> resolve profile & VLAN -> buat ONT -> execute CLI -> perbarui status -> audit -> event.
func (pm *provisioningManager) ProvisionONT(ctx context.Context, tenantID string, req domain.ProvisionONTRequest) (*domain.ONTResponse, error) {
	if err := pm.ensureWriteEnabled(); err != nil {
		return nil, err
	}

	// Validasi: serial_number unik
	exists, err := pm.ontRepo.SerialNumberExists(ctx, tenantID, req.SerialNumber, "")
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, domain.ErrONTSerialNumberExists
	}

	// Validasi: customer belum punya ONT aktif
	existingONT, err := pm.ontRepo.GetByCustomerID(ctx, req.CustomerID)
	if err != nil && err != domain.ErrONTNotFound {
		return nil, err
	}
	if existingONT != nil && existingONT.Status == domain.ONTStatusProvisioned {
		return nil, domain.ErrCustomerHasActiveONT
	}

	// Ambil OLT untuk membuat adapter
	olt, err := pm.oltRepo.GetByID(ctx, req.OLTID)
	if err != nil {
		return nil, err
	}

	// Resolve service profile
	profile, err := pm.profileRepo.GetByID(ctx, req.ServiceProfileID)
	if err != nil {
		return nil, err
	}

	// Resolve VLAN
	vlan, err := pm.vlanRepo.GetByID(ctx, req.VLANID)
	if err != nil {
		return nil, err
	}

	ontIndex, err := pm.resolveAvailableONTIndex(ctx, req.OLTID, req.PONPortIndex)
	if err != nil {
		return nil, err
	}

	// Buat record ONT dengan state in_progress
	now := time.Now()
	ont := &domain.ONT{
		ID:                uuid.New().String(),
		TenantID:          tenantID,
		OLTID:             req.OLTID,
		PONPortIndex:      req.PONPortIndex,
		ONTIndex:          ontIndex,
		SerialNumber:      req.SerialNumber,
		CustomerID:        &req.CustomerID,
		VLANID:            &req.VLANID,
		ServiceProfileID:  &req.ServiceProfileID,
		Status:            domain.ONTStatusRegistered,
		ProvisioningState: domain.ProvisioningStateInProgress,
		Description:       req.Description,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if req.ODPID != "" {
		ont.ODPID = &req.ODPID
	}

	ont, err = pm.ontRepo.Create(ctx, ont)
	if err != nil {
		return nil, err
	}

	// Buat adapter dan eksekusi CLI commands
	adapter, err := pm.createAdapter(olt)
	if err != nil {
		_ = pm.ontRepo.UpdateStatus(ctx, ont.ID, string(domain.ONTStatusRegistered), string(domain.ProvisioningStateFailed))
		return nil, domain.ErrProvisioningFailed
	}

	// AddONT ke OLT
	addResult, err := adapter.AddONT(ctx, domain.AddONTParams{
		PONPortIndex:     req.PONPortIndex,
		ONTIndex:         ont.ONTIndex,
		SerialNumber:     req.SerialNumber,
		LineProfileID:    profile.LineProfileID,
		ServiceProfileID: profile.ServiceProfileID,
		Description:      req.Description,
	})
	if err != nil || !addResult.Success {
		_ = pm.ontRepo.UpdateStatus(ctx, ont.ID, string(domain.ONTStatusRegistered), string(domain.ProvisioningStateFailed))
		pm.createAuditLog(ctx, tenantID, olt.ID, &ont.ID, domain.AuditActionONTProvision, addResult, "system")
		return nil, domain.ErrProvisioningFailed
	}
	if addResult.AssignedONTIndex > 0 && addResult.AssignedONTIndex != ont.ONTIndex {
		ont.ONTIndex = addResult.AssignedONTIndex
		updated, updateErr := pm.ontRepo.Update(ctx, ont)
		if updateErr != nil {
			_, _ = adapter.RemoveONT(ctx, domain.RemoveONTParams{
				PONPortIndex: req.PONPortIndex,
				ONTIndex:     addResult.AssignedONTIndex,
			})
			_ = pm.ontRepo.UpdateStatus(ctx, ont.ID, string(domain.ONTStatusRegistered), string(domain.ProvisioningStateFailed))
			pm.createAuditLog(ctx, tenantID, olt.ID, &ont.ID, domain.AuditActionONTProvision, addResult, "system")
			return nil, domain.ErrProvisioningFailed
		}
		ont = updated
	}

	// AddServicePort untuk VLAN assignment
	spResult, err := adapter.AddServicePort(ctx, domain.AddServicePortParams{
		PONPortIndex: req.PONPortIndex,
		ONTIndex:     ont.ONTIndex,
		VLANID:       vlan.VLANID,
		GemPort:      1,
	})
	if err != nil || !spResult.Success {
		_, _ = adapter.RemoveONT(ctx, domain.RemoveONTParams{
			PONPortIndex: req.PONPortIndex,
			ONTIndex:     ont.ONTIndex,
		})
		_ = pm.ontRepo.UpdateStatus(ctx, ont.ID, string(domain.ONTStatusRegistered), string(domain.ProvisioningStateFailed))
		pm.createAuditLog(ctx, tenantID, olt.ID, &ont.ID, domain.AuditActionServicePortAdd, spResult, "system")
		return nil, domain.ErrProvisioningFailed
	}

	// Perbarui status ke provisioned + completed
	provisionedAt := time.Now()
	ont.Status = domain.ONTStatusProvisioned
	ont.ProvisioningState = domain.ProvisioningStateCompleted
	ont.LastProvisionedAt = &provisionedAt
	ont.UpdatedAt = provisionedAt
	if _, err := pm.ontRepo.Update(ctx, ont); err != nil {
		log.Error().Err(err).Str("ont_id", ont.ID).Msg("gagal update status ONT setelah provisioning")
	}

	// Gabungkan hasil untuk audit log
	combinedResult := &domain.ProvisioningResult{
		Success:      true,
		CommandsSent: append(addResult.CommandsSent, spResult.CommandsSent...),
		Responses:    append(addResult.Responses, spResult.Responses...),
	}
	pm.createAuditLog(ctx, tenantID, olt.ID, &ont.ID, domain.AuditActionONTProvision, combinedResult, "system")

	// Terbitkan event
	_ = pm.eventPub.PublishONTProvisioned(ctx, domain.ONTProvisionedPayload{
		CorrelationID: uuid.New().String(),
		ONTID:         ont.ID,
		SerialNumber:  ont.SerialNumber,
		CustomerID:    req.CustomerID,
		OLTID:         olt.ID,
		OLTName:       olt.Name,
		PONPortIndex:  req.PONPortIndex,
		VLANID:        req.VLANID,
		TenantID:      tenantID,
	})

	return ontToResponse(ont), nil
}
