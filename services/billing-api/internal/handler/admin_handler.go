// admin_handler.go menangani HTTP request untuk fitur super admin.
// Termasuk: start impersonation dan stop impersonation.
package handler

import (
	"errors"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/pkg/auth"
	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/ispboss/ispboss/services/billing-api/internal/usecase"
)

// AdminHandler menangani HTTP request untuk fitur super admin (impersonation).
type AdminHandler struct {
	impersonationUsecase *usecase.ImpersonationUsecase
	validate             *validator.Validate
	logger               zerolog.Logger
}

// NewAdminHandler membuat instance baru AdminHandler.
func NewAdminHandler(impersonationUsecase *usecase.ImpersonationUsecase, logger zerolog.Logger) *AdminHandler {
	return &AdminHandler{
		impersonationUsecase: impersonationUsecase,
		validate:             validator.New(),
		logger:               logger,
	}
}

// Start menangani POST /v1/admin/impersonate.
// Memulai impersonasi user target oleh super admin.
func (h *AdminHandler) Start(c *fiber.Ctx) error {
	impersonatorID, ok := c.Locals("user_id").(string)
	if !ok || impersonatorID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "user tidak terautentikasi")
	}

	var req domain.ImpersonateRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "request body tidak valid")
	}

	if err := h.validate.Struct(req); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "validasi gagal", mapValidationErrors(ve)...)
		}
		return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
	}

	tokenPair, err := h.impersonationUsecase.StartImpersonation(c.Context(), impersonatorID, req)
	if err != nil {
		return h.mapAdminError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, tokenPair)
}

// Stop menangani POST /v1/admin/stop-impersonate.
// Menghentikan impersonasi dan mengembalikan JWT ke claims super admin asli.
// JWT saat impersonasi berisi impersonator_id yang digunakan untuk mengambil data super admin.
func (h *AdminHandler) Stop(c *fiber.Ctx) error {
	// Ambil impersonator_id dari JWT claims yang disimpan oleh auth middleware.
	// Saat impersonasi aktif, JWT berisi claims target user + impersonator_id.
	impersonatorID := ""

	if claims, ok := c.Locals("claims").(*auth.Claims); ok && claims != nil {
		impersonatorID = claims.ImpersonatorID
	}

	if impersonatorID == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "impersonator_id tidak ditemukan, pastikan sedang dalam mode impersonasi")
	}

	tokenPair, err := h.impersonationUsecase.StopImpersonation(c.Context(), impersonatorID)
	if err != nil {
		return h.mapAdminError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, tokenPair)
}

// mapAdminError memetakan domain error ke HTTP error response untuk admin operations.
func (h *AdminHandler) mapAdminError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domain.ErrForbidden):
		return domain.ErrorResponse(c, fiber.StatusForbidden, "FORBIDDEN", err.Error())
	case errors.Is(err, domain.ErrUserNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "USER_NOT_FOUND", err.Error())
	default:
		h.logger.Error().Err(err).Msg("internal error pada admin handler")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "terjadi kesalahan internal")
	}
}
