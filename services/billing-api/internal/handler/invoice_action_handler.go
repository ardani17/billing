// invoice_action_handler.go menangani HTTP request untuk aksi invoice.
// Termasuk: cancel, record payment, bulk reminder, bulk cancel, bulk PDF, dan export CSV.
package handler

import (
	"errors"
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/ispboss/ispboss/services/billing-api/internal/usecase"
)

// InvoiceActionHandler menangani HTTP request untuk aksi invoice.
type InvoiceActionHandler struct {
	actionUsecase *usecase.InvoiceActionUsecase
	validate      *validator.Validate
	logger        zerolog.Logger
}

// NewInvoiceActionHandler membuat instance baru InvoiceActionHandler.
func NewInvoiceActionHandler(actionUsecase *usecase.InvoiceActionUsecase, logger zerolog.Logger) *InvoiceActionHandler {
	return &InvoiceActionHandler{
		actionUsecase: actionUsecase,
		validate:      validator.New(),
		logger:        logger,
	}
}

// Cancel menangani POST /v1/invoices/:id/cancel.
// Membatalkan invoice dengan verifikasi nomor konfirmasi.
func (h *InvoiceActionHandler) Cancel(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "invoice ID wajib diisi")
	}

	var req domain.CancelInvoiceRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "request body tidak valid")
	}

	if err := h.validate.Struct(req); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			return domain.ErrorResponse(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "validasi gagal", mapValidationErrors(ve)...)
		}
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "VALIDATION_ERROR", err.Error())
	}

	actor := h.extractActor(c)

	invoice, err := h.actionUsecase.Cancel(c.Context(), id, req, actor)
	if err != nil {
		return h.mapActionError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, invoice)
}

// RecordPayment menangani POST /v1/invoices/:id/payment.
// Mencatat pembayaran terhadap invoice.
func (h *InvoiceActionHandler) RecordPayment(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "invoice ID wajib diisi")
	}

	var req domain.RecordPaymentRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "request body tidak valid")
	}

	if err := h.validate.Struct(req); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			return domain.ErrorResponse(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "validasi gagal", mapValidationErrors(ve)...)
		}
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "VALIDATION_ERROR", err.Error())
	}

	actor := h.extractActor(c)

	invoice, err := h.actionUsecase.RecordPayment(c.Context(), id, req, actor)
	if err != nil {
		return h.mapActionError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, invoice)
}

// BulkReminder menangani POST /v1/invoices/bulk/reminder.
// Mengirim pengingat pembayaran untuk beberapa invoice sekaligus.
func (h *InvoiceActionHandler) BulkReminder(c *fiber.Ctx) error {
	var req domain.BulkInvoiceIDsRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "request body tidak valid")
	}

	if err := h.validate.Struct(req); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			return domain.ErrorResponse(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "validasi gagal", mapValidationErrors(ve)...)
		}
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "VALIDATION_ERROR", err.Error())
	}

	actor := h.extractActor(c)

	result, err := h.actionUsecase.BulkReminder(c.Context(), req, actor)
	if err != nil {
		h.logger.Error().Err(err).Msg("gagal mengirim bulk reminder")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal mengirim bulk reminder")
	}

	return domain.SuccessResponse(c, fiber.StatusOK, result)
}

// BulkCancel menangani POST /v1/invoices/bulk/cancel.
// Membatalkan beberapa invoice sekaligus.
func (h *InvoiceActionHandler) BulkCancel(c *fiber.Ctx) error {
	var req domain.BulkCancelRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "request body tidak valid")
	}

	if err := h.validate.Struct(req); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			return domain.ErrorResponse(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "validasi gagal", mapValidationErrors(ve)...)
		}
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "VALIDATION_ERROR", err.Error())
	}

	actor := h.extractActor(c)

	result, err := h.actionUsecase.BulkCancel(c.Context(), req, actor)
	if err != nil {
		h.logger.Error().Err(err).Msg("gagal melakukan bulk cancel")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal melakukan bulk cancel")
	}

	return domain.SuccessResponse(c, fiber.StatusOK, result)
}

// BulkPDF menangani POST /v1/invoices/bulk/pdf.
// Menghasilkan PDF untuk beberapa invoice dalam format ZIP.
func (h *InvoiceActionHandler) BulkPDF(c *fiber.Ctx) error {
	var req domain.BulkInvoiceIDsRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "request body tidak valid")
	}

	if err := h.validate.Struct(req); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			return domain.ErrorResponse(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "validasi gagal", mapValidationErrors(ve)...)
		}
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "VALIDATION_ERROR", err.Error())
	}

	zipBytes, err := h.actionUsecase.BulkPDF(c.Context(), req)
	if err != nil {
		h.logger.Error().Err(err).Msg("gagal menghasilkan bulk PDF")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal menghasilkan bulk PDF")
	}

	c.Set("Content-Type", "application/zip")
	c.Set("Content-Disposition", "attachment; filename=invoices.zip")
	return c.Send(zipBytes)
}

// ExportCSV menangani GET /v1/invoices/export.
// Mengekspor daftar invoice ke format CSV.
func (h *InvoiceActionHandler) ExportCSV(c *fiber.Ctx) error {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	var params domain.InvoiceListParams
	params.TenantID = tenantID
	params.Page, _ = strconv.Atoi(c.Query("page", "1"))
	params.PageSize, _ = strconv.Atoi(c.Query("page_size", "50"))
	params.Search = c.Query("search")
	params.Status = c.Query("status")
	params.SortBy = c.Query("sort_by")
	params.SortOrder = c.Query("sort_order")

	csvBytes, err := h.actionUsecase.ExportCSV(c.Context(), params)
	if err != nil {
		h.logger.Error().Err(err).Msg("gagal mengekspor invoice CSV")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal mengekspor invoice")
	}

	c.Set("Content-Type", "text/csv")
	c.Set("Content-Disposition", "attachment; filename=invoices.csv")
	return c.Send(csvBytes)
}

// extractActor mengambil informasi aktor dari Fiber locals (di-set oleh auth middleware).
func (h *InvoiceActionHandler) extractActor(c *fiber.Ctx) domain.ActorInfo {
	actorID, _ := c.Locals("user_id").(string)
	actorName, _ := c.Locals("user_name").(string)
	return domain.ActorInfo{
		ActorID:   actorID,
		ActorName: actorName,
	}
}

// mapActionError memetakan domain error ke HTTP error response untuk aksi invoice.
func (h *InvoiceActionHandler) mapActionError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domain.ErrInvoiceNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "INVOICE_NOT_FOUND", err.Error())
	case errors.Is(err, domain.ErrInvoiceConfirmationMismatch):
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "CONFIRMATION_MISMATCH", err.Error())
	case errors.Is(err, domain.ErrInvoiceNotCancellable):
		return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "INVOICE_NOT_CANCELLABLE", err.Error())
	case errors.Is(err, domain.ErrInvalidInvoiceStatusTransition):
		return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "INVALID_STATUS_TRANSITION", err.Error())
	default:
		h.logger.Error().Err(err).Msg("internal error pada invoice action handler")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "terjadi kesalahan internal")
	}
}
