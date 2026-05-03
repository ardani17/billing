// customer_handler.go menangani HTTP request untuk manajemen pelanggan.
// Termasuk: list, get, create, update, delete, dan stats.
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

// CustomerHandler menangani HTTP request untuk manajemen pelanggan.
type CustomerHandler struct {
	customerUsecase *usecase.CustomerUsecase
	validate        *validator.Validate
	logger          zerolog.Logger
}

// NewCustomerHandler membuat instance baru CustomerHandler.
func NewCustomerHandler(customerUsecase *usecase.CustomerUsecase, logger zerolog.Logger) *CustomerHandler {
	v := validator.New()
	RegisterCustomValidators(v)
	return &CustomerHandler{
		customerUsecase: customerUsecase,
		validate:        v,
		logger:          logger,
	}
}

// List menangani GET /v1/customers.
// Mengembalikan daftar pelanggan dengan paginasi, filter, dan sorting.
func (h *CustomerHandler) List(c *fiber.Ctx) error {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	var params domain.CustomerListParams
	params.TenantID = tenantID
	params.Page, _ = strconv.Atoi(c.Query("page", "1"))
	params.PageSize, _ = strconv.Atoi(c.Query("page_size", "25"))
	params.Search = c.Query("search")
	params.Status = c.Query("status")
	params.PackageID = c.Query("package_id")
	params.AreaID = c.Query("area_id")
	params.SortBy = c.Query("sort_by")
	params.SortOrder = c.Query("sort_order")

	if dueDateStr := c.Query("due_date"); dueDateStr != "" {
		dd, err := strconv.Atoi(dueDateStr)
		if err == nil {
			params.DueDate = &dd
		}
	}

	if err := h.validate.Struct(params); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			return domain.ErrorResponse(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "validasi gagal", mapValidationErrors(ve)...)
		}
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "VALIDATION_ERROR", err.Error())
	}

	result, err := h.customerUsecase.List(c.Context(), params)
	if err != nil {
		h.logger.Error().Err(err).Msg("gagal mengambil daftar pelanggan")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal mengambil daftar pelanggan")
	}

	return domain.SuccessResponse(c, fiber.StatusOK, result)
}

// Get menangani GET /v1/customers/:id.
// Mengembalikan detail pelanggan, opsional termasuk audit logs.
func (h *CustomerHandler) Get(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "customer ID wajib diisi")
	}

	includeAudit := strings.Contains(c.Query("include"), "audit_logs")

	detail, err := h.customerUsecase.GetByID(c.Context(), id, includeAudit)
	if err != nil {
		return h.mapCustomerError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, detail)
}

// Create menangani POST /v1/customers.
// Membuat pelanggan baru dengan status pending.
func (h *CustomerHandler) Create(c *fiber.Ctx) error {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	var req domain.CreateCustomerRequest
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

	customer, err := h.customerUsecase.Create(c.Context(), tenantID, req, actor)
	if err != nil {
		return h.mapCustomerError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusCreated, customer)
}

// Update menangani PUT /v1/customers/:id.
// Memperbarui data pelanggan.
func (h *CustomerHandler) Update(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "customer ID wajib diisi")
	}

	var req domain.UpdateCustomerRequest
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

	customer, err := h.customerUsecase.Update(c.Context(), id, req, actor)
	if err != nil {
		return h.mapCustomerError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, customer)
}

// Delete menangani DELETE /v1/customers/:id.
// Soft delete pelanggan dengan konfirmasi nama.
func (h *CustomerHandler) Delete(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "customer ID wajib diisi")
	}

	var req domain.DeleteCustomerRequest
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

	err := h.customerUsecase.SoftDelete(c.Context(), id, req.ConfirmationName, actor)
	if err != nil {
		return h.mapCustomerError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, fiber.Map{
		"message": "pelanggan berhasil dihapus",
	})
}

// Stats menangani GET /v1/customers/stats.
// Mengembalikan jumlah pelanggan per status.
func (h *CustomerHandler) Stats(c *fiber.Ctx) error {
	stats, err := h.customerUsecase.Stats(c.Context())
	if err != nil {
		h.logger.Error().Err(err).Msg("gagal mengambil statistik pelanggan")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal mengambil statistik pelanggan")
	}

	// Convert map[CustomerStatus]int64 to map[string]int64 for JSON
	result := make(map[string]int64)
	for status, count := range stats {
		result[string(status)] = count
	}

	return domain.SuccessResponse(c, fiber.StatusOK, result)
}

// extractActor mengambil informasi aktor dari Fiber locals (di-set oleh auth middleware).
func (h *CustomerHandler) extractActor(c *fiber.Ctx) usecase.ActorInfo {
	userID, _ := c.Locals("user_id").(string)
	userName, _ := c.Locals("user_name").(string)
	return usecase.ActorInfo{
		ID:   userID,
		Name: userName,
	}
}

// mapCustomerError memetakan domain error ke HTTP error response.
func (h *CustomerHandler) mapCustomerError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domain.ErrCustomerNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "CUSTOMER_NOT_FOUND", err.Error())
	case errors.Is(err, domain.ErrPhoneDuplicate):
		return domain.ErrorResponse(c, fiber.StatusConflict, "PHONE_DUPLICATE", err.Error())
	case errors.Is(err, domain.ErrInvalidStatusTransition):
		return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "INVALID_STATUS_TRANSITION", err.Error())
	case errors.Is(err, domain.ErrConfirmationMismatch):
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "CONFIRMATION_MISMATCH", err.Error())
	case errors.Is(err, domain.ErrSamePackage):
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "SAME_PACKAGE", err.Error())
	case errors.Is(err, domain.ErrPackageNotFound):
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "PACKAGE_NOT_FOUND", err.Error())
	default:
		h.logger.Error().Err(err).Msg("internal error pada customer handler")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "terjadi kesalahan internal")
	}
}
