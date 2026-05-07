package handler

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
	"github.com/ispboss/ispboss/services/network-service/internal/modules/mikrotik/usecase"
)

type WalledGardenHandler struct {
	manager usecase.WalledGardenManager
	logger  zerolog.Logger
}

func NewWalledGardenHandler(manager usecase.WalledGardenManager, logger zerolog.Logger) *WalledGardenHandler {
	return &WalledGardenHandler{manager: manager, logger: logger}
}

func (h *WalledGardenHandler) GetStatus(c *fiber.Ctx) error {
	status, err := h.manager.GetStatus(c.UserContext(), c.Params("id"))
	if err != nil {
		return h.mapError(c, err)
	}
	return domain.SuccessResponse(c, fiber.StatusOK, status)
}

func (h *WalledGardenHandler) Apply(c *fiber.Ctx) error {
	var req domain.ApplyWalledGardenRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "payload tidak valid")
	}
	if err := validateWalledGardenRequest(req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", err.Error())
	}
	status, err := h.manager.Apply(c.UserContext(), c.Params("id"), req)
	if err != nil {
		return h.mapError(c, err)
	}
	return domain.SuccessResponse(c, fiber.StatusOK, status)
}

func (h *WalledGardenHandler) Remove(c *fiber.Ctx) error {
	status, err := h.manager.Remove(c.UserContext(), c.Params("id"))
	if err != nil {
		return h.mapError(c, err)
	}
	return domain.SuccessResponse(c, fiber.StatusOK, status)
}

func validateWalledGardenRequest(req domain.ApplyWalledGardenRequest) error {
	method := strings.TrimSpace(req.Method)
	if method != "" {
		switch method {
		case domain.WalledGardenMethodDNSRedirect, domain.WalledGardenMethodHTTPRedirect, domain.WalledGardenMethodBlockAllWhitelist:
		default:
			return errors.New("method harus dns_redirect, http_redirect, atau block_all_whitelist")
		}
	}
	return nil
}

func (h *WalledGardenHandler) mapError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domain.ErrRouterNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "ROUTER_NOT_FOUND", err.Error())
	case errors.Is(err, domain.ErrConnectionFailed):
		return domain.ErrorResponse(c, fiber.StatusBadGateway, "CONNECTION_FAILED", err.Error())
	case errors.Is(err, domain.ErrConnectionTimeout):
		return domain.ErrorResponse(c, fiber.StatusGatewayTimeout, "CONNECTION_TIMEOUT", err.Error())
	case errors.Is(err, domain.ErrDecryptionFailed):
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "CREDENTIAL_ERROR", "gagal membaca credential router")
	default:
		h.logger.Error().Err(err).Msg("unhandled error di walled garden handler")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "terjadi kesalahan internal")
	}
}
