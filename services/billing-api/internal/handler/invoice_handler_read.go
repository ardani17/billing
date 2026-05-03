// invoice_handler_read.go menangani HTTP request read-only untuk invoice.
// Termasuk: summary, PDF, dan audit logs.
package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// Summary menangani GET /v1/invoices/summary.
// Mengembalikan ringkasan invoice per status untuk dashboard.
func (h *InvoiceHandler) Summary(c *fiber.Ctx) error {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	var periodMonth, periodYear *int
	if pmStr := c.Query("period_month"); pmStr != "" {
		pm, err := strconv.Atoi(pmStr)
		if err == nil {
			periodMonth = &pm
		}
	}
	if pyStr := c.Query("period_year"); pyStr != "" {
		py, err := strconv.Atoi(pyStr)
		if err == nil {
			periodYear = &py
		}
	}

	summary, err := h.invoiceUsecase.Summary(c.Context(), tenantID, periodMonth, periodYear)
	if err != nil {
		h.logger.Error().Err(err).Msg("gagal mengambil ringkasan invoice")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal mengambil ringkasan invoice")
	}

	return domain.SuccessResponse(c, fiber.StatusOK, summary)
}

// PDF menangani GET /v1/invoices/:id/pdf.
// Menghasilkan dan mengembalikan file PDF invoice.
func (h *InvoiceHandler) PDF(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "invoice ID wajib diisi")
	}

	pdfBytes, filename, err := h.invoiceUsecase.GeneratePDF(c.Context(), id)
	if err != nil {
		return h.mapInvoiceError(c, err)
	}

	c.Set("Content-Type", "application/pdf")
	c.Set("Content-Disposition", "attachment; filename="+filename)
	return c.Send(pdfBytes)
}

// AuditLogs menangani GET /v1/invoices/:id/audit-logs.
// Mengembalikan daftar audit log untuk invoice tertentu.
func (h *InvoiceHandler) AuditLogs(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "invoice ID wajib diisi")
	}

	detail, err := h.invoiceUsecase.GetByID(c.Context(), id, true)
	if err != nil {
		return h.mapInvoiceError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, detail.AuditLogs)
}
