// Package usecase berisi implementasi business logic untuk network-service.
// File ini berisi implementasi HandlePackageChanged untuk pppoeManager.
// Flow: resolve profile → create if not exists → update secret → update queue → disconnect.
package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// HandlePackageChanged mengupdate profile assignment dan reconnect user.
// Menggunakan PriorityMedium untuk operasi upgrade/downgrade paket.
// Jika connection_method bukan "pppoe", event di-skip (return nil).
func (m *pppoeManager) HandlePackageChanged(ctx context.Context, payload domain.PackageChangedPayload) error {
	// Skip event yang bukan PPPoE
	if payload.ConnectionMethod != "pppoe" {
		m.logger.Debug().
			Str("customer_id", payload.CustomerID).
			Str("connection_method", payload.ConnectionMethod).
			Msg("skip event package_change: connection_method bukan pppoe")
		return nil
	}

	startTime := time.Now()
	correlationID := uuid.New().String()

	log := m.logger.With().
		Str("customer_id", payload.CustomerID).
		Str("tenant_id", payload.TenantID).
		Str("router_id", payload.RouterID).
		Str("old_package_id", payload.OldPackageID).
		Str("new_package_id", payload.NewPackageID).
		Str("correlation_id", correlationID).
		Logger()

	log.Info().Msg("memulai sequence package change PPPoE user")

	// Ambil PPPoE user dari DB sebelum membuka koneksi router.
	pppoeUser, userErr := m.userRepo.GetByCustomerID(ctx, payload.CustomerID)
	if userErr != nil {
		if errors.Is(userErr, domain.ErrPPPoEUserNotFound) {
			log.Warn().Msg("skip package change: PPPoE user belum ada di DB")
			return nil
		}
		log.Error().Err(userErr).Msg("gagal ambil PPPoE user dari DB")
		m.publishPackageChangeResult(ctx, correlationID, payload, startTime, userErr)
		return fmt.Errorf("gagal ambil PPPoE user: %w", userErr)
	}

	// Resolve new profile dari profileRepo atau fallback metadata dari billing-api.
	newProfile, err := m.resolveProfileForPackage(ctx, domain.PackageProfilePayload{
		TenantID:            payload.TenantID,
		PackageID:           payload.NewPackageID,
		MikrotikProfileName: payload.MikrotikProfileName,
		DownloadMbps:        payload.DownloadMbps,
		UploadMbps:          payload.UploadMbps,
		AddressPool:         payload.AddressPool,
	})
	if err != nil {
		log.Error().Err(err).Str("package_id", payload.NewPackageID).
			Msg("gagal resolve profile dari new_package_id")
		return fmt.Errorf("gagal resolve profile untuk package %s: %w", payload.NewPackageID, err)
	}

	// Ambil router dan koneksi dari pool dengan PriorityMedium setelah guard DB lolos.
	router, pool, adapter, err := m.getRouterAndPool(ctx, payload.RouterID, domain.PriorityMedium)
	if err != nil {
		log.Error().Err(err).Msg("gagal mendapatkan router dan koneksi pool")
		return err
	}
	defer pool.Put(adapter)

	cmdBuilder := m.buildCommandBuilder(router)

	// Cek apakah profile sudah ada di router, jika belum buat dulu
	err = m.ensureProfileOnRouter(ctx, adapter, cmdBuilder, newProfile, log)
	if err != nil {
		log.Error().Err(err).Msg("gagal memastikan profile ada di router")
		m.publishPackageChangeResult(ctx, correlationID, payload, startTime, err)
		return fmt.Errorf("gagal memastikan profile di router: %w", err)
	}

	// Update PPPoE secret profile → /ppp/secret/set profile=new_profile_name
	cmd, args := cmdBuilder.SetSecret(pppoeUser.Username, map[string]string{
		"profile": newProfile.ProfileName,
	})
	log.Info().Str("command", cmd).Str("new_profile", newProfile.ProfileName).
		Msg("update PPPoE secret profile")

	_, execErr := adapter.Execute(ctx, cmd, args)
	if execErr != nil {
		log.Error().Err(execErr).Msg("gagal update profile PPPoE secret di router")
		m.publishPackageChangeResult(ctx, correlationID, payload, startTime, execErr)
		return fmt.Errorf("gagal update profile di router %s: %w", payload.RouterID, execErr)
	}

	// Jika use_simple_queue enabled, update bandwidth limits
	if pppoeUser.UseSimpleQueue {
		maxLimit := fmt.Sprintf("%s/%s", newProfile.DownloadLimit, newProfile.UploadLimit)
		queueParams := map[string]string{"max-limit": maxLimit}

		// Tambahkan burst settings jika ada
		if newProfile.BurstDownload != "" && newProfile.BurstUpload != "" {
			queueParams["burst-limit"] = fmt.Sprintf("%s/%s", newProfile.BurstDownload, newProfile.BurstUpload)
		}
		if newProfile.BurstThresholdDownload != "" && newProfile.BurstThresholdUpload != "" {
			queueParams["burst-threshold"] = fmt.Sprintf("%s/%s",
				newProfile.BurstThresholdDownload, newProfile.BurstThresholdUpload)
		}
		if newProfile.BurstTime != "" {
			queueParams["burst-time"] = newProfile.BurstTime
		}

		queueCmd, queueArgs := cmdBuilder.SetSimpleQueue(pppoeUser.Username, queueParams)
		log.Info().Str("command", queueCmd).Msg("update simple queue bandwidth limits")
		if _, queueErr := adapter.Execute(ctx, queueCmd, queueArgs); queueErr != nil {
			log.Warn().Err(queueErr).Msg("gagal update simple queue (mungkin tidak ada)")
		}
	}

	// Disconnect active session untuk force reconnect dengan profile baru
	disconnectErr := m.disconnectActiveSessionByUsername(ctx, adapter, cmdBuilder, pppoeUser.Username, log)
	if disconnectErr != nil {
		log.Warn().Err(disconnectErr).Msg("gagal disconnect session untuk reconnect (mungkin tidak ada session)")
	}

	// Update PPPoE user di DB dengan profile_name baru
	pppoeUser.ProfileName = newProfile.ProfileName
	pppoeUser.UpdatedAt = time.Now()
	if _, updateErr := m.userRepo.Update(ctx, pppoeUser); updateErr != nil {
		log.Error().Err(updateErr).Msg("gagal update profile_name PPPoE user di DB")
	}

	// Publish command_result event
	m.publishPackageChangeResult(ctx, correlationID, payload, startTime, nil)
	log.Info().Str("new_profile", newProfile.ProfileName).Msg("sequence package change berhasil")
	return nil
}

