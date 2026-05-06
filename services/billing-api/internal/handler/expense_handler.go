// expense_handler.go menangani HTTP request untuk CRUD pengeluaran.
// Termasuk: List, Create, Update, Delete expenses.
// Method kategori ada di expense_handler_category.go.
package handler

import (
	"errors"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// ExpenseHandler menangani HTTP request untuk pengeluaran.
type ExpenseHandler struct {
	expenseUsecase domain.ExpenseUsecase
	validate       *validator.Validate
	logger         zerolog.Logger
}

// NewExpenseHandler membuat instance baru ExpenseHandler.
func NewExpenseHandler(expenseUsecase domain.ExpenseUsecase, logger zerolog.Logger) *ExpenseHandler {
	return &ExpenseHandler{
		expenseUsecase: expenseUsecase,
		validate:       validator.New(),
		logger:         logger,
	}
}

// List menangani GET /v1/expenses.
// Mengembalikan daftar pengeluaran dengan filter periode dan kategori.
func (h *ExpenseHandler) List(c *fiber.Ctx) error {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	periodStart := c.Query("period_start")
	periodEnd := c.Query("period_end")
	if periodStart == "" || periodEnd == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "period_start dan period_end wajib diisi")
	}

	ps, err := time.Parse("2006-01-02", periodStart)
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "format period_start tidak valid")
	}
	pe, err := time.Parse("2006-01-02", periodEnd)
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "format period_end tidak valid")
	}

	categoryID := c.Query("category_id")

	expenses, err := h.expenseUsecase.List(c.Context(), tenantID, ps, pe, categoryID)
	if err != nil {
		h.logger.Error().Err(err).Msg("gagal mengambil daftar pengeluaran")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal mengambil daftar pengeluaran")
	}

	return domain.SuccessResponse(c, fiber.StatusOK, expenses)
}

// Create menangani POST /v1/expenses.
// Membuat pengeluaran baru.
func (h *ExpenseHandler) Create(c *fiber.Ctx) error {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	var req domain.CreateExpenseRequest
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

	expense, err := h.expenseUsecase.Create(c.Context(), tenantID, req, actor)
	if err != nil {
		return h.mapExpenseError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusCreated, expense)
}

// Update menangani PUT /v1/expenses/:id.
// Memperbarui data pengeluaran.
func (h *ExpenseHandler) Update(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "expense ID wajib diisi")
	}

	var req domain.UpdateExpenseRequest
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

	expense, err := h.expenseUsecase.Update(c.Context(), id, req, actor)
	if err != nil {
		return h.mapExpenseError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, expense)
}

// Delete menangani DELETE /v1/expenses/:id.
// Soft delete pengeluaran.
func (h *ExpenseHandler) Delete(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "expense ID wajib diisi")
	}

	if err := h.expenseUsecase.Delete(c.Context(), id, h.extractActor(c)); err != nil {
		return h.mapExpenseError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// extractActor mengambil informasi aktor dari Fiber locals.
func (h *ExpenseHandler) extractActor(c *fiber.Ctx) domain.ActorInfo {
	actorID, _ := c.Locals("user_id").(string)
	actorName, _ := c.Locals("user_name").(string)
	return domain.ActorInfo{
		ActorID:   actorID,
		ActorName: actorName,
	}
}

// mapExpenseError memetakan domain error ke HTTP error response untuk pengeluaran.
func (h *ExpenseHandler) mapExpenseError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domain.ErrExpenseNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "EXPENSE_NOT_FOUND", err.Error())
	case errors.Is(err, domain.ErrExpenseCategoryNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "CATEGORY_NOT_FOUND", err.Error())
	case errors.Is(err, domain.ErrCategoryHasExpenses):
		return domain.ErrorResponse(c, fiber.StatusConflict, "CATEGORY_HAS_EXPENSES", err.Error())
	case errors.Is(err, domain.ErrCategoryNameDuplicate):
		return domain.ErrorResponse(c, fiber.StatusConflict, "CATEGORY_NAME_DUPLICATE", err.Error())
	default:
		h.logger.Error().Err(err).Msg("internal error pada expense handler")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "terjadi kesalahan internal")
	}
}
