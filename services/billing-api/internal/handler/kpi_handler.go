// kpi_handler.go menangani HTTP request untuk target KPI.
// Termasuk: Get (ambil target KPI saat ini) dan Update (upsert target KPI).
package handler

import (
	"errors"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// KPIHandler menangani HTTP request untuk target KPI.
type KPIHandler struct {
	kpiUsecase domain.KPITargetUsecase
	validate   *validator.Validate
	logger     zerolog.Logger
}

// NewKPIHandler membuat instance baru KPIHandler.
func NewKPIHandler(kpiUsecase domain.KPITargetUsecase, logger zerolog.Logger) *KPIHandler {
	return &KPIHandler{
		kpiUsecase: kpiUsecase,
		validate:   validator.New(),
		logger:     logger,
	}
}

// Get menangani GET /v1/reports/kpi-targets.
// Mengembalikan target KPI saat ini untuk tenant.
func (h *KPIHandler) Get(c *fiber.Ctx) error {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	target, err := h.kpiUsecase.Get(c.Context(), tenantID)
	if err != nil {
		if errors.Is(err, domain.ErrKPITargetNotFound) {
			// Belum ada target KPI, kembalikan objek kosong
			return domain.SuccessResponse(c, fiber.StatusOK, &domain.KPITarget{TenantID: tenantID})
		}
		h.logger.Error().Err(err).Msg("gagal mengambil target KPI")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal mengambil target KPI")
	}

	return domain.SuccessResponse(c, fiber.StatusOK, target)
}

// Update menangani PUT /v1/reports/kpi-targets.
// Membuat atau memperbarui target KPI untuk tenant.
func (h *KPIHandler) Update(c *fiber.Ctx) error {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	var req domain.UpdateKPITargetRequest
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

	target, err := h.kpiUsecase.Upsert(c.Context(), tenantID, req)
	if err != nil {
		h.logger.Error().Err(err).Msg("gagal memperbarui target KPI")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal memperbarui target KPI")
	}

	return domain.SuccessResponse(c, fiber.StatusOK, target)
}
