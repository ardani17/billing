// Package usecase berisi implementasi business logic untuk network-service.
// File ini mengimplementasikan RouterUsecase untuk manajemen router MikroTik.
package usecase

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// AdapterFactory adalah fungsi factory untuk membuat instance RouterOSAdapter baru.
// Digunakan oleh usecase untuk membuat adapter sementara (test connection, create).
type AdapterFactory func() domain.RouterOSAdapter

// routerUsecase mengimplementasikan domain.RouterUsecase.
// Mengelola business logic CRUD router, test koneksi, reboot, dan status summary.
type routerUsecase struct {
	repo           domain.RouterRepository
	crypto         domain.CredentialEncryptor
	poolMgr        domain.PoolManager
	metrics        domain.MetricsStore
	events         domain.EventPublisher
	adapterFactory AdapterFactory
}

// NewRouterUsecase membuat instance RouterUsecase baru dengan semua dependensi.
func NewRouterUsecase(
	repo domain.RouterRepository,
	crypto domain.CredentialEncryptor,
	poolMgr domain.PoolManager,
	metrics domain.MetricsStore,
	events domain.EventPublisher,
	adapterFactory AdapterFactory,
) domain.RouterUsecase {
	return &routerUsecase{
		repo:           repo,
		crypto:         crypto,
		poolMgr:        poolMgr,
		metrics:        metrics,
		events:         events,
		adapterFactory: adapterFactory,
	}
}

// Create membuat router baru dan menyimpan credential terenkripsi.
// Test koneksi bersifat opt-in lewat TestOnCreate agar router tidak menerima
// login API saat admin hanya ingin mendaftarkan data koneksi.
func (uc *routerUsecase) Create(ctx context.Context, tenantID string, req domain.CreateRouterRequest) (*domain.RouterResponse, error) {
	// Enkripsi password sebelum disimpan
	encrypted, err := uc.crypto.Encrypt(req.Password)
	if err != nil {
		log.Error().Err(err).Msg("gagal enkripsi password saat create router")
		return nil, domain.ErrEncryptionFailed
	}

	// Set default values
	port := req.Port
	if port == 0 {
		port = 8728
	}
	serviceTypes := req.ServiceTypes
	if len(serviceTypes) == 0 {
		serviceTypes = []string{string(domain.ServicePPPoE)}
	}
	interval := req.HealthCheckIntervalSec
	if interval == 0 {
		interval = 60
	}

	router := &domain.Router{
		TenantID:               tenantID,
		Name:                   req.Name,
		Host:                   req.Host,
		Port:                   port,
		Username:               req.Username,
		PasswordEncrypted:      encrypted,
		UseSSL:                 req.UseSSL,
		ServiceTypes:           serviceTypes,
		Status:                 domain.StatusOffline,
		HealthCheckIntervalSec: interval,
		Notes:                  req.Notes,
	}

	// Simpan router ke database (status awal: offline)
	created, err := uc.repo.Create(ctx, router)
	if err != nil {
		return nil, err
	}

	if !req.TestOnCreate {
		return &domain.RouterResponse{
			Router:  created,
			Warning: "router disimpan sebagai offline; jalankan test koneksi manual untuk membaca info RouterOS",
		}, nil
	}

	// Test koneksi dan auto-detect info (best-effort)
	warning := ""
	sysRes, connErr := uc.tryConnect(ctx, req.Host, port, req.Username, req.Password, req.UseSSL)
	if connErr != nil {
		log.Warn().Err(connErr).Str("router_id", created.ID).Msg("gagal test koneksi saat create, router tetap disimpan sebagai offline")
		warning = "koneksi gagal: router disimpan sebagai offline"
	} else {
		// Update router dengan info dari system resource
		created.RouterOSVersion = sysRes.Version
		created.BoardName = sysRes.BoardName
		created.CPUCount = sysRes.CPUCount
		created.TotalRAMMB = int(sysRes.TotalRAM / (1024 * 1024))
		created.Identity = sysRes.Identity
		created.Status = domain.StatusOnline
		now := time.Now()
		created.LastOnlineAt = &now
		uptime := sysRes.Uptime
		created.LastUptimeSec = &uptime

		updated, updateErr := uc.repo.Update(ctx, created)
		if updateErr != nil {
			log.Error().Err(updateErr).Str("router_id", created.ID).Msg("gagal update info router setelah auto-detect")
		} else {
			created = updated
		}

		// Publish event router online (best-effort)
		_ = uc.events.PublishRouterOnline(ctx, created, 0)
	}

	return &domain.RouterResponse{
		Router:  created,
		Warning: warning,
	}, nil
}

