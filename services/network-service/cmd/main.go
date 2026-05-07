// Paket main adalah titik masuk untuk service network-service.
// Menginisialisasi konfigurasi, logger, koneksi database, Redis,
// injeksi dependensi, pemeriksa kesehatan, dan menjalankan HTTP server menggunakan Fiber.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/hibiken/asynq"

	"github.com/ispboss/ispboss/pkg/database"
	"github.com/ispboss/ispboss/pkg/logger"
	"github.com/ispboss/ispboss/pkg/queue"
	"github.com/ispboss/ispboss/services/network-service/internal/config"
	"github.com/ispboss/ispboss/services/network-service/internal/crypto"
	"github.com/ispboss/ispboss/services/network-service/internal/domain"
	"github.com/ispboss/ispboss/services/network-service/internal/handler"
	"github.com/ispboss/ispboss/services/network-service/internal/metrics"
	fiberadapter "github.com/ispboss/ispboss/services/network-service/internal/modules/fiber/adapter"
	fiberhandler "github.com/ispboss/ispboss/services/network-service/internal/modules/fiber/handler"
	fiberusecase "github.com/ispboss/ispboss/services/network-service/internal/modules/fiber/usecase"
	fiberworker "github.com/ispboss/ispboss/services/network-service/internal/modules/fiber/worker"
	mikrotikadapter "github.com/ispboss/ispboss/services/network-service/internal/modules/mikrotik/adapter"
	mikrotikhandler "github.com/ispboss/ispboss/services/network-service/internal/modules/mikrotik/handler"
	mikrotikusecase "github.com/ispboss/ispboss/services/network-service/internal/modules/mikrotik/usecase"
	mikrotikworker "github.com/ispboss/ispboss/services/network-service/internal/modules/mikrotik/worker"
	"github.com/ispboss/ispboss/services/network-service/internal/pool"
	"github.com/ispboss/ispboss/services/network-service/internal/repository"
	"github.com/ispboss/ispboss/services/network-service/internal/usecase"
)

