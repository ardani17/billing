// Package handler berisi HTTP handler untuk billing-api.
// Setiap handler menerima permintaan Fiber dan mengembalikan respons JSON.
package handler

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// HealthResponse adalah format respons untuk endpoint /healthz.
type HealthResponse struct {
	Status    string `json:"status"`
	Service   string `json:"service"`
	Timestamp string `json:"timestamp"`
}

// ReadyResponse adalah format respons untuk endpoint /readyz.
type ReadyResponse struct {
	Status       string            `json:"status"`
	Dependencies map[string]string `json:"dependencies"`
}

// HealthHandler menangani pemeriksaan kesehatan dan pemeriksaan kesiapan.
// Memeriksa konektivitas ke database PostgreSQL dan Redis.
type HealthHandler struct {
	serviceName string
	db          *pgxpool.Pool
	redis       *redis.Client
}

// NewHealthHandler membuat instance HealthHandler baru.
func NewHealthHandler(serviceName string, db *pgxpool.Pool, redisClient *redis.Client) *HealthHandler {
	return &HealthHandler{
		serviceName: serviceName,
		db:          db,
		redis:       redisClient,
	}
}

// Healthz mengembalikan status service (selalu 200 jika service berjalan).
// Endpoint ini tidak memeriksa dependency eksternal.
// @Summary Pemeriksaan kesehatan
// @Tags System
// @Success 200 {object} HealthResponse
// @Router /healthz [get]
func (h *HealthHandler) Healthz(c *fiber.Ctx) error {
	return c.Status(fiber.StatusOK).JSON(HealthResponse{
		Status:    "ok",
		Service:   h.serviceName,
		Timestamp: time.Now().Format(time.RFC3339),
	})
}

// Readyz memeriksa konektivitas ke database dan Redis.
// Mengembalikan 200 jika semua dependency reachable, 503 jika ada yang gagal.
// @Summary Pemeriksaan kesiapan
// @Tags System
// @Success 200 {object} ReadyResponse
// @Failure 503 {object} ReadyResponse
// @Router /readyz [get]
func (h *HealthHandler) Readyz(c *fiber.Ctx) error {
	// Batas waktu untuk pengecekan dependency
	ctx, cancel := context.WithTimeout(c.Context(), 3*time.Second)
	defer cancel()

	deps := make(map[string]string)
	allHealthy := true

	// Cek koneksi PostgreSQL
	if err := h.db.Ping(ctx); err != nil {
		deps["postgres"] = "unhealthy: " + err.Error()
		allHealthy = false
	} else {
		deps["postgres"] = "healthy"
	}

	// Cek koneksi Redis
	if err := h.redis.Ping(ctx).Err(); err != nil {
		deps["redis"] = "unhealthy: " + err.Error()
		allHealthy = false
	} else {
		deps["redis"] = "healthy"
	}

	// Tentukan status dan HTTP code berdasarkan hasil pengecekan
	status := "ready"
	statusCode := fiber.StatusOK
	if !allHealthy {
		status = "not_ready"
		statusCode = fiber.StatusServiceUnavailable
	}

	return c.Status(statusCode).JSON(ReadyResponse{
		Status:       status,
		Dependencies: deps,
	})
}
