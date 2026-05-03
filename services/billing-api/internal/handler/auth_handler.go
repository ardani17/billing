// auth_handler.go menangani HTTP request untuk endpoint autentikasi.
// Termasuk: register, login, Google OAuth, verifikasi email, reset password,
// refresh token, logout, get current user, dan change password.
package handler

import (
	"errors"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/ispboss/ispboss/services/billing-api/internal/usecase"
)

// AuthHandler menangani HTTP request untuk endpoint autentikasi.
type AuthHandler struct {
	authUsecase *usecase.AuthUsecase
	validate    *validator.Validate
	logger      zerolog.Logger
}

// NewAuthHandler membuat instance baru AuthHandler.
func NewAuthHandler(authUsecase *usecase.AuthUsecase, logger zerolog.Logger) *AuthHandler {
	return &AuthHandler{
		authUsecase: authUsecase,
		validate:    validator.New(),
		logger:      logger,
	}
}

// Register menangani POST /v1/auth/register.
// Mendaftarkan tenant baru beserta user tenant_admin.
func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req domain.RegisterRequest
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

	resp, err := h.authUsecase.Register(c.Context(), req)
	if err != nil {
		return h.mapAuthError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusCreated, resp)
}

// Login menangani POST /v1/auth/login.
// Memverifikasi credential dan mengembalikan JWT + refresh token.
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req domain.LoginRequest
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

	deviceInfo := c.Get("User-Agent")
	ipAddress := c.IP()

	resp, err := h.authUsecase.Login(c.Context(), req, deviceInfo, ipAddress)
	if err != nil {
		return h.mapAuthError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, resp)
}

// LoginWithGoogle menangani POST /v1/auth/google.
// Memverifikasi Google id_token dan login/register user.
func (h *AuthHandler) LoginWithGoogle(c *fiber.Ctx) error {
	var req domain.GoogleLoginRequest
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

	deviceInfo := c.Get("User-Agent")
	ipAddress := c.IP()

	resp, err := h.authUsecase.LoginWithGoogle(c.Context(), req, deviceInfo, ipAddress)
	if err != nil {
		return h.mapAuthError(c, err)
	}

	// Tentukan status code berdasarkan apakah user baru dibuat
	// Jika user baru (Google register), return 201; jika login, return 200
	return domain.SuccessResponse(c, fiber.StatusOK, resp)
}

// VerifyEmail menangani POST /v1/auth/verify-email.
// Memverifikasi email dengan token yang dikirim via email.
func (h *AuthHandler) VerifyEmail(c *fiber.Ctx) error {
	var body struct {
		Token string `json:"token" validate:"required"`
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

	deviceInfo := c.Get("User-Agent")
	ipAddress := c.IP()

	resp, err := h.authUsecase.VerifyEmail(c.Context(), body.Token, deviceInfo, ipAddress)
	if err != nil {
		return h.mapAuthError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, resp)
}

// ResendVerification menangani POST /v1/auth/resend-verification.
// Mengirim ulang email verifikasi.
func (h *AuthHandler) ResendVerification(c *fiber.Ctx) error {
	var body struct {
		Email string `json:"email" validate:"required,email"`
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

	err := h.authUsecase.ResendVerification(c.Context(), body.Email)
	if err != nil {
		return h.mapAuthError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, fiber.Map{
		"message": "email verifikasi telah dikirim ulang",
	})
}

// ForgotPassword menangani POST /v1/auth/forgot-password.
// Mengirim email reset password (selalu return 200 untuk mencegah email enumeration).
func (h *AuthHandler) ForgotPassword(c *fiber.Ctx) error {
	var body struct {
		Email string `json:"email" validate:"required,email"`
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

	err := h.authUsecase.ForgotPassword(c.Context(), body.Email)
	if err != nil {
		h.logger.Error().Err(err).Str("email", body.Email).Msg("gagal proses forgot password")
		// Tetap return 200 untuk mencegah email enumeration
	}

	return domain.SuccessResponse(c, fiber.StatusOK, fiber.Map{
		"message": "jika email terdaftar, instruksi reset password telah dikirim",
	})
}

// ResetPassword menangani POST /v1/auth/reset-password.
// Mereset password dengan token yang dikirim via email.
func (h *AuthHandler) ResetPassword(c *fiber.Ctx) error {
	var req domain.ResetPasswordRequest
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

	deviceInfo := c.Get("User-Agent")
	ipAddress := c.IP()

	resp, err := h.authUsecase.ResetPassword(c.Context(), req, deviceInfo, ipAddress)
	if err != nil {
		return h.mapAuthError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, resp)
}

// RefreshToken menangani POST /v1/auth/refresh.
// Memperpanjang JWT dengan refresh token.
func (h *AuthHandler) RefreshToken(c *fiber.Ctx) error {
	var body struct {
		RefreshToken string `json:"refresh_token" validate:"required"`
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

	deviceInfo := c.Get("User-Agent")
	ipAddress := c.IP()

	resp, err := h.authUsecase.RefreshToken(c.Context(), body.RefreshToken, deviceInfo, ipAddress)
	if err != nil {
		return h.mapAuthError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, resp)
}

// Logout menangani POST /v1/auth/logout.
// Menghapus session aktif berdasarkan refresh token.
func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	var body struct {
		RefreshToken string `json:"refresh_token" validate:"required"`
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

	err := h.authUsecase.Logout(c.Context(), body.RefreshToken)
	if err != nil {
		h.logger.Error().Err(err).Msg("gagal logout")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal logout")
	}

	return domain.SuccessResponse(c, fiber.StatusOK, fiber.Map{
		"message": "berhasil logout",
	})
}

// GetMe menangani GET /v1/auth/me.
// Mengembalikan data user yang sedang login berdasarkan JWT claims.
func (h *AuthHandler) GetMe(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "user tidak terautentikasi")
	}

	user, err := h.authUsecase.GetCurrentUser(c.Context(), userID)
	if err != nil {
		return h.mapAuthError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, user)
}

// ChangePassword menangani POST /v1/settings/security/change-password.
// Mengubah password user yang sedang login.
func (h *AuthHandler) ChangePassword(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "user tidak terautentikasi")
	}

	var req domain.ChangePasswordRequest
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

	// Ambil refresh token dari body untuk identifikasi session saat ini
	var tokenBody struct {
		RefreshToken string `json:"refresh_token"`
	}
	// Parse ulang body untuk mendapatkan refresh_token (opsional)
	_ = c.BodyParser(&tokenBody)

	err := h.authUsecase.ChangePassword(c.Context(), userID, req, tokenBody.RefreshToken)
	if err != nil {
		return h.mapAuthError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, fiber.Map{
		"message": "password berhasil diubah",
	})
}