func main() {
	// Muat konfigurasi dari environment variables dan file .env
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("gagal memuat konfigurasi: %v", err)
	}

	// Inisialisasi logger dengan konfigurasi dari environment
	appLogger := logger.New(logger.Config{
		Level:       cfg.LogLevel,
		ServiceName: cfg.AppName,
		Pretty:      cfg.AppEnv == "development",
	})

	appLogger.Info().
		Str("env", cfg.AppEnv).
		Int("port", cfg.AppPort).
		Str("network_mode", cfg.NetworkMode).
		Msg("memulai service network-service")

	// Buat connection pool PostgreSQL
	dbPool, err := createDBPool(cfg)
	if err != nil {
		appLogger.Fatal().Err(err).Msg("gagal membuat koneksi database")
	}
	defer dbPool.Close()

	// Buat Redis client
	redisClient := createRedisClient(cfg)
	defer redisClient.Close()

	// Verifikasi koneksi Redis
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		appLogger.Fatal().Err(err).Msg("gagal koneksi ke Redis")
	}

	// --- Injeksi Dependensi Chain ---

	// 1. Credential encryptor (AES-256-GCM)
	keyBytes, err := cfg.EncryptionKeyBytes()
	if err != nil {
		appLogger.Fatal().Err(err).Msg("gagal membaca encryption key")
	}
	encryptor, err := crypto.NewAESEncryptor(keyBytes)
	if err != nil {
		appLogger.Fatal().Err(err).Msg("gagal membuat credential encryptor")
	}

	// 2. Adapter factory - membuat adapter baru sesuai NETWORK_MODE
	adapterFactory := func() domain.RouterOSAdapter {
		return mikrotikadapter.NewAdapter(cfg.NetworkMode)
	}

	// 3. Pool manager - mengelola connection pool per router
	poolMgr := pool.NewPoolManager(adapterFactory)

	// 4. Repositori - sqlc Queries membungkus DB pool
	queries := repository.New(dbPool)
	routerRepo := repository.NewRouterRepo(queries)
	pppoeUserRepo := repository.NewPPPoEUserRepo(queries)
	pppoeProfileRepo := repository.NewPPPoEProfileRepo(queries)
	dhcpBindingRepo := repository.NewDHCPBindingRepo(queries)
	mikrotikAuditRepo := repository.NewMikroTikAuditRepo(queries)
	routerBackupRepo := repository.NewRouterBackupRepo(queries)
	mikrotikBulkJobRepo := repository.NewMikroTikBulkJobRepo(queries)
	staticIPRepo := repository.NewStaticIPAssignmentRepo(queries)
	vpnTunnelRepo := repository.NewVPNTunnelRepo(queries)
	vpnSubnetRepo := repository.NewVPNSubnetRepo(queries)
	moduleEntitlementRepo := repository.NewModuleEntitlementRepo(dbPool)

	// 5. Metrics store - Redis uruted sets untuk time-series metrik
	metricsStore := metrics.NewRedisMetricsStore(redisClient)

	// 6. Event publisher - asynq client untuk terbitkan event ke Redis queue
	asynqClient, err := queue.NewClient(queue.ClientConfig{
		Host:     cfg.RedisHost,
		Port:     cfg.RedisPort,
		Password: cfg.RedisPassword,
		DB:       0,
	})
	if err != nil {
		appLogger.Fatal().Err(err).Msg("gagal membuat asynq client")
	}
	defer asynqClient.Close()
	eventPub := mikrotikusecase.NewEventPublisher(asynqClient)

	// 7. Router usecase - business logic utama
	routerUsecase := mikrotikusecase.NewRouterUsecase(
		routerRepo, encryptor, poolMgr, metricsStore, eventPub, adapterFactory,
	)

	// 8. Pemeriksa kesehatan - pemantauan periodik router
	healthChecker := mikrotikusecase.NewHealthChecker(
		routerRepo, poolMgr, metricsStore, eventPub, encryptor, adapterFactory,
	)

	// 9. PPPoE event publisher - terbitkan hasil operasi PPPoE ke queue
	pppoeEventPub := mikrotikusecase.NewPPPoEEventPublisher(asynqClient, appLogger)

	// 10. PPPoE manager - business logic manajemen PPPoE user
	pppoeManager := mikrotikusecase.NewPPPoEManager(
		pppoeUserRepo, pppoeProfileRepo, routerRepo,
		poolMgr, encryptor, pppoeEventPub,
		mikrotikadapter.NewCommandBuilder, appLogger,
	)

	// 11. PPPoE event worker - memproses event dari Billing API
	pppoeWorker := mikrotikworker.NewPPPoEEventWorker(pppoeManager, pppoeEventPub, appLogger)
	pppoeWorker.SetModuleChecker(moduleEntitlementRepo)

	// 12. Sync scheduler - periodic sync PPPoE user ke semua router
	syncScheduler := mikrotikusecase.NewSyncScheduler(pppoeManager, routerRepo, cfg.SyncIntervalMinutes, appLogger)

	// --- VPN Injeksi Dependensi ---

	// 13. VPN key generator - buat WireGuard key pair dan credential
	vpnKeyGen := mikrotikusecase.NewVPNKeyGenerator()

	// 14. VPN command builder factory - membangun perintah RouterOS sesuai versi router
	vpnCmdBuilderFactory := mikrotikadapter.NewVPNCommandBuilderForVersion

	// 15. VPN script generator - buat RouterOS script (.rsc) per protokol
	vpnScriptGen := mikrotikusecase.NewVPNScriptGenerator(mikrotikusecase.VPNScriptConfig{
		PrimaryEndpoint:          cfg.VPNServerEndpoint,
		SecondaryEndpoint:        cfg.VPNSecondaryEndpoint,
		ServerPublicKey:          cfg.VPNServerPublicKey,
		SecondaryServerPublicKey: cfg.VPNSecondaryServerPublicKey,
	})

	// 16. VPN bandwidth store - Redis uruted sets untuk bandwidth metrics per tunnel
	vpnBwStore := mikrotikusecase.NewVPNBandwidthStore(redisClient, appLogger)

	// 17. VPN event publisher - terbitkan event VPN ke Redis queue via asynq
	vpnEventPub := mikrotikusecase.NewVPNEventPublisher(asynqClient, appLogger)

	// 18. VPN manager - business logic manajemen VPN tunnel
	vpnManager := mikrotikusecase.NewVPNManager(
		vpnTunnelRepo, vpnSubnetRepo, routerRepo,
		poolMgr, encryptor, vpnKeyGen, vpnScriptGen,
		vpnEventPub, vpnCmdBuilderFactory, vpnBwStore,
		mikrotikusecase.VPNServerConfig{
			PrimaryEndpoint:          cfg.VPNServerEndpoint,
			SecondaryEndpoint:        cfg.VPNSecondaryEndpoint,
			ServerPublicKey:          cfg.VPNServerPublicKey,
			SecondaryServerPublicKey: cfg.VPNSecondaryServerPublicKey,
			ListenPort:               cfg.VPNListenPort,
		},
		appLogger,
	)

	// 19. VPN health monitor - pemantauan periodik VPN tunnel
	vpnHealthMonitor := mikrotikusecase.NewVPNHealthMonitor(
		vpnTunnelRepo, vpnEventPub, vpnBwStore, appLogger,
		cfg.VPNServerCapacityMbps, cfg.VPNServerEndpoint,
	)

	// 20. HTTP handlers (MikroTik)
	operationalManager := mikrotikusecase.NewMikroTikOperationalManager(routerRepo, encryptor, adapterFactory)
	dhcpManager := mikrotikusecase.NewDHCPManager(routerRepo, dhcpBindingRepo, mikrotikAuditRepo, encryptor, adapterFactory)
	staticIPManager := mikrotikusecase.NewStaticIPManager(routerRepo, staticIPRepo, mikrotikAuditRepo, encryptor, adapterFactory)
	walledGardenManager := mikrotikusecase.NewWalledGardenManager(routerRepo, mikrotikAuditRepo, encryptor, adapterFactory)
	hotspotManager := mikrotikusecase.NewHotspotManager(routerRepo, mikrotikAuditRepo, encryptor, adapterFactory)
	terminalManager := mikrotikusecase.NewTerminalManager(routerRepo, mikrotikAuditRepo, encryptor, adapterFactory)
	backupManager := mikrotikusecase.NewBackupManager(routerRepo, routerBackupRepo, mikrotikAuditRepo, encryptor, adapterFactory)
	mikrotikBulkManager := mikrotikusecase.NewMikroTikBulkManager(routerRepo, mikrotikBulkJobRepo, backupManager, pppoeManager)
	pppoeWorker.SetHotspotDependencies(hotspotManager, routerRepo)
	routerHandler := mikrotikhandler.NewRouterHandler(routerUsecase)
	statusHandler := mikrotikhandler.NewStatusHandler(routerUsecase)
	pppoeHandler := mikrotikhandler.NewPPPoEHandler(pppoeManager, appLogger)
	sessionHandler := mikrotikhandler.NewSessionHandler(pppoeManager, appLogger)
	vpnHandler := mikrotikhandler.NewVPNHandler(vpnManager, appLogger)
	operationalHandler := mikrotikhandler.NewMikroTikOperationalHandler(operationalManager, appLogger)
	dhcpHandler := mikrotikhandler.NewDHCPHandler(dhcpManager, appLogger)
	staticIPHandler := mikrotikhandler.NewStaticIPHandler(staticIPManager, appLogger)
	walledGardenHandler := mikrotikhandler.NewWalledGardenHandler(walledGardenManager, appLogger)
	hotspotHandler := mikrotikhandler.NewHotspotHandler(hotspotManager, appLogger)
	terminalHandler := mikrotikhandler.NewTerminalHandler(terminalManager, appLogger)
	backupHandler := mikrotikhandler.NewBackupHandler(backupManager, appLogger)
	mikrotikBulkHandler := mikrotikhandler.NewMikroTikBulkHandler(mikrotikBulkManager, appLogger)

	// --- OLT Injeksi Dependensi ---

	// 21. OLT Repositori - sqlc Queries untuk tabel olts, odps, olt_alarms
	oltRepo := repository.NewOLTRepo(queries)
	odpRepo := repository.NewODPRepo(queries)
	alarmRepo := repository.NewAlarmRepo(queries)

	// 22. SNMP Connector dan CLI Connector - koneksi ke OLT device
	snmpConnector := fiberadapter.NewSNMPConnector()
	cliConnector := fiberadapter.NewCLIConnector()

	// 23. OLT Adapter Factory - membuat adapter per brand sesuai NETWORK_MODE
	oltAdapterFactory := fiberadapter.NewOLTAdapterFactory(cfg.NetworkMode, snmpConnector, cliConnector)

	// 24. Signal Store dan Traffic Store - Redis time-series untuk OLT pemantauan
	signalStore := metrics.NewRedisSignalStore(redisClient)
	trafficStore := metrics.NewRedisTrafficStore(redisClient)

	// 25. OLT Event Publisher - terbitkan event OLT ke Redis queue via asynq
	oltEventPub := fiberusecase.NewOLTEventPublisher(asynqClient, appLogger)

	// 26. OLT Manager - business logic CRUD OLT, auto-detect, test connection
	oltManager := fiberusecase.NewOLTManager(
		oltRepo, odpRepo, alarmRepo,
		oltAdapterFactory, snmpConnector, cliConnector,
		encryptor, oltEventPub, signalStore, trafficStore,
	)

	// 27. ODP Manager - business logic CRUD ODP/splitter
	odpManager := fiberusecase.NewODPManager(odpRepo, oltRepo)

	// 28. OLT Pemeriksa Kesehatan - pemantauan periodik OLT via SNMP Ping
	oltHealthChecker := fiberusecase.NewOLTHealthChecker(
		oltRepo, oltAdapterFactory, encryptor, oltEventPub,
	)

	// 29. Alarm Manager - trap receiver + polling alarm OLT
	alarmManager := fiberusecase.NewAlarmManager(
		alarmRepo, oltRepo, oltAdapterFactory, encryptor, oltEventPub, cfg.SNMPTrapPort,
	)

	// 30. Sync Engine - periodic sync OLT data setiap 30 menit
	oltSyncEngine := fiberusecase.NewSyncEngineWithInterval(
		oltRepo, oltAdapterFactory, encryptor, signalStore, trafficStore,
		time.Duration(cfg.OLTSyncInterval)*time.Second,
	)
	oltSyncEngine.(interface{ SetImmediateSync(bool) }).SetImmediateSync(cfg.OLTSyncImmediateEnabled)

	// 31. Set pemeriksa kesehatan pada OLT manager (resolusi dependensi sirkular)
	oltManager.(interface{ SetHealthChecker(domain.OLTHealthChecker) }).SetHealthChecker(oltHealthChecker)

	// 32. HTTP handlers (OLT + ODP)
	oltHandler := fiberhandler.NewOLTHandler(oltManager, alarmManager)
	odpHandler := fiberhandler.NewODPHandler(odpManager)

	// --- Provisioning Injeksi Dependensi ---

	// 33. Provisioning Repositories - sqlc Queries untuk tabel onts, vlans, service_profiles, audit_logs, settings
	ontRepo := repository.NewONTRepo(queries)
	vlanRepo := repository.NewVLANRepo(queries)
	serviceProfileRepo := repository.NewServiceProfileRepo(queries)
	auditLogRepo := repository.NewAuditLogRepo(queries)
	provSettingsRepo := repository.NewProvisioningSettingsRepo(queries)
	oltSyncEngine.(interface{ SetONTRepository(domain.ONTRepository) }).SetONTRepository(ontRepo)

	// 34. VLAN Manager - business logic CRUD VLAN dan resolusi strategy
	vlanManager := fiberusecase.NewVLANManager(vlanRepo, oltRepo)

	// 35. Service Profile Manager - business logic CRUD service profile dan resolusi package mapping
	serviceProfileManager := fiberusecase.NewServiceProfileManager(serviceProfileRepo, oltRepo)

	// 36. Provisioning Manager - business logic provisioning ONT (single, bulk, decommission, reboot, auto)
	provisioningManager := fiberusecase.NewProvisioningManager(
		ontRepo, vlanRepo, serviceProfileRepo, auditLogRepo, provSettingsRepo,
		oltRepo, oltAdapterFactory, encryptor, oltEventPub,
		vlanManager, serviceProfileManager,
	)
	provisioningManager.(interface{ SetWriteEnabled(bool) }).SetWriteEnabled(cfg.OLTProvisioningWriteEnabled)

	// 37. HTTP handlers (Provisioning, VLAN, Service Profile)
	provisioningHandler := fiberhandler.NewProvisioningHandler(provisioningManager)
	vlanHandler := fiberhandler.NewVLANHandler(vlanManager)
	serviceProfileHandler := fiberhandler.NewServiceProfileHandler(serviceProfileManager)

	// 38. Provisioning Event Worker - memproses event customer.terminated untuk auto-decommission ONT
	provisioningWorker := fiberworker.NewProvisioningEventWorker(provisioningManager, appLogger)
	provisioningWorker.SetModuleChecker(moduleEntitlementRepo)

	// --- FTTH Visual Mapping Injeksi Dependensi ---

	// 39. Mapping Repositories - sqlc wrappers untuk tabel mapping
	mapNodeRepo := repository.NewMapNodeRepo(dbPool)
	cableRouteRepo := repository.NewCableRouteRepo(dbPool)
	nodePhotoRepo := repository.NewNodePhotoRepo(dbPool)
	changeHistoryRepo := repository.NewChangeHistoryRepo(dbPool)
	labelSettingsRepo := repository.NewLabelSettingsRepo(dbPool)
	shareLinkRepo := repository.NewShareLinkRepo(dbPool)
	geocodingCacheRepo := repository.NewGeocodingCacheRepo(dbPool)

	// 40. Mapping Usecases - business logic untuk FTTH mapping
	mapNodeManager := usecase.NewMapNodeManager(
		mapNodeRepo, nodePhotoRepo, changeHistoryRepo, labelSettingsRepo,
	)
	cableRouteManager := usecase.NewCableRouteManager(cableRouteRepo, mapNodeRepo)
	mapExportManager := usecase.NewMapExportManager(mapNodeRepo, cableRouteRepo)
	mapImportManager := usecase.NewMapImportManager(mapNodeRepo, cableRouteRepo)
	geocodingManager := usecase.NewGeocodingManager(geocodingCacheRepo, nil)
	shareManager := usecase.NewShareManager(shareLinkRepo, mapNodeRepo, cableRouteRepo)

	// 41. HTTP handlers (FTTH Mapping)
	mapNodeHandler := handler.NewMapNodeHandler(mapNodeManager)
	cableRouteHandler := handler.NewCableRouteHandler(cableRouteManager)
	searchHandler := handler.NewSearchHandler(mapNodeManager)
	exportHandler := handler.NewExportHandler(mapExportManager)
	importHandler := handler.NewImportHandler(mapImportManager)
	geocodingHandler := handler.NewGeocodingHandler(geocodingManager)
	shareHandler := handler.NewShareHandler(shareManager)
	lossCalcHandler := handler.NewLossCalcHandler()
	labelSettingsHandler := handler.NewLabelSettingsHandler(mapNodeManager)
	trashHandler := handler.NewTrashHandler(mapNodeManager)

	// Buat Fiber app dengan konfigurasi dasar
	app := fiber.New(fiber.Config{
		AppName:      cfg.AppName,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	})

	// Pasang middleware recovery untuk menangkap panic
	app.Use(recover.New())

	// Buat health handler dan daftarkan semua route
	healthHandler := handler.NewHealthHandler(cfg.AppName, dbPool, redisClient)

	handler.RegisterRoutes(handler.RouterConfig{
		App:                   app,
		HealthHandler:         healthHandler,
		RouterHandler:         routerHandler,
		StatusHandler:         statusHandler,
		PPPoEHandler:          pppoeHandler,
		SessionHandler:        sessionHandler,
		VPNHandler:            vpnHandler,
		OperationalHandler:    operationalHandler,
		DHCPHandler:           dhcpHandler,
		StaticIPHandler:       staticIPHandler,
		WalledGardenHandler:   walledGardenHandler,
		HotspotHandler:        hotspotHandler,
		TerminalHandler:       terminalHandler,
		BackupHandler:         backupHandler,
		BulkHandler:           mikrotikBulkHandler,
		OLTHandler:            oltHandler,
		ODPHandler:            odpHandler,
		ProvisioningHandler:   provisioningHandler,
		VLANHandler:           vlanHandler,
		ServiceProfileHandler: serviceProfileHandler,
		MapNodeHandler:        mapNodeHandler,
		CableRouteHandler:     cableRouteHandler,
		SearchHandler:         searchHandler,
		ExportHandler:         exportHandler,
		ImportHandler:         importHandler,
		GeocodingHandler:      geocodingHandler,
		ShareHandler:          shareHandler,
		LossCalcHandler:       lossCalcHandler,
		LabelSettingsHandler:  labelSettingsHandler,
		TrashHandler:          trashHandler,
		ModuleChecker:         moduleEntitlementRepo,
		JWTSecret:             cfg.JWTSecret,
		Logger:                appLogger,
	})

	// --- Asynq Server untuk memproses event dari Billing API ---
	asynqServer, err := queue.NewServer(queue.ClientConfig{
		Host:     cfg.RedisHost,
		Port:     cfg.RedisPort,
		Password: cfg.RedisPassword,
		DB:       0,
	}, 5, map[string]int{"default": 3, "critical": 6, "low": 1})
	if err != nil {
		appLogger.Fatal().Err(err).Msg("gagal membuat asynq server")
	}

	mux := asynq.NewServeMux()
	pppoeWorker.RegisterHandlers(mux)
	// OLT provisioning juga memakai event customer.terminated.
	// Untuk fase integrasi MikroTik live, biarkan PPPoE menjadi handler utama
	// sampai event fan-out lintas modul disiapkan.
	_ = provisioningWorker

	// Jalankan asynq worker di goroutine terpisah
	go func() {
		if err := asynqServer.Run(mux); err != nil {
			appLogger.Fatal().Err(err).Msg("gagal menjalankan asynq worker")
		}
	}()

	// Mulai sync scheduler PPPoE hanya jika diaktifkan.
	// Bawaan nonaktif supaya MikroTik tidak menerima login API berkala saat idle.
	if cfg.PPPoESyncSchedulerEnabled {
		syncScheduler.Start(context.Background())
		defer syncScheduler.Stop()
	} else {
		appLogger.Info().Msg("pppoe sync scheduler nonaktif; sync berjalan manual/event")
	}

	// Mulai pemeriksa kesehatan router hanya jika diaktifkan.
	// Tes koneksi tetap tersedia on-demand dari UI/API.
	if cfg.RouterHealthCheckEnabled {
		go func() {
			if err := healthChecker.Start(context.Background()); err != nil {
				appLogger.Error().Err(err).Msg("gagal memulai pemeriksa kesehatan")
			}
		}()
	} else {
		appLogger.Info().Msg("router pemeriksa kesehatan nonaktif; test koneksi berjalan on-demand")
	}

	// Mulai VPN health monitor di goroutine terpisah
	go func() {
		if err := vpnHealthMonitor.Start(context.Background()); err != nil {
			appLogger.Error().Err(err).Msg("gagal memulai vpn health monitor")
		}
	}()

	appLogger.Info().
		Bool("health_check_enabled", cfg.OLTHealthCheckEnabled).
		Bool("sync_enabled", cfg.OLTSyncEnabled).
		Bool("sync_immediate_enabled", cfg.OLTSyncImmediateEnabled).
		Bool("trap_enabled", cfg.OLTTrapEnabled).
		Bool("provisioning_write_enabled", cfg.OLTProvisioningWriteEnabled).
		Int("sync_interval_sec", cfg.OLTSyncInterval).
		Int("trap_port", cfg.SNMPTrapPort).
		Msg("olt runtime guard")

	if cfg.OLTHealthCheckEnabled {
		go func() {
			if err := oltHealthChecker.Start(context.Background()); err != nil {
				appLogger.Error().Err(err).Msg("gagal memulai OLT pemeriksa kesehatan")
			}
		}()
	} else {
		appLogger.Info().Msg("olt pemeriksa kesehatan nonaktif; test koneksi berjalan on-demand")
	}

	if cfg.OLTTrapEnabled {
		go func() {
			if err := alarmManager.StartTrapReceiver(context.Background()); err != nil {
				appLogger.Error().Err(err).Msg("gagal memulai SNMP trap receiver")
			}
		}()
	} else {
		appLogger.Info().Msg("snmp trap receiver olt nonaktif")
	}

	if cfg.OLTSyncEnabled {
		if err := oltSyncEngine.Start(context.Background()); err != nil {
			appLogger.Error().Err(err).Msg("gagal memulai OLT sync engine")
		}
	} else {
		appLogger.Info().Msg("olt sync engine nonaktif")
	}

	// Jalankan server di goroutine terpisah
	addr := fmt.Sprintf(":%d", cfg.AppPort)
	go func() {
		if err := app.Listen(addr); err != nil {
			appLogger.Fatal().Err(err).Msg("gagal menjalankan server")
		}
	}()

	appLogger.Info().Str("addr", addr).Msg("server berjalan")

	// Tunggu sinyal shutdown (SIGINT atau SIGTERM)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	appLogger.Info().Msg("menerima sinyal shutdown, menutup server...")

	// Graceful shutdown: hentikan semua background goroutine -> pool manager -> server
	asynqServer.Shutdown()
	healthChecker.Stop()
	vpnHealthMonitor.Stop()
	oltHealthChecker.Stop()
	alarmManager.StopTrapReceiver()
	oltSyncEngine.Stop()
	poolMgr.CloseAll()

	if err := app.ShutdownWithTimeout(10 * time.Second); err != nil {
		appLogger.Error().Err(err).Msg("gagal shutdown server dengan bersih")
	}

	appLogger.Info().Msg("server berhasil dihentikan")
}

// createDBPool membuat connection pool PostgreSQL menggunakan pkg/database.
func createDBPool(cfg *config.AppConfig) (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := database.NewPool(ctx, database.PoolConfig{
		DSN: cfg.DSN(),
	})
	if err != nil {
		return nil, fmt.Errorf("gagal membuat pool database: %w", err)
	}

	return pool, nil
}

// createRedisClient membuat Redis client dengan konfigurasi dari AppConfig.
func createRedisClient(cfg *config.AppConfig) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.RedisHost, cfg.RedisPort),
		Password: cfg.RedisPassword,
		DB:       0,
	})
}
