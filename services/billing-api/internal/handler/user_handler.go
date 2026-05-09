// user_handler.go menangani HTTP permintaan untuk manajemen user oleh tenant admin.
package handler

import (
	"errors"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/ispboss/ispboss/services/billing-api/internal/usecase"
)

// UserHandler menangani HTTP permintaan untuk manajemen user.
type UserHandler struct {
	userUsecase *usecase.UserManagementUsecase
	validate    *validator.Validate
	logger      zerolog.Logger
}

// NewUserHandler membuat instance baru UserHandler.
func NewUserHandler(userUsecase *usecase.UserManagementUsecase, logger zerolog.Logger) *UserHandler {
	return &UserHandler{
		userUsecase: userUsecase,
		validate:    validator.New(),
		logger:      logger,
	}
}

// List menangani GET /v1/settings/users.
// Mengembalikan daftar semua user dalam tenant yang sama.
func (h *UserHandler) List(c *fiber.Ctx) error {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	users, err := h.userUsecase.ListUsers(c.Context(), tenantID)
	if err != nil {
		h.logger.Error().Err(err).Str("tenant_id", tenantID).Msg("gagal mengambil daftar user")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal mengambil daftar user")
	}

	return domain.SuccessResponse(c, fiber.StatusOK, users)
}

// Buat menangani POST /v1/settings/users.
// Membuat user baru dalam tenant.
func (h *UserHandler) Create(c *fiber.Ctx) error {
	tenantID, ok := c.Locals("tenant_id").(string)
	if !ok || tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	var req domain.CreateUserRequest
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

	user, err := h.userUsecase.CreateUser(c.Context(), tenantID, req)
	if err != nil {
		return h.mapUserError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusCreated, user)
}

// Get menangani GET /v1/settings/users/:id.
// Mengembalikan detail user berdasarkan ID.
func (h *UserHandler) Get(c *fiber.Ctx) error {
	userID := c.Params("id")
	if userID == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "user ID wajib diisi")
	}

	user, err := h.userUsecase.GetUser(c.Context(), userID)
	if err != nil {
		return h.mapUserError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, user)
}

// Perbarui menangani PUT /v1/settings/users/:id.
// Memperbarui data user (name, phone, role).
func (h *UserHandler) Update(c *fiber.Ctx) error {
	userID := c.Params("id")
	if userID == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "user ID wajib diisi")
	}

	var req domain.UpdateUserRequest
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

	user, err := h.userUsecase.UpdateUser(c.Context(), userID, req)
	if err != nil {
		return h.mapUserError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, user)
}

// Hapus menangani DELETE /v1/settings/users/:id.
// Menghapus user secara permanen (memerlukan konfirmasi nama).
func (h *UserHandler) Delete(c *fiber.Ctx) error {
	targetUserID := c.Params("id")
	if targetUserID == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "user ID wajib diisi")
	}

	callerID, ok := c.Locals("user_id").(string)
	if !ok || callerID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "user tidak terautentikasi")
	}

	var body struct {
		ConfirmName string `json:"confirm_name" validate:"required"`
	}
	if err := c.BodyParser(&body); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "request body tidak valid")
	}

	if err := h.validate.Struct(body); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "validasi gagal", mapValidationErrors(ve)...)
		}
		return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
	}

	err := h.userUsecase.DeleteUser(c.Context(), targetUserID, callerID, body.ConfirmName)
	if err != nil {
		return h.mapUserError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, fiber.Map{
		"message": "user berhasil dihapus",
	})
}

// Deactivate menangani POST /v1/settings/users/:id/deactivate.
// Menonaktifkan user dan menghapus semua session aktif.
func (h *UserHandler) Deactivate(c *fiber.Ctx) error {
	targetUserID := c.Params("id")
	if targetUserID == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "user ID wajib diisi")
	}

	callerID, ok := c.Locals("user_id").(string)
	if !ok || callerID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "user tidak terautentikasi")
	}

	err := h.userUsecase.DeactivateUser(c.Context(), targetUserID, callerID)
	if err != nil {
		return h.mapUserError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, fiber.Map{
		"message": "user berhasil dinonaktifkan",
	})
}

// Activate menangani POST /v1/settings/users/:id/aktifkan.
// Mengaktifkan kembali user yang dinonaktifkan.
func (h *UserHandler) Activate(c *fiber.Ctx) error {
	targetUserID := c.Params("id")
	if targetUserID == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "user ID wajib diisi")
	}

	err := h.userUsecase.ActivateUser(c.Context(), targetUserID)
	if err != nil {
		return h.mapUserError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, fiber.Map{
		"message": "user berhasil diaktifkan",
	})
}

// ResetPassword menangani POST /v1/settings/users/:id/reset-password.
// Mengirim email reset password ke user target.
func (h *UserHandler) ResetPassword(c *fiber.Ctx) error {
	targetUserID := c.Params("id")
	if targetUserID == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "user ID wajib diisi")
	}

	err := h.userUsecase.ResetUserPassword(c.Context(), targetUserID)
	if err != nil {
		return h.mapUserError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, fiber.Map{
		"message": "email reset password telah dikirim",
	})
}

// mapUserError memetakan domain error ke HTTP error respons untuk user management.
func (h *UserHandler) mapUserError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domain.ErrEmailAlreadyExists):
		return domain.ErrorResponse(c, fiber.StatusConflict, "EMAIL_ALREADY_EXISTS", err.Error())
	case errors.Is(err, domain.ErrUserNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "USER_NOT_FOUND", err.Error())
	case errors.Is(err, domain.ErrInvalidRole):
		return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "INVALID_ROLE", err.Error())
	case errors.Is(err, domain.ErrCannotDeleteSelf):
		return domain.ErrorResponse(c, fiber.StatusForbidden, "CANNOT_DELETE_SELF", err.Error())
	case errors.Is(err, domain.ErrCannotDeactivateSelf):
		return domain.ErrorResponse(c, fiber.StatusForbidden, "CANNOT_DEACTIVATE_SELF", err.Error())
	case errors.Is(err, domain.ErrForbidden):
		return domain.ErrorResponse(c, fiber.StatusForbidden, "FORBIDDEN", err.Error())
	default:
		h.logger.Error().Err(err).Msg("internal error pada user handler")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "terjadi kesalahan internal")
	}
}
