package middleware

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/ispboss/ispboss/services/network-service/internal/domain"
	"github.com/rs/zerolog"
)

type ModuleChecker interface {
	IsEnabled(ctx context.Context, tenantID, moduleCode string) (bool, error)
}

func RequireModule(moduleCode string, checker ModuleChecker, logger zerolog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if moduleCode == domain.ModuleBillingCore {
			return c.Next()
		}
		if checker == nil {
			return domain.ErrorResponse(c, fiber.StatusForbidden, "MODULE_NOT_ENABLED", "modul belum aktif untuk tenant ini")
		}

		tenantID, ok := c.Locals("tenant_id").(string)
		if !ok || tenantID == "" {
			return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
		}

		enabled, err := checker.IsEnabled(c.UserContext(), tenantID, moduleCode)
		if err != nil {
			logger.Error().Err(err).Str("tenant_id", tenantID).Str("module", moduleCode).Msg("gagal memeriksa entitlement modul")
			return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal memeriksa modul tenant")
		}
		if !enabled {
			return domain.ErrorResponse(c, fiber.StatusForbidden, "MODULE_NOT_ENABLED", "modul belum aktif untuk tenant ini")
		}

		return c.Next()
	}
}