// ensureProfileOnRouter memastikan profile sudah ada di router.
// Jika belum ada, buat profile baru di router.
func (m *pppoeManager) ensureProfileOnRouter(
	ctx context.Context,
	adapter domain.RouterOSAdapter,
	cmdBuilder domain.CommandBuilder,
	profile *domain.PPPoEProfile,
	log zerolog.Logger,
) error {
	profiles, err := adapter.Execute(ctx, "/ppp/profile/print", map[string]string{"?name": profile.ProfileName})
	if err != nil {
		return fmt.Errorf("gagal print profile untuk cek profile: %w", err)
	}

	if len(profiles) > 0 {
		log.Debug().Str("profile", profile.ProfileName).Msg("profile sudah ada di router")
		return nil
	}

	if profile.DownloadLimit == "" || profile.UploadLimit == "" {
		return fmt.Errorf("profile %s tidak ada di router dan limit paket tidak tersedia untuk membuat profile", profile.ProfileName)
	}

	// Profile belum ada, buat baru
	onlyOne := "yes"
	if !profile.OnlyOne {
		onlyOne = "no"
	}

	rateLimit := fmt.Sprintf("%s/%s", profile.DownloadLimit, profile.UploadLimit)

	profileParams := domain.PPPoEProfileParams{
		Name:          profile.ProfileName,
		LocalAddress:  profile.LocalAddress,
		RemoteAddress: profile.AddressPool,
		RateLimit:     rateLimit,
		OnlyOne:       onlyOne,
	}

	// Tambahkan burst settings jika ada
	if profile.BurstDownload != "" && profile.BurstUpload != "" {
		profileParams.BurstLimit = fmt.Sprintf("%s/%s", profile.BurstDownload, profile.BurstUpload)
	}
	if profile.BurstThresholdDownload != "" && profile.BurstThresholdUpload != "" {
		profileParams.BurstThreshold = fmt.Sprintf("%s/%s",
			profile.BurstThresholdDownload, profile.BurstThresholdUpload)
	}
	if profile.BurstTime != "" {
		profileParams.BurstTime = profile.BurstTime
	}

	createCmd, createArgs := cmdBuilder.CreateProfile(profileParams)
	log.Info().Str("command", createCmd).Str("profile", profile.ProfileName).
		Msg("membuat profile baru di router")

	_, createErr := adapter.Execute(ctx, createCmd, createArgs)
	if createErr != nil {
		return fmt.Errorf("gagal membuat profile %s di router: %w", profile.ProfileName, createErr)
	}

	return nil
}

// publishPackageChangeResult mempublikasikan command_result event untuk package change.
func (m *pppoeManager) publishPackageChangeResult(
	ctx context.Context,
	correlationID string,
	payload domain.PackageChangedPayload,
	startTime time.Time,
	execErr error,
) {
	executedAt := time.Now()
	status := "success"
	var errMsg string
	if execErr != nil {
		status = "failed"
		errMsg = execErr.Error()
	}

	publishErr := m.eventPub.PublishCommandResult(ctx, domain.CommandResultPayload{
		CorrelationID: correlationID,
		CustomerID:    payload.CustomerID,
		RouterID:      payload.RouterID,
		TenantID:      payload.TenantID,
		Operation:     "package_change",
		Status:        status,
		ErrorMessage:  errMsg,
		ExecutedAt:    executedAt,
		DurationMs:    executedAt.Sub(startTime).Milliseconds(),
	})
	if publishErr != nil {
		m.logger.Error().Err(publishErr).Str("correlation_id", correlationID).
			Msg("gagal publish command_result event package_change")
	}
}
