// Package usecase berisi implementasi business logic untuk network-service.
// File ini berisi implementasi SyncRouter untuk pppoeManager.
// SyncRouter membandingkan data PPPoE di router dengan database dan auto-fix.
package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// SyncRouter menjalankan sinkronisasi database↔router untuk satu router.
// Flow: ambil secrets dari router → ambil users dari DB → bandingkan → auto-fix.
// Database adalah source of truth untuk user yang dikelola ISPBoss.
func (m *pppoeManager) SyncRouter(ctx context.Context, routerID string) (*domain.SyncResult, error) {
	log := m.logger.With().Str("router_id", routerID).Logger()
	log.Info().Msg("memulai sinkronisasi PPPoE user database↔router")

	// Step 1: Ambil router dan koneksi pool dengan PriorityLow (monitoring)
	router, pool, adapter, err := m.getRouterAndPool(ctx, routerID, domain.PriorityLow)
	if err != nil {
		return nil, fmt.Errorf("gagal ambil router dan pool: %w", err)
	}
	defer pool.Put(adapter)

	cmdBuilder := m.buildCommandBuilder(router)

	// Step 2: Ambil semua PPPoE secrets dari router
	cmd, args := cmdBuilder.PrintSecrets()
	routerSecrets, err := adapter.Execute(ctx, cmd, args)
	if err != nil {
		return nil, fmt.Errorf("gagal ambil PPPoE secrets dari router: %w", err)
	}

	// Step 3: Ambil semua PPPoE users dari DB untuk router ini
	dbUsers, err := m.userRepo.GetByRouterID(ctx, routerID)
	if err != nil {
		return nil, fmt.Errorf("gagal ambil PPPoE users dari DB: %w", err)
	}

	// Step 4: Bandingkan dan kategorikan
	now := time.Now()
	result := &domain.SyncResult{
		RouterID:   routerID,
		TotalUsers: len(dbUsers),
		SyncedAt:   now,
	}

	// Buat map DB users berdasarkan username untuk lookup cepat
	dbUserMap := make(map[string]*domain.PPPoEUser, len(dbUsers))
	for _, u := range dbUsers {
		dbUserMap[u.Username] = u
	}

	// Track username yang sudah diproses dari router (untuk deteksi missing)
	processedUsernames := make(map[string]bool)

	// Iterasi secrets dari router
	for _, secret := range routerSecrets {
		username := secret["name"]
		comment := secret["comment"]
		routerProfile := secret["profile"]
		routerDisabled := parseDisabledField(secret["disabled"])

		// Cek apakah user ISPBoss berdasarkan comment
		if !domain.IsISPBossComment(comment) {
			// User tanpa comment ISPBoss → orphan (user manual admin)
			result.OrphanCount++
			continue
		}

		processedUsernames[username] = true

		dbUser, exists := dbUserMap[username]
		if !exists {
			// Ada di router dengan comment ISPBoss tapi tidak di DB → orphan
			result.OrphanCount++
			continue
		}

		// Ada di kedua sisi, bandingkan profile dan disabled state
		if dbUser.ProfileName == routerProfile && dbUser.Disabled == routerDisabled {
			result.SyncedCount++
			m.updateUserSyncStatus(ctx, dbUser.ID, domain.SyncStatusSynced, &now)
		} else {
			// Out of sync → auto-fix: update router sesuai DB (DB source of truth)
			result.OutOfSyncCount++
			fixErr := m.fixOutOfSyncUser(ctx, adapter, cmdBuilder, dbUser)
			if fixErr != nil {
				result.ErrorCount++
				log.Error().Err(fixErr).Str("username", username).Msg("gagal fix out_of_sync user")
			} else {
				m.updateUserSyncStatus(ctx, dbUser.ID, domain.SyncStatusSynced, &now)
			}
		}
	}

	// Step 5: Deteksi missing users (ada di DB tapi tidak di router)
	for _, dbUser := range dbUsers {
		if processedUsernames[dbUser.Username] {
			continue
		}
		// User aktif di DB tapi tidak ada di router → missing, auto-create
		if dbUser.Status == "active" && dbUser.DeletedAt == nil {
			result.MissingCount++
			createErr := m.fixMissingUser(ctx, adapter, cmdBuilder, dbUser)
			if createErr != nil {
				result.ErrorCount++
				log.Error().Err(createErr).Str("username", dbUser.Username).
					Msg("gagal create missing user di router")
			} else {
				m.updateUserSyncStatus(ctx, dbUser.ID, domain.SyncStatusSynced, &now)
			}
		}
	}

	log.Info().
		Int("synced", result.SyncedCount).
		Int("orphan", result.OrphanCount).
		Int("missing", result.MissingCount).
		Int("out_of_sync", result.OutOfSyncCount).
		Int("errors", result.ErrorCount).
		Msg("sinkronisasi PPPoE user selesai")

	return result, nil
}

// parseDisabledField mengurai field "disabled" dari router response.
// "true" atau "yes" → true, selainnya → false.
func parseDisabledField(value string) bool {
	v := strings.ToLower(strings.TrimSpace(value))
	return v == "true" || v == "yes"
}

