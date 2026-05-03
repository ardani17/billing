// report_handler_customer.go berisi method-method laporan pelanggan pada ReportHandler.
// Termasuk: CustomerGrowth, CustomerDistribution, ChurnAnalysis.
package handler

import (
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// CustomerGrowth menangani GET /v1/reports/customers/growth.
// Mengembalikan laporan pertumbuhan pelanggan.
func (h *ReportHandler) CustomerGrowth(c *fiber.Ctx) error {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	filter, err := h.parseFilter(c)
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", err.Error())
	}

	report, err := h.reportUsecase.GetCustomerGrowthReport(c.Context(), tenantID, filter)
	if err != nil {
		return h.mapReportError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, report)
}

// CustomerDistribution menangani GET /v1/reports/customers/distribution.
// Mengembalikan laporan distribusi pelanggan per paket/area/status.
func (h *ReportHandler) CustomerDistribution(c *fiber.Ctx) error {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	periodEnd := c.Query("period_end")
	if periodEnd == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "period_end wajib diisi")
	}

	pe, err := time.Parse("2006-01-02", periodEnd)
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "format period_end tidak valid (gunakan YYYY-MM-DD)")
	}

	report, err := h.reportUsecase.GetCustomerDistributionReport(c.Context(), tenantID, pe)
	if err != nil {
		return h.mapReportError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, report)
}

// ChurnAnalysis menangani GET /v1/reports/customers/churn.
// Mengembalikan laporan analisis churn pelanggan.
func (h *ReportHandler) ChurnAnalysis(c *fiber.Ctx) error {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	filter, err := h.parseFilter(c)
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", err.Error())
	}

	report, err := h.reportUsecase.GetChurnAnalysisReport(c.Context(), tenantID, filter)
	if err != nil {
		return h.mapReportError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, report)
}
