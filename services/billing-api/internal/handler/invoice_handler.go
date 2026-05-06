// invoice_handler.go menangani HTTP request untuk manajemen invoice (CRUD).
// Termasuk: list, get, create, create prepaid, dan edit.
// Endpoint read-only (summary, PDF, audit logs) ada di invoice_handler_read.go.
package handler

import (
	"errors"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/ispboss/ispboss/services/billing-api/internal/usecase"
)

// InvoiceHandler menangani HTTP request untuk manajemen invoice.
type InvoiceHandler struct {
	invoiceUsecase *usecase.InvoiceUsecase
	cronUsecase    *usecase.InvoiceCronUsecase
	validate       *validator.Validate
	logger         zerolog.Logger
}

// NewInvoiceHandler membuat instance baru InvoiceHandler.
func NewInvoiceHandler(invoiceUsecase *usecase.InvoiceUsecase, logger zerolog.Logger) *InvoiceHandler {
	return &InvoiceHandler{
		invoiceUsecase: invoiceUsecase,
		validate:       validator.New(),
		logger:         logger,
	}
}

// SetCronUsecase memasang usecase cron agar admin dapat memicu generate invoice on-demand.
func (h *InvoiceHandler) SetCronUsecase(cronUsecase *usecase.InvoiceCronUsecase) {
	h.cronUsecase = cronUsecase
}

// List menangani GET /v1/invoices.
// Mengembalikan daftar invoice dengan paginasi, filter, dan sorting.
func (h *InvoiceHandler) List(c *fiber.Ctx) error {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	var params domain.InvoiceListParams
	params.TenantID = tenantID
	params.Page, _ = strconv.Atoi(c.Query("page", "1"))
	params.PageSize, _ = strconv.Atoi(c.Query("page_size", "25"))
	params.CustomerID = c.Query("customer_id")
	params.Search = c.Query("search")
	params.Status = c.Query("status")
	params.PackageID = c.Query("package_id")
	params.AreaID = c.Query("area_id")
	params.SortBy = c.Query("sort_by")
	params.SortOrder = c.Query("sort_order")

	// Parse period_month dan period_year (opsional)
	if pmStr := c.Query("period_month"); pmStr != "" {
		pm, err := strconv.Atoi(pmStr)
		if err == nil {
			params.PeriodMonth = &pm
		}
	}
	if pyStr := c.Query("period_year"); pyStr != "" {
		py, err := strconv.Atoi(pyStr)
		if err == nil {
			params.PeriodYear = &py
		}
	}

	if err := h.validate.Struct(params); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			return domain.ErrorResponse(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "validasi gagal", mapValidationErrors(ve)...)
		}
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "VALIDATION_ERROR", err.Error())
	}

	result, err := h.invoiceUsecase.List(c.Context(), params)
	if err != nil {
		h.logger.Error().Err(err).Msg("gagal mengambil daftar invoice")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal mengambil daftar invoice")
	}

	return domain.SuccessResponse(c, fiber.StatusOK, result)
}

// Get menangani GET /v1/invoices/:id.
// Mengembalikan detail invoice lengkap termasuk items dan payments.
func (h *InvoiceHandler) Get(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "invoice ID wajib diisi")
	}

	includeAudit := strings.Contains(c.Query("include"), "audit_logs")

	detail, err := h.invoiceUsecase.GetByID(c.Context(), id, includeAudit)
	if err != nil {
		return h.mapInvoiceError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, detail)
}

// Create menangani POST /v1/invoices.
// Membuat invoice manual dengan item-item yang ditentukan.
func (h *InvoiceHandler) Create(c *fiber.Ctx) error {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	var req domain.CreateInvoiceRequest
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

	invoice, err := h.invoiceUsecase.Create(c.Context(), tenantID, req, actor)
	if err != nil {
		return h.mapInvoiceError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusCreated, invoice)
}

// GenerateDue menangani POST /v1/invoices/generate-due.
// Menjalankan generator invoice untuk tenant aktif secara on-demand.
func (h *InvoiceHandler) GenerateDue(c *fiber.Ctx) error {
	if h.cronUsecase == nil {
		return domain.ErrorResponse(c, fiber.StatusServiceUnavailable, "GENERATOR_UNAVAILABLE", "generator invoice belum tersedia")
	}

	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	if err := h.cronUsecase.GenerateDueForTenant(c.Context(), tenantID); err != nil {
		h.logger.Error().Err(err).Str("tenant_id", tenantID).Msg("gagal generate invoice on-demand")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal generate invoice jatuh tempo")
	}

	return domain.SuccessResponse(c, fiber.StatusOK, fiber.Map{
		"message": "generate invoice jatuh tempo selesai",
	})
}

// CreatePrepaid menangani POST /v1/invoices/prepaid.
// Membuat invoice prepaid untuk beberapa bulan sekaligus.
func (h *InvoiceHandler) CreatePrepaid(c *fiber.Ctx) error {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	var req domain.CreatePrepaidInvoiceRequest
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

	invoice, err := h.invoiceUsecase.CreatePrepaid(c.Context(), tenantID, req, actor)
	if err != nil {
		return h.mapInvoiceError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusCreated, invoice)
}

// Edit menangani PUT /v1/invoices/:id.
// Memperbarui invoice yang masih berstatus belum_bayar.
func (h *InvoiceHandler) Edit(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "invoice ID wajib diisi")
	}

	var req domain.EditInvoiceRequest
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

	invoice, err := h.invoiceUsecase.Edit(c.Context(), id, req, actor)
	if err != nil {
		return h.mapInvoiceError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, invoice)
}

// extractActor mengambil informasi aktor dari Fiber locals (di-set oleh auth middleware).
func (h *InvoiceHandler) extractActor(c *fiber.Ctx) domain.ActorInfo {
	actorID, _ := c.Locals("user_id").(string)
	actorName, _ := c.Locals("user_name").(string)
	return domain.ActorInfo{
		ActorID:   actorID,
		ActorName: actorName,
	}
}

// mapInvoiceError memetakan domain error ke HTTP error response untuk invoice.
func (h *InvoiceHandler) mapInvoiceError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domain.ErrInvoiceNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "INVOICE_NOT_FOUND", err.Error())
	case errors.Is(err, domain.ErrInvoiceNotEditable):
		return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "INVOICE_NOT_EDITABLE", err.Error())
	case errors.Is(err, domain.ErrInvoiceDuplicate):
		return domain.ErrorResponse(c, fiber.StatusConflict, "INVOICE_DUPLICATE", err.Error())
	case errors.Is(err, domain.ErrCustomerNotFound):
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "CUSTOMER_NOT_FOUND", err.Error())
	default:
		h.logger.Error().Err(err).Msg("internal error pada invoice handler")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "terjadi kesalahan internal")
	}
}