// syncDiffResult berisi hasil kategorisasi sync diff antara router secrets dan DB users.
// Digunakan oleh computeSyncDiff sebagai pure function yang bisa di-test secara independen.
type syncDiffResult struct {
	SyncedCount    int
	OrphanCount    int
	MissingCount   int
	OutOfSyncCount int
}

// computeSyncDiff membandingkan PPPoE secrets dari router dengan PPPoE users dari database
// dan mengkategorikan setiap user ke dalam tepat satu kategori:
//   - synced: ada di kedua sisi dengan profile dan disabled state yang sama
//   - orphan: ada di router tanpa comment ISPBoss, ATAU ada di router dengan comment ISPBoss tapi tidak di DB
//   - missing: ada di DB (status active, belum dihapus) tapi tidak ada di router
//   - out_of_sync: ada di kedua sisi tapi profile atau disabled state berbeda
//
// Fungsi ini pure (tanpa side-effect) sehingga bisa di-test secara independen.
func computeSyncDiff(routerSecrets []map[string]string, dbUsers []*domain.PPPoEUser) syncDiffResult {
	result := syncDiffResult{}

	// Buat map DB users berdasarkan username untuk lookup cepat
	dbUserMap := make(map[string]*domain.PPPoEUser, len(dbUsers))
	for _, u := range dbUsers {
		dbUserMap[u.Username] = u
	}

	// Track username yang sudah diproses dari router (untuk deteksi missing)
	processedUsernames := make(map[string]bool)

	// Iterasi secrets dari router
	for _, secret := range routerSecrets {
		username := secret["name"]
		comment := secret["comment"]
		routerProfile := secret["profile"]
		routerDisabled := parseDisabledField(secret["disabled"])

		// Cek apakah user ISPBoss berdasarkan comment
		if !domain.IsISPBossComment(comment) {
			// User tanpa comment ISPBoss → orphan (user manual admin)
			result.OrphanCount++
			continue
		}

		processedUsernames[username] = true

		dbUser, exists := dbUserMap[username]
		if !exists {
			// Ada di router dengan comment ISPBoss tapi tidak di DB → orphan
			result.OrphanCount++
			continue
		}

		// Ada di kedua sisi, bandingkan profile dan disabled state
		if dbUser.ProfileName == routerProfile && dbUser.Disabled == routerDisabled {
			result.SyncedCount++
		} else {
			result.OutOfSyncCount++
		}
	}

	// Deteksi missing users (ada di DB tapi tidak di router)
	for _, dbUser := range dbUsers {
		if processedUsernames[dbUser.Username] {
			continue
		}
		// User aktif di DB tapi tidak ada di router → missing
		if dbUser.Status == "active" && dbUser.DeletedAt == nil {
			result.MissingCount++
		}
	}

	return result
}

// fixOutOfSyncUser mengupdate PPPoE secret di router agar sesuai DB.
func (m *pppoeManager) fixOutOfSyncUser(
	ctx context.Context,
	adapter domain.RouterOSAdapter,
	cmdBuilder domain.CommandBuilder,
	dbUser *domain.PPPoEUser,
) error {
	params := map[string]string{
		"profile": dbUser.ProfileName,
	}
	if dbUser.Disabled {
		params["disabled"] = "yes"
	} else {
		params["disabled"] = "no"
	}
	cmd, args := cmdBuilder.SetSecret(dbUser.Username, params)
	_, err := adapter.Execute(ctx, cmd, args)
	return err
}

// fixMissingUser membuat PPPoE secret di router untuk user yang hilang.
func (m *pppoeManager) fixMissingUser(
	ctx context.Context,
	adapter domain.RouterOSAdapter,
	cmdBuilder domain.CommandBuilder,
	dbUser *domain.PPPoEUser,
) error {
	// Dekripsi password untuk dikirim ke router
	password, err := m.crypto.Decrypt(dbUser.PasswordEncrypted)
	if err != nil {
		return fmt.Errorf("gagal dekripsi password user %s: %w", dbUser.Username, err)
	}

	secretParams := domain.PPPoESecretParams{
		Name:          dbUser.Username,
		Password:      password,
		Service:       dbUser.Service,
		Profile:       dbUser.ProfileName,
		RemoteAddress: dbUser.RemoteAddress,
		Comment:       dbUser.Comment,
	}

	cmd, args := cmdBuilder.CreateSecret(secretParams)
	_, execErr := adapter.Execute(ctx, cmd, args)
	return execErr
}

// updateUserSyncStatus memperbarui sync_status dan last_sync_at di DB.
func (m *pppoeManager) updateUserSyncStatus(
	ctx context.Context,
	userID string,
	status domain.SyncStatus,
	syncAt *time.Time,
) {
	if err := m.userRepo.UpdateSyncStatus(ctx, userID, status, syncAt); err != nil {
		m.logger.Warn().Err(err).Str("user_id", userID).Msg("gagal update sync status di DB")
	}
}
