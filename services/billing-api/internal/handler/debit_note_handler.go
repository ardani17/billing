// debit_note_handler.go menangani HTTP request untuk debit notes.
// Termasuk: create debit note.
package handler

import (
	"errors"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/ispboss/ispboss/services/billing-api/internal/usecase"
)

// DebitNoteHandler menangani HTTP request untuk debit notes.
type DebitNoteHandler struct {
	debitNoteUsecase *usecase.DebitNoteUsecase
	validate         *validator.Validate
	logger           zerolog.Logger
}

// NewDebitNoteHandler membuat instance baru DebitNoteHandler.
func NewDebitNoteHandler(debitNoteUsecase *usecase.DebitNoteUsecase, logger zerolog.Logger) *DebitNoteHandler {
	return &DebitNoteHandler{
		debitNoteUsecase: debitNoteUsecase,
		validate:         validator.New(),
		logger:           logger,
	}
}

// Create menangani POST /v1/debit-notes.
// Membuat debit note baru untuk tagihan tambahan.
func (h *DebitNoteHandler) Create(c *fiber.Ctx) error {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	var req domain.CreateDebitNoteRequest
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

	dn, err := h.debitNoteUsecase.Create(c.Context(), tenantID, req, actor)
	if err != nil {
		return h.mapDebitNoteError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusCreated, dn)
}

// extractActor mengambil informasi aktor dari Fiber locals (di-set oleh auth middleware).
func (h *DebitNoteHandler) extractActor(c *fiber.Ctx) domain.ActorInfo {
	actorID, _ := c.Locals("user_id").(string)
	actorName, _ := c.Locals("user_name").(string)
	return domain.ActorInfo{
		ActorID:   actorID,
		ActorName: actorName,
	}
}

// mapDebitNoteError memetakan domain error ke HTTP error response untuk debit notes.
func (h *DebitNoteHandler) mapDebitNoteError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domain.ErrCustomerNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "CUSTOMER_NOT_FOUND", err.Error())
	case errors.Is(err, domain.ErrDebitNoteNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "DEBIT_NOTE_NOT_FOUND", err.Error())
	default:
		h.logger.Error().Err(err).Msg("internal error pada debit note handler")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "terjadi kesalahan internal")
	}
}
