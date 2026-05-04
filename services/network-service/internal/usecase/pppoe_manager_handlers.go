// Package usecase berisi implementasi business logic untuk network-service.
// File ini berisi handler methods untuk pppoeManager yang memproses event
// dari Billing API: customer.activated, isolir, un_isolir, suspend, package_change.
package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// HandleCustomerActivated membuat PPPoE user di router saat pelanggan diaktivasi.
// Flow: validasi → resolve router & pool → build command → execute → save DB → publish event.
// Jika connection_method bukan "pppoe", event di-skip (return nil).
func (m *pppoeManager) HandleCustomerActivated(ctx context.Context, payload domain.CustomerActivatedPayload) error {
	// Skip event yang bukan PPPoE
	if payload.ConnectionMethod != "pppoe" {
		m.logger.Debug().
			Str("customer_id", payload.CustomerID).
			Str("connection_method", payload.ConnectionMethod).
			Msg("skip event: connection_method bukan pppoe")
		return nil
	}

	startTime := time.Now()
	correlationID := uuid.New().String()

	log := m.logger.With().
		Str("customer_id", payload.CustomerID).
		Str("tenant_id", payload.TenantID).
		Str("router_id", payload.RouterID).
		Str("pppoe_username", payload.PPPoEUsername).
		Str("correlation_id", correlationID).
		Logger()

	log.Info().Msg("memulai pembuatan PPPoE user di router")

	existingUser, existingErr := m.userRepo.GetByCustomerID(ctx, payload.CustomerID)
	if existingErr == nil && existingUser != nil && existingUser.SyncStatus == domain.SyncStatusSynced {
		log.Info().Str("user_id", existingUser.ID).Msg("skip customer.activated: PPPoE user sudah synced")
		return nil
	}
	if existingErr != nil && !errors.Is(existingErr, domain.ErrPPPoEUserNotFound) {
		return fmt.Errorf("gagal cek PPPoE user existing: %w", existingErr)
	}

	encryptedPassword, err := m.crypto.Encrypt(payload.PPPoEPassword)
	if err != nil {
		log.Error().Err(err).Msg("gagal enkripsi password PPPoE")
		return fmt.Errorf("gagal enkripsi password: %w", err)
	}

	// Build comment untuk tracking ownership di router
	comment := domain.BuildComment(payload.CustomerID, payload.TenantID)

	profile, err := m.resolveProfileForPackage(ctx, domain.PackageProfilePayload{
		TenantID:            payload.TenantID,
		PackageID:           payload.PackageID,
		MikrotikProfileName: payload.MikrotikProfileName,
		DownloadMbps:        payload.DownloadMbps,
		UploadMbps:          payload.UploadMbps,
		AddressPool:         payload.AddressPool,
	})
	if err != nil {
		log.Error().Err(err).Str("package_id", payload.PackageID).Msg("gagal resolve profile dari package_id")
		return fmt.Errorf("gagal resolve profile untuk package %s: %w", payload.PackageID, err)
	}

	// Ambil router dan koneksi dari pool setelah guard idempotency lolos.
	router, pool, adapter, err := m.getRouterAndPool(ctx, payload.RouterID, domain.PriorityMedium)
	if err != nil {
		log.Error().Err(err).Msg("gagal mendapatkan router dan koneksi pool")
		return err
	}
	defer pool.Put(adapter)

	cmdBuilder := m.buildCommandBuilder(router)
	if err := m.ensureProfileOnRouter(ctx, adapter, cmdBuilder, profile, log); err != nil {
		log.Error().Err(err).Msg("gagal memastikan PPPoE profile di router")
		return fmt.Errorf("gagal memastikan PPPoE profile di router: %w", err)
	}

	// Build parameter PPPoE secret untuk router
	secretParams := domain.PPPoESecretParams{
		Name:     payload.PPPoEUsername,
		Password: payload.PPPoEPassword,
		Service:  "pppoe",
		Profile:  profile.ProfileName,
		Comment:  comment,
	}

	// Build dan execute perintah CreateSecret di router
	cmd, args := cmdBuilder.CreateSecret(secretParams)

	log.Info().
		Str("command", cmd).
		Str("profile", profile.ProfileName).
		Msg("menjalankan perintah CreateSecret di router")

	_, execErr := adapter.Execute(ctx, cmd, args)

	executedAt := time.Now()
	durationMs := executedAt.Sub(startTime).Milliseconds()

	// Tentukan sync_status berdasarkan hasil eksekusi
	syncStatus := domain.SyncStatusSynced
	if execErr != nil {
		syncStatus = domain.SyncStatusPendingCreate
		log.Error().Err(execErr).Msg("gagal membuat PPPoE secret di router")
	} else {
		log.Info().Msg("berhasil membuat PPPoE secret di router")
	}

	// Simpan PPPoE user ke database
	now := time.Now()
	var lastSyncAt *time.Time
	if syncStatus == domain.SyncStatusSynced {
		lastSyncAt = &now
	}

	pppoeUser := &domain.PPPoEUser{
		TenantID:          payload.TenantID,
		CustomerID:        payload.CustomerID,
		RouterID:          payload.RouterID,
		Username:          payload.PPPoEUsername,
		PasswordEncrypted: encryptedPassword,
		ProfileName:       profile.ProfileName,
		Service:           "pppoe",
		Comment:           comment,
		Disabled:          false,
		Status:            "active",
		SyncStatus:        syncStatus,
		LastSyncAt:        lastSyncAt,
	}

	var saveErr error
	if existingUser != nil {
		pppoeUser.ID = existingUser.ID
		_, saveErr = m.userRepo.Update(ctx, pppoeUser)
	} else {
		_, saveErr = m.userRepo.Create(ctx, pppoeUser)
	}
	if saveErr != nil {
		log.Error().Err(saveErr).Msg("gagal menyimpan PPPoE user ke database")
		// Tetap publish event meskipun save gagal
	}

	// Publish command_result event
	resultStatus := "success"
	var errMsg string
	if execErr != nil {
		resultStatus = "failed"
		errMsg = execErr.Error()
	}

	publishErr := m.eventPub.PublishCommandResult(ctx, domain.CommandResultPayload{
		CorrelationID: correlationID,
		CustomerID:    payload.CustomerID,
		RouterID:      payload.RouterID,
		TenantID:      payload.TenantID,
		Operation:     "create",
		Status:        resultStatus,
		ErrorMessage:  errMsg,
		ExecutedAt:    executedAt,
		DurationMs:    durationMs,
	})
	if publishErr != nil {
		log.Error().Err(publishErr).Msg("gagal publish command_result event")
	}

	// Return error jika eksekusi di router gagal (untuk retry oleh worker)
	if execErr != nil {
		return fmt.Errorf("gagal membuat PPPoE secret di router %s: %w", payload.RouterID, execErr)
	}

	return nil
}