// GetByID mengambil detail router. Jika online, ambil live metrics via adapter.
func (uc *routerUsecase) GetByID(ctx context.Context, id string) (*domain.RouterDetailResponse, error) {
	router, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	resp := &domain.RouterDetailResponse{
		Router: router,
	}

	// Jika router online, coba ambil live metrics dari metrics store
	if router.Status == domain.StatusOnline {
		latest, metricsErr := uc.metrics.GetLatest(ctx, router.ID)
		if metricsErr != nil {
			log.Warn().Err(metricsErr).Str("router_id", id).Msg("gagal ambil live metrics")
		} else if latest != nil {
			resp.LiveMetrics = &latest.Metrics
		}
	}

	return resp, nil
}

// Update memperbarui data router. Encrypt password jika berubah.
// Update pool jika host/port/credentials berubah.
func (uc *routerUsecase) Update(ctx context.Context, id string, req domain.UpdateRouterRequest) (*domain.RouterResponse, error) {
	router, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Validasi transisi status jika diminta
	if req.Status != "" {
		target := domain.RouterStatus(req.Status)
		if !domain.CanTransitionRouter(router.Status, target) {
			return nil, domain.ErrInvalidStatusTransition
		}
		router.Status = target
	}

	// Track apakah koneksi perlu di-reset (host/port/credentials berubah)
	needPoolReset := false

	if req.Name != "" {
		router.Name = req.Name
	}
	if req.Host != "" {
		if req.Host != router.Host {
			needPoolReset = true
		}
		router.Host = req.Host
	}
	if req.Port != nil {
		if *req.Port != router.Port {
			needPoolReset = true
		}
		router.Port = *req.Port
	}
	if req.Username != "" {
		if req.Username != router.Username {
			needPoolReset = true
		}
		router.Username = req.Username
	}
	if req.Password != "" {
		encrypted, encErr := uc.crypto.Encrypt(req.Password)
		if encErr != nil {
			log.Error().Err(encErr).Msg("gagal enkripsi password saat update router")
			return nil, domain.ErrEncryptionFailed
		}
		router.PasswordEncrypted = encrypted
		needPoolReset = true
	}
	if req.UseSSL != nil {
		if *req.UseSSL != router.UseSSL {
			needPoolReset = true
		}
		router.UseSSL = *req.UseSSL
	}
	if req.HealthCheckIntervalSec != nil {
		router.HealthCheckIntervalSec = *req.HealthCheckIntervalSec
	}
	if req.Notes != "" {
		router.Notes = req.Notes
	}

	updated, err := uc.repo.Update(ctx, router)
	if err != nil {
		return nil, err
	}

	// Reset pool jika konfigurasi koneksi berubah
	if needPoolReset {
		uc.poolMgr.ClosePool(id)
	}

	return &domain.RouterResponse{
		Router: updated,
	}, nil
}

// Delete melakukan soft-delete router dan menutup pool koneksi.
func (uc *routerUsecase) Delete(ctx context.Context, id string) error {
	if err := uc.repo.SoftDelete(ctx, id); err != nil {
		return err
	}
	// Tutup pool koneksi untuk router yang dihapus
	uc.poolMgr.ClosePool(id)
	return nil
}

// List mengambil daftar router dengan paginasi.
func (uc *routerUsecase) List(ctx context.Context, params domain.RouterListParams) (*domain.RouterListResult, error) {
	return uc.repo.List(ctx, params)
}

