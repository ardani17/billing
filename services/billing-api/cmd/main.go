// Package main adalah entry point untuk service billing-api.
// Menginisialisasi konfigurasi, logger, koneksi database, Redis,
// dan menjalankan HTTP server menggunakan Fiber.
//
// @title ISPBoss Billing API
// @version 1.0
// @description API untuk billing, pelanggan, invoice, dan pembayaran ISPBoss.
// @host localhost:3001
// @BasePath /api/v1
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/pkg/database"
	"github.com/ispboss/ispboss/pkg/logger"
	"github.com/ispboss/ispboss/services/billing-api/internal/config"
	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/ispboss/ispboss/services/billing-api/internal/handler"
	"github.com/ispboss/ispboss/services/billing-api/internal/middleware"
	"github.com/ispboss/ispboss/services/billing-api/internal/repository"
	"github.com/ispboss/ispboss/services/billing-api/internal/usecase"
	"github.com/ispboss/ispboss/services/billing-api/internal/worker"
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
		Msg("memulai service billing-api")

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

	// Buat asynq client untuk task queue (opsional, nil jika gagal)
	queueClient := createQueueClient(cfg, appLogger)
	if queueClient != nil {
		defer queueClient.Close()
	}

	// --- Instantiate sqlc Queries ---
	queries := repository.New(dbPool)

	// --- Instantiate repositories ---
	userRepo := repository.NewUserRepo(queries, dbPool)
	sessionRepo := repository.NewSessionRepo(queries)
	resellerSessionRepo := repository.NewResellerSessionRepo(dbPool)
	tokenRepo := repository.NewTokenRepo(queries)
	customerRepo := repository.NewCustomerRepo(queries, dbPool)
	areaRepo := repository.NewAreaRepo(queries)
	auditLogRepo := repository.NewAuditLogRepo(queries)
	packageRepo := repository.NewPackageRepo(queries, dbPool)
	resellerRepo := repository.NewResellerRepo(queries, dbPool)
	voucherRepo := repository.NewVoucherRepo(queries, dbPool)
	voucherAuditLogRepo := repository.NewVoucherAuditRepo(queries)
	resellerTxRepo := repository.NewResellerTxRepo(queries)

	// --- Instantiate invoice-related repositories ---
	invoiceRepo := repository.NewInvoiceRepo(queries, dbPool)
	invoiceItemRepo := repository.NewInvoiceItemRepo(queries)
	invoicePaymentRepo := repository.NewInvoicePaymentRepo(queries, dbPool)
	invoiceAuditLogRepo := repository.NewInvoiceAuditLogRepo(queries)
	invoiceSequenceRepo := repository.NewInvoiceSequenceRepo(queries)
	billingSettingsRepo := repository.NewBillingSettingsRepo(queries)
	recurringItemRepo := repository.NewRecurringItemRepo(queries)
	creditNoteRepo := repository.NewCreditNoteRepo(dbPool)
	debitNoteRepo := repository.NewDebitNoteRepo(dbPool)
	receiptSequenceRepo := repository.NewReceiptSequenceRepo(queries)

	// --- Instantiate pending sync repository ---
	pendingSyncRepo := repository.NewPendingSyncRepo(queries)

	// --- Instantiate reporting repositories ---
	expenseRepo := repository.NewExpenseRepo(queries, dbPool)
	expenseCategoryRepo := repository.NewExpenseCategoryRepo(queries, dbPool)
	kpiTargetRepo := repository.NewKPITargetRepo(queries, dbPool)
	reportScheduleRepo := repository.NewReportScheduleRepo(queries, dbPool)
	reportJobRepo := repository.NewReportJobRepo(queries, dbPool)
	customReportTemplateRepo := repository.NewCustomReportTemplateRepo(queries, dbPool)
	reportAggregationRepo := repository.NewReportAggregationRepo(queries, dbPool)

	// --- Instantiate gateway-related repositories ---
	gatewayConfigRepo := repository.NewGatewayConfigRepo(queries, dbPool)
	paymentLinkRepo := repository.NewPaymentLinkRepo(queries, dbPool)
	webhookLogRepo := repository.NewWebhookLogRepo(queries)

	// --- Instantiate rate limiter ---
	rateLimiter := middleware.NewLoginRateLimiter(
		redisClient,
		cfg.LoginMaxAttempts,
		cfg.LoginLockDuration,
	)

	// Rate limiter terpisah untuk login reseller (phone-based)
	resellerRateLimiter := middleware.NewLoginRateLimiter(
		redisClient,
		cfg.LoginMaxAttempts,
		cfg.LoginLockDuration,
	)

	// --- Instantiate usecases ---
	authUsecase := usecase.NewAuthUsecase(usecase.AuthUsecaseConfig{
		UserRepo:         userRepo,
		SessionRepo:      sessionRepo,
		TokenRepo:        tokenRepo,
		RateLimiter:      rateLimiter,
		QueueClient:      queueClient,
		Pool:             dbPool,
		RedisClient:      redisClient,
		JWTSecret:        cfg.JWTSecret,
		JWTExpiry:        cfg.JWTExpiry,
		JWTRefreshExpiry: cfg.JWTRefreshExpiry,
		BcryptCost:       cfg.BcryptCost,
		GoogleClientID:   cfg.GoogleClientID,
	})

	userManagementUsecase := usecase.NewUserManagementUsecase(usecase.UserManagementUsecaseConfig{
		UserRepo:    userRepo,
		SessionRepo: sessionRepo,
		TokenRepo:   tokenRepo,
		QueueClient: queueClient,
		BcryptCost:  cfg.BcryptCost,
	})

	impersonationUsecase := usecase.NewImpersonationUsecase(usecase.ImpersonationUsecaseConfig{
		UserRepo:  userRepo,
		JWTSecret: cfg.JWTSecret,
		JWTExpiry: cfg.JWTExpiry,
	})

	customerUsecase := usecase.NewCustomerUsecase(customerRepo, auditLogRepo, queueClient, appLogger)
	customerUsecase.SetPackageRepository(packageRepo)
	areaUsecase := usecase.NewAreaUsecase(areaRepo, auditLogRepo, appLogger)
	packageUsecase := usecase.NewPackageUsecase(packageRepo, auditLogRepo, queueClient, appLogger)

	// --- Instantiate reseller & voucher usecases ---
	resellerUsecase := usecase.NewResellerUsecase(resellerRepo, auditLogRepo, queueClient, appLogger)
	resellerActionUsecase := usecase.NewResellerActionUsecase(
		resellerRepo, voucherRepo, voucherAuditLogRepo, resellerTxRepo,
		auditLogRepo, resellerSessionRepo, dbPool, queries, queueClient, appLogger,
	)
	resellerAuthUsecase := usecase.NewResellerAuthUsecase(usecase.ResellerAuthUsecaseConfig{
		ResellerRepo: resellerRepo,
		SessionRepo:  resellerSessionRepo,
		RateLimiter:  resellerRateLimiter,
		JWTSecret:    cfg.JWTSecret,
		JWTExpiry:    cfg.JWTExpiry,
	}, appLogger)
	voucherUsecase := usecase.NewVoucherUsecase(voucherRepo, voucherAuditLogRepo, packageRepo, queueClient, appLogger)
	voucherActionUsecase := usecase.NewVoucherActionUsecase(voucherRepo, voucherAuditLogRepo, resellerRepo, appLogger)
	voucherPurchaseUsecase := usecase.NewVoucherPurchaseUsecase(
		resellerRepo, voucherRepo, voucherAuditLogRepo, packageRepo,
		resellerTxRepo, dbPool, queries, queueClient, appLogger,
	)
	voucherExpiryUsecase := usecase.NewVoucherExpiryUsecase(
		voucherRepo, voucherAuditLogRepo, resellerRepo, resellerTxRepo,
		dbPool, queries, appLogger,
	)
	voucherPrintUsecase := usecase.NewVoucherPrintUsecase(voucherRepo, packageRepo, appLogger)

	// --- Instantiate invoice-related usecases ---
	invoiceUsecase := usecase.NewInvoiceUsecase(
		invoiceRepo, invoiceItemRepo, invoicePaymentRepo, invoiceAuditLogRepo,
		invoiceSequenceRepo, billingSettingsRepo, customerRepo, packageRepo,
		dbPool, queueClient, appLogger,
	)
	invoiceActionUsecase := usecase.NewInvoiceActionUsecase(
		invoiceRepo, invoiceItemRepo, invoicePaymentRepo, invoiceAuditLogRepo,
		billingSettingsRepo, customerRepo, dbPool, queueClient, appLogger,
	)
	invoiceCronUsecase := usecase.NewInvoiceCronUsecase(
		invoiceRepo, invoiceItemRepo, invoiceAuditLogRepo, invoiceSequenceRepo,
		billingSettingsRepo, customerRepo, packageRepo, recurringItemRepo,
		dbPool, queueClient, appLogger,
	)
	recurringItemUsecase := usecase.NewRecurringItemUsecase(recurringItemRepo, customerRepo, appLogger)
	creditNoteUsecase := usecase.NewCreditNoteUsecase(
		creditNoteRepo, invoiceRepo, invoiceAuditLogRepo, invoiceSequenceRepo,
		customerRepo, queueClient, appLogger,
	)
	debitNoteUsecase := usecase.NewDebitNoteUsecase(
		debitNoteRepo, invoiceRepo, invoiceItemRepo, invoiceAuditLogRepo,
		invoiceSequenceRepo, customerRepo, billingSettingsRepo,
		queueClient, appLogger,
	)

	// --- Instantiate payment usecase ---
	paymentUsecase := usecase.NewPaymentUsecase(
		invoiceRepo, invoiceItemRepo, invoicePaymentRepo, invoiceAuditLogRepo,
		receiptSequenceRepo, billingSettingsRepo, customerRepo,
		dbPool, queueClient, appLogger,
	)

	// --- Parse master key untuk enkripsi gateway (opsional, log warning jika belum diisi) ---
	masterKeyBytes, mkErr := cfg.MasterKeyBytes()
	if mkErr != nil {
		appLogger.Warn().Err(mkErr).Msg("GATEWAY_MASTER_KEY belum diisi, fitur payment gateway dinonaktifkan")
	}

	// Parse webhook IPs dari konfigurasi
	xenditIPs, midtransIPs := cfg.ParseWebhookIPs()

	// --- Instantiate gateway usecases ---
	gatewayUsecase := usecase.NewGatewayUsecase(
		gatewayConfigRepo, paymentLinkRepo, invoiceRepo, customerRepo,
		billingSettingsRepo, dbPool, queueClient, masterKeyBytes, appLogger,
	)
	webhookUsecase := usecase.NewWebhookUsecase(
		webhookLogRepo, paymentLinkRepo, invoiceRepo, invoicePaymentRepo,
		invoiceAuditLogRepo, receiptSequenceRepo, customerRepo, gatewayConfigRepo,
		dbPool, queueClient, masterKeyBytes, appLogger,
	)

	// --- Instantiate isolir usecase ---
	isolirUsecase := usecase.NewIsolirUsecase(
		customerRepo, invoiceRepo, invoiceItemRepo, pendingSyncRepo,
		billingSettingsRepo, invoiceAuditLogRepo,
		dbPool, queueClient, appLogger,
	)

	// --- Instantiate network client untuk cross-service calls ---
	networkClient := usecase.NewNetworkClient(cfg.NetworkServiceURL, redisClient, appLogger)

	// --- Instantiate reporting usecases ---
	expenseManager := usecase.NewExpenseManager(expenseRepo, expenseCategoryRepo, appLogger)
	scheduleManager := usecase.NewScheduleManager(reportScheduleRepo, reportJobRepo, appLogger)
	forecastEngine := usecase.NewForecastEngine(reportAggregationRepo, kpiTargetRepo, appLogger)
	comparisonEngine := usecase.NewComparisonEngine(reportAggregationRepo, kpiTargetRepo, appLogger)
	customReportBuilder := usecase.NewCustomReportBuilder(reportAggregationRepo, customReportTemplateRepo, appLogger)
	dashboardCache := usecase.NewDashboardCache(
		reportAggregationRepo, networkClient, kpiTargetRepo, redisClient, appLogger,
	)
	kpiTargetUsecase := usecase.NewKPITargetUsecase(kpiTargetRepo, appLogger)

	reportManager := usecase.NewReportManager(
		reportAggregationRepo, expenseRepo, kpiTargetRepo,
		networkClient, redisClient, appLogger,
	)
	reportManager.SetExportManager(reportJobRepo, queueClient)
	reportManager.SetDashboardCache(dashboardCache)

	// --- Instantiate handlers ---
	healthHandler := handler.NewHealthHandler(cfg.AppName, dbPool, redisClient)
	authHandler := handler.NewAuthHandler(authUsecase, appLogger)
	userHandler := handler.NewUserHandler(userManagementUsecase, appLogger)
	sessionHandler := handler.NewSessionHandler(sessionRepo, appLogger)
	adminHandler := handler.NewAdminHandler(impersonationUsecase, dbPool, appLogger)
	customerHandler := handler.NewCustomerHandler(customerUsecase, appLogger)
	areaHandler := handler.NewAreaHandler(areaUsecase, appLogger)
	packageHandler := handler.NewPackageHandler(packageUsecase, appLogger)

	// --- Instantiate reseller & voucher handlers ---
	resellerHandler := handler.NewResellerHandler(resellerUsecase, appLogger)
	resellerActionHandler := handler.NewResellerActionHandler(resellerActionUsecase, appLogger)
	voucherHandler := handler.NewVoucherHandler(voucherUsecase, voucherActionUsecase, appLogger)
	voucherPrintHandler := handler.NewVoucherPrintHandler(voucherPrintUsecase, appLogger)
	resellerAuthHandler := handler.NewResellerAuthHandler(resellerAuthUsecase, appLogger)
	resellerDashboardHandler := handler.NewResellerDashboardHandler(
		resellerUsecase, voucherPurchaseUsecase, voucherUsecase,
		voucherPrintUsecase, resellerTxRepo, appLogger,
	)

	// --- Instantiate invoice-related handlers ---
	invoiceHandler := handler.NewInvoiceHandler(invoiceUsecase, appLogger)
	invoiceActionHandler := handler.NewInvoiceActionHandler(invoiceActionUsecase, appLogger)
	recurringItemHandler := handler.NewRecurringItemHandler(recurringItemUsecase, appLogger)
	creditNoteHandler := handler.NewCreditNoteHandler(creditNoteUsecase, appLogger)
	debitNoteHandler := handler.NewDebitNoteHandler(debitNoteUsecase, appLogger)

	// --- Instantiate payment handler ---
	paymentHandler := handler.NewPaymentHandler(paymentUsecase, appLogger)

	// --- Instantiate gateway handlers ---
	gatewayHandler := handler.NewGatewayHandler(gatewayUsecase, webhookLogRepo, paymentLinkRepo, appLogger)
	webhookHandler := handler.NewWebhookHandler(webhookLogRepo, queueClient, xenditIPs, midtransIPs, appLogger)

	// --- Instantiate isolir handler ---
	isolirHandler := handler.NewIsolirHandler(isolirUsecase, appLogger)

	// --- Instantiate reporting handlers ---
	reportHandler := handler.NewReportHandler(reportManager, appLogger)
	expenseHandler := handler.NewExpenseHandler(expenseManager, appLogger)
	exportHandler := handler.NewExportHandler(reportManager, appLogger)
	scheduleHandler := handler.NewScheduleHandler(scheduleManager, appLogger)
	kpiHandler := handler.NewKPIHandler(kpiTargetUsecase, appLogger)
	forecastHandler := handler.NewForecastHandler(forecastEngine, appLogger)
	comparisonHandler := handler.NewComparisonHandler(comparisonEngine, appLogger)
	customReportHandler := handler.NewCustomReportHandler(customReportBuilder, appLogger)
	dashboardHandler := handler.NewDashboardHandler(reportManager, appLogger)

	// Buat Fiber app dengan konfigurasi dasar
	app := fiber.New(fiber.Config{
		AppName:      cfg.AppName,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	})

	// Pasang middleware recovery untuk menangkap panic
	app.Use(recover.New())

	// Daftarkan semua route
	handler.RegisterRoutes(handler.RouterConfig{
		App:                      app,
		HealthHandler:            healthHandler,
		AuthHandler:              authHandler,
		UserHandler:              userHandler,
		SessionHandler:           sessionHandler,
		AdminHandler:             adminHandler,
		CustomerHandler:          customerHandler,
		AreaHandler:              areaHandler,
		PackageHandler:           packageHandler,
		ResellerHandler:          resellerHandler,
		ResellerActionHandler:    resellerActionHandler,
		VoucherHandler:           voucherHandler,
		VoucherPrintHandler:      voucherPrintHandler,
		ResellerAuthHandler:      resellerAuthHandler,
		ResellerDashboardHandler: resellerDashboardHandler,
		InvoiceHandler:           invoiceHandler,
		InvoiceActionHandler:     invoiceActionHandler,
		RecurringItemHandler:     recurringItemHandler,
		CreditNoteHandler:        creditNoteHandler,
		DebitNoteHandler:         debitNoteHandler,
		PaymentHandler:           paymentHandler,
		GatewayHandler:           gatewayHandler,
		WebhookHandler:           webhookHandler,
		IsolirHandler:            isolirHandler,
		ReportHandler:            reportHandler,
		ExpenseHandler:           expenseHandler,
		ExportHandler:            exportHandler,
		ScheduleHandler:          scheduleHandler,
		KPIHandler:               kpiHandler,
		ForecastHandler:          forecastHandler,
		ComparisonHandler:        comparisonHandler,
		CustomReportHandler:      customReportHandler,
		DashboardHandler:         dashboardHandler,
		RateLimiter:              rateLimiter,
		ResellerRateLimiter:      resellerRateLimiter,
		JWTSecret:                cfg.JWTSecret,
		Logger:                   appLogger,
	})

	// Jalankan server di goroutine terpisah
	addr := fmt.Sprintf(":%d", cfg.AppPort)
	go func() {
		if err := app.Listen(addr); err != nil {
			appLogger.Fatal().Err(err).Msg("gagal menjalankan server")
		}
	}()

	appLogger.Info().Str("addr", addr).Msg("server berjalan")

	// --- Jalankan asynq worker untuk task async ---
	redisOpt := asynq.RedisClientOpt{
		Addr:     fmt.Sprintf("%s:%d", cfg.RedisHost, cfg.RedisPort),
		Password: cfg.RedisPassword,
		DB:       0,
	}

	// Buat asynq server untuk memproses task
	asynqServer := asynq.NewServer(redisOpt, asynq.Config{
		Concurrency: 5,
		Queues: map[string]int{
			"default": 3,
			"low":     1,
		},
	})

	// Buat VoucherWorker dan daftarkan handler
	voucherWorker := worker.NewVoucherWorker(voucherUsecase, voucherExpiryUsecase, appLogger)
	mux := asynq.NewServeMux()
	voucherWorker.RegisterHandlers(mux)

	// Buat InvoiceWorker dan daftarkan handler
	invoiceWorker := worker.NewInvoiceWorker(invoiceCronUsecase, appLogger)
	invoiceWorker.RegisterHandlers(mux)

	// Buat GatewayWorker dan daftarkan handler
	gatewayWorker := worker.NewGatewayWorker(
		gatewayUsecase, webhookUsecase, paymentLinkRepo, webhookLogRepo,
		cfg.WebhookLogRetentionDays, appLogger,
	)
	gatewayWorker.RegisterHandlers(mux)

	// Buat IsolirWorker dan daftarkan handler
	isolirWorker := worker.NewIsolirWorker(isolirUsecase, appLogger)
	isolirWorker.RegisterHandlers(mux)

	// Buat ExportWorker dan daftarkan handler
	exportWorker := worker.NewExportWorker(reportManager, reportJobRepo, appLogger)
	exportWorker.RegisterHandlers(mux)

	// Buat ScheduleWorker dan daftarkan handler
	scheduleWorker := worker.NewScheduleWorker(
		reportScheduleRepo, reportManager, reportJobRepo, queueClient, appLogger,
	)
	scheduleWorker.RegisterHandlers(mux)

	// Buat RecurringExpenseWorker dan daftarkan handler
	recurringExpenseWorker := worker.NewRecurringExpenseWorker(expenseRepo, appLogger)
	recurringExpenseWorker.RegisterHandlers(mux)

	// Jalankan asynq server di goroutine terpisah
	go func() {
		if err := asynqServer.Run(mux); err != nil {
			appLogger.Error().Err(err).Msg("gagal menjalankan asynq server")
		}
	}()

	appLogger.Info().Msg("asynq worker berjalan")

	// Buat asynq scheduler untuk cron job expiry voucher (setiap hari jam 00:00)
	scheduler := asynq.NewScheduler(redisOpt, nil)

	// Daftarkan cron job expiry voucher — dijalankan setiap hari tengah malam
	expiryTask := asynq.NewTask(worker.TaskExpiryCron, nil)
	_, err = scheduler.Register("0 0 * * *", expiryTask)
	if err != nil {
		appLogger.Error().Err(err).Msg("gagal mendaftarkan cron expiry voucher")
	}

	// Daftarkan cron job invoice generate — dijalankan setiap hari jam 00:01
	invoiceGenerateTask := asynq.NewTask(worker.TaskInvoiceGenerateCron, nil)
	_, err = scheduler.Register("1 0 * * *", invoiceGenerateTask)
	if err != nil {
		appLogger.Error().Err(err).Msg("gagal mendaftarkan cron invoice generate")
	}

	// Daftarkan cron job invoice overdue — dijalankan setiap hari jam 00:05
	invoiceOverdueTask := asynq.NewTask(worker.TaskInvoiceOverdueCron, nil)
	_, err = scheduler.Register("5 0 * * *", invoiceOverdueTask)
	if err != nil {
		appLogger.Error().Err(err).Msg("gagal mendaftarkan cron invoice overdue")
	}

	// Daftarkan cron job expire payment links — dijalankan setiap jam
	expireLinksTask := asynq.NewTask(worker.TaskExpirePaymentLinks, nil)
	_, err = scheduler.Register("0 * * * *", expireLinksTask)
	if err != nil {
		appLogger.Error().Err(err).Msg("gagal mendaftarkan cron expire payment links")
	}

	// Daftarkan cron job cleanup webhook logs — dijalankan setiap hari jam 02:00
	cleanupWebhookTask := asynq.NewTask(worker.TaskCleanupWebhookLogs, nil)
	_, err = scheduler.Register("0 2 * * *", cleanupWebhookTask)
	if err != nil {
		appLogger.Error().Err(err).Msg("gagal mendaftarkan cron cleanup webhook logs")
	}

	if cfg.IsolirAutomationEnabled {
		// Daftarkan cron job auto-isolir — dijalankan setiap hari jam 01:00
		autoIsolirTask := asynq.NewTask(domain.TaskAutoIsolirCron, nil)
		_, err = scheduler.Register("0 1 * * *", autoIsolirTask)
		if err != nil {
			appLogger.Error().Err(err).Msg("gagal mendaftarkan cron auto-isolir")
		}

		// Daftarkan cron job suspend — dijalankan setiap hari jam 02:00
		suspendTask := asynq.NewTask(domain.TaskSuspendCron, nil)
		_, err = scheduler.Register("0 2 * * *", suspendTask)
		if err != nil {
			appLogger.Error().Err(err).Msg("gagal mendaftarkan cron suspend")
		}
	} else {
		appLogger.Info().Msg("cron auto-isolir dan suspend nonaktif; aksi jaringan berjalan manual/event")
	}

	if cfg.IsolirPeriodicSyncEnabled {
		// Daftarkan cron job periodic sync — dijalankan setiap 15 menit
		periodicSyncTask := asynq.NewTask(domain.TaskPeriodicSync, nil)
		_, err = scheduler.Register("*/15 * * * *", periodicSyncTask)
		if err != nil {
			appLogger.Error().Err(err).Msg("gagal mendaftarkan cron periodic sync")
		}
	} else {
		appLogger.Info().Msg("cron periodic sync isolir nonaktif; retry sync jaringan berjalan manual/event")
	}

	// Daftarkan cron job jadwal laporan harian — dijalankan setiap hari jam 07:00
	dailySchedulePayload, _ := json.Marshal(map[string]string{"schedule_type": "daily"})
	dailyScheduleTask := asynq.NewTask(worker.TaskScheduledReport, dailySchedulePayload)
	_, err = scheduler.Register("0 7 * * *", dailyScheduleTask)
	if err != nil {
		appLogger.Error().Err(err).Msg("gagal mendaftarkan cron jadwal laporan harian")
	}

	// Daftarkan cron job jadwal laporan mingguan — dijalankan setiap Senin jam 07:00
	weeklySchedulePayload, _ := json.Marshal(map[string]string{"schedule_type": "weekly"})
	weeklyScheduleTask := asynq.NewTask(worker.TaskScheduledReport, weeklySchedulePayload)
	_, err = scheduler.Register("0 7 * * 1", weeklyScheduleTask)
	if err != nil {
		appLogger.Error().Err(err).Msg("gagal mendaftarkan cron jadwal laporan mingguan")
	}

	// Daftarkan cron job jadwal laporan bulanan — dijalankan tanggal 1 setiap bulan jam 07:00
	monthlySchedulePayload, _ := json.Marshal(map[string]string{"schedule_type": "monthly"})
	monthlyScheduleTask := asynq.NewTask(worker.TaskScheduledReport, monthlySchedulePayload)
	_, err = scheduler.Register("0 7 1 * *", monthlyScheduleTask)
	if err != nil {
		appLogger.Error().Err(err).Msg("gagal mendaftarkan cron jadwal laporan bulanan")
	}

	// Daftarkan cron job pengeluaran berulang — dijalankan setiap hari jam 00:10
	recurringExpenseTask := asynq.NewTask(worker.TaskRecurringExpense, nil)
	_, err = scheduler.Register("10 0 * * *", recurringExpenseTask)
	if err != nil {
		appLogger.Error().Err(err).Msg("gagal mendaftarkan cron pengeluaran berulang")
	}

	// Daftarkan cron job cleanup report jobs lama — dijalankan setiap hari jam 03:00
	cleanupJobsTask := asynq.NewTask(worker.TaskCleanupReportJobs, nil)
	_, err = scheduler.Register("0 3 * * *", cleanupJobsTask)
	if err != nil {
		appLogger.Error().Err(err).Msg("gagal mendaftarkan cron cleanup report jobs")
	}

	// Jalankan scheduler di goroutine terpisah
	go func() {
		if err := scheduler.Run(); err != nil {
			appLogger.Error().Err(err).Msg("gagal menjalankan asynq scheduler")
		}
	}()

	appLogger.Info().Msg("asynq scheduler berjalan untuk cron billing/reporting; cron jaringan mengikuti flag konfigurasi")

	// Tunggu sinyal shutdown (SIGINT atau SIGTERM)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	appLogger.Info().Msg("menerima sinyal shutdown, menutup server...")

	// Graceful shutdown asynq server dan scheduler
	asynqServer.Shutdown()
	scheduler.Shutdown()

	// Graceful shutdown HTTP server dengan batas waktu 10 detik
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

// createQueueClient membuat asynq client untuk task queue.
// Mengembalikan nil jika gagal membuat client (non-fatal).
func createQueueClient(cfg *config.AppConfig, appLogger zerolog.Logger) *asynq.Client {
	client := asynq.NewClient(asynq.RedisClientOpt{
		Addr:     fmt.Sprintf("%s:%d", cfg.RedisHost, cfg.RedisPort),
		Password: cfg.RedisPassword,
		DB:       0,
	})
	return client
}
