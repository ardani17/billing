// reseller_action.go menangani HTTP permintaan untuk aksi reseller (admin).
// Termasuk: suspend, aktifkan, deactivate, reset-password, deposit, withdraw.
package handler

import (
	"errors"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/ispboss/ispboss/services/billing-api/internal/usecase"
)

// ResellerActionHandler menangani HTTP permintaan untuk aksi reseller
// (suspend, aktifkan, deactivate, reset-password, deposit, withdraw).
type ResellerActionHandler struct {
	resellerActionUsecase *usecase.ResellerActionUsecase
	validate              *validator.Validate
	logger                zerolog.Logger
}

// NewResellerActionHandler membuat instance baru ResellerActionHandler.
// Mendaftarkan kustom validator untuk format telepon Indonesia.
func NewResellerActionHandler(resellerActionUsecase *usecase.ResellerActionUsecase, logger zerolog.Logger) *ResellerActionHandler {
	v := validator.New()
	RegisterCustomValidators(v)
	return &ResellerActionHandler{
		resellerActionUsecase: resellerActionUsecase,
		validate:              v,
		logger:                logger,
	}
}

// Suspend menangani POST /v1/resellers/:id/suspend.
// Mentransisikan status reseller dari aktif ke suspended.
func (h *ResellerActionHandler) Suspend(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "reseller ID wajib diisi")
	}

	actor := h.extractActor(c)

	reseller, err := h.resellerActionUsecase.Suspend(c.Context(), id, actor)
	if err != nil {
		return h.mapResellerError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, reseller)
}

// Activate menangani POST /v1/resellers/:id/aktifkan.
// Mentransisikan status reseller dari suspended ke aktif.
func (h *ResellerActionHandler) Activate(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "reseller ID wajib diisi")
	}

	actor := h.extractActor(c)

	reseller, err := h.resellerActionUsecase.Activate(c.Context(), id, actor)
	if err != nil {
		return h.mapResellerError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, reseller)
}

// Deactivate menangani POST /v1/resellers/:id/deactivate.
// Mentransisikan status reseller ke nonaktif (status akhir).
// Memerlukan confirmation_name yang cocok dengan nama reseller.
func (h *ResellerActionHandler) Deactivate(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "reseller ID wajib diisi")
	}

	var req domain.DeactivateResellerRequest
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

	reseller, err := h.resellerActionUsecase.Deactivate(c.Context(), id, req.ConfirmationName, actor)
	if err != nil {
		return h.mapResellerError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, reseller)
}

// ResetPassword menangani POST /v1/resellers/:id/reset-password.
// Menghasilkan password baru acak dan mengembalikan password plaintext.
func (h *ResellerActionHandler) ResetPassword(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "reseller ID wajib diisi")
	}

	actor := h.extractActor(c)

	plaintext, err := h.resellerActionUsecase.ResetPassword(c.Context(), id, actor)
	if err != nil {
		return h.mapResellerError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, fiber.Map{
		"password": plaintext,
	})
}

// Deposit menangani POST /v1/resellers/:id/deposit.
// Menambah saldo reseller secara atomik.
func (h *ResellerActionHandler) Deposit(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "reseller ID wajib diisi")
	}

	var req domain.DepositRequest
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

	reseller, err := h.resellerActionUsecase.Deposit(c.Context(), id, req, actor)
	if err != nil {
		return h.mapResellerError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, reseller)
}

// Withdraw menangani POST /v1/resellers/:id/withdraw.
// Mengurangi saldo reseller secara atomik.
// Mengembalikan error jika saldo tidak mencukupi.
func (h *ResellerActionHandler) Withdraw(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "reseller ID wajib diisi")
	}

	var req domain.WithdrawRequest
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

	reseller, err := h.resellerActionUsecase.Withdraw(c.Context(), id, req, actor)
	if err != nil {
		return h.mapResellerError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, reseller)
}

// extractActor mengambil informasi aktor dari Fiber locals (di-atur oleh auth middleware).
func (h *ResellerActionHandler) extractActor(c *fiber.Ctx) domain.ActorInfo {
	actorID, _ := c.Locals("user_id").(string)
	actorName, _ := c.Locals("user_name").(string)
	return domain.ActorInfo{
		ActorID:   actorID,
		ActorName: actorName,
	}
}

// mapResellerError memetakan domain error ke HTTP error respons untuk aksi reseller.
// Menggunakan pemetaan yang sama dengan reseller_handler.go.
func (h *ResellerActionHandler) mapResellerError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domain.ErrResellerNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "RESELLER_NOT_FOUND", err.Error())
	case errors.Is(err, domain.ErrResellerPhoneDuplicate):
		return domain.ErrorResponse(c, fiber.StatusConflict, "PHONE_DUPLICATE", err.Error())
	case errors.Is(err, domain.ErrInvalidResellerTransition):
		return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "INVALID_STATUS_TRANSITION", err.Error())
	case errors.Is(err, domain.ErrResellerAccountDisabled):
		return domain.ErrorResponse(c, fiber.StatusForbidden, "ACCOUNT_DISABLED", err.Error())
	case errors.Is(err, domain.ErrInsufficientBalance):
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "INSUFFICIENT_BALANCE", err.Error())
	case errors.Is(err, domain.ErrDailyLimitExceeded):
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "DAILY_LIMIT_EXCEEDED", err.Error())
	case errors.Is(err, domain.ErrConfirmationMismatch):
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "CONFIRMATION_MISMATCH", err.Error())
	case errors.Is(err, domain.ErrPackageNotActive):
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "PACKAGE_NOT_ACTIVE", err.Error())
	default:
		h.logger.Error().Err(err).Msg("internal error pada reseller action handler")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "terjadi kesalahan internal")
	}
}
