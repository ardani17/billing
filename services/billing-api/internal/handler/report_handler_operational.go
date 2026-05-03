// report_handler_operational.go berisi method-method laporan operasional pada ReportHandler.
// Termasuk: Activity, Notifications, Sync.
package handler

import (
	"github.com/gofiber/fiber/v2"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// Activity menangani GET /v1/reports/operational/activity.
// Mengembalikan laporan aktivitas admin/user.
func (h *ReportHandler) Activity(c *fiber.Ctx) error {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	filter, err := h.parseFilter(c)
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", err.Error())
	}

	report, err := h.reportUsecase.GetActivityReport(c.Context(), tenantID, filter)
	if err != nil {
		return h.mapReportError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, report)
}

// Notifications menangani GET /v1/reports/operational/notifications.
// Mengembalikan laporan statistik notifikasi.
func (h *ReportHandler) Notifications(c *fiber.Ctx) error {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	filter, err := h.parseFilter(c)
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", err.Error())
	}

	report, err := h.reportUsecase.GetNotificationReport(c.Context(), tenantID, filter)
	if err != nil {
		return h.mapReportError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, report)
}

// Sync menangani GET /v1/reports/operational/sync.
// Mengembalikan laporan status sync MikroTik dan OLT.
func (h *ReportHandler) Sync(c *fiber.Ctx) error {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	filter, err := h.parseFilter(c)
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", err.Error())
	}

	report, err := h.reportUsecase.GetSyncReport(c.Context(), tenantID, filter)
	if err != nil {
		return h.mapReportError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, report)
}
