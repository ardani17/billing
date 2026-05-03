// Package usecase berisi implementasi business logic untuk network-service.
// File ini mendefinisikan PPPoEManager interface dan struct pppoeManager
// beserta constructor dan helper methods.
package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// =============================================================================
// PPPoEManager Interface — business logic untuk manajemen PPPoE user
// =============================================================================

// PPPoEManager mendefinisikan business logic untuk manajemen PPPoE user.
// Menangani lifecycle lengkap: create, isolir, un-isolir, suspend, package change, sync.
type PPPoEManager interface {
	// HandleCustomerActivated membuat PPPoE user di router saat pelanggan diaktivasi.
	HandleCustomerActivated(ctx context.Context, payload domain.CustomerActivatedPayload) error

	// HandleIsolir menjalankan sequence isolir: disable user, disconnect, add firewall.
	HandleIsolir(ctx context.Context, payload domain.CustomerIsolirPayload) error

	// HandleUnIsolir menjalankan sequence buka isolir: enable user, remove firewall.
	HandleUnIsolir(ctx context.Context, payload domain.CustomerUnIsolirPayload) error

	// HandleSuspend menjalankan sequence suspend: disconnect, remove user, remove queue, remove firewall.
	HandleSuspend(ctx context.Context, payload domain.CustomerSuspendPayload) error

	// HandlePackageChanged mengupdate profile assignment dan reconnect user.
	HandlePackageChanged(ctx context.Context, payload domain.PackageChangedPayload) error

	// SyncRouter menjalankan sinkronisasi database↔router untuk satu router.
	SyncRouter(ctx context.Context, routerID string) (*domain.SyncResult, error)

	// GetActiveSessions mengambil active PPPoE sessions dari router.
	GetActiveSessions(ctx context.Context, routerID string) ([]domain.PPPoESession, error)

	// DisconnectSession memutus satu active session di router.
	DisconnectSession(ctx context.Context, routerID, sessionID string) error

	// DisconnectUser memutus active session milik satu PPPoE user terkelola.
	DisconnectUser(ctx context.Context, routerID, userID string) error

	// GetSessionCount mengambil jumlah active sessions di router.
	GetSessionCount(ctx context.Context, routerID string) (int, error)

	// CreateUser membuat PPPoE user secara manual (dari API).
	CreateUser(ctx context.Context, routerID string, req domain.CreatePPPoEUserRequest) (*domain.PPPoEUser, error)

	// UpdateUser mengubah PPPoE user secara manual (dari API).
	UpdateUser(ctx context.Context, routerID, userID string, req domain.UpdatePPPoEUserRequest) (*domain.PPPoEUser, error)

	// DeleteUser menghapus PPPoE user dari router dan database.
	DeleteUser(ctx context.Context, routerID, userID string) error

	// ListUsers mengambil daftar PPPoE user dari database.
	ListUsers(ctx context.Context, routerID string, params domain.PPPoEUserListParams) (*domain.PPPoEUserListResult, error)

	// GetSyncStatus mengambil ringkasan status sync untuk satu router.
	GetSyncStatus(ctx context.Context, routerID string) (*domain.SyncStatusSummary, error)

	// SyncProfile membuat atau update PPPoE profile di semua router.
	SyncProfile(ctx context.Context, profile *domain.PPPoEProfile) error
}

// =============================================================================
// pppoeManager — implementasi PPPoEManager
// =============================================================================

// pppoeManager mengimplementasikan PPPoEManager.
// Mengelola lifecycle PPPoE user: create, isolir, un-isolir, suspend, sync.
type pppoeManager struct {
	userRepo          domain.PPPoEUserRepository
	profileRepo       domain.PPPoEProfileRepository
	routerRepo        domain.RouterRepository
	poolManager       domain.PoolManager
	crypto            domain.CredentialEncryptor
	eventPub          domain.PPPoEEventPublisher
	cmdBuilderFactory func(routerOSVersion string) domain.CommandBuilder
	logger            zerolog.Logger
}

// NewPPPoEManager membuat instance PPPoEManager baru dengan semua dependensi.
func NewPPPoEManager(
	userRepo domain.PPPoEUserRepository,
	profileRepo domain.PPPoEProfileRepository,
	routerRepo domain.RouterRepository,
	poolManager domain.PoolManager,
	crypto domain.CredentialEncryptor,
	eventPub domain.PPPoEEventPublisher,
	cmdBuilderFactory func(routerOSVersion string) domain.CommandBuilder,
	logger zerolog.Logger,
) PPPoEManager {
	return &pppoeManager{
		userRepo:          userRepo,
		profileRepo:       profileRepo,
		routerRepo:        routerRepo,
		poolManager:       poolManager,
		crypto:            crypto,
		eventPub:          eventPub,
		cmdBuilderFactory: cmdBuilderFactory,
		logger:            logger,
	}
}

// =============================================================================
// Helper Methods
// =============================================================================

// getRouterAndPool mengambil router dari repository, mendekripsi password,
// dan mendapatkan koneksi dari pool. Mengembalikan router, pool, adapter, dan error.
// Caller bertanggung jawab untuk memanggil pool.Put(adapter) setelah selesai.
func (m *pppoeManager) getRouterAndPool(
	ctx context.Context,
	routerID string,
	priority domain.CommandPriority,
) (*domain.Router, domain.ConnPool, domain.RouterOSAdapter, error) {
	router, err := m.routerRepo.GetByID(ctx, routerID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("gagal ambil router %s: %w", routerID, err)
	}

	// Dekripsi password untuk konfigurasi koneksi
	password, err := m.crypto.Decrypt(router.PasswordEncrypted)
	if err != nil {
		m.logger.Error().Err(err).Str("router_id", routerID).Msg("gagal dekripsi password router")
		return nil, nil, nil, domain.ErrDecryptionFailed
	}

	cfg := domain.ConnectionConfig{
		Host:           router.Host,
		Port:           router.Port,
		Username:       router.Username,
		Password:       password,
		UseSSL:         router.UseSSL,
		ConnectTimeout: 10 * time.Second,
		CommandTimeout: 10 * time.Second,
	}

	// Ambil koneksi dari pool dengan priority yang diberikan
	pool := m.poolManager.GetPool(routerID, cfg)
	adapter, err := pool.Get(ctx, priority)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("gagal ambil koneksi pool untuk router %s: %w", routerID, err)
	}

	return router, pool, adapter, nil
}

// buildCommandBuilder membuat CommandBuilder sesuai versi RouterOS router.
func (m *pppoeManager) buildCommandBuilder(router *domain.Router) domain.CommandBuilder {
	return m.cmdBuilderFactory(router.RouterOSVersion)
}

// =============================================================================
// Implementasi di file terpisah
// =============================================================================

// HandleIsolir — implementasi di pppoe_manager_isolir.go
// HandleUnIsolir — implementasi di pppoe_manager_unisolir.go
// HandleSuspend — implementasi di pppoe_manager_suspend.go
// HandlePackageChanged — implementasi di pppoe_manager_package.go
// SyncRouter — implementasi di pppoe_sync.go
// GetActiveSessions — implementasi di pppoe_sessions.go
// DisconnectSession — implementasi di pppoe_sessions.go
// GetSessionCount — implementasi di pppoe_sessions.go
// SyncProfile — implementasi di pppoe_profile_sync.go
// CreateUser — implementasi di pppoe_crud.go
// DeleteUser — implementasi di pppoe_crud.go
// ListUsers — implementasi di pppoe_crud.go
// GetSyncStatus — implementasi di pppoe_crud.go
