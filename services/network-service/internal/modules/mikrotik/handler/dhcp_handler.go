package handler

import (
	"errors"
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/pkg/tenant"
	"github.com/ispboss/ispboss/services/network-service/internal/domain"
	"github.com/ispboss/ispboss/services/network-service/internal/modules/mikrotik/usecase"
)

type DHCPHandler struct {
	manager  usecase.DHCPManager
	logger   zerolog.Logger
	validate *validator.Validate
}

func NewDHCPHandler(manager usecase.DHCPManager, logger zerolog.Logger) *DHCPHandler {
	return &DHCPHandler{manager: manager, logger: logger, validate: validator.New()}
}

func (h *DHCPHandler) ListServers(c *fiber.Ctx) error {
	items, err := h.manager.ListServers(c.UserContext(), c.Params("id"))
	if err != nil {
		return h.mapError(c, err)
	}
	return domain.SuccessResponse(c, fiber.StatusOK, items)
}

func (h *DHCPHandler) ListLeases(c *fiber.Ctx) error {
	items, err := h.manager.ListLeases(c.UserContext(), c.Params("id"))
	if err != nil {
		return h.mapError(c, err)
	}
	return domain.SuccessResponse(c, fiber.StatusOK, items)
}

func (h *DHCPHandler) ListNetworks(c *fiber.Ctx) error {
	items, err := h.manager.ListNetworks(c.UserContext(), c.Params("id"))
	if err != nil {
		return h.mapError(c, err)
	}
	return domain.SuccessResponse(c, fiber.StatusOK, items)
}

func (h *DHCPHandler) ListBindings(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("page_size", "20"))
	result, err := h.manager.ListBindings(c.UserContext(), c.Params("id"), domain.DHCPBindingListParams{
		Page: page, PageSize: pageSize, Search: c.Query("search"),
	})
	if err != nil {
		return h.mapError(c, err)
	}
	return domain.PaginatedResponse(c, fiber.StatusOK, result.Data, result.Total, result.Page, result.PageSize, result.TotalPages)
}

func (h *DHCPHandler) CreateBinding(c *fiber.Ctx) error {
	tenantID := tenant.FromContext(c.UserContext())
	if tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}
	var req domain.CreateDHCPBindingRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "request body tidak valid")
	}
	if err := h.validate.Struct(req); err != nil {
		return h.validationError(c, err)
	}
	ctx := usecase.WithMikroTikAuditActor(c.UserContext(), fiberLocalsString(c, "user_id"), c.IP())
	item, err := h.manager.CreateBinding(ctx, tenantID, c.Params("id"), req)
	if err != nil {
		return h.mapError(c, err)
	}
	return domain.SuccessResponse(c, fiber.StatusCreated, item)
}

func (h *DHCPHandler) UpdateBinding(c *fiber.Ctx) error {
	var req domain.UpdateDHCPBindingRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "request body tidak valid")
	}
	ctx := usecase.WithMikroTikAuditActor(c.UserContext(), fiberLocalsString(c, "user_id"), c.IP())
	item, err := h.manager.UpdateBinding(ctx, c.Params("id"), c.Params("binding_id"), req)
	if err != nil {
		return h.mapError(c, err)
	}
	return domain.SuccessResponse(c, fiber.StatusOK, item)
}

func (h *DHCPHandler) DeleteBinding(c *fiber.Ctx) error {
	var req domain.DeleteDHCPBindingRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "request body tidak valid")
	}
	if err := h.validate.Struct(req); err != nil {
		return h.validationError(c, err)
	}
	ctx := usecase.WithMikroTikAuditActor(c.UserContext(), fiberLocalsString(c, "user_id"), c.IP())
	if err := h.manager.DeleteBinding(ctx, c.Params("id"), c.Params("binding_id"), req); err != nil {
		return h.mapError(c, err)
	}
	return domain.SuccessResponse(c, fiber.StatusOK, fiber.Map{"message": "dhcp binding berhasil dihapus"})
}

func (h *DHCPHandler) mapError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domain.ErrRouterNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "ROUTER_NOT_FOUND", err.Error())
	case errors.Is(err, domain.ErrDHCPBindingNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "DHCP_BINDING_NOT_FOUND", err.Error())
	case errors.Is(err, domain.ErrDHCPBindingExists):
		return domain.ErrorResponse(c, fiber.StatusConflict, "DHCP_BINDING_EXISTS", err.Error())
	case errors.Is(err, domain.ErrInvalidMACAddress), errors.Is(err, domain.ErrInvalidIPAddress), errors.Is(err, domain.ErrConfirmationMismatch):
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", err.Error())
	case errors.Is(err, domain.ErrConnectionFailed):
		return domain.ErrorResponse(c, fiber.StatusBadGateway, "CONNECTION_FAILED", err.Error())
	case errors.Is(err, domain.ErrConnectionTimeout):
		return domain.ErrorResponse(c, fiber.StatusGatewayTimeout, "CONNECTION_TIMEOUT", err.Error())
	case errors.Is(err, domain.ErrDecryptionFailed):
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "CREDENTIAL_ERROR", "gagal membaca credential router")
	default:
		h.logger.Error().Err(err).Msg("unhandled error di dhcp handler")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "terjadi kesalahan internal")
	}
}

func (h *DHCPHandler) validationError(c *fiber.Ctx, err error) error {
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		fields := make([]domain.FieldError, 0, len(ve))
		for _, fe := range ve {
			fields = append(fields, domain.FieldError{Field: toSnakeCase(fe.Field()), Message: validationMessage(fe)})
		}
		return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "validasi gagal", fields...)
	}
	return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
}
