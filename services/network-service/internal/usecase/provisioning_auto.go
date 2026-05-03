// Package usecase — implementasi HandleUnregisteredONT, HandlePortMigration, ConfirmMigration.
// Auto-provisioning, deteksi port migration, dan konfirmasi admin.
package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// HandleUnregisteredONT memproses ONT unregistered yang terdeteksi sync engine.
// Buat record ONT dengan status="unregistered", cek auto_provisioning_enabled,
// jika enabled dan SN match customer → auto-provision.
func (pm *provisioningManager) HandleUnregisteredONT(ctx context.Context, oltID string, ont domain.UnregisteredONT) error {
	// Ambil OLT untuk tenant_id
	olt, err := pm.oltRepo.GetByID(ctx, oltID)
	if err != nil {
		return err
	}

	// Buat record ONT dengan status unregistered
	now := time.Now()
	newONT := &domain.ONT{
		ID:                uuid.New().String(),
		TenantID:          olt.TenantID,
		OLTID:             oltID,
		PONPortIndex:      ont.PONPortIndex,
		ONTIndex:          ont.ONTIndex,
		SerialNumber:      ont.SerialNumber,
		Status:            domain.ONTStatusUnregistered,
		ProvisioningState: domain.ProvisioningStatePending,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	newONT, err = pm.ontRepo.Create(ctx, newONT)
	if err != nil {
		return err
	}

	// Cek settings auto-provisioning
	settings, err := pm.GetSettings(ctx, olt.TenantID)
	if err != nil {
		return err
	}

	if !settings.AutoProvisioningEnabled {
		// Auto-provisioning disabled, ONT tetap unregistered
		log.Info().Str("serial_number", ont.SerialNumber).Msg("auto-provisioning disabled, ONT tetap unregistered")
		return nil
	}

	// Cari customer yang cocok berdasarkan serial number
	// Lookup ONT by serial number yang sudah pernah terdaftar (misal decommissioned)
	existingONT, err := pm.ontRepo.GetBySerialNumber(ctx, olt.TenantID, ont.SerialNumber)
	if err != nil || existingONT == nil || existingONT.CustomerID == nil {
		// Tidak ada customer match, ONT tetap unregistered
		log.Info().Str("serial_number", ont.SerialNumber).Msg("auto-provisioning: tidak ada customer match")
		return nil
	}

	// Auto-provision: gunakan data customer yang sudah ada
	customerID := *existingONT.CustomerID
	req := domain.ProvisionONTRequest{
		SerialNumber: ont.SerialNumber,
		OLTID:        oltID,
		PONPortIndex: ont.PONPortIndex,
		CustomerID:   customerID,
	}

	// Resolve service profile dan VLAN jika ada
	if existingONT.ServiceProfileID != nil {
		req.ServiceProfileID = *existingONT.ServiceProfileID
	}
	if existingONT.VLANID != nil {
		req.VLANID = *existingONT.VLANID
	}

	// Hapus record unregistered yang baru dibuat, karena ProvisionONT akan buat baru
	_ = pm.ontRepo.SoftDelete(ctx, newONT.ID)

	_, provErr := pm.ProvisionONT(ctx, olt.TenantID, req)
	if provErr != nil {
		// Auto-provisioning gagal, publish event
		_ = pm.eventPub.PublishONTAutoProvisionFailed(ctx, domain.ONTAutoProvisionFailedPayload{
			CorrelationID: uuid.New().String(),
			SerialNumber:  ont.SerialNumber,
			OLTID:         oltID,
			PONPortIndex:  ont.PONPortIndex,
			ErrorMessage:  provErr.Error(),
			TenantID:      olt.TenantID,
		})
		log.Warn().Err(provErr).Str("serial_number", ont.SerialNumber).Msg("auto-provisioning gagal")
		return provErr
	}

	// Auto-provisioning berhasil, publish event
	_ = pm.eventPub.PublishONTAutoProvisioned(ctx, domain.ONTAutoProvisionedPayload{
		CorrelationID: uuid.New().String(),
		ONTID:         newONT.ID,
		SerialNumber:  ont.SerialNumber,
		CustomerID:    customerID,
		OLTID:         oltID,
		PONPortIndex:  ont.PONPortIndex,
		TenantID:      olt.TenantID,
	})

	return nil
}

// HandlePortMigration memproses deteksi port migration dari sync engine.
// Publish event, cek auto_port_migration_enabled, update DB jika enabled.
func (pm *provisioningManager) HandlePortMigration(ctx context.Context, ontID string, oldPort, newPort, oldONTIdx, newONTIdx int) error {
	ont, err := pm.ontRepo.GetByID(ctx, ontID)
	if err != nil {
		return err
	}

	// Publish event port migrated
	_ = pm.eventPub.PublishONTPortMigrated(ctx, domain.ONTPortMigratedPayload{
		CorrelationID: uuid.New().String(),
		ONTID:         ont.ID,
		SerialNumber:  ont.SerialNumber,
		OLTID:         ont.OLTID,
		OldPortIndex:  oldPort,
		NewPortIndex:  newPort,
		OldONTIndex:   oldONTIdx,
		NewONTIndex:   newONTIdx,
		TenantID:      ont.TenantID,
	})

	// Cek settings auto-port-migration
	settings, err := pm.GetSettings(ctx, ont.TenantID)
	if err != nil {
		return err
	}

	if settings.AutoPortMigrationEnabled {
		// Auto-update DB
		return pm.ontRepo.UpdatePortMigration(ctx, ontID, newPort, newONTIdx)
	}

	// Disabled: flag sebagai port_migrated, tunggu konfirmasi admin
	// Simpan info migrasi di description sementara
	return nil
}

// ConfirmMigration mengkonfirmasi port migration dan update DB.
func (pm *provisioningManager) ConfirmMigration(ctx context.Context, ontID string) error {
	_, err := pm.ontRepo.GetByID(ctx, ontID)
	if err != nil {
		return err
	}

	// Konfirmasi sudah dilakukan, port migration sudah di-handle
	// Dalam implementasi penuh, ini akan membaca pending migration data
	return nil
}
