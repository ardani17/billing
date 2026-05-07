package handler

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
	"github.com/ispboss/ispboss/services/network-service/internal/modules/mikrotik/usecase"
)

type HotspotHandler struct {
	manager usecase.HotspotManager
	logger  zerolog.Logger
}

func NewHotspotHandler(manager usecase.HotspotManager, logger zerolog.Logger) *HotspotHandler {
	return &HotspotHandler{manager: manager, logger: logger}
}

func (h *HotspotHandler) ListUsers(c *fiber.Ctx) error {
	items, err := h.manager.ListUsers(c.UserContext(), c.Params("id"))
	if err != nil {
		return h.mapError(c, err)
	}
	return domain.SuccessResponse(c, fiber.StatusOK, items)
}

func (h *HotspotHandler) CreateUser(c *fiber.Ctx) error {
	var req domain.CreateHotspotUserRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "payload tidak valid")
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Password = strings.TrimSpace(req.Password)
	req.Profile = strings.TrimSpace(req.Profile)
	if req.Name == "" || req.Password == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "name dan password wajib diisi")
	}
	ctx := usecase.WithMikroTikAuditActor(c.UserContext(), fiberLocalsString(c, "user_id"), c.IP())
	item, err := h.manager.CreateUser(ctx, c.Params("id"), req)
	if err != nil {
		return h.mapError(c, err)
	}
	return domain.SuccessResponse(c, fiber.StatusCreated, item)
}

func (h *HotspotHandler) UpdateUser(c *fiber.Ctx) error {
	var req domain.UpdateHotspotUserRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "payload tidak valid")
	}
	ctx := usecase.WithMikroTikAuditActor(c.UserContext(), fiberLocalsString(c, "user_id"), c.IP())
	item, err := h.manager.UpdateUser(ctx, c.Params("id"), c.Params("user_id"), req)
	if err != nil {
		return h.mapError(c, err)
	}
	return domain.SuccessResponse(c, fiber.StatusOK, item)
}

func (h *HotspotHandler) DeleteUser(c *fiber.Ctx) error {
	ctx := usecase.WithMikroTikAuditActor(c.UserContext(), fiberLocalsString(c, "user_id"), c.IP())
	if err := h.manager.DeleteUser(ctx, c.Params("id"), c.Params("user_id")); err != nil {
		return h.mapError(c, err)
	}
	return domain.SuccessResponse(c, fiber.StatusOK, fiber.Map{"message": "hotspot user berhasil dihapus"})
}

func (h *HotspotHandler) ListProfiles(c *fiber.Ctx) error {
	items, err := h.manager.ListProfiles(c.UserContext(), c.Params("id"))
	if err != nil {
		return h.mapError(c, err)
	}
	return domain.SuccessResponse(c, fiber.StatusOK, items)
}

func (h *HotspotHandler) ListActive(c *fiber.Ctx) error {
	items, err := h.manager.ListActive(c.UserContext(), c.Params("id"))
	if err != nil {
		return h.mapError(c, err)
	}
	return domain.SuccessResponse(c, fiber.StatusOK, items)
}

func (h *HotspotHandler) GenerateLoginTemplate(c *fiber.Ctx) error {
	var req domain.HotspotLoginTemplateRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "payload tidak valid")
	}
	template, err := h.manager.GenerateLoginTemplate(c.UserContext(), c.Params("id"), req)
	if err != nil {
		return h.mapError(c, err)
	}
	return domain.SuccessResponse(c, fiber.StatusOK, template)
}

func (h *HotspotHandler) mapError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domain.ErrRouterNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "ROUTER_NOT_FOUND", err.Error())
	case errors.Is(err, domain.ErrHotspotUserNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "HOTSPOT_USER_NOT_FOUND", err.Error())
	case errors.Is(err, domain.ErrConnectionFailed):
		return domain.ErrorResponse(c, fiber.StatusBadGateway, "CONNECTION_FAILED", err.Error())
	case errors.Is(err, domain.ErrConnectionTimeout):
		return domain.ErrorResponse(c, fiber.StatusGatewayTimeout, "CONNECTION_TIMEOUT", err.Error())
	case errors.Is(err, domain.ErrDecryptionFailed):
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "CREDENTIAL_ERROR", "gagal membaca credential router")
	default:
		h.logger.Error().Err(err).Msg("unhandled error di hotspot handler")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "terjadi kesalahan internal")
	}
}
