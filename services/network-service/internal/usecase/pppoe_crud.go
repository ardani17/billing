// Package usecase berisi implementasi business logic untuk network-service.
// File ini berisi implementasi manual CRUD operations untuk PPPoE user:
// CreateUser, DeleteUser, ListUsers, GetSyncStatus.
// Dipanggil dari HTTP handler, bukan dari event worker.
package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/ispboss/ispboss/pkg/tenant"
	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// CreateUser membuat PPPoE user secara manual dari API.
// Flow: get router & pool → encrypt password → build command → execute → save DB.
// Jika use_simple_queue diaktifkan, juga membuat simple queue di router.
func (m *pppoeManager) CreateUser(ctx context.Context, routerID string, req domain.CreatePPPoEUserRequest) (*domain.PPPoEUser, error) {
	tenantID := tenant.FromContext(ctx)

	log := m.logger.With().
		Str("router_id", routerID).
		Str("customer_id", req.CustomerID).
		Str("username", req.Username).
		Str("tenant_id", tenantID).
		Logger()

	log.Info().Msg("membuat PPPoE user secara manual dari API")

	// Ambil router dan koneksi dari pool
	router, pool, adapter, err := m.getRouterAndPool(ctx, routerID, domain.PriorityMedium)
	if err != nil {
		log.Error().Err(err).Msg("gagal mendapatkan router dan koneksi pool")
		return nil, err
	}
	defer pool.Put(adapter)

	cmdBuilder := m.buildCommandBuilder(router)

	// Enkripsi password untuk disimpan di database
	encryptedPassword, err := m.crypto.Encrypt(req.Password)
	if err != nil {
		log.Error().Err(err).Msg("gagal enkripsi password PPPoE")
		return nil, fmt.Errorf("gagal enkripsi password: %w", err)
	}

	// Build comment untuk tracking ownership di router
	comment := domain.BuildComment(req.CustomerID, tenantID)

	// Build parameter PPPoE secret
	secretParams := domain.PPPoESecretParams{
		Name:          req.Username,
		Password:      req.Password,
		Service:       "pppoe",
		Profile:       req.ProfileName,
		RemoteAddress: req.RemoteAddress,
		Comment:       comment,
	}

	// Execute CreateSecret di router
	cmd, args := cmdBuilder.CreateSecret(secretParams)
	log.Info().Str("command", cmd).Str("profile", req.ProfileName).Msg("menjalankan CreateSecret di router")

	_, execErr := adapter.Execute(ctx, cmd, args)
	if execErr != nil {
		log.Error().Err(execErr).Msg("gagal membuat PPPoE secret di router")
		return nil, fmt.Errorf("gagal membuat PPPoE secret di router %s: %w", routerID, execErr)
	}

	// Buat simple queue jika diaktifkan
	if req.UseSimpleQueue && req.RemoteAddress != "" {
		queueParams := domain.SimpleQueueParams{
			Name:    req.Username,
			Target:  req.RemoteAddress,
			Comment: comment,
		}
		queueCmd, queueArgs := cmdBuilder.CreateSimpleQueue(queueParams)
		log.Info().Str("command", queueCmd).Msg("membuat simple queue di router")

		if _, queueErr := adapter.Execute(ctx, queueCmd, queueArgs); queueErr != nil {
			log.Warn().Err(queueErr).Msg("gagal membuat simple queue (lanjut tanpa queue)")
		}
	}

	// Simpan PPPoE user ke database dengan sync_status "synced"
	now := time.Now()
	pppoeUser := &domain.PPPoEUser{
		TenantID:          tenantID,
		CustomerID:        req.CustomerID,
		RouterID:          routerID,
		Username:          req.Username,
		PasswordEncrypted: encryptedPassword,
		ProfileName:       req.ProfileName,
		Service:           "pppoe",
		RemoteAddress:     req.RemoteAddress,
		Comment:           comment,
		Disabled:          false,
		UseSimpleQueue:    req.UseSimpleQueue,
		Status:            "active",
		SyncStatus:        domain.SyncStatusSynced,
		LastSyncAt:        &now,
	}

	created, saveErr := m.userRepo.Create(ctx, pppoeUser)
	if saveErr != nil {
		log.Error().Err(saveErr).Msg("gagal menyimpan PPPoE user ke database")
		return nil, fmt.Errorf("gagal menyimpan PPPoE user ke database: %w", saveErr)
	}

	log.Info().Str("user_id", created.ID).Msg("PPPoE user berhasil dibuat")
	return created, nil
}

