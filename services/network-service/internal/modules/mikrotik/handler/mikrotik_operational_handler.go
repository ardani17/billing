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

type MikroTikOperationalHandler struct {
	manager usecase.MikroTikOperationalManager
	logger  zerolog.Logger
}

func NewMikroTikOperationalHandler(manager usecase.MikroTikOperationalManager, logger zerolog.Logger) *MikroTikOperationalHandler {
	return &MikroTikOperationalHandler{manager: manager, logger: logger}
}

func (h *MikroTikOperationalHandler) ListInterfaces(c *fiber.Ctx) error {
	routerID := c.Params("id")
	if routerID == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "router ID wajib diisi")
	}
	items, err := h.manager.ListInterfaces(c.UserContext(), routerID)
	if err != nil {
		return h.mapError(c, err)
	}
	return domain.SuccessResponse(c, fiber.StatusOK, items)
}

func (h *MikroTikOperationalHandler) GetTraffic(c *fiber.Ctx) error {
	routerID := c.Params("id")
	if routerID == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "router ID wajib diisi")
	}
	samples, err := h.manager.GetTraffic(c.UserContext(), routerID, splitQueryList(c.Query("interfaces")))
	if err != nil {
		return h.mapError(c, err)
	}
	return domain.SuccessResponse(c, fiber.StatusOK, samples)
}

func (h *MikroTikOperationalHandler) ListIPPools(c *fiber.Ctx) error {
	routerID := c.Params("id")
	if routerID == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "router ID wajib diisi")
	}
	items, err := h.manager.ListIPPools(c.UserContext(), routerID)
	if err != nil {
		return h.mapError(c, err)
	}
	return domain.SuccessResponse(c, fiber.StatusOK, items)
}

func (h *MikroTikOperationalHandler) ListManagedFirewall(c *fiber.Ctx) error {
	routerID := c.Params("id")
	if routerID == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "router ID wajib diisi")
	}
	rules, err := h.manager.ListManagedFirewall(c.UserContext(), routerID)
	if err != nil {
		return h.mapError(c, err)
	}
	return domain.SuccessResponse(c, fiber.StatusOK, rules)
}

func (h *MikroTikOperationalHandler) ListLogs(c *fiber.Ctx) error {
	routerID := c.Params("id")
	if routerID == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "router ID wajib diisi")
	}
	limit, _ := strconv.Atoi(c.Query("limit", "100"))
	items, err := h.manager.ListLogs(c.UserContext(), routerID, domain.RouterLogFilter{
		Topic:  c.Query("topic"),
		Search: c.Query("search"),
		Limit:  limit,
	})
	if err != nil {
		return h.mapError(c, err)
	}
	return domain.SuccessResponse(c, fiber.StatusOK, items)
}

func (h *MikroTikOperationalHandler) mapError(c *fiber.Ctx, err error) error {
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
		h.logger.Error().Err(err).Msg("unhandled error di mikrotik operational handler")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "terjadi kesalahan internal")
	}
}

func splitQueryList(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			items = append(items, part)
		}
	}
	return items
}
