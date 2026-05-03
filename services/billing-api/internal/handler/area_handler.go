// area_handler.go menangani HTTP request untuk manajemen area.
// Termasuk: list, get, create, update, dan delete.
package handler

import (
	"errors"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/ispboss/ispboss/services/billing-api/internal/usecase"
)

// AreaHandler menangani HTTP request untuk manajemen area.
type AreaHandler struct {
	areaUsecase *usecase.AreaUsecase
	validate    *validator.Validate
	logger      zerolog.Logger
}

// NewAreaHandler membuat instance baru AreaHandler.
func NewAreaHandler(areaUsecase *usecase.AreaUsecase, logger zerolog.Logger) *AreaHandler {
	return &AreaHandler{
		areaUsecase: areaUsecase,
		validate:    validator.New(),
		logger:      logger,
	}
}

// List menangani GET /v1/areas.
// Mengembalikan daftar area untuk tenant yang terautentikasi.
func (h *AreaHandler) List(c *fiber.Ctx) error {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	areas, err := h.areaUsecase.List(c.Context(), tenantID)
	if err != nil {
		h.logger.Error().Err(err).Str("tenant_id", tenantID).Msg("gagal mengambil daftar area")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal mengambil daftar area")
	}

	return domain.SuccessResponse(c, fiber.StatusOK, areas)
}

// Get menangani GET /v1/areas/:id.
// Mengembalikan detail area berdasarkan ID.
func (h *AreaHandler) Get(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "area ID wajib diisi")
	}

	area, err := h.areaUsecase.GetByID(c.Context(), id)
	if err != nil {
		return h.mapAreaError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, area)
}

// Create menangani POST /v1/areas.
// Membuat area baru.
func (h *AreaHandler) Create(c *fiber.Ctx) error {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	var req domain.CreateAreaRequest
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

	area, err := h.areaUsecase.Create(c.Context(), tenantID, req, actor)
	if err != nil {
		return h.mapAreaError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusCreated, area)
}

// Update menangani PUT /v1/areas/:id.
// Memperbarui data area.
func (h *AreaHandler) Update(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "area ID wajib diisi")
	}

	var req domain.UpdateAreaRequest
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

	area, err := h.areaUsecase.Update(c.Context(), id, req, actor)
	if err != nil {
		return h.mapAreaError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, area)
}

// Delete menangani DELETE /v1/areas/:id.
// Menghapus area jika tidak ada pelanggan yang terkait.
func (h *AreaHandler) Delete(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "area ID wajib diisi")
	}

	actor := h.extractActor(c)

	err := h.areaUsecase.Delete(c.Context(), id, actor)
	if err != nil {
		return h.mapAreaError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, fiber.Map{
		"message": "area berhasil dihapus",
	})
}

// extractActor mengambil informasi aktor dari Fiber locals.
func (h *AreaHandler) extractActor(c *fiber.Ctx) usecase.ActorInfo {
	userID, _ := c.Locals("user_id").(string)
	userName, _ := c.Locals("user_name").(string)
	return usecase.ActorInfo{
		ID:   userID,
		Name: userName,
	}
}

// mapAreaError memetakan domain error ke HTTP error response untuk area.
func (h *AreaHandler) mapAreaError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domain.ErrAreaNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "AREA_NOT_FOUND", err.Error())
	case errors.Is(err, domain.ErrAreaNameDuplicate):
		return domain.ErrorResponse(c, fiber.StatusConflict, "AREA_NAME_DUPLICATE", err.Error())
	case errors.Is(err, domain.ErrAreaHasCustomers):
		return domain.ErrorResponse(c, fiber.StatusConflict, "AREA_HAS_CUSTOMERS", err.Error())
	default:
		h.logger.Error().Err(err).Msg("internal error pada area handler")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "terjadi kesalahan internal")
	}
}
