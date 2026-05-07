// status_handler.go menangani HTTP permintaan untuk ringkasan status router.
package handler

import (
	"github.com/gofiber/fiber/v2"

	"github.com/ispboss/ispboss/pkg/tenant"
	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// StatusHandler menangani HTTP permintaan untuk status summary router.
type StatusHandler struct {
	usecase domain.RouterUsecase
}

// NewStatusHandler membuat instance baru StatusHandler.
func NewStatusHandler(usecase domain.RouterUsecase) *StatusHandler {
	return &StatusHandler{
		usecase: usecase,
	}
}

// GetSummary menangani GET /api/v1/mikrotik/status/summary.
// Mengambil ringkasan jumlah router per status untuk tenant yang sedang login.
func (h *StatusHandler) GetSummary(c *fiber.Ctx) error {
	tenantID := tenant.FromContext(c.UserContext())
	if tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	summary, err := h.usecase.GetStatusSummary(c.UserContext())
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "terjadi kesalahan internal")
	}

	return domain.SuccessResponse(c, fiber.StatusOK, summary)
}
