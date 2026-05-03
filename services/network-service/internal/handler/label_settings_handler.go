// label_settings_handler.go menangani HTTP request untuk konfigurasi label peta.
// Termasuk: get dan update label settings per tenant.
package handler

import (
	"errors"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"

	"github.com/ispboss/ispboss/pkg/tenant"
	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// LabelSettingsHandler menangani HTTP request untuk konfigurasi label.
type LabelSettingsHandler struct {
	manager  domain.MapNodeManager
	validate *validator.Validate
}

// NewLabelSettingsHandler membuat instance baru LabelSettingsHandler.
func NewLabelSettingsHandler(manager domain.MapNodeManager) *LabelSettingsHandler {
	return &LabelSettingsHandler{
		manager:  manager,
		validate: validator.New(),
	}
}

// GetLabelSettings menangani GET /settings/labels.
// Mengambil konfigurasi label untuk tenant saat ini.
// Return default jika tenant belum memiliki konfigurasi.
func (h *LabelSettingsHandler) GetLabelSettings(c *fiber.Ctx) error {
	tenantID := tenant.FromContext(c.UserContext())
	if tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	settings, err := h.manager.GetLabelSettings(c.UserContext(), tenantID)
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "terjadi kesalahan internal")
	}

	return domain.SuccessResponse(c, fiber.StatusOK, settings)
}

// UpdateLabelSettings menangani PUT /settings/labels.
// Parse body, validasi, lalu update konfigurasi label untuk tenant.
func (h *LabelSettingsHandler) UpdateLabelSettings(c *fiber.Ctx) error {
	tenantID := tenant.FromContext(c.UserContext())
	if tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	var req domain.UpdateLabelSettingsRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "request body tidak valid")
	}

	if err := h.validate.Struct(req); err != nil {
		return h.validationError(c, err)
	}

	settings, err := h.manager.UpdateLabelSettings(c.UserContext(), tenantID, req)
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "terjadi kesalahan internal")
	}

	return domain.SuccessResponse(c, fiber.StatusOK, settings)
}

// validationError menangani error validasi dari go-playground/validator.
func (h *LabelSettingsHandler) validationError(c *fiber.Ctx, err error) error {
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		fields := make([]domain.FieldError, 0, len(ve))
		for _, fe := range ve {
			fields = append(fields, domain.FieldError{
				Field:   toSnakeCase(fe.Field()),
				Message: validationMessage(fe),
			})
		}
		return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "validasi gagal", fields...)
	}
	return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
}
