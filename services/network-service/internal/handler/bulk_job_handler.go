package handler

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/ispboss/ispboss/services/network-service/internal/domain"
	"github.com/ispboss/ispboss/services/network-service/internal/usecase"
	"github.com/rs/zerolog"
)

type MikroTikBulkHandler struct {
	manager usecase.MikroTikBulkManager
	logger  zerolog.Logger
}

func NewMikroTikBulkHandler(manager usecase.MikroTikBulkManager, logger zerolog.Logger) *MikroTikBulkHandler {
	return &MikroTikBulkHandler{manager: manager, logger: logger}
}

func (h *MikroTikBulkHandler) Create(c *fiber.Ctx) error {
	if !canUseMikroTikTerminal(c) {
		return domain.ErrorResponse(c, fiber.StatusForbidden, "FORBIDDEN", "role tidak diizinkan menjalankan bulk action MikroTik")
	}
	var req domain.CreateMikroTikBulkJobRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "payload tidak valid")
	}
	req.Scope = strings.TrimSpace(req.Scope)
	ctx := usecase.WithMikroTikAuditActor(c.UserContext(), fiberLocalsString(c, "user_id"), c.IP())
	job, err := h.manager.CreateJob(ctx, req)
	if err != nil {
		return h.mapError(c, err)
	}
	return domain.SuccessResponse(c, fiber.StatusCreated, job)
}

func (h *MikroTikBulkHandler) List(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("page_size", "20"))
	result, err := h.manager.ListJobs(c.UserContext(), domain.MikroTikBulkJobListParams{
		Page: page, PageSize: pageSize, Action: strings.TrimSpace(c.Query("action")), Status: strings.TrimSpace(c.Query("status")),
	})
	if err != nil {
		return h.mapError(c, err)
	}
	return domain.SuccessResponse(c, fiber.StatusOK, result)
}

func (h *MikroTikBulkHandler) Get(c *fiber.Ctx) error {
	job, err := h.manager.GetJob(c.UserContext(), c.Params("id"))
	if err != nil {
		return h.mapError(c, err)
	}
	return domain.SuccessResponse(c, fiber.StatusOK, job)
}

func (h *MikroTikBulkHandler) mapError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domain.ErrInvalidBulkAction):
		return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "INVALID_BULK_ACTION", err.Error())
	case errors.Is(err, domain.ErrMikroTikBulkJobNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "BULK_JOB_NOT_FOUND", err.Error())
	case errors.Is(err, domain.ErrRouterNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "ROUTER_NOT_FOUND", err.Error())
	case errors.Is(err, domain.ErrConnectionFailed):
		return domain.ErrorResponse(c, fiber.StatusBadGateway, "CONNECTION_FAILED", err.Error())
	case errors.Is(err, domain.ErrConnectionTimeout):
		return domain.ErrorResponse(c, fiber.StatusGatewayTimeout, "CONNECTION_TIMEOUT", err.Error())
	default:
		h.logger.Error().Err(err).Msg("unhandled error di bulk mikrotik handler")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "terjadi kesalahan internal")
	}
}