// mapAuthError memetakan domain error ke HTTP error response.
func (h *AuthHandler) mapAuthError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domain.ErrEmailAlreadyExists):
		return domain.ErrorResponse(c, fiber.StatusConflict, "EMAIL_ALREADY_EXISTS", err.Error())
	case errors.Is(err, domain.ErrInvalidCredentials):
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "INVALID_CREDENTIALS", "email atau password salah")
	case errors.Is(err, domain.ErrEmailNotVerified):
		return domain.ErrorResponse(c, fiber.StatusForbidden, "EMAIL_NOT_VERIFIED", err.Error())
	case errors.Is(err, domain.ErrAccountDisabled):
		return domain.ErrorResponse(c, fiber.StatusForbidden, "ACCOUNT_DISABLED", err.Error())
	case errors.Is(err, domain.ErrAccountLocked):
		return domain.ErrorResponse(c, fiber.StatusTooManyRequests, "ACCOUNT_LOCKED", err.Error())
	case errors.Is(err, domain.ErrTokenExpired):
		return domain.ErrorResponse(c, fiber.StatusGone, "TOKEN_EXPIRED", err.Error())
	case errors.Is(err, domain.ErrTokenAlreadyUsed):
		return domain.ErrorResponse(c, fiber.StatusGone, "TOKEN_ALREADY_USED", err.Error())
	case errors.Is(err, domain.ErrTokenNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "TOKEN_NOT_FOUND", err.Error())
	case errors.Is(err, domain.ErrUserNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "USER_NOT_FOUND", err.Error())
	case errors.Is(err, domain.ErrResendCooldown):
		return domain.ErrorResponse(c, fiber.StatusTooManyRequests, "RESEND_COOLDOWN", err.Error())
	case errors.Is(err, domain.ErrForbidden):
		return domain.ErrorResponse(c, fiber.StatusForbidden, "FORBIDDEN", err.Error())
	default:
		// Log error internal, jangan expose detail ke client
		h.logger.Error().Err(err).Msg("internal error pada auth handler")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "terjadi kesalahan internal")
	}
}

// mapValidationErrors mengkonversi validator.ValidationErrors ke []domain.FieldError.
// Setiap field error dipetakan ke nama field JSON dan pesan error yang deskriptif.
func mapValidationErrors(ve validator.ValidationErrors) []domain.FieldError {
	fieldErrors := make([]domain.FieldError, 0, len(ve))
	for _, fe := range ve {
		fieldErrors = append(fieldErrors, domain.FieldError{
			Field:   toSnakeCase(fe.Field()),
			Message: validationMessage(fe),
		})
	}
	return fieldErrors
}

// validationMessage menghasilkan pesan error yang deskriptif berdasarkan tag validasi.
func validationMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return fe.Field() + " wajib diisi"
	case "email":
		return "format email tidak valid"
	case "min":
		return fe.Field() + " minimal " + fe.Param() + " karakter"
	case "max":
		return fe.Field() + " maksimal " + fe.Param() + " karakter"
	case "eqfield":
		return fe.Field() + " harus sama dengan " + fe.Param()
	case "oneof":
		return fe.Field() + " harus salah satu dari: " + fe.Param()
	case "startswith":
		return fe.Field() + " harus diawali dengan " + fe.Param()
	case "uuid":
		return fe.Field() + " harus berformat UUID"
	case "eq":
		return fe.Field() + " harus bernilai " + fe.Param()
	default:
		return fe.Field() + " tidak valid"
	}
}

// toSnakeCase mengkonversi PascalCase/camelCase ke snake_case sederhana.
// Menangani akronim berturut-turut seperti "ID", "URL", "API" dengan benar.
func toSnakeCase(s string) string {
	var result strings.Builder
	runes := []rune(s)
	for i, r := range runes {
		if i > 0 && r >= 'A' && r <= 'Z' {
			prev := runes[i-1]
			// Tambah underscore jika karakter sebelumnya huruf kecil,
			// atau jika ini awal kata baru setelah akronim (misal: "ID" diikuti "s" → "IDs").
			if prev >= 'a' && prev <= 'z' {
				result.WriteByte('_')
			} else if prev >= 'A' && prev <= 'Z' && i+1 < len(runes) && runes[i+1] >= 'a' && runes[i+1] <= 'z' {
				result.WriteByte('_')
			}
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}
