package usecase

import (
	"context"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// PreviewProvisionONT membangun command provisioning tanpa membuat ONT DB dan tanpa eksekusi CLI.
func (pm *provisioningManager) PreviewProvisionONT(ctx context.Context, tenantID string, req domain.ProvisionONTRequest) (*domain.ProvisioningDryRun, error) {
	exists, err := pm.ontRepo.SerialNumberExists(ctx, tenantID, req.SerialNumber, "")
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, domain.ErrONTSerialNumberExists
	}

	existingONT, err := pm.ontRepo.GetByCustomerID(ctx, req.CustomerID)
	if err != nil && err != domain.ErrONTNotFound {
		return nil, err
	}
	if existingONT != nil && existingONT.Status == domain.ONTStatusProvisioned {
		return nil, domain.ErrCustomerHasActiveONT
	}

	olt, err := pm.oltRepo.GetByID(ctx, req.OLTID)
	if err != nil {
		return nil, err
	}
	profile, err := pm.profileRepo.GetByID(ctx, req.ServiceProfileID)
	if err != nil {
		return nil, err
	}
	vlan, err := pm.vlanRepo.GetByID(ctx, req.VLANID)
	if err != nil {
		return nil, err
	}
	ontIndex, err := pm.resolveAvailableONTIndex(ctx, req.OLTID, req.PONPortIndex)
	if err != nil {
		return nil, err
	}

	adapter, err := pm.createAdapter(olt)
	if err != nil {
		return nil, err
	}
	previewer, ok := adapter.(domain.ProvisioningCommandPreviewer)
	if !ok {
		return nil, domain.ErrUnsupportedBrand
	}

	addParams := domain.AddONTParams{
		PONPortIndex:     req.PONPortIndex,
		ONTIndex:         ontIndex,
		SerialNumber:     req.SerialNumber,
		LineProfileID:    profile.LineProfileID,
		ServiceProfileID: profile.ServiceProfileID,
		Description:      req.Description,
	}
	serviceParams := domain.AddServicePortParams{
		PONPortIndex: req.PONPortIndex,
		ONTIndex:     ontIndex,
		VLANID:       vlan.VLANID,
		GemPort:      1,
	}
	result, err := previewer.PreviewProvisioningCommands(ctx, addParams, serviceParams)
	if err != nil {
		return nil, err
	}

	warnings := []string{}
	if !pm.writeEnabled {
		warnings = append(warnings, "Provisioning write sedang nonaktif; dry-run tidak mengeksekusi command.")
	}
	return &domain.ProvisioningDryRun{
		OLTID:            olt.ID,
		OLTName:          olt.Name,
		Brand:            olt.Brand,
		Model:            olt.Model,
		Transport:        defaultString(result.Transport, "cli"),
		Operation:        defaultString(result.Operation, "provision_ont_preview"),
		PONPortIndex:     req.PONPortIndex,
		ONTIndex:         ontIndex,
		VLANID:           vlan.VLANID,
		LineProfileID:    profile.LineProfileID,
		ServiceProfileID: profile.ServiceProfileID,
		Commands:         sanitizeProvisioningStrings(result.CommandsSent),
		Warnings:         warnings,
	}, nil
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
