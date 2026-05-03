// Package usecase — implementasi query methods untuk ProvisioningManager.
// GetONTByID, ListONTs, GetUnregisteredONTs, GetAuditLogs, GetSettings, UpdateSettings.
package usecase

import (
	"context"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// GetONTByID mengambil detail ONT termasuk relasi (OLT name, ODP name, dll).
func (pm *provisioningManager) GetONTByID(ctx context.Context, id string) (*domain.ONTDetailResponse, error) {
	ont, err := pm.ontRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	resp := &domain.ONTDetailResponse{
		ONTResponse: *ontToResponse(ont),
	}

	// Enrich dengan nama relasi
	pm.enrichONTResponse(ctx, &resp.ONTResponse, ont)

	// Ambil audit logs terkait ONT
	auditResult, err := pm.auditRepo.List(ctx, domain.AuditLogListParams{
		TenantID: ont.TenantID,
		ONTID:    ont.ID,
		Page:     1,
		PageSize: 20,
	})
	if err == nil && auditResult != nil {
		for _, al := range auditResult.Data {
			resp.AuditLogs = append(resp.AuditLogs, *al)
		}
	}

	return resp, nil
}

// ListONTs mengambil daftar ONT dengan paginasi dan filter.
func (pm *provisioningManager) ListONTs(ctx context.Context, params domain.ONTListParams) (*domain.ONTListResult, error) {
	return pm.ontRepo.List(ctx, params)
}

// GetUnregisteredONTs mengambil daftar ONT unregistered untuk satu OLT.
func (pm *provisioningManager) GetUnregisteredONTs(ctx context.Context, oltID string) ([]*domain.ONTResponse, error) {
	onts, err := pm.ontRepo.ListByOLTAndStatus(ctx, oltID, string(domain.ONTStatusUnregistered))
	if err != nil {
		return nil, err
	}

	var responses []*domain.ONTResponse
	for _, ont := range onts {
		responses = append(responses, ontToResponse(ont))
	}
	return responses, nil
}

// GetAuditLogs mengambil daftar audit log dengan paginasi dan filter.
func (pm *provisioningManager) GetAuditLogs(ctx context.Context, params domain.AuditLogListParams) (*domain.AuditLogListResult, error) {
	return pm.auditRepo.List(ctx, params)
}

// GetSettings mengambil provisioning settings untuk tenant.
// Jika tidak ada record, kembalikan default values.
func (pm *provisioningManager) GetSettings(ctx context.Context, tenantID string) (*domain.ProvisioningSettings, error) {
	settings, err := pm.settingsRepo.GetByTenantID(ctx, tenantID)
	if err != nil {
		// Jika tidak ditemukan, kembalikan default
		return domain.DefaultProvisioningSettings(tenantID), nil
	}
	return settings, nil
}

// UpdateSettings memperbarui provisioning settings untuk tenant.
func (pm *provisioningManager) UpdateSettings(ctx context.Context, tenantID string, req domain.UpdateSettingsRequest) (*domain.ProvisioningSettings, error) {
	// Ambil settings existing atau default
	settings, _ := pm.settingsRepo.GetByTenantID(ctx, tenantID)
	if settings == nil {
		settings = domain.DefaultProvisioningSettings(tenantID)
	}

	// Apply updates
	if req.AutoProvisioningEnabled != nil {
		settings.AutoProvisioningEnabled = *req.AutoProvisioningEnabled
	}
	if req.AutoPortMigrationEnabled != nil {
		settings.AutoPortMigrationEnabled = *req.AutoPortMigrationEnabled
	}
	if req.VLANStrategy != "" {
		settings.VLANStrategy = domain.VLANStrategy(req.VLANStrategy)
	}

	return pm.settingsRepo.Upsert(ctx, settings)
}

// enrichONTResponse menambahkan nama relasi ke ONTResponse.
func (pm *provisioningManager) enrichONTResponse(ctx context.Context, resp *domain.ONTResponse, ont *domain.ONT) {
	// OLT name
	olt, err := pm.oltRepo.GetByID(ctx, ont.OLTID)
	if err == nil {
		resp.OLTName = olt.Name
	}

	// VLAN name
	if ont.VLANID != nil {
		vlan, err := pm.vlanRepo.GetByID(ctx, *ont.VLANID)
		if err == nil {
			resp.VLANName = vlan.Name
		}
	}

	// Service profile name
	if ont.ServiceProfileID != nil {
		profile, err := pm.profileRepo.GetByID(ctx, *ont.ServiceProfileID)
		if err == nil {
			resp.ServiceProfileName = profile.Name
		}
	}
}
