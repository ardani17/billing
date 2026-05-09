// recurring_item_handler.go menangani HTTP permintaan untuk item berulangs pelanggan.
package handler

import (
	"errors"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/ispboss/ispboss/services/billing-api/internal/usecase"
)

// RecurringItemHandler menangani HTTP permintaan untuk item berulangs pelanggan.
type RecurringItemHandler struct {
	recurringItemUsecase *usecase.RecurringItemUsecase
	validate             *validator.Validate
	logger               zerolog.Logger
}

// NewRecurringItemHandler membuat instance baru RecurringItemHandler.
func NewRecurringItemHandler(recurringItemUsecase *usecase.RecurringItemUsecase, logger zerolog.Logger) *RecurringItemHandler {
	return &RecurringItemHandler{
		recurringItemUsecase: recurringItemUsecase,
		validate:             validator.New(),
		logger:               logger,
	}
}

// List menangani GET /v1/customers/:id/berulang-items.
// Mengembalikan daftar item berulangs untuk pelanggan tertentu.
func (h *RecurringItemHandler) List(c *fiber.Ctx) error {
	customerID := c.Params("id")
	if customerID == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "customer ID wajib diisi")
	}

	items, err := h.recurringItemUsecase.List(c.Context(), customerID)
	if err != nil {
		return h.mapRecurringItemError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, items)
}

// Buat menangani POST /v1/customers/:id/berulang-items.
// Membuat item berulang baru untuk pelanggan.
func (h *RecurringItemHandler) Create(c *fiber.Ctx) error {
	customerID := c.Params("id")
	if customerID == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "customer ID wajib diisi")
	}

	var req domain.CreateRecurringItemRequest
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

	item, err := h.recurringItemUsecase.Create(c.Context(), customerID, req, actor)
	if err != nil {
		return h.mapRecurringItemError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusCreated, item)
}

// Perbarui menangani PUT /v1/customers/:id/berulang-items/:item_id.
// Memperbarui item berulang yang ada.
func (h *RecurringItemHandler) Update(c *fiber.Ctx) error {
	customerID := c.Params("id")
	if customerID == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "customer ID wajib diisi")
	}

	itemID := c.Params("item_id")
	if itemID == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "recurring item ID wajib diisi")
	}

	var req domain.UpdateRecurringItemRequest
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

	item, err := h.recurringItemUsecase.Update(c.Context(), customerID, itemID, req, actor)
	if err != nil {
		return h.mapRecurringItemError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, item)
}

// Hapus menangani DELETE /v1/customers/:id/berulang-items/:item_id.
// Menonaktifkan item berulang (hapus lunak).
func (h *RecurringItemHandler) Delete(c *fiber.Ctx) error {
	customerID := c.Params("id")
	if customerID == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "customer ID wajib diisi")
	}

	itemID := c.Params("item_id")
	if itemID == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "recurring item ID wajib diisi")
	}

	actor := h.extractActor(c)

	err := h.recurringItemUsecase.Delete(c.Context(), customerID, itemID, actor)
	if err != nil {
		return h.mapRecurringItemError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, fiber.Map{
		"message": "recurring item berhasil dihapus",
	})
}

// extractActor mengambil informasi aktor dari Fiber locals (di-atur oleh auth middleware).
func (h *RecurringItemHandler) extractActor(c *fiber.Ctx) domain.ActorInfo {
	actorID, _ := c.Locals("user_id").(string)
	actorName, _ := c.Locals("user_name").(string)
	return domain.ActorInfo{
		ActorID:   actorID,
		ActorName: actorName,
	}
}

// mapRecurringItemError memetakan domain error ke HTTP error respons untuk item berulangs.
func (h *RecurringItemHandler) mapRecurringItemError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domain.ErrRecurringItemNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "RECURRING_ITEM_NOT_FOUND", err.Error())
	case errors.Is(err, domain.ErrCustomerNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "CUSTOMER_NOT_FOUND", err.Error())
	default:
		h.logger.Error().Err(err).Msg("internal error pada recurring item handler")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "terjadi kesalahan internal")
	}
}
