package handler

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
	"github.com/ispboss/ispboss/services/network-service/internal/modules/mikrotik/usecase"
)

type TerminalHandler struct {
	manager usecase.TerminalManager
	logger  zerolog.Logger
}

func NewTerminalHandler(manager usecase.TerminalManager, logger zerolog.Logger) *TerminalHandler {
	return &TerminalHandler{manager: manager, logger: logger}
}

func (h *TerminalHandler) Execute(c *fiber.Ctx) error {
	if !canUseMikroTikTerminal(c) {
		return domain.ErrorResponse(c, fiber.StatusForbidden, "FORBIDDEN", "role tidak diizinkan menjalankan terminal MikroTik")
	}
	var req domain.TerminalExecuteRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "payload tidak valid")
	}
	req.Command = strings.TrimSpace(req.Command)
	ctx := usecase.WithMikroTikAuditActor(c.UserContext(), fiberLocalsString(c, "user_id"), c.IP())
	result, err := h.manager.Execute(ctx, c.Params("id"), req)
	if err != nil {
		return h.mapError(c, err)
	}
	return domain.SuccessResponse(c, fiber.StatusOK, result)
}

func (h *TerminalHandler) ListAudit(c *fiber.Ctx) error {
	if !canUseMikroTikTerminal(c) {
		return domain.ErrorResponse(c, fiber.StatusForbidden, "FORBIDDEN", "role tidak diizinkan membaca audit terminal MikroTik")
	}
	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("page_size", "20"))
	result, err := h.manager.ListAudit(c.UserContext(), domain.MikroTikCommandAuditListParams{
		RouterID: c.Params("id"),
		Page:     page,
		PageSize: pageSize,
		Status:   strings.TrimSpace(c.Query("status")),
	})
	if err != nil {
		return h.mapError(c, err)
	}
	return domain.SuccessResponse(c, fiber.StatusOK, result)
}

func (h *TerminalHandler) mapError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domain.ErrRouterNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "ROUTER_NOT_FOUND", err.Error())
	case errors.Is(err, domain.ErrTerminalCommandDenied):
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "TERMINAL_COMMAND_DENIED", err.Error())
	case errors.Is(err, domain.ErrConnectionFailed):
		return domain.ErrorResponse(c, fiber.StatusBadGateway, "CONNECTION_FAILED", err.Error())
	case errors.Is(err, domain.ErrConnectionTimeout):
		return domain.ErrorResponse(c, fiber.StatusGatewayTimeout, "CONNECTION_TIMEOUT", err.Error())
	case errors.Is(err, domain.ErrDecryptionFailed):
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "CREDENTIAL_ERROR", "gagal membaca credential router")
	default:
		h.logger.Error().Err(err).Msg("unhandled error di terminal handler")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "terjadi kesalahan internal")
	}
}
