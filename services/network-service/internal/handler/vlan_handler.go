// vlan_handler.go menangani HTTP request untuk manajemen VLAN per OLT.
// Termasuk: CRUD VLAN (create, list, update, delete).
package handler

import (
	"errors"
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"

	"github.com/ispboss/ispboss/pkg/tenant"
	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// VLANHandler menangani HTTP request untuk operasi VLAN.
type VLANHandler struct {
	manager  domain.VLANManager
	validate *validator.Validate
}

// NewVLANHandler membuat instance baru VLANHandler.
func NewVLANHandler(manager domain.VLANManager) *VLANHandler {
	return &VLANHandler{
		manager:  manager,
		validate: validator.New(),
	}
}

// CreateVLAN menangani POST /devices/:id/vlans.
// Parse body, validasi, set OLT ID dari path, lalu buat VLAN baru.
func (h *VLANHandler) CreateVLAN(c *fiber.Ctx) error {
	tenantID := tenant.FromContext(c.UserContext())
	if tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant tidak teridentifikasi")
	}

	oltID := c.Params("id")
	if oltID == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "OLT ID wajib diisi")
	}

	var req domain.CreateVLANRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "request body tidak valid")
	}
	req.OLTID = oltID

	if err := h.validate.Struct(req); err != nil {
		return h.validationError(c, err)
	}

	resp, err := h.manager.Create(c.UserContext(), tenantID, req)
	if err != nil {
		return h.mapError(c, err)
	}
	return domain.SuccessResponse(c, fiber.StatusCreated, resp)
}

// ListVLANs menangani GET /devices/:id/vlans.
// Parse query params (page, page_size) dan return paginated list.
func (h *VLANHandler) ListVLANs(c *fiber.Ctx) error {
	oltID := c.Params("id")
	if oltID == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "OLT ID wajib diisi")
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	params := domain.VLANListParams{
		Page:     page,
		PageSize: pageSize,
	}

	result, err := h.manager.List(c.UserContext(), oltID, params)
	if err != nil {
		return h.mapError(c, err)
	}
	return domain.PaginatedResponse(c, fiber.StatusOK, result.Data, result.Total, result.Page, result.PageSize, result.TotalPages)
}

// UpdateVLAN menangani PUT /vlans/:id.
// Parse body, validasi, lalu update data VLAN.
func (h *VLANHandler) UpdateVLAN(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "VLAN ID wajib diisi")
	}

	var req domain.UpdateVLANRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "request body tidak valid")
	}
	if err := h.validate.Struct(req); err != nil {
		return h.validationError(c, err)
	}

	resp, err := h.manager.Update(c.UserContext(), id, req)
	if err != nil {
		return h.mapError(c, err)
	}
	return domain.SuccessResponse(c, fiber.StatusOK, resp)
}

// DeleteVLAN menangani DELETE /vlans/:id.
// Soft-delete VLAN (cek tidak ada ONT aktif yang menggunakan).
func (h *VLANHandler) DeleteVLAN(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "VLAN ID wajib diisi")
	}

	if err := h.manager.Delete(c.UserContext(), id); err != nil {
		return h.mapError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// mapError memetakan domain error VLAN ke HTTP error response.
func (h *VLANHandler) mapError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domain.ErrVLANNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "VLAN_NOT_FOUND", err.Error())
	case errors.Is(err, domain.ErrVLANIDExists):
		return domain.ErrorResponse(c, fiber.StatusConflict, "VLAN_ID_EXISTS", err.Error())
	case errors.Is(err, domain.ErrVLANInUse):
		return domain.ErrorResponse(c, fiber.StatusConflict, "VLAN_IN_USE", err.Error())
	case errors.Is(err, domain.ErrOLTNotFound):
		return domain.ErrorResponse(c, fiber.StatusNotFound, "OLT_NOT_FOUND", err.Error())
	default:
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "terjadi kesalahan internal")
	}
}

// validationError menangani error validasi dari go-playground/validator.
func (h *VLANHandler) validationError(c *fiber.Ctx, err error) error {
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
