// dashboard_handler.go menangani HTTP permintaan untuk dashboard widget.
// Termasuk: Dashboard (data ringkasan metrik kunci, target < 500ms).
package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// DashboardHandler menangani HTTP permintaan untuk dashboard widget.
type DashboardHandler struct {
	reportUsecase domain.ReportUsecase
	logger        zerolog.Logger
}

// NewDashboardHandler membuat instance baru DashboardHandler.
func NewDashboardHandler(reportUsecase domain.ReportUsecase, logger zerolog.Logger) *DashboardHandler {
	return &DashboardHandler{
		reportUsecase: reportUsecase,
		logger:        logger,
	}
}

// Dashboard menangani GET /v1/reports/dashboard.
// Mengembalikan data ringkasan untuk dashboard widget.
// Target respons time < 500ms (menggunakan Redis cache di usecase layer).
func (h *DashboardHandler) Dashboard(c *fiber.Ctx) error {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	data, err := h.reportUsecase.GetDashboardData(c.Context(), tenantID)
	if err != nil {
		h.logger.Error().Err(err).Msg("gagal mengambil data dashboard")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal mengambil data dashboard")
	}

	return domain.SuccessResponse(c, fiber.StatusOK, data)
}
