// voucher_print.go menangani HTTP request untuk generate PDF voucher.
// Termasuk: bulk print voucher ke format PDF.
package handler

import (
	"errors"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/ispboss/ispboss/services/billing-api/internal/usecase"
)

// VoucherPrintHandler menangani HTTP request untuk PDF generation voucher.
type VoucherPrintHandler struct {
	voucherPrintUsecase *usecase.VoucherPrintUsecase
	validate            *validator.Validate
	logger              zerolog.Logger
}

// NewVoucherPrintHandler membuat instance baru VoucherPrintHandler.
func NewVoucherPrintHandler(
	voucherPrintUsecase *usecase.VoucherPrintUsecase,
	logger zerolog.Logger,
) *VoucherPrintHandler {
	v := validator.New()
	RegisterCustomValidators(v)
	return &VoucherPrintHandler{
		voucherPrintUsecase: voucherPrintUsecase,
		validate:            v,
		logger:              logger,
	}
}

// BulkPrint menangani POST /v1/vouchers/bulk/print.
// Menghasilkan PDF berisi kartu-kartu voucher dalam layout grid.
// Mengambil informasi tenant dari JWT claims untuk ditampilkan di kartu.
func (h *VoucherPrintHandler) BulkPrint(c *fiber.Ctx) error {
	var req domain.BulkVoucherIDsRequest
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

	// Ambil informasi tenant dari JWT locals (di-set oleh auth middleware)
	tenantName, _ := c.Locals("tenant_name").(string)
	tenantPhone, _ := c.Locals("tenant_phone").(string)

	pdfBytes, err := h.voucherPrintUsecase.GeneratePDF(c.Context(), req.VoucherIDs, tenantName, tenantPhone)
	if err != nil {
		if errors.Is(err, domain.ErrVoucherNotFound) {
			return domain.ErrorResponse(c, fiber.StatusNotFound, "VOUCHER_NOT_FOUND", err.Error())
		}
		h.logger.Error().Err(err).Msg("gagal generate PDF voucher")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal generate PDF voucher")
	}

	// Set header untuk response PDF
	c.Set("Content-Type", "application/pdf")
	c.Set("Content-Disposition", "attachment; filename=vouchers.pdf")

	return c.Send(pdfBytes)
}
