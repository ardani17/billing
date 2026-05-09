// Paket main adalah titik masuk untuk service notification.
// Menginisialisasi konfigurasi, logger, koneksi database, Redis,
// dan menjalankan HTTP server menggunakan Fiber serta asynq worker.
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
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/ispboss/ispboss/pkg/database"
	"github.com/ispboss/ispboss/pkg/logger"
	"github.com/ispboss/ispboss/pkg/queue"
	"github.com/ispboss/ispboss/services/notification/internal/config"
	"github.com/ispboss/ispboss/services/notification/internal/domain"
	"github.com/ispboss/ispboss/services/notification/internal/handler"
	"github.com/ispboss/ispboss/services/notification/internal/provider"
	"github.com/ispboss/ispboss/services/notification/internal/repository"
	"github.com/ispboss/ispboss/services/notification/internal/usecase"
	"github.com/ispboss/ispboss/services/notification/internal/worker"
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
		Msg("memulai service notification")

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

	// Buat Fiber app dengan konfigurasi dasar
	app := fiber.New(fiber.Config{
		AppName:      cfg.AppName,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	})

	// Pasang middleware recovery untuk menangkap panic
	app.Use(recover.New())

	// --- Instantiate sqlc Queries ---
	queries := repository.New(dbPool)

	// --- Inisialisasi repositori ---
	configRepo := repository.NewConfigRepo(queries)
	templateRepo := repository.NewTemplateRepo(queries)
	logRepo := repository.NewLogRepo(queries)
	customerDataRepo := repository.NewCustomerDataRepo(queries)

	// --- Instantiate provider adapters ---
	// Credential diambil dari database saat runtime, bukan dari env vars.
	// Inisialisasi dengan string kosong - delivery pipeline menggunakan config dari DB.
	fonnteAdapter := provider.NewFonnteAdapter("", cfg.FonnteTimeout)
	zenzivaAdapter := provider.NewZenzivaAdapter("", "", cfg.ZenzivaTimeout)
	smtpAdapter := provider.NewSMTPAdapter("", 0, "", "", "", "")

	// --- Instantiate usecase components ---
	engine := domain.NewTemplateEngine()
	dedupChecker := usecase.NewDedupChecker(logRepo)
	quietHoursChecker := usecase.NewQuietHoursChecker()
	throttleChecker := usecase.NewThrottleChecker(logRepo)
	pipeline := usecase.NewDeliveryPipeline(
		configRepo, templateRepo, logRepo,
		customerDataRepo, customerDataRepo,
		engine, dedupChecker, quietHoursChecker, throttleChecker,
		fonnteAdapter, zenzivaAdapter, smtpAdapter,
		appLogger,
	)

	// --- Inisialisasi handler ---
	healthHandler := handler.NewHealthHandler(cfg.AppName, dbPool, redisClient)
	logHandler := handler.NewLogHandler(logRepo)
	configHandler := handler.NewConfigHandler(configRepo, templateRepo)
	templateHandler := handler.NewTemplateHandler(templateRepo)
	sendHandler := handler.NewSendHandler(pipeline)

	// Daftarkan semua route
	handler.RegisterRoutes(handler.RouterConfig{
		App:             app,
		HealthHandler:   healthHandler,
		LogHandler:      logHandler,
		ConfigHandler:   configHandler,
		TemplateHandler: templateHandler,
		SendHandler:     sendHandler,
		JWTSecret:       cfg.JWTSecret,
		Logger:          appLogger,
	})

	// --- Instantiate asynq Server dan EventConsumer ---
	asynqServer, err := queue.NewServer(queue.ClientConfig{
		Host:     cfg.RedisHost,
		Port:     cfg.RedisPort,
		Password: cfg.RedisPassword,
	}, cfg.WorkerConcurrency, cfg.QueuePriorities())
	if err != nil {
		appLogger.Fatal().Err(err).Msg("gagal membuat asynq server")
	}

	eventConsumer := worker.NewEventConsumer(pipeline, appLogger)
	mux := asynq.NewServeMux()
	eventConsumer.RegisterHandlers(mux)

	// Jalankan asynq worker di goroutine terpisah
	go func() {
		if err := asynqServer.Run(mux); err != nil {
			appLogger.Fatal().Err(err).Msg("gagal menjalankan asynq worker")
		}
	}()

	// Jalankan HTTP server di goroutine terpisah
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

	// Graceful shutdown: hentikan asynq worker terlebih dahulu
	asynqServer.Shutdown()

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
