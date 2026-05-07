// Package usecase berisi implementasi business logic untuk network-service.
// File ini mendefinisikan struct provisioningManager, constructor, dan helper methods.
package usecase

import (
	"context"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// Compile-time cek: provisioningManager harus mengimplementasikan domain.ProvisioningManager.
var _ domain.ProvisioningManager = (*provisioningManager)(nil)

// provisioningManager mengimplementasikan domain.ProvisioningManager.
// Mengelola provisioning ONT: single, bulk, decommission, reboot, auto-provisioning,
// port migration, audit trail, dan settings.
type provisioningManager struct {
	ontRepo      domain.ONTRepository
	vlanRepo     domain.VLANRepository
	profileRepo  domain.ServiceProfileRepository
	auditRepo    domain.AuditLogRepository
	settingsRepo domain.ProvisioningSettingsRepository
	oltRepo      domain.OLTRepository
	factory      domain.OLTAdapterFactory
	encryptor    domain.CredentialEncryptor
	eventPub     domain.OLTEventPublisher
	vlanMgr      domain.VLANManager
	profileMgr   domain.ServiceProfileManager
	bulkStore    map[string]*domain.BulkPreview // in-memory store untuk bulk preview
	writeEnabled bool
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
		writeEnabled: true,
	}
}

// SetWriteEnabled mengatur guard operasi write provisioning OLT.
func (pm *provisioningManager) SetWriteEnabled(enabled bool) {
	pm.writeEnabled = enabled
}

// ensureWriteEnabled menolak operasi write ketika guard dinonaktifkan.
func (pm *provisioningManager) ensureWriteEnabled() error {
	if !pm.writeEnabled {
		return domain.ErrOLTProvisioningWriteDisabled
	}
	return nil
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
	if result == nil {
		result = &domain.ProvisioningResult{
			Success:      false,
			ErrorMessage: "hasil provisioning tidak tersedia",
		}
	}
	brand, model := result.Brand, result.Model
	if (brand == "" || model == "") && pm.oltRepo != nil {
		if olt, err := pm.oltRepo.GetByID(ctx, oltID); err == nil && olt != nil {
			if brand == "" {
				brand = string(olt.Brand)
			}
			if model == "" {
				model = olt.Model
			}
		}
	}
	transport := result.Transport
	if transport == "" && len(result.CommandsSent) > 0 {
		transport = "cli"
	}
	operation := result.Operation
	if operation == "" {
		operation = string(action)
	}

	auditLog := &domain.ProvisioningAuditLog{
		ID:               uuid.New().String(),
		TenantID:         tenantID,
		OLTID:            oltID,
		ONTID:            ontID,
		Action:           action,
		CommandsSent:     sanitizeProvisioningStrings(result.CommandsSent),
		CommandResponses: sanitizeProvisioningStrings(result.Responses),
		Status:           "success",
		PerformedBy:      performedBy,
		Brand:            brand,
		Model:            model,
		Transport:        transport,
		Operation:        operation,
		CorrelationID:    uuid.New().String(),
		CreatedAt:        time.Now(),
	}
	if !result.Success {
		auditLog.Status = "failed"
		auditLog.ErrorMessage = result.ErrorMessage
	}
	if _, err := pm.auditRepo.Create(ctx, auditLog); err != nil {
		log.Error().Err(err).Str("action", string(action)).Msg("gagal membuat audit log")
	}
}

var provisioningSecretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(password\s+)(\S+)`),
	regexp.MustCompile(`(?i)(community\s+)(\S+)`),
	regexp.MustCompile(`(?i)(secret\s+)(\S+)`),
	regexp.MustCompile(`(?i)(enable\s+password\s+)(\S+)`),
	regexp.MustCompile(`(?i)(snmp\s+\S+\s+)(\S+)`),
}

func sanitizeProvisioningStrings(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	sanitized := make([]string, 0, len(values))
	for _, value := range values {
		sanitized = append(sanitized, sanitizeProvisioningText(value))
	}
	return sanitized
}

func sanitizeProvisioningText(value string) string {
	for _, pattern := range provisioningSecretPatterns {
		value = pattern.ReplaceAllString(value, "${1}[REDACTED]")
	}
	return value
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
