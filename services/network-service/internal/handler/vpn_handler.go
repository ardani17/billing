// vpn_handler.go menangani HTTP request untuk manajemen VPN tunnel.
// Termasuk: CRUD tunnel, validasi request, dan error mapping.
package handler

import (
	"errors"
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/pkg/tenant"
	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// VPNHandler menangani HTTP request untuk operasi VPN tunnel.
type VPNHandler struct {
	manager  domain.VPNManager
	logger   zerolog.Logger
	validate *validator.Validate
}

// NewVPNHandler membuat instance baru VPNHandler.
func NewVPNHandler(manager domain.VPNManager, logger zerolog.Logger) *VPNHandler {
	return &VPNHandler{manager: manager, logger: logger, validate: validator.New()}
}

// ListTunnels menangani GET /api/v1/mikrotik/vpn/tunnels.
// Parse query params (page, page_size, status, protocol, search) dan return paginated list.
func (h *VPNHandler) ListTunnels(c *fiber.Ctx) error {
	tenantID := tenant.FromContext(c.UserContext())
	if tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	params := domain.VPNTunnelListParams{
		TenantID: tenantID,
		Page:     page,
		PageSize: pageSize,
		Status:   c.Query("status"),
		Protocol: c.Query("protocol"),
		Search:   c.Query("search"),
	}

	result, err := h.manager.ListTunnels(c.UserContext(), params)
	if err != nil {
		return h.mapError(c, err)
	}

	return domain.PaginatedResponse(c, fiber.StatusOK, result.Data, result.Total, result.Page, result.PageSize, result.TotalPages)
}

// CreateTunnel menangani POST /api/v1/mikrotik/vpn/tunnels.
// Parse body, validasi, extract tenant_id, lalu buat tunnel baru.
func (h *VPNHandler) CreateTunnel(c *fiber.Ctx) error {
	tenantID := tenant.FromContext(c.UserContext())
	if tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	var req domain.CreateVPNTunnelRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "request body tidak valid")
	}

	if err := h.validate.Struct(req); err != nil {
		return h.validationError(c, err)
	}

	resp, err := h.manager.CreateTunnel(c.UserContext(), tenantID, req)
	if err != nil {
		return h.mapError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusCreated, resp)
}

// GetTunnel menangani GET /api/v1/mikrotik/vpn/tunnels/:id.
// Mengambil detail tunnel berdasarkan ID.
func (h *VPNHandler) GetTunnel(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "tunnel ID wajib diisi")
	}

	resp, err := h.manager.GetTunnel(c.UserContext(), id)
	if err != nil {
		return h.mapError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, resp)
}

// UpdateTunnel menangani PUT /api/v1/mikrotik/vpn/tunnels/:id.
// Parse body, validasi, lalu update data tunnel.
func (h *VPNHandler) UpdateTunnel(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "tunnel ID wajib diisi")
	}

	var req domain.UpdateVPNTunnelRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "request body tidak valid")
	}

	if err := h.validate.Struct(req); err != nil {
		return h.validationError(c, err)
	}

	resp, err := h.manager.UpdateTunnel(c.UserContext(), id, req)
	if err != nil {
		return h.mapError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, resp)
}

// DeleteTunnel menangani DELETE /api/v1/mikrotik/vpn/tunnels/:id.
// Soft-delete tunnel dan return 204 No Content.
func (h *VPNHandler) DeleteTunnel(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "tunnel ID wajib diisi")
	}

	if err := h.manager.DeleteTunnel(c.UserContext(), id); err != nil {
		return h.mapError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// mapError memetakan domain error ke HTTP error response.
// Mengikuti tabel error mapping dari design document.
func (h *VPNHandler) mapError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domain.ErrVPNTunnelNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "VPN_TUNNEL_NOT_FOUND", err.Error())
	case errors.Is(err, domain.ErrVPNTunnelNameExists):
		return domain.ErrorResponse(c, fiber.StatusConflict, "VPN_TUNNEL_NAME_EXISTS", err.Error())
	case errors.Is(err, domain.ErrVPNIPExists):
		return domain.ErrorResponse(c, fiber.StatusConflict, "VPN_IP_EXISTS", err.Error())
	case errors.Is(err, domain.ErrInvalidVPNProtocol):
		return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "INVALID_VPN_PROTOCOL", err.Error())
	case errors.Is(err, domain.ErrWireGuardRequiresV7):
		return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "WIREGUARD_REQUIRES_V7", err.Error())
	case errors.Is(err, domain.ErrTunnelImmutableField):
		return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "TUNNEL_IMMUTABLE_FIELD", err.Error())
	case errors.Is(err, domain.ErrVPNSubnetExhausted):
		return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "VPN_SUBNET_EXHAUSTED", err.Error())
	case errors.Is(err, domain.ErrRouterNotOnline):
		return domain.ErrorResponse(c, fiber.StatusServiceUnavailable, "ROUTER_NOT_ONLINE", err.Error())
	case errors.Is(err, domain.ErrAutoConfigFailed):
		return domain.ErrorResponse(c, fiber.StatusServiceUnavailable, "AUTO_CONFIG_FAILED", err.Error())
	case errors.Is(err, domain.ErrTunnelDeleteWarning):
		return domain.ErrorResponse(c, fiber.StatusConflict, "TUNNEL_DELETE_WARNING", err.Error())
	default:
		h.logger.Error().Err(err).Msg("unhandled error di vpn handler")
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "terjadi kesalahan internal")
	}
}

// validationError menangani error validasi dari go-playground/validator.
func (h *VPNHandler) validationError(c *fiber.Ctx, err error) error {
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		fields := make([]domain.FieldError, 0, len(ve))
		for _, fe := range ve {
			fields = append(fields, domain.FieldError{
				Field:   toSnakeCase(fe.Field()),
				Message: validationMessage(fe),
			})
		}
		return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "validasi gagal", fields...)
	}
	return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
}
