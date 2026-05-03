// loss_calc_handler.go menangani HTTP request untuk loss calculator.
// Menerima parameter input, panggil domain CalculateLoss, return hasil.
package handler

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// LossCalcHandler menangani HTTP request untuk kalkulasi optical loss budget.
type LossCalcHandler struct{}

// NewLossCalcHandler membuat instance baru LossCalcHandler.
func NewLossCalcHandler() *LossCalcHandler {
	return &LossCalcHandler{}
}

// CalculateLoss menangani POST /loss-calculator.
// Parse input, validasi, hitung optical loss budget, return hasil.
func (h *LossCalcHandler) CalculateLoss(c *fiber.Ctx) error {
	var input domain.LossCalculatorInput
	if err := c.BodyParser(&input); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "request body tidak valid")
	}

	// Validasi input menggunakan domain validator
	if err := domain.ValidateLossInput(input); err != nil {
		return h.mapError(c, err)
	}

	// Hitung loss budget menggunakan pure function domain
	result := domain.CalculateLoss(input)

	return domain.SuccessResponse(c, fiber.StatusOK, result)
}

// mapError memetakan domain error loss calculator ke HTTP error response.
func (h *LossCalcHandler) mapError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domain.ErrInvalidLossInput):
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "INVALID_LOSS_INPUT", err.Error())
	default:
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "terjadi kesalahan internal")
	}
}
