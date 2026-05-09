// reseller_auth_handler.go menangani HTTP permintaan untuk autentikasi reseller.
// Termasuk: login, logout, refresh token.
// Reseller menggunakan phone+password, terpisah dari admin auth.
package handler

import (
	"errors"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/ispboss/ispboss/services/billing-api/internal/usecase"
)

// ResellerAuthHandler menangani HTTP permintaan untuk autentikasi reseller.
type ResellerAuthHandler struct {
	resellerAuthUsecase *usecase.ResellerAuthUsecase
	validate            *validator.Validate
	logger              zerolog.Logger
}

// NewResellerAuthHandler membuat instance baru ResellerAuthHandler.
// Mendaftarkan kustom validator phone_id untuk format telepon Indonesia.
func NewResellerAuthHandler(
	resellerAuthUsecase *usecase.ResellerAuthUsecase,
	logger zerolog.Logger,
) *ResellerAuthHandler {
	v := validator.New()
	RegisterCustomValidators(v)
	return &ResellerAuthHandler{
		resellerAuthUsecase: resellerAuthUsecase,
		validate:            v,
		logger:              logger,
	}
}

// Login menangani POST /v1/reseller/auth/login.
// Memverifikasi credential reseller (phone+password) dan mengembalikan JWT + refresh token.
func (h *ResellerAuthHandler) Login(c *fiber.Ctx) error {
	var req domain.ResellerLoginRequest
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

	resp, err := h.resellerAuthUsecase.Login(c.Context(), req)
	if err != nil {
		return h.mapResellerAuthError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, resp)
}

// Logout menangani POST /v1/reseller/auth/logout.
// Menghapus session reseller berdasarkan refresh token.
func (h *ResellerAuthHandler) Logout(c *fiber.Ctx) error {
	var body struct {
		RefreshToken string `json:"refresh_token" validate:"required"`
	}
	if err := c.BodyParser(&body); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "request body tidak valid")
	}

	if err := h.validate.Struct(body); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			return domain.ErrorResponse(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "validasi gagal", mapValidationErrors(ve)...)
		}
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "VALIDATION_ERROR", err.Error())
	}

	err := h.resellerAuthUsecase.Logout(c.Context(), body.RefreshToken)
	if err != nil {
		h.logger.Error().Err(err).Msg("gagal logout reseller")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal logout")
	}

	return domain.SuccessResponse(c, fiber.StatusOK, fiber.Map{
		"message": "berhasil logout",
	})
}

// Refresh menangani POST /v1/reseller/auth/refresh.
// Memperpanjang JWT reseller dengan refresh token.
func (h *ResellerAuthHandler) Refresh(c *fiber.Ctx) error {
	var body struct {
		RefreshToken string `json:"refresh_token" validate:"required"`
	}
	if err := c.BodyParser(&body); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "request body tidak valid")
	}

	if err := h.validate.Struct(body); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			return domain.ErrorResponse(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "validasi gagal", mapValidationErrors(ve)...)
		}
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "VALIDATION_ERROR", err.Error())
	}

	resp, err := h.resellerAuthUsecase.RefreshToken(c.Context(), body.RefreshToken)
	if err != nil {
		return h.mapResellerAuthError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, resp)
}

// mapResellerAuthError memetakan domain error ke HTTP error respons untuk auth reseller.
func (h *ResellerAuthHandler) mapResellerAuthError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domain.ErrResellerInvalidCredentials):
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "INVALID_CREDENTIALS", err.Error())
	case errors.Is(err, domain.ErrResellerAccountDisabled):
		return domain.ErrorResponse(c, fiber.StatusForbidden, "ACCOUNT_DISABLED", err.Error())
	case errors.Is(err, domain.ErrResellerAccountLocked):
		return domain.ErrorResponse(c, fiber.StatusTooManyRequests, "ACCOUNT_LOCKED", err.Error())
	case errors.Is(err, domain.ErrResellerNotFound):
		// Jangan expose bahwa reseller tidak ditemukan (cegah enumerasi)
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "INVALID_CREDENTIALS", domain.ErrResellerInvalidCredentials.Error())
	default:
		h.logger.Error().Err(err).Msg("internal error pada reseller auth handler")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "terjadi kesalahan internal")
	}
}
