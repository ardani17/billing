// provisioning_handler_settings.go menangani HTTP permintaan untuk provisioning settings.
// Termasuk: get dan perbarui settings per tenant.
package handler

import (
	"errors"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"

	"github.com/ispboss/ispboss/pkg/tenant"
	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// GetSettings menangani GET /provisioning/settings.
// Mengambil provisioning settings untuk tenant saat ini.
func (h *ProvisioningHandler) GetSettings(c *fiber.Ctx) error {
	tenantID := tenant.FromContext(c.UserContext())
	if tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	settings, err := h.manager.GetSettings(c.UserContext(), tenantID)
	if err != nil {
		return h.mapError(c, err)
	}
	return domain.SuccessResponse(c, fiber.StatusOK, settings)
}

// UpdateSettings menangani PUT /provisioning/settings.
// Memperbarui provisioning settings untuk tenant saat ini.
func (h *ProvisioningHandler) UpdateSettings(c *fiber.Ctx) error {
	tenantID := tenant.FromContext(c.UserContext())
	if tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	var req domain.UpdateSettingsRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "request body tidak valid")
	}
	if err := h.validate.Struct(req); err != nil {
		return h.validationError(c, err)
	}

	settings, err := h.manager.UpdateSettings(c.UserContext(), tenantID, req)
	if err != nil {
		return h.mapError(c, err)
	}
	return domain.SuccessResponse(c, fiber.StatusOK, settings)
}

// mapError memetakan domain error provisioning ke HTTP error respons.
func (h *ProvisioningHandler) mapError(c *fiber.Ctx, err error) error {
	switch {
	// 404 Not Found
	case errors.Is(err, domain.ErrONTNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "ONT_NOT_FOUND", err.Error())
	case errors.Is(err, domain.ErrOLTNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "OLT_NOT_FOUND", err.Error())
	case errors.Is(err, domain.ErrBulkNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "BULK_NOT_FOUND", err.Error())

	// 409 Conflict
	case errors.Is(err, domain.ErrONTSerialNumberExists):
		return domain.ErrorResponse(c, fiber.StatusConflict, "ONT_SN_EXISTS", err.Error())
	case errors.Is(err, domain.ErrONTPositionExists):
		return domain.ErrorResponse(c, fiber.StatusConflict, "ONT_POSITION_EXISTS", err.Error())
	case errors.Is(err, domain.ErrONTAlreadyProvisioned):
		return domain.ErrorResponse(c, fiber.StatusConflict, "ONT_ALREADY_PROVISIONED", err.Error())
	case errors.Is(err, domain.ErrCustomerHasActiveONT):
		return domain.ErrorResponse(c, fiber.StatusConflict, "CUSTOMER_HAS_ACTIVE_ONT", err.Error())
	case errors.Is(err, domain.ErrProvisioningInProgress):
		return domain.ErrorResponse(c, fiber.StatusConflict, "PROVISIONING_IN_PROGRESS", err.Error())
	case errors.Is(err, domain.ErrBulkAlreadyExecuted):
		return domain.ErrorResponse(c, fiber.StatusConflict, "BULK_ALREADY_EXECUTED", err.Error())

	// 422 Unprocessable Entitas
	case errors.Is(err, domain.ErrONTNotProvisioned):
		return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "ONT_NOT_PROVISIONED", err.Error())
	case errors.Is(err, domain.ErrInvalidCSVFormat):
		return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "INVALID_CSV_FORMAT", err.Error())
	case errors.Is(err, domain.ErrInvalidVLANStrategy):
		return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "INVALID_VLAN_STRATEGY", err.Error())
	case errors.Is(err, domain.ErrNoProfileMapping):
		return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "NO_PROFILE_MAPPING", err.Error())
	case errors.Is(err, domain.ErrVLANResolutionFailed):
		return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "VLAN_RESOLUTION_FAILED", err.Error())
	case errors.Is(err, domain.ErrUnsupportedBrand):
		return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "UNSUPPORTED_BRAND", err.Error())
	case errors.Is(err, domain.ErrServiceProfileNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "SERVICE_PROFILE_NOT_FOUND", err.Error())
	case errors.Is(err, domain.ErrVLANNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "VLAN_NOT_FOUND", err.Error())

	// 502 Bad Gateway - CLI/koneksi gagal
	case errors.Is(err, domain.ErrProvisioningFailed):
		return domain.ErrorResponse(c, fiber.StatusBadGateway, "PROVISIONING_FAILED", err.Error())
	case errors.Is(err, domain.ErrDecommissionFailed):
		return domain.ErrorResponse(c, fiber.StatusBadGateway, "DECOMMISSION_FAILED", err.Error())
	case errors.Is(err, domain.ErrRebootFailed):
		return domain.ErrorResponse(c, fiber.StatusBadGateway, "REBOOT_FAILED", err.Error())
	case errors.Is(err, domain.ErrCLIConnectionFailed):
		return domain.ErrorResponse(c, fiber.StatusBadGateway, "CLI_CONNECTION_FAILED", err.Error())

	// 504 Gateway Timeout
	case errors.Is(err, domain.ErrCLITimeout):
		return domain.ErrorResponse(c, fiber.StatusGatewayTimeout, "CLI_TIMEOUT", err.Error())

	default:
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "terjadi kesalahan internal")
	}
}

// validationError menangani error validasi dari go-playground/validator.
func (h *ProvisioningHandler) validationError(c *fiber.Ctx, err error) error {
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
