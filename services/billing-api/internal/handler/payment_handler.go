// payment_handler.go menangani HTTP permintaan untuk modul pembayaran manual.
// void, bulk import, dan bukti transfer.
package handler

import (
	"errors"
	"io"
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/ispboss/ispboss/services/billing-api/internal/usecase"
)

// PaymentHandler menangani HTTP permintaan untuk modul pembayaran manual.
type PaymentHandler struct {
	paymentUsecase *usecase.PaymentUsecase
	validate       *validator.Validate
	logger         zerolog.Logger
}

// NewPaymentHandler membuat instance baru PaymentHandler.
func NewPaymentHandler(paymentUsecase *usecase.PaymentUsecase, logger zerolog.Logger) *PaymentHandler {
	v := validator.New()
	RegisterCustomValidators(v)
	return &PaymentHandler{
		paymentUsecase: paymentUsecase,
		validate:       v,
		logger:         logger,
	}
}

// List menangani GET /v1/payments.
// Mengembalikan daftar pembayaran dengan paginasi, filter, dan pencarian.
func (h *PaymentHandler) List(c *fiber.Ctx) error {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	var params domain.PaymentListParams
	params.TenantID = tenantID
	params.Page, _ = strconv.Atoi(c.Query("page", "1"))
	params.PageSize, _ = strconv.Atoi(c.Query("page_size", "25"))
	params.PaymentMethod = c.Query("payment_method")
	params.DateFrom = c.Query("date_from")
	params.DateTo = c.Query("date_to")
	params.RecordedBy = c.Query("recorded_by")
	params.Search = c.Query("search")
	params.IncludeVoided = c.Query("include_voided") == "true"

	if err := h.validate.Struct(params); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			return domain.ErrorResponse(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "validasi gagal", mapValidationErrors(ve)...)
		}
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "VALIDATION_ERROR", err.Error())
	}

	result, err := h.paymentUsecase.List(c.Context(), params)
	if err != nil {
		h.logger.Error().Err(err).Msg("gagal mengambil daftar pembayaran")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal mengambil daftar pembayaran")
	}

	return domain.SuccessResponse(c, fiber.StatusOK, result)
}

// Summary menangani GET /v1/payments/summary.
// Mengembalikan ringkasan statistik pembayaran untuk dashboard.
func (h *PaymentHandler) Summary(c *fiber.Ctx) error {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	var periodMonth, periodYear *int
	if pm := c.Query("period_month"); pm != "" {
		v, err := strconv.Atoi(pm)
		if err == nil {
			periodMonth = &v
		}
	}
	if py := c.Query("period_year"); py != "" {
		v, err := strconv.Atoi(py)
		if err == nil {
			periodYear = &v
		}
	}

	summary, err := h.paymentUsecase.Summary(c.Context(), tenantID, periodMonth, periodYear)
	if err != nil {
		h.logger.Error().Err(err).Msg("gagal mengambil ringkasan pembayaran")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal mengambil ringkasan pembayaran")
	}

	return domain.SuccessResponse(c, fiber.StatusOK, summary)
}

// PencarianCustomers menangani GET /v1/payments/quick/customers.
// Mencari pelanggan berdasarkan nama, ID, atau telepon untuk pembayaran cepat.
func (h *PaymentHandler) SearchCustomers(c *fiber.Ctx) error {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	searchTerm := c.Query("search")

	customers, err := h.paymentUsecase.SearchCustomers(c.Context(), tenantID, searchTerm)
	if err != nil {
		return h.mapPaymentError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, customers)
}

// GetOpenInvoices menangani GET /v1/payments/quick/customers/:customer_id/invoices.
// Mengembalikan daftar invoice terbuka untuk pelanggan dengan total tunggakan.
func (h *PaymentHandler) GetOpenInvoices(c *fiber.Ctx) error {
	customerID := c.Params("customer_id")
	if customerID == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "customer_id wajib diisi")
	}

	result, err := h.paymentUsecase.GetOpenInvoices(c.Context(), customerID)
	if err != nil {
		return h.mapPaymentError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, result)
}

// RecordMultiPayment menangani POST /v1/payments/multi.
// Mencatat pembayaran multi-invoice dengan alokasi FIFO.
func (h *PaymentHandler) RecordMultiPayment(c *fiber.Ctx) error {
	var req domain.MultiPaymentRequest
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

	result, err := h.paymentUsecase.RecordMultiPayment(c.Context(), req, actor)
	if err != nil {
		return h.mapPaymentError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, result)
}

// PayAll menangani POST /v1/payments/pay-all.
// Membayar semua invoice terbuka untuk pelanggan dalam satu transaksi.
func (h *PaymentHandler) PayAll(c *fiber.Ctx) error {
	var req domain.PayAllRequest
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

	result, err := h.paymentUsecase.PayAll(c.Context(), req, actor)
	if err != nil {
		return h.mapPaymentError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, result)
}

