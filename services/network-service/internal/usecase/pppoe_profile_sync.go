// Package usecase berisi implementasi business logic untuk network-service.
// File ini berisi implementasi SyncProfile untuk pppoeManager.
// SyncProfile menyinkronkan PPPoE profile ke semua router dengan service_type pppoe.
package usecase

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// SyncProfile membuat atau update PPPoE profile di semua router dengan service_type pppoe.
// Eksekusi paralel per router menggunakan goroutines, error per router tidak blocking.
func (m *pppoeManager) SyncProfile(ctx context.Context, profile *domain.PPPoEProfile) error {
	log := m.logger.With().Str("profile_name", profile.ProfileName).Logger()
	log.Info().Msg("memulai sinkronisasi profile ke semua router PPPoE")

	// Ambil semua router aktif
	routers, err := m.routerRepo.GetActiveRouters(ctx)
	if err != nil {
		return fmt.Errorf("gagal ambil daftar router aktif: %w", err)
	}

	// Filter router yang memiliki service_type "pppoe"
	var pppoeRouters []*domain.Router
	for _, r := range routers {
		if hasServiceType(r.ServiceTypes, "pppoe") {
			pppoeRouters = append(pppoeRouters, r)
		}
	}

	if len(pppoeRouters) == 0 {
		log.Info().Msg("tidak ada router PPPoE aktif, skip sync profile")
		return nil
	}

	// Build PPPoEProfileParams dari profile entity
	profileParams := buildProfileParams(profile)

	// Sync ke semua router secara paralel
	var wg sync.WaitGroup
	for _, r := range pppoeRouters {
		wg.Add(1)
		go func(router *domain.Router) {
			defer wg.Done()
			m.syncProfileToRouter(ctx, router, profileParams, log)
		}(r)
	}
	wg.Wait()

	log.Info().Int("router_count", len(pppoeRouters)).Msg("sinkronisasi profile selesai")
	return nil
}

// syncProfileToRouter menyinkronkan satu profile ke satu router.
// Coba create dulu, jika sudah ada (error) maka update.
func (m *pppoeManager) syncProfileToRouter(
	ctx context.Context,
	router *domain.Router,
	params domain.PPPoEProfileParams,
	log zerolog.Logger,
) {
	// Ambil koneksi pool
	password, err := m.crypto.Decrypt(router.PasswordEncrypted)
	if err != nil {
		m.logger.Error().Err(err).Str("router_id", router.ID).
			Msg("gagal dekripsi password router untuk sync profile")
		return
	}

	cfg := domain.ConnectionConfig{
		Host:     router.Host,
		Port:     router.Port,
		Username: router.Username,
		Password: password,
		UseSSL:   router.UseSSL,
	}

	pool := m.poolManager.GetPool(router.ID, cfg)
	adapter, err := pool.Get(ctx, domain.PriorityLow)
	if err != nil {
		m.logger.Error().Err(err).Str("router_id", router.ID).
			Msg("gagal ambil koneksi pool untuk sync profile")
		return
	}
	defer pool.Put(adapter)

	cmdBuilder := m.cmdBuilderFactory(router.RouterOSVersion)

	// Coba create profile terlebih dahulu
	cmd, args := cmdBuilder.CreateProfile(params)
	_, execErr := adapter.Execute(ctx, cmd, args)
	if execErr != nil {
		// Profile mungkin sudah ada, coba update
		setArgs := buildProfileSetArgs(params)
		setCmd, setA := cmdBuilder.SetProfile(params.Name, setArgs)
		_, setErr := adapter.Execute(ctx, setCmd, setA)
		if setErr != nil {
			m.logger.Error().Err(setErr).Str("router_id", router.ID).
				Str("profile", params.Name).Msg("gagal create/update profile di router")
		}
	}
}

// hasServiceType memeriksa apakah slice service types mengandung tipe tertentu.
func hasServiceType(serviceTypes []string, target string) bool {
	for _, st := range serviceTypes {
		if strings.EqualFold(st, target) {
			return true
		}
	}
	return false
}

// buildProfileParams membangun PPPoEProfileParams dari PPPoEProfile entity.
func buildProfileParams(p *domain.PPPoEProfile) domain.PPPoEProfileParams {
	onlyOne := "no"
	if p.OnlyOne {
		onlyOne = "yes"
	}

	return domain.PPPoEProfileParams{
		Name:           p.ProfileName,
		LocalAddress:   p.LocalAddress,
		RemoteAddress:  p.AddressPool,
		RateLimit:      fmt.Sprintf("%s/%s", p.DownloadLimit, p.UploadLimit),
		BurstLimit:     buildBurstField(p.BurstDownload, p.BurstUpload),
		BurstThreshold: buildBurstField(p.BurstThresholdDownload, p.BurstThresholdUpload),
		BurstTime:      p.BurstTime,
		OnlyOne:        onlyOne,
	}
}

// buildBurstField membangun field burst format "download/upload".
// Mengembalikan string kosong jika kedua parameter kosong.
func buildBurstField(download, upload string) string {
	if download == "" && upload == "" {
		return ""
	}
	return fmt.Sprintf("%s/%s", download, upload)
}

// buildProfileSetArgs membangun args map untuk SetProfile dari PPPoEProfileParams.
func buildProfileSetArgs(p domain.PPPoEProfileParams) map[string]string {
	args := map[string]string{
		"local-address": p.LocalAddress,
		"rate-limit":    p.RateLimit,
		"only-one":      p.OnlyOne,
	}
	if p.RemoteAddress != "" {
		args["remote-address"] = p.RemoteAddress
	}
	if p.BurstLimit != "" {
		args["burst-limit"] = p.BurstLimit
	}
	if p.BurstThreshold != "" {
		args["burst-threshold"] = p.BurstThreshold
	}
	if p.BurstTime != "" {
		args["burst-time"] = p.BurstTime
	}
	return args
}
