package handler

import (
	"errors"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/ispboss/ispboss/services/billing-api/internal/usecase"
)

// BillingSettingsHandler menangani HTTP permintaan konfigurasi billing tenant.
type BillingSettingsHandler struct {
	usecase  *usecase.BillingSettingsUsecase
	validate *validator.Validate
	logger   zerolog.Logger
}

// NewBillingSettingsHandler membuat instance BillingSettingsHandler.
func NewBillingSettingsHandler(usecase *usecase.BillingSettingsUsecase, logger zerolog.Logger) *BillingSettingsHandler {
	return &BillingSettingsHandler{
		usecase:  usecase,
		validate: validator.New(),
		logger:   logger,
	}
}

// Get menangani GET /api/v1/settings/billing.
func (h *BillingSettingsHandler) Get(c *fiber.Ctx) error {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	settings, err := h.usecase.Get(c.Context(), tenantID)
	if err != nil {
		h.logger.Error().Err(err).Msg("gagal mengambil billing settings")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal mengambil billing settings")
	}

	return domain.SuccessResponse(c, fiber.StatusOK, settings)
}

// Perbarui menangani PUT /api/v1/settings/billing.
func (h *BillingSettingsHandler) Update(c *fiber.Ctx) error {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	var req domain.UpdateBillingSettingsRequest
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

	settings, err := h.usecase.Update(c.Context(), tenantID, req)
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "VALIDATION_ERROR", err.Error())
	}

	return domain.SuccessResponse(c, fiber.StatusOK, settings)
}
