// report_handler_financial.go berisi method-method laporan keuangan pada ReportHandler.
// Termasuk: Revenue, Aging, Payments, Vouchers, ProfitLoss, RevenueByArea.
package handler

import (
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// Revenue menangani GET /v1/reports/financial/revenue.
// Mengembalikan laporan ringkasan pendapatan per sumber.
func (h *ReportHandler) Revenue(c *fiber.Ctx) error {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	filter, err := h.parseFilter(c)
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", err.Error())
	}

	report, err := h.reportUsecase.GetRevenueReport(c.Context(), tenantID, filter)
	if err != nil {
		return h.mapReportError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, report)
}

// Aging menangani GET /v1/reports/financial/aging.
// Mengembalikan laporan piutang/aging dengan bucket dan top debtors.
func (h *ReportHandler) Aging(c *fiber.Ctx) error {
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

	areaID := c.Query("area_id")
	packageID := c.Query("package_id")

	report, err := h.reportUsecase.GetAgingReport(c.Context(), tenantID, pe, areaID, packageID)
	if err != nil {
		return h.mapReportError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, report)
}

// Payments menangani GET /v1/reports/financial/payments.
// Mengembalikan laporan distribusi pembayaran per metode.
func (h *ReportHandler) Payments(c *fiber.Ctx) error {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	filter, err := h.parseFilter(c)
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", err.Error())
	}

	report, err := h.reportUsecase.GetPaymentReport(c.Context(), tenantID, filter)
	if err != nil {
		return h.mapReportError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, report)
}

// Vouchers menangani GET /v1/reports/financial/vouchers.
// Mengembalikan laporan pendapatan voucher per paket dan reseller.
func (h *ReportHandler) Vouchers(c *fiber.Ctx) error {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	filter, err := h.parseFilter(c)
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", err.Error())
	}

	report, err := h.reportUsecase.GetVoucherRevenueReport(c.Context(), tenantID, filter)
	if err != nil {
		return h.mapReportError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, report)
}

// ProfitLoss menangani GET /v1/reports/financial/profit-loss.
// Mengembalikan laporan laba rugi sederhana.
func (h *ReportHandler) ProfitLoss(c *fiber.Ctx) error {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	filter, err := h.parseFilter(c)
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", err.Error())
	}

	report, err := h.reportUsecase.GetProfitLossReport(c.Context(), tenantID, filter)
	if err != nil {
		return h.mapReportError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, report)
}

// RevenueByArea menangani GET /v1/reports/financial/revenue-by-area.
// Mengembalikan laporan pendapatan per area.
func (h *ReportHandler) RevenueByArea(c *fiber.Ctx) error {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	filter, err := h.parseFilter(c)
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", err.Error())
	}

	report, err := h.reportUsecase.GetRevenueByAreaReport(c.Context(), tenantID, filter)
	if err != nil {
		return h.mapReportError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, report)
}
