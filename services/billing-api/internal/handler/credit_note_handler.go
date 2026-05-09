// credit_note_handler.go menangani HTTP permintaan untuk credit notes.
// Termasuk: buat credit note.
package handler

import (
	"errors"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/ispboss/ispboss/services/billing-api/internal/usecase"
)

// CreditNoteHandler menangani HTTP permintaan untuk credit notes.
type CreditNoteHandler struct {
	creditNoteUsecase *usecase.CreditNoteUsecase
	validate          *validator.Validate
	logger            zerolog.Logger
}

// NewCreditNoteHandler membuat instance baru CreditNoteHandler.
func NewCreditNoteHandler(creditNoteUsecase *usecase.CreditNoteUsecase, logger zerolog.Logger) *CreditNoteHandler {
	return &CreditNoteHandler{
		creditNoteUsecase: creditNoteUsecase,
		validate:          validator.New(),
		logger:            logger,
	}
}

// Buat menangani POST /v1/credit-notes.
// Membuat credit note baru untuk penyesuaian invoice.
func (h *CreditNoteHandler) Create(c *fiber.Ctx) error {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	var req domain.CreateCreditNoteRequest
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

	cn, err := h.creditNoteUsecase.Create(c.Context(), tenantID, req, actor)
	if err != nil {
		return h.mapCreditNoteError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusCreated, cn)
}

// extractActor mengambil informasi aktor dari Fiber locals (di-atur oleh auth middleware).
func (h *CreditNoteHandler) extractActor(c *fiber.Ctx) domain.ActorInfo {
	actorID, _ := c.Locals("user_id").(string)
	actorName, _ := c.Locals("user_name").(string)
	return domain.ActorInfo{
		ActorID:   actorID,
		ActorName: actorName,
	}
}

// mapCreditNoteError memetakan domain error ke HTTP error respons untuk credit notes.
func (h *CreditNoteHandler) mapCreditNoteError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domain.ErrInvoiceNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "INVOICE_NOT_FOUND", err.Error())
	case errors.Is(err, domain.ErrCreditNoteNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "CREDIT_NOTE_NOT_FOUND", err.Error())
	default:
		h.logger.Error().Err(err).Msg("internal error pada credit note handler")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "terjadi kesalahan internal")
	}
}