// GetReceipt menangani GET /v1/payments/:payment_id/receipt.
// Mengembalikan data kwitansi pembayaran untuk cetak thermal.
func (h *PaymentHandler) GetReceipt(c *fiber.Ctx) error {
	paymentID := c.Params("payment_id")
	if paymentID == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "payment_id wajib diisi")
	}

	receipt, err := h.paymentUsecase.GetReceipt(c.Context(), paymentID)
	if err != nil {
		return h.mapPaymentError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, receipt)
}

// VoidPayment menangani POST /v1/payments/:payment_id/void.
// Membatalkan pembayaran dengan rollback status invoice (admin only).
func (h *PaymentHandler) VoidPayment(c *fiber.Ctx) error {
	paymentID := c.Params("payment_id")
	if paymentID == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "payment_id wajib diisi")
	}

	var req domain.VoidPaymentRequest
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

	result, err := h.paymentUsecase.VoidPayment(c.Context(), paymentID, req, actor)
	if err != nil {
		return h.mapPaymentError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, result)
}

// BulkImport menangani POST /v1/payments/import.
// Mengimpor pembayaran dari file CSV (admin only).
func (h *PaymentHandler) BulkImport(c *fiber.Ctx) error {
	file, err := c.FormFile("file")
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "file CSV wajib diunggah")
	}

	f, err := file.Open()
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "gagal membaca file")
	}
	defer f.Close()

	csvData, err := io.ReadAll(f)
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "gagal membaca isi file")
	}

	actor := h.extractActor(c)

	result, err := h.paymentUsecase.BulkImport(c.Context(), csvData, actor)
	if err != nil {
		return h.mapPaymentError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, result)
}

// UploadProof menangani POST /v1/payments/:payment_id/proof.
// Mengunggah bukti transfer untuk pembayaran.
func (h *PaymentHandler) UploadProof(c *fiber.Ctx) error {
	paymentID := c.Params("payment_id")
	if paymentID == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "payment_id wajib diisi")
	}

	file, err := c.FormFile("file")
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "file bukti transfer wajib diunggah")
	}

	f, err := file.Open()
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "gagal membaca file")
	}
	defer f.Close()

	fileData, err := io.ReadAll(f)
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "gagal membaca isi file")
	}

	proofURL, err := h.paymentUsecase.UploadProof(c.Context(), paymentID, fileData, file.Filename)
	if err != nil {
		return h.mapPaymentError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, fiber.Map{
		"proof_image_url": proofURL,
	})
}

// GetProof menangani GET /v1/payments/:payment_id/proof.
// Mengembalikan file bukti transfer untuk pembayaran.
func (h *PaymentHandler) GetProof(c *fiber.Ctx) error {
	paymentID := c.Params("payment_id")
	if paymentID == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "payment_id wajib diisi")
	}

	data, contentType, err := h.paymentUsecase.GetProof(c.Context(), paymentID)
	if err != nil {
		return h.mapPaymentError(c, err)
	}

	c.Set("Content-Type", contentType)
	return c.Send(data)
}

// extractActor mengambil informasi aktor dari Fiber locals (di-atur oleh auth middleware).
func (h *PaymentHandler) extractActor(c *fiber.Ctx) domain.ActorInfo {
	actorID, _ := c.Locals("user_id").(string)
	actorName, _ := c.Locals("user_name").(string)
	return domain.ActorInfo{
		ActorID:   actorID,
		ActorName: actorName,
	}
}

// mapPaymentError memetakan domain error ke HTTP error respons.
func (h *PaymentHandler) mapPaymentError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domain.ErrPaymentNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "PAYMENT_NOT_FOUND", err.Error())
	case errors.Is(err, domain.ErrCustomerNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "CUSTOMER_NOT_FOUND", err.Error())
	case errors.Is(err, domain.ErrInvoiceNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "INVOICE_NOT_FOUND", err.Error())
	case errors.Is(err, domain.ErrPaymentAlreadyVoided):
		return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "PAYMENT_ALREADY_VOIDED", err.Error())
	case errors.Is(err, domain.ErrVoidTimeLimitExceeded):
		return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "VOID_TIME_LIMIT_EXCEEDED", err.Error())
	case errors.Is(err, domain.ErrNoOpenInvoices):
		return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "NO_OPEN_INVOICES", err.Error())
	case errors.Is(err, domain.ErrInvalidInvoiceSelection):
		return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "INVALID_INVOICE_SELECTION", err.Error())
	case errors.Is(err, domain.ErrSearchTermTooShort):
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "SEARCH_TERM_TOO_SHORT", err.Error())
	case errors.Is(err, domain.ErrCSVTooLarge):
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "CSV_TOO_LARGE", err.Error())
	case errors.Is(err, domain.ErrConcurrentModification):
		return domain.ErrorResponse(c, fiber.StatusConflict, "CONCURRENT_MODIFICATION", err.Error())
	case errors.Is(err, domain.ErrFileTooLarge):
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "FILE_TOO_LARGE", err.Error())
	case errors.Is(err, domain.ErrInvalidFileFormat):
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "INVALID_FILE_FORMAT", err.Error())
	case errors.Is(err, domain.ErrProofNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "PROOF_NOT_FOUND", err.Error())
	default:
		h.logger.Error().Err(err).Msg("internal error pada payment handler")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "terjadi kesalahan internal")
	}
}
