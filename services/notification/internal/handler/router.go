package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/ispboss/ispboss/services/notification/internal/middleware"
	"github.com/rs/zerolog"
)

// RouterConfig berisi dependensi yang dibutuhkan untuk registrasi route.
type RouterConfig struct {
	// App adalah instance Fiber application
	App *fiber.App

	// HealthHandler adalah handler untuk health cek endpoint
	HealthHandler *HealthHandler

	// LogHandler adalah handler untuk notification logs
	LogHandler *LogHandler

	// ConfigHandler adalah handler untuk konfigurasi notifikasi
	ConfigHandler *ConfigHandler

	// TemplateHandler adalah handler untuk manajemen template notifikasi
	TemplateHandler *TemplateHandler

	// SendHandler adalah handler untuk pengiriman notifikasi (test, manual, resend)
	SendHandler *SendHandler

	// JWTSecret adalah secret key untuk validasi JWT token
	JWTSecret string

	// Logger adalah instance zerolog untuk permintaan logging
	Logger zerolog.Logger
}

// RegisterRoutes mendaftarkan semua route pada Fiber app.
// Health cek endpoint bersifat publik (tanpa auth).
// Route lainnya dilindungi oleh auth dan tenant middleware.
func RegisterRoutes(cfg RouterConfig) {
	// Middleware logging untuk semua permintaan
	cfg.App.Use(middleware.RequestLogger(cfg.Logger))

	// Route publik - health cek (tanpa autentikasi)
	cfg.App.Get("/healthz", cfg.HealthHandler.Healthz)
	cfg.App.Get("/readyz", cfg.HealthHandler.Readyz)

	// Grup route yang dilindungi oleh auth dan tenant middleware
	api := cfg.App.Group("/api/v1")
	api.Use(middleware.Auth(cfg.JWTSecret))
	api.Use(middleware.TenantContext(cfg.JWTSecret))

	// --- Notification log routes ---
	api.Get("/notifications/logs", cfg.LogHandler.List)
	api.Get("/notifications/logs/:id", cfg.LogHandler.GetByID)

	// --- Notification config routes ---
	api.Get("/notifications/config", cfg.ConfigHandler.Get)
	api.Put("/notifications/config", cfg.ConfigHandler.Update)
	api.Put("/notifications/config/settings", cfg.ConfigHandler.UpdateSettings)

	// --- Notification template routes ---
	api.Get("/notifications/templates", cfg.TemplateHandler.List)
	api.Post("/notifications/templates", cfg.TemplateHandler.Create)
	api.Put("/notifications/templates/:id", cfg.TemplateHandler.Update)
	api.Delete("/notifications/templates/:id", cfg.TemplateHandler.Delete)

	// --- Notification send routes ---
	api.Post("/notifications/test", cfg.SendHandler.TestSend)
	api.Post("/notifications/send", cfg.SendHandler.ManualSend)
	api.Post("/notifications/logs/:id/resend", cfg.SendHandler.Resend)
}
