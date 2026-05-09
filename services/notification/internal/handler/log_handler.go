// Package handler berisi HTTP handler untuk notification service.
package handler

import (
	"errors"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/ispboss/ispboss/services/notification/internal/domain"
)

// LogHandler menangani HTTP permintaan untuk notification logs.
// Menyediakan endpoint untuk melihat daftar log dan detail log.
type LogHandler struct {
	logRepo domain.LogRepository
}

// NewLogHandler membuat instance LogHandler baru dengan dependensi LogRepository.
func NewLogHandler(logRepo domain.LogRepository) *LogHandler {
	return &LogHandler{logRepo: logRepo}
}

// List menangani GET /api/v1/notifications/logs.
// Mengembalikan daftar log notifikasi dengan filter dan paginasi.
func (h *LogHandler) List(c *fiber.Ctx) error {
	// Ambil tenant_id dari Fiber locals (di-atur oleh auth middleware)
	tenantID, _ := c.Locals("tenant_id").(string)
	if tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant_id tidak ditemukan")
	}

	// Parsing kueri parameter filter
	params := domain.LogListParams{
		TenantID:   tenantID,
		Channel:    domain.Channel(c.Query("channel")),
		Status:     domain.LogStatus(c.Query("status")),
		CustomerID: c.Query("customer_id"),
		TemplateID: c.Query("template_id"),
	}

	// Parsing date_from (format ISO date: 2006-01-02)
	if df := c.Query("date_from"); df != "" {
		t, err := time.Parse("2006-01-02", df)
		if err == nil {
			params.DateFrom = &t
		}
	}

	// Parsing date_to (format ISO date: 2006-01-02)
	if dt := c.Query("date_to"); dt != "" {
		t, err := time.Parse("2006-01-02", dt)
		if err == nil {
			params.DateTo = &t
		}
	}

	// Parsing page (bawaan 1)
	page, err := strconv.Atoi(c.Query("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}
	params.Page = page

	// Parsing page_size dan normalisasi (valid: 10, 25, 50; bawaan: 25)
	pageSize, err := strconv.Atoi(c.Query("page_size", "25"))
	if err != nil {
		pageSize = 25
	}
	params.PageSize = domain.NormalizePageSize(pageSize)

	// Panggil repositori untuk mengambil data log
	result, err := h.logRepo.List(c.UserContext(), params)
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal mengambil data log")
	}

	// Kembalikan respons dengan metadata paginasi
	return domain.PaginatedResponse(
		c,
		fiber.StatusOK,
		result.Data,
		result.Total,
		result.Page,
		result.PageSize,
		result.TotalPages,
	)
}

// GetByID menangani GET /api/v1/notifications/logs/:id.
// Mengembalikan detail satu log notifikasi berdasarkan ID.
func (h *LogHandler) GetByID(c *fiber.Ctx) error {
	// Ambil tenant_id dari Fiber locals (di-atur oleh auth middleware)
	tenantID, _ := c.Locals("tenant_id").(string)
	if tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant_id tidak ditemukan")
	}

	// Ambil log ID dari URL parameter
	logID := c.Params("id")
	if logID == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "id log tidak boleh kosong")
	}

	// Panggil repositori untuk mengambil log berdasarkan ID
	log, err := h.logRepo.GetByID(c.UserContext(), logID)
	if err != nil {
		if errors.Is(err, domain.ErrLogNotFound) {
			return domain.ErrorResponse(c, fiber.StatusNotFound, "LOG_NOT_FOUND", "log notifikasi tidak ditemukan")
		}
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal mengambil data log")
	}

	// Pastikan log milik tenant yang sama
	if log.TenantID != tenantID {
		return domain.ErrorResponse(c, fiber.StatusNotFound, "LOG_NOT_FOUND", "log notifikasi tidak ditemukan")
	}

	return domain.SuccessResponse(c, fiber.StatusOK, log)
}
