// report_handler_network.go berisi method-method laporan jaringan pada ReportHandler.
// Termasuk: Uptime, Traffic, SignalQuality, Capacity.
package handler

import (
	"github.com/gofiber/fiber/v2"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// Uptime menangani GET /v1/reports/network/uptime.
// Mengembalikan laporan uptime router.
func (h *ReportHandler) Uptime(c *fiber.Ctx) error {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	filter, err := h.parseFilter(c)
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", err.Error())
	}

	report, err := h.reportUsecase.GetUptimeReport(c.Context(), tenantID, filter)
	if err != nil {
		return h.mapReportError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, report)
}

// Traffic menangani GET /v1/reports/network/traffic.
// Mengembalikan laporan traffic jaringan.
func (h *ReportHandler) Traffic(c *fiber.Ctx) error {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	filter, err := h.parseFilter(c)
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", err.Error())
	}

	report, err := h.reportUsecase.GetTrafficReport(c.Context(), tenantID, filter)
	if err != nil {
		return h.mapReportError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, report)
}

// SignalQuality menangani GET /v1/reports/network/signal-quality.
// Mengembalikan laporan kualitas signal OLT.
func (h *ReportHandler) SignalQuality(c *fiber.Ctx) error {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	filter, err := h.parseFilter(c)
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", err.Error())
	}

	report, err := h.reportUsecase.GetSignalQualityReport(c.Context(), tenantID, filter)
	if err != nil {
		return h.mapReportError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, report)
}

// Capacity menangani GET /v1/reports/network/capacity.
// Mengembalikan laporan kapasitas jaringan.
func (h *ReportHandler) Capacity(c *fiber.Ctx) error {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	report, err := h.reportUsecase.GetCapacityReport(c.Context(), tenantID)
	if err != nil {
		return h.mapReportError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, report)
}
