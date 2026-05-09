// schedule_handler.go menangani HTTP permintaan untuk CRUD jadwal laporan otomatis.
package handler

import (
	"errors"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// ScheduleHandler menangani HTTP permintaan untuk jadwal laporan.
type ScheduleHandler struct {
	scheduleUsecase domain.ScheduleUsecase
	validate        *validator.Validate
	logger          zerolog.Logger
}

// NewScheduleHandler membuat instance baru ScheduleHandler.
func NewScheduleHandler(scheduleUsecase domain.ScheduleUsecase, logger zerolog.Logger) *ScheduleHandler {
	return &ScheduleHandler{
		scheduleUsecase: scheduleUsecase,
		validate:        validator.New(),
		logger:          logger,
	}
}

// List menangani GET /v1/reports/schedules.
// Mengembalikan semua jadwal laporan aktif untuk tenant.
func (h *ScheduleHandler) List(c *fiber.Ctx) error {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	schedules, err := h.scheduleUsecase.List(c.Context(), tenantID)
	if err != nil {
		h.logger.Error().Err(err).Msg("gagal mengambil daftar jadwal laporan")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal mengambil daftar jadwal laporan")
	}

	return domain.SuccessResponse(c, fiber.StatusOK, schedules)
}

// Buat menangani POST /v1/reports/schedules.
// Membuat jadwal laporan baru.
func (h *ScheduleHandler) Create(c *fiber.Ctx) error {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	var req domain.CreateScheduleRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "request body tidak valid")
	}

	if err := h.validate.Struct(req); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			return domain.ErrorResponse(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "validasi gagal", mapValidationErrors(ve)...)
		}
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "VALIDATION_ERROR", err.Error())
	}

	actor := h.extractActor(c)

	schedule, err := h.scheduleUsecase.Create(c.Context(), tenantID, req, actor)
	if err != nil {
		return h.mapScheduleError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusCreated, schedule)
}

// Perbarui menangani PUT /v1/reports/schedules/:id.
// Memperbarui konfigurasi jadwal laporan.
func (h *ScheduleHandler) Update(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "schedule ID wajib diisi")
	}

	var req domain.UpdateScheduleRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "request body tidak valid")
	}

	if err := h.validate.Struct(req); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			return domain.ErrorResponse(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "validasi gagal", mapValidationErrors(ve)...)
		}
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "VALIDATION_ERROR", err.Error())
	}

	schedule, err := h.scheduleUsecase.Update(c.Context(), id, req)
	if err != nil {
		return h.mapScheduleError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, schedule)
}

// Hapus menangani DELETE /v1/reports/schedules/:id.
// Menonaktifkan jadwal laporan.
func (h *ScheduleHandler) Delete(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "schedule ID wajib diisi")
	}

	if err := h.scheduleUsecase.Delete(c.Context(), id); err != nil {
		return h.mapScheduleError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// extractActor mengambil informasi aktor dari Fiber locals.
func (h *ScheduleHandler) extractActor(c *fiber.Ctx) domain.ActorInfo {
	actorID, _ := c.Locals("user_id").(string)
	actorName, _ := c.Locals("user_name").(string)
	return domain.ActorInfo{
		ActorID:   actorID,
		ActorName: actorName,
	}
}

// mapScheduleError memetakan domain error ke HTTP error respons untuk jadwal.
func (h *ScheduleHandler) mapScheduleError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domain.ErrReportScheduleNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "SCHEDULE_NOT_FOUND", err.Error())
	default:
		h.logger.Error().Err(err).Msg("internal error pada schedule handler")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "terjadi kesalahan internal")
	}
}