// UpdateUser mengubah PPPoE user di router dan database.
// Flow: get user -> get router & pool -> /ppp/secret/set -> update DB.
func (m *pppoeManager) UpdateUser(ctx context.Context, routerID, userID string, req domain.UpdatePPPoEUserRequest) (*domain.PPPoEUser, error) {
	log := m.logger.With().
		Str("router_id", routerID).
		Str("user_id", userID).
		Logger()

	pppoeUser, err := m.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil PPPoE user: %w", err)
	}
	if pppoeUser.RouterID != routerID {
		return nil, domain.ErrPPPoEUserNotFound
	}

	router, pool, adapter, err := m.getRouterAndPool(ctx, routerID, domain.PriorityMedium)
	if err != nil {
		log.Error().Err(err).Msg("gagal mendapatkan router dan koneksi pool")
		return nil, err
	}
	defer pool.Put(adapter)

	cmdBuilder := m.buildCommandBuilder(router)
	params := make(map[string]string)

	if req.Password != nil {
		encryptedPassword, err := m.crypto.Encrypt(*req.Password)
		if err != nil {
			log.Error().Err(err).Msg("gagal enkripsi password PPPoE")
			return nil, fmt.Errorf("gagal enkripsi password: %w", err)
		}
		pppoeUser.PasswordEncrypted = encryptedPassword
		params["=password"] = *req.Password
	}
	if req.ProfileName != nil {
		pppoeUser.ProfileName = *req.ProfileName
		params["=profile"] = *req.ProfileName
	}
	if req.RemoteAddress != nil {
		pppoeUser.RemoteAddress = *req.RemoteAddress
		params["=remote-address"] = *req.RemoteAddress
	}
	if req.Disabled != nil {
		pppoeUser.Disabled = *req.Disabled
		if *req.Disabled {
			params["=disabled"] = "yes"
			pppoeUser.Status = "disabled"
		} else {
			params["=disabled"] = "no"
			pppoeUser.Status = "active"
		}
	}
	if req.UseSimpleQueue != nil {
		pppoeUser.UseSimpleQueue = *req.UseSimpleQueue
	}

	if len(params) > 0 {
		cmd, args := cmdBuilder.SetSecret(pppoeUser.Username, params)
		log.Info().Str("command", cmd).Msg("menjalankan SetSecret di router")
		if _, execErr := adapter.Execute(ctx, cmd, args); execErr != nil {
			log.Error().Err(execErr).Msg("gagal update PPPoE secret di router")
			return nil, fmt.Errorf("gagal update PPPoE secret di router %s: %w", routerID, execErr)
		}
		if req.Disabled != nil && *req.Disabled {
			if disconnectErr := m.disconnectActiveSessionByUsername(ctx, adapter, cmdBuilder, pppoeUser.Username, log); disconnectErr != nil {
				log.Warn().Err(disconnectErr).Msg("gagal disconnect session setelah disable PPPoE user")
			}
		}
	}

	now := time.Now()
	pppoeUser.SyncStatus = domain.SyncStatusSynced
	pppoeUser.LastSyncAt = &now

	updated, saveErr := m.userRepo.Update(ctx, pppoeUser)
	if saveErr != nil {
		log.Error().Err(saveErr).Msg("gagal menyimpan update PPPoE user ke database")
		return nil, fmt.Errorf("gagal menyimpan update PPPoE user: %w", saveErr)
	}

	return updated, nil
}