// TestConnection menguji koneksi ke router dan mengembalikan system info.
// Decrypt password → buat adapter sementara → connect → GetSystemResource → close.
func (uc *routerUsecase) TestConnection(ctx context.Context, id string) (*domain.SystemResource, error) {
	router, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Dekripsi password untuk koneksi
	password, err := uc.crypto.Decrypt(router.PasswordEncrypted)
	if err != nil {
		log.Error().Err(err).Str("router_id", id).Msg("gagal dekripsi password saat test connection")
		return nil, domain.ErrDecryptionFailed
	}

	sysRes, err := uc.tryConnect(ctx, router.Host, router.Port, router.Username, password, router.UseSSL)
	if err != nil {
		return nil, err
	}

	router.RouterOSVersion = sysRes.Version
	router.BoardName = sysRes.BoardName
	router.CPUCount = sysRes.CPUCount
	router.TotalRAMMB = int(sysRes.TotalRAM / (1024 * 1024))
	router.Identity = sysRes.Identity
	router.Status = domain.StatusOnline
	now := time.Now()
	router.LastOnlineAt = &now
	uptime := sysRes.Uptime
	router.LastUptimeSec = &uptime
	if _, updateErr := uc.repo.Update(ctx, router); updateErr != nil {
		log.Warn().Err(updateErr).Str("router_id", id).Msg("test koneksi berhasil tetapi gagal update metadata router")
	}

	return sysRes, nil
}

// Reboot mengirim perintah reboot ke router.
// Validasi: confirmation_name harus sama persis (case-sensitive) dengan router.Name.
func (uc *routerUsecase) Reboot(ctx context.Context, id string, confirmName string) error {
	router, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Validasi konfirmasi nama (case-sensitive)
	if confirmName != router.Name {
		return domain.ErrConfirmationMismatch
	}

	// Router harus online untuk bisa di-reboot
	if router.Status != domain.StatusOnline {
		return domain.ErrRouterOffline
	}

	// Dekripsi password untuk koneksi
	password, err := uc.crypto.Decrypt(router.PasswordEncrypted)
	if err != nil {
		log.Error().Err(err).Str("router_id", id).Msg("gagal dekripsi password saat reboot")
		return domain.ErrDecryptionFailed
	}

	// Buat adapter sementara dan kirim perintah reboot
	adapter := uc.adapterFactory()
	cfg := domain.ConnectionConfig{
		Host:           router.Host,
		Port:           router.Port,
		Username:       router.Username,
		Password:       password,
		UseSSL:         router.UseSSL,
		ConnectTimeout: 10 * time.Second,
		CommandTimeout: 10 * time.Second,
	}

	if connErr := adapter.Connect(ctx, cfg); connErr != nil {
		log.Error().Err(connErr).Str("router_id", id).Msg("gagal connect saat reboot")
		return domain.ErrConnectionFailed
	}
	defer func() { _ = adapter.Close() }()

	// Kirim perintah reboot ke router
	_, execErr := adapter.Execute(ctx, "/system/reboot", nil)
	if execErr != nil {
		log.Error().Err(execErr).Str("router_id", id).Msg("gagal execute reboot command")
		return execErr
	}

	return nil
}

// GetStatusSummary mengembalikan ringkasan status semua router tenant.
// Menghitung total dari penjumlahan semua status counts.
func (uc *routerUsecase) GetStatusSummary(ctx context.Context) (*domain.StatusSummary, error) {
	counts, err := uc.repo.CountByStatus(ctx)
	if err != nil {
		return nil, err
	}

	summary := &domain.StatusSummary{
		OnlineCount:      counts[domain.StatusOnline],
		OfflineCount:     counts[domain.StatusOffline],
		MaintenanceCount: counts[domain.StatusMaintenance],
	}
	// Total dihitung dari penjumlahan semua status
	summary.TotalRouters = summary.OnlineCount + summary.OfflineCount + summary.MaintenanceCount

	return summary, nil
}

// tryConnect membuat adapter sementara, connect, ambil system resource, lalu close.
// Digunakan oleh Create dan TestConnection.
func (uc *routerUsecase) tryConnect(ctx context.Context, host string, port int, username, password string, useSSL bool) (*domain.SystemResource, error) {
	adapter := uc.adapterFactory()
	cfg := domain.ConnectionConfig{
		Host:           host,
		Port:           port,
		Username:       username,
		Password:       password,
		UseSSL:         useSSL,
		ConnectTimeout: 10 * time.Second,
		CommandTimeout: 10 * time.Second,
	}

	if err := adapter.Connect(ctx, cfg); err != nil {
		_ = adapter.Close()
		return nil, domain.ErrConnectionFailed
	}
	defer func() { _ = adapter.Close() }()

	sysRes, err := adapter.GetSystemResource(ctx)
	if err != nil {
		return nil, domain.ErrConnectionFailed
	}

	return sysRes, nil
}
