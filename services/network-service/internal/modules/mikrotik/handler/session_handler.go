// session_handler.go menangani HTTP permintaan untuk manajemen PPPoE active sessions.
// Termasuk: list sessions, disconnect session, dan session count.
package handler

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
	"github.com/ispboss/ispboss/services/network-service/internal/modules/mikrotik/usecase"
)

// SessionHandler menangani HTTP permintaan untuk operasi PPPoE session.
type SessionHandler struct {
	manager usecase.PPPoEManager
	logger  zerolog.Logger
}

// NewSessionHandler membuat instance baru SessionHandler.
func NewSessionHandler(manager usecase.PPPoEManager, logger zerolog.Logger) *SessionHandler {
	return &SessionHandler{
		manager: manager,
		logger:  logger,
	}
}

// GetSessions menangani GET /api/v1/mikrotik/routers/:id/pppoe/sessions.
// Mengambil daftar active PPPoE sessions dari router secara langsung.
func (h *SessionHandler) GetSessions(c *fiber.Ctx) error {
	routerID := c.Params("id")
	if routerID == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "router ID wajib diisi")
	}

	sessions, err := h.manager.GetActiveSessions(c.UserContext(), routerID)
	if err != nil {
		return h.mapError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, sessions)
}

// DisconnectSession menangani POST /api/v1/mikrotik/routers/:id/pppoe/sessions/:session_id/disconnect.
// Memutus satu active PPPoE session di router.
func (h *SessionHandler) DisconnectSession(c *fiber.Ctx) error {
	routerID := c.Params("id")
	if routerID == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "router ID wajib diisi")
	}

	sessionID := c.Params("session_id")
	if sessionID == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "session ID wajib diisi")
	}

	if err := h.manager.DisconnectSession(c.UserContext(), routerID, sessionID); err != nil {
		return h.mapError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, fiber.Map{
		"message": "session berhasil diputus",
	})
}

// GetSessionCount menangani GET /api/v1/mikrotik/routers/:id/pppoe/sessions/count.
// Mengambil jumlah total active PPPoE sessions di router.
func (h *SessionHandler) GetSessionCount(c *fiber.Ctx) error {
	routerID := c.Params("id")
	if routerID == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "router ID wajib diisi")
	}

	count, err := h.manager.GetSessionCount(c.UserContext(), routerID)
	if err != nil {
		return h.mapError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, fiber.Map{
		"count": count,
	})
}

// mapError memetakan domain error ke HTTP error respons untuk session operations.
func (h *SessionHandler) mapError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domain.ErrRouterNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "ROUTER_NOT_FOUND", err.Error())
	case errors.Is(err, domain.ErrRouterOffline):
		return domain.ErrorResponse(c, fiber.StatusServiceUnavailable, "ROUTER_OFFLINE", err.Error())
	case errors.Is(err, domain.ErrSessionNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "SESSION_NOT_FOUND", err.Error())
	case errors.Is(err, domain.ErrConnectionFailed):
		return domain.ErrorResponse(c, fiber.StatusBadGateway, "CONNECTION_FAILED", err.Error())
	case errors.Is(err, domain.ErrConnectionTimeout):
		return domain.ErrorResponse(c, fiber.StatusGatewayTimeout, "CONNECTION_TIMEOUT", err.Error())
	default:
		h.logger.Error().Err(err).Msg("unhandled error di session handler")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "terjadi kesalahan internal")
	}
}
