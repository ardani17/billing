// session_handler.go menangani HTTP request untuk manajemen session.
// Termasuk: list active sessions, revoke single session, dan revoke all other sessions.
package handler

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/ispboss/ispboss/services/billing-api/internal/usecase"
)

// SessionHandler menangani HTTP request untuk manajemen session.
type SessionHandler struct {
	sessionRepo domain.SessionRepository
	logger      zerolog.Logger
}

// NewSessionHandler membuat instance baru SessionHandler.
func NewSessionHandler(sessionRepo domain.SessionRepository, logger zerolog.Logger) *SessionHandler {
	return &SessionHandler{
		sessionRepo: sessionRepo,
		logger:      logger,
	}
}

// List menangani GET /v1/auth/sessions.
// Mengembalikan daftar semua session aktif untuk user yang sedang login.
// Setiap session memiliki flag is_current yang menunjukkan session saat ini.
func (h *SessionHandler) List(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "user tidak terautentikasi")
	}

	sessions, err := h.sessionRepo.ListByUserID(c.Context(), userID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", userID).Msg("gagal mengambil daftar session")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal mengambil daftar session")
	}

	// Tentukan session mana yang merupakan session saat ini
	// berdasarkan refresh token dari request body/header
	currentTokenHash := getCurrentTokenHash(c)
	for _, s := range sessions {
		if currentTokenHash != "" && s.TokenHash == currentTokenHash {
			s.IsCurrent = true
		}
	}

	return domain.SuccessResponse(c, fiber.StatusOK, sessions)
}

// Revoke menangani DELETE /v1/auth/sessions/:id.
// Menghapus session tertentu jika milik user yang sedang login.
func (h *SessionHandler) Revoke(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "user tidak terautentikasi")
	}

	sessionID := c.Params("id")
	if sessionID == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "session ID wajib diisi")
	}

	// Verifikasi bahwa session milik user yang sedang login
	sessions, err := h.sessionRepo.ListByUserID(c.Context(), userID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", userID).Msg("gagal mengambil daftar session")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal memverifikasi kepemilikan session")
	}

	owned := false
	for _, s := range sessions {
		if s.ID == sessionID {
			owned = true
			break
		}
	}

	if !owned {
		return domain.ErrorResponse(c, fiber.StatusForbidden, "FORBIDDEN", "session bukan milik user ini")
	}

	err = h.sessionRepo.DeleteByID(c.Context(), sessionID)
	if err != nil {
		if errors.Is(err, domain.ErrTokenNotFound) {
			return domain.ErrorResponse(c, fiber.StatusNotFound, "SESSION_NOT_FOUND", "session tidak ditemukan")
		}
		h.logger.Error().Err(err).Str("session_id", sessionID).Msg("gagal menghapus session")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal menghapus session")
	}

	return domain.SuccessResponse(c, fiber.StatusOK, fiber.Map{
		"message": "session berhasil dihapus",
	})
}

// RevokeOthers menangani DELETE /v1/auth/sessions?other=true.
// Menghapus semua session kecuali session saat ini.
func (h *SessionHandler) RevokeOthers(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "user tidak terautentikasi")
	}

	// Ambil refresh token dari request untuk identifikasi session saat ini
	currentTokenHash := getCurrentTokenHash(c)
	if currentTokenHash == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "refresh_token diperlukan untuk identifikasi session saat ini")
	}

	// Cari session saat ini berdasarkan token hash
	currentSession, err := h.sessionRepo.GetByTokenHash(c.Context(), currentTokenHash)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", userID).Msg("gagal mengidentifikasi session saat ini")
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "session saat ini tidak ditemukan")
	}

	// Hapus semua session kecuali session saat ini
	err = h.sessionRepo.DeleteOtherSessions(c.Context(), userID, currentSession.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", userID).Msg("gagal menghapus session lain")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal menghapus session lain")
	}

	return domain.SuccessResponse(c, fiber.StatusOK, fiber.Map{
		"message": "semua session lain berhasil dihapus",
	})
}

// getCurrentTokenHash mengambil hash refresh token dari request.
// Token bisa dikirim via query parameter atau header X-Refresh-Token.
func getCurrentTokenHash(c *fiber.Ctx) string {
	// Coba dari query parameter
	refreshToken := c.Query("refresh_token")
	if refreshToken == "" {
		// Coba dari header
		refreshToken = c.Get("X-Refresh-Token")
	}

	if refreshToken == "" {
		return ""
	}

	return usecase.HashToken(refreshToken)
}
