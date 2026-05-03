// expense_handler_category.go berisi method-method kategori pengeluaran pada ExpenseHandler.
// Termasuk: ListCategories, CreateCategory, UpdateCategory, DeleteCategory.
package handler

import (
	"errors"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// ListCategories menangani GET /v1/expenses/categories.
// Mengembalikan semua kategori pengeluaran aktif.
func (h *ExpenseHandler) ListCategories(c *fiber.Ctx) error {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	categories, err := h.expenseUsecase.ListCategories(c.Context(), tenantID)
	if err != nil {
		h.logger.Error().Err(err).Msg("gagal mengambil kategori pengeluaran")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal mengambil kategori pengeluaran")
	}

	return domain.SuccessResponse(c, fiber.StatusOK, categories)
}

// CreateCategory menangani POST /v1/expenses/categories.
// Membuat kategori pengeluaran baru.
func (h *ExpenseHandler) CreateCategory(c *fiber.Ctx) error {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	var req domain.CreateCategoryRequest
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

	category, err := h.expenseUsecase.CreateCategory(c.Context(), tenantID, req.Name)
	if err != nil {
		return h.mapExpenseError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusCreated, category)
}

// UpdateCategory menangani PUT /v1/expenses/categories/:id.
// Memperbarui nama kategori pengeluaran.
func (h *ExpenseHandler) UpdateCategory(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "category ID wajib diisi")
	}

	var req domain.UpdateCategoryRequest
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

	category, err := h.expenseUsecase.UpdateCategory(c.Context(), id, req.Name)
	if err != nil {
		return h.mapExpenseError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, category)
}

// DeleteCategory menangani DELETE /v1/expenses/categories/:id.
// Menghapus kategori (ditolak jika masih ada pengeluaran terkait).
func (h *ExpenseHandler) DeleteCategory(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "category ID wajib diisi")
	}

	if err := h.expenseUsecase.DeleteCategory(c.Context(), id); err != nil {
		return h.mapExpenseError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}
