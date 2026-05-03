// Package usecase berisi implementasi business logic untuk network-service.
// File ini mendefinisikan struct provisioningManager, constructor, dan helper methods.
package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// Compile-time check: provisioningManager harus mengimplementasikan domain.ProvisioningManager.
var _ domain.ProvisioningManager = (*provisioningManager)(nil)

// provisioningManager mengimplementasikan domain.ProvisioningManager.
// Mengelola provisioning ONT: single, bulk, decommission, reboot, auto-provisioning,
// port migration, audit trail, dan settings.
type provisioningManager struct {
	ontRepo        domain.ONTRepository
	vlanRepo       domain.VLANRepository
	profileRepo    domain.ServiceProfileRepository
	auditRepo      domain.AuditLogRepository
	settingsRepo   domain.ProvisioningSettingsRepository
	oltRepo        domain.OLTRepository
	factory        domain.OLTAdapterFactory
	encryptor      domain.CredentialEncryptor
	eventPub       domain.OLTEventPublisher
	vlanMgr        domain.VLANManager
	profileMgr     domain.ServiceProfileManager
	bulkStore      map[string]*domain.BulkPreview // in-memory store untuk bulk preview
}

// NewProvisioningManager membuat instance ProvisioningManager baru dengan semua dependensi.
func NewProvisioningManager(
	ontRepo domain.ONTRepository,
	vlanRepo domain.VLANRepository,
	profileRepo domain.ServiceProfileRepository,
	auditRepo domain.AuditLogRepository,
	settingsRepo domain.ProvisioningSettingsRepository,
	oltRepo domain.OLTRepository,
	factory domain.OLTAdapterFactory,
	encryptor domain.CredentialEncryptor,
	eventPub domain.OLTEventPublisher,
	vlanMgr domain.VLANManager,
	profileMgr domain.ServiceProfileManager,
) domain.ProvisioningManager {
	return &provisioningManager{
		ontRepo:      ontRepo,
		vlanRepo:     vlanRepo,
		profileRepo:  profileRepo,
		auditRepo:    auditRepo,
		settingsRepo: settingsRepo,
		oltRepo:      oltRepo,
		factory:      factory,
		encryptor:    encryptor,
		eventPub:     eventPub,
		vlanMgr:      vlanMgr,
		profileMgr:   profileMgr,
		bulkStore:    make(map[string]*domain.BulkPreview),
	}
}

// createAdapter membuat OLTAdapter dari OLT entity (decrypt credentials + factory).
func (pm *provisioningManager) createAdapter(olt *domain.OLT) (domain.OLTAdapter, error) {
	snmpCfg, err := pm.buildSNMPConfig(olt)
	if err != nil {
		return nil, err
	}
	cliCfg, err := pm.buildCLIConfig(olt)
	if err != nil {
		return nil, err
	}
	adapter, err := pm.factory.CreateAdapter(olt.Brand, snmpCfg, cliCfg)
	if err != nil {
		log.Error().Err(err).Str("olt_id", olt.ID).Msg("gagal membuat adapter OLT")
		return nil, err
	}
	return adapter, nil
}

// buildSNMPConfig mendekripsi kredensial SNMP dan membangun SNMPConfig.
func (pm *provisioningManager) buildSNMPConfig(olt *domain.OLT) (domain.SNMPConfig, error) {
	cfg := domain.SNMPConfig{
		Host:    olt.Host,
		Port:    olt.SNMPPort,
		Version: olt.SNMPVersion,
		Timeout: 10 * time.Second,
	}
	if olt.SNMPVersion == domain.SNMPv2c {
		community, err := pm.encryptor.Decrypt(olt.SNMPCommunityEncrypted)
		if err != nil {
			return cfg, domain.ErrDecryptionFailed
		}
		cfg.Community = community
	}
	return cfg, nil
}

// buildCLIConfig mendekripsi kredensial CLI dan membangun CLIConfig.
func (pm *provisioningManager) buildCLIConfig(olt *domain.OLT) (domain.CLIConfig, error) {
	password, err := pm.encryptor.Decrypt(olt.CLIPasswordEncrypted)
	if err != nil {
		return domain.CLIConfig{}, domain.ErrDecryptionFailed
	}
	cfg := domain.CLIConfig{
		Host:        olt.Host,
		Port:        olt.CLIPort,
		Protocol:    olt.CLIProtocol,
		Username:    olt.CLIUsername,
		Password:    password,
		ConnTimeout: 10 * time.Second,
		CmdTimeout:  30 * time.Second,
	}
	return cfg, nil
}

// createAuditLog membuat record audit log untuk operasi provisioning.
func (pm *provisioningManager) createAuditLog(
	ctx context.Context,
	tenantID, oltID string,
	ontID *string,
	action domain.AuditAction,
	result *domain.ProvisioningResult,
	performedBy string,
) {
	auditLog := &domain.ProvisioningAuditLog{
		ID:            uuid.New().String(),
		TenantID:      tenantID,
		OLTID:         oltID,
		ONTID:         ontID,
		Action:        action,
		CommandsSent:  result.CommandsSent,
		CommandResponses: result.Responses,
		Status:        "success",
		PerformedBy:   performedBy,
		CorrelationID: uuid.New().String(),
		CreatedAt:     time.Now(),
	}
	if !result.Success {
		auditLog.Status = "failed"
		auditLog.ErrorMessage = result.ErrorMessage
	}
	if _, err := pm.auditRepo.Create(ctx, auditLog); err != nil {
		log.Error().Err(err).Str("action", string(action)).Msg("gagal membuat audit log")
	}
}

// ontToResponse mengkonversi entity ONT ke ONTResponse.
func ontToResponse(ont *domain.ONT) *domain.ONTResponse {
	return &domain.ONTResponse{
		ID:                   ont.ID,
		OLTID:                ont.OLTID,
		PONPortIndex:         ont.PONPortIndex,
		ONTIndex:             ont.ONTIndex,
		SerialNumber:         ont.SerialNumber,
		CustomerID:           ont.CustomerID,
		ODPID:                ont.ODPID,
		VLANID:               ont.VLANID,
		ServiceProfileID:     ont.ServiceProfileID,
		Status:               ont.Status,
		ProvisioningState:    ont.ProvisioningState,
		Description:          ont.Description,
		LastProvisionedAt:    ont.LastProvisionedAt,
		LastDecommissionedAt: ont.LastDecommissionedAt,
		CreatedAt:            ont.CreatedAt,
		UpdatedAt:            ont.UpdatedAt,
	}
}
