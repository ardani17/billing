// map_node_handler.go menangani HTTP request untuk manajemen map node.
// Endpoint foto dan riwayat ada di map_node_handler_photo.go.
package handler

import (
	"errors"
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"

	"github.com/ispboss/ispboss/pkg/tenant"
	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// MapNodeHandler menangani HTTP request untuk operasi map node.
type MapNodeHandler struct {
	manager  domain.MapNodeManager
	validate *validator.Validate
}

// NewMapNodeHandler membuat instance baru MapNodeHandler.
func NewMapNodeHandler(manager domain.MapNodeManager) *MapNodeHandler {
	return &MapNodeHandler{
		manager:  manager,
		validate: validator.New(),
	}
}

// ListNodes menangani GET /nodes.
// Parse query params bounding box (sw_lat, sw_lng, ne_lat, ne_lng) dan filter.
func (h *MapNodeHandler) ListNodes(c *fiber.Ctx) error {
	tenantID := tenant.FromContext(c.UserContext())
	if tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	params := domain.MapNodeListParams{
		TenantID:      tenantID,
		NodeType:      c.Query("node_type"),
		Status:        c.Query("status"),
		BillingStatus: c.Query("billing_status"),
		PackageID:     c.Query("package_id"),
		AreaID:        c.Query("area_id"),
		ODPID:         c.Query("odp_id"),
	}

	// Parse bounding box (opsional)
	for _, p := range []struct {
		key string
		dst *float64
	}{
		{"min_lat", &params.MinLat}, {"max_lat", &params.MaxLat},
		{"min_lng", &params.MinLng}, {"max_lng", &params.MaxLng},
	} {
		if v := c.Query(p.key); v != "" {
			f, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", p.key+" tidak valid")
			}
			*p.dst = f
		}
	}

	result, err := h.manager.ListNodes(c.UserContext(), params)
	if err != nil {
		return h.mapError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, result)
}

// CreateNode menangani POST /nodes.
// Parse body, validasi, extract tenant_id, lalu buat map node baru.
func (h *MapNodeHandler) CreateNode(c *fiber.Ctx) error {
	tenantID := tenant.FromContext(c.UserContext())
	if tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	var req domain.CreateMapNodeRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "request body tidak valid")
	}

	if err := h.validate.Struct(req); err != nil {
		return h.validationError(c, err)
	}

	resp, err := h.manager.CreateNode(c.UserContext(), tenantID, req)
	if err != nil {
		return h.mapError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusCreated, resp)
}

// GetNode menangani GET /nodes/:id.
// Mengambil detail map node berdasarkan ID termasuk foto dan riwayat.
func (h *MapNodeHandler) GetNode(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "node ID wajib diisi")
	}

	resp, err := h.manager.GetNode(c.UserContext(), id)
	if err != nil {
		return h.mapError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, resp)
}

// UpdateNode menangani PUT /nodes/:id.
// Parse body, validasi, lalu update lokasi dan/atau custom fields node.
func (h *MapNodeHandler) UpdateNode(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "node ID wajib diisi")
	}

	var req domain.UpdateMapNodeRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "request body tidak valid")
	}

	if err := h.validate.Struct(req); err != nil {
		return h.validationError(c, err)
	}

	resp, err := h.manager.UpdateNode(c.UserContext(), id, req)
	if err != nil {
		return h.mapError(c, err)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, resp)
}

// DeleteNode menangani DELETE /nodes/:id.
// Soft-delete map node dan catat riwayat perubahan.
func (h *MapNodeHandler) DeleteNode(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "node ID wajib diisi")
	}

	tenantID := tenant.FromContext(c.UserContext())
	if err := h.manager.DeleteNode(c.UserContext(), id, tenantID); err != nil {
		return h.mapError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// mapError memetakan domain error map node ke HTTP error response.
func (h *MapNodeHandler) mapError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domain.ErrMapNodeNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "MAP_NODE_NOT_FOUND", err.Error())
	case errors.Is(err, domain.ErrMapNodeDuplicate):
		return domain.ErrorResponse(c, fiber.StatusConflict, "MAP_NODE_DUPLICATE", err.Error())
	case errors.Is(err, domain.ErrMapNodeDeleted):
		return domain.ErrorResponse(c, fiber.StatusGone, "MAP_NODE_DELETED", err.Error())
	case errors.Is(err, domain.ErrInvalidNodeType):
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "INVALID_NODE_TYPE", err.Error())
	case errors.Is(err, domain.ErrInvalidCoordinates):
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "INVALID_COORDINATES", err.Error())
	case errors.Is(err, domain.ErrReferenceNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "REFERENCE_NOT_FOUND", err.Error())
	case errors.Is(err, domain.ErrPhotoLimitReached):
		return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "PHOTO_LIMIT_REACHED", err.Error())
	case errors.Is(err, domain.ErrInvalidFileType):
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "INVALID_FILE_TYPE", err.Error())
	case errors.Is(err, domain.ErrFileTooLarge):
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "FILE_TOO_LARGE", err.Error())
	case errors.Is(err, domain.ErrPhotoNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "PHOTO_NOT_FOUND", err.Error())
	default:
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "terjadi kesalahan internal")
	}
}

// validationError menangani error validasi dari go-playground/validator.
func (h *MapNodeHandler) validationError(c *fiber.Ctx, err error) error {
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
