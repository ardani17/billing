package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/ispboss/ispboss/services/billing-api/internal/usecase"
)

type TenantModuleHandler struct {
	usecase *usecase.TenantModuleUsecase
	logger  zerolog.Logger
}

func NewTenantModuleHandler(usecase *usecase.TenantModuleUsecase, logger zerolog.Logger) *TenantModuleHandler {
	return &TenantModuleHandler{usecase: usecase, logger: logger}
}

func (h *TenantModuleHandler) Current(c *fiber.Ctx) error {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	caps, err := h.usecase.Capabilities(c.UserContext(), tenantID)
	if err != nil {
		h.logger.Error().Err(err).Str("tenant_id", tenantID).Msg("gagal mengambil entitlement modul tenant")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal mengambil modul tenant")
	}

	return domain.SuccessResponse(c, fiber.StatusOK, fiber.Map{
		"modules": caps,
	})
}