// DeleteUser menghapus PPPoE user dari router dan soft-delete dari database.
// Flow: get user → get router & pool → disconnect → remove secret → remove queue → remove firewall → soft-delete DB.
func (m *pppoeManager) DeleteUser(ctx context.Context, routerID, userID string) error {
	correlationID := uuid.New().String()

	log := m.logger.With().
		Str("router_id", routerID).
		Str("user_id", userID).
		Str("correlation_id", correlationID).
		Logger()

	log.Info().Msg("menghapus PPPoE user dari router dan database")

	// Ambil PPPoE user dari DB
	pppoeUser, err := m.userRepo.GetByID(ctx, userID)
	if err != nil {
		log.Error().Err(err).Msg("gagal mengambil PPPoE user dari database")
		return fmt.Errorf("gagal mengambil PPPoE user: %w", err)
	}

	// Ambil router dan koneksi dari pool
	router, pool, adapter, err := m.getRouterAndPool(ctx, routerID, domain.PriorityMedium)
	if err != nil {
		log.Error().Err(err).Msg("gagal mendapatkan router dan koneksi pool")
		return err
	}
	defer pool.Put(adapter)

	cmdBuilder := m.buildCommandBuilder(router)

	// Step 1: Disconnect active session
	log.Info().Msg("step 1: disconnect active session")
	if disconnectErr := m.disconnectActiveSessionByUsername(ctx, adapter, cmdBuilder, pppoeUser.Username, log); disconnectErr != nil {
		log.Warn().Err(disconnectErr).Msg("gagal disconnect session (mungkin tidak ada session aktif)")
	}

	// Step 2: Remove PPPoE secret dari router
	cmd, args := cmdBuilder.RemoveSecret(pppoeUser.Username)
	log.Info().Str("command", cmd).Msg("step 2: remove PPPoE secret")

	if _, execErr := adapter.Execute(ctx, cmd, args); execErr != nil {
		log.Error().Err(execErr).Msg("gagal remove PPPoE secret di router")
		return fmt.Errorf("gagal remove PPPoE secret di router %s: %w", routerID, execErr)
	}

	// Step 3: Remove simple queue jika ada
	removeQueueCmd, removeQueueArgs := cmdBuilder.RemoveSimpleQueue(pppoeUser.Username)
	log.Info().Str("command", removeQueueCmd).Msg("step 3: remove simple queue")
	if _, queueErr := adapter.Execute(ctx, removeQueueCmd, removeQueueArgs); queueErr != nil {
		log.Warn().Err(queueErr).Msg("gagal hapus simple queue (mungkin tidak ada)")
	}

	// Step 4: Remove firewall rules by comment
	isolirComment := fmt.Sprintf("ISPBoss:isolir:%s", pppoeUser.CustomerID)
	m.removeNATRuleSafe(ctx, adapter, cmdBuilder, isolirComment, log)

	dnsComment := fmt.Sprintf("ISPBoss:dns-redirect:%s", pppoeUser.CustomerID)
	m.removeNATRuleSafe(ctx, adapter, cmdBuilder, dnsComment, log)

	// Soft-delete dari database
	if deleteErr := m.userRepo.SoftDelete(ctx, userID); deleteErr != nil {
		log.Error().Err(deleteErr).Msg("gagal soft-delete PPPoE user di database")
		return fmt.Errorf("gagal soft-delete PPPoE user: %w", deleteErr)
	}

	log.Info().Msg("PPPoE user berhasil dihapus dari router dan database")
	return nil
}

// ListUsers mengambil daftar PPPoE user dari database dengan paginasi.
func (m *pppoeManager) ListUsers(ctx context.Context, routerID string, params domain.PPPoEUserListParams) (*domain.PPPoEUserListResult, error) {
	params.RouterID = routerID
	return m.userRepo.List(ctx, params)
}

// GetSyncStatus mengambil ringkasan sync status untuk satu router dari database.
func (m *pppoeManager) GetSyncStatus(ctx context.Context, routerID string) (*domain.SyncStatusSummary, error) {
	return m.userRepo.GetSyncStatusSummary(ctx, routerID)
}
