// customer_action.go menangani HTTP request untuk aksi pelanggan.
// Termasuk: isolir, activate, dan change-package.
package handler

import (
	"errors"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// Isolir menangani POST /v1/customers/:id/isolir.
// Mentransisikan status pelanggan dari aktif ke isolir.
func (h *CustomerHandler) Isolir(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "customer ID wajib diisi")
	}

	actor := h.extractActor(c)

	customer, err := h.customerUsecase.Isolir(c.Context(), id, actor)
	if err != nil {
		return h.mapCustomerError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, customer)
}

// Activate menangani POST /v1/customers/:id/activate.
// Mentransisikan status pelanggan ke aktif.
func (h *CustomerHandler) Activate(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "customer ID wajib diisi")
	}

	actor := h.extractActor(c)

	customer, err := h.customerUsecase.Activate(c.Context(), id, actor)
	if err != nil {
		return h.mapCustomerError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, customer)
}

// ChangePackage menangani POST /v1/customers/:id/change-package.
// Mengubah paket pelanggan.
func (h *CustomerHandler) ChangePackage(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "customer ID wajib diisi")
	}

	var req domain.ChangePackageRequest
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

	customer, err := h.customerUsecase.ChangePackage(c.Context(), id, req.PackageID, actor)
	if err != nil {
		return h.mapCustomerError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, customer)
}
