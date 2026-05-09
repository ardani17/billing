// reseller_handler.go menangani HTTP permintaan untuk manajemen reseller (admin).
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

// ResellerHandler menangani HTTP permintaan untuk manajemen reseller.
type ResellerHandler struct {
	resellerUsecase *usecase.ResellerUsecase
	validate        *validator.Validate
	logger          zerolog.Logger
}

// NewResellerHandler membuat instance baru ResellerHandler.
// Mendaftarkan kustom validator phone_id untuk format telepon Indonesia.
func NewResellerHandler(resellerUsecase *usecase.ResellerUsecase, logger zerolog.Logger) *ResellerHandler {
	v := validator.New()
	RegisterCustomValidators(v)
	return &ResellerHandler{
		resellerUsecase: resellerUsecase,
		validate:        v,
		logger:          logger,
	}
}

// List menangani GET /v1/resellers.
// Mengembalikan daftar reseller dengan paginasi, filter, dan pengurutan.
func (h *ResellerHandler) List(c *fiber.Ctx) error {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	var params domain.ResellerListParams
	params.TenantID = tenantID
	params.Page, _ = strconv.Atoi(c.Query("page", "1"))
	params.PageSize, _ = strconv.Atoi(c.Query("page_size", "25"))
	params.Search = c.Query("search")
	params.Status = c.Query("status")
	params.SortBy = c.Query("sort_by")
	params.SortOrder = c.Query("sort_order")

	if err := h.validate.Struct(params); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			return domain.ErrorResponse(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "validasi gagal", mapValidationErrors(ve)...)
		}
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "VALIDATION_ERROR", err.Error())
	}

	result, err := h.resellerUsecase.List(c.Context(), params)
	if err != nil {
		h.logger.Error().Err(err).Msg("gagal mengambil daftar reseller")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal mengambil daftar reseller")
	}

	return domain.SuccessResponse(c, fiber.StatusOK, result)
}

// Get menangani GET /v1/resellers/:id.
// Mengembalikan detail reseller, opsional termasuk audit logs.
func (h *ResellerHandler) Get(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "reseller ID wajib diisi")
	}

	includeAudit := strings.Contains(c.Query("include"), "audit_logs")

	detail, err := h.resellerUsecase.GetByID(c.Context(), id, includeAudit)
	if err != nil {
		return h.mapResellerError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, detail)
}

// Buat menangani POST /v1/resellers.
// Membuat reseller baru dengan status aktif.
func (h *ResellerHandler) Create(c *fiber.Ctx) error {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	var req domain.CreateResellerRequest
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

	reseller, err := h.resellerUsecase.Create(c.Context(), tenantID, req, actor)
	if err != nil {
		return h.mapResellerError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusCreated, reseller)
}

// Perbarui menangani PUT /v1/resellers/:id.
// Memperbarui data reseller.
func (h *ResellerHandler) Update(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "reseller ID wajib diisi")
	}

	var req domain.UpdateResellerRequest
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

	reseller, err := h.resellerUsecase.Update(c.Context(), id, req, actor)
	if err != nil {
		return h.mapResellerError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, reseller)
}

// extractActor mengambil informasi aktor dari Fiber locals (di-atur oleh auth middleware).
func (h *ResellerHandler) extractActor(c *fiber.Ctx) domain.ActorInfo {
	actorID, _ := c.Locals("user_id").(string)
	actorName, _ := c.Locals("user_name").(string)
	return domain.ActorInfo{
		ActorID:   actorID,
		ActorName: actorName,
	}
}

// mapResellerError memetakan domain error ke HTTP error respons untuk reseller.
func (h *ResellerHandler) mapResellerError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domain.ErrResellerNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "RESELLER_NOT_FOUND", err.Error())
	case errors.Is(err, domain.ErrResellerPhoneDuplicate):
		return domain.ErrorResponse(c, fiber.StatusConflict, "PHONE_DUPLICATE", err.Error())
	case errors.Is(err, domain.ErrInvalidResellerTransition):
		return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "INVALID_STATUS_TRANSITION", err.Error())
	case errors.Is(err, domain.ErrResellerAccountDisabled):
		return domain.ErrorResponse(c, fiber.StatusForbidden, "ACCOUNT_DISABLED", err.Error())
	case errors.Is(err, domain.ErrInsufficientBalance):
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "INSUFFICIENT_BALANCE", err.Error())
	case errors.Is(err, domain.ErrDailyLimitExceeded):
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "DAILY_LIMIT_EXCEEDED", err.Error())
	case errors.Is(err, domain.ErrConfirmationMismatch):
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "CONFIRMATION_MISMATCH", err.Error())
	case errors.Is(err, domain.ErrPackageNotActive):
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "PACKAGE_NOT_ACTIVE", err.Error())
	default:
		h.logger.Error().Err(err).Msg("internal error pada reseller handler")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "terjadi kesalahan internal")
	}
}
