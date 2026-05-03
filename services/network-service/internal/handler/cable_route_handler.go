// cable_route_handler.go menangani HTTP request untuk manajemen cable route.
// Termasuk: CRUD dan list dengan bounding box.
package handler

import (
	"errors"
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"

	"github.com/ispboss/ispboss/pkg/tenant"
	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// CableRouteHandler menangani HTTP request untuk operasi cable route.
type CableRouteHandler struct {
	manager  domain.CableRouteManager
	validate *validator.Validate
}

// NewCableRouteHandler membuat instance baru CableRouteHandler.
func NewCableRouteHandler(manager domain.CableRouteManager) *CableRouteHandler {
	return &CableRouteHandler{
		manager:  manager,
		validate: validator.New(),
	}
}

// ListRoutes menangani GET /cables.
// Parse query params bounding box dan filter, return daftar cable route.
func (h *CableRouteHandler) ListRoutes(c *fiber.Ctx) error {
	tenantID := tenant.FromContext(c.UserContext())
	if tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	params := domain.CableRouteListParams{
		TenantID:   tenantID,
		RouteType:  c.Query("route_type"),
		FromNodeID: c.Query("from_node_id"),
		ToNodeID:   c.Query("to_node_id"),
	}

	// Parse bounding box (opsional)
	if v := c.Query("min_lat"); v != "" {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "min_lat tidak valid")
		}
		params.MinLat = f
	}
	if v := c.Query("max_lat"); v != "" {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "max_lat tidak valid")
		}
		params.MaxLat = f
	}
	if v := c.Query("min_lng"); v != "" {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "min_lng tidak valid")
		}
		params.MinLng = f
	}
	if v := c.Query("max_lng"); v != "" {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "max_lng tidak valid")
		}
		params.MaxLng = f
	}

	result, err := h.manager.ListRoutes(c.UserContext(), params)
	if err != nil {
		return h.mapError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, result)
}

// CreateRoute menangani POST /cables.
// Parse body, validasi, extract tenant_id, lalu buat cable route baru.
func (h *CableRouteHandler) CreateRoute(c *fiber.Ctx) error {
	tenantID := tenant.FromContext(c.UserContext())
	if tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	var req domain.CreateCableRouteRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "request body tidak valid")
	}

	if err := h.validate.Struct(req); err != nil {
		return h.validationError(c, err)
	}

	resp, err := h.manager.CreateRoute(c.UserContext(), tenantID, req)
	if err != nil {
		return h.mapError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusCreated, resp)
}

// GetRoute menangani GET /cables/:id.
// Mengambil detail cable route berdasarkan ID.
func (h *CableRouteHandler) GetRoute(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "cable route ID wajib diisi")
	}

	resp, err := h.manager.GetRoute(c.UserContext(), id)
	if err != nil {
		return h.mapError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, resp)
}

// UpdateRoute menangani PUT /cables/:id.
// Parse body, validasi, lalu update cable route.
func (h *CableRouteHandler) UpdateRoute(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "cable route ID wajib diisi")
	}

	var req domain.UpdateCableRouteRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "request body tidak valid")
	}

	resp, err := h.manager.UpdateRoute(c.UserContext(), id, req)
	if err != nil {
		return h.mapError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, resp)
}

// DeleteRoute menangani DELETE /cables/:id.
// Soft-delete cable route.
func (h *CableRouteHandler) DeleteRoute(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "cable route ID wajib diisi")
	}

	if err := h.manager.DeleteRoute(c.UserContext(), id); err != nil {
		return h.mapError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// mapError memetakan domain error cable route ke HTTP error response.
func (h *CableRouteHandler) mapError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domain.ErrCableRouteNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "CABLE_ROUTE_NOT_FOUND", err.Error())
	case errors.Is(err, domain.ErrInvalidRouteType):
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "INVALID_ROUTE_TYPE", err.Error())
	case errors.Is(err, domain.ErrInvalidCoordArray):
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "INVALID_COORDINATES", err.Error())
	case errors.Is(err, domain.ErrMapNodeNotFound), errors.Is(err, domain.ErrNodeNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "NODE_NOT_FOUND", err.Error())
	default:
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "terjadi kesalahan internal")
	}
}

// validationError menangani error validasi dari go-playground/validator.
func (h *CableRouteHandler) validationError(c *fiber.Ctx, err error) error {
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
