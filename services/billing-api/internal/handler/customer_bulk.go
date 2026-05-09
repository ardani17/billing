// customer_bulk.go menangani HTTP permintaan untuk bulk actions pada pelanggan.
// Termasuk: isolir massal, aktivasi, notifikasi, ubah paket, edit, dan hapus.
package handler

import (
	"errors"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// BulkIsolir menangani POST /v1/customers/bulk/isolir.
// Mentransisikan status beberapa pelanggan ke isolir.
func (h *CustomerHandler) BulkIsolir(c *fiber.Ctx) error {
	var req domain.BulkIDsRequest
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

	result, err := h.customerUsecase.BulkIsolir(c.Context(), req.CustomerIDs, actor)
	if err != nil {
		h.logger.Error().Err(err).Msg("gagal bulk isolir")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal melakukan bulk isolir")
	}

	return domain.SuccessResponse(c, fiber.StatusOK, result)
}

// BulkActivate menangani POST /v1/customers/bulk/aktifkan.
// Mentransisikan status beberapa pelanggan ke aktif.
func (h *CustomerHandler) BulkActivate(c *fiber.Ctx) error {
	var req domain.BulkIDsRequest
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

	result, err := h.customerUsecase.BulkActivate(c.Context(), req.CustomerIDs, actor)
	if err != nil {
		h.logger.Error().Err(err).Msg("gagal bulk activate")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal melakukan bulk activate")
	}

	return domain.SuccessResponse(c, fiber.StatusOK, result)
}

// BulkNotify menangani POST /v1/customers/bulk/notification.
// Mengirim notifikasi ke beberapa pelanggan.
func (h *CustomerHandler) BulkNotify(c *fiber.Ctx) error {
	var req domain.BulkNotifyRequest
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

	result, err := h.customerUsecase.BulkNotify(c.Context(), req.CustomerIDs, req.TemplateID, actor)
	if err != nil {
		h.logger.Error().Err(err).Msg("gagal bulk notify")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal melakukan bulk notifikasi")
	}

	return domain.SuccessResponse(c, fiber.StatusOK, result)
}

// BulkChangePackage menangani POST /v1/customers/bulk/change-package.
// Mengubah paket beberapa pelanggan.
func (h *CustomerHandler) BulkChangePackage(c *fiber.Ctx) error {
	var req domain.BulkChangePackageRequest
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

	result, err := h.customerUsecase.BulkChangePackage(c.Context(), req.CustomerIDs, req.PackageID, actor)
	if err != nil {
		h.logger.Error().Err(err).Msg("gagal bulk change package")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal melakukan bulk change package")
	}

	return domain.SuccessResponse(c, fiber.StatusOK, result)
}

// BulkEdit menangani POST /v1/customers/bulk/edit.
// Mengubah field tertentu pada beberapa pelanggan.
func (h *CustomerHandler) BulkEdit(c *fiber.Ctx) error {
	var req domain.BulkEditRequest
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

	result, err := h.customerUsecase.BulkEdit(c.Context(), req.CustomerIDs, req.Fields, actor)
	if err != nil {
		h.logger.Error().Err(err).Msg("gagal bulk edit")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal melakukan bulk edit")
	}

	return domain.SuccessResponse(c, fiber.StatusOK, result)
}

// BulkDelete menangani DELETE /v1/customers/bulk.
// Soft hapus beberapa pelanggan.
func (h *CustomerHandler) BulkDelete(c *fiber.Ctx) error {
	var req domain.BulkIDsRequest
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

	result, err := h.customerUsecase.BulkDelete(c.Context(), req.CustomerIDs, actor)
	if err != nil {
		h.logger.Error().Err(err).Msg("gagal bulk delete")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal melakukan bulk delete")
	}

	return domain.SuccessResponse(c, fiber.StatusOK, result)
}
